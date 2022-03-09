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
	UserId int64 `json:"user_id"`
}

type CmdArgs struct {
	Command        string    `json:"command"`
	Args           []string  `json:"args"`
	Kwargs         []*Kwarg  `json:"kwargs"`
	At             []*AtInfo `json:"atInfo"`
	RawArgs        string    `json:"rawArgs"`
	AmIBeMentioned bool      `json:"amIBeMentioned"`
	CleanArgs      string
}

/** 检查第N项参数是否为某个字符串，n >= 1，若没有第n项参数也视为失败 */
func (a *CmdArgs) IsArgEqual(n int, ss ...string) bool {
	if len(a.Args) >= n {
		for _, i := range ss {
			if a.Args[n-1] == i {
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

func CommandParse(rawCmd string, commandCompatibleMode bool, currentCmdLst []string) *CmdArgs {
	restText, atInfo := AtParse(rawCmd)
	re := regexp.MustCompile(`^\s*[.。](\S+)\s*([^\n]*)`)

	if commandCompatibleMode {
		matched := ""
		for _, i := range currentCmdLst {
			if strings.HasPrefix(restText, "."+i) || strings.HasPrefix(restText, "。"+i) {
				matched = i
				break
			}
		}
		if matched != "" {
			runes := []rune(restText)
			restParams := runes[len([]rune(matched))+1:]
			restText = "." + matched + " " + string(restParams)
		}
	}

	m := re.FindStringSubmatch(restText)
	if len(m) == 3 {
		cmdInfo := new(CmdArgs)
		cmdInfo.Command = m[1]
		cmdInfo.RawArgs = m[2]
		cmdInfo.At = atInfo

		a := ArgsParse(m[2])
		cmdInfo.Args = a.Args
		cmdInfo.Kwargs = a.Kwargs
		//log.Println(222, m[1], "[sep]", m[2])

		// 将所有args连接起来，存入一个cleanArgs变量。主要用于兼容非标准参数
		//stText := ""
		//for _, text := range cmdInfo.Args {
		//	stText += text
		//}
		stText := strings.Join(cmdInfo.Args, " ")
		cmdInfo.CleanArgs = stText

		return cmdInfo
	}

	return nil
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

func AtParse(cmd string) (string, []*AtInfo) {
	//[CQ:at,qq=3604749540]
	ret := make([]*AtInfo, 0)
	re := regexp.MustCompile(`\[CQ:at,qq=(\d+?)]`)
	m := re.FindAllStringSubmatch(cmd, -1)

	for _, i := range m {
		if len(i) == 2 {
			at := new(AtInfo)
			at.UserId, _ = strconv.ParseInt(i[1], 10, 64)
			ret = append(ret, at)
		}
	}

	replaced := re.ReplaceAllString(cmd, "")

	return replaced, ret
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
