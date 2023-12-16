package dice

import (
	"time"

	"golang.org/x/time/rate"
)

var DefaultConfig = Config{
	nil,
	ConfigVersion,
	BaseConfig{
		CommandCompatibleMode:   true, // 一直为true即可
		LastSavedTime:           nil,
		NoticeIDs:               []string{},
		OnlyLogCommandInGroup:   false,
		OnlyLogCommandInPrivate: false,
		VersionCode:             ConfigVersionCode,
		MessageDelayRangeStart:  0,
		MessageDelayRangeEnd:    0,
		WorkInQQChannel:         true,
		QQChannelAutoOn:         false,
		QQChannelLogMessage:     false,
		QQEnablePoke:            true,
		TextCmdTrustOnly:        true,
		IgnoreUnaddressedBotCmd: false,
		UILogLimit:              0,
		FriendAddComment:        "",
		MasterUnlockCode:        "",
		MasterUnlockCodeTime:    0,
		CustomReplyConfigEnable: false,
		CustomReplyConfig:       nil,
		AutoReloginEnable:       false,
		RefuseGroupInvite:       false,
		UpgradeWindowID:         "",
		UpgradeEndpointID:       "",
		BotExtFreeSwitch:        false,
		TrustOnlyMode:           false,
		AliveNoticeEnable:       false,
		AliveNoticeValue:        "@every 3h",
		ReplyDebugMode:          false,
		PlayerNameWrapEnable:    true,
	},
	RateLimitConfig{
		RateLimitEnabled:         false,
		PersonalReplenishRateStr: "@every 3s",
		PersonalReplenishRate:    rate.Every(time.Second * 3),
		GroupReplenishRateStr:    "@every 3s",
		GroupReplenishRate:       rate.Every(time.Second * 3),
		PersonalBurst:            3,
		GroupBurst:               3,
	},
	QuitInactiveConfig{
		QuitInactiveThreshold: 0,
		quitInactiveCronEntry: 0,
	},
	ExtConfig{
		DefaultCocRuleIndex: 0,
		MaxExecuteTime:      12,
		MaxCocCardGen:       5,
		ExtDefaultSettings:  nil,
	},
	BanConfig{
		BanList: nil,
	},
	JsConfig{
		JsEnable:          true,
		DisabledJsScripts: nil,
	},
	StoryLogConfig{
		LogSizeNoticeEnable: true,
		LogSizeNoticeCount:  500,
	},
	MailConfig{
		MailEnable:   false,
		MailFrom:     "",
		MailPassword: "",
		MailSMTP:     "",
	},
	NewsConfig{
		NewsMark: "",
	},
	CensorConfig{
		EnableCensor:         false,
		CensorMode:           0,
		CensorThresholds:     nil,
		CensorHandlers:       nil,
		CensorScores:         nil,
		CensorCaseSensitive:  false,
		CensorMatchPinyin:    false,
		CensorFilterRegexStr: "",
	},
}
