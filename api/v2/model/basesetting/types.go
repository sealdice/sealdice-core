package basesetting

import (
	"github.com/danielgtaylor/huma/v2"

	"sealdice-core/model/common/request"
)

type BaseSettingOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type BaseSettingNote struct {
	Tone  string   `json:"tone"`
	Lines []string `json:"lines"`
}

type BaseSettingFieldSchema struct {
	ID               string              `json:"id"`
	Key              string              `json:"key,omitempty"`
	Keys             []string            `json:"keys,omitempty"`
	Label            string              `json:"label"`
	Kind             string              `json:"kind"`
	Hint             string              `json:"hint,omitempty"`
	Keywords         []string            `json:"keywords,omitempty"`
	Placeholder      string              `json:"placeholder,omitempty"`
	Sensitive        bool                `json:"sensitive,omitempty"`
	Readonly         bool                `json:"readonly,omitempty"`
	Options          []*BaseSettingOption `json:"options,omitempty"`
	ConfirmMessage   string              `json:"confirmMessage,omitempty"`
	AllowCustomValue bool                `json:"allowCustomValue,omitempty"`
}

type BaseSettingGroupSchema struct {
	ID              string                   `json:"id"`
	Title           string                   `json:"title"`
	Description     string                   `json:"description,omitempty"`
	Collapsible     bool                     `json:"collapsible,omitempty"`
	DefaultExpanded bool                     `json:"defaultExpanded,omitempty"`
	Notes           []*BaseSettingNote       `json:"notes,omitempty"`
	Fields          []*BaseSettingFieldSchema `json:"fields"`
}

type BaseSettingTabSchema struct {
	ID          string                    `json:"id"`
	Title       string                    `json:"title"`
	Description string                    `json:"description,omitempty"`
	Groups      []*BaseSettingGroupSchema `json:"groups"`
}

type BaseSettingSchemaResp struct {
	Tabs []*BaseSettingTabSchema `json:"tabs"`
}

type BaseSettingExtDefaultSettingItem struct {
	Name            string          `json:"name"`
	AutoActive      bool            `json:"autoActive"`
	DisabledCommand map[string]bool `json:"disabledCommand"`
	Loaded          bool            `json:"loaded"`
}

type BaseSettingValueResp struct {
	CommandPrefix         []string                     `json:"commandPrefix"`
	DiceMasters           []string                     `json:"diceMasters"`
	NoticeIds             []string                     `json:"noticeIds"`
	MasterUnlockCode      string                       `json:"masterUnlockCode"`
	UIPassword            string                       `json:"uiPassword"`
	MailEnable            bool                         `json:"mailEnable"`
	MailFrom              string                       `json:"mailFrom"`
	MailPassword          string                       `json:"mailPassword"`
	MailSmtp              string                       `json:"mailSmtp"`
	TrustOnlyMode         bool                         `json:"trustOnlyMode"`
	BotExtFreeSwitch      bool                         `json:"botExtFreeSwitch"`
	QQEnablePoke          bool                         `json:"QQEnablePoke"`
	TextCmdTrustOnly      bool                         `json:"textCmdTrustOnly"`
	IgnoreUnaddressedBotCmd bool                       `json:"ignoreUnaddressedBotCmd"`
	AliveNoticeEnable     bool                         `json:"aliveNoticeEnable"`
	AliveNoticeValue      string                       `json:"aliveNoticeValue"`
	LogSizeNoticeEnable   bool                         `json:"logSizeNoticeEnable"`
	LogSizeNoticeCount    int                          `json:"logSizeNoticeCount"`
	PlayerNameWrapEnable  bool                         `json:"playerNameWrapEnable"`
	OnlyLogCommandInGroup bool                         `json:"onlyLogCommandInGroup"`
	OnlyLogCommandInPrivate bool                       `json:"onlyLogCommandInPrivate"`
	RateLimitEnabled      bool                         `json:"rateLimitEnabled"`
	PersonalReplenishRate string                       `json:"personalReplenishRate"`
	PersonalBurst         int64                        `json:"personalBurst"`
	GroupReplenishRate    string                       `json:"groupReplenishRate"`
	GroupBurst            int64                        `json:"groupBurst"`
	ServeAddress          string                       `json:"serveAddress"`
	RefuseGroupInvite     bool                         `json:"refuseGroupInvite"`
	FriendAddComment      string                       `json:"friendAddComment"`
	WorkInQQChannel       bool                         `json:"workInQQChannel"`
	QQChannelAutoOn       bool                         `json:"QQChannelAutoOn"`
	QQChannelLogMessage   bool                         `json:"QQChannelLogMessage"`
	DefaultCocRuleIndex   string                       `json:"defaultCocRuleIndex"`
	MaxCocCardGen         string                       `json:"maxCocCardGen"`
	MaxExecuteTime        string                       `json:"maxExecuteTime"`
	MessageDelayRangeStart float64                     `json:"messageDelayRangeStart"`
	MessageDelayRangeEnd   float64                     `json:"messageDelayRangeEnd"`
	QuitInactiveThreshold  float64                     `json:"quitInactiveThreshold"`
	QuitInactiveBatchSize  int64                       `json:"quitInactiveBatchSize"`
	QuitInactiveBatchWait  int64                       `json:"quitInactiveBatchWait"`
	ExtDefaultSettings    []*BaseSettingExtDefaultSettingItem `json:"extDefaultSettings"`
}

type BaseSettingUpdateReq struct {
	Body request.RequestWrapper[map[string]interface{}] `json:"body"`
}

type BaseSettingActionResp struct {
	Success bool   `json:"success"`
	Err     string `json:"err,omitempty"`
}

type BaseSettingUpgradeForm struct {
	File huma.FormFile `form:"file" required:"true"`
}

type BaseSettingUpgradeReq struct {
	RawBody huma.MultipartFormFiles[BaseSettingUpgradeForm]
}
