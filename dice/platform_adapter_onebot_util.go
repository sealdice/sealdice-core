package dice

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	socketio "github.com/PaienNate/pineutil/evsocket"
	"github.com/bytedance/sonic"
	"github.com/panjf2000/ants/v2"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"go.uber.org/zap"

	"sealdice-core/dice/events"
	"sealdice-core/dice/utils/onebot/schema"
	"sealdice-core/message"
)

// ONEBOT事件对应码表
// OnebotEventPostTypeCode
const (
	OnebotEventPostTypeMessage = "onebot_message"
	// OnebotEventPostTypeCodeMessageSent 是bot发出的消息
	OnebotEventPostTypeMessageSent = "onebot_message_sent"
	OnebotEventPostTypeRequest     = "onebot_request"
	OnebotEventPostTypeNotice      = "onebot_notice"
	OnebotEventPostTypeMetaEvent   = "onebot_meta_event"
)

const (
	OnebotReceiveMessage = "onebot_echo"
)

// serveOnebotEvent 消息分发函数。该函数将所有的传入参数，全部转换为array模式
func (p *PlatformAdapterOnebot) serveOnebotEvent(ep *socketio.EventPayload) {
	p.logger.Debugf("Message event - User: %s - Message: %s", ep.Kws.GetStringAttribute("user_id"), string(ep.Data))
	if !gjson.ValidBytes(ep.Data) {
		return
	}
	// 注册Emitter
	resp := gjson.ParseBytes(ep.Data)
	if resp.Get("self_id").Int() != 0 {
		p.once.Do(func() {
			if p.sendEmitter != nil {
				_ = p.sendEmitter.SetSelfId(p.ctx, resp.Get("self_id").Int())
			}
		})
	}
	// 解析是string还是array
	// TODO: 不知道是不是通过这种方式判断是string或者array的
	switch resp.Get("message").Type {
	case gjson.String:
		p.wsmode = "string"
	case gjson.JSON:
		p.wsmode = "array"
	default:
		p.wsmode = "string"
	}
	if p.wsmode == "string" {
		resp2, err := string2array(resp)
		if err != nil {
			p.logger.Warnf("消息转换为array异常 %s 未能正确处理", resp.String())
			return
		}
		resp = resp2
	}
	// 将数据转换为对应的事件event
	eventType := resp.Get("post_type").String()
	if eventType != "" {
		// 分发事件
		eventType = fmt.Sprintf("onebot_%s", eventType)
		ep.Kws.Fire(eventType, []byte(resp.String()))
	} else {
		// 如果没有post_type，说明不是上报信息，而是API的返回信息
		ep.Kws.Fire(OnebotReceiveMessage, []byte(resp.String()))
	}
}

func (p *PlatformAdapterOnebot) onOnebotMessageEvent(ep *socketio.EventPayload) {
	// 收到普通消息的时候：执行ExecuteNew函数
	msg, err := arrayByte2SealdiceMessage(p.logger, ep.Data)
	if err != nil {
		p.logger.Errorf("收到消息但无法进行处理，原因为 %s", err)
		return
	}
	// 注册消息发送人的缓存，以兼容dice_manager
	if msg.Sender.UserID != "" && msg.Sender.Nickname != "" {
		p.Session.Parent.Parent.UserNameCache.Store(msg.Sender.UserID, &GroupNameCacheItem{Name: msg.Sender.Nickname, time: time.Now().Unix()})
	}

	p.Session.ExecuteNew(p.EndPoint, msg)
}

func (p *PlatformAdapterOnebot) onOnebotRequestEvent(ep *socketio.EventPayload) {
	// 请求分为好友请求和加群/邀请请求
	req := gjson.ParseBytes(ep.Data)
	switch req.Get("request_type").String() {
	case "friend":
		_ = p.handleReqFriendAction(req, ep)
	case "group":
		_ = p.handleReqGroupAction(req, ep)
	}
}
func (p *PlatformAdapterOnebot) OnebotNoticeEvent(ep *socketio.EventPayload) {
	// 进群致辞
	req := gjson.ParseBytes(ep.Data)
	switch req.Get("notice_type").String() {
	// 入群（强行拉群等）
	case "group_increase":
		_ = p.handleJoinGroupAction(req, ep)
	case "friend_add":
		_ = p.handleAddFriendAction(req, ep)
	case "group_ban":
		_ = p.handleGroupBanAction(req, ep)
	case "group_recall":
		_ = p.handleGroupBanAction(req, ep)
	case "notify":
		switch req.Get("sub_type").String() {
		case "poke":
			// 戳一戳
			_ = p.handleGroupPokeAction(req, ep)
		}
	}
}

