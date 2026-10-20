//nolint:testpackage
package dice

import (
	"strings"
	"testing"

	"sealdice-core/message"
)

func TestNormalizeIncomingMessageStringToSegment(t *testing.T) {
	msg := &Message{Message: "前缀[CQ:at,qq=100]后缀"}
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
