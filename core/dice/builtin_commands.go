package dice

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/golang-module/carbon"

	"github.com/fy0/lockfree"
	"github.com/juliangruber/go-intersect"
	cp "github.com/otiai10/copy"
	ds "github.com/sealdice/dicescript"
)

/** 这几条指令不能移除 */
func (d *Dice) registerCoreCommands() {
	helpForBlack := ".ban add user <帐号> (<原因>) //添加个人\n" +
		".ban add group <群号> (<原因>) //添加群组\n" +
		".ban add <统一ID>\n" +
		".ban rm user <帐号> //解黑/移出信任\n" +
		".ban rm group <群号>\n" +
		".ban rm <统一ID> //同上\n" +
		".ban list // 展示列表\n" +
		".ban list ban/warn/trust //只显示被禁用/被警告/信任用户\n" +
		".ban trust <统一ID> //添加信任\n" +
		".ban query <统一ID> //查看指定用户拉黑情况\n" +
		".ban help //查看帮助\n" +
		"// 统一ID示例: QQ:12345、QQ-Group:12345"
	cmdBlack := &CmdItemInfo{
		Name:      "ban",
		ShortHelp: helpForBlack,
		Help:      "黑名单指令:\n" + helpForBlack,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			cmdArgs.ChopPrefixToArgsWith("add", "rm", "del", "list", "show", "find", "trust")
			if ctx.PrivilegeLevel < 100 {
				ReplyToSender(ctx, msg, "你不具备Master权限")
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			getID := func() string {
				if cmdArgs.IsArgEqual(2, "user") || cmdArgs.IsArgEqual(2, "group") {
					id := cmdArgs.GetArgN(3)
					if id == "" {
						return ""
					}

					isGroup := cmdArgs.IsArgEqual(2, "group")
					return FormatDiceID(ctx, id, isGroup)
				}

				arg := cmdArgs.GetArgN(2)
				if !strings.Contains(arg, ":") {
					return ""
				}
				return arg
			}

			var val = cmdArgs.GetArgN(1)
			var uid string
			switch strings.ToLower(val) {
			case "add":
				uid = getID()
				if uid == "" {
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}
				reason := cmdArgs.GetArgN(4)
				if reason == "" {
					reason = "骰主指令"
				}
				d.BanList.AddScoreBase(uid, d.BanList.ThresholdBan, "骰主指令", reason, ctx)
				ReplyToSender(ctx, msg, fmt.Sprintf("已将用户/群组 %s 加入黑名单，原因: %s", uid, reason))
			case "rm", "del":
				uid = getID()
				if uid == "" {
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}

				item, ok := d.BanList.GetByID(uid)
				if !ok || (item.Rank != BanRankBanned && item.Rank != BanRankTrusted && item.Rank != BanRankWarn) {
					ReplyToSender(ctx, msg, "找不到用户/群组")
					break
				}

				ReplyToSender(ctx, msg, fmt.Sprintf("已将用户/群组 %s 移出%s列表", uid, BanRankText[item.Rank]))
				item.Score = 0
				item.Rank = BanRankNormal
			case "trust":
				uid = cmdArgs.GetArgN(2)
				if !strings.Contains(uid, ":") {
					// 如果不是这种格式，那么放弃
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}

				d.BanList.SetTrustByID(uid, "骰主指令", "骰主指令")
				ReplyToSender(ctx, msg, fmt.Sprintf("已将用户/群组 %s 加入信任列表", uid))
			case "list", "show":
				// ban/warn/trust
				var extra, text string

				extra = cmdArgs.GetArgN(2)
				d.BanList.Map.Range(func(k string, v *BanListInfoItem) bool {
					if v.Rank == BanRankNormal {
						return true
					}

					match := (extra == "trust" && v.Rank == BanRankTrusted) ||
						(extra == "ban" && v.Rank == BanRankBanned) ||
						(extra == "warn" && v.Rank == BanRankWarn)
					if extra == "" || match {
						text += v.toText(d) + "\n"
					}
					return true
				})

				if text == "" {
					text = "当前名单:\n<无内容>"
				} else {
					text = "当前名单:\n" + text
				}
				ReplyToSender(ctx, msg, text)
			case "query":
				var targetID = cmdArgs.GetArgN(2)
				if targetID == "" {
					ReplyToSender(ctx, msg, "未指定要查询的对象！")
					break
				}

				v, exists := d.BanList.Map.Load(targetID)
				if !exists {
					ReplyToSender(ctx, msg, fmt.Sprintf("所查询的<%s>情况：正常(0)", targetID))
					break
				}

				var text = fmt.Sprintf("所查询的<%s>情况：", targetID)
				switch v.Rank {
				case BanRankBanned:
					text += "禁止(-30)"
				case BanRankWarn:
					text += "警告(-10)"
				case BanRankTrusted:
					text += "信任(30)"
				default:
					text += "正常(0)"
				}
				for i, reason := range v.Reasons {
					text += fmt.Sprintf(
						"\n%s在「%s」，原因：%s",
						carbon.CreateFromTimestamp(v.Times[i]).ToDateTimeString(),
						v.Places[i],
						reason,
					)
				}
				ReplyToSender(ctx, msg, text)
			default:
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
	d.CmdMap["black"] = cmdBlack
	d.CmdMap["ban"] = cmdBlack

	helpForFind := ".find/查询 <关键字> // 查找文档。关键字可以多个，用空格分割\n" +
		".find #<分组> <关键字> // 查找指定分组下的文档。关键字可以多个，用空格分割\n" +
		".find <数字ID> // 显示该ID的词条\n" +
		".find --rand // 显示随机词条\n" +
		".find <关键字> --num=10 // 需要更多结果\n" +
		".find config --group // 查看当前默认搜索分组\n" +
		".find config --group=<分组> // 设置当前默认搜索分组\n" +
		".find config --groupclr // 清空当前默认搜索分组"
	cmdFind := &CmdItemInfo{
		Name:      "find",
		ShortHelp: helpForFind,
		Help:      "查询指令，通常使用全文搜索(x86版)或快速查询(arm, 移动版):\n" + helpForFind,
		// 写不下了
		// + "\n注: 默认搭载的《怪物之锤查询》来自蜜瓜包、October整理\n默认搭载的COC《魔法大典》来自魔骨，NULL，Dr.Amber整理\n默认搭载的DND系列文档来自DicePP项目"
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if cmdArgs.IsArgEqual(1, "help") {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			if d.Parent.IsHelpReloading {
				ReplyToSender(ctx, msg, "帮助文档正在重新装载，请稍后...")
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			if _config := cmdArgs.GetArgN(1); _config == "config" {
				oldDefault := ctx.Group.DefaultHelpGroup
				if cmdArgs.GetKwarg("groupclr") != nil {
					ctx.Group.SetDefaultHelpGroup("")
					if oldDefault != "" {
						ReplyToSender(ctx, msg, "已清空默认搜索分组，原分组为"+oldDefault)
					} else {
						ReplyToSender(ctx, msg, "未指定默认搜索分组")
					}
				} else if _defaultGroup := cmdArgs.GetKwarg("group"); _defaultGroup != nil {
					defaultGroup := _defaultGroup.Value
					if defaultGroup == "" {
						// 为查看默认分组
						if oldDefault != "" {
							ReplyToSender(ctx, msg, "当前默认搜索分组为"+oldDefault)
						} else {
							ReplyToSender(ctx, msg, "未指定默认搜索分组")
						}
					} else {
						// 为设置默认分组
						ctx.Group.SetDefaultHelpGroup(defaultGroup)
						if oldDefault != "" {
							ReplyToSender(ctx, msg, fmt.Sprintf("默认搜索分组由%s切换到%s", oldDefault, defaultGroup))
						} else {
							ReplyToSender(ctx, msg, "指定默认搜索分组为"+defaultGroup)
						}
					}
				} else {
					ReplyToSender(ctx, msg, "设置选项有误")
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			var (
				useGroupSearch bool
				group          string
			)
			if _group := cmdArgs.GetArgN(1); strings.HasPrefix(_group, "#") {
				useGroupSearch = true
				fakeGroup := strings.TrimPrefix(_group, "#")

				// 转换 group 别名
				if _g, ok := d.Parent.Help.GroupAliases[fakeGroup]; ok {
					group = _g
				} else {
					group = fakeGroup
				}
			}
			var groupStr string
			if group != "" {
				groupStr = "[搜索分组" + group + "]"
			}

			var id string
			if cmdArgs.GetKwarg("rand") != nil || cmdArgs.GetKwarg("随机") != nil {
				_id := rand.Uint64()%d.Parent.Help.CurID + 1
				id = strconv.FormatUint(_id, 10)
			}

			if id == "" {
				var _id string
				if useGroupSearch {
					_id = cmdArgs.GetArgN(2)
				} else {
					_id = cmdArgs.GetArgN(1)
				}
				if _id != "" {
					_, err2 := strconv.ParseInt(_id, 10, 64)
					if err2 == nil {
						id = _id
					}
				}
			}

			if id != "" {
				text, exists := d.Parent.Help.TextMap[id]
				if exists {
					content := d.Parent.Help.GetContent(text, 0)
					ReplyToSender(ctx, msg, fmt.Sprintf("词条: %s:%s\n%s", text.PackageName, text.Title, content))
				} else {
					ReplyToSender(ctx, msg, "未发现对应ID的词条")
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			var val string
			if useGroupSearch {
				val = cmdArgs.GetArgN(2)
			} else {
				val = cmdArgs.GetArgN(1)
			}
			if val == "" {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			numLimit := 4
			numParam := cmdArgs.GetKwarg("num")
			if numParam != nil {
				_num, err := strconv.ParseInt(numParam.Value, 10, 64)
				if err == nil {
					numLimit = int(_num)
				}
			}

			page := 1
			pageParam := cmdArgs.GetKwarg("page")
			if pageParam != nil {
				if _page, err := strconv.ParseInt(pageParam.Value, 10, 64); err == nil {
					page = int(_page)
				}
			}

			text := strings.TrimPrefix(cmdArgs.CleanArgs, "#"+group+" ")

			if numLimit <= 0 {
				numLimit = 1
			} else if numLimit > 10 {
				numLimit = 10
			}
			if page <= 0 {
				page = 1
			}
			if group == "" {
				// 未指定搜索分组时，取当前群指定的分组
				group = ctx.Group.DefaultHelpGroup
			}
			search, total, pgStart, pgEnd, err := d.Parent.Help.Search(ctx, text, false, numLimit, page, group)
			if err != nil {
				ReplyToSender(ctx, msg, groupStr+"搜索故障: "+err.Error())
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			if len(search.Hits) == 0 {
				if total == 0 {
					ReplyToSender(ctx, msg, groupStr+"未找到搜索结果")
				} else {
					ReplyToSender(ctx, msg, fmt.Sprintf("%s找到%d条结果, 但在当前页码并无结果", groupStr, total))
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			hasSecond := len(search.Hits) >= 2
			best := d.Parent.Help.TextMap[search.Hits[0].ID]
			others := ""

			for _, i := range search.Hits {
				t := d.Parent.Help.TextMap[i.ID]
				if t.Group != "" && t.Group != HelpBuiltinGroup {
					others += fmt.Sprintf("[%s][%s]【%s:%s】 匹配度%.2f\n", i.ID, t.Group, t.PackageName, t.Title, i.Score)
				} else {
					others += fmt.Sprintf("[%s]【%s:%s】 匹配度%.2f\n", i.ID, t.PackageName, t.Title, i.Score)
				}
			}

			var showBest bool
			if hasSecond {
				offset := d.Parent.Help.GetShowBestOffset()
				val := search.Hits[1].Score - search.Hits[0].Score
				if val < 0 {
					val = -val
				}
				if val > float64(offset) {
					showBest = true
				}
				if best.Title == text {
					showBest = true
				}
			} else {
				showBest = true
			}

			var bestResult string
			if showBest {
				content := d.Parent.Help.GetContent(best, 0)
				bestResult = fmt.Sprintf("最优先结果%s:\n词条: %s:%s\n%s\n\n", groupStr, best.PackageName, best.Title, content)
			}

			prefix := d.Parent.Help.GetPrefixText()
			rplCurPage := fmt.Sprintf("本页结果:\n%s\n", others)
			rplDetailHint := "使用\".find <序号>\"可查看明细，如.find 123\n"
			// pgStart是下标闭左边界, 加1以获得序号; pgEnd是下标开右边界, 无需调整就是最后一条的序号
			rplPageNum := fmt.Sprintf("共%d条结果, 当前显示第%d页(第%d条 到 第%d条)\n", total, page, pgStart+1, pgEnd)
			rplPageHint := "使用\".find <词条> --page=<页码> 查看更多结果\n"
			ReplyToSender(ctx, msg, prefix+groupStr+bestResult+rplCurPage+rplDetailHint+rplPageNum+rplPageHint)
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
	d.CmdMap["查询"] = cmdFind
	d.CmdMap["査詢"] = cmdFind
	d.CmdMap["find"] = cmdFind

	helpForHelp := ".help // 查看本帮助\n" +
		".help 指令 // 查看某指令信息\n" +
		".help 扩展模块 // 查看扩展信息，如.help coc7\n" +
		".help 关键字 // 查看任意帮助，同.find\n" +
		".help reload // 重新加载帮助文档，需要Master权限"
	cmdHelp := &CmdItemInfo{
		Name:      "help",
		ShortHelp: helpForHelp,
		Help:      "帮助指令，用于查看指令帮助和helpdoc中录入的信息:\n" + helpForHelp,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			arg := cmdArgs.GetArgN(1)
			if arg == "" {
				text := "海豹核心 " + VERSION.String() + "\n"
				text += "官网: sealdice.com" + "\n"
				text += "海豹群: 524364253" + "\n"
				text += DiceFormatTmpl(ctx, "核心:骰子帮助文本_附加说明")
				ReplyToSender(ctx, msg, text)
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			if strings.EqualFold(arg, "reload") {
				if ctx.PrivilegeLevel < 100 {
					ReplyToSender(ctx, msg, "你不具备Master权限")
				} else {
					dm := d.Parent
					if dm.JustForTest {
						ReplyToSender(ctx, msg, "此指令在展示模式下不可用")
						return CmdExecuteResult{Matched: true, Solved: true}
					}

					if !dm.IsHelpReloading {
						dm.IsHelpReloading = true
						dm.Help.Close()

						dm.InitHelp()
						dm.AddHelpWithDice(dm.Dice[0])
						ReplyToSender(ctx, msg, "帮助文档已经重新装载")
					} else {
						ReplyToSender(ctx, msg, "帮助文档正在重新装载，请稍后...")
					}
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			if cmdArgs.IsArgEqual(1, "help") {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			if cmdArgs.IsArgEqual(1, "骰主", "骰主信息") {
				masterMsg := DiceFormatTmpl(ctx, "核心:骰子帮助文本_骰主")
				ReplyToSender(ctx, msg, masterMsg)
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			if cmdArgs.IsArgEqual(1, "协议", "使用协议") {
				masterMsg := DiceFormatTmpl(ctx, "核心:骰子帮助文本_协议")
				ReplyToSender(ctx, msg, masterMsg)
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			if cmdArgs.IsArgEqual(1, "娱乐") {
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:骰子帮助文本_娱乐"))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			if cmdArgs.IsArgEqual(1, "其他", "其它") {
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:骰子帮助文本_其他"))
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			if d.Parent.IsHelpReloading {
				ReplyToSender(ctx, msg, "帮助文档正在重新装载，请稍后...")
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			search, _, _, _, err := d.Parent.Help.Search(ctx, cmdArgs.CleanArgs, true, 1, 1, "")
			if err == nil {
				if len(search.Hits) > 0 {
					// 居然会出现 hits[0] 为nil的情况？？
					// a := d.Parent.ShortHelp.GetContent(search.Hits[0].ID)
					a := d.Parent.Help.TextMap[search.Hits[0].ID]
					content := d.Parent.Help.GetContent(a, 0)
					ReplyToSender(ctx, msg, fmt.Sprintf("%s:%s\n%s", a.PackageName, a.Title, content))
				} else {
					ReplyToSender(ctx, msg, "未找到搜索结果")
				}
			} else {
				ReplyToSender(ctx, msg, "搜索故障: "+err.Error())
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
	d.CmdMap["help"] = cmdHelp

	cmdBot := &CmdItemInfo{
		Name:      "bot",
		ShortHelp: ".bot on/off/about/bye/quit // 开启、关闭、查看信息、退群",
		Help:      "骰子管理:\n.bot on/off/about/bye[exit,quit] // 开启、关闭、查看信息、退群",
		Raw:       true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			inGroup := msg.MessageType == "group"

			if inGroup {
				// 不响应裸指令选项
				if len(cmdArgs.At) < 1 && ctx.Dice.IgnoreUnaddressedBotCmd {
					return CmdExecuteResult{Matched: true, Solved: false}
				}
				// 不响应at其他人
				if cmdArgs.SomeoneBeMentionedButNotMe {
					return CmdExecuteResult{Matched: true, Solved: false}
				}
			}

			if len(cmdArgs.Args) > 0 && !cmdArgs.IsArgEqual(1, "about") { //nolint:nestif
				if cmdArgs.SomeoneBeMentionedButNotMe {
					return CmdExecuteResult{Matched: true, Solved: false}
				}

				cmdArgs.ChopPrefixToArgsWith("on", "off")

				matchNumber := func() (bool, bool) {
					txt := cmdArgs.GetArgN(2)
					if len(txt) >= 4 {
						if strings.HasSuffix(ctx.EndPoint.UserID, txt) {
							return true, txt != ""
						}
					}
					return false, txt != ""
				}

				isMe, exists := matchNumber()
				if exists && !isMe {
					return CmdExecuteResult{Matched: true, Solved: false}
				}

				if cmdArgs.IsArgEqual(1, "on") {
					if !(msg.Platform == "QQ-CH" || ctx.Dice.BotExtFreeSwitch || ctx.PrivilegeLevel >= 40) {
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_无权限_非master/管理/邀请者"))
						return CmdExecuteResult{Matched: true, Solved: true}
					}

					if ctx.IsPrivate {
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_私聊不可用"))
						return CmdExecuteResult{Matched: true, Solved: true}
					}

					SetBotOnAtGroup(ctx, msg.GroupID)
					ctx.Group = ctx.Session.ServiceAtNew[msg.GroupID]
					ctx.IsCurGroupBotOn = true

					text := DiceFormatTmpl(ctx, "核心:骰子开启")
					if ctx.Group.LogOn {
						text += "\n请特别注意: 日志记录处于开启状态"
					}
					ReplyToSender(ctx, msg, text)

					return CmdExecuteResult{Matched: true, Solved: true}
				} else if cmdArgs.IsArgEqual(1, "off") {
					if !(msg.Platform == "QQ-CH" || ctx.Dice.BotExtFreeSwitch || ctx.PrivilegeLevel >= 40) {
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_无权限_非master/管理/邀请者"))
						return CmdExecuteResult{Matched: true, Solved: true}
					}

					if ctx.IsPrivate {
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_私聊不可用"))
						return CmdExecuteResult{Matched: true, Solved: true}
					}

					SetBotOffAtGroup(ctx, ctx.Group.GroupID)
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:骰子关闭"))
					return CmdExecuteResult{Matched: true, Solved: true}
				} else if cmdArgs.IsArgEqual(1, "bye", "exit", "quit") {
					if cmdArgs.GetArgN(2) != "" {
						return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
					}

					if ctx.IsPrivate {
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_私聊不可用"))
						return CmdExecuteResult{Matched: true, Solved: true}
					}

					if ctx.PrivilegeLevel < 40 {
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_无权限_非master/管理"))
						return CmdExecuteResult{Matched: true, Solved: true}
					}

					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:骰子退群预告"))

					userName := ctx.Dice.Parent.TryGetUserName(msg.Sender.UserID)
					txt := fmt.Sprintf("指令退群: 于群组<%s>(%s)中告别，操作者:<%s>(%s)",
						ctx.Group.GroupName, msg.GroupID, userName, msg.Sender.UserID)
					d.Logger.Info(txt)
					ctx.Notice(txt)

					// SetBotOffAtGroup(ctx, ctx.Group.GroupID)
					time.Sleep(3 * time.Second)
					ctx.Group.DiceIDExistsMap.Delete(ctx.EndPoint.UserID)
					ctx.Group.UpdatedAtTime = time.Now().Unix()
					ctx.EndPoint.Adapter.QuitGroup(ctx, msg.GroupID)

					return CmdExecuteResult{Matched: true, Solved: true}
				} else if cmdArgs.IsArgEqual(1, "save") {
					d.Save(false)

					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:骰子保存设置"))
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			if cmdArgs.SomeoneBeMentionedButNotMe {
				return CmdExecuteResult{Matched: false, Solved: false}
			}

			activeCount := 0
			serveCount := 0
			for _, gp := range d.ImSession.ServiceAtNew {
				if gp.GroupID != "" &&
					!strings.HasPrefix(gp.GroupID, "PG-") &&
					gp.DiceIDExistsMap.Exists(ctx.EndPoint.UserID) {
					serveCount++
					if gp.DiceIDActiveMap.Exists(ctx.EndPoint.UserID) {
						activeCount++
					}
				}
			}

			onlineVer := ""
			if d.Parent.AppVersionOnline != nil {
				ver := d.Parent.AppVersionOnline
				// 如果当前不是最新版，那么提示
				if ver.VersionLatestCode != VERSION_CODE {
					onlineVer = "\n最新版本: " + ver.VersionLatestDetail + "\n"
				}
			}

			var groupWorkInfo, activeText string
			if inGroup {
				activeText = "关闭"
				if ctx.Group.IsActive(ctx) {
					activeText = "开启"
				}
				groupWorkInfo = "\n群内工作状态: " + activeText
			}

			VarSetValueInt64(ctx, "$t供职群数", int64(serveCount))
			VarSetValueInt64(ctx, "$t启用群数", int64(activeCount))
			VarSetValueStr(ctx, "$t群内工作状态", groupWorkInfo)
			VarSetValueStr(ctx, "$t群内工作状态_仅状态", activeText)
			ver := VERSION.String()
			arch := runtime.GOARCH
			if arch != "386" && arch != "amd64" {
				ver = fmt.Sprintf("%s %s", ver, arch)
			}
			baseText := fmt.Sprintf("SealDice %s%s", ver, onlineVer)
			extText := DiceFormatTmpl(ctx, "核心:骰子状态附加文本")
			if extText != "" {
				extText = "\n" + extText
			}
			text := baseText + extText

			ReplyToSender(ctx, msg, text)

			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
	d.CmdMap["bot"] = cmdBot

	helpForDismiss := ".dismiss // 退出当前群，主用于QQ，支持机器人的平台可以直接移出成员"
	cmdDismiss := &CmdItemInfo{
		Name:              "dismiss",
		ShortHelp:         helpForDismiss,
		Help:              "退群(映射到bot bye):\n" + helpForDismiss,
		Raw:               true,
		DisabledInPrivate: true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.IsPrivate {
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_私聊不可用"))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			if cmdArgs.SomeoneBeMentionedButNotMe {
				// 如果是别人被at，置之不理
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			if !cmdArgs.AmIBeMentioned {
				// 裸指令，如果当前群内开启，予以提示
				if ctx.IsCurGroupBotOn {
					ReplyToSender(ctx, msg, "[退群指令] 请@我使用这个命令，以进行确认")
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			rest := cmdArgs.GetArgN(1)
			if rest != "" {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			cmdArgs.Args = []string{"bye"}
			cmdArgs.RawArgs = "bye " + cmdArgs.RawArgs
			if rest != "" {
				cmdArgs.Args = append(cmdArgs.Args, rest)
			}
			return cmdBot.Solve(ctx, msg, cmdArgs)
		},
	}
	d.CmdMap["dismiss"] = cmdDismiss

	readIDList := func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) []string {
		var uidLst []string
		for _, i := range cmdArgs.At {
			if i.UserID == ctx.EndPoint.UserID {
				// 不许添加自己
				continue
			}
			uidLst = append(uidLst, i.UserID)
		}

		if len(cmdArgs.Args) > 1 {
			for _, i := range cmdArgs.Args[1:] {
				if i == "me" {
					uidLst = append(uidLst, ctx.Player.UserID)
					continue
				}
				uid := FormatDiceIDQQ(i)
				uidLst = append(uidLst, uid)
			}
		}
		return uidLst
	}

	botListHelp := ".botlist add @A @B @C // 标记群内其他机器人，以免发生误触和无限对话\n" +
		".botlist add @A @B --s  // 同上，不过骰子不会做出回复\n" +
		".botlist del @A @B @C // 去除机器人标记\n" +
		".botlist list/show // 查看当前列表"

	cmdBotList := &CmdItemInfo{
		Name:      "botlist",
		ShortHelp: botListHelp,
		Help:      "机器人列表:\n" + botListHelp,
		Raw:       true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.IsPrivate {
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_私聊不可用"))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			cmdArgs.ChopPrefixToArgsWith("add", "rm", "del", "show", "list")

			checkSlience := func() bool {
				return (!cmdArgs.AmIBeMentionedFirst) || cmdArgs.GetKwarg("s") != nil ||
					cmdArgs.GetKwarg("slience") != nil
			}

			subCmd := cmdArgs.GetArgN(1)
			switch subCmd {
			case "add":
				allCount := 0
				existsCount := 0
				newCount := 0
				for _, uid := range readIDList(ctx, msg, cmdArgs) {
					allCount++
					if ctx.Group.BotList.Exists(uid) {
						existsCount++
					} else {
						ctx.Group.BotList.Store(uid, true)
						newCount++
					}
				}

				text := fmt.Sprintf("新增标记了%d/%d个帐号，这些账号将被视为机器人。\n因此他们被人@，或主动发出指令时，海豹将不会回复。\n另外对于botlist add/rm，如果群里有多个海豹，只有第一个被@的会回复，其余的执行指令但不回应", newCount, allCount)
				if !checkSlience() {
					ReplyToSender(ctx, msg, text)
				} else {
					d.Logger.Infof("静默执行: " + text)
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			case "del", "rm":
				allCount := 0
				existsCount := 0
				for _, uid := range readIDList(ctx, msg, cmdArgs) {
					allCount++
					if ctx.Group.BotList.Exists(uid) {
						existsCount++
						ctx.Group.BotList.Delete(uid)
					}
				}

				text := fmt.Sprintf("删除标记了%d/%d个帐号，这些账号将不再被视为机器人。\n海豹将继续回应他们的命令", existsCount, allCount)
				if !checkSlience() {
					ReplyToSender(ctx, msg, text)
				} else {
					d.Logger.Infof("静默执行: " + text)
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			case "list", "show":
				if cmdArgs.SomeoneBeMentionedButNotMe {
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				text := ""
				ctx.Group.BotList.Range(func(k string, _ bool) bool {
					text += "- " + k + "\n"
					return true
				})
				if text == "" {
					text = "无"
				}
				ReplyToSender(ctx, msg, fmt.Sprintf("群内其他机器人列表:\n%s", text))
				return CmdExecuteResult{Matched: true, Solved: true}
			default:
				if cmdArgs.SomeoneBeMentionedButNotMe {
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
		},
	}
	d.CmdMap["botlist"] = cmdBotList

	var (
		reloginFlag     bool
		reloginLastTime int64
		updateCode      = "0000"
	)

	var masterListHelp = `.master add me // 将自己标记为骰主
.master add @A @B // 将别人标记为骰主
.master del @A @B @C // 去除骰主标记
.master unlock <密码(在UI中查看)> // (当Master被人抢占时)清空骰主列表，并使自己成为骰主
.master list // 查看当前骰主列表
.master reboot // 重新启动(需要二次确认)
.master checkupdate // 检查更新(需要二次确认)
.master relogin // 30s后重新登录，有机会清掉风控(仅master可用)
.master backup // 做一次备份
.master reload deck/js/helpdoc // 重新加载牌堆/js/帮助文档
.master quitgroup <群组ID> <理由(可选)> // 从指定群组中退出，必须在同一平台使用`

	cmdMaster := &CmdItemInfo{
		Name:          "master",
		ShortHelp:     masterListHelp,
		Help:          "骰主指令:\n" + masterListHelp,
		AllowDelegate: true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			var subCmd string

			cmdArgs.ChopPrefixToArgsWith(
				"unlock", "rm", "del", "add", "checkupdate", "reboot", "backup", "reload",
			)
			ctx.DelegateText = ""
			subCmd = cmdArgs.GetArgN(1)

			if subCmd != "add" && subCmd != "del" && subCmd != "rm" {
				// 如果不是add/del/rm，那么就不需要代骰
				// 补充，在组内才这样，私聊不需要at
				if ctx.MessageType == "group" && (!cmdArgs.AmIBeMentionedFirst && len(cmdArgs.At) > 0) {
					return CmdExecuteResult{Matched: false, Solved: false}
				}
			}

			var pRequired int
			if len(ctx.Dice.DiceMasters) >= 1 {
				// 如果帐号没有UI:1001以外的master，所有人都是master
				count := 0
				for _, uid := range ctx.Dice.DiceMasters {
					if uid != "UI:1001" {
						count += 1
					}
				}

				if count >= 1 {
					pRequired = 100
				}
			}

			// 单独处理解锁指令
			if subCmd == "unlock" {
				// 特殊解锁指令
				code := cmdArgs.GetArgN(2)
				if ctx.Dice.UnlockCodeVerify(code) {
					ctx.Dice.MasterRefresh()
					ctx.Dice.MasterAdd(ctx.Player.UserID)

					ctx.Dice.UnlockCodeUpdate(true) // 强制刷新解锁码
					ReplyToSender(ctx, msg, "你已成为Master")
				} else {
					ReplyToSender(ctx, msg, "错误的解锁码")
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			if ctx.PrivilegeLevel < pRequired {
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_无权限"))
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			switch subCmd {
			case "add":
				var count int
				for _, uid := range readIDList(ctx, msg, cmdArgs) {
					if uid != ctx.EndPoint.UserID {
						ctx.Dice.MasterAdd(uid)
						count++
					}
				}
				ctx.Dice.Save(false)
				ReplyToSender(ctx, msg, fmt.Sprintf("海豹将新增%d位master", count))
			case "del", "rm":
				var count int
				for _, uid := range readIDList(ctx, msg, cmdArgs) {
					if ctx.Dice.MasterRemove(uid) {
						count++
					}
				}
				ctx.Dice.Save(false)
				ReplyToSender(ctx, msg, fmt.Sprintf("海豹移除了%d名master", count))
			case "relogin":
				var kw *Kwarg

				if kw = cmdArgs.GetKwarg("cancel"); kw != nil {
					if reloginFlag {
						reloginFlag = false
						ReplyToSender(ctx, msg, "已取消重登录")
					} else {
						ReplyToSender(ctx, msg, "错误: 不存在能够取消的重新登录行为")
					}
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				doRelogin := func() {
					reloginLastTime = time.Now().Unix()
					ReplyToSender(ctx, msg, "开始执行重新登录")
					reloginFlag = false
					time.Sleep(1 * time.Second)
					ctx.EndPoint.Adapter.DoRelogin()
				}

				if time.Now().Unix()-reloginLastTime < 5*60 {
					ReplyToSender(
						ctx,
						msg,
						fmt.Sprintf(
							"执行过不久，指令将在%d秒后可以使用",
							5*60-(time.Now().Unix()-reloginLastTime),
						),
					)
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				if kw = cmdArgs.GetKwarg("now"); kw != nil {
					doRelogin()
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				reloginFlag = true
				ReplyToSender(ctx, msg, "将在30s后重新登录，期间可以输入.master relogin --cancel解除\n若遭遇风控，可能会没有任何输出。静等或输入.master relogin --now立即执行\n此指令每5分钟只能执行一次，可能解除风控，也可能使骰子失联。后果自负")

				go func() {
					time.Sleep(30 * time.Second)
					if reloginFlag {
						doRelogin()
					}
				}()
			case "backup":
				ReplyToSender(ctx, msg, "开始备份数据")

				_, err := ctx.Dice.Parent.BackupSimple()
				if err == nil {
					ReplyToSender(ctx, msg, "备份成功！请到UI界面(综合设置-备份)处下载备份，或在骰子backup目录下读取")
				} else {
					d.Logger.Error("骰子备份:", err)
					ReplyToSender(ctx, msg, "备份失败！错误已写入日志。可能是磁盘已满所致，建议立即进行处理！")
				}
			case "checkupdate":
				var dm = ctx.Dice.Parent
				if dm.JustForTest {
					ReplyToSender(ctx, msg, "此指令在展示模式下不可用")
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				if runtime.GOOS == "android" {
					ReplyToSender(ctx, msg, "检测到手机版，手机版海豹不支持指令更新，请手动下载新版本安装包")
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				code := cmdArgs.GetArgN(2)
				if code == "" {
					var text string
					dm.AppVersionOnline = nil
					dm.UpdateCheckRequestChan <- 1

					// 等待获取新版本，最多10s
					for i := 0; i < 5; i++ {
						time.Sleep(2 * time.Second)
						if dm.AppVersionOnline != nil {
							break
						}
					}

					if dm.AppVersionOnline != nil {
						text = fmt.Sprintf("当前本地版本为: %s\n当前线上版本为: %s", VERSION.String(), dm.AppVersionOnline.VersionLatestDetail)
						if dm.AppVersionCode != dm.AppVersionOnline.VersionLatestCode {
							updateCode = strconv.FormatInt(rand.Int63()%8999+1000, 10)
							text += fmt.Sprintf("\n如需升级，请输入.master checkupdate %s 确认进行升级\n升级将花费约2分钟，升级失败可能导致进程关闭，建议在接触服务器情况下操作。\n当前进程启动时间: %s", updateCode, time.Unix(dm.AppBootTime, 0).Format("2006-01-02 15:04:05"))
						}
					} else {
						text = fmt.Sprintf("当前本地版本为: %s\n当前线上版本为: %s", VERSION.String(), "未知")
					}
					ReplyToSender(ctx, msg, text)
					break
				}

				if code != updateCode || updateCode == "0000" {
					ReplyToSender(ctx, msg, "无效的升级指令码")
					break
				}

				ReplyToSender(ctx, msg, "开始下载新版本，完成后将自动进行一次备份")
				go func() {
					ret := <-dm.UpdateDownloadedChan

					if ctx.IsPrivate {
						ctx.Dice.UpgradeWindowID = msg.Sender.UserID
					} else {
						ctx.Dice.UpgradeWindowID = ctx.Group.GroupID
					}
					ctx.Dice.UpgradeEndpointID = ctx.EndPoint.ID
					ctx.Dice.Save(true)

					bakFn, _ := ctx.Dice.Parent.BackupSimple()
					tmpPath := path.Join(os.TempDir(), bakFn)
					_ = os.MkdirAll(tmpPath, 0755)
					ctx.Dice.Logger.Infof("将备份文件复制到此路径: %s", tmpPath)
					_ = cp.Copy(path.Join(BackupDir, bakFn), tmpPath)

					if ret == "" {
						ReplyToSender(ctx, msg, "准备开始升级，服务即将离线")
					} else {
						ReplyToSender(ctx, msg, "升级失败，原因: "+ret)
					}
				}()
				dm.UpdateRequestChan <- d
			case "reboot":
				var dm = ctx.Dice.Parent
				if dm.JustForTest {
					ReplyToSender(ctx, msg, "此指令在展示模式下不可用")
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				code := cmdArgs.GetArgN(2)
				if code == "" {
					updateCode = strconv.FormatInt(rand.Int63()%8999+1000, 10)
					text := fmt.Sprintf("进程重启:\n如需重启，请输入.master reboot %s 确认进行重启\n重启将花费约2分钟，失败可能导致进程关闭，建议在接触服务器情况下操作。\n当前进程启动时间: %s", updateCode, time.Unix(dm.AppBootTime, 0).Format("2006-01-02 15:04:05"))
					ReplyToSender(ctx, msg, text)
					break
				}

				if code == updateCode && updateCode != "0000" {
					ReplyToSender(ctx, msg, "3秒后开始重启")
					time.Sleep(3 * time.Second)
					dm.RebootRequestChan <- 1
				} else {
					ReplyToSender(ctx, msg, "无效的重启指令码")
				}
			case "list":
				text := ""
				for _, i := range ctx.Dice.DiceMasters {
					// uid := FormatDiceIdQQ(i)
					text += "- " + i + "\n"
				}
				if text == "" {
					text = "无"
				}
				ReplyToSender(ctx, msg, fmt.Sprintf("Master列表:\n%s", text))
			case "reload":
				dice := ctx.Dice
				switch cmdArgs.GetArgN(2) {
				case "deck":
					DeckReload(dice)
					ReplyToSender(ctx, msg, "牌堆已重载")
				case "js":
					dice.JsReload()
					ReplyToSender(ctx, msg, "js已重载")
				case "help", "helpdoc":
					dm := dice.Parent
					if !dm.IsHelpReloading {
						dm.IsHelpReloading = true
						dm.Help.Close()
						dm.InitHelp()
						dm.AddHelpWithDice(dice)
						ReplyToSender(ctx, msg, "帮助文档已重载")
					} else {
						ReplyToSender(ctx, msg, "帮助文档正在重新装载")
					}
				}
			case "quitgroup":
				gid := cmdArgs.GetArgN(2)
				if gid == "" {
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				n := strings.Split(gid, ":") // 不验证是否合法，反正下面会检查是否在 ServiceAtNew
				platform := strings.Split(n[0], "-")[0]

				gp, ok := ctx.Session.ServiceAtNew[gid]
				if !ok || len(n[0]) < 2 {
					ReplyToSender(ctx, msg, fmt.Sprintf("群组列表中没有找到%s", gid))
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				if msg.Platform != platform {
					ReplyToSender(ctx, msg, fmt.Sprintf("目标群组不在当前平台，请前往%s完成操作", platform))
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				// 既然是骰主自己操作，就不通知了
				// 除非有多骰主……
				ReplyToSender(ctx, msg, fmt.Sprintf("收到指令，将在5秒后退出群组%s", gp.GroupID))

				txt := "注意，收到骰主指令，5秒后将从该群组退出。"
				wherefore := cmdArgs.GetArgN(3)
				if wherefore != "" {
					txt += fmt.Sprintf("原因: %s", wherefore)
				}

				ReplyGroup(ctx, &Message{GroupID: gp.GroupID}, txt)

				mctx := &MsgContext{
					MessageType: "group",
					Group:       gp,
					EndPoint:    ctx.EndPoint,
					Session:     ctx.Session,
					Dice:        ctx.Dice,
					IsPrivate:   false,
				}
				// SetBotOffAtGroup(mctx, gp.GroupID)
				time.Sleep(3 * time.Second)
				gp.DiceIDExistsMap.Delete(mctx.EndPoint.UserID)
				gp.UpdatedAtTime = time.Now().Unix()
				mctx.EndPoint.Adapter.QuitGroup(mctx, gp.GroupID)

				return CmdExecuteResult{Matched: true, Solved: true}
			default:
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
	d.CmdMap["master"] = cmdMaster

	cmdSend := &CmdItemInfo{
		Name:      "send",
		ShortHelp: ".send // 向骰主留言",
		Help: "留言指令:\n.send XXXXXX // 向骰主留言\n" +
			".send to <对方ID> 要说的话 // 骰主回复，举例. send to QQ:12345 感谢留言\n" +
			".send to <群组ID> 要说的话 // 举例. send to QQ-Group:12345 感谢留言\n" +
			"> 指令.userid可以查看当前群的ID",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			val := cmdArgs.GetArgN(1)
			if val == "to" { //nolint:nestif // TODO
				if ctx.PrivilegeLevel >= 100 {
					uid := cmdArgs.GetArgN(2)
					txt := cmdArgs.GetRestArgsFrom(3)
					if uid != "" && strings.HasPrefix(uid, ctx.EndPoint.Platform) && txt != "" {
						isGroup := strings.Contains(uid, "-Group:")
						txt = fmt.Sprintf("本消息由骰主<%s>通过指令发送:\n", ctx.Player.Name) + txt
						if isGroup {
							ReplyGroup(ctx, &Message{GroupID: uid}, txt)
						} else {
							ReplyPerson(ctx, &Message{Sender: SenderBase{UserID: uid}}, txt)
						}
						ReplyToSender(ctx, msg, "信息已经发送至"+uid)
						return CmdExecuteResult{Matched: true, Solved: true}
					}
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}
				ReplyToSender(ctx, msg, "你不具备Master权限")
			} else if val == "help" || val == "" {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			} else {
				if d.MailEnable {
					_ = ctx.Dice.SendMail(cmdArgs.CleanArgs, MailTypeSendNote)
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:留言_已记录"))
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				for _, uid := range ctx.Dice.DiceMasters {
					text := ""

					if ctx.IsCurGroupBotOn {
						text += fmt.Sprintf("一条来自群组<%s>(%s)，作者<%s>(%s)的留言:\n", ctx.Group.GroupName, ctx.Group.GroupID, ctx.Player.Name, ctx.Player.UserID)
					} else {
						text += fmt.Sprintf("一条来自私聊，作者<%s>(%s)的留言:\n", ctx.Player.Name, ctx.Player.UserID)
					}

					text += cmdArgs.CleanArgs
					if strings.Contains(uid, "Group") {
						ctx.EndPoint.Adapter.SendToGroup(ctx, uid, text, "")
					} else {
						ctx.EndPoint.Adapter.SendToPerson(ctx, uid, text, "")
					}
				}
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:留言_已记录"))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
		},
	}
	d.CmdMap["send"] = cmdSend

	helpRoll := ".r <表达式> <原因> // 骰点指令\n.rh <表达式> <原因> // 暗骰"
	cmdRoll := &CmdItemInfo{
		EnableExecuteTimesParse: true,
		Name:                    "roll",
		ShortHelp:               helpRoll,
		Help:                    "骰点:\n" + helpRoll,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			var text string
			var diceResult int64
			var diceResultExists bool
			var detail string

			if cmdArgs.IsArgEqual(1, "help") {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			ctx.SystemTemplate = ctx.Group.GetCharTemplate(ctx.Dice)
			if ctx.Dice.CommandCompatibleMode {
				if (cmdArgs.Command == "rd" || cmdArgs.Command == "rhd" || cmdArgs.Command == "rdh") && len(cmdArgs.Args) >= 1 {
					if m, _ := regexp.MatchString(`^\d|优势|劣势|\+|-`, cmdArgs.CleanArgs); m {
						if cmdArgs.IsSpaceBeforeArgs {
							cmdArgs.CleanArgs = "d " + cmdArgs.CleanArgs
						} else {
							cmdArgs.CleanArgs = "d" + cmdArgs.CleanArgs
						}
					}
				}
			}

			var r *VMResult
			var commandInfoItems []interface{}

			rollOne := func() *CmdExecuteResult {
				forWhat := ""
				if len(cmdArgs.Args) >= 1 { //nolint:nestif
					var err error
					r, detail, err = ctx.Dice.ExprEvalBase(cmdArgs.CleanArgs, ctx, RollExtraFlags{
						DefaultDiceSideNum: getDefaultDicePoints(ctx),
						DisableBlock:       true,
					})

					if r != nil && !r.Parser.Calculated {
						forWhat = cmdArgs.CleanArgs

						defExpr := "d"
						if ctx.diceExprOverwrite != "" {
							defExpr = ctx.diceExprOverwrite
						}
						r, detail, err = ctx.Dice.ExprEvalBase(defExpr, ctx, RollExtraFlags{
							DefaultDiceSideNum: getDefaultDicePoints(ctx),
							DisableBlock:       true,
						})
					}

					if r != nil && r.TypeID == 0 {
						diceResult = r.Value.(int64)
						diceResultExists = true
					}

					if err == nil {
						if forWhat == "" {
							forWhat = r.restInput
						}
					} else {
						errs := err.Error()
						if strings.HasPrefix(errs, "E1:") || strings.HasPrefix(errs, "E5:") || strings.HasPrefix(errs, "E6:") || strings.HasPrefix(errs, "E7:") || strings.HasPrefix(errs, "E8:") {
							ReplyToSender(ctx, msg, errs)
							return &CmdExecuteResult{Matched: true, Solved: true}
						}
						forWhat = cmdArgs.CleanArgs
					}
				}

				VarSetValueStr(ctx, "$t原因", forWhat)
				if forWhat != "" {
					forWhatText := DiceFormatTmpl(ctx, "核心:骰点_原因")
					VarSetValueStr(ctx, "$t原因句子", forWhatText)
				} else {
					VarSetValueStr(ctx, "$t原因句子", "")
				}

				if diceResultExists { //nolint:nestif
					detailWrap := ""
					if detail != "" {
						detailWrap = "=" + detail
						re := regexp.MustCompile(`\[((\d+)d\d+)\=(\d+)\]`)
						match := re.FindStringSubmatch(detail)
						if len(match) > 0 {
							num := match[2]
							if num == "1" && (match[1] == r.Matched || match[1] == "1"+r.Matched) {
								detailWrap = ""
							}
						}
					}

					// 指令信息标记
					item := map[string]interface{}{
						"expr":   r.Matched,
						"result": diceResult,
						"reason": forWhat,
					}
					if forWhat == "" {
						delete(item, "reason")
					}
					commandInfoItems = append(commandInfoItems, item)

					VarSetValueStr(ctx, "$t表达式文本", r.Matched)
					VarSetValueStr(ctx, "$t计算过程", detailWrap)
					VarSetValueInt64(ctx, "$t计算结果", diceResult)
				} else {
					var val int64
					var detail string
					dicePoints := getDefaultDicePoints(ctx)
					if ctx.diceExprOverwrite != "" {
						r, detail, _ = ctx.Dice.ExprEvalBase(cmdArgs.CleanArgs, ctx, RollExtraFlags{
							DefaultDiceSideNum: dicePoints,
							DisableBlock:       true,
						})
						if r != nil && r.TypeID == 0 {
							val, _ = r.ReadInt64()
						}
					} else {
						val = DiceRoll64(dicePoints)
					}

					// 指令信息标记
					item := map[string]interface{}{
						"expr":       fmt.Sprintf("D%d", dicePoints),
						"reason":     forWhat,
						"dicePoints": dicePoints,
						"result":     val,
					}
					if forWhat == "" {
						delete(item, "reason")
					}
					commandInfoItems = append(commandInfoItems, item)

					VarSetValueStr(ctx, "$t表达式文本", fmt.Sprintf("D%d", dicePoints))
					VarSetValueStr(ctx, "$t计算过程", detail)
					VarSetValueInt64(ctx, "$t计算结果", val)
				}
				return nil
			}

			if cmdArgs.SpecialExecuteTimes > 1 {
				VarSetValueInt64(ctx, "$t次数", int64(cmdArgs.SpecialExecuteTimes))
				if cmdArgs.SpecialExecuteTimes > int(ctx.Dice.MaxExecuteTime) {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:骰点_轮数过多警告"))
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				var texts []string
				for i := 0; i < cmdArgs.SpecialExecuteTimes; i++ {
					ret := rollOne()
					if ret != nil {
						return *ret
					}
					texts = append(texts, DiceFormatTmpl(ctx, "核心:骰点_单项结果文本"))
				}
				VarSetValueStr(ctx, "$t结果文本", strings.Join(texts, `\n`))
				text = DiceFormatTmpl(ctx, "核心:骰点_多轮")
			} else {
				ret := rollOne()
				if ret != nil {
					return *ret
				}
				VarSetValueStr(ctx, "$t结果文本", DiceFormatTmpl(ctx, "核心:骰点_单项结果文本"))
				text = DiceFormatTmpl(ctx, "核心:骰点")
			}

			isHide := strings.Contains(cmdArgs.Command, "h")

			// 指令信息
			commandInfo := map[string]interface{}{
				"cmd":    "roll",
				"pcName": ctx.Player.Name,
				"items":  commandInfoItems,
			}
			if isHide {
				commandInfo["hide"] = isHide
			}
			ctx.CommandInfo = commandInfo

			if kw := cmdArgs.GetKwarg("asm"); r != nil && kw != nil {
				if ctx.PrivilegeLevel >= 40 {
					asm := r.Parser.GetAsmText()
					text += "\n" + asm
				}
			}

			if kw := cmdArgs.GetKwarg("ci"); kw != nil {
				info, err := json.Marshal(ctx.CommandInfo)
				if err == nil {
					text += "\n" + string(info)
				} else {
					text += "\n" + "指令信息无法序列化"
				}
			}

			if isHide {
				if msg.Platform == "QQ-CH" {
					ReplyToSender(ctx, msg, "QQ频道内尚不支持暗骰")
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				if ctx.Group != nil {
					if ctx.IsPrivate {
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_私聊不可用"))
					} else {
						ctx.CommandHideFlag = ctx.Group.GroupID
						prefix := DiceFormatTmpl(ctx, "核心:暗骰_私聊_前缀")
						ReplyGroup(ctx, msg, DiceFormatTmpl(ctx, "核心:暗骰_群内"))
						ReplyPerson(ctx, msg, prefix+text)
					}
				} else {
					ReplyToSender(ctx, msg, text)
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			ReplyToSender(ctx, msg, text)
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	helpRollX := ".rx <表达式> <原因> // 骰点指令\n.rxh <表达式> <原因> // 暗骰"
	cmdRollX := &CmdItemInfo{
		Name:          "roll",
		ShortHelp:     helpRoll,
		Help:          "骰点(和r相同，但支持代骰):\n" + helpRollX,
		AllowDelegate: true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			mctx := GetCtxProxyFirst(ctx, cmdArgs)
			return cmdRoll.Solve(mctx, msg, cmdArgs)
		},
	}

	d.CmdMap["r"] = cmdRoll
	d.CmdMap["rd"] = cmdRoll
	d.CmdMap["roll"] = cmdRoll
	d.CmdMap["rh"] = cmdRoll
	d.CmdMap["rhd"] = cmdRoll
	d.CmdMap["rdh"] = cmdRoll
	d.CmdMap["rx"] = cmdRollX
	d.CmdMap["rxh"] = cmdRollX
	d.CmdMap["rhx"] = cmdRollX

	vm := ds.NewVM()
	vm.Config.EnableDiceWoD = true
	vm.Config.EnableDiceCoC = true
	vm.Config.EnableDiceFate = true
	vm.Config.EnableDiceDoubleCross = true

	helpRollNew := ".fox <表达式> <原因> // 使用dicescript的骰点，测试用"
	cmdFox := &CmdItemInfo{
		Name:      "roll",
		ShortHelp: helpRollNew,
		Help:      "骰点:\n" + helpRollNew,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if ctx.Dice.TextCmdTrustOnly {
				// 检查master和信任权限
				// 拒绝无权限访问
				if ctx.PrivilegeLevel < 70 {
					ReplyToSender(ctx, msg, "你不具备Master权限")
					return CmdExecuteResult{Matched: true, Solved: true}
				}
			}
			if cmdArgs.IsArgEqual(1, "help") {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			ctx.SystemTemplate = ctx.Group.GetCharTemplate(ctx.Dice)

			expr := cmdArgs.GetRestArgsFrom(1)
			if expr == "" {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			tmpl := cardRuleCheck(ctx, msg)
			if tmpl == nil {
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			loadValueFromRollVMv1 := func(name string) *ds.VMValue {
				val, err := tmpl.getShowAsBase(ctx, name)
				if err != nil {
					vm.Error = err
					return ds.VMValueNewUndefined()
				}
				if val == nil {
					return nil
				}
				return val.ConvertToDiceScriptValue()
			}

			vm.GlobalValueLoadFunc = loadValueFromRollVMv1
			valueToRollVMv1 := func(v *ds.VMValue) *VMValue {
				var v2 *VMValue
				switch v.TypeId {
				case ds.VMTypeInt:
					v2 = &VMValue{TypeID: VMTypeInt64, Value: v.MustReadInt()}
				case ds.VMTypeFloat:
					v2 = &VMValue{TypeID: VMTypeInt64, Value: int64(v.MustReadFloat())}
				default:
					v2 = &VMValue{TypeID: VMTypeString, Value: v.ToString()}
				}
				return v2
			}

			funcWrap := func(name string, val *ds.VMValue) *ds.VMValue {
				return ds.VMValueNewNativeFunction(&ds.NativeFunctionData{
					Name:   name,
					Params: []string{},
					NativeFunc: func(vm *ds.Context, this *ds.VMValue, params []*ds.VMValue) *ds.VMValue {
						return val
					},
				})
			}

			od := &ds.NativeObjectData{
				Name: "player",
				AttrGet: func(vm *ds.Context, name string) *ds.VMValue {
					// 注: 未来切换人物卡时，角色数据会被转换为ds版本，所以这里写的丑陋了一些
					switch name {
					case "keys":
						vars, _ := ctx.ChVarsGet()
						items := []*ds.VMValue{}
						_ = vars.Iterate(func(_k interface{}, _v interface{}) error {
							items = append(items, ds.VMValueNewStr(_k.(string)))
							return nil
						})
						return funcWrap("keys", ds.VMValueNewArrayRaw(items))
					case "values":
						vars, _ := ctx.ChVarsGet()
						items := []*ds.VMValue{}
						_ = vars.Iterate(func(_k interface{}, _v interface{}) error {
							v := (_v).(*VMValue)
							items = append(items, v.ConvertToDiceScriptValue())
							return nil
						})
						return funcWrap("values", ds.VMValueNewArrayRaw(items))
					case "items":
						vars, _ := ctx.ChVarsGet()
						items := []*ds.VMValue{}
						_ = vars.Iterate(func(_k interface{}, _v interface{}) error {
							items = append(items, ds.VMValueNewArray(ds.VMValueNewStr(_k.(string)), (_v).(*VMValue).ConvertToDiceScriptValue()))
							return nil
						})
						return funcWrap("items", ds.VMValueNewArrayRaw(items))
					}
					return loadValueFromRollVMv1(name)
				},
				ItemGet: func(vm *ds.Context, index *ds.VMValue) *ds.VMValue {
					if index.TypeId != ds.VMTypeString {
						vm.Error = errors.New("index must be string")
						return nil
					}
					return loadValueFromRollVMv1(index.ToString())
				},
				AttrSet: func(vm *ds.Context, name string, v *ds.VMValue) {
					VarSetValue(ctx, tmpl.GetAlias(name), valueToRollVMv1(v))
				},
				ItemSet: func(vm *ds.Context, index *ds.VMValue, v *ds.VMValue) {
					if index.TypeId != ds.VMTypeString {
						vm.Error = errors.New("index must be string")
						return
					}
					name := index.ToString()
					VarSetValue(ctx, tmpl.GetAlias(name), valueToRollVMv1(v))
				},
				DirFunc: func(vm *ds.Context) []*ds.VMValue {
					vars, _ := ctx.ChVarsGet()
					items := []*ds.VMValue{}
					_ = vars.Iterate(func(_k interface{}, _v interface{}) error {
						items = append(items, ds.VMValueNewStr(_k.(string)))
						return nil
					})
					return items
				},
			}

			vPlayer := ds.VMValueNewNativeObject(od)
			vm.StoreNameLocal("player", vPlayer)
			vm.StoreNameLocal("玩家", vPlayer)

			vm.ValueStoreHookFunc = func(vm *ds.Context, name string, v *ds.VMValue) (solved bool) {
				if strings.HasPrefix(name, "$") {
					VarSetValue(ctx, name, valueToRollVMv1(v))
					return true
				}
				return false
			}

			storeName := func(name string, v *ds.VMValue) {
				if strings.HasPrefix(name, "$") {
					vm.StoreName(name, v) // 走hook
				}
				vm.StoreNameLocal(name, v)
			}

			var text string
			if err := vm.Run(expr); err == nil {
				storeName("$t表达式文本", ds.VMValueNewStr(expr))
				storeName("$t计算过程", ds.VMValueNewStr(vm.Detail))
				storeName("$t计算结果", vm.Ret)
				storeName("$t剩余文本", ds.VMValueNewStr(vm.RestInput))
				// text = fmt.Sprintf("%s=%s=%s", expr, vm.Detail, vm.Ret.ToString())
				// 注: 加一个format
				expr := "过程: {$t计算过程}\n结果: {$t计算结果}"
				if vm.RestInput != "" {
					expr += "\n剩余: {$t剩余文本}"
				}
				err = vm.Run("\x1e" + expr + "\x1e")
				if err == nil {
					text = vm.Ret.ToString()
				} else {
					text = fmt.Sprintf("错误: %s\n", err.Error())
				}
			} else {
				text = fmt.Sprintf("错误: %s\n", err.Error())
			}
			ReplyToSender(ctx, msg, text)
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
	d.CmdMap["fox"] = cmdFox

	helpExt := ".ext // 查看扩展列表"
	cmdExt := &CmdItemInfo{
		Name:      "ext",
		ShortHelp: helpExt,
		Help:      "群扩展模块管理:\n" + helpExt,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if cmdArgs.IsArgEqual(1, "help") {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			showList := func() {
				text := "检测到以下扩展(名称-版本-作者)：\n"
				for index, i := range ctx.Dice.ExtList {
					state := "关"
					for _, j := range ctx.Group.ActivatedExtList {
						if i.Name == j.Name {
							state = "开"
							break
						}
					}
					var officialMark string
					if i.Official {
						officialMark = "[官方]"
					}
					author := i.Author
					if author == "" {
						author = "<未注明>"
					}
					aliases := ""
					if len(i.Aliases) > 0 {
						aliases = "(" + strings.Join(i.Aliases, ",") + ")"
					}
					text += fmt.Sprintf("%d. [%s]%s%s %s - %s - %s\n", index+1, state, officialMark, i.Name, aliases, i.Version, author)
				}
				text += "使用命令: .ext <扩展名> on/off 可以在当前群开启或关闭某扩展。\n"
				text += "命令: .ext <扩展名> 可以查看扩展介绍及帮助"
				ReplyToSender(ctx, msg, text)
			}

			if len(cmdArgs.Args) == 0 {
				showList()
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			var last int
			if len(cmdArgs.Args) >= 2 {
				last = len(cmdArgs.Args)
			}

			//nolint:nestif
			if cmdArgs.IsArgEqual(1, "list") {
				showList()
			} else if cmdArgs.IsArgEqual(last, "on") {
				if !ctx.Dice.BotExtFreeSwitch && ctx.PrivilegeLevel < 40 {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_无权限_非master/管理/邀请者"))
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				checkConflict := func(ext *ExtInfo) []string {
					var actived []string
					for _, i := range ctx.Group.ActivatedExtList {
						actived = append(actived, i.Name)
					}

					if ext.ConflictWith != nil {
						var ret []string
						for _, i := range intersect.Simple(actived, ext.ConflictWith) {
							ret = append(ret, i.(string))
						}
						return ret
					}
					return []string{}
				}

				var extNames []string
				var conflictsAll []string
				for index := 0; index < len(cmdArgs.Args); index++ {
					extName := strings.ToLower(cmdArgs.Args[index])
					if i := d.ExtFind(extName); i != nil {
						extNames = append(extNames, extName)
						conflictsAll = append(conflictsAll, checkConflict(i)...)
						ctx.Group.ExtActive(i)
					}
				}

				if len(extNames) == 0 {
					ReplyToSender(ctx, msg, "输入的扩展类别名无效")
				} else {
					text := fmt.Sprintf("打开扩展 %s", strings.Join(extNames, ","))
					if len(conflictsAll) > 0 {
						text += "\n检测到可能冲突的扩展，建议关闭: " + strings.Join(conflictsAll, ",")
						text += "\n对于扩展中存在的同名指令，则越晚开启的扩展，优先级越高。"
					}
					ReplyToSender(ctx, msg, text)
				}
			} else if cmdArgs.IsArgEqual(last, "off") {
				if !ctx.Dice.BotExtFreeSwitch && ctx.PrivilegeLevel < 40 {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_无权限_非master/管理/邀请者"))
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				var closed []string
				var notfound []string
				for index := 0; index < len(cmdArgs.Args); index++ {
					extName := cmdArgs.Args[index]
					extName = d.ExtAliasToName(extName)
					ei := ctx.Group.ExtInactiveByName(extName)
					if ei != nil {
						closed = append(closed, ei.Name)
					} else {
						notfound = append(notfound, extName)
					}
				}

				var text string

				if len(closed) > 0 {
					text += fmt.Sprintf("关闭扩展: %s", strings.Join(closed, ","))
				} else {
					text += fmt.Sprintf(" 已关闭或未找到: %s", strings.Join(notfound, ","))
				}
				ReplyToSender(ctx, msg, text)
				return CmdExecuteResult{Matched: true, Solved: true}
			} else {
				extName := cmdArgs.Args[0]
				if i := d.ExtFind(extName); i != nil {
					text := fmt.Sprintf("> [%s] 版本%s 作者%s\n", i.Name, i.Version, i.Author)
					i.callWithJsCheck(d, func() {
						ReplyToSender(ctx, msg, text+i.GetDescText(i))
					})
					return CmdExecuteResult{Matched: true, Solved: true}
				}
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
	d.CmdMap["ext"] = cmdExt

	helpNN := ".nn // 查看当前角色名\n" +
		".nn <角色名> // 改为指定角色名，若有卡片不会连带修改\n" +
		".nn clr // 重置回群名片"
	cmdNN := &CmdItemInfo{
		Name:      "nn",
		ShortHelp: helpNN,
		Help:      "角色名设置:\n" + helpNN,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			val := strings.ToLower(cmdArgs.GetArgN(1))
			switch val {
			case "":
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:昵称_当前"))
			case "help":
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			case "clr", "reset":
				p := ctx.Player
				VarSetValueStr(ctx, "$t旧昵称", fmt.Sprintf("<%s>", ctx.Player.Name))
				VarSetValueStr(ctx, "$t旧昵称_RAW", ctx.Player.Name)
				p.Name = msg.Sender.Nickname
				p.UpdatedAtTime = time.Now().Unix()
				VarSetValueStr(ctx, "$t玩家", fmt.Sprintf("<%s>", ctx.Player.Name))
				VarSetValueStr(ctx, "$t玩家_RAW", ctx.Player.Name)
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:昵称_重置"))
				if ctx.Player.AutoSetNameTemplate != "" {
					_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
				}
			default:
				p := ctx.Player
				VarSetValueStr(ctx, "$t旧昵称", fmt.Sprintf("<%s>", ctx.Player.Name))
				VarSetValueStr(ctx, "$t旧昵称_RAW", ctx.Player.Name)

				p.Name = cmdArgs.Args[0]
				p.UpdatedAtTime = time.Now().Unix()
				VarSetValueStr(ctx, "$t玩家", fmt.Sprintf("<%s>", ctx.Player.Name))
				VarSetValueStr(ctx, "$t玩家_RAW", ctx.Player.Name)

				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:昵称_改名"))
				if ctx.Player.AutoSetNameTemplate != "" {
					_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
				}
			}

			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
	d.CmdMap["nn"] = cmdNN

	d.CmdMap["userid"] = &CmdItemInfo{
		Name:      "userid",
		ShortHelp: ".userid // 查看当前帐号和群组ID",
		Help:      "查看ID:\n.userid // 查看当前帐号和群组ID",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if cmdArgs.IsArgEqual(1, "help") {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			text := fmt.Sprintf("个人账号ID为 %s", ctx.Player.UserID)
			if !ctx.IsPrivate {
				text += fmt.Sprintf("\n当前群组ID为 %s", ctx.Group.GroupID)
			}

			ReplyToSender(ctx, msg, text)
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	helpSet := ".set info// 查看当前面数设置\n" +
		".set dnd/coc // 设置群内骰子面数为20/100，并自动开启对应扩展 \n" +
		".set <面数> // 设置群内骰子面数\n" +
		".set clr // 清除群内骰子面数设置"
	cmdSet := &CmdItemInfo{
		Name:      "set",
		ShortHelp: helpSet,
		Help:      "设定骰子面数:\n" + helpSet,
		HelpFunc: func(isShort bool) string {
			text := ".set info // 查看当前面数设置\n"
			text += ".set <面数> // 设置群内骰子面数\n"
			text += ".set dnd // 设置群内骰子面数为20，并自动开启对应扩展\n"
			d.GameSystemMap.Range(func(key string, tmpl *GameSystemTemplate) bool {
				textHelp := fmt.Sprintf("设置群内骰子面数为%d，并自动开启对应扩展", tmpl.SetConfig.DiceSides)
				text += fmt.Sprintf(".set %s // %s\n", strings.Join(tmpl.SetConfig.Keys, "/"), textHelp)
				return true
			})
			text += `.set clr // 清除群内骰子面数设置`
			if isShort {
				return text
			}
			return "设定骰子面数:\n" + text
		},
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			p := ctx.Player
			isSetGroup := true
			my := cmdArgs.GetKwarg("my")
			if my != nil {
				isSetGroup = false
			}

			arg1 := cmdArgs.GetArgN(1)
			modSwitch := false
			if arg1 == "" {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			tipText := "\n提示:"
			if strings.EqualFold(arg1, "dnd") {
				cmdArgs.Args[0] = "20"
				ctx.Group.ExtActive(d.ExtFind("dnd5e"))
				tipText += "已切换至20面骰，并自动开启dnd5e扩展。"
				modSwitch = true
				ctx.Group.System = "dnd5e"
				ctx.Group.UpdatedAtTime = time.Now().Unix()
			}
			ctx.Dice.GameSystemMap.Range(func(key string, tmpl *GameSystemTemplate) bool {
				isMatch := false
				for _, k := range tmpl.SetConfig.Keys {
					if strings.EqualFold(arg1, k) {
						isMatch = true
						break
					}
				}

				if isMatch {
					modSwitch = true
					ctx.Group.System = key
					ctx.Group.DiceSideNum = tmpl.SetConfig.DiceSides
					ctx.Group.UpdatedAtTime = time.Now().Unix()
					tipText += tmpl.SetConfig.EnableTip

					// TODO: 命令该要进步啦
					cmdArgs.Args[0] = strconv.FormatInt(tmpl.SetConfig.DiceSides, 10)

					for _, name := range tmpl.SetConfig.RelatedExt {
						// 开启相关扩展
						ei := ctx.Dice.ExtFind(name)
						if ei != nil {
							ctx.Group.ExtActive(ei)
						}
					}
					return false
				}
				return true
			})

			num, err := strconv.ParseInt(cmdArgs.Args[0], 10, 64)
			if num < 0 {
				num = 0
			}
			//nolint:nestif
			if err == nil {
				if isSetGroup {
					ctx.Group.DiceSideNum = num
					if !modSwitch {
						if num == 20 {
							tipText += "20面骰。如果要进行DND游戏，建议执行.set dnd以确保开启dnd5e指令"
						} else if num == 100 {
							tipText += "100面骰。如果要进行COC游戏，建议执行.set coc以确保开启coc7指令"
						}
					}
					if tipText == "\n提示:" {
						tipText = ""
					}

					VarSetValueInt64(ctx, "$t群组骰子面数", ctx.Group.DiceSideNum)
					VarSetValueInt64(ctx, "$t当前骰子面数", getDefaultDicePoints(ctx))
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:设定默认群组骰子面数")+tipText)
				} else {
					p.DiceSideNum = int(num)
					p.UpdatedAtTime = time.Now().Unix()
					VarSetValueInt64(ctx, "$t个人骰子面数", int64(ctx.Player.DiceSideNum))
					VarSetValueInt64(ctx, "$t当前骰子面数", getDefaultDicePoints(ctx))
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:设定默认骰子面数"))
				}
			} else {
				switch arg1 {
				case "clr":
					if isSetGroup {
						ctx.Group.DiceSideNum = 0
						ctx.Group.UpdatedAtTime = time.Now().Unix()
					} else {
						p.DiceSideNum = 0
						p.UpdatedAtTime = time.Now().Unix()
					}
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:设定默认骰子面数_重置"))
				case "help":
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				case "info":
					ReplyToSender(ctx, msg, DiceFormat(ctx, `个人骰子面数: {$t个人骰子面数}\n`+
						`群组骰子面数: {$t群组骰子面数}\n当前骰子面数: {$t当前骰子面数}`))
				default:
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:设定默认骰子面数_错误"))
				}
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
	d.CmdMap["set"] = cmdSet

	helpCh := ".pc new <角色名> // 新建角色并绑卡\n" +
		".pc tag <角色名> | <角色序号> // 当前群绑卡/解除绑卡(不填角色名)\n" +
		".pc untagAll <角色名> | <角色序号> // 全部群解绑\n" +
		".pc list // 列出当前角色和序号\n" +
		// ".ch group // 列出各群当前绑卡\n" +
		".pc save <角色名> // [不绑卡]保存角色，角色名可省略\n" +
		".pc load <角色名> | <角色序号> // [不绑卡]加载角色\n" +
		".pc del/rm <角色名> | <角色序号> // 删除角色 角色序号可用pc list查询\n" +
		"> 注: 海豹各群数据独立(多张空白卡)，单群游戏不需要存角色。"
	cmdChar := &CmdItemInfo{
		Name:      "pc",
		ShortHelp: helpCh,
		Help:      "角色管理:\n" + helpCh,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			cmdArgs.ChopPrefixToArgsWith("list", "load", "save", "del", "rm", "new", "tag", "untagAll", "group1", "grp1")

			getNicknameRaw := func(usePlayerName bool) string {
				name := cmdArgs.CleanArgsChopRest
				if usePlayerName && name == "" {
					name = ctx.Player.Name
				}
				name = strings.ReplaceAll(name, "\n", "")
				name = strings.ReplaceAll(name, "\r", "")
				return name
			}

			getNickname := func() string {
				return getNicknameRaw(true)
			}

			if cmdArgs.IsArgEqual(1, "list") { //nolint:nestif
				vars := ctx.LoadPlayerGlobalVars()
				var characters []string
				curBind := ctx.ChBindCurGet()

				_ = vars.ValueMap.Iterate(func(_k interface{}, _v interface{}) error {
					k := _k.(string)
					if strings.HasPrefix(k, "$ch:") {
						name := k[4:]
						characters = append(characters, name)
					}
					return nil
				})

				// 分两次防止死锁
				var newChars []string
				for idx, name := range characters {
					// HACK(Xiangze Li): lockfree.HashMap的迭代顺序在每次启动中是稳定的, 可以加序号
					// 但是, 骰子重启之后顺序是会变化的. 如果用户记录了这个序号, 并且跨重启使用, 会出现问题
					idxStr := fmt.Sprintf("%2d ", idx+1)
					prefix := "[×] "
					if ctx.ChBindGet(name) != nil {
						prefix = "[★] "
					}
					if curBind == name {
						prefix = "[√] "
					}
					var cardType string
					vars2 := ctx.ChGet(name)
					if _cardType, ok := vars2.Get("$cardType"); ok {
						v := _cardType.(*VMValue)
						cardType = v.ToString()
					}
					suffix := ""
					if cardType != "" {
						suffix = fmt.Sprintf(" #%s", cardType)
					}

					newChars = append(newChars, idxStr+prefix+name+suffix)
				}

				if len(characters) == 0 {
					ReplyToSender(ctx, msg, fmt.Sprintf("<%s>当前还没有角色列表", ctx.Player.Name))
				} else {
					ReplyToSender(ctx, msg, fmt.Sprintf("<%s>的角色列表为:\n%s\n[√]已绑 [×]未绑 [★]其他群绑定", ctx.Player.Name, strings.Join(newChars, "\n")))
				}
			} else if cmdArgs.IsArgEqual(1, "new") {
				name := getNickname()
				if len(name) > 90 {
					name = name[:90]
				}
				if ctx.ChNew(name) {
					ctx.ChUnbindCur() // 先移除绑定
					ctx.ChBindCur(name)
					VarSetValueStr(ctx, "$t角色名", name)
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_新建"))
				} else {
					VarSetValueStr(ctx, "$t角色名", name)
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_新建_已存在"))
				}
				if ctx.Player.AutoSetNameTemplate != "" {
					_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
				}
			} else if cmdArgs.IsArgEqual(1, "load") {
				cur := ctx.ChBindCurGet()
				if cur != "" {
					VarSetValueStr(ctx, "$t角色名", cur)
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_加载失败_已绑定"))
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				name := getNickname()
				name = tryConvertIndex2Name(ctx, name)
				ret := ctx.ChLoad(name)
				VarSetValueStr(ctx, "$t角色名", name)
				if ret != nil {
					VarSetValueStr(ctx, "$t玩家", fmt.Sprintf("<%s>", ctx.Player.Name))
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_加载成功"))
					// ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_序列化失败"))
					if ctx.Player.AutoSetNameTemplate != "" {
						_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
					}
				} else {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_角色不存在"))
				}
			} else if cmdArgs.IsArgEqual(1, "tag") {
				// 当不输入角色的时候，不用当前角色填充，因此做到不写角色名就取消绑定的效果
				name := getNicknameRaw(false)
				VarSetValueStr(ctx, "$t角色名", name)
				if name != "" {
					name = tryConvertIndex2Name(ctx, name)
					VarSetValueStr(ctx, "$t角色名", name)
					curBind := ctx.ChBindCurGet()
					if curBind == name {
						// 已经绑定，直接成功
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_绑定_成功"))
					} else {
						ctx.ChUnbindCur() // 先移除绑定
						ok := ctx.ChBindCur(name)
						if ok {
							ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_绑定_成功"))
						} else {
							ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_绑定_失败"))
						}
					}
				} else {
					if name, success := ctx.ChUnbindCur(); success {
						ctx.Player.Name = msg.Sender.Nickname
						ctx.Player.UpdatedAtTime = time.Now().Unix()
						VarSetValueStr(ctx, "$t角色名", name)
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_绑定_解除"))
					} else {
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_绑定_并未绑定"))
					}
				}
				if ctx.Player.AutoSetNameTemplate != "" {
					_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
				}
			} else if cmdArgs.IsArgEqual(1, "group1", "grp1") {
				name := getNickname()
				lst := ctx.ChBindGetList(name)
				ReplyToSender(ctx, msg, "卡片绑定: "+strings.Join(lst, " "))
			} else if cmdArgs.IsArgEqual(1, "untagAll") {
				name := getNickname()
				name = tryConvertIndex2Name(ctx, name)
				lst := ctx.ChUnbind(name)

				for _, i := range lst {
					if i == ctx.Group.GroupID {
						ctx.Player.Name = msg.Sender.Nickname
						ctx.Player.UpdatedAtTime = time.Now().Unix()

						// TODO: 其他群的设置sn的怎么办？先不管了。。
						if ctx.Player.AutoSetNameTemplate != "" {
							_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
						}
					}
				}

				if len(lst) > 0 {
					ReplyToSender(ctx, msg, "绑定已全部解除:\n"+strings.Join(lst, "\n"))
				} else {
					ReplyToSender(ctx, msg, "这张卡片并未绑定到任何群")
				}
			} else if cmdArgs.IsArgEqual(1, "save") {
				name := getNickname()
				if len(name) > 90 {
					name = name[:90]
				}

				card := ctx.ChBindGet(name)
				if card == nil {
					// TODO: 改到ctx里
					chVars, _ := ctx.ChVarsGet()
					v, err := json.Marshal(LockFreeMapToMap(chVars))

					if err == nil {
						vars := ctx.LoadPlayerGlobalVars()
						vars.ValueMap.Set("$ch:"+name, &VMValue{
							TypeID: VMTypeString,
							Value:  string(v),
						})
						vars.LastWriteTime = time.Now().Unix()

						VarSetValueStr(ctx, "$t角色名", name)
						VarSetValueStr(ctx, "$t新角色名", fmt.Sprintf("<%s>", name))
						// replyToSender(ctx, msg, fmt.Sprintf("角色<%s>储存成功", Name))
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_储存成功"))
					} else {
						// replyToSender(ctx, msg, "无法储存角色：序列化失败")
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_序列化失败"))
					}
				} else {
					VarSetValueStr(ctx, "$t角色名", name)
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_储存失败_已绑定"))
				}
			} else if cmdArgs.IsArgEqual(1, "del", "rm") {
				name := getNickname()
				if ctx.ChBindGet(name) != nil {
					VarSetValueStr(ctx, "$t角色名", name)
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_删除失败_已绑定"))
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				vars := ctx.LoadPlayerGlobalVars()

				VarSetValueStr(ctx, "$t角色名", name)
				VarSetValueStr(ctx, "$t新角色名", fmt.Sprintf("<%s>", name))

				name = tryConvertIndex2Name(ctx, name)
				if _, exists := vars.ValueMap.Get("$ch:" + name); !exists {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:角色管理_角色不存在"))
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				// 如果name原是序号，这里将被更新为角色名
				VarSetValueStr(ctx, "$t角色名", name)
				VarSetValueStr(ctx, "$t新角色名", fmt.Sprintf("<%s>", name))

				vars.ValueMap.Del("$ch:" + name)
				vars.LastWriteTime = time.Now().Unix()

				text := DiceFormatTmpl(ctx, "核心:角色管理_删除成功")
				if name == ctx.Player.Name {
					VarSetValueStr(ctx, "$t新角色名", fmt.Sprintf("<%s>", msg.Sender.Nickname))
					text += "\n" + DiceFormatTmpl(ctx, "核心:角色管理_删除成功_当前卡")
					p := ctx.Player
					p.Name = msg.Sender.Nickname
					p.UpdatedAtTime = time.Now().Unix()
					p.Vars.ValueMap = lockfree.NewHashMap()
					p.Vars.LastWriteTime = time.Now().Unix()
				}
				ReplyToSender(ctx, msg, text)
			} else {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
	d.CmdMap["角色"] = cmdChar
	d.CmdMap["ch"] = cmdChar
	d.CmdMap["char"] = cmdChar
	d.CmdMap["character"] = cmdChar
	d.CmdMap["pc"] = cmdChar

	botWelcomeHelp := ".welcome on // 开启\n" +
		".welcome off // 关闭\n" +
		".welcome show // 查看当前欢迎语\n" +
		".welcome set <欢迎语> // 设定欢迎语"
	cmdWelcome := &CmdItemInfo{
		Name:              "welcome",
		ShortHelp:         botWelcomeHelp,
		Help:              "新人入群自动发言设定:\n" + botWelcomeHelp,
		DisabledInPrivate: true,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			pRequired := 50 // 50管理 60群主 100master
			if ctx.PrivilegeLevel < pRequired {
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_无权限_非master/管理"))
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			if cmdArgs.IsArgEqual(1, "on") {
				ctx.Group.ShowGroupWelcome = true
				ctx.Group.UpdatedAtTime = time.Now().Unix()
				ReplyToSender(ctx, msg, "入群欢迎语已打开")
			} else if cmdArgs.IsArgEqual(1, "off") {
				ctx.Group.ShowGroupWelcome = false
				ctx.Group.UpdatedAtTime = time.Now().Unix()
				ReplyToSender(ctx, msg, "入群欢迎语已关闭")
			} else if cmdArgs.IsArgEqual(1, "show") {
				welcome := ctx.Group.GroupWelcomeMessage
				var info string
				if ctx.Group.ShowGroupWelcome {
					info = "\n状态: 开启"
				} else {
					info = "\n状态: 关闭"
				}
				ReplyToSender(ctx, msg, "当前欢迎语:\n"+welcome+info)
			} else if _, ok := cmdArgs.EatPrefixWith("set"); ok {
				text2 := strings.TrimSpace(cmdArgs.RawArgs[len("set"):])
				ctx.Group.GroupWelcomeMessage = text2
				ctx.Group.ShowGroupWelcome = true
				ctx.Group.UpdatedAtTime = time.Now().Unix()
				ReplyToSender(ctx, msg, "当前欢迎语设定为:\n"+text2+"\n入群欢迎语已自动打开(注意，会在bot off时起效)")
			} else {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
	d.CmdMap["welcome"] = cmdWelcome

	cmdReply := &CmdItemInfo{
		Name:      "reply",
		ShortHelp: ".reply on/off",
		Help:      "打开或关闭自定义回复:\n.reply on/off",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			val := cmdArgs.GetArgN(1)
			switch val {
			case "on":
				onText := "开"
				if ctx.Group.ExtGetActive("reply") == nil {
					onText = "关"
				}
				extReply := ctx.Dice.ExtFind("reply")
				ctx.Group.ExtActive(extReply)
				ReplyToSender(ctx, msg, fmt.Sprintf("已在当前群开启自定义回复(%s➯开)。\n此指令等价于.ext reply on", onText))
			case "off":
				onText := "开"
				if ctx.Group.ExtGetActive("reply") == nil {
					onText = "关"
				}
				ctx.Group.ExtInactiveByName("reply")
				ReplyToSender(ctx, msg, fmt.Sprintf("已在当前群关闭自定义回复(%s➯关)。\n此指令等价于.ext reply off", onText))
			default:
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
	d.CmdMap["reply"] = cmdReply

	cmdPing := &CmdItemInfo{
		Name:      "ping",
		ShortHelp: ".ping // 触发发送一条回复",
		Help:      "触发回复:\n触发发送一条回复。特别地，如果是qq官方bot，并且是在频道中触发，会以私信消息形式回复",
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if cmdArgs.IsArgEqual(1, "help") {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			if msg.Platform == "OpenQQCH" &&
				strings.HasPrefix(msg.GuildID, "OpenQQCH-Guild:") &&
				strings.HasPrefix(msg.GroupID, "OpenQQCH-Channel:") {
				// 从 official qq 的频道触发的，就触发私信的回复
				ReplyPerson(ctx, msg, DiceFormatTmpl(ctx, "其它:ping响应"))
			} else {
				// 其它的情况就直接回复
				ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "其它:ping响应"))
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
	d.CmdMap["ping"] = cmdPing

	aliasHelp := ".alias <别名> <指令> // 将 .&<别名> 定义为指定指令的快捷触发方式\n" +
		".alias --my <别名> <指令> // 将 .&<别名> 定义为个人快捷指令\n" +
		".alias del/rm <别名> // 删除群快捷指令\n" +
		".alias del/rm --my <别名> // 删除个人快捷指令\n" +
		".alias show/list // 显示目前可用的快捷指令\n" +
		".alias help // 查看帮助\n" +
		"// 执行快捷命令见 .& 命令"
	cmdAlias := &CmdItemInfo{
		Name:      "alias",
		ShortHelp: aliasHelp,
		Help:      "可以定义一条指令的快捷方式。\n" + aliasHelp,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if len(cmdArgs.Args) == 0 {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			_isPersonal := cmdArgs.GetKwarg("my")
			isPersonal := ctx.MessageType == "private" || _isPersonal != nil
			if !isPersonal {
				ctx.LoadGroupVars()
			}

			playerVars := ctx.LoadPlayerGlobalVars()
			subCmd := cmdArgs.GetArgN(1)
			switch subCmd {
			case "help":
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			case "del", "rm":
				name := cmdArgs.GetArgN(2)
				key := "$g:alias:" + name
				m := ctx.Group.ValueMap
				VarSetValueStr(ctx, "$t指令来源", "群")
				if isPersonal {
					key = "$m:alias:" + name
					m = playerVars.ValueMap
					VarSetValueStr(ctx, "$t指令来源", "个人")
				}
				if _cmd, ok := m.Get(key); ok {
					if cmd, ok := _cmd.(*VMValue); ok && cmd != nil && cmd.TypeID == VMTypeString {
						VarSetValueStr(ctx, "$t快捷指令名", name)
						VarSetValueStr(ctx, "$t旧指令", cmd.Value.(string))
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:快捷指令_删除"))
					}
					m.Del(key)
					if isPersonal {
						playerVars.LastWriteTime = time.Now().Unix()
					} else if ctx.Group != nil {
						ctx.Group.UpdatedAtTime = time.Now().Unix()
					}
				} else {
					VarSetValueStr(ctx, "$t快捷指令名", name)
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:快捷指令_删除_未定义"))
				}
			case "list", "show":
				var personCmds, groupCmds []string
				_ = playerVars.ValueMap.Iterate(func(k interface{}, v interface{}) error {
					if key, ok := k.(string); ok {
						if strings.HasPrefix(key, "$m:alias:") {
							_cmd := key[len("$m:alias:"):]
							if val, ok := v.(*VMValue); ok && val != nil && val.TypeID == VMTypeString {
								VarSetValueStr(ctx, "$t快捷指令名", _cmd)
								VarSetValueStr(ctx, "$t指令", val.Value.(string))
								VarSetValueStr(ctx, "$t指令来源", "个人")
								personCmds = append(personCmds, DiceFormatTmpl(ctx, "核心:快捷指令_列表_单行"))
							}
						}
					}
					return nil
				})
				if ctx.MessageType == "group" {
					groupValueMap := ctx.Group.ValueMap
					_ = groupValueMap.Iterate(func(k interface{}, v interface{}) error {
						if key, ok := k.(string); ok {
							if strings.HasPrefix(key, "$g:alias:") {
								_cmd := key[len("$g:alias:"):]
								if val, ok := v.(*VMValue); ok && val != nil && val.TypeID == VMTypeString {
									VarSetValueStr(ctx, "$t快捷指令名", _cmd)
									VarSetValueStr(ctx, "$t指令", val.Value.(string))
									VarSetValueStr(ctx, "$t指令来源", "群")
									groupCmds = append(groupCmds, DiceFormatTmpl(ctx, "核心:快捷指令_列表_单行"))
								}
							}
						}
						return nil
					})
				}
				sep := DiceFormatTmpl(ctx, "核心:快捷指令_列表_分隔符")
				// 保证群在前个人在后的顺序
				var totalCmds []string
				totalCmds = append(totalCmds, groupCmds...)
				totalCmds = append(totalCmds, personCmds...)
				if len(totalCmds) > 0 {
					VarSetValueStr(ctx, "$t列表内容", strings.Join(totalCmds, sep))
				}

				if len(totalCmds) == 0 {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:快捷指令_列表_空"))
				} else {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:快捷指令_列表"))
				}
			default:
				if len(cmdArgs.Args) < 2 {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:快捷指令_新增_无指令"))
					break
				}
				name := subCmd
				_args := cmdArgs.Args[1:]
				for _, kwa := range cmdArgs.Kwargs {
					if kwa.Name != "my" {
						_args = append(_args, kwa.String())
					}
				}
				cmd := strings.TrimSpace(strings.Join(_args, " "))

				m := ctx.Group.ValueMap
				key := "$g:alias:" + name
				VarSetValueStr(ctx, "$t指令来源", "群")
				if isPersonal {
					key = "$m:alias:" + name
					m = playerVars.ValueMap
					VarSetValueStr(ctx, "$t指令来源", "个人")
				}

				if _oldCmd, ok := m.Get(key); ok && _oldCmd != nil {
					if oldCmd, ok := _oldCmd.(*VMValue); ok && oldCmd.TypeID == VMTypeString {
						m.Set(key, &VMValue{TypeID: VMTypeString, Value: cmd})
						if isPersonal {
							playerVars.LastWriteTime = time.Now().Unix()
						} else if ctx.Group != nil {
							ctx.Group.UpdatedAtTime = time.Now().Unix()
						}
						VarSetValueStr(ctx, "$t快捷指令名", name)
						VarSetValueStr(ctx, "$t指令", cmd)
						VarSetValueStr(ctx, "$t旧指令", oldCmd.Value.(string))
						ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:快捷指令_替换"))
					} else {
						// 防止错误的数据一直卡着
						m.Del(key)
						if isPersonal {
							playerVars.LastWriteTime = time.Now().Unix()
						} else if ctx.Group != nil {
							ctx.Group.UpdatedAtTime = time.Now().Unix()
						}
					}
				} else {
					m.Set(key, &VMValue{TypeID: VMTypeString, Value: cmd})
					if isPersonal {
						playerVars.LastWriteTime = time.Now().Unix()
					} else if ctx.Group != nil {
						ctx.Group.UpdatedAtTime = time.Now().Unix()
					}
					VarSetValueStr(ctx, "$t快捷指令名", name)
					VarSetValueStr(ctx, "$t指令", cmd)
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:快捷指令_新增"))
				}
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
	d.CmdMap["alias"] = cmdAlias

	aHelp := ".&/a <快捷指令名> [参数] // 执行对应快捷指令\n" +
		".& help // 查看帮助\n" +
		"// 定义快捷指令见 .alias 命令"
	cmdA := &CmdItemInfo{
		Name:      "&",
		ShortHelp: aHelp,
		Help:      "执行一条快捷指令。\n" + aHelp,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			if len(cmdArgs.Args) == 0 {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			name := cmdArgs.GetArgN(1)
			if name == "help" {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			log := d.Logger
			args := cmdArgs.Args
			for _, kwa := range cmdArgs.Kwargs {
				args = append(args, kwa.String())
			}

			if msg.MessageType == "group" {
				ctx.LoadGroupVars()
				_cmdValue, ok := ctx.Group.ValueMap.Get("$g:alias:" + name)
				if ok {
					if cmdValue, ok2 := _cmdValue.(*VMValue); ok2 && cmdValue != nil && cmdValue.TypeID == VMTypeString {
						args[0] = cmdValue.Value.(string)
						targetCmd := strings.Join(args, " ")
						targetArgs := CommandParse(targetCmd, []string{}, d.CommandPrefix, msg.Platform, false)
						if targetArgs != nil {
							log.Infof("群快捷指令映射: .&%s -> %s", cmdArgs.CleanArgs, targetCmd)

							VarSetValueStr(ctx, "$t指令来源", "群")
							VarSetValueStr(ctx, "$t目标指令", targetCmd)
							ctx.AliasPrefixText = DiceFormatTmpl(ctx, "核心:快捷指令触发_前缀")

							ctx.EndPoint.TriggerCommand(ctx, msg, targetArgs)
							return CmdExecuteResult{Matched: true, Solved: true}
						}
					}
				}
			}

			playerVars := ctx.LoadPlayerGlobalVars()
			_cmdValue, ok := playerVars.ValueMap.Get("$m:alias:" + name)
			if ok {
				if cmdValue, ok := _cmdValue.(*VMValue); ok && cmdValue != nil && cmdValue.TypeID == VMTypeString {
					args[0] = cmdValue.Value.(string)
					targetCmd := strings.Join(args, " ")
					msg.Message = targetCmd
					targetArgs := CommandParse(targetCmd, []string{}, d.CommandPrefix, msg.Platform, false)
					if targetArgs != nil {
						log.Infof("个人快捷指令映射: .&%s -> %s", cmdArgs.CleanArgs, targetCmd)

						VarSetValueStr(ctx, "$t指令来源", "个人")
						VarSetValueStr(ctx, "$t目标指令", targetCmd)
						ctx.AliasPrefixText = DiceFormatTmpl(ctx, "核心:快捷指令触发_前缀")

						ctx.EndPoint.TriggerCommand(ctx, msg, targetArgs)
						return CmdExecuteResult{Matched: true, Solved: true}
					}
				}
			}

			VarSetValueStr(ctx, "$t目标指令名", name)
			ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:快捷指令触发_无指令"))
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}
	d.CmdMap["&"] = cmdA
	d.CmdMap["a"] = cmdA
}

func getDefaultDicePoints(ctx *MsgContext) int64 {
	diceSides := int64(ctx.Player.DiceSideNum)
	if diceSides == 0 && ctx.Group != nil {
		diceSides = ctx.Group.DiceSideNum
	}
	if diceSides <= 0 {
		diceSides = 100
	}
	return diceSides
}

func setRuleByName(ctx *MsgContext, name string) {
	if name == "" {
		return
	}
	diceFaces := ""
	d := ctx.Dice

	modSwitch := false

	tipText := "\n提示:"

	if strings.EqualFold(name, "dnd") {
		diceFaces = "20"
		ctx.Group.ExtActive(d.ExtFind("dnd5e"))
		tipText += "已切换至20面骰，并自动开启dnd5e扩展。若不希望，请执行.ext dnd5e off"
		modSwitch = true
		ctx.Group.System = "dnd5e"
		ctx.Group.UpdatedAtTime = time.Now().Unix()
	}
	d.GameSystemMap.Range(func(key string, tmpl *GameSystemTemplate) bool {
		isMatch := false
		for _, k := range tmpl.SetConfig.Keys {
			if strings.EqualFold(name, k) {
				isMatch = true
				break
			}
		}

		if isMatch {
			modSwitch = true
			ctx.Group.System = key
			ctx.Group.DiceSideNum = tmpl.SetConfig.DiceSides
			ctx.Group.UpdatedAtTime = time.Now().Unix()
			tipText += tmpl.SetConfig.EnableTip

			// TODO: 命令该要进步啦
			diceFaces = strconv.FormatInt(tmpl.SetConfig.DiceSides, 10)

			for _, name := range tmpl.SetConfig.RelatedExt {
				// 开启相关扩展
				ei := ctx.Dice.ExtFind(name)
				if ei != nil {
					ctx.Group.ExtActive(ei)
				}
			}
			return false
		}
		return true
	})

	num, err := strconv.ParseInt(diceFaces, 10, 64)
	if num < 0 {
		num = 0
	}
	if err == nil {
		ctx.Group.DiceSideNum = num
		if !modSwitch {
			if num == 20 {
				tipText += "20面骰。如果要进行DND游戏，建议执行.set dnd以确保开启dnd5e指令"
			} else if num == 100 {
				tipText += "100面骰。如果要进行COC游戏，建议执行.set coc以确保开启coc7指令"
			}
		}
		if tipText == "\n提示:" {
			tipText = ""
		}
	}
}

// tryConvertIndex2Name 确认name是否存在, 如果不存在, 尝试将name解析为序号并查出对应name
//   - 如果name存在, 直接返回name
//   - name不存在, name不是数字, 返回原name
//   - name不存在, name是数字, 且超过用户的角色卡数, 返回原name
//   - name不存在, name是数字, 且小于等于用户的角色卡数, 返回对应序号的卡名
func tryConvertIndex2Name(ctx *MsgContext, name string) string {
	// 确认name是否存在, 如果不存在, 尝试解析为序号并查出对应name
	vars := ctx.LoadPlayerGlobalVars()
	_, exists := vars.ValueMap.Get("$ch:" + name)
	if !exists {
		if idx, errInt := strconv.ParseInt(name, 10, 64); errInt == nil && idx > 0 {
			_ = vars.ValueMap.Iterate(func(_k interface{}, _v interface{}) error {
				k := _k.(string)
				if strings.HasPrefix(k, "$ch:") {
					idx--
					if idx == 0 {
						name = k[4:]
						return fmt.Errorf("break")
					}
				}
				return nil
			})
		}
	}
	return name
}
