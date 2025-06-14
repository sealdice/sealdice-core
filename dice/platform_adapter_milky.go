package dice

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	milky "github.com/Szzrain/Milky-go-sdk"

	"sealdice-core/message"
	log "sealdice-core/utils/kratos"
)

type PlatformAdapterMilky struct {
	Session       *IMSession     `yaml:"-" json:"-"`
	EndPoint      *EndPointInfo  `yaml:"-" json:"-"`
	WsGateway     string         `yaml:"ws_gateway" json:"ws_gateway"`
	RestGateway   string         `yaml:"rest_gateway" json:"rest_gateway"`
	Token         string         `yaml:"token" json:"token"` // 暂时没支持
	IntentSession *milky.Session `yaml:"-" json:"-"`
}

type loggerWrapper struct{}

func (l *loggerWrapper) Log(level milky.Level, keyvals ...interface{}) error {
	log.Log(log.Level(level), keyvals...)
	return nil
}

func (pa *PlatformAdapterMilky) SendSegmentToGroup(ctx *MsgContext, groupID string, msg []message.IMessageElement, flag string) {
}

func (pa *PlatformAdapterMilky) SendSegmentToPerson(ctx *MsgContext, userID string, msg []message.IMessageElement, flag string) {
}

func (pa *PlatformAdapterMilky) GetGroupInfoAsync(_ string) {}

func (pa *PlatformAdapterMilky) Serve() int {
	pa.EndPoint.State = 2 // 设置状态为连接中
	session, err := milky.New(pa.WsGateway, pa.RestGateway, &loggerWrapper{})
	if err != nil {
		log.Errorf("Milky SDK initialization failed: %v", err)
		return 1
	}
	pa.IntentSession = session
	session.AddHandler(func(session2 *milky.Session, m *milky.ReceiveMessage) {
		if m == nil {
			return
		}
		log.Debugf("Received message: Sender %d", m.SenderId)
		msg := &Message{
			Platform: "QQ",
			Time:     m.Time,
			RawID:    m.MessageSeq,
			Sender: SenderBase{
				UserID: FormatDiceIDQQ(strconv.FormatInt(m.SenderId, 10)),
			},
		}
		if m.MessageScene == "group" {
			if m.Group != nil || m.GroupMember != nil {
				msg.GroupID = FormatDiceIDQQGroup(strconv.FormatInt(m.Group.GroupId, 10))
				msg.MessageType = "group"
				msg.GroupName = m.Group.Name
				msg.Sender.GroupRole = m.GroupMember.Role
				msg.Sender.Nickname = m.GroupMember.Nickname
			} else {
				log.Warnf("Received group message without group info: %v", m)
				return // 无法处理的消息
			}
		} else if m.MessageScene == "friend" {
			if m.Friend != nil {
				msg.MessageType = "private"
				msg.Sender.Nickname = m.Friend.Nickname
			} else {
				log.Warnf("Received friend message without friend info: %v", m)
				return // 无法处理的消息
			}
		} else {
			return // 临时对话消息，不处理
		}
		if m.Segments != nil {
			for _, segment := range m.Segments {
				switch seg := segment.(type) {
				case *milky.TextElement:
					log.Debugf(" Text: %s", seg.Text)
					msg.Segment = append(msg.Segment, &message.TextElement{
						Content: seg.Text,
					})
				case *milky.ImageElement:
					log.Debugf(" Image: %s", seg.TempURL)
					msg.Segment = append(msg.Segment, &message.ImageElement{
						URL: seg.TempURL,
					})
				case *milky.AtElement:
					log.Debugf(" At: %d", seg.UserID)
					msg.Segment = append(msg.Segment, &message.AtElement{
						Target: strconv.FormatInt(seg.UserID, 10),
					})
				case *milky.ReplyElement:
					log.Debugf(" Reply to message ID: %d", seg.MessageSeq)
					msg.Segment = append(msg.Segment, &message.ReplyElement{
						ReplySeq: strconv.FormatInt(seg.MessageSeq, 10),
					})
				default:
					log.Debugf("Unknown segment type: %T", segment)
				}
			}
		}
		pa.Session.ExecuteNew(pa.EndPoint, msg)
	})
	err = pa.IntentSession.Open()
	if err != nil {
		log.Errorf("Failed to open Milky session: %v", err)
		return 1
	}
	info, err := session.GetLoginInfo()
	if err != nil {
		log.Errorf("Failed to get login info: %v", err)
	} else {
		log.Infof("Milky login info: UserId %d, Nickname %s", info.UIN, info.Nickname)
		pa.EndPoint.UserID = fmt.Sprintf("QQ:%d", info.UIN)
		pa.EndPoint.Nickname = info.Nickname
	}
	pa.EndPoint.State = 1
	pa.EndPoint.Enable = true
	return 0
}

func (pa *PlatformAdapterMilky) DoRelogin() bool {
	if pa.IntentSession != nil {
		log.Infof("Reconnecting Milky session...")
		if err := pa.IntentSession.Close(); err != nil {
			log.Errorf("Failed to close Milky session: %v", err)
		}
		if err := pa.IntentSession.Open(); err != nil {
			log.Errorf("Failed to reopen Milky session: %v", err)
			return false
		}
		log.Infof("Milky session reconnected successfully")
		return true
	} else {
		log.Warnf("No Milky session to reconnect, reinitializing...")
		if pa.Serve() != 0 {
			log.Errorf("Failed to reinitialize Milky session")
			return false
		}
		log.Infof("Milky session reinitialized successfully")
		return true
	}
}

