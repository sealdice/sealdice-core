package dice

import (
	"fmt"
	"github.com/dop251/goja"
	"github.com/fy0/lockfree"
	"github.com/jmoiron/sqlx"
	"gopkg.in/yaml.v3"
	"runtime/debug"
	"sealdice-core/dice/model"
	"sort"
	"strings"
	"time"
)

type SenderBase struct {
	Nickname  string `json:"nickname" jsbind:"nickname"`
	UserId    string `json:"userId" jsbind:"userId"`
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
	GroupId     string      `json:"groupId" jsbind:"groupId"`         // 群号，如果是群聊消息
	GuildId     string      `json:"guildId" jsbind:"guildId"`         // 服务器群组号，会在discord,kook,dodo等平台见到
	Sender      SenderBase  `json:"sender" jsbind:"sender"`           // 发送者
	Message     string      `json:"message" jsbind:"message"`         // 消息内容
	RawId       interface{} `json:"rawId" jsbind:"rawId"`             // 原始信息ID，用于处理撤回等
	Platform    string      `json:"platform" jsbind:"platform"`       // 当前平台
	TmpUid      string      `json:"-" yaml:"-"`
}

// GroupPlayerInfoBase 群内玩家信息
type GroupPlayerInfoBase struct {
	Name                string `yaml:"name" jsbind:"name"` // 玩家昵称
	UserId              string `yaml:"userId" jsbind:"userId"`
	InGroup             bool   `yaml:"inGroup"`                                          // 是否在群内，有时一个人走了，信息还暂时残留
	LastCommandTime     int64  `yaml:"lastCommandTime" jsbind:"lastCommandTime"`         // 上次发送指令时间
	AutoSetNameTemplate string `yaml:"autoSetNameTemplate" jsbind:"autoSetNameTemplate"` // 名片模板

	// level int 权限
	DiceSideNum  int                  `yaml:"diceSideNum"` // 面数，为0时等同于d100
	Vars         *PlayerVariablesItem `yaml:"-"`           // 玩家的群内变量
	ValueMapTemp lockfree.HashMap     `yaml:"-"`           // 玩家的群内临时变量
	//ValueMapTemp map[string]*VMValue  `yaml:"-"`           // 玩家的群内临时变量

	TempValueAlias *map[string][]string `yaml:"-"` // 群内临时变量别名 - 其实这个有点怪的，为什么在这里？

	UpdatedAtTime int64 `yaml:"-" json:"-"`
}

// GroupPlayerInfo 这是一个YamlWrapper，没有实际作用
// 原因见 https://github.com/go-yaml/yaml/issues/712
//type GroupPlayerInfo struct {
//	GroupPlayerInfoBase `yaml:",inline,flow"`
//}

type GroupPlayerInfo model.GroupPlayerInfoBase

