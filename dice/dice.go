package dice

import (
	"errors"
	"fmt"
	wr "github.com/mroth/weightedrand"
	"go.etcd.io/bbolt"
	"go.uber.org/zap"
	"math/rand"
	"os"
	"path/filepath"
	"sealdice-core/dice/logger"
	"sealdice-core/dice/model"
	"strconv"
	"strings"
	"time"
)

var APPNAME = "SealDice"
var VERSION = "1.0.0rc1 v20220513"
var VERSION_CODE = int64(1000001) // 991404

type CmdExecuteResult struct {
	Matched       bool // 是否是指令
	Solved        bool // 是否响应此指令
	ShowShortHelp bool
	ShowLongHelp  bool
}

type CmdItemInfo struct {
	Name     string
	Solve    func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult
	Help     string // 短帮助，格式是 .xxx a b // 说明
	LongHelp string // 长帮助，带换行的较详细说明
	//Keywords []string // 其他帮助关键字
	ChopWords []string
}

type CmdMapCls map[string]*CmdItemInfo

type ExtInfo struct {
	Name    string `yaml:"name"` // 名字
	Version string `yaml:"-"`    // 版本
	// 作者
	// 更新时间
	AutoActive      bool      `yaml:"-"` // 是否自动开启
	CmdMap          CmdMapCls `yaml:"-"` // 指令集合
	Brief           string    `yaml:"-"`
	ActiveOnPrivate bool      `yaml:"-"`

	defaultSetting *ExtDefaultSettingItem `yaml:"-"` // 默认配置

	Author       string   `yaml:"-"`
	ConflictWith []string `yaml:"-"`
	//activeInSession bool; // 在当前会话中开启

	OnNotCommandReceived func(ctx *MsgContext, msg *Message)                        `yaml:"-"` // 指令过滤后剩下的
	OnCommandOverride    func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) bool `yaml:"-"` // 覆盖指令行为

	OnCommandReceived func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs)                              `yaml:"-"`
	OnMessageReceived func(ctx *MsgContext, msg *Message)                                                `yaml:"-"`
	OnMessageSend     func(ctx *MsgContext, messageType string, userId string, text string, flag string) `yaml:"-"`
	GetDescText       func(i *ExtInfo) string                                                            `yaml:"-"`
	IsLoaded          bool                                                                               `yaml:"-"`
	OnLoad            func()                                                                             `yaml:"-"`
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
	ImSession               *IMSession             `yaml:"imSession"`
	CmdMap                  CmdMapCls              `yaml:"-"`
	ExtList                 []*ExtInfo             `yaml:"-"`
	RollParser              *DiceRollParser        `yaml:"-"`
	CommandCompatibleMode   bool                   `yaml:"commandCompatibleMode"`
	LastSavedTime           *time.Time             `yaml:"lastSavedTime"`
	TextMap                 map[string]*wr.Chooser `yaml:"-"`
	BaseConfig              DiceConfig             `yaml:"-"`
	DB                      *bbolt.DB              `yaml:"-"`                       // 数据库对象
	Logger                  *zap.SugaredLogger     `yaml:"logger"`                  // 日志
	LogWriter               *logger.WriterX        `yaml:"-"`                       // 用于api的log对象
	IsDeckLoading           bool                   `yaml:"-"`                       // 正在加载中
	DeckList                []*DeckInfo            `yaml:"deckList"`                // 牌堆信息
	CommandPrefix           []string               `yaml:"commandPrefix"`           // 指令前导
	DiceMasters             []string               `yaml:"diceMasters"`             // 骰主设置，需要格式: 平台:帐号
	NoticeIds               []string               `yaml:"noticeIds"`               // 通知ID
	OnlyLogCommandInGroup   bool                   `yaml:"onlyLogCommandInGroup"`   // 日志中仅记录命令
	OnlyLogCommandInPrivate bool                   `yaml:"onlyLogCommandInPrivate"` // 日志中仅记录命令
	VersionCode             int                    `json:"versionCode"`             // 版本ID(配置文件)
	MessageDelayRangeStart  float64                `yaml:"messageDelayRangeStart"`  // 指令延迟区间
	MessageDelayRangeEnd    float64                `yaml:"messageDelayRangeEnd"`
	WorkInQQChannel         bool                   `yaml:"workInQQChannel"`
	QQChannelAutoOn         bool                   `yaml:"QQChannelAutoOn"`     // QQ频道中自动开启(默认不开)
	QQChannelLogMessage     bool                   `yaml:"QQChannelLogMessage"` // QQ频道中记录消息(默认不开)
	UILogLimit              int64                  `yaml:"UILogLimit"`
	FriendAddComment        string                 `yaml:"friendAddComment"` // 加好友验证信息
	MasterUnlockCode        string                 `yaml:"-"`                // 解锁码，每20分钟变化一次，使用后立即变化
	MasterUnlockCodeTime    int64                  `yaml:"-"`
	CustomReplyConfigEnable bool                   `yaml:"customReplyConfigEnable"`
	CustomReplyConfig       []*ReplyConfig         `yaml:"-"`
	AutoReloginEnable       bool                   `yaml:"autoReloginEnable"` // 启用自动重新登录
	RefuseGroupInvite       bool                   `yaml:"refuseGroupInvite"` // 拒绝加入新群

	HelpMasterInfo      string `yaml:"helpMasterInfo"`      // help中骰主信息
	HelpMasterLicense   string `yaml:"helpMasterLicense"`   // help中使用协议
	DefaultCocRuleIndex int64  `yaml:"defaultCocRuleIndex"` // 默认coc index

	ExtDefaultSettings []*ExtDefaultSettingItem `yaml:"extDefaultSettings"` // 新群扩展按此顺序加载

	BanList *BanListInfo `yaml:"banList"`

	//ConfigVersion         int                    `yaml:"configVersion"`
	//InPackGoCqHttpExists bool                       `yaml:"-"` // 是否存在同目录的gocqhttp
	TextMapRaw      TextTemplateWithWeightDict `yaml:"-"`
	TextMapHelpInfo TextTemplateWithHelpDict   `yaml:"-"`
	Parent          *DiceManager               `yaml:"-"`

	//InPackGoCqHttpLoginSuccess bool                       `yaml:"-"` // 是否登录成功
	//InPackGoCqHttpRunning      bool                       `yaml:"-"` // 是否仍在运行
}

