//nolint:testpackage
package dice

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	ds "github.com/sealdice/dicescript"

	"sealdice-core/model"
	"sealdice-core/utils/constant"
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

func TestReplyToSenderEvaluatesNameCardBeforeReturning(t *testing.T) {
	for _, tc := range []struct {
		name        string
		send        func(*MsgContext, *Message, string)
		wantMessage string
	}{
		{"checked", ReplyToSender, "<b>HP10</b>\ndelegate:alias:reply"},
		{"unchecked", ReplyToSenderNoCheck, "delegate:alias:<b>HP10</b>\nreply"},
	} {
		for _, messageType := range []string{"group", "private"} {
			t.Run(tc.name+"/"+messageType, func(t *testing.T) {
				ctx, spy := newNameCardTestContext(t)
				ctx.MessageType = messageType
				ctx.Player.AutoSetNameTemplate = "HP{reviewValue}"
				ctx.Dice.Config.OfficialQQNameCardTmpl = "<b>{card}</b>"
				ctx.AliasPrefixText = "alias:"
				ctx.DelegateText = "delegate:"
				ctx.CreateVmIfNotExists()

				var evaluations atomic.Int32
				var hp atomic.Int64
				hp.Store(10)
				ctx.vm.GlobalValueLoadOverwriteFunc = func(_ string, _ *ds.VMValue) *ds.VMValue {
					evaluations.Add(1)
					return ds.NewIntVal(ds.IntType(hp.Load()))
				}
				msg := &Message{
					MessageType: messageType,
					GroupID:     ctx.Group.GroupID,
					Sender:      SenderBase{UserID: ctx.Player.UserID},
				}

				tc.send(ctx, msg, "reply")
				if got := evaluations.Load(); got != 1 {
					t.Errorf("name-card evaluations before return = %d, want 1", got)
				}
				// 后续命令处理可以继续使用 VM，已排队的回复应保留发送时的名片。
				hp.Store(20)
				if result := ctx.Eval("40+2", nil); result.vm.Error != nil {
					t.Errorf("subsequent evaluation failed: %v", result.vm.Error)
				}
				reply, ok := spy.waitForMsg(2 * time.Second)
				if !ok {
					t.Fatal("timed out waiting for reply")
				}
				if reply != tc.wantMessage {
					t.Errorf("reply = %q, want %q", reply, tc.wantMessage)
				}
				if got := evaluations.Load(); got != 1 {
					t.Errorf("name-card evaluations after send = %d, want 1", got)
				}
			})
		}
	}
}

func TestOfficialQQPCTagWithNameCard(t *testing.T) {
	ctx, spy := newNameCardTestContext(t)
	if err := ctx.Dice.DBOperator.GetDataDB(constant.WRITE).AutoMigrate(&model.AttributesItemModel{}); err != nil {
		t.Fatal(err)
	}
	ctx.Group.System = "coc7"
	ctx.SystemTemplate = ctx.Group.GetCharTemplate(ctx.Dice)
	ctx.Player.AutoSetNameTemplate = "{$t玩家_RAW} SAN{理智} HP{生命值}/{生命值上限} DEX{敏捷}"
	character, err := ctx.Dice.AttrsManager.CharNew(ctx.Player.UserID, "调查员", "coc7")
	if err != nil {
		t.Fatal(err)
	}
	msg := newGroupMsg(ctx.Group.GroupID, ctx.Player.UserID, ".pc tag 1")
	args := CommandParse(msg.Message, []string{"pc"}, []string{"."}, msg.Platform, false)
	result := ctx.Dice.CmdMap["pc"].Solve(ctx, msg, args)
	if !result.Solved {
		t.Fatal("pc tag was not handled")
	}
	reply, ok := spy.waitForMsg(2 * time.Second)
	if !ok {
		t.Fatal("timed out waiting for pc tag reply")
	}
	if !strings.Contains(reply, "调查员 SAN") || strings.Count(reply, "<font") != 1 || strings.Contains(reply, "错误:") {
		t.Fatalf("unexpected pc tag reply: %q", reply)
	}
	bindingID, err := ctx.Dice.AttrsManager.CharGetBindingId(ctx.Group.GroupID, ctx.Player.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if bindingID != character.Id {
		t.Fatalf("bound character = %q, want %q", bindingID, character.Id)
	}
}
