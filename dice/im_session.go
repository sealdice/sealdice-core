package dice

import (
	"fmt"
	"regexp"
	"runtime/debug"
	"sealdice-core/dice/model"
	"sort"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/fy0/lockfree"
	"github.com/jmoiron/sqlx"
	"golang.org/x/time/rate"
	"gopkg.in/yaml.v3"
)

type SenderBase struct {
	Nickname  string `json:"nickname" jsbind:"nickname"`
	UserID    string `json:"userId" jsbind:"userId"`
	GroupRole string `json:"-"` // 群内角色 admin管理员 owner群主
}

// Message 消息的重要信息
// 时间
// 发送地点(群聊/私聊)
// 人物(是谁发的)
// 内容
type Message struct {
	Time        int64       `json:"time" jsbind:"time"`               // 发送时间
	MessageType string      `json:"messageType" jsbind:"messageType"` // group private
	GroupID     string      `json:"groupId" jsbind:"groupId"`         // 群号，如果是群聊消息
	GuildID     string      `json:"guildId" jsbind:"guildId"`         // 服务器群组号，会在discord,kook,dodo等平台见到
	Sender      SenderBase  `json:"sender" jsbind:"sender"`           // 发送者
	Message     string      `json:"message" jsbind:"message"`         // 消息内容
	RawID       interface{} `json:"rawId" jsbind:"rawId"`             // 原始信息ID，用于处理撤回等
	Platform    string      `json:"platform" jsbind:"platform"`       // 当前平台
	TmpUID      string      `json:"-" yaml:"-"`
}

// GroupPlayerInfoBase 群内玩家信息
type GroupPlayerInfoBase struct {
	Name                string `yaml:"name" jsbind:"name"` // 玩家昵称
	UserID              string `yaml:"userId" jsbind:"userId"`
	InGroup             bool   `yaml:"inGroup"`                                          // 是否在群内，有时一个人走了，信息还暂时残留
	LastCommandTime     int64  `yaml:"lastCommandTime" jsbind:"lastCommandTime"`         // 上次发送指令时间
	AutoSetNameTemplate string `yaml:"autoSetNameTemplate" jsbind:"autoSetNameTemplate"` // 名片模板

	DiceSideNum  int                  `yaml:"diceSideNum"` // 面数，为0时等同于d100
	Vars         *PlayerVariablesItem `yaml:"-"`           // 玩家的群内变量
	ValueMapTemp lockfree.HashMap     `yaml:"-"`           // 玩家的群内临时变量

	TempValueAlias *map[string][]string `yaml:"-"` // 群内临时变量别名 - 其实这个有点怪的，为什么在这里？

	UpdatedAtTime int64 `yaml:"-" json:"-"`
}

// GroupPlayerInfo 这是一个YamlWrapper，没有实际作用
// 原因见 https://github.com/go-yaml/yaml/issues/712
// type GroupPlayerInfo struct {
// 	GroupPlayerInfoBase `yaml:",inline,flow"`
// }

type GroupPlayerInfo model.GroupPlayerInfoBase

type GroupInfo struct {
	Active           bool                               `json:"active" yaml:"active" jsbind:"active"`          // 是否在群内开启 - 过渡为象征意义
	ActivatedExtList []*ExtInfo                         `yaml:"activatedExtList,flow" json:"activatedExtList"` // 当前群开启的扩展列表
	Players          *SyncMap[string, *GroupPlayerInfo] `yaml:"-" json:"-"`                                    // 群员角色数据

	GroupID         string                 `yaml:"groupId" json:"groupId" jsbind:"groupId"`
	GuildID         string                 `yaml:"guildId" json:"guildId" jsbind:"guildId"`
	GroupName       string                 `yaml:"groupName" json:"groupName" jsbind:"groupName"`
	DiceIDActiveMap *SyncMap[string, bool] `yaml:"diceIds,flow" json:"diceIdActiveMap"` // 对应的骰子ID(格式 平台:ID)，对应单骰多号情况，例如骰A B都加了群Z，A退群不会影响B在群内服务
	DiceIDExistsMap *SyncMap[string, bool] `yaml:"-" json:"diceIdExistsMap"`            // 对应的骰子ID(格式 平台:ID)是否存在于群内
	BotList         *SyncMap[string, bool] `yaml:"botList,flow" json:"botList"`         // 其他骰子列表
	DiceSideNum     int64                  `yaml:"diceSideNum" json:"diceSideNum"`      // 以后可能会支持 1d4 这种默认面数，暂不开放给js
	System          string                 `yaml:"system" json:"system"`                // 规则系统，概念同bcdice的gamesystem，距离如dnd5e coc7

	// ValueMap     map[string]*VMValue `yaml:"-"`
	ValueMap     lockfree.HashMap `yaml:"-" json:"-"`
	HelpPackages []string         `yaml:"-" json:"helpPackages"`
	CocRuleIndex int              `yaml:"cocRuleIndex" json:"cocRuleIndex" jsbind:"cocRuleIndex"`
	LogCurName   string           `yaml:"logCurFile" json:"logCurName" jsbind:"logCurName"`
	LogOn        bool             `yaml:"logOn" json:"logOn" jsbind:"logOn"`

	QuitMarkAutoClean   bool   `yaml:"-" json:"-"` // 自动清群 - 播报，即将自动退出群组
	QuitMarkMaster      bool   `yaml:"-" json:"-"` // 骰主命令退群 - 播报，即将自动退出群组
	RecentDiceSendTime  int64  `json:"recentDiceSendTime" jsbind:"recentDiceSendTime"`
	ShowGroupWelcome    bool   `yaml:"showGroupWelcome" json:"showGroupWelcome" jsbind:"showGroupWelcome"` // 是否迎新
	GroupWelcomeMessage string `yaml:"groupWelcomeMessage" json:"groupWelcomeMessage" jsbind:"groupWelcomeMessage"`
	// FirstSpeechMade     bool   `yaml:"firstSpeechMade"` // 是否做过进群发言
	LastCustomReplyTime float64 `yaml:"-" json:"-"` // 上次自定义回复时间

	RateLimiter     *rate.Limiter `yaml:"-" json:"-"`
	RateLimitWarned bool          `yaml:"-" json:"-"`

	EnteredTime  int64  `yaml:"enteredTime" json:"enteredTime" jsbind:"enteredTime"`    // 入群时间
	InviteUserID string `yaml:"inviteUserId" json:"inviteUserId" jsbind:"inviteUserId"` // 邀请人
	// 仅用于http接口
	TmpPlayerNum int64    `yaml:"-" json:"tmpPlayerNum"`
	TmpExtList   []string `yaml:"-" json:"tmpExtList"`

	UpdatedAtTime int64 `yaml:"-" json:"-"`
}

