package apollo

import (
	"errors"
	"fmt"
)

const (
	ApolloConfTypeDispatch = "dispatch" // 分发层
)

type DispatchConf struct {
	Type string `json:"type" binding:"required" yaml:"type" mapstructure:"type" `
}

type DispatchConfigHandler ConfigHandler

func (h *ConfigHandler) Dispatch() *DispatchConfigHandler {
	return (*DispatchConfigHandler)(h)
}

func (h *DispatchConfigHandler) GetConfigByKey(key string) (data *DispatchConf, err error) {
	if h == nil {
		return nil, errors.New(fmt.Sprintf("GetConfigByKey apollo xconf init fail %s: ", key))
	}
	cfgMap := map[string]*DispatchConf{}
	err = h.Unmarshal(&cfgMap)
	if err != nil {
		return nil, err
	}

	data, ok := cfgMap[key]
	if !ok {
		return nil, errors.New("GetConfigByKey not find key value")
	}
	return data, nil
}
