package message_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"sealdice-core/message"
)

func TestEscapeUnescapeCQParamRoundTrip(t *testing.T) {
	cases := []string{
		"http://127.0.0.1:8090/?url=a.png&cx=1%25&precrop&w=1024&h=1024",
		"plain text",
		"a,b[c]d&e",
		"",
		"&#44;&#91;&#93;&amp;",
	}
	for _, c := range cases {
		if got := message.UnescapeCQParam(message.EscapeCQParam(c)); got != c {
			t.Fatalf("round-trip %q mismatch: got %q", c, got)
		}
	}
}

// TestConvertStringMessageRemoteImageQueryParams 复现 issue #1783：
// 海豹码 [图:URL] 经 SealCodeToCqCode 转义后，解析 CQ 参数必须逆转义，
// 否则 URL 中的 & 会变成 &amp; 或 amp;，导致图床查询参数失效。
func TestConvertStringMessageRemoteImageQueryParams(t *testing.T) {
	var calls atomic.Int32
	var gotQuery atomic.Value
	fake := []byte("fake image bytes")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		gotQuery.Store(r.URL.RawQuery)
		_, _ = w.Write(fake)
	}))
	defer srv.Close()

	u := fmt.Sprintf("%s/?url=target.png&cx=39.6973%%25&cy=33.0078%%25&cw=19.9707%%25", srv.URL)
	elems := message.ConvertStringMessage("[图:" + u + "]")
	if len(elems) != 1 {
		t.Fatalf("expected exactly 1 element, got %d (%#v)", len(elems), elems)
	}
	img, ok := elems[0].(*message.ImageElement)
	if !ok {
		t.Fatalf("expected *ImageElement, got %T", elems[0])
	}

	parsed, err := url.Parse(img.URL)
	if err != nil {
		t.Fatalf("parse img.URL %q failed: %v", img.URL, err)
	}
	q := parsed.Query()
	if q.Get("cx") != "39.6973%" {
		t.Fatalf("cx param corrupted/lost in final URL: img.URL=%q query=%q", img.URL, parsed.RawQuery)
	}
	if q.Get("cy") != "33.0078%" {
		t.Fatalf("cy param corrupted/lost in final URL: img.URL=%q query=%q", img.URL, parsed.RawQuery)
	}

	// 必须真实发起过图片服务器请求，否则下方的“无转义残留”断言会空洞通过
	if calls.Load() == 0 {
		t.Fatal("expected the image server to be requested")
	}

	// 图床实际收到的请求也不能含转义残留（&amp; 或 amp;）
	if raw, _ := gotQuery.Load().(string); strings.Contains(raw, "&amp;") || strings.Contains(raw, "amp;") {
		t.Fatalf("image server received escaped query: %q", raw)
	}
}

func TestConvertStringMessageCQParamUnescape(t *testing.T) {
	elems := message.ConvertStringMessage("[CQ:tts,text=a&amp;b&#91;x&#93;]")
	if len(elems) != 1 {
		t.Fatalf("expected exactly 1 element, got %d (%#v)", len(elems), elems)
	}
	tts, ok := elems[0].(*message.TTSElement)
	if !ok {
		t.Fatalf("expected *TTSElement, got %T", elems[0])
	}
	if want := "a&b[x]"; tts.Content != want {
		t.Fatalf("tts content = %q, want %q", tts.Content, want)
	}
}

func localPathFromFileURL(t *testing.T, raw string) string {
	t.Helper()

	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse file URL %q: %v", raw, err)
	}
	if parsed.Scheme != "file" {
		t.Fatalf("resource URL scheme = %q, want file", parsed.Scheme)
	}
	localPath := filepath.FromSlash(parsed.Path)
	if runtime.GOOS == "windows" && len(localPath) >= 3 && localPath[0] == filepath.Separator && localPath[2] == ':' {
		localPath = localPath[1:]
	}
	return filepath.Clean(localPath)
}

func writeResourceFile(t *testing.T, name string) string {
	t.Helper()

	filename := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(filename, []byte("resource"), 0o600); err != nil {
		t.Fatalf("write resource file: %v", err)
	}
	return filename
}

func defaultElementData(t *testing.T, element message.IMessageElement) map[string]string {
	t.Helper()

	defaultElement, ok := element.(*message.DefaultElement)
	if !ok {
		t.Fatalf("element type = %T, want *message.DefaultElement", element)
	}
	data := map[string]string{}
	if err := json.Unmarshal(defaultElement.Data, &data); err != nil {
		t.Fatalf("unmarshal default element data: %v", err)
	}
	return data
}

