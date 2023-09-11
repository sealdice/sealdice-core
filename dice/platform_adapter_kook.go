package dice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/lonelyevil/kook"
	"github.com/lonelyevil/kook/log_adapter/plog"
	"github.com/phuslu/log"
	"github.com/yuin/goldmark"
	"html"
	"io"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

// ConsoleWriterShutUp Kook go的作者要求必须使用他们自己的logger用于构造Intent Session，并且该logger不可缺省，因此这里重新实现一个不干活的logger以保证控制台log的干净整洁
type ConsoleWriterShutUp struct {
	*log.ConsoleWriter
}

func (c *ConsoleWriterShutUp) Close() (err error)                         { return nil }    //nolint
func (c *ConsoleWriterShutUp) write(out io.Writer, p []byte) (int, error) { return 0, nil } //nolint
func (c *ConsoleWriterShutUp) format(out io.Writer, args *log.FormatterArgs) (n int, err error) { //nolint
	return 0, nil
}
func (c *ConsoleWriterShutUp) WriteEntry(e *log.Entry) (int, error)              { return 0, nil } //nolint
func (c *ConsoleWriterShutUp) writew(out io.Writer, p []byte) (n int, err error) { return 0, nil } //nolint

// kook go的鉴权目前并不好用，这里重写一遍
const (
	RolePermissionAdmin                  kook.RolePermission = 1 << iota
	RolePermissionManageGuild            kook.RolePermission = 1 << 1
	RolePermissionViewAuditLog           kook.RolePermission = 1 << 2
	RolePermissionCreateInvite           kook.RolePermission = 1 << 3
	RolePermissionManageInvite           kook.RolePermission = 1 << 4
	RolePermissionManageChannel          kook.RolePermission = 1 << 5
	RolePermissionKickUser               kook.RolePermission = 1 << 6
	RolePermissionBanUser                kook.RolePermission = 1 << 7
	RolePermissionManageGuildEmoji       kook.RolePermission = 1 << 8
	RolePermissionChangeNickname         kook.RolePermission = 1 << 9
	RolePermissionManageRolePermission   kook.RolePermission = 1 << 10
	RolePermissionViewChannel            kook.RolePermission = 1 << 11
	RolePermissionSendMessage            kook.RolePermission = 1 << 12
	RolePermissionManageMessage          kook.RolePermission = 1 << 13
	RolePermissionUploadFile             kook.RolePermission = 1 << 14
	RolePermissionConnectVoice           kook.RolePermission = 1 << 15
	RolePermissionManageVoice            kook.RolePermission = 1 << 16
	RolePermissionMentionEveryone        kook.RolePermission = 1 << 17
	RolePermissionCreateReaction         kook.RolePermission = 1 << 18
	RolePermissionFollowReaction         kook.RolePermission = 1 << 19
	RolePermissionInvitedToVoice         kook.RolePermission = 1 << 20
	RolePermissionForceManualVoice       kook.RolePermission = 1 << 21
	RolePermissionFreeVoice              kook.RolePermission = 1 << 22
	RolePermissionVoice                  kook.RolePermission = 1 << 23
	RolePermissionManageUserVoiceReceive kook.RolePermission = 1 << 24
	RolePermissionManageUserVoiceCreate  kook.RolePermission = 1 << 25
	RolePermissionManageNickname         kook.RolePermission = 1 << 26
	RolePermissionPlayMusic              kook.RolePermission = 1 << 27
)

// RolePermissionAll 有两种情况会使一个用户拥有一个服务器内的所有权限，第一种是作为服务器创建者的用户，即Owner，第二种是被授予了Admin权限的用户
// 但是尽管他们可以bypass所有的权限检查，但是他们自身并不一定拥有所有的权限，这导致检查时会出现问题，因此这里创建一个permissionAll权限，
// 并把Admin和Owner当作拥有该权限的用户进行处理
const (
	RolePermissionAll = RolePermissionAdmin |
		RolePermissionManageGuild |
		RolePermissionViewAuditLog |
		RolePermissionCreateInvite |
		RolePermissionManageInvite |
		RolePermissionManageChannel |
		RolePermissionKickUser |
		RolePermissionBanUser |
		RolePermissionManageGuildEmoji |
		RolePermissionChangeNickname |
		RolePermissionManageRolePermission |
		RolePermissionViewChannel |
		RolePermissionSendMessage |
		RolePermissionManageMessage |
		RolePermissionUploadFile |
		RolePermissionConnectVoice |
		RolePermissionManageVoice |
		RolePermissionMentionEveryone |
		RolePermissionCreateReaction |
		RolePermissionFollowReaction |
		RolePermissionInvitedToVoice |
		RolePermissionForceManualVoice |
		RolePermissionFreeVoice |
		RolePermissionVoice |
		RolePermissionManageUserVoiceReceive |
		RolePermissionManageUserVoiceCreate |
		RolePermissionManageNickname |
		RolePermissionPlayMusic
)

