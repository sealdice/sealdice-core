package dice

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/go-creed/sat"
	wr "github.com/mroth/weightedrand"
	"github.com/robfig/cron/v3"
	ds "github.com/sealdice/dicescript"
	"github.com/tidwall/buntdb"
	"go.uber.org/zap"
	rand2 "golang.org/x/exp/rand" //nolint:staticcheck // against my better judgment, but this was mandated due to a strongly held opinion from you know who

	"sealdice-core/dice/events"
	"sealdice-core/logger"
	"sealdice-core/utils/dboperator/engine"
	"sealdice-core/utils/public_dice"
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
	JSLoopVersion int64                                                                  // Loop版本号
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
	Name    string   `jsbind:"name"    json:"name"    yaml:"name"` // 名字
	Aliases []string `jsbind:"aliases" json:"aliases" yaml:"-"`    // 别名
	Version string   `jsbind:"version" json:"version" yaml:"-"`    // 版本
	// 作者
	// 更新时间
	AutoActive      bool      `jsbind:"autoActive" json:"-" yaml:"-"` // 是否自动开启
	CmdMap          CmdMapCls `jsbind:"cmdMap"     json:"-" yaml:"-"` // 指令集合
	Brief           string    `json:"-"            yaml:"-"`
	ActiveOnPrivate bool      `json:"-"            yaml:"-"`

	DefaultSetting *ExtDefaultSettingItem `json:"-" yaml:"-"` // 默认配置

	Author       string   `jsbind:"author" json:"-" yaml:"-"`
	ConflictWith []string `json:"-"        yaml:"-"`
	Official     bool     `json:"-"        yaml:"-"` // 官方插件

	dice          *Dice
	IsJsExt       bool          `json:"-"`
	JSLoopVersion int64         `json:"-"`
	Source        *JsScriptInfo `json:"-" yaml:"-"`
	Storage       *buntdb.DB    `json:"-" yaml:"-"`
	// 为Storage使用互斥锁,并根据ID佬的说法修改为合适的名称
	dbMu sync.Mutex `yaml:"-"` // 互斥锁
	init bool       `yaml:"-"` // 标记Storage是否已初始化

	// 定时任务列表，用于避免 task 失去引用
	taskList []*JsScriptTask `json:"-" yaml:"-"`

	OnNotCommandReceived func(ctx *MsgContext, msg *Message)                        `jsbind:"onNotCommandReceived" json:"-" yaml:"-"` // 指令过滤后剩下的
	OnCommandOverride    func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) bool `json:"-"                      yaml:"-"`          // 覆盖指令行为

	OnCommandReceived   func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) `jsbind:"onCommandReceived"   json:"-" yaml:"-"`
	OnMessageReceived   func(ctx *MsgContext, msg *Message)                   `jsbind:"onMessageReceived"   json:"-" yaml:"-"`
	OnMessageSend       func(ctx *MsgContext, msg *Message, flag string)      `jsbind:"onMessageSend"       json:"-" yaml:"-"`
	OnMessageDeleted    func(ctx *MsgContext, msg *Message)                   `jsbind:"onMessageDeleted"    json:"-" yaml:"-"`
	OnMessageEdit       func(ctx *MsgContext, msg *Message)                   `jsbind:"onMessageEdit"       json:"-" yaml:"-"`
	OnGroupJoined       func(ctx *MsgContext, msg *Message)                   `jsbind:"onGroupJoined"       json:"-" yaml:"-"`
	OnGroupMemberJoined func(ctx *MsgContext, msg *Message)                   `jsbind:"onGroupMemberJoined" json:"-" yaml:"-"`
	OnGuildJoined       func(ctx *MsgContext, msg *Message)                   `jsbind:"onGuildJoined"       json:"-" yaml:"-"`
	OnBecomeFriend      func(ctx *MsgContext, msg *Message)                   `jsbind:"onBecomeFriend"      json:"-" yaml:"-"`
	OnPoke              func(ctx *MsgContext, event *events.PokeEvent)        `jsbind:"onPoke"              json:"-" yaml:"-"` // 戳一戳
	OnGroupLeave        func(ctx *MsgContext, event *events.GroupLeaveEvent)  `jsbind:"onGroupLeave"        json:"-" yaml:"-"` // 群成员被踢出
	GetDescText         func(i *ExtInfo) string                               `jsbind:"getDescText"         json:"-" yaml:"-"`
	IsLoaded            bool                                                  `jsbind:"isLoaded"            json:"-" yaml:"-"`
	OnLoad              func()                                                `jsbind:"onLoad"              json:"-" yaml:"-"`
}

