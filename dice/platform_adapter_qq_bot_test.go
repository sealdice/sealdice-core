//nolint:testpackage // Tests cover unexported adapter conversion and dispatch helpers.
package dice

import (
	"encoding/json"
	"testing"
	"time"

	"go.uber.org/zap"

	"sealdice-core/message"
)

func TestIsQQBotUserID(t *testing.T) {
	tests := []struct {
		name   string
		userID string
		want   bool
	}{
		{name: "single range 66600000", userID: "QQ:66600000", want: true},
		{name: "below 66600000", userID: "QQ:66599999", want: false},
		{name: "above 66600000", userID: "QQ:66600001", want: false},
		{name: "range 2854 lower bound", userID: "QQ:2854196301", want: true},
		{name: "range 2854 upper bound", userID: "QQ:2854216399", want: true},
		{name: "below range 2854", userID: "QQ:2854196300", want: false},
		{name: "above range 2854", userID: "QQ:2854216400", want: false},
		{name: "single range 3328144510", userID: "QQ:3328144510", want: true},
		{name: "below 3328144510", userID: "QQ:3328144509", want: false},
		{name: "above 3328144510", userID: "QQ:3328144511", want: false},
		{name: "range 3889 lower bound", userID: "QQ:3889000000", want: true},
		{name: "range 3889 upper bound", userID: "QQ:3889999999", want: true},
		{name: "below range 3889", userID: "QQ:3888999999", want: false},
		{name: "above range 3889", userID: "QQ:3890000000", want: false},
		{name: "range 4010 lower bound", userID: "QQ:4010000000", want: true},
		{name: "range 4010 upper bound", userID: "QQ:4019999999", want: true},
		{name: "below range 4010", userID: "QQ:4009999999", want: false},
		{name: "above range 4010", userID: "QQ:4020000000", want: false},
		{name: "empty UIN", userID: "QQ:", want: false},
		{name: "invalid UIN", userID: "QQ:not-a-number", want: false},
		{name: "signed UIN", userID: "QQ:+66600000", want: false},
		{name: "UIN with suffix", userID: "QQ:66600000-extra", want: false},
		{name: "QQ group", userID: "QQ-Group:4010000000", want: false},
		{name: "official QQ OpenID", userID: "OpenQQ:4010000000-member-open-id", want: false},
		{name: "other platform", userID: "TG:4010000000", want: false},
		{name: "empty user ID", userID: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isQQBotUserID(tt.userID); got != tt.want {
				t.Fatalf("isQQBotUserID(%q) = %t, want %t", tt.userID, got, tt.want)
			}
		})
	}
}

func TestGroupInfoIsBotCombinesManualAndVirtualBots(t *testing.T) {
	group := &GroupInfo{BotList: new(SyncMap[string, bool])}
	group.BotList.Store("QQ:12345678", true)

	if !group.IsBot("QQ:12345678", false) {
		t.Fatal("manually configured bot was not recognized")
	}
	if !group.IsBot("QQ:4010000000", true) {
		t.Fatal("adapter-detected QQ bot was not recognized")
	}
	if group.IsBot("QQ:12345679", false) {
		t.Fatal("ordinary QQ user was recognized as a bot")
	}

	group.BotList.Store("QQ:4010000000", true)
	group.BotList.Delete("QQ:4010000000")
	if !group.IsBot("QQ:4010000000", true) {
		t.Fatal("deleting a manual entry disabled virtual QQ bot recognition")
	}
	if group.BotList.Len() != 1 {
		t.Fatalf("automatic recognition changed the persisted bot list, length = %d", group.BotList.Len())
	}
}

func TestQQAdaptersPopulateRobotSenderInfo(t *testing.T) {
	legacy := (&MessageQQ{MessageQQBase: MessageQQBase{
		MessageType: "group",
		Sender:      &Sender{UserID: json.RawMessage("4010000000")},
	}}).toStdMessage()
	if !legacy.Sender.IsRobot {
		t.Fatal("legacy OneBot conversion did not mark a QQ bot UIN")
	}

	modern := (&MessageOBQQ{MessageQQOBBase: MessageQQOBBase{
		MessageType: "group",
		Sender:      &Sender{UserID: json.RawMessage("3889000000")},
	}}).toStdMessage()
	if !modern.Sender.IsRobot {
		t.Fatal("OneBot conversion did not mark a QQ bot UIN")
	}

	protocolDetected := (&MessageOBQQ{MessageQQOBBase: MessageQQOBBase{
		MessageType: "group",
		Sender: &Sender{
			UserID:  json.RawMessage("12345678"),
			IsRobot: true,
		},
	}}).toStdMessage()
	if !protocolDetected.Sender.IsRobot {
		t.Fatal("OneBot conversion discarded the protocol is_robot field")
	}
}

