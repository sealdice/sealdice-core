package config

import "sealdice-core/dice"

type ReplyModuleConfig struct {
	CustomReplyConfigEnable bool `json:"customReplyConfigEnable"`
}

type ReplyConfigReq struct {
	Body ReplyModuleConfig `json:"body"`
}

type AdvancedConfigReq struct {
	Body dice.AdvancedConfig `json:"body"`
}
