package dice

import (
	"errors"
	"fmt"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-creed/sat"
	"github.com/jmoiron/sqlx"
	wr "github.com/mroth/weightedrand"
	"github.com/robfig/cron/v3"
	"github.com/tidwall/buntdb"
	"go.uber.org/zap"
	"math/rand"
	"os"
	"path/filepath"
	"runtime/debug"
	"sealdice-core/dice/logger"
	"sealdice-core/dice/model"
	"strconv"
	"strings"
	"time"
)

var APPNAME = "SealDice"
var VERSION = "1.2.7-dev v20230719"

// var VERSION_CODE = int64(1001000) // 991404
var VERSION_CODE = int64(1002006) // 坏了，1.1的版本号标错了，标成了1.10.0
var APP_BRANCH = ""

type CmdExecuteResult struct {
	Matched  bool // 是否是指令
	Solved   bool `jsbind:"solved"` // 是否响应此指令
	ShowHelp bool `jsbind:"showHelp"`
}

type CmdItemInfo struct {
	Name              string                    `jsbind:"name"`
	ShortHelp         string                    // 短帮助，格式是 .xxx a b // 说明
	Help              string                    `jsbind:"help"`              // 长帮助，带换行的较详细说明
	HelpFunc          func(isShort bool) string `jsbind:"helpFunc"`          // 函数形式帮助，存在时优先于其他
	AllowDelegate     bool                      `jsbind:"allowDelegate"`     // 允许代骰
	DisabledInPrivate bool                      `jsbind:"disabledInPrivate"` // 私聊不可用

	IsJsSolveFunc bool
	Solve         func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult `jsbind:"solve"`
	//Keywords []string // 其他帮助关键字
	//ChopWords     []string

	Raw                bool `jsbind:"raw"`                // 高级模式。默认模式下行为是：需要在当前群/私聊开启，或@自己时生效(需要为第一个@目标)
	CheckCurrentBotOn  bool `jsbind:"checkCurrentBotOn"`  // 是否检查当前可用状况，包括群内可用和是私聊两种方式，如失败不进入solve
	CheckMentionOthers bool `jsbind:"checkMentionOthers"` // 是否检查@了别的骰子，如失败不进入solve
}

type CmdMapCls map[string]*CmdItemInfo

//type ExtInfoStorage interface {
//
//}

type ExtInfo struct {
	Name    string `yaml:"name" json:"name" jsbind:"name"`    // 名字
	Version string `yaml:"-" json:"version" jsbind:"version"` // 版本
	// 作者
	// 更新时间
	AutoActive      bool      `yaml:"-" json:"-" jsbind:"autoActive"` // 是否自动开启
	CmdMap          CmdMapCls `yaml:"-" json:"-" jsbind:"cmdMap"`     // 指令集合
	Brief           string    `yaml:"-" json:"-"`
	ActiveOnPrivate bool      `yaml:"-" json:"-"`

	DefaultSetting *ExtDefaultSettingItem `yaml:"-" json:"-"` // 默认配置

	Author       string   `yaml:"-" json:"-" jsbind:"author"`
	ConflictWith []string `yaml:"-" json:"-"`
	//activeInSession bool; // 在当前会话中开启

	dice    *Dice
	IsJsExt bool       `json:"-"`
	Storage *buntdb.DB `yaml:"-"  json:"-"`
	//Storage ExtInfoStorage `yaml:"-" jsbind:"storage"`

	OnNotCommandReceived func(ctx *MsgContext, msg *Message)                        `yaml:"-" json:"-" jsbind:"onNotCommandReceived"` // 指令过滤后剩下的
	OnCommandOverride    func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) bool `yaml:"-" json:"-"`                               // 覆盖指令行为

	OnCommandReceived func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) `yaml:"-" json:"-" jsbind:"onCommandReceived"`
	OnMessageReceived func(ctx *MsgContext, msg *Message)                   `yaml:"-" json:"-" jsbind:"onMessageReceived"`
	OnMessageSend     func(ctx *MsgContext, msg *Message, flag string)      `yaml:"-" json:"-" jsbind:"onMessageSend"`
	OnMessageDeleted  func(ctx *MsgContext, msg *Message)                   `yaml:"-" json:"-" jsbind:"onMessageDeleted"`
	OnGroupJoined     func(ctx *MsgContext, msg *Message)                   `yaml:"-" json:"-" jsbind:"onGroupJoined"`
	OnBecomeFriend    func(ctx *MsgContext, msg *Message)                   `yaml:"-" json:"-" jsbind:"onBecomeFriend"`
	GetDescText       func(i *ExtInfo) string                               `yaml:"-" json:"-" jsbind:"getDescText"`
	IsLoaded          bool                                                  `yaml:"-" json:"-" jsbind:"isLoaded"`
	OnLoad            func()                                                `yaml:"-" json:"-" jsbind:"onLoad"`
}

