package dice //nolint:testpackage

import (
	"testing"

	randcore "sealdice-core/utils/random"
)

func TestDisableDiceRandomGMWhenSourceInitFailed(t *testing.T) {
	prev := globalRandSource
	globalRandSource = randcore.NewEmptyGlobalOwner()
	t.Cleanup(func() {
		globalRandSource = prev
	})

	d := &Dice{Config: Config{BaseConfig: BaseConfig{
		DiceRandomMode:      string(DiceRandomModePCG),
		DiceRandomGMEnabled: true,
	}}}
	if err := d.DisableDiceRandomGM(); err != nil {
		t.Fatalf("DisableDiceRandomGM() = %v", err)
	}
	if d.Config.DiceRandomGMEnabled {
		t.Fatal("expected persisted GM flag to be cleared even when runtime GM is unavailable")
	}
}

func TestMigrateDiceRandomGMConfig(t *testing.T) {
	cases := []struct {
		name         string
		mode         DiceRandomMode
		version      int
		hasGMField   bool
		initialGM    bool
		wantGM       bool
		wantMigrated bool
	}{
		{name: "old gm without field", mode: DiceRandomModeGM, version: 10300, wantGM: true, wantMigrated: true},
		{name: "old hybrid without field", mode: DiceRandomModeHybrid, version: 10300, wantGM: true, wantMigrated: true},
		{name: "old pcg without field", mode: DiceRandomModePCG, version: 10300, wantGM: false, wantMigrated: true},
		{name: "old nist without field", mode: DiceRandomModeNIST, version: 10300, wantGM: false, wantMigrated: true},
		{name: "downgraded config without field", mode: DiceRandomModeGM, version: ConfigVersionCode, wantGM: true, wantMigrated: true},
		{name: "explicit false is respected", mode: DiceRandomModeGM, version: ConfigVersionCode, hasGMField: true, wantGM: false, wantMigrated: false},
		{name: "explicit true is respected", mode: DiceRandomModeGM, version: ConfigVersionCode, hasGMField: true, initialGM: true, wantGM: true, wantMigrated: false},
		{name: "old version with field keeps false", mode: DiceRandomModeGM, version: 10300, hasGMField: true, wantGM: false, wantMigrated: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := &Dice{Config: Config{BaseConfig: BaseConfig{
				VersionCode:         tc.version,
				DiceRandomMode:      string(tc.mode),
				DiceRandomGMEnabled: tc.initialGM,
			}}}

			if got := d.migrateDiceRandomGMConfig(tc.hasGMField); got != tc.wantMigrated {
				t.Fatalf("migrateDiceRandomGMConfig() = %v, want %v", got, tc.wantMigrated)
			}
			if d.Config.DiceRandomGMEnabled != tc.wantGM {
				t.Fatalf("DiceRandomGMEnabled = %v, want %v", d.Config.DiceRandomGMEnabled, tc.wantGM)
			}
			if tc.wantMigrated && d.Config.VersionCode != ConfigVersionCode {
				t.Fatalf("VersionCode = %d, want %d", d.Config.VersionCode, ConfigVersionCode)
			}
		})
	}
}
