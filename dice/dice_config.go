package dice

import (
	"sealdice-core/dice/censor"
	"time"

	"github.com/robfig/cron/v3"

	"golang.org/x/time/rate"
)

type Config struct {
	ConfigVersion int `yaml:"configVersion"` // 配置版本

	// 基础设置
	BaseConfig
	// 刷屏警告设置
	RateLimitConfig
	// 退出不活跃设置
	QuitInactiveConfig
	// 扩展设置
	ExtConfig
	// 黑名单设置
	BanConfig
	// js 设置
	JsConfig
	// 跑团日志设置
	StoryLogConfig
	// 邮件设置
	MailConfig
	// 新闻设置
	NewsConfig
	// 敏感词设置
	CensorConfig
}

type BaseConfig struct {
	ImSession               *IMSession     `yaml:"imSession" jsbind:"imSession"`
	CommandCompatibleMode   bool           `yaml:"commandCompatibleMode"`
	LastSavedTime           *time.Time     `yaml:"lastSavedTime"`
	DeckList                []*DeckInfo    `yaml:"deckList" jsbind:"deckList"`           // 牌堆信息
	CommandPrefix           []string       `yaml:"commandPrefix" jsbind:"commandPrefix"` // 指令前导
	DiceMasters             []string       `yaml:"diceMasters" jsbind:"diceMasters"`     // 骰主设置，需要格式: 平台:帐号
	NoticeIDs               []string       `yaml:"noticeIds"`                            // 通知ID
	OnlyLogCommandInGroup   bool           `yaml:"onlyLogCommandInGroup"`                // 日志中仅记录命令
	OnlyLogCommandInPrivate bool           `yaml:"onlyLogCommandInPrivate"`              // 日志中仅记录命令
	VersionCode             int            `json:"versionCode"`                          // 版本ID(配置文件)
	MessageDelayRangeStart  float64        `yaml:"messageDelayRangeStart"`               // 指令延迟区间
	MessageDelayRangeEnd    float64        `yaml:"messageDelayRangeEnd"`
	WorkInQQChannel         bool           `yaml:"workInQQChannel"`
	QQChannelAutoOn         bool           `yaml:"QQChannelAutoOn"`         // QQ频道中自动开启(默认不开)
	QQChannelLogMessage     bool           `yaml:"QQChannelLogMessage"`     // QQ频道中记录消息(默认不开)
	QQEnablePoke            bool           `yaml:"QQEnablePoke"`            // 启用戳一戳
	TextCmdTrustOnly        bool           `yaml:"textCmdTrustOnly"`        // 只允许信任用户或master使用text指令
	IgnoreUnaddressedBotCmd bool           `yaml:"ignoreUnaddressedBotCmd"` // 不响应群聊裸bot指令
	UILogLimit              int64          `yaml:"UILogLimit"`
	FriendAddComment        string         `yaml:"friendAddComment"` // 加好友验证信息
	MasterUnlockCode        string         `yaml:"-"`                // 解锁码，每20分钟变化一次，使用后立即变化
	MasterUnlockCodeTime    int64          `yaml:"-"`
	CustomReplyConfigEnable bool           `yaml:"customReplyConfigEnable"`
	CustomReplyConfig       []*ReplyConfig `yaml:"-"`
	AutoReloginEnable       bool           `yaml:"autoReloginEnable"`    // 启用自动重新登录
	RefuseGroupInvite       bool           `yaml:"refuseGroupInvite"`    // 拒绝加入新群
	UpgradeWindowID         string         `yaml:"upgradeWindowId"`      // 执行升级指令的窗口
	UpgradeEndpointID       string         `yaml:"upgradeEndpointId"`    // 执行升级指令的端点
	BotExtFreeSwitch        bool           `yaml:"botExtFreeSwitch"`     // 允许任意人员开关: 否则邀请者、群主、管理员、master有权限
	TrustOnlyMode           bool           `yaml:"trustOnlyMode"`        // 只有信任的用户/master可以拉群和使用
	AliveNoticeEnable       bool           `yaml:"aliveNoticeEnable"`    // 定时通知
	AliveNoticeValue        string         `yaml:"aliveNoticeValue"`     // 定时通知间隔
	ReplyDebugMode          bool           `yaml:"replyDebugMode"`       // 回复调试
	PlayerNameWrapEnable    bool           `yaml:"playerNameWrapEnable"` // 启用玩家名称外框
}

