package dice //nolint:testpackage // Tests the unexported message formatter directly.

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
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
