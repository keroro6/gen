package dist

import (
	"fmt"
	"sort"
	"time"
	"xgit.xiaoniangao.cn/recsys/traffic_dist_recall/conf"
	"xgit.xiaoniangao.cn/recsys/traffic_dist_recall/dao/distdao"
	"xgit.xiaoniangao.cn/recsys/traffic_dist_recall/redis/distcluster"
	"xgit.xiaoniangao.cn/recsys/traffic_dist_recall/service/abtest"
	"xgit.xiaoniangao.cn/recsys/traffic_dist_recall_api"
	rediscluster "xgit.xiaoniangao.cn/xngo/lib/github.com.chasex.redis-go-cluster"
	"xgit.xiaoniangao.cn/xngo/lib/github.com.globalsign.mgo/bson"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/service/bizsrv_api"
)

var ExpCanDispatchValueSlot = []string{"slot10", "slot11", "slot12", "slot13", "slot14", "slot16", "slot17"}

const (
	offset         = 0
	limit          = 5
	CacheTime      = time.Second * 1
	CanDistTimeOut = time.Minute * 10
	IsDistTimeOut  = time.Minute * 1

	DispatchConfigKey   = "dispatch_config"
	ExpCanDispatchValue = "dispatch_video"

	//ExpVideoCanDispatchValue = "dispatch_video"

	DefaultMaxNum = 500000
)

var MultiQueue = []traffic_dist_recall_api.LevelType{traffic_dist_recall_api.NewPublishLevel, traffic_dist_recall_api.HighQualityLevel, traffic_dist_recall_api.SelfDefineLevel}

func TrafficDistContent(distContentReq *traffic_dist_recall_api.TrafficDistReq) (err error) {

	ok, errIgnore := distcluster.IsDisting(distContentReq.ProfileId, distContentReq.PriorityLevel)

	//如果在队列里不管有没有count都更新分
	if errIgnore == nil && ok {
		err = distcluster.UpdateDistScore(distContentReq.ProfileId, distContentReq.ProfileMid, distContentReq.PriorityLevel, distContentReq.Score)
		return
	}
	if distContentReq.Count <= 0 {
		return
	}
	//如果不在队列里，才能加分加count
	err = TrafficDist(distContentReq)

	return
}

// TrafficDist 添加加分发id
func TrafficDist(req *traffic_dist_recall_api.TrafficDistReq) (err error) {

	beginTime := time.Now().UnixNano() / 1e6
	if err = distcluster.AddDistIdCluster(req.ProfileId, req.ProfileMid, req.Count, req.PriorityLevel, req.Score); err != nil {
		return
	}
	distModel := &distdao.DistModel{
		AlbumId:       req.AlbumId,
		ProfileMid:    req.ProfileMid,
		ProfileId:     req.ProfileId,
		OpMId:         req.OpMId,
		Tag:           req.Tag,
		Count:         req.Count,
		PriorityLevel: req.PriorityLevel,
		BeginTime:     beginTime,
	}
	err = distdao.AddDistRecord(distModel)

	return
}

