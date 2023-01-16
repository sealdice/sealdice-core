package dice

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"io"
	"strconv"
	"strings"
	"time"
)

type PlatformAdapterTelegram struct {
	Session       *IMSession       `yaml:"-" json:"-"`
	Token         string           `yaml:"token" json:"token"`
	EndPoint      *EndPointInfo    `yaml:"-" json:"-"`
	IntentSession *tgbotapi.BotAPI `yaml:"-" json:"-"`
}

func (pa *PlatformAdapterTelegram) GetGroupInfoAsync(groupId string) {
	if pa.IntentSession == nil {
		return
	}
	rawid, err2 := strconv.ParseInt(ExtractTelegramGroupId(groupId), 10, 64)
	if err2 != nil {
		return
	}
	chatinfo := tgbotapi.ChatConfig{ChatID: rawid}
	chat, err := pa.IntentSession.GetChat(tgbotapi.ChatInfoConfig{ChatConfig: chatinfo})
	if err != nil {
		return
	}
	dm := pa.Session.Parent.Parent
	dm.GroupNameCache.Set(groupId, &GroupNameCacheItem{
		chat.Title,
		time.Now().Unix(),
	})
	group := pa.Session.ServiceAtNew[groupId]
	if group != nil {
		group.GroupName = chat.Title
	}
}

func (pa *PlatformAdapterTelegram) Serve() int {
	logger := pa.Session.Parent.Logger
	ep := pa.EndPoint
	logger.Info("尝试连接Telegram服务……")
	bot, err := tgbotapi.NewBotAPI(pa.Token)
	if err != nil {
		pa.Session.Parent.Logger.Errorf("与Telegram服务进行连接时出错:%s", err.Error())
		ep.State = 3
		ep.Enable = false
		return 1
	}
	pa.IntentSession = bot
	ep.UserId = FormatDiceIdTelegram(strconv.FormatInt(bot.Self.ID, 10))
	ep.Nickname = bot.Self.UserName
	ep.State = 1
	ep.Enable = true
	pa.Session.Parent.Logger.Infof("Telegram 服务连接成功，账号<%s>(%s)", bot.Self.UserName, pa.EndPoint.UserId)
	updateConfig := tgbotapi.NewUpdate(0)

	updateConfig.Timeout = 30

	updates := bot.GetUpdatesChan(updateConfig)
	go func() {
		for update := range updates {
			if update.Message == nil {
				continue
			}
			msgRaw := update.Message
			msg := pa.toStdMessage(msgRaw)
			if msgRaw.IsCommand() && msgRaw.Text == "/start" && msg.MessageType == "private" {
				go pa.friendAdded(msg)
				continue
			}
			pa.Session.Execute(pa.EndPoint, msg, false)
		}
	}()
	return 0
}

func (pa *PlatformAdapterTelegram) friendAdded(msg *Message) {
	logger := pa.Session.Parent.Logger
	ep := pa.EndPoint
	ctx := &MsgContext{MessageType: msg.MessageType, EndPoint: ep, Session: pa.Session, Dice: pa.Session.Parent}
	ctx.Group, ctx.Player = GetPlayerInfoBySender(ctx, msg)
	uid := msg.Sender.UserId
	welcome := DiceFormatTmpl(ctx, "核心:骰子成为好友")
	logger.Infof("与 %s 成为好友，发送好友致辞: %s", uid, welcome)
	for _, i := range strings.Split(welcome, "###SPLIT###") {
		pa.SendToPerson(ctx, uid, strings.TrimSpace(i), "")
	}
}

func (pa *PlatformAdapterTelegram) toStdMessage(m *tgbotapi.Message) *Message {
	msg := new(Message)
	msg.Time = int64(m.Date)
	msg.Message = m.Text
	msg.RawId = m.MessageID
	msg.Platform = "TELEGRAM"
	send := new(SenderBase)
	msg.MessageType = "group"
	msg.GroupId = FormatDiceIdTelegramGroup(strconv.FormatInt(m.Chat.ID, 10))
	var null string
	if m.From != nil {
		send.UserId = FormatDiceIdTelegram(strconv.FormatInt(m.From.ID, 10))
		send.Nickname = m.From.UserName
		if m.From.ID == m.Chat.ID {
			msg.MessageType = "private"
			//GroupId不变成nil会出问题
			msg.GroupId = null
		}
	}
	msg.Sender = *send

	return msg
}

func FormatDiceIdTelegram(diceTg string) string {
	return fmt.Sprintf("TG:%s", diceTg)
}

func FormatDiceIdTelegramGroup(diceTg string) string {
	return fmt.Sprintf("TG-Group:%s", diceTg)
}

func ExtractTelegramUserId(id string) string {
	if strings.HasPrefix(id, "TG:") {
		return id[len("TG:"):]
	}
	return id
}

