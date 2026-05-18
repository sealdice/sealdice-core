package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"sealdice-core/api/v2/config"
	configm "sealdice-core/api/v2/model/config"
	"sealdice-core/dice"
	"sealdice-core/logger"
	"sealdice-core/model/common/request"
)

func newTestService(t *testing.T) *config.Service {
	t.Helper()

	dataDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dataDir, "configs"), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}

	d := &dice.Dice{
		Logger: logger.M(),
	}
	d.BaseConfig.Name = "test"
	d.BaseConfig.DataDir = dataDir
	d.Config = dice.NewConfig(d)
	d.ImSession = &dice.IMSession{
		Parent:       d,
		EndPoints:    []*dice.EndPointInfo{},
		ServiceAtNew: new(dice.SyncMap[string, *dice.GroupInfo]),
		PendingQuits: new(dice.SyncMap[string, *dice.PendingQuitInfo]),
	}
	d.AttrsManager = &dice.AttrsManager{}
	d.AttrsManager.Init(d)
	dm := &dice.DiceManager{
		Dice: []*dice.Dice{d},
	}
	d.Parent = dm
	return config.NewService(dm)
}

func TestReplyConfigRoundTrip(t *testing.T) {
	svc := newTestService(t)
	defer svc.Dice().AttrsManager.Stop()

	getResp, err := svc.GetReplyConfig(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("GetReplyConfig returned error: %v", err)
	}
	if getResp.Body.Item.CustomReplyConfigEnable {
		t.Fatalf("default custom reply switch should be false")
	}

	setResp, err := svc.SetReplyConfig(t.Context(), &configm.ReplyConfigReq{
		Body: request.RequestWrapper[configm.ReplyModuleConfig]{
			Body: configm.ReplyModuleConfig{CustomReplyConfigEnable: true},
		},
	})
	if err != nil {
		t.Fatalf("SetReplyConfig returned error: %v", err)
	}
	if !setResp.Body.Item.CustomReplyConfigEnable {
		t.Fatalf("custom reply switch should be true")
	}
}

func TestAdvancedConfigRoundTrip(t *testing.T) {
	svc := newTestService(t)
	defer svc.Dice().AttrsManager.Stop()

	_, err := svc.SetAdvancedConfig(t.Context(), &configm.AdvancedConfigReq{
		Body: request.RequestWrapper[dice.AdvancedConfig]{
			Body: dice.AdvancedConfig{
				Show:                 true,
				Enable:               true,
				StoryLogBackendUrl:   "https://example.com",
				StoryLogApiVersion:   "v2",
				StoryLogBackendToken: "token",
			},
		},
	})
	if err != nil {
		t.Fatalf("SetAdvancedConfig returned error: %v", err)
	}

	getResp, err := svc.GetAdvancedConfig(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("GetAdvancedConfig returned error: %v", err)
	}
	if !getResp.Body.Item.Show || !getResp.Body.Item.Enable {
		t.Fatalf("advanced config flags not persisted")
	}
	if getResp.Body.Item.StoryLogBackendUrl != "https://example.com" {
		t.Fatalf("StoryLogBackendUrl = %q, want https://example.com", getResp.Body.Item.StoryLogBackendUrl)
	}
}
