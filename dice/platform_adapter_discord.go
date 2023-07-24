package dice

import (
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/gorilla/websocket"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// PlatformAdapterDiscord 只有token需要记录，别的是生成的
type PlatformAdapterDiscord struct {
	Session       *IMSession         `yaml:"-" json:"-"`
	Token         string             `yaml:"token" json:"token"`
	ProxyURL      string             `yaml:"proxyURL" json:"proxyURL"`
	EndPoint      *EndPointInfo      `yaml:"-" json:"-"`
	IntentSession *discordgo.Session `yaml:"-" json:"-"`
}

// GetGroupInfoAsync 同步一下群组信息
func (pa *PlatformAdapterDiscord) GetGroupInfoAsync(groupId string) {
	//极罕见情况下，未连接成功或被禁用的Endpoint也会去call GetGroupInfoAsync，并且由于IntentSession并未被实例化而抛出nil错误，因此这里做一个检查
	if pa.IntentSession == nil {
		return
	}
	go pa.updateChannelNum()
	logger := pa.Session.Parent.Logger
	dm := pa.Session.Parent.Parent
	channel, err := pa.IntentSession.Channel(ExtractDiscordChannelId(groupId))
	if err != nil {
		logger.Errorf("获取Discord频道信息#%s时出错:%s", groupId, err.Error())
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

// 更新一下频道的数量
func (pa *PlatformAdapterDiscord) updateChannelNum() {
	guilds := pa.IntentSession.State.Guilds
	GroupNum := 0
	//guilds是bot加入的服务器list，channels是每个服务器里的频道（有权限访问的）
	for _, guild := range guilds {
		channels, _ := pa.IntentSession.GuildChannels(guild.ID)
		GroupNum += len(channels)
	}
	pa.EndPoint.GroupNum = int64(GroupNum)
}

// Serve 启动服务，返回0就是成功，1就是失败
func (pa *PlatformAdapterDiscord) Serve() int {
	dg, err := discordgo.New("Bot " + pa.Token)
	//这里出错很大概率是token不对
	if err != nil {
		pa.Session.Parent.Logger.Errorf("创建DiscordSession时出错:%s", err.Error())
		return 1
	}
	if pa.ProxyURL != "" {
		u, e := url.Parse(pa.ProxyURL)
		if e != nil {
			pa.Session.Parent.Logger.Errorf("代理地址解析错误%s", e.Error())
			return 1
		}
		dg.Client.Transport = &http.Transport{
			Proxy: http.ProxyURL(u),
		}
		dg.Dialer = &websocket.Dialer{HandshakeTimeout: 45 * time.Second}
		dg.Dialer.Proxy = http.ProxyURL(u)
	}
	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		//忽略自己的消息……以及其他机器人的消息和系统消息
		if m.Author.Bot || m.Author.System {
			return
		}
		msg, err := pa.toStdMessage(m)
		if err != nil {
			return
		}
		pa.Session.Execute(pa.EndPoint, msg, false)
	})
	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageDelete) {
		ch, err := pa.IntentSession.Channel(m.ChannelID)
		if err != nil {
			pa.Session.Parent.Logger.Errorf("获取Discord频道#%s信息时出错:%s", FormatDiceIdDiscordChannel(m.ChannelID), err.Error())
			return
		}
		msg := &Message{}
		if ch.Type != discordgo.ChannelTypeDM {
			msg.GroupId = FormatDiceIdDiscordChannel(m.ChannelID)
			msg.MessageType = "group"
		} else {
			msg.MessageType = "private"
		}
		msg.GuildId = m.GuildID
		msg.Sender = SenderBase{}
		if m.BeforeDelete != nil {
			msg.Sender.UserId = FormatDiceIdDiscord(m.BeforeDelete.Author.ID)

			msg.Sender.Nickname = m.BeforeDelete.Author.Username
		} else if m.Author != nil {
			msg.Sender.UserId = FormatDiceIdDiscord(m.Author.ID)
			msg.Sender.Nickname = m.Author.Username
		}
		msg.RawId = m.ID
		msg.Time = m.Timestamp.Unix()
		mctx := &MsgContext{Session: pa.Session, EndPoint: pa.EndPoint, Dice: pa.Session.Parent, MessageType: msg.MessageType}
		pa.Session.OnMessageDeleted(mctx, msg)
	})
	//这里只处理消息，未来根据需要再改这里
	dg.Identify.Intents = discordgo.IntentsAll
	pa.IntentSession = dg
	err = dg.Open()
	//这里出错要么没连上，要么连接被阻止了（懂得都懂）
	if err != nil {
		pa.Session.Parent.Logger.Errorf("与Discord服务进行连接时出错:%s", err.Error())
		return 1
	}
	//把bot的状态改成正在玩SealDice，这一行可以删掉，但是他很cool欸
	_ = pa.IntentSession.UpdateGameStatus(0, "SealDice")
	pa.EndPoint.UserId = FormatDiceIdDiscord(pa.IntentSession.State.User.ID)
	pa.EndPoint.Nickname = pa.IntentSession.State.User.Username
	pa.EndPoint.State = 1
	pa.EndPoint.Enable = true
	pa.Session.Parent.Logger.Infof("Discord 服务连接成功，账号<%s>(%s)", pa.IntentSession.State.User.Username, FormatDiceIdDiscord(pa.IntentSession.State.User.ID))

	d := pa.Session.Parent
	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
	return 0
}

