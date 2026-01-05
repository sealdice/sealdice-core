package dice

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"sealdice-core/dice/events"
	"sealdice-core/dice/service"
	"sealdice-core/logger"
	"sealdice-core/message"
	"sealdice-core/model"
	"sealdice-core/utils/dboperator/engine"

	"github.com/golang-module/carbon"
	ds "github.com/sealdice/dicescript"
	rand2 "golang.org/x/exp/rand" //nolint:staticcheck // against my better judgment, but this was mandated due to a strongly held opinion from you know who

	"github.com/dop251/goja"
	"golang.org/x/time/rate"
	"gopkg.in/yaml.v3"
)

type SenderBase struct {
	Nickname  string `jsbind:"nickname" json:"nickname"`
	UserID    string `jsbind:"userId"   json:"userId"`
	GroupRole string `json:"-"` // 群内角色 admin管理员 owner群主
}

// Message 消息的重要信息
// 时间
// 发送地点(群聊/私聊)
// 人物(是谁发的)
// 内容
type Message struct {
	Time        int64       `jsbind:"time"        json:"time"`        // 发送时间
	MessageType string      `jsbind:"messageType" json:"messageType"` // group private
	GroupID     string      `jsbind:"groupId"     json:"groupId"`     // 群号，如果是群聊消息
	GuildID     string      `jsbind:"guildId"     json:"guildId"`     // 服务器群组号，会在discord,kook,dodo等平台见到
	ChannelID   string      `jsbind:"channelId"   json:"channelId"`
	Sender      SenderBase  `jsbind:"sender"      json:"sender"`   // 发送者
	Message     string      `jsbind:"message"     json:"message"`  // 消息内容
	RawID       interface{} `jsbind:"rawId"       json:"rawId"`    // 原始信息ID，用于处理撤回等
	Platform    string      `jsbind:"platform"    json:"platform"` // 当前平台
	GroupName   string      `json:"groupName"`
	TmpUID      string      `json:"-"             yaml:"-"`
	// Note(Szzrain): 这里是消息段，为了支持多种消息类型，目前只有 Milky 支持，其他平台也应该尽快迁移支持，并使用 Session.ExecuteNew 方法
	Segment []message.IMessageElement `jsbind:"segment" json:"-" yaml:"-"`
}

// GroupPlayerInfo 这是一个YamlWrapper，没有实际作用
// 原因见 https://github.com/go-yaml/yaml/issues/712
// type GroupPlayerInfo struct {
// 	GroupPlayerInfoBase `yaml:",inline,flow"`
// }

type GroupPlayerInfo model.GroupPlayerInfoBase

type GroupInfo struct {
	Active    bool                               `jsbind:"active" json:"active" yaml:"active"` // 是否在群内开启 - 过渡为象征意义
	extInitMu sync.Mutex                         `json:"-" yaml:"-"`                           // 延迟初始化锁
	Players   *SyncMap[string, *GroupPlayerInfo] `json:"-" yaml:"-"`                           // 群员角色数据

	activatedExtList  []*ExtInfo // 当前群开启的扩展列表（私有，通过 Getter 访问，由 MarshalJSON/UnmarshalJSON 处理序列化）
	InactivatedExtSet StringSet  `json:"inactivatedExtSet" yaml:"inactivatedExtSet,flow"` // 手动关闭或尚未启用的扩展

	GroupID         string                 `jsbind:"groupId"       json:"groupId"      yaml:"groupId"`
	GuildID         string                 `jsbind:"guildId"       json:"guildId"      yaml:"guildId"`
	ChannelID       string                 `jsbind:"channelId"     json:"channelId"    yaml:"channelId"`
	GroupName       string                 `jsbind:"groupName"     json:"groupName"    yaml:"groupName"`
	DiceIDActiveMap *SyncMap[string, bool] `json:"diceIdActiveMap" yaml:"diceIds,flow"` // 对应的骰子ID(格式 平台:ID)，对应单骰多号情况，例如骰A B都加了群Z，A退群不会影响B在群内服务
	DiceIDExistsMap *SyncMap[string, bool] `json:"diceIdExistsMap" yaml:"-"`            // 对应的骰子ID(格式 平台:ID)是否存在于群内
	BotList         *SyncMap[string, bool] `json:"botList"         yaml:"botList,flow"` // 其他骰子列表
	DiceSideNum     int64                  `json:"diceSideNum"     yaml:"diceSideNum"`  // 以后可能会支持 1d4 这种默认面数，暂不开放给js
	DiceSideExpr    string                 `json:"diceSideExpr"    yaml:"diceSideExpr"` //
	System          string                 `json:"system"          yaml:"system"`       // 规则系统，概念同bcdice的gamesystem，距离如dnd5e coc7

	HelpPackages []string `json:"helpPackages"   yaml:"-"`
	CocRuleIndex int      `jsbind:"cocRuleIndex" json:"cocRuleIndex" yaml:"cocRuleIndex"`
	LogCurName   string   `jsbind:"logCurName"   json:"logCurName"   yaml:"logCurFile"`
	LogOn        bool     `jsbind:"logOn"        json:"logOn"        yaml:"logOn"`

	QuitMarkAutoClean   bool   `json:"-"                     yaml:"-"` // 自动清群 - 播报，即将自动退出群组
	QuitMarkMaster      bool   `json:"-"                     yaml:"-"` // 骰主命令退群 - 播报，即将自动退出群组
	RecentDiceSendTime  int64  `jsbind:"recentDiceSendTime"  json:"recentDiceSendTime"`
	ShowGroupWelcome    bool   `jsbind:"showGroupWelcome"    json:"showGroupWelcome"    yaml:"showGroupWelcome"` // 是否迎新
	GroupWelcomeMessage string `jsbind:"groupWelcomeMessage" json:"groupWelcomeMessage" yaml:"groupWelcomeMessage"`
	// FirstSpeechMade     bool   `yaml:"firstSpeechMade"` // 是否做过进群发言
	LastCustomReplyTime float64 `json:"-" yaml:"-"` // 上次自定义回复时间

	RateLimiter     *rate.Limiter `json:"-" yaml:"-"`
	RateLimitWarned bool          `json:"-" yaml:"-"`

	EnteredTime  int64  `jsbind:"enteredTime"  json:"enteredTime"  yaml:"enteredTime"`  // 入群时间
	InviteUserID string `jsbind:"inviteUserId" json:"inviteUserId" yaml:"inviteUserId"` // 邀请人
	// 仅用于http接口
	TmpPlayerNum int64    `json:"tmpPlayerNum" yaml:"-"`
	TmpExtList   []string `json:"tmpExtList"   yaml:"-"`

	UpdatedAtTime int64 `json:"-" yaml:"-"`

	DefaultHelpGroup string `json:"defaultHelpGroup" yaml:"defaultHelpGroup"` // 当前群默认的帮助条目

	PlayerGroups      *SyncMap[string, []string] `json:"playerGroups"      yaml:"playerGroups"` // 给team指令使用，和玩家、群等信息一样，都来自Players，不会重复存储
	ExtAppliedVersion int64                      `json:"extAppliedVersion" yaml:"extAppliedVersion"`

	/* Wrapper 架构 */
	ExtAppliedTime int64 `json:"-" yaml:"-"` // 群组应用扩展的时间戳，运行时使用，不序列化（强制每次启动重新初始化）
}

// GetActivatedExtList 获取激活的扩展列表，自动处理延迟初始化
// 通过 ExtAppliedTime == 0 判断是否需要初始化
// 同时处理新扩展的延迟激活
func (g *GroupInfo) GetActivatedExtList(d *Dice) []*ExtInfo {
	// 快速路径：已初始化
	if atomic.LoadInt64(&g.ExtAppliedTime) != 0 {
		g.extInitMu.Lock()
		list := g.activatedExtList
		g.extInitMu.Unlock()
		return list
	}
	g.extInitMu.Lock()
	defer g.extInitMu.Unlock()
	if atomic.LoadInt64(&g.ExtAppliedTime) != 0 {
		return g.activatedExtList // double-check
	}

	// 延迟初始化：用全局 ExtList 替换反序列化的占位对象
	extMap := make(map[string]*ExtInfo)
	for _, ext := range d.ExtList {
		extMap[ext.Name] = ext
	}

	oldCount := len(g.activatedExtList)
	var newList []*ExtInfo
	activated := make(map[string]bool)
	for _, item := range g.activatedExtList {
		if item != nil && extMap[item.Name] != nil {
			newList = append(newList, extMap[item.Name])
			activated[item.Name] = true
		}
	}

	// 延迟激活新扩展：检查 ExtList 中是否有新扩展需要激活
	// 新扩展 = 不在 activatedExtList 中，也不在 InactivatedExtSet 中
	g.ensureInactivatedSet()
	newExtCount := 0
	for _, ext := range d.ExtList {
		if ext == nil {
			continue
		}
		// 跳过已激活的扩展
		if activated[ext.Name] {
			continue
		}
		// 跳过被用户手动关闭的扩展
		if g.IsExtInactivated(ext.Name) {
			continue
		}
		// 新扩展：根据 AutoActive 决定是否激活
		if ext.AutoActive || (ext.DefaultSetting != nil && ext.DefaultSetting.AutoActive) {
			newList = append([]*ExtInfo{ext}, newList...) // 插入头部
			activated[ext.Name] = true
			newExtCount++
		} else {
			g.AddToInactivated(ext.Name)
		}
	}

	g.activatedExtList = newList
	// 标记已初始化，确保值不为 0（否则下次检查会再次进入初始化）
	appliedTime := d.ExtUpdateTime
	if appliedTime == 0 {
		appliedTime = 1
	}
	atomic.StoreInt64(&g.ExtAppliedTime, appliedTime)

	// 如果激活了新扩展，标记群组为 dirty
	if newExtCount > 0 {
		g.MarkDirty(d)
	}

	// 打印初始化日志
	d.Logger.Infof("群组扩展初始化: %s, 扩展数 %d -> %d (新激活 %d)", g.GroupID, oldCount, len(newList), newExtCount)
	return g.activatedExtList
}

// TriggerExtHook 遍历已激活的扩展并触发钩子
// getHook 返回要执行的函数，若返回 nil 则跳过该扩展
func (g *GroupInfo) TriggerExtHook(d *Dice, getHook func(*ExtInfo) func()) {
	for _, wrapper := range g.GetActivatedExtList(d) {
		ext := wrapper.GetRealExt()
		if ext == nil {
			continue
		}
		if hook := getHook(ext); hook != nil {
			ext.callWithJsCheck(d, hook)
		}
	}
}

// GetActivatedExtListRaw 直接访问扩展列表（用于序列化、内部修改等场景）
func (g *GroupInfo) GetActivatedExtListRaw() []*ExtInfo {
	g.extInitMu.Lock()
	defer g.extInitMu.Unlock()
	return g.activatedExtList
}

// SetActivatedExtList 设置扩展列表（用于新群组创建等场景）
func (g *GroupInfo) SetActivatedExtList(list []*ExtInfo, d *Dice) {
	g.extInitMu.Lock()
	defer g.extInitMu.Unlock()
	g.activatedExtList = list
	if d != nil {
		atomic.StoreInt64(&g.ExtAppliedTime, d.ExtUpdateTime) // 标记已初始化
	} else {
		atomic.StoreInt64(&g.ExtAppliedTime, 1) // 没有 Dice 时设置非零值标记已初始化
	}
}

// groupInfoAlias 用于避免 MarshalJSON 递归调用
type groupInfoAlias GroupInfo

