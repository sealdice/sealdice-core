package types

import "sealdice-core/dice/utils/onebot/schema"

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
