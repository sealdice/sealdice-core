package dice //nolint:testpackage // Tests exercise the unexported raw segment parser directly.

import (
	"testing"

	"sealdice-core/message"
)

func TestPlatformAdapterHTTPSendSegmentCapturesReply(t *testing.T) {
	adapter := &PlatformAdapterHTTP{}
	var captured HTTPTestMessage
	adapter.OnMessage = func(item HTTPTestMessage) {
		captured = item
	}
	adapter.SendSegmentToPerson(nil, "UI:1001", []message.IMessageElement{
		&message.TextElement{Content: "hello"},
		&message.AtElement{Target: "UI:1002"},
	}, "")

	if len(adapter.RecentMessage) != 1 {
		t.Fatalf("recent message count = %d, want 1", len(adapter.RecentMessage))
	}
	if adapter.RecentMessage[0].UID != "UI:1001" || adapter.RecentMessage[0].Message != "hello[at]" {
		t.Fatalf("recent message = %#v, want captured segment reply", adapter.RecentMessage[0])
	}
	if len(captured.Segments) != 2 || captured.Segments[1].Type != "at" {
		t.Fatalf("captured segments = %#v, want text and at", captured.Segments)
	}
	if captured.Direction != "outgoing" || !captured.IsBot {
		t.Fatalf("captured metadata = %#v, want outgoing bot message", captured)
	}
}

func TestParseHTTPTestSegmentsPreservesCodesWithoutIO(t *testing.T) {
	segments := parseHTTPTestSegments("骰点 [CQ:at,qq=UI:1001] [img:https://example.com/image.png]")
	if len(segments) != 4 {
		t.Fatalf("segment count = %d, want 4: %#v", len(segments), segments)
	}
	if segments[1].Type != "at" || segments[1].Data["qq"] != "UI:1001" {
		t.Fatalf("at segment = %#v", segments[1])
	}
	if segments[3].Type != "image" || segments[3].Data["file"] != "https://example.com/image.png" {
		t.Fatalf("image segment = %#v", segments[3])
	}
}
