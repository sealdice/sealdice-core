package dice

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"sealdice-core/utils/satori"

	"github.com/google/uuid"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/sacOO7/gowebsocket"
)

type PlatformAdapterSealChat struct {
	Session  *IMSession    `yaml:"-" json:"-"`
	EndPoint *EndPointInfo `yaml:"-" json:"-"`

	ConnectURL string                    `yaml:"connectUrl" json:"connectUrl"` // 连接地址
	Token      string                    `yaml:"token" json:"token"`
	Socket     *gowebsocket.Socket       `yaml:"-" json:"-"`
	EchoMap    SyncMap[string, chan any] `yaml:"-" json:"-"`
	UserID     string                    `yaml:"-" json:"-"`

	Reconnecting bool `yaml:"-" json:"-"`
	RetryTimes   int  `yaml:"-" json:"-"`
}

func (pa *PlatformAdapterSealChat) Serve() int {
	if !strings.HasPrefix(pa.ConnectURL, "ws://") {
		pa.ConnectURL = "ws://" + pa.ConnectURL
	}
	socket := gowebsocket.New(pa.ConnectURL)
	pa.Socket = &socket
	pa.EndPoint.Nickname = "SealChat Bot"
	pa.EndPoint.UserID = "SEALCHAT:BOT"
	d := pa.Session.Parent
	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
	pa.socketSetup()
	socket.Connect()
	return 0
}

func (pa *PlatformAdapterSealChat) _sendJSON(socket *gowebsocket.Socket, data any) bool {
	// TODO: 修改上游代码，使其支持发送 JSON
	marshal, err := json.Marshal(data)
	if err != nil {
		return false
	}
	socket.SendText(string(marshal))
	return true
}

func (pa *PlatformAdapterSealChat) socketSetup() {
	ep := pa.EndPoint
	log := pa.Session.Parent.Logger
	socket := pa.Socket
	socket.OnConnected = func(socket gowebsocket.Socket) {
		pa.Reconnecting = true
		ep.State = 2
		ep.Enable = true
		pa.RetryTimes = 0

		d := pa.Session.Parent
		d.LastUpdatedTime = time.Now().Unix()
		d.Save(false)

		pa._sendJSON(&socket, satori.GatewayPayloadStructure{
			Op: satori.OpIdentify,
			Body: map[string]string{
				"token": pa.Token,
			},
		})

		log.Info("SealChat 已连接，正在发送身份验证信息")
		pa.Reconnecting = false
	}
	socket.OnTextMessage = func(message string, socket gowebsocket.Socket) {
		gatewayMsg := satori.GatewayPayloadStructure2{}
		err := json.Unmarshal([]byte(message), &gatewayMsg)
		if len(message) == 0 {
			return
		}
		// fmt.Printf("XXXXX: %s\n", message)

		solved := false
		if err == nil && gatewayMsg.Op != nil {
			switch *gatewayMsg.Op {
			case satori.OpReady:
				info := gatewayMsg.Body.(map[string]any)
				if info["errorMsg"] != nil {
					log.Infof("SealChat 连接失败: %s", info["errorMsg"])
					ep.State = 3
				} else {
					data := struct {
						Body struct {
							User satori.User `json:"user"`
						} `json:"body"`
					}{}
					err = json.Unmarshal([]byte(message), &data)
					if err != nil {
						log.Errorf("SealChat 解析用户信息失败: %s", err)
					} else {
						pa.UserID = data.Body.User.ID
						ep.UserID = FormatDiceIDSealChat(data.Body.User.ID)
						ep.Nickname = data.Body.User.Nick
						ep.State = 1
						log.Infof("SealChat 连接成功: %s", ep.Nickname)
					}

					go func() {
						// 等一会再发，因为好像有的模块会在这个事件之后注册指令
						time.Sleep(time.Duration(5) * time.Second)
						pa.registerCommands()
					}()
				}
				// TODO: 心跳
				solved = true
			case satori.OpEvent:
				pa.dispatchMessage(message)
				solved = true
			default:
				log.Infof("SealChat: %s", message)
			}
		}

		if solved {
			return
		}

		apiMsg := satori.ScApiMsgPayload{}
		err = json.Unmarshal([]byte(message), &apiMsg)
		if err == nil {
			if x, ok := pa.EchoMap.Load(apiMsg.Echo); ok {
				x <- apiMsg.Data
			}
		}
	}
	socket.OnConnectError = func(err error, socket gowebsocket.Socket) {
		log.Errorf("SealChat websocket出现错误: %s", err)
		if !socket.IsConnected {
			// socket.Close()
			if !pa.tryReconnect(*pa.Socket) {
				log.Errorf("短时间内连接失败次数过多，不再进行重连")
				ep.State = 3
			}
		}
	}
	socket.OnDisconnected = func(err error, socket gowebsocket.Socket) {
		log.Errorf("与SealChat服务器断开连接")
		time.Sleep(time.Duration(2) * time.Second)
		if !pa.tryReconnect(*pa.Socket) {
			log.Errorf("尝试进行重连")
			ep.State = 3
		}
	}
	pa.Socket = socket
}

