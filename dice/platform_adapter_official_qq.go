package dice

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	qqbot "github.com/JustAnotherID/botgo"
	"github.com/JustAnotherID/botgo/dto"
	"github.com/JustAnotherID/botgo/event"
	qqapi "github.com/JustAnotherID/botgo/openapi"
	qqtoken "github.com/JustAnotherID/botgo/token"
	qqws "github.com/JustAnotherID/botgo/websocket"
)

type PlatformAdapterOfficialQQ struct {
	Session  *IMSession    `yaml:"-" json:"-"`
	EndPoint *EndPointInfo `yaml:"-" json:"-"`

	AppID     uint64 `yaml:"appID" json:"appID"`
	AppSecret string `yaml:"appSecret" json:"appSecret"`
	Token     string `yaml:"token" json:"token"`

	Api            qqapi.OpenAPI        `yaml:"-" json:"-"`
	SessionManager qqbot.SessionManager `yaml:"-" json:"-"`
	Ctx            context.Context      `yaml:"-" json:"-"`
	CancelFunc     context.CancelFunc   `yaml:"-" json:"-"`
}

func (pa *PlatformAdapterOfficialQQ) Serve() int {
	ep := pa.EndPoint
	s := pa.Session
	log := s.Parent.Logger
	d := pa.Session.Parent

	log.Debug("official qq server")
	qqbot.SetLogger(NewDummyLogger(log.Desugar()))
	token := qqtoken.BotToken(pa.AppID, pa.Token)
	pa.Api = qqbot.NewOpenAPI(token).WithTimeout(3 * time.Second)
	pa.Ctx, pa.CancelFunc = context.WithCancel(context.Background())
	pa.SessionManager = qqbot.NewSessionManager()

	log.Debug("official qq connecting")
	ws, _ := pa.Api.WS(pa.Ctx, nil, "")

	// 文字子频道at消息
	var channelAtMessage event.ATMessageEventHandler = pa.ChannelAtMessageReceive
	// 群聊at消息
	var groupAtMessage event.GroupATMessageEventHandler = pa.GroupAtMessageReceive

	intent := qqws.RegisterHandlers(
		channelAtMessage, groupAtMessage,
	)

	go func() {
		defer func() {
			// 防止崩掉进程
			if r := recover(); r != nil {
				log.Error("official qq 启动失败")
			}
		}()
		_ = pa.SessionManager.Start(ws, token, &intent)
	}()
	ep.State = 1
	ep.Enable = true
	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
	log.Info("official qq 连接成功")

	botInfo, err := pa.Api.Me(pa.Ctx)
	if err == nil {
		ep.UserID = formatDiceIDOfficialQQ(botInfo.ID)
		ep.Nickname = botInfo.Username
	}

	//nolint
	for {
		select {
		case <-pa.Ctx.Done():
		}
	}
}

func (pa *PlatformAdapterOfficialQQ) ChannelAtMessageReceive(event *dto.WSPayload, data *dto.WSATMessageData) error {
	toStdMessage := func(msgQQ *dto.WSATMessageData) *Message {
		msg := new(Message)
		timestamp, _ := msgQQ.Timestamp.Time()
		msg.Time = timestamp.Unix()
		msg.MessageType = "group"
		msg.Message = msgQQ.Content
		msg.RawID = msgQQ.ID
		msg.Platform = "QQ-CH"
		msg.GroupID = formatDiceIDOfficialQQChGroup(msgQQ.GuildID, msgQQ.ChannelID)
		if msgQQ.Author != nil {
			msg.Sender.Nickname = msgQQ.Author.Username
			msg.Sender.UserID = formatDiceIDOfficialQQCh(msgQQ.Author.ID)
		}
		return msg
	}

	s := pa.Session
	log := s.Parent.Logger
	log.Debugf("收到文字频道消息：%v, %v", event, data)

	s.Execute(pa.EndPoint, toStdMessage(data), false)
	return nil
}

