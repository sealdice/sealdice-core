package dice

import (
	"time"

	"golang.org/x/time/rate"

	"sealdice-core/dice/censor"
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
		MessageDelayRangeStart:  0.0,
		MessageDelayRangeEnd:    0.4,
		WorkInQQChannel:         true,
		QQChannelAutoOn:         false,
		QQChannelLogMessage:     false,
		QQEnablePoke:            true,
		TextCmdTrustOnly:        true,
		IgnoreUnaddressedBotCmd: false,
		UILogLimit:              0,
		FriendAddComment:        "",
		CustomReplyConfigEnable: false,
		AutoReloginEnable:       false,
		RefuseGroupInvite:       false,
		UpgradeWindowID:         "",
		UpgradeEndpointID:       "",
		BotExtFreeSwitch:        false,
		BotExitWithoutAt:        false,
		TrustOnlyMode:           false,
		AliveNoticeEnable:       false,
		AliveNoticeValue:        "@every 3h",
		ReplyDebugMode:          false,
		PlayerNameWrapEnable:    true,
		VMVersionForReply:       "v1",
		VMVersionForDeck:        "v2",
		VMVersionForCustomText:  "v2",
		VMVersionForMsg:         "v2",
		Name:                    "default",
		DataDir:                 "data/default",
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
		QuitInactiveBatchSize: 10,
		QuitInactiveBatchWait: 30,
	},
	ExtConfig{
		DefaultCocRuleIndex: 0,
		MaxExecuteTime:      12,
		MaxCocCardGen:       5,
		CocCardMergeForward: false,
		ExtDefaultSettings:  make([]*ExtDefaultSettingItem, 0),
	},
	BanConfig{
		BanList: nil,
	},
	JsConfig{
		JsEnable:          true,
		DisabledJsScripts: make(map[string]bool),
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
		CensorThresholds:     make(map[censor.Level]int),
		CensorHandlers:       make(map[censor.Level]uint8),
		CensorScores:         make(map[censor.Level]int),
		CensorCaseSensitive:  false,
		CensorMatchPinyin:    false,
		CensorFilterRegexStr: "",
	},
	PublicDiceConfig{
		Enable: false,
	},
	StoreConfig{
		BackendUrls: []string{},
	},
	DirtyConfig{
		DeckList: nil,
		CommandPrefix: []string{
			"!",
			".",
			"。",
			"/",
		},
		DiceMasters: []string{"UI:1001"},
	},
}