type RateLimitConfig struct {
	RateLimitEnabled         bool       `yaml:"rateLimitEnabled"`      // 启用频率限制 (刷屏限制)
	PersonalReplenishRateStr string     `yaml:"personalReplenishRate"` // 个人刷屏警告速率，字符串格式
	PersonalReplenishRate    rate.Limit `yaml:"-"`                     // 个人刷屏警告速率
	GroupReplenishRateStr    string     `yaml:"groupReplenishRate"`    // 群组刷屏警告速率，字符串格式
	GroupReplenishRate       rate.Limit `yaml:"-"`                     // 群组刷屏警告速率
	PersonalBurst            int64      `yaml:"personalBurst"`         // 个人自定义上限
	GroupBurst               int64      `yaml:"groupBurst"`            // 群组自定义上限
}

type QuitInactiveConfig struct {
	QuitInactiveThreshold time.Duration `yaml:"quitInactiveThreshold"` // 退出不活跃群组的时间阈值
	quitInactiveCronEntry cron.EntryID
}

type ExtConfig struct {
	DefaultCocRuleIndex int64 `yaml:"defaultCocRuleIndex" jsbind:"defaultCocRuleIndex"` // 默认coc index
	MaxExecuteTime      int64 `yaml:"maxExecuteTime" jsbind:"maxExecuteTime"`           // 最大骰点次数
	MaxCocCardGen       int64 `yaml:"maxCocCardGen" jsbind:"maxCocCardGen"`             // 最大coc制卡数

	ExtDefaultSettings []*ExtDefaultSettingItem `yaml:"extDefaultSettings"` // 新群扩展按此顺序加载
}

type BanConfig struct {
	BanList *BanListInfo `yaml:"banList"`
}

type JsConfig struct {
	JsEnable          bool            `yaml:"jsEnable" json:"jsEnable"`
	DisabledJsScripts map[string]bool `yaml:"disabledJsScripts" json:"disabledJsScripts"` // 作为set
}

type StoryLogConfig struct {
	LogSizeNoticeEnable bool `yaml:"logSizeNoticeEnable"` // 开启日志数量提示
	LogSizeNoticeCount  int  `yaml:"LogSizeNoticeCount"`  // 日志数量提示阈值，默认500
}

type MailConfig struct {
	MailEnable   bool   `json:"mailEnable" yaml:"mailEnable"`     // 是否启用
	MailFrom     string `json:"mailFrom" yaml:"mailFrom"`         // 邮箱来源
	MailPassword string `json:"mailPassword" yaml:"mailPassword"` // 邮箱密钥/密码
	MailSMTP     string `json:"mailSmtp" yaml:"mailSmtp"`         // 邮箱 smtp 地址
}

type NewsConfig struct {
	NewsMark string `json:"newsMark" yaml:"newsMark"` // 已读新闻的md5
}

type CensorConfig struct {
	EnableCensor         bool                   `json:"enableCensor" yaml:"enableCensor"` // 启用敏感词审查
	CensorMode           CensorMode             `json:"censorMode" yaml:"censorMode"`
	CensorThresholds     map[censor.Level]int   `json:"censorThresholds" yaml:"censorThresholds"` // 敏感词阈值
	CensorHandlers       map[censor.Level]uint8 `json:"censorHandlers" yaml:"censorHandlers"`
	CensorScores         map[censor.Level]int   `json:"censorScores" yaml:"censorScores"`                 // 敏感词怒气值
	CensorCaseSensitive  bool                   `json:"censorCaseSensitive" yaml:"censorCaseSensitive"`   // 敏感词大小写敏感
	CensorMatchPinyin    bool                   `json:"censorMatchPinyin" yaml:"censorMatchPinyin"`       // 敏感词匹配拼音
	CensorFilterRegexStr string                 `json:"censorFilterRegexStr" yaml:"censorFilterRegexStr"` // 敏感词过滤字符正则
}
