package dice_test

import (
	"testing"

	"sealdice-core/dice"
)

func TestNameGenerateNegativeWeight(t *testing.T) {
	ng := &dice.NamesGenerator{NamesInfo: map[string]map[string][]string{
		"test": {
			"name": {"alpha", "beta"},
			// 两个负数回绕后的权重之和会溢出
			"weight": {"-5", "-5"},
		},
	}}

	for range 100 {
		if got := ng.NameGenerate("{test:name@test:weight}"); got != "alpha" && got != "beta" {
			t.Fatalf("NameGenerate returned %q, want one of the names", got)
		}
	}
}
