package group

import "sealdice-core/model/common/request"

type GroupFilter struct {
	Platform        []string `json:"platforms" form:"platforms" required:"false"`
	OrderByLastTime bool     `json:"orderByLastTime" form:"orderByLastTime" required:"false"`
	UnusedDays      int      `json:"queryUnusedDays" form:"queryUnusedDays" required:"false"`
	IsLogging       bool     `json:"isLogging" form:"isLogging" required:"false"`
}

type GroupPageRequest struct {
	request.PageInfo
	Filter GroupFilter `json:"filter" required:"false"`
}

type GroupModifyRequest struct {
	Active  bool   `json:"active"`
	GroupID string `json:"groupId"`
}

type QuitGroupRequest struct {
	GroupID   string `json:"groupId"`
	DiceID    string `json:"diceId,omitempty"`
	Silence   bool   `json:"silence"`
	ExtraText string `json:"extraText"`
}

type BatchQuitGroupRequest struct {
	GroupIDs  []string `json:"groupIds"`
	Silence   bool     `json:"silence"`
	ExtraText string   `json:"extraText"`
}

type BatchNotifyGroupRequest struct {
	GroupIDs []string `json:"groupIds"`
	Text     string   `json:"text"`
}

type GroupPageReq struct {
	Body GroupPageRequest
}

type ModifyGroupReq struct {
	Body GroupModifyRequest
}

type QuitGroupReq struct {
	Body QuitGroupRequest
}

type BatchQuitGroupReq struct {
	Body BatchQuitGroupRequest
}

type BatchNotifyGroupReq struct {
	Body BatchNotifyGroupRequest
}
