package dist

import (
	"errors"
	"github.com/gin-gonic/gin"
	"strconv"
	"xgit.xiaoniangao.cn/recsys/traffic_dist_recall/conf"
	"xgit.xiaoniangao.cn/recsys/traffic_dist_recall/service/dist"
	"xgit.xiaoniangao.cn/recsys/traffic_dist_recall_api"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
)

const (
	NewUserLimit       = 10
	UnPublishThreshold = 8000e10
)

type AlbumType struct {
	TplID int64 `json:"tpl_id"`
	S     int   `json:"s"`
	Ct    int64 `json:"ct"`
	AID   int64 `json:"_id"`
	Mid   int64 `json:"mid"`
}

const (
	ContributeAlbum = 1
	FeatureAlbum    = 2

	ScoreThreshold    = 0.02
	MaxScoreThreshold = 0.5

	ScoreBaseNum = 10

	CountBaseNum = 50
)

type ProfileDataType struct {
	ProfileCt    int64 `json:"profile_ct"`
	ExpUv        int64 `json:"exp_uv"`
	ExpUvFeed    int64 `json:"exp_uv_feed"`
	ExpUvRcmd    int64 `json:"exp_uv_rcmd"`
	ExpUvShare   int64 `json:"exp_uv_share"`
	ClickUv      int64 `json:"click_uv"`
	ShareUv      int64 `json:"share_uv"`
	CommentTotal int64 `json:"comment_total"`
	FavorUv      int64 `json:"favor_uv"`
	ViewsAlbum   int64 `json:"views_album"`
	ShareBack    int64 `json:"share_back"`
}
type ContentReq struct {
	ProfileID   int64            `json:"profile_id"`
	Album       *AlbumType       `json:"album"`
	ProfileData *ProfileDataType `json:"profile_data"`
	IsOriginal  bool             `json:"is_original"`
}

func checkParam(req *ContentReq) bool {
	if req == nil {
		return false
	}
	if req.Album == nil {
		return false
	}
	if req.ProfileData == nil {
		return false
	}
	if req.ProfileID <= 0 {
		return false
	}
	return true
}

