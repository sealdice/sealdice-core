package message_test

import (
	"strings"
	"testing"

	"sealdice-core/message"
)

func TestSegmentProjectionPreservesNonTextSegments(t *testing.T) {
	image := &message.ImageElement{URL: "https://example.invalid/image.png"}
	segments := []message.IMessageElement{
		&message.TextElement{Content: ".foo before "},
		image,
		&message.TextElement{Content: " after"},
	}

	projection := message.ProjectSegmentsToText(segments)

	if projection.Text == "" {
		t.Fatal("projection text should not be empty")
	}
	if projection.Text == ".foo before  after" {
		t.Fatal("projection should include a placeholder for non-text segments")
	}
	if len(projection.Placeholders) != 1 {
		t.Fatalf("placeholder count = %d, want 1", len(projection.Placeholders))
	}

	rebuilt := projection.ToSegments()
	if len(rebuilt) != 3 {
		t.Fatalf("rebuilt segment count = %d, want 3", len(rebuilt))
	}
	if rebuilt[1] != image {
		t.Fatalf("non-text segment was not preserved: %#v", rebuilt[1])
	}
}

func TestSegmentProjectionResolvePlaceholders(t *testing.T) {
	image := &message.ImageElement{URL: "https://example.invalid/image.png"}
	projection := message.ProjectSegmentsToText([]message.IMessageElement{
		&message.TextElement{Content: "before "},
		image,
		&message.TextElement{Content: " after"},
	})

	resolved := projection.ResolvePlaceholders(projection.Text, func(elem message.IMessageElement) string {
		if elem == image {
			return "[img]"
		}
		return "[unknown]"
	})

	if strings.ContainsRune(resolved, '\x1f') {
		t.Fatalf("resolved text leaked placeholder: %q", resolved)
	}
	if resolved != "before [img] after" {
		t.Fatalf("resolved text = %q, want %q", resolved, "before [img] after")
	}
}

// 字面占位符文本与真实占位符 ID 冲突时，必须按字面文本保留，不能被替换成对应的消息段。
func TestSegmentProjectionKeepsLiteralPlaceholderWhenIdCollides(t *testing.T) {
	image := &message.ImageElement{URL: "https://example.invalid/image.png"}
	// image 位于 idx=1，投影占位符 ID 为 2；字面文本刻意引用 seg:2。
	segments := []message.IMessageElement{
		&message.TextElement{Content: "x \x1fseg:2\x1f y "},
		image,
	}

	projection := message.ProjectSegmentsToText(segments)
	if len(projection.Placeholders) != 1 {
		t.Fatalf("placeholder count = %d, want 1", len(projection.Placeholders))
	}

	rebuilt := projection.ToSegments()
	if len(rebuilt) != 2 {
		t.Fatalf("rebuilt segment count = %d, want 2", len(rebuilt))
	}
	text, ok := rebuilt[0].(*message.TextElement)
	if !ok {
		t.Fatalf("rebuilt segment type = %T, want *TextElement", rebuilt[0])
	}
	if text.Content != "x \x1fseg:2\x1f y " {
		t.Fatalf("literal placeholder text was consumed: %q", text.Content)
	}
	if rebuilt[1] != image {
		t.Fatalf("non-text segment was not preserved: %#v", rebuilt[1])
	}

	resolved := projection.ResolvePlaceholders(projection.Text, func(elem message.IMessageElement) string {
		return "[resolved]"
	})
	// 字面 seg:2 保留，真实占位符（结尾那个）被解析。
	if !strings.Contains(resolved, "x \x1fseg:2\x1f y ") {
		t.Fatalf("literal placeholder was replaced during resolve: %q", resolved)
	}
	if !strings.Contains(resolved, "[resolved]") {
		t.Fatalf("real placeholder was not resolved: %q", resolved)
	}
}

func TestSegmentProjectionTreatsUserPlaceholderTextAsText(t *testing.T) {
	segments := []message.IMessageElement{
		&message.TextElement{Content: ".foo \x1fseg:1\x1f"},
	}

	projection := message.ProjectSegmentsToText(segments)
	rebuilt := projection.ToSegments()

	if len(rebuilt) != 1 {
		t.Fatalf("rebuilt segment count = %d, want 1", len(rebuilt))
	}
	text, ok := rebuilt[0].(*message.TextElement)
	if !ok {
		t.Fatalf("rebuilt segment type = %T, want *TextElement", rebuilt[0])
	}
	if text.Content != ".foo \x1fseg:1\x1f" {
		t.Fatalf("text content = %q", text.Content)
	}
}
