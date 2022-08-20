package dice

import (
	"archive/zip"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/fy0/lockfree"
	"go.etcd.io/bbolt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type LogOneItem struct {
	Id          uint64      `json:"id"`
	Nickname    string      `json:"nickname"`
	IMUserId    string      `json:"IMUserId"`
	Time        int64       `json:"time"`
	Message     string      `json:"message"`
	IsDice      bool        `json:"isDice"`
	CommandId   uint64      `json:"commandId"`
	CommandInfo interface{} `json:"commandInfo"`
	RawMsgId    interface{} `json:"rawMsgId"`

	UniformId string `json:"uniformId"`
	Channel   string `json:"channel"` // 用于秘密团
}

func SetPlayerGroupCardByTemplate(ctx *MsgContext, tmpl string) (string, error) {
	ctx.Player.TempValueAlias = nil // 防止dnd的hp被转为“生命值”
	val, _, err := ctx.Dice.ExprTextBase(tmpl, ctx, RollExtraFlags{
		CocDefaultAttrOn: true,
	})
	if err != nil {
		ctx.Dice.Logger.Infof("SN指令模板错误: %v", err.Error())
		return "", err
	}

	var text string
	if err == nil && (val.TypeId == VMTypeString || val.TypeId == VMTypeNone) {
		text = val.Value.(string)
	}

	ctx.EndPoint.Adapter.SetGroupCardName(ctx.Group.GroupId, ctx.Player.UserId, text)
	return text, nil
}

// {"data":null,"msg":"SEND_MSG_API_ERROR","retcode":100,"status":"failed","wording":"请参考 go-cqhttp 端输出"}

