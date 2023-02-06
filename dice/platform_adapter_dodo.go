package dice

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Szzrain/dodo-open-go/client"
	"github.com/Szzrain/dodo-open-go/model"
	"github.com/Szzrain/dodo-open-go/websocket"
	"strings"
	"time"
)

type PlatformAdapterDodo struct {
	Session   *IMSession       `yaml:"-" json:"-"`
	ClientID  string           `yaml:"clientID" json:"clientID"`
	Token     string           `yaml:"token" json:"token"`
	EndPoint  *EndPointInfo    `yaml:"-" json:"-"`
	Client    client.Client    `yaml:"-" json:"-"`
	WebSocket websocket.Client `yaml:"-" json:"-"`
}

func (pa *PlatformAdapterDodo) GetGroupInfoAsync(groupId string) {}

func (pa *PlatformAdapterDodo) Serve() int {
	logger := pa.Session.Parent.Logger
	clientId := pa.ClientID
	token := pa.Token

	instance, err := client.New(clientId, token, client.WithTimeout(time.Second*3))
	pa.Client = instance
	if err != nil {
		return 1
	}
	selfid, err := instance.GetBotInfo(context.Background())
	if err == nil {
		pa.EndPoint.UserId = FormatDiceIdDodo(selfid.DodoSourceId)
		pa.EndPoint.Nickname = selfid.NickName
	}
	msgHandlers := &websocket.MessageHandlers{}
	channelMessageHandler := func(event *websocket.WSEventMessage, data *websocket.ChannelMessageEventBody) error {
		//fmt.Printf("%v\n", data)
		msg, err := pa.toStdChannelMessage(data)
		if err != nil {
			return err
		}
		pa.Session.Execute(pa.EndPoint, msg, false)
		return nil
	}
	msgHandlers.ChannelMessage = channelMessageHandler

	ws, err := websocket.New(instance, websocket.WithMessageHandlers(msgHandlers))
	// 主动连接到 WebSocket 服务器
	if err = ws.Connect(); err != nil {
		return 1
	}
	go func() {
		err := ws.Listen()
		if err != nil {
			logger.Errorf("Dodo监听错误:%s", err.Error())
		}
	}()
	pa.WebSocket = ws
	pa.Session.Parent.Logger.Infof("Dodo 连接成功")
	pa.EndPoint.Enable = true
	pa.EndPoint.State = 1
	return 0
}

func (pa *PlatformAdapterDodo) toStdChannelMessage(msgRaw *websocket.ChannelMessageEventBody) (*Message, error) {
	msg := &Message{}
	msg.MessageType = "group"
	msg.Time = time.Now().Unix()
	msg.Platform = "DODO"
	send := new(SenderBase)
	send.Nickname = msgRaw.Member.NickName
	send.UserId = FormatDiceIdDodo(msgRaw.DodoSourceId)
	msg.Sender = *send
	msg.GroupId = FormatDiceIdDodoGroup(msgRaw.ChannelId)
	switch msgRaw.MessageType {
	case 1:
		msgDodo := new(model.TextMessage)
		err := json.Unmarshal(msgRaw.MessageBody, msgDodo)
		if err == nil {
			msg.Message = msgDodo.Content
		} else {
			return nil, err
		}
	}
	return msg, nil
}

func (pa *PlatformAdapterDodo) DoRelogin() bool {
	defer func() {
		if recover() != nil {
		}
	}()
	logger := pa.Session.Parent.Logger
	if pa.WebSocket != nil {
		pa.WebSocket.Close()
		pa.Client = nil
		pa.WebSocket = nil
		pa.EndPoint.State = 0
		pa.EndPoint.Enable = false
	}
	logger.Infof("正在启用Dodo……")
	pa.Serve()
	return false
}

func (pa *PlatformAdapterDodo) SetEnable(enable bool) {
	defer func() {
		if recover() != nil {
		}
	}()
	logger := pa.Session.Parent.Logger
	if enable {
		if pa.Client != nil && pa.WebSocket != nil {
			pa.WebSocket.Close()
			pa.Client = nil
			pa.WebSocket = nil
			pa.EndPoint.State = 0
			pa.EndPoint.Enable = false
		}
		logger.Infof("正在启用Dodo……")
		pa.Serve()
	} else {
		pa.WebSocket.Close()
		pa.Client = nil
		pa.WebSocket = nil
		pa.EndPoint.State = 0
		pa.EndPoint.Enable = false
	}
}

func (pa *PlatformAdapterDodo) SendToPerson(ctx *MsgContext, uid string, text string, flag string) {
}

func (pa *PlatformAdapterDodo) SendToGroup(ctx *MsgContext, uid string, text string, flag string) {
	_, err := pa.Client.SendChannelMessage(context.Background(), &model.SendChannelMessageReq{
		ChannelId:   ExtractDodoGroupId(uid),
		MessageBody: &model.TextMessage{Content: text},
	})
	if err != nil {
		pa.Session.Parent.Logger.Errorf("发送消息失败：%v\n", err)
		return
	}
}
func FormatDiceIdDodo(diceDodo string) string {
	return fmt.Sprintf("DODO:%s", diceDodo)
}

func FormatDiceIdDodoGroup(diceDodo string) string {
	return fmt.Sprintf("DODO-Group:%s", diceDodo)
}

func ExtractDodoUserId(id string) string {
	if strings.HasPrefix(id, "DODO:") {
		return id[len("DODO:"):]
	}
	return id
}

func ExtractDodoGroupId(id string) string {
	if strings.HasPrefix(id, "DODO-Group:") {
		return id[len("DODO-Group:"):]
	}
	return id
}
func (pa *PlatformAdapterDodo) QuitGroup(ctx *MsgContext, id string) {}

func (pa *PlatformAdapterDodo) SetGroupCardName(groupId string, userId string, name string) {}
