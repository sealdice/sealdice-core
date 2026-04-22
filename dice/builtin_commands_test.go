//nolint:testpackage
package dice

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	milky "github.com/Szzrain/Milky-go-sdk"
	ds "github.com/sealdice/dicescript"
)

const (
	testMilkyBotUserID               = "QQ:10010"
	testMilkyBotQQID           int64 = 10010
	testMilkyOwnerGroupID            = "QQ-Group:2010"
	testMilkyOwnerGroupQQID    int64 = 2010
	testMilkyFallbackGroupID         = "QQ-Group:2020"
	testMilkyFallbackGroupQQID int64 = 2020
	testMilkyOwnerRole               = "owner"
	testMilkyOwnerRoleUpper          = "OWNER"
	testMilkyOwnerRoleChinese        = "群主"
	testMilkyAdminRoleChinese        = "管理员"
)

type noopMilkyLogger struct{}

func (noopMilkyLogger) Infof(string, ...interface{})  {}
func (noopMilkyLogger) Errorf(string, ...interface{}) {}
func (noopMilkyLogger) Debugf(string, ...interface{}) {}
func (noopMilkyLogger) Warnf(string, ...interface{})  {}
func (noopMilkyLogger) Info(...interface{})           {}
func (noopMilkyLogger) Error(...interface{})          {}
func (noopMilkyLogger) Debug(...interface{})          {}
func (noopMilkyLogger) Warn(...interface{})           {}

type milkyRESTHarness struct {
	t               *testing.T
	role            string
	apiError        string
	expectedGroupID int64
	expectedUserID  int64

	mu         sync.Mutex
	quitGroups []int64
}

func newMilkyRESTHarness(t *testing.T, role string, apiError string, expectedGroupID, expectedUserID int64) (*milky.Session, *milkyRESTHarness, func()) {
	t.Helper()

	h := &milkyRESTHarness{
		t:               t,
		role:            role,
		apiError:        apiError,
		expectedGroupID: expectedGroupID,
		expectedUserID:  expectedUserID,
	}

	server := httptest.NewServer(http.HandlerFunc(h.handle))
	session, err := milky.New("ws://127.0.0.1", server.URL, "", noopMilkyLogger{})
	if err != nil {
		server.Close()
		t.Fatalf("failed to create milky session: %v", err)
	}

	return session, h, server.Close
}

func (h *milkyRESTHarness) handle(w http.ResponseWriter, r *http.Request) {
	h.t.Helper()
	w.Header().Set("Content-Type", "application/json")

	switch strings.TrimPrefix(r.URL.Path, "/") {
	case milky.EndpointGetGroupMemberInfo:
		var payload struct {
			GroupID int64 `json:"group_id"`
			UserID  int64 `json:"user_id"`
			NoCache bool  `json:"no_cache"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			h.t.Fatalf("failed to decode get_group_member_info payload: %v", err)
		}
		if payload.GroupID != h.expectedGroupID || payload.UserID != h.expectedUserID || payload.NoCache {
			h.t.Fatalf("unexpected get_group_member_info payload: %#v", payload)
		}
		if h.apiError != "" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":  "failed",
				"retcode": 1,
				"message": h.apiError,
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "ok",
			"retcode": 0,
			"data": map[string]any{
				"member": map[string]any{
					"group_id": h.expectedGroupID,
					"user_id":  h.expectedUserID,
					"role":     h.role,
				},
			},
		})
	case milky.EndpointSendGroupMessage:
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "ok",
			"retcode": 0,
			"data": map[string]any{
				"message_seq": 1,
				"time":        1,
			},
		})
	case milky.EndpointQuitGroup:
		var payload struct {
			GroupID int64 `json:"group_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			h.t.Fatalf("failed to decode quit_group payload: %v", err)
		}
		h.mu.Lock()
		h.quitGroups = append(h.quitGroups, payload.GroupID)
		h.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "ok",
			"retcode": 0,
			"data":    nil,
		})
	default:
		http.NotFound(w, r)
	}
}

func (h *milkyRESTHarness) quitGroupSnapshot() []int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]int64(nil), h.quitGroups...)
}

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

