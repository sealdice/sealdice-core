package dice

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/fy0/lockfree"
)

func IsCurGroupBotOnById(session *IMSession, ep *EndPointInfo, messageType string, groupId string) bool {
	a := messageType == "group" &&
		session.ServiceAtNew[groupId] != nil
	if !a {
		return false
	}
	_, exists := session.ServiceAtNew[groupId].DiceIdActiveMap.Load(ep.UserId)
	return exists
}

func SetBotOffAtGroup(ctx *MsgContext, groupId string) {
	session := ctx.Session
	group := session.ServiceAtNew[groupId]
	if group != nil {
		if group.DiceIdActiveMap == nil {
			group.DiceIdActiveMap = new(SyncMap[string, bool])
		}

		// TODO: 进行更好的是否变更的检查
		group.DiceIdActiveMap.Delete(ctx.EndPoint.UserId)
		if group.DiceIdActiveMap.Len() == 0 {
			group.Active = false
		}
		group.UpdatedAtTime = time.Now().Unix()
	}
}

// SetBotOnAtGroup 在群内开启
func SetBotOnAtGroup(ctx *MsgContext, groupId string) *GroupInfo {
	session := ctx.Session
	group := session.ServiceAtNew[groupId]
	if group != nil {
		if group.DiceIdActiveMap == nil {
			group.DiceIdActiveMap = new(SyncMap[string, bool])
		}
		if group.DiceIdExistsMap == nil {
			group.DiceIdActiveMap = new(SyncMap[string, bool])
		}
		group.DiceIdActiveMap.Store(ctx.EndPoint.UserId, true)
		group.Active = true
	} else {
		// 设定扩展情况
		sort.Sort(ExtDefaultSettingItemSlice(session.Parent.ExtDefaultSettings))
		var extLst []*ExtInfo
		for _, i := range session.Parent.ExtDefaultSettings {
			if i.ExtItem != nil {
				if i.AutoActive {
					extLst = append(extLst, i.ExtItem)
				}
			}
		}

		session.ServiceAtNew[groupId] = &GroupInfo{
			Active:           true,
			ActivatedExtList: extLst,
			Players:          new(SyncMap[string, *GroupPlayerInfo]),
			GroupId:          groupId,
			ValueMap:         lockfree.NewHashMap(),
			DiceIdActiveMap:  new(SyncMap[string, bool]),
			DiceIdExistsMap:  new(SyncMap[string, bool]),
			CocRuleIndex:     int(session.Parent.DefaultCocRuleIndex),
			UpdatedAtTime:    time.Now().Unix(),
		}
		group = session.ServiceAtNew[groupId]
	}

	if group.DiceIdActiveMap == nil {
		group.DiceIdActiveMap = new(SyncMap[string, bool])
	}
	if group.DiceIdExistsMap == nil {
		group.DiceIdExistsMap = new(SyncMap[string, bool])
	}
	if group.BotList == nil {
		group.BotList = new(SyncMap[string, bool])
	}

	group.DiceIdActiveMap.Store(ctx.EndPoint.UserId, true)
	group.UpdatedAtTime = time.Now().Unix()
	return group
}

// GetPlayerInfoBySender 获取玩家群内信息，没有就创建
func GetPlayerInfoBySender(ctx *MsgContext, msg *Message) (*GroupInfo, *GroupPlayerInfo) {
	session := ctx.Session
	var groupId string
	if msg.MessageType == "group" {
		// 群信息
		groupId = msg.GroupId
	} else {
		// 私聊信息 PrivateGroup
		groupId = "PG-" + msg.Sender.UserId
		SetBotOnAtGroup(ctx, groupId)
	}
	group := session.ServiceAtNew[groupId]
	if msg.GuildId != "" {
		group.GuildId = msg.GuildId
	}
	if group == nil {
		return nil, nil
	}

	p := group.PlayerGet(ctx.Dice.DBData, msg.Sender.UserId)
	if p == nil {
		p = &GroupPlayerInfo{
			Name:          msg.Sender.Nickname,
			UserId:        msg.Sender.UserId,
			ValueMapTemp:  lockfree.NewHashMap(),
			UpdatedAtTime: 0, // 新创建时不赋值，这样不会入库保存，减轻数据库负担
		}
		group.Players.Store(msg.Sender.UserId, p)
	}
	if p.ValueMapTemp == nil {
		p.ValueMapTemp = lockfree.NewHashMap()
	}
	p.InGroup = true
	ctx.LoadPlayerGroupVars(group, p)
	return group, p
}

func ReplyToSenderRaw(ctx *MsgContext, msg *Message, text string, flag string) {
	inGroup := msg.MessageType == "group"
	if inGroup {
		ReplyGroupRaw(ctx, msg, text, flag)
	} else {
		ReplyPersonRaw(ctx, msg, text, flag)
	}
}

func ReplyToSender(ctx *MsgContext, msg *Message, text string) {
	go ReplyToSenderRaw(ctx, msg, text, "")
}

