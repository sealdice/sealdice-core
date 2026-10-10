package dice //nolint:testpackage // Tests the unexported message formatter directly.

import (
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"testing"

	wr "github.com/mroth/weightedrand/v3"
	"go.uber.org/zap"

	"sealdice-core/dice/censor"
)

func TestFormatCensorHitDetailsEncodesWordsAndContext(t *testing.T) {
	words := []string{"敏感词一", "敏感词二"}
	content := "前文 敏感词一 后文"

	got := formatCensorHitDetails("警告", words, content)
	want := fmt.Sprintf(
		"检测到<警告>级敏感词。\n命中词(Base64): %s | %s\n上下文片段(Base64): %s",
		base64.StdEncoding.EncodeToString([]byte(words[0])),
		base64.StdEncoding.EncodeToString([]byte(words[1])),
		base64.StdEncoding.EncodeToString([]byte(content)),
	)

	if got != want {
		t.Fatalf("unexpected encoded details:\n got: %q\nwant: %q", got, want)
	}
	for _, raw := range append(words, content) {
		if strings.Contains(got, raw) {
			t.Fatalf("encoded details leaked raw content %q", raw)
		}
	}
}

func TestFormatCensorHitDetailsLimitsContextAroundHit(t *testing.T) {
	word := "敏感词"
	content := strings.Repeat("前", maxCensorHitContextRunes) + word + strings.Repeat("后", maxCensorHitContextRunes)

	got := formatCensorHitDetails("警告", []string{word}, content)
	const contextPrefix = "\n上下文片段(Base64): "
	contextAt := strings.LastIndex(got, contextPrefix)
	if contextAt < 0 {
		t.Fatalf("encoded context missing from details: %q", got)
	}

	encodedContext := got[contextAt+len(contextPrefix):]
	decodedContext, err := base64.StdEncoding.DecodeString(encodedContext)
	if err != nil {
		t.Fatalf("decode context: %v", err)
	}
	context := string(decodedContext)
	if !strings.Contains(context, word) {
		t.Fatalf("limited context does not include hit word: %q", context)
	}
	if !strings.HasPrefix(context, "...") || !strings.HasSuffix(context, "...") {
		t.Fatalf("limited context does not mark omitted text: %q", context)
	}
	if gotRunes, wantMax := len([]rune(context)), maxCensorHitContextRunes+6; gotRunes > wantMax {
		t.Fatalf("limited context has %d runes, want at most %d", gotRunes, wantMax)
	}
	if context == content {
		t.Fatal("long context was returned in full")
	}
}

func TestCensorHitContextOmitsLongContentWithoutDirectHit(t *testing.T) {
	content := strings.Repeat("内容", maxCensorHitContextRunes)

	if got := censorHitContext(content, []string{"not-present"}); got != "..." {
		t.Fatalf("context without a direct hit = %q, want omission marker", got)
	}
}

func TestCensorMaskContent_UsesPlaceholderTemplate(t *testing.T) {
	chooser, err := wr.NewChooser(wr.NewChoice("■", uint(1)))
	if err != nil {
		t.Fatal(err)
	}
	d := &Dice{Logger: zap.NewNop().Sugar()}
	d.TextMap = map[string]*wr.Chooser[string, uint]{
		"核心:拦截_敏感词过滤_替换占位符": chooser,
	}
	mctx := &MsgContext{Dice: d}

	got := censorMaskContent(mctx, "黑夜总会来临", []censor.Span{{Start: 1, End: 4}})
	if got != "黑■■■来临" {
		t.Fatalf("masked=%q want 黑■■■来临", got)
	}
}

func TestDice_CensorManagerAtomicSwap(t *testing.T) {
	d := &Dice{Logger: zap.NewNop().Sugar()}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 500 {
			d.SetCensorManager(&CensorManager{})
		}
		d.SetCensorManager(nil)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 500 {
			_ = d.CensorManager()
		}
	}()
	wg.Wait()
}

func TestCensorManager_WordFilesSnapshotConcurrent(t *testing.T) {
	cm := &CensorManager{wordFiles: map[string]*censor.WordFile{"a": {Key: "a"}}}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for range 300 {
			_ = cm.WordFiles()
		}
	}()
	go func() {
		defer wg.Done()
		for range 300 {
			cm.DeleteCensorWordFiles([]string{"missing"})
		}
	}()
	wg.Wait()
}
