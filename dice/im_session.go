package dice

import (
	"encoding/json"
	"fmt"
	"github.com/sacOO7/gowebsocket"
	"math/rand"
	"os"
	"os/signal"
	"sealdice-core/core"
	"sort"
	"syscall"
	"time"
)

type Sender struct {
	Age      int32  `json:"age"`
	Card     string `json:"card"`
	Nickname string `json:"nickname"`
	Role     string `json:"owner"`
	UserId   int64  `json:"user_id"`
}

type Message struct {
	MessageId     int64  `json:"message_id"`
	MessageType   string `json:"message_type"` // Group
	Sender        Sender `json:"sender"`       // 发送者
	RawMessage    string `json:"raw_message"`
	Message       string `json:"message"` // 消息内容
	Time          int64  `json:"time"`    // 发送时间
	MetaEventType string `json:"meta_event_type"`
	GroupId       int64  `json:"group_id"`     // 群号
	PostType      string `json:"post_type"`    // 上报类型，如group、notice
	RequestType   string `json:"request_type"` // 请求类型，如group
	SubType       string `json:"sub_type"`     // 子类型，如add invite
	Flag          string `json:"flag"`         // 请求 flag, 在调用处理请求的 API 时需要传入
	NoticeType    string `json:"notice_type"`
	UserId        int64  `json:"user_id"`
	SelfId        int64  `json:"self_id"`

	Data *struct {
		// 个人信息
		Nickname string `json:"nickname"`
		UserId   int64  `json:"user_id"`

		// 群信息
		GroupId         int64  `json:"group_id"`          // 群号
		GroupCreateTime uint32 `json:"group_create_time"` // 群号
		MemberCount     int64  `json:"member_count"`
		GroupName       string `json:"group_name"`
	} `json:"data"`
	Retcode int64 `json:"retcode"`
	//Status string `json:"status"`
	Echo int `json:"echo"`
}

type PlayerInfo struct {
	UserId int64  `yaml:"userId"`
	Name   string // 玩家昵称
	//ValueNumMap    map[string]int64  `yaml:"valueNumMap"`
	//ValueStrMap    map[string]string `yaml:"valueStrMap"`
	RpToday        int    `yaml:"rpToday"`
	RpTime         string `yaml:"rpTime"`
	LastUpdateTime int64  `yaml:"lastUpdateTime"`

	// level int 权限
	DiceSideNum    int                  `yaml:"diceSideNum"` // 面数，为0时等同于d100
	TempValueAlias *map[string][]string `yaml:"-"`

	ValueMap     map[string]VMValue `yaml:"-"`
	ValueMapTemp map[string]VMValue `yaml:"-"`
}

type ServiceAtItem struct {
	Active           bool                  `json:"active" yaml:"active"` // 需要能记住配置，故有此选项
	ActivatedExtList []*ExtInfo            `yaml:"activatedExtList"`     // 当前群开启的扩展列表
	Players          map[int64]*PlayerInfo // 群员信息

	LogCurName string `yaml:"logCurFile"`
	LogOn      bool   `yaml:"logOn"`
	GroupId    int64  `yaml:"groupId"`
	GroupName  string `yaml:"groupName"`

	ValueMap     map[string]VMValue `yaml:"-"`
	CocRuleIndex int                `yaml:"cocRuleIndex"`

	// http://www.antagonistes.com/files/CoC%20CheatSheet.pdf
	//RuleCriticalSuccessValue *int64 // 大成功值，1默认
	//RuleFumbleValue *int64 // 大失败值 96默认
}

type PlayerVariablesItem struct {
	Loaded              bool               `yaml:"-"`
	ValueMap            map[string]VMValue `yaml:"-"`
	LastSyncToCloudTime int64              `yaml:"lastSyncToCloudTime"`
	LastUsedTime        int64              `yaml:"lastUsedTime"`
}

type IMSession struct {
	Socket   *gowebsocket.Socket `yaml:"-"`
	Nickname string              `yaml:"-"`
	UserId   int64               `yaml:"userId"`
	Parent   *Dice               `yaml:"-"`

	PlayerVarsData map[int64]*PlayerVariablesItem `yaml:"PlayerVarsData"`

	ServiceAt    map[int64]*ServiceAtItem `json:"serviceAt" yaml:"serviceAt"`
	CommandIndex int64                    `yaml:"-"`
	ConnectUrl   string                   `yaml:"connectUrl"`

	//GroupId int64 `json:"group_id"`
}

