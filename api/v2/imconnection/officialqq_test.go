package imconnection_test

import (
	"testing"

	. "sealdice-core/api/v2/imconnection"
	"sealdice-core/dice"
	"sealdice-core/logger"
)

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
	dm := &dice.DiceManager{Dice: []*dice.Dice{d}}
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