type CardMessage struct {
	Type    string        `json:"type"`
	Modules []interface{} `json:"modules"`
	Theme   string        `json:"theme"`
	Size    string        `json:"size"`
}

type CardMessageModuleText struct {
	Type string `json:"type"`
	Text struct {
		Content string `json:"content"`
		Type    string `json:"type"`
	} `json:"text"`
}

type CardMessageModuleImage struct {
	Type     string `json:"type"`
	Elements []struct {
		Type string `json:"type"`
		Src  string `json:"src"`
	} `json:"elements"`
}

type CardMessageModuleFile struct {
	Type  string `json:"type"`
	Title string `json:"title"`
	Src   string `json:"src"`
	//Size  string `json:"size"`
}

// PlatformAdapterKook 与 PlatformAdapterDiscord 基本相同的实现，因此不详细写注释了，可以去参考隔壁的实现
type PlatformAdapterKook struct {
	Session       *IMSession    `yaml:"-" json:"-"`
	Token         string        `yaml:"token" json:"token"`
	EndPoint      *EndPointInfo `yaml:"-" json:"-"`
	IntentSession *kook.Session `yaml:"-" json:"-"`
}

func (pa *PlatformAdapterKook) GetGroupInfoAsync(groupId string) {
	//极罕见情况下，未连接成功或被禁用的Endpoint也会去call GetGroupInfoAsync，并且由于IntentSession并未被实例化而抛出nil错误，因此这里做一个检查
	if pa.IntentSession == nil {
		return
	}
	logger := pa.Session.Parent.Logger
	dm := pa.Session.Parent.Parent
	go pa.updateChannelNum()
	channel, err := pa.IntentSession.ChannelView(ExtractKookChannelId(groupId))
	if err != nil {
		logger.Errorf("获取Kook频道信息#%s时出错:%s", groupId, err.Error())
		return
	}
	dm.GroupNameCache.Set(groupId, &GroupNameCacheItem{
		Name: channel.Name,
		time: time.Now().Unix(),
	})
	group := pa.Session.ServiceAtNew[groupId]
	if group != nil {
		if channel.Name != group.GroupName {
			group.GroupName = channel.Name
			group.UpdatedAtTime = time.Now().Unix()
		}
	}
}

func (pa *PlatformAdapterKook) updateChannelNum() {
	page := new(kook.PageSetting)
	guilds, _, _ := pa.IntentSession.GuildList(page)
	GroupNum := 0
	//guilds是bot加入的服务器list，channels是每个服务器里的频道（有权限访问的）
	for _, guild := range guilds {
		GroupNum += len(guild.Channels)
	}
	pa.EndPoint.GroupNum = int64(GroupNum)
}

func (pa *PlatformAdapterKook) updateGameStatus() {
	logger := pa.Session.Parent.Logger
	//注释掉的部分是遗留代码，用于在kook中注册一个叫做SealDice的GameStatus，只需要执行一次因此注释掉
	//gameupdate := new(kook.GameUpdate)
	//gameupdate.ID = int64(768222)
	//gameupdate.Icon = "https://img.kookapp.cn/assets/2022-12/DfYli1buyO0e80c0.png"
	//gameupdate.Name = "SealDice"
	//_, _ = pa.IntentSession.GameUpdate(gameupdate)
	err := pa.IntentSession.GameActivity(int64(768222))
	if err != nil {
		logger.Errorf("更新游戏状态时出错:%s", err.Error())
		return
	}
}

