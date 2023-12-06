package dice

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/fy0/lockfree"
)

type Kwarg struct {
	Name        string `json:"name" jsbind:"name"`
	ValueExists bool   `json:"valueExists" jsbind:"valueExists"`
	Value       string `json:"value" jsbind:"value"`
	AsBool      bool   `json:"asBool" jsbind:"asBool"`
}

// [CQ:at,qq=22]
type AtInfo struct {
	UserID string `json:"userId" jsbind:"userId"`
	// UID    string `json:"uid"`
}

func (i *AtInfo) CopyCtx(ctx *MsgContext) (*MsgContext, bool) {
	c1 := *ctx
	mctx := &c1 // 复制一个ctx，用于其他用途
	if ctx.Group != nil {
		p := ctx.Group.PlayerGet(ctx.Dice.DBData, i.UserID)
		if p != nil {
			mctx.Player = p
		} else {
			// TODO: 主动获取用户名
			mctx.Player = &GroupPlayerInfo{
				Name:          "",
				UserID:        i.UserID,
				ValueMapTemp:  lockfree.NewHashMap(),
				UpdatedAtTime: 0,
			}
			// 特殊处理 official qq
			if strings.HasPrefix(i.UserID, "OpenQQCH:") {
				mctx.Player.Name = "<@!" + strings.TrimPrefix(i.UserID, "OpenQQCH:") + ">"
			} else if strings.HasPrefix(i.UserID, "OpenQQ-Member-T:") {
				mctx.Player.Name = i.UserID[len(i.UserID)-4:]
			}
		}
		return mctx, p != nil
	}
	return mctx, false
}

type CmdArgs struct {
	Command                    string    `json:"command" jsbind:"command"`
	Args                       []string  `json:"args" jsbind:"args"`
	Kwargs                     []*Kwarg  `json:"kwargs" jsbind:"kwargs"`
	At                         []*AtInfo `json:"atInfo" jsbind:"at"`
	RawArgs                    string    `json:"rawArgs" jsbind:"rawArgs"`
	AmIBeMentioned             bool      `json:"amIBeMentioned" jsbind:"amIBeMentioned"`
	AmIBeMentionedFirst        bool      `json:"amIBeMentionedFirst" jsbind:"amIBeMentionedFirst"` // 同上，但要求是第一个被@的
	SomeoneBeMentionedButNotMe bool      `json:"someoneBeMentionedButNotMe"`
	IsSpaceBeforeArgs          bool      `json:"isSpaceBeforeArgs"`     // 命令前面是否有空格，用于区分rd20和rd 20
	CleanArgs                  string    `jsbind:"cleanArgs"`           // 一种格式化后的参数，也就是中间所有分隔符都用一个空格替代
	SpecialExecuteTimes        int       `jsbind:"specialExecuteTimes"` // 特殊的执行次数，对应 3# 这种
	RawText                    string    `jsbind:"rawText"`             // 原始命令
	prefixStr                  string    // 命令前导符号，这几个用于基于当前cmdArgs信息重走解析流程，暂不对js开放
	platformPrefix             string    // 平台前缀
	uidForAtInfo               string    // 用于处理@的uid

	MentionedOtherDice bool   // 似乎没有在用
	CleanArgsChopRest  string // 未来可能移除
}

/** 检查第N项参数是否为某个字符串，n从1开始，若没有第n项参数也视为失败 */
func (cmdArgs *CmdArgs) IsArgEqual(n int, ss ...string) bool {
	if n <= 0 {
		return false
	}
	if len(cmdArgs.Args) >= n {
		for _, i := range ss {
			if strings.EqualFold(cmdArgs.Args[n-1], i) {
				return true
			}
		}
	}

	return false
}

func (cmdArgs *CmdArgs) EatPrefixWith(ss ...string) (string, bool) {
	text := cmdArgs.CleanArgs
	if len(text) > 0 {
		for _, i := range ss {
			if len(text) < len(i) {
				continue
			}
			if strings.EqualFold(text[:len(i)], i) {
				return strings.TrimSpace(text[len(i):]), true
			}
		}
	}

	return "", false
}