// DoRelogin 重新登录，虽然似乎没什么实现的必要，但还是写一下
func (pa *PlatformAdapterDiscord) DoRelogin() bool {
	if pa.IntentSession == nil {
		success := pa.Serve()
		return success == 0
	}
	_ = pa.IntentSession.Close()
	err := pa.IntentSession.Open()
	if err != nil {
		pa.Session.Parent.Logger.Errorf("与Discord服务进行连接时出错:%s", err.Error())
		pa.EndPoint.State = 0
		return false
	}
	_ = pa.IntentSession.UpdateGameStatus(0, "SealDice")
	pa.EndPoint.State = 1
	pa.EndPoint.Enable = true
	d := pa.Session.Parent
	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
	return true
}

// SetEnable 禁用之后骰子仍然有可能显示在线一段时间，可能是因为没有挥手机制，不过已经不会接收到事件了
func (pa *PlatformAdapterDiscord) SetEnable(enable bool) {
	if enable {
		pa.Session.Parent.Logger.Infof("正在启用Discord服务……")
		if pa.IntentSession == nil {
			pa.Serve()
			return
		}
		err := pa.IntentSession.Open()
		if err != nil {
			pa.Session.Parent.Logger.Errorf("与Discord服务进行连接时出错:%s", err.Error())
			pa.EndPoint.State = 3
			pa.EndPoint.Enable = false
			return
		}
		_ = pa.IntentSession.UpdateGameStatus(0, "SealDice")
		pa.Session.Parent.Logger.Infof("Discord 服务连接成功，账号<%s>(%s)", pa.IntentSession.State.User.Username, FormatDiceIdDiscord(pa.IntentSession.State.User.ID))
		pa.EndPoint.State = 1
		pa.EndPoint.Enable = true
	} else {
		pa.EndPoint.State = 0
		pa.EndPoint.Enable = false
		_ = pa.IntentSession.Close()
	}
	d := pa.Session.Parent
	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
}

// SendToPerson 这里发送的是私聊（dm）消息，私信对于discord来说也被视为一个频道
func (pa *PlatformAdapterDiscord) SendToPerson(ctx *MsgContext, userId string, text string, flag string) {
	is := pa.IntentSession
	ch, err := is.UserChannelCreate(ExtractDiscordUserId(userId))
	if err != nil {
		pa.Session.Parent.Logger.Errorf("创建Discord用户#%s的私聊频道时出错:%s", userId, err)
		return
	}
	pa.sendToChannelRaw(ch.ID, text)
	pa.Session.OnMessageSend(ctx, &Message{
		Platform:    "DISCORD",
		MessageType: "private",
		Message:     text,
		Sender: SenderBase{
			UserId:   pa.EndPoint.UserId,
			Nickname: pa.EndPoint.Nickname,
		},
	}, flag)
}

// SendToGroup 发送群聊（实际上是频道）消息
func (pa *PlatformAdapterDiscord) SendToGroup(ctx *MsgContext, groupId string, text string, flag string) {
	//_, err := pa.IntentSession.ChannelMessageSend(ExtractDiscordChannelId(groupId), text)
	pa.sendToChannelRaw(groupId, text)
	pa.Session.OnMessageSend(ctx, &Message{
		Platform:    "DISCORD",
		MessageType: "group",
		Message:     text,
		GroupId:     groupId,
		Sender: SenderBase{
			UserId:   pa.EndPoint.UserId,
			Nickname: pa.EndPoint.Nickname,
		},
	}, flag)
}

