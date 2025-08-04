package dice

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	milky "github.com/Szzrain/Milky-go-sdk"

	"sealdice-core/dice/events"
	"sealdice-core/message"
	log "sealdice-core/utils/kratos"
)

type PlatformAdapterMilky struct {
	Session       *IMSession     `json:"-"            yaml:"-"`
	EndPoint      *EndPointInfo  `json:"-"            yaml:"-"`
	WsGateway     string         `json:"ws_gateway"   yaml:"ws_gateway"`
	RestGateway   string         `json:"rest_gateway" yaml:"rest_gateway"`
	Token         string         `json:"token"        yaml:"token"`
	IntentSession *milky.Session `json:"-"            yaml:"-"`
}

func (pa *PlatformAdapterMilky) SendSegmentToGroup(ctx *MsgContext, groupID string, msg []message.IMessageElement, flag string) {
	id, err := strconv.ParseInt(ExtractQQGroupID(groupID), 10, 64)
	if err != nil {
		log.Errorf("Invalid group ID %s: %v", groupID, err)
		return
	}
	elements := ParseMessageToMilky(msg)
	ret, err := pa.IntentSession.SendGroupMessage(id, &elements)
	if err != nil {
		log.Errorf("Failed to send group message to %s: %v", groupID, err)
		return
	}
	pa.Session.OnMessageSend(ctx, &Message{
		Platform:    "QQ",
		MessageType: "group",
		Segment:     msg,
		Sender: SenderBase{
			UserID:   pa.EndPoint.UserID,
			Nickname: pa.EndPoint.Nickname,
		},
		RawID: ret.MessageSeq,
	}, flag)
}

func (pa *PlatformAdapterMilky) SendSegmentToPerson(ctx *MsgContext, userID string, msg []message.IMessageElement, flag string) {
	id, err := strconv.ParseInt(ExtractQQUserID(userID), 10, 64)
	if err != nil {
		log.Errorf("Invalid user ID %s: %v", userID, err)
		return
	}
	elements := ParseMessageToMilky(msg)
	ret, err := pa.IntentSession.SendPrivateMessage(id, &elements)
	if err != nil {
		log.Errorf("Failed to send private message to %s: %v", userID, err)
		return
	}
	pa.Session.OnMessageSend(ctx, &Message{
		Platform:    "QQ",
		MessageType: "private",
		Segment:     msg,
		Sender: SenderBase{
			UserID:   pa.EndPoint.UserID,
			Nickname: pa.EndPoint.Nickname,
		},
		RawID: ret.MessageSeq,
	}, flag)
}

func (pa *PlatformAdapterMilky) GetGroupInfoAsync(groupID string) {
	id, err := strconv.ParseInt(ExtractQQGroupID(groupID), 10, 64)
	if err != nil {
		log.Errorf("Invalid group ID %s: %v", groupID, err)
		return
	}
	groupInfo, err := pa.IntentSession.GetGroupInfo(id, true)
	if err != nil {
		log.Errorf("Failed to get group info for %s: %v", groupID, err)
		return
	}
	if groupInfo == nil {
		log.Warnf("Group info for %s is nil", groupID)
		return
	}
	pa.Session.Parent.Parent.GroupNameCache.Store(groupID, &GroupNameCacheItem{
		Name: groupInfo.Name,
		time: time.Now().Unix(),
	})
}