func (pa *PlatformAdapterSealChat) tryReconnect(socket gowebsocket.Socket) bool {
	log := pa.Session.Parent.Logger
	if pa.Reconnecting {
		return false
	}
	pa.Reconnecting = true
	pa.RetryTimes = 0
	allTimes := 500
	for pa.RetryTimes <= allTimes && !socket.IsConnected {
		pa.RetryTimes++
		log.Infof("尝试重新连接SealChat中[%d/%d]", pa.RetryTimes, allTimes)
		socket = gowebsocket.New(pa.ConnectURL)
		pa.Socket = &socket
		pa.socketSetup()
		socket.Connect()
		time.Sleep(time.Duration(10) * time.Second)
	}
	return true
}

func (pa *PlatformAdapterSealChat) GetGroupInfoAsync(_ string) {}

func FormatDiceIDSealChat(id string) string {
	return fmt.Sprintf("SEALCHAT:%s", id)
}

func FormatDiceIDSealChatPrivate(id string) string {
	return fmt.Sprintf("PG-SEALCHAT:%s", id)
}

func FormatDiceIDSealChatGroup(id string) string {
	return fmt.Sprintf("SEALCHAT-Group:%s", id)
}

func (pa *PlatformAdapterSealChat) DoRelogin() bool {
	log := pa.Session.Parent.Logger
	pa.Reconnecting = true
	if pa.Socket != nil {
		pa.Socket.Close()
	}

	socket := gowebsocket.New(pa.ConnectURL)
	log.Infof("SealChat 重新连接")
	pa.Socket = &socket
	pa.socketSetup()
	socket.Connect()
	pa.Reconnecting = false
	return true
}

func (pa *PlatformAdapterSealChat) SetEnable(enable bool) {
	log := pa.Session.Parent.Logger
	if enable {
		log.Infof("Sealchat 连接中")
		if pa.Socket != nil && pa.Socket.IsConnected {
			pa.Reconnecting = true
			pa.Socket.Close()
			socket := gowebsocket.New(pa.ConnectURL)
			pa.Socket = &socket
			pa.socketSetup()
			socket.Connect()
			pa.Reconnecting = false
		} else {
			pa.Reconnecting = true
			socket := gowebsocket.New(pa.ConnectURL)
			pa.Socket = &socket
			pa.socketSetup()
			socket.Connect()
			pa.Reconnecting = false
		}
	} else {
		pa.Reconnecting = true
		if pa.Socket != nil && pa.Socket.IsConnected {
			pa.Socket.Close()
		}
		pa.Reconnecting = false
	}
}

func (pa *PlatformAdapterSealChat) sendAPI(api string, data any) chan any {
	echo := gonanoid.Must()
	ch := make(chan any, 1)
	pa.EchoMap.Store(echo, ch)
	pa._sendJSON(pa.Socket, &satori.ScApiMsgPayload{
		Api:  api,
		Echo: echo,
		Data: data,
	})
	return ch
}

func ExtractSealChatPrivateChatID(id string, userId string) string {
	if strings.HasPrefix(id, "SEALCHAT:") {
		id1 := id[len("SEALCHAT:"):]
		id2 := userId[len("SEALCHAT:"):]
		if id1 > id2 {
			id1, id2 = id2, id1
		}
		return fmt.Sprintf("%s:%s", id1, id2)
	}
	return id
}

