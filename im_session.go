package main

import (
	"encoding/json"
	"fmt"
	"github.com/sacOO7/gowebsocket"
	"math/rand"
	"os"
	"os/signal"
	"sealdice-core/core"
	"sort"
	"time"
)

// 2022/02/03 11:47:42 Recieved message {"font":0,"message":"test","message_id":-487913662,"message_type":"private","post_type":"message","raw_message":"test","self_id":1001,"sender":{"age":0,"nickname":"鏈ㄨ惤","sex":"unknown","user_id":1002},"sub_type":"friend","target_id":1001,"time":1643860062,"user_id":1002}
// {"anonymous":null,"font":0,"group_id":111,"message":"qqq","message_id":884917177,"message_seq":1434,"message_type":"group","post_type":"message","raw_message":"qqq","self_id":1001,"sender":{"age":0,"area":"","card":"","level":"","nickname":"鏈ㄨ惤","role":"member","sex":"unknown","title":"","user_id":1002},"sub_type":"normal","time":1643863961,"user_id":1002}
// {"anonymous":null,"font":0,"group_id":111,"message":"[CQ:at,qq=1001]   .r test","message_id":888971055,"message_seq":1669,"message_type":"group","post_type":"message","raw_message":"[CQ:at,qq=1001]   .r test","self_id":1001,"sender":{"age":0,"area":"","card":"","level":"","nickname":"鏈ㄨ惤","role":"member","sex":"unknown","title":"","user_id":1002},"sub_type":"normal","time":1644127751,"user_id":1002}

func replyPerson(ctx *MsgContext, userId int64, text string) {
	replyPersonRaw(ctx, userId, text, "")
}

func replyPersonRaw(ctx *MsgContext, userId int64, text string, flag string) {
	for _, i := range ctx.dice.extList {
		if i.OnMessageSend != nil {
			i.OnMessageSend(ctx, "private", userId, text, flag)
		}
	}
	time.Sleep(time.Duration((0.8 + rand.Float64()) * float64(time.Second)))

	type GroupMessageParams struct {
		MessageType string        `json:"message_type"`
		UserId int64             `json:"user_id"`
		Message string            `json:"message"`
	}

	a, _ := json.Marshal(struct {
		Action string             `json:"action"`
		Params GroupMessageParams `json:"params"`
	}{
		Action: "send_msg",
		Params: GroupMessageParams{
			MessageType: "private",
			UserId:      userId,
			Message:     text,
		},
	})

	ctx.session.Socket.SendText(string(a))
}

func GetGroupInfo(socket *gowebsocket.Socket, groupId int64) {
	type GroupMessageParams struct {
		GroupId int64  `json:"group_id"`
	}

	a, _ := json.Marshal(struct {
		Action string             `json:"action"`
		Params GroupMessageParams `json:"params"`
		Echo int64 `json:"echo"`
	}{
		"get_group_info",
		GroupMessageParams{
			groupId,
		},
		-2,
	})
	socket.SendText(string(a))
}

func SetGroupAddRequest(socket *gowebsocket.Socket, flag string, subType string, approve bool, reason string) {
	type DetailParams struct {
		Flag    string `json:"flag"`
		SubType string `json:"sub_type"`
		Approve bool   `json:"approve"`
		Reason  string `json:"reason"`
	}

	a, _ := json.Marshal(struct {
		Action string       `json:"action"`
		Params DetailParams `json:"params"`
	}{
		"set_group_add_request",
		DetailParams{
			Flag:    flag,
			SubType: subType,
			Approve: approve,
			Reason:  reason,
		},
	})
	socket.SendText(string(a))
}


func quitGroup(s *IMSession, groupId int64) {
	type GroupMessageParams struct {
		GroupId int64  `json:"group_id"`
	}

	a, _ := json.Marshal(struct {
		Action string             `json:"action"`
		Params GroupMessageParams `json:"params"`
	}{
		"set_group_leave",
		GroupMessageParams{
			groupId,
		},
	})
	s.Socket.SendText(string(a))
}

func replyGroup(ctx *MsgContext, groupId int64, text string) {
	replyGroupRaw(ctx, groupId, text, "")
}

func replyGroupRaw(ctx *MsgContext, groupId int64, text string, flag string) {
	if ctx.session.ServiceAt[groupId] != nil {
		for _, i := range ctx.session.ServiceAt[groupId].ActivatedExtList {
			if i.OnMessageSend != nil {
				i.OnMessageSend(ctx, "group", groupId, text, flag)
			}
		}
	}

	time.Sleep(time.Duration((0.8 + rand.Float64()) * float64(time.Second)))

	type GroupMessageParams struct {
		GroupId int64  `json:"group_id"`
		Message string `json:"message"`
	}

	a, _ := json.Marshal(struct {
		Action string             `json:"action"`
		Params GroupMessageParams `json:"params"`
	}{
		"send_group_msg",
		GroupMessageParams{
			groupId,
			text, // "golang client test",
		},
	})
	ctx.session.Socket.SendText(string(a))
}

