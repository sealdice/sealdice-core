package dice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Szzrain/dodo-open-go/client"
	"github.com/Szzrain/dodo-open-go/model"
	"github.com/Szzrain/dodo-open-go/websocket"
)

type PlatformAdapterDodo struct {
	Session   *IMSession       `yaml:"-" json:"-"`
	ClientID  string           `yaml:"clientID" json:"clientID"`
	Token     string           `yaml:"token" json:"token"`
	EndPoint  *EndPointInfo    `yaml:"-" json:"-"`
	Client    client.Client    `yaml:"-" json:"-"`
	WebSocket websocket.Client `yaml:"-" json:"-"`
}

func (pa *PlatformAdapterDodo) GetGroupInfoAsync(groupID string) {
	info, err := pa.Client.GetChannelInfo(context.Background(), &model.GetChannelInfoReq{
		ChannelId: ExtractDodoGroupID(groupID),
	})
	if err != nil {
		return
	}
	dm := pa.Session.Parent.Parent
	dm.GroupNameCache.Set(groupID, &GroupNameCacheItem{
		Name: info.ChannelName,
		time: time.Now().Unix(),
	})
	group := pa.Session.ServiceAtNew[groupID]
	if group != nil {
		group.GroupName = info.ChannelName
	}
}

func (pa *PlatformAdapterDodo) Serve() int {
	logger := pa.Session.Parent.Logger
	clientID := pa.ClientID
	token := pa.Token

	instance, err := client.New(clientID, token, client.WithTimeout(time.Second*3))
	pa.Client = instance
	if err != nil {
		return 1
	}
	selfid, err := instance.GetBotInfo(context.Background())
	if err == nil {
		pa.EndPoint.UserID = FormatDiceIDDodo(selfid.DodoSourceId)
		pa.EndPoint.Nickname = selfid.NickName
		d := pa.Session.Parent
		d.LastUpdatedTime = time.Now().Unix()
		d.Save(false)
	}
	msgHandlers := &websocket.MessageHandlers{}
	channelMessageHandler := func(event *websocket.WSEventMessage, data *websocket.ChannelMessageEventBody) error {
		// defer func() {
		// 	if recoverError := recover(); recoverError != nil {
		// 		pa.Session.Parent.Logger.Errorf("Dodo消息处理错误:%v\n Stack: \n%v", recoverError, string(debug.Stack()))
		// 	}
		// }()
		// pa.Session.Parent.Logger.Infof("WS-收到Dodo频道消息:%s", string(data.MessageBody))
		msg, errs := pa.toStdChannelMessage(data)
		if errs != nil {
			pa.Session.Parent.Logger.Errorf("Dodo消息转换错误:%s", err.Error())
			return errs
		}
		pa.Session.Execute(pa.EndPoint, msg, false)
		return nil
	}
	personalMessageHandler := func(event *websocket.WSEventMessage, data *websocket.PersonalMessageEventBody) error {
		msg, errs := pa.toStdPersonalMessage(data)
		if errs != nil {
			pa.Session.Parent.Logger.Errorf("Dodo消息转换错误:%s", err.Error())
			return errs
		}
		pa.Session.Execute(pa.EndPoint, msg, false)
		return nil
	}
	msgHandlers.ChannelMessage = channelMessageHandler
	msgHandlers.PersonalMessage = personalMessageHandler

	ws, _ := websocket.New(instance, websocket.WithMessageHandlers(msgHandlers))
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
	d := pa.Session.Parent
	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
	return 0
}

