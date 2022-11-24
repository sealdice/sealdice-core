package dice

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
)

type PlatformAdapterDiscord struct {
	Session       *IMSession
	Token         string        `yaml:"token" json:"token"`
	EndPoint      *EndPointInfo `yaml:"-" json:"-"`
	IntentSession *discordgo.Session
}

func (pa *PlatformAdapterDiscord) GetGroupInfoAsync(groupId string) {}

func (pa *PlatformAdapterDiscord) Serve() int {
	dg, err := discordgo.New("Bot " + pa.Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return 1
	}
	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		//忽略自己的消息
		if m.Author.ID == s.State.User.ID {
			return
		}
		pa.Session.Execute(pa.EndPoint, toStdMessage(m), false)
	})

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.IntentsGuildMessages
	pa.IntentSession = dg
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
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
	pa.IntentSession.ChannelMessageSend(uid, text)
}

func (pa *PlatformAdapterDiscord) QuitGroup(ctx *MsgContext, id string) {}

func (pa *PlatformAdapterDiscord) SetGroupCardName(groupId string, userId string, name string) {}

func toStdMessage(m *discordgo.MessageCreate) *Message {
	msg := new(Message)
	msg.Time = m.Timestamp.Unix()
	msg.Message = m.Content
	msg.RawId = m.ID
	msg.GroupId = m.ChannelID
	msg.Platform = "Discord"
	send := new(SenderBase)
	send.UserId = m.Author.ID
	send.Nickname = m.Author.Username
	msg.Sender = *send
	return msg
}
