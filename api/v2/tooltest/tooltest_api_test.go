package tooltest

import (
	"testing"
	"time"

	"sealdice-core/dice"
	"sealdice-core/logger"
	"sealdice-core/model/common/request"
)

func newTestService(t *testing.T) (*Service, *dice.PlatformAdapterHTTP) {
	t.Helper()

	d := &dice.Dice{
		Logger:      logger.M(),
		CmdMap:      dice.CmdMapCls{},
		ExtRegistry: new(dice.SyncMap[string, *dice.ExtInfo]),
	}
	d.ImSession = &dice.IMSession{
		Parent:       d,
		ServiceAtNew: new(dice.SyncMap[string, *dice.GroupInfo]),
		PendingQuits: new(dice.SyncMap[string, *dice.PendingQuitInfo]),
	}

	dm := &dice.DiceManager{
		Dice: []*dice.Dice{d},
	}
	d.Parent = dm
	d.Config = dice.NewConfig(d)
	d.CommandPrefix = []string{"."}

	adapter := &dice.PlatformAdapterHTTP{
		Session: d.ImSession,
	}
	ep := &dice.EndPointInfo{
		EndPointInfoBase: dice.EndPointInfoBase{
			ID:       "ui-endpoint",
			UserID:   "UI:1000",
			Nickname: "SealDice",
			Platform: "UI",
			Enable:   true,
			Session:  d.ImSession,
		},
		Adapter: adapter,
	}
	adapter.EndPoint = ep
	d.UIEndpoint = ep

	svc := NewService(dm)
	svc.now = func() time.Time {
		return time.UnixMilli(10_000)
	}
	return svc, adapter
}

func TestPostMessageBuildsPrivateUISender(t *testing.T) {
	svc, adapter := newTestService(t)

	var dispatched *dice.Message
	svc.dispatch = func(_ *dice.EndPointInfo, msg *dice.Message) {
		dispatched = msg
		adapter.RecentMessage = append(adapter.RecentMessage, dice.HTTPSimpleMessage{
			UID:         msg.Sender.UserID,
			Message:     "pong",
			MessageType: msg.MessageType,
		})
	}

	resp, err := svc.PostMessage(t.Context(), &PostMessageReq{
		Body: PostMessageReqBody{
			Text: ".ping",
			Mode: "private",
		},
	})
	if err != nil {
		t.Fatalf("PostMessage returned error: %v", err)
	}
	if !resp.Body.Item.Success {
		t.Fatalf("PostMessage success = false, want true")
	}
	if dispatched == nil {
		t.Fatal("dispatch was not called")
	}
	if dispatched.MessageType != "private" {
		t.Fatalf("MessageType = %q, want private", dispatched.MessageType)
	}
	if dispatched.Sender.UserID != "UI:1001" {
		t.Fatalf("Sender.UserID = %q, want UI:1001", dispatched.Sender.UserID)
	}
	if dispatched.GroupID != "" {
		t.Fatalf("GroupID = %q, want empty", dispatched.GroupID)
	}
}

func TestPostMessageBuildsGroupUISender(t *testing.T) {
	svc, _ := newTestService(t)

	var dispatched *dice.Message
	svc.dispatch = func(_ *dice.EndPointInfo, msg *dice.Message) {
		dispatched = msg
	}

	_, err := svc.PostMessage(t.Context(), &PostMessageReq{
		Body: PostMessageReqBody{
			Text: ".ping",
			Mode: "group",
		},
	})
	if err != nil {
		t.Fatalf("PostMessage returned error: %v", err)
	}
	if dispatched == nil {
		t.Fatal("dispatch was not called")
	}
	if dispatched.MessageType != "group" {
		t.Fatalf("MessageType = %q, want group", dispatched.MessageType)
	}
	if dispatched.Sender.UserID != "UI:1002" {
		t.Fatalf("Sender.UserID = %q, want UI:1002", dispatched.Sender.UserID)
	}
	if dispatched.Sender.GroupRole != "owner" {
		t.Fatalf("Sender.GroupRole = %q, want owner", dispatched.Sender.GroupRole)
	}
	if dispatched.GroupID != "UI-Group:2001" {
		t.Fatalf("GroupID = %q, want UI-Group:2001", dispatched.GroupID)
	}
}

func TestPostMessageRateLimitsPerMode(t *testing.T) {
	svc, _ := newTestService(t)
	svc.dispatch = func(_ *dice.EndPointInfo, _ *dice.Message) {}

	_, err := svc.PostMessage(t.Context(), &PostMessageReq{
		Body: PostMessageReqBody{Text: ".a", Mode: "private"},
	})
	if err != nil {
		t.Fatalf("first private PostMessage returned error: %v", err)
	}

	_, err = svc.PostMessage(t.Context(), &PostMessageReq{
		Body: PostMessageReqBody{Text: ".b", Mode: "private"},
	})
	if err == nil {
		t.Fatal("second private PostMessage returned nil error, want rate limit rejection")
	}

	_, err = svc.PostMessage(t.Context(), &PostMessageReq{
		Body: PostMessageReqBody{Text: ".c", Mode: "group"},
	})
	if err != nil {
		t.Fatalf("group PostMessage returned error: %v, want independent limiter", err)
	}
}

func TestGetPendingMessagesReturnsAndClearsQueue(t *testing.T) {
	svc, adapter := newTestService(t)
	adapter.RecentMessage = []dice.HTTPSimpleMessage{
		{UID: "UI:1001", Message: "hello", MessageType: "private"},
		{UID: "UI:1002", Message: "world", MessageType: "group"},
	}

	firstResp, err := svc.GetPendingMessages(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("GetPendingMessages returned error: %v", err)
	}
	if len(firstResp.Body.Item.Items) != 2 {
		t.Fatalf("pending count = %d, want 2", len(firstResp.Body.Item.Items))
	}
	if len(adapter.RecentMessage) != 0 {
		t.Fatalf("adapter queue length = %d, want 0 after read", len(adapter.RecentMessage))
	}

	secondResp, err := svc.GetPendingMessages(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("second GetPendingMessages returned error: %v", err)
	}
	if len(secondResp.Body.Item.Items) != 0 {
		t.Fatalf("second pending count = %d, want 0", len(secondResp.Body.Item.Items))
	}
}

func TestGetCommandsIncludesBaseAndExtensionCommandsSortedByLength(t *testing.T) {
	svc, _ := newTestService(t)
	svc.dice.CmdMap["a"] = &dice.CmdItemInfo{Name: "a"}
	svc.dice.CmdMap["alphabet"] = &dice.CmdItemInfo{Name: "alphabet"}
	svc.dice.ExtList = []*dice.ExtInfo{
		{CmdMap: dice.CmdMapCls{
			"mid": &dice.CmdItemInfo{Name: "mid"},
		}},
	}

	resp, err := svc.GetCommands(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("GetCommands returned error: %v", err)
	}

	got := resp.Body.Item.Items
	if len(got) != 3 {
		t.Fatalf("command count = %d, want 3", len(got))
	}
	if got[0] != "alphabet" || got[1] != "mid" || got[2] != "a" {
		t.Fatalf("commands = %#v, want [alphabet mid a]", got)
	}
}