type GroupInfo struct {
	Active           bool                               `json:"active" yaml:"active" jsbind:"active"`          // 是否在群内开启 - 过渡为象征意义
	ActivatedExtList []*ExtInfo                         `yaml:"activatedExtList,flow" json:"activatedExtList"` // 当前群开启的扩展列表
	Players          *SyncMap[string, *GroupPlayerInfo] `yaml:"-" json:"-"`                                    // 群员角色数据

	GroupId         string                 `yaml:"groupId" json:"groupId" jsbind:"groupId"`
	GuildId         string                 `yaml:"guildId" json:"guildId" jsbind:"guildId"`
	GroupName       string                 `yaml:"groupName" json:"groupName" jsbind:"groupName"`
	DiceIdActiveMap *SyncMap[string, bool] `yaml:"diceIds,flow" json:"diceIdActiveMap"` // 对应的骰子ID(格式 平台:ID)，对应单骰多号情况，例如骰A B都加了群Z，A退群不会影响B在群内服务
	DiceIdExistsMap *SyncMap[string, bool] `yaml:"-" json:"diceIdExistsMap"`            // 对应的骰子ID(格式 平台:ID)是否存在于群内
	BotList         *SyncMap[string, bool] `yaml:"botList,flow" json:"botList"`         // 其他骰子列表
	DiceSideNum     int64                  `yaml:"diceSideNum" json:"diceSideNum"`      // 以后可能会支持 1d4 这种默认面数，暂不开放给js
	System          string                 `yaml:"system" json:"system"`                // 规则系统，概念同bcdice的gamesystem，距离如dnd5e coc7

	//ValueMap     map[string]*VMValue `yaml:"-"`
	ValueMap     lockfree.HashMap `yaml:"-" json:"-"`
	HelpPackages []string         `yaml:"-" json:"helpPackages"`
	CocRuleIndex int              `yaml:"cocRuleIndex" json:"cocRuleIndex" jsbind:"cocRuleIndex"`
	LogCurName   string           `yaml:"logCurFile" json:"logCurName" jsbind:"logCurName"`
	LogOn        bool             `yaml:"logOn" json:"logOn" jsbind:"logOn"`

	QuitMarkAutoClean   bool   `yaml:"-" json:"-"` // 自动清群 - 播报，即将自动退出群组
	QuitMarkMaster      bool   `yaml:"-" json:"-"` // 骰主命令退群 - 播报，即将自动退出群组
	RecentDiceSendTime  int64  `json:"recentDiceSendTime"`
	ShowGroupWelcome    bool   `yaml:"showGroupWelcome" json:"showGroupWelcome" jsbind:"showGroupWelcome"` // 是否迎新
	GroupWelcomeMessage string `yaml:"groupWelcomeMessage" json:"groupWelcomeMessage" jsbind:"groupWelcomeMessage"`
	//FirstSpeechMade     bool   `yaml:"firstSpeechMade"` // 是否做过进群发言
	LastCustomReplyTime float64 `yaml:"-" json:"-"` // 上次自定义回复时间

	EnteredTime  int64  `yaml:"enteredTime" json:"enteredTime" jsbind:"enteredTime"`    // 入群时间
	InviteUserId string `yaml:"inviteUserId" json:"inviteUserId" jsbind:"inviteUserId"` // 邀请人
	// 仅用于http接口
	TmpPlayerNum int64    `yaml:"-" json:"tmpPlayerNum"`
	TmpExtList   []string `yaml:"-" json:"tmpExtList"`

	UpdatedAtTime int64 `yaml:"-" json:"-"`
}

// ExtActive 开启扩展
func (group *GroupInfo) ExtActive(ei *ExtInfo) {
	lst := []*ExtInfo{ei}
	oldLst := group.ActivatedExtList
	group.ActivatedExtList = append(lst, oldLst...)
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
	firstCheck := group.Active && group.DiceIdActiveMap.Len() >= 1
	if firstCheck {
		v, _ := group.DiceIdActiveMap.Load(ctx.EndPoint.UserId)
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
		p = (*GroupPlayerInfo)(model.GroupPlayerInfoGet(db, group.GroupId, id))
		if p != nil {
			group.Players.Store(id, p)
		}
	}
	return (*GroupPlayerInfo)(p)
}