func (p *PlatformAdapterOnebot) handleGroupPokeAction(req gjson.Result, _ *socketio.EventPayload) error {
	go func() {
		defer ErrorLogAndContinue(p.Session.Parent)
		msgContext := p.makeCtx(req)
		isPrivate := msgContext.MessageType == "private"
		p.Session.OnPoke(msgContext, &events.PokeEvent{
			GroupID:   FormatDiceIDQQGroup(req.Get("group_id").String()),
			SenderID:  FormatDiceIDQQ(req.Get("user_id").String()),
			TargetID:  FormatDiceIDQQ(req.Get("target_id").String()),
			IsPrivate: isPrivate,
		})
	}()
	return nil
}

func (p *PlatformAdapterOnebot) handleGroupRecallAction(_ gjson.Result, ep *socketio.EventPayload) error {
	ctx := &MsgContext{EndPoint: p.EndPoint, Session: p.Session, Dice: p.Session.Parent}
	msg, err := arrayByte2SealdiceMessage(p.logger, ep.Data)
	if err != nil {
		return err
	}
	p.Session.OnMessageDeleted(ctx, msg)
	return nil
}

func (p *PlatformAdapterOnebot) handleGroupBanAction(req gjson.Result, _ *socketio.EventPayload) error {
	ctx := &MsgContext{EndPoint: p.EndPoint, Session: p.Session, Dice: p.Session.Parent}
	subType := req.Get("sub_type").String()
	userID := FormatOnebotDiceIDQQ(req.Get("user_id").String())
	selfID := FormatOnebotDiceIDQQ(req.Get("self_id").String())
	groupId := FormatOnebotDiceIDQQGroup(req.Get("group_id").String())
	operatorID := FormatOnebotDiceIDQQ(req.Get("operator_id").String())
	durationTime := int(req.Get("duration").Int())
	duration := time.Duration(durationTime) * time.Second
	switch subType {
	case "ban":
		if userID == selfID {
			groupName := p.Session.Parent.Parent.TryGetGroupName(groupId)
			userName := p.Session.Parent.Parent.TryGetUserName(operatorID)
			ctx.Dice.Config.BanList.AddScoreByGroupMuted(operatorID, groupId, ctx)
			txt := fmt.Sprintf("被禁言: 在群组<%s>(%s)中被禁言，时长%d秒，操作者:<%s>(%s)", groupName, groupId, duration, userName, operatorID)
			p.logger.Info(txt)
			ctx.Notice(txt)
		}
	}
	return nil
}

func (p *PlatformAdapterOnebot) handleAddFriendAction(req gjson.Result, _ *socketio.EventPayload) error {
	ctx := &MsgContext{EndPoint: p.EndPoint, Session: p.Session, Dice: p.Session.Parent}
	msg, err := arrayByte2SealdiceMessage(p.logger, []byte(req.String()))
	if err != nil {
		return err
	}
	userId := FormatOnebotDiceIDQQ(req.Get("user_id").String())
	// 先查看
	ctx.Group, ctx.Player = GetPlayerInfoBySender(ctx, msg)
	welcomeStr := DiceFormatTmpl(ctx, "核心:骰子成为好友")
	p.logger.Infof("与 %s 成为好友，发送好友致辞: %s", req.Get("user_id").String(), welcomeStr)
	_ = p.antPool.Submit(func() {
		time.Sleep(2 * time.Second)
		for _, i := range ctx.SplitText(welcomeStr) {
			doSleepQQ(ctx)
			p.SendToPerson(ctx, userId, strings.TrimSpace(i), "")
		}
		groupInfo, ok := ctx.Session.ServiceAtNew.Load(msg.GroupID)
		if ok {
			for _, i := range groupInfo.ActivatedExtList {
				if i.OnBecomeFriend != nil {
					i.callWithJsCheck(ctx.Dice, func() {
						i.OnBecomeFriend(ctx, msg)
					})
				}
			}
		}
	})
	return nil
}