func (pa *PlatformAdapterMilky) SetEnable(enable bool) {
	if pa.EndPoint != nil {
		pa.EndPoint.Enable = true
	}
	if pa.IntentSession != nil {
		if enable {
			if err := pa.IntentSession.Open(); err != nil {
				log.Errorf("Failed to open Milky session: %v", err)
			} else {
				log.Infof("Milky session opened successfully")
			}
		} else {
			if err := pa.IntentSession.Close(); err != nil {
				log.Errorf("Failed to close Milky session: %v", err)
			} else {
				log.Infof("Milky session closed successfully")
			}
		}
	}
}

func ParseMessageToMilky(send []message.IMessageElement) []milky.IMessageElement {
	var elements []milky.IMessageElement
	for _, elem := range send {
		switch e := elem.(type) {
		case *message.TextElement:
			elements = append(elements, &milky.TextElement{Text: e.Content})
		case *message.ImageElement:
			elements = append(elements, &milky.ImageElement{URI: e.URL})
		case *message.AtElement:
			log.Debugf("At user: %s", e.Target)
			if uid, err := strconv.ParseInt(e.Target, 10, 64); err == nil {
				elements = append(elements, &milky.AtElement{UserID: uid})
			}
		case *message.ReplyElement:
			if seq, err := strconv.ParseInt(e.ReplySeq, 10, 64); err == nil {
				elements = append(elements, &milky.ReplyElement{MessageSeq: seq})
			}
		default:
			log.Warnf("Unsupported message element type: %T", elem)
		}
	}
	return elements
}

func (pa *PlatformAdapterMilky) SendToPerson(ctx *MsgContext, uid string, text string, flag string) {
	send := message.ConvertStringMessage(text)
	elements := ParseMessageToMilky(send)
	id, err := strconv.ParseInt(ExtractQQUserID(uid), 10, 64)
	if err != nil {
		log.Errorf("Invalid user ID %s: %v", uid, err)
		return
	}
	_, err = pa.IntentSession.SendPrivateMessage(id, &elements)
	if err != nil {
		log.Errorf("Failed to send private message to %s: %v", uid, err)
		return
	}
	pa.Session.OnMessageSend(ctx, &Message{
		Platform:    "QQ",
		MessageType: "private",
		Message:     text,
		Sender: SenderBase{
			UserID:   pa.EndPoint.UserID,
			Nickname: pa.EndPoint.Nickname,
		},
	}, flag)
}

func (pa *PlatformAdapterMilky) SendToGroup(ctx *MsgContext, groupID string, text string, flag string) {
	send := message.ConvertStringMessage(text)
	elements := ParseMessageToMilky(send)
	id, err := strconv.ParseInt(ExtractQQGroupID(groupID), 10, 64)
	if err != nil {
		log.Errorf("Invalid group ID %s: %v", groupID, err)
		return
	}
	_, err = pa.IntentSession.SendGroupMessage(id, &elements)
	if err != nil {
		log.Errorf("Failed to send group message to %s: %v", groupID, err)
		return
	}
	pa.Session.OnMessageSend(ctx, &Message{
		Platform:    "QQ",
		MessageType: "group",
		Message:     text,
		Sender: SenderBase{
			UserID:   pa.EndPoint.UserID,
			Nickname: pa.EndPoint.Nickname,
		},
	}, flag)
}

func (pa *PlatformAdapterMilky) SendFileToPerson(ctx *MsgContext, uid string, path string, flag string) {
	pa.SendToPerson(ctx, uid, fmt.Sprintf("[尝试发送文件: %s，但不支持]", filepath.Base(path)), flag)
}

func (pa *PlatformAdapterMilky) SendFileToGroup(ctx *MsgContext, uid string, path string, flag string) {
	pa.SendToGroup(ctx, uid, fmt.Sprintf("[尝试发送文件: %s，但不支持]", filepath.Base(path)), flag)
}

func (pa *PlatformAdapterMilky) QuitGroup(_ *MsgContext, _ string) {}

func (pa *PlatformAdapterMilky) SetGroupCardName(_ *MsgContext, _ string) {}

func (pa *PlatformAdapterMilky) MemberBan(_ string, _ string, _ int64) {}

func (pa *PlatformAdapterMilky) MemberKick(_ string, _ string) {}

func (pa *PlatformAdapterMilky) EditMessage(_ *MsgContext, _, _ string) {}

func (pa *PlatformAdapterMilky) RecallMessage(_ *MsgContext, _ string) {}

func ExtractQQUserID(id string) string {
	if strings.HasPrefix(id, "QQ:") {
		return id[len("QQ:"):]
	}
	return id
}

func ExtractQQGroupID(id string) string {
	if strings.HasPrefix(id, "QQ-Group:") {
		return id[len("QQ-Group:"):]
	}
	return id
}
