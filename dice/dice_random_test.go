package dice //nolint:testpackage

import "testing"

func TestMigrateDiceRandomGMConfig(t *testing.T) {
	cases := []struct {
		name     string
		mode     DiceRandomMode
		version  int
		wantGM   bool
		wantDone bool
	}{
		{name: "old gm", mode: DiceRandomModeGM, version: 10300, wantGM: true, wantDone: true},
		{name: "old hybrid", mode: DiceRandomModeHybrid, version: 10300, wantGM: true, wantDone: true},
		{name: "old pcg", mode: DiceRandomModePCG, version: 10300, wantGM: false, wantDone: true},
		{name: "old nist", mode: DiceRandomModeNIST, version: 10300, wantGM: false, wantDone: true},
		{name: "current version", mode: DiceRandomModeHybrid, version: ConfigVersionCode, wantGM: false, wantDone: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := &Dice{Config: Config{BaseConfig: BaseConfig{
				VersionCode:    tc.version,
				DiceRandomMode: string(tc.mode),
			}}}

			if got := d.migrateDiceRandomGMConfig(); got != tc.wantDone {
				t.Fatalf("migrateDiceRandomGMConfig() = %v, want %v", got, tc.wantDone)
			}
			if d.Config.DiceRandomGMEnabled != tc.wantGM {
				t.Fatalf("DiceRandomGMEnabled = %v, want %v", d.Config.DiceRandomGMEnabled, tc.wantGM)
			}
			if tc.wantDone && d.Config.VersionCode != ConfigVersionCode {
				t.Fatalf("VersionCode = %d, want %d", d.Config.VersionCode, ConfigVersionCode)
			}
		})
	}
}
