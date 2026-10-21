//nolint:testpackage
package dice

import (
	"errors"
	"strings"
	"testing"

	ds "github.com/sealdice/dicescript"
)

type nameCardSpyAdapter struct {
	*mockPlatformAdapter
	setCardCalls int
	lastCard     string
}

func (s *nameCardSpyAdapter) SetGroupCardName(_ *MsgContext, name string) {
	s.setCardCalls++
	s.lastCard = name
}

func newNameCardTestContext(t *testing.T) (*MsgContext, *nameCardSpyAdapter) {
	t.Helper()

	d, ep, baseAdapter, _ := newExecuteNewTestDice(t)
	spy := &nameCardSpyAdapter{mockPlatformAdapter: baseAdapter}
	ep.Platform = "QQ"
	ep.ProtocolType = "official"
	ep.Adapter = spy

	group := &GroupInfo{
		GroupID: "QQ-Group:10001",
		Players: new(SyncMap[string, *GroupPlayerInfo]),
	}
	player := &GroupPlayerInfo{
		UserID:       "QQ:20002",
		Name:         "茶叶",
		ValueMapTemp: &ds.ValueMap{},
	}
	group.Players.Store(player.UserID, player)

	ctx := &MsgContext{
		Dice:        d,
		EndPoint:    ep,
		Session:     d.ImSession,
		Group:       group,
		Player:      player,
		MessageType: "group",
	}
	ctx.SystemTemplate = group.GetCharTemplate(d)

	d.Config.OfficialQQUseMarkdown = true
	d.Config.OfficialQQUseNameCard = true

	return ctx, spy
}

func TestEvalPlayerGroupCardTemplateDoesNotSetCard(t *testing.T) {
	ctx, spy := newNameCardTestContext(t)

	text, err := EvalPlayerGroupCardTemplate(ctx, "茶叶 HP10/10")
	if err != nil {
		t.Fatalf("EvalPlayerGroupCardTemplate returned error: %v", err)
	}
	if text != "茶叶 HP10/10" {
		t.Fatalf("text = %q, want %q", text, "茶叶 HP10/10")
	}
	if spy.setCardCalls != 0 {
		t.Fatalf("SetGroupCardName called %d times, want 0", spy.setCardCalls)
	}
}

func TestSetPlayerGroupCardByTemplateSkipsOfficialQQ(t *testing.T) {
	ctx, spy := newNameCardTestContext(t)
	longCard := strings.Repeat("a", 70)

	text, err := SetPlayerGroupCardByTemplate(ctx, longCard)
	if err != nil {
		t.Fatalf("SetPlayerGroupCardByTemplate returned error: %v", err)
	}
	if text != longCard {
		t.Fatalf("text = %q, want %q", text, longCard)
	}
	if spy.setCardCalls != 0 {
		t.Fatalf("SetGroupCardName called %d times, want 0", spy.setCardCalls)
	}
}

func TestSetPlayerGroupCardByTemplateKeepsRealCardForOtherPlatforms(t *testing.T) {
	ctx, spy := newNameCardTestContext(t)
	ctx.EndPoint.ProtocolType = "onebot"

	text, err := SetPlayerGroupCardByTemplate(ctx, "茶叶 HP10/10")
	if err != nil {
		t.Fatalf("SetPlayerGroupCardByTemplate returned error: %v", err)
	}
	if text != "茶叶 HP10/10" {
		t.Fatalf("text = %q, want %q", text, "茶叶 HP10/10")
	}
	if spy.setCardCalls != 1 || spy.lastCard != text {
		t.Fatalf("SetGroupCardName calls = %d, last = %q; want 1, %q",
			spy.setCardCalls, spy.lastCard, text)
	}
}

