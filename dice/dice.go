package dice

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-creed/sat"
	"github.com/jmoiron/sqlx"
	wr "github.com/mroth/weightedrand"
	"github.com/robfig/cron/v3"
	"github.com/tidwall/buntdb"
	"go.uber.org/zap"

	rand2 "golang.org/x/exp/rand"
	"golang.org/x/exp/slices"
	"golang.org/x/time/rate"

	"sealdice-core/dice/censor"
	"sealdice-core/dice/logger"
	"sealdice-core/dice/model"
)

var (
	APPNAME = "SealDice"

	// VERSION 版本号，按固定格式，action 在构建时可能会自动注入部分信息
	// 正式：主版本号+yyyyMMdd，如 1.4.5+20240308
	// dev：主版本号-dev+yyyyMMdd.7位hash，如 1.4.5-dev+20240308.1a2b3c4
	// rc：主版本号-rc.序号+yyyyMMdd.7位hash如 1.4.5-rc.0+20240308.1a2b3c4，1.4.5-rc.1+20240309.2a3b4c4，……
	VERSION = semver.MustParse(VERSION_MAIN + VERSION_PRERELEASE + VERSION_BUILD_METADATA)

	// VERSION_MAIN 主版本号
	VERSION_MAIN = "1.4.6"
	// VERSION_PRERELEASE 先行版本号
	VERSION_PRERELEASE = "-dev"
	// VERSION_BUILD_METADATA 版本编译信息
	VERSION_BUILD_METADATA = ""

	// APP_CHANNEL 更新频道，stable/dev，在 action 构建时自动注入
	APP_CHANNEL = "dev" //nolint:revive

	VERSION_CODE = int64(1004005) //nolint:revive

	VERSION_JSAPI_COMPATIBLE = []*semver.Version{
		VERSION,
		semver.MustParse("1.4.5"),
		semver.MustParse("1.4.4"),
		semver.MustParse("1.4.3"),
	}
)

type CmdExecuteResult struct {
	Matched  bool // 是否是指令
	Solved   bool `jsbind:"solved"` // 是否响应此指令
	ShowHelp bool `jsbind:"showHelp"`
}

type CmdItemInfo struct {
	Name                    string                    `jsbind:"name"`
	ShortHelp               string                    // 短帮助，格式是 .xxx a b // 说明
	Help                    string                    `jsbind:"help"`                    // 长帮助，带换行的较详细说明
	HelpFunc                func(isShort bool) string `jsbind:"helpFunc"`                // 函数形式帮助，存在时优先于其他
	AllowDelegate           bool                      `jsbind:"allowDelegate"`           // 允许代骰
	DisabledInPrivate       bool                      `jsbind:"disabledInPrivate"`       // 私聊不可用
	EnableExecuteTimesParse bool                      `jsbind:"enableExecuteTimesParse"` // 启用执行次数解析，也就是解析3#这样的文本

	IsJsSolveFunc bool
	Solve         func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult `jsbind:"solve"`

	Raw                bool `jsbind:"raw"`                // 高级模式。默认模式下行为是：需要在当前群/私聊开启，或@自己时生效(需要为第一个@目标)
	CheckCurrentBotOn  bool `jsbind:"checkCurrentBotOn"`  // 是否检查当前可用状况，包括群内可用和是私聊两种方式，如失败不进入solve
	CheckMentionOthers bool `jsbind:"checkMentionOthers"` // 是否检查@了别的骰子，如失败不进入solve
}

type CmdMapCls map[string]*CmdItemInfo

// type ExtInfoStorage interface {
//
// }