func (pa *PlatformAdapterKook) Serve() int {
	//不喜欢太安静的控制台可以把ConsoleWriterShutUp换成log.ConsoleWriter
	l := log.Logger{
		Level:  log.TraceLevel,
		Writer: &ConsoleWriterShutUp{},
	}
	s := kook.New(pa.Token, plog.NewLogger(&l))
	s.AddHandler(func(ctx *kook.KmarkdownMessageContext) {
		if ctx.Extra.Author.Bot {
			return
		}
		if ctx.Common.Type == kook.MessageTypeKMarkdown || ctx.Common.Type == kook.MessageTypeImage {
			pa.Session.Execute(pa.EndPoint, pa.toStdMessage(ctx), false)
			return
		}
	})
	s.AddHandler(func(ctx *kook.ImageMessageContext) {
		msg := new(Message)
		msg.Time = ctx.Common.MsgTimestamp
		msg.RawId = ctx.Common.MsgID
		msg.Platform = "KOOK"
		send := new(SenderBase)
		send.UserId = FormatDiceIdKook(ctx.Common.AuthorID)
		send.Nickname = ctx.Extra.Author.Nickname
		if ctx.Common.ChannelType == "PERSON" {
			msg.MessageType = "private"
		} else {
			msg.MessageType = "group"
			msg.GroupId = FormatDiceIdKookChannel(ctx.Common.TargetID)
			msg.GuildId = FormatDiceIdKookGuild(ctx.Extra.GuildID)
			//if pa.checkIfGuildAdmin(ctx) {
			//	send.GroupRole = "admin"
			//}
		}
		msg.Message = fmt.Sprintf("[CQ:image,file=someimage,url=%s]", ctx.Common.Content)
		msg.Sender = *send
		pa.Session.Execute(pa.EndPoint, msg, false)
	})
	s.AddHandler(func(ctx *kook.MessageDeleteContext) {
		msg := new(Message)
		msg.Time = ctx.Common.MsgTimestamp
		msg.RawId = ctx.Extra.MsgID
		msg.GroupId = FormatDiceIdKookChannel(ctx.Extra.ChannelID)
		msg.Sender.UserId = FormatDiceIdKook(ctx.Common.AuthorID)
		msg.Sender.Nickname = "系统"
		if ctx.Common.ChannelType == "PERSON" {
			msg.MessageType = "private"
		} else {
			msg.MessageType = "group"
		}
		mctx := &MsgContext{Session: pa.Session, EndPoint: pa.EndPoint, Dice: pa.Session.Parent, MessageType: msg.MessageType}
		//pa.Session.Parent.Logger.Infof("删除信息#%s(%s)", msg.RawId, msg.GroupId)
		pa.Session.OnMessageDeleted(mctx, msg)
	})

	s.AddHandler(func(ctx *kook.BotJoinContext) {
		msg := new(Message)
		msg.Time = ctx.Common.MsgTimestamp
		msg.RawId = ctx.Common.MsgID
		msg.Platform = "KOOK"

		guild, err := s.GuildView(ctx.Extra.GuildID)
		if err != nil {
			pa.Session.Parent.Logger.Errorf("无法获取服务器信息，跳过入群致辞")
			return
		}

		msg.GuildId = FormatDiceIdKookGuild(ctx.Extra.GuildID)
		// WelcomeChannel 是发送“xxx加入群组”消息的频道，DefaultChannel 是加入后第一个看到的频道
		// 这两个可能都为空
		if guild.WelcomeChannelID != "" {
			msg.GroupId = FormatDiceIdKookChannel(guild.WelcomeChannelID)
		} else if guild.DefaultChannelID != "" {
			msg.GroupId = FormatDiceIdKookChannel(guild.DefaultChannelID)
		}

		// 如果获取不到默认频道的话，入群致辞和 OnGuildJoined 基本上没什么意义
		if msg.GroupId == "" {
			return
		}

		msg.Sender.UserId = FormatDiceIdKook(ctx.Common.AuthorID)
		msg.Sender.Nickname = "系统"
		if ctx.Common.ChannelType == "PERSON" {
			msg.MessageType = "private"
		} else {
			msg.MessageType = "group"
		}

		mctx := &MsgContext{Session: pa.Session, EndPoint: pa.EndPoint, Dice: pa.Session.Parent, MessageType: msg.MessageType}
		pa.GetGroupInfoAsync(msg.GroupId)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					pa.Session.Parent.Logger.Errorf("入群致辞异常: %v 堆栈: %v", r, string(debug.Stack()))
				}
			}()

			// 稍作等待后发送入群致词
			time.Sleep(1 * time.Second)

			mctx.Player = &GroupPlayerInfo{}
			pa.Session.Parent.Logger.Infof("发送入群致辞，群: <%s>(%s)", guild.Name, msg.GuildId)
			text := DiceFormatTmpl(mctx, "核心:骰子进群")
			for _, i := range strings.Split(text, "###SPLIT###") {
				pa.SendToGroup(mctx, msg.GroupId, strings.TrimSpace(i), "")
			}
		}()

		// 此时 ServiceAtNew 中这个频道一般为空，照 im_session.go 中的方法处理
		channel := mctx.Session.ServiceAtNew[msg.GroupId]
		if channel == nil {
			channel = SetBotOnAtGroup(mctx, msg.GroupId)
			channel.Active = true
			channel.DiceIdExistsMap.Store(pa.EndPoint.UserId, true)
			channel.UpdatedAtTime = time.Now().Unix()
		}

		if mctx.Session.ServiceAtNew[msg.GroupId] != nil {
			for _, i := range mctx.Session.ServiceAtNew[msg.GroupId].ActivatedExtList {
				if i.OnGuildJoined != nil {
					i.callWithJsCheck(mctx.Dice, func() {
						i.OnGuildJoined(mctx, msg)
					})
				}
			}
		}
	})
	err := s.Open()
	if err != nil {
		pa.Session.Parent.Logger.Errorf("与KOOK服务建立连接时出错:%s", err.Error())
		return 1
	}
	pa.IntentSession = s
	go pa.updateGameStatus()
	pa.EndPoint.State = 1
	pa.EndPoint.Enable = true
	self, _ := s.UserMe()
	pa.EndPoint.Nickname = self.Nickname
	pa.EndPoint.UserId = FormatDiceIdKook(self.ID)
	pa.Session.Parent.Logger.Infof("KOOK 连接成功，账号<%s>(%s)", pa.EndPoint.Nickname, pa.EndPoint.UserId)
	d := pa.Session.Parent
	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
	return 0
}

