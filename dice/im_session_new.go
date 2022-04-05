package dice

import (
	"gopkg.in/yaml.v3"
	"runtime/debug"
	"sort"
	"time"
)

type SenderBase struct {
	Nickname string `json:"nickname"`
	UserId   string `json:"userId"`
}

// Message 消息的重要信息
// 时间
// 发送地点(群聊/私聊)
// 人物(是谁发的)
// 内容
type Message struct {
	Time        int64      `json:"time"`        // 发送时间
	MessageType string     `json:"messageType"` // group private
	GroupId     string     `json:"groupId"`     // 群号，如果是群聊消息
	Sender      SenderBase `json:"sender"`      // 发送者
	Message     string     `json:"message"`     // 消息内容
}

// GroupPlayerInfo 群内玩家信息
type GroupPlayerInfoBase struct {
	Name            string `yaml:"name"` // 玩家昵称
	UserId          string `yaml:"userId"`
	InGroup         bool   `yaml:"inGroup"`         // 是否在群内，有时一个人走了，信息还暂时残留
	LastCommandTime int64  `yaml:"lastCommandTime"` // 上次发送指令时间

	// level int 权限
	DiceSideNum  int                  `yaml:"diceSideNum"` // 面数，为0时等同于d100
	Vars         *PlayerVariablesItem `yaml:"-"`           // 玩家的群内变量
	ValueMapTemp map[string]*VMValue  `yaml:"-"`           // 玩家的群内临时变量

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

	ValueMap     map[string]*VMValue `yaml:"-"`
	HelpPackages []string            `yaml:"-"`
	CocRuleIndex int                 `yaml:"cocRuleIndex"`
	LogCurName   string              `yaml:"logCurFile"`
	LogOn        bool                `yaml:"logOn"`
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

	Session *IMSession `yaml:"-" json:"-"`
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
	PlayerVarsData map[string]*PlayerVariablesItem `yaml:"playerVarsData"`

	// 注意，旧数据！
	LegacyConns          []*ConnectInfoItem             `yaml:"connections"` // 仅为
	LegacyServiceAt      map[int64]*ServiceAtItem       `json:"serviceAt" yaml:"serviceAt"`
	LegacyPlayerVarsData map[int64]*PlayerVariablesItem `yaml:"PlayerVarsData"`
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
	CommandHideFlag string // 暗骰标记
	PrivilegeLevel  int    // 权限等级
}

func (s *IMSession) Execute(ep *EndPointInfo, msg *Message) {
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
		var cmdLst []string

		group := s.ServiceAtNew[msg.GroupId]
		if group == nil && msg.GroupId != "" {
			group = SetBotOnAtGroup(mctx, msg.GroupId)
			log.Infof("自动激活: 发现无记录群组(%s)，因为已是群成员，所以自动激活", group.GroupId)
			ep.Adapter.GetGroupInfoAsync(msg.GroupId)
		}

		if group != nil && group.Active {
			for _, i := range group.ActivatedExtList {
				if i.OnMessageReceived != nil {
					i.OnMessageReceived(mctx, msg)
				}
			}
		}

		maybeCommand := CommandCheckPrefix(msg.Message, d.CommandPrefix)
		if maybeCommand {
			mctx.Group, mctx.Player = GetPlayerInfoBySender(mctx, msg)
			mctx.IsCurGroupBotOn = IsCurGroupBotOn(s, msg)
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
				if g != nil && g.Active {
					for _, i := range g.ActivatedExtList {
						for k := range i.CmdMap {
							cmdLst = append(cmdLst, k)
						}
					}
				}
				sort.Sort(ByLength(cmdLst))
			}
		}

		cmdArgs := CommandParse(msg.Message, d.CommandCompatibleMode, cmdLst, d.CommandPrefix)
		if cmdArgs != nil {
			mctx.CommandId = getNextCommandId()

			// 设置AmIBeMentioned
			cmdArgs.AmIBeMentioned = false
			for _, i := range cmdArgs.At {
				if i.UserId == ep.UserId {
					cmdArgs.AmIBeMentioned = true
					break
				}
			}
		}

		// 收到群 test(1111) 内 XX(222) 的消息: 好看 (1232611291)
		if msg.MessageType == "group" {
			if mctx.CommandId != 0 {
				log.Infof("收到群(%s)内<%s>(%s)的指令: %s", msg.GroupId, msg.Sender.Nickname, msg.Sender.UserId, msg.Message)
			} else {
				if !d.OnlyLogCommandInGroup {
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

				ret := s.commandSolve(mctx, msg, cmdArgs)
				if ret {
					ep.CmdExecutedNum += 1
					ep.CmdExecutedLastTime = time.Now().Unix()
					mctx.Player.LastCommandTime = ep.CmdExecutedLastTime
				} else {
					if msg.MessageType == "group" {
						log.Infof("忽略指令(骰子关闭/扩展关闭/未知指令): 来自群(%s)内<%s>(%s): %s", msg.GroupId, msg.Sender.Nickname, msg.Sender.UserId, msg.Message)
					}

					if msg.MessageType == "private" {
						log.Infof("忽略指令(骰子关闭/扩展关闭/未知指令): 来自<%s>(%s)的私聊: %s", msg.Sender.Nickname, msg.Sender.UserId, msg.Message)
					}
				}
			}
			go f()
		} else {
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
