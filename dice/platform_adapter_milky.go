package dice

import (
	"fmt"
	"math/rand/v2"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	milky "github.com/Szzrain/Milky-go-sdk"
	"go.uber.org/zap"

	"sealdice-core/dice/events"
	logger "sealdice-core/logger"
	"sealdice-core/message"
)

type PlatformAdapterMilky struct {
	Session             *IMSession     `json:"-"                     yaml:"-"`
	EndPoint            *EndPointInfo  `json:"-"                     yaml:"-"`
	IntentSession       *milky.Session `json:"-"                     yaml:"-"`
	WsGateway           string         `json:"ws_gateway"            yaml:"ws_gateway"`
	RestGateway         string         `json:"rest_gateway"          yaml:"rest_gateway"`
	Token               string         `json:"token"                 yaml:"token"`
	IgnoreFriendRequest bool           `json:"ignore_friend_request" yaml:"ignore_friend_request"`
}

func (pa *PlatformAdapterMilky) SendSegmentToGroup(ctx *MsgContext, groupID string, msg []message.IMessageElement, flag string) {
	log := zap.S().Named(logger.LogKeyAdapter)
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
		GroupID:     groupID,
		Sender: SenderBase{
			UserID:   pa.EndPoint.UserID,
			Nickname: pa.EndPoint.Nickname,
		},
		RawID: ret.MessageSeq,
	}, flag)
}

