package dice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Szzrain/dodo-open-go/client"
	"github.com/Szzrain/dodo-open-go/model"
	"github.com/Szzrain/dodo-open-go/websocket"
	"io"
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

func (pa *PlatformAdapterDodo) GetGroupInfoAsync(groupId string) {
	info, err := pa.Client.GetChannelInfo(context.Background(), &model.GetChannelInfoReq{
		ChannelId: ExtractDodoGroupId(groupId),
	})
	if err != nil {
		return
	}
	dm := pa.Session.Parent.Parent
	dm.GroupNameCache.Set(groupId, &GroupNameCacheItem{
		Name: info.ChannelName,
		time: time.Now().Unix(),
	})
	group := pa.Session.ServiceAtNew[groupId]
	if group != nil {
		group.GroupName = info.ChannelName
	}
}

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
	personalMessageHandler := func(event *websocket.WSEventMessage, data *websocket.PersonalMessageEventBody) error {
		msg, err := pa.toStdPersonalMessage(data)
		if err != nil {
			return err
		}
		pa.Session.Execute(pa.EndPoint, msg, false)
		return nil
	}
	msgHandlers.ChannelMessage = channelMessageHandler
	msgHandlers.PersonalMessage = personalMessageHandler

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

func (pa *PlatformAdapterDodo) toStdPersonalMessage(msgRaw *websocket.PersonalMessageEventBody) (*Message, error) {
	msg := &Message{}
	msg.MessageType = "private"
	msg.Time = time.Now().Unix()
	msg.Platform = "DODO"
	send := new(SenderBase)
	send.Nickname = msgRaw.Personal.NickName
	send.UserId = FormatDiceIdDodo(msgRaw.DodoSourceId)
	msg.Sender = *send
	if msgRaw.IslandSourceId != "" {
		msg.GuildId = msgRaw.IslandSourceId
	}
	//pa.Session.Parent.Logger.Infof("source id: %s", msgRaw.IslandSourceId)
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
	msg.GuildId = msgRaw.IslandSourceId
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
	//pa.Session.Parent.Logger.Infof("send to %s", ExtractDodoUserId(uid))
	err := pa.SendToChatRaw(ctx, uid, text, true)
	if err != nil {
		pa.Session.Parent.Logger.Errorf("DODO 发送私聊消息失败：%v\n", err)
		return
	}
	pa.Session.OnMessageSend(ctx, "private", uid, text, flag)
}

func (pa *PlatformAdapterDodo) SendToGroup(ctx *MsgContext, uid string, text string, flag string) {
	err := pa.SendToChatRaw(ctx, uid, text, false)
	if err != nil {
		pa.Session.Parent.Logger.Errorf("DODO 发送消息失败：%v\n", err)
		return
	}
	pa.Session.OnMessageSend(ctx, "group", uid, text, flag)
}

func (pa *PlatformAdapterDodo) SendToChatRaw(ctx *MsgContext, uid string, text string, isPrivate bool) error {
	instance := pa.Client
	dice := pa.Session.Parent
	elem := dice.ConvertStringMessage(text)
	StreamToByte := func(stream io.Reader) []byte {
		buf := new(bytes.Buffer)
		_, err := buf.ReadFrom(stream)
		if err != nil {
			return nil
		}
		return buf.Bytes()
	}
	for _, element := range elem {
		switch e := element.(type) {
		case *TextElement:
			err := pa.SendMessageRaw(ctx, &model.TextMessage{Content: e.Content}, uid, isPrivate)
			if err != nil {
				return err
			}
		case *ImageElement:
			resourceResp, err := instance.UploadImageByBytes(context.Background(), &model.UploadImageByBytesReq{
				Filename: e.file.File,
				Bytes:    StreamToByte(e.file.Stream),
			})
			if err != nil {
				return err
			}
			msgBody := &model.ImageMessage{
				Url:        resourceResp.Url,
				Width:      resourceResp.Width,
				Height:     resourceResp.Height,
				IsOriginal: 0,
			}
			err = pa.SendMessageRaw(ctx, msgBody, uid, isPrivate)
			if err != nil {
				return err
			}
		case *AtElement:
			err := pa.SendMessageRaw(ctx, &model.TextMessage{Content: fmt.Sprintf("<@!%s>", e.Target)}, uid, isPrivate)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (pa *PlatformAdapterDodo) SendMessageRaw(ctx *MsgContext, msgBody model.IMessageBody, uid string, isPrivate bool) error {
	var rawId string
	if isPrivate {
		rawId = ExtractDodoUserId(uid)
		_, err := pa.Client.SendDirectMessage(context.Background(), &model.SendDirectMessageReq{
			IslandSourceId: ctx.Group.GuildId,
			DodoSourceId:   rawId,
			MessageBody:    msgBody,
		})
		return err
	} else {
		rawId = ExtractDodoGroupId(uid)
		_, err := pa.Client.SendChannelMessage(context.Background(), &model.SendChannelMessageReq{
			ChannelId:   ExtractDodoGroupId(uid),
			MessageBody: msgBody,
		})
		return err
	}
}

func (pa *PlatformAdapterDodo) MemberBan(groupId string, userId string, last int64) {

}

func (pa *PlatformAdapterDodo) MemberKick(groupId string, userId string) {

}

func FormatDiceIdDodo(sourceid string) string {
	return fmt.Sprintf("DODO:%s", sourceid)
}

func FormatDiceIdDodoGroup(channelid string) string {
	return fmt.Sprintf("DODO-Group:%s", channelid)
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
