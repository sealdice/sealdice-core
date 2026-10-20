//nolint:testpackage
package dice

import (
	"fmt"
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

// 入站图片带协议端 file 引用时不能被当作本地文件解析/丢弃，参数要完整保留。
func TestNormalizeIncomingMessageKeepsOpaqueImageParams(t *testing.T) {
	const (
		rawFile = "3B71E9799730D77111FD3EA5E4076773.jpg"
		rawURL  = "https://multimedia.example.invalid/download?appid=1&sign=abc"
	)
	msg := &Message{
		Platform: "QQ",
		Message: fmt.Sprintf(
			"[CQ:image,file=%s,url=%s,subType=1,cache=0]",
			rawFile, message.EscapeCQParam(rawURL),
		),
	}
	NormalizeIncomingMessage(msg)

	if len(msg.Segment) != 1 {
		t.Fatalf("segment count = %d, want 1 (%#v)", len(msg.Segment), msg.Segment)
	}
	img, ok := msg.Segment[0].(*message.ImageElement)
	if !ok {
		t.Fatalf("segment type = %T, want *ImageElement", msg.Segment[0])
	}
	if img.File == nil || img.File.File != rawFile {
		t.Fatalf("File = %#v, want %q", img.File, rawFile)
	}
	if img.File.URL != rawURL || img.URL != rawURL {
		t.Fatalf("URL = %q / %q, want %q", img.File.URL, img.URL, rawURL)
	}

	compat := MessageSegmentsToCompatibilityText(msg.Segment)
	cq := CQParse(compat)
	if cq.Type != "image" {
		t.Fatalf("compat CQ type = %q, want image (%q)", cq.Type, compat)
	}
	if cq.Args["file"] != rawFile {
		t.Fatalf("compat file = %q, want %q (%q)", cq.Args["file"], rawFile, compat)
	}
	if got := message.UnescapeCQParam(cq.Args["url"]); got != rawURL {
		t.Fatalf("compat url = %q, want %q (%q)", got, rawURL, compat)
	}
	if cq.Args["subType"] != "1" || cq.Args["cache"] != "0" {
		t.Fatalf("compat extra params lost: %#v (%q)", cq.Args, compat)
	}
}

func TestCommandParseNewPreservesOpaqueImageFile(t *testing.T) {
	const rawFile = "3B71E9799730D77111FD3EA5E4076773.jpg"
	d, session, ep := newParserTestSession()
	ctx := &MsgContext{Session: session, EndPoint: ep, Dice: d}

	msg := &Message{Platform: "QQ", Message: ".img [CQ:image,file=" + rawFile + "]"}
	NormalizeIncomingMessage(msg)

	cmdArgs := CommandParseNew(ctx, msg)
	if cmdArgs == nil {
		t.Fatal("CommandParseNew returned nil")
	}
	if !strings.Contains(cmdArgs.CleanArgs, "[CQ:image,file="+rawFile+"]") {
		t.Fatalf("CleanArgs lost opaque image file: %q", cmdArgs.CleanArgs)
	}
	if strings.Contains(cmdArgs.CleanArgs, "找不到") {
		t.Fatalf("opaque image file was treated as a missing local file: %q", cmdArgs.CleanArgs)
	}
}

func TestMessageSegmentsToCompatibilityTextPreservesImageParams(t *testing.T) {
	img := &message.ImageElement{
		File: &message.FileElement{
			File: "hash.jpg",
			URL:  "https://example.invalid/a.png?x=1&y=2",
			RawParams: map[string]string{
				"file":    "hash.jpg",
				"url":     "https://example.invalid/a.png?x=1&y=2",
				"subType": "1",
				"cache":   "0",
			},
		},
		URL: "https://example.invalid/a.png?x=1&y=2",
	}
	text := MessageSegmentsToCompatibilityText([]message.IMessageElement{img})
	cq := CQParse(text)
	if cq.Type != "image" || cq.Args["file"] != "hash.jpg" || cq.Args["subType"] != "1" || cq.Args["cache"] != "0" {
		t.Fatalf("compat params lost: %#v (%q)", cq.Args, text)
	}
	if got := message.UnescapeCQParam(cq.Args["url"]); got != "https://example.invalid/a.png?x=1&y=2" {
		t.Fatalf("compat url = %q (%q)", got, text)
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
