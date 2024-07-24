package dice

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/slack-go/slack"
	se "github.com/slack-go/slack/slackevents"
	sm "github.com/slack-go/slack/socketmode"

	"sealdice-core/message"
	"sealdice-core/utils/syncmap"
)

type PlatformAdapterSlack struct {
	Session   *IMSession    `yaml:"-" json:"-"`
	EndPoint  *EndPointInfo `yaml:"-" json:"-"`
	Client    *sm.Client    `yaml:"-" json:"-"`
	BotToken  string        `yaml:"botToken" json:"botToken"`
	AppToken  string        `yaml:"appToken" json:"appToken"`
	cancel    func()
	userCache *syncmap.SyncMap[string, *slack.User]
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
		test, err := api.AuthTest()
		if err != nil {
			log.Error("Slack 测试连接失败，需要重新登录：", err.Error())
			return
		}
		log.Infof("Slack 连接成功：账号<%s>(%s)", test.User, FormatDiceIDSlack(test.UserID))
		pa.EndPoint.UserID = FormatDiceIDSlack(test.UserID)
		pa.EndPoint.Nickname = test.User
		ep.State = 1
		ep.Enable = true
	})
	sh.Handle(sm.EventTypeConnectionError, func(event *sm.Event, client *sm.Client) {
		ep.State = 0
		log.Errorf("Slack 账号 <%s> 连接失败: %v", pa.EndPoint.UserID, event.Data)
	})
	sh.Handle(sm.EventTypeDisconnect, func(event *sm.Event, client *sm.Client) {
		ep.State = 0
		log.Errorf("Slack 账号 <%s> 连接断开：%v", pa.EndPoint.UserID, event.Data)
	})
	sh.HandleEvents(se.AppMention, func(event *sm.Event, client *sm.Client) {
		go client.Ack(*event.Request)
		e := event.Data.(se.EventsAPIEvent)
		m := e.InnerEvent.Data.(*se.AppMentionEvent)
		u := pa.getUser(m.User)
		msg := &Message{
			GuildID:     FormatDiceIDSlackGuild(e.TeamID),
			GroupID:     FormatDiceIDSlackChannel(m.Channel),
			Message:     m.Text,
			MessageType: "group",
			Sender: SenderBase{
				UserID:   FormatDiceIDSlack(m.User),
				Nickname: u.Name,
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
			RawID:    e.Token,
			Time: func() int64 {
				i, err := strconv.ParseInt(m.TimeStamp, 10, 64)
				if err != nil {
					return time.Now().Unix()
				}
				return i
			}(),
		}
		s.Execute(ep, msg, false)
	})
	// Message
	sh.HandleEvents(se.Message, func(event *sm.Event, client *sm.Client) {
		go client.Ack(*event.Request)
		e := event.Data.(se.EventsAPIEvent)
		m := e.InnerEvent.Data.(*se.MessageEvent)
		u := pa.getUser(m.User)
		msg := &Message{
			GuildID: FormatDiceIDSlackGuild(e.TeamID),
			GroupID: FormatDiceIDSlackChannel(m.Channel),
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
				UserID: FormatDiceIDSlack(m.User),
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
			RawID:    m.ClientMsgID,
			Time: func() int64 {
				i, err := strconv.ParseInt(m.TimeStamp, 10, 64)
				if err != nil {
					return time.Now().Unix()
				}
				return i
			}(),
		}
		s.Execute(ep, msg, false)
	})
	// Other
	sh.HandleEvents(se.AppHomeOpened, func(event *sm.Event, client *sm.Client) {

	})
	// Start
	pa.Client = client
	ctx, cancel := context.WithCancel(context.Background())
	pa.cancel = cancel
	err := sh.RunEventLoopContext(ctx)
	if err != nil {
		log.Error("SlackEventLoopErr：", err.Error())
		return 1
	}
	return 0
}

func (pa *PlatformAdapterSlack) SendFileToPerson(ctx *MsgContext, userID string, path string, flag string) {
	// TODO
	fileElement, err := message.FilepathToFileElement(path)
	if err == nil {
		pa.SendToPerson(ctx, userID, fmt.Sprintf("[尝试发送文件: %s，但不支持]", fileElement.File), flag)
	} else {
		pa.SendToPerson(ctx, userID, fmt.Sprintf("[尝试发送文件出错: %s]", err.Error()), flag)
	}
}

func (pa *PlatformAdapterSlack) SendFileToGroup(ctx *MsgContext, groupID string, path string, flag string) {
	// TODO
	fileElement, err := message.FilepathToFileElement(path)
	if err == nil {
		pa.SendToGroup(ctx, groupID, fmt.Sprintf("[尝试发送文件: %s，但不支持]", fileElement.File), flag)
	} else {
		pa.SendToGroup(ctx, groupID, fmt.Sprintf("[尝试发送文件出错: %s]", err.Error()), flag)
	}
}

