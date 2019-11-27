package apollo

import (
	"xgit.xiaoniangao.cn/xngo/lib/xconf"
)

type ApolloConfType string

const (
	ApolloConfTypeFilter = "filter" //过滤层
)

type ConfigHandler struct {
	*xconf.XConfig
}

/*
nameSpace为空 用默认值"application"
configType 为空 时默认值"properties"
*/
func NewConfigHandler(param xconf.XParam) (handel *ConfigHandler, err error) {

	xConf, err := xconf.NewWithParam(param)

	if err != nil {
		return nil, err
	}
	return &ConfigHandler{xConf}, nil
}
