package imconnection_test

import (
	"errors"
	"testing"

	"github.com/danielgtaylor/huma/v2"

	. "sealdice-core/api/v2/imconnection"
	"sealdice-core/dice"
	"sealdice-core/logger"
)

func TestCreateOfficialQQValidatesCredentialsWithoutPriorProbe(t *testing.T) {
	for _, scenario := range []struct {
		name      string
		probeErr  error
		duplicate bool
		wantCode  int
	}{
		{name: "valid credentials"},
		{name: "invalid credentials", probeErr: errors.New("invalid credentials"), wantCode: 400},
		{name: "duplicate account", duplicate: true, wantCode: 409},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			_, d := newTestService(t, false)
			if scenario.duplicate {
				ep := dice.NewOfficialQQConnItem("existing", "secret", "202401", false)
				ep.UserID = "OpenQQ:202401"
				ep.BindRuntime(d.ImSession)
				d.ImSession.EndPoints = append(d.ImSession.EndPoints, ep)
			}
			before := len(d.ImSession.EndPoints)
			calls := 0
			svc := NewServiceWithOfficialProbe(d.Parent, false, false, func(appID, secret string) (*dice.OfficialQQAccountProbeResult, error) {
				calls++
				if appID != "10001" || secret != "secret" {
					t.Fatalf("unexpected credentials passed to probe")
				}
				return &dice.OfficialQQAccountProbeResult{UIN: "202401", Nickname: "Seal Bot"}, scenario.probeErr
			})
			resp, err := svc.CreateConnection(t.Context(), &CreateReq{Body: CreateBody{
				Platform: "officialqq", Config: map[string]interface{}{"appID": "10001", "appSecret": "secret"},
			}})
			if calls != 1 {
				t.Fatalf("probe calls = %d, want 1", calls)
			}
			if scenario.wantCode != 0 {
				var status huma.StatusError
				if !errors.As(err, &status) || status.GetStatus() != scenario.wantCode {
					t.Fatalf("error = %v, want status %d", err, scenario.wantCode)
				}
				if len(d.ImSession.EndPoints) != before {
					t.Fatal("rejected creation changed the connection list")
				}
				return
			}
			if err != nil || resp.Body.Item.UserID != "OpenQQ:202401" || len(d.ImSession.EndPoints) != before+1 {
				t.Fatalf("creation failed: response = %#v, error = %v", resp, err)
			}
		})
	}
}

func TestTestOfficialQQReturnsProbeMetadataAndDuplicateState(t *testing.T) {
	d := &dice.Dice{
		Logger: logger.M(),
	}
	d.BaseConfig.Name = "test"
	d.BaseConfig.DataDir = t.TempDir()
	d.ImSession = &dice.IMSession{
		Parent:       d,
		EndPoints:    []*dice.EndPointInfo{},
		ServiceAtNew: new(dice.SyncMap[string, *dice.GroupInfo]),
		PendingQuits: new(dice.SyncMap[string, *dice.PendingQuitInfo]),
	}
	dm := &dice.DiceManager{Dice: d}
	d.Parent = dm

	svc := NewServiceWithOfficialProbe(dm, false, false, func(_ string, _ string) (*dice.OfficialQQAccountProbeResult, error) {
		return &dice.OfficialQQAccountProbeResult{
			UIN:      "202401",
			Nickname: "Seal Bot",
		}, nil
	})

	existing := dice.NewOfficialQQConnItem("10001", "secret", "202401", false)
	existing.UserID = "OpenQQ:202401"
	existing.BindRuntime(d.ImSession)
	d.ImSession.EndPoints = append(d.ImSession.EndPoints, existing)

	resp, err := svc.TestOfficialQQ(t.Context(), &OfficialQQTestReq{
		Body: OfficialQQTestBody{
			Config: map[string]interface{}{
				"appID":     "10001",
				"appSecret": "secret",
			},
		},
	})
	if err != nil {
		t.Fatalf("TestOfficialQQ returned error: %v", err)
	}
	if !resp.Body.Item.TestOnly {
		t.Fatalf("TestOnly = false, want true")
	}
	if !resp.Body.Item.Exists {
		t.Fatalf("Exists = false, want true")
	}
	if resp.Body.Item.ID != existing.ID {
		t.Fatalf("ID = %q, want %q", resp.Body.Item.ID, existing.ID)
	}
	if resp.Body.Item.UserID != "OpenQQ:202401" {
		t.Fatalf("UserID = %q, want OpenQQ:202401", resp.Body.Item.UserID)
	}
	if resp.Body.Item.UIN != "202401" || resp.Body.Item.Nickname != "Seal Bot" {
		t.Fatalf("test result = %#v", resp.Body.Item)
	}
}
