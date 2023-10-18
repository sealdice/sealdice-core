package dice

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sacOO7/gowebsocket"
)

type PlatformAdapterMinecraft struct {
	Session      *IMSession          `yaml:"-" json:"-"`
	EndPoint     *EndPointInfo       `yaml:"-" json:"-"`
	Socket       *gowebsocket.Socket `yaml:"-" json:"-"`
	RetryTimes   int                 `yaml:"-" json:"-"`
	Reconnecting bool                `yaml:"-" json:"-"`
	ConnectURL   string              `yaml:"connectUrl" json:"connectUrl"` // 连接地址
}

type MessageMinecraft struct {
	Event *MessageEvent `json:"event"`
	Type  string        `json:"type"`
}

type MessageEvent struct {
	Content     string `json:"content"`
	IsAdmin     bool   `json:"isAdmin"`
	Name        string `json:"name"`
	UUID        string `json:"uuid"`
	MessageType string `json:"messageType"`
}

type SendMessageMinecraft struct {
	Content     string `json:"content"`
	Name        string `json:"name"`
	UUID        string `json:"uuid"`
	MessageType string `json:"messageType"`
}

func (pa *PlatformAdapterMinecraft) GetGroupInfoAsync(_ string) {}

func (pa *PlatformAdapterMinecraft) Serve() int {
	if !strings.HasPrefix(pa.ConnectURL, "ws://") {
		pa.ConnectURL = "ws://" + pa.ConnectURL
	}
	socket := gowebsocket.New(pa.ConnectURL)
	pa.Socket = &socket
	pa.EndPoint.Nickname = "A Minecraft Server"
	pa.EndPoint.UserID = "WebSocket"
	d := pa.Session.Parent
	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
	pa.socketSetup()
	socket.Connect()
	return 0
}

func (pa *PlatformAdapterMinecraft) tryReconnect(socket gowebsocket.Socket) bool {
	log := pa.Session.Parent.Logger
	if pa.Reconnecting {
		return false
	}
	pa.Reconnecting = true
	if pa.RetryTimes <= 5 && !socket.IsConnected {
		pa.RetryTimes++
		log.Infof("MC server 尝试重新连接中[%d/5]", pa.RetryTimes)
		socket = gowebsocket.New(pa.ConnectURL)
		pa.Socket = &socket
		pa.socketSetup()
		socket.Connect()
	}
	return true
}

func (pa *PlatformAdapterMinecraft) socketSetup() {
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

		log.Info("Minecraft 连接成功")
		time.Sleep(time.Duration(5) * time.Second)
		pa.Reconnecting = false
	}
	socket.OnTextMessage = func(message string, socket gowebsocket.Socket) {
		msgMC := new(MessageMinecraft)
		err := json.Unmarshal([]byte(message), msgMC)
		if err == nil {
			pa.Session.Execute(ep, pa.toStdMessage(msgMC), false)
		}
	}
	socket.OnConnectError = func(err error, socket gowebsocket.Socket) {
		log.Errorf("MC websocket出现错误: %s", err)
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
		log.Errorf("与MC服务器断开连接")
		time.Sleep(time.Duration(2) * time.Second)
		if !pa.tryReconnect(*pa.Socket) {
			log.Errorf("短时间内连接失败次数过多，不再进行重连")
			ep.State = 3
			ep.Enable = false
		}
	}
	pa.Socket = socket
}

func (pa *PlatformAdapterMinecraft) toStdMessage(msgMC *MessageMinecraft) *Message {
	msg := new(Message)
	msg.Time = time.Now().Unix()
	msg.Message = msgMC.Event.Content
	msg.Platform = "MC"
	msg.MessageType = msgMC.Event.MessageType
	if msg.MessageType == "group" {
		msg.GroupID = "server"
	}
	send := new(SenderBase)
	send.UserID = FormatDiceIDMC(msgMC.Event.UUID)
	send.Nickname = msgMC.Event.Name
	if msgMC.Event.IsAdmin {
		send.GroupRole = "admin"
	}
	msg.Sender = *send
	return msg
}

