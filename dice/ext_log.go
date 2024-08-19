package dice

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/golang-module/carbon"
	ds "github.com/sealdice/dicescript"
	"go.uber.org/zap"

	"sealdice-core/dice/model"
	"sealdice-core/dice/storylog"
	"sealdice-core/utils"
)

var ErrGroupCardOverlong = errors.New("群名片长度超过限制")

func SetPlayerGroupCardByTemplate(ctx *MsgContext, tmpl string) (string, error) {
	ctx.Player.TempValueAlias = nil // 防止dnd的hp被转为“生命值”

	v := ctx.EvalFString(tmpl, nil)
	if v.vm.Error != nil {
		ctx.Dice.Logger.Infof("SN指令模板错误: %v", v.vm.Error.Error())
		return "", v.vm.Error
	}

	text := v.ToString()
	if ctx.EndPoint.Platform == "QQ" && len(text) >= 60 { // Note(Xiangze-Li): 2023-08-09实测群名片长度限制为59个英文字符, 20个中文字符是可行的, 但分别判断过于繁琐
		return text, ErrGroupCardOverlong
	}

	ctx.EndPoint.Adapter.SetGroupCardName(ctx, text)
	return text, nil
}

// {"data":null,"msg":"SEND_MSG_API_ERROR","retcode":100,"status":"failed","wording":"请参考 go-cqhttp 端输出"}