// RootConfig TODO：历史遗留问题，由于不输出DICE日志效果过差，已经抹除日志输出选项，剩余两个选项，私以为可以想办法也抹除掉。
type RootConfig struct { //nolint:revive
	Name    string `yaml:"name"`    // 名称，默认为default
	DataDir string `yaml:"dataDir"` // 数据路径，为./data/{name}，例如data/default
}

type ExtDefaultSettingItem struct {
	Name            string          `json:"name"            yaml:"name"`
	AutoActive      bool            `json:"autoActive"      yaml:"autoActive"`           // 是否自动开启
	DisabledCommand map[string]bool `json:"disabledCommand" yaml:"disabledCommand,flow"` // 实际为set
	ExtItem         *ExtInfo        `json:"-"               yaml:"-"`
	Loaded          bool            `json:"loaded"          yaml:"-"` // 当前插件是否正确加载. serve.yaml不保存, 前端请求时提供
}

type ExtDefaultSettingItemSlice []*ExtDefaultSettingItem

type JsLoopManager struct {
	loop     *eventloop.EventLoop
	loopLock sync.RWMutex
	version  int64
}

func NewJsLoopManager() *JsLoopManager {
	return &JsLoopManager{
		loop:     nil,
		loopLock: sync.RWMutex{},
		version:  0,
	}
}

// GetLoop 通过版本号获取 loop，如果版本号不匹配则返回错误
func (m *JsLoopManager) GetLoop(expectedVersion int64) (*eventloop.EventLoop, error) {
	m.loopLock.RLock()
	defer m.loopLock.RUnlock()

	if m.version != expectedVersion {
		return nil, fmt.Errorf("version mismatch: expected %d, current %d", expectedVersion, m.version)
	}

	return m.loop, nil
}

func (m *JsLoopManager) GetWebLoop() *eventloop.EventLoop {
	m.loopLock.RLock()
	defer m.loopLock.RUnlock()
	// 给WEB用，不需要管是哪个版本的Loop，是最新的就行
	return m.loop
}

// SetLoop 写入新的 loop 并自动递增版本号
func (m *JsLoopManager) SetLoop(newLoop *eventloop.EventLoop) int64 {
	m.loopLock.Lock()
	defer m.loopLock.Unlock()

	// 停止旧的 loop（如果存在）
	if m.loop != nil {
		m.loop.Terminate()
	}

	// 设置新的 loop 并递增版本号
	m.loop = newLoop
	m.version++
	return m.version
}

// 强制coc7排序在较前位置

func (x ExtDefaultSettingItemSlice) Len() int           { return len(x) }
func (x ExtDefaultSettingItemSlice) Less(i, _ int) bool { return x[i].Name == "coc7" }
func (x ExtDefaultSettingItemSlice) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }

