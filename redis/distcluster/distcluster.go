package distcluster

import (
	"fmt"
	"xgit.xiaoniangao.cn/recsys/traffic_dist_recall_api"
	redisCluster "xgit.xiaoniangao.cn/xngo/lib/github.com.chasex.redis-go-cluster"

	"xgit.xiaoniangao.cn/recsys/traffic_dist_recall/conf"
)

const (
	RedisNameDist      = "dist_cluster"
	TimeOutSet         = 604800 * 3  //21 天
	TimeOutSetCandDist = 60 * 60 * 1 // 1小时
	Flag               = 1
)

func getSetKey(level traffic_dist_recall_api.LevelType) string {
	return fmt.Sprintf("dist_cluster:level:%d", level)
}
func getCntKey(profileId int64, level traffic_dist_recall_api.LevelType) string {
	return fmt.Sprintf(getSetKey(level)+":cnt:%d", profileId)
}
func getMidKey(profileId int64) string {
	return fmt.Sprintf("dist_cluster:mid:%d", profileId)
}

// GetDistMidByIdCluster 根据动态id获取mid的映射
func GetDistMidByIdCluster(id int64) (mid int64, err error) {
	cluster := conf.RDSCluster[RedisNameDist]

	mid, err = redisCluster.Int64(cluster.Do("GET", getMidKey(id)))
	return
}

// UpdateDistScore 更新分数
func UpdateDistScore(id, mid int64, level traffic_dist_recall_api.LevelType, score float32) (err error) {
	cluster := conf.RDSCluster[RedisNameDist]
	setKey := getSetKey(level)

	_, err = cluster.Do("ZADD", setKey, float64(score), id)
	if err != nil {
		return
	}
	return
}

// AddDistIdCluster 加入id 并且 设置count
func AddDistIdCluster(id, mid int64, count int, level traffic_dist_recall_api.LevelType, score float32) (err error) {
	cluster := conf.RDSCluster[RedisNameDist]
	batch := cluster.NewBatch()

	setKey := getSetKey(level)
	cntKey := getCntKey(id, level)
	err = batch.Put("INCRBY", cntKey, count)
	if err != nil {
		return
	}
	err = batch.Put("SETEX", getMidKey(id), TimeOutSet, mid)
	if err != nil {
		return
	}
	err = batch.Put("ZADD", setKey, float64(score), id)
	if err != nil {
		return
	}
	_, err = cluster.RunBatch(batch)
	return
}

// GetDistIdsCluster 获取要分发的id
func GetDistIdsCluster(level traffic_dist_recall_api.LevelType, limit int) (ids []int64, err error) {
	cluster := conf.RDSCluster[RedisNameDist]
	setKey := getSetKey(level)
	ids, err = redisCluster.Int64s(cluster.Do("ZREVRANGE", setKey, 0, limit))
	return
}

// MinusCountCluster 分发的id对应的计数-1
func MinusCountCluster(id int64, level traffic_dist_recall_api.LevelType) (ret int, err error) {
	cluster := conf.RDSCluster[RedisNameDist]
	cntKey := getCntKey(id, level)
	ret, err = redisCluster.Int(cluster.Do("INCRBY", cntKey, -1))
	return
}

// GetCountCluster 获取id的count
func GetCountCluster(id int64, level traffic_dist_recall_api.LevelType) (ret int, err error) {
	cluster := conf.RDSCluster[RedisNameDist]
	cntKey := getCntKey(id, level)
	ret, err = redisCluster.Int(cluster.Do("GET", cntKey))
	return
}

// DelDistId 删除要分发的id及其计数
func DelDistIdCluster(id int64, level traffic_dist_recall_api.LevelType) (ret int, err error) {
	cluster := conf.RDSCluster[RedisNameDist]
	batch := cluster.NewBatch()

	setKey := getSetKey(level)
	cntKey := getCntKey(id, level)
	err = batch.Put("ZREM", setKey, id)
	if err != nil {
		return
	}
	err = batch.Put("DEL", cntKey)
	if err != nil {
		return
	}
	//不这样弄，还有问题，大并发情况下，可能100个拿到了id但是还没获取到mid，101个因为没有id 可拿了，然后删了就其余的都拿不到mid了，全都失败了，设置超时时间好了
	//err = batch.Put("DEL", getMidKey(id))
	reply, err := cluster.RunBatch(batch)
	if err != nil {
		return
	}
	//取第一个的返回值 认为就是它删除了
	_, err = redisCluster.Scan(reply, &ret)
	return
}

// IsDisting 是否在分发队列中
func IsDisting(id int64, level traffic_dist_recall_api.LevelType) (ok bool, err error) {
	cluster := conf.RDSCluster[RedisNameDist]
	setKey := getSetKey(level)
	ret, err := redisCluster.Int(cluster.Do("ZRANK", setKey, id))
	//肯定在
	if err == nil && ret >= 0 {
		ok = true
		return
	}
	//肯定不在
	if err == redisCluster.ErrNil {
		return
	}
	//其它错误，就认为在
	ok = true
	return
}

// GetMultiQueueCount 获取多个队列的长度
func GetMultiQueueCount(levels []traffic_dist_recall_api.LevelType) (levelCountMap map[traffic_dist_recall_api.LevelType]int, err error) {
	cluster := conf.RDSCluster[RedisNameDist]
	batch := cluster.NewBatch()
	levelCountMap = make(map[traffic_dist_recall_api.LevelType]int)
	for _, level := range levels {
		setKey := getSetKey(level)
		err = batch.Put("ZCARD", setKey)
		if err != nil {
			continue
		}
	}
	reply, err := cluster.RunBatch(batch)
	if err != nil {
		return
	}
	var ret int
	i := 0
	for range reply {
		reply, err = redisCluster.Scan(reply, &ret)
		if err != nil {
			continue
		}
		levelCountMap[levels[i]] = ret
		i++
	}
	return
}

func SetCanDistKeyToRedis(key string) (err error) {
	cluster := conf.RDSCluster[RedisNameDist]
	_, err = cluster.Do("SETEX", key, TimeOutSetCandDist, Flag)
	return
}

func IsExists(key string) (ok bool, err error) {
	cluster := conf.RDSCluster[RedisNameDist]
	ok, err = redisCluster.Bool(cluster.Do("EXISTS", key))
	return
}