func (pa *PlatformAdapterOfficialQQ) GroupAtMessageReceive(event *dto.WSPayload, data *dto.WSGroupATMessageData) error {
	appID := strconv.FormatUint(pa.AppID, 10)
	toStdMessage := func(msgQQ *dto.WSGroupATMessageData) *Message {
		msg := new(Message)
		timestamp, _ := msgQQ.Timestamp.Time()
		msg.Time = timestamp.Unix()
		msg.MessageType = "group"
		msg.Message = msgQQ.Content
		msg.RawID = msgQQ.ID
		msg.Platform = "QQ"
		msg.GroupID = formatDiceIDOfficialQQGroupOpenID(appID, msgQQ.GroupOpenID)
		if msgQQ.Author != nil {
			// FIXME: 我要用户名啊kora
			msg.Sender.Nickname = "未知"
			msg.Sender.UserID = formatDiceIDOfficialQQMemberOpenID(appID, msgQQ.GroupOpenID, msgQQ.Author.MemberOpenID)
		}
		return msg
	}

	s := pa.Session
	log := s.Parent.Logger
	log.Debugf("收到群聊消息：%v, %v", event, data)

	s.Execute(pa.EndPoint, toStdMessage(data), false)
	return nil
}

func (pa *PlatformAdapterOfficialQQ) DoRelogin() bool {
	pa.CancelFunc()
	pa.Session.Parent.Logger.Infof("正在启用 official qq 服务")
	pa.EndPoint.State = 0
	pa.EndPoint.Enable = false
	pa.Api = nil
	pa.SessionManager = nil
	pa.Ctx = nil
	pa.CancelFunc = nil
	return pa.Serve() == 0
}

func (pa *PlatformAdapterOfficialQQ) SetEnable(enable bool) {
	d := pa.Session.Parent
	ep := pa.EndPoint
	if enable {
		if pa.Ctx == nil {
			ep.Enable = false
			ep.State = 2
			ServerOfficialQQ(d, ep)
		} else {
			ep.Enable = true
			ep.State = 1
		}
	} else {
		ep.State = 0
		ep.Enable = false
		if pa.CancelFunc != nil {
			pa.CancelFunc()
		}
	}
	d.LastUpdatedTime = time.Now().Unix()
}

func (pa *PlatformAdapterOfficialQQ) SendToPerson(ctx *MsgContext, uid string, text string, flag string) {
	pa.Session.Parent.Logger.Error("official qq 发送私聊消息失败：不支持该功能")
}

func (pa *PlatformAdapterOfficialQQ) SendToGroup(ctx *MsgContext, uid string, text string, flag string) {
	rowID, ok := VarGetValueStr(ctx, "$tMsgID")
	if !ok {
		// TODO：允许主动消息发送，并校验频率
		pa.Session.Parent.Logger.Error("official qq 发送群聊消息失败：无法直接发送消息")
	}
	groupId, idType := pa.mustExtractID(ctx.Group.GroupID)
	if idType == OpenQQGroupOpenid {
		qctx := context.Background()
		toCreate := &dto.MessageToCreate{
			Content: text,
			MsgType: 0,
			MsgID:   rowID,
		}
		if _, err := pa.Api.PostGroupMessage(qctx, groupId, toCreate); err != nil {
			pa.Session.Parent.Logger.Error("official qq 发送群聊消息失败：" + err.Error())
		}
	} else if idType == OpenQQCHChannel {
		qctx := context.Background()
		toCreate := &dto.MessageToCreate{
			Content: text,
			MsgType: 0,
			MsgID:   rowID,
		}
		if _, err := pa.Api.PostMessage(qctx, groupId, toCreate); err != nil {
			pa.Session.Parent.Logger.Error("official qq 发送频道消息失败：" + err.Error())
		}
	} else {
		pa.Session.Parent.Logger.Errorf("official qq 发送群聊消息失败：错误的群聊id[%s]类型-%d", ctx.Group.GroupID, idType)
		return
	}
}

func (pa *PlatformAdapterOfficialQQ) GetGroupInfoAsync(groupID string) {
	pa.Session.Parent.Logger.Infof("official qq 更新群信息失败：不支持该功能")
}

