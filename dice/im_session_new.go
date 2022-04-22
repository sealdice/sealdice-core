package dice

import (
	"fmt"
	"github.com/fy0/lockfree"
	"gopkg.in/yaml.v3"
	"runtime/debug"
	"sort"
	"strings"
	"time"
)

type SenderBase struct {
	Nickname  string `json:"nickname"`
	UserId    string `json:"userId"`
	GroupRole string `json:"-"` // 群内角色 admin管理员 owner群主
}

// Message 消息的重要信息
// 时间
// 发送地点(群聊/私聊)
// 人物(是谁发的)
// 内容
type Message struct {
	Time        int64       `json:"time"`        // 发送时间
	MessageType string      `json:"messageType"` // group private
	GroupId     string      `json:"groupId"`     // 群号，如果是群聊消息
	Sender      SenderBase  `json:"sender"`      // 发送者
	Message     string      `json:"message"`     // 消息内容
	RawId       interface{} `json:"rawId"`       // 原始信息ID，用于处理撤回等
	Platform    string      `json:"platform"`    // 当前平台
	TmpUid      string      `json:"-" yaml:"-"`
}

// GroupPlayerInfoBase 群内玩家信息
type GroupPlayerInfoBase struct {
	Name            string `yaml:"name"` // 玩家昵称
	UserId          string `yaml:"userId"`
	InGroup         bool   `yaml:"inGroup"`         // 是否在群内，有时一个人走了，信息还暂时残留
	LastCommandTime int64  `yaml:"lastCommandTime"` // 上次发送指令时间

	// level int 权限
	DiceSideNum  int                  `yaml:"diceSideNum"` // 面数，为0时等同于d100
	Vars         *PlayerVariablesItem `yaml:"-"`           // 玩家的群内变量
	ValueMapTemp lockfree.HashMap     `yaml:"-"`           // 玩家的群内临时变量
	//ValueMapTemp map[string]*VMValue  `yaml:"-"`           // 玩家的群内临时变量

	TempValueAlias *map[string][]string `yaml:"-"` // 群内临时变量别名 - 其实这个有点怪的，为什么在这里？
}

// GroupPlayerInfo 这是一个YamlWrapper，没有实际作用
// 原因见 https://github.com/go-yaml/yaml/issues/712
type GroupPlayerInfo struct {
	GroupPlayerInfoBase `yaml:",inline,flow"`
}