func replyToSenderRaw(ctx *MsgContext, msg *Message, text string, flag string) {
	inGroup := msg.MessageType == "group"
	if inGroup {
		replyGroupRaw(ctx, msg.GroupId, text, flag)
	} else {
		replyPersonRaw(ctx, msg.Sender.UserId, text, flag)
	}
}

func replyToSender(ctx *MsgContext, msg *Message, text string) {
	replyToSenderRaw(ctx, msg, text, "")
}

func (s *IMSession) GetLoginInfo() {
	a, _ := json.Marshal(struct {
		Action string `json:"action"`
		Echo int64 `json:"echo"`
	}{
		Action: "get_login_info",
		Echo: -1,
	})
	s.Socket.SendText(string(a))
}

type Sender struct {
	Age      int32  `json:"age"`
	Card     string `json:"card"`
	Nickname string `json:"nickname"`
	Role     string `json:"owner"`
	UserId   int64  `json:"user_id"`
}

type Message struct {
	MessageId     int64  `json:"message_id"`
	MessageType   string `json:"message_type"` // group
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
	UserId int64 `json:"user_id"`
	SelfId int64 `json:"self_id"`

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
	Echo   int `json:"echo"`
}

type PlayerInfo struct {
	UserId         int64 `yaml:"userId"`
	Name           string // 玩家昵称
	ValueNumMap    map[string]int64  `yaml:"valueNumMap"`
	ValueStrMap    map[string]string `yaml:"valueStrMap"`
	RpToday        int               `yaml:"rpToday"`
	RpTime         string            `yaml:"rpTime"`
	lastUpdateTime int64             `yaml:"lastUpdateTime"`

	// level int 权限
	DiceSideNum    int `yaml:"diceSideNum"` // 面数，为0时等同于d100
	TempValueAlias *map[string][]string `yaml:"-"`

	ValueMap map[string]VMValue `yaml:"-"`
	ValueMapTemp map[string]VMValue `yaml:"-"`
}

type ServiceAtItem struct {
	Active bool `json:"active" yaml:"active"` // 需要能记住配置，故有此选项
	ActivatedExtList []*ExtInfo `yaml:"activatedExtList"` // 当前群开启的扩展列表
	Players map[int64]*PlayerInfo // 群员信息

	LogCurName  string   `yaml:"logCurFile"`
	LogOn       bool     `yaml:"logOn"`
	GroupId int64 `yaml:"groupId"`
	GroupName   string   `yaml:"groupName"`

	ValueMap map[string]VMValue `yaml:"-"`
	// http://www.antagonistes.com/files/CoC%20CheatSheet.pdf
	//RuleCriticalSuccessValue *int64 // 大成功值，1默认
	//RuleFumbleValue *int64 // 大失败值 96默认
}

// 大失败
func (i *ServiceAtItem) GetFumbleValue() int64 {
	return 96
}

// 大成功
func (i *ServiceAtItem) getCriticalSuccessValue() int64 {
	return 1
}

type PlayerVariablesItem struct{
	Loaded bool `yaml:"-"`
	ValueMap map[string]VMValue `yaml:"-"`
	LastSyncToCloudTime int64 `yaml:"lastSyncToCloudTime"`
	LastUsedTime int64 `yaml:"lastUsedTime"`
}

type IMSession struct {
	Socket   *gowebsocket.Socket `yaml:"-"`
	Nickname string              `yaml:"-"`
	UserId   int64               `yaml:"userId"`
	parent   *Dice               `yaml:"-"`

	PlayerVarsData map[int64]*PlayerVariablesItem `yaml:"PlayerVarsData"`

	ServiceAt    map[int64]*ServiceAtItem `json:"serviceAt" yaml:"serviceAt"`
	CommandIndex int64                    `yaml:"-"`
	ConnectUrl   string                   `yaml:"connectUrl"`

	//GroupId int64 `json:"group_id"`
}


type ByLength []string

func (s ByLength) Len() int {
	return len(s)
}
func (s ByLength) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s ByLength) Less(i, j int) bool {
	return len(s[i]) > len(s[j])
}

type MsgContext struct {
	MessageType     string
	session         *IMSession
	group           *ServiceAtItem
	player          *PlayerInfo
	dice            *Dice
	isCurGroupBotOn bool
}

func (ctx *MsgContext) LoadPlayerVars() *PlayerVariablesItem {
	if ctx.player != nil {
		return LoadPlayerVars(ctx.session, ctx.player.UserId)
	}
	return nil
}

