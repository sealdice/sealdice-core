package dice

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	milky "github.com/Szzrain/Milky-go-sdk"
	"github.com/bytedance/sonic"
	"go.uber.org/zap"

	"sealdice-core/dice/events"
	"sealdice-core/logger"
	"sealdice-core/message"
	"sealdice-core/utils/procs"
)

type PlatformAdapterMilky struct {
	EndPoint            *EndPointInfo  `json:"-"                     yaml:"-"`
	IntentSession       *milky.Session `json:"-"                     yaml:"-"`
	WsGateway           string         `json:"ws_gateway"            yaml:"ws_gateway"`
	RestGateway         string         `json:"rest_gateway"          yaml:"rest_gateway"`
	Token               string         `json:"-"                 yaml:"token"`
	IgnoreFriendRequest bool           `json:"ignore_friend_request" yaml:"ignore_friend_request"`
	// 内置
	// BuiltInMode 留空则视为分离，目前支持的字段为 lagrangeV2
	BuiltInMode       string          `json:"built_in_mode" yaml:"built_in_mode"`
	MilkyProcess      *procs.Process  `json:"-" yaml:"-"`
	BuiltInLoginState MilkyLoginState `json:"loginState" yaml:"-"`
	QrCodeData        []byte          `json:"-"                          yaml:"-"`
}

type MilkyLoginState int64

const (
	MilkyLoginStateInit MilkyLoginState = iota
	MilkyLoginStatePlaceholder
	MilkyLoginStateQRWaitingForScan
	MilkyLoginStateConnecting
	MilkyLoginStateQRConnected
	MilkyLoginStateFailed
)