func (p *PlatformAdapterOnebot) handleJoinGroupAction(req gjson.Result, _ *socketio.EventPayload) error {
	// {"group_id":111,"notice_type":"group_increase","operator_id":0,"post_type":"notice","self_id":333,"sub_type":"approve","time":1646782012,"user_id":333}
	// 入群要做的事情：
	// 1. 如果发现进群的是自己，要和大家发入群致辞
	// 2. 如果发现进群的不是自己，对他进行节流的迎新
	ctx := &MsgContext{EndPoint: p.EndPoint, Session: p.Session, Dice: p.Session.Parent}
	msg, err := arrayByte2SealdiceMessage(p.logger, []byte(req.String()))
	if err != nil {
		return err
	}
	userId := FormatOnebotDiceIDQQ(req.Get("user_id").String())
	selfId := FormatOnebotDiceIDQQ(req.Get("self_id").String())
	groupId := FormatOnebotDiceIDQQGroup(req.Get("group_id").String())
	// 迎新逻辑
	// 发送入群致辞逻辑
	if userId == selfId {
		p.logger.Infof("收到自己的入群请求，准备发送入群致辞")
		ctx.Group = SetBotOnAtGroup(ctx, groupId)
		// TODO 补充注释，这是个他吗啥？
		ctx.Group.DiceIDExistsMap.Store(ctx.EndPoint.UserID, true)
		// 入群时间
		ctx.Group.EnteredTime = time.Now().Unix()
		// 更新时间
		ctx.Group.UpdatedAtTime = time.Now().Unix()
		// 获取群信息 并发送入群致辞
		_ = p.antPool.Submit(func() {
			time.Sleep(1 * time.Second)
			cache := p.GetGroupCacheInfo(groupId)
			ctx.Player = &GroupPlayerInfo{}
			p.logger.Infof("发送入群致辞，群: <%s>(%s)", cache.GroupName, groupId)
			text := DiceFormatTmpl(ctx, "核心:骰子进群")
			for _, i := range ctx.SplitText(text) {
				doSleepQQ(ctx)
				p.SendToGroup(ctx, groupId, strings.TrimSpace(i), "")
			}
			groupInfo, ok := ctx.Session.ServiceAtNew.Load(groupId)
			if ok {
				for _, i := range groupInfo.ActivatedExtList {
					if i.OnGroupJoined != nil {
						i.callWithJsCheck(ctx.Dice, func() {
							i.OnGroupJoined(ctx, msg)
						})
					}
				}
			}
		})
	} else {
		p.logger.Infof("收到非自己的入群请求，准备迎新")
		_ = p.antPool.Submit(func() {
			time.Sleep(1 * time.Second) // 避免是正在拉人进群的情况，先等一下再取数据
			group, ok := ctx.Session.ServiceAtNew.Load(msg.GroupID)
			if ok {
				ctx.Group = group
				ctx.Player = &GroupPlayerInfo{}
				uidRaw := req.Get("user_id").String()
				VarSetValueStr(ctx, "$t帐号ID_RAW", uidRaw)
				VarSetValueStr(ctx, "$t账号ID_RAW", uidRaw)
				stdID := userId
				VarSetValueStr(ctx, "$t帐号ID", stdID)
				VarSetValueStr(ctx, "$t账号ID", stdID)
				text := DiceFormat(ctx, group.GroupWelcomeMessage)
				for _, i := range ctx.SplitText(text) {
					doSleepQQ(ctx)
					p.SendToGroup(ctx, msg.GroupID, strings.TrimSpace(i), "")
				}
			}
		})
	}

	return nil
}

// 加群逻辑里比较复杂，列在这里
// 加群：被好友邀请-> 获取群信息 -> 根据获取的群信息，判断是否应该加群
func (p *PlatformAdapterOnebot) handleReqGroupAction(req gjson.Result, _ *socketio.EventPayload) error {
	// 创建虚拟Context
	ctx := &MsgContext{EndPoint: p.EndPoint, Session: p.Session, Dice: p.Session.Parent}
	switch req.Get("sub_type").String() {
	case "invite":
		// 获取群信息
		diceGroupId := FormatOnebotDiceIDQQGroup(req.Get("group_id").String())
		diceUserId := FormatOnebotDiceIDQQ(req.Get("user_id").String())
		res := p.GetGroupCacheInfo(diceGroupId)
		if res == nil {
			// 没有群信息，默认群信息创建
			res = &GroupCache{
				GroupAllShut:   0,
				GroupRemark:    "",
				GroupId:        diceGroupId,
				GroupName:      "%未知群聊%",
				MemberCount:    1,
				MaxMemberCount: 2000,
			}
		}
		// 先判断是否需要加群
		ok, reason := checkPassBlackListGroup(diceUserId, diceGroupId, ctx)
		if !ok {
			p.logger.Infof("群组 %s 加群请求被拒绝，原因为 %s", req.Get("group_id").String(), reason)
			err := ants.Submit(func() {
				err := p.sendEmitter.SetGroupAddRequest(p.ctx, req.Get("flag").String(), false, reason)
				if err != nil {
					p.logger.Errorf("处理加群请求时发送消息失败 %s", err)
				}
			})
			if err != nil {
				return err
			}
		}
		// 没问题，加群
		_ = ants.Submit(func() {
			err := p.sendEmitter.SetGroupAddRequest(p.ctx, req.Get("flag").String(), true, "")
			if err != nil {
				p.logger.Errorf("处理加群请求时发送消息失败 %s", err)
			}
		})
	default:
		// DO NOTHING NOW
	}
	return nil
}

func checkPassBlackListGroup(userId string, groupID string, ctx *MsgContext) (bool, string) {
	userResult := checkBlackList(userId, "user", ctx)
	if !userResult.Passed {
		return false, userResult.Reason
	}
	groupResult := checkBlackList(groupID, "group", ctx)
	if !groupResult.Passed {
		return false, groupResult.Reason
	}
	return true, ""
}

