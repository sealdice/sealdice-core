//nolint:testpackage // Tests intentionally exercise unexported service hooks.
package tooltest

import (
	"encoding/json"
	"strings"
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

	adapter := &dice.PlatformAdapterHTTP{}
	ep := &dice.EndPointInfo{
		EndPointInfoBase: dice.EndPointInfoBase{
			ID:       "ui-endpoint",
			UserID:   "UI:1000",
			Nickname: "SealDice",
			Platform: "UI",
			Enable:   true,
		},
		Adapter: adapter,
	}
	adapter.EndPoint = ep
	ep.BindRuntime(d.ImSession)
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

func TestPostMessageUsesSelectedUISender(t *testing.T) {
	svc, _ := newTestService(t)

	var dispatched *dice.Message
	svc.dispatch = func(_ *dice.EndPointInfo, msg *dice.Message) {
		dispatched = msg
	}

	var req PostMessageReq
	if err := json.Unmarshal([]byte(`{
		"body": {
			"text": ".ping",
			"mode": "group",
			"senderId": "UI:1007",
			"groupId": "UI-Group:2001"
		}
	}`), &req); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	_, err := svc.PostMessage(t.Context(), &req)
	if err != nil {
		t.Fatalf("PostMessage returned error: %v", err)
	}
	if dispatched == nil {
		t.Fatal("dispatch was not called")
	}
	if dispatched.Sender.UserID != "UI:1007" {
		t.Fatalf("Sender.UserID = %q, want selected sender UI:1007", dispatched.Sender.UserID)
	}
	if dispatched.GroupID != "UI-Group:2001" {
		t.Fatalf("GroupID = %q, want selected group UI-Group:2001", dispatched.GroupID)
	}
	if dispatched.UITestRole != "blacklisted" {
		t.Fatalf("UITestRole = %q, want blacklisted", dispatched.UITestRole)
	}
}

func TestPostMessageCarriesSplitLenFromJSONBody(t *testing.T) {
	svc, _ := newTestService(t)

	var req PostMessageReq
	if err := json.Unmarshal([]byte(`{
		"body": {
			"text": ".ping",
			"mode": "private",
			"messageSplitLen": 300
		}
	}`), &req); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}

	var dispatched *dice.Message
	svc.dispatch = func(_ *dice.EndPointInfo, msg *dice.Message) {
		dispatched = msg
	}

	if _, err := svc.PostMessage(t.Context(), &req); err != nil {
		t.Fatalf("PostMessage returned error: %v", err)
	}
	if dispatched == nil {
		t.Fatal("dispatch was not called")
	}
	if dispatched.UITestReplySplitLen == nil || *dispatched.UITestReplySplitLen != 300 {
		t.Fatalf("UITestReplySplitLen = %#v, want 300", dispatched.UITestReplySplitLen)
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

func TestGetContextSeedsPersistentUITestProfiles(t *testing.T) {
	svc, _ := newTestService(t)

	resp, err := svc.GetContext(t.Context(), &ContextReq{Mode: "group", GroupID: "UI-Group:2001"})
	if err != nil {
		t.Fatalf("GetContext returned error: %v", err)
	}
	if len(resp.Body.Item.Members) != 7 {
		t.Fatalf("member count = %d, want 7", len(resp.Body.Item.Members))
	}
	if resp.Body.Item.Members[1].UserID != "UI:1002" || resp.Body.Item.Members[1].Role != UITestRoleOwner {
		t.Fatalf("owner profile = %#v", resp.Body.Item.Members[1])
	}

	group, ok := svc.dice.ImSession.ServiceAtNew.Load("UI-Group:2001")
	if !ok || group.UITest == nil || !group.UITest.Initialized {
		t.Fatalf("UI test config was not initialized: %#v", group)
	}
}

func TestUpdateProfileChangesPlayerAndGroupMetadata(t *testing.T) {
	svc, _ := newTestService(t)

	var req UpdateProfileReq
	req.Body.Mode = "group"
	req.Body.GroupID = "UI-Group:2001"
	req.Body.UserID = "UI:1003"
	req.Body.Name = "新的管理员"
	req.Body.Role = UITestRoleAdmin
	req.Body.AvatarKey = "owner"
	enabled := true
	req.Body.Enabled = &enabled

	resp, err := svc.UpdateProfile(t.Context(), &req)
	if err != nil {
		t.Fatalf("UpdateProfile returned error: %v", err)
	}
	if resp.Body.Item.CurrentSenderID != "UI:1003" {
		t.Fatalf("current sender = %q, want UI:1003", resp.Body.Item.CurrentSenderID)
	}
	group, _ := svc.dice.ImSession.ServiceAtNew.Load("UI-Group:2001")
	player, _ := group.Players.Load("UI:1003")
	if player.Name != "新的管理员" {
		t.Fatalf("player name = %q, want updated name", player.Name)
	}
	if group.UITest.Members["UI:1003"].AvatarKey != "owner" {
		t.Fatalf("avatar key = %q, want owner", group.UITest.Members["UI:1003"].AvatarKey)
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
		{Name: "测试扩展", AutoActive: true, CmdMap: dice.CmdMapCls{
			"mid": &dice.CmdItemInfo{Name: "mid"},
			"a":   &dice.CmdItemInfo{Name: "a"},
		}},
	}

	resp, err := svc.GetCommands(t.Context(), &ContextReq{Mode: privateMode, SenderID: "UI:1001"})
	if err != nil {
		t.Fatalf("GetCommands returned error: %v", err)
	}

	got := resp.Body.Item.Items
	if len(got) != 3 {
		t.Fatalf("command count = %d, want 3", len(got))
	}
	if got[0].Name != "alphabet" || got[1].Name != "mid" || got[2].Name != "a" {
		t.Fatalf("commands = %#v, want [alphabet mid a]", got)
	}
}

func TestGetCommandsIncludesMetadataAndContextPrefixes(t *testing.T) {
	svc, _ := newTestService(t)
	svc.dice.CommandPrefix = []string{".", "!"}
	svc.dice.CmdMap["reply"] = &dice.CmdItemInfo{
		Name:      "reply",
		ShortHelp: ".reply on/off // 控制自动回复",
	}
	svc.dice.ExtList = []*dice.ExtInfo{
		{Name: "测试扩展", AutoActive: true, CmdMap: dice.CmdMapCls{
			"roll": &dice.CmdItemInfo{
				Name:      "roll",
				ShortHelp: ".roll // 投掷骰子",
			},
		}},
	}

	contextResp, err := svc.GetContext(t.Context(), &ContextReq{Mode: privateMode, SenderID: "UI:1001"})
	if err != nil {
		t.Fatalf("GetContext returned error: %v", err)
	}
	contextJSON, err := json.Marshal(contextResp.Body.Item)
	if err != nil {
		t.Fatalf("marshal context: %v", err)
	}
	if !strings.Contains(string(contextJSON), `"commandPrefix":[".","!"]`) {
		t.Fatalf("context JSON = %s, want configured command prefixes", contextJSON)
	}

	commandsResp, err := svc.GetCommands(t.Context(), &ContextReq{Mode: privateMode, SenderID: "UI:1001"})
	if err != nil {
		t.Fatalf("GetCommands returned error: %v", err)
	}
	commandsJSON, err := json.Marshal(commandsResp.Body.Item)
	if err != nil {
		t.Fatalf("marshal commands: %v", err)
	}
	for _, fragment := range []string{
		`"name":"reply"`,
		`"description":".reply on/off // 控制自动回复"`,
		`"source":"核心"`,
		`"name":"roll"`,
		`"source":"测试扩展"`,
	} {
		if !strings.Contains(string(commandsJSON), fragment) {
			t.Fatalf("commands JSON = %s, missing %s", commandsJSON, fragment)
		}
	}
}
