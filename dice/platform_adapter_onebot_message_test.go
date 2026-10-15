//nolint:testpackage
package dice

import (
	"strings"
	"testing"

	"github.com/tidwall/gjson"

	"sealdice-core/message"
)

func TestConvertSealMsgToMessageChainUsesFileElementURL(t *testing.T) {
	fileURL := "file:///absolute/path/document.txt"
	chain, cqMessage := convertSealMsgToMessageChain([]message.IMessageElement{
		&message.FileElement{File: "document.txt", URL: fileURL},
	})

	if len(chain) != 1 {
		t.Fatalf("message chain length = %d, want 1", len(chain))
	}
	if got := gjson.GetBytes(chain[0].Data, "file").String(); got != fileURL {
		t.Fatalf("OneBot file value = %q, want %q", got, fileURL)
	}
	if !strings.Contains(cqMessage, "file="+fileURL) {
		t.Fatalf("CQ message = %q, want absolute file URL", cqMessage)
	}
}

func TestConvertSealMsgToMessageChainPreservesDefaultResourceParameters(t *testing.T) {
	elements := message.ConvertStringMessage("[CQ:video,file=opaque-video-id,cache=0,timeout=30]")
	if len(elements) != 1 {
		t.Fatalf("element count = %d, want 1", len(elements))
	}

	chain, _ := convertSealMsgToMessageChain(elements)
	if len(chain) != 1 || chain[0].Type != "video" {
		t.Fatalf("message chain = %#v, want one video segment", chain)
	}
	data := gjson.ParseBytes(chain[0].Data)
	if data.Get("file").String() != "opaque-video-id" ||
		data.Get("cache").String() != "0" ||
		data.Get("timeout").String() != "30" {
		t.Fatalf("video data was changed: %s", chain[0].Data)
	}
}
