//nolint:testpackage // These tests exercise unexported parser internals.
package dice

import (
	"strings"
	"testing"

	"sealdice-core/message"
)

func newParserTestSession() (*Dice, *IMSession, *EndPointInfo) {
	d := &Dice{CommandPrefix: DefaultConfig.CommandPrefix}
	d.CmdMap = CmdMapCls{
		"r":       &CmdItemInfo{},
		"rd":      &CmdItemInfo{},
		"img":     &CmdItemInfo{},
		"botlist": &CmdItemInfo{},
	}
	session := &IMSession{Parent: d}
	ep := &EndPointInfo{EndPointInfoBase: EndPointInfoBase{UserID: "QQ:1000", Platform: "QQ"}}
	return d, session, ep
}

func TestCommandParseNewStringAndSegmentEquivalent(t *testing.T) {
	cases := []struct {
		name      string
		text      string
		wantCmd   string
		wantArg   string
		wantSpace bool
		wantTimes int
	}{
		{name: "普通命令", text: ".rd 1d6", wantCmd: "rd", wantArg: "1d6", wantSpace: true},
		{name: "无空格快捷", text: ".rd20", wantCmd: "rd", wantArg: "20", wantSpace: false},
		{name: "次数前缀", text: "3#.r 1d6", wantCmd: "r", wantArg: "1d6", wantSpace: true, wantTimes: 3},
		{name: "全角次数前缀", text: "3＃.r 1d6", wantCmd: "r", wantArg: "1d6", wantSpace: true, wantTimes: 3},
		{name: "bot list 兼容", text: ".bot list", wantCmd: "botlist"},
		{name: "未知命令", text: ".noSuch foo bar", wantCmd: "noSuch", wantArg: "foo bar"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, session, ep := newParserTestSession()
			ctx := &MsgContext{Session: session, EndPoint: ep, Dice: d}

			stringMsg := &Message{Platform: "QQ", Message: tc.text}
			NormalizeIncomingMessage(stringMsg)
			stringArgs := CommandParseNew(ctx, stringMsg)

			segmentMsg := &Message{
				Platform: "QQ",
				Segment:  []message.IMessageElement{&message.TextElement{Content: tc.text}},
			}
			NormalizeIncomingMessage(segmentMsg)
			segmentArgs := CommandParseNew(ctx, segmentMsg)

			for name, got := range map[string]*CmdArgs{"string": stringArgs, "segment": segmentArgs} {
				if got == nil {
					t.Fatalf("%s: expected command %q, got nil", name, tc.wantCmd)
				}
				if got.Command != tc.wantCmd {
					t.Errorf("%s: Command = %q, want %q", name, got.Command, tc.wantCmd)
				}
				if got.CleanArgs != tc.wantArg {
					t.Errorf("%s: CleanArgs = %q, want %q", name, got.CleanArgs, tc.wantArg)
				}
				if got.IsSpaceBeforeArgs != tc.wantSpace {
					t.Errorf("%s: IsSpaceBeforeArgs = %v, want %v", name, got.IsSpaceBeforeArgs, tc.wantSpace)
				}
				if got.SpecialExecuteTimes != tc.wantTimes {
					t.Errorf("%s: SpecialExecuteTimes = %d, want %d", name, got.SpecialExecuteTimes, tc.wantTimes)
				}
			}
		})
	}
}

func TestCommandParseNewOnlyPrefixReturnsNil(t *testing.T) {
	d, session, ep := newParserTestSession()
	ctx := &MsgContext{Session: session, EndPoint: ep, Dice: d}
	msg := &Message{Platform: "QQ", Message: "."}
	NormalizeIncomingMessage(msg)

	if got := CommandParseNew(ctx, msg); got != nil {
		t.Fatalf("expected nil for prefix-only message, got %#v", got)
	}
}