func (pa *PlatformAdapterMilky) Serve() int {
	pa.EndPoint.State = 2 // 设置状态为连接中

	if pa.RestGateway[len(pa.RestGateway)-1] == '/' {
		pa.RestGateway = pa.RestGateway[:len(pa.RestGateway)-1] // 去掉末尾的斜杠
	}
	if pa.WsGateway[len(pa.WsGateway)-1] == '/' {
		pa.WsGateway = pa.WsGateway[:len(pa.WsGateway)-1]
	}
	session, err := milky.New(pa.WsGateway, pa.RestGateway, pa.Token, log.NewHelper(log.GetLogger()))
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
		if msg.Sender.UserID == pa.EndPoint.UserID {
			log.Debugf("Ignoring self message: %v", m)
			return // 忽略自己的消息
		}
		switch m.MessageScene {
		case "group":
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
		case "friend":
			if m.Friend != nil {
				msg.MessageType = "private"
				msg.Sender.Nickname = m.Friend.Nickname
			} else {
				log.Warnf("Received friend message without friend info: %v", m)
				return // 无法处理的消息
			}
		default:
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
		if len(msg.Segment) == 0 {
			return // 如果没有消息内容，忽略
		}
		pa.Session.ExecuteNew(pa.EndPoint, msg)
	})
	session.AddHandler(func(session2 *milky.Session, m *milky.GroupNudge) {
		if m == nil {
			return
		}
		log.Debugf("Received group nudge: Group %d, Sender %d", m.GroupID, m.SenderID)
		msg := &Message{
			Platform:    "QQ",
			GroupID:     FormatDiceIDQQGroup(strconv.FormatInt(m.GroupID, 10)),
			MessageType: "group",
			Sender: SenderBase{
				UserID: FormatDiceIDQQ(strconv.FormatInt(m.SenderID, 10)),
			},
		}
		pa.Session.OnPoke(CreateTempCtx(pa.EndPoint, msg), &events.PokeEvent{
			GroupID:   msg.GroupID,
			SenderID:  msg.Sender.UserID,
			TargetID:  FormatDiceIDQQ(strconv.FormatInt(m.ReceiverID, 10)),
			IsPrivate: false,
		})
	})
	session.AddHandler(func(session2 *milky.Session, m *milky.GroupMemberDecrease) {
		if m == nil {
			return
		}
		log.Debugf("Group member decrease: Group %d, User %d, Operator %d", m.GroupID, m.UserID, m.OperatorID)
		msg := &Message{
			Platform:    "QQ",
			GroupID:     FormatDiceIDQQGroup(strconv.FormatInt(m.GroupID, 10)),
			MessageType: "group",
			Sender: SenderBase{
				UserID: FormatDiceIDQQ(strconv.FormatInt(m.OperatorID, 10)),
			},
		}
		if FormatDiceIDQQ(strconv.FormatInt(m.UserID, 10)) == pa.EndPoint.UserID {
			log.Infof("Bot has left group %s", msg.GroupID)
			if m.OperatorID == 0 {
				log.Debugf("Bot left group %s without an operator ID, treating as a normal leave", msg.GroupID)
				pa.Session.OnGroupLeave(CreateTempCtx(pa.EndPoint, msg), &events.GroupLeaveEvent{
					GroupID:    msg.GroupID,
					UserID:     pa.EndPoint.UserID,
					OperatorID: "",
				})
			} else {
				log.Debugf("Bot left group %s with operator ID %d", msg.GroupID, m.OperatorID)
				pa.Session.OnGroupLeave(CreateTempCtx(pa.EndPoint, msg), &events.GroupLeaveEvent{
					GroupID:    msg.GroupID,
					UserID:     pa.EndPoint.UserID,
					OperatorID: FormatDiceIDQQ(strconv.FormatInt(m.OperatorID, 10)),
				})
			}
		}
	})
	d := pa.Session.Parent
	err = pa.IntentSession.Open()
	if err != nil {
		log.Errorf("Failed to open Milky session: %v", err)
		pa.EndPoint.State = 3 // 设置状态为连接失败
		pa.EndPoint.Enable = false
		d.LastUpdatedTime = time.Now().Unix()
		d.Save(false)
		return 1
	}
	info, err := session.GetLoginInfo()
	if err != nil {
		// 获取登录信息失败，视为连接失败
		log.Errorf("Failed to get login info: %v", err)
		_ = pa.IntentSession.Close()
		pa.EndPoint.State = 3
		pa.EndPoint.Enable = false
		d.LastUpdatedTime = time.Now().Unix()
		d.Save(false)
		return 1
	} else {
		log.Infof("Milky 服务连接成功，账号<%s>(%d)", info.Nickname, info.UIN)
		pa.EndPoint.UserID = fmt.Sprintf("QQ:%d", info.UIN)
		pa.EndPoint.Nickname = info.Nickname
	}
	pa.EndPoint.State = 1
	pa.EndPoint.Enable = true
	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
	return 0
}

func (pa *PlatformAdapterMilky) DoRelogin() bool {
	if pa.IntentSession == nil {
		success := pa.Serve()
		return success == 0
	}
	_ = pa.IntentSession.Close()
	err := pa.IntentSession.Open()
	if err != nil {
		log.Errorf("Milky Connect Error:%s", err.Error())
		pa.EndPoint.State = 0
		return false
	}
	pa.EndPoint.State = 1
	pa.EndPoint.Enable = true
	d := pa.Session.Parent
	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
	return true
}

