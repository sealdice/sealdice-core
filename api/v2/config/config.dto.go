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

type PublicDiceConfig struct {
	PublicDiceEnable bool   `json:"publicDiceEnable"`
	PublicDiceID     string `json:"publicDiceId"`
	PublicDiceName   string `json:"publicDiceName"`
	PublicDiceBrief  string `json:"publicDiceBrief"`
	PublicDiceNote   string `json:"publicDiceNote"`
	PublicDiceAvatar string `json:"publicDiceAvatar"`
}

type PublicDiceEndpointItem struct {
	ID           string             `json:"id"`
	UserID       string             `json:"userId"`
	Platform     string             `json:"platform"`
	ProtocolType string             `json:"protocolType"`
	State        dice.EndpointState `json:"state"`
	IsPublic     bool               `json:"isPublic"`
}

type PublicDiceInfoResp struct {
	Config    PublicDiceConfig         `json:"config"`
	Endpoints []PublicDiceEndpointItem `json:"endpoints"`
}

type PublicDiceUpdateBody struct {
	Config              PublicDiceConfig `json:"config"`
	SelectedEndpointIDs []string         `json:"selectedEndpointIds"`
}

type PublicDiceUpdateReq struct {
	Body PublicDiceUpdateBody `json:"body"`
}