func (pa *PlatformAdapterKook) DoRelogin() bool {
	pa.Session.Parent.Logger.Infof("正在重新登录KOOK服务……")
	pa.EndPoint.State = 0
	pa.EndPoint.Enable = false
	if pa.IntentSession != nil {
		_ = pa.IntentSession.Close()
	}
	pa.IntentSession = nil
	return pa.Serve() == 0
}

func (pa *PlatformAdapterKook) SetEnable(enable bool) {
	if enable {
		pa.Session.Parent.Logger.Infof("正在启用KOOK服务……")
		if pa.IntentSession == nil {
			pa.Serve()
			return
		}
		err := pa.IntentSession.Open()
		if err != nil {
			pa.Session.Parent.Logger.Errorf("与KOOK服务进行连接时出错:%s", err)
			pa.EndPoint.State = 0
			pa.EndPoint.Enable = false
			return
		}
		pa.updateGameStatus()
		pa.EndPoint.State = 1
		pa.EndPoint.Enable = true
		pa.Session.Parent.Logger.Infof("KOOK 连接成功，账号<%s>(%s)", pa.EndPoint.Nickname, pa.EndPoint.UserId)
	} else {
		if pa.IntentSession == nil {
			return
		}
		pa.EndPoint.State = 0
		pa.EndPoint.Enable = false
		_ = pa.IntentSession.Close()
		pa.IntentSession = nil
	}
	d := pa.Session.Parent
	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
}

func (pa *PlatformAdapterKook) SendToPerson(ctx *MsgContext, userId string, text string, flag string) {
	channel, err := pa.IntentSession.UserChatCreate(ExtractKookUserId(userId))
	if err != nil {
		pa.Session.Parent.Logger.Errorf("创建Kook用户#%s的私聊频道时出错:%s", userId, err)
		return
	}
	pa.SendToChannelRaw(channel.Code, text, true)
	pa.Session.OnMessageSend(ctx, &Message{
		Platform:    "KOOK",
		MessageType: "private",
		Message:     text,
		Sender: SenderBase{
			UserId:   pa.EndPoint.UserId,
			Nickname: pa.EndPoint.Nickname,
		},
	}, flag)
}