func (pa *PlatformAdapterMilky) SendSegmentToGroup(ctx *MsgContext, groupID string, msg []message.IMessageElement, flag string) {
	log := zap.S().Named(logger.LogKeyAdapter)
	id, err := strconv.ParseInt(ExtractQQGroupID(groupID), 10, 64)
	if err != nil {
		log.Errorf("Invalid group ID %s: %v", groupID, err)
		return
	}
	elements := ParseMessageToMilky(msg)
	if len(elements) == 0 {
		log.Warnf("Skipping Milky group message to %s: no supported message elements", groupID)
		return
	}
	ret, err := pa.IntentSession.SendGroupMessage(id, &elements)
	if err != nil {
		log.Errorf("Failed to send group message to %s: %v", groupID, err)
		return
	}
	pa.EndPoint.Session.OnMessageSend(ctx, &Message{
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
	if len(elements) == 0 {
		log.Warnf("Skipping Milky private message to %s: no supported message elements", userID)
		return
	}
	ret, err := pa.IntentSession.SendPrivateMessage(id, &elements)
	if err != nil {
		log.Errorf("Failed to send private message to %s: %v", userID, err)
		return
	}
	pa.EndPoint.Session.OnMessageSend(ctx, &Message{
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
	groupInfoMilky, err := pa.IntentSession.GetGroupInfo(id, true)
	if err != nil {
		log.Errorf("Failed to get group info for %s: %v", groupID, err)
		return
	}
	if groupInfoMilky == nil {
		log.Warnf("Group info for %s is nil", groupID)
		return
	}
	dm := pa.EndPoint.Session.Parent.Parent
	dm.GroupNameCache.Store(groupID, &GroupNameCacheItem{
		Name: groupInfoMilky.Name,
		time: time.Now().Unix(),
	})
	session := pa.EndPoint.Session
	groupInfo, ok := session.ServiceAtNew.Load(groupID)
	if ok {
		if groupInfoMilky.Name != groupInfo.GroupName {
			// 更新群名
			groupInfo.GroupName = groupInfoMilky.Name
			groupInfo.MarkDirty(session.Parent)
		}
	}
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
				UserID:  FormatDiceIDQQ(strconv.FormatInt(m.SenderId, 10)),
				IsRobot: isQQBotUIN(m.SenderId),
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
						Target:  strconv.FormatInt(seg.UserID, 10),
						IsRobot: isQQBotUIN(seg.UserID),
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
		pa.EndPoint.Session.ExecuteNew(pa.EndPoint, msg)
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
		pa.EndPoint.Session.OnPoke(CreateTempCtx(pa.EndPoint, msg), &events.PokeEvent{
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
		pa.EndPoint.Session.OnPoke(CreateTempCtx(pa.EndPoint, msg), event)
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
				pa.EndPoint.Session.OnGroupLeave(CreateTempCtx(pa.EndPoint, msg), &events.GroupLeaveEvent{
					GroupID:    msg.GroupID,
					UserID:     pa.EndPoint.UserID,
					OperatorID: "",
				})
			} else {
				log.Debugf("Bot left group %s with operator ID %d", msg.GroupID, m.OperatorID)
				pa.EndPoint.Session.OnGroupLeave(CreateTempCtx(pa.EndPoint, msg), &events.GroupLeaveEvent{
					GroupID:    msg.GroupID,
					UserID:     pa.EndPoint.UserID,
					OperatorID: FormatDiceIDQQ(strconv.FormatInt(m.OperatorID, 10)),
				})
			}
		}
	})
	session.AddHandler(func(session2 *milky.Session, m *milky.GroupMemberIncrease) {
		ctx := &MsgContext{MessageType: "group", EndPoint: pa.EndPoint, Session: pa.EndPoint.Session, Dice: pa.EndPoint.Session.Parent}
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
			pa.EndPoint.Session.OnGroupJoined(ctx, msg)
		} else {
			// 其他人被邀请加群
			msg.Sender.UserID = newMemberUID
			pa.EndPoint.Session.OnGroupMemberJoined(ctx, msg)
		}
	})
	session.AddHandler(func(session *milky.Session, m *milky.GroupMute) {
		if m == nil {
			return
		}
		ctx := &MsgContext{MessageType: "group", EndPoint: pa.EndPoint, Session: pa.EndPoint.Session, Dice: pa.EndPoint.Session.Parent}
		dm := pa.EndPoint.Session.Parent.Parent
		groupId := FormatDiceIDQQGroup(strconv.FormatInt(m.GroupID, 10))
		if FormatDiceIDQQ(strconv.FormatInt(m.UserID, 10)) == pa.EndPoint.UserID {
			opUID := FormatDiceIDQQ(strconv.FormatInt(m.OperatorID, 10))
			groupName := dm.TryGetGroupName(groupId)
			userName := dm.TryGetUserName(opUID)

			ctx.Dice.Config.BanList.AddScoreByGroupMuted(opUID, groupId, ctx)
			txt := fmt.Sprintf("被禁言: 在群组<%s>(%s)中被禁言，时长%d秒，操作者:<%s>(%d)", groupName, groupId, m.Duration, userName, m.OperatorID)
			log.Info(txt)
			ctx.Notice(txt, NoticeTypeGroup)
		}
	})
	session.AddHandler(func(session2 *milky.Session, m *milky.FriendRequest) {
		if m != nil {
			ctx := &MsgContext{MessageType: "private", EndPoint: pa.EndPoint, Session: pa.EndPoint.Session, Dice: pa.EndPoint.Session.Parent}
			pa.handelFriendRequest(ctx, m)
		}
	})
	session.AddHandler(func(session2 *milky.Session, m *milky.GroupInvitation) {
		dm := pa.EndPoint.Session.Parent.Parent
		if m == nil {
			return
		}
		ctx := &MsgContext{MessageType: "group", EndPoint: pa.EndPoint, Session: pa.EndPoint.Session, Dice: pa.EndPoint.Session.Parent}
		uid := FormatDiceIDQQ(strconv.FormatInt(m.InitiatorID, 10))
		groupId := FormatDiceIDQQGroup(strconv.FormatInt(m.GroupID, 10))
		groupName := dm.TryGetGroupName(groupId)
		userName := dm.TryGetUserName(uid)
		txt := fmt.Sprintf("收到QQ加群邀请: 群组<%s>(%s) 邀请人:<%s>(%d)", groupName, groupId, userName, m.InitiatorID)
		log.Info(txt)
		ctx.Notice(txt, NoticeTypeInvite)

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
		mctx := &MsgContext{Session: pa.EndPoint.Session, EndPoint: pa.EndPoint, Dice: pa.EndPoint.Session.Parent, MessageType: msg.MessageType}
		pa.EndPoint.Session.OnMessageDeleted(mctx, msg)
	})
	d := pa.EndPoint.Session.Parent
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
	}

	log.Infof("Milky 服务连接成功，账号<%s>(%d)", info.Nickname, info.UIN)
	pa.EndPoint.UserID = fmt.Sprintf("QQ:%d", info.UIN)
	pa.EndPoint.Nickname = info.Nickname
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
		comment = normalizeMilkyFriendRequestComment(event.Comment)
	}

	toMatch := strings.TrimSpace(pa.EndPoint.Session.Parent.Config.FriendAddComment)
	willAccept := comment == DiceFormat(ctx, toMatch)
	if toMatch == "" {
		willAccept = true
	}

	if !willAccept {
		willAccept = checkMilkyFriendAddVerify(comment, toMatch)
	}

	if comment == "" {
		comment = "(无)"
	} else {
		comment = formatMilkyFriendRequestCommentForLog(comment)
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
	ctx.Notice(txt, NoticeTypeInvite)

	// 通知类事件：好友申请（仅通知，不携带同意/拒绝能力）
	EmitFriendRequest(ctx, uid, comment)

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
	sendSpeech := func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("好友致辞异常: %v 堆栈: %v", r, string(debug.Stack()))
			}
		}()

		// 稍作等待后发好友致辞
		time.Sleep(2 * time.Second)
		selfId, err := strconv.ParseInt(ExtractQQUserID(pa.EndPoint.UserID), 10, 64)
		if err != nil {
			log.Errorf("好友致辞异常: 无法解析自己的QQ号: %v", err)
			return
		}
		event.InitiatorID = selfId
		msg := &Message{
			MessageType: "private",
			Platform:    "QQ",
			Sender: SenderBase{
				UserID: uid,
			},
		}
		ctx.Group, ctx.Player = GetPlayerInfoBySender(ctx, msg)

		// 通知类事件：成为好友
		EmitFriendJoined(ctx, msg)

		welcome := DiceFormatTmpl(ctx, "核心:骰子成为好友")
		log.Infof("与 %s 成为好友，发送好友致辞: %s", uid, welcome)

		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("好友致辞异常: %v 堆栈: %v", r, string(debug.Stack()))
				}
			}()

			for _, i := range ctx.SplitText(welcome) {
				doSleepQQ(ctx)
				pa.SendToPerson(ctx, uid, strings.TrimSpace(i), "")
			}
			if groupInfo, ok := ctx.Session.ServiceAtNew.Load(ctx.Group.GroupID); ok {
				groupInfo.TriggerExtHook(ctx.Dice, func(ext *ExtInfo) func() {
					if ext.OnBecomeFriend == nil {
						return nil
					}
					return func() { ext.OnBecomeFriend(ctx, msg) }
				})
			}
		}()
	}
	if willAccept {
		sendSpeech()
	}
}