// ExtActive 开启扩展
func (group *GroupInfo) ExtActive(ei *ExtInfo) {
	lst := []*ExtInfo{ei}
	oldLst := group.ActivatedExtList
	group.ActivatedExtList = append(lst, oldLst...) //nolint:gocritic
	group.ExtClear()
}

// ExtClear 清除多余的扩展项
func (group *GroupInfo) ExtClear() {
	m := map[string]bool{}
	var lst []*ExtInfo

	for _, i := range group.ActivatedExtList {
		if !m[i.Name] {
			m[i.Name] = true
			lst = append(lst, i)
		}
	}
	group.ActivatedExtList = lst
}

func (group *GroupInfo) ExtInactive(ei *ExtInfo) *ExtInfo {
	if ei.Storage != nil {
		_ = ei.Storage.Close()
		ei.Storage = nil
	}
	for index, i := range group.ActivatedExtList {
		if ei == i {
			group.ActivatedExtList = append(group.ActivatedExtList[:index], group.ActivatedExtList[index+1:]...)
			group.ExtClear()
			return i
		}
	}
	return nil
}

func (group *GroupInfo) ExtInactiveByName(name string) *ExtInfo {
	for index, i := range group.ActivatedExtList {
		if i.Name == name {
			group.ActivatedExtList = append(group.ActivatedExtList[:index], group.ActivatedExtList[index+1:]...)
			group.ExtClear()
			return i
		}
	}
	return nil
}

func (group *GroupInfo) ExtGetActive(name string) *ExtInfo {
	for _, i := range group.ActivatedExtList {
		if i.Name == name {
			return i
		}
	}
	return nil
}

func (group *GroupInfo) IsActive(ctx *MsgContext) bool {
	firstCheck := group.Active && group.DiceIDActiveMap.Len() >= 1
	if firstCheck {
		v, _ := group.DiceIDActiveMap.Load(ctx.EndPoint.UserID)
		return v
	}
	return false
}

func (group *GroupInfo) PlayerGet(db *sqlx.DB, id string) *GroupPlayerInfo {
	if group.Players == nil {
		group.Players = new(SyncMap[string, *GroupPlayerInfo])
	}
	p, exists := group.Players.Load(id)
	if !exists {
		p = (*GroupPlayerInfo)(model.GroupPlayerInfoGet(db, group.GroupID, id))
		if p != nil {
			group.Players.Store(id, p)
		}
	}
	return p
}

func (group *GroupInfo) GetCharTemplate(dice *Dice) *GameSystemTemplate {
	// 有system优先system
	if group.System != "" {
		v, _ := dice.GameSystemMap.Load(group.System)
		if v != nil {
			return v
		}
		// 返回这个单纯是为了不让st将其覆盖
		// 这种情况属于卡片的规则模板被删除了
		return &GameSystemTemplate{
			Name:     group.System,
			FullName: "空白模板",
			AliasMap: &SyncMap[string, string]{},
		}
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
		Name:     "空白模板",
		FullName: "空白模板",
		AliasMap: &SyncMap[string, string]{},
	}
	return blankTmpl
}

type EndPointInfoBase struct {
	ID                  string `yaml:"id" json:"id" jsbind:"id"` // uuid
	Nickname            string `yaml:"nickname" json:"nickname" jsbind:"nickname"`
	State               int    `yaml:"state" json:"state" jsbind:"state"` // 状态 0断开 1已连接 2连接中 3连接失败
	UserID              string `yaml:"userId" json:"userId" jsbind:"userId"`
	GroupNum            int64  `yaml:"groupNum" json:"groupNum" jsbind:"groupNum"`                                  // 拥有群数
	CmdExecutedNum      int64  `yaml:"cmdExecutedNum" json:"cmdExecutedNum" jsbind:"cmdExecutedNum"`                // 指令执行次数
	CmdExecutedLastTime int64  `yaml:"cmdExecutedLastTime" json:"cmdExecutedLastTime" jsbind:"cmdExecutedLastTime"` // 指令执行次数
	OnlineTotalTime     int64  `yaml:"onlineTotalTime" json:"onlineTotalTime" jsbind:"onlineTotalTime"`             // 在线时长

	Platform     string `yaml:"platform" json:"platform" jsbind:"platform"` // 平台，如QQ等
	RelWorkDir   string `yaml:"relWorkDir" json:"relWorkDir"`               // 工作目录
	Enable       bool   `yaml:"enable" json:"enable" jsbind:"enable"`       // 是否启用
	ProtocolType string `yaml:"protocolType" json:"protocolType"`           // 协议类型，如onebot、koishi等

	IsPublic bool       `yaml:"isPublic"`
	Session  *IMSession `yaml:"-" json:"-"`
}

type EndPointInfo struct {
	EndPointInfoBase `yaml:"baseInfo" jsbind:"baseInfo"`

	Adapter PlatformAdapter `yaml:"adapter" json:"adapter"`
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
	}
	return err
}

