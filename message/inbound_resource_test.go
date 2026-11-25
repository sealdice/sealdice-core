package message_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"sealdice-core/message"
)

// 入站 CQ 资源引用必须保持不透明：协议端负责获取图片，核心不能 Stat/下载。
func TestConvertStringMessageInboundKeepsOpaqueFile(t *testing.T) {
	const rawFile = "3B71E9799730D77111FD3EA5E4076773.jpg"

	elems := message.ConvertStringMessage(
		"[CQ:image,file="+rawFile+"]",
		message.WithResourceMode(message.ResourceInbound),
	)
	if len(elems) != 1 {
		t.Fatalf("element count = %d, want 1 (%#v)", len(elems), elems)
	}
	img, ok := elems[0].(*message.ImageElement)
	if !ok {
		t.Fatalf("element type = %T, want *ImageElement", elems[0])
	}
	if img.File == nil || img.File.File != rawFile {
		t.Fatalf("File = %#v, want raw file %q", img.File, rawFile)
	}
	if img.URL != rawFile {
		t.Fatalf("URL = %q, want raw file fallback %q", img.URL, rawFile)
	}
	if got := img.File.RawParams["file"]; got != rawFile {
		t.Fatalf("RawParams[file] = %q, want %q", got, rawFile)
	}
}

func TestConvertStringMessageInboundKeepsFileAndURL(t *testing.T) {
	const (
		rawFile = "3B71E9799730D77111FD3EA5E4076773.jpg"
		rawURL  = "https://multimedia.example.invalid/download?appid=1&sign=abc"
	)
	elems := message.ConvertStringMessage(
		fmt.Sprintf("[CQ:image,file=%s,url=%s,subType=1,cache=0]", rawFile, message.EscapeCQParam(rawURL)),
		message.WithResourceMode(message.ResourceInbound),
	)
	if len(elems) != 1 {
		t.Fatalf("element count = %d, want 1 (%#v)", len(elems), elems)
	}
	img, ok := elems[0].(*message.ImageElement)
	if !ok {
		t.Fatalf("element type = %T, want *ImageElement", elems[0])
	}
	if img.File == nil || img.File.File != rawFile {
		t.Fatalf("File = %#v, want %q", img.File, rawFile)
	}
	if img.File.URL != rawURL || img.URL != rawURL {
		t.Fatalf("URL = %q / %q, want %q", img.File.URL, img.URL, rawURL)
	}
	if img.File.RawParams["subType"] != "1" || img.File.RawParams["cache"] != "0" {
		t.Fatalf("raw params lost: %#v", img.File.RawParams)
	}
}

// 入站不翻译海豹码：它是海豹扩展，只应在核心主动发送时转换。
func TestConvertStringMessageInboundDoesNotTranslateSealCodes(t *testing.T) {
	const input = "[图:./missing.png]"
	elems := message.ConvertStringMessage(input, message.WithResourceMode(message.ResourceInbound))
	if len(elems) != 1 {
		t.Fatalf("element count = %d, want 1 (%#v)", len(elems), elems)
	}
	text, ok := elems[0].(*message.TextElement)
	if !ok {
		t.Fatalf("element type = %T, want *TextElement", elems[0])
	}
	if text.Content != input {
		t.Fatalf("text = %q, want %q", text.Content, input)
	}
}

// 入站解析不能向远程资源发起请求。
func TestConvertStringMessageInboundDoesNotFetchRemoteResource(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte("should-not-be-fetched"))
	}))
	defer srv.Close()

	url := srv.URL + "/image.png?sign=secret"
	elems := message.ConvertStringMessage(
		"[CQ:image,file="+message.EscapeCQParam(url)+"]",
		message.WithResourceMode(message.ResourceInbound),
	)
	if calls.Load() != 0 {
		t.Fatalf("inbound parsing fetched remote resource %d times", calls.Load())
	}
	if len(elems) != 1 {
		t.Fatalf("element count = %d, want 1 (%#v)", len(elems), elems)
	}
	img, ok := elems[0].(*message.ImageElement)
	if !ok {
		t.Fatalf("element type = %T, want *ImageElement", elems[0])
	}
	if img.File == nil || img.File.File != url {
		t.Fatalf("File = %#v, want opaque URL %q", img.File, url)
	}
}

// 出站遇到协议端不透明引用（裸 hash/缓存 id）应透传，而不是报错丢弃。
func TestConvertStringMessageOutboundKeepsOpaqueReference(t *testing.T) {
	const rawFile = "adapter-cache-id"
	elems := message.ConvertStringMessage("[CQ:image,file=" + rawFile + "]")
	if len(elems) != 1 {
		t.Fatalf("element count = %d, want 1 (%#v)", len(elems), elems)
	}
	img, ok := elems[0].(*message.ImageElement)
	if !ok {
		t.Fatalf("element type = %T, want *ImageElement", elems[0])
	}
	if img.File == nil || img.File.File != rawFile {
		t.Fatalf("File = %#v, want opaque reference %q", img.File, rawFile)
	}
	if img.URL != rawFile {
		t.Fatalf("URL = %q, want opaque reference %q", img.URL, rawFile)
	}
}
