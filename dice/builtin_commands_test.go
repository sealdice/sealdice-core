//nolint:testpackage
package dice

import (
	"testing"

	ds "github.com/sealdice/dicescript"
)

func newQuitCommandTestContext(t *testing.T, d *Dice, ep *EndPointInfo, senderID, groupID, groupName string) (*MsgContext, *Message) {
	t.Helper()

	group := &GroupInfo{
		Active:          true,
		GroupID:         groupID,
		GroupName:       groupName,
		DiceIDActiveMap: new(SyncMap[string, bool]),
		DiceIDExistsMap: new(SyncMap[string, bool]),
		BotList:         new(SyncMap[string, bool]),
		Players:         new(SyncMap[string, *GroupPlayerInfo]),
		PlayerGroups:    new(SyncMap[string, []string]),
	}
	group.DiceIDActiveMap.Store(ep.UserID, true)
	group.DiceIDExistsMap.Store(ep.UserID, true)

	player := &GroupPlayerInfo{
		Name:         "Tester",
		UserID:       senderID,
		ValueMapTemp: &ds.ValueMap{},
	}
	group.Players.Store(senderID, player)
	d.ImSession.ServiceAtNew.Store(groupID, group)

	ctx := &MsgContext{
		MessageType:     "group",
		Group:           group,
		Player:          player,
		EndPoint:        ep,
		Session:         d.ImSession,
		Dice:            d,
		IsCurGroupBotOn: true,
		PrivilegeLevel:  100,
	}

	msg := newGroupMsg(groupID, senderID, "")
	msg.GroupName = groupName
	return ctx, msg
}

func TestBotByeWithoutConfirmQuitsCurrentGroup(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	ep.Platform = "TG"
	d.Config.BotExitWithoutAt = true
	ctx, msg := newQuitCommandTestContext(t, d, ep, "TG:9001", "TG-Group:2001", "CurrentGroup")

	result := d.CmdMap["bot"].Solve(ctx, msg, &CmdArgs{Args: []string{"bye"}})
	if !result.Matched || !result.Solved {
		t.Fatalf("unexpected result: %#v", result)
	}

	if len(adapter.quitGroups) != 1 || adapter.quitGroups[0] != msg.GroupID {
		t.Fatalf("expected quit current group once, got %#v", adapter.quitGroups)
	}
}

func TestBotByeTargetGroupNormalizesAndQuitsTarget(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	ep.Platform = "TG"
	d.Config.BotExitWithoutAt = true
	ctx, msg := newQuitCommandTestContext(t, d, ep, "TG:9002", "TG-Group:2002", "CommandGroup")

	result := d.CmdMap["bot"].Solve(ctx, msg, &CmdArgs{Args: []string{"bye", "3003"}})
	if !result.Matched || !result.Solved {
		t.Fatalf("unexpected result: %#v", result)
	}

	wantGroupID := "TG-Group:3003"
	if len(adapter.quitGroups) != 1 || adapter.quitGroups[0] != wantGroupID {
		t.Fatalf("expected quit normalized target group %q, got %#v", wantGroupID, adapter.quitGroups)
	}
}

func TestBotByeWithExtraArgsShowsHelpAndDoesNotQuit(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	ep.Platform = "TG"
	d.Config.BotExitWithoutAt = true
	ctx, msg := newQuitCommandTestContext(t, d, ep, "TG:9003", "TG-Group:2003", "CommandGroup")

	result := d.CmdMap["bot"].Solve(ctx, msg, &CmdArgs{Args: []string{"bye", "3003", "1234", "extra"}})
	if !result.Matched || !result.Solved || !result.ShowHelp {
		t.Fatalf("expected help result, got %#v", result)
	}

	if len(adapter.quitGroups) != 0 {
		t.Fatalf("expected no quit for invalid args, got %#v", adapter.quitGroups)
	}
}

func TestDismissAcceptsFourDigitConfirmationCode(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	d.Config.BotExitWithoutAt = true
	ctx, msg := newQuitCommandTestContext(t, d, ep, "QQ:9004", "QQ-Group:2004", "DismissGroup")

	first := d.CmdMap["dismiss"].Solve(ctx, msg, &CmdArgs{})
	if !first.Matched || !first.Solved {
		t.Fatalf("unexpected initial result: %#v", first)
	}
	if len(adapter.quitGroups) != 0 {
		t.Fatalf("dismiss should not quit before confirmation, got %#v", adapter.quitGroups)
	}

	confirmKey := getDismissConfirmKeyForGroup(ctx, msg.Sender.UserID, msg.GroupID)
	confirmCode, ok := loadDismissConfirmCode(confirmKey)
	if !ok {
		t.Fatal("expected dismiss confirmation code to be stored")
	}

	second := d.CmdMap["dismiss"].Solve(ctx, msg, &CmdArgs{Args: []string{confirmCode}})
	if !second.Matched || !second.Solved {
		t.Fatalf("unexpected confirmed result: %#v", second)
	}

	if len(adapter.quitGroups) != 1 || adapter.quitGroups[0] != msg.GroupID {
		t.Fatalf("expected dismiss confirmation to quit current group, got %#v", adapter.quitGroups)
	}
}
