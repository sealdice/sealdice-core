package dice

import (
	"strings"
	"testing"
)

func TestLimitCommandReasonText(t *testing.T) {
	short := "普通原因"
	if got := limitCommandReasonText(short); got != short {
		t.Fatalf("expected short text to remain unchanged, got %q", got)
	}

	suffixLen := len([]rune(commandReasonOmitSuffix))
	assertTruncate := func(t *testing.T, input string, expectedPrefix string) {
		t.Helper()
		got := limitCommandReasonText(input)
		if !strings.HasSuffix(got, commandReasonOmitSuffix) {
			t.Fatalf("expected truncated text to include suffix, got %q", got)
		}
		gotRunes := []rune(got)
		expectedLen := commandReasonMaxLen + suffixLen
		if len(gotRunes) != expectedLen {
			t.Fatalf("expected truncated text length %d, got %d", expectedLen, len(gotRunes))
		}
		if string(gotRunes[:commandReasonMaxLen]) != expectedPrefix {
			t.Fatalf("expected truncated text to keep prefix, got %q", got)
		}
	}

	assertTruncate(t, strings.Repeat("测", commandReasonMaxLen+1), strings.Repeat("测", commandReasonMaxLen))
	assertTruncate(t, strings.Repeat("a", commandReasonMaxLen+1), strings.Repeat("a", commandReasonMaxLen))
}
