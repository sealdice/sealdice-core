package dice

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	ds "github.com/sealdice/dicescript"

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
	UserID  string `jsbind:"userId"  json:"userId"`
	IsRobot bool   `jsbind:"isRobot" json:"isRobot"`
	Name    string `jsbind:"name"    json:"name"`
	// UID    string `json:"uid"`
}

func (i *AtInfo) CopyCtx(ctx *MsgContext) (*MsgContext, bool) {
	mctx := ctx.ShallowCopy() // 复制一个ctx，用于其他用途
	mctx.vm = nil

	// 复制临时的消息ID和事件ID，以支持官方QQ的被动消息发送
	if msgID, ok := VarGetValueStr(ctx, "$tMsgID"); ok {
		VarSetValueStr(mctx, "$tMsgID", msgID)
	}
	if eventID, ok := VarGetValueStr(ctx, "$tEventID"); ok {
		VarSetValueStr(mctx, "$tEventID", eventID)
	}

	if ctx.Group != nil {
		playerUserID := i.UserID
		officialQQUIN := ""
		if ctx.EndPoint != nil {
			if pa, ok := ctx.EndPoint.Adapter.(*PlatformAdapterOfficialQQ); ok {
				officialQQUIN = pa.UIN
				// 官方 QQ 的 @ 仅提供 MemberOpenID；人物卡则以
				// OpenQQ:<UIN>-<MemberOpenID> 为键，代骰时必须先补全身份。
				playerUserID = normalizeDiceIDOfficialQQMentionUserID(officialQQUIN, playerUserID)
			}
		}

		p := ctx.Group.PlayerGet(ctx.Dice.DBOperator, playerUserID)
		if p != nil {
			// 平台事件中的 @ 昵称只用于当前代骰消息。PlayerGet 可能返回
			// 群缓存中的共享指针，因此必须复制后再补名称，不能修改原对象。
			if p.Name == "" && i.Name != "" {
				playerCopy := *p
				playerCopy.Name = i.Name
				mctx.Player = &playerCopy
			} else {
				mctx.Player = p
			}
		} else {
			// TODO: 主动获取用户名
			mctx.Player = &GroupPlayerInfo{
				Name:          i.Name,
				UserID:        playerUserID,
				ValueMapTemp:  &ds.ValueMap{},
				UpdatedAtTime: 0,
			}
			// 特殊处理 official qq
			// 只有平台没有提供昵称时，才退回到可发送的 @ 文本。
			if mctx.Player.Name == "" && strings.HasPrefix(i.UserID, "OpenQQCH:") {
				mctx.Player.Name = "<@!" + strings.TrimPrefix(i.UserID, "OpenQQCH:") + ">"
			} else if mctx.Player.Name == "" && strings.HasPrefix(i.UserID, "OpenQQ:") {
				mentionTarget := normalizeOfficialQQGroupAtTarget(officialQQUIN, i.UserID)
				mctx.Player.Name = formatOfficialQQAtUser(mentionTarget)
			}
		}
		return mctx, p != nil
	}
	return mctx, false
}

type ParsedCommand struct {
	Command             string
	Args                []string
	Kwargs              []*Kwarg
	At                  []*AtInfo
	RawArgs             string
	CleanArgs           string
	RawText             string
	Projection          message.SegmentText
	Prefix              string
	PlatformPrefix      string
	SpecialExecuteTimes int
	IsSpaceBeforeArgs   bool
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
	parsed                     *ParsedCommand

	MentionedOtherDice bool   // 似乎没有在用
	CleanArgsChopRest  string // 未来可能移除
}