func RegisterBuiltinExtLog(self *Dice) {
	privateCommandListen := map[int64]int64{}

	// 这个机制作用是记录私聊指令？？忘记了
	privateCommandListenCheck := func() {
		now := time.Now().Unix()
		newMap := map[int64]int64{}
		for k, v := range privateCommandListen {
			// 30s间隔以上清除
			if now-v < 30 {
				newMap[k] = v
			}
		}
		privateCommandListen = newMap
	}

	// 避免群信息重复记录
	groupMsgInfo := SyncMap[any, int64]{}
	groupMsgInfoLastClean := int64(0)
	groupMsgInfoClean := func() {
		// 清理过久的消息
		now := time.Now().Unix()

		if now-groupMsgInfoLastClean < 60 {
			// 60s清理一次
			return
		}

		groupMsgInfoLastClean = now
		var toDelete []any
		groupMsgInfo.Range(func(key any, t int64) bool {
			if now-t > 5 { // 5秒内如果有此消息，那么不记录
				toDelete = append(toDelete, key)
			}
			return true
		})

		for _, i := range toDelete {
			groupMsgInfo.Delete(i)
		}
	}

	// 检查是否已经记录过 如果记录过则跳过
	groupMsgInfoCheckOk := func(_k interface{}) bool {
		groupMsgInfoClean()
		if _k == nil {
			return false
		}
		t, exists := groupMsgInfo.Load(_k)
		if exists {
			now := time.Now().Unix()
			return now-t > 5 // 5秒内如果有此消息，那么不记录
		}
		return true
	}

	groupMsgInfoSet := func(_k any) {
		if _k != nil {
			groupMsgInfo.Store(_k, time.Now().Unix())
		}
	}

	// 获取logname，第一项是默认名字
	getLogName := func(ctx *MsgContext, _ *Message, cmdArgs *CmdArgs, index int) (string, string) {
		bakLogCurName := ctx.Group.LogCurName
		if newName := cmdArgs.GetArgN(index); newName != "" {
			return bakLogCurName, newName
		}
		return bakLogCurName, bakLogCurName
	}

	const helpLog = `.log new [<日志名>] // 新建日志并开始记录，注意new后跟空格！
.log on [<日志名>]  // 开始记录，不写日志名则开启最近一次日志，注意on后跟空格！
.log off // 暂停记录
.log end // 完成记录并发送日志文件
.log get [<日志名>] // 重新上传日志，并获取链接
.log halt // 强行关闭当前log，不上传日志
.log list // 查看当前群的日志列表
.log del <日志名> // 删除一份日志
.log stat [<日志名>] // 查看统计
.log stat [<日志名>] --all // 查看统计(全团)，--all前必须有空格
.log list <群号> // 查看指定群的日志列表(无法取得日志时，找骰主做这个操作)
.log masterget <群号> <日志名> // 重新上传日志，并获取链接(无法取得日志时，找骰主做这个操作)
.log export <日志名> // 直接取得日志txt(服务出问题或有其他需要时使用)
.log export <日志名> <邮箱地址> // 通过邮件取得日志txt，多个邮箱用空格隔开`

	// const txtLogTip = "若未出现线上日志地址，可换时间获取，或联系骰主在data/default/log-exports路径下取出日志\n文件名: 群号_日志名_随机数.zip\n注意此文件log end/get后才会生成"

	cmdLog := &CmdItemInfo{
		Name:      "log",
		ShortHelp: helpLog,
		Help:      "日志指令:\n" + helpLog,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			group := ctx.Group
			cmdArgs.ChopPrefixToArgsWith("on", "off", "del", "rm", "masterget",
				"get", "end", "halt", "list", "new", "stat", "export")

			groupNotActiveCheck := func() bool {
				if !group.IsActive(ctx) {
					ReplyToSender(ctx, msg, "未开启时不会记录日志，请先.bot on")
					return true
				}
				return false
			}

			if len(cmdArgs.Args) == 0 {
				onText := "关闭"
				if group.LogOn {
					onText = "开启"
				}
				lines, _ := model.LogLinesCountGet(ctx.Dice.DBLogs, group.GroupID, group.LogCurName)
				text := fmt.Sprintf("当前故事: %s\n当前状态: %s\n已记录文本%d条", group.LogCurName, onText, lines)
				ReplyToSender(ctx, msg, text)
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			getAndUpload := func(gid, lname string) {
				unofficial, fn, err := LogSendToBackend(ctx, gid, lname)
				if err != nil {
					reason := strings.TrimPrefix(err.Error(), "#")
					VarSetValueStr(ctx, "$t错误原因", reason)

					tmpl := DiceFormatTmpl(ctx, "日志:记录_上传_失败")
					ReplyToSenderRaw(ctx, msg, tmpl, "skip")
				} else {
					VarSetValueStr(ctx, "$t日志链接", fn)
					tmpl := DiceFormatTmpl(ctx, "日志:记录_上传_成功")
					if unofficial {
						tmpl += "\n[注意：该链接非海豹官方染色器]"
					}
					ReplyToSenderRaw(ctx, msg, tmpl, "skip")
				}
			}

			if cmdArgs.IsArgEqual(1, "on") { //nolint:nestif
				if ctx.IsPrivate {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_私聊不可用"))
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				// 如果日志已经开启，报错返回
				if group.LogOn {
					VarSetValueStr(ctx, "$t记录名称", group.LogCurName)
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "日志:记录_开启_失败_未结束的记录"))
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				name := cmdArgs.GetArgN(2)
				if name == "" {
					name = group.LogCurName
				}

				if name != "" {
					lines, exists := model.LogLinesCountGet(ctx.Dice.DBLogs, group.GroupID, name)

					if exists {
						if groupNotActiveCheck() {
							return CmdExecuteResult{Matched: true, Solved: true}
						}

						group.LogOn = true
						group.LogCurName = name
						group.UpdatedAtTime = time.Now().Unix()

						VarSetValueStr(ctx, "$t记录名称", name)
						VarSetValueInt64(ctx, "$t当前记录条数", lines)
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "日志:记录_开启_成功"))
					} else {
						VarSetValueStr(ctx, "$t记录名称", name)
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "日志:记录_开启_失败_无此记录"))
					}
				} else {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "日志:记录_开启_失败_尚未新建"))
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			} else if cmdArgs.IsArgEqual(1, "off") {
				if group.LogCurName != "" && group.LogOn {
					group.LogOn = false
					group.UpdatedAtTime = time.Now().Unix()
					lines, _ := model.LogLinesCountGet(ctx.Dice.DBLogs, group.GroupID, group.LogCurName)
					VarSetValueStr(ctx, "$t记录名称", group.LogCurName)
					VarSetValueInt64(ctx, "$t当前记录条数", lines)
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "日志:记录_关闭_成功"))
				} else {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "日志:记录_关闭_失败"))
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			} else if cmdArgs.IsArgEqual(1, "del", "rm") {
				name := cmdArgs.GetArgN(2)
				if name == "" {
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}

				VarSetValueStr(ctx, "$t记录名称", name)
				if name == group.LogCurName {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "日志:记录_删除_失败_正在进行"))
				} else {
					ok := model.LogDelete(ctx.Dice.DBLogs, group.GroupID, name)
					if ok {
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "日志:记录_删除_成功"))
					} else {
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "日志:记录_删除_失败_找不到"))
					}
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			} else if cmdArgs.IsArgEqual(1, "masterget") {
				groupID, requestForAnotherGroup := getSpecifiedGroupIfMaster(ctx, msg, cmdArgs)
				if requestForAnotherGroup && groupID == "" {
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				logName := cmdArgs.GetArgN(3)
				if logName == "" {
					ReplyToSenderRaw(ctx, msg, "请遵循 .log masterget <群号> <日志名> 格式给出日志名，注意空格\n若不清楚可以.log list <群号>查询", "skip")
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				getAndUpload(groupID, logName)
				return CmdExecuteResult{Matched: true, Solved: true}
			} else if cmdArgs.IsArgEqual(1, "get") {
				logName := group.LogCurName
				if newName := cmdArgs.GetArgN(2); newName != "" {
					logName = newName
				}

				if logName == "" {
					text := DiceFormatTmpl(ctx, "日志:记录_取出_未指定记录")
					ReplyToSenderRaw(ctx, msg, text, "skip")
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				getAndUpload(group.GroupID, logName)
				return CmdExecuteResult{Matched: true, Solved: true}
			} else if cmdArgs.IsArgEqual(1, "end") {
				if group.LogCurName == "" {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "日志:记录_关闭_失败"))
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				lines, _ := model.LogLinesCountGet(ctx.Dice.DBLogs, group.GroupID, group.LogCurName)
				VarSetValueInt64(ctx, "$t当前记录条数", lines)
				VarSetValueStr(ctx, "$t记录名称", group.LogCurName)
				text := DiceFormatTmpl(ctx, "日志:记录_结束")
				// Note: 2024-02-28 经过讨论，日志在 off 的情况下 end 属于合理操作，这里不再检查是否开启
				// if !group.LogOn {
				//	 text = strings.TrimRightFunc(DiceFormatTmpl(ctx, "日志:记录_关闭_失败"), unicode.IsSpace) + "\n" + text
				// }
				ReplyToSender(ctx, msg, text)
				group.LogOn = false
				group.UpdatedAtTime = time.Now().Unix()

				time.Sleep(time.Duration(0.3 * float64(time.Second)))
				getAndUpload(group.GroupID, group.LogCurName)
				group.LogCurName = ""
				group.UpdatedAtTime = time.Now().Unix()
				return CmdExecuteResult{Matched: true, Solved: true}
			} else if cmdArgs.IsArgEqual(1, "halt") {
				if len(group.LogCurName) > 0 {
					lines, _ := model.LogLinesCountGet(ctx.Dice.DBLogs, group.GroupID, group.LogCurName)
					VarSetValueInt64(ctx, "$t当前记录条数", lines)
					VarSetValueStr(ctx, "$t记录名称", group.LogCurName)
				}
				text := DiceFormatTmpl(ctx, "日志:记录_结束")
				ReplyToSender(ctx, msg, text)
				group.LogOn = false
				group.LogCurName = ""
				group.UpdatedAtTime = time.Now().Unix()
				return CmdExecuteResult{Matched: true, Solved: true}
			} else if cmdArgs.IsArgEqual(1, "list") {
				groupID, requestForAnotherGroup := getSpecifiedGroupIfMaster(ctx, msg, cmdArgs)
				if requestForAnotherGroup && groupID == "" {
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				if groupID == "" {
					groupID = ctx.Group.GroupID
				}

				text := DiceFormatTmpl(ctx, "日志:记录_列出_导入语") + "\n"
				lst, err := model.LogGetList(ctx.Dice.DBLogs, groupID)
				if err == nil {
					for _, i := range lst {
						text += "- " + i + "\n"
					}
					if len(lst) == 0 {
						text += "暂无记录"
					}
				} else {
					text += "获取记录出错: " + err.Error()
				}
				ReplyToSender(ctx, msg, text)
				return CmdExecuteResult{Matched: true, Solved: true}
			} else if cmdArgs.IsArgEqual(1, "new") {
				if ctx.IsPrivate {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_私聊不可用"))
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				name := cmdArgs.GetArgN(2)
				if group.LogCurName != "" && name == "" {
					VarSetValueStr(ctx, "$t记录名称", group.LogCurName)
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "日志:记录_新建_失败_未结束的记录"))
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				if groupNotActiveCheck() {
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				if name == "" {
					name = time.Now().Format("2006_01_02_15_04_05")
				}
				if group.LogCurName != "" {
					VarSetValueInt64(ctx, "$t存在开启记录", 1)
				} else {
					VarSetValueInt64(ctx, "$t存在开启记录", 0)
				}
				VarSetValueStr(ctx, "$t上一记录名称", group.LogCurName)
				VarSetValueStr(ctx, "$t记录名称", name)
				group.LogCurName = name
				group.LogOn = true
				group.UpdatedAtTime = time.Now().Unix()

				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "日志:记录_新建"))
				return CmdExecuteResult{Matched: true, Solved: true}
			} else if cmdArgs.IsArgEqual(1, "stat") {
				// group := ctx.Group
				_, name := getLogName(ctx, msg, cmdArgs, 2)
				items, err := model.LogGetAllLines(ctx.Dice.DBLogs, group.GroupID, name)
				if err == nil && len(items) > 0 {
					// showDetail := cmdArgs.GetKwarg("detail")
					// var showDetail *Kwarg
					showAll := cmdArgs.GetKwarg("all")

					/* if showDetail != nil { //nolint // 故意保留
						results := LogRollBriefDetail(items)

						if len(results) > 0 {
							ReplyToSender(ctx, msg, "统计结果如下:\n"+strings.Join(results, "\n"))
							return CmdExecuteResult{Matched: true, Solved: true}
						}
					} else */{
						isShowAll := showAll != nil
						text := LogRollBriefByPC(ctx, items, isShowAll, ctx.Player.Name)
						if text == "" {
							if isShowAll {
								ReplyToSender(ctx, msg, fmt.Sprintf("没有找到故事“%s”的检定记录", name))
							} else {
								ReplyToSender(ctx, msg, fmt.Sprintf("没有找到角色<%s>的任何记录\n若需查看全团，请在指令后加 --all", ctx.Player.Name))
							}
						} else {
							if !isShowAll {
								text += "\n\n若需查看全团，请在指令后加 --all"
							}
							ReplyToSender(ctx, msg, text)
						}
						return CmdExecuteResult{Matched: true, Solved: true}
					}
				}
				ReplyToSender(ctx, msg, "没有发现可供统计的信息，请确保记录名正确，且有进行骰点/检定行为")
				return CmdExecuteResult{Matched: true, Solved: true}
			} else if cmdArgs.IsArgEqual(1, "export") {
				logName := group.LogCurName
				if newName := cmdArgs.GetArgN(2); newName != "" {
					logName = newName
				}
				if logName == "" {
					ReplyToSenderRaw(ctx, msg, DiceFormatTmpl(ctx, "日志:记录_导出_未指定记录"), "skip")
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				now := carbon.Now()
				VarSetValueStr(ctx, "$t记录名", logName)
				VarSetValueStr(ctx, "$t日期", now.ToShortDateString())
				VarSetValueStr(ctx, "$t时间", now.ToShortTimeString())
				logFileNamePrefix := DiceFormatTmpl(ctx, "日志:记录_导出_文件名前缀")
				logFile, err := GetLogTxt(ctx, group.GroupID, logName, logFileNamePrefix)
				if err != nil {
					ReplyToSenderRaw(ctx, msg, err.Error(), "skip")
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				defer os.Remove(logFile.Name())

				var emails []string
				if len(cmdArgs.Args) > 2 {
					emails = cmdArgs.Args[2:]
					// 试图发送邮件
					dice := ctx.Session.Parent
					if dice.CanSendMail() {
						rightEmails := make([]string, 0, len(emails))
						emailExp := regexp.MustCompile(`.*@.*`)
						for _, email := range emails {
							if emailExp.MatchString(email) {
								rightEmails = append(rightEmails, email)
							}
						}
						if len(rightEmails) > 0 {
							emailMsg := DiceFormatTmpl(ctx, "日志:记录_导出_邮件附言")
							dice.SendMailRow(
								fmt.Sprintf("Seal 记录提取: %s", logFileNamePrefix),
								rightEmails,
								emailMsg,
								[]string{logFile.Name()},
							)
							text := DiceFormatTmpl(ctx, "日志:记录_导出_邮箱发送前缀") + strings.Join(rightEmails, "\n")
							ReplyToSenderRaw(ctx, msg, text, "skip")
							return CmdExecuteResult{Matched: true, Solved: true}
						}
						ReplyToSenderRaw(ctx, msg, DiceFormatTmpl(ctx, "日志:记录_导出_无格式有效邮箱"), "skip")
					}
					ReplyToSenderRaw(ctx, msg, DiceFormat(ctx, "{核心:骰子名字}未配置邮箱，将直接发送记录文件"), "skip")
				}

				var uri string
				if runtime.GOOS == "windows" {
					uri = "files:///" + logFile.Name()
				} else {
					uri = "files://" + logFile.Name()
				}
				SendFileToSenderRaw(ctx, msg, uri, "skip")
				VarSetValueStr(ctx, "$t文件名字", logFileNamePrefix)
				ReplyToSenderRaw(ctx, msg, DiceFormatTmpl(ctx, "日志:记录_导出_成功"), "skip")
				return CmdExecuteResult{Matched: true, Solved: true}
			} else {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
		},
	}

	helpStat := `.stat log [<日志名>] // 查看当前或指定日志的骰点统计
.stat log [<日志名>] --all // 查看全团
.stat help // 帮助
`
	cmdStat := &CmdItemInfo{
		Name:      "stat",
		ShortHelp: helpStat,
		Help:      "查看统计:\n" + helpStat,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			val := cmdArgs.GetArgN(1)
			switch strings.ToLower(val) {
			case "log":
				group := ctx.Group
				_, name := getLogName(ctx, msg, cmdArgs, 2)
				items, err := model.LogGetAllLines(ctx.Dice.DBLogs, group.GroupID, name)
				if err == nil && len(items) > 0 {
					// showDetail := cmdArgs.GetKwarg("detail")
					// var showDetail *Kwarg
					showAll := cmdArgs.GetKwarg("all")

					/* if showDetail != nil { //nolint // 故意保留
						results := LogRollBriefDetail(items)

						if len(results) > 0 {
							ReplyToSender(ctx, msg, "统计结果如下:\n"+strings.Join(results, "\n"))
							return CmdExecuteResult{Matched: true, Solved: true}
						}
					} else */{
						isShowAll := showAll != nil
						text := LogRollBriefByPC(ctx, items, isShowAll, ctx.Player.Name)
						if text == "" {
							if isShowAll {
								ReplyToSender(ctx, msg, fmt.Sprintf("没有找到故事“%s”的检定记录", name))
							} else {
								ReplyToSender(ctx, msg, fmt.Sprintf("没有找到角色<%s>的任何记录\n若需查看全团，请在指令后加 --all", ctx.Player.Name))
							}
						} else {
							if !isShowAll {
								text += "\n\n若需查看全团，请在指令后加 --all"
							}
							ReplyToSender(ctx, msg, text)
						}
						return CmdExecuteResult{Matched: true, Solved: true}
					}
				}
				if err != nil || len(items) == 0 {
					ReplyToSender(ctx, msg, "没有发现可供统计的信息，请确保记录名正确，且有进行骰点/检定行为")
				}
			default:
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	helpOb := `.ob // 进入ob模式
.ob exit // 退出ob
`
	cmdOb := &CmdItemInfo{
		Name:          "ob",
		ShortHelp:     helpOb,
		Help:          "观众指令:\n" + helpOb,
		AllowDelegate: true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			ctx.DelegateText = fmt.Sprintf("由<%s>操作:\n", ctx.Player.Name)
			mctx := GetCtxProxyFirst(ctx, cmdArgs)
			subcommand := cmdArgs.GetArgN(1)

			c := ctx
			if mctx != nil && mctx.Player.UserID != ctx.Player.UserID {
				if ctx.PrivilegeLevel < 50 && subcommand != "help" {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "通用:提示_无权限_非master/管理"))
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				c = mctx
			}

			switch strings.ToLower(subcommand) {
			case "help":
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			case "exit":
				if strings.HasPrefix(strings.ToLower(c.Player.Name), "ob") {
					c.Player.Name = c.Player.Name[len("ob"):]
					c.Player.UpdatedAtTime = time.Now().Unix()
				}
				c.EndPoint.Adapter.SetGroupCardName(c, c.Player.Name)
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "日志:OB_关闭"))
			default:
				if !strings.HasPrefix(strings.ToLower(c.Player.Name), "ob") {
					c.Player.Name = "ob" + c.Player.Name
					c.Player.UpdatedAtTime = time.Now().Unix()
				}
				c.EndPoint.Adapter.SetGroupCardName(c, c.Player.Name)
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "日志:OB_开启"))
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	helpSn := `.sn coc // 自动设置coc名片
.sn dnd // 自动设置dnd名片
.sn none // 设置为空白格式
.sn off // 取消自动设置
`
	cmdSn := &CmdItemInfo{
		Name:               "sn",
		ShortHelp:          helpSn,
		Help:               "跑团名片(需要管理权限):\n" + helpSn,
		CheckCurrentBotOn:  true,
		CheckMentionOthers: true,
		HelpFunc: func(isShort bool) string {
			text := ""
			self.GameSystemMap.Range(func(key string, value *GameSystemTemplate) bool {
				for k, v := range value.NameTemplate {
					text += fmt.Sprintf(".sn %s // %s\n", k, v.HelpText)
				}
				return true
			})
			text += ".sn expr {$t玩家_RAW} HP{hp}/{hpmax} // 自设格式\n" +
				".sn none // 设置为空白格式\n" +
				".sn off // 取消自动设置"
			if isShort {
				return text
			}
			return "跑团名片(需要管理权限):\n" + text
		},
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			val := cmdArgs.GetArgN(1)

			handleOverlong := func(ctx *MsgContext, msg *Message, card string) CmdExecuteResult {
				ReplyToSender(ctx, msg, fmt.Sprintf(
					"尝试将群名片修改为 %q 失败，名片长度超过限制。\n请尝试缩短角色名或使用 .sn expr 自定义名片格式。",
					card,
				))
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			switch strings.ToLower(val) {
			case "help":
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			case "coc", "coc7":
				ctx.Player.AutoSetNameTemplate = "{$t玩家_RAW} SAN{理智} HP{生命值}/{生命值上限} DEX{敏捷}"
				ctx.Player.UpdatedAtTime = time.Now().Unix()
				text, err := SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
				if errors.Is(err, ErrGroupCardOverlong) {
					return handleOverlong(ctx, msg, text)
				}
				VarSetValueStr(ctx, "$t名片格式", val)
				VarSetValueStr(ctx, "$t名片预览", text)
				// 玩家 SAN60 HP10/10 DEX65
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "日志:名片_自动设置"))
			case "dnd", "dnd5e":
				// PW{pw}
				ctx.Player.AutoSetNameTemplate = "{$t玩家_RAW} HP{hp}/{hpmax} AC{ac} DC{dc} PP{pp}"
				ctx.Player.UpdatedAtTime = time.Now().Unix()
				text, err := SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
				if errors.Is(err, ErrGroupCardOverlong) {
					return handleOverlong(ctx, msg, text)
				}
				VarSetValueStr(ctx, "$t名片格式", val)
				VarSetValueStr(ctx, "$t名片预览", text)
				// 玩家 HP10/10 AC15 DC15 PW10
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "日志:名片_自动设置"))
			case "none":
				ctx.Player.AutoSetNameTemplate = "{$t玩家_RAW}"
				ctx.Player.UpdatedAtTime = time.Now().Unix()
				text, err := SetPlayerGroupCardByTemplate(ctx, "{$t玩家_RAW}")
				if errors.Is(err, ErrGroupCardOverlong) { // 大约不至于会走到这里，但是为了统一也这样写了
					return handleOverlong(ctx, msg, text)
				}
				VarSetValueStr(ctx, "$t名片格式", "空白")
				VarSetValueStr(ctx, "$t名片预览", text)
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "日志:名片_自动设置"))
			case "off", "cancel":
				_, _ = SetPlayerGroupCardByTemplate(ctx, "{$t玩家_RAW}")
				ctx.Player.AutoSetNameTemplate = ""
				ctx.Player.UpdatedAtTime = time.Now().Unix()
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "日志:名片_取消设置"))
			case "expr":
				t := cmdArgs.GetRestArgsFrom(2)
				if len(t) > 80 {
					t = t[:80]
				}
				if t == "" {
					_, _ = SetPlayerGroupCardByTemplate(ctx, "{$t玩家_RAW}")
					ctx.Player.AutoSetNameTemplate = ""
					ctx.Player.UpdatedAtTime = time.Now().Unix()
					ReplyToSender(ctx, msg, "玩家自设内容为空，已自动关闭此功能")
				} else {
					last := ctx.Player.AutoSetNameTemplate
					ctx.Player.AutoSetNameTemplate = t
					text, err := SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
					if err != nil && !errors.Is(err, ErrGroupCardOverlong) {
						ctx.Player.AutoSetNameTemplate = last
						ReplyToSender(ctx, msg, "玩家自设sn格式错误，已自动还原之前模板")
					} else if errors.Is(err, ErrGroupCardOverlong) {
						return handleOverlong(ctx, msg, text)
					} else {
						ctx.Player.UpdatedAtTime = time.Now().Unix()
						VarSetValueStr(ctx, "$t名片格式", "玩家自设")
						VarSetValueStr(ctx, "$t名片预览", text)
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "日志:名片_自动设置"))
					}
				}
			default:
				ok := false
				ctx.Dice.GameSystemMap.Range(func(key string, value *GameSystemTemplate) bool {
					var t NameTemplateItem
					var exists bool

					// 先检查绝对匹配, 不存在则检查小写匹配
					if t, exists = value.NameTemplate[val]; !exists {
						t, exists = value.NameTemplate[strings.ToLower(val)]
					}

					if !exists {
						return true
					}

					text, _ := SetPlayerGroupCardByTemplate(ctx, t.Template)
					ctx.Player.AutoSetNameTemplate = t.Template
					VarSetValueStr(ctx, "$t名片格式", val)
					VarSetValueStr(ctx, "$t名片预览", text)
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "日志:名片_自动设置"))
					ok = true
					return false
				})

				if ok {
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	self.RegisterExtension(&ExtInfo{
		Name:       "log",
		Version:    "1.0.1",
		Brief:      "跑团辅助扩展，提供日志、染色等功能",
		Author:     "木落",
		AutoActive: true,
		Official:   true,
		OnLoad: func() {
			_ = os.MkdirAll(filepath.Join(self.BaseConfig.DataDir, "log-exports"), 0o755)
		},
		OnMessageSend: func(ctx *MsgContext, msg *Message, flag string) {
			// 记录骰子发言
			if flag == "skip" {
				return
			}
			privateCommandListenCheck()
			if msg.MessageType == "private" && ctx.CommandHideFlag != "" {
				if _, exists := privateCommandListen[ctx.CommandID]; exists {
					session := ctx.Session
					// TODO： 这里的OK被忽略了，没问题？
					groupInfo, ok := session.ServiceAtNew.Load(ctx.CommandHideFlag)
					if !ok {
						ctx.Dice.Logger.Warn("ServiceAtNew ext_log加载groupInfo异常")
						return
					}
					a := model.LogOneItem{
						Nickname:    ctx.EndPoint.Nickname,
						IMUserID:    UserIDExtract(ctx.EndPoint.UserID),
						UniformID:   ctx.EndPoint.UserID,
						Time:        time.Now().Unix(),
						Message:     msg.Message,
						IsDice:      true,
						CommandID:   ctx.CommandID,
						CommandInfo: ctx.CommandInfo,
					}

					LogAppend(ctx, groupInfo.GroupID, groupInfo.LogCurName, &a)
				}
			}

			if IsCurGroupBotOnByID(ctx.Session, ctx.EndPoint, msg.MessageType, msg.GroupID) {
				session := ctx.Session
				groupInfo, ok := session.ServiceAtNew.Load(msg.GroupID)
				if !ok {
					ctx.Dice.Logger.Warn("ServiceAtNew ext_log加载groupInfo异常")
					return
				}
				if groupInfo.LogOn {
					// <2022-02-15 09:54:14.0> [摸鱼king]: 有的 但我不知道
					if ctx.CommandHideFlag != "" {
						// 记录当前指令和时间
						privateCommandListen[ctx.CommandID] = time.Now().Unix()
					}

					a := model.LogOneItem{
						Nickname:    ctx.EndPoint.Nickname,
						IMUserID:    UserIDExtract(ctx.EndPoint.UserID),
						UniformID:   ctx.EndPoint.UserID,
						Time:        time.Now().Unix(),
						Message:     msg.Message,
						IsDice:      true,
						CommandID:   ctx.CommandID,
						CommandInfo: ctx.CommandInfo,
					}
					LogAppend(ctx, groupInfo.GroupID, groupInfo.LogCurName, &a)
				}
			}
		},
		OnMessageReceived: func(ctx *MsgContext, msg *Message) {
			// 处理日志
			if ctx.Group != nil {
				if ctx.Group.LogOn {
					// 去重，用于同群多骰情况
					if !groupMsgInfoCheckOk(msg.RawID) {
						return
					}
					groupMsgInfoSet(msg.RawID)

					// <2022-02-15 09:54:14.0> [摸鱼king]: 有的 但我不知道
					a := model.LogOneItem{
						Nickname:  ctx.Player.Name,
						IMUserID:  UserIDExtract(ctx.Player.UserID),
						UniformID: ctx.Player.UserID,
						Time:      msg.Time,
						Message:   msg.Message,
						IsDice:    false,
						CommandID: ctx.CommandID,
						RawMsgID:  msg.RawID,
					}

					LogAppend(ctx, ctx.Group.GroupID, ctx.Group.LogCurName, &a)
				}
			}
		},
		OnMessageDeleted: func(ctx *MsgContext, msg *Message) {
			if ctx.Group != nil {
				if ctx.Group.LogOn {
					LogDeleteByID(ctx, ctx.Group.GroupID, ctx.Group.LogCurName, msg.RawID)
					// ctx.Session.Parent.Logger.Infof("删除日志 %s %s", ctx.Group.GroupId, msg.RawId.(string))
				}
			}
		},
		OnMessageEdit: func(ctx *MsgContext, msg *Message) {
			if ctx.Group == nil {
				return
			}

			if ctx.Group.LogOn {
				LogEditByID(ctx, ctx.Group.GroupID, ctx.Group.LogCurName, msg.Message, msg.RawID)
			}
		},
		GetDescText: GetExtensionDesc,
		CmdMap: CmdMapCls{
			"log":  cmdLog,
			"stat": cmdStat,
			"hiy":  cmdStat,
			"ob":   cmdOb,
			"sn":   cmdSn,
		},
	})
}