func (pa *PlatformAdapterSlack) DoRelogin() bool {
	if pa.cancel != nil {
		pa.cancel()
	}
	pa.Client = nil
	pa.EndPoint.Enable = false
	pa.EndPoint.State = 0
	go pa.Serve()
	return true
}

func (pa *PlatformAdapterSlack) SetEnable(enable bool) {
	if enable {
		if pa.Client == nil {
			go pa.Serve()
		} else {
			pa.Client = nil
			pa.cancel = nil
			go pa.Serve()
		}
	} else {
		if pa.cancel != nil {
			pa.cancel()
		}
		pa.Client = nil
		pa.EndPoint.Enable = false
		pa.EndPoint.State = 0
	}
}

func (pa *PlatformAdapterSlack) QuitGroup(ctx *MsgContext, id string) {
	pa.Session.Parent.Logger.Error("Slack 退出群组失败：暂不支持")
}

func (pa *PlatformAdapterSlack) SendSegmentToGroup(ctx *MsgContext, groupID string, msg []message.IMessageElement, flag string) {
}

func (pa *PlatformAdapterSlack) SendSegmentToPerson(ctx *MsgContext, userID string, msg []message.IMessageElement, flag string) {
}

func (pa *PlatformAdapterSlack) SendToPerson(ctx *MsgContext, userID string, text string, flag string) {
	pa.send(ctx, ExtractSlackUserID(userID), text, flag)
	pa.Session.OnMessageSend(ctx, &Message{
		MessageType: "private",
		Platform:    "SLACK",
		Message:     text,
		Sender: SenderBase{
			UserID:   pa.EndPoint.UserID,
			Nickname: pa.EndPoint.Nickname,
		},
	}, flag)
}

func (pa *PlatformAdapterSlack) SendToGroup(ctx *MsgContext, groupID string, text string, flag string) {
	pa.send(ctx, ExtractSlackChannelID(groupID), text, flag)
	pa.Session.OnMessageSend(ctx, &Message{
		MessageType: "group",
		Platform:    "SLACK",
		Message:     text,
		GroupID:     groupID,
		Sender: SenderBase{
			UserID:   pa.EndPoint.UserID,
			Nickname: pa.EndPoint.Nickname,
		},
	}, flag)
}

func (pa *PlatformAdapterSlack) SetGroupCardName(ctx *MsgContext, name string) {
	pa.Session.Parent.Logger.Error("Slack 设置群名片失败：暂不支持")
}

func (pa *PlatformAdapterSlack) MemberBan(groupId string, userId string, duration int64) {

}

func (pa *PlatformAdapterSlack) MemberKick(groupId string, userId string) {

}

func (pa *PlatformAdapterSlack) GetGroupInfoAsync(groupId string) {
	// TODO
}

func (pa *PlatformAdapterSlack) EditMessage(_ *MsgContext, _, _ string) {}

func (pa *PlatformAdapterSlack) RecallMessage(_ *MsgContext, _ string) {}

func (pa *PlatformAdapterSlack) send(_ *MsgContext, id string, text string, _ string) {
	// pa.Client.PostMessage 没看懂 Post 和 Send 有什么区别 先用语义更好的一个好了
	// 频道以 C 开头 用户以 U 开头 老粗暴了
	message, s, s2, err := pa.Client.SendMessage(id, slack.MsgOptionText(text, false))
	if err != nil {
		pa.Session.Parent.Logger.Error("Slack 发送消息失败", message, s, s2, err.Error())
	}
}

func (pa *PlatformAdapterSlack) getUser(user string) *slack.User {
	if pa.userCache == nil {
		pa.userCache = syncmap.NewSyncMap[string, *slack.User]()
	}
	if u, ok := pa.userCache.Load(user); ok {
		return u
	}
	info, err := pa.Client.GetUserInfo(user)
	if err != nil {
		pa.Session.Parent.Logger.Error("Slack 获取用户数据失败：", err.Error())
		return &slack.User{}
	}
	pa.userCache.Store(user, info)
	return info
}

func (pa *PlatformAdapterSlack) formatOneBot11(m *se.MessageEvent) string {
	return m.Text
}

// 格式化

func FormatDiceIDSlack(id string) string {
	return fmt.Sprintf("SLACK:%s", id)
}

func FormatDiceIDSlackChannel(id string) string {
	return fmt.Sprintf("SLACK-CH-Group:%s", id)
}
func FormatDiceIDSlackGuild(id string) string {
	return fmt.Sprintf("SLACK-Guild:%s", id)
}

func ExtractSlackUserID(id string) string {
	if strings.HasPrefix(id, "SLACK:") {
		return id[len("SLACK:"):]
	}
	return id
}

func ExtractSlackChannelID(id string) string {
	if strings.HasPrefix(id, "SLACK-CH-Group:") {
		return id[len("SLACK-CH-Group:"):]
	}
	return id
}