func formatDiceIDOfficialQQ(userUnionID string) string {
	return fmt.Sprintf("OpenQQ:%s", userUnionID)
}

// func formatDiceIDOfficialQQUserOpenID(botID, userOpenID string) string {
// 	return fmt.Sprintf("OpenQQ-User:%s-%s", botID, userOpenID)
// }

func formatDiceIDOfficialQQGroupOpenID(botID, groupOpenID string) string {
	return fmt.Sprintf("OpenQQ-Group:%s-%s", botID, groupOpenID)
}

func formatDiceIDOfficialQQMemberOpenID(botID, groupOpenID, memberOpenID string) string {
	return fmt.Sprintf("OpenQQ-Member:%s-%s-%s", botID, groupOpenID, memberOpenID)
}

func formatDiceIDOfficialQQCh(userID string) string {
	return fmt.Sprintf("QQ-CH:%s", userID)
}

func formatDiceIDOfficialQQChGroup(guildID, channelID string) string {
	return fmt.Sprintf("QQ-CH-Group:%s-%s", guildID, channelID)
}

type OpenQQIDType = int

const (
	OpenQQUnknown OpenQQIDType = iota
	OpenQQUserUnionid
	OpenQQUserOpenid
	OpenQQGroupOpenid
	OpenQQGroupMemberOpenid

	OpenQQCHUser
	OpenQQCHGuild
	OpenQQCHChannel
)

func (pa *PlatformAdapterOfficialQQ) mustExtractID(text string) (string, OpenQQIDType) {
	if strings.HasPrefix(text, "OpenQQ:") {
		return text[len("OpenQQ:"):], OpenQQUserUnionid
	}
	if strings.HasPrefix(text, "OpenQQ-User:") {
		temp := text[len("OpenQQ-User:"):]
		lst := strings.Split(temp, "-")
		return lst[1], OpenQQUserOpenid
	}
	if strings.HasPrefix(text, "OpenQQ-Group:") {
		temp := text[len("OpenQQ-Group:"):]
		lst := strings.Split(temp, "-")
		return lst[1], OpenQQGroupOpenid
	}
	if strings.HasPrefix(text, "QQ-CH:") {
		return text[len("QQ-CH:"):], OpenQQCHUser
	}
	if strings.HasPrefix(text, "QQ-CH-Group:") {
		temp := text[len("QQ-CH-Group:"):]
		lst := strings.Split(temp, "-")
		return lst[1], OpenQQCHChannel
	}
	return "", OpenQQUnknown
}

func (pa *PlatformAdapterOfficialQQ) SendFileToPerson(ctx *MsgContext, uid string, path string, flag string) {
	pa.SendToPerson(ctx, uid, fmt.Sprintf("[尝试发送文件: %s，但不支持]", filepath.Base(path)), flag)
}

func (pa *PlatformAdapterOfficialQQ) SendFileToGroup(ctx *MsgContext, uid string, path string, flag string) {
	pa.SendToPerson(ctx, uid, fmt.Sprintf("[尝试发送文件: %s，但不支持]", filepath.Base(path)), flag)
}

func (pa *PlatformAdapterOfficialQQ) QuitGroup(_ *MsgContext, _ string) {
	pa.Session.Parent.Logger.Error("official qq 退出群组失败：不支持该功能")
}

func (pa *PlatformAdapterOfficialQQ) SetGroupCardName(_ *MsgContext, _ string) {
	pa.Session.Parent.Logger.Error("official qq 修改名片失败：不支持该功能")
}

func (pa *PlatformAdapterOfficialQQ) MemberBan(_ string, _ string, _ int64) {
	pa.Session.Parent.Logger.Error("official qq 禁言用户失败：不支持该功能")
}

func (pa *PlatformAdapterOfficialQQ) MemberKick(_ string, _ string) {
	pa.Session.Parent.Logger.Error("official qq 踢出用户失败：不支持该功能")
}
