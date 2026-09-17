//nolint:testpackage // These tests exercise unexported parser internals and compatibility wrappers.
package dice

import (
	"testing"

	"sealdice-core/message"
)

// TestCommandParseUnified_StringVsSegment 验证同一消息的"字符串输入"和"Segment 输入"
// 产出等价的 CmdArgs（统一解析的核心契约）。
func TestCommandParseUnified_StringVsSegment(t *testing.T) {
	d := &Dice{CommandPrefix: DefaultConfig.CommandPrefix}
	d.CmdMap = CmdMapCls{
		"rd": &CmdItemInfo{},
		"r":  &CmdItemInfo{},
	}
	session := &IMSession{Parent: d}
	ep := &EndPointInfo{EndPointInfoBase: EndPointInfoBase{UserID: "QQ:1000", Platform: "QQ"}}

	cases := []struct {
		name    string
		text    string
		cmdLst  []string
		wantCmd string
		wantArg string
	}{
		{"普通命令", ".rd 1d6", []string{"rd"}, "rd", "1d6"},
		{"无空格快捷", ".rd20", []string{"rd"}, "rd", "20"},
		{"bot list 兼容", ".bot list", []string{"botlist"}, "botlist", ""},
		{"未知命令", ".xyz arg1", []string{"rd"}, "xyz", "arg1"},
		{"只有前缀不触发", ".", []string{"rd"}, "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name+"_string", func(t *testing.T) {
			msg := &Message{Platform: "QQ", Message: tc.text}
			ctx := &MsgContext{Session: session, EndPoint: ep, Dice: d}
			ensureSegment(msg)
			ensureMessage(msg)
			cmdArgs := CommandParseNew(ctx, msg)
			if tc.wantCmd == "" {
				if cmdArgs != nil {
					t.Fatalf("expected nil, got Command=%q", cmdArgs.Command)
				}
				return
			}
			if cmdArgs == nil {
				t.Fatalf("expected Command=%q, got nil", tc.wantCmd)
			}
			if cmdArgs.Command != tc.wantCmd {
				t.Errorf("Command = %q, want %q", cmdArgs.Command, tc.wantCmd)
			}
			if cmdArgs.CleanArgs != tc.wantArg {
				t.Errorf("CleanArgs = %q, want %q", cmdArgs.CleanArgs, tc.wantArg)
			}
		})

		t.Run(tc.name+"_segment", func(t *testing.T) {
			msg := &Message{
				Platform: "QQ",
				Segment:  []message.IMessageElement{&message.TextElement{Content: tc.text}},
			}
			ctx := &MsgContext{Session: session, EndPoint: ep, Dice: d}
			ensureSegment(msg)
			ensureMessage(msg)
			cmdArgs := CommandParseNew(ctx, msg)
			if tc.wantCmd == "" {
				if cmdArgs != nil {
					t.Fatalf("expected nil, got Command=%q", cmdArgs.Command)
				}
				return
			}
			if cmdArgs == nil {
				t.Fatalf("expected Command=%q, got nil", tc.wantCmd)
			}
			if cmdArgs.Command != tc.wantCmd {
				t.Errorf("Command = %q, want %q", cmdArgs.Command, tc.wantCmd)
			}
			if cmdArgs.CleanArgs != tc.wantArg {
				t.Errorf("CleanArgs = %q, want %q", cmdArgs.CleanArgs, tc.wantArg)
			}
		})
	}
}

