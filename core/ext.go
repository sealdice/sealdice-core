package main

import (
	"fmt"
	"math/rand"
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
						p := getPlayerInfoBySender(session, msg)

						re1 := regexp.MustCompile(`^\d+$`);
						if re1.MatchString(cmdArgs.Args[0]) {
							// .ra [0-9]+
							cond, _ := strconv.Atoi(cmdArgs.Args[0])
							val := DiceRoll(100);

							suffix := "成功"
							if val > cond {
								suffix = "失败"
							} else if val <= 5 {
								suffix = "大成功！"
							}

							text := fmt.Sprintf("<%s>的%s检定结果为: D100=%d/%d %s", p.Name, "", val, cond, suffix)
							replyGroup(session.Socket, msg.GroupId, text);
						} else {
							re2, _ := regexp.Compile(`^([^\d]+)(\d+)$`)
							if re2.MatchString(cmdArgs.Args[0]) {
								valueMap := map[string]int64{};
								m := re2.FindAllStringSubmatch(cmdArgs.Args[0], -1)

								for _, i := range m {
									num, err := strconv.ParseInt(i[2], 10, 64);
									if err == nil {
										valueMap[i[1]] = num;
									} else {
										valueMap[i[1]] = 50; // 默认值 50
									}
									break // TODO: 先只判定一个，偷个懒
								}

								for k, cond := range valueMap {
									val := DiceRoll64(100);

									suffix := "成功"
									if val > cond {
										suffix = "失败"
									} else if val <= 5 {
										suffix = "大成功！"
									}

									text := fmt.Sprintf("<%s>的%s检定结果为: D100=%d/%d %s", p.Name, k, val, cond, suffix)
									replyGroup(session.Socket, msg.GroupId, text);
								}
							} else {
								re3, _ := regexp.Compile(`[^\d]+`)
								if re3.MatchString(cmdArgs.Args[0]) {
									attrName := re3.FindString(cmdArgs.Args[0])

									if cond, ok := p.ValueNumMap[attrName]; ok {
										val := rand.Int63() % 100;

										suffix := "成功"
										if val > cond {
											suffix = "失败"
										} else if val <= 5 {
											suffix = "大成功！"
										}

										text := fmt.Sprintf("<%s>的%s检定结果为: D100=%d/%d %s", p.Name, attrName, val, cond, suffix)
										replyGroup(session.Socket, msg.GroupId, text);
									} else {
										text := fmt.Sprintf("<%s>检定失败，找不到属性:%s", p.Name, attrName)
										replyGroup(session.Socket, msg.GroupId, text);
									}
								}
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
						switch cmdArgs.Args[0] {
						case "help":
							text := "属性设置指令，支持分支指令如下：\n"
							text += ".st show // 展示个人属性\n"
							text += ".st help // 帮助\n"
							text += ".st <属性名><值> // 例：.st 敏捷50"
							replyGroup(session.Socket, msg.GroupId, text);

						case "show":
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
							re, _ := regexp.Compile(`([^\d]+)(\d+)`)

							// 读取所有参数中的值
							for _, text := range cmdArgs.Args {
								m := re.FindAllStringSubmatch(text, -1)

								for _, i := range m {
									num, err := strconv.ParseInt(i[2], 10, 64);
									if err == nil {
										valueMap[i[1]] = num;
									}
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
