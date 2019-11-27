/*
Package conf 用于项目基础配置。
*/
package conf

import (
	"flag"
	"fmt"
	rd "github.com/garyburd/redigo/redis"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"
	redis "xgit.xiaoniangao.cn/xngo/lib/github.com.chasex.redis-go-cluster"
	"xgit.xiaoniangao.cn/xngo/lib/xconf"
	"xgit.xiaoniangao.cn/xngo/lib/xlog"
)

type Experiments struct {
	Dispatch string `json:"dispatch" yaml:"dispatch" mapstructure:"dispatch"` // 分发层实验名
}

type Conf struct {
	Redis map[string]RedisConf `mapstructure:"redis"`

	RedisCluster map[string]RedisClusterConf `mapstructure:"redisCluster"`

	Mongo map[string]MongoConf `mapstructure:"mongo"`

	// 基本配置
	App *struct {
		Name string `mapstructure:"name"`
		Port int    `mapstructure:"port"`
	} `mapstructure:"app"`

	//日志输出
	Xlog *xlog.Config `mapstructure:"xlog"`

	LogConfig *xlog.Config `mapstructure:"logger"`

	TraceLogConfig *xlog.Config `mapstructure:"traceLog"`

	// 各种名字或addr配置
	Addrs *struct {
		Clog   string `mapstructure:"clog"`
		Bizsrv string `mapstructure:"bizsrv"`
	} `mapstructure:"addrs"`

	//LRUSize int `mapstructure:"lruSize"`
	Experiments Experiments `json:"experiments" yaml:"experiments" mapstructure:"experiments"`
	//XconfMap    map[string]xconf.XParam `json:"xConf" yaml:"xConf" mapstructure:"xConf"`
}

var (
	//全局的gin
	Gin *gin.Engine

	//全局配置
	C Conf

	// RDS 表示redis连接，key是业务名，value是redis连接
	RDS map[string]*rd.Pool

	// RDSCluster 表示redis连接，key是业务名，value是redis cluster连接
	// RDSCluster XRedisClusterMap
	RDSCluster map[string]*redis.Cluster = map[string]*redis.Cluster{}

	// DBS 表示mongo连接，key是db名，value是db连接
	DBS XMongoMap

	// 为true时只检查配置不启动服务
	CheckConfig = false

	//LRU *lru.Cache

	LocalCache *cache.Cache

	Env string
)

type RedisPool struct {
	rd.Pool
}

func initGin() {
	Gin = gin.New()
	//Gin.Use(gin.Logger())
	Gin.Use(gin.Recovery())
	//Gin.Use(xng.Boss())
}

func initConfig(env string) {
	configDir := getConfDir()
	configName := fmt.Sprintf("%s.conf.yaml", env)
	configPath := path.Join(configDir, configName)

	C = Conf{}
	err := xconf.LoadConfig(configPath, &C)
	if err != nil {
		log.Fatal(err)
	}
}

func getConfDir() string {
	dir := "conf"
	for i := 0; i < 3; i++ {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			break
		}
		dir = filepath.Join("..", dir)
	}

	return dir
}

func init() {
	var env string
	flag.StringVar(&env, "env", "dev", "set env")
	var test bool
	flag.BoolVar(&test, "test", false, "set test case flag")
	flag.BoolVar(&CheckConfig, "checkconfig", false, "check config file")
	flag.Parse()

	initConfig(env)
	initGin()

	Env = env
	//lc.Init(1e5) // lc is not concurrent safe, user github.com/hashicorp/golang-lru instead
	//var err error
	//LRU, err = lru.New(C.LRUSize)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//err := apollo.InitConfigHandler(C.XconfMap)
	//if err != nil {
	//	fmt.Println(err)
	//}
	//初始化log
	xlog.Init(C.Xlog)
	InitLogger()

	LocalCache = cache.New(5*time.Minute, 5*time.Minute)
	if LocalCache == nil {
		panic("fail to new go cahce")
	}

	//初始化mongo
	DBS = GetMongoMap(C.Mongo)

	//if Env == lib.PROD {
	RDSCluster = GetRedisClusterMap(C.RedisCluster)
	//} else {
	RDS = GetRedisMap(C.Redis)
	//}
}
