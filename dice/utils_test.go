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

	overlong := strings.Repeat("测", commandReasonMaxLen+1)
	got := limitCommandReasonText(overlong)
	if !strings.HasSuffix(got, commandReasonOmitSuffix) {
		t.Fatalf("expected truncated text to include suffix, got %q", got)
	}
	if !strings.HasPrefix(got, strings.Repeat("测", commandReasonMaxLen)) {
		t.Fatalf("expected truncated text to keep prefix, got %q", got)
	}
}