func (pa *PlatformAdapterKook) SendToGroup(ctx *MsgContext, groupId string, text string, flag string) {
	pa.SendToChannelRaw(ExtractKookChannelId(groupId), text, false)
	pa.Session.OnMessageSend(ctx, &Message{
		Platform:    "KOOK",
		MessageType: "group",
		Message:     text,
		GroupId:     groupId,
		Sender: SenderBase{
			UserId:   pa.EndPoint.UserId,
			Nickname: pa.EndPoint.Nickname,
		},
	}, flag)
}

func (pa *PlatformAdapterKook) SendFileToPerson(ctx *MsgContext, userId string, path string, flag string) {
	channel, err := pa.IntentSession.UserChatCreate(ExtractKookUserId(userId))
	if err != nil {
		pa.Session.Parent.Logger.Errorf("创建Kook用户#%s的私聊频道时出错:%s", userId, err)
		return
	}
	pa.SendFileToChannelRaw(channel.Code, path, true)
}

func (pa *PlatformAdapterKook) SendFileToGroup(ctx *MsgContext, groupId string, path string, flag string) {
	pa.SendFileToChannelRaw(ExtractKookChannelId(groupId), path, false)
}

func (pa *PlatformAdapterKook) MemberBan(groupId string, userId string, last int64) {

}

func (pa *PlatformAdapterKook) MemberKick(groupId string, userId string) {

}

func (pa *PlatformAdapterKook) SendFileToChannelRaw(id string, path string, private bool) {
	bot := pa.IntentSession
	dice := pa.Session.Parent
	e, err := dice.FilepathToFileElement(path)
	if err != nil {
		dice.Logger.Errorf("向Kook频道#%s发送文件[path=%s]时出错:%s", id, path, err)
		return
	}

	StreamToByte := func(stream io.Reader) []byte {
		buf := new(bytes.Buffer)
		_, err := buf.ReadFrom(stream)
		if err != nil {
			return nil
		}
		return buf.Bytes()
	}
	assert, err := bot.AssetCreate(e.File, StreamToByte(e.Stream))
	if err != nil {
		dice.Logger.Errorf("Kook创建asserts时出错:%s", err)
		return
	}

	card := CardMessage{
		Type:  "card",
		Theme: "primary",
		Size:  "lg",
	}
	cardModule := CardMessageModuleFile{
		Type:  "file",
		Title: e.File,
		Src:   assert,
	}
	card.Modules = append(card.Modules, cardModule)
	cardArray := []CardMessage{card}
	sendText, err := json.Marshal(cardArray)
	if err != nil {
		dice.Logger.Errorf("Kook创建card时出错:%s", err)
		return
	}
	msgb := kook.MessageCreateBase{
		Content: "",
		Type:    kook.MessageTypeCard,
	}
	msgb.Content = string(sendText)
	err = pa.MessageCreateRaw(msgb, id, private)
	if err != nil {
		dice.Logger.Errorf("向Kook频道#%s发送文件[path=%s]时出错:%s", id, path, err)
	}
}