func ReplyGroupRaw(ctx *MsgContext, msg *Message, text string, flag string) {
	if ctx.DelegateText != "" {
		text = ctx.DelegateText + text
		ctx.DelegateText = ""
	}

	// 敏感词拦截：回复（群）
	d := ctx.Dice
	if d.EnableCensor && d.CensorMode == OnlyReply {
		hit, needToTerminate, _ := d.CensorMsg(ctx, msg, text)
		if needToTerminate {
			return
		}
		if hit {
			d.Logger.Infof("拒绝回复命中敏感词的内容(%s): 来自群(%s)内<%s>(%s): %s", text, msg.GroupId, msg.Sender.Nickname, msg.Sender.UserId, msg.Message)
			text = DiceFormatTmpl(ctx, "核心:拦截_拦截提示_仅回复模式")
		}
	}

	if lenWithoutBase64(text) > 15000 {
		text = "要发送的文本过长"
	}
	if d != nil {
		d.Logger.Infof("发给(群%s): %s", msg.GroupId, text)
	}
	if ctx.Group != nil {
		now := time.Now().Unix()
		ctx.Group.RecentDiceSendTime = now
		ctx.Group.UpdatedAtTime = now
	}
	text = strings.TrimSpace(text)
	for _, i := range strings.Split(text, "###SPLIT###") {
		if ctx.EndPoint != nil && ctx.EndPoint.Platform == "QQ" {
			doSleepQQ(ctx)
		}
		ctx.EndPoint.Adapter.SendToGroup(ctx, msg.GroupId, strings.TrimSpace(i), flag)
	}
}

func ReplyGroup(ctx *MsgContext, msg *Message, text string) {
	ReplyGroupRaw(ctx, msg, text, "")
}

func ReplyPersonRaw(ctx *MsgContext, msg *Message, text string, flag string) {
	if ctx.DelegateText != "" {
		text = ctx.DelegateText + text
		ctx.DelegateText = ""
	}

	// 敏感词拦截：回复（个人）
	d := ctx.Dice
	if d.EnableCensor && d.CensorMode == OnlyReply {
		hit, needToTerminate, _ := d.CensorMsg(ctx, msg, text)
		if needToTerminate {
			return
		}
		if hit {
			d.Logger.Infof("拒绝回复命中敏感词的内容(%s): 来自<%s>(%s): %s", text, msg.Sender.Nickname, msg.Sender.UserId, msg.Message)
			text = DiceFormatTmpl(ctx, "核心:拦截_拦截提示_仅回复模式")
		}
	}

	if lenWithoutBase64(text) > 15000 {
		text = "要发送的文本过长"
	}
	if d != nil {
		d.Logger.Infof("发给(帐号%s): %s", msg.Sender.UserId, text)
	}
	text = strings.TrimSpace(text)
	for _, i := range strings.Split(text, "###SPLIT###") {
		if ctx.EndPoint != nil && ctx.EndPoint.Platform == "QQ" {
			doSleepQQ(ctx)
		}
		ctx.EndPoint.Adapter.SendToPerson(ctx, msg.Sender.UserId, strings.TrimSpace(i), flag)
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

	if g, ok := mctx.Session.ServiceAtNew[t]; ok {
		mctx.IsCurGroupBotOn = g.Active
		mctx.Group = g
	}

	if !pr {
		mctx.MessageType = "group"
		ReplyGroup(mctx, &Message{GroupId: t}, txt)
	} else {
		mctx.IsPrivate = true
		mctx.MessageType = "private"
		ReplyPerson(mctx, &Message{Sender: SenderBase{UserId: t}}, txt)
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
		ctx.Dice.Logger.Infof("发文件给(账号%s): %s", msg.Sender.UserId, path)
	}
	ctx.EndPoint.Adapter.SendFileToPerson(ctx, msg.Sender.UserId, path, flag)
}

func SendFileToGroupRaw(ctx *MsgContext, msg *Message, path string, flag string) {
	if ctx.Dice != nil {
		ctx.Dice.Logger.Infof("发文件给(群%s): %s", msg.GroupId, path)
	}
	ctx.EndPoint.Adapter.SendFileToGroup(ctx, msg.GroupId, path, flag)
}

func MemberBan(ctx *MsgContext, groupId string, userId string, duration int64) {
	ctx.EndPoint.Adapter.MemberBan(groupId, userId, duration)
}

func MemberKick(ctx *MsgContext, groupId string, userId string) {
	ctx.EndPoint.Adapter.MemberKick(groupId, userId)
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

func DiceFormatTmpl(ctx *MsgContext, s string) string {
	var text string
	a := ctx.Dice.TextMap[s]
	if a == nil {
		text = "<%未知项-" + s + "%>"
	} else {
		text = ctx.Dice.TextMap[s].Pick().(string)
	}
	return DiceFormat(ctx, text)
}

func CompatibleReplace(ctx *MsgContext, s string) string {
	s = strings.ReplaceAll(s, "#{SPLIT}", "###SPLIT###")
	s = strings.ReplaceAll(s, "{FormFeed}", "###SPLIT###")
	s = strings.ReplaceAll(s, "{formfeed}", "###SPLIT###")
	s = strings.ReplaceAll(s, "\f", "###SPLIT###")
	s = strings.ReplaceAll(s, "\\f", "###SPLIT###")

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
			deckName, _, _ = ctx.Dice.ExprText(deckName, ctx)

			exists, result, err := deckDraw(ctx, deckName, false)
			if exists {
				if err != nil {
					return "<%抽取错误-" + deckName + "%>"
				} else {
					return result
				}
			} else {
				return "<%未知牌组-" + deckName + "%>"
			}
		})
	}
	return s
}

func DiceFormat(ctx *MsgContext, s string) string {
	//s = strings.ReplaceAll(s, "\n", `\n`)
	//fmt.Println("???", s)
	s = CompatibleReplace(ctx, s)

	r, _, _ := ctx.Dice.ExprText(s, ctx)
	return r
}

func FormatDiceId(ctx *MsgContext, Id interface{}, isGroup bool) string {
	prefix := ctx.EndPoint.Platform
	if isGroup {
		prefix += "-Group"
	}
	return fmt.Sprintf("%s:%v", prefix, Id)
}

func lenWithoutBase64(text string) int {
	re := regexp.MustCompile(`base64://[A-Za-z0-9+/=]+`)
	croppedText := re.ReplaceAllString(text, "")
	return len(croppedText)
}