func ExtractSealChatUserID(id string) string {
	if strings.HasPrefix(id, "SEALCHAT:") {
		return id[len("SEALCHAT:"):]
	}
	if strings.HasPrefix(id, "SEALCHAT-Group:") {
		return id[len("SEALCHAT-Group:"):]
	}
	return id
}

func (pa *PlatformAdapterSealChat) _sendTo(ctx *MsgContext, chId string, text string, flag string, msgType string) {
	msg := new(SendMessageMinecraft)
	msg.Content = text
	msg.MessageType = "group"
	parse, _ := json.Marshal(msg)

	pa.sendAPI("message.create", map[string]any{
		"channel_id": chId,
		"content":    text,
	})

	pa._sendJSON(pa.Socket, &satori.Message{})
	pa.Socket.SendText(string(parse))
	pa.Session.OnMessageSend(ctx, &Message{
		Platform:    "SEALCHAT",
		MessageType: msgType,
		Message:     text,
		GroupID:     chId,
		Sender: SenderBase{
			UserID:   pa.EndPoint.UserID,
			Nickname: pa.EndPoint.Nickname,
		},
	}, flag)
}

func (pa *PlatformAdapterSealChat) SendToPerson(ctx *MsgContext, uid string, text string, flag string) {
	<-pa.sendAPI("channel.private.create", map[string]string{
		"user_id": ExtractSealChatUserID(uid),
	})

	text = satori.ContentEscape(text)
	gid := ExtractSealChatPrivateChatID(uid, pa.EndPoint.UserID)
	pa._sendTo(ctx, gid, text, flag, "private")
}

func (pa *PlatformAdapterSealChat) SendToGroup(ctx *MsgContext, uid string, text string, flag string) {
	chId := ExtractSealChatUserID(uid)
	text = satori.ContentEscape(text)
	pa._sendTo(ctx, chId, text, flag, "group")
}

func (pa *PlatformAdapterSealChat) SendFileToPerson(ctx *MsgContext, uid string, path string, flag string) {
	dice := pa.Session.Parent
	fileElement, err := dice.FilepathToFileElement(path)
	if err == nil {
		pa.SendToPerson(ctx, uid, fmt.Sprintf("[尝试发送文件: %s，但不支持]", fileElement.File), flag)
	} else {
		pa.SendToPerson(ctx, uid, fmt.Sprintf("[尝试发送文件出错: %s]", err.Error()), flag)
	}
}

func (pa *PlatformAdapterSealChat) SendFileToGroup(ctx *MsgContext, uid string, path string, flag string) {
	dice := pa.Session.Parent
	fileElement, err := dice.FilepathToFileElement(path)
	if err == nil {
		pa.SendToGroup(ctx, uid, fmt.Sprintf("[尝试发送文件: %s，但不支持]", fileElement.File), flag)
	} else {
		pa.SendToGroup(ctx, uid, fmt.Sprintf("[尝试发送文件出错: %s]", err.Error()), flag)
	}
}

func (pa *PlatformAdapterSealChat) MemberBan(_ string, _ string, _ int64) {}

func (pa *PlatformAdapterSealChat) MemberKick(_ string, _ string) {}

func (pa *PlatformAdapterSealChat) QuitGroup(_ *MsgContext, _ string) {}

func (pa *PlatformAdapterSealChat) SetGroupCardName(mctx *MsgContext, text string) {
	pa.sendAPI("bot.channel_member.set_name", map[string]string{
		"user_id":    ExtractSealChatUserID(mctx.Player.UserID),
		"channel_id": ExtractSealChatUserID(mctx.Group.GroupID),
		"name":       text,
	})
}

func (pa *PlatformAdapterSealChat) dispatchMessage(msg string) {
	ev := satori.Event{}
	err := json.Unmarshal([]byte(msg), &ev)
	if err != nil {
		fmt.Println(err)
		return
	}

	switch ev.Type {
	case satori.EventMessageCreated:
		if ev.Message.User.ID == pa.UserID {
			// 自己发的消息，不管
			return
		}
		pa.Session.Execute(pa.EndPoint, pa.toStdMessage(ev.Message), false)
		return
	case satori.EventMessageDeleted:
		stdMsg := pa.toStdMessage(ev.Message)
		mctx := CreateTempCtx(pa.EndPoint, stdMsg)
		pa.Session.OnMessageDeleted(mctx, stdMsg)
		return
	default:
		// fmt.Println("msg", ev.Type, "|", ev)
	}
}