func (p *PlatformAdapterOnebot) handleReqFriendAction(req gjson.Result, _ *socketio.EventPayload) error {
	// 只有一种情况 就是好友添加
	// 获取请求详情
	var comment string
	if req.Get("comment").Exists() {
		comment = strings.TrimSpace(req.Get("comment").String())
		comment = strings.ReplaceAll(comment, "\u00a0", "")
	}
	// 将匹配的验证问题
	toMatch := strings.TrimSpace(p.Session.Parent.Config.FriendAddComment)
	// 创建虚构MsgContext
	ctx := &MsgContext{EndPoint: p.EndPoint, Session: p.Session, Dice: p.Session.Parent}
	var extra string
	// 匹配验证问题检查
	var passQuestion bool
	var passblackList bool
	if comment != DiceFormat(ctx, toMatch) {
		passQuestion = checkMultiFriendAddVerify(comment, toMatch)
	}
	// 匹配黑名单检查
	result := checkBlackList(req.Get("user_id").String(), "user", ctx)
	// 格式化请求的数据
	comment = strconv.Quote(comment)
	if comment == "" {
		comment = "(无)"
	}
	if !passQuestion {
		extra = "。回答错误"
	} else {
		extra = "。回答正确"
	}
	if !result.Passed {
		extra += "。（被禁止用户）"
		extra += "，原因：" + result.Reason
	}
	// TODO：暂时不做
	// if p.IgnoreFriendRequest {
	//	extra += "。由于设置了忽略邀请，此信息仅为通报"
	// }

	txt := fmt.Sprintf("收到QQ好友邀请: 邀请人:%s, 验证信息: %s, 是否自动同意: %t%s", req.Get("user_id").String(), comment, passQuestion && passblackList, extra)
	p.logger.Info(txt)
	ctx.Notice(txt)
	err := p.sendEmitter.SetFriendAddRequest(p.ctx, req.Get("flag").String(), true, "")
	if err != nil {
		return err
	}
	return nil
}

// 检查加好友是否成功
func checkMultiFriendAddVerify(comment string, toMatch string) bool {
	// 根据GPT的描述，这里干的事情是：从评论中提取回答内容，并与目标字符串进行逐项匹配，最终决定是否接受。
	// 我只是从木落那里拆了过来，太热闹了。
	var willAccept bool
	re := regexp.MustCompile(`\n回答:([^\n]+)`)
	m := re.FindAllStringSubmatch(comment, -1)
	// 要匹配的是空，说明不验证
	if toMatch == "" {
		willAccept = true
		return willAccept
	}
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
	return willAccept
}

// TODO

// 					// 处理被强制拉群的情况
//					uid := groupInfo.InviteUserID
//					banInfo, ok := ctx.Dice.Config.BanList.GetByID(uid)
//					if ok {
//						if banInfo.Rank == BanRankBanned && ctx.Dice.Config.BanList.BanBehaviorRefuseInvite {
//							// 如果是被ban之后拉群，判定为强制拉群
//							if groupInfo.EnteredTime > 0 && groupInfo.EnteredTime > banInfo.BanTime {
//								text := fmt.Sprintf("本次入群为遭遇强制邀请，即将主动退群，因为邀请人%s正处于黑名单上。打扰各位还请见谅。感谢使用海豹核心。", groupInfo.InviteUserID)
//								ReplyGroupRaw(ctx, &Message{GroupID: groupID}, text, "")
//								time.Sleep(1 * time.Second)
//								pa.QuitGroup(ctx, groupID)
//							}
//							return
//						}
//					}
//
//					// 强制拉群情况2 - 群在黑名单
//					banInfo, ok = ctx.Dice.Config.BanList.GetByID(groupID)
//					if ok {
//						if banInfo.Rank == BanRankBanned {
//							// 如果是被ban之后拉群，判定为强制拉群
//							if groupInfo.EnteredTime > 0 && groupInfo.EnteredTime > banInfo.BanTime {
//								text := fmt.Sprintf("被群已被拉黑，即将自动退出，解封请联系骰主。打扰各位还请见谅。感谢使用海豹核心:\n当前情况: %s", banInfo.toText(ctx.Dice))
//								ReplyGroupRaw(ctx, &Message{GroupID: groupID}, text, "")
//								time.Sleep(1 * time.Second)
//								pa.QuitGroup(ctx, groupID)
//							}
//							return
//						}
//					}

type BlackListCheckResult struct {
	Passed      bool
	FailureType string           // "user_banned", "group_banned", "trust_mode", "refuse_invite"
	Reason      string           // 具体原因
	BanInfo     *BanListInfoItem // 详细的封禁信息，用于后续逻辑
}

