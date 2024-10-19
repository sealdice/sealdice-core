package dice

import (
	"time"

	"github.com/robfig/cron/v3"
	"golang.org/x/time/rate"
	"gopkg.in/yaml.v3"

	"sealdice-core/dice/censor"
	"sealdice-core/utils"
)

// ConfigVersion 当前设置版本
const (
	ConfigVersion     = 1
	ConfigVersionCode = 10300 // 旧的设置版本标记
)

type Config struct {
	d             *Dice `yaml:"-"`
	ConfigVersion int   `yaml:"configVersion" json:"configVersion"` // 配置版本

	// 基础设置
	BaseConfig `yaml:",inline"`
	// 刷屏警告设置
	RateLimitConfig `yaml:",inline"`
	// 退出不活跃设置
	QuitInactiveConfig `yaml:",inline"`
	// 扩展设置
	ExtConfig `yaml:",inline"`
	// 黑名单设置
	BanConfig `yaml:",inline"`
	// js 设置
	JsConfig `yaml:",inline"`
	// 跑团日志设置
	StoryLogConfig `yaml:",inline"`
	// 邮件设置
	MailConfig `yaml:",inline"`
	// 新闻设置
	NewsConfig `yaml:",inline"`
	// 敏感词设置
	CensorConfig `yaml:",inline"`
}

func NewConfig(d *Dice) Config {
	c := DefaultConfig
	c.d = d

	// set other default
	c.BanList = &BanListInfo{Parent: c.d}
	c.BanList.Init()
	return c
}

func (c *Config) LoadYamlConfig(data []byte) error {
	err := yaml.Unmarshal(data, &c)
	if err != nil {
		return err
	}
	c.migrateOld2Version1()
	return nil
}

// migrateOld2Version1 旧格式设置项的迁移
func (c *Config) migrateOld2Version1() {
	if c.ConfigVersion != 0 {
		return
	}
	c.ConfigVersion = ConfigVersion

	c.CommandCompatibleMode = DefaultConfig.CommandCompatibleMode

	if c.MaxExecuteTime == 0 {
		c.MaxExecuteTime = DefaultConfig.MaxExecuteTime
	}

	if c.MaxCocCardGen == 0 {
		c.MaxCocCardGen = DefaultConfig.MaxCocCardGen
	}

	if c.PersonalReplenishRateStr == "" {
		c.PersonalReplenishRateStr = DefaultConfig.PersonalReplenishRateStr
		c.PersonalReplenishRate = DefaultConfig.PersonalReplenishRate
	} else {
		if parsed, errParse := utils.ParseRate(c.PersonalReplenishRateStr); errParse == nil {
			c.PersonalReplenishRate = parsed
		} else {
			c.d.Logger.Errorf("解析PersonalReplenishRate失败: %v", errParse)
			c.PersonalReplenishRateStr = DefaultConfig.PersonalReplenishRateStr
			c.PersonalReplenishRate = DefaultConfig.PersonalReplenishRate
		}
	}

	if c.PersonalBurst == 0 {
		c.PersonalBurst = DefaultConfig.PersonalBurst
	}

	if c.GroupReplenishRateStr == "" {
		c.GroupReplenishRateStr = DefaultConfig.GroupReplenishRateStr
		c.GroupReplenishRate = DefaultConfig.GroupReplenishRate
	} else {
		if parsed, errParse := utils.ParseRate(c.GroupReplenishRateStr); errParse == nil {
			c.GroupReplenishRate = parsed
		} else {
			c.d.Logger.Errorf("解析GroupReplenishRate失败: %v", errParse)
			c.GroupReplenishRateStr = DefaultConfig.GroupReplenishRateStr
			c.GroupReplenishRate = DefaultConfig.GroupReplenishRate
		}
	}

	if c.GroupBurst == 0 {
		c.GroupBurst = DefaultConfig.GroupBurst
	}

	if c.VersionCode != 0 && c.VersionCode < 10001 {
		c.AliveNoticeValue = DefaultConfig.AliveNoticeValue
	}

	if c.VersionCode != 0 && c.VersionCode < 10003 {
		c.LogSizeNoticeCount = DefaultConfig.LogSizeNoticeCount
		c.LogSizeNoticeEnable = DefaultConfig.LogSizeNoticeEnable
		c.CustomReplyConfigEnable = DefaultConfig.CustomReplyConfigEnable
	}

	if c.VersionCode != 0 && c.VersionCode < 10004 {
		c.AutoReloginEnable = DefaultConfig.AutoReloginEnable
	}
}