func (d *Dice) Init() {
	d.BaseConfig.DataDir = filepath.Join("./data", d.BaseConfig.Name)
	os.MkdirAll(d.BaseConfig.DataDir, 0755)
	os.MkdirAll(filepath.Join(d.BaseConfig.DataDir, "configs"), 0755)
	os.MkdirAll(filepath.Join(d.BaseConfig.DataDir, "extensions"), 0755)
	os.MkdirAll(filepath.Join(d.BaseConfig.DataDir, "logs"), 0755)
	os.MkdirAll(filepath.Join(d.BaseConfig.DataDir, "extra"), 0755)

	d.DB = model.BoltDBInit(filepath.Join(d.BaseConfig.DataDir, "data.bdb"))
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

	d.registerCoreCommands()
	d.RegisterBuiltinExt()
	d.loads()
	d.BanList.AfterLoads()

	for _, i := range d.ExtList {
		if i.OnLoad != nil {
			i.OnLoad()
		}
	}

	autoSave := func() {
		t := time.Tick(30 * time.Second)
		for {
			<-t
			d.Save(true)
		}
	}
	go autoSave()

	refreshGroupInfo := func() {
		t := time.Tick(35 * time.Second)
		defer func() {
			// 防止报错
			if r := recover(); r != nil {
				d.Logger.Error(r)
			}
		}()

		for {
			<-t

			// 自动更新群信息
			for _, i := range d.ImSession.EndPoints {
				if i.Enable {
					for k, v := range d.ImSession.ServiceAtNew {
						// TODO: 注意这里的Active可能不需要改
						if !strings.HasPrefix(k, "PG-") && v.Active {
							diceId := i.UserId
							if len(v.ActiveDiceIds) == 0 {
								v.ActiveDiceIds[diceId] = true
							}
							if v.ActiveDiceIds[diceId] {
								i.Adapter.GetGroupInfoAsync(k)
							}
						}
					}
				}
			}
		}
	}
	go refreshGroupInfo()
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
	err := parser.Parse()
	parser.RollExpression.flags = flags

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

func (d *Dice) ExprTextBase(buffer string, ctx *MsgContext) (*VmResult, string, error) {
	buffer = strings.ReplaceAll(buffer, "#{SPLIT}", "###SPLIT###")
	buffer = strings.ReplaceAll(buffer, "{FormFeed}", "###SPLIT###")
	buffer = strings.ReplaceAll(buffer, "{formfeed}", "###SPLIT###")

	// 隐藏的内置字符串符号 \x1e
	val, detail, err := d.ExprEval("\x1e"+buffer+"\x1e", ctx)
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
	val, detail, err := d.ExprTextBase(buffer, ctx)
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

func (d *Dice) MasterClear() {
	m := map[string]bool{}
	var lst []string

	for _, i := range d.DiceMasters {
		if !m[i] {
			m[i] = true
			lst = append(lst, i)
		}
	}
	d.DiceMasters = lst
}

func (d *Dice) MasterAdd(uid string) {
	d.DiceMasters = append(d.DiceMasters, uid)
	d.MasterClear()
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
