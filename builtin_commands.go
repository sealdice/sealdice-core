package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var VERSION = "0.9测试版 v20220226"

/** 这几条指令不能移除 */
func (self *Dice) registerCoreCommands() {
	cmdHelp := &CmdItemInfo{
		name: "help",
		Help: ".help // 查看本帮助",
		solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
			if ctx.isCurGroupBotOn {
				text := "SealDice " + VERSION + "\n"
				text += "-----------------------------------------------\n"
				text += "核心指令列表如下:\n"

				used := map[*CmdItemInfo]bool{}
				keys := make([]string, 0, len(self.cmdMap))
				for k, v := range self.cmdMap {
					if used[v] {
						continue
					}
					keys = append(keys, k)
					used[v] = true
				}
				sort.Strings(keys)

				for _, i := range keys {
					i := self.cmdMap[i]
					if i.Help != "" {
						text += i.Help + "\n"
					} else {
						brief := i.Brief
						if brief != "" {
							brief = "   // " + brief
						}
						text += "." + i.name + brief + "\n"
					}
				}

				text += "注意：由于篇幅此处仅列出核心指令。\n"
				text += "扩展指令请输入 .ext 和 .ext <扩展名称> 进行查看\n"
				text += "-----------------------------------------------\n"
				text += "SealDice 目前 7*24h 运行于一块陈年OrangePi卡片电脑上，随时可能因为软硬件故障停机（例如过热、被猫打翻）。届时可以来Q群524364253询问。"
				replyToSender(ctx, msg, text)
			}
			return struct{ success bool }{
				success: true,
			}
		},
	}
	self.cmdMap["help"] = cmdHelp

	cmdBot := &CmdItemInfo{
		name: "bot on/off/about/bye",
		Brief: "开启、关闭、查看信息、退群",
		Help: ".bot on/off/about/bye // 开启、关闭、查看信息、退群",
		solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
			inGroup := msg.MessageType == "group"

			if len(cmdArgs.Args) == 0 || cmdArgs.isArgEqual(1, "about") {
				count := 0
				for _, i := range self.ImSession.ServiceAt {
					if i.Active {
						count += 1
					}
				}
				lastSavedTimeText := "从未"
				if self.lastSavedTime != nil {
					lastSavedTimeText = self.lastSavedTime.Format("2006-01-02 15:04:05") + " UTC"
				}
				text := fmt.Sprintf("SealDice %s\n兼容模式: 已开启\n供职于%d个群，其中%d个处于开启状态\n上次自动保存时间: %s", VERSION, len(self.ImSession.ServiceAt), count, lastSavedTimeText)

				if inGroup {
					if cmdArgs.AmIBeMentioned {
						replyGroup(ctx, msg.GroupId, text)
					}
				} else {
					replyPerson(ctx, msg.Sender.UserId, text)
				}
			} else {
				if inGroup && cmdArgs.AmIBeMentioned {
					if len(cmdArgs.Args) >= 1 {
						if cmdArgs.Args[0] == "on" {
							if ctx.group != nil {
								ctx.group.Active = true
							} else {
								extLst := []*ExtInfo{}
								for _, i := range self.extList {
									if i.autoActive {
										extLst = append(extLst, i)
									}
								}
								ctx.session.ServiceAt[msg.GroupId] = &ServiceAtItem{
									Active:           true,
									ActivatedExtList: extLst,
									Players:          map[int64]*PlayerInfo{},
									GroupId: msg.GroupId,
									ValueMap: map[string]VMValue{},
								}
								ctx.group = ctx.session.ServiceAt[msg.GroupId]
								ctx.isCurGroupBotOn = true
							}
							replyGroup(ctx, msg.GroupId, "SealDice 已启用(开发中) " + VERSION)
						} else if cmdArgs.Args[0] == "off" {
							if len(ctx.group.ActivatedExtList) == 0 {
								delete(ctx.session.ServiceAt, msg.GroupId)
							} else {
								ctx.group.Active = false
							}
							replyGroup(ctx, msg.GroupId, "停止服务")
						} else if cmdArgs.Args[0] == "bye" {
							replyGroup(ctx, msg.GroupId, "收到指令，5s后将退出当前群组")
							time.Sleep(6 * time.Second)
							quitGroup(ctx.session, msg.GroupId)
						} else if cmdArgs.Args[0] == "save" {
							self.save()
							replyGroup(ctx, msg.GroupId, fmt.Sprintf("数据已保存"))
						}
					}
				}
			}

			return struct{ success bool }{
				success: true,
			}
		},
	}
	self.cmdMap["bot"] = cmdBot

	cmdRoll := &CmdItemInfo{
		name: "r <表达式> <原因>",
		Brief: "骰点指令，案例:“.r d16” “.r 3d10*2+3” “.r d10+力量” “.r 2d(力量+1d3)” “.rh d16 (暗骰)” ",
		Help: ".r <表达式> <原因> // 骰点指令\n.rh <表达式> <原因> // 暗骰",
		solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
			if ctx.isCurGroupBotOn {
				var text string
				var prefix string
				var diceResult int64
				var diceResultExists bool
				var detail string

				if ctx.dice.CommandCompatibleMode {
					if (cmdArgs.Command == "rd" || cmdArgs.Command == "rhd") && len(cmdArgs.Args) >= 1 {
						if m, _ := regexp.MatchString(`^\d`, cmdArgs.Args[0]); m {
							cmdArgs.Args[0] = "d" + cmdArgs.Args[0]
						}
					}
				} else {
					return struct{ success bool }{
						success: false,
					}
				}

				forWhat := ""
				var r *vmResult
				if len(cmdArgs.Args) >= 1 {
					var err error
					r, detail, err = ctx.dice.exprEval(cmdArgs.Args[0], ctx)

					if r != nil && r.TypeId == 0 {
						diceResult = r.Value.(int64)
						diceResultExists = true
						//return errors.New("错误的类型")
					}

					if err == nil {
						if len(cmdArgs.Args) >= 2 {
							forWhat = cmdArgs.Args[1]
						}
					} else {
						errs := string(err.Error())
						if strings.HasPrefix(errs, "E1:") {
							replyGroup(ctx, msg.GroupId, errs)

							return struct{ success bool }{
								success: true,
							}
						}
						forWhat = cmdArgs.Args[0]
					}
				}

				if forWhat != "" {
					prefix = "为了" + forWhat + "，"
				}

				if diceResultExists {
					detailWrap := ""
					if detail != "" {
						detailWrap = "=" + detail
					}
					text = fmt.Sprintf("%s<%s>掷出了 %s%s=%d", prefix, ctx.player.Name, cmdArgs.Args[0], detailWrap, diceResult)
				} else {
					dicePoints := ctx.player.DiceSideNum
					if dicePoints <= 0 {
						dicePoints = 100
					}
					val := DiceRoll(dicePoints)
					text = fmt.Sprintf("%s<%s>掷出了 D%d=%d", prefix, ctx.player.Name, dicePoints, val)
				}

				if kw := cmdArgs.GetKwarg("asm"); r != nil && kw != nil {
					asm := r.parser.GetAsmText()
					text += "\n" + asm
				}

				if cmdArgs.Command == "rh" || cmdArgs.Command == "rhd" {
					prefix := fmt.Sprintf("来自群<%s>(%d)的暗骰，", ctx.group.GroupName, msg.GroupId)
					replyGroup(ctx, msg.GroupId, "黑暗的角落里，传来命运转动的声音")
					replyPerson(ctx, msg.Sender.UserId, prefix + text)
				} else {
					replyGroup(ctx, msg.GroupId, text)
				}
			}

			return struct{ success bool }{
				success: true,
			}
		},
	}
	self.cmdMap["r"] = cmdRoll
	self.cmdMap["rd"] = cmdRoll
	self.cmdMap["roll"] = cmdRoll
	self.cmdMap["rh"] = cmdRoll
	self.cmdMap["rhd"] = cmdRoll

	cmdExt := &CmdItemInfo{
		name: "ext",
		Brief: "查看扩展列表",
		Help: ".ext // 查看扩展列表",
		solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
			if ctx.isCurGroupBotOn {
				showList := func () {
					text := "检测到以下扩展：\n"
					for index, i := range ctx.dice.extList {
						state := "关"
						for _, j := range ctx.group.ActivatedExtList {
							if i.Name == j.Name {
								state = "开"
								break
							}
						}
						author := i.Author
						if author == "" {
							author = "<未注明>"
						}
						text += fmt.Sprintf("%d. [%s]%s - 版本:%s 作者:%s\n", index + 1, state, i.Name, i.version, author)
					}
					text += "使用命令: .ext <扩展名> on/off 可以在当前群开启或关闭某扩展。\n"
					text += "命令: .ext <扩展名> 可以查看扩展介绍及帮助"
					replyGroup(ctx, msg.GroupId, text)
				}

				if len(cmdArgs.Args) == 0 {
					showList()
				}

				if len(cmdArgs.Args) >= 1 {
					if cmdArgs.isArgEqual(1, "list") {
						showList()
					} else if cmdArgs.isArgEqual(2, "on") {
						extName := cmdArgs.Args[0]
						for _, i := range self.extList {
							if i.Name == extName {
								ctx.group.ActivatedExtList = append(ctx.group.ActivatedExtList, i)
								replyGroup(ctx, msg.GroupId, fmt.Sprintf("打开扩展 %s", extName))
								break
							}
						}
					} else if cmdArgs.isArgEqual(2, "off") {
						extName := cmdArgs.Args[0]
						for index, i := range ctx.group.ActivatedExtList {
							if i.Name == extName {
								ctx.group.ActivatedExtList = append(ctx.group.ActivatedExtList[:index], ctx.group.ActivatedExtList[index+1:]...)
								replyGroup(ctx, msg.GroupId, fmt.Sprintf("关闭扩展 %s", extName))
							}
						}
					} else {
						extName := cmdArgs.Args[0]
						for _, i := range self.extList {
							if i.Name == extName {
								text := fmt.Sprintf("> [%s] 版本%s 作者%s\n", i.Name, i.version, i.Author)
								replyToSender(ctx, msg, text + i.GetDescText(i))
								break
							}
						}
					}
				}
			}

			return struct{ success bool }{
				success: true,
			}
		},
	}
	self.cmdMap["ext"] = cmdExt

	cmdNN := &CmdItemInfo{
		name: "nn <角色名>",
		Brief: ".nn后跟角色名则改角色名，不带则重置角色名",
		solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
			if msg.MessageType == "group" {
				if ctx.isCurGroupBotOn {
					if len(cmdArgs.Args) == 0 {
						p := ctx.player
						p.Name = msg.Sender.Nickname
						replyGroup(ctx, msg.GroupId, fmt.Sprintf("%s(%d) 的昵称已重置为<%s>", msg.Sender.Nickname, msg.Sender.UserId, p.Name))
					}
					if len(cmdArgs.Args) >= 1 {
						p := ctx.player
						p.Name = cmdArgs.Args[0]
						replyGroup(ctx, msg.GroupId, fmt.Sprintf("%s(%d) 的昵称被设定为<%s>", msg.Sender.Nickname, msg.Sender.UserId, p.Name))
					}
				}
			}

			return struct{ success bool }{
				success: true,
			}
		},
	}
	self.cmdMap["nn"] = cmdNN

	cmdSet := &CmdItemInfo{
		name: "set <面数>",
		Brief: "设置默认骰子面数，只对自己有效",
		solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
			if ctx.isCurGroupBotOn {
				p := ctx.player
				if len(cmdArgs.Args) >= 1 {
					num, err := strconv.Atoi(cmdArgs.Args[0])
					if err == nil {
						p.DiceSideNum = num
						replyGroup(ctx, msg.GroupId, fmt.Sprintf("设定默认骰子面数为 %d", num))
					} else {
						replyGroup(ctx, msg.GroupId, fmt.Sprintf("设定默认骰子面数: 格式错误"))
					}
				} else {
					p.DiceSideNum = 0
					replyGroup(ctx, msg.GroupId, fmt.Sprintf("重设默认骰子面数为初始"))
				}
			}

			return struct{ success bool }{
				success: true,
			}
		},
	}
	self.cmdMap["set"] = cmdSet

	cmdText := &CmdItemInfo{
		name: "text",
		Brief: "文本指令(测试)，举例: .text 1D16={ 1d16 }，属性计算: 攻击 - 防御 = {攻击} - {防御} = {攻击 - 防御}",
		Help: ".text <文本模板> // 文本指令，例: .text 丢个16面骰看看: {1d16}",
		solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
			if ctx.isCurGroupBotOn || ctx.MessageType == "private" {
				val, _, err := self.exprText(cmdArgs.RawArgs, ctx)

				if err == nil {
					replyToSender(ctx, msg, val)
				} else {
					replyToSender(ctx, msg, "格式错误")
				}
			}

			return struct{ success bool }{
				success: true,
			}
		},
	}
	self.cmdMap["text"] = cmdText


	cmdChar := &CmdItemInfo{
		name: "ch",
		//Help: ".ch save <角色名> // 保存角色，角色名省略则为当前昵称\n.ch load <角色名> // 加载角色\n.ch list // 列出当前角色",
		Help: ".ch list/save/load // 角色管理",
		solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
			if ctx.isCurGroupBotOn {
				getNickname := func() string {
					name, _ := cmdArgs.GetArgN(2)
					if name == "" {
						name = ctx.player.Name
					}
					return name
				}

				if cmdArgs.isArgEqual(1, "list") {
					vars := ctx.LoadPlayerVars()
					characters := []string{}
					for k, _ := range vars.ValueMap {
						if strings.HasPrefix(k, "$ch:") {
							characters = append(characters, k[4:])
						}
					}
					if len(characters) == 0 {
						replyToSender(ctx, msg, fmt.Sprintf("<%s>当前还没有角色列表", ctx.player.Name))
					} else {
						replyToSender(ctx, msg, fmt.Sprintf("<%s>的角色列表为:\n%s", ctx.player.Name, strings.Join(characters, "\n")))
					}
				} else if cmdArgs.isArgEqual(1, "load") {
					name := getNickname()
					vars := ctx.LoadPlayerVars()
					data, exists := vars.ValueMap["$ch:" + name]

					if exists {
						ctx.player.ValueMap = make(map[string]VMValue)
						err := JsonValueMapUnmarshal([]byte(data.Value.(string)), &ctx.player.ValueMap)
						if err == nil {
							ctx.player.Name = name
							replyToSender(ctx, msg, fmt.Sprintf("角色<%s>加载成功，欢迎回来。", name))
						} else {
							replyToSender(ctx, msg, "无法加载角色：序列化失败")
						}
					} else {
						replyToSender(ctx, msg, "无法加载角色：你所指定的角色不存在")
					}
				}  else if cmdArgs.isArgEqual(1, "save") {
					name := getNickname()
					vars := ctx.LoadPlayerVars()
					v, err := json.Marshal(ctx.player.ValueMap)

					if err == nil {
						vars.ValueMap["$ch:" + name] = VMValue{
							VMTypeString,
							string(v),
						}
						replyToSender(ctx, msg, fmt.Sprintf("角色<%s>储存成功", name))
					} else {
						replyToSender(ctx, msg, "无法储存角色：序列化失败")
					}
				} else {
					help := "角色指令\n"
					help += ".ch save <角色名> // 保存角色，角色名省略则为当前昵称\n.ch load <角色名> // 加载角色，角色名省略则为当前昵称\n.ch list // 列出当前角色"
					replyToSender(ctx, msg, help)
				}
			}

			return struct{ success bool }{
				success: true,
			}
		},
	}
	self.cmdMap["角色"] = cmdChar
	self.cmdMap["ch"] = cmdChar
	self.cmdMap["char"] = cmdChar
	self.cmdMap["character"] = cmdChar
}
