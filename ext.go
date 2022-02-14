package main

import (
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"time"
)

func (self *Dice) registerBuiltinExt() {
	self.extList = append(self.extList, &ExtInfo{
		"coc7",
		"0.0.1",
		true,
		CmdMapCls{
			"test": &CmdItemInfo{
				name: "test",
				solve: func(session *IMSession, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
					if isCurGroupBotOn(session, msg) && len(cmdArgs.Args) >= 0 {
						replyGroup(session.Socket, msg.GroupId, "模块装载测试");
					}
					return struct{ success bool }{
						success: true,
					}
				},
			},
			"ra": &CmdItemInfo{
				name: "ra",
				solve: func(session *IMSession, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
					if isCurGroupBotOn(session, msg) && len(cmdArgs.Args) >= 1 {
						var cond int64
						p := getPlayerInfoBySender(session, msg)

						if len(cmdArgs.Args) >= 1 {
							var err error
							var suffix, detail string
							var r *vmStack
							d100 := DiceRoll64(100)
							r, detail, err = session.parent.exprEval(cmdArgs.RawArgs, p)
							if r != nil && r.typeId == 0 {
								cond = r.value.(*big.Int).Int64()
							}

							if d100 <= cond {
								suffix = "成功"
								if d100 <= cond / 2 {
									suffix = "成功(困难)"
								}
								if d100 <= cond / 4 {
									suffix = "成功(极难)"
								}
								if d100 <= 5 {
									suffix = "大成功！"
								}
							} else {
								if d100 > 95 {
									suffix = "大失败！"
								} else {
									suffix = "失败！"
								}
							}

							if err == nil {
								detailWrap := ""
								if detail != "" {
									detailWrap = "=(" + detail + ")"
								}

								text := fmt.Sprintf("<%s>的“%s”检定结果为: D100=%d/%d%s %s", p.Name, cmdArgs.RawArgs, d100, cond, detailWrap, suffix)
								replyGroup(session.Socket, msg.GroupId, text);
							} else {
								replyGroup(session.Socket, msg.GroupId, "表达式不正确，可能是找不到属性");
							}
						}
					}
					return struct{ success bool }{
						success: true,
					}
				},
			},
			"st": &CmdItemInfo{
				name: "st",
				solve: func(session *IMSession, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
					// .st show
					// .st help
					// .st (<Name>[0-9]+)+
					if isCurGroupBotOn(session, msg) && len(cmdArgs.Args) >= 0 {
						var param1 string
						if len(cmdArgs.Args) == 0 {
							param1 = ""
						} else {
							param1 = cmdArgs.Args[0]
						}
						switch param1 {
						case "help", "":
							text := "属性设置指令，支持分支指令如下：\n"
							text += ".st show/list // 展示个人属性\n"
							text += ".st clr/clear // 清除属性\n"
							text += ".st del <属性名1> <属性名2> ... // 删除属性，可多项，以空格间隔\n"
							text += ".st help // 帮助\n"
							text += ".st <属性名><值> // 例：.st 敏捷50"
							replyGroup(session.Socket, msg.GroupId, text);

						case "del":
							p := getPlayerInfoBySender(session, msg)
							vm := p.ValueNumMap
							nums := []string{}
							failed := []string{}

							for _, varname := range cmdArgs.Args[1:] {
								_, ok := vm[varname]
								if ok {
									nums = append(nums, varname)
									delete(p.ValueNumMap, varname)
								} else {
									failed = append(failed, varname)
								}
							}

							text := fmt.Sprintf("<%s>的如下属性被成功删除:%s，失败%d项\n", p.Name, nums, len(failed))
							replyGroup(session.Socket, msg.GroupId, text);

						case "clr", "clear":
							p := getPlayerInfoBySender(session, msg)
							num := len(p.ValueNumMap)
							p.ValueNumMap = map[string]int64{};
							text := fmt.Sprintf("<%s>的属性数据已经清除，共计%d条", p.Name, num)
							replyGroup(session.Socket, msg.GroupId, text);

						case "show", "list":
							info := ""
							name := msg.Sender.Nickname

							players := session.ServiceAt[msg.GroupId].Players;
							p := players[msg.Sender.UserId]
							name = p.Name

							if len(p.ValueNumMap) == 0 {
								info = "未发现属性记录"
							} else {
								for k, v := range p.ValueNumMap {
									info += fmt.Sprintf("%s: %d\n", k, v)
								}
							}

							text := fmt.Sprintf("<%s>的个人属性为：\n%s", name, info)
							replyGroup(session.Socket, msg.GroupId, text);

						default:
							valueMap := map[string]int64{};
							re, _ := regexp.Compile(`([^\d]+?)[:=]?(\d+)`)

							// 读取所有参数中的值
							stText := ""
							for _, text := range cmdArgs.Args {
								stText += text
							}

							m := re.FindAllStringSubmatch(RemoveSpace(stText), -1)

							for _, i := range m {
								num, err := strconv.ParseInt(i[2], 10, 64);
								if err == nil {
									valueMap[i[1]] = num;
								}
							}

							for _, v := range cmdArgs.Kwargs {
								vint, err := strconv.ParseInt(v.Value, 10, 64)
								if err == nil {
									valueMap[v.Name] = vint
								}
							}

							p := getPlayerInfoBySender(session, msg)

							for k, v := range valueMap {
								p.ValueNumMap[k] = v;
							}

							p.lastUpdateTime = time.Now().Unix();
							//s, _ := json.Marshal(valueMap)
							text := fmt.Sprintf("<%s>的属性录入完成，本次共记录了%d条数据", p.Name, len(valueMap))
							replyGroup(session.Socket, msg.GroupId, text);
						}
					}
					return struct{ success bool }{
						success: true,
					}
				},
			},
		},
	})
}