func FormatDiceIDMC(diceMC string) string {
	return fmt.Sprintf("MC:%s", diceMC)
}

func ExtractMCUserID(id string) string {
	if strings.HasPrefix(id, "MC:") {
		return id[len("MC:"):]
	}
	return id
}

func (pa *PlatformAdapterMinecraft) DoRelogin() bool {
	log := pa.Session.Parent.Logger
	pa.Reconnecting = true
	pa.Socket.Close()
	socket := gowebsocket.New(pa.ConnectURL)
	log.Infof("MC server 重新连接")
	pa.Socket = &socket
	pa.socketSetup()
	socket.Connect()
	pa.Reconnecting = false
	return true
}

func (pa *PlatformAdapterMinecraft) SetEnable(enable bool) {
	log := pa.Session.Parent.Logger
	if enable {
		log.Infof("MC server 连接中")
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

func (pa *PlatformAdapterMinecraft) SendToPerson(ctx *MsgContext, uid string, text string, flag string) {
	id := ExtractMCUserID(uid)
	msg := new(SendMessageMinecraft)
	msg.UUID = id
	msg.Content = text
	msg.MessageType = "private"
	parse, _ := json.Marshal(msg)
	pa.Socket.SendText(string(parse))
	pa.Session.OnMessageSend(ctx, &Message{
		Platform:    "MC",
		MessageType: "private",
		Message:     text,
		Sender: SenderBase{
			UserID:   pa.EndPoint.UserID,
			Nickname: pa.EndPoint.Nickname,
		},
	}, flag)
}

func (pa *PlatformAdapterMinecraft) SendToGroup(ctx *MsgContext, uid string, text string, flag string) {
	msg := new(SendMessageMinecraft)
	msg.Content = text
	msg.MessageType = "group"
	parse, _ := json.Marshal(msg)
	pa.Socket.SendText(string(parse))
	pa.Session.OnMessageSend(ctx, &Message{
		Platform:    "MC",
		MessageType: "group",
		Message:     text,
		GroupID:     uid,
		Sender: SenderBase{
			UserID:   pa.EndPoint.UserID,
			Nickname: pa.EndPoint.Nickname,
		},
	}, flag)
}

func (pa *PlatformAdapterMinecraft) SendFileToPerson(ctx *MsgContext, uid string, path string, flag string) {
	dice := pa.Session.Parent
	fileElement, err := dice.FilepathToFileElement(path)
	if err == nil {
		pa.SendToPerson(ctx, uid, fmt.Sprintf("[尝试发送文件: %s，但不支持]", fileElement.File), flag)
	} else {
		pa.SendToPerson(ctx, uid, fmt.Sprintf("[尝试发送文件出错: %s]", err.Error()), flag)
	}
}

func (pa *PlatformAdapterMinecraft) SendFileToGroup(ctx *MsgContext, uid string, path string, flag string) {
	dice := pa.Session.Parent
	fileElement, err := dice.FilepathToFileElement(path)
	if err == nil {
		pa.SendToGroup(ctx, uid, fmt.Sprintf("[尝试发送文件: %s，但不支持]", fileElement.File), flag)
	} else {
		pa.SendToGroup(ctx, uid, fmt.Sprintf("[尝试发送文件出错: %s]", err.Error()), flag)
	}
}

func (pa *PlatformAdapterMinecraft) MemberBan(_ string, _ string, _ int64) {}

func (pa *PlatformAdapterMinecraft) MemberKick(_ string, _ string) {}

func (pa *PlatformAdapterMinecraft) QuitGroup(_ *MsgContext, _ string) {}

func (pa *PlatformAdapterMinecraft) SetGroupCardName(_ string, _ string, _ string) {}