type GroupInfo struct {
	Active           bool                        `json:"active" yaml:"active"`  // 是否在群内开启
	ActivatedExtList []*ExtInfo                  `yaml:"activatedExtList,flow"` // 当前群开启的扩展列表
	Players          map[string]*GroupPlayerInfo `yaml:"players"`               // 群员角色数据
	NotInGroup       bool                        `yaml:"notInGroup"`            // 是否已经离开群

	GroupId     string          `yaml:"groupId"`
	GroupName   string          `yaml:"groupName"`
	DiceIds     map[string]bool `yaml:"diceIds,flow"` // 对应的骰子ID(格式 平台:ID)，对应单骰多号情况，例如骰A B都加了群Z，A退群不会影响B在群内服务
	BotList     map[string]bool `yaml:"botList,flow"` // 其他骰子列表
	DiceSideNum int64           `yaml:"diceSideNum"`

	//ValueMap     map[string]*VMValue `yaml:"-"`
	ValueMap     lockfree.HashMap `yaml:"-"`
	HelpPackages []string         `yaml:"-"`
	CocRuleIndex int              `yaml:"cocRuleIndex"`
	LogCurName   string           `yaml:"logCurFile"`
	LogOn        bool             `yaml:"logOn"`

	QuitMarkAutoClean   bool   `yaml:"-"`                 // 自动清群 - 播报，即将自动退出群组
	QuitMarkMaster      bool   `yaml:"-"`                 // 骰主命令退群 - 播报，即将自动退出群组
	RecentCommandTime   int64  `yaml:"recentCommandTime"` // 最近一次发送有效指令的时间
	ShowGroupWelcome    bool   `yaml:"showGroupWelcome"`  // 是否迎新
	GroupWelcomeMessage string `yaml:"groupWelcomeMessage"`
	//FirstSpeechMade     bool   `yaml:"firstSpeechMade"` // 是否做过进群发言
	LastCustomReplyTime float64 `yaml:"-"` // 上次自定义回复时间
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

func (group *GroupInfo) ExtInactive(name string) *ExtInfo {
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

type EndPointInfoBase struct {
	Id                  string `yaml:"id" json:"id"` // uuid
	Nickname            string `yaml:"nickname" json:"nickname"`
	State               int    `yaml:"state" json:"state"` // 状态 0 断开 1已连接 2连接中
	UserId              string `yaml:"userId" json:"userId"`
	GroupNum            int64  `yaml:"groupNum" json:"groupNum"`                       // 拥有群数
	CmdExecutedNum      int64  `yaml:"cmdExecutedNum" json:"cmdExecutedNum"`           // 指令执行次数
	CmdExecutedLastTime int64  `yaml:"cmdExecutedLastTime" json:"cmdExecutedLastTime"` // 指令执行次数
	OnlineTotalTime     int64  `yaml:"onlineTotalTime" json:"onlineTotalTime"`         // 在线时长

	Platform     string `yaml:"platform" json:"platform"`     // 平台，如QQ等
	RelWorkDir   string `yaml:"relWorkDir" json:"relWorkDir"` // 工作目录
	Enable       bool   `yaml:"enable" json:"enable"`         // 是否启用
	ProtocolType string `yaml:"protocolType"`                 // 协议类型，如onebot、koishi等

	IsPublic bool       `yaml:"isPublic"`
	Session  *IMSession `yaml:"-" json:"-"`
}

type EndPointInfo struct {
	EndPointInfoBase `yaml:"baseInfo"`

	Adapter PlatformAdapter `yaml:"adapter" json:"adapter"`
}

func (d *EndPointInfo) UnmarshalYAML(value *yaml.Node) error {
	if d.Adapter != nil {
		return value.Decode(d)
	}

	var val struct {
		EndPointInfoBase `yaml:"baseInfo"`
	}
	err := value.Decode(&val)
	if err != nil {
		return err
	}
	d.EndPointInfoBase = val.EndPointInfoBase

	if val.Platform == "QQ" {
		var val struct {
			Adapter *PlatformAdapterQQOnebot `yaml:"adapter"`
		}

		err := value.Decode(&val)
		if err != nil {
			return err
		}

		d.Adapter = val.Adapter
	}
	return err
}

type IMSession struct {
	Parent    *Dice           `yaml:"-"`
	EndPoints []*EndPointInfo `yaml:"endPoints"`

	ServiceAtNew   map[string]*GroupInfo           `json:"servicesAt" yaml:"servicesAt"`
	PlayerVarsData map[string]*PlayerVariablesItem `yaml:"-"` // 感觉似乎没有什么存本地的必要

	// 注意，旧数据！
	LegacyConns     []*ConnectInfoItem       `yaml:"connections"` // 仅为
	LegacyServiceAt map[int64]*ServiceAtItem `json:"serviceAt" yaml:"serviceAt"`
	//LegacyPlayerVarsData map[int64]*PlayerVariablesItem `yaml:"PlayerVarsData"`
}

type MsgContext struct {
	MessageType string
	Group       *GroupInfo
	Player      *GroupPlayerInfo

	EndPoint        *EndPointInfo
	Session         *IMSession
	Dice            *Dice
	IsCurGroupBotOn bool

	IsPrivate       bool
	CommandId       uint64
	CommandHideFlag string      // 暗骰标记
	CommandInfo     interface{} // 命令信息
	PrivilegeLevel  int         // 权限等级
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
	if msg.MessageType == "group" || msg.MessageType == "private" {
		group := s.ServiceAtNew[msg.GroupId]
		if group == nil && msg.GroupId != "" {
			// 注意: 此处必须开启，不然下面mctx.player取不到
			autoOn := true
			if msg.Platform == "QQ-CH" {
				autoOn = d.QQChannelAutoOn
			}
			group = SetBotOnAtGroup(mctx, msg.GroupId)
			group.Active = autoOn
			txt := fmt.Sprintf("自动激活: 发现无记录群组(%s)，因为已是群成员，所以自动激活，开启状态: %t", group.GroupId, autoOn)
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
			if d.CustomReplyConfig.Enable && group.ExtGetActive("reply") != nil {
				mustLoadUser = true
			}
		}

		// 可能是发命令时，必须加载信息
		maybeCommand := CommandCheckPrefix(msg.Message, d.CommandPrefix)
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
			mctx.IsCurGroupBotOn = IsCurGroupBotOn(s, msg)
		}

		if mctx.Group != nil && mctx.Group.Active {
			for _, i := range mctx.Group.ActivatedExtList {
				if i.OnMessageReceived != nil {
					i.OnMessageReceived(mctx, msg)
				}
			}
		}

		var cmdLst []string
		if maybeCommand {
			if msg.Sender.GroupRole == "admin" {
				mctx.PrivilegeLevel = 50 // 群管理
			}
			if msg.Sender.GroupRole == "owner" {
				mctx.PrivilegeLevel = 60 // 群主
			}

			if d.MasterCheck(mctx.Player.UserId) {
				mctx.PrivilegeLevel = 100
			}

			// 兼容模式检查
			if d.CommandCompatibleMode {
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
		}

		// 收到群 test(1111) 内 XX(222) 的消息: 好看 (1232611291)
		if msg.MessageType == "group" {
			if mctx.CommandId != 0 {
				// 关闭状态下，如果被@那么视为开启
				if !mctx.IsCurGroupBotOn && cmdArgs.AmIBeMentioned {
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
						ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "核心:骰子崩溃"))
					}
				}()

				// 跳过@其他骰子而不@自己的
				cmdArgs.SomeoneBeMentionedButNotMe = len(cmdArgs.At) > 0 && (!cmdArgs.AmIBeMentioned)
				if cmdArgs.MentionedOtherDice {
					// @其他骰子
					return
				}

				if cmdArgs.Command != "botlist" && !cmdArgs.AmIBeMentioned {
					myuid := ep.UserId
					// 屏蔽机器人发送的消息
					if mctx.MessageType == "group" {
						if mctx.Group.BotList[msg.Sender.UserId] {
							return
						}
						// 当其他机器人被@，不回应
						for _, i := range cmdArgs.At {
							uid := i.UserId
							if uid == myuid {
								// 忽略自己
								continue
							}
							if mctx.Group.BotList[uid] {
								return
							}
						}
					}
				}

				var ret bool

				// 试图匹配自定义指令
				if mctx.Group != nil && mctx.Group.Active {
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
					if mctx.Group != nil {
						mctx.Group.RecentCommandTime = ep.CmdExecutedLastTime
					}
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
			// 试图匹配自定义回复
			if mctx.Group != nil && (mctx.Group.Active || amIBeMentioned) {
				for _, i := range mctx.Group.ActivatedExtList {
					if i.OnNotCommandReceived != nil {
						i.OnNotCommandReceived(mctx, msg)
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
	}

	tryItemSolve := func(item *CmdItemInfo) bool {
		if item != nil {
			ret := item.Solve(ctx, msg, cmdArgs)
			if ret.Solved {
				if ret.ShowLongHelp {
					help := item.LongHelp
					if help == "" {
						// 这是为了防止别的骰子误触发
						help = item.Name + ":\n" + item.Help
					}
					ReplyToSender(ctx, msg, help)
				}

				return true
			}
		}
		return false
	}

	group := ctx.Group
	if group.Active || ctx.IsCurGroupBotOn {
		for _, i := range group.ActivatedExtList {
			if i.OnCommandReceived != nil {
				i.OnCommandReceived(ctx, msg, cmdArgs)
			}
		}
	}

	item := ctx.Session.Parent.CmdMap[cmdArgs.Command]
	if tryItemSolve(item) {
		return true
	}

	//if msg.MessageType == "private" {
	//	// 个人消息
	//	for _, i := range ctx.Dice.ExtList {
	//		if i.ActiveOnPrivate {
	//			item := i.CmdMap[cmdArgs.Command]
	//			if tryItemSolve(item) {
	//				return true
	//			}
	//		}
	//	}
	//} else {
	// 群消息
	if group != nil && (group.Active || ctx.IsCurGroupBotOn) {
		for _, i := range group.ActivatedExtList {
			item := i.CmdMap[cmdArgs.Command]
			if tryItemSolve(item) {
				return true
			}
		}
	}
	//}
	return false
}

// SetEnable
/* 如果已连接，将断开连接，如果开着GCQ将自动结束。如果启用的话，则反过来  */
func (c *EndPointInfo) SetEnable(d *Dice, enable bool) {
	if c.Enable != enable {
		c.Adapter.SetEnable(enable)
	}
}

func (ep *EndPointInfo) AdapterSetup() {
	if ep.Platform == "QQ" {
		pa := ep.Adapter.(*PlatformAdapterQQOnebot)
		pa.Session = ep.Session
		pa.EndPoint = ep
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
