//nolint:testpackage
package dice

import (
	"testing"

	"sealdice-core/message"
)

func TestParseOutboundText(t *testing.T) {
	if got := parseOutboundText(""); got != nil {
		t.Fatalf("empty text segments = %#v, want nil", got)
	}

	segments := parseOutboundText("前缀[CQ:at,qq=100]后缀")
	if len(segments) != 3 {
		t.Fatalf("segment count = %d, want 3", len(segments))
	}
	if text, ok := segments[0].(*message.TextElement); !ok || text.Content != "前缀" {
		t.Fatalf("segment 0 = %#v, want text 前缀", segments[0])
	}
	if at, ok := segments[1].(*message.AtElement); !ok || at.Target != "100" {
		t.Fatalf("segment 1 = %#v, want at 100", segments[1])
	}
	if text, ok := segments[2].(*message.TextElement); !ok || text.Content != "后缀" {
		t.Fatalf("segment 2 = %#v, want text 后缀", segments[2])
	}
}
