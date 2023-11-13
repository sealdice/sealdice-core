package dice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Szzrain/dodo-open-go/client"
	"github.com/Szzrain/dodo-open-go/model"
	"github.com/Szzrain/dodo-open-go/websocket"
)

type PlatformAdapterDodo struct {
	Session           *IMSession                                             `yaml:"-" json:"-"`
	ClientID          string                                                 `yaml:"clientID" json:"clientID"`
	Token             string                                                 `yaml:"token" json:"token"`
	EndPoint          *EndPointInfo                                          `yaml:"-" json:"-"`
	Client            client.Client                                          `yaml:"-" json:"-"`
	WebSocket         websocket.Client                                       `yaml:"-" json:"-"`
	UserPermCache     SyncMap[string, *SyncMap[string, *GuildPermCacheItem]] `yaml:"-" json:"-"`
	RetryConnectTimes int                                                    `yaml:"-" json:"-"` // 重连次数
}

const (
	DodoPermManageMember = 1 << 2
	DodoPermAdmin        = 1 << 3
)

type GuildPermCacheItem struct {
	Perm int64
	time int64
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
		logger.Errorf("Dodo连接错误:%s", err.Error())
		for pa.RetryConnectTimes <= 5 {
			pa.RetryConnectTimes++
			time.Sleep(time.Second * 5)
			pa.Session.Parent.Logger.Infof("Dodo 尝试重连, 第 [%d/5] 次", pa.RetryConnectTimes)
			ws.Close()
			if err = ws.Connect(); err == nil {
				pa.RetryConnectTimes = 0
				pa.EndPoint.State = 1
				break
			} else {
				logger.Errorf("Dodo连接错误:%s", err.Error())
			}
		}
		if err != nil {
			logger.Errorf("Dodo 短时间内重试次数过多，先行中断")
			pa.EndPoint.State = 3
			d := pa.Session.Parent
			d.LastUpdatedTime = time.Now().Unix()
			d.Save(false)
			return 1
		}
	}
	pa.WebSocket = ws
	pa.EndPoint.State = 1
	pa.RetryConnectTimes = 0
	pa.Session.Parent.Logger.Infof("Dodo 连接成功")
	pa.EndPoint.Enable = true
	d := pa.Session.Parent
	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
	go func() {
		err := ws.Listen()
		if err != nil {
			logger.Errorf("Dodo监听错误:%s", err.Error())
			if pa.EndPoint.Enable {
				pa.EndPoint.State = 3
				d := pa.Session.Parent
				d.LastUpdatedTime = time.Now().Unix()
				d.Save(false)
				logger.Infof("Dodo 连接断开，正在尝试重连……")
				pa.DoRelogin()
			}
		}
	}()
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
	msg.Sender.GroupRole = pa.checkGuildAdmin(msgRaw.IslandSourceId, msgRaw.DodoSourceId)
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

func (pa *PlatformAdapterDodo) checkGuildAdmin(guildID string, userID string) string {
	var aperm int64
	guildMap, ok := pa.UserPermCache.Load(guildID)
	if ok {
		userCache, ok := guildMap.Load(userID)
		if ok {
			if userCache.Perm == -1 {
				return "owner"
			}
			// 60秒刷新一次, 感觉也许可以提出来作为配置？
			if time.Now().Unix()-userCache.time > 60 {
				aperm = pa.refreshPermCache(guildID, userID)
			} else {
				aperm = userCache.Perm
			}
		} else {
			aperm = pa.refreshPermCache(guildID, userID)
		}
	} else {
		aperm = pa.refreshPermCache(guildID, userID)
	}
	if aperm == -1 {
		return "owner"
	}
	if aperm&int64(DodoPermAdmin|DodoPermManageMember) > 0 {
		return "admin"
	}
	return ""
}