// groupInfoJSON 用于序列化/反序列化 GroupInfo
// 由于 activatedExtList 是私有字段，需要通过此结构体处理
type groupInfoJSON struct {
	*groupInfoAlias
	ActivatedExtList []*ExtInfo `json:"activatedExtList"`
}

// MarshalJSON 自定义序列化，处理私有字段 activatedExtList
// 同时过滤掉已删除的 wrapper（IsDeleted=true）
func (g *GroupInfo) MarshalJSON() ([]byte, error) {
	g.extInitMu.Lock()
	// 过滤掉已删除的 wrapper
	var filteredList []*ExtInfo
	for _, ext := range g.activatedExtList {
		if ext != nil && !ext.IsDeleted {
			filteredList = append(filteredList, ext)
		}
	}
	g.extInitMu.Unlock()

	return json.Marshal(&groupInfoJSON{
		groupInfoAlias:   (*groupInfoAlias)(g),
		ActivatedExtList: filteredList,
	})
}

// UnmarshalJSON 自定义反序列化，处理私有字段 activatedExtList
func (g *GroupInfo) UnmarshalJSON(data []byte) error {
	temp := &groupInfoJSON{
		groupInfoAlias: (*groupInfoAlias)(g),
	}
	if err := json.Unmarshal(data, temp); err != nil {
		return err
	}
	g.extInitMu.Lock()
	g.activatedExtList = temp.ActivatedExtList
	g.extInitMu.Unlock()
	return nil
}

// MarkDirty 标记群组为脏数据，需要保存到数据库
// 同时将群组 ID 加入脏列表，Save 时只遍历脏列表
func (g *GroupInfo) MarkDirty(d *Dice) {
	now := time.Now().Unix()
	g.UpdatedAtTime = now
	if d != nil && d.DirtyGroups != nil {
		d.DirtyGroups.Store(g.GroupID, now)
	}
}

func (group *GroupInfo) IsActive(ctx *MsgContext) bool {
	if strings.HasPrefix(group.GroupID, "UI-Group:") {
		return true
	}
	firstCheck := group.Active && group.DiceIDActiveMap.Len() >= 1
	if firstCheck {
		v, _ := group.DiceIDActiveMap.Load(ctx.EndPoint.UserID)
		return v
	}
	return false
}

func (group *GroupInfo) PlayerGet(operator engine.DatabaseOperator, id string) *GroupPlayerInfo {
	if group.Players == nil {
		group.Players = new(SyncMap[string, *GroupPlayerInfo])
	}
	p, exists := group.Players.Load(id)
	if !exists {
		basePtr := service.GroupPlayerInfoGet(operator, group.GroupID, id)
		p = (*GroupPlayerInfo)(basePtr)
		if p != nil {
			group.Players.Store(id, p)
		}
	}
	return p
}

// GetCharTemplate 这个函数最好给ctx，在group下不合理，传入dice就很滑稽了
func (group *GroupInfo) GetCharTemplate(dice *Dice) *GameSystemTemplate {
	// 有system优先system
	if group.System != "" {
		v, _ := dice.GameSystemMap.Load(group.System)
		if v != nil {
			return v
		}
		// 返回这个单纯是为了不让st将其覆盖
		// 这种情况属于卡片的规则模板被删除了
		tmpl := &GameSystemTemplate{
			GameSystemTemplateV2: &GameSystemTemplateV2{
				Name:     group.System,
				FullName: "空白模板",
			},
		}
		tmpl.Init()
		return tmpl
	}
	// 没有system，查看扩展的启动情况
	if group.ExtGetActive("coc7") != nil {
		v, _ := dice.GameSystemMap.Load("coc7")
		return v
	}

	if group.ExtGetActive("dnd5e") != nil {
		v, _ := dice.GameSystemMap.Load("dnd5e")
		return v
	}

	// 啥都没有，返回空，还是白卡？
	// 返回个空白模板好了
	blankTmpl := &GameSystemTemplate{
		GameSystemTemplateV2: &GameSystemTemplateV2{
			Name:     "空白模板",
			FullName: "空白模板",
		},
	}
	blankTmpl.Init()
	return blankTmpl
}

type EndpointState int

type EndPointInfoBase struct {
	ID                  string        `jsbind:"id"                  json:"id"                  yaml:"id"` // uuid
	Nickname            string        `jsbind:"nickname"            json:"nickname"            yaml:"nickname"`
	State               EndpointState `jsbind:"state"               json:"state"               yaml:"state"` // 状态 0断开 1已连接 2连接中 3连接失败
	UserID              string        `jsbind:"userId"              json:"userId"              yaml:"userId"`
	GroupNum            int64         `jsbind:"groupNum"            json:"groupNum"            yaml:"groupNum"`            // 拥有群数
	CmdExecutedNum      int64         `jsbind:"cmdExecutedNum"      json:"cmdExecutedNum"      yaml:"cmdExecutedNum"`      // 指令执行次数
	CmdExecutedLastTime int64         `jsbind:"cmdExecutedLastTime" json:"cmdExecutedLastTime" yaml:"cmdExecutedLastTime"` // 指令执行次数
	OnlineTotalTime     int64         `jsbind:"onlineTotalTime"     json:"onlineTotalTime"     yaml:"onlineTotalTime"`     // 在线时长

	Platform     string `jsbind:"platform"   json:"platform"     yaml:"platform"` // 平台，如QQ等
	RelWorkDir   string `json:"relWorkDir"   yaml:"relWorkDir"`                   // 工作目录
	Enable       bool   `jsbind:"enable"     json:"enable"       yaml:"enable"`   // 是否启用
	ProtocolType string `json:"protocolType" yaml:"protocolType"`                 // 协议类型，如onebot、koishi等

	IsPublic bool       `json:"isPublic" yaml:"isPublic"`
	Session  *IMSession `json:"-"        yaml:"-"`
}

const (
	StateDisconnected     EndpointState = iota // 0: 断开
	StateConnected                             // 1: 已连接
	StateConnecting                            // 2: 连接中
	StateConnectionFailed                      // 3: 连接失败
)

type EndPointInfo struct {
	EndPointInfoBase `jsbind:"baseInfo" yaml:"baseInfo"`

	Adapter PlatformAdapter `json:"adapter" yaml:"adapter"`
}

func (ep *EndPointInfo) UnmarshalYAML(value *yaml.Node) error {
	if ep.Adapter != nil {
		return value.Decode(ep)
	}

	var val struct {
		EndPointInfoBase `yaml:"baseInfo"`
	}
	err := value.Decode(&val)
	if err != nil {
		return err
	}
	ep.EndPointInfoBase = val.EndPointInfoBase

	switch val.Platform {
	case "QQ":
		switch ep.ProtocolType {
		case "onebot":
			var val struct {
				Adapter *PlatformAdapterGocq `yaml:"adapter"`
			}

			err = value.Decode(&val)
			if err != nil {
				return err
			}
			ep.Adapter = val.Adapter
		case "walle-q":
			var val struct {
				Adapter *PlatformAdapterWalleQ `yaml:"adapter"`
			}

			err = value.Decode(&val)
			if err != nil {
				return err
			}
			ep.Adapter = val.Adapter
		case "red":
			var val struct {
				Adapter *PlatformAdapterRed `yaml:"adapter"`
			}
			err = value.Decode(&val)
			if err != nil {
				return err
			}
			ep.Adapter = val.Adapter
		case "official":
			var val struct {
				Adapter *PlatformAdapterOfficialQQ `yaml:"adapter"`
			}
			err = value.Decode(&val)
			if err != nil {
				return err
			}
			ep.Adapter = val.Adapter
		case "satori":
			var val struct {
				Adapter *PlatformAdapterSatori `yaml:"adapter"`
			}
			err = value.Decode(&val)
			if err != nil {
				return err
			}
			ep.Adapter = val.Adapter
		// case "LagrangeGo":
		//	var val struct {
		//		Adapter *PlatformAdapterLagrangeGo `yaml:"adapter"`
		//	}
		//	err = value.Decode(&val)
		//	if err != nil {
		//		return err
		//	}
		//	ep.Adapter = val.Adapter
		case "milky":
			var val struct {
				Adapter *PlatformAdapterMilky `yaml:"adapter"`
			}
			err = value.Decode(&val)
			if err != nil {
				return err
			}
			ep.Adapter = val.Adapter
		case "pureonebot":
			var val struct {
				Adapter *PlatformAdapterOnebot `yaml:"adapter"`
			}
			err = value.Decode(&val)
			if err != nil {
				return err
			}
			ep.Adapter = val.Adapter
		}
	case "DISCORD":
		var val struct {
			Adapter *PlatformAdapterDiscord `yaml:"adapter"`
		}
		err = value.Decode(&val)
		if err != nil {
			return err
		}
		ep.Adapter = val.Adapter
	case "KOOK":
		var val struct {
			Adapter *PlatformAdapterKook `yaml:"adapter"`
		}
		err = value.Decode(&val)
		if err != nil {
			return err
		}
		ep.Adapter = val.Adapter
	case "TG":
		var val struct {
			Adapter *PlatformAdapterTelegram `yaml:"adapter"`
		}
		err = value.Decode(&val)
		if err != nil {
			return err
		}
		ep.Adapter = val.Adapter
	case "MC":
		var val struct {
			Adapter *PlatformAdapterMinecraft `yaml:"adapter"`
		}
		err = value.Decode(&val)
		if err != nil {
			return err
		}
		ep.Adapter = val.Adapter
	case "DODO":
		var val struct {
			Adapter *PlatformAdapterDodo `yaml:"adapter"`
		}
		err = value.Decode(&val)
		if err != nil {
			return err
		}
		ep.Adapter = val.Adapter
	case "DINGTALK":
		var val struct {
			Adapter *PlatformAdapterDingTalk `yaml:"adapter"`
		}
		err = value.Decode(&val)
		if err != nil {
			return err
		}
		ep.Adapter = val.Adapter
	case "SLACK":
		var val struct {
			Adapter *PlatformAdapterSlack `yaml:"adapter"`
		}
		err = value.Decode(&val)
		if err != nil {
			return err
		}
		ep.Adapter = val.Adapter
	case "SEALCHAT":
		var val struct {
			Adapter *PlatformAdapterSealChat `yaml:"adapter"`
		}
		err = value.Decode(&val)
		if err != nil {
			return err
		}
		ep.Adapter = val.Adapter
	}
	return err
}

// StatsRestore 尝试从数据库中恢复EP的统计数据
func (ep *EndPointInfo) StatsRestore(d *Dice) {
	if len(ep.UserID) == 0 {
		return // 尚未连接完成的新账号没有UserId, 跳过
	}

	m := model.EndpointInfo{UserID: ep.UserID}
	err := service.Query(d.DBOperator, &m)
	if err != nil {
		d.Logger.Errorf("恢复endpoint统计数据失败 %v : %v", ep.UserID, err)
		return
	}

	if m.UpdatedAt <= ep.CmdExecutedLastTime {
		// 只在数据库中保存的数据比当前数据新时才替换, 避免上次Dump之后新的指令统计被覆盖
		return
	}

	// 虽然觉得不至于, 还是判断一下, 只进行增长方向的更新
	if ep.CmdExecutedNum < m.CmdNum {
		ep.CmdExecutedNum = m.CmdNum
	}
	if ep.CmdExecutedLastTime < m.CmdLastTime {
		ep.CmdExecutedLastTime = m.CmdLastTime
	}
	if ep.OnlineTotalTime < m.OnlineTime {
		ep.OnlineTotalTime = m.OnlineTime
	}
}