// TrafficGetId 获取分发id
func TrafficGetId(req *traffic_dist_recall_api.TrafficGetReq) (RetId, mid int64, expValue string, level traffic_dist_recall_api.LevelType, err error) {

	//判断是否给这个用户分发id
	ok, expValue := canDist(req)
	if !ok {
		conf.Logger.Debug("the user can't dist, last visited time less than 10 minutes")
		return
	}
	//需要1h内，只会给同一个用户分发一个id
	//设置1h不能取的缓存
	setCanDist(req)

	localKey := getCacheKey()
	//由于会删除的，后面的总会排到，所以没必要offset limit多次， 取5个id就够了
	var ids []int64
	//var level traffic_dist_recall_api.LevelType
	levelIdsMap := map[traffic_dist_recall_api.LevelType][]int64{}
	if expValue == ExpCanDispatchValue || HitExp(expValue) {
		result, ok := conf.LocalCache.Get(localKey)
		if ok {
			switch val := result.(type) {
			case map[traffic_dist_recall_api.LevelType][]int64:
				levelIdsMap = val
			default:
				conf.Logger.Error("unknown local cache type", "val", val, "result", result)
			}
		}
	}

	if len(levelIdsMap) <= 0 {
		//想要随机的id
		//ids, err = distcluster.GetDistIdsCluster(req.PriorityLevel, req.Tag, limit)
		ids, level, err = getIdsFromMultiQueue(limit, expValue)
		if err != nil {
			return
		}
		if expValue == ExpCanDispatchValue || HitExp(expValue) {
			levelIdsMap[level] = ids
			conf.LocalCache.Set(localKey, levelIdsMap, CacheTime)
		}
	} else {
		for lvl, idArr := range levelIdsMap {
			level = lvl
			ids = idArr
		}
	}

	var ret int
	var errIgnore error

	for _, id := range ids {
		//过滤历史
		if BingoHistory(id, req.Kind, req.HistoryIds) {
			conf.Logger.Debug("bingo history", "req", req)
			continue
		}
		conf.Logger.Debug("it's not bingo history", "id", id)
		//先减count
		ret, errIgnore = distcluster.MinusCountCluster(id, level)
		if errIgnore != nil {
			conf.Logger.Error("minus count err", "id", id, "level", level, "errIgnore", err)
			continue
		}
		//conf.Logger.Info("minus count", "id", id, "ret", ret)
		//判断count的数量， >=0 就返回
		if ret >= 0 {
			RetId = id
			mid, errIgnore = distcluster.GetDistMidByIdCluster(id)
			if errIgnore != nil && errIgnore != rediscluster.ErrNil {
				conf.Logger.Error("fail to get mid by id", "id", id, "err", errIgnore)
				continue
			}
			// == nil 正常获取到了id和mid
			if errIgnore == nil && mid > 0 {
				conf.Logger.Debug("succ get id mid", "id", id, "mid", mid)
				return
			}
			// == ErrNil 或者mid无效  可能过期了或者插入失败，要往下面走，删除该id的分发
		}

		//删掉该分发id
		ret, errIgnore = distcluster.DelDistIdCluster(id, level)
		if errIgnore != nil {
			conf.Logger.Error("fail to del dist id", "req", req, "errIgnore", errIgnore)
			continue
		}
		if ret != 1 {
			conf.Logger.Debug("it's del a empty key")
			continue
		}
		//只有一个请求删除成功，redis返回1，此时再去记录历史
		//记录结束历史
		go SetEndTime(id, level)
	}
	return
}

// IsDisting 判断是否在分发队列中
func IsDisting(id int64) (yes bool, level traffic_dist_recall_api.LevelType, err error) {
	var errIgnore error
	for _, val := range MultiQueue {
		level = val
		yes, errIgnore = distcluster.IsDisting(id, level)
		if errIgnore != nil {
			conf.Logger.Debug("fail to judge is dist", "errIgnore", errIgnore)
			continue
		}
		if yes {
			return
		}
	}
	level = traffic_dist_recall_api.HighQualityLevel
	return
}

// TrafficList 获取分发列表
func TrafficList(req *traffic_dist_recall_api.TrafficListReq) (resp *traffic_dist_recall_api.TrafficListResp, err error) {

	//用albumid查询的
	//var q bson.M

	q := map[string]interface{}{}
	q["b_t"] = bson.M{
		"$gt": req.StartTime,
	}
	q["e_t"] = bson.M{
		"$lt": req.EndTime,
	}
	if req.AlbumId > 0 {
		if req.Level > 0 {
			q["aid"] = req.AlbumId
			q["lvl"] = req.Level
		} else {
			q["aid"] = req.AlbumId
		}
	}
	if req.Level > 0 {
		q["lvl"] = req.Level
	}
	if req.Tag > 0 {
		q["tag"] = req.Tag
	}

	conf.Logger.Debug("start query", "q", q)
	ch := make(chan []*distdao.DistModel, 1)
	go getDistList(bson.M(q), req, ch)

	//total, errIgnore := distdao.GetDistProcessListCount(bson.M(q))
	//if errIgnore != nil {
	//	conf.Logger.Error("fail to get dist process list count", "q", q)
	//	//return
	//}
	total := DefaultMaxNum
	//获取三个队列的长度
	queuesInfo, errIgnore := distcluster.GetMultiQueueCount(MultiQueue)
	if errIgnore != nil {
		conf.Logger.Error("fail to get multi queue count", "q", q)
	}
	distList := <-ch
	//conf.Logger.Debug("get list", "list", distList)
	resp = &traffic_dist_recall_api.TrafficListResp{}
	resp.QueueInfo = queuesInfo
	resp.Total = int64(total)
	//resp.List =
	distDataList := make([]*traffic_dist_recall_api.TrafficData, 0, len(distList))
	mapIdCount := make(map[int64]int)
	for _, distData := range distList {
		if distData == nil {
			continue
		}
		reCount, errIgnore := distcluster.GetCountCluster(distData.ProfileId, distData.PriorityLevel)
		if errIgnore != nil {
			conf.Logger.Error("get count cluster err", "id", distData.ProfileId, "level", distData.PriorityLevel, "errIgnore", errIgnore)
		}
		mapIdCount[distData.AlbumId] += distData.Count
		//conf.Logger.Debug("get dist data", "data list", distDataList[i])
		distDataList = append(distDataList, &traffic_dist_recall_api.TrafficData{
			AlbumId:    distData.AlbumId,
			ProfileId:  distData.ProfileId,
			ProfileMid: distData.ProfileMid,
			//AllCount:   int64(distData.Count),
			ReCount: reCount,
			Tasks: []*traffic_dist_recall_api.TaskInfo{
				{
					Level:     distData.PriorityLevel,
					StartTime: distData.BeginTime,
					EndTime:   distData.EndTime,
					OpMid:     distData.OpMId,
					OneCount:  distData.Count,
					Tag:       distData.Tag,
					//ReCount:   distData.Count - reCount,
				},
			},
		})
	}
	sumSameCount(&distDataList, mapIdCount)
	resp.List = distDataList
	return
}