func (group *GroupInfo) GetCharTemplate(dice *Dice) *GameSystemTemplate {
	// 有system优先system
	if group.System != "" {
		v, _ := dice.GameSystemMap.Load(group.System)
		return v
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
	return nil
}

type EndPointInfoBase struct {
	Id                  string `yaml:"id" json:"id" jsbind:"id"` // uuid
	Nickname            string `yaml:"nickname" json:"nickname" jsbind:"nickname"`
	State               int    `yaml:"state" json:"state" jsbind:"state"` // 状态 0 断开 1已连接 2连接中 3连接失败
	UserId              string `yaml:"userId" json:"userId" jsbind:"userId"`
	GroupNum            int64  `yaml:"groupNum" json:"groupNum" jsbind:"groupNum"`                                  // 拥有群数
	CmdExecutedNum      int64  `yaml:"cmdExecutedNum" json:"cmdExecutedNum" jsbind:"cmdExecutedNum"`                // 指令执行次数
	CmdExecutedLastTime int64  `yaml:"cmdExecutedLastTime" json:"cmdExecutedLastTime" jsbind:"cmdExecutedLastTime"` // 指令执行次数
	OnlineTotalTime     int64  `yaml:"onlineTotalTime" json:"onlineTotalTime" jsbind:"onlineTotalTime"`             // 在线时长

	Platform     string `yaml:"platform" json:"platform" jsbind:"platform"` // 平台，如QQ等
	RelWorkDir   string `yaml:"relWorkDir" json:"relWorkDir"`               // 工作目录
	Enable       bool   `yaml:"enable" json:"enable" jsbind:"enable"`       // 是否启用
	ProtocolType string `yaml:"protocolType"`                               // 协议类型，如onebot、koishi等

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
		var val struct {
			Adapter *PlatformAdapterQQOnebot `yaml:"adapter"`
		}

		err := value.Decode(&val)
		if err != nil {
			return err
		}

		ep.Adapter = val.Adapter
	case "DISCORD":
		var val struct {
			Adapter *PlatformAdapterDiscord `yaml:"adapter"`
		}
		err := value.Decode(&val)
		if err != nil {
			return err
		}
		ep.Adapter = val.Adapter
	case "KOOK":
		var val struct {
			Adapter *PlatformAdapterKook `yaml:"adapter"`
		}
		err := value.Decode(&val)
		if err != nil {
			return err
		}
		ep.Adapter = val.Adapter
	case "TG":
		var val struct {
			Adapter *PlatformAdapterTelegram `yaml:"adapter"`
		}
		err := value.Decode(&val)
		if err != nil {
			return err
		}
		ep.Adapter = val.Adapter
	case "MC":
		var val struct {
			Adapter *PlatformAdapterMinecraft `yaml:"adapter"`
		}
		err := value.Decode(&val)
		if err != nil {
			return err
		}
		ep.Adapter = val.Adapter
	case "DODO":
		var val struct {
			Adapter *PlatformAdapterDodo `yaml:"adapter"`
		}
		err := value.Decode(&val)
		if err != nil {
			return err
		}
		ep.Adapter = val.Adapter
	}
	return err
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
	CommandId       int64       // 指令ID
	CommandHideFlag string      // 暗骰标记
	CommandInfo     interface{} // 命令信息
	PrivilegeLevel  int         `jsbind:"privilegeLevel"` // 权限等级 40邀请者 50管理 60群主 70信任 100master
	DelegateText    string      `jsbind:"delegateText"`   // 代骰附加文本

	deckDepth      int                                         // 抽牌递归深度
	DeckPools      map[*DeckInfo]map[string]*ShuffleRandomPool // 不放回抽取的缓存
	SystemTemplate *GameSystemTemplate
}