type ExtInfo struct {
	Name    string   `yaml:"name" json:"name" jsbind:"name"`    // 名字
	Aliases []string `yaml:"-" json:"aliases" jsbind:"aliases"` // 别名
	Version string   `yaml:"-" json:"version" jsbind:"version"` // 版本
	// 作者
	// 更新时间
	AutoActive      bool      `yaml:"-" json:"-" jsbind:"autoActive"` // 是否自动开启
	CmdMap          CmdMapCls `yaml:"-" json:"-" jsbind:"cmdMap"`     // 指令集合
	Brief           string    `yaml:"-" json:"-"`
	ActiveOnPrivate bool      `yaml:"-" json:"-"`

	DefaultSetting *ExtDefaultSettingItem `yaml:"-" json:"-"` // 默认配置

	Author       string   `yaml:"-" json:"-" jsbind:"author"`
	ConflictWith []string `yaml:"-" json:"-"`
	Official     bool     `yaml:"-" json:"-"` // 官方插件

	dice    *Dice
	IsJsExt bool          `json:"-"`
	Source  *JsScriptInfo `yaml:"-" json:"-"`
	Storage *buntdb.DB    `yaml:"-"  json:"-"`
	// 为Storage使用互斥锁,并根据ID佬的说法修改为合适的名称
	dbMu sync.Mutex `yaml:"-"` // 互斥锁
	init bool       `yaml:"-"` // 标记Storage是否已初始化

	// 定时任务列表，用于避免 task 失去引用
	taskList []*JsScriptTask `yaml:"-" json:"-"`

	OnNotCommandReceived func(ctx *MsgContext, msg *Message)                        `yaml:"-" json:"-" jsbind:"onNotCommandReceived"` // 指令过滤后剩下的
	OnCommandOverride    func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) bool `yaml:"-" json:"-"`                               // 覆盖指令行为

	OnCommandReceived   func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) `yaml:"-" json:"-" jsbind:"onCommandReceived"`
	OnMessageReceived   func(ctx *MsgContext, msg *Message)                   `yaml:"-" json:"-" jsbind:"onMessageReceived"`
	OnMessageSend       func(ctx *MsgContext, msg *Message, flag string)      `yaml:"-" json:"-" jsbind:"onMessageSend"`
	OnMessageDeleted    func(ctx *MsgContext, msg *Message)                   `yaml:"-" json:"-" jsbind:"onMessageDeleted"`
	OnMessageEdit       func(ctx *MsgContext, msg *Message)                   `yaml:"-" json:"-" jsbind:"onMessageEdit"`
	OnGroupJoined       func(ctx *MsgContext, msg *Message)                   `yaml:"-" json:"-" jsbind:"onGroupJoined"`
	OnGroupMemberJoined func(ctx *MsgContext, msg *Message)                   `yaml:"-" json:"-" jsbind:"onGroupMemberJoined"`
	OnGuildJoined       func(ctx *MsgContext, msg *Message)                   `yaml:"-" json:"-" jsbind:"onGuildJoined"`
	OnBecomeFriend      func(ctx *MsgContext, msg *Message)                   `yaml:"-" json:"-" jsbind:"onBecomeFriend"`
	GetDescText         func(i *ExtInfo) string                               `yaml:"-" json:"-" jsbind:"getDescText"`
	IsLoaded            bool                                                  `yaml:"-" json:"-" jsbind:"isLoaded"`
	OnLoad              func()                                                `yaml:"-" json:"-" jsbind:"onLoad"`
}

type DiceConfig struct { //nolint:revive
	Name       string `yaml:"name"`       // 名称，默认为default
	DataDir    string `yaml:"dataDir"`    // 数据路径，为./data/{name}，例如data/default
	IsLogPrint bool   `yaml:"isLogPrint"` // 是否在控制台打印log
}

type ExtDefaultSettingItem struct {
	Name            string          `yaml:"name" json:"name"`
	AutoActive      bool            `yaml:"autoActive" json:"autoActive"`                // 是否自动开启
	DisabledCommand map[string]bool `yaml:"disabledCommand,flow" json:"disabledCommand"` // 实际为set
	ExtItem         *ExtInfo        `yaml:"-" json:"-"`
	Loaded          bool            `yaml:"-" json:"loaded"` // 当前插件是否正确加载. serve.yaml不保存, 前端请求时提供
}

type ExtDefaultSettingItemSlice []*ExtDefaultSettingItem

// 强制coc7排序在较前位置

func (x ExtDefaultSettingItemSlice) Len() int           { return len(x) }
func (x ExtDefaultSettingItemSlice) Less(i, _ int) bool { return x[i].Name == "coc7" }
func (x ExtDefaultSettingItemSlice) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }

