package dice

func IsCurGroupBotOn(session *IMSession, msg *Message) bool {
	return msg.MessageType == "group" && session.ServiceAt[msg.GroupId] != nil && session.ServiceAt[msg.GroupId].Active
}

func IsCurGroupBotOnById(session *IMSession, messageType string, groupId int64) bool {
	return messageType == "group" && session.ServiceAt[groupId] != nil && session.ServiceAt[groupId].Active
}

/* 获取玩家群内信息，没有就创建 */
func GetPlayerInfoBySender(session *IMSession, msg *Message) *PlayerInfo {
	if msg.MessageType == "group" {
		g := session.ServiceAt[msg.GroupId]
		if g == nil {
			return nil
		}
		players := g.Players
		p := players[msg.Sender.UserId]
		if p == nil {
			p = &PlayerInfo{
				Name:   msg.Sender.Nickname,
				UserId: msg.Sender.UserId,
				//ValueNumMap:  map[string]int64{},
				//ValueStrMap:  map[string]string{},
				ValueMap:     map[string]VMValue{},
				ValueMapTemp: map[string]VMValue{},
			}
			players[msg.Sender.UserId] = p
		}
		if p.ValueMap == nil {
			p.ValueMap = map[string]VMValue{}
		}
		if p.ValueMapTemp == nil {
			p.ValueMapTemp = map[string]VMValue{}
		}
		return p
	}
	return nil
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

func DiceFormat(ctx *MsgContext, s string) string {
	r, _, _ := ctx.Dice.ExprText(s, ctx)
	return r
}