// checkBlackList 检查用户或群组是否通过黑名单检查
// 参数:
//   - userId: 用户ID或群组ID
//   - checkType: 检查类型，"user"或"group"
//   - ctx: 消息上下文
//
// 返回值:
//   - BlackListCheckResult: 包含检查结果和详细信息
func checkBlackList(userId string, checkType string, ctx *MsgContext) BlackListCheckResult {
	result := BlackListCheckResult{
		Passed: true,
	}

	// 检查 userId 是否有效
	if userId == "" {
		return result
	}

	// 获取禁用信息
	banInfo, ok := ctx.Dice.Config.BanList.GetByID(userId)
	if !ok || banInfo == nil {
		return result // 如果不在黑名单中，默认通过
	}

	result.BanInfo = banInfo

	switch checkType {
	case "user":
		// 检查用户是否被ban且配置了拒绝邀请
		if banInfo.Rank == BanRankBanned && ctx.Dice.Config.BanList.BanBehaviorRefuseInvite {
			result.Passed = false
			result.FailureType = "user_banned"
			result.Reason = "邀请人在黑名单上"
		}
	case "group":
		// 检查群组是否被ban
		if banInfo.Rank == BanRankBanned {
			result.Passed = false
			result.FailureType = "group_banned"
			result.Reason = "群组在黑名单上"
			return result
		}

		// 信任模式检查
		isMaster := ctx.Dice.IsMaster(userId)
		if ctx.Dice.Config.TrustOnlyMode && banInfo.Rank != BanRankTrusted && !isMaster {
			result.Passed = false
			result.FailureType = "trust_mode"
			result.Reason = "只允许信任的人拉群"
			return result
		}

		// 拒绝所有群邀请的配置检查
		if ctx.Dice.Config.RefuseGroupInvite {
			result.Passed = false
			result.FailureType = "refuse_invite"
			result.Reason = "拒绝拉群邀请"
		}
	}

	return result
}

func (p *PlatformAdapterOnebot) onOnebotMetaDataEvent(ep *socketio.EventPayload) {

}

type GroupCache struct {
	GroupAllShut   int    `json:"group_all_shut"`   // 是否全员禁言
	GroupRemark    string `json:"group_remark"`     // 群备注
	GroupId        string `json:"group_id"`         // 群ID
	GroupName      string `json:"group_name"`       // 群名
	MemberCount    int    `json:"member_count"`     // 群人数
	MaxMemberCount int    `json:"max_member_count"` // 群最大人数
}

func FormatOnebotDiceIDQQ(diceQQ string) string {
	return fmt.Sprintf("QQ:%s", diceQQ)
}

func FormatOnebotDiceIDQQGroup(diceQQ string) string {
	return fmt.Sprintf("QQ-Group:%s", diceQQ)
}

type MessageQQOBBase struct {
	MessageID     int64           `json:"message_id"`   // QQ信息此类型为int64，频道中为string
	MessageType   string          `json:"message_type"` // Group
	Sender        *Sender         `json:"sender"`       // 发送者
	RawMessage    string          `json:"raw_message"`
	Time          int64           `json:"time"` // 发送时间
	MetaEventType string          `json:"meta_event_type"`
	OperatorID    json.RawMessage `json:"operator_id"`  // 操作者帐号
	GroupID       json.RawMessage `json:"group_id"`     // 群号
	PostType      string          `json:"post_type"`    // 上报类型，如group、notice
	RequestType   string          `json:"request_type"` // 请求类型，如group
	SubType       string          `json:"sub_type"`     // 子类型，如add invite
	Flag          string          `json:"flag"`         // 请求 flag, 在调用处理请求的 API 时需要传入
	NoticeType    string          `json:"notice_type"`
	UserID        json.RawMessage `json:"user_id"`
	SelfID        json.RawMessage `json:"self_id"`
	Duration      int64           `json:"duration"`
	Comment       string          `json:"comment"`
	TargetID      json.RawMessage `json:"target_id"`

	Data *struct {
		// 个人信息
		Nickname string          `json:"nickname"`
		UserID   json.RawMessage `json:"user_id"`

		// 群信息
		GroupID         json.RawMessage `json:"group_id"`          // 群号
		GroupCreateTime uint32          `json:"group_create_time"` // 群号
		MemberCount     int64           `json:"member_count"`
		GroupName       string          `json:"group_name"`
		MaxMemberCount  int32           `json:"max_member_count"`

		// 群成员信息
		Card string `json:"card"`
	} `json:"data"`
	Retcode int64 `json:"retcode"`
	// Status string `json:"status"`
	Echo json.RawMessage `json:"echo"` // 声明类型而不是interface的原因是interface下数字不能正确转换

	Msg string `json:"msg"`
	// Status  interface{} `json:"status"`
	Wording string `json:"wording"`
}

type MessageOBQQ struct {
	MessageQQOBBase
}

func (msgQQ *MessageOBQQ) toStdMessage() *Message {
	msg := new(Message)
	msg.Time = msgQQ.Time
	msg.MessageType = msgQQ.MessageType
	msg.RawID = msgQQ.MessageID
	msg.Platform = "QQ"

	if msg.MessageType == "" {
		msg.MessageType = "private"
	}

	if msgQQ.Data != nil && len(msgQQ.Data.GroupID) > 0 {
		msg.GroupID = FormatOnebotDiceIDQQGroup(string(msgQQ.Data.GroupID))
	}
	if string(msgQQ.GroupID) != "" {
		if msg.MessageType == "private" {
			msg.MessageType = "group"
		}
		msg.GroupID = FormatOnebotDiceIDQQGroup(string(msgQQ.GroupID))
	}
	if msgQQ.Sender != nil {
		msg.Sender.Nickname = msgQQ.Sender.Nickname
		if msgQQ.Sender.Card != "" {
			msg.Sender.Nickname = msgQQ.Sender.Card
		}
		msg.Sender.GroupRole = msgQQ.Sender.Role
		msg.Sender.UserID = FormatOnebotDiceIDQQ(string(msgQQ.Sender.UserID))
	}
	return msg
}