func (pa *PlatformAdapterDodo) refreshPermCache(guildID string, userID string) (aperm int64) {
	aperm = int64(0)
	list, err := pa.Client.GetMemberRoleList(context.Background(), &model.GetMemberRoleListReq{
		IslandSourceId: guildID,
		DodoSourceId:   userID,
	})
	if err != nil {
		pa.Session.Parent.Logger.Errorf("Dodo获取权限列表失败:%s", err.Error())
		return
	}

	for _, role := range list {
		num, err := strconv.ParseInt(role.Permission, 16, 64) // 16 表示这是一个十六进制数
		if err != nil {
			pa.Session.Parent.Logger.Errorf("Dodo权限转换错误:%s", err.Error())
			return
		}
		aperm |= num
	}
	guildMap, ok := pa.UserPermCache.Load(guildID)
	if ok {
		guildMap.Store(userID, &GuildPermCacheItem{
			Perm: aperm,
			time: time.Now().Unix(),
		})
	} else {
		info, err := pa.Client.GetIslandInfo(context.Background(), &model.GetIslandInfoReq{
			IslandSourceId: guildID,
		})
		if err != nil {
			pa.Session.Parent.Logger.Errorf("Dodo获取群信息失败:%s", err.Error())
			return
		}
		guildIDMap := &SyncMap[string, *GuildPermCacheItem]{}
		guildIDMap.Store(userID, &GuildPermCacheItem{
			Perm: aperm,
			time: time.Now().Unix(),
		})
		guildIDMap.Store(info.OwnerDodoSourceId, &GuildPermCacheItem{
			Perm: -1,
		})
		pa.UserPermCache.Store(guildID, guildIDMap)
		if info.OwnerDodoSourceId == userID {
			aperm = -1
		}
	}
	return
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
	err := pa.SendToPersonRaw(ctx, uid, text, true)
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

type DoDoTextMessageComponent struct {
	Type string `json:"type"` // section
	Text struct {
		Content string `json:"content"`
		Type    string `json:"type"` // plain-text
	} `json:"text"`
}

type DoDoImageMessageComponent struct {
	Type     string `json:"type"` // image-group
	Elements []struct {
		Type string `json:"type"` // image
		Src  string `json:"src"`
	} `json:"elements"`
}

// convertLinksToMarkdown 接受一个包含普通文本的字符串，
// 并将其中的URLs转换为Markdown格式的链接。
func convertLinksToMarkdown(text string) string {
	// 正则表达式匹配更严格的URL，确保URL的结尾不包括非字母数字字符。
	re := regexp.MustCompile(`\b(http://www\.|https://www\.|http://|https://)?[a-zA-Z0-9]+([\-\.]{1}[a-zA-Z0-9]+)*\.[a-zA-Z]{2,5}(:[0-9]{1,5})?(/[a-zA-Z0-9#]+[\w\-./?%&=]*)?\b`)

	// 查找文本中的所有链接
	matches := re.FindAllString(text, -1)

	// 遍历所有找到的链接
	for _, match := range matches {
		// 构造Markdown链接
		markdownLink := fmt.Sprintf("[%s](%s)", match, match)

		// 在原文本中替换URL为Markdown链接
		text = regexp.MustCompile(regexp.QuoteMeta(match)).ReplaceAllString(text, markdownLink)
	}

	return text
}

func (pa *PlatformAdapterDodo) SendToPersonRaw(ctx *MsgContext, uid string, text string, isPrivate bool) error {
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
			err := pa.SendMessageRaw(ctx, &model.TextMessage{Content: e.Content}, uid, isPrivate, "")
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
			err = pa.SendMessageRaw(ctx, msgBody, uid, isPrivate, "")
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (pa *PlatformAdapterDodo) SendToChatRaw(ctx *MsgContext, uid string, text string, isPrivate bool) error {
	referenceMessageId := ""
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
	msgSend := &model.CardMessage{
		Content: "",
		Card: &model.CardBodyElement{
			Type:       "card",
			Theme:      "default",
			Components: []interface{}{},
			Title:      "",
		},
	}
	for _, element := range elem {
		switch e := element.(type) {
		case *TextElement:
			if msgSend.Card != nil && len(msgSend.Card.Components) > 0 {
				component, ok := msgSend.Card.Components[len(msgSend.Card.Components)-1].(*DoDoTextMessageComponent)
				if ok {
					component.Text.Content += e.Content
					continue
				}
			}
			msgSend.Card.Components = append(msgSend.Card.Components, &DoDoTextMessageComponent{
				Type: "section",
				Text: struct {
					Content string `json:"content"`
					Type    string `json:"type"`
				}{
					Content: convertLinksToMarkdown(e.Content),
					Type:    "dodo-md",
				},
			})
		case *ImageElement:
			resourceResp, err := instance.UploadImageByBytes(context.Background(), &model.UploadImageByBytesReq{
				Filename: e.file.File,
				Bytes:    streamToByte(e.file.Stream),
			})
			if err != nil {
				return err
			}
			msgSend.Card.Components = append(msgSend.Card.Components, &DoDoImageMessageComponent{
				Type: "image-group",
				Elements: []struct {
					Type string `json:"type"`
					Src  string `json:"src"`
				}{
					{
						Type: "image",
						Src:  resourceResp.Url,
					},
				},
			})
		case *AtElement:
			if msgSend.Card != nil && len(msgSend.Card.Components) > 0 {
				component, ok := msgSend.Card.Components[len(msgSend.Card.Components)-1].(*DoDoTextMessageComponent)
				if ok {
					component.Text.Content += fmt.Sprintf("<@!%s>", e.Target)
					continue
				}
			}
			msgSend.Card.Components = append(msgSend.Card.Components, &DoDoTextMessageComponent{
				Type: "section",
				Text: struct {
					Content string `json:"content"`
					Type    string `json:"type"`
				}{
					Content: fmt.Sprintf("<@!%s>", e.Target),
					Type:    "dodo-md",
				},
			})
		case *ReplyElement:
			referenceMessageId = e.Target
		}
	}
	err := pa.SendMessageRaw(ctx, msgSend, uid, isPrivate, referenceMessageId)
	if err != nil {
		return err
	}
	return nil
}

func (pa *PlatformAdapterDodo) SendMessageRaw(ctx *MsgContext, msgBody model.IMessageBody, uid string, isPrivate bool, referenceMessageId string) error {
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
		ChannelId:           rawID,
		MessageBody:         msgBody,
		ReferencedMessageId: referenceMessageId,
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
