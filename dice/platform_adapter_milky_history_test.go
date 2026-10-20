//nolint:testpackage
package dice

import (
	"testing"

	milky "github.com/Szzrain/Milky-go-sdk"

	"sealdice-core/message"
)

func TestPlatformAdapterMilkySegmentHooksOmitRejectedElements(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		send     func(pa *PlatformAdapterMilky, ctx *MsgContext, segments []message.IMessageElement)
	}{
		{
			name: "group", endpoint: milky.EndpointSendGroupMessage,
			send: func(pa *PlatformAdapterMilky, ctx *MsgContext, segments []message.IMessageElement) {
				pa.SendSegmentToGroup(ctx, "QQ-Group:20020", segments, "")
			},
		},
		{
			name: "private", endpoint: milky.EndpointSendPrivateMessage,
			send: func(pa *PlatformAdapterMilky, ctx *MsgContext, segments []message.IMessageElement) {
				pa.SendSegmentToPerson(ctx, "QQ:30030", segments, "")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			session, harness, cleanup := newMilkyForwardHarness(t, "")
			defer cleanup()
			var hookMessage *Message
			pa, ctx := newMilkyForwardTestContext(session, func(_ *MsgContext, msg *Message, _ string) {
				hookMessage = msg
			})
			before := &message.TextElement{Content: "before"}
			after := &message.TextElement{Content: "after"}
			segments := []message.IMessageElement{
				before,
				&message.ImageElement{}, // Rejected: no resource reference.
				&message.AtElement{Target: "invalid"},
				after,
			}

			test.send(pa, ctx, segments)

			endpoint, request := harness.snapshot()
			if endpoint != test.endpoint || len(request.Message) != 2 {
				t.Fatalf("request = (%q, %#v), want two text segments", endpoint, request.Message)
			}
			if hookMessage == nil {
				t.Fatal("OnMessageSend was not called")
			}
			if len(hookMessage.Segment) != 2 || hookMessage.Segment[0] != before || hookMessage.Segment[1] != after {
				t.Fatalf("hook segments = %#v, want only the sent text segments", hookMessage.Segment)
			}
		})
	}
}

func TestPlatformAdapterMilkyForwardHistoryOmitsRejectedElements(t *testing.T) {
	session, _, cleanup := newMilkyForwardHarness(t, "")
	defer cleanup()
	var hookMessage *Message
	pa, ctx := newMilkyForwardTestContext(session, func(_ *MsgContext, msg *Message, _ string) {
		hookMessage = msg
	})
	nodes := []forwardNode{
		{Data: forwardNodeData{Uin: "10010", Name: "MilkyBot", Content: "title"}},
		{Data: forwardNodeData{Uin: "10010", Name: "MilkyBot", Content: "前缀[CQ:shake]后缀"}},
	}

	if !pa.SendGroupForwardMsg(ctx, "QQ-Group:20020", nodes) {
		t.Fatal("Milky forward send returned false")
	}
	if hookMessage == nil || hookMessage.Message != "title\n前缀后缀" {
		t.Fatalf("hook message = %#v, want only the forwarded content that was sent", hookMessage)
	}
	if nodes[1].Data.Content != "前缀[CQ:shake]后缀" {
		t.Fatalf("caller nodes were modified: %q", nodes[1].Data.Content)
	}
}

func TestMilkySentTextPreservesOriginalWhenNothingWasDropped(t *testing.T) {
	const text = "前缀[图:data/images/sealdice.png]后缀"
	source := []message.IMessageElement{
		&message.TextElement{Content: "前缀"},
		&message.ImageElement{URL: "file:///E:/dice/data/images/sealdice.png"},
		&message.TextElement{Content: "后缀"},
	}
	if got := milkySentText(text, source, []bool{true, true, true}); got != text {
		t.Fatalf("sent text = %q, want original %q", got, text)
	}
	// Rejected elements are omitted; history keeps standard resource references.
	want := "前缀[CQ:image,file=file:///E:/dice/data/images/sealdice.png]"
	if got := milkySentText(text, source, []bool{true, true, false}); got != want {
		t.Fatalf("sent text = %q, want %q", got, want)
	}
}