func (cmdArgs *CmdArgs) applyParsed(parsed *ParsedCommand) *CmdArgs {
	if parsed == nil {
		return nil
	}
	cmdArgs.parsed = parsed
	cmdArgs.Command = parsed.Command
	cmdArgs.Args = append(cmdArgs.Args[:0], parsed.Args...)
	cmdArgs.Kwargs = append(cmdArgs.Kwargs[:0], parsed.Kwargs...)
	cmdArgs.At = parsed.At
	cmdArgs.RawArgs = parsed.RawArgs
	cmdArgs.CleanArgs = parsed.CleanArgs
	cmdArgs.RawText = parsed.RawText
	cmdArgs.IsSpaceBeforeArgs = parsed.IsSpaceBeforeArgs
	cmdArgs.SpecialExecuteTimes = parsed.SpecialExecuteTimes
	cmdArgs.prefixStr = parsed.Prefix
	cmdArgs.platformPrefix = parsed.PlatformPrefix
	return cmdArgs
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

// RevokeExecuteTimesParse 因为次数解析进行的太早了，影响太大无法还原，这里干脆重新解析一遍。
// 以当前 canonical Segment 重新投影解析，避免用 RawText 覆盖消息段。
func (cmdArgs *CmdArgs) RevokeExecuteTimesParse(ctx *MsgContext, msg *Message) {
	NormalizeIncomingMessage(msg)
	cmdArgs.commandParseNew(ctx, msg, true)
}

func (cmdArgs *CmdArgs) applyMentionedInfo(msg *Message) {
	if cmdArgs == nil || msg == nil || len(msg.MentionedInfo) == 0 {
		return
	}
	for _, at := range cmdArgs.At {
		if name := msg.MentionedInfo[at.UserID]; name != "" {
			at.Name = name
		}
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

func projectCommandSegmentsToText(segments []message.IMessageElement) message.SegmentText {
	firstTextIndex := -1
	for idx, elem := range segments {
		if _, ok := elem.(*message.TextElement); ok {
			firstTextIndex = idx
			break
		}
	}
	if firstTextIndex < 0 {
		return message.SegmentText{}
	}

	commandSegments := make([]message.IMessageElement, 0, len(segments)-firstTextIndex)
	for _, elem := range segments[firstTextIndex:] {
		if _, ok := elem.(*message.AtElement); ok {
			continue
		}
		commandSegments = append(commandSegments, elem)
	}
	return message.ProjectSegmentsToText(commandSegments)
}

// commandParseNew 新版命令解析器，支持消息段解析，替代旧的字符串解析方式
// 核心功能：从消息段投影文本和非文本占位符，检测命令前缀，匹配命令，解析参数。
func (cmdArgs *CmdArgs) commandParseNew(ctx *MsgContext, msg *Message, isParseExecuteTimes bool) *CmdArgs {
	d := ctx.Session.Parent

	projection := projectCommandSegmentsToText(msg.Segment)
	rawCmd := strings.ReplaceAll(projection.Text, "\r\n", "\n") // 统一换行符格式

	parseAtInfo(cmdArgs, msg, computeAtUID(ctx.EndPoint, msg))

	restText := strings.TrimSpace(rawCmd)
	specialExecuteTimes := 0
	strippedRestText, parsedExecuteTimes := SpecialExecuteTimesParse(restText)
	if isParseExecuteTimes {
		restText = strings.TrimSpace(strippedRestText)
		specialExecuteTimes = parsedExecuteTimes
	} else if detectCommandPrefix(restText, ctx.Session.Parent.CommandPrefix) == "" {
		candidateText := strings.TrimSpace(strippedRestText)
		if detectCommandPrefix(candidateText, ctx.Session.Parent.CommandPrefix) != "" {
			restText = candidateText
		}
	}

	prefixStr := detectCommandPrefix(restText, ctx.Session.Parent.CommandPrefix)
	if prefixStr == "" {
		return nil // 没有有效前缀，不是命令
	}

	restText = strings.TrimSpace(restText[len(prefixStr):])

	// 处理历史遗留的特殊情况，如"bot list"转换为"botlist"
	if strings.HasPrefix(restText, "bot list") {
		restText = "botlist" + restText[len("bot list"):]
	}

	matched, isSpaceBeforeArgs := findMatchingCommand(restText, d, ctx.Group)
	if matched == "" {
		// 保留旧路语义：未知命令不返回 nil，提取首个 token 作为命令名。
		// 这样 commandSolve 仍会触发 OnCommandReceived 等钩子（插件契约）。
		// 与旧 commandParse 的正则兜底 `^\s*(\S+)\s*([\S\s]*)` 行为一致。
		if fields := strings.Fields(restText); len(fields) > 0 {
			matched = fields[0]
		}
		isSpaceBeforeArgs = false
	}
	if matched == "" {
		return nil // restText 为空（如消息只有一个前缀 "."）
	}

	// 构建最终的命令参数对象
	return buildCmdArgs(cmdArgs, matched, restText, rawCmd, projection, specialExecuteTimes, prefixStr, msg.Platform, isSpaceBeforeArgs)
}

// isAtMe 判断 @ 的目标是否是当前机器人，兼容 OpenQQ/OpenQQCH 前缀互换（QQ 频道场景）。
// platform 和 target 来自 AtElement，atUID 是经过 computeAtUID 处理后的 bot UID。
func isAtMe(platform, target, atUID string) bool {
	userID := platform + ":" + target
	if userID == atUID {
		return true
	}
	// OpenQQ 与 OpenQQCH 前缀互换：同一个 QQ 号在频道里可能是 OpenQQCH: 前缀
	if strings.HasPrefix(userID, "OpenQQ:") || strings.HasPrefix(userID, "OpenQQCH:") {
		uid := strings.TrimPrefix(strings.TrimPrefix(atUID, "OpenQQ:"), "OpenQQCH:")
		if uid != atUID { // atUID 确实带有 OpenQQ/OpenQQCH 前缀
			return userID == "OpenQQ:"+uid || userID == "OpenQQCH:"+uid
		}
	}
	return false
}

// computeAtUID 计算 @ 判定用的 bot UID，兼容 TmpUID 和 OpenQQ 频道前缀互换。
func computeAtUID(ep *EndPointInfo, msg *Message) string {
	atUID := ep.UserID
	if msg.Platform == "OpenQQCH" {
		atUID = "OpenQQCH:" + strings.TrimPrefix(ep.UserID, "OpenQQ:")
	}
	if msg.TmpUID != "" {
		atUID = msg.TmpUID
	}
	return atUID
}

// parseAtInfo 解析@信息，设置相关的@状态标志。atUID 由 computeAtUID 计算。
func parseAtInfo(cmdArgs *CmdArgs, msg *Message, atUID string) {
	// 初始化@状态
	cmdArgs.AmIBeMentioned = false
	cmdArgs.AmIBeMentionedFirst = false
	cmdArgs.SomeoneBeMentionedButNotMe = false
	cmdArgs.uidForAtInfo = atUID

	var atInfo []*AtInfo
	for _, elem := range msg.Segment {
		if e, ok := elem.(*message.AtElement); ok {
			userID := msg.Platform + ":" + e.Target

			// 检查是否@了机器人
			if isAtMe(msg.Platform, e.Target, atUID) {
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
				UserID:  userID,
				IsRobot: e.IsRobot,
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
func buildCmdArgs(cmdArgs *CmdArgs, matched, restText, rawCmd string, projection message.SegmentText,
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

	parsed := &ParsedCommand{
		Command:             m[1],
		RawArgs:             m[2],
		Args:                a.Args,
		Kwargs:              a.Kwargs,
		At:                  cmdArgs.At,
		RawText:             rawCmd,
		CleanArgs:           strings.TrimSpace(strings.Join(a.Args, " ")),
		Projection:          projection,
		IsSpaceBeforeArgs:   isSpaceBeforeArgs,
		SpecialExecuteTimes: specialExecuteTimes,
		Prefix:              prefixStr,
		PlatformPrefix:      platform,
	}
	return cmdArgs.applyParsed(parsed)
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
	case "OpenQQ":
		re = officialQQAtRegex
	case "OpenQQCH":
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
		if len(i) >= 2 {
			target := i[1]
			if prefix == "OpenQQ" {
				target = officialQQMentionTarget(i)
			}
			if target != "" {
				at := new(AtInfo)
				at.UserID = prefix + ":" + target
				at.IsRobot = prefix == "QQ" && isQQBotUserID(at.UserID)
				ret = append(ret, at)
			}
		}
	}

	replaced := re.ReplaceAllString(cmd, "")
	return replaced, ret
}

func AtBuild(uid string) string {
	if uid == "" {
		return ""
	}
	re := regexp.MustCompile("^(OpenQQ|QQ|DISCORD|KOOK|TG|DODO).*?:(.*)")
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
