package message_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
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
