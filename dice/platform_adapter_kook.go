package dice

import (
	"bytes"
	"fmt"
	"github.com/lonelyevil/kook"
	"github.com/lonelyevil/kook/log_adapter/plog"
	"github.com/phuslu/log"
	"io"
	"strconv"
	"strings"
	"time"
)

// ConsoleWriterShutUp Kook go的作者要求必须使用他们自己的logger用于构造Intent Session，并且该logger不可缺省，因此这里重新实现一个不干活的logger以保证控制台log的干净整洁
type ConsoleWriterShutUp struct {
	*log.ConsoleWriter
}

func (c *ConsoleWriterShutUp) Close() (err error)                         { return nil }
func (c *ConsoleWriterShutUp) write(out io.Writer, p []byte) (int, error) { return 0, nil }
func (c *ConsoleWriterShutUp) format(out io.Writer, args *log.FormatterArgs) (n int, err error) {
	return 0, nil
}
func (c *ConsoleWriterShutUp) WriteEntry(e *log.Entry) (int, error)              { return 0, nil }
func (c *ConsoleWriterShutUp) writew(out io.Writer, p []byte) (n int, err error) { return 0, nil }

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
		group.GroupName = channel.Name
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
		if ctx.Common.Type != kook.MessageTypeKMarkdown || ctx.Extra.Author.Bot {
			return
		}
		pa.Session.Execute(pa.EndPoint, pa.toStdMessage(ctx), false)
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
}

func (pa *PlatformAdapterKook) SendToPerson(ctx *MsgContext, userId string, text string, flag string) {
	channel, err := pa.IntentSession.UserChatCreate(ExtractKookUserId(userId))
	if err != nil {
		pa.Session.Parent.Logger.Errorf("创建Kook用户#%s的私聊频道时出错:%s", userId, err)
		return
	}
	pa.SendToChannelRaw(channel.Code, text, true)
	for _, i := range ctx.Dice.ExtList {
		if i.OnMessageSend != nil {
			i.callWithJsCheck(ctx.Dice, func() {
				i.OnMessageSend(ctx, "private", userId, text, flag)
			})
		}
	}
}

func (pa *PlatformAdapterKook) SendToGroup(ctx *MsgContext, groupId string, text string, flag string) {
	pa.SendToChannelRaw(ExtractKookChannelId(groupId), text, false)
	if ctx.Session.ServiceAtNew[groupId] != nil {
		for _, i := range ctx.Session.ServiceAtNew[groupId].ActivatedExtList {
			if i.OnMessageSend != nil {
				i.callWithJsCheck(ctx.Dice, func() {
					i.OnMessageSend(ctx, "group", groupId, text, flag)
				})
			}
		}
	}
}

func (pa *PlatformAdapterKook) SendToChannelRaw(id string, text string, private bool) {
	bot := pa.IntentSession
	dice := pa.Session.Parent
	elem := dice.ConvertStringMessage(text)
	var err error
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
		Type:    kook.MessageTypeKMarkdown,
	}
	for _, element := range elem {
		switch e := element.(type) {
		case *TextElement:
			msgb.Content += e.Content
		case *ImageElement:
			if msgb.Content != "" {
				err = pa.MessageCreateRaw(msgb, id, private)
				if err != nil {
					pa.Session.Parent.Logger.Errorf("向Kook频道#%s发送消息时出错:%s", id, err)
					break
				}
			}
			msgb = kook.MessageCreateBase{
				Content: "",
				Type:    kook.MessageTypeImage,
			}
			assert, err := bot.AssetCreate(e.file.File, StreamToByte(e.file.Stream))
			if err != nil {
				pa.Session.Parent.Logger.Errorf("Kook创建asserts时出错:%s", err)
				break
			}
			msgb.Content = assert
			err = pa.MessageCreateRaw(msgb, id, private)
			if err != nil {
				pa.Session.Parent.Logger.Errorf("向Kook频道#%s发送消息时出错:%s", id, err)
				break
			}
			msgb = kook.MessageCreateBase{
				Content: "",
				Type:    kook.MessageTypeKMarkdown,
			}
		case *FileElement:
			if msgb.Content != "" {
				err = pa.MessageCreateRaw(msgb, id, private)
				if err != nil {
					pa.Session.Parent.Logger.Errorf("向Kook频道#%s发送消息时出错:%s", id, err)
					break
				}
			}
			msgb = kook.MessageCreateBase{
				Content: "",
				Type:    kook.MessageTypeFile,
			}
			assert, err := bot.AssetCreate(e.File, StreamToByte(e.Stream))
			if err != nil {
				pa.Session.Parent.Logger.Errorf("Kook创建asserts时出错:%s", err)
				break
			}
			msgb.Content = assert
			err = pa.MessageCreateRaw(msgb, id, private)
			if err != nil {
				pa.Session.Parent.Logger.Errorf("向Kook频道#%s发送消息时出错:%s", id, err)
				break
			}
			msgb = kook.MessageCreateBase{
				Content: "",
				Type:    kook.MessageTypeKMarkdown,
			}
		case *AtElement:
			msgb.Content = msgb.Content + fmt.Sprintf("(met)%s(met)", e.Target)
		case *TTSElement:
			msgb.Content += e.Content
		}
	}
	if msgb.Content != "" {
		err = pa.MessageCreateRaw(msgb, id, private)
		if err != nil {
			pa.Session.Parent.Logger.Errorf("向Kook频道#%s发送消息时出错:%s", id, err)
		}
	}
}