func (cmdArgs *CmdArgs) ChopPrefixToArgsWith(ss ...string) bool {
	if len(cmdArgs.Args) > 0 {
		text := cmdArgs.Args[0]
		for _, i := range ss {
			if len(text) < len(i) {
				continue
			}
			if strings.EqualFold(text[:len(i)], i) {
				base := []string{i} // 要不要 toLower ?
				t := strings.TrimSpace(text[len(i):])
				if t != "" {
					base = append(base, t)
				}

				cmdArgs.Args = append(
					base,
					cmdArgs.Args[1:]...,
				)
				cmdArgs.CleanArgsChopRest = strings.TrimSpace(cmdArgs.RawArgs[len(i):])
				return true
			}
		}
	}

	return false
}

func (cmdArgs *CmdArgs) GetArgN(n int) string {
	if len(cmdArgs.Args) >= n {
		return cmdArgs.Args[n-1]
	}

	return ""
}

func (cmdArgs *CmdArgs) GetKwarg(s string) *Kwarg {
	for _, i := range cmdArgs.Kwargs {
		if i.Name == s {
			return i
		}
	}
	return nil
}

func (cmdArgs *CmdArgs) GetRestArgsFrom(index int) string {
	txt := []string{}
	for i := index; i < len(cmdArgs.Args)+1; i++ {
		info := cmdArgs.GetArgN(i)
		if info != "" {
			txt = append(txt, info)
		} else {
			break
		}
	}
	return strings.Join(txt, " ")
}

func (cmdArgs *CmdArgs) RevokeExecuteTimesParse() {
	// 因为次数解析进行的太早了，影响太大无法还原，这里干脆重新解析一遍
	cmdArgs.commandParse(cmdArgs.RawText, []string{cmdArgs.Command}, []string{cmdArgs.prefixStr}, cmdArgs.platformPrefix, true)
	cmdArgs.SetupAtInfo(cmdArgs.uidForAtInfo)
}

func (cmdArgs *CmdArgs) SetupAtInfo(uid string) {
	// 设置AmIBeMentioned
	cmdArgs.AmIBeMentioned = false
	cmdArgs.AmIBeMentionedFirst = false
	cmdArgs.uidForAtInfo = uid

	for _, i := range cmdArgs.At {
		if i.UserID == uid {
			cmdArgs.AmIBeMentioned = true
			break
		}
	}
	if cmdArgs.AmIBeMentioned {
		// 检查是不是第一个被AT的
		if cmdArgs.At[0].UserID == uid {
			cmdArgs.AmIBeMentionedFirst = true
		}
	}

	// 有人被@了，但不是我
	// 后面的代码保证了如果@的名单中有任何已知骰子，不会进入下一步操作
	// 所以不用考虑其他骰子被@的情况
	cmdArgs.SomeoneBeMentionedButNotMe = len(cmdArgs.At) > 0 && (!cmdArgs.AmIBeMentioned)
}

func CommandCheckPrefix(rawCmd string, prefix []string, platform string) bool {
	restText, _ := AtParse(rawCmd, platform)
	restText = strings.TrimSpace(restText)
	restText, _ = SpecialExecuteTimesParse(restText)

	// 先导符号检测
	var prefixStr string
	for _, i := range prefix {
		if strings.HasPrefix(restText, i) {
			// 忽略两种非常容易误判的情况
			// if i == "。" && strings.HasPrefix(restText, "。。") {
			// 	continue
			// }
			// if i == "." && strings.HasPrefix(restText, "..") {
			// 	continue
			// }
			prefixStr = i
			break
		}
	}
	return prefixStr != ""
}

