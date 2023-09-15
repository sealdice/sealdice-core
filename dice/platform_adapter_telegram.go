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
		Name: chat.Title,
		time: time.Now().Unix(),
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
		d := pa.Session.Parent
		d.LastUpdatedTime = time.Now().Unix()
		d.Save(false)
		return 1
	}
	pa.IntentSession = bot
	ep.UserId = FormatDiceIdTelegram(strconv.FormatInt(bot.Self.ID, 10))
	ep.Nickname = bot.Self.UserName
	ep.State = 1
	ep.Enable = true
	d := pa.Session.Parent
	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
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
			if msgRaw.From.IsBot {
				continue
			}
			msg := pa.toStdMessage(msgRaw)
			if msgRaw.NewChatMembers != nil {
				for _, member := range msgRaw.NewChatMembers {
					// 骰子进群
					if member.ID == bot.Self.ID {
						go pa.groupAdded(msg, msgRaw)
					} else {
						// 新人进群
						go pa.groupNewMember(msg, msgRaw, &member)
					}
				}
				continue
			}
			if msgRaw.IsCommand() && msgRaw.Text == "/start" && msg.MessageType == "private" {
				go pa.friendAdded(msg)
				continue
			}
			go pa.Session.Execute(pa.EndPoint, msg, false)
		}
	}()
	return 0
}

func (pa *PlatformAdapterTelegram) groupNewMember(msg *Message, msgRaw *tgbotapi.Message, member *tgbotapi.User) {
	ucache := pa.Session.Parent.Parent.UserIdCache
	logger := pa.Session.Parent.Logger
	ep := pa.EndPoint
	group := pa.Session.ServiceAtNew[msg.GroupId]
	if member.UserName != "" {
		_, cacheExist := ucache.Get(member.UserName)
		if !cacheExist {
			ucache.Set(member.UserName, member.ID)
		}
	}
	if group != nil && group.ShowGroupWelcome {
		ctx := &MsgContext{MessageType: msg.MessageType, EndPoint: ep, Session: pa.Session, Dice: pa.Session.Parent}
		ctx.Player = &GroupPlayerInfo{}
		uidRaw := strconv.FormatInt(member.ID, 10)
		VarSetValueStr(ctx, "$t帐号ID_RAW", uidRaw)
		VarSetValueStr(ctx, "$t账号ID_RAW", uidRaw)
		stdId := FormatDiceIdTelegram(strconv.FormatInt(member.ID, 10))
		VarSetValueStr(ctx, "$t帐号ID", stdId)
		VarSetValueStr(ctx, "$t账号ID", stdId)
		groupName := msgRaw.Chat.Title
		text := DiceFormat(ctx, group.GroupWelcomeMessage)
		logger.Infof("发送欢迎致辞，群: <%s>(%d),新成员id:%d", groupName, msgRaw.Chat.ID, member.ID)
		for _, i := range strings.Split(text, "###SPLIT###") {
			pa.SendToGroup(ctx, msg.GroupId, strings.TrimSpace(i), "")
		}
	}
}

func (pa *PlatformAdapterTelegram) groupAdded(msg *Message, msgRaw *tgbotapi.Message) {
	logger := pa.Session.Parent.Logger
	ep := pa.EndPoint
	ctx := &MsgContext{MessageType: msg.MessageType, EndPoint: ep, Session: pa.Session, Dice: pa.Session.Parent}
	gi := SetBotOnAtGroup(ctx, msg.GroupId)
	gi.InviteUserId = msg.Sender.UserId
	gi.EnteredTime = msg.Time
	pa.GetGroupInfoAsync(msg.GroupId)
	groupName := msgRaw.Chat.Title
	ctx.Player = &GroupPlayerInfo{}
	logger.Infof("发送入群致辞，群: <%s>(%d)", groupName, msgRaw.Chat.ID)
	text := DiceFormatTmpl(ctx, "核心:骰子进群")
	for _, i := range strings.Split(text, "###SPLIT###") {
		pa.SendToGroup(ctx, msg.GroupId, strings.TrimSpace(i), "")
	}
	if ctx.Session.ServiceAtNew[msg.GroupId] != nil {
		for _, i := range ctx.Session.ServiceAtNew[msg.GroupId].ActivatedExtList {
			if i.OnGroupJoined != nil {
				i.callWithJsCheck(ctx.Dice, func() {
					i.OnGroupJoined(ctx, msg)
				})
			}
		}
	}
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
	if ctx.Session.ServiceAtNew[msg.GroupId] != nil {
		for _, i := range ctx.Session.ServiceAtNew[msg.GroupId].ActivatedExtList {
			if i.OnBecomeFriend != nil {
				i.callWithJsCheck(ctx.Dice, func() {
					i.OnBecomeFriend(ctx, msg)
				})
			}
		}
	}
}