func TestSetPlayerGroupCardByTemplateKeepsOverlongCheckForOtherPlatforms(t *testing.T) {
	ctx, spy := newNameCardTestContext(t)
	ctx.EndPoint.ProtocolType = "onebot"
	longCard := strings.Repeat("a", 70)

	text, err := SetPlayerGroupCardByTemplate(ctx, longCard)
	if !errors.Is(err, ErrGroupCardOverlong) {
		t.Fatalf("err = %v, want ErrGroupCardOverlong", err)
	}
	if text != longCard {
		t.Fatalf("text = %q, want %q", text, longCard)
	}
	if spy.setCardCalls != 0 {
		t.Fatalf("SetGroupCardName called %d times, want 0", spy.setCardCalls)
	}
}

func TestBuildOfficialQQNameCardLine(t *testing.T) {
	ctx, _ := newNameCardTestContext(t)
	ctx.Player.AutoSetNameTemplate = "茶叶 HP10/10"

	line := buildOfficialQQNameCardLine(ctx)
	const want = "<font color=\"#FF0000\" size=\"1\">茶叶 HP10/10</font>\n"
	if line != want {
		t.Fatalf("line = %q, want %q", line, want)
	}
}

func TestBuildOfficialQQNameCardLineUsesConfiguredTemplate(t *testing.T) {
	ctx, _ := newNameCardTestContext(t)
	ctx.Player.AutoSetNameTemplate = "茶叶 HP10/10"
	ctx.Dice.Config.OfficialQQNameCardTmpl = "<b>{card}</b>"

	line := buildOfficialQQNameCardLine(ctx)
	const want = "<b>茶叶 HP10/10</b>\n"
	if line != want {
		t.Fatalf("line = %q, want %q", line, want)
	}
}

func TestBuildOfficialQQNameCardLineFallsBackWhenTemplateIsEmpty(t *testing.T) {
	ctx, _ := newNameCardTestContext(t)
	ctx.Player.AutoSetNameTemplate = "茶叶 HP10/10"
	ctx.Dice.Config.OfficialQQNameCardTmpl = ""

	line := buildOfficialQQNameCardLine(ctx)
	const want = "<font color=\"#FF0000\" size=\"1\">茶叶 HP10/10</font>\n"
	if line != want {
		t.Fatalf("line = %q, want %q", line, want)
	}
}

func TestBuildOfficialQQNameCardLineFallsBackWhenPlaceholderMissing(t *testing.T) {
	ctx, _ := newNameCardTestContext(t)
	ctx.Player.AutoSetNameTemplate = "茶叶 HP10/10"
	ctx.Dice.Config.OfficialQQNameCardTmpl = "<b>没有占位符</b>"

	line := buildOfficialQQNameCardLine(ctx)
	const want = "<font color=\"#FF0000\" size=\"1\">茶叶 HP10/10</font>\n"
	if line != want {
		t.Fatalf("line = %q, want %q", line, want)
	}
}

func TestBuildOfficialQQNameCardLineDoesNotDuplicateTrailingNewline(t *testing.T) {
	ctx, _ := newNameCardTestContext(t)
	ctx.Player.AutoSetNameTemplate = "茶叶 HP10/10"
	ctx.Dice.Config.OfficialQQNameCardTmpl = "<b>{card}</b>\n"

	line := buildOfficialQQNameCardLine(ctx)
	const want = "<b>茶叶 HP10/10</b>\n"
	if line != want {
		t.Fatalf("line = %q, want %q", line, want)
	}
}

func TestBuildOfficialQQNameCardLineRequiresBothSwitches(t *testing.T) {
	for name, mutate := range map[string]func(*MsgContext){
		"name card off": func(ctx *MsgContext) { ctx.Dice.Config.OfficialQQUseNameCard = false },
		"markdown off":  func(ctx *MsgContext) { ctx.Dice.Config.OfficialQQUseMarkdown = false },
	} {
		t.Run(name, func(t *testing.T) {
			ctx, _ := newNameCardTestContext(t)
			ctx.Player.AutoSetNameTemplate = "茶叶 HP10/10"
			mutate(ctx)
			if line := buildOfficialQQNameCardLine(ctx); line != "" {
				t.Fatalf("line = %q, want empty", line)
			}
		})
	}
}