// QQ 官方 bot 的 UIN 区间在旧字符串解析（AtParse）里会标记 IsRobot，
// 字符串归一化成 AtElement 后该信息丢失，解析器必须补回，否则“@其他机器人不回应”失效。
func TestCommandParseNewStringAtKeepsQQBotDetection(t *testing.T) {
	d, session, ep := newParserTestSession()
	ctx := &MsgContext{Session: session, EndPoint: ep, Dice: d}
	msg := &Message{Platform: "QQ", Message: "[CQ:at,qq=2854196301] .r 1d6"}
	NormalizeIncomingMessage(msg)

	cmdArgs := CommandParseNew(ctx, msg)
	if cmdArgs == nil {
		t.Fatal("CommandParseNew returned nil")
	}
	if len(cmdArgs.At) != 1 {
		t.Fatalf("At count = %d, want 1", len(cmdArgs.At))
	}
	if !cmdArgs.At[0].IsRobot {
		t.Fatal("QQ bot UIN @ should be marked IsRobot after string normalization")
	}
	if cmdArgs.Command != "r" {
		t.Fatalf("Command = %q, want r", cmdArgs.Command)
	}
}

func TestCommandParseNewPreservesNonTextSegmentProjection(t *testing.T) {
	d, session, ep := newParserTestSession()
	ctx := &MsgContext{Session: session, EndPoint: ep, Dice: d}
	image := &message.ImageElement{URL: "https://example.invalid/a.png"}
	msg := &Message{
		Platform: "QQ",
		Segment: []message.IMessageElement{
			&message.TextElement{Content: ".img before "},
			image,
			&message.TextElement{Content: " after"},
		},
	}

	cmdArgs := CommandParseNew(ctx, msg)
	if cmdArgs == nil {
		t.Fatal("CommandParseNew returned nil")
	}
	if cmdArgs.Command != "img" {
		t.Fatalf("Command = %q, want img", cmdArgs.Command)
	}
	if strings.ContainsRune(cmdArgs.CleanArgs, '\x1f') {
		t.Fatalf("CleanArgs leaked placeholder: %q", cmdArgs.CleanArgs)
	}
	if !strings.Contains(cmdArgs.CleanArgs, "[CQ:image") {
		t.Fatalf("CleanArgs missing compatibility text: %q", cmdArgs.CleanArgs)
	}
	if strings.ContainsRune(cmdArgs.RawText, '\x1f') {
		t.Fatalf("RawText leaked placeholder: %q", cmdArgs.RawText)
	}
	if len(msg.Segment) != 3 || msg.Segment[1] != image {
		t.Fatalf("parser mutated message segments: %#v", msg.Segment)
	}
}

func TestCommandParseNewLiteralCQTextIsText(t *testing.T) {
	d, session, ep := newParserTestSession()
	ctx := &MsgContext{Session: session, EndPoint: ep, Dice: d}
	msg := &Message{
		Platform: "QQ",
		Segment: []message.IMessageElement{
			&message.TextElement{Content: ".img [CQ:at,qq=12345]"},
		},
	}

	cmdArgs := CommandParseNew(ctx, msg)
	if cmdArgs == nil {
		t.Fatal("CommandParseNew returned nil")
	}
	if len(cmdArgs.At) != 0 {
		t.Fatalf("At count = %d, want 0 for literal CQ text", len(cmdArgs.At))
	}
	if cmdArgs.CleanArgs != "[CQ:at,qq=12345]" {
		t.Fatalf("CleanArgs = %q", cmdArgs.CleanArgs)
	}
}

func TestCommandParseNewAtAllIsNotAMention(t *testing.T) {
	d, session, ep := newParserTestSession()
	ctx := &MsgContext{Session: session, EndPoint: ep, Dice: d}
	msg := &Message{Platform: "QQ", Message: ".img [CQ:at,qq=all] 1d100"}
	NormalizeIncomingMessage(msg)

	cmdArgs := CommandParseNew(ctx, msg)
	if cmdArgs == nil {
		t.Fatal("CommandParseNew returned nil")
	}
	if len(cmdArgs.At) != 0 {
		t.Fatalf("At count = %d, want 0 for qq=all", len(cmdArgs.At))
	}
	if cmdArgs.SomeoneBeMentionedButNotMe {
		t.Fatal("@全体不应置 SomeoneBeMentionedButNotMe")
	}
}

