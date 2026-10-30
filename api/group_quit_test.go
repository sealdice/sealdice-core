package api //nolint:testpackage

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	milky "github.com/Szzrain/Milky-go-sdk"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"sealdice-core/dice"
)

func TestLegacyGroupQuitBlocksOwnersAndUnknownIdentity(t *testing.T) {
	for _, role := range []string{"owner", "unknown"} {
		t.Run(role, func(t *testing.T) {
			previousDice, previousDM := myDice, dm
			t.Cleanup(func() { myDice, dm = previousDice, previousDM })
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.TrimPrefix(r.URL.Path, "/") != milky.EndpointGetGroupMemberInfo {
					t.Errorf("unexpected platform action before confirmation: %s", r.URL.Path)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				var payload struct {
					GroupID int64 `json:"group_id"`
					UserID  int64 `json:"user_id"`
					NoCache bool  `json:"no_cache"`
				}
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Errorf("decode member lookup: %v", err)
				}
				if payload.GroupID != 2010 || payload.UserID != 1001 || !payload.NoCache {
					t.Errorf("unexpected identity query: %#v", payload)
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"status": "ok", "retcode": 0,
					"data": map[string]any{"member": map[string]any{
						"group_id": payload.GroupID, "user_id": payload.UserID, "role": role,
					}},
				})
			}))
			defer server.Close()
			log := zap.NewNop().Sugar()
			intent, err := milky.New("ws://127.0.0.1", server.URL, "", log)
			if err != nil {
				t.Fatal(err)
			}
			d := &dice.Dice{Logger: log}
			d.Config = dice.NewConfig(d)
			dm = &dice.DiceManager{Dice: d}
			d.Parent = dm
			dm.AccessTokens.Store("group-quit-test", true)
			group := &dice.GroupInfo{GroupID: "QQ-Group:2010", DiceIDExistsMap: new(dice.SyncMap[string, bool])}
			group.DiceIDExistsMap.Store("QQ:1001", true)
			ep := &dice.EndPointInfo{EndPointInfoBase: dice.EndPointInfoBase{
				ID: t.Name(), UserID: "QQ:1001", Platform: "QQ", ProtocolType: "milky",
			}}
			ep.Adapter = &dice.PlatformAdapterMilky{EndPoint: ep, IntentSession: intent}
			d.ImSession = &dice.IMSession{Parent: d, EndPoints: []*dice.EndPointInfo{ep}, ServiceAtNew: new(dice.SyncMap[string, *dice.GroupInfo])}
			d.ImSession.ServiceAtNew.Store(group.GroupID, group)
			myDice = d

			req := httptest.NewRequest(http.MethodPost, "/sd-api/group/quit", strings.NewReader(`{"groupId":"QQ-Group:2010","diceId":"QQ:1001","silence":true}`))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			req.Header.Set("Token", "group-quit-test")
			rec := httptest.NewRecorder()
			if err := groupQuit(echo.New().NewContext(req, rec)); err != nil {
				t.Fatal(err)
			}
			if rec.Code != http.StatusConflict || !group.DiceIDExistsMap.Exists(ep.UserID) {
				t.Fatalf("blocked legacy quit must preserve membership: status=%d body=%s", rec.Code, rec.Body.String())
			}
			var result dice.GroupQuitConfirmation
			if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if (result.ConfirmCode != "") != (role == "owner") {
				t.Fatalf("only verified owners should get a confirmation code: %#v", result)
			}
		})
	}
}