type DiceConfig struct {
	Name       string `yaml:"name"`       // 名称，默认为default
	DataDir    string `yaml:"dataDir"`    // 数据路径，为./data/{name}，例如data/default
	IsLogPrint bool   `yaml:"isLogPrint"` // 是否在控制台打印log
}

type ExtDefaultSettingItem struct {
	Name            string          `yaml:"name" json:"name"`
	AutoActive      bool            `yaml:"autoActive" json:"autoActive"`                // 是否自动开启
	DisabledCommand map[string]bool `yaml:"disabledCommand,flow" json:"disabledCommand"` // 实际为set
	ExtItem         *ExtInfo        `yaml:"-" json:"-"`
}

type ExtDefaultSettingItemSlice []*ExtDefaultSettingItem

// 强制coc7排序在较前位置

func (x ExtDefaultSettingItemSlice) Len() int           { return len(x) }
func (x ExtDefaultSettingItemSlice) Less(i, j int) bool { return x[i].Name == "coc7" }
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
	NoticeIds               []string               `yaml:"noticeIds"`                            // 通知ID
	OnlyLogCommandInGroup   bool                   `yaml:"onlyLogCommandInGroup"`                // 日志中仅记录命令
	OnlyLogCommandInPrivate bool                   `yaml:"onlyLogCommandInPrivate"`              // 日志中仅记录命令
	VersionCode             int                    `json:"versionCode"`                          // 版本ID(配置文件)
	MessageDelayRangeStart  float64                `yaml:"messageDelayRangeStart"`               // 指令延迟区间
	MessageDelayRangeEnd    float64                `yaml:"messageDelayRangeEnd"`
	WorkInQQChannel         bool                   `yaml:"workInQQChannel"`
	QQChannelAutoOn         bool                   `yaml:"QQChannelAutoOn"`     // QQ频道中自动开启(默认不开)
	QQChannelLogMessage     bool                   `yaml:"QQChannelLogMessage"` // QQ频道中记录消息(默认不开)
	QQEnablePoke            bool                   `yaml:"QQEnablePoke"`        // 启用戳一戳
	RateLimitEnabled        bool                   `yaml:"rateLimitEnabled"`    // 启用频率限制 (刷屏限制)
	TextCmdTrustOnly        bool                   `yaml:"textCmdTrustOnly"`    // 只允许信任用户或master使用text指令
	UILogLimit              int64                  `yaml:"UILogLimit"`
	FriendAddComment        string                 `yaml:"friendAddComment"` // 加好友验证信息
	MasterUnlockCode        string                 `yaml:"-"`                // 解锁码，每20分钟变化一次，使用后立即变化
	MasterUnlockCodeTime    int64                  `yaml:"-"`
	CustomReplyConfigEnable bool                   `yaml:"customReplyConfigEnable"`
	CustomReplyConfig       []*ReplyConfig         `yaml:"-"`
	AutoReloginEnable       bool                   `yaml:"autoReloginEnable"`    // 启用自动重新登录
	RefuseGroupInvite       bool                   `yaml:"refuseGroupInvite"`    // 拒绝加入新群
	UpgradeWindowId         string                 `yaml:"upgradeWindowId"`      // 执行升级指令的窗口
	UpgradeEndpointId       string                 `yaml:"upgradeEndpointId"`    // 执行升级指令的端点
	BotExtFreeSwitch        bool                   `yaml:"botExtFreeSwitch"`     // 允许任意人员开关: 否则邀请者、群主、管理员、master有权限
	TrustOnlyMode           bool                   `yaml:"trustOnlyMode"`        // 只有信任的用户/master可以拉群和使用
	AliveNoticeEnable       bool                   `yaml:"aliveNoticeEnable"`    // 定时通知
	AliveNoticeValue        string                 `yaml:"aliveNoticeValue"`     // 定时通知间隔
	ReplyDebugMode          bool                   `yaml:"replyDebugMode"`       // 回复调试
	PlayerNameWrapEnable    bool                   `yaml:"playerNameWrapEnable"` // 启用玩家名称外框

	HelpMasterInfo      string `yaml:"helpMasterInfo" jsbind:"helpMasterInfo"`           // help中骰主信息
	HelpMasterLicense   string `yaml:"helpMasterLicense" jsbind:"helpMasterLicense"`     // help中使用协议
	DefaultCocRuleIndex int64  `yaml:"defaultCocRuleIndex" jsbind:"defaultCocRuleIndex"` // 默认coc index

	CustomBotExtraText       string `yaml:"customBotExtraText"`       // bot自定义文本
	CustomDrawKeysText       string `yaml:"customDrawKeysText"`       // draw keys自定义文本
	CustomDrawKeysTextEnable bool   `yaml:"customDrawKeysTextEnable"` // 应用draw keys自定义文本

	ExtDefaultSettings []*ExtDefaultSettingItem `yaml:"extDefaultSettings"` // 新群扩展按此顺序加载

	BanList *BanListInfo `yaml:"banList"` //

	//ConfigVersion         int                    `yaml:"configVersion"`
	//InPackGoCqHttpExists bool                       `yaml:"-"` // 是否存在同目录的gocqhttp
	TextMapRaw      TextTemplateWithWeightDict `yaml:"-"`
	TextMapHelpInfo TextTemplateWithHelpDict   `yaml:"-"`
	Parent          *DiceManager               `yaml:"-"`

	CocExtraRules    map[int]*CocRuleInfo `yaml:"-" json:"cocExtraRules"`
	Cron             *cron.Cron           `yaml:"-" json:"-"`
	AliveNoticeEntry cron.EntryID         `yaml:"-" json:"-"`
	//JsVM             *goja.Runtime          `yaml:"-" json:"-"`
	JsPrinter    *PrinterFunc           `yaml:"-" json:"-"`
	JsRequire    *require.RequireModule `yaml:"-" json:"-"`
	JsLoop       *eventloop.EventLoop   `yaml:"-" json:"-"`
	JsScriptList []*JsScriptInfo        `yaml:"-" json:"-"`

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
	MailSmtp     string `json:"mailSmtp" yaml:"mailSmtp"`         // 邮箱 smtp 地址
	//InPackGoCqHttpLoginSuccess bool                       `yaml:"-"` // 是否登录成功
	//InPackGoCqHttpRunning      bool                       `yaml:"-"` // 是否仍在运行
}