func TestNormalizeLocalResourcePath(t *testing.T) {
	filename := writeResourceFile(t, "card with space.png")
	wantPath, err := filepath.Abs(filename)
	if err != nil {
		t.Fatalf("resolve expected absolute path: %v", err)
	}

	normalized, err := message.NormalizeLocalResourcePath(filename)
	if err != nil {
		t.Fatalf("NormalizeLocalResourcePath() error = %v", err)
	}
	if got := localPathFromFileURL(t, normalized); got != filepath.Clean(wantPath) {
		t.Fatalf("normalized local path = %q, want %q", got, filepath.Clean(wantPath))
	}

	normalizedAgain, err := message.NormalizeLocalResourcePath(normalized)
	if err != nil {
		t.Fatalf("normalize canonical file URL: %v", err)
	}
	if normalizedAgain != normalized {
		t.Fatalf("canonical file URL changed: got %q, want %q", normalizedAgain, normalized)
	}

	passthrough := []string{
		"https://example.com/card.png",
		"base64://cmVzb3VyY2U=",
		"opaque-onebot-resource-id",
		"adapter/cache/resource-id",
		"0123456789abcdef0123456789abcdef.image",
		"mxc://server/resource",
	}
	for _, input := range passthrough {
		got, normalizeErr := message.NormalizeLocalResourcePath(input)
		if normalizeErr != nil {
			t.Fatalf("NormalizeLocalResourcePath(%q) error = %v", input, normalizeErr)
		}
		if got != input {
			t.Fatalf("NormalizeLocalResourcePath(%q) = %q, want unchanged", input, got)
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	outside := filepath.Dir(cwd)
	_, err = message.NormalizeLocalResourcePath(outside)
	var fileErr *message.CQFileError
	if !errors.As(err, &fileErr) || fileErr.Kind != message.CQFileErrRestricted {
		t.Fatalf("outside path error = %v, want CQFileErrRestricted", err)
	}
}

func TestConvertStringMessageNormalizesAllLocalResourceKinds(t *testing.T) {
	imagePath := writeResourceFile(t, "image.png")
	recordPath := writeResourceFile(t, "record.wav")
	videoPath := writeResourceFile(t, "video.mp4")
	filePath := writeResourceFile(t, "document.txt")

	expectedImage, err := message.NormalizeLocalResourcePath(imagePath)
	if err != nil {
		t.Fatal(err)
	}
	expectedRecord, err := message.NormalizeLocalResourcePath(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	expectedVideo, err := message.NormalizeLocalResourcePath(videoPath)
	if err != nil {
		t.Fatal(err)
	}
	expectedFile, err := message.NormalizeLocalResourcePath(filePath)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		input string
		check func(t *testing.T, element message.IMessageElement)
	}{
		{
			name:  "explicit CQ image",
			input: "[CQ:image,file=" + message.EscapeCQParam(imagePath) + "]",
			check: func(t *testing.T, element message.IMessageElement) {
				imageElement, ok := element.(*message.ImageElement)
				if !ok || imageElement.URL != expectedImage {
					t.Fatalf("image element = %#v, want URL %q", element, expectedImage)
				}
			},
		},
		{
			name:  "explicit CQ record",
			input: "[CQ:record,file=" + message.EscapeCQParam(recordPath) + "]",
			check: func(t *testing.T, element message.IMessageElement) {
				recordElement, ok := element.(*message.RecordElement)
				if !ok || recordElement.File == nil || recordElement.File.URL != expectedRecord {
					t.Fatalf("record element = %#v, want URL %q", element, expectedRecord)
				}
			},
		},
		{
			name:  "explicit CQ video",
			input: "[CQ:video,file=" + message.EscapeCQParam(videoPath) + ",cache=0]",
			check: func(t *testing.T, element message.IMessageElement) {
				data := defaultElementData(t, element)
				if data["file"] != expectedVideo || data["cache"] != "0" {
					t.Fatalf("video data = %#v, want file=%q and cache=0", data, expectedVideo)
				}
			},
		},
		{
			name:  "explicit CQ file",
			input: "[CQ:file,file=" + message.EscapeCQParam(filePath) + "]",
			check: func(t *testing.T, element message.IMessageElement) {
				fileElement, ok := element.(*message.FileElement)
				if !ok || fileElement.URL != expectedFile {
					t.Fatalf("file element = %#v, want URL %q", element, expectedFile)
				}
			},
		},
		{
			name:  "Seal image",
			input: "[图:" + imagePath + "]",
			check: func(t *testing.T, element message.IMessageElement) {
				imageElement, ok := element.(*message.ImageElement)
				if !ok || imageElement.URL != expectedImage {
					t.Fatalf("image element = %#v, want URL %q", element, expectedImage)
				}
			},
		},
		{
			name:  "Seal voice",
			input: "[voice:" + recordPath + "]",
			check: func(t *testing.T, element message.IMessageElement) {
				recordElement, ok := element.(*message.RecordElement)
				if !ok || recordElement.File == nil || recordElement.File.URL != expectedRecord {
					t.Fatalf("record element = %#v, want URL %q", element, expectedRecord)
				}
			},
		},
		{
			name:  "Seal video",
			input: "[视频:" + videoPath + "]",
			check: func(t *testing.T, element message.IMessageElement) {
				data := defaultElementData(t, element)
				if data["file"] != expectedVideo {
					t.Fatalf("video data = %#v, want file=%q", data, expectedVideo)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			elements := message.ConvertStringMessage(test.input)
			if len(elements) != 1 {
				t.Fatalf("element count = %d, want 1 (%#v)", len(elements), elements)
			}
			test.check(t, elements[0])
		})
	}
}

func TestConvertStringMessageLeavesNonLocalVideoReferencesUnchanged(t *testing.T) {
	tests := []string{
		"https://example.com/video.mp4",
		"base64://dmlkZW8=",
		"opaque-onebot-video-id",
	}
	for _, resource := range tests {
		elements := message.ConvertStringMessage("[CQ:video,file=" + message.EscapeCQParam(resource) + ",cache=0]")
		if len(elements) != 1 {
			t.Fatalf("resource %q produced %d elements, want 1", resource, len(elements))
		}
		data := defaultElementData(t, elements[0])
		if data["file"] != resource || data["cache"] != "0" {
			t.Fatalf("resource %q converted to %#v", resource, data)
		}
	}

	elements := message.ConvertStringMessage("[CQ:video,file=adapter-cache-id,url=https://example.com/video.mp4]")
	data := defaultElementData(t, elements[0])
	if data["file"] != "adapter-cache-id" || data["url"] != "https://example.com/video.mp4" {
		t.Fatalf("video with URL converted to %#v", data)
	}
}
