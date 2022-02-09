package main

import (
	"github.com/jessevdk/go-flags"
	"log"
	"regexp"
	"strconv"
)

type Kwarg struct {
	Name        string `json:"name"`
	ValueExists bool   `json:"valueExists"`
	Value       string `json:"value"`
	AsBool      bool   `json:"asBool"`
}

// [CQ:at,qq=3604749540]
type AtInfo struct {
	UserId int64 `json:"user_id"`
}

type CmdArgs struct {
	Command string `json:"command"`
	Args   []string `json:"args"`
	Kwargs []*Kwarg `json:"kwargs"`
	At []*AtInfo `json:"atInfo"`
	RawArgs string `json:"rawArgs"`
	AmIBeMentioned bool `json:"amIBeMentioned"`
}

func CommandParse(rawCmd string) *CmdArgs {
	restText, atInfo := AtParse(rawCmd);
	re := regexp.MustCompile(`^\s*[.。](\S+)\s*([^\n]*)`)
	m := re.FindStringSubmatch(restText)
	if len(m) == 3 {
		cmdInfo := new(CmdArgs)
		cmdInfo.Command = m[1];
		cmdInfo.RawArgs = m[2];
		cmdInfo.At = atInfo;

		a := ArgsParse(m[2])
		cmdInfo.Args = a.Args;
		cmdInfo.Kwargs = a.Kwargs;
		//log.Println(222, m[1], "[sep]", m[2])

		return cmdInfo;
	}

	return nil;
}

func AtParse(cmd string) (string, []*AtInfo) {
	//[CQ:at,qq=3604749540]
	ret := make([]*AtInfo, 0);
	re := regexp.MustCompile(`\[CQ:at,qq=(\d+?)]`)
	m := re.FindAllStringSubmatch(cmd, -1);

	for _, i := range m {
		if len(i) == 2 {
			at := new(AtInfo)
			at.UserId, _ = strconv.ParseInt(i[1], 10, 64);
			ret = append(ret, at);
		}
	}

	replaced := re.ReplaceAllString(cmd, "")
	log.Println(replaced, ret);

	return replaced, ret;
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
	return cmdArgs;
}
