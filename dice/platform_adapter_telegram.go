package dice

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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
	logger.Info("尝试连接Telegram服务……")
	bot, err := tgbotapi.NewBotAPI(pa.Token)
	ep := pa.EndPoint
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
				ctx := &MsgContext{MessageType: msg.MessageType, EndPoint: ep, Session: pa.Session, Dice: pa.Session.Parent}
				ctx.Group, ctx.Player = GetPlayerInfoBySender(ctx, msg)
				uid := msg.Sender.UserId
				welcome := DiceFormatTmpl(ctx, "核心:骰子成为好友")
				logger.Infof("与 %s 成为好友，发送好友致辞: %s", uid, welcome)
				for _, i := range strings.Split(welcome, "###SPLIT###") {
					pa.SendToPerson(ctx, uid, strings.TrimSpace(i), "")
				}
				continue
			}
			pa.Session.Execute(pa.EndPoint, msg, false)
		}
	}()
	return 0
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
	bot := pa.IntentSession
	id, _ := strconv.ParseInt(ExtractTelegramUserId(uid), 10, 64)
	msg := tgbotapi.NewMessage(id, text)
	if _, err := bot.Send(msg); err != nil {
		fmt.Println(err.Error())
	}
	for _, i := range ctx.Dice.ExtList {
		if i.OnMessageSend != nil {
			i.callWithJsCheck(ctx.Dice, func() {
				i.OnMessageSend(ctx, "private", uid, text, flag)
			})
		}
	}
}

func (pa *PlatformAdapterTelegram) SendToGroup(ctx *MsgContext, uid string, text string, flag string) {
	bot := pa.IntentSession
	id, _ := strconv.ParseInt(ExtractTelegramGroupId(uid), 10, 64)
	msg := tgbotapi.NewMessage(id, text)
	if _, err := bot.Send(msg); err != nil {
		fmt.Println(err.Error())
	}
	if ctx.Session.ServiceAtNew[uid] != nil {
		for _, i := range ctx.Session.ServiceAtNew[uid].ActivatedExtList {
			if i.OnMessageSend != nil {
				i.callWithJsCheck(ctx.Dice, func() {
					i.OnMessageSend(ctx, "group", uid, text, flag)
				})
			}
		}
	}
}

// 没有这两个接口捏，不实现

func (pa *PlatformAdapterTelegram) QuitGroup(ctx *MsgContext, id string) {}

func (pa *PlatformAdapterTelegram) SetGroupCardName(groupId string, userId string, name string) {}
