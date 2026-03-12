package dice_test

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"sealdice-core/dice"
)

// --- Smoke tests for LimitCommandReasonText ---

func TestLimitCommandReasonText_ExactBoundary(t *testing.T) {
	// Text at exactly the max length should not be truncated
	text := strings.Repeat("a", dice.CommandReasonMaxLen)
	if got := dice.LimitCommandReasonText(text); got != text {
		t.Errorf("text at max length should not be truncated")
	}
}

func TestLimitCommandReasonText_Empty(t *testing.T) {
	if got := dice.LimitCommandReasonText(""); got != "" {
		t.Errorf("empty string should stay empty, got %q", got)
	}
}

func TestLimitCommandReasonText_Unicode(t *testing.T) {
	// Chinese chars are counted as single runes
	text := strings.Repeat("测", dice.CommandReasonMaxLen+1)
	got := dice.LimitCommandReasonText(text)
	gotRunes := []rune(got)
	wantLen := dice.CommandReasonMaxLen + utf8.RuneCountInString(dice.CommandReasonOmitSuffix)
	if len(gotRunes) != wantLen {
		t.Errorf("truncated length = %d, want %d", len(gotRunes), wantLen)
	}
	if !strings.HasSuffix(got, dice.CommandReasonOmitSuffix) {
		t.Errorf("truncated text should end with omit suffix")
	}
}

// --- Smoke tests for RandStringBytesMaskImprSrcSB ---

func TestRandString_Length(t *testing.T) {
	for _, n := range []int{0, 1, 8, 16, 32, 64} {
		got := dice.RandStringBytesMaskImprSrcSB(n)
		if len(got) != n {
			t.Errorf("RandStringBytesMaskImprSrcSB(%d) len = %d, want %d", n, len(got), n)
		}
	}
}

func TestRandString_CharSet(t *testing.T) {
	const allowed = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ123456789"
	got := dice.RandStringBytesMaskImprSrcSB(200)
	for _, c := range got {
		if !strings.ContainsRune(allowed, c) {
			t.Errorf("RandStringBytesMaskImprSrcSB produced disallowed char %q", c)
		}
	}
}

func TestRandString2_Length(t *testing.T) {
	for _, n := range []int{1, 16, 32} {
		got := dice.RandStringBytesMaskImprSrcSB2(n)
		if len(got) != n {
			t.Errorf("RandStringBytesMaskImprSrcSB2(%d) len = %d, want %d", n, len(got), n)
		}
	}
}

// --- Smoke tests for DeckToShuffleRandomPool ---

func TestDeckToShuffleRandomPool_SingleItem(t *testing.T) {
	deck := []string{"only"}
	pool := dice.DeckToShuffleRandomPool(deck)
	got := pool.Pick().(string)
	if got != "only" {
		t.Errorf("single-item deck Pick() = %q, want %q", got, "only")
	}
}

func TestDeckToShuffleRandomPool_WeightedItems(t *testing.T) {
	// "::10::A" weight 10, "::90::B" weight 90 → B should dominate
	var deck []string
	_ = json.Unmarshal([]byte(`["::10::A","::90::B"]`), &deck)

	const iters = 1000
	counts := map[string]int{}
	for range iters {
		pool := dice.DeckToShuffleRandomPool(deck)
		counts[pool.Pick().(string)]++
	}

	// B should be picked ~90% of the time; allow wide margin for randomness
	if counts["B"] < 700 {
		t.Errorf("expected B to be dominant (got %d/1000), weights may not be applied correctly", counts["B"])
	}
}

func TestDeckToShuffleRandomPool_AllItemsEventuallyDrawn(t *testing.T) {
	var deck []string
	_ = json.Unmarshal([]byte(`["1","2","3","4","5"]`), &deck)

	// Run multiple pools and confirm all items are reachable
	seen := map[string]bool{}
	for range 200 {
		pool := dice.DeckToShuffleRandomPool(deck)
		seen[pool.Pick().(string)] = true
	}
	for _, item := range deck {
		if !seen[item] {
			t.Errorf("item %q was never drawn in 200 trials", item)
		}
	}
}

func TestDeckToShuffleRandomPool_NoRepeatUntilExhausted(t *testing.T) {
	var deck []string
	_ = json.Unmarshal([]byte(`["a","b","c","d","e"]`), &deck)
	pool := dice.DeckToShuffleRandomPool(deck)

	seen := map[string]bool{}
	for range deck {
		item := pool.Pick().(string)
		if seen[item] {
			t.Errorf("item %q was drawn twice before deck was exhausted", item)
		}
		seen[item] = true
	}
}

// --- Benchmark tests ---

func BenchmarkLimitCommandReasonText_Short(b *testing.B) {
	text := strings.Repeat("x", 50)
	b.ResetTimer()
	for range b.N {
		_ = dice.LimitCommandReasonText(text)
	}
}

func BenchmarkLimitCommandReasonText_Long(b *testing.B) {
	text := strings.Repeat("测试文字", dice.CommandReasonMaxLen)
	b.ResetTimer()
	for range b.N {
		_ = dice.LimitCommandReasonText(text)
	}
}

func BenchmarkRandString_16(b *testing.B) {
	b.ResetTimer()
	for range b.N {
		_ = dice.RandStringBytesMaskImprSrcSB(16)
	}
}

func BenchmarkRandString_64(b *testing.B) {
	b.ResetTimer()
	for range b.N {
		_ = dice.RandStringBytesMaskImprSrcSB(64)
	}
}

func BenchmarkDeckToShuffleRandomPool(b *testing.B) {
	var deck []string
	_ = json.Unmarshal(
		[]byte(`["::10::1","::10::2","::10::3","::10::4","::10::5","::10::6","::40::7"]`),
		&deck,
	)
	b.ResetTimer()
	for range b.N {
		_ = dice.DeckToShuffleRandomPool(deck)
	}
}

func BenchmarkShuffleRandomPool_Pick(b *testing.B) {
	var deck []string
	_ = json.Unmarshal(
		[]byte(`["::10::1","::10::2","::10::3","::10::4","::10::5","::10::6","::40::7"]`),
		&deck,
	)
	b.ResetTimer()
	for range b.N {
		pool := dice.DeckToShuffleRandomPool(deck)
		_ = pool.Pick()
	}
}