func TestOneBotConversionPopulatesRobotMentionInfo(t *testing.T) {
	raw := []byte(`{
		"time": 1,
		"message_id": 2,
		"message_type": "group",
		"group_id": 3,
		"sender": {"user_id": 12345678, "nickname": "user"},
		"message": [
			{"type": "text", "data": {"text": ".r 1d6"}},
			{"type": "at", "data": {"qq": "4010000000"}}
		]
	}`)

	msg, err := arrayByte2SealdiceMessage(zap.NewNop().Sugar(), raw)
	if err != nil {
		t.Fatalf("arrayByte2SealdiceMessage returned error: %v", err)
	}
	if len(msg.Segment) != 2 {
		t.Fatalf("converted segment count = %d, want 2", len(msg.Segment))
	}
	at, ok := msg.Segment[1].(*message.AtElement)
	if !ok {
		t.Fatalf("converted segment type = %T, want *message.AtElement", msg.Segment[1])
	}
	if !at.IsRobot {
		t.Fatal("OneBot conversion did not mark a mentioned QQ bot UIN")
	}
}

func TestLegacyAtParsePopulatesRobotInfo(t *testing.T) {
	_, at := AtParse("[CQ:at,qq=3889000000].r 1d6", "QQ")
	if len(at) != 1 || !at[0].IsRobot {
		t.Fatalf("legacy QQ mention was not marked as a robot: %#v", at)
	}
}

func TestGetCtxProxyAtPosRawSkipsQQBot(t *testing.T) {
	ctx := &MsgContext{
		EndPoint:     &EndPointInfo{EndPointInfoBase: EndPointInfoBase{UserID: "QQ:100000"}},
		Group:        &GroupInfo{BotList: new(SyncMap[string, bool])},
		DelegateText: "delegated roll",
	}
	cmdArgs := &CmdArgs{At: []*AtInfo{{UserID: "QQ:4010000000", IsRobot: true}}}

	got := GetCtxProxyAtPosRaw(ctx, cmdArgs, 0, false)
	if got != ctx {
		t.Fatal("QQ bot UIN was selected as a delegated roll target")
	}
	if ctx.DelegateText != "" {
		t.Fatalf("delegate text was not cleared, got %q", ctx.DelegateText)
	}
}

func TestExecuteNewFiltersQQBotInteractions(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	robotCommand := newGroupMsg("QQ-Group:81101", "QQ:4010000000", ".r 1d6")
	robotCommand.Sender.IsRobot = true
	d.ImSession.ExecuteNew(ep, robotCommand)
	assertNoAdapterMessage(t, adapter, "command sent by a QQ bot")

	mentionMsg := newGroupMsg("QQ-Group:81102", "QQ:12345678", ".r 1d6")
	mentionMsg.Segment = []message.IMessageElement{
		&message.TextElement{Content: ".r 1d6"},
		&message.AtElement{Target: "4010000000", IsRobot: true},
	}
	d.ImSession.ExecuteNew(ep, mentionMsg)
	assertNoAdapterMessage(t, adapter, "command mentioning another QQ bot")

	const hookGroupID = "QQ-Group:81103"
	setupCtx := &MsgContext{Dice: d, EndPoint: ep, Session: d.ImSession}
	hookGroup := SetBotOnAtGroup(setupCtx, hookGroupID)
	hookCalls := make(chan string, 1)
	hookGroup.SetActivatedExtList([]*ExtInfo{{
		Name: "qq-bot-filter-test",
		OnNotCommandReceived: func(_ *MsgContext, msg *Message) {
			hookCalls <- msg.Sender.UserID
		},
	}}, nil)

	robotText := newGroupMsg(hookGroupID, "QQ:3889000000", "plain text")
	robotText.Sender.IsRobot = true
	d.ImSession.ExecuteNew(ep, robotText)
	select {
	case senderID := <-hookCalls:
		t.Fatalf("non-command hook ran for QQ bot %q", senderID)
	case <-time.After(400 * time.Millisecond):
	}

	d.ImSession.ExecuteNew(ep, newGroupMsg(hookGroupID, "QQ:12345678", "plain text"))
	select {
	case senderID := <-hookCalls:
		if senderID != "QQ:12345678" {
			t.Fatalf("non-command hook sender = %q, want ordinary QQ user", senderID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("non-command hook did not run for an ordinary QQ user")
	}
}

func assertNoAdapterMessage(t *testing.T, adapter *mockPlatformAdapter, context string) {
	t.Helper()
	select {
	case unexpected := <-adapter.msgCh:
		t.Fatalf("unexpected reply for %s: %q", context, unexpected)
	case <-time.After(400 * time.Millisecond):
	}
}