func (pa *PlatformAdapterDiscord) sendToChannelRaw(channelId string, text string) {
	dice := pa.Session.Parent
	logger := pa.Session.Parent.Logger
	elem := dice.ConvertStringMessage(text)
	id := ExtractDiscordChannelId(channelId)
	var err error
	msgSend := &discordgo.MessageSend{Content: ""}
	for _, element := range elem {
		switch e := element.(type) {
		case *TextElement:
			msgSend.Content = msgSend.Content + antiMarkdownFormat(e.Content)
		case *AtElement:
			if e.Target == "all" {
				msgSend.Content = msgSend.Content + "@everyone "
				break
			}
			msgSend.Content = msgSend.Content + fmt.Sprintf("<@%s>", e.Target)
		case *FileElement:
			var files []*discordgo.File
			files = append(files, &discordgo.File{
				Name:        e.File,
				ContentType: e.ContentType,
				Reader:      e.Stream,
			})
			msgSend.Files = files
			_, err = pa.IntentSession.ChannelMessageSendComplex(id, msgSend)
			msgSend = &discordgo.MessageSend{Content: ""}
		case *ImageElement:
			var files []*discordgo.File
			f := e.file
			files = append(files, &discordgo.File{
				Name:        f.File,
				ContentType: f.ContentType,
				Reader:      f.Stream,
			})
			msgSend.Files = files
			_, err = pa.IntentSession.ChannelMessageSendComplex(id, msgSend)
			msgSend = &discordgo.MessageSend{Content: ""}
		case *TTSElement:
			if msgSend.Content != "" || msgSend.Files != nil {
				_, err = pa.IntentSession.ChannelMessageSendComplex(id, msgSend)
			}
			if err != nil {
				pa.Session.Parent.Logger.Errorf("向Discord频道#%s发送消息时出错:%s", id, err)
				break
			}
			msgSend = &discordgo.MessageSend{Content: ""}
			_, err = pa.IntentSession.ChannelMessageSendComplex(id, &discordgo.MessageSend{
				Content: e.Content,
				TTS:     true,
			})
		case *ReplyElement:
			channel, err := pa.IntentSession.Channel(id)
			if err != nil {
				logger.Errorf("获取Discord频道信息#%s时出错:%s", id, err.Error())
				break
			}
			ref := &discordgo.MessageReference{MessageID: e.Target, ChannelID: id, GuildID: channel.GuildID}
			msgSend.Reference = ref
		}
		if err != nil {
			pa.Session.Parent.Logger.Errorf("向Discord频道#%s发送消息时出错:%s", id, err)
			return
		}
	}
	if msgSend.Content != "" || msgSend.Files != nil {
		_, err = pa.IntentSession.ChannelMessageSendComplex(id, msgSend)
		//pa.Session.Parent.Logger.Infof("向Discord频道#%s发送消息:%s", id, msgSend.Content)
	}
	if err != nil {
		pa.Session.Parent.Logger.Errorf("向Discord频道#%s发送消息时出错:%s", id, err)
	}
}

// QuitGroup 退出服务器
func (pa *PlatformAdapterDiscord) QuitGroup(ctx *MsgContext, id string) {
	//没有退出单个频道的功能，这里一旦退群退的就是整个服务器，所以可能会产生一些问题，慎用
	//另一个不建议使用此功能的原因是把discordBot重新拉回服务器很麻烦，需要去discord开发者页面再生成一遍链接
	ch, err := pa.IntentSession.Channel(ExtractDiscordUserId(id))
	if err != nil {
		pa.Session.Parent.Logger.Errorf("获取Discord频道#%s信息时出错:%s", id, err.Error())
		return
	}
	guildID := ch.GuildID
	err = pa.IntentSession.GuildLeave(guildID)
	if err != nil {
		pa.Session.Parent.Logger.Errorf("退出Discord群组#%s时出错:%s", guildID, err.Error())
		return
	}
}