type Dice struct {
	ImSession               *IMSession             `yaml:"imSession" jsbind:"imSession"`
	CmdMap                  CmdMapCls              `yaml:"-" json:"-"`
	ExtList                 []*ExtInfo             `yaml:"-"`
	RollParser              *DiceRollParser        `yaml:"-"`
	CommandCompatibleMode   bool                   `yaml:"commandCompatibleMode"`
	LastSavedTime           *time.Time             `yaml:"lastSavedTime"`
	LastUpdatedTime         int64                  `yaml:"-"`
	TextMap                 map[string]*wr.Chooser `yaml:"-"`
	BaseConfig              DiceConfig             `yaml:"-"`
	DBData                  *sqlx.DB               `yaml:"-"`                                    // 数据库对象
	DBLogs                  *sqlx.DB               `yaml:"-"`                                    // 数据库对象
	Logger                  *zap.SugaredLogger     `yaml:"-"`                                    // 日志
	LogWriter               *logger.WriterX        `yaml:"-"`                                    // 用于api的log对象
	IsDeckLoading           bool                   `yaml:"-"`                                    // 正在加载中
	DeckList                []*DeckInfo            `yaml:"deckList" jsbind:"deckList"`           // 牌堆信息
	CommandPrefix           []string               `yaml:"commandPrefix" jsbind:"commandPrefix"` // 指令前导
	DiceMasters             []string               `yaml:"diceMasters" jsbind:"diceMasters"`     // 骰主设置，需要格式: 平台:帐号
	NoticeIDs               []string               `yaml:"noticeIds"`                            // 通知ID
	OnlyLogCommandInGroup   bool                   `yaml:"onlyLogCommandInGroup"`                // 日志中仅记录命令
	OnlyLogCommandInPrivate bool                   `yaml:"onlyLogCommandInPrivate"`              // 日志中仅记录命令
	VersionCode             int                    `json:"versionCode"`                          // 版本ID(配置文件)
	MessageDelayRangeStart  float64                `yaml:"messageDelayRangeStart"`               // 指令延迟区间
	MessageDelayRangeEnd    float64                `yaml:"messageDelayRangeEnd"`
	WorkInQQChannel         bool                   `yaml:"workInQQChannel"`
	QQChannelAutoOn         bool                   `yaml:"QQChannelAutoOn"`         // QQ频道中自动开启(默认不开)
	QQChannelLogMessage     bool                   `yaml:"QQChannelLogMessage"`     // QQ频道中记录消息(默认不开)
	QQEnablePoke            bool                   `yaml:"QQEnablePoke"`            // 启用戳一戳
	TextCmdTrustOnly        bool                   `yaml:"textCmdTrustOnly"`        // 只允许信任用户或master使用text指令
	IgnoreUnaddressedBotCmd bool                   `yaml:"ignoreUnaddressedBotCmd"` // 不响应群聊裸bot指令
	UILogLimit              int64                  `yaml:"UILogLimit"`
	FriendAddComment        string                 `yaml:"friendAddComment"` // 加好友验证信息
	MasterUnlockCode        string                 `yaml:"-"`                // 解锁码，每20分钟变化一次，使用后立即变化
	MasterUnlockCodeTime    int64                  `yaml:"-"`
	CustomReplyConfigEnable bool                   `yaml:"customReplyConfigEnable"`
	CustomReplyConfig       []*ReplyConfig         `yaml:"-"`
	RefuseGroupInvite       bool                   `yaml:"refuseGroupInvite"`    // 拒绝加入新群
	UpgradeWindowID         string                 `yaml:"upgradeWindowId"`      // 执行升级指令的窗口
	UpgradeEndpointID       string                 `yaml:"upgradeEndpointId"`    // 执行升级指令的端点
	BotExtFreeSwitch        bool                   `yaml:"botExtFreeSwitch"`     // 允许任意人员开关: 否则邀请者、群主、管理员、master有权限
	TrustOnlyMode           bool                   `yaml:"trustOnlyMode"`        // 只有信任的用户/master可以拉群和使用
	AliveNoticeEnable       bool                   `yaml:"aliveNoticeEnable"`    // 定时通知
	AliveNoticeValue        string                 `yaml:"aliveNoticeValue"`     // 定时通知间隔
	ReplyDebugMode          bool                   `yaml:"replyDebugMode"`       // 回复调试
	PlayerNameWrapEnable    bool                   `yaml:"playerNameWrapEnable"` // 启用玩家名称外框

	RateLimitEnabled         bool       `yaml:"rateLimitEnabled"`      // 启用频率限制 (刷屏限制)
	PersonalReplenishRateStr string     `yaml:"personalReplenishRate"` // 个人刷屏警告速率，字符串格式
	PersonalReplenishRate    rate.Limit `yaml:"-"`                     // 个人刷屏警告速率
	GroupReplenishRateStr    string     `yaml:"groupReplenishRate"`    // 群组刷屏警告速率，字符串格式
	GroupReplenishRate       rate.Limit `yaml:"-"`                     // 群组刷屏警告速率
	PersonalBurst            int64      `yaml:"personalBurst"`         // 个人自定义上限
	GroupBurst               int64      `yaml:"groupBurst"`            // 群组自定义上限

	QuitInactiveThreshold time.Duration `yaml:"quitInactiveThreshold"` // 退出不活跃群组的时间阈值
	quitInactiveCronEntry cron.EntryID
	QuitInactiveBatchSize int64 `yaml:"quitInactiveBatchSize"` // 退出不活跃群组的批量大小
	QuitInactiveBatchWait int64 `yaml:"quitInactiveBatchWait"` // 退出不活跃群组的批量等待时间（分）

	DefaultCocRuleIndex int64 `yaml:"defaultCocRuleIndex" jsbind:"defaultCocRuleIndex"` // 默认coc index
	MaxExecuteTime      int64 `yaml:"maxExecuteTime" jsbind:"maxExecuteTime"`           // 最大骰点次数
	MaxCocCardGen       int64 `yaml:"maxCocCardGen" jsbind:"maxCocCardGen"`             // 最大coc制卡数

	ExtDefaultSettings []*ExtDefaultSettingItem `yaml:"extDefaultSettings"` // 新群扩展按此顺序加载

	BanList *BanListInfo `yaml:"banList"` //

	TextMapRaw      TextTemplateWithWeightDict `yaml:"-"`
	TextMapHelpInfo TextTemplateWithHelpDict   `yaml:"-"`
	ConfigManager   *ConfigManager             `yaml:"-"`
	Parent          *DiceManager               `yaml:"-"`

	CocExtraRules     map[int]*CocRuleInfo   `yaml:"-" json:"cocExtraRules"`
	Cron              *cron.Cron             `yaml:"-" json:"-"`
	AliveNoticeEntry  cron.EntryID           `yaml:"-" json:"-"`
	JsEnable          bool                   `yaml:"jsEnable" json:"jsEnable"`
	DisabledJsScripts map[string]bool        `yaml:"disabledJsScripts" json:"disabledJsScripts"` // 作为set
	JsPrinter         *PrinterFunc           `yaml:"-" json:"-"`
	JsRequire         *require.RequireModule `yaml:"-" json:"-"`
	JsLoop            *eventloop.EventLoop   `yaml:"-" json:"-"`
	JsScriptList      []*JsScriptInfo        `yaml:"-" json:"-"`
	JsScriptCron      *cron.Cron             `yaml:"-" json:"-"`
	JsScriptCronLock  *sync.Mutex            `yaml:"-" json:"-"`
	// 重载使用的互斥锁
	JsReloadLock sync.Mutex `yaml:"-" json:"-"`

	// 内置脚本摘要表，用于判断内置脚本是否有更新
	JsBuiltinDigestSet map[string]bool `yaml:"-" json:"-"`
	// 当前在加载的脚本路径，用于关联 jsScriptInfo 和 ExtInfo
	JsLoadingScript *JsScriptInfo `yaml:"-" json:"-"`

	// 游戏系统规则模板
	GameSystemMap *SyncMap[string, *GameSystemTemplate] `yaml:"-" json:"-"`

	RunAfterLoaded []func() `yaml:"-" json:"-"`

	LogSizeNoticeEnable bool `yaml:"logSizeNoticeEnable"` // 开启日志数量提示
	LogSizeNoticeCount  int  `yaml:"LogSizeNoticeCount"`  // 日志数量提示阈值，默认500

	IsAlreadyLoadConfig  bool                 `yaml:"-"` // 如果在loads前崩溃，那么不写入配置，防止覆盖为空的
	deckCommandItemsList DeckCommandListItems // 牌堆key信息，辅助作为模糊搜索使用

	UIEndpoint *EndPointInfo `yaml:"-" json:"-"` // UI Endpoint

	MailEnable   bool   `json:"mailEnable" yaml:"mailEnable"`     // 是否启用
	MailFrom     string `json:"mailFrom" yaml:"mailFrom"`         // 邮箱来源
	MailPassword string `json:"mailPassword" yaml:"mailPassword"` // 邮箱密钥/密码
	MailSMTP     string `json:"mailSmtp" yaml:"mailSmtp"`         // 邮箱 smtp 地址

	NewsMark string `json:"newsMark" yaml:"newsMark"` // 已读新闻的md5

	EnableCensor         bool                   `json:"enableCensor" yaml:"enableCensor"` // 启用敏感词审查
	CensorManager        *CensorManager         `json:"-" yaml:"-"`
	CensorMode           CensorMode             `json:"censorMode" yaml:"censorMode"`
	CensorThresholds     map[censor.Level]int   `json:"censorThresholds" yaml:"censorThresholds"` // 敏感词阈值
	CensorHandlers       map[censor.Level]uint8 `json:"censorHandlers" yaml:"censorHandlers"`
	CensorScores         map[censor.Level]int   `json:"censorScores" yaml:"censorScores"`                 // 敏感词怒气值
	CensorCaseSensitive  bool                   `json:"censorCaseSensitive" yaml:"censorCaseSensitive"`   // 敏感词大小写敏感
	CensorMatchPinyin    bool                   `json:"censorMatchPinyin" yaml:"censorMatchPinyin"`       // 敏感词匹配拼音
	CensorFilterRegexStr string                 `json:"censorFilterRegexStr" yaml:"censorFilterRegexStr"` // 敏感词过滤字符正则

	AdvancedConfig AdvancedConfig `json:"-" yaml:"-"`

	ContainerMode bool `yaml:"-" json:"-"` // 容器模式：禁用内置适配器，不允许使用内置Lagrange和旧的内置Gocq
}

