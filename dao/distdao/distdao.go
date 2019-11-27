package distdao

import (
	"fmt"
	"time"
	"xgit.xiaoniangao.cn/recsys/traffic_dist_recall/conf"
	"xgit.xiaoniangao.cn/recsys/traffic_dist_recall_api"
	"xgit.xiaoniangao.cn/xngo/lib/github.com.globalsign.mgo"
	"xgit.xiaoniangao.cn/xngo/lib/github.com.globalsign.mgo/bson"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
	"xgit.xiaoniangao.cn/xngo/lib/xmongo"
)

var Dao *xmongo.Client

//@dbName
const (

	DistDBName                  = "traffic_dist"
	//@colName
	DistTableName               = "traffic_dist_0"
	GetPriorityLevelByIdTimeOut = time.Minute * 20
	//SortAsceInStartTime
	//SortAsceInEndTime
	//SortDescInStartTime
	//SortDescInEndTime
)

var OrderWay = map[int]string{
	1: "-b_t",
	2: "-e_t",
	3: "b_t",
	4: "e_t",
}

// DistModel 定义流量分发记录结构
type DistModel struct {
	ProfileId     int64                             `bson:"id"`            //动态id
	AlbumId       int64                             `bson:"aid"`           //影集id
	ProfileMid    int64                             `bson:"mid"`           //动态mid
	OpMId         int64                             `bson:"op_mid"`        //op mid
	Count         int                               `bson:"cnt"`           // 分发的数量
	Tag           traffic_dist_recall_api.DistTag   `bson:"tag,omitempty"` //什么标签，原创等
	PriorityLevel traffic_dist_recall_api.LevelType `bson:"lvl"`           //优先级
	BeginTime     int64                             `bson:"b_t"`           //开始时间
	EndTime       int64                             `bson:"e_t"`           //结束时间
}

type IsDistModel struct {
	ProfileId     int64                             `bson:"id"`  //动态id
	PriorityLevel traffic_dist_recall_api.LevelType `bson:"lvl"` //优先级
}

func init() {
	var err error
	Dao, err = xmongo.NewClient(conf.DBS[DistDBName], DistDBName, DistTableName)
	if err != nil {
		xlog.Fatal("Create %s Xmongo Client Failed: %v", DistDBName, err)
	}
}

// AddDistRecord 增加一条分发记录,多个id会同时加多条，但是开始时间不一样
func AddDistRecord(distModel *DistModel) (err error) {

	//q := bson.M{"id": distModel.ProfileMid}

	err = Dao.Insert(distModel)
	return
}

// SetDistEndTime 设置分发结束时间，redis count会累加，设置最终的结束时间
func SetDistEndTime(id int64, level traffic_dist_recall_api.LevelType, endTime int64) (err error) {
	q := bson.M{"id": id, "lvl": level, "e_t": 0}
	up := bson.M{
		"$set": bson.M{
			"e_t": endTime,
		},
	}
	_, err = Dao.UpdateAll(q, up)
	return
}

// GetPriorityLevelById 根据动态id获取level
func GetPriorityLevelById(id int64) (distModel *IsDistModel, err error) {

	localCacheKey := fmt.Sprintf("get_level_by_id:%d", id)
	val, ok := conf.LocalCache.Get(localCacheKey)
	if ok {
		switch result := val.(type) {
		case *IsDistModel:
			distModel = result
			return
		default:
			conf.Logger.Error("unknown get level by id type", "val", val, "result", result, "local key", localCacheKey)
		}
	}
	q := bson.M{"id": id}
	err = Dao.FindOne(q, &distModel, nil)
	if err != nil {
		if err == mgo.ErrNotFound {
			err = nil
		}
		return
	}
	if distModel == nil {
		return
	}

	conf.LocalCache.Set(localCacheKey, distModel, GetPriorityLevelByIdTimeOut)
	return
}

// GetDistProcessList 获取分发进度列表
func GetDistProcessList(q bson.M, req *traffic_dist_recall_api.TrafficListReq) (distList []*DistModel, err error) {
	s, c := Dao.GetSessionAndCollection()
	defer s.Close()
	err = c.Find(q).Sort(OrderWay[req.Order]).Skip(req.Offset).Limit(req.Limit).All(&distList)
	if err != nil {
		if err == mgo.ErrNotFound {
			err = nil
			return
		}
	}
	return
}

// GetDistProcessListCount 获取分发查询条件总count数
func GetDistProcessListCount(q bson.M) (n int, err error) {
	s, c := Dao.GetSessionAndCollection()
	defer s.Close()
	n, err = c.Find(q).Count()
	if err != nil {
		if err == mgo.ErrNotFound {
			err = nil
			return
		}
	}
	return
}
