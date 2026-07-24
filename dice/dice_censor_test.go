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
		"检测到<警告>级敏感词。\n命中词(Base64): %s | %s\n上下文(Base64): %s",
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