func (pa *PlatformAdapterMilky) SetEnable(enable bool) {
	if enable {
		log.Infof("正在启用Milky服务……")
		if pa.IntentSession == nil {
			pa.Serve()
			return
		}
		err := pa.IntentSession.Open()
		if err != nil {
			log.Errorf("与Milky服务进行连接时出错:%s", err.Error())
			pa.EndPoint.State = 3
			pa.EndPoint.Enable = false
			return
		}
		info, err := pa.IntentSession.GetLoginInfo()
		if err != nil {
			log.Errorf("Failed to get login info: %v", err)
		} else {
			pa.EndPoint.UserID = fmt.Sprintf("QQ:%d", info.UIN)
			pa.EndPoint.Nickname = info.Nickname
			log.Infof("Milky 服务连接成功，账号<%s>(%d)", info.Nickname, info.UIN)
		}
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

func ParseMessageToMilky(send []message.IMessageElement) []milky.IMessageElement {
	var elements []milky.IMessageElement
	for _, elem := range send {
		switch e := elem.(type) {
		case *message.TextElement:
			elements = append(elements, &milky.TextElement{Text: e.Content})
		case *message.ImageElement:
			log.Debugf("Image: %s", e.URL)
			elements = append(elements, &milky.ImageElement{URI: e.URL, Summary: e.File.File, SubType: "normal"})
		case *message.AtElement:
			log.Debugf("At user: %s", e.Target)
			if uid, err := strconv.ParseInt(e.Target, 10, 64); err == nil {
				elements = append(elements, &milky.AtElement{UserID: uid})
			}
		case *message.ReplyElement:
			if seq, err := strconv.ParseInt(e.ReplySeq, 10, 64); err == nil {
				elements = append(elements, &milky.ReplyElement{MessageSeq: seq})
			}
		case *message.RecordElement:
			log.Debugf("Record: %s", e.File.URL)
			elements = append(elements, &milky.RecordElement{URI: e.File.URL})
		case *message.PokeElement:
			continue
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
	ret, err := pa.IntentSession.SendPrivateMessage(id, &elements)
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
		RawID: ret.MessageSeq,
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
	var ret *milky.MessageRet
	if len(elements) == 0 {
		log.Debugf("No valid message elements to send to group %s", groupID)
		ret = &milky.MessageRet{}
	} else {
		ret, err = pa.IntentSession.SendGroupMessage(id, &elements)
	}
	if err != nil {
		log.Errorf("Failed to send group message to %s: %v", groupID, err)
		return
	}
	go func() {
		for _, element := range send {
			if poke, ok := element.(*message.PokeElement); ok {
				log.Debugf("Sending group Nudge: %s", poke.Target)
				userid, err2 := strconv.ParseInt(poke.Target, 10, 64)
				if err2 != nil {
					return
				}
				_ = pa.IntentSession.SendGroupNudge(id, userid)
				doSleepQQ(ctx)
			}
		}
	}()
	pa.Session.OnMessageSend(ctx, &Message{
		Platform:    "QQ",
		MessageType: "group",
		Message:     text,
		Sender: SenderBase{
			UserID:   pa.EndPoint.UserID,
			Nickname: pa.EndPoint.Nickname,
		},
		RawID: ret.MessageSeq,
	}, flag)
}

func (pa *PlatformAdapterMilky) SendFileToPerson(ctx *MsgContext, userID string, path string, flag string) {
	pa.SendToPerson(ctx, userID, fmt.Sprintf("[尝试发送文件: %s，但不支持]", filepath.Base(path)), flag)
}

func (pa *PlatformAdapterMilky) SendFileToGroup(_ *MsgContext, groupID string, path string, _ string) {
	id := ExtractQQGroupID(groupID)
	rawID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		log.Errorf("Invalid group ID %s: %v", groupID, err)
		return
	}
	filename := filepath.Base(path)
	if strings.HasPrefix(path, "files://") {
		path = "file://" + path[len("files://"):]
	}
	_, err = pa.IntentSession.UploadGroupFile(rawID, path, filename, "")
	if err != nil {
		log.Errorf("Failed to send file to group %s: %v", groupID, err)
		return
	}
}

func (pa *PlatformAdapterMilky) QuitGroup(ctx *MsgContext, groupID string) {
	id, err := strconv.ParseInt(ExtractQQGroupID(groupID), 10, 64)
	if err != nil {
		log.Errorf("Invalid group ID %s: %v", groupID, err)
		return
	}
	err = pa.IntentSession.QuitGroup(id)
	if err != nil {
		log.Errorf("Failed to quit group %s: %v", groupID, err)
		return
	}
	log.Infof("Successfully quit group %s", groupID)
}

func (pa *PlatformAdapterMilky) SetGroupCardName(ctx *MsgContext, cardName string) {
	groupID := ctx.Group.GroupID
	rawGroupID := ExtractQQGroupID(groupID)
	rawGroupIDInt, err := strconv.ParseInt(rawGroupID, 10, 64)
	if err != nil {
		log.Errorf("Invalid group ID %s: %v", groupID, err)
		return
	}
	userID := ctx.Player.UserID
	rawUserID := ExtractQQUserID(userID)
	rawUserIDInt, err := strconv.ParseInt(rawUserID, 10, 64)
	if err != nil {
		log.Errorf("Invalid user ID %s: %v", userID, err)
		return
	}
	err = pa.IntentSession.SetGroupMemberCard(rawGroupIDInt, rawUserIDInt, cardName)
	if err != nil {
		log.Errorf("Failed to set group card name for %s in group %s: %v", userID, groupID, err)
	}
}

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