type CensorMode int

const (
	OnlyOutputReply CensorMode = iota
	OnlyInputCommand
	AllInput
)

const (
	// SendWarning 发送警告
	SendWarning CensorHandler = iota
	// SendNotice 向通知列表/邮件发送通知
	SendNotice
	// BanUser 拉黑用户
	BanUser
	// BanGroup 拉黑群
	BanGroup
	// BanInviter 拉黑邀请人
	BanInviter
	// AddScore 增加怒气值
	AddScore
)

var CensorHandlerText = map[CensorHandler]string{
	SendWarning: "SendWarning",
	SendNotice:  "SendNotice",
	BanUser:     "BanUser",
	BanGroup:    "BanGroup",
	BanInviter:  "BanInviter",
	AddScore:    "AddScore",
}

type CensorHandler int

func (d *Dice) MarkModified() {
	d.LastUpdatedTime = time.Now().Unix()
}

func (d *Dice) CocExtraRulesAdd(ruleInfo *CocRuleInfo) bool {
	if _, ok := d.CocExtraRules[ruleInfo.Index]; ok {
		return false
	}
	d.CocExtraRules[ruleInfo.Index] = ruleInfo
	return true
}

func (d *Dice) Init() {
	d.BaseConfig.DataDir = filepath.Join("./data", d.BaseConfig.Name)
	_ = os.MkdirAll(d.BaseConfig.DataDir, 0o755)
	_ = os.MkdirAll(filepath.Join(d.BaseConfig.DataDir, "configs"), 0o755)
	_ = os.MkdirAll(filepath.Join(d.BaseConfig.DataDir, "extensions"), 0o755)
	_ = os.MkdirAll(filepath.Join(d.BaseConfig.DataDir, "log-exports"), 0o755)
	_ = os.MkdirAll(filepath.Join(d.BaseConfig.DataDir, "extra"), 0o755)
	_ = os.MkdirAll(filepath.Join(d.BaseConfig.DataDir, "scripts"), 0o755)

	d.Cron = cron.New()
	d.Cron.Start()

	d.CocExtraRules = map[int]*CocRuleInfo{}

	var err error
	d.DBData, d.DBLogs, err = model.SQLiteDBInit(d.BaseConfig.DataDir)
	if err != nil {
		// TODO:
		fmt.Println(err)
	}

	log := logger.Init(filepath.Join(d.BaseConfig.DataDir, "record.log"), d.BaseConfig.Name, d.BaseConfig.IsLogPrint)
	d.Logger = log.Logger
	d.LogWriter = log.WX
	d.BanList = &BanListInfo{Parent: d}
	d.BanList.Init()

	initVerify()

	d.CommandCompatibleMode = true
	d.ImSession = &IMSession{}
	d.ImSession.Parent = d
	d.ImSession.ServiceAtNew = make(map[string]*GroupInfo)
	d.CmdMap = CmdMapCls{}
	d.GameSystemMap = InitializeSyncMap[string, *GameSystemTemplate]()
	d.ConfigManager = NewConfigManager(filepath.Join(d.BaseConfig.DataDir, "configs", "plugin-configs.json"))
	_ = d.ConfigManager.Load()

	d.registerCoreCommands()
	d.RegisterBuiltinExt()
	d.loads()
	d.loadAdvanced()
	d.BanList.Loads()
	d.BanList.AfterLoads()
	d.IsAlreadyLoadConfig = true

	if d.EnableCensor {
		d.NewCensorManager()
	}

	// 创建js运行时
	if d.JsEnable {
		d.Logger.Info("js扩展支持：开启")
		d.JsInit()
	} else {
		d.Logger.Info("js扩展支持：关闭")
	}

	for _, i := range d.ExtList {
		if i.OnLoad != nil {
			i.callWithJsCheck(d, func() {
				i.OnLoad()
			})
		}
	}

	for _, i := range d.RunAfterLoaded {
		defer func() {
			// 防止报错
			if r := recover(); r != nil {
				d.Logger.Error("RunAfterLoaded 报错: ", r)
			}
		}()
		i()
	}
	d.RunAfterLoaded = []func(){}

	autoSave := func() {
		count := 0
		t := time.NewTicker(30 * time.Second)
		for {
			<-t.C
			if d.IsAlreadyLoadConfig {
				count++
				d.Save(true)
				if count%5 == 0 {
					// d.Logger.Info("测试: flush wal")
					_ = model.FlushWAL(d.DBData)
					_ = model.FlushWAL(d.DBLogs)
					if d.CensorManager != nil && d.CensorManager.DB != nil {
						_ = model.FlushWAL(d.CensorManager.DB)
					}
				}
			}
		}
	}
	go autoSave()

	refreshGroupInfo := func() {
		t := time.NewTicker(35 * time.Second)
		defer func() {
			// 防止报错
			if r := recover(); r != nil {
				d.Logger.Error(r)
			}
		}()

		for {
			<-t.C

			// 自动更新群信息
			for _, i := range d.ImSession.EndPoints {
				if i.Enable {
					for k, v := range d.ImSession.ServiceAtNew {
						// TODO: 注意这里的Active可能不需要改
						if !strings.HasPrefix(k, "PG-") && v.Active {
							diceID := i.UserID
							now := time.Now().Unix()

							// 上次被人使用小于60s
							if now-v.RecentDiceSendTime < 60 {
								// 在群内存在，且开启时
								if _, exists := v.DiceIDExistsMap.Load(diceID); exists {
									if _, exists := v.DiceIDActiveMap.Load(diceID); exists {
										i.Adapter.GetGroupInfoAsync(k)
									}
								}
							}
						}
					}
				}
			}
		}
	}
	go refreshGroupInfo()

	d.ApplyAliveNotice()
	if d.JsEnable {
		d.JsBuiltinDigestSet = make(map[string]bool)
		d.JsLoadScripts()
	} else {
		d.Logger.Info("js扩展支持已关闭，跳过js脚本的加载")
	}

	if d.UpgradeWindowID != "" {
		go func() {
			defer ErrorLogAndContinue(d)

			var ep *EndPointInfo
			for _, _ep := range d.ImSession.EndPoints {
				if _ep.ID == d.UpgradeEndpointID {
					ep = _ep
					break
				}
			}

			// 发送指令所用的端点不存在
			if ep == nil {
				return
			}

			for {
				time.Sleep(30 * time.Second)
				text := fmt.Sprintf("升级完成，当前版本: %s", VERSION.String())

				if ep.State == 2 {
					// 还没好，继续等待
					continue
				}

				// 可以了，发送消息
				ctx := &MsgContext{Dice: d, EndPoint: ep, Session: d.ImSession}
				isGroup := strings.Contains(d.UpgradeWindowID, "-Group:")
				if isGroup {
					ReplyGroup(ctx, &Message{GroupID: d.UpgradeWindowID}, text)
				} else {
					ReplyPerson(ctx, &Message{Sender: SenderBase{UserID: d.UpgradeWindowID}}, text)
				}

				d.Logger.Infof("升级完成，当前版本: %s", VERSION.String())
				d.UpgradeWindowID = ""
				d.UpgradeEndpointID = ""
				d.MarkModified()
				d.Save(false)
				break
			}
		}()
	}

	d.ResetQuitInactiveCron()

	d.MarkModified()
}