func arrayByte2SealdiceMessage(log *zap.SugaredLogger, raw []byte) (*Message, error) {
	// 不合法的信息体
	if !gjson.ValidBytes(raw) {
		log.Warn("无法解析 onebot11 字段:", raw)
		return nil, errors.New("解析失败")
	}
	var obMsg MessageOBQQ
	// 原版本转换为gjson对象
	parseContent := gjson.ParseBytes(raw)
	err := sonic.Unmarshal(raw, &obMsg)
	if err != nil {
		return nil, err
	}
	m := obMsg.toStdMessage()
	arrayContent := parseContent.Get("message").Array()
	seg := make([]message.IMessageElement, 0)
	cqMessage := strings.Builder{}
	for _, i := range arrayContent {
		// 使用String()方法，如果为空，会自动产生空字符串
		typeStr := i.Get("type").String()
		dataObj := i.Get("data")
		switch typeStr {
		case "text":
			rawTxt := dataObj.Get("text").String()
			seg = append(seg, &message.TextElement{
				Content: rawTxt,
			})
			cqMessage.WriteString(rawTxt)
		case "image":
			rawImg := &message.ImageElement{
				File: &message.FileElement{
					URL: dataObj.Get("file").String(),
				},
			}
			// 兼容NC情况, 此时file字段只有文件名, 完整URL在url字段
			if !hasURLScheme(dataObj.Get("file").String()) && hasURLScheme(dataObj.Get("url").String()) {
				rawImg.File.URL = dataObj.Get("url").String()
				cqMessage.WriteString(fmt.Sprintf("[CQ:image,file=%v]", dataObj.Get("url").String()))
			} else {
				rawImg.File.URL = dataObj.Get("file").String()
				cqMessage.WriteString(fmt.Sprintf("[CQ:image,file=%v]", dataObj.Get("file").String()))
			}
			seg = append(seg, rawImg)
		case "face":
			cqMessage.WriteString(fmt.Sprintf("[CQ:face,id=%v]", dataObj.Get("id").String()))
			seg = append(seg, &message.FaceElement{FaceID: dataObj.Get("id").String()})
		case "record":
			recordRaw := message.RecordElement{File: &message.FileElement{
				URL: "",
			}}
			// 兼容NC情况, 此时file字段只有文件名, 完整路径在path字段
			if !hasURLScheme(dataObj.Get("file").String()) && dataObj.Get("path").String() != "" {
				recordRaw.File.URL = dataObj.Get("path").String()
				cqMessage.WriteString(fmt.Sprintf("[CQ:record,file=%v]", dataObj.Get("path").String()))
			} else {
				recordRaw.File.URL = dataObj.Get("file").String()
				cqMessage.WriteString(fmt.Sprintf("[CQ:record,file=%v]", dataObj.Get("file").String()))
			}
			seg = append(seg, &recordRaw)
		case "at":
			cqMessage.WriteString(fmt.Sprintf("[CQ:at,qq=%v]", dataObj.Get("qq").String()))
			seg = append(seg, &message.AtElement{Target: dataObj.Get("qq").String()})
		case "poke":
			cqMessage.WriteString("[CQ:poke]")
			seg = append(seg, &message.PokeElement{})
		case "reply":
			cqMessage.WriteString(fmt.Sprintf("[CQ:reply,id=%v]", dataObj.Get("id").String()))
			seg = append(seg, &message.ReplyElement{
				ReplySeq: dataObj.Get("id").String(),
			})
		}
	}
	// 获取Message
	m.Message = cqMessage.String()
	m.Message = strings.ReplaceAll(m.Message, "&#91;", "[")
	m.Message = strings.ReplaceAll(m.Message, "&#93;", "]")
	m.Message = strings.ReplaceAll(m.Message, "&amp;", "&")
	// 获取Segment
	m.Segment = seg
	return m, nil
}