func (s *IMSession) serve() {
	log := core.GetLogger()
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	session := s
	if s.ConnectUrl == "" {
		s.ConnectUrl = "ws://127.0.0.1:6700"
	}
	socket := gowebsocket.New(s.ConnectUrl)
	session.Socket = &socket

	socket.OnConnected = func(socket gowebsocket.Socket) {
		log.Info("Connected to server")
		//  {"data":{"nickname":"闃斧鐗岃�佽檸鏈�","user_id":1001},"retcode":0,"status":"ok"}
		s.GetLoginInfo()
	}

	socket.OnConnectError = func(err error, socket gowebsocket.Socket) {
		log.Info("Recieved connect error: ", err)
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

				log.Debug("User info received.")
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
					log.Debug("Group info received: ", msg.Data.GroupName)
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
			if msg.PostType == "notice"  && msg.NoticeType == "group_increase" {
				if msg.UserId == msg.SelfId {
					// 判断进群的人是自己，自动启动
					SetBotOnAtGroup(session, msg)
					replyGroupRaw(&MsgContext{session: session, dice: session.parent}, msg.GroupId, fmt.Sprintf("<%s>已经就绪。可通过.help查看指令列表", session.Nickname), "")
				}
				return
			}

			fmt.Println("Recieved message " + message)

			// 处理命令
			if msg.MessageType == "group" || msg.MessageType == "private" {
				mctx := &MsgContext{}
				mctx.dice = session.parent
				mctx.MessageType = msg.MessageType
				mctx.session = session
				var cmdLst []string

				// 兼容模式检查
				if s.parent.CommandCompatibleMode {
					for k := range session.parent.cmdMap {
						cmdLst = append(cmdLst, k)
					}

					sa := session.ServiceAt[msg.GroupId]
					if sa != nil && sa.Active {
						for _, i := range sa.ActivatedExtList {
							for k := range i.cmdMap {
								cmdLst = append(cmdLst, k)
							}
						}
					}
					sort.Sort(ByLength(cmdLst))
				}

				// 收到信息回调
				sa := session.ServiceAt[msg.GroupId]
				mctx.group = sa
				mctx.player = getPlayerInfoBySender(session, msg)
				mctx.isCurGroupBotOn = isCurGroupBotOn(session, msg)

				if sa != nil && sa.Active {
					for _, i := range sa.ActivatedExtList {
						if i.OnMessageReceived != nil {
							i.OnMessageReceived(mctx, msg)
						}
					}
				}

				msgInfo := CommandParse(msg.Message, s.parent.CommandCompatibleMode, cmdLst)

				if msgInfo != nil {
					f := func() {
						defer func() {
							if r := recover(); r != nil {
								//  + fmt.Sprintf("%s", r)
								core.GetLogger().Error(r)
								replyToSender(mctx, msg, "已从核心崩溃中恢复，请带指令联系开发者。注意不要重复发送本指令以免风控。")
							}
						}()
						session.commandSolve(mctx, msg, msgInfo)
					}
					go f()
					//session.commandSolve(mctx, msg, msgInfo)

					//c, _ := json.Marshal(msgInfo)
					//text := fmt.Sprintf("指令测试，来自群%d - %s(%d)：参数 %s", msg.GroupId, msg.Sender.Nickname, msg.Sender.UserId, c);
					//replyGroup(Socket, 11, text)
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
		return
	}

	socket.Connect()

	for {
		select {
		case <-interrupt:
			log.Info("interrupt")
			socket.Close()
			return
		}
	}
}

func (s *IMSession) commandSolve(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
	cmdArgs.AmIBeMentioned = false
	for _, i := range cmdArgs.At {
		if i.UserId == ctx.session.UserId {
			cmdArgs.AmIBeMentioned = true
			break
		}
	}

	tryItemSolve := func (item *CmdItemInfo) bool {
		if item != nil {
			ret := item.solve(ctx, msg, cmdArgs)
			if ret.success {
				return true
			}
		}
		return false
	}

	sa := ctx.group
	if sa != nil && sa.Active {
		for _, i := range sa.ActivatedExtList {
			if i.OnCommandReceived != nil {
				i.OnCommandReceived(ctx, msg, cmdArgs)
			}
		}
	}

	item := ctx.session.parent.cmdMap[cmdArgs.Command]
	if tryItemSolve(item) {
		return
	}

	if sa != nil && sa.Active {
		for _, i := range sa.ActivatedExtList {
			item := i.cmdMap[cmdArgs.Command]
			if tryItemSolve(item) {
				return
			}
		}
	}

	if msg.MessageType == "private" {
		for _, i := range ctx.dice.extList {
			if i.ActiveOnPrivate {
				item := i.cmdMap[cmdArgs.Command]
				if tryItemSolve(item) {
					return
				}
			}
		}
	}
}
