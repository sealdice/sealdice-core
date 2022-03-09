package dice

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

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
				Name:         msg.Sender.Nickname,
				UserId:       msg.Sender.UserId,
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
		if p.InGroup == false {
			p.InGroup = true
		}
		return p
	}

	// 私聊信息
	return &PlayerInfo{
		Name:         msg.Sender.Nickname,
		UserId:       msg.Sender.UserId,
		ValueMap:     map[string]VMValue{},
		ValueMapTemp: map[string]VMValue{},
	}
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

	convert := func(s string) string {
		//var s2 string
		//raw := []byte(`"` + strings.Replace(s, `"`, `\"`, -1) + `"`)
		//err := json.Unmarshal(raw, &s2)
		//if err != nil {
		//	ctx.Dice.Logger.Info(err)
		//	return s
		//}

		solve := func(cq *CQCommand) {
			if cq.Type == "image" {
				fn, exists := cq.Args["file"]
				if exists {
					afn, _ := filepath.Abs(fn)
					cwd, _ := os.Getwd()

					if strings.HasPrefix(afn, cwd) {
						if _, err := os.Stat(afn); errors.Is(err, os.ErrNotExist) {
							cq.Overwrite = "[找不到图片]"
						} else {
							// 这里使用绝对路径，windows上gocqhttp会裁掉一个斜杠，所以我这里加一个
							if runtime.GOOS == `windows` {
								afn = "/" + afn
							}
							u := url.URL{
								Scheme: "file",
								Path:   afn,
							}
							cq.Args["file"] = u.String()
						}
					} else {
						cq.Overwrite = "[CQ码读取非当前目录图片，可能是恶意行为，已禁止]"
					}
				}
			}
		}

		text := strings.Replace(s, `\n`, "\n", -1)
		return CQRewrite(text, solve)
	}

	return convert(r)
}