type MsgContext struct {
	MessageType     string
	Session         *IMSession
	Group           *ServiceAtItem
	Player          *PlayerInfo
	Dice            *Dice
	IsCurGroupBotOn bool
}

func (s *IMSession) Serve() int {
	log := core.GetLogger()
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	disconnected := make(chan int, 1)

	session := s
	if s.ConnectUrl == "" {
		s.ConnectUrl = "ws://127.0.0.1:6700"
	}
	socket := gowebsocket.New(s.ConnectUrl)
	session.Socket = &socket

	socket.OnConnected = func(socket gowebsocket.Socket) {
		fmt.Println("onebot 连接成功")
		log.Info("onebot 连接成功")
		//  {"data":{"nickname":"闃斧鐗岃�佽檸鏈�","user_id":1001},"retcode":0,"status":"ok"}
		s.GetLoginInfo()
	}

	socket.OnConnectError = func(err error, socket gowebsocket.Socket) {
		log.Info("Recieved connect error: ", err)
		fmt.Println("连接失败")
		disconnected <- 2
	}

	socket.OnTextMessage = func(message string, socket gowebsocket.Socket) {
		msg := new(Message)
		err := json.Unmarshal([]byte(message), msg)

		if err == nil {
			// 心跳包，忽略
			if msg.MetaEventType == "heartbeat" {
				return
			}
			if msg.MetaEventType == "heartbeat" {
				return
			}

			// 获得用户信息
			if msg.Echo == -1 {
				session.UserId = msg.Data.UserId
				session.Nickname = msg.Data.Nickname

				log.Debug("骰子信息已刷新")
				return
			}

			// 获得群信息
			if msg.Echo == -2 {
				if msg.Data != nil {
					group := session.ServiceAt[msg.Data.GroupId]
					if group != nil {
						group.GroupName = msg.Data.GroupName
						group.GroupId = msg.Data.GroupId
					}
					log.Debug("群信息刷新: ", msg.Data.GroupName)
				}
				return
			}

			// 处理加群请求
			if msg.PostType == "request" && msg.RequestType == "group" && msg.SubType == "invite" {
				time.Sleep(time.Duration((0.8 + rand.Float64()) * float64(time.Second)))
				SetGroupAddRequest(s.Socket, msg.Flag, msg.SubType, true, "")
				return
			}

			// 入群后自动开启
			if msg.PostType == "notice" && msg.NoticeType == "group_increase" {
				if msg.UserId == msg.SelfId {
					// 判断进群的人是自己，自动启动
					SetBotOnAtGroup(session, msg)
					replyGroupRaw(&MsgContext{Session: session, Dice: session.Parent}, msg.GroupId, fmt.Sprintf("<%s>已经就绪。可通过.help查看指令列表", session.Nickname), "")
				}
				return
			}

			fmt.Println("Recieved message " + message)

			// 处理命令
			if msg.MessageType == "group" || msg.MessageType == "private" {
				mctx := &MsgContext{}
				mctx.Dice = session.Parent
				mctx.MessageType = msg.MessageType
				mctx.Session = session
				var cmdLst []string

				// 兼容模式检查
				if s.Parent.CommandCompatibleMode {
					for k := range session.Parent.CmdMap {
						cmdLst = append(cmdLst, k)
					}

					sa := session.ServiceAt[msg.GroupId]
					if sa != nil && sa.Active {
						for _, i := range sa.ActivatedExtList {
							for k := range i.CmdMap {
								cmdLst = append(cmdLst, k)
							}
						}
					}
					sort.Sort(ByLength(cmdLst))
				}

				// 收到信息回调
				sa := session.ServiceAt[msg.GroupId]
				mctx.Group = sa
				mctx.Player = GetPlayerInfoBySender(session, msg)
				mctx.IsCurGroupBotOn = IsCurGroupBotOn(session, msg)

				if sa != nil && sa.Active {
					for _, i := range sa.ActivatedExtList {
						if i.OnMessageReceived != nil {
							i.OnMessageReceived(mctx, msg)
						}
					}
				}

				msgInfo := CommandParse(msg.Message, s.Parent.CommandCompatibleMode, cmdLst)

				if msgInfo != nil {
					//f := func() {
					//	defer func() {
					//		if r := recover(); r != nil {
					//			//  + fmt.Sprintf("%s", r)
					//			core.GetLogger().Error(r)
					//			ReplyToSender(mctx, msg, DiceFormatTmpl(mctx, "核心:骰子崩溃"))
					//		}
					//	}()
					//	session.commandSolve(mctx, msg, msgInfo)
					//}
					//go f()
					session.commandSolve(mctx, msg, msgInfo)
				} else {
					//text := fmt.Sprintf("信息 来自群%d - %s(%d)：%s", msg.GroupId, msg.Sender.Nickname, msg.Sender.UserId, msg.Message);
					//replyGroup(Socket, 22, text)
				}
				//}
				//}
			}
		} else {
			log.Error("error" + err.Error())
		}
	}

	socket.OnBinaryMessage = func(data []byte, socket gowebsocket.Socket) {
		log.Debug("Recieved binary data ", data)
	}

	socket.OnPingReceived = func(data string, socket gowebsocket.Socket) {
		log.Debug("Recieved ping " + data)
	}

	socket.OnPongReceived = func(data string, socket gowebsocket.Socket) {
		log.Debug("Recieved pong " + data)
	}

	socket.OnDisconnected = func(err error, socket gowebsocket.Socket) {
		log.Info("Disconnected from server ")
		disconnected <- 1
	}

	socket.Connect()
	defer func() {
		fmt.Println("socket close")
		go func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Println("关闭连接时遭遇异常")
					//core.GetLogger().Error(r)
				}
			}()

			// 可能耗时好久
			socket.Close()
		}()
	}()

	for {
		select {
		case <-interrupt:
			log.Info("interrupt")
			disconnected <- 0
			return 0
		case val := <-disconnected:
			return val
		}
	}
}