func (pa *PlatformAdapterKook) SendToChannelRaw(id string, text string, private bool) {
	bot := pa.IntentSession
	dice := pa.Session.Parent
	elem := dice.ConvertStringMessage(text)
	//var err error
	StreamToByte := func(stream io.Reader) []byte {
		buf := new(bytes.Buffer)
		_, err := buf.ReadFrom(stream)
		if err != nil {
			return nil
		}
		return buf.Bytes()
	}
	msgb := kook.MessageCreateBase{
		Content: "",
		Type:    kook.MessageTypeCard,
	}
	card := CardMessage{
		Type:  "card",
		Theme: "primary",
		Size:  "lg",
	}
	for _, element := range elem {
		switch e := element.(type) {
		case *TextElement:
			//goldmark.DefaultParser().Parse(txt.NewReader([]byte(e.Content)))
			//msgb.Content += antiMarkdownFormat(e.Content)
			cardModule := CardMessageModuleText{
				Type: "section",
				Text: struct {
					Content string `json:"content"`
					Type    string `json:"type"`
				}{Content: e.Content, Type: "plain-text"},
			}
			card.Modules = append(card.Modules, cardModule)
		case *ImageElement:
			assert, err := bot.AssetCreate(e.file.File, StreamToByte(e.file.Stream))
			if err != nil {
				pa.Session.Parent.Logger.Errorf("Kook创建asserts时出错:%s", err)
				break
			}
			cardModule := CardMessageModuleImage{
				Type: "container",
			}
			cardModule.Elements = append(cardModule.Elements, struct {
				Type string `json:"type"`
				Src  string `json:"src"`
			}{"image", assert})
			card.Modules = append(card.Modules, cardModule)
		case *FileElement:
			assert, err := bot.AssetCreate(e.File, StreamToByte(e.Stream))
			if err != nil {
				pa.Session.Parent.Logger.Errorf("Kook创建asserts时出错:%s", err)
				break
			}
			cardModule := CardMessageModuleFile{
				Type:  "file",
				Title: e.File,
				Src:   assert,
			}
			card.Modules = append(card.Modules, cardModule)
		case *AtElement:
			cardModule := CardMessageModuleText{
				Type: "section",
				Text: struct {
					Content string `json:"content"`
					Type    string `json:"type"`
				}{Content: "(met)" + e.Target + "(met)", Type: "kmarkdown"},
			}
			card.Modules = append(card.Modules, cardModule)
			//msgb.Content = msgb.Content + fmt.Sprintf("(met)%s(met)", e.Target)
		case *TTSElement:
			//msgb.Content += antiMarkdownFormat(e.Content)
		case *ReplyElement:
			msgb.Quote = e.Target
		}
	}
	cardArray := []CardMessage{card}
	if true {
		sendText, err := json.Marshal(cardArray)
		if err != nil {
			pa.Session.Parent.Logger.Errorf("Kook创建card时出错:%s", err)
			return
		}
		msgb.Content = string(sendText)
		//pa.Session.Parent.Logger.Infof("Kook发送消息:%s", msgb.Content)
		err = pa.MessageCreateRaw(msgb, id, private)
		if err != nil {
			pa.Session.Parent.Logger.Errorf("向Kook频道#%s发送消息时出错:%s", id, err)
		}
	}
}

func antiMarkdownFormat(text string) string {
	text = strings.ReplaceAll(text, "\\", "\\\\")
	//text = strings.ReplaceAll(text, "_", "\\_")
	text = strings.ReplaceAll(text, "~", "\\~")
	//text = strings.ReplaceAll(text, "|", "\\|")
	//text = strings.ReplaceAll(text, ">", "\\>")
	//text = strings.ReplaceAll(text, "<", "\\<")
	text = strings.ReplaceAll(text, "`", "\\`")
	//text = strings.ReplaceAll(text, "#", "\\#")
	//text = strings.ReplaceAll(text, "+", "\\+")
	//text = strings.ReplaceAll(text, "-", "\\-")
	//text = strings.ReplaceAll(text, "=", "\\=")
	//text = strings.ReplaceAll(text, "{", "\\{")
	//text = strings.ReplaceAll(text, "}", "\\}")
	//text = strings.ReplaceAll(text, ".", "\\.")
	text = strings.ReplaceAll(text, "!", "\\!")
	text = strings.ReplaceAll(text, "(", "\\(")
	text = strings.ReplaceAll(text, ")", "\\)")
	text = strings.ReplaceAll(text, "[", "\\[")
	text = strings.ReplaceAll(text, "]", "\\]")
	text = strings.ReplaceAll(text, "*", "\\*")
	//text = strings.ReplaceAll(text, ":", "\\:")
	//text = strings.ReplaceAll(text, "\"", "\\\"")
	//text = strings.ReplaceAll(text, "'", "\\'")
	//text = strings.ReplaceAll(text, "/", "\\/")
	//text = strings.ReplaceAll(text, "@", "\\@")
	//text = strings.ReplaceAll(text, "%", "\\%")
	//text = strings.ReplaceAll(text, ",", "\\,")
	//text = strings.ReplaceAll(text, " ", "\\ ")
	return text
}

func (pa *PlatformAdapterKook) MessageCreateRaw(base kook.MessageCreateBase, id string, isPrivate bool) error {
	bot := pa.IntentSession
	if isPrivate {
		_, err := bot.DirectMessageCreate(&kook.DirectMessageCreate{ChatCode: id, MessageCreateBase: base})
		return err
	} else {
		base.TargetID = id
		_, err := bot.MessageCreate(&kook.MessageCreate{MessageCreateBase: base})
		//pa.Session.Parent.Logger.Infof("Kook发送消息返回:%s", ret)
		return err
	}
}

