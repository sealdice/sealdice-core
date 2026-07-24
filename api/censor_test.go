package api //nolint:testpackage // Tests exercise the unexported handler config conversion.

import (
	"slices"
	"testing"

	"sealdice-core/dice"
	"sealdice-core/dice/censor"
)

func TestCensorEncodedDetailsHandlerConfigRoundTrip(t *testing.T) {
	previousDice := myDice
	t.Cleanup(func() {
		myDice = previousDice
	})

	myDice = &dice.Dice{}
	myDice.Config.CensorHandlers = map[censor.Level]uint8{}
	setLevelHandlers(censor.Warning, []string{"SendWarning", "SendEncodedDetails"})

	got := getLevelConfig(
		censor.Warning,
		map[censor.Level]int{
			censor.Ignore:  0,
			censor.Notice:  0,
			censor.Caution: 0,
			censor.Warning: 1,
			censor.Danger:  0,
		},
		myDice.Config.CensorHandlers,
		map[censor.Level]int{
			censor.Ignore:  0,
			censor.Notice:  0,
			censor.Caution: 0,
			censor.Warning: 100,
			censor.Danger:  0,
		},
	)

	if !slices.Contains(got.Handlers, "SendEncodedDetails") {
		t.Fatalf("encoded details handler missing after config round trip: %v", got.Handlers)
	}
}
