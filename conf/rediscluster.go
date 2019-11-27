package conf

import (
	"fmt"
	"time"
	redis "xgit.xiaoniangao.cn/xngo/lib/github.com.chasex.redis-go-cluster"
)

type RedisClusterConf struct {
	Addrs        []string `mapstructure:"addrs"`
	Auth         string   `mapstructure:"auth"`
	PoolSize     int      `mapstructure:"poolSize"`
	DialTimeout  int      `mapstructure:"dialTimeout"`
	ReadTimeout  int      `mapstructure:"readTimeout"`
	WriteTimeout int      `mapstructure:"writeTimeout"`
}

func GetRedisClusterMap(confMap map[string]RedisClusterConf) (clusterMap map[string]*redis.Cluster) {

	var err error
	clusterMap = make(map[string]*redis.Cluster)
	for dbName, r := range confMap {
		clusterMap[dbName], err = redis.NewCluster(
			&redis.Options{
				StartNodes:   r.Addrs,
				ConnTimeout:  time.Duration(int64(r.DialTimeout) * int64(time.Millisecond)),
				ReadTimeout:  time.Duration(int64(r.ReadTimeout) * int64(time.Millisecond)),
				WriteTimeout: time.Duration(int64(r.WriteTimeout) * int64(time.Millisecond)),
				KeepAlive:    16,
				AliveTime:    60 * time.Second,
				Auth:         r.Auth,
			})
		if err != nil {
			panic(fmt.Sprintf("redis new cluster err:%v", err))
			return
		}
	}
	return
}