func sumSameCount(distDataList *[]*traffic_dist_recall_api.TrafficData, mapIdCount map[int64]int) {
	//for albumId, count := range mapIdCount {
	//
	//}
	for _, distData := range *distDataList {
		distData.AllCount += int64(mapIdCount[distData.AlbumId])
	}
}
func isTheSameDistData(distDataList []*traffic_dist_recall_api.TrafficData, albumId, mid int64) (i int) {
	for i, _ := range distDataList {
		if distDataList[i] == nil {
			continue
		}
		if distDataList[i].AlbumId == albumId && distDataList[i].ProfileMid == mid {
			return i
		}
	}
	return -1
}

func getDistList(q bson.M, req *traffic_dist_recall_api.TrafficListReq, ch chan []*distdao.DistModel) {

	listDistData, err := distdao.GetDistProcessList(q, req)
	if err != nil {
		conf.Logger.Error("fail to get dist process dist", "q", q)
		listDistData = []*distdao.DistModel{}
		ch <- listDistData
		return
	}
	ch <- listDistData
}

// getIdsFromMultiQueue 获取需要分发的ids从多级队列里面
func getIdsFromMultiQueue(limit int, expValue string) (ids []int64, level traffic_dist_recall_api.LevelType, err error) {
	//获取1-100的随机数
	//rand.Seed(time.Now().UnixNano())
	//min := 1
	//max := 100

	//等于这个才正常
	if expValue == ExpCanDispatchValue || HitExp(expValue) {
		//randNum := rand.Intn(max-min) + min
		//_ = randNum

		ids, _ = distcluster.GetDistIdsCluster(traffic_dist_recall_api.SelfDefineLevel, limit)
		level = traffic_dist_recall_api.SelfDefineLevel
		//if err != nil {
		//	return
		//}
		if len(ids) > 0 { //ids为空的话
			return
		}

		ids, err = distcluster.GetDistIdsCluster(traffic_dist_recall_api.HighQualityLevel, limit)
		level = traffic_dist_recall_api.HighQualityLevel
		if err != nil {
			return
		}
		if len(ids) > 0 { //ids为空的话，去低优先级队列取
			return
		}
		//1-50 51-80 81-100
		//switch {
		//case randNum <= 50: //50%
		//	ids, err = distcluster.GetDistIdsCluster(traffic_dist_recall_api.HighQualityLevel, limit)
		//	level = traffic_dist_recall_api.HighQualityLevel
		//	if err != nil {
		//		return
		//	}
		//	if len(ids) > 0 { //ids为空的话，去低优先级队列取
		//		return
		//	}
		//	fallthrough
		//case randNum <= 80: //30%
		//	ids, err = distcluster.GetDistIdsCluster(traffic_dist_recall_api.SelfDefineLevel, limit)
		//	level = traffic_dist_recall_api.SelfDefineLevel
		//	if err != nil {
		//		return
		//	}
		//	if len(ids) > 0 {
		//		return
		//	}
		//	fallthrough
		//case randNum <= 100: //20%
		//	level = traffic_dist_recall_api.NewPublishLevel
		//	ids, err = distcluster.GetDistIdsCluster(traffic_dist_recall_api.NewPublishLevel, limit)
		//	if err != nil {
		//		return
		//	}
		//}
	} else { //否则只看自定义有无，无就不分发
		ids, err = distcluster.GetDistIdsCluster(traffic_dist_recall_api.SelfDefineLevel, limit)
		level = traffic_dist_recall_api.SelfDefineLevel
		if err != nil {
			return
		}
	}
	return
}

