package tooltest

import (
	"sealdice-core/model/common/response"
)

const (
	UITestRoleOwner       = "owner"
	UITestRoleAdmin       = "admin"
	UITestRoleInviter     = "inviter"
	UITestRoleMaster      = "master"
	UITestRoleMember      = "member"
	UITestRoleBlacklisted = "blacklisted"
)

type PostMessageReqBody struct {
	Text            string `json:"text"`
	Mode            string `json:"mode"`
	SenderID        string `json:"senderId,omitempty"`
	GroupID         string `json:"groupId,omitempty"`
	MessageSplitLen *int   `json:"messageSplitLen,omitempty"`
}

type PostMessageReq struct {
	Body PostMessageReqBody `json:"body"`
}

type ContextReq struct {
	Mode     string `query:"mode"`
	SenderID string `query:"senderId"`
	GroupID  string `query:"groupId"`
}

type ProfileItem struct {
	UserID    string `json:"userId"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	AvatarKey string `json:"avatarKey"`
	Enabled   bool   `json:"enabled"`
	IsBot     bool   `json:"isBot"`
}

type ContextResp struct {
	Mode            string         `json:"mode"`
	ConversationID  string         `json:"conversationId"`
	GroupID         string         `json:"groupId,omitempty"`
	GroupName       string         `json:"groupName"`
	GroupAccess     string         `json:"groupAccess,omitempty"`
	CurrentSenderID string         `json:"currentSenderId"`
	Members         []*ProfileItem `json:"members"`
	BotName         string         `json:"botName"`
	BotAvatarKey    string         `json:"botAvatarKey"`
	CommandPrefix   []string       `json:"commandPrefix"`
}

type UpdateProfileReq struct {
	Body struct {
		Mode      string `json:"mode"`
		GroupID   string `json:"groupId,omitempty"`
		UserID    string `json:"userId"`
		Name      string `json:"name"`
		Role      string `json:"role"`
		AvatarKey string `json:"avatarKey"`
		Enabled   *bool  `json:"enabled,omitempty"`
	} `json:"body"`
}

type UpdateContextReq struct {
	Body struct {
		GroupID     string `json:"groupId"`
		GroupName   string `json:"groupName"`
		GroupAccess string `json:"groupAccess"`
	} `json:"body"`
}

type MessageItem struct {
	UID         string `json:"uid"`
	Message     string `json:"message"`
	MessageType string `json:"messageType"`
}

type PendingMessagesResp struct {
	Items []MessageItem `json:"items"`
}

type CommandsResp struct {
	Items []*CommandOption `json:"items"`
}

type CommandOption struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source,omitempty"`
}

type SplitOption struct {
	Key             string `json:"key"`
	Label           string `json:"label"`
	MessageSplitLen int    `json:"messageSplitLen"`
}

type SplitOptionsResp struct {
	DefaultKey string         `json:"defaultKey"`
	Options    []*SplitOption `json:"options"`
}

type PendingMessagesItemResponse = response.ItemResponse[PendingMessagesResp]
type CommandsItemResponse = response.ItemResponse[CommandsResp]
type SplitOptionsItemResponse = response.ItemResponse[SplitOptionsResp]
type SimpleItemResponse = response.ItemResponse[response.SimpleOK]
type ContextItemResponse = response.ItemResponse[ContextResp]
