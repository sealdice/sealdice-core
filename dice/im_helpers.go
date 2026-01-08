package dice

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"

	ds "github.com/sealdice/dicescript"

	"sealdice-core/logger"
	"sealdice-core/utils/panicHandler"
)

var (
	sealCodeRe = regexp.MustCompile(`\[(img|图|文本|text|语音|voice|视频|video):(.+?)]`)
	cqCodeRe   = regexp.MustCompile(`\[CQ:.+?]`)
)

type forwardMsgSender interface {
	SendGroupForwardMsg(ctx *MsgContext, groupID string, nodes []forwardNode) bool
	SendPrivateForwardMsg(ctx *MsgContext, userID string, nodes []forwardNode) bool
}

type forwardNodeData struct {
	Name    string `json:"name"`
	Uin     string `json:"uin"`
	Content string `json:"content"`
}

type forwardNode struct {
	Type string          `json:"type"`
	Data forwardNodeData `json:"data"`
}

func buildForwardNodes(senderName string, senderUin string, title string, contents []string) []forwardNode {
	nodes := make([]forwardNode, 0, len(contents)+1)
	if title != "" {
		nodes = append(nodes, forwardNode{
			Type: "node",
			Data: forwardNodeData{
				Name:    senderName,
				Uin:     senderUin,
				Content: title,
			},
		})
	}

	for _, c := range contents {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}

		nodes = append(nodes, forwardNode{
			Type: "node",
			Data: forwardNodeData{
				Name:    senderName,
				Uin:     senderUin,
				Content: c,
			},
		})
	}
	return nodes
}

func BuildForwardNodesFromContext(ctx *MsgContext, title string, contents []string) []forwardNode {
	if ctx == nil || ctx.EndPoint == nil || ctx.EndPoint.Adapter == nil {
		return nil
	}
	name := ctx.EndPoint.Nickname
	if diceName := strings.TrimSpace(DiceFormatTmpl(ctx, "核心:骰子名字")); diceName != "" {
		name = diceName
	}

	var uin string
	switch a := ctx.EndPoint.Adapter.(type) {
	case *PlatformAdapterGocq:
		botID, _ := a.mustExtractID(a.EndPoint.UserID)
		if botID != 0 {
			uin = strconv.FormatInt(botID, 10)
		}
	case *PlatformAdapterOnebot:
		botID := ExtractQQEmitterUserID(ctx.EndPoint.UserID)
		if botID != 0 {
			uin = strconv.FormatInt(botID, 10)
		}
	default:
		trimmed := strings.TrimPrefix(ctx.EndPoint.UserID, "PG-")
		trimmed = strings.TrimPrefix(trimmed, "QQ-Group:")
		trimmed = strings.TrimPrefix(trimmed, "QQ-CH-Group:")
		trimmed = strings.TrimPrefix(trimmed, "QQ-CH:")
		trimmed = strings.TrimPrefix(trimmed, "QQ:")
		if trimmed != ctx.EndPoint.UserID {
			uin = trimmed
		}
	}
	if uin == "" {
		return nil
	}
	return buildForwardNodes(name, uin, title, contents)
}