// StatsDump EP统计数据落库
func (ep *EndPointInfo) StatsDump(d *Dice) {
	if len(ep.UserID) == 0 {
		return // 尚未连接完成的新账号没有UserId, 跳过
	}

	m := model.EndpointInfo{UserID: ep.UserID, CmdNum: ep.CmdExecutedNum, CmdLastTime: ep.CmdExecutedLastTime, OnlineTime: ep.OnlineTotalTime}
	err := service.Save(d.DBOperator, &m)
	if err != nil {
		d.Logger.Errorf("保存endpoint数据到数据库失败 %v : %v", ep.UserID, err)
	}
}

type IMSession struct {
	Parent       *Dice                        `yaml:"-"`
	EndPoints    []*EndPointInfo              `yaml:"endPoints"`
	ServiceAtNew *SyncMap[string, *GroupInfo] `json:"servicesAt" yaml:"-"`
}

type MsgContext struct {
	MessageType string
	Group       *GroupInfo       `jsbind:"group"`  // 当前群信息
	Player      *GroupPlayerInfo `jsbind:"player"` // 当前群的玩家数据

	IsCompatibilityTest bool // 是否为兼容性测试环境，用于跳过不必要的数据库查询

	EndPoint        *EndPointInfo `jsbind:"endPoint"` // 对应的Endpoint
	Session         *IMSession    // 对应的IMSession
	Dice            *Dice         // 对应的 Dice
	IsCurGroupBotOn bool          `jsbind:"isCurGroupBotOn"` // 在群内是否bot on

	IsPrivate       bool        `jsbind:"isPrivate"` // 是否私聊
	CommandID       int64       // 指令ID
	CommandHideFlag string      `jsbind:"commandHideFlag"` // 暗骰来源群号
	CommandInfo     interface{} // 命令信息
	PrivilegeLevel  int         `jsbind:"privilegeLevel"` // 权限等级 -30ban 40邀请者 50管理 60群主 70信任 100master
	GroupRoleLevel  int         // 群内权限 40邀请者 50管理 60群主 70信任 100master，相当于不考虑ban的权限等级
	DelegateText    string      `jsbind:"delegateText"`  // 代骰附加文本
	AliasPrefixText string      `json:"aliasPrefixText"` // 快捷指令回复前缀文本

	deckDepth         int                                         // 抽牌递归深度
	DeckPools         map[*DeckInfo]map[string]*ShuffleRandomPool // 不放回抽取的缓存
	diceExprOverwrite string                                      // 默认骰表达式覆盖
	SystemTemplate    *GameSystemTemplate
	Censored          bool // 已检查过敏感词
	SpamCheckedGroup  bool
	SpamCheckedPerson bool

	splitKey      string
	vm            *ds.Context
	AttrsCurCache *AttributesItem
	_v1Rand       *rand2.PCGSource
}

// fillPrivilege 填写MsgContext中的权限字段, 并返回填写的权限等级
//   - msg 使用其中的msg.Sender.GroupRole
//
// MsgContext.Dice需要指向一个有效的Dice对象
func (ctx *MsgContext) fillPrivilege(msg *Message) int {
	switch {
	case msg.Sender.GroupRole == "owner":
		ctx.PrivilegeLevel = 60 // 群主
	case ctx.IsPrivate || msg.Sender.GroupRole == "admin":
		ctx.PrivilegeLevel = 50 // 群管理
	case ctx.Group != nil && msg.Sender.UserID == ctx.Group.InviteUserID:
		ctx.PrivilegeLevel = 40 // 邀请者
	default: /* no-op */
	}

	ctx.GroupRoleLevel = ctx.PrivilegeLevel

	if ctx.Dice == nil || ctx.Player == nil {
		return ctx.PrivilegeLevel
	}

	// 加入黑名单相关权限
	if val, exists := ctx.Dice.Config.BanList.GetByID(ctx.Player.UserID); exists {
		switch val.Rank {
		case BanRankBanned:
			ctx.PrivilegeLevel = -30
		case BanRankTrusted:
			ctx.PrivilegeLevel = 70
		default: /* no-op */
		}
	}

	grpID := ""
	if ctx.Group != nil {
		grpID = ctx.Group.GroupID
	}
	// master 权限大于黑名单权限
	if ctx.Dice.MasterCheck(grpID, ctx.Player.UserID) {
		ctx.PrivilegeLevel = 100
	}

	return ctx.PrivilegeLevel
}