type Dice struct {
	// 由于被导出的原因，暂时不迁移至 config
	ImSession *IMSession `jsbind:"imSession" json:"-" yaml:"imSession"`

	CmdMap          CmdMapCls              `json:"-" yaml:"-"`
	ExtList         []*ExtInfo             `yaml:"-"`
	RollParser      *DiceRollParser        `yaml:"-"`
	LastUpdatedTime int64                  `yaml:"-"`
	TextMap         map[string]*wr.Chooser `yaml:"-"`
	BaseConfig      BaseConfig             `yaml:"-"`
	// DBData          *gorm.DB               `yaml:"-"` // 数据库对象
	// DBLogs          *gorm.DB               `yaml:"-"` // 数据库对象
	DBOperator    engine.DatabaseOperator
	Logger        *zap.SugaredLogger `yaml:"-"` // 日志
	LogWriter     *logger.UIWriter   `yaml:"-"` // 用于api的log对象
	IsDeckLoading bool               `yaml:"-"` // 正在加载中

	// 由于被导出的原因，暂时不迁移至 config
	DeckList      []*DeckInfo `jsbind:"deckList"      yaml:"deckList"`      // 牌堆信息
	CommandPrefix []string    `jsbind:"commandPrefix" yaml:"commandPrefix"` // 指令前导
	DiceMasters   []string    `jsbind:"diceMasters"   yaml:"diceMasters"`   // 骰主设置，需要格式: 平台:帐号

	MasterUnlockCode     string         `json:"masterUnlockCode"     yaml:"-"` // 解锁码，每20分钟变化一次，使用后立即变化
	MasterUnlockCodeTime int64          `json:"masterUnlockCodeTime" yaml:"-"`
	CustomReplyConfig    []*ReplyConfig `json:"-"                    yaml:"-"`

	TextMapRaw        TextTemplateWithWeightDict `yaml:"-"`
	TextMapHelpInfo   TextTemplateWithHelpDict   `yaml:"-"`
	TextMapCompatible TextTemplateCompatibleDict `yaml:"-"` // 兼容信息，格式 { "COC:测试": { "回复A": {...}, "回复B": ... } } 这样字符串可以不占据新的内存

	ConfigManager *ConfigManager `yaml:"-"`
	Parent        *DiceManager   `yaml:"-"`

	CocExtraRules    map[int]*CocRuleInfo `json:"cocExtraRules" yaml:"-"`
	Cron             *cron.Cron           `json:"-"             yaml:"-"`
	AliveNoticeEntry cron.EntryID         `json:"-"             yaml:"-"`
	JsPrinter        *PrinterFunc         `json:"-"             yaml:"-"`

	// JsLoop           *eventloop.EventLoop `yaml:"-" json:"-"`
	ExtLoopManager   *JsLoopManager  `json:"-" yaml:"-"`
	JsScriptList     []*JsScriptInfo `json:"-" yaml:"-"`
	JsScriptCron     *cron.Cron      `json:"-" yaml:"-"`
	JsScriptCronLock *sync.Mutex     `json:"-" yaml:"-"`
	// 重载使用的互斥锁
	JsReloadLock sync.Mutex `json:"-" yaml:"-"`
	// 内置脚本摘要表，用于判断内置脚本是否有更新
	JsBuiltinDigestSet map[string]bool `json:"-" yaml:"-"`
	// 当前在加载的脚本路径，用于关联 jsScriptInfo 和 ExtInfo
	JsLoadingScript *JsScriptInfo `json:"-" yaml:"-"`

	// 游戏系统规则模板
	GameSystemMap *SyncMap[string, *GameSystemTemplate] `json:"-" yaml:"-"`

	RunAfterLoaded []func() `json:"-" yaml:"-"`

	deckCommandItemsList DeckCommandListItems // 牌堆key信息，辅助作为模糊搜索使用

	UIEndpoint *EndPointInfo `json:"-" yaml:"-"` // UI Endpoint

	CensorManager *CensorManager `json:"-" yaml:"-"`

	AttrsManager *AttrsManager `json:"-" yaml:"-"`

	Config Config `json:"-" yaml:"-"`

	AdvancedConfig AdvancedConfig `json:"-" yaml:"-"`

	PublicDice        *public_dice.PublicDiceClient `json:"-" yaml:"-"`
	PublicDiceTimerId cron.EntryID                  `json:"-" yaml:"-"`

	ContainerMode bool `json:"-" yaml:"-"` // 容器模式：禁用内置适配器，不允许使用内置Lagrange和旧的内置Gocq

	IsAlreadyLoadConfig bool `yaml:"-"` // 如果在loads前崩溃，那么不写入配置，防止覆盖为空的

	// 用于检查是否需要插入到数据库的哈希表 150因为没有对应插入 到时候这个就没用了
	SaveDatabaseInsertCheckMapFlag sync.Once                `json:"-" yaml:"-"`
	SaveDatabaseInsertCheckMap     *SyncMap[string, string] `json:"-" yaml:"-"`

	/* 扩展商店 */
	StoreManager *StoreManager `json:"-" yaml:"-"`
}

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