func (pa *PlatformAdapterKook) MessageCreateRaw(base kook.MessageCreateBase, id string, isPrivate bool) error {
	bot := pa.IntentSession
	var err error
	if isPrivate {
		_, err = bot.DirectMessageCreate(&kook.DirectMessageCreate{ChatCode: id, MessageCreateBase: base})
	} else {
		base.TargetID = id
		_, err = bot.MessageCreate(&kook.MessageCreate{MessageCreateBase: base})
	}
	return err
}

func FormatDiceIdKook(diceKook string) string {
	return fmt.Sprintf("KOOK:%s", diceKook)
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

func (pa *PlatformAdapterKook) toStdMessage(ctx *kook.KmarkdownMessageContext) *Message {
	msg := new(Message)
	msg.Time = ctx.Common.MsgTimestamp
	msg.RawId = ctx.Common.MsgID
	msg.Message = ctx.Common.Content
	msg.Message = strings.ReplaceAll(msg.Message, `\[`, "[")
	msg.Message = strings.ReplaceAll(msg.Message, `\]`, "]")
	msg.Platform = "KOOK"
	send := new(SenderBase)
	send.UserId = FormatDiceIdKook(ctx.Common.AuthorID)
	send.Nickname = ctx.Extra.Author.Nickname
	if ctx.Common.ChannelType == "PERSON" {
		msg.MessageType = "private"
	} else {
		msg.MessageType = "group"
		msg.GroupId = FormatDiceIdKookChannel(ctx.Common.TargetID)
		if pa.checkIfGuildAdmin(ctx) {
			send.GroupRole = "admin"
		}
	}
	msg.Sender = *send
	return msg
}

func (pa *PlatformAdapterKook) checkIfGuildAdmin(ctx *kook.KmarkdownMessageContext) bool {
	user, err := pa.IntentSession.UserView(ctx.Common.AuthorID)
	if err != nil {
		return false
	}
	perm := pa.memberPermissions(&ctx.Extra.GuildID, &ctx.Common.TargetID, ctx.Common.AuthorID, user.Roles)
	return perm&int64(RolePermissionAdmin|RolePermissionBanUser|RolePermissionKickUser) > 0 || perm == int64(RolePermissionAll)
}

func (pa *PlatformAdapterKook) memberPermissions(guildId *string, channelId *string, userID string, roles []int64) (apermissions int64) {
	guild, err := pa.IntentSession.GuildView(*guildId)
	if userID == guild.MasterID {
		apermissions = int64(RolePermissionAll)
		return
	}
	if err != nil {
		return 0
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

	if apermissions&int64(RolePermissionAdmin) == int64(RolePermissionAdmin) {
		apermissions |= int64(RolePermissionAll)
	}

	return apermissions
}
