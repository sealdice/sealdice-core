package dice

import (
	"encoding/json"
	"fmt"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/sacOO7/gowebsocket"
	"sealdice-core/utils/satori"
	"strings"
	"time"
)

type PlatformAdapterSealChat struct {
	Session  *IMSession    `yaml:"-" json:"-"`
	EndPoint *EndPointInfo `yaml:"-" json:"-"`

	ConnectURL string               `yaml:"connectUrl" json:"connectUrl"` // 连接地址
	Token      string               `yaml:"token" json:"token"`
	Socket     *gowebsocket.Socket  `yaml:"-" json:"-"`
	EchoMap    SyncMap[string, any] `yaml:"-" json:"-"`
	UserID     string               `yaml:"-" json:"-"`

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
	pa.EndPoint.UserID = "WebSocket"
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
		ep.State = 1
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
		gatewayMsg := satori.GatewayPayloadStructure{}
		err := json.Unmarshal([]byte(message), &gatewayMsg)

		solved := false
		if err == nil {
			switch gatewayMsg.Op {
			case satori.OpReady:
				info := gatewayMsg.Body.(map[string]any)
				if info["errorMsg"] != nil {
					log.Infof("SealChat 连接失败: %s", info["errorMsg"])
				} else {
					data := struct {
						Body struct {
							User satori.User `json:"user"`
						} `json:"body"`
					}{}
					err := json.Unmarshal([]byte(message), &data)
					if err != nil {
						log.Errorf("SealChat 解析用户信息失败: %s", err)
					} else {
						pa.UserID = data.Body.User.ID
						ep.UserID = data.Body.User.ID
						ep.Nickname = data.Body.User.Nick
						ep.State = 2
						log.Infof("SealChat 连接成功: %s", ep.Nickname)
					}

					pa.registerCommands()
				}
				// TODO: 心跳
				solved = true
			case satori.OpEvent:
				pa.dispatchMessage(message)
				solved = true
			}
		}

		if solved {
			return
		}

		log.Infof("SealChat: %s %d", message, solved)

		//fmt.Println("SealChat: " + message)
		//msgMC := new(MessageMinecraft)
		//err := json.Unmarshal([]byte(message), msgMC)
		//if err == nil {
		//	pa.Session.Execute(ep, pa.toStdMessage(msgMC), false)
		//}
	}
	socket.OnConnectError = func(err error, socket gowebsocket.Socket) {
		log.Errorf("SealChat websocket出现错误: %s", err)
		if !socket.IsConnected {
			// socket.Close()
			if !pa.tryReconnect(*pa.Socket) {
				log.Errorf("短时间内连接失败次数过多，不再进行重连")
				ep.State = 3
				ep.Enable = false
			}
		}
	}
	socket.OnDisconnected = func(err error, socket gowebsocket.Socket) {
		log.Errorf("与SealChat服务器断开连接")
		time.Sleep(time.Duration(2) * time.Second)
		if !pa.tryReconnect(*pa.Socket) {
			log.Errorf("短时间内连接失败次数过多，不再进行重连")
			ep.State = 3
			ep.Enable = false
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
	if pa.RetryTimes <= 5 && !socket.IsConnected {
		pa.RetryTimes++
		log.Infof("尝试重新连接SealChat中[%d/5]", pa.RetryTimes)
		socket = gowebsocket.New(pa.ConnectURL)
		pa.Socket = &socket
		pa.socketSetup()
		socket.Connect()
	}
	return true
}

func (pa *PlatformAdapterSealChat) GetGroupInfoAsync(_ string) {}

func FormatDiceIDSealChat(id string) string {
	return fmt.Sprintf("SEALCHAT:%s", id)
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

func (pa *PlatformAdapterSealChat) SendToPerson(ctx *MsgContext, uid string, text string, flag string) {
	id := ExtractMCUserID(uid)
	msg := new(SendMessageMinecraft)
	msg.UUID = id
	msg.Content = text
	msg.MessageType = "private"
	parse, _ := json.Marshal(msg)
	pa.Socket.SendText(string(parse))
	pa.Session.OnMessageSend(ctx, &Message{
		Platform:    "SEALCHAT",
		MessageType: "private",
		Message:     text,
		Sender: SenderBase{
			UserID:   pa.EndPoint.UserID,
			Nickname: pa.EndPoint.Nickname,
		},
	}, flag)
}

func (pa *PlatformAdapterSealChat) sendAPI(api string, data any) {
	echo := gonanoid.Must()
	// 等下，这个好像没必要
	pa.EchoMap.Store(echo, "")
	pa._sendJSON(pa.Socket, &satori.ScApiMsgPayload{
		Api:  api,
		Echo: echo,
		Data: data,
	})
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

func (pa *PlatformAdapterSealChat) SendToGroup(ctx *MsgContext, uid string, text string, flag string) {
	msg := new(SendMessageMinecraft)
	msg.Content = text
	msg.MessageType = "group"
	parse, _ := json.Marshal(msg)

	chId := ExtractSealChatUserID(uid)
	pa.sendAPI("message.create", map[string]any{
		"channel_id": chId,
		"content":    text,
	})

	pa._sendJSON(pa.Socket, &satori.Message{})
	pa.Socket.SendText(string(parse))
	pa.Session.OnMessageSend(ctx, &Message{
		Platform:    "SEALCHAT",
		MessageType: "group",
		Message:     text,
		GroupID:     uid,
		Sender: SenderBase{
			UserID:   pa.EndPoint.UserID,
			Nickname: pa.EndPoint.Nickname,
		},
	}, flag)
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

func (pa *PlatformAdapterSealChat) SetGroupCardName(_ *MsgContext, _ string) {}

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
	}
	//fmt.Println("!!!", msg)
	fmt.Println("msg", ev.Type, ev)
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
			break
		case "root":
			// 啥都不做
			break
		default:
			cqMsg.WriteString(el.ToString())
		}
	})

	msg.Message = strings.TrimSpace(cqMsg.String())

	msg.Platform = "SEALCHAT"
	msg.MessageType = "group"
	if msg.MessageType == "group" {
		msg.GroupID = FormatDiceIDSealChatGroup(scMsg.Channel.ID)
	}
	send := new(SenderBase)
	send.UserID = FormatDiceIDSealChat(scMsg.User.ID)
	// send.Nickname = scMsg.Member.Nick
	send.Nickname = "用户"
	fmt.Println("!!!", scMsg.Member, "|", scMsg.User.Name)
	//if msgMC.Event.IsAdmin {
	//	send.GroupRole = "admin"
	//}
	msg.Sender = *send
	return msg
}

func (pa *PlatformAdapterSealChat) registerCommands() {
	cmdMap := pa.EndPoint.Session.Parent.CmdMap
	m := map[string]string{}
	for k, v := range cmdMap {
		//fmt.Println("??", k, v)
		m[k] = v.ShortHelp
	}
	pa.sendAPI("command.register", m)
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
