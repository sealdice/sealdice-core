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

func TestLegacyGroupQuitRequiresOwnerConfirmation(t *testing.T) {
	previousDice, previousDM := myDice, dm
	t.Cleanup(func() { myDice, dm = previousDice, previousDM })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+milky.EndpointGetGroupMemberInfo {
			t.Errorf("unexpected platform action before confirmation: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","retcode":0,"data":{"member":{"role":"owner"}}}`))
	}))
	t.Cleanup(server.Close)
	log := zap.NewNop().Sugar()
	intent, err := milky.New("ws://127.0.0.1", server.URL, "", log)
	if err != nil {
		t.Fatal(err)
	}
	d := &dice.Dice{Logger: log}
	d.Config = dice.NewConfig(d)
	dm = &dice.DiceManager{Dice: d}
	d.Parent, myDice = dm, d
	dm.AccessTokens.Store("group-quit-test", true)
	group := &dice.GroupInfo{GroupID: "QQ-Group:2010", DiceIDExistsMap: new(dice.SyncMap[string, bool])}
	group.DiceIDExistsMap.Store("QQ:1001", true)
	ep := &dice.EndPointInfo{EndPointInfoBase: dice.EndPointInfoBase{
		ID: t.Name(), UserID: "QQ:1001", Platform: "QQ", ProtocolType: "milky",
	}}
	ep.Adapter = &dice.PlatformAdapterMilky{EndPoint: ep, IntentSession: intent}
	d.ImSession = &dice.IMSession{Parent: d, EndPoints: []*dice.EndPointInfo{ep}, ServiceAtNew: new(dice.SyncMap[string, *dice.GroupInfo])}
	d.ImSession.ServiceAtNew.Store(group.GroupID, group)

	req := httptest.NewRequest(http.MethodPost, "/sd-api/group/quit", strings.NewReader(`{"groupId":"QQ-Group:2010","diceId":"QQ:1001","silence":true}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Token", "group-quit-test")
	rec := httptest.NewRecorder()
	if err := groupQuit(echo.New().NewContext(req, rec)); err != nil {
		t.Fatal(err)
	}
	var result dice.GroupQuitConfirmation
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusConflict || result.ConfirmCode == "" || !group.DiceIDExistsMap.Exists(ep.UserID) {
		t.Fatalf("owner quit must wait for confirmation and preserve membership: status=%d body=%s", rec.Code, rec.Body.String())
	}
}
