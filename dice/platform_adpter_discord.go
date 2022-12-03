package dice

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"strconv"
)

type PlatformAdapterDiscord struct {
	DiceServing   bool               `yaml:"-" json:"-"`
	Session       *IMSession         `yaml:"-" json:"-"`
	Token         string             `yaml:"token" json:"token"`
	EndPoint      *EndPointInfo      `yaml:"-" json:"-"`
	IntentSession *discordgo.Session `yaml:"-" json:"-"`
}

func (pa *PlatformAdapterDiscord) GetGroupInfoAsync(groupId string) {
	return
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
	return 0
}

func (pa *PlatformAdapterDiscord) DoRelogin() bool {
	return false
}

func (pa *PlatformAdapterDiscord) SetEnable(enable bool) {}

func (pa *PlatformAdapterDiscord) SendToPerson(ctx *MsgContext, uid string, text string, flag string) {
}

func (pa *PlatformAdapterDiscord) SendToGroup(ctx *MsgContext, uid string, text string, flag string) {
	_, err := pa.IntentSession.ChannelMessageSend(uid, text)
	if err != nil {
		pa.Session.Parent.Logger.Errorf("向Discord频道#{%s}发送消息时出错:{%s}", uid, err)
		return
	}
}

func (pa *PlatformAdapterDiscord) QuitGroup(ctx *MsgContext, id string) {}

func (pa *PlatformAdapterDiscord) SetGroupCardName(groupId string, userId string, name string) {}

func FormatDiceIdDiscord(diceDiscord int64) string {
	return fmt.Sprintf("Discord:%s", strconv.FormatInt(diceDiscord, 10))
}

func FormatDiceIdDiscordGroup(diceDiscord string) string {
	return fmt.Sprintf("Discord-Channel:%s", diceDiscord)
}

func FormatDiceIdDiscordCh(userId string) string {
	return fmt.Sprintf("Discord-CH:%s", userId)
}

func FormatDiceIdDiscordChGroup(GuildId, ChannelId string) string {
	return fmt.Sprintf("Discord-CH-Channel:%s-%s", GuildId, ChannelId)
}

func toStdMessage(m *discordgo.MessageCreate) *Message {
	msg := new(Message)
	msg.Time = m.Timestamp.Unix()
	msg.Message = m.Content
	msg.RawId = m.ID
	msg.GroupId = m.ChannelID
	msg.Platform = "Discord"
	msg.MessageType = "group"
	send := new(SenderBase)
	//TODO:身份鉴权
	//send.GroupRole = "admin"
	send.UserId = m.Author.ID
	send.Nickname = m.Author.Username
	msg.Sender = *send
	return msg
}