func (d *Dice) rebuildParser(buffer string) *DiceRollParser {
	p := &DiceRollParser{Buffer: buffer}
	_ = p.Init()
	p.RollExpression.Init(512)
	return p
}

func (d *Dice) ExprEvalBase(buffer string, ctx *MsgContext, flags RollExtraFlags) (*VMResult, string, error) {
	parser := d.rebuildParser(buffer)
	parser.RollExpression.flags = flags // 千万记得在parse之前赋值
	err := parser.Parse()

	if flags.vmDepth > 64 {
		return nil, "", errors.New("E8: 递归次数超过上限")
	}

	if err == nil {
		parser.Execute()
		if parser.Error != nil {
			return nil, "", parser.Error
		}
		num, detail, errEval := parser.Evaluate(d, ctx)
		if errEval != nil {
			return nil, "", errEval
		}

		ret := VMResult{}
		ret.Value = num.Value
		ret.TypeID = num.TypeID
		ret.Parser = parser

		tks := parser.Tokens()
		// 注意，golang的string下标等同于[]byte下标，也就是说中文会被打断
		// parser里有一个[]rune类型的，但问题是他句尾带了一个endsymbol
		runeBuffer := []rune(buffer)
		lastToken := tks[len(tks)-1]
		ret.restInput = strings.TrimSpace(string(runeBuffer[lastToken.end:]))
		ret.Matched = strings.TrimSpace(string(runeBuffer[:lastToken.end]))
		return &ret, detail, nil
	}
	return nil, "", err
}