// 将OB11的Array数据转换为string字符串
func array2string(parseContent gjson.Result) (gjson.Result, error) {
	arrayContent := parseContent.Get("message").Array()
	cqMessage := strings.Builder{}

	for _, i := range arrayContent {
		typeStr := i.Get("type").String()
		dataObj := i.Get("data")
		switch typeStr {
		case "text":
			cqMessage.WriteString(dataObj.Get("text").String())
		case "image":
			// 兼容NC情况, 此时file字段只有文件名, 完整URL在url字段
			if !hasURLScheme(dataObj.Get("file").String()) && hasURLScheme(dataObj.Get("url").String()) {
				cqMessage.WriteString(fmt.Sprintf("[CQ:image,file=%v]", dataObj.Get("url").String()))
			} else {
				cqMessage.WriteString(fmt.Sprintf("[CQ:image,file=%v]", dataObj.Get("file").String()))
			}
		case "face":
			// 兼容四叶草，移除 .(string)。自动获取的信息表示此类型为 float64，这是go解析的问题
			cqMessage.WriteString(fmt.Sprintf("[CQ:face,id=%v]", dataObj.Get("id").String()))
		case "record":
			// 兼容NC情况, 此时file字段只有文件名, 完整路径在path字段
			if !hasURLScheme(dataObj.Get("file").String()) && dataObj.Get("path").String() != "" {
				cqMessage.WriteString(fmt.Sprintf("[CQ:record,file=%v]", dataObj.Get("path").String()))
			} else {
				cqMessage.WriteString(fmt.Sprintf("[CQ:record,file=%v]", dataObj.Get("file").String()))
			}
		case "at":
			cqMessage.WriteString(fmt.Sprintf("[CQ:at,qq=%v]", dataObj.Get("qq").String()))
		case "poke":
			cqMessage.WriteString(fmt.Sprintf("[CQ:poke,qq=%v]", dataObj.Get("qq").String()))
		case "reply":
			cqMessage.WriteString(fmt.Sprintf("[CQ:reply,id=%v]", dataObj.Get("id").String()))
		}
	}
	// 赋值对应的Message
	tempStr, err := sjson.Set(parseContent.String(), "message", cqMessage.String())
	if err != nil {
		return gjson.Result{}, err
	}
	return gjson.Parse(tempStr), nil
}

// 将CQ码字符串消息转换为OB11 Array格式
func string2array(parseContent gjson.Result) (gjson.Result, error) {
	messageStr := parseContent.Get("message").String()
	var messageArray []map[string]interface{}

	// 使用正则表达式匹配CQ码和普通文本
	re := regexp.MustCompile(`(\[CQ:\w+,[^\]]+\])|([^\[\]]+)`)
	matches := re.FindAllStringSubmatch(messageStr, -1)

	for _, match := range matches {
		if match[1] != "" {
			// 处理CQ码
			cqCode := match[1]
			cqType := strings.TrimPrefix(strings.Split(cqCode, ",")[0], "[CQ:")
			cqType = strings.TrimSuffix(cqType, "]")

			// 解析CQ码参数
			params := make(map[string]string)
			paramPairs := strings.Split(strings.TrimSuffix(strings.Split(cqCode, ",")[1], "]"), ",")
			for _, pair := range paramPairs {
				kv := strings.SplitN(pair, "=", 2)
				if len(kv) == 2 {
					params[kv[0]] = kv[1]
				}
			}

			// 转换为OB11格式
			item := map[string]interface{}{
				"type": cqType,
				"data": params,
			}
			messageArray = append(messageArray, item)
		} else if match[2] != "" {
			// 处理普通文本
			item := map[string]interface{}{
				"type": "text",
				"data": map[string]string{
					"text": match[2],
				},
			}
			messageArray = append(messageArray, item)
		}
	}

	// 赋值对应的Message
	tempStr, err := sjson.Set(parseContent.String(), "message", messageArray)
	if err != nil {
		return gjson.Result{}, err
	}
	return gjson.Parse(tempStr), nil
}

func convertSealMsgToMessageChain(msg []message.IMessageElement) (schema.MessageChain, string) {
	cqMessage := strings.Builder{}
	rawMsg := schema.MessageChain{}
	for _, v := range msg {
		switch v.Type() {
		case message.At:
			res, ok := v.(*message.AtElement)
			if !ok {
				continue
			}
			rawMsg.At(res.Target)
			cqMessage.WriteString(fmt.Sprintf("[CQ:at,qq=%v]", res.Target))
		case message.Text:
			res, ok := v.(*message.TextElement)
			if !ok {
				continue
			}
			rawMsg = rawMsg.Text(res.Content)
			cqMessage.WriteString(res.Content)
		case message.Face:
			res, ok := v.(*message.FaceElement)
			if !ok {
				continue
			}
			rawMsg = rawMsg.Face(res.FaceID)
			cqMessage.WriteString(fmt.Sprintf("[CQ:face,id=%v]", res.FaceID))
		case message.File:
			res, ok := v.(*message.FileElement)
			if !ok {
				continue
			}
			rawMsg = rawMsg.File(res.File)
			cqMessage.WriteString(fmt.Sprintf("[CQ:file,file=%v]", res.File))
		case message.Image:
			res, ok := v.(*message.ImageElement)
			if !ok {
				continue
			}
			url := res.URL
			if res.URL == "" {
				url = res.File.URL
			}
			rawMsg = rawMsg.Image(url)
			cqMessage.WriteString(fmt.Sprintf("[CQ:image,file=%v]", url))
		case message.Record:
			res, ok := v.(*message.RecordElement)
			if !ok {
				continue
			}
			rawMsg = rawMsg.Record(res.File.URL)
			cqMessage.WriteString(fmt.Sprintf("[CQ:record,file=%v]", res.File.URL))
		case message.Reply:
			res, ok := v.(*message.ReplyElement)
			if !ok {
				continue
			}
			parseInt, err := strconv.Atoi(res.ReplySeq)
			if err != nil {
				continue
			}
			rawMsg = rawMsg.Reply(parseInt)
			cqMessage.WriteString(fmt.Sprintf("[CQ:reply,id=%v]", parseInt))
		case message.TTS:
			res, ok := v.(*message.TTSElement)
			if !ok {
				continue
			}
			m := map[string]string{
				"text": res.Content,
			}
			marshal, err := sonic.Marshal(m)
			if err != nil {
				continue
			}
			rawMsg = rawMsg.Append(schema.Message{
				Type: "tts",
				Data: marshal,
			})
			cqMessage.WriteString(fmt.Sprintf("[CQ:tts,text=%v]", res.Content))
		case message.Poke:
			res, ok := v.(*message.PokeElement)
			if !ok {
				continue
			}
			m := map[string]string{
				"qq": res.Target,
			}
			marshal, err := sonic.Marshal(m)
			if err != nil {
				continue
			}
			rawMsg = rawMsg.Append(schema.Message{
				Type: "poke",
				Data: marshal,
			})
			cqMessage.WriteString(fmt.Sprintf("[CQ:poke,qq=%v]", res.Target))
		}
	}
	return rawMsg, cqMessage.String()
}

