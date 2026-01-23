package dice

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	evsocket "github.com/PaienNate/pineutil/evsocket"
	"github.com/bytedance/sonic"
	"github.com/panjf2000/ants/v2"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"go.uber.org/zap"

	"sealdice-core/dice/events"
	"sealdice-core/dice/imsdk/onebot/schema"
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
func (p *PlatformAdapterOnebot) serveOnebotEvent(ep *evsocket.EventPayload) {
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

func (p *PlatformAdapterOnebot) onOnebotMessageEvent(ep *evsocket.EventPayload) {
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

func (p *PlatformAdapterOnebot) onOnebotRequestEvent(ep *evsocket.EventPayload) {
	// 请求分为好友请求和加群/邀请请求
	req := gjson.ParseBytes(ep.Data)
	switch req.Get("request_type").String() {
	case "friend":
		_ = p.handleReqFriendAction(req, ep)
	case "group":
		_ = p.handleReqGroupAction(req, ep)
	}
}
func (p *PlatformAdapterOnebot) OnebotNoticeEvent(ep *evsocket.EventPayload) {
	// 进群致辞
	req := gjson.ParseBytes(ep.Data)
	switch req.Get("notice_type").String() {
	// 入群（强行拉群等）
	case "group_increase":
		_ = p.handleJoinGroupAction(req, ep)
	case "group_decrease":
		_ = p.handleGroupDecreaseAction(req, ep)
	case "friend_add":
		_ = p.handleAddFriendAction(req, ep)
	case "group_ban":
		_ = p.handleGroupBanAction(req, ep)
	case "group_recall":
		_ = p.handleGroupRecallAction(req, ep)
	case "notify":
		switch req.Get("sub_type").String() {
		case "poke":
			// 戳一戳
			_ = p.handleGroupPokeAction(req, ep)
		}
	}
}

func (p *PlatformAdapterOnebot) handleGroupDecreaseAction(req gjson.Result, _ *evsocket.EventPayload) error {
	ctx := &MsgContext{EndPoint: p.EndPoint, Session: p.Session, Dice: p.Session.Parent}
	subType := req.Get("sub_type").String()
	switch subType {
	case "kick_me":
		p.Session.OnGroupLeave(ctx, &events.GroupLeaveEvent{
			GroupID:    FormatOnebotDiceIDQQGroup(req.Get("group_id").String()),
			UserID:     FormatOnebotDiceIDQQ(req.Get("user_id").String()),
			OperatorID: FormatOnebotDiceIDQQ(req.Get("operator_id").String()),
		})
		// 离开群 群解散 别人被踹了
	case "leave", "disband":
		// 先获取被操作者，看看是否和自己是同一个人
		selfID := FormatOnebotDiceIDQQ(req.Get("self_id").String())
		operatorId := FormatOnebotDiceIDQQ(req.Get("operator_id").String())
		if selfID != operatorId {
			// 别人离开群的情况
			return nil
		}
		groupId := FormatOnebotDiceIDQQGroup(req.Get("group_id").String())
		groupName := p.Session.Parent.Parent.TryGetGroupName(groupId)
		txt := fmt.Sprintf("离开群组或群解散: <%s>(%s)", groupName, groupId)
		group, exists := p.Session.ServiceAtNew.Load(groupId)
		if !exists {
			txtErr := fmt.Sprintf("离开群组或群解散，删除对应群聊信息失败: <%s>(%s)", groupName, groupId)
			p.logger.Error(txtErr)
			ctx.Notice(txtErr)
		}
		group.DiceIDExistsMap.Delete(p.EndPoint.UserID)
		group.MarkDirty(p.Session.Parent)
		p.logger.Info(txt)
		ctx.Notice(txt)
	}

	return nil
}

func (p *PlatformAdapterOnebot) handleGroupPokeAction(req gjson.Result, _ *evsocket.EventPayload) error {
	go func() {
		defer ErrorLogAndContinue(p.Session.Parent)
		msgContext := p.makeCtx(req)
		isPrivate := msgContext.MessageType == "private"
		groupID := ""
		if req.Get("group_id").Exists() {
			groupID = FormatDiceIDQQGroup(req.Get("group_id").String())
		}
		p.Session.OnPoke(msgContext, &events.PokeEvent{
			GroupID:   groupID,
			SenderID:  FormatDiceIDQQ(req.Get("user_id").String()),
			TargetID:  FormatDiceIDQQ(req.Get("target_id").String()),
			IsPrivate: isPrivate,
		})
	}()
	return nil
}

func (p *PlatformAdapterOnebot) handleGroupRecallAction(_ gjson.Result, ep *evsocket.EventPayload) error {
	ctx := &MsgContext{EndPoint: p.EndPoint, Session: p.Session, Dice: p.Session.Parent}
	msg, err := arrayByte2SealdiceMessage(p.logger, ep.Data)
	if err != nil {
		return err
	}
	p.Session.OnMessageDeleted(ctx, msg)
	return nil
}

func (p *PlatformAdapterOnebot) handleGroupBanAction(req gjson.Result, _ *evsocket.EventPayload) error {
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

func (p *PlatformAdapterOnebot) handleAddFriendAction(req gjson.Result, _ *evsocket.EventPayload) error {
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
		if groupInfo, ok := ctx.Session.ServiceAtNew.Load(msg.GroupID); ok {
			groupInfo.TriggerExtHook(ctx.Dice, func(ext *ExtInfo) func() {
				if ext.OnBecomeFriend == nil {
					return nil
				}
				return func() { ext.OnBecomeFriend(ctx, msg) }
			})
		}
	})
	return nil
}

func (p *PlatformAdapterOnebot) handleJoinGroupAction(req gjson.Result, _ *evsocket.EventPayload) error {
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
		ctx.Group.DiceIDExistsMap.Store(ctx.EndPoint.UserID, true)
		// 入群时间
		ctx.Group.EnteredTime = time.Now().Unix()
		// 标记脏数据
		ctx.Group.MarkDirty(ctx.Dice)
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
			if groupInfo, ok := ctx.Session.ServiceAtNew.Load(groupId); ok {
				groupInfo.TriggerExtHook(ctx.Dice, func(ext *ExtInfo) func() {
					if ext.OnGroupJoined == nil {
						return nil
					}
					return func() { ext.OnGroupJoined(ctx, msg) }
				})
			}
		})
	} else {
		p.logger.Infof("收到非自己的入群请求，准备迎新")
		_ = p.antPool.Submit(func() {
			time.Sleep(1 * time.Second) // 避免是正在拉人进群的情况（此时会出现大量的迎新），先等一下再取数据
			group, ok := ctx.Session.ServiceAtNew.Load(msg.GroupID)
			if ok && group.ShowGroupWelcome {
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
func (p *PlatformAdapterOnebot) handleReqGroupAction(req gjson.Result, _ *evsocket.EventPayload) error {
	// 创建虚拟Context
	ctx := &MsgContext{EndPoint: p.EndPoint, Session: p.Session, Dice: p.Session.Parent}
	switch req.Get("sub_type").String() {
	case "invite":
		// 获取群信息
		diceGroupId := FormatOnebotDiceIDQQGroup(req.Get("group_id").String())
		diceUserId := FormatOnebotDiceIDQQ(req.Get("user_id").String())
		userName := p.Session.Parent.Parent.TryGetUserName(diceUserId)
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
			txt := fmt.Sprintf("收到QQ加群邀请: 群组<%s>(%s) 邀请人:<%s>(%s)", res.GroupName, res.GroupId, userName, diceUserId)
			p.logger.Info(txt)
			ctx.Notice(txt)
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

func (p *PlatformAdapterOnebot) handleReqFriendAction(req gjson.Result, _ *evsocket.EventPayload) error {
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
	if comment != DiceFormat(ctx, toMatch) {
		passQuestion = checkMultiFriendAddVerify(comment, toMatch)
	}
	// 匹配黑名单检查
	result := checkBlackList(req.Get("user_id").String(), "user", ctx)

	// 格式化请求的数据
	if comment == "" {
		comment = "(无)"
	} else {
		comment = strconv.Quote(comment)
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
	if p.IgnoreFriendRequest {
		extra += "。由于设置了忽略邀请，此信息仅为通报"
	}
	txt := fmt.Sprintf("收到QQ好友邀请: 邀请人:%s, 验证信息: %s, 是否自动同意: %t%s", req.Get("user_id").String(), comment, passQuestion && result.Passed, extra)
	p.logger.Info(txt)
	ctx.Notice(txt)
	// 若忽略邀请，对操作不通过也不拒绝，哪怕他是黑名单里的
	if !p.IgnoreFriendRequest {
		err := p.sendEmitter.SetFriendAddRequest(p.ctx, req.Get("flag").String(), result.Passed && passQuestion, "")
		if err != nil {
			return err
		}
	}
	return nil
}

// 检查加好友是否成功
func checkMultiFriendAddVerify(comment string, toMatch string) bool {
	// 如果目标匹配字符串为空，直接返回true
	if toMatch == "" {
		return true
	}

	// 提取评论中的所有回答
	re := regexp.MustCompile(`\n回答:([^\n]+)`)
	matches := re.FindAllStringSubmatch(comment, -1)

	// 提取回答内容
	answers := make([]string, 0, len(matches))
	for _, match := range matches {
		answers = append(answers, match[1])
	}

	// 分割目标匹配字符串
	expectedItems := regexp.MustCompile(`\s+`).Split(toMatch, -1)

	// 比较长度和内容
	if len(expectedItems) != len(answers) {
		return false
	}

	for i, item := range expectedItems {
		if item != answers[i] {
			return false
		}
	}

	return true
}

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

func (p *PlatformAdapterOnebot) onOnebotMetaDataEvent(ep *evsocket.EventPayload) {

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
	MessageID     int64                  `json:"message_id"`   // QQ信息此类型为int64，频道中为string
	MessageType   string                 `json:"message_type"` // Group
	Sender        *Sender                `json:"sender"`       // 发送者
	RawMessage    string                 `json:"raw_message"`
	Time          int64                  `json:"time"` // 发送时间
	MetaEventType string                 `json:"meta_event_type"`
	OperatorID    sonic.NoCopyRawMessage `json:"operator_id"`  // 操作者帐号
	GroupID       sonic.NoCopyRawMessage `json:"group_id"`     // 群号
	PostType      string                 `json:"post_type"`    // 上报类型，如group、notice
	RequestType   string                 `json:"request_type"` // 请求类型，如group
	SubType       string                 `json:"sub_type"`     // 子类型，如add invite
	Flag          string                 `json:"flag"`         // 请求 flag, 在调用处理请求的 API 时需要传入
	NoticeType    string                 `json:"notice_type"`
	UserID        sonic.NoCopyRawMessage `json:"user_id"`
	SelfID        sonic.NoCopyRawMessage `json:"self_id"`
	Duration      int64                  `json:"duration"`
	Comment       string                 `json:"comment"`
	TargetID      sonic.NoCopyRawMessage `json:"target_id"`

	Data *struct {
		// 个人信息
		Nickname string                 `json:"nickname"`
		UserID   sonic.NoCopyRawMessage `json:"user_id"`

		// 群信息
		GroupID         sonic.NoCopyRawMessage `json:"group_id"`          // 群号
		GroupCreateTime uint32                 `json:"group_create_time"` // 群号
		MemberCount     int64                  `json:"member_count"`
		GroupName       string                 `json:"group_name"`
		MaxMemberCount  int32                  `json:"max_member_count"`

		// 群成员信息
		Card string `json:"card"`
	} `json:"data"`
	Retcode int64 `json:"retcode"`
	// Status string `json:"status"`
	Echo sonic.NoCopyRawMessage `json:"echo"` // 声明类型而不是interface的原因是interface下数字不能正确转换

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
		default:
			// 转换为CQ码
			var params []string
			dMap := dataObj.Map()
			for paramStr, paramValue := range dMap {
				params = append(params, fmt.Sprintf("%s=%s", paramStr, paramValue))
			}
			cqMessage.WriteString(fmt.Sprintf("[CQ:%s,%s]", typeStr, strings.Join(params, ",")))
			// 生成对应的DefaultElement
			seg = append(seg, &message.DefaultElement{
				RawType: typeStr,
				Data:    sonic.NoCopyRawMessage(dataObj.String()),
			})
		}
	}
	// 获取Message
	m.Message = cqMessage.String()
	// 获取Segment
	m.Segment = seg
	return m, nil
}

// 将OB11的Array数据转换为string字符串 确实没在使用，但备份一下这个实用函数
// func array2string(parseContent gjson.Result) (gjson.Result, error) {
//	arrayContent := parseContent.Get("message").Array()
//	cqMessage := strings.Builder{}
//
//	for _, i := range arrayContent {
//		typeStr := i.Get("type").String()
//		dataObj := i.Get("data")
//		switch typeStr {
//		case "text":
//			cqMessage.WriteString(dataObj.Get("text").String())
//		case "image":
//			// 兼容NC情况, 此时file字段只有文件名, 完整URL在url字段
//			if !hasURLScheme(dataObj.Get("file").String()) && hasURLScheme(dataObj.Get("url").String()) {
//				cqMessage.WriteString(fmt.Sprintf("[CQ:image,file=%v]", dataObj.Get("url").String()))
//			} else {
//				cqMessage.WriteString(fmt.Sprintf("[CQ:image,file=%v]", dataObj.Get("file").String()))
//			}
//		case "face":
//			// 兼容四叶草，移除 .(string)。自动获取的信息表示此类型为 float64，这是go解析的问题
//			cqMessage.WriteString(fmt.Sprintf("[CQ:face,id=%v]", dataObj.Get("id").String()))
//		case "record":
//			// 兼容NC情况, 此时file字段只有文件名, 完整路径在path字段
//			if !hasURLScheme(dataObj.Get("file").String()) && dataObj.Get("path").String() != "" {
//				cqMessage.WriteString(fmt.Sprintf("[CQ:record,file=%v]", dataObj.Get("path").String()))
//			} else {
//				cqMessage.WriteString(fmt.Sprintf("[CQ:record,file=%v]", dataObj.Get("file").String()))
//			}
//		case "at":
//			cqMessage.WriteString(fmt.Sprintf("[CQ:at,qq=%v]", dataObj.Get("qq").String()))
//		case "poke":
//			cqMessage.WriteString(fmt.Sprintf("[CQ:poke,qq=%v]", dataObj.Get("qq").String()))
//		case "reply":
//			cqMessage.WriteString(fmt.Sprintf("[CQ:reply,id=%v]", dataObj.Get("id").String()))
//		default:
//			var cqParam string
//			dMap := dataObj.Map()
//			for paramStr, paramValue := range dMap {
//				cqParam += fmt.Sprintf("%s=%s", paramStr, paramValue)
//			}
//			cqMessage.WriteString(fmt.Sprintf("[CQ:%s,%s]", typeStr, cqParam))
//		}
//	}
//	// 赋值对应的Message
//	tempStr, err := sjson.Set(parseContent.String(), "message", cqMessage.String())
//	if err != nil {
//		return gjson.Result{}, err
//	}
//	return gjson.Parse(tempStr), nil
// }

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
			fileVal := res.File
			if fileVal == "" {
				fileVal = res.URL
			}
			if fileVal == "" {
				continue
			}
			rawMsg = rawMsg.File(fileVal)
			cqMessage.WriteString(fmt.Sprintf("[CQ:file,file=%v]", fileVal))
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
			var recordFile string
			if res.File != nil {
				recordFile = res.File.URL
				if recordFile == "" {
					recordFile = res.File.File
				}
			}
			if recordFile == "" {
				continue
			}
			rawMsg = rawMsg.Record(recordFile)
			cqMessage.WriteString(fmt.Sprintf("[CQ:record,file=%v]", recordFile))
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
		default:
			res, ok := v.(*message.DefaultElement)
			if !ok {
				continue
			}
			rawMsg = rawMsg.Append(schema.Message{
				Type: res.RawType,
				Data: res.Data,
			})
			// 将其转换为CQ码
			var params []string
			dMap := gjson.ParseBytes(res.Data).Map()
			for paramStr, paramValue := range dMap {
				params = append(params, fmt.Sprintf("%s=%s", paramStr, paramValue))
			}
			cqMessage.WriteString(fmt.Sprintf("[CQ:%s,%s]", res.RawType, strings.Join(params, ",")))
		}
	}
	messageStr := cqMessage.String()
	return rawMsg, messageStr
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

func (p *PlatformAdapterOnebot) makeCtx(req gjson.Result) *MsgContext {
	ep := p.EndPoint
	session := p.Session
	var messageType = "private"
	if req.Get("group_id").Exists() {
		messageType = "group"
	}
	ctx := &MsgContext{MessageType: messageType, EndPoint: ep, Session: session, Dice: session.Parent}
	wrapper := MessageWrapper{
		MessageType: ctx.MessageType,
		GroupID:     FormatOnebotDiceIDQQGroup(req.Get("group_id").String()),
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
		// 私聊戳一戳可能拿不到用户信息（协议端异常/限流等），退化为仅依赖 user_id 的上下文。
		wrapper.Sender.UserID = FormatOnebotDiceIDQQ(req.Get("user_id").String())
		info, err := p.sendEmitter.GetStrangerInfo(p.ctx, req.Get("user_id").Int(), false)
		if err == nil {
			wrapper.Sender.UserID = FormatOnebotDiceIDQQ(strconv.FormatInt(info.UserId, 10))
			wrapper.Sender.Nickname = info.NickName
		}
		ctx.Group, ctx.Player = GetPlayerInfoBySenderRaw(ctx, &wrapper)
		if ctx.Player.Name == "" {
			if wrapper.Sender.Nickname != "" {
				ctx.Player.Name = wrapper.Sender.Nickname
			} else {
				ctx.Player.Name = wrapper.Sender.UserID
			}
			ctx.Player.UpdatedAtTime = time.Now().Unix()
			if ctx.Group != nil {
				ctx.Group.MarkDirty(ctx.Dice)
			}
		}
		if wrapper.Sender.Nickname != "" {
			SetTempVars(ctx, wrapper.Sender.Nickname)
		}
	case "group":
		groupID, _ := strconv.ParseInt(req.Get("group_id").String(), 10, 64)
		userID, _ := strconv.ParseInt(req.Get("user_id").String(), 10, 64)
		memberInfo, err := p.sendEmitter.GetGroupMemberInfo(p.ctx, groupID, userID, false)
		// 群戳一戳事件中，获取群成员信息可能失败（协议端异常/限流/机器人不在群等）。
		// 这种情况下仍构造最小上下文，避免后续处理链路空指针崩溃。
		wrapper.Sender.UserID = FormatOnebotDiceIDQQ(req.Get("user_id").String())
		if err == nil {
			wrapper.Sender.UserID = FormatOnebotDiceIDQQ(strconv.FormatInt(memberInfo.UserId, 10))
			wrapper.Sender.Nickname = memberInfo.Nickname
		}
		ctx.Group, ctx.Player = GetPlayerInfoBySenderRaw(ctx, &wrapper)
		if ctx.Group == nil {
			// 注意：GetPlayerInfoBySenderRaw 内部已调用 SetBotOnAtGroup，正常不会返回 nil
			// 若仍为 nil，说明出现异常情况，此处使用 SetBotOnAtGroup 确保群组被正确存入全局列表
			gi := p.GetGroupCacheInfo(FormatOnebotDiceIDQQGroup(req.Get("group_id").String()))
			ctx.Group = SetBotOnAtGroup(ctx, gi.GroupId)
			ctx.Group.GroupName = gi.GroupName
			ctx.Group.MarkDirty(ctx.Dice)
		}
		if ctx.Player == nil {
			ctx.Player = &GroupPlayerInfo{UserID: wrapper.Sender.UserID}
		}
		if ctx.Player.Name == "" {
			if err == nil {
				if memberInfo.Card == "" {
					ctx.Player.Name = memberInfo.Nickname
				} else {
					ctx.Player.Name = memberInfo.Card
				}
			} else {
				ctx.Player.Name = wrapper.Sender.UserID
			}
			ctx.Player.UpdatedAtTime = time.Now().Unix()
			if ctx.Group != nil {
				ctx.Group.MarkDirty(ctx.Dice)
			}
		}
		if wrapper.Sender.Nickname != "" {
			SetTempVars(ctx, wrapper.Sender.Nickname)
		}
	}

	return ctx
}