func getSpecifiedGroupIfMaster(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) (groupID string, requestForAnotherGroup bool) {
	if data := cmdArgs.GetArgN(2); data != "" {
		if ctx.PrivilegeLevel < 100 {
			ReplyToSender(ctx, msg, "你并非Master，请检查指令输入是否正确")
			return "", true
		}

		var prefix string
		if ctx.EndPoint.Platform == "QQ" {
			prefix = "QQ-Group"
		}
		if !strings.HasPrefix(data, prefix) {
			data = prefix + ":" + data
		}

		// _newGroup := ctx.Session.ServiceAtNew[data]
		// if _newGroup == nil {
		// 	ReplyToSender(ctx, msg, "找不到指定的群组，请输入正确群号。如在非QQ平台取log，群号请写 QQ-Group:12345")
		// 	return nil, true
		// }
		return data, true
	}
	// 对应的组，是否存在第二个参数
	return "", false
}

func FilenameReplace(name string) string {
	re := regexp.MustCompile(`[/:\*\?"<>\|\\]`)
	return re.ReplaceAllString(name, "")
}

func LogAppend(ctx *MsgContext, groupID string, logName string, logItem *model.LogOneItem) bool {
	ok := model.LogAppend(ctx.Dice.DBLogs, groupID, logName, logItem)
	if ok {
		if size, okCount := model.LogLinesCountGet(ctx.Dice.DBLogs, groupID, logName); okCount {
			// 默认每记录500条发出提示
			if ctx.Dice.LogSizeNoticeEnable {
				if ctx.Dice.LogSizeNoticeCount == 0 {
					ctx.Dice.LogSizeNoticeCount = 500
				}
				if size > 0 && int(size)%ctx.Dice.LogSizeNoticeCount == 0 {
					VarSetValueInt64(ctx, "$t条数", size)
					text := DiceFormatTmpl(ctx, "日志:记录_条数提醒")
					// text := fmt.Sprintf("提示: 当前故事的文本已经记录了 %d 条", size)
					ReplyToSenderRaw(ctx, &Message{MessageType: "group", GroupID: groupID}, text, "skip")
				}
			}
		}
	}
	return ok
}

