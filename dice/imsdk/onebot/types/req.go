package types

import "sealdice-core/dice/imsdk/onebot/schema"

type SendPrivateMsgReq struct {
	UserId  int64            `json:"user_id"`
	Message []schema.Message `json:"message"`
}

type SendGrMsgReq struct {
	GroupId int64            `json:"group_id"`
	Message []schema.Message `json:"message"`
}

type GetStrangerInfo struct {
	UserId  int64 `json:"user_id"`
	NoCache bool  `json:"no_cache"`
}

type GetMsgReq struct {
	MessageId int `json:"message_id"`
}

type DelMsgReq struct {
	MessageId int `json:"message_id"`
}

type FriendAddReq struct {
	Flag    string `json:"flag"`
	Remark  string `json:"remark"`
	Approve bool   `json:"approve"`
}

type GroupAddReq struct {
	Flag    string `json:"flag"`
	SubType string `json:"sub_type"`
	Approve bool   `json:"approve"`
	Reason  string `json:"reason"`
}

type SpecialTitleReq struct {
	GroupId      int64  `json:"group_id"`
	UserId       int64  `json:"user_id"`
	SpecialTitle string `json:"special_title"`
}

// QuitGroupReq 退群逻辑
type QuitGroupReq struct {
	GroupId int64 `json:"group_id"`
}

// SetGroupCardReq 设置群名片
type SetGroupCardReq struct {
	GroupId int64  `json:"group_id"`
	UserId  int64  `json:"user_id"`
	Card    string `json:"card"`
}

// GetGroupInfoReq 获取群信息
type GetGroupInfoReq struct {
	GroupId int64 `json:"group_id"`
	NoCache bool  `json:"no_cache"`
}

// GetGroupMemberInfoReq 获取群成员信息
type GetGroupMemberInfoReq struct {
	GroupId int64 `json:"group_id"`
	UserId  int64 `json:"user_id"`
	NoCache bool  `json:"no_cache"`
}