// BingoHistory 过滤浏览历史
func BingoHistory(id int64, kind bizsrv_api.KindID, historys []*traffic_dist_recall_api.HistoryIdType) (yes bool) {
	idEncode := int64(bizsrv_api.ID(id).Encode(kind))

	for idx, _ := range historys {
		if historys[idx] == nil {
			continue
		}
		ids := *historys[idx]
		conf.Logger.Debug("bingo hisotry", "id", id, "ids", ids)
		i := sort.Search(len(ids), func(n int) bool { return ids[n] <= idEncode })
		if i != len(ids) && ids[i] == idEncode {
			yes = true
			return
		}
	}
	return
}

// canDist 判断是否要分发给这个人
func canDist(req *traffic_dist_recall_api.TrafficGetReq) (isDist bool, exp string) {
	redisKey := getCanDist(req)
	ok, err := distcluster.IsExists(redisKey)
	if err != nil {
		conf.Logger.Error("redis is can dist error", "Key", redisKey, "err", err)
		ok = true
	}
	//如果不在就可以发
	if !ok {
		isDist = true
	}
	//val, ok := conf.LocalCache.Get(localKey)
	//如果存在就不发 实验就是default
	//if ok {
	//	switch result := val.(type) {
	//	case bool:
	//		ok = result
	//		return
	//	default:
	//		conf.Logger.Error("unknown type", "val", val, "result", result, "localKey", localKey)
	//	}
	//}
	// 获取实验和动态配置
	exp, err = abtest.GetDispatchABtestExpValue(req)
	if err != nil {
		exp = "default"
		conf.Logger.ErrorW("abtest.GetDispatchABtestExpValue err", map[string]interface{}{"err": err})
		return
	}
	conf.Logger.Debug("get abtest exp", "req", req, "exp", exp)
	//if exp == "dispatch_disable" || exp == "default" {
	//	//ok = false
	//	return
	//}
	//handler, err := apollo.GetConfigHandler(apollo.ApolloConfTypeDispatch)
	//if err != nil {
	//	conf.Logger.ErrorW("apollo.GetConfigHandler err", map[string]interface{}{"err": err})
	//	return
	//}
	//cfg, err := handler.Dispatch().GetConfigByKey(exp)
	//if err != nil {
	//	conf.Logger.ErrorW("apollo.GetConfigByKey err", map[string]interface{}{"err": err})
	//	return
	//}
	//
	//if cfg == nil {
	//	return
	//}

	return
}

// setCanDist 设置缓存，1h内不再拉到分发的id
func setCanDist(req *traffic_dist_recall_api.TrafficGetReq) (ok bool) {
	if conf.Env == lib.PROD {
		redisKey := getCanDist(req)
		if err := distcluster.SetCanDistKeyToRedis(redisKey); err != nil {
			conf.Logger.Error("fail to set can dist key", "key", redisKey, "err", err)
		}
	}
	return
}

// getCanDist 获取缓存key
func getCanDist(req *traffic_dist_recall_api.TrafficGetReq) string {
	return fmt.Sprintf("can:dist:%d:%d", req.Mid, req.Kind)
}

// getCacheKey 获取本地缓存key
func getCacheKey() string {
	return fmt.Sprintf("traffic:ids:%d:%d", offset, limit)
}

// SetEndTime 设置id的分发结束时间
func SetEndTime(id int64, level traffic_dist_recall_api.LevelType) {
	endTime := time.Now().UnixNano() / 1e6
	if errIgnore := distdao.SetDistEndTime(id, level, endTime); errIgnore != nil {
		conf.Logger.Error("fail to set dist endtime", "id", id, "level", level, "endTime", endTime, "errIgnore", errIgnore)
		return
	}
	return
}

// HitExp 是否命中实验
func HitExp(expValue string) bool {
	for _, val := range ExpCanDispatchValueSlot {
		if val == expValue {
			return true
		}
	}
	return false
}
