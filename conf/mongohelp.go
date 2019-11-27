package conf

import (
	"log"
)

type XMongoMap map[string]*mgo.Session

type MongoConf struct {
	Dsn string `mapstructure:"dsn"`
}

func GetMongoMap(confMap map[string]MongoConf) XMongoMap {
	xMongoMap := XMongoMap{}
	for db, m := range confMap {
		session, err := mgo.Dial(m.Dsn)
		if err != nil {

			log.Fatalf("mongo [dsn: %s]  load fail: %v", m.Dsn, err)
		}
		session.SetMode(mgo.SecondaryPreferred, true)
		xMongoMap[db] = session
	}
	return xMongoMap
}