// SetGroupCardName 没有改变用户在某个频道中昵称的功能，一旦更改就是整个服务器范围内都改
func (pa *PlatformAdapterDiscord) SetGroupCardName(groupId string, userId string, name string) {
	ch, err := pa.IntentSession.Channel(ExtractDiscordChannelId(groupId))
	if err != nil {
		pa.Session.Parent.Logger.Errorf("获取Discord频道#%s信息时出错:%s", groupId, err.Error())
		return
	}
	guildID := ch.GuildID
	err = pa.IntentSession.GuildMemberNickname(guildID, ExtractDiscordUserId(userId), name)
	if err != nil {
		pa.Session.Parent.Logger.Errorf("修改用户#%s在Discord服务器%s(来源频道#%s)的昵称时出错:%s", userId, guildID, groupId, err.Error())
	}
}

func (pa *PlatformAdapterDiscord) MemberBan(groupId string, userId string, duration int64) {

}

func (pa *PlatformAdapterDiscord) MemberKick(groupId string, userId string) {

}

//下面四个函数是格式化和反格式化的

func FormatDiceIdDiscord(diceDiscord string) string {
	return fmt.Sprintf("DISCORD:%s", diceDiscord)
}

func FormatDiceIdDiscordChannel(diceDiscord string) string {
	return fmt.Sprintf("DISCORD-CH-Group:%s", diceDiscord)
}
func FormatDiceIdDiscordGuild(diceDiscord string) string {
	return fmt.Sprintf("DISCORD-Guild:%s", diceDiscord)
}

func ExtractDiscordUserId(id string) string {
	if strings.HasPrefix(id, "DISCORD:") {
		return id[len("DISCORD:"):]
	}
	return id
}

func ExtractDiscordChannelId(id string) string {
	if strings.HasPrefix(id, "DISCORD-CH-Group:") {
		return id[len("DISCORD-CH-Group:"):]
	}
	return id
}

// 把discordgo的message转换成豹的message
func (pa *PlatformAdapterDiscord) toStdMessage(m *discordgo.MessageCreate) (*Message, error) {
	msg := new(Message)
	msg.Time = m.Timestamp.Unix()
	msg.Message = m.Content
	msg.RawId = m.ID
	msg.Platform = "DISCORD"
	ch, err := pa.IntentSession.Channel(m.ChannelID)
	if err != nil {
		pa.Session.Parent.Logger.Errorf("获取Discord频道#%s信息时出错:%s", FormatDiceIdDiscordChannel(m.ChannelID), err.Error())
		return nil, errors.New("")
	}
	if ch != nil && ch.Type == discordgo.ChannelTypeDM {
		msg.MessageType = "private"
	} else {
		msg.MessageType = "group"
		msg.GroupId = FormatDiceIdDiscordChannel(m.ChannelID)
		msg.GuildId = ch.GuildID
	}
	send := new(SenderBase)
	send.UserId = FormatDiceIdDiscord(m.Author.ID)
	send.Nickname = m.Author.Username
	if msg.MessageType == "group" && pa.checkIfGuildAdmin(m.Message) {
		send.GroupRole = "admin"
	}
	msg.Sender = *send
	return msg, nil
}

func (pa *PlatformAdapterDiscord) checkIfGuildAdmin(m *discordgo.Message) bool {
	p, err := pa.IntentSession.State.MessagePermissions(m)
	//pa.Session.Parent.Logger.Info(m.Author.Username, p)
	if err != nil {
		pa.Session.Parent.Logger.Errorf("鉴权时出现错误，\n发送者:%s，\n频道:%s，\n服务器id:%s，\n消息id:%s，\n时间:%s，\n消息内容:\"%s\"，\n错误详情:%s",
			FormatDiceIdDiscord(m.Author.ID),
			FormatDiceIdDiscordChannel(m.ChannelID),
			m.GuildID,
			m.ID,
			strconv.FormatInt(m.Timestamp.Unix(), 10),
			m.Content,
			err.Error())
		return false
	}
	//https://discord.com/developers/docs/topics/permissions
	//KICK_MEMBERS *	0x0000000000000002 (1 << 1)
	//BAN_MEMBERS *	0x0000000000000004 (1 << 2)	Allows banning members
	//ADMINISTRATOR *	0x0000000000000008 (1 << 3)	Allows all permissions and bypasses channel permission overwrites
	return p&(1<<1|1<<2|1<<3) > 0 || p == discordgo.PermissionAll
}