func (cmdArgs *CmdArgs) commandParse(rawCmd string, currentCmdLst []string, prefix []string, platformPrefix string, isParseExecuteTimes bool) *CmdArgs {
	specialExecuteTimes := 0
	rawCmd = strings.ReplaceAll(rawCmd, "\r\n", "\n") // 替换\r\n为\n
	restText, atInfo := AtParse(rawCmd, platformPrefix)
	restText = strings.TrimSpace(restText)
	if isParseExecuteTimes {
		restText, specialExecuteTimes = SpecialExecuteTimesParse(restText)
	}

	// 先导符号检测
	var prefixStr string
	for _, i := range prefix {
		if strings.HasPrefix(restText, i) {
			prefixStr = i
			break
		}
	}
	if prefixStr == "" {
		return nil
	}
	restText = restText[len(prefixStr):]   // 排除先导符号
	restText = strings.TrimSpace(restText) // 清除剩余文本的空格，以兼容. rd20 形式
	isSpaceBeforeArgs := false

	// 兼容模式，进行格式化
	// 之前的 commandCompatibleMode 现在不再有兼容模式的区分
	if strings.HasPrefix(restText, "bot list") {
		restText = "botlist" + restText[len("bot list"):]
	}

	matched := ""
	for _, i := range currentCmdLst {
		if len(i) > len(restText) {
			continue
		}

		if strings.EqualFold(restText[:len(i)], i) {
			matched = i
			break
		}
	}
	if matched != "" {
		runes := []rune(restText)
		restParams := runes[len([]rune(matched)):]
		// 检查是否有空格，例如.rd 20，以区别于.rd20
		if len(restParams) > 0 && unicode.IsSpace(restParams[0]) {
			isSpaceBeforeArgs = true
		}
		restText = matched + " " + string(restParams)
	}
	// 之前的兼容模式代码结束标记，已经不再使用

	re := regexp.MustCompile(`^\s*(\S+)\s*([\S\s]*)`)
	m := re.FindStringSubmatch(restText)

	if len(m) == 3 {
		cmdArgs.Command = m[1]
		cmdArgs.RawArgs = m[2]
		cmdArgs.At = atInfo
		cmdArgs.IsSpaceBeforeArgs = isSpaceBeforeArgs

		a := ArgsParse(m[2])
		cmdArgs.Args = a.Args
		cmdArgs.Kwargs = a.Kwargs

		// 将所有args连接起来，存入一个cleanArgs变量。主要用于兼容非标准参数
		stText := strings.Join(cmdArgs.Args, " ")
		cmdArgs.CleanArgs = strings.TrimSpace(stText)
		// NOTE(Xiangze Li): 不要在解析指令时直接修改轮数
		// if specialExecuteTimes > 25 {
		// 	specialExecuteTimes = 25
		// }
		cmdArgs.SpecialExecuteTimes = specialExecuteTimes

		// 以下信息用于重组解析使用
		cmdArgs.RawText = rawCmd
		cmdArgs.prefixStr = prefixStr
		cmdArgs.platformPrefix = platformPrefix

		return cmdArgs
	}

	return nil
}

func CommandParse(rawCmd string, currentCmdLst []string, prefix []string, platformPrefix string, isParseExecuteTimes bool) *CmdArgs {
	cmdInfo := new(CmdArgs)
	return cmdInfo.commandParse(rawCmd, currentCmdLst, prefix, platformPrefix, isParseExecuteTimes)
}

func SpecialExecuteTimesParse(cmd string) (string, int) {
	re := regexp.MustCompile(`\d+?[#＃]`)
	m := re.FindAllStringIndex(cmd, 1)
	var times int64

	for _, i := range m {
		text := cmd[i[0]:i[1]]
		times, _ = strconv.ParseInt(text[:len(text)-1], 10, 32)
		cmd = cmd[:i[0]] + cmd[i[1]:]
	}

	return cmd, int(times)
}

type CQCommand struct {
	Type      string
	Args      map[string]string
	Overwrite string
}

func (c *CQCommand) Compile() string {
	if c.Overwrite != "" {
		return c.Overwrite
	}
	argsPart := ""
	for k, v := range c.Args {
		argsPart += fmt.Sprintf(",%s=%s", k, v)
	}
	return fmt.Sprintf("[CQ:%s%s]", c.Type, argsPart)
}

func ImageRewrite(longText string, solve func(text string) string) string {
	re := regexp.MustCompile(`\[(img|图|文本|text|语音|voice|视频|video):(.+?)]`) // [img:] 或 [图:]
	m := re.FindAllStringIndex(longText, -1)

	newText := longText
	for i := len(m) - 1; i >= 0; i-- {
		p := m[i]
		text := solve(longText[p[0]:p[1]])
		newText = newText[:p[0]] + text + newText[p[1]:]
	}

	return newText
}