func FormatDiceIdKook(diceKook string) string {
	return fmt.Sprintf("KOOK:%s", diceKook)
}

func FormatDiceIdKookGuild(diceKook string) string {
	return fmt.Sprintf("KOOK-Guild:%s", diceKook)
}

func FormatDiceIdKookChannel(diceKook string) string {
	return fmt.Sprintf("KOOK-CH-Group:%s", diceKook)
}

func ExtractKookUserId(id string) string {
	if strings.HasPrefix(id, "KOOK:") {
		return id[len("KOOK:"):]
	}
	return id
}

func ExtractKookChannelId(id string) string {
	if strings.HasPrefix(id, "KOOK-CH-Group:") {
		return id[len("KOOK-CH-Group:"):]
	}
	return id
}

func (pa *PlatformAdapterKook) QuitGroup(ctx *MsgContext, groupId string) {
	channel, err := pa.IntentSession.ChannelView(ExtractKookChannelId(groupId))
	if err != nil {
		pa.Session.Parent.Logger.Errorf("获取Kook频道信息#%s时出错:%s", groupId, err.Error())
		return
	}
	err = pa.IntentSession.GuildLeave(channel.GuildID)
	if err != nil {
		pa.Session.Parent.Logger.Errorf("退出Kook服务器#%s时出错:%s", channel.GuildID, err.Error())
		return
	}
}

func (pa *PlatformAdapterKook) SetGroupCardName(groupId string, userId string, name string) {
	nick := &kook.GuildNickname{}
	channel, err := pa.IntentSession.ChannelView(ExtractKookChannelId(groupId))
	if err != nil {
		pa.Session.Parent.Logger.Errorf("获取Kook频道信息#%s时出错:%s", groupId, err.Error())
		return
	}
	nick.GuildID = channel.GuildID
	nick.Nickname = name
	nick.UserID = ExtractKookUserId(userId)
	err = pa.IntentSession.GuildNickname(nick)
	if err != nil {
		pa.Session.Parent.Logger.Errorf("修改Kook用户#%s在服务器#%s(来源频道#%s)的昵称时出错:%s", userId, channel.GuildID, groupId, err.Error())
		return
	}
}

// nolint
func trimHtml(src string) string {
	//将HTML标签全转换成小写
	re, _ := regexp.Compile("\\<[\\S\\s]+?\\>")
	src = re.ReplaceAllStringFunc(src, strings.ToLower)
	//去除STYLE
	re, _ = regexp.Compile("\\<style[\\S\\s]+?\\</style\\>")
	src = re.ReplaceAllString(src, "")
	//去除SCRIPT
	re, _ = regexp.Compile("\\<script[\\S\\s]+?\\</script\\>")
	src = re.ReplaceAllString(src, "")
	//去除所有尖括号内的HTML代码，并换成换行符
	re, _ = regexp.Compile("\\<[\\S\\s]+?\\>")
	src = re.ReplaceAllString(src, "\n")
	//去除连续的换行符
	re, _ = regexp.Compile("\\s{2,}")
	src = re.ReplaceAllString(src, "\n")
	return strings.TrimSpace(src)
}

