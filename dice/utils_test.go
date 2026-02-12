package dice_test

import (
	"strings"
	"testing"

	"sealdice-core/dice"
)

func TestLimitCommandReasonText(t *testing.T) {
	short := "普通原因"
	if got := dice.LimitCommandReasonText(short); got != short {
		t.Fatalf("expected short text to remain unchanged, got %q", got)
	}

	suffixLen := len([]rune(dice.CommandReasonOmitSuffix))
	assertTruncate := func(t *testing.T, input string, expectedPrefix string) {
		t.Helper()
		got := dice.LimitCommandReasonText(input)
		if !strings.HasSuffix(got, dice.CommandReasonOmitSuffix) {
			t.Fatalf("expected truncated text to include suffix, got %q", got)
		}
		gotRunes := []rune(got)
		expectedLen := dice.CommandReasonMaxLen + suffixLen
		if len(gotRunes) != expectedLen {
			t.Fatalf("expected truncated text length %d, got %d", expectedLen, len(gotRunes))
		}
		if string(gotRunes[:dice.CommandReasonMaxLen]) != expectedPrefix {
			t.Fatalf("expected truncated text to keep prefix, got %q", got)
		}
	}

	assertTruncate(t, strings.Repeat("测", dice.CommandReasonMaxLen+1), strings.Repeat("测", dice.CommandReasonMaxLen))
	assertTruncate(t, strings.Repeat("a", dice.CommandReasonMaxLen+1), strings.Repeat("a", dice.CommandReasonMaxLen))
}
