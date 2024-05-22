package api

import (
	"sealdice-core/dice"
)

type apiPluginConfig struct {
	PluginName string             `json:"pluginName"`
	Configs    []*dice.ConfigItem `json:"configs" jsbind:"configs"`
}

type getConfigResp map[string]*apiPluginConfig

type setConfigReq map[string]*apiPluginConfig
