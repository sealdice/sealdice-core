package dice //nolint:testpackage

import "testing"

func TestRollExpressionFateUsesContextRandomSource(t *testing.T) {
	source := &countingDiceSource{values: []uint64{0, 1, 2, 0}}
	ctx := &MsgContext{Dice: &Dice{}, diceRandSrc: source}
	if _, _, err := ctx.Dice._ExprEvalBaseV1("F", ctx, RollExtraFlags{}); err != nil {
		t.Fatalf("_ExprEvalBaseV1 returned error: %v", err)
	}
	if source.index != 4 {
		t.Fatalf("Fate roll consumed %d random values, want 4", source.index)
	}
}
