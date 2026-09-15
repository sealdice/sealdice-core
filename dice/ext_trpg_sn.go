package dice

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	ds "github.com/sealdice/dicescript"
)

var ErrGroupCardOverlong = errors.New("群名片长度超过限制")

func SetPlayerGroupCardByTemplate(ctx *MsgContext, tmpl string) (string, error) {
	if ctx.SystemTemplate == nil {
		ctx.SystemTemplate = ctx.Group.GetCharTemplate(ctx.Dice)
	}
	config := ctx.GenDefaultRollVmConfig()
	config.HookValueStore = func(ctx *ds.Context, name string, v *ds.VMValue) (overwrite *ds.VMValue, solved bool) {
		return nil, true
	}
	v := ctx.EvalFString(tmpl, config)
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

func getCmdTrpgSn(d *Dice) *CmdItemInfo {
	helpSn := `.sn coc // 自动设置coc名片
.sn dnd // 自动设置dnd名片
.sn none // 设置为空白格式
.sn off // 取消自动设置
`
	return &CmdItemInfo{
		Name:               "sn",
		ShortHelp:          helpSn,
		Help:               "跑团名片(需要管理权限):\n" + helpSn,
		CheckCurrentBotOn:  true,
		CheckMentionOthers: true,
		HelpFunc: func(isShort bool) string {
			// 手动添加特定的命令示例到帮助信息的开头
			fixedExamples := ".sn coc // 自动设置coc名片\n" +
				".sn cocL // 自动设置coc名片，小写\n" +
				".sn dnd // 自动设置dnd名片\n"

			text := fixedExamples
			var tempStrList []string
			d.GameSystemMap.Range(func(key string, value *GameSystemTemplate) bool {
				for k, v := range value.NameTemplate {
					if k != "coc" && k != "dnd" && k != "cocL" {
						// 考虑到这里的量级不会太大，所以直接排序已经生成好的提示文本或许更划算
						tempStrList = append(tempStrList, fmt.Sprintf(".sn %s // %s\n", k, v.HelpText))
					}
				}
				return true
			})

			sort.Strings(tempStrList)
			text += strings.Join(tempStrList, "")
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
			valLower := strings.ToLower(val)

			handleOverlong := func(ctx *MsgContext, msg *Message, card string) CmdExecuteResult {
				ReplyToSender(ctx, msg, fmt.Sprintf(
					"尝试将群名片修改为 %q 失败，名片长度超过限制。\n请尝试缩短角色名或使用 .sn expr 自定义名片格式。",
					card,
				))
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			// sn 指令(除 help 外)需要骰子具有群管理员权限。提前检测 bot 在群中的角色,
			// 若明确无管理权限则给出明确提示, 由于协议端问题，目前只能做提示。不支持角色检查的适配器
			// 返回 ok=false, 此时保持原有行为(直接尝试设置)不做阻断。
			if valLower != "help" && ctx.Group != nil {
				if detail, ok := checkBotGroupRole(ctx, ctx.Group.GroupID); ok && detail != "owner" && detail != "admin" {
					ReplyToSender(ctx, msg, "【警告】骰子当前可能不具备管理员权限，请检查，若确认具备管理员权限可无视此误报。")
				}
			}

			switch valLower {
			case "help":
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			case "coc", "coc7":
				ctx.Player.AutoSetNameTemplate = "{$t玩家_RAW} SAN{理智} HP{生命值}/{生命值上限} DEX{敏捷}"
				ctx.Player.UpdatedAtTime = time.Now().Unix()
				if ctx.Group != nil {
					ctx.Group.MarkDirty(ctx.Dice)
				}
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
				if ctx.Group != nil {
					ctx.Group.MarkDirty(ctx.Dice)
				}
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
				if ctx.Group != nil {
					ctx.Group.MarkDirty(ctx.Dice)
				}
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
				if ctx.Group != nil {
					ctx.Group.MarkDirty(ctx.Dice)
				}
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
					if ctx.Group != nil {
						ctx.Group.MarkDirty(ctx.Dice)
					}
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
						if ctx.Group != nil {
							ctx.Group.MarkDirty(ctx.Dice)
						}
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

					// 增加使用sn设置自定义规则的名片模板时的错误反馈
					text, err := SetPlayerGroupCardByTemplate(ctx, t.Template)
					if errors.Is(err, ErrGroupCardOverlong) {
						handleOverlong(ctx, msg, text)
						ok = true
						return false
					} else if err != nil {
						ReplyToSender(ctx, msg, "命名模版错误或不存在，请使用.sn help查看使用说明")
						ok = true
						return false
					}
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
}