func LogDeleteByID(ctx *MsgContext, groupID string, logName string, messageID interface{}) bool {
	err := model.LogMarkDeleteByMsgID(ctx.Dice.DBLogs, groupID, logName, messageID)
	if err != nil {
		ctx.Dice.Logger.Error("LogDeleteById:", zap.Error(err))
		return false
	}
	return true
}

// LogEditByID finds the log item under logName with messageID and replace it with content.
// If the log item cannot be found or an error happens, it returns false.
func LogEditByID(ctx *MsgContext, groupID, logName, content string, messageID interface{}) bool {
	err := model.LogEditByMsgID(ctx.Dice.DBLogs, groupID, logName, content, messageID)
	if err != nil {
		ctx.Dice.Logger.Error("LogEditByID:", zap.Error(err))
		return false
	}
	return true
}

func GetLogTxt(ctx *MsgContext, groupID string, logName string, fileNamePrefix string) (*os.File, error) {
	tempLog, err := os.CreateTemp("", fmt.Sprintf(
		"%s(*).txt",
		utils.FilenameClean(fileNamePrefix),
	))
	if err != nil {
		return nil, errors.New("log导出出现未知错误")
	}
	defer func() {
		if err != nil {
			_ = os.Remove(tempLog.Name())
		}
	}()

	lines, err := model.LogGetAllLines(ctx.Dice.DBLogs, groupID, logName)
	if len(lines) == 0 {
		err = errors.New("此log不存在，或条目数为空，名字是否正确？")
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	for _, line := range lines {
		timeTxt := time.Unix(line.Time, 0).Format("2006-01-02 15:04:05")
		text := fmt.Sprintf("%s(%v) %s\n%s\n\n", line.Nickname, line.IMUserID, timeTxt, line.Message)
		_, _ = tempLog.WriteString(text)
	}
	return tempLog, nil
}

func LogSendToBackend(ctx *MsgContext, groupID string, logName string) (bool, string, error) {
	dice := ctx.Dice
	dirPath := filepath.Join(dice.BaseConfig.DataDir, "log-exports")

	var sealBackends []string
	for _, sealBackend := range BackendUrls {
		sealBackends = append(sealBackends, sealBackend+"/dice/api/log")
	}

	uploadCtx := storylog.UploadEnv{
		Dir:      dirPath,
		Db:       dice.DBLogs,
		Log:      dice.Logger,
		Backends: sealBackends,

		LogName:   logName,
		UniformID: ctx.EndPoint.UserID,
		GroupID:   groupID,
	}
	uploadCtx.Version = storylog.StoryVersionV1

	var unofficial bool
	if dice.AdvancedConfig.Enable && dice.AdvancedConfig.StoryLogBackendUrl != "" {
		unofficial = true
		uploadCtx.Backends = []string{dice.AdvancedConfig.StoryLogBackendUrl}
		uploadCtx.Token = dice.AdvancedConfig.StoryLogBackendToken

		// 现在只有一个版本的 api，未来这里根据 advancedConfig.StoryLogBackendToken 切换
		uploadCtx.Version = storylog.StoryVersionV1
	}

	url, err := storylog.Upload(uploadCtx)
	if err != nil {
		return unofficial, "", err
	}
	if len(url) == 0 {
		return unofficial, "", errors.New("上传 log 到服务器失败，未能获取染色器链接")
	}
	return unofficial, url, nil
}

// LogRollBriefByPC 根据log生成骰点简报
func LogRollBriefByPC(ctx *MsgContext, items []*model.LogOneItem, showAll bool, name string) string {
	pcInfo := map[string]map[string]int{}
	// 加载同义词
	tmpl := ctx.Group.GetCharTemplate(ctx.Dice)

	getName := func(s string) string {
		re := regexp.MustCompile(`^([^\d\s]+)(\d+)?$`)
		m := re.FindStringSubmatch(s)
		if len(m) > 0 {
			s = m[1]
		}

		return tmpl.GetAlias(s)
	}

	for _, i := range items {
		if i.CommandInfo != nil { //nolint:nestif
			info, _ := i.CommandInfo.(map[string]interface{})
			// t := time.Unix(i.Time, 0).Format("[04:05]")

			setupName := func(name string) {
				if _, exists := pcInfo[name]; !exists {
					pcInfo[name] = map[string]int{}
				}
			}

			if info["rule"] == nil {
				switch info["cmd"] {
				case "roll":
					items, ok2 := info["items"].([]interface{})
					if !ok2 {
						continue
					}
					nickname := fmt.Sprintf("%v", info["pcName"])
					setupName(nickname)
					pcInfo[nickname]["骰点"] += len(items)
				}
				continue
			}
			if info["rule"] == "coc7" {
				switch info["cmd"] {
				case "ra":
					items, ok2 := info["items"].([]interface{})
					if !ok2 {
						continue
					}
					nickname := fmt.Sprintf("%v", info["pcName"])
					setupName(nickname)

					for _, _j := range items {
						j, ok2 := _j.(map[string]interface{})
						if !ok2 {
							continue
						}

						rank := int(j["rank"].(float64))
						attr := getName(fmt.Sprintf("%v", j["expr2"]))
						if rank > 0 {
							key := fmt.Sprintf("%v:%v", attr, "成功")
							pcInfo[nickname][key]++
						} else if rank < 0 {
							key := fmt.Sprintf("%v:%v", attr, "失败")
							pcInfo[nickname][key]++
						}
					}
					continue
				case "sc":
					items, ok2 := info["items"].([]interface{})
					if !ok2 {
						continue
					}
					nickname := fmt.Sprintf("%v", info["pcName"])
					setupName(nickname)

					for _, _j := range items {
						j, ok2 := _j.(map[string]interface{})
						if !ok2 {
							continue
						}

						rank := int(j["rank"].(float64))
						if rank > 0 {
							key := fmt.Sprintf("%v:%v", "理智", "成功")
							pcInfo[nickname][key]++
						} else if rank < 0 {
							key := fmt.Sprintf("%v:%v", "理智", "失败")
							pcInfo[nickname][key]++
						}

						// 如果没有旧值，弄一个
						key := "理智:旧值"
						if pcInfo[nickname][key] == 0 {
							pcInfo[nickname][key] = int(j["sanOld"].(float64))
						}

						key2 := "理智:新值"
						// if pcInfo[nickname][key2] == 0 {
						pcInfo[nickname][key2] = int(j["sanNew"].(float64))
						// }
					}
					continue
				case "st":
					items, ok2 := info["items"].([]any)
					if !ok2 {
						continue
					}
					for _, _j := range items {
						j, ok2 := _j.(map[string]any)
						if !ok2 {
							continue
						}
						nickname := fmt.Sprintf("%v", info["pcName"])
						setupName(nickname)

						if j["type"] == "mod" {
							readNum := func(dataKey, key string) {
								if val, ok := j[dataKey].(float64); ok {
									// 旧版本兼容，float64是因为json unmarshal默认就是这个
									pcInfo[nickname][key] = int(val)
								} else {
									// TODO: 处理的不是很好，这里后续大段代码依赖了值为int的情况，但是现在实际可以为任何类型，只是不常用
									b, _ := json.Marshal(j[dataKey])
									var v ds.VMValue
									if err := v.UnmarshalJSON(b); err == nil {
										if v.TypeId == ds.VMTypeInt {
											i, _ := v.ReadInt()
											pcInfo[nickname][key] = int(i)
										}
									}
								}
							}

							attr := getName(j["attr"].(string))
							// 如果没有旧值，弄一个
							key := fmt.Sprintf("%v:旧值", attr)
							if pcInfo[nickname][key] == 0 {
								readNum("valOld", key)
							}

							key2 := fmt.Sprintf("%v:新值", attr)
							// if pcInfo[nickname][key2] == 0 {
							readNum("valNew", key2)
							// }
						}
					}
					continue
				}
			}
		}
	}

	if !showAll {
		pcInfo2 := map[string]map[string]int{}
		if pcInfo[name] != nil {
			pcInfo2[name] = pcInfo[name]
		}
		pcInfo = pcInfo2
	}

	texts := ""
	for k, v := range pcInfo {
		if len(v) == 0 {
			continue
		}
		texts += fmt.Sprintf("<%v>当前团内检定情况:\n", k)
		success := map[string]int{}
		failed := map[string]int{}
		var others []string

		oldVal := map[string]int{}
		newVal := map[string]int{}

		for k2, v2 := range v {
			if strings.HasSuffix(k2, ":成功") {
				success[k2] = v2
			} else if strings.HasSuffix(k2, ":失败") {
				failed[k2] = v2
			} else if strings.HasSuffix(k2, ":旧值") {
				oldVal[k2[:len(k2)-len(":旧值")]] = v2
			} else if strings.HasSuffix(k2, ":新值") {
				newVal[k2[:len(k2)-len(":新值")]] = v2
			} else {
				others = append(others, k2)
			}
		}

		// 排序: 一次挑选一个最大的，直到结束
		doSort := func(m map[string]int) []string {
			var ret []string
			for len(m) > 0 {
				val := -1
				theKey := ""
				for k2, v2 := range m {
					if v2 > val {
						theKey = k2
						val = v2
					}
				}
				ret = append(ret, theKey)
				delete(m, theKey)
			}
			return ret
		}
		successList := doSort(success)
		failedList := doSort(failed)

		if len(successList) > 0 {
			text := "成功: "
			for _, j := range successList {
				text += fmt.Sprintf("%v%d ", j[:len(j)-len(":成功")], v[j])
			}
			texts += strings.TrimSpace(text) + "\n"
		}

		if len(failedList) > 0 {
			text := "失败: "
			for _, j := range failedList {
				text += fmt.Sprintf("%v%d ", j[:len(j)-len(":失败")], v[j])
			}
			texts += strings.TrimSpace(text) + "\n"
		}

		if len(oldVal) > 0 {
			text := ""
			for k2, v2 := range oldVal {
				text += fmt.Sprintf("%v[%v➯%v] ", k2, v2, newVal[k2])
			}
			texts += "属性: " + strings.TrimSpace(text) + "\n"
		}

		if len(others) > 0 {
			text := "其他: "
			for _, j := range others {
				text += fmt.Sprintf("%v%d ", j, v[j])
			}
			texts += strings.TrimSpace(text) + "\n"
		}
		texts += "\n"
	}
	return strings.TrimSpace(texts)
}

// LogRollBriefDetail 根据log生成骰点简报
func LogRollBriefDetail(items []*model.LogOneItem) []string {
	var texts []string
	for _, i := range items {
		if i.CommandInfo != nil { //nolint:nestif
			info, _ := i.CommandInfo.(map[string]interface{})
			t := time.Unix(i.Time, 0).Format("[04:05]")

			if info["rule"] == nil {
				switch info["cmd"] {
				case "roll":
					// [03分20秒] 木落 骰点d100，出目15
					items, ok2 := info["items"].([]interface{})
					if !ok2 {
						continue
					}
					for _, _j := range items {
						j, ok2 := _j.(map[string]interface{})
						if !ok2 {
							continue
						}

						reasonText := ""
						if j["reason"] != nil {
							reasonText = fmt.Sprintf(" 原因:%v", j["reason"])
						}

						texts = append(texts, fmt.Sprintf("%v %s 骰点%v 出目%v%v", t, info["pcName"], j["expr"], j["result"], reasonText))
					}
				}
				continue
			}
			if info["rule"] == "coc7" {
				switch info["cmd"] {
				case "ra":
					items, ok2 := info["items"].([]interface{})
					if !ok2 {
						continue
					}
					for _, _j := range items {
						j, ok2 := _j.(map[string]interface{})
						if !ok2 {
							continue
						}

						// [18分60秒] 木落 "力量50"检定，出目39/50，成功
						texts = append(texts, fmt.Sprintf("%v %s \"%s\"检定 出目%v/%v，%v", t, info["pcName"], j["expr2"], j["checkVal"], j["attrVal"], SimpleCocSuccessRankToText[int(j["rank"].(float64))]))
					}
					continue
				case "sc":
					items, ok2 := info["items"].([]interface{})
					if !ok2 {
						continue
					}
					for _, _j := range items {
						j, ok2 := _j.(map[string]interface{})
						if !ok2 {
							continue
						}

						// [18分60秒] 木落 理智检定[d100 1 2]，出目15/60，失败。理智39➯38
						texts = append(texts, fmt.Sprintf("%v %s 理智检定%v 出目%v/%v，%v。理智%v➯%v",
							t, info["pcName"], j["exprs"], j["checkVal"], j["sanOld"],
							SimpleCocSuccessRankToText[int(j["rank"].(float64))], j["sanOld"], j["sanNew"]))
					}
					continue
				case "st":
					items, ok2 := info["items"].([]interface{})
					if !ok2 {
						continue
					}
					for _, _j := range items {
						j, ok2 := _j.(map[string]interface{})
						if !ok2 {
							continue
						}

						if j["type"] == "mod" {
							// [18分60秒] 木落 hp变更1d4，39➯38
							texts = append(texts, fmt.Sprintf("%v %s %v变更%v，%v➯%v",
								t, info["pcName"], j["attr"], j["modExpr"], j["valOld"], j["valNew"]))
						}
					}
					continue
				}
			}

			if info["rule"] == "dnd5e" {
				switch info["cmd"] {
				case "rc":
					items, ok2 := info["items"].([]interface{})
					if !ok2 {
						continue
					}
					for _, _j := range items {
						j, ok2 := _j.(map[string]interface{})
						if !ok2 {
							continue
						}

						// [18分60秒] 木落 力量检定，出目24
						texts = append(texts, fmt.Sprintf("%v %s %s检定 出目%v", t, info["pcName"], j["reason"], j["result"]))
					}
					continue
				case "st":
					items, ok2 := info["items"].([]interface{})
					if !ok2 {
						continue
					}
					for _, _j := range items {
						j, ok2 := _j.(map[string]interface{})
						if !ok2 {
							continue
						}

						if j["type"] == "mod" {
							// [18分60秒] 木落 hp变更1d4，39➯38
							texts = append(texts, fmt.Sprintf("%v %s %v变更%v，%v➯%v",
								t, info["pcName"], j["attr"], j["modExpr"], j["valOld"], j["valNew"]))
						}
					}
					continue
				}
			}

			texts = append(texts, fmt.Sprintf("%v\n", i.CommandInfo))
		}
	}
	return texts
}