func (s *IMSession) Execute(ep *EndPointInfo, msg *Message, runInSync bool) {
	d := s.Parent

	mctx := &MsgContext{}
	mctx.Dice = d
	mctx.MessageType = msg.MessageType
	mctx.IsPrivate = mctx.MessageType == "private"
	mctx.Session = s
	mctx.EndPoint = ep
	log := d.Logger

	// 处理命令
	if msg.MessageType == "group" || msg.MessageType == "private" { //nolint:nestif
		// GroupEnableCheck TODO: 后续看看是否需要
		groupInfo, ok := s.ServiceAtNew.Load(msg.GroupID)
		if !ok && msg.GroupID != "" {
			// 注意: 此处必须开启，不然下面mctx.player取不到
			autoOn := true
			if msg.Platform == "QQ-CH" {
				autoOn = d.Config.QQChannelAutoOn
			}
			groupInfo = SetBotOnAtGroup(mctx, msg.GroupID)
			groupInfo.Active = autoOn
			groupInfo.DiceIDExistsMap.Store(ep.UserID, true)
			if msg.GroupName != "" {
				groupInfo.GroupName = msg.GroupName
			}
			groupInfo.MarkDirty(d) // SetBotOnAtGroup 已调用过一次，这里确保后续修改也被标记

			dm := d.Parent
			groupName := dm.TryGetGroupName(groupInfo.GroupID)

			txt := fmt.Sprintf("自动激活: 发现无记录群组%s(%s)，因为已是群成员，所以自动激活，开启状态: %t", groupName, groupInfo.GroupID, autoOn)
			if dm.ShouldRefreshGroupInfo(msg.GroupID) {
				ep.Adapter.GetGroupInfoAsync(msg.GroupID)
			}
			log.Info(txt)
			mctx.Notice(txt)

			if msg.Platform == "QQ" || msg.Platform == "TG" {
				// ServiceAtNew changed
				// Pinenutn:这个i不知道是啥，放你一马（
				activatedList, _ := mctx.Session.ServiceAtNew.Load(msg.GroupID)
				if ok {
					for _, wrapper := range activatedList.GetActivatedExtList(mctx.Dice) {
						ext := wrapper.GetRealExt()
						if ext == nil {
							continue
						}
						if ext.OnGroupJoined != nil {
							ext.callWithJsCheck(mctx.Dice, func() {
								ext.OnGroupJoined(mctx, msg)
							})
						}
					}
				}
			}
		}

		// 当文本可能是在发送命令时，必须加载信息
		maybeCommand := CommandCheckPrefix(msg.Message, d.CommandPrefix, msg.Platform)

		amIBeMentioned := false
		if true {
			// 被@时，必须加载信息
			// 这段代码重复了，以后重构
			_, ats := AtParse(msg.Message, msg.Platform)
			tmpUID := ep.UserID
			if msg.TmpUID != "" {
				tmpUID = msg.TmpUID
			}
			for _, i := range ats {
				// 特殊处理 OpenQQ 和 OpenQQCH
				if i.UserID == tmpUID {
					amIBeMentioned = true
					break
				} else if strings.HasPrefix(i.UserID, "OpenQQ:") ||
					strings.HasPrefix(i.UserID, "OpenQQCH:") {
					uid := strings.TrimPrefix(tmpUID, "OpenQQ:")
					if i.UserID == "OpenQQ:"+uid || i.UserID == "OpenQQCH:"+uid {
						amIBeMentioned = true
						break
					}
				}
			}
		}

		mctx.Group, mctx.Player = GetPlayerInfoBySender(mctx, msg)
		mctx.IsCurGroupBotOn = msg.MessageType == "group" && mctx.Group.IsActive(mctx)

		if mctx.Group != nil && mctx.Group.System != "" {
			mctx.SystemTemplate = mctx.Group.GetCharTemplate(d)
			// tmpl, _ := d.GameSystemMap.Load(group.System)
			// mctx.SystemTemplate = tmpl
		}

		if groupInfo != nil && !strings.HasPrefix(groupInfo.GroupID, "UI-Group:") {
			// 自动激活存在状态
			if _, exists := groupInfo.DiceIDExistsMap.Load(ep.UserID); !exists {
				groupInfo.DiceIDExistsMap.Store(ep.UserID, true)
				groupInfo.MarkDirty(d)
			}
		}

		// 权限号设置
		_ = mctx.fillPrivilege(msg)

		if mctx.Group != nil && mctx.Group.IsActive(mctx) {
			if mctx.PrivilegeLevel != -30 {
				for _, wrapper := range mctx.Group.GetActivatedExtList(mctx.Dice) {
					ext := wrapper.GetRealExt()
					if ext == nil {
						continue
					}
					if ext.OnMessageReceived != nil {
						ext.callWithJsCheck(mctx.Dice, func() {
							ext.OnMessageReceived(mctx, msg)
						})
					}
				}
			}
		}

		var cmdLst []string
		if maybeCommand {
			// 兼容模式检查已经移除
			for k := range d.CmdMap {
				cmdLst = append(cmdLst, k)
			}
			// 这里不用group是为了私聊
			g := mctx.Group
			if g != nil {
				for _, wrapper := range g.GetActivatedExtList(d) {
					for k := range wrapper.GetCmdMap() {
						cmdLst = append(cmdLst, k)
					}
				}
			}
			sort.Sort(ByLength(cmdLst))
		}

		if notReply := checkBan(mctx, msg); notReply {
			return
		}

		platformPrefix := msg.Platform
		cmdArgs := CommandParse(msg.Message, cmdLst, d.CommandPrefix, platformPrefix, false)
		if cmdArgs != nil {
			mctx.CommandID = getNextCommandID()

			var tmpUID string
			if platformPrefix == "OpenQQCH" {
				// 特殊处理 OpenQQ频道
				uid := strings.TrimPrefix(ep.UserID, "OpenQQ:")
				tmpUID = "OpenQQCH:" + uid
			} else {
				tmpUID = ep.UserID
			}
			if msg.TmpUID != "" {
				tmpUID = msg.TmpUID
			}

			// 设置at信息
			cmdArgs.SetupAtInfo(tmpUID)
		}

		// 收到群 test(1111) 内 XX(222) 的消息: 好看 (1232611291)
		if msg.MessageType == "group" {
			if mctx.CommandID != 0 {
				// 关闭状态下，如果被@，且是第一个被@的，那么视为开启
				if !mctx.IsCurGroupBotOn && cmdArgs.AmIBeMentionedFirst {
					mctx.IsCurGroupBotOn = true
				}

				log.Infof("收到群(%s)内<%s>(%s)的指令: %s", msg.GroupID, msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
			} else {
				doLog := true
				if d.Config.OnlyLogCommandInGroup {
					// 检查上级选项
					doLog = false
				}
				if doLog {
					// 检查QQ频道的独立选项
					if msg.Platform == "QQ-CH" && (!d.Config.QQChannelLogMessage) {
						doLog = false
					}
				}
				if doLog {
					log.Infof("收到群(%s)内<%s>(%s)的消息: %s", msg.GroupID, msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
					// fmt.Printf("消息长度 %v 内容 %v \n", len(msg.Message), []byte(msg.Message))
				}
			}
		}

		// 敏感词拦截：全部输入
		if mctx.IsCurGroupBotOn && d.Config.EnableCensor && d.Config.CensorMode == AllInput {
			hit, words, needToTerminate, _ := d.CensorMsg(mctx, msg, msg.Message, "")
			if needToTerminate {
				return
			}
			if hit {
				text := DiceFormatTmpl(mctx, "核心:拦截_完全拦截_收到的所有消息")
				if text != "" {
					ReplyToSender(mctx, msg, text)
				}
				if msg.MessageType == "group" {
					log.Infof(
						"拒绝处理命中敏感词「%s」的内容「%s」- 来自群(%s)内<%s>(%s)",
						strings.Join(words, "|"),
						msg.Message, msg.GroupID, msg.Sender.Nickname, msg.Sender.UserID,
					)
				} else {
					log.Infof(
						"拒绝处理命中敏感词「%s」的内容「%s」- 来自<%s>(%s)",
						strings.Join(words, "|"),
						msg.Message,
						msg.Sender.Nickname,
						msg.Sender.UserID,
					)
				}
				return
			}
		}

		if msg.MessageType == "private" {
			if mctx.CommandID != 0 {
				log.Infof("收到<%s>(%s)的私聊指令: %s", msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
			} else if !d.Config.OnlyLogCommandInPrivate {
				log.Infof("收到<%s>(%s)的私聊消息: %s", msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
			}
		}
		// Note(Szzrain): 赋值临时变量，不然有些地方没法用
		SetTempVars(mctx, msg.Sender.Nickname)
		if cmdArgs != nil {
			// 收到信息回调
			f := func() {
				defer func() {
					if r := recover(); r != nil {
						//  + fmt.Sprintf("%s", r)
						log.Errorf("异常: %v 堆栈: %v", r, string(debug.Stack()))
						ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "核心:骰子执行异常"))
					}
				}()

				// 敏感词拦截：命令输入
				if (msg.MessageType == "private" || mctx.IsCurGroupBotOn) && d.Config.EnableCensor && d.Config.CensorMode == OnlyInputCommand {
					hit, words, needToTerminate, _ := d.CensorMsg(mctx, msg, msg.Message, "")
					if needToTerminate {
						return
					}
					if hit {
						text := DiceFormatTmpl(mctx, "核心:拦截_完全拦截_收到的指令")
						if text != "" {
							ReplyToSender(mctx, msg, text)
						}
						if msg.MessageType == "group" {
							log.Infof(
								"拒绝处理命中敏感词「%s」的指令「%s」- 来自群(%s)内<%s>(%s)",
								strings.Join(words, "|"),
								msg.Message,
								msg.GroupID,
								msg.Sender.Nickname,
								msg.Sender.UserID,
							)
						} else {
							log.Infof(
								"拒绝处理命中敏感词「%s」的指令「%s」- 来自<%s>(%s)",
								strings.Join(words, "|"),
								msg.Message,
								msg.Sender.Nickname,
								msg.Sender.UserID,
							)
						}
						return
					}
				}

				if cmdArgs.Command != "botlist" && !cmdArgs.AmIBeMentioned {
					myuid := ep.UserID
					// 屏蔽机器人发送的消息
					if mctx.MessageType == "group" {
						// fmt.Println("YYYYYYYYY", myuid, mctx.Group != nil)
						if mctx.Group.BotList.Exists(msg.Sender.UserID) {
							log.Infof("忽略指令(机器人): 来自群(%s)内<%s>(%s): %s", msg.GroupID, msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
							return
						}
						// 当其他机器人被@，不回应
						for _, i := range cmdArgs.At {
							uid := i.UserID
							if uid == myuid {
								// 忽略自己
								continue
							}
							if mctx.Group.BotList.Exists(uid) {
								return
							}
						}
					}
				}

				ep.TriggerCommand(mctx, msg, cmdArgs)
			}
			if runInSync {
				f()
			} else {
				go f()
			}
		} else {
			if mctx.PrivilegeLevel == -30 {
				// 黑名单用户
				return
			}

			// 试图匹配自定义回复
			isSenderBot := false
			if mctx.MessageType == "group" {
				if mctx.Group != nil && mctx.Group.BotList.Exists(msg.Sender.UserID) {
					isSenderBot = true
				}
			}

			if !isSenderBot {
				if mctx.Group != nil && (mctx.Group.IsActive(mctx) || amIBeMentioned) {
					for _, wrapper := range mctx.Group.GetActivatedExtList(mctx.Dice) {
						ext := wrapper.GetRealExt()
						if ext == nil {
							continue
						}
						i := ext // 保留引用
						if i.OnNotCommandReceived != nil {
							notCommandReceiveCall := func() {
								if i.IsJsExt {
									// 先判断运行环境
									loop, err := d.ExtLoopManager.GetLoop(i.JSLoopVersion)
									if err != nil {
										// 打个DEBUG日志？
										mctx.Dice.Logger.Errorf("扩展<%s>运行环境已经过期: %v", i.Name, err)
										return
									}
									waitRun := make(chan int, 1)
									loop.RunOnLoop(func(runtime *goja.Runtime) {
										defer func() {
											if r := recover(); r != nil {
												mctx.Dice.Logger.Errorf("扩展<%s>处理非指令消息异常: %v 堆栈: %v", i.Name, r, string(debug.Stack()))
											}
											waitRun <- 1
										}()

										i.OnNotCommandReceived(mctx, msg)
									})
									<-waitRun
								} else {
									i.OnNotCommandReceived(mctx, msg)
								}
							}

							if runInSync {
								notCommandReceiveCall()
							} else {
								go notCommandReceiveCall()
							}
						}
					}
				}
			}
		}
	}
}

// ExecuteNew Note(Szzrain): 既不破坏兼容性还要支持新 feature 我真是草了，这里是 copy paste 的代码稍微改了一下，我知道这是在屎山上建房子，但是没办法
// 只有在 Adapter 内部实现了新的消息段解析才能使用这个方法，即 Message.Segment 有值
// 为了避免破坏兼容性，Message.Message 中的内容不会被解析但仍然会赋值
// 这个 ExcuteNew 方法优化了对消息段的解析，其他平台应当尽快实现消息段解析并使用这个方法
func (s *IMSession) ExecuteNew(ep *EndPointInfo, msg *Message) {
	d := s.Parent

	mctx := &MsgContext{}
	mctx.Dice = d
	mctx.MessageType = msg.MessageType
	mctx.IsPrivate = mctx.MessageType == "private"
	mctx.Session = s
	mctx.EndPoint = ep
	log := d.Logger

	// 处理消息段，如果 2.0 要完全抛弃依赖 Message.Message 的字符串解析，把这里删掉
	if msg.Message == "" {
		for _, elem := range msg.Segment {
			// 类型断言
			if e, ok := elem.(*message.TextElement); ok {
				msg.Message += e.Content
			}
		}
	}

	if msg.MessageType != "group" && msg.MessageType != "private" {
		return
	}

	// 处理命令
	groupInfo, ok := s.ServiceAtNew.Load(msg.GroupID)
	if !ok && msg.GroupID != "" {
		// 注意: 此处必须开启，不然下面mctx.player取不到
		autoOn := true
		if msg.Platform == "QQ-CH" {
			autoOn = d.Config.QQChannelAutoOn
		}
		groupInfo = SetBotOnAtGroup(mctx, msg.GroupID)
		groupInfo.Active = autoOn
		groupInfo.DiceIDExistsMap.Store(ep.UserID, true)
		if msg.GroupName != "" {
			groupInfo.GroupName = msg.GroupName
		}
		groupInfo.MarkDirty(d) // SetBotOnAtGroup 已调用过一次，这里确保后续修改也被标记

		// dm := d.Parent
		// 愚蠢调用，改了
		// groupName := dm.TryGetGroupName(group.GroupID)
		groupName := msg.GroupName

		txt := fmt.Sprintf("自动激活: 发现无记录群组%s(%s)，因为已是群成员，所以自动激活，开启状态: %t", groupName, groupInfo.GroupID, autoOn)
		// 意义不明，删掉
		// 疑似是为了获取群信息然后塞到奇怪的地方
		// ep.Adapter.GetGroupInfoAsync(msg.GroupID)
		log.Info(txt)
		mctx.Notice(txt)

		if msg.Platform == "QQ" || msg.Platform == "TG" {
			groupInfo, ok = mctx.Session.ServiceAtNew.Load(msg.GroupID)
			if ok {
				for _, wrapper := range groupInfo.GetActivatedExtList(mctx.Dice) {
					ext := wrapper.GetRealExt()
					if ext == nil {
						continue
					}
					if ext.OnGroupJoined != nil {
						ext.callWithJsCheck(mctx.Dice, func() {
							ext.OnGroupJoined(mctx, msg)
						})
					}
				}
			}
		}
	}
	// 重新赋值
	if groupInfo != nil {
		groupInfo.GroupName = msg.GroupName
	}

	// Note(Szzrain): 判断是否被@
	amIBeMentioned := false
	for _, elem := range msg.Segment {
		// 类型断言
		if e, ok := elem.(*message.AtElement); ok {
			if msg.Platform+":"+e.Target == ep.UserID {
				amIBeMentioned = true
				break
			}
		}
	}

	mctx.Group, mctx.Player = GetPlayerInfoBySender(mctx, msg)
	mctx.IsCurGroupBotOn = msg.MessageType == "group" && mctx.Group.IsActive(mctx)

	if mctx.Group != nil && mctx.Group.System != "" {
		mctx.SystemTemplate = mctx.Group.GetCharTemplate(d)
		// tmpl, _ := d.GameSystemMap.Load(group.System)
		// mctx.SystemTemplate = tmpl
	}

	if groupInfo != nil {
		// 自动激活存在状态
		if _, exists := groupInfo.DiceIDExistsMap.Load(ep.UserID); !exists {
			groupInfo.DiceIDExistsMap.Store(ep.UserID, true)
			groupInfo.MarkDirty(mctx.Dice)
		}
	}

	// 权限设置
	_ = mctx.fillPrivilege(msg)

	if mctx.Group != nil && mctx.Group.IsActive(mctx) {
		if mctx.PrivilegeLevel != -30 {
			for _, wrapper := range mctx.Group.GetActivatedExtList(mctx.Dice) {
				ext := wrapper.GetRealExt()
				if ext == nil {
					continue
				}
				if ext.OnMessageReceived != nil {
					ext.callWithJsCheck(mctx.Dice, func() {
						ext.OnMessageReceived(mctx, msg)
					})
				}
			}
		}
	}

	// Note(Szzrain): 兼容模式相关的代码被挪到了 cmdArgs.commandParseNew 里面

	if notReply := checkBan(mctx, msg); notReply {
		return
	}

	// Note(Szzrain): platformPrefix 弃用
	// platformPrefix := msg.Platform
	cmdArgs := CommandParseNew(mctx, msg)
	if cmdArgs != nil {
		mctx.CommandID = getNextCommandID()
		// var tmpUID string
		// if platformPrefix == "OpenQQCH" {
		//	// 特殊处理 OpenQQ频道
		//	uid := strings.TrimPrefix(ep.UserID, "OpenQQ:")
		//	tmpUID = "OpenQQCH:" + uid
		// } else {
		//	tmpUID = ep.UserID
		// }
		// if msg.TmpUID != "" {
		//	tmpUID = msg.TmpUID
		// }

		// 设置at信息，这里不再需要，因为已经在 CommandParseNew 里面设置了
		// cmdArgs.SetupAtInfo(tmpUID)
	}

	// 收到群 test(1111) 内 XX(222) 的消息: 好看 (1232611291)
	if msg.MessageType == "group" {
		// TODO(Szzrain):  需要优化的写法，不应根据 CommandID 来判断是否是指令，而应该根据 cmdArgs 是否 match 到指令来判断
		if mctx.CommandID != 0 {
			// 关闭状态下，如果被@，且是第一个被@的，那么视为开启
			if !mctx.IsCurGroupBotOn && cmdArgs.AmIBeMentionedFirst {
				mctx.IsCurGroupBotOn = true
			}

			log.Infof("收到群(%s)内<%s>(%s)的指令: %s", msg.GroupID, msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
		} else {
			doLog := true
			if d.Config.OnlyLogCommandInGroup {
				// 检查上级选项
				doLog = false
			}
			if doLog {
				// 检查QQ频道的独立选项
				if msg.Platform == "QQ-CH" && (!d.Config.QQChannelLogMessage) {
					doLog = false
				}
			}
			if doLog {
				log.Infof("收到群(%s)内<%s>(%s)的消息: %s", msg.GroupID, msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
				// fmt.Printf("消息长度 %v 内容 %v \n", len(msg.Message), []byte(msg.Message))
			}
		}
	}

	// Note(Szzrain): 这里的代码本来在敏感词检测下面，会产生预期之外的行为，所以挪到这里
	if msg.MessageType == "private" {
		// TODO(Szzrain): 需要优化的写法，不应根据 CommandID 来判断是否是指令，而应该根据 cmdArgs 是否 match 到指令来判断，同上
		if mctx.CommandID != 0 {
			log.Infof("收到<%s>(%s)的私聊指令: %s", msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
		} else if !d.Config.OnlyLogCommandInPrivate {
			log.Infof("收到<%s>(%s)的私聊消息: %s", msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
		}
	}

	// 敏感词拦截：全部输入
	if mctx.IsCurGroupBotOn && d.Config.EnableCensor && d.Config.CensorMode == AllInput {
		hit, words, needToTerminate, _ := d.CensorMsg(mctx, msg, msg.Message, "")
		if needToTerminate {
			return
		}
		if hit {
			text := DiceFormatTmpl(mctx, "核心:拦截_完全拦截_收到的所有消息")
			if text != "" {
				ReplyToSender(mctx, msg, text)
			}
			if msg.MessageType == "group" {
				log.Infof(
					"拒绝处理命中敏感词「%s」的内容「%s」- 来自群(%s)内<%s>(%s)",
					strings.Join(words, "|"),
					msg.Message, msg.GroupID, msg.Sender.Nickname, msg.Sender.UserID,
				)
			} else {
				log.Infof(
					"拒绝处理命中敏感词「%s」的内容「%s」- 来自<%s>(%s)",
					strings.Join(words, "|"),
					msg.Message,
					msg.Sender.Nickname,
					msg.Sender.UserID,
				)
			}
			return
		}
	}
	// Note(Szzrain): 赋值临时变量，不然有些地方没法用
	SetTempVars(mctx, msg.Sender.Nickname)
	if cmdArgs != nil {
		go s.PreTriggerCommand(mctx, msg, cmdArgs)
	} else {
		// if cmdArgs == nil will execute this block
		if mctx.PrivilegeLevel == -30 {
			// 黑名单用户
			return
		}

		// 试图匹配自定义回复
		isSenderBot := false
		if mctx.MessageType == "group" {
			if mctx.Group != nil && mctx.Group.BotList.Exists(msg.Sender.UserID) {
				isSenderBot = true
			}
		}

		if !isSenderBot {
			if mctx.Group != nil && (mctx.Group.IsActive(mctx) || amIBeMentioned) {
				for _, wrapper := range mctx.Group.GetActivatedExtList(mctx.Dice) {
					ext := wrapper.GetRealExt()
					if ext == nil {
						continue
					}
					i := ext // 保留引用
					if i.OnNotCommandReceived != nil {
						notCommandReceiveCall := func() {
							if i.IsJsExt {
								loop, err := d.ExtLoopManager.GetLoop(i.JSLoopVersion)
								if err != nil {
									// 打个DEBUG日志？
									i.dice.Logger.Errorf("扩展<%s>运行环境已经过期: %v", i.Name, err)
									return
								}
								waitRun := make(chan int, 1)
								loop.RunOnLoop(func(runtime *goja.Runtime) {
									defer func() {
										if r := recover(); r != nil {
											mctx.Dice.Logger.Errorf("扩展<%s>处理非指令消息异常: %v 堆栈: %v", i.Name, r, string(debug.Stack()))
										}
										waitRun <- 1
									}()
									i.OnNotCommandReceived(mctx, msg)
								})
								<-waitRun
							} else {
								i.OnNotCommandReceived(mctx, msg)
							}
						}

						go notCommandReceiveCall()
					}
				}
			}
		}
	}
}

func (s *IMSession) PreTriggerCommand(mctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
	d := s.Parent
	ep := mctx.EndPoint
	log := d.Logger
	defer func() {
		if r := recover(); r != nil {
			//  + fmt.Sprintf("%s", r)
			log.Errorf("异常: %v 堆栈: %v", r, string(debug.Stack()))
			ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "核心:骰子执行异常"))
		}
	}()

	// 敏感词拦截：命令输入
	if (msg.MessageType == "private" || mctx.IsCurGroupBotOn) && d.Config.EnableCensor && d.Config.CensorMode == OnlyInputCommand {
		hit, words, needToTerminate, _ := d.CensorMsg(mctx, msg, msg.Message, "")
		if needToTerminate {
			return
		}
		if hit {
			text := DiceFormatTmpl(mctx, "核心:拦截_完全拦截_收到的指令")
			if text != "" {
				ReplyToSender(mctx, msg, text)
			}
			if msg.MessageType == "group" {
				log.Infof(
					"拒绝处理命中敏感词「%s」的指令「%s」- 来自群(%s)内<%s>(%s)",
					strings.Join(words, "|"),
					msg.Message,
					msg.GroupID,
					msg.Sender.Nickname,
					msg.Sender.UserID,
				)
			} else {
				log.Infof(
					"拒绝处理命中敏感词「%s」的指令「%s」- 来自<%s>(%s)",
					strings.Join(words, "|"),
					msg.Message,
					msg.Sender.Nickname,
					msg.Sender.UserID,
				)
			}
			return
		}
	}

	if cmdArgs.Command != "botlist" && !cmdArgs.AmIBeMentioned {
		myuid := ep.UserID
		// 屏蔽机器人发送的消息
		if mctx.MessageType == "group" {
			// fmt.Println("YYYYYYYYY", myuid, mctx.Group != nil)
			if mctx.Group.BotList.Exists(msg.Sender.UserID) {
				log.Infof("忽略指令(机器人): 来自群(%s)内<%s>(%s): %s", msg.GroupID, msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
				return
			}
			// 当其他机器人被@，不回应
			for _, i := range cmdArgs.At {
				uid := i.UserID
				if uid == myuid {
					// 忽略自己
					continue
				}
				if mctx.Group.BotList.Exists(uid) {
					return
				}
			}
		}
	}
	ep.TriggerCommand(mctx, msg, cmdArgs)
}

func (ep *EndPointInfo) TriggerCommand(mctx *MsgContext, msg *Message, cmdArgs *CmdArgs) bool {
	s := mctx.Session
	d := mctx.Dice
	log := d.Logger

	var ret bool
	// 试图匹配自定义指令
	if mctx.Group != nil && mctx.Group.IsActive(mctx) {
		for _, wrapper := range mctx.Group.GetActivatedExtList(mctx.Dice) {
			ext := wrapper.GetRealExt()
			if ext == nil {
				continue
			}
			if ext.OnCommandOverride != nil {
				ret = ext.OnCommandOverride(mctx, msg, cmdArgs)
				if ret {
					break
				}
			}
		}
	}

	if !ret {
		// 若自定义指令未匹配，匹配标准指令
		ret = s.commandSolve(mctx, msg, cmdArgs)
	}

	if ret {
		// 刷屏检测已经迁移到 im_helpers.go，此处不再处理
		ep.CmdExecutedNum++
		ep.CmdExecutedLastTime = time.Now().Unix()
		mctx.Player.LastCommandTime = ep.CmdExecutedLastTime
		mctx.Player.UpdatedAtTime = time.Now().Unix()
		if mctx.Group != nil {
			mctx.Group.MarkDirty(mctx.Dice)
		}
	} else {
		if msg.MessageType == "group" {
			log.Infof("忽略指令(骰子关闭/扩展关闭/未知指令): 来自群(%s)内<%s>(%s): %s", msg.GroupID, msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
		}

		if msg.MessageType == "private" {
			log.Infof("忽略指令(骰子关闭/扩展关闭/未知指令): 来自<%s>(%s)的私聊: %s", msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
		}
	}
	return ret
}

// OnGroupJoined 群组进群事件处理，其他 Adapter 应当尽快迁移至此方法实现
func (s *IMSession) OnGroupJoined(ctx *MsgContext, msg *Message) {
	d := ctx.Dice
	log := d.Logger
	ep := ctx.EndPoint
	dm := d.Parent
	// 判断进群的人是自己，自动启动
	group := SetBotOnAtGroup(ctx, msg.GroupID)
	// Ensure context has group set for formatting and attrs access
	ctx.Group = group
	// 获取邀请人ID，Adapter 应当按照统一格式将邀请人 ID 放入 Sender 字段
	group.InviteUserID = msg.Sender.UserID
	group.DiceIDExistsMap.Store(ep.UserID, true)
	group.EnteredTime = time.Now().Unix() // 设置入群时间
	group.MarkDirty(ctx.Dice)
	if dm.ShouldRefreshGroupInfo(msg.GroupID) {
		ep.Adapter.GetGroupInfoAsync(msg.GroupID)
	}
	time.Sleep(2 * time.Second)
	groupName := dm.TryGetGroupName(msg.GroupID)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("入群致辞异常: %v 堆栈: %v", r, string(debug.Stack()))
			}
		}()

		// 稍作等待后发送入群致词
		time.Sleep(2 * time.Second)

		ctx.Player = &GroupPlayerInfo{}
		log.Infof("发送入群致辞，群: <%s>(%d)", groupName, msg.GroupID)
		text := DiceFormatTmpl(ctx, "核心:骰子进群")
		for _, i := range ctx.SplitText(text) {
			doSleepQQ(ctx)
			ReplyGroup(ctx, msg, strings.TrimSpace(i))
		}
	}()
	txt := fmt.Sprintf("加入群组: <%s>(%s)", groupName, msg.GroupID)
	log.Info(txt)
	ctx.Notice(txt)
	for _, wrapper := range group.GetActivatedExtList(ctx.Dice) {
		ext := wrapper.GetRealExt()
		if ext == nil {
			continue
		}
		if ext.OnGroupJoined != nil {
			ext.callWithJsCheck(d, func() {
				ext.OnGroupJoined(ctx, msg)
			})
		}
	}
}

var lastWelcome *LastWelcomeInfo

// OnGroupMemberJoined 群成员进群事件处理，除了 bot 自己以外的群成员入群时调用。其他 Adapter 应当尽快迁移至此方法实现
func (s *IMSession) OnGroupMemberJoined(ctx *MsgContext, msg *Message) {
	log := s.Parent.Logger

	groupInfo, ok := s.ServiceAtNew.Load(msg.GroupID)
	// 进群的是别人，是否迎新？
	// 这里很诡异，当手机QQ客户端审批进群时，入群后会有一句默认发言
	// 此时会收到两次完全一样的某用户入群信息，导致发两次欢迎词
	if ok && groupInfo.ShowGroupWelcome {
		isDouble := false
		if lastWelcome != nil {
			isDouble = msg.GroupID == lastWelcome.GroupID &&
				msg.Sender.UserID == lastWelcome.UserID &&
				msg.Time == lastWelcome.Time
		}
		lastWelcome = &LastWelcomeInfo{
			GroupID: msg.GroupID,
			UserID:  msg.Sender.UserID,
			Time:    msg.Time,
		}

		if !isDouble {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Errorf("迎新致辞异常: %v 堆栈: %v", r, string(debug.Stack()))
					}
				}()

				// Ensure context has group set for formatting and attrs access
				ctx.Group = groupInfo
				ctx.Player = &GroupPlayerInfo{}
				// VarSetValueStr(ctx, "$t新人昵称", "<"+msgQQ.Sender.Nickname+">")
				uidRaw := UserIDExtract(msg.Sender.UserID)
				VarSetValueStr(ctx, "$t帐号ID_RAW", uidRaw)
				VarSetValueStr(ctx, "$t账号ID_RAW", uidRaw)
				stdID := msg.Sender.UserID
				VarSetValueStr(ctx, "$t帐号ID", stdID)
				VarSetValueStr(ctx, "$t账号ID", stdID)
				text := DiceFormat(ctx, groupInfo.GroupWelcomeMessage)
				for _, i := range ctx.SplitText(text) {
					doSleepQQ(ctx)
					ReplyGroup(ctx, msg, strings.TrimSpace(i))
				}
			}()
		}
	}
}

var platformRE = regexp.MustCompile(`^(.*)-Group:`)

// LongTimeQuitInactiveGroupReborn
// 完全抛弃当初不懂Go的时候的方案，改成如下方案：
// 每次尝试找到n个符合要求的群，然后启一个线程，将群统一干掉
// 这样子牺牲了可显示的总群数，但大大增强了稳定性，而且总群数的参考并无意义，因为已经在的群很可能突然活了而不符合判定
// 当前版本的问题：如果用户设置了很短的时间，那可能之前的群还没退完，就又退那部分的群，造成一些奇怪的问题，但应该概率不大 + 豹错会被捕获
func (s *IMSession) LongTimeQuitInactiveGroupReborn(threshold time.Time, groupsPerRound int) {
	s.Parent.Logger.Infof("开始清理不活跃群聊. 判定线 %s, 本次退群数: %d", threshold.Format(time.RFC3339), groupsPerRound)
	type GroupEndpointPair struct {
		Group    *GroupInfo
		Endpoint *EndPointInfo
		Last     time.Time
	}
	var selectedGroupEndpoints = make([]*GroupEndpointPair, 0)
	var groupCount int
	s.ServiceAtNew.Range(func(key string, grp *GroupInfo) bool {
		// 如果是PG开头的，忽略掉
		if strings.HasPrefix(grp.GroupID, "PG-") {
			return true
		}
		// 如果在BanList（这应该是白名单？）内，忽略掉
		if s.Parent.Config.BanList != nil {
			info, ok := s.Parent.Config.BanList.GetByID(grp.GroupID)
			if ok && info.Rank > BanRankNormal {
				return true // 信任等级高于普通的不清理
			}
		}
		// 看看是不是QQ群，如果是QQ群，才进一步判断
		match := platformRE.FindStringSubmatch(grp.GroupID)
		if len(match) != 2 {
			return true
		}
		platform := match[1]
		if platform != "QQ" {
			return true
		}
		// 获取上次骰子活动时间
		last := time.Unix(grp.RecentDiceSendTime, 0)
		// 如果enter是进入时间，它比活动时间更晚（说明骰子刚进去，但是骰子还没有说话），那么上次骰子活动时间=进入时间
		if enter := time.Unix(grp.EnteredTime, 0); enter.After(last) {
			last = enter
		}
		// 如果在上述所有操作后，发现时间仍然是0，那么必须忽略该值，因为可能是还没初始化的群，不能人家刚进来就走
		// 注意不能用last.Equal(time.Time{})，因为这里是时间戳的1970-01-01，而Go初始时间是0000-01-01.
		// 预防性代码：如果last是0000-01-01，那也不应该被退群。
		if last.Unix() <= 0 {
			return true
		}
		// 如果时间比要退群的时间早
		if last.Before(threshold) {
			for _, ep := range s.EndPoints {
				// 找到对应的endpoints，并准备退掉它的群
				if ep.Platform != platform || !grp.DiceIDExistsMap.Exists(ep.UserID) {
					continue
				}
				selectedGroupEndpoints = append(selectedGroupEndpoints, &GroupEndpointPair{Group: grp, Endpoint: ep, Last: last})
				// 如果群数量超过本次要退的群数量，就不再继续了，退出出去
				groupCount++
				// 如果已经超过了一次退群的数量，则退出循环
				if groupCount > groupsPerRound {
					return false
				}
			}
		}
		return true
	})
	// 循环完毕，要不然是因为够了要退的数量，要不就是遍历完毕了，但是不够，总之要进行退群活动了
	go func() {
		if r := recover(); r != nil {
			log := zap.S().Named(logger.LogKeyAdapter)
			log.Errorf("自动退群异常: %v 堆栈: %v", r, string(debug.Stack()))
		}
		for i, pair := range selectedGroupEndpoints {
			grp := pair.Group
			ep := pair.Endpoint
			last := pair.Last
			hint := fmt.Sprintf("检测到群 %s 上次活动时间为 %s，尝试退出,当前为本轮第 %d 个", grp.GroupID, last.Format(time.RFC3339), i+1)
			s.Parent.Logger.Info(hint)
			// 创建对应退群信息
			msgCtx := CreateTempCtx(ep, &Message{
				MessageType: "group",
				Sender:      SenderBase{UserID: ep.UserID},
				GroupID:     grp.GroupID,
			})
			// 发送退群消息
			msgText := DiceFormatTmpl(msgCtx, "核心:骰子自动退群告别语")
			ep.Adapter.SendToGroup(msgCtx, grp.GroupID, msgText, "")
			// 退群在退群消息延迟两秒后发送，确保消息发送完成
			time.Sleep(2 * time.Second)
			// 删除群聊绑定信息，更新群处理时间
			grp.DiceIDExistsMap.Delete(ep.UserID)
			grp.MarkDirty(msgCtx.Dice)
			// 执行真正的退群活动，理论上这个msgCtx就能直接用
			ep.Adapter.QuitGroup(msgCtx, grp.GroupID)
			// 发出提示
			msgCtx.Notice(hint)
			// 生成一个随机值（8~11秒随机）
			randomSleep := time.Duration(rand.Intn(3000)+8000) * time.Millisecond
			logger.M().Infof("退群等待，等待 %f 秒后继续", randomSleep.Seconds())
			time.Sleep(randomSleep)
		}
	}()
}

// FormatBlacklistReasons 格式化黑名单原因文本
func FormatBlacklistReasons(v *BanListInfoItem) string {
	var sb strings.Builder
	sb.WriteString("黑名单原因：")
	if v == nil {
		sb.WriteString("\n")
		sb.WriteString("原因未知，请联系开发者获取进一步信息")
		return sb.String()
	}
	for i, reason := range v.Reasons {
		sb.WriteString("\n")
		sb.WriteString(carbon.CreateFromTimestamp(v.Times[i]).ToDateTimeString())
		sb.WriteString("在「")
		sb.WriteString(v.Places[i])
		sb.WriteString("」，原因：")
		sb.WriteString(reason)
	}
	reasontext := sb.String()
	return reasontext
}

// checkBan 黑名单拦截
func checkBan(ctx *MsgContext, msg *Message) (notReply bool) {
	d := ctx.Dice
	log := d.Logger
	var isBanGroup, isWhiteGroup bool
	// log.Info("check ban ", msg.MessageType, " ", msg.GroupID, " ", ctx.PrivilegeLevel)
	if msg.MessageType == "group" {
		value, exists := d.Config.BanList.GetByID(msg.GroupID)
		if exists {
			if value.Rank == BanRankBanned {
				isBanGroup = true
			}
			if value.Rank == BanRankTrusted {
				isWhiteGroup = true
			}
		}
	}

	banQuitGroup := func() {
		banListInfoItem, _ := ctx.Dice.Config.BanList.GetByID(msg.Sender.UserID)
		reasontext := FormatBlacklistReasons(banListInfoItem)
		groupID := msg.GroupID
		noticeMsg := fmt.Sprintf("检测到群(%s)内黑名单用户<%s>(%s)，自动退群\n%s", groupID, msg.Sender.Nickname, msg.Sender.UserID, reasontext)
		log.Info(noticeMsg)

		text := fmt.Sprintf("因<%s>(%s)是黑名单用户，将自动退群。", msg.Sender.Nickname, msg.Sender.UserID)
		ReplyGroupRaw(ctx, &Message{GroupID: groupID}, text, "")

		ctx.Notice(noticeMsg)

		time.Sleep(1 * time.Second)
		ctx.EndPoint.Adapter.QuitGroup(ctx, groupID)
	}

	if ctx.PrivilegeLevel == -30 {
		groupLevel := ctx.GroupRoleLevel
		if (d.Config.BanList.BanBehaviorQuitIfAdmin || d.Config.BanList.BanBehaviorQuitIfAdminSilentIfNotAdmin) && msg.MessageType == "group" {
			// 黑名单用户 - 立即退出所在群
			banListInfoItem, _ := ctx.Dice.Config.BanList.GetByID(msg.Sender.UserID)
			reasontext := FormatBlacklistReasons(banListInfoItem)
			groupID := msg.GroupID
			notReply = true
			if groupLevel >= 40 {
				if isWhiteGroup {
					log.Infof("收到群(%s)内邀请者以上权限黑名单用户<%s>(%s)的消息，但在信任群所以不尝试退群", groupID, msg.Sender.Nickname, msg.Sender.UserID)
				} else {
					text := fmt.Sprintf("警告: <%s>(%s)是黑名单用户，将对骰主进行通知并退群。", msg.Sender.Nickname, msg.Sender.UserID)
					ReplyGroupRaw(ctx, &Message{GroupID: groupID}, text, "")

					noticeMsg := fmt.Sprintf("检测到群(%s)内黑名单用户<%s>(%s)，因是管理以上权限，执行通告后自动退群\n%s", groupID, msg.Sender.Nickname, msg.Sender.UserID, reasontext)
					log.Info(noticeMsg)
					ctx.Notice(noticeMsg)
					banQuitGroup()
				}
			} else {
				if isWhiteGroup {
					log.Infof("收到群(%s)内普通群员黑名单用户<%s>(%s)的消息，但在信任群所以不做其他操作", groupID, msg.Sender.Nickname, msg.Sender.UserID)
				} else {
					notReply = true
					if d.Config.BanList.BanBehaviorQuitIfAdmin {
						noticeMsg := fmt.Sprintf("检测到群(%s)内黑名单用户<%s>(%s)，因是普通群员，进行群内通告\n%s", groupID, msg.Sender.Nickname, msg.Sender.UserID, reasontext)
						log.Info(noticeMsg)

						text := fmt.Sprintf("警告: <%s>(%s)是黑名单用户，将对骰主进行通知。", msg.Sender.Nickname, msg.Sender.UserID)
						ReplyGroupRaw(ctx, &Message{GroupID: groupID}, text, "")

						ctx.Notice(noticeMsg)
					} else {
						noticeMsg := fmt.Sprintf("检测到群(%s)内黑名单用户<%s>(%s)，因是普通群员，忽略黑名单用户信息，不做其他操作\n%s", groupID, msg.Sender.Nickname, msg.Sender.UserID, reasontext)
						log.Info(noticeMsg)
					}
				}
			}
		} else if d.Config.BanList.BanBehaviorQuitPlaceImmediately && msg.MessageType == "group" {
			notReply = true
			// 黑名单用户 - 立即退出所在群
			groupID := msg.GroupID
			if isWhiteGroup {
				log.Infof("收到群(%s)内黑名单用户<%s>(%s)的消息，但在信任群所以不尝试退群", groupID, msg.Sender.Nickname, msg.Sender.UserID)
			} else {
				banQuitGroup()
			}
		} else if d.Config.BanList.BanBehaviorRefuseReply {
			notReply = true
			// 黑名单用户 - 拒绝回复
			log.Infof("忽略黑名单用户信息: 来自群(%s)内<%s>(%s): %s", msg.GroupID, msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
		}
	} else if isBanGroup {
		if d.Config.BanList.BanBehaviorQuitPlaceImmediately && !isWhiteGroup {
			notReply = true
			// 黑名单群 - 立即退出
			// 退群使用GroupID进行判断
			banListInfoItem, _ := ctx.Dice.Config.BanList.GetByID(msg.GroupID)
			reasontext := FormatBlacklistReasons(banListInfoItem)
			groupID := msg.GroupID
			if isWhiteGroup {
				log.Infof("群(%s)处于黑名单中，但在信任群所以不尝试退群", groupID)
			} else {
				noticeMsg := fmt.Sprintf("群(%s)处于黑名单中，自动退群\n%s", groupID, reasontext)
				log.Info(noticeMsg)

				ReplyGroupRaw(ctx, &Message{GroupID: groupID}, "因本群处于黑名单中，将自动退群。", "")

				ctx.Notice(noticeMsg)

				time.Sleep(1 * time.Second)
				ctx.EndPoint.Adapter.QuitGroup(ctx, groupID)
			}
		} else if d.Config.BanList.BanBehaviorRefuseReply {
			notReply = true
			// 黑名单群 - 拒绝回复
			log.Infof("忽略黑名单群消息: 来自群(%s)内<%s>(%s): %s", msg.GroupID, msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
		}
	}
	return notReply
}

func (s *IMSession) commandSolve(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) bool {
	// 设置临时变量
	if ctx.Player != nil {
		SetTempVars(ctx, msg.Sender.Nickname)
		VarSetValueStr(ctx, "$tMsgID", fmt.Sprintf("%v", msg.RawID))
		VarSetValueInt64(ctx, "$t轮数", int64(cmdArgs.SpecialExecuteTimes))
	}

	tryItemSolve := func(ext *ExtInfo, item *CmdItemInfo) bool {
		if item == nil {
			return false
		}

		if item.Raw { //nolint:nestif
			if item.CheckCurrentBotOn {
				if !ctx.IsCurGroupBotOn && !ctx.IsPrivate {
					return false
				}
			}

			if item.CheckMentionOthers {
				if cmdArgs.SomeoneBeMentionedButNotMe {
					return false
				}
			}
		} else { //nolint:gocritic
			// 默认模式行为：需要在当前群/私聊开启，或@自己时生效(需要为第一个@目标)
			if !ctx.IsCurGroupBotOn && !ctx.IsPrivate {
				return false
			}
		}

		if ext != nil && ext.DefaultSetting.DisabledCommand[item.Name] {
			ReplyToSender(ctx, msg, fmt.Sprintf("此指令已被骰主禁用: %s:%s", ext.Name, item.Name))
			return true
		}

		// Note(Szzrain): TODO: 意义不明，需要想办法干掉
		if item.EnableExecuteTimesParse {
			cmdArgs.RevokeExecuteTimesParse(ctx, msg)
		}

		if ctx.Player != nil {
			VarSetValueInt64(ctx, "$t轮数", int64(cmdArgs.SpecialExecuteTimes))
		}

		if !item.Raw {
			if item.DisabledInPrivate && ctx.IsPrivate {
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_私聊不可用"))
				return false
			}

			if item.AllowDelegate {
				// 允许代骰时，发一句话
				cur := -1
				for index, i := range cmdArgs.At {
					if i.UserID == ctx.EndPoint.UserID {
						continue
					} else if strings.HasPrefix(ctx.EndPoint.UserID, "OpenQQ:") {
						// 特殊处理 OpenQQ频道
						uid := strings.TrimPrefix(i.UserID, "OpenQQCH:")
						diceId := strings.TrimPrefix(ctx.EndPoint.UserID, "OpenQQ:")
						if uid == diceId {
							continue
						}
					}
					cur = index
				}

				if cur != -1 {
					if ctx.Dice.Config.PlayerNameWrapEnable {
						ctx.DelegateText = fmt.Sprintf("由<%s>代骰:\n", ctx.Player.Name)
					} else {
						ctx.DelegateText = fmt.Sprintf("由%s代骰:\n", ctx.Player.Name)
					}
				}
			} else if cmdArgs.SomeoneBeMentionedButNotMe {
				// 如果其他人被@了就不管
				// 注: 如果被@的对象在botlist列表，那么不会走到这一步
				return false
			}
		}

		// 加载规则模板
		// TODO: 注意一下这里使用群模板还是个人卡模板，目前群模板，可有情况特殊？
		tmpl := ctx.SystemTemplate
		if tmpl != nil {
			ctx.Eval(tmpl.InitScript, nil)
			if tmpl.Name == "dnd5e" {
				// 这里面有buff机制的代码，所以需要加载
				ctx.setDndReadForVM(false)
			}
		}

		var ret CmdExecuteResult
		// 如果是js命令，那么加锁
		if item.IsJsSolveFunc {
			loop, err := s.Parent.ExtLoopManager.GetLoop(item.JSLoopVersion)
			if err != nil {
				// 打个DEBUG日志？
				s.Parent.Logger.Errorf("扩展注册的指令<%s>运行环境已经过期: %v", item.Name, err)
				return false
			}
			waitRun := make(chan int, 1)
			loop.RunOnLoop(func(vm *goja.Runtime) {
				defer func() {
					if r := recover(); r != nil {
						// log.Errorf("异常: %v 堆栈: %v", r, string(debug.Stack()))
						ReplyToSender(ctx, msg, fmt.Sprintf("JS执行异常，请反馈给该扩展的作者：\n%v", r))
					}
					waitRun <- 1
				}()

				ret = item.Solve(ctx, msg, cmdArgs)
			})
			<-waitRun
		} else {
			ret = item.Solve(ctx, msg, cmdArgs)
		}

		if ret.Solved {
			if ret.ShowHelp {
				help := ""
				// 优先考虑函数
				if item.HelpFunc != nil {
					help = item.HelpFunc(false)
				}
				// 其次考虑help
				if help == "" {
					help = item.Help
				}
				// 最后用短help拼
				if help == "" {
					// 这是为了防止别的骰子误触发
					help = item.Name + ":\n" + item.ShortHelp
				}
				ReplyToSender(ctx, msg, help)
			}

			return true
		}
		return false
	}

	group := ctx.Group
	builtinSolve := func() bool {
		item := ctx.Session.Parent.CmdMap[cmdArgs.Command]
		if tryItemSolve(nil, item) {
			return true
		}

		if group != nil && (group.Active || ctx.IsCurGroupBotOn) {
			for _, wrapper := range group.GetActivatedExtList(ctx.Dice) {
				cmdMap := wrapper.GetCmdMap()
				item := cmdMap[cmdArgs.Command]
				if tryItemSolve(wrapper, item) {
					return true
				}
			}
		}
		return false
	}

	solved := builtinSolve()
	if group.Active || ctx.IsCurGroupBotOn {
		for _, wrapper := range group.GetActivatedExtList(ctx.Dice) {
			ext := wrapper.GetRealExt()
			if ext == nil {
				continue
			}
			if ext.OnCommandReceived != nil {
				ext.callWithJsCheck(ctx.Dice, func() {
					ext.OnCommandReceived(ctx, msg, cmdArgs)
				})
			}
		}
	}

	return solved
}

func (s *IMSession) OnMessageDeleted(mctx *MsgContext, msg *Message) {
	d := mctx.Dice
	mctx.MessageType = msg.MessageType
	mctx.IsPrivate = mctx.MessageType == "private"
	group, ok := s.ServiceAtNew.Load(msg.GroupID)
	if !ok {
		return
	}
	mctx.Group = group
	mctx.Group, mctx.Player = GetPlayerInfoBySender(mctx, msg)

	mctx.IsCurGroupBotOn = msg.MessageType == "group" && mctx.Group.IsActive(mctx)
	if mctx.Group != nil && mctx.Group.System != "" {
		mctx.SystemTemplate = mctx.Group.GetCharTemplate(d)
		// tmpl, _ := d.GameSystemMap.Load(group.System)
		// mctx.SystemTemplate = tmpl
	}

	_ = mctx.fillPrivilege(msg)

	for _, i := range s.Parent.ExtList {
		i.CallOnMessageDeleted(mctx.Dice, mctx, msg)
	}
}

func (s *IMSession) OnMessageSend(ctx *MsgContext, msg *Message, flag string) {
	for _, i := range s.Parent.ExtList {
		i.CallOnMessageSend(ctx.Dice, ctx, msg, flag)
	}
}

func (s *IMSession) OnPoke(ctx *MsgContext, event *events.PokeEvent) {
	if ctx == nil || event == nil {
		return
	}
	// Poke 事件可能缺少群/成员信息（例如 OneBot 获取群成员信息失败时），避免空指针导致崩溃。
	if ctx.Group == nil && event.GroupID != "" {
		if group, ok := s.ServiceAtNew.Load(event.GroupID); ok {
			ctx.Group = group
		} else {
			// 确保群信息至少被初始化到全局列表，便于后续扩展读取/写入
			ctx.Group = SetBotOnAtGroup(ctx, event.GroupID)
		}
	}
	if ctx.Group == nil {
		return
	}
	if ctx.MessageType == "group" && !ctx.Group.IsActive(ctx) {
		return
	}
	for _, wrapper := range ctx.Group.GetActivatedExtList(ctx.Dice) {
		ext := wrapper.GetRealExt()
		if ext == nil {
			continue
		}
		if ext.OnPoke != nil {
			ext.callWithJsCheck(ctx.Dice, func() {
				ext.OnPoke(ctx, event)
			})
		}
	}
}

func (s *IMSession) OnGroupLeave(ctx *MsgContext, event *events.GroupLeaveEvent) {
	for _, i := range s.Parent.ExtList {
		i.CallOnGroupLeave(ctx.Dice, ctx, event)
	}
}

// OnMessageEdit 消息编辑事件
//
// msg.Message 应为更新后的消息, msg.Time 应为更新时间而非发送时间，同时
// msg.RawID 应确保为原消息的 ID (一些 API 同时会有系统事件 ID，勿混淆)
//
// 依据 API，Sender 不一定存在，ctx 信息亦不一定有效
func (s *IMSession) OnMessageEdit(ctx *MsgContext, msg *Message) {
	m := fmt.Sprintf("来自%s的消息修改事件: %s",
		msg.GroupID,
		msg.Message,
	)
	s.Parent.Logger.Info(m)

	if group, ok := s.ServiceAtNew.Load(msg.GroupID); ok {
		ctx.Group = group
	} else {
		return
	}

	group := ctx.Group
	if group.Active || ctx.IsCurGroupBotOn {
		for _, wrapper := range group.GetActivatedExtList(ctx.Dice) {
			ext := wrapper.GetRealExt()
			if ext == nil {
				continue
			}
			if ext.OnMessageEdit != nil {
				ext.callWithJsCheck(ctx.Dice, func() {
					ext.OnMessageEdit(ctx, msg)
				})
			}
		}
	}
}

// GetEpByPlatform
// 在 EndPoints 中找到第一个符合平台 p 且启用的
func (s *IMSession) GetEpByPlatform(p string) *EndPointInfo {
	for _, ep := range s.EndPoints {
		if ep.Enable && ep.Platform == p {
			return ep
		}
	}
	return nil
}

// SetEnable
/* 如果已连接，将断开连接，如果开着GCQ将自动结束。如果启用的话，则反过来  */
func (ep *EndPointInfo) SetEnable(_ *Dice, enable bool) {
	if ep.Enable != enable {
		ep.Adapter.SetEnable(enable)
	}
}

func (ep *EndPointInfo) AdapterSetup() {
	switch ep.Platform {
	case "QQ":
		switch ep.ProtocolType {
		case "onebot":
			pa := ep.Adapter.(*PlatformAdapterGocq)
			pa.Session = ep.Session
			pa.EndPoint = ep
		case "walle-q":
			pa := ep.Adapter.(*PlatformAdapterWalleQ)
			pa.Session = ep.Session
			pa.EndPoint = ep
		case "red":
			pa := ep.Adapter.(*PlatformAdapterRed)
			pa.Session = ep.Session
			pa.EndPoint = ep
		case "official":
			pa := ep.Adapter.(*PlatformAdapterOfficialQQ)
			pa.Session = ep.Session
			pa.EndPoint = ep
		case "satori":
			pa := ep.Adapter.(*PlatformAdapterSatori)
			pa.Session = ep.Session
			pa.EndPoint = ep
		case "milky":
			pa := ep.Adapter.(*PlatformAdapterMilky)
			pa.Session = ep.Session
			pa.EndPoint = ep
		case "pureonebot":
			pa := ep.Adapter.(*PlatformAdapterOnebot)
			log := zap.S().Named(logger.LogKeyAdapter)
			pa.Session = ep.Session
			pa.EndPoint = ep
			pa.logger = log
			pa.desiredEnabled = ep.Enable
			// case "LagrangeGo":
			//	pa := ep.Adapter.(*PlatformAdapterLagrangeGo)
			//	pa.Session = ep.Session
			//	pa.EndPoint = ep
		}
	case "DISCORD":
		pa := ep.Adapter.(*PlatformAdapterDiscord)
		pa.Session = ep.Session
		pa.EndPoint = ep
	case "KOOK":
		pa := ep.Adapter.(*PlatformAdapterKook)
		pa.Session = ep.Session
		pa.EndPoint = ep
	case "TG":
		pa := ep.Adapter.(*PlatformAdapterTelegram)
		pa.Session = ep.Session
		pa.EndPoint = ep
	case "MC":
		pa := ep.Adapter.(*PlatformAdapterMinecraft)
		pa.Session = ep.Session
		pa.EndPoint = ep
	case "DODO":
		pa := ep.Adapter.(*PlatformAdapterDodo)
		pa.Session = ep.Session
		pa.EndPoint = ep
	case "DINGTALK":
		pa := ep.Adapter.(*PlatformAdapterDingTalk)
		pa.Session = ep.Session
		pa.EndPoint = ep
	case "SLACK":
		pa := ep.Adapter.(*PlatformAdapterSlack)
		pa.Session = ep.Session
		pa.EndPoint = ep
	case "SEALCHAT":
		pa := ep.Adapter.(*PlatformAdapterSealChat)
		pa.Session = ep.Session
		pa.EndPoint = ep
	}
}

func (ep *EndPointInfo) RefreshGroupNum() {
	serveCount := 0
	session := ep.Session
	if session != nil && session.ServiceAtNew != nil {
		// Pinenutn: Range模板 ServiceAtNew重构代码
		session.ServiceAtNew.Range(func(key string, groupInfo *GroupInfo) bool {
			// Pinenutn: ServiceAtNew重构
			if groupInfo.GroupID != "" {
				if strings.HasPrefix(groupInfo.GroupID, "PG-") {
					return true
				}
				if groupInfo.DiceIDExistsMap.Exists(ep.UserID) {
					serveCount++
					// 在群内的开启数量才被计算，虽然也有被踢出的
					// if groupInfo.DiceIdActiveMap.Exists(ep.UserId) {
					// activeCount += 1
					// }
				}
			}
			return true
		})
		ep.GroupNum = int64(serveCount)
	}
}

func (d *Dice) NoticeForEveryEndpoint(txt string, allowCrossPlatform bool) {
	_ = allowCrossPlatform
	// 通知种类之一：每个noticeId  *  每个平台匹配的ep：存活
	// TODO: 先复制几次实现，后面重构
	// Pinenutn: 啥时候重构啊.jpg
	foo := func() {
		defer func() {
			if r := recover(); r != nil {
				d.Logger.Errorf("发送通知异常: %v 堆栈: %v", r, string(debug.Stack()))
			}
		}()

		if d.Config.MailEnable {
			_ = d.SendMail(txt, MailTypeNotice)
			return
		}

		for _, ep := range d.ImSession.EndPoints {
			for _, i := range d.Config.NoticeIDs {
				n := strings.Split(i, ":")
				// 如果文本中没有-，则会取到整个字符串
				// 但好像不严谨，比如QQ-CH-Group
				prefix := strings.Split(n[0], "-")[0]

				if len(n) >= 2 && prefix == ep.Platform && ep.Enable && ep.State == 1 {
					if ep.Session == nil {
						ep.Session = d.ImSession
					}
					if strings.HasSuffix(n[0], "-Group") {
						msg := &Message{GroupID: i, MessageType: "private", Sender: SenderBase{UserID: i}}
						ctx := CreateTempCtx(ep, msg)
						ReplyGroup(ctx, msg, txt)
					} else {
						msg := &Message{GroupID: i, MessageType: "group", Sender: SenderBase{UserID: i}}
						ctx := CreateTempCtx(ep, msg)
						ReplyPerson(ctx, &Message{Sender: SenderBase{UserID: i}}, txt)
					}
				}
				time.Sleep(1 * time.Second)
			}
		}
	}
	go foo()
}

func (ctx *MsgContext) NoticeCrossPlatform(txt string) {
	// 通知种类之二：每个noticeID  *  第一个平台匹配的ep：跨平台通知
	// TODO: 先复制几次实现，后面重构
	foo := func() {
		defer func() {
			if r := recover(); r != nil {
				ctx.Dice.Logger.Errorf("发送通知异常: %v 堆栈: %v", r, string(debug.Stack()))
			}
		}()

		if ctx.Dice.Config.MailEnable {
			_ = ctx.Dice.SendMail(txt, MailTypeNotice)
			return
		}

		sent := false

		for _, i := range ctx.Dice.Config.NoticeIDs {
			n := strings.Split(i, ":")
			if len(n) < 2 {
				continue
			}

			seg := strings.Split(n[0], "-")[0]

			messageType := "private"
			if strings.HasSuffix(n[0], "-Group") {
				messageType = "group"
			}

			if ctx.EndPoint.Platform == seg {
				if messageType == "group" {
					ReplyGroup(ctx, &Message{GroupID: i}, txt)
				} else {
					ReplyPerson(ctx, &Message{Sender: SenderBase{UserID: i}}, txt)
				}
				time.Sleep(1 * time.Second)
				sent = true
				continue // 找到对应平台、调用了发送的在此即切出循环
			}

			// 如果走到这里，说明当前ep不是noticeID对应的平台
			if done := CrossMsgBySearch(ctx.Session, seg, i, txt, messageType == "private"); !done {
				ctx.Dice.Logger.Errorf("尝试跨平台后仍未能向 %s 发送通知：%s", i, txt)
			} else {
				sent = true
				time.Sleep(1 * time.Second)
			}
		}

		if !sent {
			ctx.Dice.Logger.Errorf("未能发送来自%s的通知：%s", ctx.EndPoint.Platform, txt)
		}
	}
	go foo()
}

func (ctx *MsgContext) Notice(txt string) {
	// Notice
	// 通知种类之三：每个noticeID  * 当前mctx的ep：不跨平台通知
	// TODO: 先复制几次实现，后面重构
	foo := func() {
		defer func() {
			if r := recover(); r != nil {
				ctx.Dice.Logger.Errorf("发送通知异常: %v 堆栈: %v", r, string(debug.Stack()))
			}
		}()

		if ctx.Dice.Config.MailEnable {
			_ = ctx.Dice.SendMail(txt, MailTypeNotice)
			return
		}

		sent := false
		if ctx.EndPoint.Enable {
			for _, i := range ctx.Dice.Config.NoticeIDs {
				n := strings.Split(i, ":")
				if len(n) >= 2 {
					if strings.HasSuffix(n[0], "-Group") {
						ReplyGroup(ctx, &Message{GroupID: i}, txt)
					} else {
						ReplyPerson(ctx, &Message{Sender: SenderBase{UserID: i}}, txt)
					}
					sent = true
				}
				time.Sleep(1 * time.Second)
			}
		}

		if !sent {
			if len(ctx.Dice.Config.NoticeIDs) != 0 {
				ctx.Dice.Logger.Errorf("未能发送来自%s的通知：%s", ctx.EndPoint.Platform, txt)
			} else {
				ctx.Dice.Logger.Warnf("因为没有配置通知列表，无法发送来自%s的通知：%s", ctx.EndPoint.Platform, txt)
			}
		}
	}
	go foo()
}

var randSourceSplitKey = rand2.NewSource(uint64(time.Now().Unix()))

func (ctx *MsgContext) InitSplitKey() {
	if len(ctx.splitKey) > 0 {
		return
	}
	r := randSourceSplitKey.Uint64()
	bArray := make([]byte, 12)
	binary.LittleEndian.PutUint64(bArray[:8], r)
	r = randSourceSplitKey.Uint64()
	binary.LittleEndian.PutUint32(bArray[8:], uint32(r))

	s := base64.StdEncoding.EncodeToString(bArray)
	ctx.splitKey = "###" + s + "###"
}

func (ctx *MsgContext) TranslateSplit(s string) string {
	if len(ctx.splitKey) == 0 {
		ctx.InitSplitKey()
	}
	s = strings.ReplaceAll(s, "#{SPLIT}", ctx.splitKey)
	s = strings.ReplaceAll(s, "{FormFeed}", ctx.splitKey)
	s = strings.ReplaceAll(s, "{formfeed}", ctx.splitKey)
	s = strings.ReplaceAll(s, "\f", ctx.splitKey)
	s = strings.ReplaceAll(s, "\\f", ctx.splitKey)
	return s
}

func (ctx *MsgContext) SplitText(text string) []string {
	if len(ctx.splitKey) == 0 {
		return []string{text}
	}
	return strings.Split(text, ctx.splitKey)
}

var curCommandID int64 = 0

func getNextCommandID() int64 {
	curCommandID++
	return curCommandID
}