//	func markdownAntiConvert(src string) string {
//		re, _ := regexp.Compile("\\\\")
//		src = re.ReplaceAllStringFunc(src, addSlash)
//		re, _ = regexp.Compile("`")
//		src = re.ReplaceAllStringFunc(src, addSlash)
//		re, _ = regexp.Compile("\\*")
//		src = re.ReplaceAllStringFunc(src, addSlash)
//		re, _ = regexp.Compile("_")
//		src = re.ReplaceAllStringFunc(src, addSlash)
//		re, _ = regexp.Compile(`{}`)
//		src = re.ReplaceAllStringFunc(src, addSlash)
//		re, _ = regexp.Compile("\\[\\]")
//		src = re.ReplaceAllStringFunc(src, addSlash)
//		re, _ = regexp.Compile("\\(\\)")
//		src = re.ReplaceAllStringFunc(src, addSlash)
//		re, _ = regexp.Compile("#")
//		src = re.ReplaceAllStringFunc(src, addSlash)
//		//re, _ = regexp.Compile("\\\n")
//		//src = re.ReplaceAllString(src, "\n")
//		//re, _ = regexp.Compile("-")
//		//src = re.ReplaceAllStringFunc(src, addSlash)
//		re, _ = regexp.Compile("\\.")
//		src = re.ReplaceAllStringFunc(src, addSlash)
//		re, _ = regexp.Compile("!")
//		src = re.ReplaceAllStringFunc(src, addSlash)
//		return strings.TrimSpace(src)
//	}
func (pa *PlatformAdapterKook) toStdMessage(ctx *kook.KmarkdownMessageContext) *Message {
	logger := pa.Session.Parent.Logger
	msg := new(Message)
	msg.Time = ctx.Common.MsgTimestamp
	msg.RawId = ctx.Common.MsgID
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(ctx.Common.Content), &buf); err != nil {
		logger.Errorf("Kook Markdown 解析错误:%s 内容:%s", err, ctx.Common.Content)
		return nil
	}
	msg.Message = trimHtml(buf.String())
	msg.Message = strings.ReplaceAll(msg.Message, `\[`, "[")
	msg.Message = strings.ReplaceAll(msg.Message, `\]`, "]")
	msg.Message = html.UnescapeString(msg.Message)
	msg.Platform = "KOOK"
	send := new(SenderBase)
	send.UserId = FormatDiceIdKook(ctx.Common.AuthorID)
	send.Nickname = ctx.Extra.Author.Nickname
	if ctx.Common.ChannelType == "PERSON" {
		msg.MessageType = "private"
	} else {
		msg.MessageType = "group"
		msg.GroupId = FormatDiceIdKookChannel(ctx.Common.TargetID)
		msg.GuildId = FormatDiceIdKookGuild(ctx.Extra.GuildID)
		if pa.checkIfGuildAdmin(ctx) {
			send.GroupRole = "admin"
		}
	}
	msg.Sender = *send
	return msg
}

func (pa *PlatformAdapterKook) checkIfGuildAdmin(ctx *kook.KmarkdownMessageContext) bool {
	user, err := pa.IntentSession.UserView(ctx.Common.AuthorID, kook.UserViewWithGuildID(ctx.Extra.GuildID))
	if err != nil {
		return false
	}
	perm := pa.memberPermissions(&ctx.Extra.GuildID, &ctx.Common.TargetID, ctx.Common.AuthorID, user.Roles)
	return perm&int64(RolePermissionAdmin|RolePermissionBanUser|RolePermissionKickUser) > 0 || perm == int64(RolePermissionAll)
}

func (pa *PlatformAdapterKook) memberPermissions(guildId *string, channelId *string, userID string, roles []int64) (apermissions int64) {
	guild, err := pa.IntentSession.GuildView(*guildId)
	if err != nil {
		pa.Session.Parent.Logger.Errorf("Kook GuildView 错误:%s", err)
		return 0
	}
	if userID == guild.MasterID {
		apermissions = int64(RolePermissionAll)
		return
	}
	for _, role := range roles {
		if strconv.FormatInt(role, 10) == guild.ID {
			apermissions |= role
			break
		}
	}

	for _, role := range guild.Roles {
		for _, roleID := range roles {
			if role.RoleID == roleID {
				apermissions |= int64(role.Permissions)
				break
			}
		}
	}

	if apermissions&int64(RolePermissionAdmin) == int64(RolePermissionAdmin) {
		apermissions |= int64(RolePermissionAll)
	}

	//下面两部分用于判断频道权限覆写，由于该函数的目的并不是用于完整鉴权而是用于判断用户是否为管理员，而一般不考虑频道覆写中给予用户管理员的情况，
	//因为这种操作并不合理也会产生安全性问题，同时在省略了这两段循环之后也可以提升性能

	//var denies, allows int64
	// Member overwrites can override role overrides, so do two passes
	//for _, overwrite := range channel.PermissionOverwrites {
	//	for _, roleID := range roles {
	//		if overwrite.Type == PermissionOverwriteTypeRole && roleID == overwrite.ID {
	//			denies |= overwrite.Deny
	//			allows |= overwrite.Allow
	//			break
	//		}
	//	}
	//}

	//apermissions &= ^denies
	//apermissions |= allows

	//for _, overwrite := range channel.PermissionOverwrites {
	//	if overwrite.Type == PermissionOverwriteTypeMember && overwrite.ID == userID {
	//		apermissions &= ^overwrite.Deny
	//		apermissions |= overwrite.Allow
	//		break
	//	}
	//}
	return apermissions
}
