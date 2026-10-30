package group_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	milky "github.com/Szzrain/Milky-go-sdk"

	. "sealdice-core/api/v2/group"
	"sealdice-core/dice"
	"sealdice-core/logger"
)

type quitGroupPlatform struct {
	mu      sync.Mutex
	roles   map[int64]string
	quits   []int64
	failure bool
}

func newQQQuitGroupService(t *testing.T, roles map[int64]string) (*GroupService, *quitGroupPlatform) {
	t.Helper()
	platform := &quitGroupPlatform{roles: roles}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			GroupID int64 `json:"group_id"`
			UserID  int64 `json:"user_id"`
			NoCache bool  `json:"no_cache"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode platform request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		platform.mu.Lock()
		defer platform.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch strings.TrimPrefix(r.URL.Path, "/") {
		case milky.EndpointGetGroupMemberInfo:
			if !payload.NoCache || payload.UserID != 1001 {
				t.Errorf("identity must be refreshed for the selected bot: %#v", payload)
			}
			if platform.failure {
				_ = json.NewEncoder(w).Encode(map[string]any{"status": "failed", "retcode": 1, "message": "lookup failed"})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "ok", "retcode": 0,
				"data": map[string]any{"member": map[string]any{
					"group_id": payload.GroupID, "user_id": payload.UserID, "role": platform.roles[payload.GroupID],
				}},
			})
		case milky.EndpointQuitGroup:
			platform.quits = append(platform.quits, payload.GroupID)
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "retcode": 0})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	intent, err := milky.New("ws://127.0.0.1", server.URL, "", logger.M())
	if err != nil {
		t.Fatal(err)
	}
	svc := newTestGroupService(t)
	ep := &dice.EndPointInfo{EndPointInfoBase: dice.EndPointInfoBase{
		ID: t.Name(), UserID: "QQ:1001", Platform: "QQ", ProtocolType: "milky", Enable: true,
	}}
	ep.Adapter = &dice.PlatformAdapterMilky{EndPoint: ep, IntentSession: intent}
	ep.Session = svc.Dice().ImSession
	svc.Dice().ImSession.EndPoints = []*dice.EndPointInfo{ep}
	for groupID := range roles {
		gid := "QQ-Group:" + strconv.FormatInt(groupID, 10)
		svc.Dice().ImSession.ServiceAtNew.Store(gid, newTestGroup(gid, gid, ep.UserID))
	}
	return svc, platform
}

func (p *quitGroupPlatform) snapshot() []int64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]int64(nil), p.quits...)
}

func TestQQOwnerQuitRequiresSecondRequest(t *testing.T) {
	svc, platform := newQQQuitGroupService(t, map[int64]string{2010: "owner"})
	req := &QuitGroupReq{Body: QuitGroupRequest{GroupID: "QQ-Group:2010", DiceID: "QQ:1001", Silence: true}}
	first, err := svc.QuitGroup(t.Context(), req)
	if err != nil || first.Body.ConfirmCode == "" || !strings.Contains(first.Body.Message, "解散") {
		t.Fatalf("expected explicit dissolution warning and code: response=%#v err=%v", first, err)
	}
	group, _ := svc.Dice().ImSession.ServiceAtNew.Load(req.Body.GroupID)
	if len(platform.snapshot()) != 0 || !group.DiceIDExistsMap.Exists(req.Body.DiceID) {
		t.Fatal("first request must not quit or remove the bot from the group")
	}
	req.Body.ConfirmCode = "invalid"
	invalid, err := svc.QuitGroup(t.Context(), req)
	if err != nil || invalid.Body.Message == "ok" || len(platform.snapshot()) != 0 {
		t.Fatal("invalid confirmation must not quit")
	}
	req.Body.ConfirmCode = first.Body.ConfirmCode
	confirmed, err := svc.QuitGroup(t.Context(), req)
	if err != nil || confirmed.Body.Message != "ok" {
		t.Fatalf("confirmed quit: response=%#v err=%v", confirmed, err)
	}
	if quits := platform.snapshot(); len(quits) != 1 || quits[0] != 2010 || group.DiceIDExistsMap.Exists(req.Body.DiceID) {
		t.Fatalf("confirmed quit did not update the correct group: %v", quits)
	}
}

func TestQQQuitBlocksUnconfirmedIdentity(t *testing.T) {
	for _, tt := range []struct {
		name    string
		role    string
		failure bool
	}{
		{name: "lookup error", failure: true},
		{name: "unknown role", role: "unknown"},
		{name: "empty role"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc, platform := newQQQuitGroupService(t, map[int64]string{2010: tt.role})
			platform.mu.Lock()
			platform.failure = tt.failure
			platform.mu.Unlock()
			resp, err := svc.QuitGroup(t.Context(), &QuitGroupReq{Body: QuitGroupRequest{
				GroupID: "QQ-Group:2010", DiceID: "QQ:1001", Silence: true, ConfirmCode: "1234",
			}})
			if err != nil || resp.Body.Message == "ok" || resp.Body.ConfirmCode != "" || len(platform.snapshot()) != 0 {
				t.Fatalf("failed identity must block the quit: response=%#v err=%v", resp, err)
			}
			group, _ := svc.Dice().ImSession.ServiceAtNew.Load("QQ-Group:2010")
			if !group.DiceIDExistsMap.Exists("QQ:1001") {
				t.Fatal("blocked quit must preserve group membership")
			}
		})
	}
}

func TestBatchQQQuitWaitsForOwnerConfirmation(t *testing.T) {
	svc, platform := newQQQuitGroupService(t, map[int64]string{2010: "owner", 2020: "member"})
	req := &BatchQuitGroupReq{Body: BatchQuitGroupRequest{
		GroupIDs: []string{"QQ-Group:2010", "QQ-Group:2020"}, Silence: true,
	}}
	first, err := svc.BatchQuitGroup(t.Context(), req)
	if err != nil || first.Body.Item != 0 || len(first.Body.Confirmations) != 1 || len(platform.snapshot()) != 0 {
		t.Fatalf("batch must wait without partially quitting: response=%#v err=%v", first, err)
	}
	confirmation := first.Body.Confirmations[0]
	if confirmation.GroupID != "QQ-Group:2010" || confirmation.DiceID != "QQ:1001" || confirmation.ConfirmCode == "" {
		t.Fatalf("confirmation must identify the exact owned group and account: %#v", confirmation)
	}
	req.Body.ConfirmCodes = map[string]string{confirmation.GroupID: confirmation.ConfirmCode}
	second, err := svc.BatchQuitGroup(t.Context(), req)
	if err != nil || second.Body.Item != 2 || len(second.Body.Confirmations) != 0 || len(platform.snapshot()) != 2 {
		t.Fatalf("confirmed batch failed: response=%#v err=%v", second, err)
	}
}

func TestBatchQQQuitSkipsIdentityFailures(t *testing.T) {
	svc, platform := newQQQuitGroupService(t, map[int64]string{2010: "unknown", 2020: "admin"})
	resp, err := svc.BatchQuitGroup(t.Context(), &BatchQuitGroupReq{Body: BatchQuitGroupRequest{
		GroupIDs: []string{"QQ-Group:2010", "QQ-Group:2020", "QQ-Group:2020"}, Silence: true,
	}})
	if err != nil || resp.Body.Item != 1 || len(resp.Body.Skipped) != 1 {
		t.Fatalf("batch should quit only verified non-owners once: response=%#v err=%v", resp, err)
	}
	if quits := platform.snapshot(); len(quits) != 1 || quits[0] != 2020 {
		t.Fatalf("unexpected platform quits: %v", quits)
	}
	group, _ := svc.Dice().ImSession.ServiceAtNew.Load("QQ-Group:2010")
	if !group.DiceIDExistsMap.Exists("QQ:1001") {
		t.Fatal("failed identity group must retain membership")
	}
}

func TestBatchQQQuitPreservesConfirmationsWhenOwnershipChanges(t *testing.T) {
	svc, platform := newQQQuitGroupService(t, map[int64]string{2010: "owner", 2020: "member"})
	req := &BatchQuitGroupReq{Body: BatchQuitGroupRequest{
		GroupIDs: []string{"QQ-Group:2010", "QQ-Group:2020"}, Silence: true,
	}}
	first, err := svc.BatchQuitGroup(t.Context(), req)
	if err != nil || len(first.Body.Confirmations) != 1 {
		t.Fatalf("expected first owner confirmation: response=%#v err=%v", first, err)
	}
	initial := first.Body.Confirmations[0]
	req.Body.ConfirmCodes = map[string]string{initial.GroupID: initial.ConfirmCode}
	platform.mu.Lock()
	platform.roles[2020] = "owner"
	platform.mu.Unlock()
	second, err := svc.BatchQuitGroup(t.Context(), req)
	if err != nil || second.Body.Item != 0 || len(second.Body.Confirmations) != 2 || len(platform.snapshot()) != 0 {
		t.Fatalf("new ownership must wait and preserve earlier confirmations: response=%#v err=%v", second, err)
	}
	for _, confirmation := range second.Body.Confirmations {
		req.Body.ConfirmCodes[confirmation.GroupID] = confirmation.ConfirmCode
	}
	if req.Body.ConfirmCodes[initial.GroupID] != initial.ConfirmCode {
		t.Fatal("valid confirmation code must be preserved while confirming another owner")
	}
	confirmed, err := svc.BatchQuitGroup(t.Context(), req)
	if err != nil || confirmed.Body.Item != 2 || len(platform.snapshot()) != 2 {
		t.Fatalf("confirmed owners should quit once each: response=%#v err=%v", confirmed, err)
	}
}
