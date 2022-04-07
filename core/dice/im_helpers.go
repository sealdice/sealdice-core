package dice

import (
	"errors"
	"github.com/fy0/lockfree"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

func IsCurGroupBotOn(session *IMSession, msg *Message) bool {
	return msg.MessageType == "group" && session.ServiceAtNew[msg.GroupId] != nil && session.ServiceAtNew[msg.GroupId].Active
}

func IsCurGroupBotOnById(session *IMSession, messageType string, groupId string) bool {
	return messageType == "group" && session.ServiceAtNew[groupId] != nil && session.ServiceAtNew[groupId].Active
}

func SetBotOnAtGroup(ctx *MsgContext, groupId string) *GroupInfo {
	session := ctx.Session
	group := session.ServiceAtNew[groupId]
	if group != nil {
		group.Active = true
	} else {
		extLst := []*ExtInfo{}
		for _, i := range session.Parent.ExtList {
			if i.AutoActive {
				extLst = append(extLst, i)
			}
		}

		session.ServiceAtNew[groupId] = &GroupInfo{
			Active:           true,
			ActivatedExtList: extLst,
			Players:          map[string]*GroupPlayerInfo{},
			GroupId:          groupId,
			ValueMap:         lockfree.NewHashMap(),
			DiceIds:          map[string]bool{},
		}
		group = session.ServiceAtNew[groupId]
	}

	if group.DiceIds == nil {
		group.DiceIds = map[string]bool{}
	}
	if group.BotList == nil {
		group.BotList = map[string]bool{}
	}

	group.DiceIds[ctx.EndPoint.UserId] = true
	return group
}

/* 获取玩家群内信息，没有就创建 */
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

func ReplyToSender(ctx *MsgContext, msg *Message, text string) {
	ctx.EndPoint.Adapter.ReplyToSender(ctx, msg, text)
}

func ReplyGroup(ctx *MsgContext, msg *Message, text string) {
	ctx.EndPoint.Adapter.ReplyGroup(ctx, msg, text)
}

func ReplyPerson(ctx *MsgContext, msg *Message, text string) {
	ctx.EndPoint.Adapter.ReplyPerson(ctx, msg, text)
}

func ReplyToSenderRaw(ctx *MsgContext, msg *Message, text string, flag string) {
	ctx.EndPoint.Adapter.ReplyToSenderRaw(ctx, msg, text, flag)
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

		solve2 := func(text string) string {
			re := regexp.MustCompile(`\[(img|图):(.+?)]`) // [img:] 或 [图:]
			m := re.FindStringSubmatch(text)
			if m != nil {
				fn := m[2]
				if strings.HasPrefix(fn, "file://") || strings.HasPrefix(fn, "http://") || strings.HasPrefix(fn, "https://") {
					u, err := url.Parse(fn)
					if err != nil {
						return text
					}
					cq := CQCommand{
						Type: "image",
						Args: map[string]string{"file": u.String()},
					}
					return cq.Compile()
				}

				afn, err := filepath.Abs(fn)
				if err != nil {
					return text // 不是文件路径，不管
				}
				cwd, _ := os.Getwd()
				if strings.HasPrefix(afn, cwd) {
					if _, err := os.Stat(afn); errors.Is(err, os.ErrNotExist) {
						return "[找不到图片]"
					} else {
						// 这里使用绝对路径，windows上gocqhttp会裁掉一个斜杠，所以我这里加一个
						if runtime.GOOS == `windows` {
							afn = "/" + afn
						}
						u := url.URL{
							Scheme: "file",
							Path:   afn,
						}
						cq := CQCommand{
							Type: "image",
							Args: map[string]string{"file": u.String()},
						}
						return cq.Compile()
					}
				} else {
					return "[图片指向非当前程序目录，已禁止]"
				}
			}
			return text
		}

		solve := func(cq *CQCommand) {
			if cq.Type == "image" {
				fn, exists := cq.Args["file"]
				if exists {
					if strings.HasPrefix(fn, "file://") || strings.HasPrefix(fn, "http://") || strings.HasPrefix(fn, "https://") {
						return
					}

					afn, err := filepath.Abs(fn)
					if err != nil {
						return // 不是文件路径，不管
					}
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
		text = ImageRewrite(text, solve2)
		return CQRewrite(text, solve)
	}

	return convert(r)
}