func (pa *PlatformAdapterTelegram) toStdMessage(m *tgbotapi.Message) *Message {
	ucache := pa.Session.Parent.Parent.UserIdCache
	logger := pa.Session.Parent.Logger
	self := pa.IntentSession.Self
	msg := new(Message)
	msg.Time = int64(m.Date)
	if m.Entities != nil {
		replacedText := ""
		index := 0
		for _, entity := range m.Entities {
			// 是否包含@信息
			if entity.IsMention() || entity.Type == "text_mention" {
				// text mention时User不为nil
				if entity.User != nil {
					replacedText += string([]rune(m.Text)[index:entity.Offset]) + fmt.Sprintf("tg://user?id=%d", entity.User.ID)
				} else {
					// 这里处理最烦人的username mention 首先判断是不是@了机器人自己
					if self.UserName == string([]rune(m.Text)[entity.Offset+1:entity.Offset+entity.Length]) {
						replacedText += string([]rune(m.Text)[index:entity.Offset]) + fmt.Sprintf("tg://user?id=%d", self.ID)
					} else {
						//@的不是自己，查看是否能从用户名缓存中找到username
						name := string([]rune(m.Text)[entity.Offset+1 : entity.Offset+entity.Length])
						v, exist := ucache.Get(name)
						if exist {
							replacedText += string([]rune(m.Text)[index:entity.Offset]) + fmt.Sprintf("tg://user?id=%d", v)
						} else {
							//找不到，没有办法了，现阶段没有通过username获取userid的api
							replacedText += string([]rune(m.Text)[index : entity.Offset+entity.Length])
						}
					}
				}
			} else {
				//不是mention 忽略
				replacedText += string([]rune(m.Text)[index : entity.Offset+entity.Length])
			}
			index = entity.Offset + entity.Length
		}
		msg.Message = replacedText + string([]rune(m.Text)[index:])
	} else {
		msg.Message = m.Text
	}
	msg.RawId = m.MessageID
	msg.Platform = "TG"
	send := new(SenderBase)
	msg.MessageType = "group"
	msg.GroupId = FormatDiceIdTelegramGroup(strconv.FormatInt(m.Chat.ID, 10))
	if m.From != nil {
		send.UserId = FormatDiceIdTelegram(strconv.FormatInt(m.From.ID, 10))
		if m.From.UserName == "" {
			send.Nickname = m.From.FirstName
		} else {
			send.Nickname = m.From.UserName
			_, cacheExist := ucache.Get(m.From.UserName)
			if !cacheExist {
				ucache.Set(m.From.UserName, m.From.ID)
			}
		}
		if m.From.ID == m.Chat.ID {
			msg.MessageType = "private"
			msg.GroupId = ""
		} else {
			info := tgbotapi.GetChatMemberConfig{ChatConfigWithUser: tgbotapi.ChatConfigWithUser{ChatID: m.Chat.ID, UserID: m.From.ID}}
			member, err := pa.IntentSession.GetChatMember(info)
			if err != nil {
				logger.Errorf("获取TG用户#%d信息失败:%s", m.From.ID, err.Error())
			} else {
				if member.IsCreator() {
					send.GroupRole = "owner"
				} else if member.IsAdministrator() {
					send.GroupRole = "admin"
				}
			}
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

func (pa *PlatformAdapterTelegram) MemberBan(groupId string, userId string, duration int64) {

}

func (pa *PlatformAdapterTelegram) MemberKick(groupId string, userId string) {

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
	pa.Session.OnMessageSend(ctx, &Message{
		Platform:    "TG",
		MessageType: "private",
		Message:     text,
		Sender: SenderBase{
			UserId:   pa.EndPoint.UserId,
			Nickname: pa.EndPoint.Nickname,
		},
	}, flag)
}

func (pa *PlatformAdapterTelegram) SendToGroup(ctx *MsgContext, uid string, text string, flag string) {
	pa.SendToChatRaw(ExtractTelegramGroupId(uid), text)
	pa.Session.OnMessageSend(ctx, &Message{
		Platform:    "TG",
		MessageType: "group",
		Message:     text,
		GroupId:     uid,
		Sender: SenderBase{
			UserId:   pa.EndPoint.UserId,
			Nickname: pa.EndPoint.Nickname,
		},
	}, flag)
}

func (pa *PlatformAdapterTelegram) SendFileToPerson(ctx *MsgContext, uid string, path string, flag string) {
	userId := ExtractTelegramUserId(uid)
	id, _ := strconv.ParseInt(userId, 10, 64)
	bot := pa.IntentSession
	dice := pa.Session.Parent

	e, err := dice.FilepathToFileElement(path)
	if err != nil {
		dice.Logger.Errorf("向Telegram聊天#%d发送文件[path=%s]时出错:%s", id, path, err.Error())
		return
	}

	f := tgbotapi.NewDocument(id, &RequestFileDataImpl{File: e.File, Reader: e.Stream})
	_, err = bot.Send(f)
	if err != nil {
		dice.Logger.Errorf("向Telegram聊天#%d发送文件[path=%s]时出错:%s", id, path, err.Error())
		return
	}
}

func (pa *PlatformAdapterTelegram) SendFileToGroup(ctx *MsgContext, uid string, path string, flag string) {
	groupId := ExtractTelegramGroupId(uid)
	id, _ := strconv.ParseInt(groupId, 10, 64)
	bot := pa.IntentSession
	dice := pa.Session.Parent

	e, err := dice.FilepathToFileElement(path)
	if err != nil {
		dice.Logger.Errorf("向Telegram聊天#%d发送文件[path=%s]时出错:%s", id, path, err.Error())
		return
	}

	f := tgbotapi.NewDocument(id, &RequestFileDataImpl{File: e.File, Reader: e.Stream})
	_, err = bot.Send(f)
	if err != nil {
		dice.Logger.Errorf("向Telegram聊天#%d发送文件[path=%s]时出错:%s", id, path, err.Error())
		return
	}
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
				pa.Session.Parent.Logger.Errorf("向Telegram聊天#%d发送消息时出错:%s", id, err)
				return
			}
			msg = tgbotapi.NewMessage(id, "")
			data := &RequestFileDataImpl{File: e.File, Reader: e.Stream}
			f := tgbotapi.NewDocument(id, data)
			_, err = bot.Send(f)
		case *ImageElement:
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
			if err != nil {
				pa.Session.Parent.Logger.Errorf("向Telegram聊天#%d发送消息时出错:%s", id, err)
				return
			}
		case *TTSElement:
			msg.Text += e.Content
		case *ReplyElement:
			parseInt, err := strconv.ParseInt(e.Target, 10, 64)
			if err != nil {
				pa.Session.Parent.Logger.Errorf("向Telegram聊天#%d发送消息时出错:%s", id, err)
				break
			}
			msg.BaseChat.ReplyToMessageID = int(parseInt)
		}
		if err != nil {
			pa.Session.Parent.Logger.Errorf("向Telegram聊天#%d发送消息时出错:%s", id, err)
			return
		}
	}
	if msg.Text != "" {
		_, err = bot.Send(msg)
	}
	if err != nil {
		pa.Session.Parent.Logger.Errorf("向Telegram聊天#%d发送消息时出错:%s", id, err)
		return
	}
}

func (pa *PlatformAdapterTelegram) QuitGroup(ctx *MsgContext, id string) {
	parseInt, err := strconv.ParseInt(ExtractTelegramGroupId(id), 10, 64)
	if err != nil {
		pa.Session.Parent.Logger.Errorf("退出Telegram群组%s时出错:%s", id, err.Error())
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
