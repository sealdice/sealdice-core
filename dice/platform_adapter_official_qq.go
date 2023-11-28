package dice

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	qqbot "github.com/sealdice/botgo"
	"github.com/sealdice/botgo/dto"
	"github.com/sealdice/botgo/event"
	qqapi "github.com/sealdice/botgo/openapi"
	qqtoken "github.com/sealdice/botgo/token"
	qqws "github.com/sealdice/botgo/websocket"
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
	// 频道私聊消息
	var guildDirectMessage event.DirectMessageEventHandler = pa.GuildDirectMessageReceive
	// 群聊at消息
	var groupAtMessage event.GroupATMessageEventHandler = pa.GroupAtMessageReceive

	intent := qqws.RegisterHandlers(
		channelAtMessage, guildDirectMessage, groupAtMessage,
	)

	go func() {
		defer func() {
			// 防止崩掉进程
			if r := recover(); r != nil {
				log.Error("official qq 启动失败: ", r)
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

	return 0
}

func (pa *PlatformAdapterOfficialQQ) ChannelAtMessageReceive(event *dto.WSPayload, data *dto.WSATMessageData) error {
	s := pa.Session
	log := s.Parent.Logger
	log.Debugf("收到文字频道消息：%v, %v", event, data)

	s.Execute(pa.EndPoint, pa.channelMsgToStdMsg(data), false)
	return nil
}

func (pa *PlatformAdapterOfficialQQ) channelMsgToStdMsg(msgQQ *dto.WSATMessageData) *Message {
	msg := new(Message)
	timestamp, _ := msgQQ.Timestamp.Time()
	msg.Time = timestamp.Unix()
	msg.MessageType = "group"
	msg.Message = msgQQ.Content
	msg.RawID = msgQQ.ID
	msg.Platform = "OpenQQ"
	msg.GuildID = formatDiceIDOfficialQQChGuild(msgQQ.GuildID)
	msg.GroupID = formatDiceIDOfficialQQChannel(msgQQ.GuildID, msgQQ.ChannelID)
	if msgQQ.Author != nil {
		msg.Sender.Nickname = msgQQ.Author.Username
		msg.Sender.UserID = formatDiceIDOfficialQQCh(msgQQ.Author.ID)
	}
	return msg
}

func (pa *PlatformAdapterOfficialQQ) GuildDirectMessageReceive(event *dto.WSPayload, data *dto.WSDirectMessageData) error {
	s := pa.Session
	log := s.Parent.Logger
	log.Debugf("收到频道私信消息：%v, %v", event, data)

	s.Execute(pa.EndPoint, pa.guildDirectMsgToStdMsg(data), false)
	return nil
}

func (pa *PlatformAdapterOfficialQQ) guildDirectMsgToStdMsg(msgQQ *dto.WSDirectMessageData) *Message {
	msg := new(Message)
	timestamp, _ := msgQQ.Timestamp.Time()
	msg.Time = timestamp.Unix()
	msg.MessageType = "private"
	msg.Message = msgQQ.Content
	msg.RawID = msgQQ.ID
	msg.Platform = "OpenQQ"
	if msgQQ.Author != nil {
		msg.Sender.Nickname = msgQQ.Author.Username
		msg.Sender.UserID = formatDiceIDOfficialQQCh(msgQQ.Author.ID)
	}
	return msg
}

func (pa *PlatformAdapterOfficialQQ) GroupAtMessageReceive(event *dto.WSPayload, data *dto.WSGroupATMessageData) error {
	s := pa.Session
	log := s.Parent.Logger
	log.Debugf("收到群聊消息：%v, %v", event, data)

	s.Execute(pa.EndPoint, pa.groupMsgToStdMsg(data), false)
	return nil
}

func (pa *PlatformAdapterOfficialQQ) groupMsgToStdMsg(msgQQ *dto.WSGroupATMessageData) *Message {
	appID := strconv.FormatUint(pa.AppID, 10)
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
		msg.Sender.Nickname = "用户" + msgQQ.Author.MemberOpenID[len(msgQQ.Author.MemberOpenID)-4:]
		msg.Sender.UserID = formatDiceIDOfficialQQMemberOpenID(appID, msgQQ.GroupOpenID, msgQQ.Author.MemberOpenID)
	}
	return msg
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
	groupId, idType := pa.mustExtractID(uid)
	if idType == OpenQQGroupOpenid {
		pa.sendQQGroupMsgRaw(ctx, rowID, groupId, text)
	} else if idType == OpenQQCHChannel {
		pa.sendQQChannelMsgRaw(ctx, rowID, groupId, text)
	} else {
		pa.Session.Parent.Logger.Errorf("official qq 发送群聊消息失败：错误的群聊id[%s]类型-%d", ctx.Group.GroupID, idType)
		return
	}
}

func (pa *PlatformAdapterOfficialQQ) sendQQGroupMsgRaw(ctx *MsgContext, rowMsgID, groupID string, text string) {
	dice := pa.Session.Parent
	qctx := context.Background()
	elems := dice.ConvertStringMessage(text)
	var (
		content  string
		toCreate *dto.MessageToCreate
	)

	toCreate = &dto.MessageToCreate{
		MsgID: rowMsgID,
	}

	for _, element := range elems {
		switch elem := element.(type) {
		case *TextElement:
			content += elem.Content
		case *AtElement:
			pa.Session.Parent.Logger.Warn("official qq 群聊消息暂不支持发送 @，跳过该部分")
		case *ImageElement:
			url := elem.file.URL
			// 目前不支持本地发送，检查一下url
			if url == "" ||
				strings.Contains(url, "localhost") ||
				strings.Contains(url, "127.0.0.1") {
				pa.Session.Parent.Logger.Warn("official qq 群聊消息暂不支持发送本地图片，跳过该部分")
			}
			fMsg := &dto.MessageMediaToCreate{
				FileType:   1,
				URL:        url,
				SrvSendMsg: false,
			}
			media, err := pa.Api.PostGroupFile(qctx, groupID, fMsg)
			if err != nil {
				pa.Session.Parent.Logger.Error("official qq 发送群聊消息时，准备图片信息失败：" + err.Error())
				continue
			}

			toCreate.MsgType = 7
			toCreate.Media = &dto.Media{
				FileInfo: media.FileInfo,
			}
		case *RecordElement:
			url := elem.file.URL
			// 目前不支持本地发送，检查一下url
			if url == "" ||
				strings.Contains(url, "localhost") ||
				strings.Contains(url, "127.0.0.1") {
				pa.Session.Parent.Logger.Warn("official qq 群聊消息暂不支持发送本地语音，跳过该部分")
			}
			fMsg := &dto.MessageMediaToCreate{
				FileType:   3,
				URL:        url,
				SrvSendMsg: false,
			}
			media, err := pa.Api.PostGroupFile(qctx, groupID, fMsg)
			if err != nil {
				pa.Session.Parent.Logger.Error("official qq 发送群聊消息时，准备语音信息失败：" + err.Error())
				continue
			}

			toCreate.MsgType = 7
			toCreate.Media = &dto.Media{
				FileInfo: media.FileInfo,
			}
		}
	}

	toCreate.Content = content

	if _, err := pa.Api.PostGroupMessage(qctx, groupID, toCreate); err != nil {
		pa.Session.Parent.Logger.Error("official qq 发送群聊消息失败：" + err.Error())
	}
}

func (pa *PlatformAdapterOfficialQQ) sendQQChannelMsgRaw(ctx *MsgContext, rowMsgID, channelID string, text string) {
	dice := pa.Session.Parent
	qctx := context.Background()
	elems := dice.ConvertStringMessage(text)
	var (
		content  string
		toCreate *dto.MessageToCreate
	)

	for _, elem := range elems {
		switch e := elem.(type) {
		case *TextElement:
			content += e.Content
		case *AtElement:
			if e.Target == "all" {
				content += "@everyone"
			} else {
				content += fmt.Sprintf("<@%s>", e.Target)
			}
		case *ImageElement:
		}
	}

	toCreate = &dto.MessageToCreate{
		Content: content,
		MsgType: 0,
		MsgID:   rowMsgID,
	}
	if _, err := pa.Api.PostMessage(qctx, channelID, toCreate); err != nil {
		pa.Session.Parent.Logger.Error("official qq 发送频道消息失败：" + err.Error())
	}
}

func (pa *PlatformAdapterOfficialQQ) GetGroupInfoAsync(groupID string) {
	// 警告太频繁了，拿掉
	// pa.Session.Parent.Logger.Infof("official qq 更新群信息失败：不支持该功能")
}

func formatDiceIDOfficialQQCh(userID string) string {
	return formatDiceIDOfficialQQ(userID)
}

func formatDiceIDOfficialQQChGuild(guildID string) string {
	return fmt.Sprintf("OpenQQ-Guild:%s", guildID)
}

func formatDiceIDOfficialQQChannel(guildID, channelID string) string {
	return fmt.Sprintf("OpenQQ-Channel:%s-%s", guildID, channelID)
}

func formatDiceIDOfficialQQ(userUnionID string) string {
	return fmt.Sprintf("OpenQQ:%s", userUnionID)
}

func formatDiceIDOfficialQQGroupOpenID(botID, groupOpenID string) string {
	// 在没有qq_unionid时的临时方案
	return fmt.Sprintf("OpenQQ-Group-T:%s-%s", botID, groupOpenID)
}

func formatDiceIDOfficialQQMemberOpenID(botID, groupOpenID, memberOpenID string) string {
	// 在没有qq_unionid时的临时方案
	return fmt.Sprintf("OpenQQ-Member-T:%s-%s-%s", botID, groupOpenID, memberOpenID)
}

type OpenQQIDType = int

const (
	OpenQQUnknown OpenQQIDType = iota

	OpenQQUser
	OpenQQGroupOpenid
	OpenQQGroupMemberOpenid

	OpenQQCHGuild
	OpenQQCHChannel
)

func (pa *PlatformAdapterOfficialQQ) mustExtractID(text string) (string, OpenQQIDType) {
	if strings.HasPrefix(text, "OpenQQ:") {
		return text[len("OpenQQ:"):], OpenQQUser
	}
	if strings.HasPrefix(text, "OpenQQ-Group-T:") {
		temp := text[len("OpenQQ-Group-T:"):]
		lst := strings.Split(temp, "-")
		return lst[1], OpenQQGroupOpenid
	}
	if strings.HasPrefix(text, "OpenQQ-Member-T:") {
		temp := text[len("OpenQQ-Member-T:"):]
		lst := strings.Split(temp, "-")
		return lst[1], OpenQQGroupMemberOpenid
	}
	if strings.HasPrefix(text, "OpenQQ-Guild:") {
		return text[len("OpenQQ-Guild:"):], OpenQQCHGuild
	}
	if strings.HasPrefix(text, "OpenQQ-Channel:") {
		temp := text[len("OpenQQ-Channel:"):]
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
