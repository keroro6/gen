package abtest

import (
	"strconv"
	"xgit.xiaoniangao.cn/recsys/traffic_dist_recall/conf"
	"xgit.xiaoniangao.cn/recsys/traffic_dist_recall_api"

	"github.com/garyburd/redigo/redis"
	"xgit.xiaoniangao.cn/recsys/xng.abtest.client/abtest"
)

var AbTestClient *abtest.Client

func InitAbtestClient(redisPool *redis.Pool) {
	AbTestClient = abtest.DefaultClient(redisPool)
}

func GetDispatchABtestExpValue(getReq *traffic_dist_recall_api.TrafficGetReq) (string, error) {
	value, err := AbTestClient.GetValueByExpNameAndSlotId(conf.C.Experiments.Dispatch,
		strconv.FormatInt(getReq.Mid, 10), map[string]interface{}{})
	if err != nil {
		conf.Logger.ErrorW("GetDispatchABtestExpValue err",
			map[string]interface{}{"err": err, "mid": getReq.Mid})
		return "", err
	}
	return value, nil
}
