//nolint:testpackage
package dice

import (
	"strings"
	"testing"

	"sealdice-core/message"
)

func TestNormalizeIncomingMessageStringToSegment(t *testing.T) {
	msg := &Message{Platform: "QQ", Message: "前缀[CQ:at,qq=100]后缀"}
	NormalizeIncomingMessage(msg)

	if len(msg.Segment) != 3 {
		t.Fatalf("segment count = %d, want 3", len(msg.Segment))
	}
	at, ok := msg.Segment[1].(*message.AtElement)
	if !ok || at.Target != "100" {
		t.Fatalf("segment 1 = %#v, want at 100", msg.Segment[1])
	}
	if !msg.segmentSynthesized {
		t.Fatal("expected synthesized marker for string-only message")
	}
	if msg.Message != "前缀[CQ:at,qq=100]后缀" {
		t.Fatalf("Message was overwritten: %q", msg.Message)
	}
}

func TestNormalizeIncomingMessageSegmentToMessage(t *testing.T) {
	msg := &Message{Segment: []message.IMessageElement{
		&message.TextElement{Content: "前缀"},
		&message.AtElement{Target: "100"},
		&message.TextElement{Content: "后缀"},
	}}
	NormalizeIncomingMessage(msg)

	if msg.Message != "前缀[CQ:at,qq=100]后缀" {
		t.Fatalf("Message = %q", msg.Message)
	}
	if msg.segmentSynthesized {
		t.Fatal("native segment message must not be marked synthesized")
	}
}

func TestNormalizeIncomingMessageIsIdempotent(t *testing.T) {
	msg := &Message{Message: "hello"}
	NormalizeIncomingMessage(msg)
	first := msg.Segment
	NormalizeIncomingMessage(msg)

	if len(msg.Segment) != len(first) {
		t.Fatalf("segment count changed after second normalize: %d -> %d", len(first), len(msg.Segment))
	}
	if msg.Message != "hello" {
		t.Fatalf("Message changed: %q", msg.Message)
	}
}

func TestMessageSegmentsToCompatibilityText(t *testing.T) {
	text := MessageSegmentsToCompatibilityText([]message.IMessageElement{
		&message.TextElement{Content: "图片:"},
		&message.ImageElement{URL: "https://example.invalid/a.png"},
	})
	if !strings.Contains(text, "图片:") {
		t.Fatalf("text missing plain segment: %q", text)
	}
	if !strings.Contains(text, "[CQ:image") {
		t.Fatalf("text missing image compatibility code: %q", text)
	}
}

// 文件段渲染保持旧 extractResultFromSegments 的优先级：File 优先于 URL。
func TestMessageSegmentsToCompatibilityTextFilePrefersFile(t *testing.T) {
	text := MessageSegmentsToCompatibilityText([]message.IMessageElement{
		&message.FileElement{File: "data/help.pdf", URL: "https://cdn.example.invalid/help.pdf"},
	})
	if !strings.Contains(text, "[CQ:file,file=data/help.pdf]") {
		t.Fatalf("file compat text should prefer File over URL: %q", text)
	}
}

// 非 CQ 平台的字符串输入保持为单个文本段，字面 CQ 码不参与解析。
func TestNormalizeIncomingMessageNonCQPlatformKeepsText(t *testing.T) {
	msg := &Message{Platform: "DISCORD", Message: "look [CQ:at,qq=1000] plain text"}
	NormalizeIncomingMessage(msg)

	if len(msg.Segment) != 1 {
		t.Fatalf("segment count = %d, want 1", len(msg.Segment))
	}
	if _, ok := msg.Segment[0].(*message.TextElement); !ok {
		t.Fatalf("segment type = %T, want *TextElement", msg.Segment[0])
	}
	if !msg.segmentSynthesized {
		t.Fatal("expected synthesized marker")
	}
}

func TestRedactURLQueries(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{
			in:   "[CQ:image,file=https://multimedia.example.invalid/download?appid=1&file_id=x&sign=SECRET]",
			want: "[CQ:image,file=https://multimedia.example.invalid/download?...]",
		},
		{
			in:   "看这个 https://cdn.example.invalid/a.png?token=abc 和 [CQ:file,file=data/x.pdf]",
			want: "看这个 https://cdn.example.invalid/a.png?... 和 [CQ:file,file=data/x.pdf]",
		},
		{
			in:   "没有查询串的 https://example.invalid/a.png 保持不变",
			want: "没有查询串的 https://example.invalid/a.png 保持不变",
		},
	}
	for _, tc := range cases {
		if got := redactURLQueries(tc.in); got != tc.want {
			t.Errorf("redactURLQueries(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
