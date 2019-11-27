package apollo

import (
	"errors"
	"fmt"
	"xgit.xiaoniangao.cn/xngo/lib/xconf"
)

var ConfigMap map[ApolloConfType]*ConfigHandler

func InitConfigHandler(xconfs map[string]xconf.XParam) error {

	ConfigMap = make(map[ApolloConfType]*ConfigHandler)
	for key, value := range xconfs {
		configHandler, err := NewConfigHandler(value)
		if err != nil {
			return errors.New(fmt.Sprintf("InitRecallApolloConfig fail key : %s, err: %s", key, err.Error()))
		} else {
			//conf.Logger.Debug("init config handler", "key", key)
			ConfigMap[ApolloConfType(key)] = configHandler
		}
	}
	return nil
}

func GetConfigHandler(key ApolloConfType) (*ConfigHandler, error) {
	configHandler, ok := ConfigMap[key]
	if ok {
		return configHandler, nil
	}
	return nil, errors.New("This Handler is not init")
}