func (d *Dice) ExprEval(buffer string, ctx *MsgContext) (*VMResult, string, error) {
	return d.ExprEvalBase(buffer, ctx, RollExtraFlags{})
}

func (d *Dice) ExprTextBase(buffer string, ctx *MsgContext, flags RollExtraFlags) (*VMResult, string, error) {
	buffer = CompatibleReplace(ctx, buffer)

	// 隐藏的内置字符串符号 \x1e
	val, detail, err := d.ExprEvalBase("\x1e"+buffer+"\x1e", ctx, flags)
	if err != nil {
		fmt.Println("脚本执行出错: ", buffer, "->", err)
	}

	if err == nil && (val.TypeID == VMTypeString || val.TypeID == VMTypeNone) {
		return val, detail, err
	}

	return nil, "", errors.New("错误的表达式")
}

func (d *Dice) ExprText(buffer string, ctx *MsgContext) (string, string, error) {
	val, detail, err := d.ExprTextBase(buffer, ctx, RollExtraFlags{})

	if err == nil && (val.TypeID == VMTypeString || val.TypeID == VMTypeNone) {
		return val.Value.(string), detail, err
	}

	return "格式化错误:" + strconv.Quote(buffer), "", errors.New("错误的表达式")
}

// ExtFind 根据名称或别名查找扩展
func (d *Dice) ExtFind(s string) *ExtInfo {
	for _, i := range d.ExtList {
		// 名字匹配，优先级最高
		if i.Name == s {
			return i
		}
	}
	for _, i := range d.ExtList {
		// 别名匹配，优先级次之
		if slices.Contains(i.Aliases, s) {
			return i
		}
	}
	for _, i := range d.ExtList {
		// 忽略大小写匹配，优先级最低
		if strings.EqualFold(i.Name, s) || slices.Contains(i.Aliases, strings.ToLower(s)) {
			return i
		}
	}
	return nil
}

