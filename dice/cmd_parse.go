package dice

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	ds "github.com/sealdice/dicescript"
	"github.com/tidwall/gjson"

	"sealdice-core/message"
)

type Kwarg struct {
	Name        string `jsbind:"name"        json:"name"`
	ValueExists bool   `jsbind:"valueExists" json:"valueExists"`
	Value       string `jsbind:"value"       json:"value"`
	AsBool      bool   `jsbind:"asBool"      json:"asBool"`
}

func (kwa *Kwarg) String() string {
	if kwa.Value == "" {
		return fmt.Sprintf("--%s", kwa.Name)
	} else {
		return fmt.Sprintf("--%s=%s", kwa.Name, kwa.Value)
	}
}

// [CQ:at,qq=22]
type AtInfo struct {
	UserID string `jsbind:"userId" json:"userId"`
	// UID    string `json:"uid"`
}

func (i *AtInfo) CopyCtx(ctx *MsgContext) (*MsgContext, bool) {
	c1 := *ctx
	mctx := &c1 // 复制一个ctx，用于其他用途
	mctx.vm = nil
	if ctx.Group != nil {
		p := ctx.Group.PlayerGet(ctx.Dice.DBOperator, i.UserID)
		if p != nil {
			mctx.Player = p
		} else {
			// TODO: 主动获取用户名
			mctx.Player = &GroupPlayerInfo{
				Name:          "",
				UserID:        i.UserID,
				ValueMapTemp:  &ds.ValueMap{},
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
	Command                    string    `jsbind:"command"                  json:"command"`
	Args                       []string  `jsbind:"args"                     json:"args"`
	Kwargs                     []*Kwarg  `jsbind:"kwargs"                   json:"kwargs"`
	At                         []*AtInfo `jsbind:"at"                       json:"atInfo"`
	RawArgs                    string    `jsbind:"rawArgs"                  json:"rawArgs"`
	AmIBeMentioned             bool      `jsbind:"amIBeMentioned"           json:"amIBeMentioned"`
	AmIBeMentionedFirst        bool      `jsbind:"amIBeMentionedFirst"      json:"amIBeMentionedFirst"` // 同上，但要求是第一个被@的
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

// RevokeExecuteTimesParse 因为次数解析进行的太早了，影响太大无法还原，这里干脆重新解析一遍
func (cmdArgs *CmdArgs) RevokeExecuteTimesParse(ctx *MsgContext, msg *Message) {
	// 对于使用消息段的信息，使用新的解析方式
	if len(msg.Segment) > 0 {
		cmdArgs.commandParseNew(ctx, msg, true)
	} else {
		cmdArgs.commandParse(cmdArgs.RawText, []string{cmdArgs.Command}, []string{cmdArgs.prefixStr}, cmdArgs.platformPrefix, true)
		cmdArgs.SetupAtInfo(cmdArgs.uidForAtInfo)
	}
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

// CommandCheckPrefixNew for new command parser ExecuteNew func, 干掉了 AtParse 和 Platform 参数
func CommandCheckPrefixNew(rawCmd string, prefix []string) bool {
	restText := strings.TrimSpace(rawCmd)
	restText, _ = SpecialExecuteTimesParse(restText)

	// 先导符号检测
	var prefixStr string
	for _, i := range prefix {
		if strings.HasPrefix(restText, i) {
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

// commandParseNew 新版命令解析器，支持消息段解析，替代旧的字符串解析方式
// 核心功能：从消息段中提取文本和@信息，检测命令前缀，匹配命令，解析参数
func (cmdArgs *CmdArgs) commandParseNew(ctx *MsgContext, msg *Message, isParseExecuteTimes bool) *CmdArgs {
	d := ctx.Session.Parent

	// === 第一步：从消息段提取文本内容 ===
	// 消息混合，但如果是指令，从第一个文本消息开始后面的一定是参数。
	textMsg := extractResultFromSegments(msg.Segment)
	rawCmd := strings.ReplaceAll(textMsg, "\r\n", "\n") // 统一换行符格式

	// === 第二步：解析@信息 ===
	// 分析消息段中的@元素，设置机器人被@状态
	parseAtInfo(cmdArgs, msg, ctx.EndPoint.UserID)

	// === 第三步：处理特殊执行次数和命令前缀 ===
	restText := strings.TrimSpace(rawCmd)
	specialExecuteTimes := 0
	if isParseExecuteTimes {
		restText, specialExecuteTimes = SpecialExecuteTimesParse(restText)
	}

	// 检测命令前缀（如 . ! 等）
	prefixStr := detectCommandPrefix(restText, ctx.Session.Parent.CommandPrefix)
	if prefixStr == "" {
		return nil // 没有有效前缀，不是命令
	}

	// 移除前缀并清理空格
	restText = strings.TrimSpace(restText[len(prefixStr):])

	// === 第四步：兼容性处理 ===
	// 处理历史遗留的特殊情况，如"bot list"转换为"botlist"
	if strings.HasPrefix(restText, "bot list") {
		restText = "botlist" + restText[len("bot list"):]
	}

	// === 第五步：命令匹配和参数解析 ===
	matched, isSpaceBeforeArgs := findMatchingCommand(restText, d, ctx.Group)
	if matched == "" {
		return nil // 没有匹配的命令
	}

	// 构建最终的命令参数对象
	return buildCmdArgs(cmdArgs, matched, restText, rawCmd, specialExecuteTimes, prefixStr, msg.Platform, isSpaceBeforeArgs)
}

// extractResultFromSegments 从消息段中提取纯文本内容 部分文本使用了CQ码。问题的原因在于，CmdArgs的参数可能是图片等其他数据，但CmdArgs缺乏对这个功能的支持。
func extractResultFromSegments(segments []message.IMessageElement) string {
	cqMessage := strings.Builder{}
	var foundFirstText bool
	for _, v := range segments {
		// 警告，这个函数不要复用到其他地方，如果复用，请删掉下面这个代码
		// 代码的意思是：从第一个有文本的元素开始，后面的全部认为是参数。
		// 检查是否是文本元素
		if v.Type() == message.Text {
			foundFirstText = true
		}
		// 只有找到第一个文本元素后才开始将剩下的拼凑到结果中
		if !foundFirstText {
			continue
		}
		switch v.Type() {
		case message.At:
			// 跳过 @ 信息，因为已通过 parseAtInfo 单独处理
			// 不要转换为 CQ 码，否则会污染 CleanArgs 导致表达式解析失败
			continue
		case message.Text:
			res, ok := v.(*message.TextElement)
			if !ok {
				continue
			}
			cqMessage.WriteString(res.Content)
		case message.Face:
			res, ok := v.(*message.FaceElement)
			if !ok {
				continue
			}
			_, _ = fmt.Fprintf(&cqMessage, "[CQ:face,id=%v]", res.FaceID)
		case message.File:
			res, ok := v.(*message.FileElement)
			if !ok {
				continue
			}
			fileVal := res.File
			if fileVal == "" {
				fileVal = res.URL
			}
			if fileVal == "" {
				continue
			}
			_, _ = fmt.Fprintf(&cqMessage, "[CQ:file,file=%v]", fileVal)
		case message.Image:
			res, ok := v.(*message.ImageElement)
			if !ok {
				continue
			}
			url := res.URL
			if res.URL == "" {
				url = res.File.URL
			}
			_, _ = fmt.Fprintf(&cqMessage, "[CQ:image,file=%v]", url)
		case message.Record:
			res, ok := v.(*message.RecordElement)
			if !ok {
				continue
			}
			var recordFile string
			if res.File != nil {
				recordFile = res.File.URL
				if recordFile == "" {
					recordFile = res.File.File
				}
			}
			if recordFile == "" {
				continue
			}
			_, _ = fmt.Fprintf(&cqMessage, "[CQ:record,file=%v]", recordFile)
		case message.Reply:
			res, ok := v.(*message.ReplyElement)
			if !ok {
				continue
			}
			parseInt, err := strconv.Atoi(res.ReplySeq)
			if err != nil {
				continue
			}
			_, _ = fmt.Fprintf(&cqMessage, "[CQ:reply,id=%v]", parseInt)
		case message.TTS:
			res, ok := v.(*message.TTSElement)
			if !ok {
				continue
			}
			_, _ = fmt.Fprintf(&cqMessage, "[CQ:tts,text=%v]", res.Content)
		case message.Poke:
			res, ok := v.(*message.PokeElement)
			if !ok {
				continue
			}
			_, _ = fmt.Fprintf(&cqMessage, "[CQ:poke,qq=%v]", res.Target)
		default:
			// 不是标准类型的情况
			res, ok := v.(*message.DefaultElement)
			if !ok {
				continue
			}
			// 将其转换为CQ码
			var (
				cqParamParts []string
			)
			dMap := gjson.ParseBytes(res.Data).Map()
			for paramStr, paramValue := range dMap {
				cqParamParts = append(cqParamParts, fmt.Sprintf("%s=%s", paramStr, paramValue))
			}
			cqParam := strings.Join(cqParamParts, ",")
			_, _ = fmt.Fprintf(&cqMessage, "[CQ:%s,%s]", res.RawType, cqParam)
		}
	}
	return cqMessage.String()
}

// parseAtInfo 解析@信息，设置相关的@状态标志
func parseAtInfo(cmdArgs *CmdArgs, msg *Message, botUserID string) {
	// 初始化@状态
	cmdArgs.AmIBeMentioned = false
	cmdArgs.AmIBeMentionedFirst = false
	cmdArgs.SomeoneBeMentionedButNotMe = false

	var atInfo []*AtInfo
	for _, elem := range msg.Segment {
		if e, ok := elem.(*message.AtElement); ok {
			// 检查是否@了机器人
			if msg.Platform+":"+e.Target == botUserID {
				cmdArgs.AmIBeMentioned = true
				cmdArgs.SomeoneBeMentionedButNotMe = false
				if len(atInfo) == 0 {
					cmdArgs.AmIBeMentionedFirst = true
				}
			} else if !cmdArgs.AmIBeMentioned {
				cmdArgs.SomeoneBeMentionedButNotMe = true
			}

			// 记录@信息
			atInfo = append(atInfo, &AtInfo{
				UserID: msg.Platform + ":" + e.Target,
			})
		}
	}
	cmdArgs.At = atInfo
}

// detectCommandPrefix 检测命令前缀，返回匹配的前缀字符串
func detectCommandPrefix(text string, prefixes []string) string {
	for _, prefix := range prefixes {
		if strings.HasPrefix(text, prefix) {
			return prefix
		}
	}
	return ""
}

// findMatchingCommand 查找匹配的命令，返回命令名和是否有前导空格
func findMatchingCommand(restText string, d *Dice, group *GroupInfo) (string, bool) {
	// 收集所有可用命令
	var cmdLst []string
	for k := range d.CmdMap {
		cmdLst = append(cmdLst, k)
	}

	// 添加群组激活的扩展命令
	if group != nil {
		for _, ext := range group.GetActivatedExtList(d) {
			for k := range ext.GetCmdMap() {
				cmdLst = append(cmdLst, k)
			}
		}
	}

	// 按长度排序，优先匹配长命令
	sort.Sort(ByLength(cmdLst))

	// 查找匹配的命令
	for _, cmd := range cmdLst {
		if len(cmd) > len(restText) {
			continue
		}

		if strings.EqualFold(restText[:len(cmd)], cmd) {
			// 检查命令后是否有空格（用于区分.rd20和.rd 20）
			runes := []rune(restText)
			restParams := runes[len([]rune(cmd)):]
			isSpaceBeforeArgs := len(restParams) > 0 && unicode.IsSpace(restParams[0])

			return cmd, isSpaceBeforeArgs
		}
	}

	return "", false
}

// buildCmdArgs 构建最终的命令参数对象
func buildCmdArgs(cmdArgs *CmdArgs, matched, restText, rawCmd string,
	specialExecuteTimes int, prefixStr, platform string, isSpaceBeforeArgs bool) *CmdArgs {
	// 提取参数部分
	runes := []rune(restText)
	restParams := runes[len([]rune(matched)):]
	restText = matched + " " + string(restParams)

	// 使用正则表达式解析命令和参数
	re := regexp.MustCompile(`^\s*(\S+)\s*([\S\s]*)`)
	m := re.FindStringSubmatch(restText)

	if len(m) != 3 {
		return nil
	}

	// 解析位置参数和关键字参数
	a := ArgsParse(m[2])

	// 填充命令参数对象
	cmdArgs.Command = m[1]
	cmdArgs.RawArgs = m[2]
	cmdArgs.Args = a.Args
	cmdArgs.Kwargs = a.Kwargs
	cmdArgs.RawText = rawCmd
	cmdArgs.CleanArgs = strings.TrimSpace(strings.Join(cmdArgs.Args, " "))
	cmdArgs.IsSpaceBeforeArgs = isSpaceBeforeArgs
	cmdArgs.SpecialExecuteTimes = specialExecuteTimes
	cmdArgs.prefixStr = prefixStr
	cmdArgs.platformPrefix = platform
	return cmdArgs
}

func CommandParse(rawCmd string, currentCmdLst []string, prefix []string, platformPrefix string, isParseExecuteTimes bool) *CmdArgs {
	cmdInfo := new(CmdArgs)
	return cmdInfo.commandParse(rawCmd, currentCmdLst, prefix, platformPrefix, isParseExecuteTimes)
}

func CommandParseNew(ctx *MsgContext, msg *Message) *CmdArgs {
	cmdInfo := new(CmdArgs)
	return cmdInfo.commandParseNew(ctx, msg, false)
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
	var argsPart strings.Builder
	for k, v := range c.Args {
		_, _ = fmt.Fprintf(&argsPart, ",%s=%s", k, v)
	}
	return fmt.Sprintf("[CQ:%s%s]", c.Type, argsPart.String())
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
		re = regexp.MustCompile(`\[CQ:at,qq=(\d+)(?:,name=(?:.*?))?\]`)
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
