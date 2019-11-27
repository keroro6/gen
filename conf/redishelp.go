package conf

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"time"
)

type RedisConf struct {
	Addr         string `mapstructure:"addr"`
	Auth         string `mapstructure:"auth"`
	Db           int    `mapstructure:"db"`
	PoolSize     int    `mapstructure:"poolSize"`
	DialTimeout  int    `mapstructure:"dialTimeout"`
	ReadTimeout  int    `mapstructure:"readTimeout"`
	WriteTimeout int    `mapstructure:"writeTimeout"`
}

const (
	DefaultPoolName = "default"
)

func GetRedisMap(confMap map[string]RedisConf) (RDS map[string]*redis.Pool) {

	RDS = make(map[string]*redis.Pool)
	for dbName, r := range confMap {
		r := r
		RDS[dbName] = &redis.Pool{
			MaxIdle:     30,
			IdleTimeout: time.Minute,
			Dial: func() (conn redis.Conn, err error) {
				conn, err = redis.Dial("tcp", r.Addr,
					redis.DialReadTimeout(time.Duration(int64(r.ReadTimeout)*int64(time.Millisecond))),
					redis.DialConnectTimeout(time.Duration(int64(r.DialTimeout)*int64(time.Millisecond))),
					redis.DialWriteTimeout(time.Duration(int64(r.WriteTimeout)*int64(time.Millisecond))),
				)
				if err != nil {
					panic(fmt.Sprintf("redis.Dial err: %v, req: %v", err, r.Addr))
					return
				}

				if auth := r.Auth; auth != "" {
					if _, err = conn.Do("AUTH", auth); err != nil {
						panic(fmt.Sprintf("redis AUTH err: %v, req: %v,%v", err, r.Addr, auth))
						conn.Close()
						return
					}
				}
				if db := r.Db; db > 0 {
					if _, err = conn.Do("SELECT", db); err != nil {
						panic(fmt.Sprintf("redis SELECT err: %v, req: %v,%v", err, r.Addr, db))
						conn.Close()
						return
					}
				}
				return
			},
		}
	}
	return
}

func GetDefaultRedisPool() *redis.Pool {
	return RDS[DefaultPoolName]
}