func ExtractTelegramGroupId(id string) string {
	if strings.HasPrefix(id, "TG-Group:") {
		return id[len("TG-Group:"):]
	}
	return id
}

func (pa *PlatformAdapterTelegram) DoRelogin() bool {
	pa.Session.Parent.Logger.Infof("正在启用Telegram服务……")
	if pa.IntentSession == nil {
		go pa.Serve()
		return true
	}
	pa.IntentSession.StopReceivingUpdates()
	go pa.Serve()
	return true
}

func (pa *PlatformAdapterTelegram) SetEnable(enable bool) {
	ep := pa.EndPoint
	if enable {
		pa.Session.Parent.Logger.Infof("正在启用Telegram服务……")
		if pa.IntentSession == nil {
			go pa.Serve()
			return
		}
		pa.IntentSession.StopReceivingUpdates()
		go pa.Serve()
	} else {
		if pa.IntentSession != nil {
			pa.IntentSession.StopReceivingUpdates()
		}
		pa.EndPoint.State = 0
		ep.Enable = false
	}
}

func (pa *PlatformAdapterTelegram) SendToPerson(ctx *MsgContext, uid string, text string, flag string) {
	pa.SendToChatRaw(ExtractTelegramUserId(uid), text)
	pa.Session.OnMessageSend(ctx, "private", uid, text, flag)
}

func (pa *PlatformAdapterTelegram) SendToGroup(ctx *MsgContext, uid string, text string, flag string) {
	pa.SendToChatRaw(ExtractTelegramGroupId(uid), text)
	pa.Session.OnMessageSend(ctx, "group", uid, text, flag)
}

type RequestFileDataImpl struct {
	Reader io.Reader
	File   string
}

func (r *RequestFileDataImpl) NeedsUpload() bool {
	return true
}
func (r *RequestFileDataImpl) UploadData() (string, io.Reader, error) {
	return r.File, r.Reader, nil
}
func (r *RequestFileDataImpl) SendData() string {
	return r.File
}
func (pa *PlatformAdapterTelegram) SendToChatRaw(uid string, text string) {
	bot := pa.IntentSession
	dice := pa.Session.Parent
	id, _ := strconv.ParseInt(uid, 10, 64)
	elem := dice.ConvertStringMessage(text)
	msg := tgbotapi.NewMessage(id, "")
	var err error
	for _, element := range elem {
		switch e := element.(type) {
		case *TextElement:
			msg.Text += e.Content
		case *AtElement:
			leng := len(msg.Text)
			uid, _ := strconv.ParseInt(e.Target, 10, 64)
			user := &tgbotapi.User{ID: uid}
			data := fmt.Sprintf("@%s ", e.Target)
			msg.Text += data
			entity := tgbotapi.MessageEntity{Type: "text_mention", Offset: leng, Length: len(data), User: user}
			msg.Entities = append(msg.Entities, entity)
		case *FileElement:
			if msg.Text != "" {
				_, err = bot.Send(msg)
			}
			if err != nil {
				pa.Session.Parent.Logger.Errorf("向Telegram聊天#%s发送消息时出错:%s", id, err)
				return
			}
			msg = tgbotapi.NewMessage(id, "")
			data := &RequestFileDataImpl{File: e.File, Reader: e.Stream}
			f := tgbotapi.NewDocument(id, data)
			_, err = bot.Send(f)
		case *ImageElement:
			if err != nil {
				pa.Session.Parent.Logger.Errorf("向Telegram聊天#%s发送消息时出错:%s", id, err)
				return
			}
			fi := e.file
			data := &RequestFileDataImpl{File: fi.File, Reader: fi.Stream}
			f := tgbotapi.NewPhoto(id, data)
			if msg.Text != "" {
				f.Caption = msg.Text
				f.CaptionEntities = msg.Entities
			}
			msg = tgbotapi.NewMessage(id, "")
			f.Thumb = data
			_, err = bot.Send(f)
		case *TTSElement:
			msg.Text += e.Content
		}
		if err != nil {
			pa.Session.Parent.Logger.Errorf("向Telegram聊天#%s发送消息时出错:%s", id, err)
			return
		}
	}
	if msg.Text != "" {
		_, err = bot.Send(msg)
	}
	if err != nil {
		pa.Session.Parent.Logger.Errorf("向Telegram聊天#%s发送消息时出错:%s", id, err)
		return
	}
}

func (pa *PlatformAdapterTelegram) QuitGroup(ctx *MsgContext, id string) {
	parseInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return
	}
	msg := &tgbotapi.LeaveChatConfig{ChatID: parseInt}
	_, err = pa.IntentSession.Send(msg)
	if err != nil {
		pa.Session.Parent.Logger.Errorf("退出Telegram群组%s时出错:%s", id, err.Error())
	}
}

// SetGroupCardName 没有这个接口 不实现
func (pa *PlatformAdapterTelegram) SetGroupCardName(groupId string, userId string, name string) {}