func (d *Dice) MarkModified() {
	d.LastUpdatedTime = time.Now().Unix()
}

func (d *Dice) CocExtraRulesAdd(ruleInfo *CocRuleInfo) bool {
	//d.JsLock.Lock()

	if _, ok := d.CocExtraRules[ruleInfo.Index]; ok {
		//d.JsLock.Unlock()
		return false
	}
	d.CocExtraRules[ruleInfo.Index] = ruleInfo
	//d.JsLock.Unlock()
	return true
}

func (d *Dice) Init() {
	d.BaseConfig.DataDir = filepath.Join("./data", d.BaseConfig.Name)
	_ = os.MkdirAll(d.BaseConfig.DataDir, 0755)
	_ = os.MkdirAll(filepath.Join(d.BaseConfig.DataDir, "configs"), 0755)
	_ = os.MkdirAll(filepath.Join(d.BaseConfig.DataDir, "extensions"), 0755)
	_ = os.MkdirAll(filepath.Join(d.BaseConfig.DataDir, "log-exports"), 0755)
	_ = os.MkdirAll(filepath.Join(d.BaseConfig.DataDir, "extra"), 0755)
	_ = os.MkdirAll(filepath.Join(d.BaseConfig.DataDir, "scripts"), 0755)

	d.Cron = cron.New()
	d.Cron.Start()

	d.CocExtraRules = map[int]*CocRuleInfo{}

	var err error
	d.DBData, d.DBLogs, err = model.SQLiteDBInit(d.BaseConfig.DataDir)
	if err != nil {
		// TODO:
		fmt.Println(err)
	}

	//d.DB = model.BoltDBInit(filepath.Join(d.BaseConfig.DataDir, "data.bdb"))
	log := logger.LoggerInit(filepath.Join(d.BaseConfig.DataDir, "record.log"), d.BaseConfig.Name, d.BaseConfig.IsLogPrint)
	d.Logger = log.Logger
	d.LogWriter = log.WX
	d.BanList = &BanListInfo{Parent: d}
	d.BanList.Init()

	d.CommandCompatibleMode = true
	d.ImSession = &IMSession{}
	d.ImSession.Parent = d
	d.ImSession.ServiceAtNew = make(map[string]*GroupInfo)
	d.CmdMap = CmdMapCls{}
	d.GameSystemMap = new(SyncMap[string, *GameSystemTemplate])

	d.registerCoreCommands()
	d.RegisterBuiltinExt()
	d.loads()
	d.BanList.Loads()
	d.BanList.AfterLoads()
	d.IsAlreadyLoadConfig = true

	// 创建js运行时
	d.JsInit()

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
				count += 1
				d.Save(true)
				if count%5 == 0 {
					// 注: 这种用法我不太清楚是否有必要
					_ = model.FlushWAL(d.DBData)
					_ = model.FlushWAL(d.DBLogs)
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
							diceId := i.UserId
							//if v.DiceIdActiveMap.Len() == 0 {
							//	v.DiceIdActiveMap.Store(diceId, true)
							//}
							now := time.Now().Unix()

							// 上次被人使用小于60s
							if now-v.RecentDiceSendTime < 60 {
								// 在群内存在，且开启时
								if _, exists := v.DiceIdExistsMap.Load(diceId); exists {
									if _, exists := v.DiceIdActiveMap.Load(diceId); exists {
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
	d.JsLoadScripts()

	if d.UpgradeWindowId != "" {
		go func() {
			defer ErrorLogAndContinue(d)

			var ep *EndPointInfo
			for _, _ep := range d.ImSession.EndPoints {
				if _ep.Id == d.UpgradeEndpointId {
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
				text := fmt.Sprintf("升级完成，当前版本: %s", VERSION)

				if ep.State == 2 {
					// 还没好，继续等待
					continue
				}

				// 可以了，发送消息
				ctx := &MsgContext{Dice: d, EndPoint: ep, Session: d.ImSession}
				isGroup := strings.Contains(d.UpgradeWindowId, "-Group:")
				if isGroup {
					ReplyGroup(ctx, &Message{GroupId: d.UpgradeWindowId}, text)
				} else {
					ReplyPerson(ctx, &Message{Sender: SenderBase{UserId: d.UpgradeWindowId}}, text)
				}

				d.Logger.Infof("升级完成，当前版本: %s", VERSION)
				d.UpgradeWindowId = ""
				d.UpgradeEndpointId = ""
				d.MarkModified()
				d.Save(false)
				break
			}
		}()
	}

	d.MarkModified()
}

func (d *Dice) rebuildParser(buffer string) *DiceRollParser {
	p := &DiceRollParser{Buffer: buffer}
	_ = p.Init()
	p.RollExpression.Init(512)
	//d.RollParser = p;
	return p
}

func (d *Dice) ExprEvalBase(buffer string, ctx *MsgContext, flags RollExtraFlags) (*VmResult, string, error) {
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
		num, detail, err := parser.Evaluate(d, ctx)
		if err != nil {
			return nil, "", err
		}

		ret := VmResult{}
		ret.Value = num.Value
		ret.TypeId = num.TypeId
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

func (d *Dice) ExprEval(buffer string, ctx *MsgContext) (*VmResult, string, error) {
	return d.ExprEvalBase(buffer, ctx, RollExtraFlags{})
}

func (d *Dice) ExprTextBase(buffer string, ctx *MsgContext, flags RollExtraFlags) (*VmResult, string, error) {
	buffer = CompatibleReplace(ctx, buffer)

	// 隐藏的内置字符串符号 \x1e
	val, detail, err := d.ExprEvalBase("\x1e"+buffer+"\x1e", ctx, flags)
	//val, detail, err := d.ExprEval("`"+buffer+"`", ctx)
	//fmt.Println("???", buffer, val, detail, err, "'"+buffer+"'")

	if err != nil {
		fmt.Println("脚本执行出错: ", buffer, "->", err)
	}

	if err == nil && (val.TypeId == VMTypeString || val.TypeId == VMTypeNone) {
		return val, detail, err
	}

	return nil, "", errors.New("错误的表达式")
}

func (d *Dice) ExprText(buffer string, ctx *MsgContext) (string, string, error) {
	val, detail, err := d.ExprTextBase(buffer, ctx, RollExtraFlags{})
	//fmt.Println("!XX", buffer, val, detail, err)

	if err == nil && (val.TypeId == VMTypeString || val.TypeId == VMTypeNone) {
		return val.Value.(string), detail, err
	}

	return "格式化错误:" + strconv.Quote(buffer), "", errors.New("错误的表达式")
}

func (d *Dice) ExtFind(s string) *ExtInfo {
	for _, i := range d.ExtList {
		if i.Name == s {
			return i
		}
	}
	return nil
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

func (d *Dice) MasterCheck(uid string) bool {
	for _, i := range d.DiceMasters {
		if i == uid {
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
			for _, ep := range d.ImSession.EndPoints {
				ctx := &MsgContext{Dice: d, EndPoint: ep, Session: d.ImSession}
				ctx.Notice(fmt.Sprintf("存活, D100=%d", DiceRoll64(100)))
			}
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
		tmpl.AliasMap = new(SyncMap[string, string])
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

func DiceRoll(dicePoints int) int {
	if dicePoints <= 0 {
		return 0
	}
	val := rand.Int()%dicePoints + 1
	return val
}

func DiceRoll64(dicePoints int64) int64 {
	if dicePoints == 0 {
		return 0
	}
	val := rand.Int63()%dicePoints + 1
	return val
}

func CrashLog() {
	if r := recover(); r != nil {
		text := fmt.Sprintf("报错: %v\n堆栈: %v", r, string(debug.Stack()))
		now := time.Now()
		_ = os.WriteFile(fmt.Sprintf("崩溃日志_%s.txt", now.Format("20060201_150405")), []byte(text), 0644)
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