// TestUnknownCommand_NotNil 验证未知命令不返回 nil（旧路契约：OnCommandReceived 仍触发）。
func TestUnknownCommand_NotNil(t *testing.T) {
	d := &Dice{CommandPrefix: DefaultConfig.CommandPrefix}
	d.CmdMap = CmdMapCls{"rd": &CmdItemInfo{}}
	session := &IMSession{Parent: d}
	ep := &EndPointInfo{EndPointInfoBase: EndPointInfoBase{UserID: "QQ:1000", Platform: "QQ"}}

	msg := &Message{
		Platform: "QQ",
		Segment:  []message.IMessageElement{&message.TextElement{Content: ".noSuchCmd foo bar"}},
	}
	ctx := &MsgContext{Session: session, EndPoint: ep, Dice: d}
	cmdArgs := CommandParseNew(ctx, msg)

	if cmdArgs == nil {
		t.Fatal("unknown command returned nil; expected non-nil to preserve OnCommandReceived contract")
	}
	if cmdArgs.Command != "noSuchCmd" {
		t.Errorf("Command = %q, want %q", cmdArgs.Command, "noSuchCmd")
	}
	if cmdArgs.CleanArgs != "foo bar" {
		t.Errorf("CleanArgs = %q, want %q", cmdArgs.CleanArgs, "foo bar")
	}
}

// TestIsAtMe_OpenQQCHCrossMatch 验证 OpenQQ / OpenQQCH 前缀互换的 @ 交叉匹配。
func TestIsAtMe_OpenQQCHCrossMatch(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		target   string
		atUID    string
		want     bool
	}{
		{"直接匹配", "QQ", "1000", "QQ:1000", true},
		{"不匹配", "QQ", "2000", "QQ:1000", false},
		{"OpenQQ→OpenQQCH 互换", "OpenQQCH", "1000", "OpenQQ:1000", true},
		{"OpenQQCH→OpenQQ 互换", "OpenQQ", "1000", "OpenQQCH:1000", true},
		{"OpenQQ 同前缀", "OpenQQ", "1000", "OpenQQ:1000", true},
		{"OpenQQCH 同前缀", "OpenQQCH", "1000", "OpenQQCH:1000", true},
		{"OpenQQ 不同号", "OpenQQCH", "2000", "OpenQQ:1000", false},
		{"非 OpenQQ 平台不交叉", "DISCORD", "1000", "QQ:1000", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isAtMe(tc.platform, tc.target, tc.atUID)
			if got != tc.want {
				t.Errorf("isAtMe(%q, %q, %q) = %v, want %v", tc.platform, tc.target, tc.atUID, got, tc.want)
			}
		})
	}
}