type BaseConfig struct {
	CommandCompatibleMode   bool       `yaml:"commandCompatibleMode" json:"-"`
	LastSavedTime           *time.Time `yaml:"lastSavedTime" json:"-"`
	NoticeIDs               []string   `yaml:"noticeIds" json:"noticeIds"`                             // 通知ID
	OnlyLogCommandInGroup   bool       `yaml:"onlyLogCommandInGroup" json:"onlyLogCommandInGroup"`     // 日志中仅记录命令
	OnlyLogCommandInPrivate bool       `yaml:"onlyLogCommandInPrivate" json:"onlyLogCommandInPrivate"` // 日志中仅记录命令
	VersionCode             int        `yaml:"versionCode" json:"versionCode"`                         // 版本ID(配置文件)
	MessageDelayRangeStart  float64    `yaml:"messageDelayRangeStart" json:"messageDelayRangeStart"`   // 指令延迟区间
	MessageDelayRangeEnd    float64    `yaml:"messageDelayRangeEnd" json:"messageDelayRangeEnd"`
	WorkInQQChannel         bool       `yaml:"workInQQChannel" json:"workInQQChannel"`
	QQChannelAutoOn         bool       `yaml:"QQChannelAutoOn" json:"QQChannelAutoOn"`                 // QQ频道中自动开启(默认不开)
	QQChannelLogMessage     bool       `yaml:"QQChannelLogMessage" json:"QQChannelLogMessage"`         // QQ频道中记录消息(默认不开)
	QQEnablePoke            bool       `yaml:"QQEnablePoke" json:"QQEnablePoke"`                       // 启用戳一戳
	TextCmdTrustOnly        bool       `yaml:"textCmdTrustOnly" json:"textCmdTrustOnly"`               // 只允许信任用户或master使用text指令
	IgnoreUnaddressedBotCmd bool       `yaml:"ignoreUnaddressedBotCmd" json:"ignoreUnaddressedBotCmd"` // 不响应群聊裸bot指令
	UILogLimit              int64      `yaml:"UILogLimit" json:"-"`
	FriendAddComment        string     `yaml:"friendAddComment" json:"friendAddComment"` // 加好友验证信息
	CustomReplyConfigEnable bool       `yaml:"customReplyConfigEnable" json:"customReplyConfigEnable"`
	AutoReloginEnable       bool       `yaml:"autoReloginEnable" json:"autoReloginEnable"`       // 启用自动重新登录
	RefuseGroupInvite       bool       `yaml:"refuseGroupInvite" json:"refuseGroupInvite"`       // 拒绝加入新群
	UpgradeWindowID         string     `yaml:"upgradeWindowId" json:"-"`                         // 执行升级指令的窗口
	UpgradeEndpointID       string     `yaml:"upgradeEndpointId" json:"-"`                       // 执行升级指令的端点
	BotExtFreeSwitch        bool       `yaml:"botExtFreeSwitch" json:"botExtFreeSwitch"`         // 允许任意人员开关: 否则邀请者、群主、管理员、master有权限
	TrustOnlyMode           bool       `yaml:"trustOnlyMode" json:"trustOnlyMode"`               // 只有信任的用户/master可以拉群和使用
	AliveNoticeEnable       bool       `yaml:"aliveNoticeEnable" json:"aliveNoticeEnable"`       // 定时通知
	AliveNoticeValue        string     `yaml:"aliveNoticeValue" json:"aliveNoticeValue"`         // 定时通知间隔
	ReplyDebugMode          bool       `yaml:"replyDebugMode" json:"replyDebugMode"`             // 回复调试
	PlayerNameWrapEnable    bool       `yaml:"playerNameWrapEnable" json:"playerNameWrapEnable"` // 启用玩家名称外框

	VMVersionForReply string `json:"VMVersionForReply" yaml:"VMVersionForReply"` // 自定义回复使用的vm版本
	VMVersionForDeck  string `json:"VMVersionForDeck" yaml:"VMVersionForDeck"`   // 牌堆使用的vm版本

	// TODO: 历史遗留问题，由于不输出DICE日志效果过差，已经抹除日志输出选项，剩余两个选项，私以为可以想办法也抹除掉。
	Name    string `yaml:"name"`    // 名称，默认为default
	DataDir string `yaml:"dataDir"` // 数据路径，为./data/{name}，例如data/default
}