// ExtAliasToName 将扩展别名转换成主用名, 如果没有找到则返回原值
func (d *Dice) ExtAliasToName(s string) string {
	ext := d.ExtFind(s)
	if ext != nil {
		return ext.Name
	}
	return s
}

func (d *Dice) ExtRemove(ei *ExtInfo) bool {
	for _, i := range d.ImSession.ServiceAtNew {
		i.ExtInactive(ei)
	}

	for index, i := range d.ExtList {
		if i == ei {
			d.ExtList = append(d.ExtList[:index], d.ExtList[index+1:]...)
			return true
		}
	}

	return false
}

func (d *Dice) MasterRefresh() {
	m := map[string]bool{}
	var lst []string

	for _, i := range d.DiceMasters {
		if !m[i] {
			m[i] = true
			lst = append(lst, i)
		}
	}
	d.DiceMasters = lst
	d.MarkModified()
}

func (d *Dice) MasterAdd(uid string) {
	d.DiceMasters = append(d.DiceMasters, uid)
	d.MasterRefresh()
}

// MasterCheck 检查是否有Master权限.
//   - gid, uid: 群组和用户的统一ID(实际上并不区分哪个是哪个)
func (d *Dice) MasterCheck(gid, uid string) bool {
	for _, i := range d.DiceMasters {
		if i == uid || i == gid {
			return true
		}
	}
	return false
}

func (d *Dice) MasterRemove(uid string) bool {
	for index, i := range d.DiceMasters {
		if i == uid {
			d.DiceMasters = append(d.DiceMasters[:index], d.DiceMasters[index+1:]...)
			d.MarkModified()
			return true
		}
	}
	return false
}

