package group_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	milky "github.com/Szzrain/Milky-go-sdk"

	. "sealdice-core/api/v2/group"
	"sealdice-core/dice"
	"sealdice-core/logger"
)

type quitGroupPlatform struct {
	mu    sync.Mutex
	quits []int64
}

func newQQQuitGroupService(t *testing.T, roles map[int64]string) (*GroupService, *quitGroupPlatform) {
	t.Helper()
	platform := &quitGroupPlatform{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			GroupID int64 `json:"group_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode platform request: %v", err)
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/" + milky.EndpointGetGroupMemberInfo:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "ok", "retcode": 0,
				"data": map[string]any{"member": map[string]any{"role": roles[payload.GroupID]}},
			})
		case "/" + milky.EndpointQuitGroup:
			platform.mu.Lock()
			platform.quits = append(platform.quits, payload.GroupID)
			platform.mu.Unlock()
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
	if err != nil || first.Body.ConfirmCode == "" {
		t.Fatalf("expected owner confirmation: response=%#v err=%v", first, err)
	}
	group, _ := svc.Dice().ImSession.ServiceAtNew.Load(req.Body.GroupID)
	if len(platform.snapshot()) != 0 || !group.DiceIDExistsMap.Exists(req.Body.DiceID) {
		t.Fatal("first request must preserve membership without quitting")
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

func TestBatchQQQuitWaitsForOwnerConfirmationAndSkipsUnknownRole(t *testing.T) {
	svc, platform := newQQQuitGroupService(t, map[int64]string{2010: "owner", 2020: "member", 2030: "unknown"})
	req := &BatchQuitGroupReq{Body: BatchQuitGroupRequest{
		GroupIDs: []string{"QQ-Group:2010", "QQ-Group:2020", "QQ-Group:2030"}, Silence: true,
	}}
	first, err := svc.BatchQuitGroup(t.Context(), req)
	if err != nil || first.Body.Item != 0 || len(first.Body.Confirmations) != 1 || len(platform.snapshot()) != 0 {
		t.Fatalf("batch must wait without partially quitting: response=%#v err=%v", first, err)
	}
	confirmation := first.Body.Confirmations[0]
	if confirmation.GroupID != "QQ-Group:2010" || confirmation.DiceID != "QQ:1001" || confirmation.ConfirmCode == "" {
		t.Fatalf("confirmation must identify the owned group and account: %#v", confirmation)
	}
	req.Body.ConfirmCodes = map[string]string{confirmation.GroupID: confirmation.ConfirmCode}
	second, err := svc.BatchQuitGroup(t.Context(), req)
	if err != nil || second.Body.Item != 2 || len(second.Body.Skipped) != 1 || len(second.Body.Confirmations) != 0 || len(platform.snapshot()) != 2 {
		t.Fatalf("confirmed batch must skip the unknown role: response=%#v err=%v", second, err)
	}
	group, _ := svc.Dice().ImSession.ServiceAtNew.Load("QQ-Group:2030")
	if !group.DiceIDExistsMap.Exists("QQ:1001") {
		t.Fatal("unknown role must preserve group membership")
	}
}
