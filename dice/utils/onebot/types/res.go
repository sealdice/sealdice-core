package types

import "sealdice-core/dice/utils/onebot/schema"

type SendMsgRes struct {
	MessageId int `json:"message_id"`
}

type GetMsgRes struct {
	Time        int              `json:"time"`
	MessageType string           `json:"message_type"`
	MessageId   int              `json:"message_id"`
	RealId      int              `json:"real_id"`
	Sender      schema.Sender    `json:"sender"`
	Message     []schema.Message `json:"message"`
}

type LoginInfo struct {
	UserId   int64  `json:"user_id"`
	NickName string `json:"nickname"`
}

type StrangerInfo struct {
	UserId   int64  `json:"user_id"`
	NickName string `json:"nickname"`
	Sex      string `json:"sex"`
	Age      int    `json:"age"`
}

type Status struct {
	Online bool `json:"online"`
	Good   bool `json:"good"`
}

type VersionInfo struct {
	AppName         string `json:"app_name"`
	ProtocolVersion string `json:"protocol_version"`
	AppVersion      string `json:"app_version"`
}