func TestBuildOfficialQQNameCardLineOnlyForOfficialQQ(t *testing.T) {
	ctx, _ := newNameCardTestContext(t)
	ctx.Player.AutoSetNameTemplate = "茶叶 HP10/10"
	ctx.EndPoint.ProtocolType = "onebot"

	if line := buildOfficialQQNameCardLine(ctx); line != "" {
		t.Fatalf("line = %q, want empty", line)
	}
}

func TestBuildOfficialQQNameCardLineWithoutTemplateOrPlayer(t *testing.T) {
	ctx, _ := newNameCardTestContext(t)
	if line := buildOfficialQQNameCardLine(ctx); line != "" {
		t.Fatalf("line without template = %q, want empty", line)
	}

	ctx.Player.AutoSetNameTemplate = "茶叶 HP10/10"
	ctx.Player = nil
	if line := buildOfficialQQNameCardLine(ctx); line != "" {
		t.Fatalf("line without player = %q, want empty", line)
	}
}

func TestBuildOfficialQQNameCardLineEscapesMarkup(t *testing.T) {
	ctx, _ := newNameCardTestContext(t)
	ctx.Player.AutoSetNameTemplate = "A<B>&C"

	line := buildOfficialQQNameCardLine(ctx)
	const want = "<font color=\"#FF0000\" size=\"1\">A&lt;B&gt;&amp;C</font>\n"
	if line != want {
		t.Fatalf("line = %q, want %q", line, want)
	}
}

func TestWithOfficialQQNameCardPrependsOnce(t *testing.T) {
	ctx, _ := newNameCardTestContext(t)
	ctx.Player.AutoSetNameTemplate = "茶叶 HP10/10"

	text := withOfficialQQNameCard(ctx, "D100=50")
	const want = "<font color=\"#FF0000\" size=\"1\">茶叶 HP10/10</font>\nD100=50"
	if text != want {
		t.Fatalf("text = %q, want %q", text, want)
	}
}

func TestReplyGroupAndPersonIncludeNameCard(t *testing.T) {
	ctx, spy := newNameCardTestContext(t)
	ctx.Player.AutoSetNameTemplate = "茶叶 HP10/10"
	msg := &Message{
		MessageType: "group",
		GroupID:     ctx.Group.GroupID,
		Sender: SenderBase{
			UserID:   ctx.Player.UserID,
			Nickname: ctx.Player.Name,
		},
	}

	ReplyGroup(ctx, msg, "D100=50")
	ReplyPerson(ctx, msg, "D100=60")

	wantGroup := "<font color=\"#FF0000\" size=\"1\">茶叶 HP10/10</font>\nD100=50"
	if len(spy.groupMsgs) != 1 || spy.groupMsgs[0] != wantGroup {
		t.Fatalf("group messages = %#v, want [%q]", spy.groupMsgs, wantGroup)
	}
	wantPerson := "<font color=\"#FF0000\" size=\"1\">茶叶 HP10/10</font>\nD100=60"
	if len(spy.personMsgs) != 1 || spy.personMsgs[0] != wantPerson {
		t.Fatalf("person messages = %#v, want [%q]", spy.personMsgs, wantPerson)
	}
}

func TestReplyDoesNotIncludeNameCardForOtherPlatforms(t *testing.T) {
	ctx, spy := newNameCardTestContext(t)
	ctx.EndPoint.ProtocolType = "onebot"
	ctx.Player.AutoSetNameTemplate = "茶叶 HP10/10"
	msg := &Message{
		MessageType: "group",
		GroupID:     ctx.Group.GroupID,
		Sender: SenderBase{
			UserID:   ctx.Player.UserID,
			Nickname: ctx.Player.Name,
		},
	}

	ReplyGroup(ctx, msg, "D100=50")

	if len(spy.groupMsgs) != 1 || spy.groupMsgs[0] != "D100=50" {
		t.Fatalf("group messages = %#v, want [%q]", spy.groupMsgs, "D100=50")
	}
}