func (s *IMSession) commandSolve(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
	// 设置AmIBeMentioned
	cmdArgs.AmIBeMentioned = false
	for _, i := range cmdArgs.At {
		if i.UserId == ctx.Session.UserId {
			cmdArgs.AmIBeMentioned = true
			break
		}
	}

	// 设置临时变量
	if ctx.Player != nil {
		VarSetValue(ctx, "$t玩家", &VMValue{VMTypeString, fmt.Sprintf("<%s>", ctx.Player.Name)})
		VarSetValue(ctx, "$tQQ昵称", &VMValue{VMTypeString, fmt.Sprintf("<%s>", msg.Sender.Nickname)})
		VarSetValue(ctx, "$t个人骰子面数", &VMValue{VMTypeInt64, ctx.Player.DiceSideNum})
		VarSetValue(ctx, "$tQQ", &VMValue{VMTypeInt64, msg.Sender.UserId})
		// 注: 未来将私聊视为空群吧
	}

	tryItemSolve := func(item *CmdItemInfo) bool {
		if item != nil {
			ret := item.Solve(ctx, msg, cmdArgs)
			if ret.Success {
				return true
			}
		}
		return false
	}

	sa := ctx.Group
	if sa != nil && sa.Active {
		for _, i := range sa.ActivatedExtList {
			if i.OnCommandReceived != nil {
				i.OnCommandReceived(ctx, msg, cmdArgs)
			}
		}
	}

	item := ctx.Session.Parent.CmdMap[cmdArgs.Command]
	if tryItemSolve(item) {
		return
	}

	if sa != nil && sa.Active {
		for _, i := range sa.ActivatedExtList {
			item := i.CmdMap[cmdArgs.Command]
			if tryItemSolve(item) {
				return
			}
		}
	}

	if msg.MessageType == "private" {
		for _, i := range ctx.Dice.ExtList {
			if i.ActiveOnPrivate {
				item := i.CmdMap[cmdArgs.Command]
				if tryItemSolve(item) {
					return
				}
			}
		}
	}
}