func TestShouldDismissRequireOwnerConfirmMilkyRoleNormalization(t *testing.T) {
	tests := []struct {
		name            string
		groupQQID       int64
		userQQID        int64
		role            string
		wantNeedConfirm bool
		wantChecked     bool
		wantDetail      string
	}{
		{
			name:            "lowercase owner",
			groupQQID:       2005,
			userQQID:        10001,
			role:            testMilkyOwnerRole,
			wantNeedConfirm: true,
			wantChecked:     true,
			wantDetail:      "owner",
		},
		{
			name:            "uppercase owner",
			groupQQID:       2006,
			userQQID:        10002,
			role:            testMilkyOwnerRoleUpper,
			wantNeedConfirm: true,
			wantChecked:     true,
			wantDetail:      "owner",
		},
		{
			name:            "chinese owner",
			groupQQID:       2007,
			userQQID:        10003,
			role:            testMilkyOwnerRoleChinese,
			wantNeedConfirm: true,
			wantChecked:     true,
			wantDetail:      "owner",
		},
		{
			name:            "chinese admin",
			groupQQID:       2008,
			userQQID:        10004,
			role:            testMilkyAdminRoleChinese,
			wantNeedConfirm: false,
			wantChecked:     true,
			wantDetail:      "admin",
		},
		{
			name:            "member",
			groupQQID:       2009,
			userQQID:        10005,
			role:            "member",
			wantNeedConfirm: false,
			wantChecked:     true,
			wantDetail:      "member",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupID := "QQ-Group:" + strconv.FormatInt(tt.groupQQID, 10)
			userID := "QQ:" + strconv.FormatInt(tt.userQQID, 10)
			session, _, cleanup := newMilkyRESTHarness(t, tt.role, "", tt.groupQQID, tt.userQQID)
			defer cleanup()

			pa := &PlatformAdapterMilky{
				IntentSession: session,
			}

			ctx := &MsgContext{
				EndPoint: &EndPointInfo{
					EndPointInfoBase: EndPointInfoBase{
						UserID:       userID,
						Platform:     "QQ",
						ProtocolType: "milky",
					},
					Adapter: pa,
				},
			}

			needConfirm, checked, detail := shouldDismissRequireOwnerConfirm(ctx, groupID)
			if checked != tt.wantChecked {
				t.Fatalf("checked = %v, want %v, detail=%q", checked, tt.wantChecked, detail)
			}
			if needConfirm != tt.wantNeedConfirm {
				t.Fatalf("needConfirm = %v, want %v, detail=%q", needConfirm, tt.wantNeedConfirm, detail)
			}
			if detail != tt.wantDetail {
				t.Fatalf("detail = %q, want %q", detail, tt.wantDetail)
			}
		})
	}
}

func newMilkyQuitCommandTestContext(t *testing.T, d *Dice, senderID, groupID, groupName string) (*MsgContext, *Message, *milkyRESTHarness, func()) {
	t.Helper()
	d.ExtList = nil

	session, harness, cleanup := newMilkyRESTHarness(t, testMilkyOwnerRole, "", testMilkyOwnerGroupQQID, testMilkyBotQQID)
	pa := &PlatformAdapterMilky{}
	ep := &EndPointInfo{
		EndPointInfoBase: EndPointInfoBase{
			UserID:       testMilkyBotUserID,
			Platform:     "QQ",
			ProtocolType: "milky",
			Nickname:     "MilkyBot",
		},
	}
	pa.EndPoint = ep
	pa.Session = d.ImSession
	pa.IntentSession = session
	ep.Adapter = pa

	d.Config.BotExitWithoutAt = true
	ctx, msg := newQuitCommandTestContext(t, d, ep, senderID, groupID, groupName)
	return ctx, msg, harness, cleanup
}