// StatsRestore 尝试从数据库中恢复EP的统计数据
func (ep *EndPointInfo) StatsRestore(d *Dice) {
	if len(ep.UserID) == 0 {
		return // 尚未连接完成的新账号没有UserId, 跳过
	}

	m := model.EndpointInfo{UserID: ep.UserID}
	err := m.Query(d.DBData)
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
	err := m.Save(d.DBData)
	if err != nil {
		d.Logger.Errorf("保存endpoint数据到数据库失败 %v : %v", ep.UserID, err)
	}
}

type PlayerVariablesItem model.PlayerVariablesItem

type IMSession struct {
	Parent    *Dice           `yaml:"-"`
	EndPoints []*EndPointInfo `yaml:"endPoints"`

	ServiceAtNew   map[string]*GroupInfo           `json:"servicesAt" yaml:"-"`
	PlayerVarsData map[string]*PlayerVariablesItem `yaml:"-"` // 感觉似乎没有什么存本地的必要
}

type MsgContext struct {
	MessageType string
	Group       *GroupInfo       `jsbind:"group"`  // 当前群信息
	Player      *GroupPlayerInfo `jsbind:"player"` // 当前群的玩家数据

	EndPoint        *EndPointInfo `jsbind:"endPoint"` // 对应的Endpoint
	Session         *IMSession    // 对应的IMSession
	Dice            *Dice         // 对应的 Dice
	IsCurGroupBotOn bool          `jsbind:"isCurGroupBotOn"` // 在群内是否bot on

	IsPrivate       bool        `jsbind:"isPrivate"` // 是否私聊
	CommandID       int64       // 指令ID
	CommandHideFlag string      // 暗骰标记
	CommandInfo     interface{} // 命令信息
	PrivilegeLevel  int         `jsbind:"privilegeLevel"` // 权限等级 40邀请者 50管理 60群主 70信任 100master
	DelegateText    string      `jsbind:"delegateText"`   // 代骰附加文本

	deckDepth         int                                         // 抽牌递归深度
	DeckPools         map[*DeckInfo]map[string]*ShuffleRandomPool // 不放回抽取的缓存
	diceExprOverwrite string                                      // 默认骰表达式覆盖
	SystemTemplate    *GameSystemTemplate
	Censored          bool // 已检查过敏感词
}

// func (s *IMSession) GroupEnableCheck(ep *EndPointInfo, msg *Message, runInSync bool) {
// }

