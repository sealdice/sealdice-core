package dice

import (
	"fmt"
	"github.com/fy0/lockfree"
	"regexp"
	"sort"
	"strings"
)

func IsCurGroupBotOnById(session *IMSession, ep *EndPointInfo, messageType string, groupId string) bool {
	return messageType == "group" &&
		session.ServiceAtNew[groupId] != nil &&
		session.ServiceAtNew[groupId].ActiveDiceIds[ep.UserId]
}

func SetBotOffAtGroup(ctx *MsgContext, groupId string) {
	session := ctx.Session
	group := session.ServiceAtNew[groupId]
	if group != nil {
		if group.ActiveDiceIds == nil {
			group.ActiveDiceIds = map[string]bool{}
		}
		delete(group.ActiveDiceIds, ctx.EndPoint.UserId)
		if len(group.ActiveDiceIds) == 0 {
			group.Active = false
		}
	}
}

// SetBotOnAtGroup 在群内开启
func SetBotOnAtGroup(ctx *MsgContext, groupId string) *GroupInfo {
	session := ctx.Session
	group := session.ServiceAtNew[groupId]
	if group != nil {
		if group.ActiveDiceIds == nil {
			group.ActiveDiceIds = map[string]bool{}
		}
		group.ActiveDiceIds[ctx.EndPoint.UserId] = true
		group.Active = true
	} else {
		// 设定扩展情况
		sort.Sort(ExtDefaultSettingItemSlice(session.Parent.ExtDefaultSettings))
		extLst := []*ExtInfo{}
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
			Players:          map[string]*GroupPlayerInfo{},
			GroupId:          groupId,
			ValueMap:         lockfree.NewHashMap(),
			ActiveDiceIds:    map[string]bool{},
			CocRuleIndex:     int(session.Parent.DefaultCocRuleIndex),
		}
		group = session.ServiceAtNew[groupId]
	}

	if group.ActiveDiceIds == nil {
		group.ActiveDiceIds = map[string]bool{}
	}
	if group.BotList == nil {
		group.BotList = map[string]bool{}
	}

	group.ActiveDiceIds[ctx.EndPoint.UserId] = true
	group.NotInGroup = false
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
	if group == nil {
		return nil, nil
	}
	players := group.Players
	p := players[msg.Sender.UserId]
	if p == nil {
		p = &GroupPlayerInfo{
			GroupPlayerInfoBase{
				Name:         msg.Sender.Nickname,
				UserId:       msg.Sender.UserId,
				ValueMapTemp: lockfree.NewHashMap(),
			},
		}
		players[msg.Sender.UserId] = p
	}
	if p.ValueMapTemp == nil {
		p.ValueMapTemp = lockfree.NewHashMap()
	}
	if p.InGroup == false {
		p.InGroup = true
	}
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
	ReplyToSenderRaw(ctx, msg, text, "")
}

func ReplyGroupRaw(ctx *MsgContext, msg *Message, text string, flag string) {
	if len(text) > 15000 {
		text = "要发送的文本过长"
	}
	if ctx.Dice != nil {
		ctx.Dice.Logger.Infof("发给(群%s): %s", msg.GroupId, text)
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
	if len(text) > 15000 {
		text = "要发送的文本过长"
	}
	if ctx.Dice != nil {
		ctx.Dice.Logger.Infof("发给(帐号%s): %s", msg.Sender.UserId, text)
	}
	text = strings.TrimSpace(text)
	for _, i := range strings.Split(text, "###SPLIT###") {
		if ctx.EndPoint != nil && ctx.EndPoint.Platform == "QQ" {
			doSleepQQ(ctx)
		}
		ctx.EndPoint.Adapter.SendToPerson(ctx, msg.Sender.UserId, strings.TrimSpace(i), flag)
	}
}

func ReplyPerson(ctx *MsgContext, msg *Message, text string) {
	ReplyPersonRaw(ctx, msg, text, "")
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

	re := regexp.MustCompile(`#\{DRAW-(\S+?)\}`)
	s = re.ReplaceAllString(s, "###DRAW-$1###")

	if ctx != nil {
		s = DeckRewrite(s, func(deckName string) string {
			exists, result, err := deckDraw(ctx, deckName)
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