func DeckRewrite(longText string, solve func(text string) string) string {
	re := regexp.MustCompile(`###DRAW-(\S+?)###`)
	m := re.FindAllStringIndex(longText, -1)

	newText := longText
	for i := len(m) - 1; i >= 0; i-- {
		p := m[i]
		text := solve(longText[p[0]+len("###DRAW-") : p[1]-len("###")])
		newText = newText[:p[0]] + text + newText[p[1]:]
	}

	return newText
}

func CQRewrite(longText string, solve func(cq *CQCommand)) string {
	re := regexp.MustCompile(`\[CQ:.+?]`)
	m := re.FindAllStringIndex(longText, -1)

	newText := longText
	for i := len(m) - 1; i >= 0; i-- {
		p := m[i]
		cq := CQParse(longText[p[0]:p[1]])
		solve(cq)
		newText = newText[:p[0]] + cq.Compile() + newText[p[1]:]
	}

	return newText
}

func CQParse(cmd string) *CQCommand {
	// [CQ:image,file=data/images/1.png,type=show,id=40004]
	var main string
	args := make(map[string]string)

	re := regexp.MustCompile(`\[CQ:([^],]+)(,[^]]+)?]`)
	m := re.FindStringSubmatch(cmd)
	if m != nil {
		main = m[1]
		if m[2] != "" {
			argList := strings.Split(m[2], ",")
			for _, i := range argList {
				pair := strings.SplitN(i, "=", 2)
				if len(pair) >= 2 {
					args[pair[0]] = pair[1]
				}
			}
		}
	}
	return &CQCommand{
		Type: main,
		Args: args,
	}
}

func AtParse(cmd string, prefix string) (string, []*AtInfo) {
	// gocq的@:		[CQ:at,qq=3604749540]
	// discordGo的@:	<@1048209604938563736>
	ret := make([]*AtInfo, 0)
	re := regexp.MustCompile("")
	switch prefix {
	case "QQ":
		re = regexp.MustCompile(`\[CQ:at,qq=(\d+?)]`)
	case "OpenQQ", "OpenQQCH":
		re = regexp.MustCompile(`<@!?(\S+?)>`)
	case "DISCORD":
		re = regexp.MustCompile(`<@(\d+?)>`)
	case "KOOK":
		re = regexp.MustCompile(`\(met\)(\d+?)\(met\)`)
	case "TG":
		re = regexp.MustCompile(`tg:\/\/user\?id=(\d+)`)
	case "DODO":
		re = regexp.MustCompile(`<@\!(\d+?)>`)
	case "SLACK":
		re = regexp.MustCompile(`<@(.+?)>`)
	case "SEALCHAT":
		re = regexp.MustCompile(`<@(\S+?)>`)
	}

	m := re.FindAllStringSubmatch(cmd, -1)

	for _, i := range m {
		if len(i) == 2 {
			at := new(AtInfo)
			at.UserID = prefix + ":" + i[1]
			ret = append(ret, at)
		}
	}

	replaced := re.ReplaceAllString(cmd, "")
	return replaced, ret
}

func AtBuild(uid string) string {
	if uid == "" {
		return ""
	}
	re := regexp.MustCompile("(QQ|DISCORD|KOOK|TG|DODO).*?:(.*)")
	m := re.FindStringSubmatch(uid)
	var text string
	if len(m) == 3 {
		text = fmt.Sprintf("[CQ:at,qq=%s]", m[2])
	} else {
		text = fmt.Sprintf("[At:%s]", uid)
	}
	return text
}

var reSpace = regexp.MustCompile(`\s+`)
var reKeywordParam = regexp.MustCompile(`^--([^\s=]+)(?:=(\S+))?$`)

func ArgsParse(rawCmd string) *CmdArgs {
	args := reSpace.Split(rawCmd, -1)
	newArgs := []string{}

	cmd := CmdArgs{}
	cmd.Kwargs = make([]*Kwarg, 0)

	for _, oneText := range args {
		newText := oneText
		if oneText == "" {
			continue
		}
		m := reKeywordParam.FindStringSubmatch(oneText)
		if len(m) > 0 {
			kw := Kwarg{}
			kw.Name = m[1]
			kw.Value = m[2]
			kw.ValueExists = m[2] != ""
			kw.AsBool = m[2] != "false"
			cmd.Kwargs = append(cmd.Kwargs, &kw)
		} else {
			newArgs = append(newArgs, newText)
		}
	}

	cmd.Args = newArgs
	return &cmd
}