func TestDismissMilkyOwnerRequiresConfirmationBeforeQuit(t *testing.T) {
	d, _, _, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	ctx, msg, harness, closeHarness := newMilkyQuitCommandTestContext(t, d, "QQ:9010", testMilkyOwnerGroupID, "MilkyDismissGroup")
	defer closeHarness()

	first := d.CmdMap["dismiss"].Solve(ctx, msg, &CmdArgs{})
	if !first.Matched || !first.Solved {
		t.Fatalf("unexpected initial result: %#v", first)
	}
	if got := harness.quitGroupSnapshot(); len(got) != 0 {
		t.Fatalf("dismiss should not quit before confirmation, got %#v", got)
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
	if got := harness.quitGroupSnapshot(); len(got) != 1 || got[0] != testMilkyOwnerGroupQQID {
		t.Fatalf("expected exactly one milky quit call for group 2010, got %#v", got)
	}
}

func TestDismissMilkyLookupErrorFallsBackToSafetyConfirmation(t *testing.T) {
	d, _, _, cleanup := newExecuteNewTestDice(t)
	defer cleanup()
	d.ExtList = nil

	session, harness, closeHarness := newMilkyRESTHarness(t, "", "milky lookup failed", testMilkyFallbackGroupQQID, 10020)
	defer closeHarness()

	pa := &PlatformAdapterMilky{}
	ep := &EndPointInfo{
		EndPointInfoBase: EndPointInfoBase{
			UserID:       "QQ:10020",
			Platform:     "QQ",
			ProtocolType: "milky",
			Nickname:     "MilkyBot",
		},
	}
	pa.EndPoint = ep
	pa.Session = d.ImSession
	pa.IntentSession = session
	ep.Adapter = pa

	d.Config.BotExitWithoutAt = true
	ctx, msg := newQuitCommandTestContext(t, d, ep, "QQ:9020", testMilkyFallbackGroupID, "MilkyFallbackGroup")

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
	if got := harness.quitGroupSnapshot(); len(got) != 0 {
		t.Fatalf("dismiss should not quit before confirmation on lookup error, got %#v", got)
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
	if got := harness.quitGroupSnapshot(); len(got) != 1 || got[0] != testMilkyFallbackGroupQQID {
		t.Fatalf("expected exactly one milky quit call for group 2020, got %#v", got)
	}
}

func TestBotByeMilkyOwnerRequiresConfirmationBeforeQuit(t *testing.T) {
	d, _, _, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	ctx, msg, harness, closeHarness := newMilkyQuitCommandTestContext(t, d, "QQ:9011", testMilkyOwnerGroupID, "MilkyBotByeGroup")
	defer closeHarness()

	first := d.CmdMap["bot"].Solve(ctx, msg, &CmdArgs{Args: []string{"bye"}})
	if !first.Matched || !first.Solved {
		t.Fatalf("unexpected initial result: %#v", first)
	}
	if got := harness.quitGroupSnapshot(); len(got) != 0 {
		t.Fatalf(".bot bye should not quit before confirmation, got %#v", got)
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
	if got := harness.quitGroupSnapshot(); len(got) != 1 || got[0] != testMilkyOwnerGroupQQID {
		t.Fatalf("expected exactly one milky quit call for group 2010, got %#v", got)
	}
}

func TestBotExitMilkyOwnerRequiresConfirmationBeforeQuit(t *testing.T) {
	d, _, _, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	ctx, msg, harness, closeHarness := newMilkyQuitCommandTestContext(t, d, "QQ:9013", testMilkyOwnerGroupID, "MilkyBotExitGroup")
	defer closeHarness()

	first := d.CmdMap["bot"].Solve(ctx, msg, &CmdArgs{Args: []string{"exit"}})
	if !first.Matched || !first.Solved {
		t.Fatalf("unexpected initial result: %#v", first)
	}
	if got := harness.quitGroupSnapshot(); len(got) != 0 {
		t.Fatalf(".bot exit should not quit before confirmation, got %#v", got)
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
	if got := harness.quitGroupSnapshot(); len(got) != 1 || got[0] != testMilkyOwnerGroupQQID {
		t.Fatalf("expected exactly one milky quit call for group 2010, got %#v", got)
	}
}

func TestBotQuitMilkyOwnerRequiresConfirmationBeforeQuit(t *testing.T) {
	d, _, _, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	ctx, msg, harness, closeHarness := newMilkyQuitCommandTestContext(t, d, "QQ:9012", testMilkyOwnerGroupID, "MilkyBotQuitGroup")
	defer closeHarness()

	first := d.CmdMap["bot"].Solve(ctx, msg, &CmdArgs{Args: []string{"quit"}})
	if !first.Matched || !first.Solved {
		t.Fatalf("unexpected initial result: %#v", first)
	}
	if got := harness.quitGroupSnapshot(); len(got) != 0 {
		t.Fatalf(".bot quit should not quit before confirmation, got %#v", got)
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
	if got := harness.quitGroupSnapshot(); len(got) != 1 || got[0] != testMilkyOwnerGroupQQID {
		t.Fatalf("expected exactly one milky quit call for group 2010, got %#v", got)
	}
}

func TestShouldDismissRequireOwnerConfirmMilkyEmptyRole(t *testing.T) {
	session, _, cleanup := newMilkyRESTHarness(t, "", "", 2008, 10004)
	defer cleanup()

	pa := &PlatformAdapterMilky{
		IntentSession: session,
	}

	ctx := &MsgContext{
		EndPoint: &EndPointInfo{
			EndPointInfoBase: EndPointInfoBase{
				UserID:       "QQ:10004",
				Platform:     "QQ",
				ProtocolType: "milky",
			},
			Adapter: pa,
		},
	}

	needConfirm, checked, detail := shouldDismissRequireOwnerConfirm(ctx, "QQ-Group:2008")
	if checked || needConfirm {
		t.Fatalf("expected milky empty role result to fail check, got needConfirm=%v checked=%v detail=%q", needConfirm, checked, detail)
	}
	if !strings.Contains(detail, errGetGroupMemberInfoEmptyRole) {
		t.Fatalf("expected empty-role detail, got %q", detail)
	}
}
