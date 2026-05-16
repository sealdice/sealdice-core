//nolint:testpackage
package censor

import (
	"strings"
	"testing"
)

// --- Smoke tests for HigherLevel ---

func TestHigherLevel(t *testing.T) {
	tests := []struct {
		l1, l2 Level
		want   Level
	}{
		{Ignore, Ignore, Ignore},
		{Ignore, Danger, Danger},
		{Danger, Ignore, Danger},
		{Warning, Caution, Warning},
		{Notice, Warning, Warning},
		{Danger, Danger, Danger},
	}
	for _, tt := range tests {
		got := HigherLevel(tt.l1, tt.l2)
		if got != tt.want {
			t.Errorf("HigherLevel(%v, %v) = %v, want %v", tt.l1, tt.l2, got, tt.want)
		}
	}
}

// --- Smoke tests for Censor ---

func newTestCensor(words map[string]Level) *Censor {
	c := &Censor{
		SensitiveKeys: make(map[string]WordInfo),
	}
	for word, level := range words {
		c.SensitiveKeys[strings.ToLower(word)] = WordInfo{Level: level, Origin: word}
	}
	_ = c.Load()
	return c
}

func TestCensor_Check_Hit(t *testing.T) {
	c := newTestCensor(map[string]Level{
		"badword": Danger,
		"spam":    Warning,
	})

	result := c.Check("this contains badword in it")
	if result.HighestLevel != Danger {
		t.Errorf("expected Danger, got %v", result.HighestLevel)
	}
	if _, ok := result.SensitiveWords["badword"]; !ok {
		t.Errorf("expected 'badword' in SensitiveWords")
	}
}

func TestCensor_Check_Miss(t *testing.T) {
	c := newTestCensor(map[string]Level{
		"badword": Danger,
	})

	result := c.Check("this is a perfectly clean message")
	if result.HighestLevel != Ignore {
		t.Errorf("expected Ignore, got %v", result.HighestLevel)
	}
	if len(result.SensitiveWords) != 0 {
		t.Errorf("expected no sensitive words, got %v", result.SensitiveWords)
	}
}

func TestCensor_Check_MultipleWords(t *testing.T) {
	c := newTestCensor(map[string]Level{
		"apple": Notice,
		"bomb":  Danger,
		"gun":   Warning,
	})

	result := c.Check("apple gun bomb")
	if result.HighestLevel != Danger {
		t.Errorf("expected Danger, got %v", result.HighestLevel)
	}
}

func TestCensor_Check_EmptyText(t *testing.T) {
	c := newTestCensor(map[string]Level{
		"badword": Danger,
	})

	result := c.Check("")
	if result.HighestLevel != Ignore {
		t.Errorf("expected Ignore for empty text, got %v", result.HighestLevel)
	}
}

func TestCensor_Check_FilterRegex(t *testing.T) {
	c := &Censor{
		SensitiveKeys:  make(map[string]WordInfo),
		FilterRegexStr: `\s+`, // strip all whitespace before matching
	}
	c.SensitiveKeys["badword"] = WordInfo{Level: Danger, Origin: "badword"}
	_ = c.Load()

	// Whitespace between letters should be stripped, so "bad word" → "badword"
	result := c.Check("bad word")
	if result.HighestLevel != Danger {
		t.Errorf("expected Danger after regex filter, got %v", result.HighestLevel)
	}
}

func TestCensor_Check_CaseSensitive_Miss(t *testing.T) {
	c := &Censor{
		CaseSensitive: true,
		SensitiveKeys: make(map[string]WordInfo),
	}
	// addWord lowercases, so uppercase version is not matched when CaseSensitive=true
	// (the key stored is lowercase, so "BADWORD" never appears in trie)
	c.SensitiveKeys["badword"] = WordInfo{Level: Danger, Origin: "badword"}
	_ = c.Load()

	result := c.Check("BADWORD")
	if result.HighestLevel != Ignore {
		t.Errorf("expected case-sensitive miss, got %v", result.HighestLevel)
	}
}

func TestCensor_Load_EmptySensitiveKeys(t *testing.T) {
	c := &Censor{SensitiveKeys: make(map[string]WordInfo)}
	if err := c.Load(); err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	result := c.Check("anything")
	if result.HighestLevel != Ignore {
		t.Errorf("expected Ignore when no keys loaded, got %v", result.HighestLevel)
	}
}

func TestCensor_addWord_CaseInsensitive(t *testing.T) {
	c := &Censor{
		CaseSensitive: false,
		SensitiveKeys: make(map[string]WordInfo),
	}
	var counter FileCounter
	c.addWord("BadWord", Danger, &counter)

	if counter[Danger] != 1 {
		t.Errorf("expected counter[Danger]=1, got %d", counter[Danger])
	}
	if _, ok := c.SensitiveKeys["badword"]; !ok {
		t.Errorf("expected lowercase key 'badword' in SensitiveKeys")
	}
}

// --- Benchmark tests ---

func BenchmarkTrie_Insert(b *testing.B) {
	words := []string{
		"alpha", "beta", "gamma", "delta", "epsilon",
		"zeta", "eta", "theta", "iota", "kappa",
	}
	b.ResetTimer()
	for range b.N {
		t := newTire()
		for j, w := range words {
			t.Insert(w, Level(j%5))
		}
	}
}

func BenchmarkTrie_Match_Hit(b *testing.B) {
	t := newTire()
	words := []string{
		"sensitive", "badword", "forbidden", "blocked", "illegal",
		"spam", "abuse", "toxic", "harmful", "dangerous",
	}
	for i, w := range words {
		t.Insert(w, Level(i%5))
	}
	text := "this message contains a sensitive word that is forbidden and harmful in nature"
	b.ResetTimer()
	for range b.N {
		_ = t.Match(text)
	}
}

func BenchmarkTrie_Match_Miss(b *testing.B) {
	t := newTire()
	for i := range 20 {
		t.Insert(strings.Repeat("x", i+3), Danger)
	}
	text := "a completely clean message without any matching keywords at all in this text"
	b.ResetTimer()
	for range b.N {
		_ = t.Match(text)
	}
}

func BenchmarkCensor_Check(b *testing.B) {
	c := &Censor{SensitiveKeys: make(map[string]WordInfo)}
	keywords := []string{
		"alpha", "beta", "gamma", "delta", "epsilon",
		"zeta", "eta", "theta", "iota", "kappa",
		"lambda", "mu", "nu", "xi", "omicron",
		"pi", "rho", "sigma", "tau", "upsilon",
	}
	for i, w := range keywords {
		c.SensitiveKeys[w] = WordInfo{Level: Level(i % 5), Origin: w}
	}
	_ = c.Load()

	texts := []string{
		"a normal message with no bad words",
		"this message contains alpha and beta keywords",
		strings.Repeat("clean text with no issues. ", 20),
		"sigma and tau and pi are all found here",
	}

	b.ResetTimer()
	for i := range b.N {
		_ = c.Check(texts[i%len(texts)])
	}
}

func BenchmarkCensor_Check_LargeWordlist(b *testing.B) {
	c := &Censor{SensitiveKeys: make(map[string]WordInfo)}
	for i := range 500 {
		word := strings.Repeat(string(rune('a'+i%26)), (i%8)+3)
		c.SensitiveKeys[word] = WordInfo{Level: Danger, Origin: word}
	}
	_ = c.Load()

	text := "this is a representative message of average length sent by a typical user in chat"
	b.ResetTimer()
	for range b.N {
		_ = c.Check(text)
	}
}