func forwardNodesToText(nodes []forwardNode) string {
	parts := make([]string, 0, len(nodes))
	for _, n := range nodes {
		text := strings.TrimSpace(n.Data.Content)
		if text != "" {
			parts = append(parts, text)
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

// TryReplyToSenderMergedForward 尝试用“合并转发”发送多条内容。
//
// 返回值含义：
//   - true：已发送（或当前平台支持且已处理）
//   - false：当前平台/适配器不支持，调用方应自行回退到普通 ReplyToSender
func TryReplyToSenderMergedForward(ctx *MsgContext, msg *Message, title string, contents []string) bool {
	if ctx == nil || msg == nil || ctx.Dice == nil || ctx.EndPoint == nil || ctx.EndPoint.Adapter == nil {
		return false
	}
	if len(contents) == 0 {
		return false
	}

	// 避免绕过“仅输出回复”的敏感词拦截逻辑：此模式下回退到普通 ReplyToSender 流程
	if ctx.Dice.Config.EnableCensor && ctx.Dice.Config.CensorMode == OnlyOutputReply {
		return false
	}

	if ctx.Dice.Config.RateLimitEnabled && msg.Platform == "QQ" {
		if msg.MessageType == "group" {
			if !spamCheckPerson(ctx, msg) {
				spamCheckGroup(ctx, msg)
			}
		} else {
			spamCheckPerson(ctx, msg)
		}
	}

	if ctx.AliasPrefixText != "" {
		title = ctx.AliasPrefixText + title
		ctx.AliasPrefixText = ""
	}
	if ctx.DelegateText != "" {
		title = ctx.DelegateText + title
		ctx.DelegateText = ""
	}

	s, ok := ctx.EndPoint.Adapter.(forwardMsgSender)
	if !ok {
		return false
	}

	nodes := BuildForwardNodesFromContext(ctx, title, contents)
	if len(nodes) == 0 {
		return false
	}
	switch msg.MessageType {
	case "group":
		ok := s.SendGroupForwardMsg(ctx, msg.GroupID, nodes)
		if ok && ctx.Group != nil {
			ctx.Group.RecentDiceSendTime = time.Now().Unix()
			ctx.Group.MarkDirty(ctx.Dice)
		}
		return ok
	case "private":
		return s.SendPrivateForwardMsg(ctx, msg.Sender.UserID, nodes)
	default:
		return false
	}
}

func IsCurGroupBotOnByID(session *IMSession, ep *EndPointInfo, messageType string, groupID string) bool {
	// Pinenutn: 总觉得这里还能优化，但是又想不到怎么优化，可恶，要长脑子了
	a := messageType == "group" && session.ServiceAtNew.Exists(groupID)
	if !a {
		return false
	}
	groupInfo, ok := session.ServiceAtNew.Load(groupID)
	if !ok {
		// Pinenutn: 这里是否要打一下日志呢……
		return false
	}
	_, exists := groupInfo.DiceIDActiveMap.Load(ep.UserID)
	return exists
}

func SetBotOffAtGroup(ctx *MsgContext, groupID string) {
	session := ctx.Session
	groupInfo, ok := session.ServiceAtNew.Load(groupID)
	if ok {
		if groupInfo.DiceIDActiveMap == nil {
			groupInfo.DiceIDActiveMap = new(SyncMap[string, bool])
		}

		// TODO: 进行更好的是否变更的检查
		groupInfo.DiceIDActiveMap.Delete(ctx.EndPoint.UserID)
		if groupInfo.DiceIDActiveMap.Len() == 0 {
			groupInfo.Active = false
		}
		groupInfo.MarkDirty(ctx.Dice)
	}
}

// SetBotOnAtGroup 在群内开启
func SetBotOnAtGroup(ctx *MsgContext, groupID string) *GroupInfo {
	session := ctx.Session
	group, ok := session.ServiceAtNew.Load(groupID)
	if ok {
		if group.DiceIDActiveMap == nil {
			group.DiceIDActiveMap = new(SyncMap[string, bool])
		}
		if group.DiceIDExistsMap == nil {
			group.DiceIDExistsMap = new(SyncMap[string, bool])
		}
		if group.InactivatedExtSet == nil {
			group.InactivatedExtSet = StringSet{}
		}
		group.DiceIDActiveMap.Store(ctx.EndPoint.UserID, true)
		group.Active = true
	} else {
		// 设定扩展情况
		sort.Sort(ExtDefaultSettingItemSlice(session.Parent.Config.ExtDefaultSettings))
		var extLst []*ExtInfo
		for _, i := range session.Parent.Config.ExtDefaultSettings {
			if i.ExtItem != nil {
				if i.AutoActive {
					extLst = append(extLst, i.ExtItem)
				}
			}
		}

		session.ServiceAtNew.Store(groupID, &GroupInfo{
			Active:            true,
			activatedExtList:  extLst,
			ExtAppliedTime:    session.Parent.ExtUpdateTime, // 标记已初始化
			InactivatedExtSet: StringSet{},
			Players:           new(SyncMap[string, *GroupPlayerInfo]),
			GroupID:           groupID,
			DiceIDActiveMap:   new(SyncMap[string, bool]),
			DiceIDExistsMap:   new(SyncMap[string, bool]),
			CocRuleIndex:      int(session.Parent.Config.DefaultCocRuleIndex),
			UpdatedAtTime:     time.Now().Unix(),
		})
		// TODO: Pinenutn:总觉得这里不太对，但是又觉得合理,GPT也没说怎么改更好一些，求教
		group, _ = session.ServiceAtNew.Load(groupID)
	}

	if group.DiceIDActiveMap == nil {
		group.DiceIDActiveMap = new(SyncMap[string, bool])
	}
	if group.DiceIDExistsMap == nil {
		group.DiceIDExistsMap = new(SyncMap[string, bool])
	}
	if group.BotList == nil {
		group.BotList = new(SyncMap[string, bool])
	}

	group.DiceIDActiveMap.Store(ctx.EndPoint.UserID, true)
	group.MarkDirty(ctx.Dice)
	return group
}

// GetPlayerInfoBySender 获取玩家群内信息，没有就创建
func GetPlayerInfoBySender(ctx *MsgContext, msg *Message) (*GroupInfo, *GroupPlayerInfo) {
	wrapper := MessageWrapper{
		MessageType: msg.MessageType,
		GroupID:     msg.GroupID,
		Sender: struct {
			UserID   string
			Nickname string
		}{
			UserID:   msg.Sender.UserID,
			Nickname: msg.Sender.Nickname,
		},
		GuildID:   msg.GuildID,
		ChannelID: msg.ChannelID,
	}
	return GetPlayerInfoBySenderRaw(ctx, &wrapper)
}

// GetPlayerInfoBySenderRaw 获取玩家群内信息的轻量版，不依赖完整的msg信息，因为实质上，大部分数据并没什么卵用。
func GetPlayerInfoBySenderRaw(ctx *MsgContext, msg *MessageWrapper) (*GroupInfo, *GroupPlayerInfo) {
	session := ctx.Session
	var groupID string
	if msg.MessageType == "group" {
		// 群信息
		groupID = msg.GroupID
	} else {
		// 私聊信息 PrivateGroup
		groupID = "PG-" + msg.Sender.UserID
		SetBotOnAtGroup(ctx, groupID)
	}

	// Pinenutn:ServiceAtNew
	groupInfo, ok := session.ServiceAtNew.Load(groupID)
	if !ok {
		groupInfo = SetBotOnAtGroup(ctx, groupID)
	}
	if msg.GuildID != "" {
		groupInfo.GuildID = msg.GuildID
	}
	if msg.ChannelID != "" {
		groupInfo.ChannelID = msg.ChannelID
	}

	if ctx.Dice != nil {
		groupInfo.SyncWrapperStatus(ctx.Dice)       // 移除无效 wrapper
		groupInfo.SyncExtensionsOnMessage(ctx.Dice) // 新增 AutoActive 扩展
	}

	p := groupInfo.PlayerGet(ctx.Dice.DBOperator, msg.Sender.UserID)
	if p == nil {
		p = &GroupPlayerInfo{
			Name:          msg.Sender.Nickname,
			UserID:        msg.Sender.UserID,
			ValueMapTemp:  &ds.ValueMap{},
			UpdatedAtTime: 0, // 新创建时不赋值，这样不会入库保存，减轻数据库负担
		}
		groupInfo.Players.Store(msg.Sender.UserID, p)
	}
	if p.ValueMapTemp == nil {
		p.ValueMapTemp = &ds.ValueMap{}
	}
	p.InGroup = true
	return groupInfo, p
}

type MessageWrapper struct {
	MessageType string // "group" 或私聊
	GroupID     string
	Sender      struct {
		UserID   string
		Nickname string
	}
	GuildID   string // 可选
	ChannelID string // 可选
}

func ReplyToSenderRaw(ctx *MsgContext, msg *Message, text string, flag string) {
	inGroup := msg.MessageType == "group"
	if inGroup {
		ReplyGroupRaw(ctx, msg, text, flag)
	} else {
		ReplyPersonRaw(ctx, msg, text, flag)
	}
}

func replyToSenderRawNoCheck(ctx *MsgContext, msg *Message, text string, flag string) {
	inGroup := msg.MessageType == "group"
	if inGroup {
		replyGroupRawNoCheck(ctx, msg, text, flag)
	} else {
		replyPersonRawNoCheck(ctx, msg, text, flag)
	}
}
func ReplyToSender(ctx *MsgContext, msg *Message, text string) {
	if ctx == nil || msg == nil || ctx.Dice == nil {
		logger.M().Errorf("ReplyToSender 被调用，但没有正确传递参数！请检查您的参数！: ctx=%v, msg=%v", ctx, msg)
		return
	}
	panicHandler.Once(logger.M(), func() {
		ReplyToSenderRaw(ctx, msg, text, "")
	})
}

func ReplyToSenderNoCheck(ctx *MsgContext, msg *Message, text string) {
	if ctx == nil || msg == nil || ctx.Dice == nil {
		logger.M().Errorf("ReplyToSenderNoCheck 被调用，但没有正确传递参数！请检查您的参数！: ctx=%v, msg=%v", ctx, msg)
		return
	}
	panicHandler.Once(logger.M(), func() {
		replyToSenderRawNoCheck(ctx, msg, text, "")
	})
}

func ReplyGroupRaw(ctx *MsgContext, msg *Message, text string, flag string) {
	if ctx.AliasPrefixText != "" {
		text = ctx.AliasPrefixText + text
		ctx.AliasPrefixText = ""
	}
	if ctx.DelegateText != "" {
		text = ctx.DelegateText + text
		ctx.DelegateText = ""
	}

	if ctx.Dice.Config.RateLimitEnabled && msg.Platform == "QQ" {
		if !spamCheckPerson(ctx, msg) {
			spamCheckGroup(ctx, msg)
		}
	}

	d := ctx.Dice
	if d != nil {
		d.Logger.Infof("发给(群%s): %s", msg.GroupID, text)
		// 敏感词拦截：回复（群）
		if d.Config.EnableCensor && d.Config.CensorMode == OnlyOutputReply {
			// 先拿掉海豹码和CQ码再检查敏感词
			checkText := sealCodeRe.ReplaceAllString(text, "")
			checkText = cqCodeRe.ReplaceAllString(checkText, "")

			hit, words, needToTerminate, _ := d.CensorMsg(ctx, msg, checkText, text)
			if needToTerminate {
				return
			}
			if hit {
				d.Logger.Infof(
					"拒绝回复命中敏感词「%s」的内容「%s」，原消息「%s」- 来自群(%s)内<%s>(%s)",
					strings.Join(words, "|"),
					text, msg.Message,
					msg.GroupID,
					msg.Sender.Nickname,
					msg.Sender.UserID,
				)
				text = DiceFormatTmpl(ctx, "核心:拦截_完全拦截_发出的消息")
			}
		}
	}
	replyGroupRawNoCheck(ctx, msg, text, flag)
}

func replyGroupRawNoCheck(ctx *MsgContext, msg *Message, text string, flag string) {
	if ctx.AliasPrefixText != "" {
		text = ctx.AliasPrefixText + text
		ctx.AliasPrefixText = ""
	}
	if ctx.DelegateText != "" {
		text = ctx.DelegateText + text
		ctx.DelegateText = ""
	}
	if lenWithoutBase64(text) > 15000 {
		text = "要发送的文本过长"
	}
	if ctx.Group != nil {
		ctx.Group.RecentDiceSendTime = time.Now().Unix()
		ctx.Group.MarkDirty(ctx.Dice)
	}
	text = strings.TrimSpace(text)
	for _, i := range ctx.SplitText(text) {
		if ctx.EndPoint != nil && ctx.EndPoint.Platform == "QQ" {
			doSleepQQ(ctx)
		}
		ctx.EndPoint.Adapter.SendToGroup(ctx, msg.GroupID, strings.TrimSpace(i), flag)
	}
}

func ReplyGroup(ctx *MsgContext, msg *Message, text string) {
	ReplyGroupRaw(ctx, msg, text, "")
}

func ReplyPersonRaw(ctx *MsgContext, msg *Message, text string, flag string) {
	if ctx.AliasPrefixText != "" {
		text = ctx.AliasPrefixText + text
		ctx.AliasPrefixText = ""
	}
	if ctx.DelegateText != "" {
		text = ctx.DelegateText + text
		ctx.DelegateText = ""
	}

	if ctx.Dice.Config.RateLimitEnabled && msg.Platform == "QQ" {
		spamCheckPerson(ctx, msg)
	}

	d := ctx.Dice
	if d != nil {
		d.Logger.Infof("发给(帐号%s): %s", msg.Sender.UserID, text)
		// 敏感词拦截：回复（个人）
		if d.Config.EnableCensor && d.Config.CensorMode == OnlyOutputReply {
			// 先拿掉海豹码和CQ码再检查敏感词
			checkText := sealCodeRe.ReplaceAllString(text, "")
			checkText = cqCodeRe.ReplaceAllString(checkText, "")

			hit, words, needToTerminate, _ := d.CensorMsg(ctx, msg, checkText, text)
			if needToTerminate {
				return
			}
			if hit {
				d.Logger.Infof("拒绝回复命中敏感词「%s」的内容「%s」，原消息「%s」- 来自<%s>(%s)",
					strings.Join(words, "|"),
					text,
					msg.Message,
					msg.Sender.Nickname,
					msg.Sender.UserID,
				)
				text = DiceFormatTmpl(ctx, "核心:拦截_完全拦截_发出的消息")
			}
		}
	}
	replyPersonRawNoCheck(ctx, msg, text, flag)
}

func replyPersonRawNoCheck(ctx *MsgContext, msg *Message, text string, flag string) {
	if ctx.AliasPrefixText != "" {
		text = ctx.AliasPrefixText + text
		ctx.AliasPrefixText = ""
	}
	if ctx.DelegateText != "" {
		text = ctx.DelegateText + text
		ctx.DelegateText = ""
	}
	if lenWithoutBase64(text) > 15000 {
		text = "要发送的文本过长"
	}
	text = strings.TrimSpace(text)
	for _, i := range ctx.SplitText(text) {
		if ctx.EndPoint != nil && ctx.EndPoint.Platform == "QQ" {
			doSleepQQ(ctx)
		}
		ctx.EndPoint.Adapter.SendToPerson(ctx, msg.Sender.UserID, strings.TrimSpace(i), flag)
	}
}

// CrossMsgBySearch
// 在 se 中找到第一个平台等于 p 且启用的 EndPointInfo, 并向目标 t 发送消息,
// pr 判断是否为私聊消息
func CrossMsgBySearch(se *IMSession, p, t, txt string, pr bool) bool {
	ep := se.GetEpByPlatform(p)
	if ep == nil {
		return false
	}
	mctx := &MsgContext{
		EndPoint: ep,
		Session:  ep.Session,
		Dice:     ep.Session.Parent,
	}

	if groupInfo, ok := mctx.Session.ServiceAtNew.Load(t); ok {
		mctx.IsCurGroupBotOn = groupInfo.Active
		mctx.Group = groupInfo
	}

	if !pr {
		mctx.MessageType = "group"
		ReplyGroup(mctx, &Message{GroupID: t}, txt)
	} else {
		mctx.IsPrivate = true
		mctx.MessageType = "private"
		ReplyPerson(mctx, &Message{Sender: SenderBase{UserID: t}}, txt)
	}

	return true
}

// TODO: CrossMsgById 用指定 Id 的 EndPoint 发送跨平台消息，现在似乎没有这个需求

func ReplyPerson(ctx *MsgContext, msg *Message, text string) {
	ReplyPersonRaw(ctx, msg, text, "")
}

func SendFileToSenderRaw(ctx *MsgContext, msg *Message, path string, flag string) {
	inGroup := msg.MessageType == "group"
	if inGroup {
		SendFileToGroupRaw(ctx, msg, path, flag)
	} else {
		SendFileToPersonRaw(ctx, msg, path, flag)
	}
}

func SendFileToPersonRaw(ctx *MsgContext, msg *Message, path string, flag string) {
	if ctx.Dice != nil {
		ctx.Dice.Logger.Infof("发文件给(账号%s): %s", msg.Sender.UserID, path)
	}
	ctx.EndPoint.Adapter.SendFileToPerson(ctx, msg.Sender.UserID, path, flag)
}

func SendFileToGroupRaw(ctx *MsgContext, msg *Message, path string, flag string) {
	if ctx.Dice != nil {
		ctx.Dice.Logger.Infof("发文件给(群%s): %s", msg.GroupID, path)
	}
	ctx.EndPoint.Adapter.SendFileToGroup(ctx, msg.GroupID, path, flag)
}

func MemberBan(ctx *MsgContext, groupID string, userID string, duration int64) {
	ctx.EndPoint.Adapter.MemberBan(groupID, userID, duration)
}

func MemberKick(ctx *MsgContext, groupID string, userID string) {
	ctx.EndPoint.Adapter.MemberKick(groupID, userID)
}

type ByLength []string

func (s ByLength) Len() int {
	return len(s)
}

func (s ByLength) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s ByLength) Less(i, j int) bool {
	return len(s[i]) > len(s[j])
}

func CompatibleReplace(ctx *MsgContext, s string) string {
	s = ctx.TranslateSplit(s)

	// 匹配 #{DRAW-$1}, 其中$1执行最短匹配且允许左侧右侧各有一个花括号
	// #{DRAW-aaa} => aaa
	// #{DRAW-{aaa} => {aaa
	// #{DRAW-aaa}} => aaa}
	// #{DRAW-{aaa}} => {aaa}
	// 这允许在牌组名中使用不含空格的表达式(主要是为了变量)
	re := regexp.MustCompile(`#\{DRAW-(\{?\S+?\}?)\}`)
	s = re.ReplaceAllString(s, "###DRAW-$1###")

	if ctx != nil {
		s = DeckRewrite(s, func(deckName string) string {
			// 如果牌组名中含有表达式, 在此进行求值
			// 不含表达式也无妨, 求值完还是原来的字符串
			r, _, err := DiceExprTextBase(ctx, deckName, RollExtraFlags{})
			if err == nil {
				deckName = r.ToString()
			}

			exists, result, err := deckDraw(ctx, deckName, false)
			if !exists {
				return "<%未知牌组-" + deckName + "%>"
			}
			if err != nil {
				return "<%抽取错误-" + deckName + "%>"
			}
			return result
		})
	}
	return s
}

func FormatDiceID(ctx *MsgContext, id interface{}, isGroup bool) string {
	prefix := ctx.EndPoint.Platform
	if isGroup {
		prefix += "-Group"
	}
	return fmt.Sprintf("%s:%v", prefix, id)
}

func spamCheckPerson(ctx *MsgContext, msg *Message) bool {
	if ctx.SpamCheckedPerson {
		return false
	}

	// 同一个 ctx 只需检查一次
	defer func() {
		ctx.SpamCheckedPerson = true
	}()

	if ctx.PrivilegeLevel >= 100 {
		return false
	}

	if ctx.Player.RateLimiter == nil {
		ctx.Player.RateLimitWarned = false
		if ctx.Dice.Config.PersonalReplenishRateStr == "" {
			ctx.Dice.Config.PersonalReplenishRateStr = DefaultConfig.PersonalReplenishRateStr
			ctx.Dice.Config.PersonalReplenishRate = DefaultConfig.PersonalReplenishRate
		}
		if ctx.Dice.Config.PersonalBurst == 0 {
			ctx.Dice.Config.PersonalBurst = DefaultConfig.PersonalBurst
		}
		ctx.Player.RateLimiter = rate.NewLimiter(
			ctx.Dice.Config.PersonalReplenishRate,
			int(ctx.Dice.Config.PersonalBurst),
		)
	}

	if ctx.Player.RateLimiter.Allow() {
		ctx.Player.RateLimitWarned = false
		return false
	}

	// Check if user is already banned to avoid sending multiple warnings in concurrent scenarios
	if banItem, exists := ctx.Dice.Config.BanList.GetByID(ctx.Player.UserID); exists && banItem.Rank == BanRankBanned {
		return true
	}

	if ctx.Player.RateLimitWarned {
		ctx.Dice.Config.BanList.AddScoreByCommandSpam(ctx.Player.UserID, msg.GroupID, ctx)
	} else {
		ctx.Player.RateLimitWarned = true
		replyToSenderRawNoCheck(
			ctx, msg,
			DiceFormatTmpl(ctx, "核心:刷屏_警告内容_个人"),
			"",
		)
	}

	return true
}

func spamCheckGroup(ctx *MsgContext, msg *Message) bool {
	if ctx.SpamCheckedGroup {
		return false
	}

	// 同一个 ctx 只需检查一次
	defer func() {
		ctx.SpamCheckedGroup = true
	}()

	// Skip privileged groups
	for _, g := range ctx.Dice.DiceMasters {
		if ctx.Group.GroupID == g {
			return false
		}
	}

	if ctx.Group.RateLimiter == nil {
		ctx.Group.RateLimitWarned = false
		if ctx.Dice.Config.GroupReplenishRateStr == "" {
			ctx.Dice.Config.GroupReplenishRateStr = DefaultConfig.GroupReplenishRateStr
			ctx.Dice.Config.GroupReplenishRate = DefaultConfig.GroupReplenishRate
		}
		if ctx.Dice.Config.GroupBurst == 0 {
			ctx.Dice.Config.GroupBurst = DefaultConfig.GroupBurst
		}
		ctx.Group.RateLimiter = rate.NewLimiter(
			ctx.Dice.Config.GroupReplenishRate,
			int(ctx.Dice.Config.GroupBurst),
		)
	}

	if ctx.Group.RateLimiter.Allow() {
		ctx.Group.RateLimitWarned = false
		return false
	}

	// Check if group is already banned to avoid sending multiple warnings in concurrent scenarios
	if banItem, exists := ctx.Dice.Config.BanList.GetByID(ctx.Group.GroupID); exists && banItem.Rank == BanRankBanned {
		return true
	}

	// If not allow
	if ctx.Group.RateLimitWarned {
		ctx.Dice.Config.BanList.AddScoreByCommandSpam(ctx.Group.GroupID, msg.GroupID, ctx)
	} else {
		ctx.Group.RateLimitWarned = true
		replyToSenderRawNoCheck(
			ctx, msg,
			DiceFormatTmpl(ctx, "核心:刷屏_警告内容_群组"),
			"",
		)
	}

	return true
}

func lenWithoutBase64(text string) int {
	re := regexp.MustCompile(`base64://[A-Za-z0-9+/=]+`)
	croppedText := re.ReplaceAllString(text, "")
	return len(croppedText)
}