func (pa *PlatformAdapterMilky) SendSegmentToPerson(ctx *MsgContext, userID string, msg []message.IMessageElement, flag string) {
	log := zap.S().Named(logger.LogKeyAdapter)
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
	log := zap.S().Named(logger.LogKeyAdapter)
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
	log := zap.S().Named(logger.LogKeyAdapter)
	pa.EndPoint.State = 2 // 设置状态为连接中

	if pa.RestGateway[len(pa.RestGateway)-1] == '/' {
		pa.RestGateway = pa.RestGateway[:len(pa.RestGateway)-1] // 去掉末尾的斜杠
	}
	if pa.WsGateway[len(pa.WsGateway)-1] == '/' {
		pa.WsGateway = pa.WsGateway[:len(pa.WsGateway)-1]
	}
	session, err := milky.New(pa.WsGateway, pa.RestGateway, pa.Token, log.Named(logger.LogKeyAdapter))
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
	session.AddHandler(func(session2 *milky.Session, m *milky.FriendNudge) {
		if m == nil {
			return
		}
		log.Debugf("Received friend nudge: Sender %d", m.UserID)
		msg := &Message{
			Platform:    "QQ",
			MessageType: "private",
			Sender: SenderBase{
				UserID: FormatDiceIDQQ(strconv.FormatInt(m.UserID, 10)),
			},
		}
		event := &events.PokeEvent{
			SenderID:  msg.Sender.UserID,
			IsPrivate: true,
		}
		if m.IsSelfReceive {
			event.TargetID = pa.EndPoint.UserID
		} else {
			event.TargetID = msg.Sender.UserID
		}
		pa.Session.OnPoke(CreateTempCtx(pa.EndPoint, msg), event)
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
	session.AddHandler(func(session2 *milky.Session, m *milky.GroupMemberIncrease) {
		ctx := &MsgContext{MessageType: "group", EndPoint: pa.EndPoint, Session: pa.Session, Dice: pa.Session.Parent}
		inviterID := FormatDiceIDQQ(strconv.FormatInt(m.InvitorID, 10))
		msg := &Message{
			Time:        time.Now().Unix(),
			MessageType: "group",
			GroupID:     "QQ-Group:" + strconv.FormatInt(m.GroupID, 10),
			Platform:    "QQ",
			Sender: SenderBase{
				UserID: inviterID,
			},
		}
		newMemberUID := FormatDiceIDQQ(strconv.FormatInt(m.UserID, 10))
		// 自己加群
		if newMemberUID == pa.EndPoint.UserID {
			pa.Session.OnGroupJoined(ctx, msg)
		} else {
			// 其他人被邀请加群
			msg.Sender.UserID = newMemberUID
			pa.Session.OnGroupMemberJoined(ctx, msg)
		}
	})
	session.AddHandler(func(session *milky.Session, m *milky.GroupMute) {
		if m == nil {
			return
		}
		ctx := &MsgContext{MessageType: "group", EndPoint: pa.EndPoint, Session: pa.Session, Dice: pa.Session.Parent}
		dm := pa.Session.Parent.Parent
		groupId := FormatDiceIDQQGroup(strconv.FormatInt(m.GroupID, 10))
		if FormatDiceIDQQ(strconv.FormatInt(m.UserID, 10)) == pa.EndPoint.UserID {
			opUID := FormatDiceIDQQ(strconv.FormatInt(m.OperatorID, 10))
			groupName := dm.TryGetGroupName(groupId)
			userName := dm.TryGetUserName(opUID)

			ctx.Dice.Config.BanList.AddScoreByGroupMuted(opUID, groupId, ctx)
			txt := fmt.Sprintf("被禁言: 在群组<%s>(%s)中被禁言，时长%d秒，操作者:<%s>(%d)", groupName, groupId, m.Duration, userName, m.OperatorID)
			log.Info(txt)
			ctx.Notice(txt)
		}
	})
	session.AddHandler(func(session2 *milky.Session, m *milky.FriendRequest) {
		if m == nil {
			ctx := &MsgContext{MessageType: "private", EndPoint: pa.EndPoint, Session: pa.Session, Dice: pa.Session.Parent}
			pa.handelFriendRequest(ctx, m)
		}
	})
	session.AddHandler(func(session2 *milky.Session, m *milky.GroupInvitation) {
		dm := pa.Session.Parent.Parent
		if m == nil {
			return
		}
		ctx := &MsgContext{MessageType: "group", EndPoint: pa.EndPoint, Session: pa.Session, Dice: pa.Session.Parent}
		uid := FormatDiceIDQQ(strconv.FormatInt(m.InitiatorID, 10))
		groupId := FormatDiceIDQQGroup(strconv.FormatInt(m.GroupID, 10))
		pa.GetGroupInfoAsync(groupId)
		groupName := dm.TryGetGroupName(groupId)
		userName := dm.TryGetUserName(uid)
		txt := fmt.Sprintf("收到QQ加群邀请: 群组<%s>(%s) 邀请人:<%s>(%d)", groupName, groupId, userName, m.InitiatorID)
		log.Info(txt)
		ctx.Notice(txt)

		// 邀请人在黑名单上
		banInfo, ok := ctx.Dice.Config.BanList.GetByID(uid)
		if ok {
			if banInfo.Rank == BanRankBanned && ctx.Dice.Config.BanList.BanBehaviorRefuseInvite {
				pa.SetGroupAddRequest(m.GroupID, m.InvitationSeq, false)
				return
			}
		}

		// 信任模式，如果不是信任，又不是master则拒绝拉群邀请
		isMaster := ctx.Dice.IsMaster(uid)
		if ctx.Dice.Config.TrustOnlyMode && ((banInfo != nil && banInfo.Rank != BanRankTrusted) && !isMaster) {
			pa.SetGroupAddRequest(m.GroupID, m.InvitationSeq, false)
			return
		}

		// 群在黑名单上
		banInfo, ok = ctx.Dice.Config.BanList.GetByID(groupId)
		if ok {
			if banInfo.Rank == BanRankBanned {
				pa.SetGroupAddRequest(m.GroupID, m.InvitationSeq, false)
				return
			}
		}

		if ctx.Dice.Config.RefuseGroupInvite {
			pa.SetGroupAddRequest(m.GroupID, m.InvitationSeq, false)
			return
		}

		pa.SetGroupAddRequest(m.GroupID, m.InvitationSeq, true)
	})
	session.AddHandler(func(session2 *milky.Session, m *milky.MessageRecall) {
		if m == nil {
			return
		}
		msg := new(Message)
		msg.Time = time.Now().Unix()
		msg.Platform = "QQ"
		msg.RawID = m.MessageSeq
		switch m.MessageScene {
		case "group":
			msg.MessageType = "group"
			msg.GroupID = FormatDiceIDQQGroup(strconv.FormatInt(m.PeerID, 10))
			msg.Sender = SenderBase{
				UserID: FormatDiceIDQQ(strconv.FormatInt(m.SenderID, 10)),
			}
		case "friend":
			msg.MessageType = "private"
			msg.Sender = SenderBase{
				UserID: FormatDiceIDQQ(strconv.FormatInt(m.SenderID, 10)),
			}
		default:
			return
		}
		mctx := &MsgContext{Session: pa.Session, EndPoint: pa.EndPoint, Dice: pa.Session.Parent, MessageType: msg.MessageType}
		pa.Session.OnMessageDeleted(mctx, msg)
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

func (pa *PlatformAdapterMilky) SetGroupAddRequest(groupId int64, invitationSeq int64, approve bool) {
	log := zap.S().Named(logger.LogKeyAdapter)
	if approve {
		err := pa.IntentSession.AcceptGroupInvitation(groupId, invitationSeq)
		if err != nil {
			log.Errorf("Failed to accept group invitation: %v", err)
			return
		}
	} else {
		// 拒绝加群邀请
		err := pa.IntentSession.RejectGroupInvitation(groupId, invitationSeq)
		if err != nil {
			log.Errorf("Failed to refuse group invitation: %v", err)
			return
		}
	}
}

func (pa *PlatformAdapterMilky) handelFriendRequest(ctx *MsgContext, event *milky.FriendRequest) {
	log := zap.S().Named(logger.LogKeyAdapter)
	var comment string
	if event.Comment != "" {
		comment = strings.TrimSpace(event.Comment)
		comment = strings.ReplaceAll(comment, "\u00a0", "")
	}

	toMatch := strings.TrimSpace(pa.Session.Parent.Config.FriendAddComment)
	willAccept := comment == DiceFormat(ctx, toMatch)
	if toMatch == "" {
		willAccept = true
	}

	if !willAccept {
		// 如果是问题校验，只填写回答即可
		re := regexp.MustCompile(`\n回答:([^\n]+)`)
		m := re.FindAllStringSubmatch(comment, -1)

		var items []string
		for _, i := range m {
			items = append(items, i[1])
		}

		re2 := regexp.MustCompile(`\s+`)
		m2 := re2.Split(toMatch, -1)

		if len(m2) == len(items) {
			ok := true
			for i := range m2 {
				if m2[i] != items[i] {
					ok = false
					break
				}
			}
			willAccept = ok
		}
	}

	if comment == "" {
		comment = "(无)"
	} else {
		comment = strconv.Quote(comment)
	}

	// 检查黑名单
	extra := ""
	uid := FormatDiceIDQQ(strconv.FormatInt(event.InitiatorID, 10))
	banInfo, ok := ctx.Dice.Config.BanList.GetByID(uid)
	if ok {
		if banInfo.Rank == BanRankBanned && ctx.Dice.Config.BanList.BanBehaviorRefuseInvite {
			if willAccept {
				extra = "。回答正确，但为被禁止用户，准备自动拒绝"
			} else {
				extra = "。回答错误，且为被禁止用户，准备自动拒绝"
			}
			willAccept = false
		}
	}

	if pa.IgnoreFriendRequest {
		extra += "。由于设置了忽略邀请，此信息仅为通报"
	}

	txt := fmt.Sprintf("收到QQ好友邀请: 邀请人:%s, 验证信息: %s, 是否自动同意: %t%s", strconv.FormatInt(event.InitiatorID, 10), comment, willAccept, extra)
	log.Info(txt)
	ctx.Notice(txt)

	// 忽略邀请
	if pa.IgnoreFriendRequest {
		return
	}

	time.Sleep(time.Duration((0.8 + rand.Float64()) * float64(time.Second)))

	if willAccept {
		pa.SetFriendAddRequest(event.InitiatorUID, true, "")
	} else {
		pa.SetFriendAddRequest(event.InitiatorUID, false, "验证信息不符")
	}
}

func (pa *PlatformAdapterMilky) SetFriendAddRequest(initiatorUid string, approve bool, reason string) {
	log := zap.S().Named(logger.LogKeyAdapter)
	if approve {
		// 同意好友请求，目前都是 unfiltered 的
		err := pa.IntentSession.AcceptFriendRequest(initiatorUid, false)
		if err != nil {
			log.Errorf("Failed to accept friend request: %v", err)
			return
		}
	} else {
		// 拒绝好友请求
		err := pa.IntentSession.RejectFriendRequest(initiatorUid, false, reason)
		if err != nil {
			log.Errorf("Failed to refuse friend request: %v", err)
			return
		}
	}
}

func (pa *PlatformAdapterMilky) DoRelogin() bool {
	log := zap.S().Named(logger.LogKeyAdapter)
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
	log := zap.S().Named(logger.LogKeyAdapter)
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
	log := zap.S().Named(logger.LogKeyAdapter)
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
	log := zap.S().Named(logger.LogKeyAdapter)
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
	log := zap.S().Named(logger.LogKeyAdapter)
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
		GroupID:     groupID,
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
	log := zap.S().Named(logger.LogKeyAdapter)
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
	log := zap.S().Named(logger.LogKeyAdapter)
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
	log := zap.S().Named(logger.LogKeyAdapter)
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
