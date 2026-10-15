//nolint:testpackage
package censor

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
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

func hasWord(res CheckResult, word string) bool {
	for _, h := range res.Hits {
		if h.Word == word {
			return true
		}
	}
	return false
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
	if !hasWord(result, "badword") {
		t.Errorf("expected 'badword' in Hits")
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
	if len(result.Hits) != 0 {
		t.Errorf("expected no hits, got %v", result.Hits)
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
	// key is stored lowercase, and CaseSensitive keeps input case, so "BADWORD" never matches
	c.SensitiveKeys["badword"] = WordInfo{Level: Danger, Origin: "badword"}
	_ = c.Load()

	result := c.Check("BADWORD")
	if result.HighestLevel != Ignore {
		t.Errorf("expected case-sensitive miss, got %v", result.HighestLevel)
	}
}

func TestCensor_Check_CaseInsensitive(t *testing.T) {
	c := newTestCensor(map[string]Level{"badword": Danger})
	result := c.Check("contains BADWORD here")
	if result.HighestLevel != Danger {
		t.Errorf("expected case-insensitive hit, got %v", result.HighestLevel)
	}
}

func TestCensor_Check_LongestMatch(t *testing.T) {
	c := newTestCensor(map[string]Level{
		"色情":  Warning,
		"色情片": Danger,
	})
	res := c.Check("这里有色情片内容")
	if res.HighestLevel != Danger {
		t.Fatalf("want Danger, got %v (hits=%+v)", res.HighestLevel, res.Hits)
	}
	if !hasWord(res, "色情片") {
		t.Fatalf("expected hit 色情片, got %+v", res.Hits)
	}
}

func TestCensor_Check_SpanOffset(t *testing.T) {
	c := newTestCensor(map[string]Level{"bad": Danger})
	res := c.Check("xx bad yy")
	if len(res.Hits) != 1 {
		t.Fatalf("want 1 hit, got %+v", res.Hits)
	}
	h := res.Hits[0]
	if h.Span.Start != 3 || h.Span.End != 6 {
		t.Fatalf("span=%+v want {3 6}", h.Span)
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

func TestCheckAndMaskPipeline(t *testing.T) {
	c := newTestCensor(map[string]Level{"夜总会": Danger})
	res := c.Check("黑夜总会来临")
	if len(res.Hits) != 1 {
		t.Fatalf("want 1 hit, got %+v", res.Hits)
	}
	if res.Hits[0].Span != (Span{Start: 1, End: 4}) {
		t.Fatalf("span=%+v want {1 4}", res.Hits[0].Span)
	}
	masked := MaskSpans("黑夜总会来临", []Span{res.Hits[0].Span}, "■")
	if masked != "黑■■■来临" {
		t.Fatalf("masked=%q", masked)
	}
}

func TestCensor_Check_FullWidthDictionaryKey(t *testing.T) {
	// 词表键为全角形式，正文为半角：键归一化后应可匹配
	c := &Censor{SensitiveKeys: map[string]WordInfo{
		"ｂａｄ": {Level: Danger, Origin: "ｂａｄ"},
	}}
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	if res := c.Check("very bad word"); res.HighestLevel != Danger {
		t.Fatalf("want Danger, got %v (hits=%+v)", res.HighestLevel, res.Hits)
	}
}

func TestCensor_Check_CombiningFormMatches(t *testing.T) {
	// 词表预组合 é，正文分解形式 e+◌́ 也应命中，且 span 覆盖两个原文 rune
	c := &Censor{SensitiveKeys: map[string]WordInfo{
		"café": {Level: Danger, Origin: "café"},
	}}
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	res := c.Check("x cafe\u0301 y")
	if res.HighestLevel != Danger || len(res.Hits) != 1 {
		t.Fatalf("want 1 Danger hit, got %+v", res.Hits)
	}
	if res.Hits[0].Span.Start != 2 || res.Hits[0].Span.End != 7 {
		t.Fatalf("span=%+v want {2 7}", res.Hits[0].Span)
	}
}

func TestCensor_addWord_CaseSensitive_SetsOrigin(t *testing.T) {
	c := &Censor{CaseSensitive: true, SensitiveKeys: map[string]WordInfo{}}
	var counter FileCounter
	c.addWord("BadWord", Danger, &counter)
	info, ok := c.SensitiveKeys["BadWord"]
	if !ok || info.Origin != "BadWord" {
		t.Fatalf("info=%+v ok=%v want Origin=BadWord", info, ok)
	}
}

func TestCensor_LoadWords_InvalidRegexKeepsLastGood(t *testing.T) {
	c := newTestCensor(map[string]Level{"bad": Danger})
	c.FilterRegexStr = "("
	if err := c.Load(); err == nil {
		t.Fatal("want error for invalid regex")
	}
	if c.Check("a bad b").HighestLevel != Danger {
		t.Fatal("old matcher should still hit after failed reload")
	}

	fresh := &Censor{SensitiveKeys: map[string]WordInfo{"bad": {Level: Danger, Origin: "bad"}}, FilterRegexStr: "("}
	if err := fresh.Load(); err == nil {
		t.Fatal("want error for invalid regex")
	}
	if fresh.Check("a bad b").HighestLevel != Ignore {
		t.Fatal("never-loaded censor should report no hits")
	}
}

func TestCensor_ConcurrentCheckAndReload(t *testing.T) {
	words := map[string]WordInfo{"bad": {Level: Danger, Origin: "bad"}}
	c := &Censor{}
	if err := c.LoadWords(words); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 300 {
				if c.Check("a bad b").HighestLevel != Danger {
					t.Error("reload lost matches")
					return
				}
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 100 {
			if err := c.LoadWords(words); err != nil {
				t.Error(err)
				return
			}
		}
	}()
	wg.Wait()
}

func TestCensor_PreloadFile_ZeroValueCensor(t *testing.T) {
	dir := t.TempDir()

	tomlPath := filepath.Join(dir, "w.toml")
	if err := os.WriteFile(tomlPath, []byte("[meta]\nname = \"t\"\n[words]\ndanger = [\"坏词\"]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := &Censor{} // SensitiveKeys 为 nil，addWord 必须自愈，不得 panic
	if _, err := c.PreloadFile(tomlPath); err != nil {
		t.Fatal(err)
	}
	if err := c.LoadWords(c.SensitiveKeys); err != nil {
		t.Fatal(err)
	}
	if c.Check("这里有坏词").HighestLevel != Danger {
		t.Fatal("expected Danger hit from toml word file")
	}

	txtPath := filepath.Join(dir, "w.txt")
	if err := os.WriteFile(txtPath, []byte("#danger\n坏词\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c2 := &Censor{}
	if _, err := c2.PreloadFile(txtPath); err != nil {
		t.Fatal(err)
	}
	if err := c2.LoadWords(c2.SensitiveKeys); err != nil {
		t.Fatal(err)
	}
	if c2.Check("坏词在此").HighestLevel != Danger {
		t.Fatal("expected Danger hit from txt word file")
	}
}
