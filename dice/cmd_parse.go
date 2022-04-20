package dice

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"regexp"
	"strconv"
	"strings"
)

type Kwarg struct {
	Name        string `json:"Name"`
	ValueExists bool   `json:"valueExists"`
	Value       string `json:"value"`
	AsBool      bool   `json:"asBool"`
}

// [CQ:at,qq=22]
type AtInfo struct {
	UserId string `json:"user_id"`
	//UID    string `json:"uid"`
}

func (i *AtInfo) CopyCtx(ctx *MsgContext) (*MsgContext, bool) {
	c1 := *ctx
	mctx := &c1 // 复制一个ctx，用于其他用途
	if ctx.Group != nil {
		p, exists := ctx.Group.Players[i.UserId]
		if exists {
			mctx.Player = p
		}
		return mctx, exists
	}
	return mctx, false
}

type CmdArgs struct {
	Command                    string    `json:"command"`
	Args                       []string  `json:"args"`
	Kwargs                     []*Kwarg  `json:"kwargs"`
	At                         []*AtInfo `json:"atInfo"`
	RawArgs                    string    `json:"rawArgs"`
	AmIBeMentioned             bool      `json:"amIBeMentioned"`
	SomeoneBeMentionedButNotMe bool      `json:"someoneBeMentionedButNotMe"`
	MentionedOtherDice         bool      `json:"mentionedOtherDice"`
	CleanArgs                  string
	SpecialExecuteTimes        int // 特殊的执行次数，对应 3# 这种
	CleanArgsChopRest          string
}

/** 检查第N项参数是否为某个字符串，n从1开始，若没有第n项参数也视为失败 */
func (a *CmdArgs) IsArgEqual(n int, ss ...string) bool {
	if len(a.Args) >= n {
		for _, i := range ss {
			if strings.EqualFold(a.Args[n-1], i) {
				return true
			}
		}
	}

	return false
}

func (a *CmdArgs) EatPrefixWith(ss ...string) (string, bool) {
	text := a.CleanArgs
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

func (a *CmdArgs) ChopPrefixToArgsWith(ss ...string) bool {
	if len(a.Args) > 0 {
		text := a.Args[0]
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

				a.Args = append(
					base,
					a.Args[1:]...,
				)
				a.CleanArgsChopRest = strings.TrimSpace(a.CleanArgs[len(i):])
				return true
			}
		}
	}

	return false
}

func (a *CmdArgs) GetArgN(n int) (string, bool) {
	if len(a.Args) >= n {
		return a.Args[n-1], true
	}

	return "", false
}

func (a *CmdArgs) GetKwarg(s string) *Kwarg {
	for _, i := range a.Kwargs {
		if i.Name == s {
			return i
		}
	}
	return nil
}

func CommandCheckPrefix(rawCmd string, prefix []string) bool {
	restText, _ := AtParse(rawCmd, "")
	restText = strings.TrimSpace(restText)
	restText, _ = SpecialExecuteTimesParse(restText)

	// 先导符号检测
	var prefixStr string
	for _, i := range prefix {
		if strings.HasPrefix(restText, i) {
			prefixStr = i
			break
		}
	}
	if prefixStr == "" {
		return false
	}
	return true
}

func CommandParse(rawCmd string, commandCompatibleMode bool, currentCmdLst []string, prefix []string, platformPrefix string) *CmdArgs {
	specialExecuteTimes := 0
	restText, atInfo := AtParse(rawCmd, platformPrefix)
	restText = strings.TrimSpace(restText)
	restText, specialExecuteTimes = SpecialExecuteTimesParse(restText)

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
	restText = restText[len(prefixStr):] // 排除先导符号

	// 兼容模式，进行格式化
	if commandCompatibleMode {
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
			restText = matched + " " + string(restParams)
		}
	}

	re := regexp.MustCompile(`^\s*(\S+)\s*([^\n]*)`)
	m := re.FindStringSubmatch(restText)
	if len(m) == 3 {
		cmdInfo := new(CmdArgs)
		cmdInfo.Command = m[1]
		cmdInfo.RawArgs = m[2]
		cmdInfo.At = atInfo

		a := ArgsParse2(m[2])
		cmdInfo.Args = a.Args
		cmdInfo.Kwargs = a.Kwargs
		//log.Println(222, m[1], "[sep]", m[2])

		// 将所有args连接起来，存入一个cleanArgs变量。主要用于兼容非标准参数
		stText := strings.Join(cmdInfo.Args, " ")
		cmdInfo.CleanArgs = strings.TrimSpace(stText)
		cmdInfo.SpecialExecuteTimes = specialExecuteTimes

		return cmdInfo
	}

	return nil
}

func SpecialExecuteTimesParse(cmd string) (string, int) {
	re := regexp.MustCompile(`\d+?[#＃]`)
	m := re.FindAllStringIndex(cmd, 1)
	var times int64
	if m != nil {
		for _, i := range m {
			text := cmd[i[0]:i[1]]
			times, _ = strconv.ParseInt(text[:len(text)-1], 10, 32)
			cmd = cmd[:i[0]] + cmd[i[1]:]
		}
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
	re := regexp.MustCompile(`\[(img|图):.+?]`) // [img:] 或 [图:]
	m := re.FindAllStringIndex(longText, -1)

	newText := longText
	for i := len(m) - 1; i >= 0; i-- {
		p := m[i]
		text := solve(longText[p[0]:p[1]])
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
				pair := strings.Split(i, "=")
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
	//[CQ:at,qq=3604749540]
	ret := make([]*AtInfo, 0)
	re := regexp.MustCompile(`\[CQ:at,qq=(\d+?)]`)
	m := re.FindAllStringSubmatch(cmd, -1)

	for _, i := range m {
		if len(i) == 2 {
			at := new(AtInfo)
			//at.UserId, _ = strconv.ParseInt(i[1], 10, 64)
			at.UserId = prefix + ":" + i[1]
			ret = append(ret, at)
		}
	}

	replaced := re.ReplaceAllString(cmd, "")
	return replaced, ret
}

var reSpace = regexp.MustCompile(`\s+`)
var reKeywordParam = regexp.MustCompile(`^--([^\s=]+)(?:=(\S+))?$`)

func ArgsParse2(rawCmd string) *CmdArgs {
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
			newText = ""
		} else {
			newArgs = append(newArgs, newText)
		}
	}

	cmd.Args = newArgs
	return &cmd
}

func ArgsParse(rawCmd string) *CmdArgs {
	re := regexp.MustCompile(`\s+`)
	args := re.Split(rawCmd, -1)

	// 处理一种特殊情况：当rawCmd为空，会Split出长度为1的数组
	if len(rawCmd) == 0 {
		args = []string{}
	}

	cmdArgs := new(CmdArgs)
	cmdArgs.Kwargs = make([]*Kwarg, 0)

	p := flags.NewParser(&struct{}{}, flags.Default)
	p.UnknownOptionHandler = func(option string, arg flags.SplitArgument, args []string) ([]string, error) {
		kwInfo := new(Kwarg)
		kwInfo.Name = option
		kwInfo.ValueExists = false
		if arg != nil {
			kwInfo.ValueExists = true
			a, b := arg.Value()
			kwInfo.AsBool = b
			kwInfo.Value = a
		}
		cmdArgs.Kwargs = append(cmdArgs.Kwargs, kwInfo)
		return args, nil
	}

	cmds, _ := p.ParseArgs(args)
	cmdArgs.Args = cmds

	//log.Println(cmdArgs)
	//a, _ := json.Marshal(cmdArgs)
	//log.Println(string((a)))
	return cmdArgs
}