func (pa *PlatformAdapterDodo) toStdPersonalMessage(msgRaw *websocket.PersonalMessageEventBody) (*Message, error) {
	msg := &Message{}
	msg.MessageType = "private"
	msg.Time = time.Now().Unix()
	msg.Platform = "DODO"
	send := new(SenderBase)
	send.Nickname = msgRaw.Personal.NickName
	send.UserID = FormatDiceIDDodo(msgRaw.DodoSourceId)
	msg.Sender = *send
	if msgRaw.IslandSourceId != "" {
		msg.GuildID = msgRaw.IslandSourceId
	}
	// pa.Session.Parent.Logger.Infof("source id: %s", msgRaw.IslandSourceId)
	if msgRaw.MessageType == 1 {
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
	msg.RawID = msgRaw.MessageId
	send := new(SenderBase)
	send.Nickname = msgRaw.Member.NickName
	send.UserID = FormatDiceIDDodo(msgRaw.DodoSourceId)
	msg.Sender = *send
	msg.GroupID = FormatDiceIDDodoGroup(msgRaw.ChannelId)
	msg.GuildID = msgRaw.IslandSourceId
	if msgRaw.MessageType == 1 {
		msgDodo := new(model.TextMessage)
		// pa.Session.Parent.Logger.Infof("Dodo消息内容:%s", string(msgRaw.MessageBody))
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
		_ = recover()
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
		_ = recover()
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
	// pa.Session.Parent.Logger.Infof("send to %s", ExtractDodoUserId(uid))
	err := pa.SendToChatRaw(ctx, uid, text, true)
	if err != nil {
		pa.Session.Parent.Logger.Errorf("DODO 发送私聊消息失败：%v\n", err)
		return
	}
	pa.Session.OnMessageSend(ctx, &Message{
		MessageType: "private",
		Platform:    "DODO",
		Message:     text,
		Sender: SenderBase{
			UserID:   pa.EndPoint.UserID,
			Nickname: pa.EndPoint.Nickname,
		},
	}, flag)
}

func (pa *PlatformAdapterDodo) SendToGroup(ctx *MsgContext, uid string, text string, flag string) {
	err := pa.SendToChatRaw(ctx, uid, text, false)
	if err != nil {
		pa.Session.Parent.Logger.Errorf("DODO 发送消息失败：%v\n", err)
		return
	}
	pa.Session.OnMessageSend(ctx, &Message{
		MessageType: "group",
		Platform:    "DODO",
		Message:     text,
		GroupID:     uid,
		Sender: SenderBase{
			UserID:   pa.EndPoint.UserID,
			Nickname: pa.EndPoint.Nickname,
		},
	}, flag)
}

func (pa *PlatformAdapterDodo) SendFileToPerson(ctx *MsgContext, userID string, path string, flag string) {
	dice := pa.Session.Parent
	fileElement, err := dice.FilepathToFileElement(path)
	if err == nil {
		pa.SendToPerson(ctx, userID, fmt.Sprintf("[尝试发送文件: %s，但不支持]", fileElement.File), flag)
	} else {
		pa.SendToPerson(ctx, userID, fmt.Sprintf("[尝试发送文件出错: %s]", err.Error()), flag)
	}
}

func (pa *PlatformAdapterDodo) SendFileToGroup(ctx *MsgContext, groupID string, path string, flag string) {
	dice := pa.Session.Parent
	fileElement, err := dice.FilepathToFileElement(path)
	if err == nil {
		pa.SendToGroup(ctx, groupID, fmt.Sprintf("[尝试发送文件: %s，但不支持]", fileElement.File), flag)
	} else {
		pa.SendToGroup(ctx, groupID, fmt.Sprintf("[尝试发送文件出错: %s]", err.Error()), flag)
	}
}

func (pa *PlatformAdapterDodo) SendToChatRaw(ctx *MsgContext, uid string, text string, isPrivate bool) error {
	instance := pa.Client
	dice := pa.Session.Parent
	elem := dice.ConvertStringMessage(text)
	streamToByte := func(stream io.Reader) []byte {
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
				Bytes:    streamToByte(e.file.Stream),
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
	if isPrivate {
		rawID := ExtractDodoUserID(uid)
		_, err := pa.Client.SendDirectMessage(context.Background(), &model.SendDirectMessageReq{
			IslandSourceId: ctx.Group.GuildID,
			DodoSourceId:   rawID,
			MessageBody:    msgBody,
		})
		return err
	}
	rawID := ExtractDodoGroupID(uid)
	_, err := pa.Client.SendChannelMessage(context.Background(), &model.SendChannelMessageReq{
		ChannelId:   rawID,
		MessageBody: msgBody,
	})
	return err
}

func (pa *PlatformAdapterDodo) MemberBan(_ string, _ string, _ int64) {}

func (pa *PlatformAdapterDodo) MemberKick(_ string, _ string) {}

func FormatDiceIDDodo(sourceid string) string {
	return fmt.Sprintf("DODO:%s", sourceid)
}

func FormatDiceIDDodoGroup(channelid string) string {
	return fmt.Sprintf("DODO-Group:%s", channelid)
}

func ExtractDodoUserID(id string) string {
	if strings.HasPrefix(id, "DODO:") {
		return id[len("DODO:"):]
	}
	return id
}

func ExtractDodoGroupID(id string) string {
	if strings.HasPrefix(id, "DODO-Group:") {
		return id[len("DODO-Group:"):]
	}
	return id
}
func (pa *PlatformAdapterDodo) QuitGroup(ctx *MsgContext, groupId string) {
	_, err := pa.Client.SetBotIslandLeave(context.Background(), &model.SetBotLeaveIslandReq{
		IslandSourceId: ctx.Group.GuildID,
	})
	if err != nil {
		pa.Session.Parent.Logger.Errorf("Dodo退群失败:%v", err)
	}
}

func (pa *PlatformAdapterDodo) SetGroupCardName(ctx *MsgContext, name string) {
	_, err := pa.Client.SetMemberNick(context.Background(), &model.SetMemberNickReq{
		IslandSourceId: ctx.Group.GuildID,
		DodoSourceId:   ExtractDodoUserID(ctx.Player.UserID),
		NickName:       name,
	})
	if err != nil {
		pa.Session.Parent.Logger.Errorf("Dodo设置群名片失败:%v", err)
	}
}
