package dice

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"strings"
	"time"
)

type PlatformAdapterDiscord struct {
	DiceServing   bool               `yaml:"-" json:"-"`
	Session       *IMSession         `yaml:"-" json:"-"`
	Token         string             `yaml:"token" json:"token"`
	EndPoint      *EndPointInfo      `yaml:"-" json:"-"`
	IntentSession *discordgo.Session `yaml:"-" json:"-"`
}

func (pa *PlatformAdapterDiscord) GetGroupInfoAsync(groupId string) {
	pa.updateChannelNum()
	logger := pa.Session.Parent.Logger
	dm := pa.Session.Parent.Parent
	channel, err := pa.IntentSession.Channel(ExtractDiscordChannelId(groupId))
	if err != nil {
		logger.Errorf("获取Discord频道信息#{%s}时出错:{%s}", groupId, err.Error())
		return
	}
	dm.GroupNameCache.Set(groupId, &GroupNameCacheItem{
		channel.Name,
		time.Now().Unix(),
	})
	group := pa.Session.ServiceAtNew[groupId]
	if group != nil {
		group.GroupName = channel.Name
	}
}

func (pa *PlatformAdapterDiscord) updateChannelNum() {
	guilds := pa.IntentSession.State.Guilds
	GroupNum := 0
	for _, guild := range guilds {
		channels, _ := pa.IntentSession.GuildChannels(guild.ID)
		GroupNum += len(channels)
	}
	pa.EndPoint.GroupNum = int64(GroupNum)
}

func (pa *PlatformAdapterDiscord) Serve() int {
	dg, err := discordgo.New("Bot " + pa.Token)
	if err != nil {
		pa.Session.Parent.Logger.Error("创建DiscordSession时出错:", err)
		return 1
	}
	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		//忽略自己的消息
		if m.Author.ID == s.State.User.ID {
			return
		}
		pa.Session.Execute(pa.EndPoint, toStdMessage(m), false)
	})
	dg.Identify.Intents = discordgo.IntentsGuildMessages
	pa.IntentSession = dg
	err = dg.Open()
	if err != nil {
		pa.Session.Parent.Logger.Error("与Discord服务进行连接时出错:", err)
		return 1
	}
	pa.EndPoint.UserId = FormatDiceIdDiscord(pa.IntentSession.State.User.ID)
	pa.EndPoint.Nickname = pa.IntentSession.State.User.Username
	pa.EndPoint.State = 1
	pa.EndPoint.Enable = true
	return 0
}

func (pa *PlatformAdapterDiscord) DoRelogin() bool {
	if pa.IntentSession == nil {
		success := pa.Serve()
		return success == 0
	}
	_ = pa.IntentSession.Close()
	success := pa.IntentSession.Open()
	return success == nil
}

func (pa *PlatformAdapterDiscord) SetEnable(enable bool) {
	if enable {
		pa.Session.Parent.Logger.Infof("正在启用Discord服务……")
		if pa.IntentSession == nil {
			pa.Serve()
			return
		}
		err := pa.IntentSession.Open()
		if err != nil {
			pa.Session.Parent.Logger.Error("与Discord服务进行连接时出错:", err)
			pa.EndPoint.State = 0
			pa.EndPoint.Enable = false
			return
		}
		pa.EndPoint.State = 1
		pa.EndPoint.Enable = true
	} else {
		pa.EndPoint.State = 0
		pa.EndPoint.Enable = false
		_ = pa.IntentSession.Close()
	}
}

func (pa *PlatformAdapterDiscord) SendToPerson(ctx *MsgContext, uid string, text string, flag string) {
	is := pa.IntentSession
	ch, _ := is.UserChannelCreate(ExtractDiscordUserId(uid))
	_, err := is.ChannelMessageSend(ch.ID, text)
	if err != nil {
		pa.Session.Parent.Logger.Errorf("向Discord用户#{%s}发送消息时出错:{%s}", uid, err)
		return
	}
}

func (pa *PlatformAdapterDiscord) SendToGroup(ctx *MsgContext, uid string, text string, flag string) {
	_, err := pa.IntentSession.ChannelMessageSend(ExtractDiscordChannelId(uid), text)
	if err != nil {
		pa.Session.Parent.Logger.Errorf("向Discord频道#{%s}发送消息时出错:{%s}", uid, err)
		return
	}
}

func (pa *PlatformAdapterDiscord) QuitGroup(ctx *MsgContext, id string) {
	ch, _ := pa.IntentSession.Channel(ExtractDiscordUserId(id))
	group := ch.GuildID
	err := pa.IntentSession.GuildLeave(group)
	if err != nil {
		return
	}
}

func (pa *PlatformAdapterDiscord) SetGroupCardName(groupId string, userId string, name string) {}

func FormatDiceIdDiscord(diceDiscord string) string {
	return fmt.Sprintf("DISCORD:%s", diceDiscord)
}

func FormatDiceIdDiscordChannel(diceDiscord string) string {
	return fmt.Sprintf("DISCORD-CH-Group:%s", diceDiscord)
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

func toStdMessage(m *discordgo.MessageCreate) *Message {
	msg := new(Message)
	msg.Time = m.Timestamp.Unix()
	msg.Message = m.Content
	msg.RawId = m.ID
	msg.GroupId = FormatDiceIdDiscordChannel(m.ChannelID)
	msg.Platform = "Discord"
	msg.MessageType = "group"
	send := new(SenderBase)
	send.UserId = FormatDiceIdDiscord(m.Author.ID)
	send.Nickname = m.Author.Username
	msg.Sender = *send
	return msg
}