func (pa *PlatformAdapterSealChat) toStdMessage(scMsg *satori.Message) *Message {
	msg := new(Message)

	elRoot := satori.ElementParse(scMsg.Content)
	msg.Time = scMsg.Timestamp

	// TODO: 这里会有一个很怪的行为，也就是结构化数据转cq码，以后改掉

	cqMsg := strings.Builder{}
	elRoot.Traverse(func(el *satori.Element) {
		switch el.Type {
		case "at":
			if el.Attrs["role"] != "all" {
				cqMsg.WriteString(fmt.Sprintf("<@%s>", el.Attrs["id"]))
			}
		case "root":
			// 啥都不做
		default:
			cqMsg.WriteString(el.ToString())
		}
	})

	msg.Message = strings.TrimSpace(cqMsg.String())

	msg.Platform = "SEALCHAT"
	if scMsg.Channel.Type == satori.DirectChannelType {
		msg.MessageType = "private"
		msg.GroupID = FormatDiceIDSealChatPrivate(scMsg.Channel.ID)
	} else {
		msg.MessageType = "group"
		msg.GroupID = FormatDiceIDSealChatGroup(scMsg.Channel.ID)
		msg.GroupName = scMsg.Channel.Name
	}

	msg.RawID = scMsg.ID
	send := new(SenderBase)
	send.UserID = FormatDiceIDSealChat(scMsg.User.ID)

	if scMsg.Member != nil {
		// 注: 部分消息，比如message-deleted没有member
		send.Nickname = scMsg.Member.Nick
	}
	if send.Nickname == "" && scMsg.User != nil {
		send.Nickname = scMsg.User.Nick
	}

	if send.Nickname == "" {
		send.Nickname = fmt.Sprintf("用户%4s", scMsg.Channel.ID)
	}
	// fmt.Println("!!!", scMsg.Member, "|", scMsg.User.Name)
	// if msgMC.Event.IsAdmin {
	//	send.GroupRole = "admin"
	// }
	msg.Sender = *send
	return msg
}

func (pa *PlatformAdapterSealChat) registerCommands() {
	cmdMap := pa.EndPoint.Session.Parent.CmdMap
	m := map[string]string{}
	for k, v := range cmdMap {
		// fmt.Println("??", k, v)
		m[k] = v.ShortHelp
	}

	for _, i := range pa.EndPoint.Session.Parent.ExtList {
		for k, v := range i.CmdMap {
			// fmt.Println("??", k, v)
			m[k] = v.ShortHelp
		}
	}
	mctx := CreateTempCtx(pa.EndPoint, &Message{
		MessageType: "group",
		Sender:      SenderBase{UserID: pa.EndPoint.UserID},
	})
	pa.sendAPI("bot.info.set_name", map[string]string{"name": DiceFormatTmpl(mctx, "核心:骰子名字")})
	pa.sendAPI("bot.command.register", m)
}

func ServeSealChat(d *Dice, ep *EndPointInfo) {
	defer CrashLog()
	if ep.Platform == "SEALCHAT" {
		conn := ep.Adapter.(*PlatformAdapterSealChat)
		d.Logger.Infof("SealChat 尝试连接")
		if conn.Serve() != 0 {
			d.Logger.Errorf("连接SealChat服务失败")
			ep.State = 3
			d.LastUpdatedTime = time.Now().Unix()
			d.Save(false)
		}
	}
}

func NewSealChatConnItem(url string, token string) *EndPointInfo {
	conn := new(EndPointInfo)
	conn.ID = uuid.New().String()
	conn.Platform = "SEALCHAT"
	conn.ProtocolType = ""
	conn.Enable = true
	conn.RelWorkDir = "extra/sealchat-" + conn.ID
	conn.Adapter = &PlatformAdapterSealChat{
		EndPoint:   conn,
		ConnectURL: url,
		Token:      token,
	}
	return conn
}