//func (s *IMSession) GroupEnableCheck(ep *EndPointInfo, msg *Message, runInSync bool) {
//}

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
	if msg.MessageType == "group" || msg.MessageType == "private" {
		// GroupEnableCheck TODO: 后续看看是否需要
		group := s.ServiceAtNew[msg.GroupId]
		if group == nil && msg.GroupId != "" {
			// 注意: 此处必须开启，不然下面mctx.player取不到
			autoOn := true
			if msg.Platform == "QQ-CH" {
				autoOn = d.QQChannelAutoOn
			}
			group = SetBotOnAtGroup(mctx, msg.GroupId)
			group.Active = autoOn
			group.DiceIdExistsMap.Store(ep.UserId, true)
			group.UpdatedAtTime = time.Now().Unix()

			dm := d.Parent
			groupName := dm.TryGetGroupName(group.GroupId)

			txt := fmt.Sprintf("自动激活: 发现无记录群组%s(%s)，因为已是群成员，所以自动激活，开启状态: %t", groupName, group.GroupId, autoOn)
			ep.Adapter.GetGroupInfoAsync(msg.GroupId)
			log.Info(txt)
			mctx.Notice(txt)
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
			tmpUid := ep.UserId
			if msg.TmpUid != "" {
				tmpUid = msg.TmpUid
			}
			for _, i := range ats {
				if i.UserId == tmpUid {
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
				//tmpl, _ := d.GameSystemMap.Load(group.System)
				//mctx.SystemTemplate = tmpl
			}
		}

		if group != nil {
			// 自动激活存在状态
			if _, exists := group.DiceIdExistsMap.Load(ep.UserId); !exists {
				group.DiceIdExistsMap.Store(ep.UserId, true)
				group.UpdatedAtTime = time.Now().Unix()
			}
		}

		// 权限号设置
		if mctx.Group != nil {
			if mctx.Group != nil {
				if msg.Sender.UserId == mctx.Group.InviteUserId {
					mctx.PrivilegeLevel = 40 // 邀请者
				}
			}

			if msg.Sender.GroupRole == "admin" {
				mctx.PrivilegeLevel = 50 // 群管理
			}
			if msg.Sender.GroupRole == "owner" {
				mctx.PrivilegeLevel = 60 // 群主
			}

			// 加入黑名单相关权限
			if val, exists := d.BanList.Map.Load(mctx.Player.UserId); exists {
				if val.Rank == BanRankBanned {
					mctx.PrivilegeLevel = -30
				}
				if val.Rank == BanRankTrusted {
					mctx.PrivilegeLevel = 70
				}
			}

			// master 权限大于黑名单权限
			if d.MasterCheck(mctx.Player.UserId) {
				mctx.PrivilegeLevel = 100
			}
		}

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

		PlatformPrefix := msg.Platform
		cmdArgs := CommandParse(msg.Message, d.CommandCompatibleMode, cmdLst, d.CommandPrefix, PlatformPrefix)
		if cmdArgs != nil {
			mctx.CommandId = getNextCommandId()

			// 设置AmIBeMentioned
			cmdArgs.AmIBeMentioned = false
			cmdArgs.AmIBeMentionedFirst = false
			tmpUid := ep.UserId
			if msg.TmpUid != "" {
				tmpUid = msg.TmpUid
			}
			for _, i := range cmdArgs.At {
				if i.UserId == tmpUid {
					cmdArgs.AmIBeMentioned = true
					break
				}
			}
			if cmdArgs.AmIBeMentioned {
				// 检查是不是第一个被AT的
				if cmdArgs.At[0].UserId == tmpUid {
					cmdArgs.AmIBeMentionedFirst = true
				}
			}
		}

		// 收到群 test(1111) 内 XX(222) 的消息: 好看 (1232611291)
		if msg.MessageType == "group" {
			if mctx.CommandId != 0 {
				// 关闭状态下，如果被@，且是第一个被@的，那么视为开启
				if !mctx.IsCurGroupBotOn && cmdArgs.AmIBeMentionedFirst {
					mctx.IsCurGroupBotOn = true
				}

				log.Infof("收到群(%s)内<%s>(%s)的指令: %s", msg.GroupId, msg.Sender.Nickname, msg.Sender.UserId, msg.Message)
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
					log.Infof("收到群(%s)内<%s>(%s)的消息: %s", msg.GroupId, msg.Sender.Nickname, msg.Sender.UserId, msg.Message)
					//fmt.Printf("消息长度 %v 内容 %v \n", len(msg.Message), []byte(msg.Message))
				}
			}
		}

		if msg.MessageType == "private" {
			if mctx.CommandId != 0 {
				log.Infof("收到<%s>(%s)的私聊指令: %s", msg.Sender.Nickname, msg.Sender.UserId, msg.Message)
			} else {
				if !d.OnlyLogCommandInPrivate {
					log.Infof("收到<%s>(%s)的私聊消息: %s", msg.Sender.Nickname, msg.Sender.UserId, msg.Message)
				}
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

				// 有人被@了，但不是我
				// 后面的代码保证了如果@的名单中有任何已知骰子，不会进入下一步操作
				// 所以不用考虑其他骰子被@的情况
				cmdArgs.SomeoneBeMentionedButNotMe = len(cmdArgs.At) > 0 && (!cmdArgs.AmIBeMentioned)
				//if cmdArgs.MentionedOtherDice {
				//	// @其他骰子
				//	return
				//}

				if mctx.PrivilegeLevel == -30 {
					// 黑名单用户 - 拒绝回复
					if d.BanList.BanBehaviorRefuseReply {
						log.Infof("忽略黑名单用户指令: 来自群(%s)内<%s>(%s): %s", msg.GroupId, msg.Sender.Nickname, msg.Sender.UserId, msg.Message)
						return
					}
				}

				if cmdArgs.Command != "botlist" && !cmdArgs.AmIBeMentioned {
					myuid := ep.UserId
					// 屏蔽机器人发送的消息
					if mctx.MessageType == "group" {
						//fmt.Println("YYYYYYYYY", myuid, mctx.Group != nil)
						if mctx.Group.BotList.Exists(msg.Sender.UserId) {
							log.Infof("忽略指令(机器人): 来自群(%s)内<%s>(%s): %s", msg.GroupId, msg.Sender.Nickname, msg.Sender.UserId, msg.Message)
							return
						}
						// 当其他机器人被@，不回应
						for _, i := range cmdArgs.At {
							uid := i.UserId
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
					ep.CmdExecutedNum += 1
					ep.CmdExecutedLastTime = time.Now().Unix()
					mctx.Player.LastCommandTime = ep.CmdExecutedLastTime
					mctx.Player.UpdatedAtTime = time.Now().Unix()
				} else {
					if msg.MessageType == "group" {
						log.Infof("忽略指令(骰子关闭/扩展关闭/未知指令): 来自群(%s)内<%s>(%s): %s", msg.GroupId, msg.Sender.Nickname, msg.Sender.UserId, msg.Message)
					}

					if msg.MessageType == "private" {
						log.Infof("忽略指令(骰子关闭/扩展关闭/未知指令): 来自<%s>(%s)的私聊: %s", msg.Sender.Nickname, msg.Sender.UserId, msg.Message)
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
				if mctx.Group != nil && mctx.Group.BotList.Exists(msg.Sender.UserId) {
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

			//text := fmt.Sprintf("信息 来自群%d - %s(%d)：%s", msg.GroupId, msg.Sender.Nickname, msg.Sender.UserId, msg.Message);
			//replyGroup(Socket, 22, text)
		}
	}
}

func (s *IMSession) commandSolve(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) bool {
	// 设置临时变量
	if ctx.Player != nil {
		SetTempVars(ctx, msg.Sender.Nickname)
		VarSetValueStr(ctx, "$tMsgID", fmt.Sprintf("%v", msg.RawId))
		VarSetValueInt64(ctx, "$t轮数", int64(cmdArgs.SpecialExecuteTimes))
	}

	tryItemSolve := func(ext *ExtInfo, item *CmdItemInfo) bool {
		if item != nil {
			if ext != nil && ext.DefaultSetting.DisabledCommand[item.Name] {
				ReplyToSender(ctx, msg, fmt.Sprintf("此指令已被骰主禁用: %s:%s", ext.Name, item.Name))
				return true
			}

			if item.Raw {
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
					ctx.DelegateText = fmt.Sprintf("由<%s>代骰:\n", ctx.Player.Name)
				} else {
					// 如果其他人被@了就不管
					// 注: 如果被@的对象在botlist列表，那么不会走到这一步
					if cmdArgs.SomeoneBeMentionedButNotMe {
						return false
					}
				}
			}

			var ret CmdExecuteResult
			// 如果是js命令，那么加锁
			if item.IsJsSolveFunc {
				waitRun := make(chan int, 1)
				s.Parent.JsLoop.RunOnLoop(func(vm *goja.Runtime) {
					defer func() {
						if r := recover(); r != nil {
							//log.Errorf("异常: %v 堆栈: %v", r, string(debug.Stack()))
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

				//// 进行指令统计
				//vPlayer := ctx.LoadPlayerGlobalVars()
				//key := "#" + item.Name
				//_v, ok := vPlayer.ValueMap.Get(key)
				//if ok {
				//	v, ok := _v.(int64)
				//	if ok {
				//		vPlayer.ValueMap.Set(key, v+1)
				//	}
				//}

				return true
			}
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
	//}
	return false
}

func (s *IMSession) OnMessageSend(ctx *MsgContext, messageType string, groupId string, text string, flag string) {
	if s.ServiceAtNew[groupId] != nil {
		for _, i := range s.Parent.ExtList {
			if i.OnMessageSend != nil {
				i.callWithJsCheck(ctx.Dice, func() {
					i.OnMessageSend(ctx, messageType, groupId, text, flag)
				})
			}
		}
	}
}

// SetEnable
/* 如果已连接，将断开连接，如果开着GCQ将自动结束。如果启用的话，则反过来  */
func (ep *EndPointInfo) SetEnable(d *Dice, enable bool) {
	if ep.Enable != enable {
		ep.Adapter.SetEnable(enable)
	}
}

func (ep *EndPointInfo) AdapterSetup() {
	switch ep.Platform {
	case "QQ":
		pa := ep.Adapter.(*PlatformAdapterQQOnebot)
		pa.Session = ep.Session
		pa.EndPoint = ep
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
	}
}

func (ep *EndPointInfo) RefreshGroupNum() {
	serveCount := 0
	session := ep.Session
	if session != nil && session.ServiceAtNew != nil {
		for _, i := range session.ServiceAtNew {
			if i.GroupId != "" {
				if strings.HasPrefix(i.GroupId, "PG-") {
					continue
				}
				if i.DiceIdExistsMap.Exists(ep.UserId) {
					serveCount += 1
					// 在群内的开启数量才被计算，虽然也有被踢出的
					//if i.DiceIdActiveMap.Exists(ep.UserId) {
					//activeCount += 1
					//}
				}
			}
		}
		ep.GroupNum = int64(serveCount)
	}
}

func (ctx *MsgContext) Notice(txt string) {
	foo := func() {
		defer func() {
			if r := recover(); r != nil {
				ctx.Dice.Logger.Errorf("发送通知异常: %v 堆栈: %v", r, string(debug.Stack()))
			}
		}()

		for _, i := range ctx.Dice.NoticeIds {
			n := strings.Split(i, ":")
			if len(n) >= 2 {
				if strings.HasSuffix(n[0], "-Group") {
					ReplyGroup(ctx, &Message{GroupId: i}, txt)
				} else {
					ReplyPerson(ctx, &Message{Sender: SenderBase{UserId: i}}, txt)
				}
			}
			time.Sleep(1 * time.Second)
		}
	}
	go foo()
}

// ChVarsGet 获取当前的角色变量
func (ctx *MsgContext) ChVarsGet() (lockfree.HashMap, bool) {
	//gvar := ctx.LoadPlayerGlobalVars()
	_card, exists := ctx.Player.Vars.ValueMap.Get("$:card")
	if exists {
		card, ok := _card.(lockfree.HashMap)
		if ok {
			// 绑卡
			//card.Iterate(func(_k interface{}, _v interface{}) error {
			//	fmt.Println("????", _k, _v)
			//	return nil
			//})
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
		//gvar := ctx.LoadPlayerGlobalVars()
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
		//gvar.LastWriteTime = time.Now().Unix()
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
		err := JsonValueMapUnmarshal([]byte(data.Value.(string)), &mapData)

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
		TypeId: VMTypeString,
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
		m.Set("$:cardName", &VMValue{TypeId: VMTypeString, Value: name}) // 防止出事，覆盖一次
		vars.ValueMap.Set(key2, m)                                       // 同上，$:ch-bind-data:角色 = 数据

		// $:group-bind:群号  = 卡片名
		key := fmt.Sprintf("$:group-bind:%s", ctx.Group.GroupId)
		vars.ValueMap.Set(key, &VMValue{TypeId: VMTypeString, Value: name})
		//fmt.Println("$$$$$$$$$$$$$$", key)
		vars.LastWriteTime = time.Now().Unix()

		// $:card = 卡片数据
		ctx.Player.Vars.ValueMap.Set("$:card", m)
		ctx.Player.Vars.ValueMap.Set("$:cardBindMark", &VMValue{TypeId: VMTypeInt64, Value: 1})
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
		key := fmt.Sprintf("$:group-bind:%s", ctx.Group.GroupId)
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

	for _, groupId := range lst {
		g := ctx.Session.ServiceAtNew[groupId]
		p := g.PlayerGet(ctx.Dice.DBData, ctx.Player.UserId)
		if p.Vars == nil || !p.Vars.Loaded {
			LoadPlayerGroupVars(ctx.Dice, g, p)
		}
		p.Vars.ValueMap.Del("$:card")
		p.Vars.ValueMap.Del("$:cardBindMark")
		p.Vars.LastWriteTime = time.Now().Unix()
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
	grps := []string{}
	for k, _ := range groups {
		grps = append(grps, k)
	}
	return grps
}

var curCommandId int64 = 0

func getNextCommandId() int64 {
	curCommandId += 1
	return curCommandId
}