func RegisterBuiltinExtLog(self *Dice) {
	privateCommandListen := map[uint64]int64{}

	// 这个机制作用是记录私聊指令？？忘记了
	privateCommandListenCheck := func() {
		now := time.Now().Unix()
		newMap := map[uint64]int64{}
		for k, v := range privateCommandListen {
			// 30s间隔以上清除
			if now-v < 30 {
				newMap[k] = v
			}
		}
		privateCommandListen = newMap
	}

	// 避免群信息重复记录
	groupMsgInfo := lockfree.NewHashMap()
	groupMsgInfoLastClean := int64(0)
	groupMsgInfoClean := func() {
		// 清理过久的消息
		now := time.Now().Unix()

		if now-groupMsgInfoLastClean < 60 {
			// 60s清理一次
			return
		}

		groupMsgInfoLastClean = now
		toDelete := []interface{}{}
		_ = groupMsgInfo.Iterate(func(_k interface{}, _v interface{}) error {
			t, ok := _v.(int64)
			if ok {
				if now-t > 5 { // 5秒内如果有此消息，那么不记录
					toDelete = append(toDelete, _k)
				}
			} else {
				toDelete = append(toDelete, _k)
			}
			return nil
		})

		for _, i := range toDelete {
			groupMsgInfo.Del(i)
		}
	}

	// 检查是否已经记录过 如果记录过则跳过
	groupMsgInfoCheckOk := func(_k interface{}) bool {
		groupMsgInfoClean()
		_val, exists := groupMsgInfo.Get(_k)
		if exists {
			t, ok := _val.(int64)
			if ok {
				now := time.Now().Unix()
				return now-t > 5 // 5秒内如果有此消息，那么不记录
			}
		}
		return true
	}

	groupMsgInfoSet := func(_k interface{}) {
		if _k != nil {
			groupMsgInfo.Set(_k, time.Now().Unix())
		}
	}

	// 获取logname，第一项是默认名字
	getLogName := func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs, index int) (string, string) {
		bakLogCurName := ctx.Group.LogCurName
		if newName, exists := cmdArgs.GetArgN(index); exists {
			if exists {
				return bakLogCurName, newName
			}
		}
		return bakLogCurName, bakLogCurName
	}

	helpLog := `.log new (<日志名>) // 新建日志并开始记录，注意new后跟空格！
.log on (<日志名>)  // 开始记录，不写日志名则开启最近一次日志，注意on后跟空格！
.log off // 暂停记录
.log end // 完成记录并发送日志文件
.log get (<日志名>) // 重新上传日志，并获取链接
.log halt // 强行关闭当前log，不上传日志
.log list // 查看当前群的日志列表
.log del <日志名> // 删除一份日志
.log stat (<日志名>) // 查看统计
.log stat (<日志名>) --all // 查看统计(全团)，--all前必须有空格
.log list <群号> // 查看指定群的日志列表(无法取得日志时，找骰主做这个操作)
.log masterget <群号> <日志名> // 重新上传日志，并获取链接(无法取得日志时，找骰主做这个操作)`

	txtLogTip := "若未出现线上日志地址，可换时间获取，或联系骰主在data/default/logs路径下取出日志\n文件名: 群号_日志名_随机数.zip\n注意此文件log end/get后才会生成"

	cmdLog := &CmdItemInfo{
		Name:      "log",
		ShortHelp: helpLog,
		Help:      "日志指令:\n" + helpLog,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			group := ctx.Group
			cmdArgs.ChopPrefixToArgsWith("on", "off", "new", "end", "del", "halt")

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
				lines, _ := LogLinesGet(ctx, group, group.LogCurName)
				text := fmt.Sprintf("当前故事: %s\n当前状态: %s\n已记录文本%d条", group.LogCurName, onText, lines)
				ReplyToSender(ctx, msg, text)
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			if cmdArgs.IsArgEqual(1, "on") {
				if ctx.IsPrivate {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_私聊不可用"))
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				name, _ := cmdArgs.GetArgN(2)
				if name == "" {
					name = group.LogCurName
				}

				if name != "" {
					lines, exists := LogLinesGet(ctx, group, name)

					if exists {
						if groupNotActiveCheck() {
							return CmdExecuteResult{Matched: true, Solved: true}
						}

						group.LogOn = true
						group.LogCurName = name

						VarSetValueStr(ctx, "$t记录名称", name)
						VarSetValueInt64(ctx, "$t当前记录条数", int64(lines))
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
				if group.LogCurName != "" {
					group.LogOn = false
					lines, _ := LogLinesGet(ctx, group, group.LogCurName)
					VarSetValueStr(ctx, "$t记录名称", group.LogCurName)
					VarSetValueInt64(ctx, "$t当前记录条数", int64(lines))
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "日志:记录_关闭_成功"))
				} else {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "日志:记录_关闭_失败"))
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			} else if cmdArgs.IsArgEqual(1, "del", "rm") {
				name, _ := cmdArgs.GetArgN(2)
				if name != "" {
					if name == group.LogCurName {
						ReplyToSender(ctx, msg, "不能删除正在进行的log，请用log new开启新的，或log end结束后再行删除")
					} else {
						ok := LogDelete(ctx, group, name)
						if ok {
							ReplyToSender(ctx, msg, "删除log成功")
						} else {
							ReplyToSender(ctx, msg, "删除log失败，可能是名字不对？")
						}
					}
				} else {
					return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			} else if cmdArgs.IsArgEqual(1, "masterget") {
				newGroup, requestForAnotherGroup := getSpecifiedGroupIfMaster(ctx, msg, cmdArgs)
				if requestForAnotherGroup {
					if newGroup == nil {
						return CmdExecuteResult{Matched: true, Solved: true}
					}
					group = newGroup
				}

				bakLogCurName := group.LogCurName
				if newName, exists := cmdArgs.GetArgN(3); exists {
					if exists {
						group.LogCurName = newName
					}
				}

				if group.LogCurName != "" {
					fn := LogSendToBackend(ctx, group)
					if fn == "" {
						ReplyToSenderRaw(ctx, msg, txtLogTip, "skip")
					} else {
						ReplyToSenderRaw(ctx, msg, fmt.Sprintf("跑团日志已上传服务器，链接如下：\n%s", fn), "skip")
						time.Sleep(time.Duration(0.3 * float64(time.Second)))
						ReplyToSenderRaw(ctx, msg, txtLogTip, "skip")
					}
				}
				group.LogCurName = bakLogCurName
				return CmdExecuteResult{Matched: true, Solved: true}
			} else if cmdArgs.IsArgEqual(1, "get") {
				bakLogCurName := group.LogCurName
				if newName, exists := cmdArgs.GetArgN(2); exists {
					if exists {
						group.LogCurName = newName
					}
				}

				if group.LogCurName != "" {
					fn := LogSendToBackend(ctx, group)
					if fn == "" {
						ReplyToSenderRaw(ctx, msg, txtLogTip, "skip")
					} else {
						ReplyToSenderRaw(ctx, msg, fmt.Sprintf("跑团日志已上传服务器，链接如下：\n%s", fn), "skip")
						time.Sleep(time.Duration(0.3 * float64(time.Second)))
						ReplyToSenderRaw(ctx, msg, txtLogTip, "skip")
					}
				}
				group.LogCurName = bakLogCurName
				return CmdExecuteResult{Matched: true, Solved: true}
			} else if cmdArgs.IsArgEqual(1, "end") {
				text := DiceFormatTmpl(ctx, "日志:记录_结束")
				ReplyToSender(ctx, msg, text)
				group.LogOn = false

				time.Sleep(time.Duration(0.3 * float64(time.Second)))
				fn := LogSendToBackend(ctx, group)
				if fn == "" {
					text := fmt.Sprintf("跑团日志上传失败，可换时间获取，或联系骰主在data/default/logs路径下取出日志\n文件名: 群号_日志名_随机数.zip\n注意此文件log end/get后才会生成")
					ReplyToSenderRaw(ctx, msg, text, "skip")
				} else {
					ReplyToSenderRaw(ctx, msg, fmt.Sprintf("跑团日志已上传服务器，链接如下：\n%s", fn), "skip")
					time.Sleep(time.Duration(0.3 * float64(time.Second)))
					ReplyToSenderRaw(ctx, msg, txtLogTip, "skip")
				}
				group.LogCurName = ""
				return CmdExecuteResult{Matched: true, Solved: true}
			} else if cmdArgs.IsArgEqual(1, "halt") {
				text := DiceFormatTmpl(ctx, "日志:记录_结束")
				ReplyToSender(ctx, msg, text)
				group.LogOn = false
				group.LogCurName = ""
				return CmdExecuteResult{Matched: true, Solved: true}
			} else if cmdArgs.IsArgEqual(1, "list") {
				newGroup, requestForAnotherGroup := getSpecifiedGroupIfMaster(ctx, msg, cmdArgs)
				if requestForAnotherGroup {
					if newGroup == nil {
						return CmdExecuteResult{Matched: true, Solved: true}
					}
					group = newGroup
				}

				text := DiceFormatTmpl(ctx, "日志:记录_列出_导入语") + "\n"
				lst, err := LogGetList(ctx, group)
				if err == nil {
					for _, i := range lst {
						text += "- " + i + "\n"
					}
					if len(lst) == 0 {
						text += "暂无记录"
					}
				} else {
					text += "获取记录出错，请联系骰主查看服务日志"
				}
				ReplyToSender(ctx, msg, text)
				return CmdExecuteResult{Matched: true, Solved: true}
			} else if cmdArgs.IsArgEqual(1, "new") {
				if ctx.IsPrivate {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "核心:提示_私聊不可用"))
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				name, _ := cmdArgs.GetArgN(2)

				if group.LogCurName != "" && name == "" {
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "日志:记录_新建_失败_未结束的记录"))
				} else {
					if groupNotActiveCheck() {
						return CmdExecuteResult{Matched: true, Solved: true}
					}

					if name == "" {
						todayTime := time.Now().Format("2006_01_02_15_04_05")
						name = todayTime
					}
					VarSetValueStr(ctx, "$t记录名称", name)

					group.LogCurName = name
					group.LogOn = true
					ReplyToSender(ctx, msg, DiceFormatTmpl(ctx, "日志:记录_新建"))
				}
				return CmdExecuteResult{Matched: true, Solved: true}
			} else if cmdArgs.IsArgEqual(1, "stat") {
				group := ctx.Group
				_, name := getLogName(ctx, msg, cmdArgs, 2)
				items, err := LogGetAllLinesWithoutDeleted(ctx, group, name)
				if err == nil && len(items) > 0 {
					//showDetail := cmdArgs.GetKwarg("detail")
					var showDetail *Kwarg
					showAll := cmdArgs.GetKwarg("all")

					if showDetail != nil {
						results := LogRollBriefDetail(items)

						if len(results) > 0 {
							ReplyToSender(ctx, msg, "统计结果如下:\n"+strings.Join(results, "\n"))
							return CmdExecuteResult{Matched: true, Solved: true}
						}
					} else {
						isShowAll := showAll != nil
						text := LogRollBriefByPC(ctx.Dice, items, isShowAll, ctx.Player.Name)
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
			} else {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
		},
	}

	helpStat := `.stat log (<日志名>) // 查看当前或指定日志的骰点统计
.stat log (<日志名>) --all // 查看全团
.stat help // 帮助
`
	cmdStat := &CmdItemInfo{
		Name:      "stat",
		ShortHelp: helpStat,
		Help:      "查看统计:\n" + helpStat,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			val, _ := cmdArgs.GetArgN(1)
			switch strings.ToLower(val) {
			case "log":
				group := ctx.Group
				_, name := getLogName(ctx, msg, cmdArgs, 2)
				items, err := LogGetAllLinesWithoutDeleted(ctx, group, name)
				if err == nil && len(items) > 0 {
					//showDetail := cmdArgs.GetKwarg("detail")
					var showDetail *Kwarg
					showAll := cmdArgs.GetKwarg("all")

					if showDetail != nil {
						results := LogRollBriefDetail(items)

						if len(results) > 0 {
							ReplyToSender(ctx, msg, "统计结果如下:\n"+strings.Join(results, "\n"))
							return CmdExecuteResult{Matched: true, Solved: true}
						}
					} else {
						isShowAll := showAll != nil
						text := LogRollBriefByPC(ctx.Dice, items, isShowAll, ctx.Player.Name)
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
.stat help // 帮助
`
	cmdOb := &CmdItemInfo{
		Name:      "ob",
		ShortHelp: helpOb,
		Help:      "观众指令:\n" + helpOb,
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			val, _ := cmdArgs.GetArgN(1)
			switch strings.ToLower(val) {
			case "help":
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			case "exit":
				if strings.HasPrefix(strings.ToLower(ctx.Player.Name), "ob") {
					ctx.Player.Name = ctx.Player.Name[len("ob"):]
				}
				ctx.EndPoint.Adapter.SetGroupCardName(ctx.Group.GroupId, ctx.Player.UserId, ctx.Player.Name)
				ReplyToSender(ctx, msg, "你不再是观众了（自动修改昵称和群名片[如有权限]）。")
			default:
				if !strings.HasPrefix(strings.ToLower(ctx.Player.Name), "ob") {
					ctx.Player.Name = "ob" + ctx.Player.Name
				}
				ctx.EndPoint.Adapter.SetGroupCardName(ctx.Group.GroupId, ctx.Player.UserId, ctx.Player.Name)
				ReplyToSender(ctx, msg, "你将成为观众（自动修改昵称和群名片[如有权限]，并不会给观众发送暗骰结果）。")
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
		Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
			val, _ := cmdArgs.GetArgN(1)
			switch strings.ToLower(val) {
			case "help":
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			case "coc", "coc7":
				ctx.Player.AutoSetNameTemplate = "{$t玩家_RAW} SAN{理智} HP{生命值}/{生命值上限} DEX{敏捷}"
				text, _ := SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
				// 玩家 SAN60 HP10/10 DEX65
				ReplyToSender(ctx, msg, "已自动设置名片为COC7格式: "+text+"\n如有权限会持续自动改名片。使用.sn off可关闭")
			case "dnd", "dnd5e":
				// PW{pw}
				ctx.Player.AutoSetNameTemplate = "{$t玩家_RAW} HP{hp}/{hpmax} AC{ac} DC{dc} PW{_pw}"
				text, _ := SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
				// 玩家 HP10/10 AC15 DC15 PW10
				ReplyToSender(ctx, msg, "已自动设置名片为DND5E格式: "+text+"\n使用.sn off可关闭")
			case "none":
				ctx.Player.AutoSetNameTemplate = "{$t玩家_RAW}"
				text, _ := SetPlayerGroupCardByTemplate(ctx, "{$t玩家_RAW}")
				ReplyToSender(ctx, msg, "已自动设置名片为空白格式: "+text+"\n使用.sn off可关闭")
			case "off", "cancel":
				ctx.Player.AutoSetNameTemplate = ""
				ReplyToSender(ctx, msg, "已关闭自动设置名片功能")
			default:
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		},
	}

	self.ExtList = append(self.ExtList, &ExtInfo{
		Name:       "log",
		Version:    "1.0.1",
		Brief:      "跑团辅助扩展，提供日志、染色等功能",
		Author:     "木落",
		AutoActive: true,
		OnLoad: func() {
			os.MkdirAll(filepath.Join(self.BaseConfig.DataDir, "logs"), 0755)
			self.DB.Update(func(tx *bbolt.Tx) error {
				_, err := tx.CreateBucketIfNotExists([]byte("logs"))
				return err
			})
		},
		OnMessageSend: func(ctx *MsgContext, messageType string, userId string, text string, flag string) {
			// 记录骰子发言
			if flag == "skip" {
				return
			}
			privateCommandListenCheck()

			if messageType == "private" && ctx.CommandHideFlag != "" {
				if _, exists := privateCommandListen[ctx.CommandId]; exists {
					session := ctx.Session
					group := session.ServiceAtNew[ctx.CommandHideFlag]

					a := LogOneItem{
						Nickname:    ctx.EndPoint.Nickname,
						IMUserId:    UserIdExtract(ctx.EndPoint.UserId),
						UniformId:   ctx.EndPoint.UserId,
						Time:        time.Now().Unix(),
						Message:     text,
						IsDice:      true,
						CommandId:   ctx.CommandId,
						CommandInfo: ctx.CommandInfo,
					}
					LogAppend(ctx, group, &a)
				}
			}

			if IsCurGroupBotOnById(ctx.Session, ctx.EndPoint, messageType, userId) {
				session := ctx.Session
				group := session.ServiceAtNew[userId]
				if group.LogOn {
					// <2022-02-15 09:54:14.0> [摸鱼king]: 有的 但我不知道
					if ctx.CommandHideFlag != "" {
						// 记录当前指令和时间
						privateCommandListen[ctx.CommandId] = time.Now().Unix()
					}

					a := LogOneItem{
						Nickname:    ctx.EndPoint.Nickname,
						IMUserId:    UserIdExtract(ctx.EndPoint.UserId),
						UniformId:   ctx.EndPoint.UserId,
						Time:        time.Now().Unix(),
						Message:     text,
						IsDice:      true,
						CommandId:   ctx.CommandId,
						CommandInfo: ctx.CommandInfo,
					}
					LogAppend(ctx, group, &a)
				}
			}
		},
		OnMessageReceived: func(ctx *MsgContext, msg *Message) {
			// 处理日志
			if ctx.Group != nil {
				if ctx.Group.LogOn {
					// 去重，用于同群多骰情况
					if !groupMsgInfoCheckOk(msg.RawId) {
						return
					}
					groupMsgInfoSet(msg.RawId)

					// <2022-02-15 09:54:14.0> [摸鱼king]: 有的 但我不知道
					a := LogOneItem{
						Nickname:  ctx.Player.Name,
						IMUserId:  UserIdExtract(ctx.Player.UserId),
						UniformId: ctx.Player.UserId,
						Time:      msg.Time,
						Message:   msg.Message,
						IsDice:    false,
						CommandId: ctx.CommandId,
						RawMsgId:  msg.RawId,
					}

					LogAppend(ctx, ctx.Group, &a)
				}
			}
		},
		GetDescText: func(ei *ExtInfo) string {
			return GetExtensionDesc(ei)
		},
		CmdMap: CmdMapCls{
			"log":  cmdLog,
			"stat": cmdStat,
			"hiy":  cmdStat,
			"ob":   cmdOb,
			"sn":   cmdSn,
		},
	})
}

func getSpecifiedGroupIfMaster(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) (*GroupInfo, bool) {
	if data, exists := cmdArgs.GetArgN(2); exists {
		if ctx.PrivilegeLevel < 100 {
			ReplyToSender(ctx, msg, "你并非Master，请检查指令输入是否正确")
			return nil, true
		}

		var prefix string
		if ctx.EndPoint.Platform == "QQ" {
			prefix = "QQ-Group"
		}
		if !strings.HasPrefix(data, prefix) {
			data = prefix + ":" + data
		}

		_newGroup := ctx.Session.ServiceAtNew[data]
		if _newGroup == nil {
			ReplyToSender(ctx, msg, "找不到指定的群组，请输入正确群号。如在非QQ平台取log，群号请写 QQ-Group:12345")
			return nil, true
		}
		return _newGroup, true
	}
	// 对应的组，是否存在第二个参数
	return nil, false
}

func filenameReplace(name string) string {
	re := regexp.MustCompile(`[/:\*\?"<>\|\\]`)
	return re.ReplaceAllString(name, "")
}

func LogSendToBackend(ctx *MsgContext, group *GroupInfo) string {
	dirpath := filepath.Join(ctx.Dice.BaseConfig.DataDir, "logs")
	os.MkdirAll(dirpath, 0755)

	lines, err := LogGetAllLines(ctx, group)
	badRawIds, err2 := LogGetAllDeleted(ctx, group)

	if err == nil {
		// 洗掉撤回的消息
		if err2 == nil {
			var linesNew []*LogOneItem
			for _, i := range lines {
				if !badRawIds[i.RawMsgId] {
					linesNew = append(linesNew, i)
				}
			}
			lines = linesNew
		}

		// 本地进行一个zip留档，以防万一
		gid := group.GroupId
		fzip, _ := ioutil.TempFile(dirpath, filenameReplace(gid+"_"+group.LogCurName)+".*.zip")
		writer := zip.NewWriter(fzip)

		text := ""
		for _, i := range lines {
			timeTxt := time.Unix(i.Time, 0).Format("2006-01-02 15:04:05")
			text += fmt.Sprintf("%s(%v) %s\n%s\n\n", i.Nickname, i.IMUserId, timeTxt, i.Message)
		}

		fileWriter, _ := writer.Create("文本log.txt")
		fileWriter.Write([]byte(text))

		data, err := json.Marshal(map[string]interface{}{
			"version": 100,
			"items":   lines,
		})
		if err == nil {
			fileWriter2, _ := writer.Create("海豹标准log-粘贴到染色器可格式化.txt")
			fileWriter2.Write(data)
		}

		_ = writer.Close()
		_ = fzip.Close()
	}

	if err == nil {
		// 压缩log，发往后端
		data, err := json.Marshal(map[string]interface{}{
			"version": 100,
			"items":   lines,
		})

		if err == nil {
			var zlibBuffer bytes.Buffer
			w := zlib.NewWriter(&zlibBuffer)
			w.Write(data)
			w.Close()

			return UploadFileToWeizaima(ctx.Dice.Logger, group.LogCurName, ctx.EndPoint.UserId, &zlibBuffer)
		}
	}
	return ""
}

// LogRollBriefByPC 根据log生成骰点简报
func LogRollBriefByPC(dice *Dice, items []*LogOneItem, showAll bool, name string) string {
	pcInfo := map[string]map[string]int{}
	// coc 同义词
	acCoc7 := setupConfig(dice)

	getName := func(s string) string {
		re := regexp.MustCompile(`^([^\d\s]+)(\d+)?$`)
		m := re.FindStringSubmatch(s)
		if len(m) > 0 {
			s = m[1]
		}

		return GetValueNameByAlias(s, acCoc7.Alias)
	}

	for _, i := range items {
		if i.CommandInfo != nil {
			info, _ := i.CommandInfo.(map[string]interface{})
			//t := time.Unix(i.Time, 0).Format("[04:05]")

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
							pcInfo[nickname][key] += 1
						} else if rank < 0 {
							key := fmt.Sprintf("%v:%v", attr, "失败")
							pcInfo[nickname][key] += 1
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
							pcInfo[nickname][key] += 1
						} else if rank < 0 {
							key := fmt.Sprintf("%v:%v", "理智", "失败")
							pcInfo[nickname][key] += 1
						}

						// 如果没有旧值，弄一个
						key := fmt.Sprintf("理智:旧值")
						if pcInfo[nickname][key] == 0 {
							pcInfo[nickname][key] = int(j["sanOld"].(float64))
						}

						key2 := fmt.Sprintf("理智:新值")
						//if pcInfo[nickname][key2] == 0 {
						pcInfo[nickname][key2] = int(j["sanNew"].(float64))
						//}
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
						nickname := fmt.Sprintf("%v", info["pcName"])
						setupName(nickname)

						if j["type"] == "mod" {
							attr := getName(j["attr"].(string))
							// 如果没有旧值，弄一个
							key := fmt.Sprintf("%v:旧值", attr)
							if pcInfo[nickname][key] == 0 {
								pcInfo[nickname][key] = int(j["valOld"].(float64))
							}

							key2 := fmt.Sprintf("%v:新值", attr)
							//if pcInfo[nickname][key2] == 0 {
							pcInfo[nickname][key2] = int(j["valNew"].(float64))
							//}
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
		others := []string{}

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
			ret := []string{}
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
func LogRollBriefDetail(items []*LogOneItem) []string {
	var texts []string
	for _, i := range items {
		if i.CommandInfo != nil {
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

func LogLinesGet(ctx *MsgContext, group *GroupInfo, name string) (int, bool) {
	var size int
	var exists bool
	ctx.Dice.DB.View(func(tx *bbolt.Tx) error {
		// Retrieve the users bucket.
		// This should be created when the DB is first opened.
		b0 := tx.Bucket([]byte("logs"))
		if b0 == nil {
			return nil
		}
		b1 := b0.Bucket([]byte(group.GroupId))
		if b1 == nil {
			return nil
		}
		b := b1.Bucket([]byte(name))
		if b == nil {
			return nil
		}
		exists = true
		size = b.Stats().KeyN
		return nil
	})
	return size, exists
}

func LogDelete(ctx *MsgContext, group *GroupInfo, name string) bool {
	var exists bool
	ctx.Dice.DB.Update(func(tx *bbolt.Tx) error {
		// Retrieve the users bucket.
		// This should be created when the DB is first opened.
		b0 := tx.Bucket([]byte("logs"))
		if b0 == nil {
			return nil
		}
		b1 := b0.Bucket([]byte(group.GroupId))
		if b1 == nil {
			return nil
		}

		err := b1.DeleteBucket([]byte(name))
		if err != nil {
			return err
		}
		exists = true

		_ = b1.DeleteBucket([]byte(name + "-delMark"))

		err = b1.Delete([]byte(name))
		if err != nil {
			return err
		}
		return nil
	})
	return exists
}

// LogGetList 获取列表
func LogGetList(ctx *MsgContext, group *GroupInfo) ([]string, error) {
	ret := []string{}
	return ret, ctx.Dice.DB.View(func(tx *bbolt.Tx) error {
		b0 := tx.Bucket([]byte("logs"))
		if b0 == nil {
			return nil
		}
		b1 := b0.Bucket([]byte(group.GroupId))
		if b1 == nil {
			return nil
		}

		return b1.ForEach(func(k, v []byte) error {
			if strings.HasSuffix(string(k), "-delMark") {
				// 跳过撤回记录
				return nil
			}
			ret = append(ret, string(k))
			return nil
		})
	})
}

func LogGetAllLines(ctx *MsgContext, group *GroupInfo) ([]*LogOneItem, error) {
	ret := []*LogOneItem{}
	return ret, ctx.Dice.DB.View(func(tx *bbolt.Tx) error {
		b0 := tx.Bucket([]byte("logs"))
		if b0 == nil {
			return nil
		}
		b1 := b0.Bucket([]byte(group.GroupId))
		if b1 == nil {
			return nil
		}

		b := b1.Bucket([]byte(group.LogCurName))
		if b == nil {
			return nil
		}

		return b.ForEach(func(k, v []byte) error {
			logItem := LogOneItem{}
			err := json.Unmarshal(v, &logItem)
			if err == nil {
				ret = append(ret, &logItem)
			}

			return nil
		})
	})
}

func LogGetAllLinesWithoutDeleted(ctx *MsgContext, group *GroupInfo, logName string) ([]*LogOneItem, error) {
	badRawIds, err2 := LogGetAllDeleted(ctx, group)

	ret := []*LogOneItem{}
	return ret, ctx.Dice.DB.View(func(tx *bbolt.Tx) error {
		b0 := tx.Bucket([]byte("logs"))
		if b0 == nil {
			return nil
		}
		b1 := b0.Bucket([]byte(group.GroupId))
		if b1 == nil {
			return nil
		}

		b := b1.Bucket([]byte(logName))
		if b == nil {
			return nil
		}

		return b.ForEach(func(k, v []byte) error {
			logItem := LogOneItem{}
			err := json.Unmarshal(v, &logItem)
			if err == nil {
				// 跳过撤回
				if err2 == nil {
					if badRawIds[logItem.RawMsgId] {
						return nil
					}
				}
				// 正常添加
				ret = append(ret, &logItem)
			}

			return nil
		})
	})
}

func LogAppend(ctx *MsgContext, group *GroupInfo, l *LogOneItem) error {
	return ctx.Dice.DB.Update(func(tx *bbolt.Tx) error {
		// Retrieve the users bucket.
		// This should be created when the DB is first opened.
		b0 := tx.Bucket([]byte("logs"))
		b1, err := b0.CreateBucketIfNotExists([]byte(group.GroupId))
		if err != nil {
			return err
		}

		b, err := b1.CreateBucketIfNotExists([]byte(group.LogCurName))
		_ = b.Put([]byte("modified"), []byte(strconv.FormatInt(time.Now().Unix(), 10)))
		if err == nil {
			l.Id, _ = b.NextSequence()
			buf, err := json.Marshal(l)
			if err != nil {
				return err
			}

			// 每记录1000条发出提示
			if ctx.Dice.LogSizeNoticeEnable {
				if ctx.Dice.LogSizeNoticeCount == 0 {
					ctx.Dice.LogSizeNoticeCount = 500
				}
				size := b.Stats().KeyN
				if size > 0 && size%ctx.Dice.LogSizeNoticeCount == 0 {
					text := fmt.Sprintf("提示: 当前故事的文本已经记录了 %d 条", size)
					ReplyToSenderRaw(ctx, &Message{MessageType: "group", GroupId: group.GroupId}, text, "skip")
				}
			}

			return b.Put(itob(l.Id), buf)
		}
		return err
	})
}

func LogMarkDeleteByMsgId(ctx *MsgContext, group *GroupInfo, rawId interface{}) error {
	if rawId == nil {
		return nil
	}
	return ctx.Dice.DB.Update(func(tx *bbolt.Tx) error {
		b0 := tx.Bucket([]byte("logs"))
		b1, err := b0.CreateBucketIfNotExists([]byte(group.GroupId))
		if err != nil {
			return err
		}

		b, err := b1.CreateBucketIfNotExists([]byte(group.LogCurName + "-delMark"))
		if err == nil {
			id, _ := b.NextSequence()
			buf, err := json.Marshal(rawId)
			if err != nil {
				return err
			}

			return b.Put(itob(id), buf)
		}
		return err
	})
}

func LogGetAllDeleted(ctx *MsgContext, group *GroupInfo) (map[interface{}]bool, error) {
	ret := map[interface{}]bool{}
	return ret, ctx.Dice.DB.View(func(tx *bbolt.Tx) error {
		b0 := tx.Bucket([]byte("logs"))
		if b0 == nil {
			return nil
		}
		b1 := b0.Bucket([]byte(group.GroupId))
		if b1 == nil {
			return nil
		}

		b := b1.Bucket([]byte(group.LogCurName + "-delMark"))
		if b == nil {
			return nil
		}

		return b.ForEach(func(k, v []byte) error {
			var val interface{}
			err := json.Unmarshal(v, &val)
			if err == nil {
				ret[val] = true
			}
			return nil
		})
	})
}

// itob returns an 8-byte big endian representation of v.
func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}
