package dice

import (
	"fmt"
	"github.com/slack-go/slack"
	se "github.com/slack-go/slack/slackevents"
	sm "github.com/slack-go/slack/socketmode"
	"strconv"
	"strings"
	"time"
)

type PlatformAdapterSlack struct {
	Session   *IMSession    `yaml:"-" json:"-"`
	EndPoint  *EndPointInfo `yaml:"-" json:"-"`
	Client    *sm.Client    `yaml:"-" json:"-"`
	BotToken  string        `yaml:"botToken" json:"botToken"`
	AppToken  string        `yaml:"appToken" json:"appToken"`
	userCache *SyncMap[string, *slack.User]
	// msgCache  *SyncMap[string, int]
}

func (pa *PlatformAdapterSlack) Serve() int {
	ep := pa.EndPoint
	s := pa.Session
	log := s.Parent.Logger
	api := slack.New(pa.BotToken, slack.OptionAppLevelToken(pa.AppToken))
	client := sm.New(api)
	sh := sm.NewSocketmodeHandler(client)
	// Connect
	sh.Handle(sm.EventTypeConnecting, func(event *sm.Event, client *sm.Client) {
		ep.State = 2
		log.Info("使用 Socket Mode/套接字模式 连接到 Slack 中")
	})
	sh.Handle(sm.EventTypeConnected, func(event *sm.Event, client *sm.Client) {
		ep.State = 1
		ep.Enable = true
		log.Info("Slack 连接成功")
	})
	sh.Handle(sm.EventTypeConnectionError, func(event *sm.Event, client *sm.Client) {
		ep.State = 0
		log.Error("连接失败：", event.Data)
	})
	sh.Handle(sm.EventTypeDisconnect, func(event *sm.Event, client *sm.Client) {
		ep.State = 0
		log.Error("连接断开：", event.Data)
	})
	// Message
	sh.HandleEvents(se.Message, func(event *sm.Event, client *sm.Client) {
		e := event.Data.(se.EventsAPIEvent)
		m := e.InnerEvent.Data.(*se.MessageEvent)
		u := pa.getUser(m.User)
		msg := &Message{
			GuildId: FormatDiceIdSlackGuild(e.TeamID),
			GroupId: FormatDiceIdSlackChannel(m.Channel),
			Message: pa.formatOneBot11(m),
			MessageType: func() string {
				// https://api.slack.com/events/message#events_api
				if m.ChannelType == "im" {
					return "private"
				} else {
					return "group"
				}
			}(),
			Sender: SenderBase{
				UserId: FormatDiceIdSlack(m.User),
				Nickname: func() string { // 不知道为什么为空，以后再看看库的源码
					if m.Username == "" {
						return u.Name
					} else {
						return m.Username
					}
				}(),
				GroupRole: func() string {
					if u.IsOwner {
						return "owner"
					}
					if u.IsAdmin {
						return "admin"
					}
					return ""
				}(),
			},
			Platform: pa.EndPoint.Platform,
			RawId:    m.ClientMsgID,
			Time: func() int64 {
				i, err := strconv.ParseInt(m.TimeStamp, 10, 64)
				if err != nil {
					return time.Now().Unix()
				}
				return i
			}(),
		}
		s.Execute(ep, msg, false)
		client.Ack(*event.Request) // 必须回应一下 不然 Slack 会重发 3 次事件看你死还是没死
	})
	// Other
	sh.HandleEvents(se.AppHomeOpened, func(event *sm.Event, client *sm.Client) {

	})
	// Start
	pa.Client = client
	err := sh.RunEventLoop()
	if err != nil {
		log.Error("SlackEventLoopErr：", err.Error())
		return 1
	}
	return 0
}

func (pa *PlatformAdapterSlack) DoRelogin() bool {
	return true
}

func (pa *PlatformAdapterSlack) SetEnable(enable bool) {

}

func (pa *PlatformAdapterSlack) QuitGroup(ctx *MsgContext, id string) {

}

func (pa *PlatformAdapterSlack) SendToPerson(ctx *MsgContext, uid string, text string, flag string) {
	pa.send(ctx, ExtractSlackUserId(uid), text, flag)
}

func (pa *PlatformAdapterSlack) SendToGroup(ctx *MsgContext, cid string, text string, flag string) {
	pa.send(ctx, ExtractSlackChannelId(cid), text, flag)
}
func (pa *PlatformAdapterSlack) SetGroupCardName(groupId string, userId string, name string) {

}

func (pa *PlatformAdapterSlack) MemberBan(groupId string, userId string, duration int64) {

}

func (pa *PlatformAdapterSlack) MemberKick(groupId string, userId string) {

}

func (pa *PlatformAdapterSlack) GetGroupInfoAsync(groupId string) {

}

func (pa *PlatformAdapterSlack) send(ctx *MsgContext, id string, text string, flag string) {
	// pa.Client.PostMessage 没看懂 Post 和 Send 有什么区别 先用语义更好的一个好了
	// 频道以 C 开头 用户以 U 开头 老粗暴了
	message, s, s2, err := pa.Client.SendMessage(id, slack.MsgOptionText(text, false))
	if err != nil {
		pa.Session.Parent.Logger.Error("发送失败", message, s, s2, err.Error())
	}
}

func (pa *PlatformAdapterSlack) getUser(user string) *slack.User {
	if pa.userCache == nil {
		pa.userCache = new(SyncMap[string, *slack.User])
	}
	if u, e := pa.userCache.Load(user); e {
		return u
	}
	info, err := pa.Client.GetUserInfo(user)
	if err != nil {
		pa.Session.Parent.Logger.Error("获取用户数据失败：", err.Error())
		return &slack.User{}
	}
	pa.userCache.Store(user, info)
	return info
}

func (pa *PlatformAdapterSlack) formatOneBot11(m *se.MessageEvent) string {
	return m.Text
}

// 格式化

func FormatDiceIdSlack(id string) string {
	return fmt.Sprintf("SLACK:%s", id)
}

func FormatDiceIdSlackChannel(id string) string {
	return fmt.Sprintf("SLACK-CH-Group:%s", id)
}
func FormatDiceIdSlackGuild(id string) string {
	return fmt.Sprintf("SLACK-Guild:%s", id)
}

func ExtractSlackUserId(id string) string {
	if strings.HasPrefix(id, "SLACK:") {
		return id[len("SLACK:"):]
	}
	return id
}

func ExtractSlackChannelId(id string) string {
	if strings.HasPrefix(id, "SLACK-CH-Group:") {
		return id[len("SLACK-CH-Group:"):]
	}
	return id
}