// fillPrivilege 填写MsgContext中的权限字段, 并返回填写的权限等级
//   - msg 使用其中的msg.Sender.GroupRole
//
// MsgContext.Dice需要指向一个有效的Dice对象
func (ctx *MsgContext) fillPrivilege(msg *Message) int {
	if ctx.Group != nil && ctx.Dice != nil {
		switch {
		case msg.Sender.GroupRole == "owner":
			ctx.PrivilegeLevel = 60 // 群主
		case msg.Sender.GroupRole == "admin":
			ctx.PrivilegeLevel = 50 // 群管理
		case msg.Sender.UserID == ctx.Group.InviteUserID:
			ctx.PrivilegeLevel = 40 // 邀请者
		default: /* no-op */
		}

		// 加入黑名单相关权限
		if val, exists := ctx.Dice.BanList.Map.Load(ctx.Player.UserID); exists {
			switch val.Rank {
			case BanRankBanned:
				ctx.PrivilegeLevel = -30
			case BanRankTrusted:
				ctx.PrivilegeLevel = 70
			default: /* no-op */
			}
		}

		// master 权限大于黑名单权限
		if ctx.Dice.MasterCheck(ctx.Group.GroupID, ctx.Player.UserID) {
			ctx.PrivilegeLevel = 100
		}
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
		group := s.ServiceAtNew[msg.GroupID]
		if group == nil && msg.GroupID != "" {
			// 注意: 此处必须开启，不然下面mctx.player取不到
			autoOn := true
			if msg.Platform == "QQ-CH" {
				autoOn = d.QQChannelAutoOn
			}
			group = SetBotOnAtGroup(mctx, msg.GroupID)
			group.Active = autoOn
			group.DiceIDExistsMap.Store(ep.UserID, true)
			group.UpdatedAtTime = time.Now().Unix()

			dm := d.Parent
			groupName := dm.TryGetGroupName(group.GroupID)

			txt := fmt.Sprintf("自动激活: 发现无记录群组%s(%s)，因为已是群成员，所以自动激活，开启状态: %t", groupName, group.GroupID, autoOn)
			ep.Adapter.GetGroupInfoAsync(msg.GroupID)
			log.Info(txt)
			mctx.Notice(txt)

			if msg.Platform == "QQ" || msg.Platform == "TG" {
				if mctx.Session.ServiceAtNew[msg.GroupID] != nil {
					for _, i := range mctx.Session.ServiceAtNew[msg.GroupID].ActivatedExtList {
						if i.OnGroupJoined != nil {
							i.callWithJsCheck(mctx.Dice, func() {
								i.OnGroupJoined(mctx, msg)
							})
						}
					}
				}
			}
		}

		var mustLoadUser bool
		if group != nil && group.Active {
			// 开启log时必须加载用户信息
			if group.LogOn {
				mustLoadUser = true
			}
			// 开启reply时，必须加载信息
			// d.CustomReplyConfigEnable
			extReply := group.ExtGetActive("reply")
			if extReply != nil {
				for _, i := range d.CustomReplyConfig {
					if i.Enable {
						mustLoadUser = true
						break
					}
				}
			}

			// 如果非reply扩展存在OnNotCommandReceived功能，那么加载用户数据
			if !mustLoadUser {
				for _, i := range group.ActivatedExtList {
					if i.Name == "reply" {
						// 跳过reply
						continue
					}
					if i.OnNotCommandReceived != nil {
						mustLoadUser = true
						break
					}
				}
			}
		}

		// 当文本可能是在发送命令时，必须加载信息
		maybeCommand := CommandCheckPrefix(msg.Message, d.CommandPrefix, msg.Platform)
		if maybeCommand {
			mustLoadUser = true
		}

		// 私聊时，必须加载信息
		if msg.MessageType == "private" {
			// 这会使得私聊者的发言能触发自定义回复
			mustLoadUser = true
		}

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
				if i.UserID == tmpUID {
					amIBeMentioned = true
					break
				}
			}
			if amIBeMentioned {
				mustLoadUser = true
			}
		}

		if mustLoadUser {
			mctx.Group, mctx.Player = GetPlayerInfoBySender(mctx, msg)
			mctx.IsCurGroupBotOn = msg.MessageType == "group" && mctx.Group.IsActive(mctx)

			if mctx.Group != nil && mctx.Group.System != "" {
				mctx.SystemTemplate = mctx.Group.GetCharTemplate(d)
				// tmpl, _ := d.GameSystemMap.Load(group.System)
				// mctx.SystemTemplate = tmpl
			}
		}

		if group != nil {
			// 自动激活存在状态
			if _, exists := group.DiceIDExistsMap.Load(ep.UserID); !exists {
				group.DiceIDExistsMap.Store(ep.UserID, true)
				group.UpdatedAtTime = time.Now().Unix()
			}
		}

		// 权限号设置
		_ = mctx.fillPrivilege(msg)

		if mctx.Group != nil && mctx.Group.IsActive(mctx) {
			if mctx.PrivilegeLevel != -30 {
				for _, i := range mctx.Group.ActivatedExtList {
					if i.OnMessageReceived != nil {
						i.callWithJsCheck(mctx.Dice, func() {
							i.OnMessageReceived(mctx, msg)
						})
					}
				}
			}
		}

		var cmdLst []string
		if maybeCommand {
			// 兼容模式检查
			// 是的，永远使用兼容模式
			if true || d.CommandCompatibleMode {
				for k := range d.CmdMap {
					cmdLst = append(cmdLst, k)
				}

				// 这里不用group是为了私聊
				g := mctx.Group
				if g != nil {
					for _, i := range g.ActivatedExtList {
						for k := range i.CmdMap {
							cmdLst = append(cmdLst, k)
						}
					}
				}
				sort.Sort(ByLength(cmdLst))
			}
		}

		if notReply := checkBan(mctx, msg); notReply {
			return
		}

		platformPrefix := msg.Platform
		cmdArgs := CommandParse(msg.Message, cmdLst, d.CommandPrefix, platformPrefix, false)
		if cmdArgs != nil {
			mctx.CommandID = getNextCommandID()

			tmpUID := ep.UserID
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
				if d.OnlyLogCommandInGroup {
					// 检查上级选项
					doLog = false
				}
				if doLog {
					// 检查QQ频道的独立选项
					if msg.Platform == "QQ-CH" && (!d.QQChannelLogMessage) {
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
		if mctx.IsCurGroupBotOn && d.EnableCensor && d.CensorMode == AllInput {
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
			} else if !d.OnlyLogCommandInPrivate {
				log.Infof("收到<%s>(%s)的私聊消息: %s", msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
			}
		}

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
				if (msg.MessageType == "private" || mctx.IsCurGroupBotOn) && d.EnableCensor && d.CensorMode == OnlyInputCommand {
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

				var ret bool

				// 试图匹配自定义指令
				if mctx.Group != nil && mctx.Group.IsActive(mctx) {
					for _, i := range mctx.Group.ActivatedExtList {
						if i.OnCommandOverride != nil {
							ret = i.OnCommandOverride(mctx, msg, cmdArgs)
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
					if s.Parent.RateLimitEnabled && msg.Platform == "QQ" {
						if mctx.Group.RateLimiter == nil {
							mctx.Group.RateLimitWarned = false
							if mctx.Dice.GroupReplenishRateStr == "" {
								mctx.Dice.GroupReplenishRateStr = "@every 3s"
								mctx.Dice.GroupReplenishRate = rate.Every(time.Second * 3)
							}
							if mctx.Dice.GroupBurst == 0 {
								mctx.Dice.GroupBurst = 3
							}
							mctx.Group.RateLimiter = rate.NewLimiter(mctx.Dice.GroupReplenishRate, int(mctx.Dice.GroupBurst))
						}
						if mctx.Player.RateLimiter == nil {
							mctx.Player.RateLimitWarned = false
							if mctx.Dice.PersonalReplenishRateStr == "" {
								mctx.Dice.PersonalReplenishRateStr = "@every 3s"
								mctx.Dice.PersonalReplenishRate = rate.Every(time.Second * 3)
							}
							if mctx.Dice.PersonalBurst == 0 {
								mctx.Dice.PersonalBurst = 3
							}
							mctx.Player.RateLimiter = rate.NewLimiter(mctx.Dice.PersonalReplenishRate, int(mctx.Dice.PersonalBurst))
						}

						if mctx.PrivilegeLevel < 100 {
							var handled bool

							if !mctx.Player.RateLimiter.Allow() {
								if mctx.Player.RateLimitWarned {
									mctx.Dice.BanList.AddScoreByCommandSpam(mctx.Player.UserID, msg.GroupID, mctx)
								} else {
									mctx.Player.RateLimitWarned = true
									t := DiceFormatTmpl(mctx, "核心:刷屏_警告内容_个人")
									ReplyToSender(mctx, msg, t)
								}
								handled = true
							} else {
								mctx.Player.RateLimitWarned = false
							}

							if !handled && !mctx.Group.RateLimiter.Allow() {
								if mctx.Group.RateLimitWarned {
									mctx.Dice.BanList.AddScoreByCommandSpam(mctx.Group.GroupID, mctx.Group.GroupID, mctx)
								} else {
									mctx.Group.RateLimitWarned = true
									t := DiceFormatTmpl(mctx, "核心:刷屏_警告内容_群组")
									ReplyToSender(mctx, msg, t)
								}
							} else {
								// Verplitic: 如果已经因为个人刷屏进行了处理，就不追责群组了。这样可以吗？
								mctx.Group.RateLimitWarned = false
							}
						}
					}
					ep.CmdExecutedNum++
					ep.CmdExecutedLastTime = time.Now().Unix()
					mctx.Player.LastCommandTime = ep.CmdExecutedLastTime
					mctx.Player.UpdatedAtTime = time.Now().Unix()
				} else {
					if msg.MessageType == "group" {
						log.Infof("忽略指令(骰子关闭/扩展关闭/未知指令): 来自群(%s)内<%s>(%s): %s", msg.GroupID, msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
					}

					if msg.MessageType == "private" {
						log.Infof("忽略指令(骰子关闭/扩展关闭/未知指令): 来自<%s>(%s)的私聊: %s", msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
					}
				}
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
					for _, _i := range mctx.Group.ActivatedExtList {
						i := _i // 保留引用
						if i.OnNotCommandReceived != nil {
							notCommandReceiveCall := func() {
								if i.IsJsExt {
									waitRun := make(chan int, 1)
									d.JsLoop.RunOnLoop(func(runtime *goja.Runtime) {
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

func (s *IMSession) QuitInactiveGroup(threshold, hint time.Time) {
	platformRE := regexp.MustCompile(`^(.*)-Group:`)

	s.Parent.Logger.Infof("开始清理不活跃群聊. 判定线 %s", threshold.Format(time.RFC3339))

	for _, grp := range s.ServiceAtNew {
		if strings.HasPrefix(grp.GroupID, "PG-") {
			continue
		}
		if grp.RecentDiceSendTime == 0 {
			// 防止骰子从未发言过的新加群被立即清理掉
			continue
		}
		if s.Parent.BanList != nil {
			if info := s.Parent.BanList.GetByID(grp.GroupID); info != nil {
				if info.Rank > BanRankNormal {
					continue // 信任等级高于普通的不清理
				}
			}
		}

		last := time.Unix(grp.RecentDiceSendTime, 0)
		if last.Before(threshold) {
			match := platformRE.FindStringSubmatch(grp.GroupID)
			if len(match) != 2 {
				continue
			}
			hint := fmt.Sprintf("检测到群 %s 上次活动时间为 %s，尝试退出", grp.GroupID, last.Format(time.RFC3339))
			s.Parent.Logger.Info(hint)
			platform := match[1]
			for _, ep := range s.EndPoints {
				if ep.Platform != platform || !grp.DiceIDExistsMap.Exists(ep.UserID) {
					continue
				}
				grp.DiceIDExistsMap.Delete(ep.UserID)
				grp.UpdatedAtTime = time.Now().Unix()

				msgText := DiceFormatTmpl(&MsgContext{Dice: s.Parent}, "核心:骰子自动退群告别语")
				msgCtx := CreateTempCtx(ep, &Message{MessageType: "group", Sender: SenderBase{UserID: ep.UserID}})
				ep.Adapter.SendToGroup(msgCtx, grp.GroupID, msgText, "")

				ep.Adapter.QuitGroup(&MsgContext{Dice: s.Parent}, grp.GroupID)
				(&MsgContext{Dice: s.Parent, EndPoint: ep, Session: s}).Notice(hint)
			}
		} else if last.Before(hint) {
			s.Parent.Logger.Warnf("检测到群 %s 上次活动时间为 %s，将在未来自动退出", grp.GroupID, last.Format(time.RFC3339))
			// TODO: 要不要给通知列表发消息？
			// 不能给当事群发通知，否则会刷last
		}
	}
}

// checkBan 黑名单拦截
func checkBan(ctx *MsgContext, msg *Message) (notReply bool) {
	d := ctx.Dice
	log := d.Logger
	var isBanGroup, isWhiteGroup bool
	// log.Info("check ban ", msg.MessageType, " ", msg.GroupID, " ", ctx.PrivilegeLevel)
	if msg.MessageType == "group" {
		value, exists := d.BanList.Map.Load(msg.GroupID)
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
		groupID := msg.GroupID
		noticeMsg := fmt.Sprintf("检测到群(%s)内黑名单用户<%s>(%s)，自动退群", groupID, msg.Sender.Nickname, msg.Sender.UserID)
		log.Info(noticeMsg)

		text := fmt.Sprintf("因<%s>(%s)是黑名单用户，将自动退群。", msg.Sender.Nickname, msg.Sender.UserID)
		ReplyGroupRaw(ctx, &Message{GroupID: groupID}, text, "")

		ctx.Notice(noticeMsg)

		time.Sleep(1 * time.Second)
		ctx.EndPoint.Adapter.QuitGroup(ctx, groupID)
	}

	if ctx.PrivilegeLevel == -30 {
		if d.BanList.BanBehaviorQuitIfAdmin && msg.MessageType == "group" {
			// 黑名单用户 - 立即退出所在群
			groupID := msg.GroupID
			notReply = true
			if ctx.PrivilegeLevel >= 40 {
				if isWhiteGroup {
					log.Infof("收到群(%s)内邀请者以上权限黑名单用户<%s>(%s)的消息，但在信任群所以不尝试退群", groupID, msg.Sender.Nickname, msg.Sender.UserID)
				} else {
					banQuitGroup()
				}
			} else {
				if isWhiteGroup {
					log.Infof("收到群(%s)内普通群员黑名单用户<%s>(%s)的消息，但在信任群所以不做其他操作", groupID, msg.Sender.Nickname, msg.Sender.UserID)
				} else {
					notReply = true
					noticeMsg := fmt.Sprintf("检测到群(%s)内黑名单用户<%s>(%s)，因是普通群员，进行群内通告", groupID, msg.Sender.Nickname, msg.Sender.UserID)
					log.Info(noticeMsg)

					text := fmt.Sprintf("警告: <%s>(%s)是黑名单用户，将对骰主进行通知。", msg.Sender.Nickname, msg.Sender.UserID)
					ReplyGroupRaw(ctx, &Message{GroupID: groupID}, text, "")

					ctx.Notice(noticeMsg)
				}
			}
		} else if d.BanList.BanBehaviorQuitPlaceImmediately && msg.MessageType == "group" {
			notReply = true
			// 黑名单用户 - 立即退出所在群
			groupID := msg.GroupID
			if isWhiteGroup {
				log.Infof("收到群(%s)内黑名单用户<%s>(%s)的消息，但在信任群所以不尝试退群", groupID, msg.Sender.Nickname, msg.Sender.UserID)
			} else {
				banQuitGroup()
			}
		} else if d.BanList.BanBehaviorRefuseReply {
			notReply = true
			// 黑名单用户 - 拒绝回复
			log.Infof("忽略黑名单用户信息: 来自群(%s)内<%s>(%s): %s", msg.GroupID, msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
		}
	} else if isBanGroup {
		if d.BanList.BanBehaviorQuitPlaceImmediately && !isWhiteGroup {
			notReply = true
			// 黑名单群 - 立即退出
			groupID := msg.GroupID
			if isWhiteGroup {
				log.Infof("群(%s)处于黑名单中，但在信任群所以不尝试退群", groupID)
			} else {
				noticeMsg := fmt.Sprintf("群(%s)处于黑名单中，自动退群", groupID)
				log.Info(noticeMsg)

				ReplyGroupRaw(ctx, &Message{GroupID: groupID}, "因本群处于黑名单中，将自动退群。", "")

				ctx.Notice(noticeMsg)

				time.Sleep(1 * time.Second)
				ctx.EndPoint.Adapter.QuitGroup(ctx, groupID)
			}
		} else if d.BanList.BanBehaviorRefuseReply {
			notReply = true
			// 黑名单群 - 拒绝回复
			log.Infof("忽略黑名单群消息: 来自群(%s)内<%s>(%s): %s", msg.GroupID, msg.Sender.Nickname, msg.Sender.UserID, msg.Message)
		}
	}
	return
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

		if ext != nil && ext.DefaultSetting.DisabledCommand[item.Name] {
			ReplyToSender(ctx, msg, fmt.Sprintf("此指令已被骰主禁用: %s:%s", ext.Name, item.Name))
			return true
		}

		if item.EnableExecuteTimesParse {
			cmdArgs.RevokeExecuteTimesParse()
		}

		if ctx.Player != nil {
			VarSetValueInt64(ctx, "$t轮数", int64(cmdArgs.SpecialExecuteTimes))
		}

		if item.Raw { //nolint:nestif
			if item.CheckCurrentBotOn {
				if !(ctx.IsCurGroupBotOn || ctx.IsPrivate) {
					return false
				}
			}

			if item.CheckMentionOthers {
				if cmdArgs.SomeoneBeMentionedButNotMe {
					return false
				}
			}
		} else {
			// 默认模式行为：需要在当前群/私聊开启，或@自己时生效(需要为第一个@目标)
			if !(ctx.IsCurGroupBotOn || ctx.IsPrivate) {
				return false
			}

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
					}
					cur = index
				}

				if cur != -1 {
					if ctx.Dice.PlayerNameWrapEnable {
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

		var ret CmdExecuteResult
		// 如果是js命令，那么加锁
		if item.IsJsSolveFunc {
			waitRun := make(chan int, 1)
			s.Parent.JsLoop.RunOnLoop(func(vm *goja.Runtime) {
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
	if group.Active || ctx.IsCurGroupBotOn {
		for _, i := range group.ActivatedExtList {
			if i.OnCommandReceived != nil {
				i.callWithJsCheck(ctx.Dice, func() {
					i.OnCommandReceived(ctx, msg, cmdArgs)
				})
			}
		}
	}

	item := ctx.Session.Parent.CmdMap[cmdArgs.Command]
	if tryItemSolve(nil, item) {
		return true
	}

	if group != nil && (group.Active || ctx.IsCurGroupBotOn) {
		for _, i := range group.ActivatedExtList {
			item := i.CmdMap[cmdArgs.Command]
			if tryItemSolve(i, item) {
				return true
			}
		}
	}
	return false
}

func (s *IMSession) OnMessageDeleted(mctx *MsgContext, msg *Message) {
	d := mctx.Dice
	mctx.MessageType = msg.MessageType
	mctx.IsPrivate = mctx.MessageType == "private"
	group := s.ServiceAtNew[msg.GroupID]
	mctx.Group = group
	if group == nil {
		return
	}
	mctx.Group, mctx.Player = GetPlayerInfoBySender(mctx, msg)

	mctx.IsCurGroupBotOn = msg.MessageType == "group" && mctx.Group.IsActive(mctx)
	if mctx.Group != nil && mctx.Group.System != "" {
		mctx.SystemTemplate = mctx.Group.GetCharTemplate(d)
		// tmpl, _ := d.GameSystemMap.Load(group.System)
		// mctx.SystemTemplate = tmpl
	}

	_ = mctx.fillPrivilege(msg)

	for _, i := range s.Parent.ExtList {
		if i.OnMessageDeleted != nil {
			i.callWithJsCheck(mctx.Dice, func() {
				i.OnMessageDeleted(mctx, msg)
			})
		}
	}
}

func (s *IMSession) OnMessageSend(ctx *MsgContext, msg *Message, flag string) {
	for _, i := range s.Parent.ExtList {
		if i.OnMessageSend != nil {
			i.callWithJsCheck(ctx.Dice, func() {
				i.OnMessageSend(ctx, msg, flag)
			})
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
	}
}

func (ep *EndPointInfo) RefreshGroupNum() {
	serveCount := 0
	session := ep.Session
	if session != nil && session.ServiceAtNew != nil {
		for _, i := range session.ServiceAtNew {
			if i.GroupID != "" {
				if strings.HasPrefix(i.GroupID, "PG-") {
					continue
				}
				if i.DiceIDExistsMap.Exists(ep.UserID) {
					serveCount++
					// 在群内的开启数量才被计算，虽然也有被踢出的
					// if i.DiceIdActiveMap.Exists(ep.UserId) {
					// activeCount += 1
					// }
				}
			}
		}
		ep.GroupNum = int64(serveCount)
	}
}

func (d *Dice) NoticeForEveryEndpoint(txt string, allowCrossPlatform bool) {
	_ = allowCrossPlatform
	// 通知种类之一：每个noticeId  *  每个平台匹配的ep：存活
	// TODO: 先复制几次实现，后面重构
	foo := func() {
		defer func() {
			if r := recover(); r != nil {
				d.Logger.Errorf("发送通知异常: %v 堆栈: %v", r, string(debug.Stack()))
			}
		}()

		if d.MailEnable {
			_ = d.SendMail(txt, MailTypeNotice)
			return
		}

		for _, ep := range d.ImSession.EndPoints {
			for _, i := range d.NoticeIDs {
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

		if ctx.Dice.MailEnable {
			_ = ctx.Dice.SendMail(txt, MailTypeNotice)
			return
		}

		sent := false

		for _, i := range ctx.Dice.NoticeIDs {
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

		if ctx.Dice.MailEnable {
			_ = ctx.Dice.SendMail(txt, MailTypeNotice)
			return
		}

		sent := false
		if ctx.EndPoint.Enable {
			for _, i := range ctx.Dice.NoticeIDs {
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
			ctx.Dice.Logger.Errorf("未能发送来自%s的通知：%s", ctx.EndPoint.Platform, txt)
		}
	}
	go foo()
}

// ChVarsGet 获取当前的角色变量
func (ctx *MsgContext) ChVarsGet() (lockfree.HashMap, bool) {
	// gvar := ctx.LoadPlayerGlobalVars()
	_card, exists := ctx.Player.Vars.ValueMap.Get("$:card")
	if exists {
		card, ok := _card.(lockfree.HashMap)
		if ok {
			// 绑卡
			// card.Iterate(func(_k interface{}, _v interface{}) error {
			//	fmt.Println("????", _k, _v)
			//	return nil
			// })
			return card, true
		}
	}
	// 不绑卡
	return ctx.Player.Vars.ValueMap, false
}

func (ctx *MsgContext) ChVarsUpdateTime() {
	_card, exists := ctx.Player.Vars.ValueMap.Get("$:card")
	if exists {
		// 绑卡情况
		if card, ok := _card.(lockfree.HashMap); ok {
			if _v, ok := card.Get("$:cardName"); ok {
				if v, ok := _v.(*VMValue); ok {
					name, _ := v.ReadString()

					if name != "" {
						vars := ctx.LoadPlayerGlobalVars()
						key := fmt.Sprintf("$:ch-bind-mtime:%s", name)
						vars.ValueMap.Set(key, time.Now().Unix())
						vars.LastWriteTime = time.Now().Unix()
					}
				}
			}
		}
	} else {
		// 不绑卡情况
		ctx.Player.Vars.LastWriteTime = time.Now().Unix()
	}
}

func (ctx *MsgContext) ChVarsClear() int {
	vars, isBind := ctx.ChVarsGet()
	num := vars.Len()
	if isBind {
		// gvar := ctx.LoadPlayerGlobalVars()
		num = 0
		if _card, ok := ctx.Player.Vars.ValueMap.Get("$:card"); ok {
			// 因为card可能在多个群关联，所以只有通过这种方式清空
			if card, ok := _card.(lockfree.HashMap); ok {
				items := []interface{}{}
				_ = card.Iterate(func(_k interface{}, _v interface{}) error {
					if _k == "$:cardName" {
						return nil
					}
					if _k == "$cardType" {
						return nil
					}
					items = append(items, _k)
					return nil
				})

				num = len(items)
				for _, i := range items {
					card.Del(i)
				}
			}
		}

		ctx.ChVarsUpdateTime()
		// gvar.LastWriteTime = time.Now().Unix()
	} else {
		p := ctx.Player
		p.Vars.ValueMap = lockfree.NewHashMap()
		p.Vars.LastWriteTime = time.Now().Unix()
		ctx.ChVarsUpdateTime()
	}
	return num
}

func (ctx *MsgContext) ChVarsNumGet() int {
	vars, _ := ctx.ChVarsGet()
	num := vars.Len()
	return num
}

func (ctx *MsgContext) ChExists(name string) bool {
	vars := ctx.LoadPlayerGlobalVars()
	varName := "$ch:" + name

	if _, exists := vars.ValueMap.Get(varName); exists {
		return true
	}
	return false
}

func (ctx *MsgContext) ChGet(name string) lockfree.HashMap {
	vars := ctx.LoadPlayerGlobalVars()
	varName := "$ch:" + name

	if _data, exists := vars.ValueMap.Get(varName); exists {
		data := _data.(*VMValue)
		mapData := make(map[string]*VMValue)
		err := JSONValueMapUnmarshal([]byte(data.Value.(string)), &mapData)

		if err == nil {
			m := lockfree.NewHashMap()
			for k, v := range mapData {
				m.Set(k, v)
			}
			return m
		}
	}
	return nil
}

// ChLoad 加载角色，成功返回角色表，失败返回nil
func (ctx *MsgContext) ChLoad(name string) lockfree.HashMap {
	m := ctx.ChGet(name)
	if m != nil {
		ctx.Player.Name = name
		ctx.Player.Vars.ValueMap = m
		ctx.Player.Vars.LastWriteTime = time.Now().Unix()
		ctx.Player.UpdatedAtTime = time.Now().Unix()
		return m
	}
	return nil
}

// ChNew 新建角色
func (ctx *MsgContext) ChNew(name string) bool {
	vars := ctx.LoadPlayerGlobalVars()
	varName := "$ch:" + name

	if _, exists := vars.ValueMap.Get(varName); exists {
		return false
	}

	vars.ValueMap.Set(varName, &VMValue{
		TypeID: VMTypeString,
		Value:  "{}",
	})

	vars.LastWriteTime = time.Now().Unix()
	return true
}

func (ctx *MsgContext) ChBindCur(name string) bool {
	// 绑卡过程:
	// 全局变量 $:group-bind:群号  = 卡片名 // 至少需要保留一个，用于序列化，VMValue
	// 全局变量 $:ch-bind-data:角色  = 卡片数据 // 不序列化
	// 全局变量 $:ch-bind-mtime:角色 = 时间 // 卡片被修改时，不序列化
	// 个人群内 $:card = 卡片数据 // 不序列化
	// 个人群内 $:cardBindMark = 1 // 标记
	// 卡片数据中存放卡片名称，不序列化的部分，在加载个人全局变量时临时生成
	vars := ctx.LoadPlayerGlobalVars()
	key2 := fmt.Sprintf("$:ch-bind-data:%s", name)

	// 如果已经绑定过，继续用
	var m lockfree.HashMap
	_m, exists := vars.ValueMap.Get(key2)
	if exists {
		m, _ = _m.(lockfree.HashMap)
	} else {
		// 如果不存在，整一份新的
		m = ctx.ChGet(name)
	}

	if m != nil {
		m.Set("$:cardName", &VMValue{TypeID: VMTypeString, Value: name}) // 防止出事，覆盖一次
		vars.ValueMap.Set(key2, m)                                       // 同上，$:ch-bind-data:角色 = 数据

		// $:group-bind:群号  = 卡片名
		key := fmt.Sprintf("$:group-bind:%s", ctx.Group.GroupID)
		vars.ValueMap.Set(key, &VMValue{TypeID: VMTypeString, Value: name})
		// fmt.Println("$$$$$$$$$$$$$$", key)
		vars.LastWriteTime = time.Now().Unix()

		// $:card = 卡片数据
		ctx.Player.Vars.ValueMap.Set("$:card", m)
		ctx.Player.Vars.ValueMap.Set("$:cardBindMark", &VMValue{TypeID: VMTypeInt64, Value: 1})
		ctx.Player.Vars.LastWriteTime = time.Now().Unix()
		ctx.Player.Name = name
		ctx.Player.UpdatedAtTime = time.Now().Unix()
		return true
	}
	return false
}

func (ctx *MsgContext) ChUnbindCur() (string, bool) {
	if _, exists := ctx.Player.Vars.ValueMap.Get("$:card"); exists {
		name := ctx.ChBindCurGet()
		vars := ctx.LoadPlayerGlobalVars()
		key := fmt.Sprintf("$:group-bind:%s", ctx.Group.GroupID)
		vars.ValueMap.Del(key)

		ctx.Player.Vars.ValueMap.Del("$:card")
		ctx.Player.Vars.ValueMap.Del("$:cardBindMark")
		ctx.Player.Vars.LastWriteTime = time.Now().Unix()

		lst := ctx.ChBindGetList(name)

		if len(lst) == 0 {
			// 没有群绑这个卡了，释放内存
			vars := ctx.LoadPlayerGlobalVars()
			key2 := fmt.Sprintf("$:ch-bind-data:%s", name)
			vars.ValueMap.Del(key2)
			vars.LastWriteTime = time.Now().Unix()
		}

		return name, true
	}
	return "", false
}

// ChBindCurGet 获取当前群绑定角色
func (ctx *MsgContext) ChBindCurGet() string {
	if _card, exists := ctx.Player.Vars.ValueMap.Get("$:card"); exists {
		if card, ok := _card.(lockfree.HashMap); ok {
			if _v, ok := card.Get("$:cardName"); ok {
				if v, ok := _v.(*VMValue); ok {
					name, _ := v.ReadString()
					return name
				}
			}
		}
	}
	return ""
}

// ChBindGet 获取一个正在绑定状态的卡，可用于该卡片是否绑卡检测
func (ctx *MsgContext) ChBindGet(name string) lockfree.HashMap {
	vars := ctx.LoadPlayerGlobalVars()
	key2 := fmt.Sprintf("$:ch-bind-data:%s", name)

	var m lockfree.HashMap
	_m, exists := vars.ValueMap.Get(key2)
	if exists {
		m, _ = _m.(lockfree.HashMap)
		if m != nil {
			return m
		}
	}

	return nil
}

// ChUnbind 解除某个角色的绑定
func (ctx *MsgContext) ChUnbind(name string) []string {
	lst := ctx.ChBindGetList(name)

	for _, groupID := range lst {
		g := ctx.Session.ServiceAtNew[groupID]
		if g != nil {
			p := g.PlayerGet(ctx.Dice.DBData, ctx.Player.UserID)
			if p.Vars == nil || !p.Vars.Loaded {
				LoadPlayerGroupVars(ctx.Dice, g, p)
			}
			p.Vars.ValueMap.Del("$:card")
			p.Vars.ValueMap.Del("$:cardBindMark")
			p.Vars.LastWriteTime = time.Now().Unix()
		}
	}

	if len(lst) > 0 {
		// 没有群绑这个卡了，释放内存
		vars := ctx.LoadPlayerGlobalVars()
		key2 := fmt.Sprintf("$:ch-bind-data:%s", name)
		vars.ValueMap.Del(key2)
		for _, i := range lst {
			// 删除绑定标记
			vars.ValueMap.Del(fmt.Sprintf("$:group-bind:%s", i))
		}
		vars.LastWriteTime = time.Now().Unix()
	}

	return lst
}

func (ctx *MsgContext) ChBindGetList(name string) []string {
	vars := ctx.LoadPlayerGlobalVars()
	groups := map[string]bool{}
	_ = vars.ValueMap.Iterate(func(_k interface{}, _v interface{}) error {
		k := _k.(string)
		if v, ok := _v.(*VMValue); ok {
			if strings.HasPrefix(k, "$:group-bind:") {
				if val, _ := v.ReadString(); val == name {
					groups[k[len("$:group-bind:"):]] = true
				}
			}
		}
		return nil
	})
	var grps []string
	for k := range groups {
		grps = append(grps, k)
	}
	return grps
}

var curCommandID int64 = 0

func getNextCommandID() int64 {
	curCommandID++
	return curCommandID
}