func TestCommandParseNewFallsBackToPlatformAtParse(t *testing.T) {
	d := &Dice{CommandPrefix: DefaultConfig.CommandPrefix}
	d.CmdMap = CmdMapCls{"r": &CmdItemInfo{}}
	session := &IMSession{Parent: d}
	ep := &EndPointInfo{EndPointInfoBase: EndPointInfoBase{UserID: "TG:1000", Platform: "TG"}}
	ctx := &MsgContext{Session: session, EndPoint: ep, Dice: d}

	msg := &Message{Platform: "TG", Message: "tg://user?id=1000 .r 1d100"}
	NormalizeIncomingMessage(msg)

	cmdArgs := CommandParseNew(ctx, msg)
	if cmdArgs == nil {
		t.Fatal("CommandParseNew returned nil")
	}
	if len(cmdArgs.At) != 1 {
		t.Fatalf("At count = %d, want 1 (from platform AtParse fallback)", len(cmdArgs.At))
	}
	if !cmdArgs.AmIBeMentioned {
		t.Fatal("AmIBeMentioned should be true")
	}
	if cmdArgs.Command != "r" {
		t.Fatalf("Command = %q, want r", cmdArgs.Command)
	}
}

func TestIsAtMeOpenQQCHCrossMatch(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		target   string
		atUID    string
		want     bool
	}{
		{name: "直接匹配", platform: "QQ", target: "1000", atUID: "QQ:1000", want: true},
		{name: "不匹配", platform: "QQ", target: "2000", atUID: "QQ:1000", want: false},
		{name: "OpenQQ→OpenQQCH 互换", platform: "OpenQQCH", target: "1000", atUID: "OpenQQ:1000", want: true},
		{name: "OpenQQCH→OpenQQ 互换", platform: "OpenQQ", target: "1000", atUID: "OpenQQCH:1000", want: true},
		{name: "OpenQQ 同前缀", platform: "OpenQQ", target: "1000", atUID: "OpenQQ:1000", want: true},
		{name: "OpenQQCH 同前缀", platform: "OpenQQCH", target: "1000", atUID: "OpenQQCH:1000", want: true},
		{name: "OpenQQ 不同号", platform: "OpenQQCH", target: "2000", atUID: "OpenQQ:1000", want: false},
		{name: "非 OpenQQ 平台不交叉", platform: "DISCORD", target: "1000", atUID: "QQ:1000", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isAtMe(tc.platform, tc.target, tc.atUID); got != tc.want {
				t.Errorf("isAtMe(%q, %q, %q) = %v, want %v", tc.platform, tc.target, tc.atUID, got, tc.want)
			}
		})
	}
}