var milkyFriendRequestAnswerPattern = regexp.MustCompile(`\n回答:([^\n]+)`)
var milkyFriendRequestExpectedItemPattern = regexp.MustCompile(`\s+`)

func normalizeMilkyFriendRequestComment(comment string) string {
	comment = strings.TrimSpace(comment)
	comment = strings.ReplaceAll(comment, "\u00a0", " ")
	comment = strings.ReplaceAll(comment, "\r\n", "\n")
	// 上游可能把好友验证里的换行以字面量转义形式传过来，这里统一还原为真实换行，
	// 以便问题校验和日志展示看到的是同一种文本布局。
	comment = strings.ReplaceAll(comment, `\r\n`, "\n")
	comment = strings.ReplaceAll(comment, `\n`, "\n")
	return comment
}

func formatMilkyFriendRequestCommentForLog(comment string) string {
	// 日志里保留真实换行，避免再次编码成 `\n` 影响人工查看。
	comment = strings.ReplaceAll(comment, `\`, `\\`)
	comment = strings.ReplaceAll(comment, `"`, `\"`)
	return `"` + comment + `"`
}

func checkMilkyFriendAddVerify(comment string, toMatch string) bool {
	if toMatch == "" {
		return true
	}

	matches := milkyFriendRequestAnswerPattern.FindAllStringSubmatch(comment, -1)
	answers := make([]string, 0, len(matches))
	for _, match := range matches {
		answer := strings.TrimSpace(strings.ReplaceAll(match[1], "\u00a0", " "))
		answers = append(answers, answer)
	}

	expectedItems := milkyFriendRequestExpectedItemPattern.Split(toMatch, -1)
	if len(expectedItems) != len(answers) {
		return false
	}

	for i, item := range expectedItems {
		expected := strings.TrimSpace(strings.ReplaceAll(item, "\u00a0", " "))
		if expected != answers[i] {
			return false
		}
	}

	return true
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
	pa.EndPoint.State = 2
	// 分离
	if pa.BuiltInMode == "" {
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
		d := pa.EndPoint.Session.Parent
		d.LastUpdatedTime = time.Now().Unix()
		d.Save(false)
		return true
	}
	// 内置
	// 先断开连接
	if pa.IntentSession != nil {
		_ = pa.IntentSession.Close()
	}
	// kill
	BuiltinMilkyClientKill(pa.EndPoint.Session.Parent, pa.EndPoint)
	MilkyRemoveSession(pa.EndPoint.Session.Parent, pa.EndPoint)
	go ServeMilkyBuiltIn(pa.EndPoint.Session.Parent, pa.EndPoint)
	return true
}

func (pa *PlatformAdapterMilky) SetEnable(enable bool) {
	log := zap.S().Named(logger.LogKeyAdapter)
	if pa.BuiltInMode == "" {
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
		d := pa.EndPoint.Session.Parent
		d.LastUpdatedTime = time.Now().Unix()
		d.Save(false)
		return
	}
	if enable {
		go ServeMilkyBuiltIn(pa.EndPoint.Session.Parent, pa.EndPoint)
	} else {
		if pa.IntentSession != nil {
			_ = pa.IntentSession.Close()
		}
		BuiltinMilkyClientKill(pa.EndPoint.Session.Parent, pa.EndPoint)
		pa.EndPoint.State = 0
		pa.EndPoint.Enable = false
		d := pa.EndPoint.Session.Parent
		d.LastUpdatedTime = time.Now().Unix()
		d.Save(false)
	}
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

func buildMilkyForwardElement(nodes []forwardNode) (*milky.ForwardElement, error) {
	if len(nodes) == 0 {
		return nil, errors.New("forward message has no nodes")
	}

	messages := make([]milky.OutgoingForwardedMessage, 0, len(nodes))
	for index, node := range nodes {
		userID, err := strconv.ParseInt(strings.TrimSpace(node.Data.Uin), 10, 64)
		if err != nil || userID <= 0 {
			return nil, fmt.Errorf("forward node %d has invalid user ID %q", index, node.Data.Uin)
		}
		if strings.TrimSpace(node.Data.Content) == "" {
			return nil, fmt.Errorf("forward node %d has empty content", index)
		}

		segments := ParseMessageToMilky(message.ConvertStringMessage(node.Data.Content))
		if len(segments) == 0 {
			return nil, fmt.Errorf("forward node %d has no supported message segments", index)
		}
		messages = append(messages, milky.OutgoingForwardedMessage{
			UserID:     userID,
			SenderName: node.Data.Name,
			Segments:   segments,
		})
	}

	return &milky.ForwardElement{Messages: messages}, nil
}

func (pa *PlatformAdapterMilky) recordForwardMessageSent(ctx *MsgContext, messageType string, targetID string, nodes []forwardNode, messageSeq int64) {
	if ctx == nil || pa == nil || pa.EndPoint == nil || pa.EndPoint.Session == nil {
		return
	}

	msg := &Message{
		Platform:    "QQ",
		MessageType: messageType,
		Message:     forwardNodesToText(nodes),
		Sender: SenderBase{
			UserID:   pa.EndPoint.UserID,
			Nickname: pa.EndPoint.Nickname,
		},
		RawID: messageSeq,
	}
	if messageType == "group" {
		msg.GroupID = targetID
	}
	pa.EndPoint.Session.OnMessageSend(ctx, msg, "")
}

func (pa *PlatformAdapterMilky) SendGroupForwardMsg(ctx *MsgContext, groupID string, nodes []forwardNode) bool {
	log := zap.S().Named(logger.LogKeyAdapter)
	if pa == nil || pa.IntentSession == nil {
		log.Error("Failed to send Milky group forward message: session unavailable")
		return false
	}

	id, err := strconv.ParseInt(ExtractQQGroupID(groupID), 10, 64)
	if err != nil || id <= 0 {
		log.Errorf("Invalid group ID %s for Milky forward message", groupID)
		return false
	}
	forward, err := buildMilkyForwardElement(nodes)
	if err != nil {
		log.Errorf("Failed to build Milky group forward message: %v", err)
		return false
	}

	if ctx != nil && ctx.EndPoint != nil && ctx.EndPoint.Platform == "QQ" {
		doSleepQQ(ctx)
	}
	elements := []milky.IMessageElement{forward}
	ret, err := pa.IntentSession.SendGroupMessage(id, &elements)
	if err != nil {
		log.Errorf("Failed to send group forward message to %s: %v", groupID, err)
		return false
	}

	pa.recordForwardMessageSent(ctx, "group", groupID, nodes, ret.MessageSeq)
	return true
}

func (pa *PlatformAdapterMilky) SendPrivateForwardMsg(ctx *MsgContext, userID string, nodes []forwardNode) bool {
	log := zap.S().Named(logger.LogKeyAdapter)
	if pa == nil || pa.IntentSession == nil {
		log.Error("Failed to send Milky private forward message: session unavailable")
		return false
	}

	id, err := strconv.ParseInt(ExtractQQUserID(userID), 10, 64)
	if err != nil || id <= 0 {
		log.Errorf("Invalid user ID %s for Milky forward message", userID)
		return false
	}
	forward, err := buildMilkyForwardElement(nodes)
	if err != nil {
		log.Errorf("Failed to build Milky private forward message: %v", err)
		return false
	}

	if ctx != nil && ctx.EndPoint != nil && ctx.EndPoint.Platform == "QQ" {
		doSleepQQ(ctx)
	}
	elements := []milky.IMessageElement{forward}
	ret, err := pa.IntentSession.SendPrivateMessage(id, &elements)
	if err != nil {
		log.Errorf("Failed to send private forward message to %s: %v", userID, err)
		return false
	}

	pa.recordForwardMessageSent(ctx, "private", userID, nodes, ret.MessageSeq)
	return true
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
	if len(elements) == 0 {
		log.Warnf("Skipping Milky private message to %s: no supported message elements", uid)
		return
	}
	ret, err := pa.IntentSession.SendPrivateMessage(id, &elements)
	if err != nil {
		log.Errorf("Failed to send private message to %s: %v", uid, err)
		return
	}
	pa.EndPoint.Session.OnMessageSend(ctx, &Message{
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
	nudgeTargets := make([]int64, 0)
	for _, element := range send {
		poke, ok := element.(*message.PokeElement)
		if !ok {
			continue
		}
		userID, parseErr := strconv.ParseInt(poke.Target, 10, 64)
		if parseErr != nil || userID <= 0 {
			log.Warnf("Skipping invalid Milky group nudge target %q", poke.Target)
			continue
		}
		nudgeTargets = append(nudgeTargets, userID)
	}
	if len(elements) == 0 && len(nudgeTargets) == 0 {
		log.Warnf("Skipping Milky group message to %s: no supported message elements", groupID)
		return
	}
	var ret *milky.MessageRet
	if len(elements) == 0 {
		ret = &milky.MessageRet{}
	} else {
		ret, err = pa.IntentSession.SendGroupMessage(id, &elements)
	}
	if err != nil {
		log.Errorf("Failed to send group message to %s: %v", groupID, err)
		return
	}
	go func(targets []int64) {
		for _, userID := range targets {
			log.Debugf("Sending group Nudge: %d", userID)
			_ = pa.IntentSession.SendGroupNudge(id, userID)
			doSleepQQ(ctx)
		}
	}(nudgeTargets)
	pa.EndPoint.Session.OnMessageSend(ctx, &Message{
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

func (pa *PlatformAdapterMilky) GetGroupMemberInfo(groupID string, userID string) (*milky.GroupMemberInfo, error) {
	return pa.getGroupMemberInfo(groupID, userID, false)
}

func (pa *PlatformAdapterMilky) getGroupMemberInfo(groupID string, userID string, noCache bool) (*milky.GroupMemberInfo, error) {
	if pa == nil || pa.IntentSession == nil {
		return nil, errors.New("milky session unavailable")
	}

	rawGroupID := strings.TrimSpace(ExtractQQGroupID(groupID))
	rawUserID := strings.TrimSpace(ExtractQQUserID(userID))
	if rawGroupID == "" || rawUserID == "" {
		return nil, errors.New("cannot resolve milky group/user id")
	}

	groupIDInt, err := strconv.ParseInt(rawGroupID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid milky group id %q: %w", groupID, err)
	}
	userIDInt, err := strconv.ParseInt(rawUserID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid milky user id %q: %w", userID, err)
	}
	return pa.IntentSession.GetGroupMemberInfo(groupIDInt, userIDInt, noCache)
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

// registerMilkyAdapterCapabilities milky 能力声明。RawAction 直接包装 Milky SDK 的 REST 能力（标量参数动作）。
func registerMilkyAdapterCapabilities() {
	RegisterAdapterCapabilities(AdapterCapabilitySet{
		ProtocolType: "milky",
		Platform:     "QQ",
		EmitEvents: map[string]AdapterEventSpec{
			EventNamePoke:              {Name: EventNamePoke, Description: "戳一戳"},
			EventNameGroupJoined:       {Name: EventNameGroupJoined, Description: "骰子加入群"},
			EventNameGroupMemberJoined: {Name: EventNameGroupMemberJoined, Description: "群成员加入"},
			EventNameGroupLeave:        {Name: EventNameGroupLeave, Description: "群成员离开/被踢"},
			EventNameFriendJoined:      {Name: EventNameFriendJoined, Description: "成为好友"},
			EventNameFriendRequest:     {Name: EventNameFriendRequest, Description: "好友申请（仅通知）", RequestOnly: true},
			EventNameMessageDeleted:    {Name: EventNameMessageDeleted, Description: "消息撤回"},
		},
		RawActions: map[string]AdapterRawActionSpec{
			"get_login_info":                 {Name: "get_login_info", Description: "获取登录号信息"},
			"get_user_profile":               {Name: "get_user_profile", Description: "获取用户资料", Params: map[string]string{"user_id": "int64"}},
			"get_friend_list":                {Name: "get_friend_list", Description: "获取好友列表", Params: map[string]string{"no_cache": "bool 可选"}},
			"get_friend_info":                {Name: "get_friend_info", Description: "获取好友信息", Params: map[string]string{"user_id": "int64", "no_cache": "bool 可选"}},
			"get_group_list":                 {Name: "get_group_list", Description: "获取群列表", Params: map[string]string{"no_cache": "bool 可选"}},
			"get_group_info":                 {Name: "get_group_info", Description: "获取群信息", Params: map[string]string{"group_id": "int64", "no_cache": "bool 可选"}},
			"get_group_member_list":          {Name: "get_group_member_list", Description: "获取群成员列表", Params: map[string]string{"group_id": "int64", "no_cache": "bool 可选"}},
			"get_group_member_info":          {Name: "get_group_member_info", Description: "获取群成员信息", Params: map[string]string{"group_id": "int64", "user_id": "int64", "no_cache": "bool 可选"}},
			"get_message":                    {Name: "get_message", Description: "获取消息", Params: map[string]string{"message_scene": "string group/private", "peer_id": "int64", "message_seq": "int64"}},
			"get_history_messages":           {Name: "get_history_messages", Description: "获取历史消息", Params: map[string]string{"message_scene": "string", "peer_id": "int64", "start_message_seq": "int64", "limit": "int32"}},
			"send_group_nudge":               {Name: "send_group_nudge", Description: "群内戳一戳", Params: map[string]string{"group_id": "int64", "user_id": "int64"}},
			"send_friend_nudge":              {Name: "send_friend_nudge", Description: "好友戳一戳", Params: map[string]string{"user_id": "int64", "is_self": "bool 可选"}},
			"set_group_member_mute":          {Name: "set_group_member_mute", Description: "禁言群成员", Params: map[string]string{"group_id": "int64", "user_id": "int64", "duration": "int32秒 0解禁"}},
			"set_group_whole_mute":           {Name: "set_group_whole_mute", Description: "全员禁言", Params: map[string]string{"group_id": "int64", "is_mute": "bool"}},
			"set_group_member_card":          {Name: "set_group_member_card", Description: "设置群名片", Params: map[string]string{"group_id": "int64", "user_id": "int64", "card": "string"}},
			"set_group_member_special_title": {Name: "set_group_member_special_title", Description: "设置群头衔", Params: map[string]string{"group_id": "int64", "user_id": "int64", "special_title": "string"}},
			"set_group_member_admin":         {Name: "set_group_member_admin", Description: "设置/取消群管理员", Params: map[string]string{"group_id": "int64", "user_id": "int64", "is_set": "bool"}},
			"set_group_name":                 {Name: "set_group_name", Description: "设置群名", Params: map[string]string{"group_id": "int64", "new_group_name": "string"}},
			"kick_group_member":              {Name: "kick_group_member", Description: "移出群聊", Params: map[string]string{"group_id": "int64", "user_id": "int64", "reject_add_request": "bool 可选"}},
			"quit_group":                     {Name: "quit_group", Description: "退出群聊", Params: map[string]string{"group_id": "int64"}},
			"recall_group_message":           {Name: "recall_group_message", Description: "撤回群消息", Params: map[string]string{"group_id": "int64", "message_seq": "int64"}},
			"recall_private_message":         {Name: "recall_private_message", Description: "撤回私聊消息", Params: map[string]string{"user_id": "int64", "message_seq": "int64"}},
			"set_group_essence_message":      {Name: "set_group_essence_message", Description: "设置/取消群精华消息", Params: map[string]string{"group_id": "int64", "message_seq": "int64", "is_set": "bool"}},
			"send_group_message_reaction":    {Name: "send_group_message_reaction", Description: "群消息表情回应", Params: map[string]string{"group_id": "int64", "message_seq": "int64", "reaction": "string", "is_set": "bool"}},
			"mark_message_as_read":           {Name: "mark_message_as_read", Description: "标记消息已读", Params: map[string]string{"message_scene": "string", "peer_id": "int64", "message_seq": "int64"}},
			"accept_friend_request":          {Name: "accept_friend_request", Description: "同意好友申请", Params: map[string]string{"initiator_uid": "string", "is_filtered": "bool 可选"}},
			"reject_friend_request":          {Name: "reject_friend_request", Description: "拒绝好友申请", Params: map[string]string{"initiator_uid": "string", "is_filtered": "bool 可选", "reason": "string 可选"}},
			"accept_group_request":           {Name: "accept_group_request", Description: "同意加群请求/邀请", Params: map[string]string{"notification_seq": "int64", "notification_type": "string", "group_id": "int64", "is_filtered": "bool 可选"}},
			"reject_group_invitation":        {Name: "reject_group_invitation", Description: "拒绝加群邀请", Params: map[string]string{"group_id": "int64", "invitation_seq": "int64"}},
		},
	})
}

// rawActionParam 从 map 参数取 int64，缺失或类型不符时报错。
func rawActionParam(params map[string]any, key string) (int64, error) {
	v, ok := params[key]
	if !ok {
		return 0, fmt.Errorf("缺少参数 %s", key)
	}
	switch n := v.(type) {
	case int64:
		return n, nil
	case int:
		return int64(n), nil
	case float64:
		return int64(n), nil
	default:
		return 0, fmt.Errorf("参数 %s 类型不符: %T", key, v)
	}
}

// rawActionInt32 从 map 参数取 int32，缺省返回 def。
func rawActionInt32(params map[string]any, key string, def int32) (int32, error) {
	v, ok := params[key]
	if !ok || v == nil {
		return def, nil
	}
	n, err := rawActionParam(params, key)
	if err != nil {
		return 0, err
	}
	return int32(n), nil
}

// rawActionStr 从 map 参数取 string，缺省返回 def。
func rawActionStr(params map[string]any, key string, def string) string {
	v, ok := params[key]
	if !ok || v == nil {
		return def
	}
	if s, ok := v.(string); ok {
		return s
	}
	return def
}

// rawActionBool 从 map 参数取 bool，缺省返回 def。
func rawActionBool(params map[string]any, key string, def bool) bool {
	v, ok := params[key]
	if !ok || v == nil {
		return def
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return def
}

// milkyRawResult 把 SDK 返回的结构体转为 map（sonic 往返），字段与 JSON 一致。
func milkyRawResult(ret any, err error) (any, error) {
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return map[string]any{"ok": true}, nil
	}
	raw, mErr := sonic.Marshal(ret)
	if mErr != nil {
		//nolint:nilerr // 尽力而为的序列化，失败回退 ok 标记
		return map[string]any{"ok": true}, nil
	}
	var out any
	if uErr := sonic.Unmarshal(raw, &out); uErr != nil {
		//nolint:nilerr // 尽力而为的序列化，失败回退 ok 标记
		return map[string]any{"ok": true}, nil
	}
	return out, nil
}

// RawAction milky 动作透传（标量参数动作，直接包装 SDK）。
func (pa *PlatformAdapterMilky) RawAction(ctx context.Context, action string, params map[string]any) (any, error) {
	if pa.IntentSession == nil {
		return nil, errors.New("milky 端点未连接")
	}
	s := pa.IntentSession
	switch action {
	case "get_login_info":
		return milkyRawResult(s.GetLoginInfo())
	case "get_user_profile":
		uid, err := rawActionParam(params, "user_id")
		if err != nil {
			return nil, err
		}
		return milkyRawResult(s.GetUserProfile(uid))
	case "get_friend_list":
		return milkyRawResult(s.GetFriendList(rawActionBool(params, "no_cache", false)))
	case "get_friend_info":
		uid, err := rawActionParam(params, "user_id")
		if err != nil {
			return nil, err
		}
		return milkyRawResult(s.GetFriendInfo(uid, rawActionBool(params, "no_cache", false)))
	case "get_group_list":
		return milkyRawResult(s.GetGroupList(rawActionBool(params, "no_cache", false)))
	case "get_group_info":
		gid, err := rawActionParam(params, "group_id")
		if err != nil {
			return nil, err
		}
		return milkyRawResult(s.GetGroupInfo(gid, rawActionBool(params, "no_cache", false)))
	case "get_group_member_list":
		gid, err := rawActionParam(params, "group_id")
		if err != nil {
			return nil, err
		}
		return milkyRawResult(s.GetGroupMemberList(gid, rawActionBool(params, "no_cache", false)))
	case "get_group_member_info":
		gid, err := rawActionParam(params, "group_id")
		if err != nil {
			return nil, err
		}
		uid, err := rawActionParam(params, "user_id")
		if err != nil {
			return nil, err
		}
		return milkyRawResult(s.GetGroupMemberInfo(gid, uid, rawActionBool(params, "no_cache", false)))
	case "get_message":
		peerID, err := rawActionParam(params, "peer_id")
		if err != nil {
			return nil, err
		}
		seq, err := rawActionParam(params, "message_seq")
		if err != nil {
			return nil, err
		}
		return milkyRawResult(s.GetMessage(rawActionStr(params, "message_scene", "group"), peerID, seq))
	case "get_history_messages":
		peerID, err := rawActionParam(params, "peer_id")
		if err != nil {
			return nil, err
		}
		startSeq, err := rawActionParam(params, "start_message_seq")
		if err != nil {
			return nil, err
		}
		limit, err := rawActionInt32(params, "limit", 10)
		if err != nil {
			return nil, err
		}
		msgs, nextSeq, err := s.GetHistoryMessages(rawActionStr(params, "message_scene", "group"), peerID, startSeq, limit)
		if err != nil {
			return nil, err
		}
		return map[string]any{"messages": msgs, "next_message_seq": nextSeq}, nil
	case "send_group_nudge":
		gid, err := rawActionParam(params, "group_id")
		if err != nil {
			return nil, err
		}
		uid, err := rawActionParam(params, "user_id")
		if err != nil {
			return nil, err
		}
		return milkyRawResult(nil, s.SendGroupNudge(gid, uid))
	case "send_friend_nudge":
		uid, err := rawActionParam(params, "user_id")
		if err != nil {
			return nil, err
		}
		return milkyRawResult(nil, s.SendFriendNudge(uid, rawActionBool(params, "is_self", false)))
	case "set_group_member_mute":
		gid, err := rawActionParam(params, "group_id")
		if err != nil {
			return nil, err
		}
		uid, err := rawActionParam(params, "user_id")
		if err != nil {
			return nil, err
		}
		duration, err := rawActionInt32(params, "duration", 0)
		if err != nil {
			return nil, err
		}
		return milkyRawResult(nil, s.SetGroupMemberMute(gid, uid, duration))
	case "set_group_whole_mute":
		gid, err := rawActionParam(params, "group_id")
		if err != nil {
			return nil, err
		}
		return milkyRawResult(nil, s.SetGroupWholeMute(gid, rawActionBool(params, "is_mute", true)))
	case "set_group_member_card":
		gid, err := rawActionParam(params, "group_id")
		if err != nil {
			return nil, err
		}
		uid, err := rawActionParam(params, "user_id")
		if err != nil {
			return nil, err
		}
		return milkyRawResult(nil, s.SetGroupMemberCard(gid, uid, rawActionStr(params, "card", "")))
	case "set_group_member_special_title":
		gid, err := rawActionParam(params, "group_id")
		if err != nil {
			return nil, err
		}
		uid, err := rawActionParam(params, "user_id")
		if err != nil {
			return nil, err
		}
		return milkyRawResult(nil, s.SetGroupMemberSpecialTitle(gid, uid, rawActionStr(params, "special_title", "")))
	case "set_group_member_admin":
		gid, err := rawActionParam(params, "group_id")
		if err != nil {
			return nil, err
		}
		uid, err := rawActionParam(params, "user_id")
		if err != nil {
			return nil, err
		}
		return milkyRawResult(nil, s.SetGroupMemberAdmin(gid, uid, rawActionBool(params, "is_set", true)))
	case "set_group_name":
		gid, err := rawActionParam(params, "group_id")
		if err != nil {
			return nil, err
		}
		return milkyRawResult(nil, s.SetGroupName(gid, rawActionStr(params, "new_group_name", "")))
	case "kick_group_member":
		gid, err := rawActionParam(params, "group_id")
		if err != nil {
			return nil, err
		}
		uid, err := rawActionParam(params, "user_id")
		if err != nil {
			return nil, err
		}
		return milkyRawResult(nil, s.KickGroupMember(gid, uid, rawActionBool(params, "reject_add_request", false)))
	case "quit_group":
		gid, err := rawActionParam(params, "group_id")
		if err != nil {
			return nil, err
		}
		return milkyRawResult(nil, s.QuitGroup(gid))
	case "recall_group_message":
		gid, err := rawActionParam(params, "group_id")
		if err != nil {
			return nil, err
		}
		seq, err := rawActionParam(params, "message_seq")
		if err != nil {
			return nil, err
		}
		return milkyRawResult(nil, s.RecallGroupMessage(gid, seq))
	case "recall_private_message":
		uid, err := rawActionParam(params, "user_id")
		if err != nil {
			return nil, err
		}
		seq, err := rawActionParam(params, "message_seq")
		if err != nil {
			return nil, err
		}
		return milkyRawResult(nil, s.RecallPrivateMessage(uid, seq))
	case "set_group_essence_message":
		gid, err := rawActionParam(params, "group_id")
		if err != nil {
			return nil, err
		}
		seq, err := rawActionParam(params, "message_seq")
		if err != nil {
			return nil, err
		}
		return milkyRawResult(nil, s.SetGroupEssenceMessage(gid, seq, rawActionBool(params, "is_set", true)))
	case "send_group_message_reaction":
		gid, err := rawActionParam(params, "group_id")
		if err != nil {
			return nil, err
		}
		seq, err := rawActionParam(params, "message_seq")
		if err != nil {
			return nil, err
		}
		return milkyRawResult(nil, s.SendGroupMessageReaction(gid, seq, rawActionStr(params, "reaction", ""), rawActionBool(params, "is_set", true)))
	case "mark_message_as_read":
		peerID, err := rawActionParam(params, "peer_id")
		if err != nil {
			return nil, err
		}
		seq, err := rawActionParam(params, "message_seq")
		if err != nil {
			return nil, err
		}
		return milkyRawResult(nil, s.MarkMessageAsRead(rawActionStr(params, "message_scene", "group"), peerID, seq))
	case "accept_friend_request":
		return milkyRawResult(nil, s.AcceptFriendRequest(rawActionStr(params, "initiator_uid", ""), rawActionBool(params, "is_filtered", false)))
	case "reject_friend_request":
		return milkyRawResult(nil, s.RejectFriendRequest(rawActionStr(params, "initiator_uid", ""), rawActionBool(params, "is_filtered", false), rawActionStr(params, "reason", "")))
	case "accept_group_request":
		seq, err := rawActionParam(params, "notification_seq")
		if err != nil {
			return nil, err
		}
		gid, err := rawActionParam(params, "group_id")
		if err != nil {
			return nil, err
		}
		return milkyRawResult(nil, s.AcceptGroupRequest(seq, rawActionStr(params, "notification_type", "group_request"), gid, rawActionBool(params, "is_filtered", false)))
	case "reject_group_invitation":
		gid, err := rawActionParam(params, "group_id")
		if err != nil {
			return nil, err
		}
		seq, err := rawActionParam(params, "invitation_seq")
		if err != nil {
			return nil, err
		}
		return milkyRawResult(nil, s.RejectGroupInvitation(gid, seq))
	default:
		return nil, fmt.Errorf("milky 不支持动作 %s", action)
	}
}