func (d *Dice) Init(operator engine.DatabaseOperator, uiWriter *logger.UIWriter) {
	loggerInstance := logger.M()
	d.Logger = loggerInstance
	d.LogWriter = uiWriter

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
	d.DBOperator = operator
	d.AttrsManager = &AttrsManager{}
	d.AttrsManager.Init(d)

	(&d.Config).BanList = &BanListInfo{Parent: d}
	(&d.Config).BanList.Init()

	initVerify()

	d.BaseConfig.CommandCompatibleMode = true
	// Pinenutn: 预先初始化对应的SyncMap
	d.ImSession = &IMSession{}
	d.ImSession.Parent = d
	d.ImSession.ServiceAtNew = new(SyncMap[string, *GroupInfo])
	d.CmdMap = CmdMapCls{}
	d.GameSystemMap = new(SyncMap[string, *GameSystemTemplate])
	d.ConfigManager = NewConfigManager(filepath.Join(d.BaseConfig.DataDir, "configs", "plugin-configs.json"))
	err = d.ConfigManager.Load()
	if err != nil {
		loggerInstance.Error("Failed to load plugin configs: ", err)
	}

	d.registerCoreCommands()
	d.RegisterBuiltinExt()
	d.loads()
	d.loadAdvanced()
	(&d.Config).BanList.Loads()
	(&d.Config).BanList.AfterLoads()
	d.IsAlreadyLoadConfig = true

	if d.Config.EnableCensor {
		d.NewCensorManager()
	}

	go d.PublicDiceSetup()

	go d.StoreSetup()

	// 创建js运行时
	if d.Config.JsEnable {
		loggerInstance.Info("js扩展支持：开启")
		d.ExtLoopManager = NewJsLoopManager() // 此时不初始化loop，Init才初始化哦
		d.JsInit()
	} else {
		loggerInstance.Info("js扩展支持：关闭")
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
				loggerInstance.Error("RunAfterLoaded 报错: ", r)
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
				// if count%2 == 0 {
				//	if err := model.FlushWAL(d.DBData); err != nil {
				//		log.Error("Failed to flush WAL: ", err)
				//	}
				//	if err := model.FlushWAL(d.DBLogs); err != nil {
				//		log.Error("Failed to flush WAL: ", err)
				//	}
				//	if d.CensorManager != nil && d.CensorManager.DB != nil {
				//		if err := model.FlushWAL(d.CensorManager.DB); err != nil {
				//			log.Error("Failed to flush WAL: ", err)
				//		}
				//	}
				// }
			}
		}
	}
	go autoSave()

	refreshGroupInfo := func() {
		t := time.NewTicker(35 * time.Second)
		defer func() {
			// 防止报错
			if r := recover(); r != nil {
				loggerInstance.Error(r)
			}
		}()

		for {
			<-t.C

			// 自动更新群信息
			for _, i := range d.ImSession.EndPoints {
				if i.Enable {
					// Pinenutn: Range模板 ServiceAtNew重构代码
					d.ImSession.ServiceAtNew.Range(func(key string, groupInfo *GroupInfo) bool {
						// Pinenutn: ServiceAtNew重构
						// TODO: 注意这里的Active可能不需要改
						if !strings.HasPrefix(key, "PG-") && groupInfo.Active {
							diceID := i.UserID
							now := time.Now().Unix()

							// 上次被人使用小于60s
							if now-groupInfo.RecentDiceSendTime < 60 {
								// 在群内存在，且开启时
								if groupInfo.DiceIDExistsMap.Exists(diceID) && groupInfo.DiceIDActiveMap.Exists(diceID) {
									i.Adapter.GetGroupInfoAsync(key)
								}
							}
						}
						return true
					})
				}
			}
		}
	}
	go refreshGroupInfo()

	d.ApplyAliveNotice()
	if d.Config.JsEnable {
		d.JsBuiltinDigestSet = make(map[string]bool)
		d.JsLoadScripts()
	} else {
		loggerInstance.Info("js扩展支持已关闭，跳过js脚本的加载")
	}

	if d.Config.UpgradeWindowID != "" {
		go func() {
			defer ErrorLogAndContinue(d)

			var ep *EndPointInfo
			for _, _ep := range d.ImSession.EndPoints {
				if _ep.ID == d.Config.UpgradeEndpointID {
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
				isGroup := strings.Contains(d.Config.UpgradeWindowID, "-Group:")
				if isGroup {
					ReplyGroup(ctx, &Message{GroupID: d.Config.UpgradeWindowID}, text)
				} else {
					ReplyPerson(ctx, &Message{Sender: SenderBase{UserID: d.Config.UpgradeWindowID}}, text)
				}

				loggerInstance.Infof("升级完成，当前版本: %s", VERSION.String())
				(&d.Config).UpgradeWindowID = ""
				(&d.Config).UpgradeEndpointID = ""
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

type VMResultV2 struct {
	ds.VMValue
	vm *ds.Context
}

func (d *Dice) _ExprEvalBaseV1(buffer string, ctx *MsgContext, flags RollExtraFlags) (*VMResult, string, error) {
	parser := d.rebuildParser(buffer)
	parser.flags = flags // 千万记得在parse之前赋值
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

func (d *Dice) _ExprTextBaseV1(buffer string, ctx *MsgContext, flags RollExtraFlags) (*VMResult, string, error) {
	buffer = CompatibleReplace(ctx, buffer)

	// 隐藏的内置字符串符号 \x1e
	val, detail, err := d._ExprEvalBaseV1("\x1e"+buffer+"\x1e", ctx, flags)
	if err != nil {
		d.Logger.Warnf("脚本执行出错: %s -> %v", buffer, err)
	}

	if err == nil && (val.TypeID == VMTypeString || val.TypeID == VMTypeNone) {
		return val, detail, err
	}

	return nil, "", errors.New("错误的表达式")
}

func (d *Dice) _ExprTextV1(buffer string, ctx *MsgContext) (string, string, error) {
	val, detail, err := d._ExprTextBaseV1(buffer, ctx, RollExtraFlags{})

	if err == nil && (val.TypeID == VMTypeString || val.TypeID == VMTypeNone) {
		return val.Value.(string), detail, err
	}

	textQuote := strconv.Quote(buffer) // 主意，这个返回中对err进行改写是错误的，但是历史代码，不做修改
	return "格式化错误V1:" + textQuote, "", errors.New("格式化错误V1:" + textQuote)
}

// ExtFind 根据名称或别名查找扩展
func (d *Dice) ExtFind(s string, fromJS bool) *ExtInfo {
	find := func(_ string) *ExtInfo {
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
	ext := find(s)
	if ext != nil && ext.Official && fromJS {
		// return a copy of the official extension
		cmdMap := make(CmdMapCls, len(ext.CmdMap))
		for s2, info := range ext.CmdMap {
			cmdMap[s2] = &CmdItemInfo{
				Name:                    info.Name,
				ShortHelp:               info.ShortHelp,
				Help:                    info.Help,
				HelpFunc:                info.HelpFunc,
				AllowDelegate:           info.AllowDelegate,
				DisabledInPrivate:       info.DisabledInPrivate,
				EnableExecuteTimesParse: info.EnableExecuteTimesParse,
				IsJsSolveFunc:           info.IsJsSolveFunc,
				Solve:                   info.Solve,
				Raw:                     info.Raw,
				CheckCurrentBotOn:       info.CheckCurrentBotOn,
				CheckMentionOthers:      info.CheckMentionOthers,
			}
		}
		return &ExtInfo{
			Name:       ext.Name,
			Aliases:    ext.Aliases,
			Author:     ext.Author,
			Version:    ext.Version,
			AutoActive: ext.AutoActive,
			CmdMap:     cmdMap,
			Brief:      ext.Brief,
			Official:   ext.Official,
		}
	}
	return ext
}

// ExtAliasToName 将扩展别名转换成主用名, 如果没有找到则返回原值
func (d *Dice) ExtAliasToName(s string) string {
	ext := d.ExtFind(s, false)
	if ext != nil {
		return ext.Name
	}
	return s
}

func (d *Dice) ExtRemove(ei *ExtInfo) bool {
	// Pinenutn: Range模板 ServiceAtNew重构代码
	d.ImSession.ServiceAtNew.Range(func(key string, groupInfo *GroupInfo) bool {
		// Pinenutn: ServiceAtNew重构
		// 系统移除扩展时，不应记录为用户禁用或破坏快照
		groupInfo.ExtInactiveSystem(ei)
		return true
	})

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
	if d.Config.AliveNoticeEnable {
		entry, err := d.Cron.AddFunc((&d.Config).AliveNoticeValue, func() {
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

// GameSystemTemplateAddEx 应用一个角色模板
func (d *Dice) GameSystemTemplateAddEx(tmpl *GameSystemTemplate, overwrite bool) bool {
	if tmpl == nil {
		return false
	}
	_, exists := d.GameSystemMap.Load(tmpl.Name)
	if !exists || overwrite {
		tmpl.Init()
		d.GameSystemMap.Store(tmpl.Name, tmpl)
		return true
	}
	return false
}

// GameSystemTemplateAdd 应用一个角色模板，当已存在时返回false
func (d *Dice) GameSystemTemplateAdd(tmpl *GameSystemTemplate) bool {
	return d.GameSystemTemplateAddEx(tmpl, false)
}

// generateRandSeed 生成一个随机种子，由当前时间戳、对象指针、进程ID和堆栈信息组成
func generateRandSeed() uint64 {
	timestamp := time.Now().UnixNano()

	type tempObj struct{ val int }
	obj := tempObj{val: 42}
	objPtr := uint64(uintptr(unsafe.Pointer(&obj)))

	pid := uint64(os.Getpid())

	buf := make([]byte, 1024)
	n := runtime.Stack(buf, true)
	stackInfo := buf[:n]

	h := fnv.New64a()

	_ = binary.Write(h, binary.LittleEndian, timestamp)

	_ = binary.Write(h, binary.LittleEndian, objPtr)

	_ = binary.Write(h, binary.LittleEndian, pid)

	_, _ = h.Write(stackInfo)

	return h.Sum64()
}

var randSource = rand2.NewSource(generateRandSeed()).(*rand2.PCGSource)

func DiceRoll(dicePoints int) int { //nolint:revive
	if dicePoints <= 0 {
		return 0
	}
	val := ds.Roll(randSource, ds.IntType(dicePoints), 0)
	return int(val)
}

func DiceRoll64x(src *rand2.PCGSource, dicePoints int64) int64 { //nolint:revive
	if src == nil {
		src = randSource
	}
	val := ds.Roll(src, ds.IntType(dicePoints), 0)
	return int64(val)
}

func DiceRoll64(dicePoints int64) int64 { //nolint:revive
	return DiceRoll64x(nil, dicePoints)
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
var taskId cron.EntryID
var quitMutex sync.Mutex

func (d *Dice) ResetQuitInactiveCron() {
	// TODO: 这里加锁是否有必要？
	quitMutex.Lock()
	defer quitMutex.Unlock()
	dm := d.Parent
	if d.Config.quitInactiveCronEntry > 0 {
		dm.Cron.Remove(d.Config.quitInactiveCronEntry)
		(&d.Config).quitInactiveCronEntry = DefaultConfig.quitInactiveCronEntry
	}
	// 如果退群功能开启，那么设定退群的Cron
	if d.Config.QuitInactiveThreshold > 0 {
		duration := time.Duration(d.Config.QuitInactiveBatchWait) * time.Minute
		// 每隔上面的退群时间，执行一次函数
		if taskId != 0 {
			dm.Cron.Remove(taskId)
		}
		taskId = dm.Cron.Schedule(cron.Every(duration), cron.FuncJob(func() {
			thr := time.Now().Add(-d.Config.QuitInactiveThreshold)
			// 进入退出判定线的9/10开始提醒, 但是目前来看，原版退群只有一个提示，提示会被大量刷屏然后消失不见。同时并没有告知对应的群
			// 或许也不应该告知对应的群，因为群可能被解散了，大量告知容易出问题？
			// hint := thr.Add(d.Config.QuitInactiveThreshold / 10)
			// Threshold > 0 时才应当进行退群，不然改了设置之后会疯狂退群
			if d.Config.QuitInactiveThreshold > 0 {
				d.ImSession.LongTimeQuitInactiveGroupReborn(thr, int(d.Config.QuitInactiveBatchSize))
			}
		}))
		d.Logger.Infof("退群功能已启动，每 %s 执行一次退群判定", duration.String())
		// Cancel the task
	} else if taskId != 0 {
		dm.Cron.Remove(taskId)
		taskId = 0
	}
}

func (d *Dice) PublicDiceEndpointRefresh() {
	cfg := &d.Config.PublicDiceConfig

	var endpointItems []*public_dice.Endpoint
	for _, i := range d.ImSession.EndPoints {
		if !i.IsPublic {
			continue
		}
		endpointItems = append(endpointItems, &public_dice.Endpoint{
			Platform: i.Platform,
			UID:      i.UserID,
			IsOnline: i.State == 1,
		})
	}

	_, code := d.PublicDice.EndpointUpdate(&public_dice.EndpointUpdateRequest{
		DiceID:    cfg.ID,
		Endpoints: endpointItems,
	}, GenerateVerificationKeyForPublicDice)
	if code != 200 {
		d.Logger.Warn("[公骰]无法通过服务器校验，不再进行更新")
		return
	}
}

func (d *Dice) PublicDiceInfoRegister() {
	cfg := &d.Config.PublicDiceConfig
	pd, code := d.PublicDice.Register(&public_dice.RegisterRequest{
		ID:    cfg.ID,
		Name:  cfg.Name,
		Brief: cfg.Brief,
		Note:  cfg.Note,
	}, GenerateVerificationKeyForPublicDice)
	if code != 200 {
		d.Logger.Warn("[公骰]无法通过服务器校验，不再进行骰号注册")
		return
	}
	// ID为空时才将注册好的ID覆写配置
	if pd.Item.ID != "" && cfg.ID == "" {
		cfg.ID = pd.Item.ID
	}
}

func (d *Dice) PublicDiceSetupTick() {
	cfg := &d.Config.PublicDiceConfig

	doTickUpdate := func() {
		if !cfg.Enable {
			d.Cron.Remove(d.PublicDiceTimerId)
			return
		}
		var tickEndpointItems []*public_dice.TickEndpoint
		for _, i := range d.ImSession.EndPoints {
			if !i.IsPublic {
				continue
			}
			tickEndpointItems = append(tickEndpointItems, &public_dice.TickEndpoint{
				UID:      i.UserID,
				IsOnline: i.State == 1,
			})
		}
		d.PublicDice.TickUpdate(&public_dice.TickUpdateRequest{
			ID:        cfg.ID,
			Endpoints: tickEndpointItems,
		}, GenerateVerificationKeyForPublicDice)
	}

	if d.PublicDiceTimerId != 0 {
		d.Cron.Remove(d.PublicDiceTimerId)
	}

	go func() {
		// 20s后进行第一次调用，此后3min进行一次更新
		time.Sleep(20 * time.Second)
		doTickUpdate()
	}()

	d.PublicDiceTimerId, _ = d.Cron.AddFunc("@every 3m", doTickUpdate)
}

func (d *Dice) PublicDiceSetup() {
	d.PublicDice = public_dice.NewClient("https://api.weizaima.com", "")

	cfg := &d.Config.PublicDiceConfig
	if !cfg.Enable {
		return
	}
	d.PublicDiceInfoRegister()
	d.PublicDiceEndpointRefresh()
	d.PublicDiceSetupTick()
}

func (d *Dice) StoreSetup() {
	d.StoreManager = NewStoreManager(d)
}