type RateLimitConfig struct {
	RateLimitEnabled         bool       `yaml:"rateLimitEnabled" json:"rateLimitEnabled"`           // 启用频率限制 (刷屏限制)
	PersonalReplenishRateStr string     `yaml:"personalReplenishRate" json:"personalReplenishRate"` // 个人刷屏警告速率，字符串格式
	PersonalReplenishRate    rate.Limit `yaml:"-" json:"-"`                                         // 个人刷屏警告速率
	GroupReplenishRateStr    string     `yaml:"groupReplenishRate" json:"groupReplenishRate"`       // 群组刷屏警告速率，字符串格式
	GroupReplenishRate       rate.Limit `yaml:"-" json:"-"`                                         // 群组刷屏警告速率
	PersonalBurst            int64      `yaml:"personalBurst" json:"personalBurst"`                 // 个人自定义上限
	GroupBurst               int64      `yaml:"groupBurst" json:"groupBurst"`                       // 群组自定义上限
}

type QuitInactiveConfig struct {
	QuitInactiveThreshold time.Duration `yaml:"quitInactiveThreshold" json:"-"` // 退出不活跃群组的时间阈值
	quitInactiveCronEntry cron.EntryID

	QuitInactiveThresholdDays float64 `yaml:"-" json:"quitInactiveThreshold"` // 为了和前端通信

	QuitInactiveBatchSize int64 `yaml:"quitInactiveBatchSize"` // 退出不活跃群组的批量大小
	QuitInactiveBatchWait int64 `yaml:"quitInactiveBatchWait"` // 退出不活跃群组的批量等待时间（分）
}

type ExtConfig struct {
	DefaultCocRuleIndex int64 `yaml:"defaultCocRuleIndex" json:"-" jsbind:"defaultCocRuleIndex"` // 默认coc index
	MaxExecuteTime      int64 `yaml:"maxExecuteTime" json:"-" jsbind:"maxExecuteTime"`           // 最大骰点次数
	MaxCocCardGen       int64 `yaml:"maxCocCardGen" json:"-" jsbind:"maxCocCardGen"`             // 最大coc制卡数

	ExtDefaultSettings []*ExtDefaultSettingItem `yaml:"extDefaultSettings" json:"extDefaultSettings"` // 新群扩展按此顺序加载
}

type BanConfig struct {
	BanList *BanListInfo `yaml:"banList" json:"-"`
}

type JsConfig struct {
	JsEnable          bool            `yaml:"jsEnable" json:"jsEnable"`
	DisabledJsScripts map[string]bool `yaml:"disabledJsScripts" json:"disabledJsScripts"` // 作为set
}

type StoryLogConfig struct {
	LogSizeNoticeEnable bool `yaml:"logSizeNoticeEnable" json:"logSizeNoticeEnable"` // 开启日志数量提示
	LogSizeNoticeCount  int  `yaml:"LogSizeNoticeCount" json:"logSizeNoticeCount"`   // 日志数量提示阈值，默认500
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