func ExtractQQEmitterUserID(id string) int64 {
	if strings.HasPrefix(id, "QQ:") {
		atoi, _ := strconv.ParseInt(id[len("QQ:"):], 10, 64)
		return atoi
	}
	return 0
}

func ExtractQQEmitterGroupID(id string) int64 {
	if strings.HasPrefix(id, "QQ-Group:") {
		atoi, _ := strconv.ParseInt(id[len("QQ-Group:"):], 10, 64)
		return atoi
	}
	atoi, _ := strconv.ParseInt(id[len("QQ-Group:"):], 10, 64)
	return atoi
}

func (pa *PlatformAdapterOnebot) makeCtx(req gjson.Result) *MsgContext {
	ep := pa.EndPoint
	session := pa.Session
	var messageType = "private"
	if req.Get("group_id").Exists() {
		messageType = "group"
	}
	ctx := &MsgContext{MessageType: messageType, EndPoint: ep, Session: session, Dice: session.Parent}
	wrapper := MessageWrapper{
		MessageType: ctx.MessageType,
		GroupID:     FormatOnebotDiceIDQQ(req.Get("group_id").String()),
		Sender: struct {
			UserID   string
			Nickname string
		}{},
		// TODO: 这两个暂时不知道干啥的
		GuildID:   "",
		ChannelID: "",
	}
	switch ctx.MessageType {
	case "private":
		// 拿到ID
		info, err := pa.sendEmitter.GetStrangerInfo(pa.ctx, req.Get("user_id").Int(), false)
		if err != nil {
			return ctx
		}
		// 设置名称
		wrapper.Sender.UserID = FormatOnebotDiceIDQQ(strconv.FormatInt(info.UserId, 10))
		wrapper.Sender.Nickname = info.NickName
		ctx.Group, ctx.Player = GetPlayerInfoBySenderRaw(ctx, &wrapper)
		if ctx.Player.Name == "" {
			ctx.Player.Name = info.NickName
			ctx.Player.UpdatedAtTime = time.Now().Unix()
		}
		SetTempVars(ctx, info.NickName)
	case "group":
		resp, err := pa.sendEmitter.Raw(pa.ctx, "get_group_member_info", map[string]interface{}{
			"group_id": req.Get("group_id").String(),
			"user_id":  req.Get("user_id").String(),
			"no_cache": false,
		})
		if err != nil {
			return ctx
		}
		respResult := gjson.ParseBytes(resp)
		wrapper.Sender.UserID = FormatOnebotDiceIDQQ(respResult.Get("data.user_id").String())
		wrapper.Sender.Nickname = respResult.Get("data.nickname").String()
		ctx.Group, ctx.Player = GetPlayerInfoBySenderRaw(ctx, &wrapper)
		if ctx.Group == nil {
			gi := pa.GetGroupCacheInfo(FormatOnebotDiceIDQQGroup(req.Get("group_id").String()))
			ctx.Group = &GroupInfo{GroupID: gi.GroupId, GroupName: gi.GroupName}
			ctx.Group.UpdatedAtTime = time.Now().Unix()
		}
		if ctx.Player == nil {
			ctx.Player = &GroupPlayerInfo{UserID: wrapper.Sender.UserID}
		}
		if ctx.Player.Name == "" {
			if respResult.Get("data.card").String() == "" {
				ctx.Player.Name = respResult.Get("data.nickname").String()
			} else {
				ctx.Player.Name = respResult.Get("data.card").String()
			}
			ctx.Player.UpdatedAtTime = time.Now().Unix()
		}
		SetTempVars(ctx, respResult.Get("data.nickname").String())
	}

	return ctx
}