// TrafficDistContent 用于设置分发内容
func TrafficDistContent(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req *ContentReq
	if !xc.GetReqObject(&req) {
		return
	}

	if !checkParam(req) {
		xc.ReplyFail(lib.CodePara)
		conf.Logger.Error("fail to check param", "req", req)
		return
	}
	//如果该影集不是佳作,原创,视频剪辑就不给分发
	if req.Album.S != FeatureAlbum && req.IsOriginal == false && req.Album.TplID != 0 {
		conf.Logger.Info("req is not feature or not original or not video", "s", req.Album.S, "isOriginal", req.IsOriginal, "tpl_id", req.Album.TplID)
		xc.ReplyOKWithoutData()
		return
	}

	//param1 := req.ProfileData.ViewsAlbum - req.ProfileData.ShareBack
	//if param1 <= 0 {
	//	conf.Logger.Error("views > share back", "req profile data", req.ProfileData)
	//	xc.ReplyFail(lib.CodePara)
	//	return
	//}

	expUvDist := req.ProfileData.ExpUvFeed + req.ProfileData.ExpUvRcmd

	//点击/（曝光+10） *  回流/（阅读-回流+10）
	//（点击+1） /（曝光+10） *  分享 /（阅读+10） * 回流 / （分享+10）
	score := float32(req.ProfileData.ClickUv+1) / float32(req.ProfileData.ExpUv+ScoreBaseNum) * float32(req.ProfileData.ShareUv) / float32(req.ProfileData.ViewsAlbum+ScoreBaseNum) * float32(req.ProfileData.ShareBack) / float32(req.ProfileData.ShareUv+ScoreBaseNum)
	if score >= MaxScoreThreshold {
		conf.Logger.Error("score > max score threshold", "req profile data", req.ProfileData, "score", score, "max score", MaxScoreThreshold)
		xc.ReplyOKWithoutData()
		return
	}
	count := 0
	if score >= ScoreThreshold {
		count = int(score * float32(CountBaseNum) * float32(expUvDist+10))
	}

	distReq := &traffic_dist_recall_api.TrafficDistReq{
		ProfileId:     req.ProfileID,
		AlbumId:       req.Album.AID,
		ProfileMid:    req.Album.Mid,
		Count:         count,
		Tag:           traffic_dist_recall_api.SecondDistTag,
		PriorityLevel: traffic_dist_recall_api.HighQualityLevel,
		Score:         score,
	}

	if err := dist.TrafficDistContent(distReq); err != nil {
		conf.Logger.Error("traffic dist content", "req profile data", req.ProfileData, "req album", req.Album, "err", err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	xc.ReplyOK(struct{}{})
}

// TrafficDist 用于设置分发内容 id count
func TrafficDist(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req traffic_dist_recall_api.TrafficDistReq
	if !xc.GetReqObject(&req) {
		return
	}
	if req.Count <= 0 || req.ProfileId <= 0 || req.AlbumId <= 0 || req.ProfileMid <= 0 || req.PriorityLevel <= 0 || req.ProfileId >= UnPublishThreshold {
		xc.ReplyFail(lib.CodePara)
		return
	}
	//新发表的暂时不进
	if req.PriorityLevel == traffic_dist_recall_api.NewPublishLevel {
		xc.ReplyOK(struct{}{})
		return
	}
	if err := dist.TrafficDist(&req); err != nil {
		conf.Logger.Error("fail to traffic dist", "req", req, "err", err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	xc.ReplyOK(struct{}{})
}

// TrafficGetIds获取分发的id
func TrafficGetIds(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req traffic_dist_recall_api.TrafficGetReq
	if !xc.GetReqObject(&req) {
		return
	}

	//如果历史记录少于10条，认为是新用户，就不给发这样的id
	if len(req.HistoryIds) <= 1 && conf.Env == lib.PROD {
		if len(req.HistoryIds) <= 0 || req.HistoryIds[0] == nil || len(*req.HistoryIds[0]) <= NewUserLimit {
			conf.Logger.Info("it's a new user, because of history ids less then 10", "mid", req.Mid)
			xc.ReplyOK(map[string]int64{"id": 0, "mid": 0})
			return
		}
	}
	id, mid, expValue, level, err := dist.TrafficGetId(&req)
	if err != nil {
		conf.Logger.Error("fail to traffic get id", "req", req, "err", err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	//view.AddAnonView(&req)
	resp := map[string]interface{}{
		"id":    id,
		"mid":   mid,
		"ab":    map[string]string{conf.C.Experiments.Dispatch: expValue},
		"level": level,
	}
	xc.ReplyOK(resp)
}

// TrafficList获取分发列表
func TrafficList(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req traffic_dist_recall_api.TrafficListReq
	if !xc.GetReqObject(&req) {
		return
	}
	if req.Order <= 0 || req.Limit <= 0 {
		xc.ReplyFail(lib.CodePara)
		return
	}
	resp, err := dist.TrafficList(&req)
	if err != nil {
		conf.Logger.Error("fail to traffic list", "req", req, "err", err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	xc.ReplyOK(resp)
}

// IsDisting 是否在分发队列里
func IsDisting(c *gin.Context) {
	id := parseParams(c)
	xc := xng.NewXContext(c)
	if id <= 0 {
		conf.Logger.Error("fail to parse params", "id", id)
		xc.ReplyFail(lib.CodePara)
		return
	}
	yes, level, err := dist.IsDisting(id)
	if err != nil {
		conf.Logger.Error("fail to judge is dist", "id", id, "err", err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	xc.ReplyOK(map[string]interface{}{
		"yes":   yes,
		"level": level,
	})
	return
}

func parseInt64FromQuery(ctx *gin.Context, key string) (result int64, err error) {
	resultStr := ctx.Param(key)

	if resultStr == "" {
		err = errors.New("no param of key:" + key)
		return
	}
	return strconv.ParseInt(resultStr, 10, 64)
}
func parseParams(ctx *gin.Context) (id int64) {
	var err error
	id, err = parseInt64FromQuery(ctx, "id")
	if err != nil {
		return
	}
	return
}
