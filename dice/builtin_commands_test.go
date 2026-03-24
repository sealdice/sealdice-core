//nolint:testpackage
package dice

import (
	"errors"
	"strings"
	"testing"

	milky "github.com/Szzrain/Milky-go-sdk"
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

func TestShouldDismissRequireOwnerConfirmMilkyOwner(t *testing.T) {
	orig := milkyGetGroupMemberInfo
	t.Cleanup(func() {
		milkyGetGroupMemberInfo = orig
	})
	milkyGetGroupMemberInfo = func(_ *milky.Session, groupID, userID int64, noCache bool) (*milky.GroupMemberInfo, error) {
		if groupID != 2005 || userID != 10001 || noCache {
			t.Fatalf("unexpected lookup args: groupID=%d userID=%d noCache=%v", groupID, userID, noCache)
		}
		return &milky.GroupMemberInfo{Role: "owner"}, nil
	}

	ctx := &MsgContext{
		EndPoint: &EndPointInfo{
			EndPointInfoBase: EndPointInfoBase{
				UserID:       "QQ:10001",
				Platform:     "QQ",
				ProtocolType: "milky",
			},
			Adapter: &PlatformAdapterMilky{
				IntentSession: &milky.Session{},
			},
		},
	}

	needConfirm, checked, detail := shouldDismissRequireOwnerConfirm(ctx, "QQ-Group:2005")
	if !checked {
		t.Fatalf("expected milky owner check to succeed, detail=%q", detail)
	}
	if !needConfirm {
		t.Fatalf("expected milky owner to require confirmation, detail=%q", detail)
	}
	if detail != "owner" {
		t.Fatalf("expected owner detail, got %q", detail)
	}
}

func TestShouldDismissRequireOwnerConfirmMilkyMember(t *testing.T) {
	orig := milkyGetGroupMemberInfo
	t.Cleanup(func() {
		milkyGetGroupMemberInfo = orig
	})
	milkyGetGroupMemberInfo = func(_ *milky.Session, _, _ int64, _ bool) (*milky.GroupMemberInfo, error) {
		return &milky.GroupMemberInfo{Role: "member"}, nil
	}

	ctx := &MsgContext{
		EndPoint: &EndPointInfo{
			EndPointInfoBase: EndPointInfoBase{
				UserID:       "QQ:10002",
				Platform:     "QQ",
				ProtocolType: "milky",
			},
			Adapter: &PlatformAdapterMilky{
				IntentSession: &milky.Session{},
			},
		},
	}

	needConfirm, checked, detail := shouldDismissRequireOwnerConfirm(ctx, "QQ-Group:2006")
	if !checked {
		t.Fatalf("expected milky member check to succeed, detail=%q", detail)
	}
	if needConfirm {
		t.Fatalf("expected milky member not to require owner confirmation, detail=%q", detail)
	}
	if detail != "member" {
		t.Fatalf("expected member detail, got %q", detail)
	}
}

func newMilkyQuitCommandTestContext(t *testing.T, d *Dice, senderID, groupID, groupName string) (*MsgContext, *Message, *[]int64, func()) {
	t.Helper()

	origGet := milkyGetGroupMemberInfo
	origSend := milkySendGroupMessage
	origQuit := milkyQuitGroup
	restore := func() {
		milkyGetGroupMemberInfo = origGet
		milkySendGroupMessage = origSend
		milkyQuitGroup = origQuit
	}

	quitCalls := []int64{}
	milkyGetGroupMemberInfo = func(_ *milky.Session, groupIDInt, userIDInt int64, noCache bool) (*milky.GroupMemberInfo, error) {
		if groupIDInt != 2010 || userIDInt != 10010 || noCache {
			t.Fatalf("unexpected lookup args: groupID=%d userID=%d noCache=%v", groupIDInt, userIDInt, noCache)
		}
		return &milky.GroupMemberInfo{Role: "owner"}, nil
	}
	milkySendGroupMessage = func(_ *milky.Session, _ int64, _ *[]milky.IMessageElement) (*milky.MessageRet, error) {
		return &milky.MessageRet{}, nil
	}
	milkyQuitGroup = func(_ *milky.Session, groupIDInt int64) error {
		quitCalls = append(quitCalls, groupIDInt)
		return nil
	}

	ep := &EndPointInfo{
		EndPointInfoBase: EndPointInfoBase{
			UserID:       "QQ:10010",
			Platform:     "QQ",
			ProtocolType: "milky",
			Nickname:     "MilkyBot",
		},
	}
	ep.Adapter = &PlatformAdapterMilky{
		EndPoint:      ep,
		Session:       d.ImSession,
		IntentSession: &milky.Session{},
	}

	d.Config.BotExitWithoutAt = true
	ctx, msg := newQuitCommandTestContext(t, d, ep, senderID, groupID, groupName)
	return ctx, msg, &quitCalls, restore
}

func TestDismissMilkyOwnerRequiresConfirmationBeforeQuit(t *testing.T) {
	d, _, _, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	ctx, msg, quitCalls, restore := newMilkyQuitCommandTestContext(t, d, "QQ:9010", "QQ-Group:2010", "MilkyDismissGroup")
	t.Cleanup(restore)

	first := d.CmdMap["dismiss"].Solve(ctx, msg, &CmdArgs{})
	if !first.Matched || !first.Solved {
		t.Fatalf("unexpected initial result: %#v", first)
	}
	if len(*quitCalls) != 0 {
		t.Fatalf("dismiss should not quit before confirmation, got %#v", *quitCalls)
	}

	confirmKey := getDismissConfirmKeyForGroup(ctx, msg.Sender.UserID, msg.GroupID)
	confirmCode, ok := loadDismissConfirmCode(confirmKey)
	if !ok {
		t.Fatal("expected milky dismiss confirmation code to be stored")
	}

	second := d.CmdMap["dismiss"].Solve(ctx, msg, &CmdArgs{Args: []string{confirmCode}})
	if !second.Matched || !second.Solved {
		t.Fatalf("unexpected confirmed result: %#v", second)
	}
	if len(*quitCalls) != 1 || (*quitCalls)[0] != 2010 {
		t.Fatalf("expected exactly one milky quit call for group 2010, got %#v", *quitCalls)
	}
}

func TestDismissMilkyLookupErrorFallsBackToSafetyConfirmation(t *testing.T) {
	d, _, _, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	origGet := milkyGetGroupMemberInfo
	origSend := milkySendGroupMessage
	origQuit := milkyQuitGroup
	t.Cleanup(func() {
		milkyGetGroupMemberInfo = origGet
		milkySendGroupMessage = origSend
		milkyQuitGroup = origQuit
	})

	var quitCalls []int64
	milkyGetGroupMemberInfo = func(_ *milky.Session, _, _ int64, _ bool) (*milky.GroupMemberInfo, error) {
		return nil, errors.New("milky lookup failed")
	}
	milkySendGroupMessage = func(_ *milky.Session, _ int64, _ *[]milky.IMessageElement) (*milky.MessageRet, error) {
		return &milky.MessageRet{}, nil
	}
	milkyQuitGroup = func(_ *milky.Session, groupIDInt int64) error {
		quitCalls = append(quitCalls, groupIDInt)
		return nil
	}

	ep := &EndPointInfo{
		EndPointInfoBase: EndPointInfoBase{
			UserID:       "QQ:10020",
			Platform:     "QQ",
			ProtocolType: "milky",
			Nickname:     "MilkyBot",
		},
	}
	ep.Adapter = &PlatformAdapterMilky{
		EndPoint:      ep,
		Session:       d.ImSession,
		IntentSession: &milky.Session{},
	}

	d.Config.BotExitWithoutAt = true
	ctx, msg := newQuitCommandTestContext(t, d, ep, "QQ:9020", "QQ-Group:2020", "MilkyFallbackGroup")

	needConfirm, checked, detail := shouldDismissRequireOwnerConfirm(ctx, msg.GroupID)
	if checked || needConfirm {
		t.Fatalf("expected milky lookup error to fail direct role check, got needConfirm=%v checked=%v detail=%q", needConfirm, checked, detail)
	}
	if !strings.Contains(detail, "milky lookup failed") {
		t.Fatalf("expected milky lookup error detail, got %q", detail)
	}

	first := d.CmdMap["dismiss"].Solve(ctx, msg, &CmdArgs{})
	if !first.Matched || !first.Solved {
		t.Fatalf("unexpected initial result: %#v", first)
	}
	if len(quitCalls) != 0 {
		t.Fatalf("dismiss should not quit before confirmation on lookup error, got %#v", quitCalls)
	}

	confirmKey := getDismissConfirmKeyForGroup(ctx, msg.Sender.UserID, msg.GroupID)
	confirmCode, ok := loadDismissConfirmCode(confirmKey)
	if !ok {
		t.Fatal("expected dismiss confirmation code to be stored on lookup error")
	}

	second := d.CmdMap["dismiss"].Solve(ctx, msg, &CmdArgs{Args: []string{confirmCode}})
	if !second.Matched || !second.Solved {
		t.Fatalf("unexpected confirmed result: %#v", second)
	}
	if len(quitCalls) != 1 || quitCalls[0] != 2020 {
		t.Fatalf("expected exactly one milky quit call for group 2020, got %#v", quitCalls)
	}
}

func TestBotByeMilkyOwnerRequiresConfirmationBeforeQuit(t *testing.T) {
	d, _, _, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	ctx, msg, quitCalls, restore := newMilkyQuitCommandTestContext(t, d, "QQ:9011", "QQ-Group:2010", "MilkyBotByeGroup")
	t.Cleanup(restore)

	first := d.CmdMap["bot"].Solve(ctx, msg, &CmdArgs{Args: []string{"bye"}})
	if !first.Matched || !first.Solved {
		t.Fatalf("unexpected initial result: %#v", first)
	}
	if len(*quitCalls) != 0 {
		t.Fatalf(".bot bye should not quit before confirmation, got %#v", *quitCalls)
	}

	confirmKey := getDismissConfirmKeyForGroup(ctx, msg.Sender.UserID, msg.GroupID)
	confirmCode, ok := loadDismissConfirmCode(confirmKey)
	if !ok {
		t.Fatal("expected .bot bye confirmation code to be stored")
	}

	second := d.CmdMap["bot"].Solve(ctx, msg, &CmdArgs{Args: []string{"bye", confirmCode}})
	if !second.Matched || !second.Solved {
		t.Fatalf("unexpected confirmed result: %#v", second)
	}
	if len(*quitCalls) != 1 || (*quitCalls)[0] != 2010 {
		t.Fatalf("expected exactly one milky quit call for group 2010, got %#v", *quitCalls)
	}
}

func TestBotExitMilkyOwnerRequiresConfirmationBeforeQuit(t *testing.T) {
	d, _, _, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	ctx, msg, quitCalls, restore := newMilkyQuitCommandTestContext(t, d, "QQ:9013", "QQ-Group:2010", "MilkyBotExitGroup")
	t.Cleanup(restore)

	first := d.CmdMap["bot"].Solve(ctx, msg, &CmdArgs{Args: []string{"exit"}})
	if !first.Matched || !first.Solved {
		t.Fatalf("unexpected initial result: %#v", first)
	}
	if len(*quitCalls) != 0 {
		t.Fatalf(".bot exit should not quit before confirmation, got %#v", *quitCalls)
	}

	confirmKey := getDismissConfirmKeyForGroup(ctx, msg.Sender.UserID, msg.GroupID)
	confirmCode, ok := loadDismissConfirmCode(confirmKey)
	if !ok {
		t.Fatal("expected .bot exit confirmation code to be stored")
	}

	second := d.CmdMap["bot"].Solve(ctx, msg, &CmdArgs{Args: []string{"exit", confirmCode}})
	if !second.Matched || !second.Solved {
		t.Fatalf("unexpected confirmed result: %#v", second)
	}
	if len(*quitCalls) != 1 || (*quitCalls)[0] != 2010 {
		t.Fatalf("expected exactly one milky quit call for group 2010, got %#v", *quitCalls)
	}
}

func TestBotQuitMilkyOwnerRequiresConfirmationBeforeQuit(t *testing.T) {
	d, _, _, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	ctx, msg, quitCalls, restore := newMilkyQuitCommandTestContext(t, d, "QQ:9012", "QQ-Group:2010", "MilkyBotQuitGroup")
	t.Cleanup(restore)

	first := d.CmdMap["bot"].Solve(ctx, msg, &CmdArgs{Args: []string{"quit"}})
	if !first.Matched || !first.Solved {
		t.Fatalf("unexpected initial result: %#v", first)
	}
	if len(*quitCalls) != 0 {
		t.Fatalf(".bot quit should not quit before confirmation, got %#v", *quitCalls)
	}

	confirmKey := getDismissConfirmKeyForGroup(ctx, msg.Sender.UserID, msg.GroupID)
	confirmCode, ok := loadDismissConfirmCode(confirmKey)
	if !ok {
		t.Fatal("expected .bot quit confirmation code to be stored")
	}

	second := d.CmdMap["bot"].Solve(ctx, msg, &CmdArgs{Args: []string{"quit", confirmCode}})
	if !second.Matched || !second.Solved {
		t.Fatalf("unexpected confirmed result: %#v", second)
	}
	if len(*quitCalls) != 1 || (*quitCalls)[0] != 2010 {
		t.Fatalf("expected exactly one milky quit call for group 2010, got %#v", *quitCalls)
	}
}

func TestShouldDismissRequireOwnerConfirmMilkyNilMember(t *testing.T) {
	orig := milkyGetGroupMemberInfo
	t.Cleanup(func() {
		milkyGetGroupMemberInfo = orig
	})
	milkyGetGroupMemberInfo = func(_ *milky.Session, _, _ int64, _ bool) (*milky.GroupMemberInfo, error) {
		return nil, nil
	}

	ctx := &MsgContext{
		EndPoint: &EndPointInfo{
			EndPointInfoBase: EndPointInfoBase{
				UserID:       "QQ:10003",
				Platform:     "QQ",
				ProtocolType: "milky",
			},
			Adapter: &PlatformAdapterMilky{
				IntentSession: &milky.Session{},
			},
		},
	}

	needConfirm, checked, detail := shouldDismissRequireOwnerConfirm(ctx, "QQ-Group:2007")
	if checked || needConfirm {
		t.Fatalf("expected milky nil member result to fail check, got needConfirm=%v checked=%v detail=%q", needConfirm, checked, detail)
	}
	if !strings.Contains(detail, "returned nil") {
		t.Fatalf("expected nil-member detail, got %q", detail)
	}
}

func TestShouldDismissRequireOwnerConfirmMilkyEmptyRole(t *testing.T) {
	orig := milkyGetGroupMemberInfo
	t.Cleanup(func() {
		milkyGetGroupMemberInfo = orig
	})
	milkyGetGroupMemberInfo = func(_ *milky.Session, _, _ int64, _ bool) (*milky.GroupMemberInfo, error) {
		return &milky.GroupMemberInfo{}, nil
	}

	ctx := &MsgContext{
		EndPoint: &EndPointInfo{
			EndPointInfoBase: EndPointInfoBase{
				UserID:       "QQ:10004",
				Platform:     "QQ",
				ProtocolType: "milky",
			},
			Adapter: &PlatformAdapterMilky{
				IntentSession: &milky.Session{},
			},
		},
	}

	needConfirm, checked, detail := shouldDismissRequireOwnerConfirm(ctx, "QQ-Group:2008")
	if checked || needConfirm {
		t.Fatalf("expected milky empty role result to fail check, got needConfirm=%v checked=%v detail=%q", needConfirm, checked, detail)
	}
	if !strings.Contains(detail, "empty role") {
		t.Fatalf("expected empty-role detail, got %q", detail)
	}
}
