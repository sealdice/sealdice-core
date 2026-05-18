package base_test

import (
	"runtime"
	"testing"

	"sealdice-core/api/v2/base"
	"sealdice-core/dice"
	"sealdice-core/model/common/request"
)

func TestOverviewIncludesSplitBaseInfoFields(t *testing.T) {
	dm := &dice.DiceManager{
		UIPasswordSalt: "salt-value",
		JustForTest:    true,
		ContainerMode:  true,
		Dice:           []*dice.Dice{{Parent: nil}},
	}
	d := &dice.Dice{Parent: dm}
	dm.Dice[0] = d

	svc := base.NewBaseService(dm)
	resp, err := svc.Overview(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("Overview returned error: %v", err)
	}
	item := resp.Body.Item

	if item.AppName != dice.APPNAME {
		t.Fatalf("AppName = %q, want %q", item.AppName, dice.APPNAME)
	}
	if item.Version.Value != dice.VERSION.String() {
		t.Fatalf("Version.Value = %q, want %q", item.Version.Value, dice.VERSION.String())
	}
	if item.Runtime.OS != runtime.GOOS {
		t.Fatalf("Runtime.OS = %q, want %q", item.Runtime.OS, runtime.GOOS)
	}
	if !item.Runtime.JustForTest {
		t.Fatalf("Runtime.JustForTest = false, want true")
	}
	if !item.Runtime.ContainerMode {
		t.Fatalf("Runtime.ContainerMode = false, want true")
	}
	if item.Runtime.Uptime < 0 {
		t.Fatalf("Runtime.Uptime = %d, want non-negative", item.Runtime.Uptime)
	}
	if item.Memory.UsedSys > item.Memory.Sys {
		t.Fatalf("Memory.UsedSys = %d greater than Sys = %d", item.Memory.UsedSys, item.Memory.Sys)
	}
}

func TestLoginSaltReturnsManagerSalt(t *testing.T) {
	dm := &dice.DiceManager{
		UIPasswordSalt: "salt-value",
		Dice:           []*dice.Dice{{Parent: nil}},
	}
	d := &dice.Dice{Parent: dm}
	dm.Dice[0] = d

	svc := base.NewBaseService(dm)
	resp, err := svc.LoginSalt(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("LoginSalt returned error: %v", err)
	}
	if resp.Body.Item.Salt != "salt-value" {
		t.Fatalf("Salt = %q, want salt-value", resp.Body.Item.Salt)
	}
}

func TestHealthIncludesInitializedFlag(t *testing.T) {
	dm := &dice.DiceManager{JustForTest: true}
	d := &dice.Dice{Parent: dm}
	dm.Dice = []*dice.Dice{d}

	svc := base.NewBaseService(dm)
	resp, err := svc.Health(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("health returned error: %v", err)
	}
	if resp.Body.Item.Status != "ok" {
		t.Fatalf("Status = %q, want ok", resp.Body.Item.Status)
	}
	if !resp.Body.Item.TestMode {
		t.Fatalf("TestMode = false, want true")
	}
	if !resp.Body.Item.Initialized {
		t.Fatalf("Initialized = false, want true")
	}
}