func (d *Dice) UnlockCodeUpdate(force bool) {
	now := time.Now().Unix()
	// 大于20分钟重置
	if now-d.MasterUnlockCodeTime > 20*60 || force {
		d.MasterUnlockCode = ""
	}
	if d.MasterUnlockCode == "" {
		d.MasterUnlockCode = RandStringBytesMaskImprSrcSB(8)
		d.MasterUnlockCodeTime = now
	}
}

func (d *Dice) UnlockCodeVerify(code string) bool {
	d.UnlockCodeUpdate(false)
	return code == d.MasterUnlockCode
}

func (d *Dice) IsMaster(uid string) bool {
	for _, i := range d.DiceMasters {
		if i == uid {
			return true
		}
	}
	return false
}

// ApplyAliveNotice 存活消息(骰狗)
func (d *Dice) ApplyAliveNotice() {
	if d.Cron != nil && d.AliveNoticeEntry != 0 {
		d.Cron.Remove(d.AliveNoticeEntry)
	}
	if d.AliveNoticeEnable {
		entry, err := d.Cron.AddFunc(d.AliveNoticeValue, func() {
			d.NoticeForEveryEndpoint(fmt.Sprintf("存活, D100=%d", DiceRoll64(100)), false)
		})
		if err == nil {
			d.AliveNoticeEntry = entry
			d.Logger.Infof("创建存活确认消息成功")
		} else {
			d.Logger.Error("创建存活确认消息发生错误，可能是间隔设置有误:", err)
		}
	}
}

// GameSystemTemplateAdd 应用一个角色模板
func (d *Dice) GameSystemTemplateAdd(tmpl *GameSystemTemplate) bool {
	if _, exists := d.GameSystemMap.Load(tmpl.Name); !exists {
		d.GameSystemMap.Store(tmpl.Name, tmpl)
		// sn 从这里读取
		// set 时从这里读取对应System名字的模板

		// 同义词缓存
		tmpl.AliasMap = InitializeSyncMap[string, string]()
		alias := tmpl.Alias
		for k, v := range alias {
			for _, i := range v {
				tmpl.AliasMap.Store(strings.ToLower(i), k)
			}
			tmpl.AliasMap.Store(strings.ToLower(k), k)
		}
		return true
	}
	return false
}

var randSource = rand2.NewSource(uint64(time.Now().Unix()))

func DiceRoll(dicePoints int) int { //nolint:revive
	if dicePoints <= 0 {
		return 0
	}
	val := int(randSource.Uint64()%math.MaxInt32)%dicePoints + 1
	return val
}

func DiceRoll64(dicePoints int64) int64 { //nolint:revive
	if dicePoints == 0 {
		return 0
	}
	val := int64(randSource.Uint64()%math.MaxInt64)%dicePoints + 1
	return val
}

func CrashLog() {
	if r := recover(); r != nil {
		text := fmt.Sprintf("报错: %v\n堆栈: %v", r, string(debug.Stack()))
		now := time.Now()
		_ = os.WriteFile(fmt.Sprintf("崩溃日志_%s.txt", now.Format("20060201_150405")), []byte(text), 0o644)
		panic(r)
	}
}

func ErrorLogAndContinue(d *Dice) {
	if r := recover(); r != nil {
		d.Logger.Errorf("报错: %v 堆栈: %v", r, string(debug.Stack()))
		d.Logger.Infof("已从报错中恢复，建议将此错误回报给开发者")
	}
}

var chsS2T = sat.DefaultDict()

func (d *Dice) ResetQuitInactiveCron() {
	dm := d.Parent
	if d.quitInactiveCronEntry > 0 {
		dm.Cron.Remove(d.quitInactiveCronEntry)
		d.quitInactiveCronEntry = 0
	}

	if d.QuitInactiveThreshold > 0 {
		var err error
		d.quitInactiveCronEntry, err = dm.Cron.AddFunc("0 4 * * *", func() {
			thr := time.Now().Add(-d.QuitInactiveThreshold)
			hint := thr.Add(d.QuitInactiveThreshold / 10) // 进入退出判定线的9/10开始提醒
			d.ImSession.LongTimeQuitInactiveGroup(thr, hint,
				int(d.QuitInactiveBatchWait),
				int(d.QuitInactiveBatchSize))
		})
		if err != nil {
			d.Logger.Errorf("创建自动清理群聊cron任务失败: %v", err)
		}
	}
}