func TestComputeAtUID(t *testing.T) {
	tests := []struct {
		name     string
		userID   string
		platform string
		tmpUID   string
		want     string
	}{
		{name: "普通", userID: "QQ:100", platform: "QQ", want: "QQ:100"},
		{name: "TmpUID 覆盖", userID: "QQ:100", platform: "QQ", tmpUID: "QQ:999", want: "QQ:999"},
		{name: "OpenQQCH 前缀互换", userID: "OpenQQ:100", platform: "OpenQQCH", want: "OpenQQCH:100"},
		{name: "OpenQQCH + TmpUID", userID: "OpenQQ:100", platform: "OpenQQCH", tmpUID: "OpenQQCH:999", want: "OpenQQCH:999"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ep := &EndPointInfo{EndPointInfoBase: EndPointInfoBase{UserID: tc.userID}}
			msg := &Message{Platform: tc.platform, TmpUID: tc.tmpUID}
			if got := computeAtUID(ep, msg); got != tc.want {
				t.Errorf("computeAtUID = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseAtInfoUsesTmpUIDAndIgnoresAll(t *testing.T) {
	cmdArgs := &CmdArgs{}
	msg := &Message{
		Platform: "QQ",
		Segment: []message.IMessageElement{
			&message.AtElement{Target: "all"},
			&message.AtElement{Target: "1000", IsRobot: true},
		},
	}
	parseAtInfo(cmdArgs, msg, "QQ:1000")

	if len(cmdArgs.At) != 1 {
		t.Fatalf("At count = %d, want 1", len(cmdArgs.At))
	}
	if !cmdArgs.AmIBeMentioned || !cmdArgs.AmIBeMentionedFirst {
		t.Fatalf("expected mentioned first, got mentioned=%v first=%v", cmdArgs.AmIBeMentioned, cmdArgs.AmIBeMentionedFirst)
	}
	if cmdArgs.SomeoneBeMentionedButNotMe {
		t.Fatal("SomeoneBeMentionedButNotMe should be false")
	}
}

func TestMessageMentionsBot(t *testing.T) {
	// 原生消息段：AtElement 命中
	segMsg := &Message{Platform: "QQ", Segment: []message.IMessageElement{
		&message.AtElement{Target: "1000"},
		&message.TextElement{Content: "hi"},
	}}
	if !messageMentionsBot(segMsg, "QQ:1000") {
		t.Fatal("segment At should match bot UID")
	}
	if messageMentionsBot(segMsg, "QQ:2000") {
		t.Fatal("segment At of someone else should not match")
	}

	// 非 CQ 平台：@ 记在兼容字符串里，回退 AtParse
	tgMsg := &Message{Platform: "TG", Message: "tg://user?id=1000 hello"}
	NormalizeIncomingMessage(tgMsg)
	if !messageMentionsBot(tgMsg, "TG:1000") {
		t.Fatal("synthesized TG message should fall back to AtParse")
	}
	if messageMentionsBot(tgMsg, "TG:2000") {
		t.Fatal("TG fallback should not match another UID")
	}

	// 原生段无 @ 且非合成：不回退
	nativeMsg := &Message{Platform: "TG", Message: "tg://user?id=1000 hello", Segment: []message.IMessageElement{
		&message.TextElement{Content: "tg://user?id=1000 hello"},
	}}
	if messageMentionsBot(nativeMsg, "TG:1000") {
		t.Fatal("native segment without AtElement must not fall back to AtParse")
	}
}

func TestRevokeExecuteTimesParseKeepsSegments(t *testing.T) {
	d, session, ep := newParserTestSession()
	ctx := &MsgContext{Session: session, EndPoint: ep, Dice: d}
	image := &message.ImageElement{URL: "https://example.invalid/a.png"}
	msg := &Message{
		Platform: "QQ",
		Segment: []message.IMessageElement{
			&message.TextElement{Content: "3#.r "},
			image,
		},
	}

	cmdArgs := CommandParseNew(ctx, msg)
	if cmdArgs == nil {
		t.Fatal("CommandParseNew returned nil")
	}
	if cmdArgs.SpecialExecuteTimes != 3 {
		t.Fatalf("SpecialExecuteTimes = %d, want 3", cmdArgs.SpecialExecuteTimes)
	}

	cmdArgs.RevokeExecuteTimesParse(ctx, msg)

	if len(msg.Segment) != 2 || msg.Segment[1] != image {
		t.Fatalf("message segments were overwritten: %#v", msg.Segment)
	}
	if cmdArgs.SpecialExecuteTimes != 3 {
		t.Fatalf("SpecialExecuteTimes after revoke = %d, want 3", cmdArgs.SpecialExecuteTimes)
	}
}
