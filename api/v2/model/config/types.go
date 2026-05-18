package config

import (
	configm "sealdice-core/dice"
	"sealdice-core/model/common/request"
)

type ReplyModuleConfig struct {
	CustomReplyConfigEnable bool `json:"customReplyConfigEnable"`
}

type ReplyConfigReq struct {
	Body request.RequestWrapper[ReplyModuleConfig] `json:"body"`
}

type AdvancedConfigReq struct {
	Body request.RequestWrapper[configm.AdvancedConfig] `json:"body"`
}