// TestComputeAtUID 验证 TmpUID / OpenQQCH 的 UID 计算。
func TestComputeAtUID(t *testing.T) {
	tests := []struct {
		name     string
		userID   string
		platform string
		tmpUID   string
		want     string
	}{
		{"普通", "QQ:100", "QQ", "", "QQ:100"},
		{"TmpUID 覆盖", "QQ:100", "QQ", "QQ:999", "QQ:999"},
		{"OpenQQCH 前缀互换", "OpenQQ:100", "OpenQQCH", "", "OpenQQCH:100"},
		{"OpenQQCH + TmpUID", "OpenQQ:100", "OpenQQCH", "OpenQQCH:999", "OpenQQCH:999"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ep := &EndPointInfo{EndPointInfoBase: EndPointInfoBase{UserID: tc.userID}}
			msg := &Message{Platform: tc.platform, TmpUID: tc.tmpUID}
			got := computeAtUID(ep, msg)
			if got != tc.want {
				t.Errorf("computeAtUID = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestParseAtInfo_Segment 验证从 Segment 解析@信息（包括 AmIBeMentionedFirst）。
func TestParseAtInfo_Segment(t *testing.T) {
	tests := []struct {
		name             string
		platform         string
		segments         []message.IMessageElement
		atUID            string
		wantMentioned    bool
		wantFirst        bool
		wantSomeoneNotMe bool
		wantAtCount      int
	}{
		{
			name:     "@自己（第一个）",
			platform: "QQ",
			segments: []message.IMessageElement{
				&message.AtElement{Target: "1000"},
			},
			atUID:            "QQ:1000",
			wantMentioned:    true,
			wantFirst:        true,
			wantSomeoneNotMe: false,
			wantAtCount:      1,
		},
		{
			name:     "@别人再@自己",
			platform: "QQ",
			segments: []message.IMessageElement{
				&message.AtElement{Target: "2000"},
				&message.AtElement{Target: "1000"},
			},
			atUID:            "QQ:1000",
			wantMentioned:    true,
			wantFirst:        false,
			wantSomeoneNotMe: false,
			wantAtCount:      2,
		},
		{
			name:     "@别人",
			platform: "QQ",
			segments: []message.IMessageElement{
				&message.AtElement{Target: "2000"},
			},
			atUID:            "QQ:1000",
			wantMentioned:    false,
			wantFirst:        false,
			wantSomeoneNotMe: true,
			wantAtCount:      1,
		},
		{
			name:     "无@",
			platform: "QQ",
			segments: []message.IMessageElement{
				&message.TextElement{Content: "hello"},
			},
			atUID:            "QQ:1000",
			wantMentioned:    false,
			wantFirst:        false,
			wantSomeoneNotMe: false,
			wantAtCount:      0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := &Message{Platform: tc.platform, Segment: tc.segments}
			cmdArgs := &CmdArgs{}
			parseAtInfo(cmdArgs, msg, tc.atUID)
			if cmdArgs.AmIBeMentioned != tc.wantMentioned {
				t.Errorf("AmIBeMentioned = %v, want %v", cmdArgs.AmIBeMentioned, tc.wantMentioned)
			}
			if cmdArgs.AmIBeMentionedFirst != tc.wantFirst {
				t.Errorf("AmIBeMentionedFirst = %v, want %v", cmdArgs.AmIBeMentionedFirst, tc.wantFirst)
			}
			if cmdArgs.SomeoneBeMentionedButNotMe != tc.wantSomeoneNotMe {
				t.Errorf("SomeoneBeMentionedButNotMe = %v, want %v", cmdArgs.SomeoneBeMentionedButNotMe, tc.wantSomeoneNotMe)
			}
			if len(cmdArgs.At) != tc.wantAtCount {
				t.Errorf("At count = %d, want %d", len(cmdArgs.At), tc.wantAtCount)
			}
		})
	}
}

func TestCommandParseNewPreservesNonTextSegmentProjection(t *testing.T) {
	d := &Dice{CommandPrefix: DefaultConfig.CommandPrefix}
	d.CmdMap = CmdMapCls{"img": &CmdItemInfo{}}
	session := &IMSession{Parent: d}
	ep := &EndPointInfo{EndPointInfoBase: EndPointInfoBase{UserID: "QQ:1000", Platform: "QQ"}}
	image := &message.ImageElement{URL: "https://example.invalid/a.png"}
	msg := &Message{
		Platform: "QQ",
		Segment: []message.IMessageElement{
			&message.TextElement{Content: ".img before "},
			image,
			&message.TextElement{Content: " after"},
		},
	}
	ctx := &MsgContext{Session: session, EndPoint: ep, Dice: d}

	cmdArgs := CommandParseNew(ctx, msg)
	if cmdArgs == nil {
		t.Fatal("CommandParseNew returned nil")
	}
	if cmdArgs.Command != "img" {
		t.Fatalf("Command = %q, want img", cmdArgs.Command)
	}
	if cmdArgs.parsed == nil {
		t.Fatal("internal parsed command should be set")
	}
	if len(cmdArgs.parsed.Projection.Placeholders) != 1 {
		t.Fatalf("placeholder count = %d, want 1", len(cmdArgs.parsed.Projection.Placeholders))
	}
}

func TestCommandParseNewKeepsLiteralCQTextAsTextWhenSegmentInputIsText(t *testing.T) {
	d := &Dice{CommandPrefix: DefaultConfig.CommandPrefix}
	d.CmdMap = CmdMapCls{"echo": &CmdItemInfo{}}
	session := &IMSession{Parent: d}
	ep := &EndPointInfo{EndPointInfoBase: EndPointInfoBase{UserID: "QQ:1000", Platform: "QQ"}}
	msg := &Message{
		Platform: "QQ",
		Segment: []message.IMessageElement{
			&message.TextElement{Content: ".echo [CQ:at,qq=12345]"},
		},
	}
	ctx := &MsgContext{Session: session, EndPoint: ep, Dice: d}

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

func TestRevokeExecuteTimesParseDoesNotOverwriteSegmentsFromRawText(t *testing.T) {
	d := &Dice{CommandPrefix: DefaultConfig.CommandPrefix}
	d.CmdMap = CmdMapCls{"r": &CmdItemInfo{}}
	session := &IMSession{Parent: d}
	ep := &EndPointInfo{EndPointInfoBase: EndPointInfoBase{UserID: "QQ:1000", Platform: "QQ"}}
	image := &message.ImageElement{URL: "https://example.invalid/a.png"}
	msg := &Message{
		Platform: "QQ",
		Segment: []message.IMessageElement{
			&message.TextElement{Content: "3#.r "},
			image,
		},
	}
	ctx := &MsgContext{Session: session, EndPoint: ep, Dice: d}

	cmdArgs := new(CmdArgs).commandParseNew(ctx, msg, false)
	if cmdArgs == nil {
		t.Fatal("commandParseNew returned nil")
	}
	cmdArgs.RevokeExecuteTimesParse(ctx, msg)

	if len(msg.Segment) != 2 || msg.Segment[1] != image {
		t.Fatalf("message segments were overwritten: %#v", msg.Segment)
	}
	if cmdArgs.SpecialExecuteTimes != 3 {
		t.Fatalf("SpecialExecuteTimes = %d, want 3", cmdArgs.SpecialExecuteTimes)
	}
}

func TestCmdArgsLegacyMethodsDelegateToParsedModel(t *testing.T) {
	cmdArgs := &CmdArgs{}
	cmdArgs.applyParsed(&ParsedCommand{
		Command:   "test",
		Args:      []string{"on", "target"},
		Kwargs:    []*Kwarg{{Name: "flag", Value: "true", ValueExists: true, AsBool: true}},
		RawArgs:   "on target --flag=true",
		CleanArgs: "on target",
		RawText:   ".test on target --flag=true",
	})

	if !cmdArgs.IsArgEqual(1, "on") {
		t.Fatal("IsArgEqual should read wrapper args")
	}
	if cmdArgs.GetArgN(2) != "target" {
		t.Fatalf("GetArgN(2) = %q", cmdArgs.GetArgN(2))
	}
	if cmdArgs.GetKwarg("flag") == nil {
		t.Fatal("GetKwarg should find flag")
	}
	if cmdArgs.GetRestArgsFrom(1) != "on target" {
		t.Fatalf("GetRestArgsFrom(1) = %q", cmdArgs.GetRestArgsFrom(1))
	}
}

func TestCmdArgsChopPrefixToArgsWithMaintainsLegacyFields(t *testing.T) {
	cmdArgs := &CmdArgs{}
	cmdArgs.applyParsed(&ParsedCommand{
		Command:   "bot",
		Args:      []string{"on123", "extra"},
		RawArgs:   "on123 extra",
		CleanArgs: "on123 extra",
		RawText:   ".bot on123 extra",
	})

	if !cmdArgs.ChopPrefixToArgsWith("on", "off") {
		t.Fatal("ChopPrefixToArgsWith should match prefix")
	}
	if cmdArgs.GetArgN(1) != "on" || cmdArgs.GetArgN(2) != "123" {
		t.Fatalf("unexpected args after chop: %#v", cmdArgs.Args)
	}
	if cmdArgs.CleanArgsChopRest != "123 extra" {
		t.Fatalf("CleanArgsChopRest = %q", cmdArgs.CleanArgsChopRest)
	}
}
