package types

// fork from https://github.com/nsxdevx/nsxbot

import "sealdice-core/dice/imsdk/onebot/schema"

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

type GroupInfo struct {
	GroupId        int64  `json:"group_id"`         // 群号
	GroupName      string `json:"group_name"`       // 群名
	GroupAllShut   int    `json:"group_all_shut"`   // 是否全员禁言
	MemberCount    int    `json:"member_count"`     // 群成员数量
	MaxMemberCount int    `json:"max_member_count"` // 最大成员数量
	GroupRemark    string `json:"group_remark"`     // 群备注
}

type GroupMemberInfo struct {
	GroupId         int64  `json:"group_id"`
	UserId          int64  `json:"user_id"`
	Nickname        string `json:"nickname"`
	Card            string `json:"card"`
	Sex             string `json:"sex"`
	Age             int    `json:"age"`
	Area            string `json:"area"`
	JoinTime        int64  `json:"join_time"`
	LastSentTime    int64  `json:"last_sent_time"`
	Level           string `json:"level"`
	Role            string `json:"role"`
	Unfriendly      bool   `json:"unfriendly"`
	Title           string `json:"title"`
	TitleExpireTime int64  `json:"title_expire_time"`
	CardChangeable  bool   `json:"card_changeable"`
	ShutUpTimestamp int64  `json:"shut_up_timestamp"`
}
