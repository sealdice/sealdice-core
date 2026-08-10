//nolint:testpackage
package dice

import (
	"testing"
	"time"

	milky "github.com/Szzrain/Milky-go-sdk"

	"sealdice-core/message"
)

func (h *milkyForwardHarness) requestCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.count
}

func (h *milkyForwardHarness) waitForEndpoint(t *testing.T, expected string) milkyForwardRequest {
	t.Helper()
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		endpoint, request := h.snapshot()
		if endpoint == expected {
			return request
		}
		select {
		case <-h.signal:
		case <-timer.C:
			t.Fatalf("Milky endpoint = %q, want %q", endpoint, expected)
			return milkyForwardRequest{}
		}
	}
}

func TestParseMessageToMilkyDropsShake(t *testing.T) {
	segments := message.ConvertStringMessage("[CQ:shake]")
	if len(segments) != 1 {
		t.Fatalf("parsed segment count = %d, want 1", len(segments))
	}
	if _, ok := segments[0].(*message.DefaultElement); !ok {
		t.Fatalf("parsed segment type = %T, want *message.DefaultElement", segments[0])
	}
	if converted := ParseMessageToMilky(segments); len(converted) != 0 {
		t.Fatalf("converted Milky segment count = %d, want 0", len(converted))
	}
}

func TestPlatformAdapterMilkySkipsMessagesWithoutSupportedElements(t *testing.T) {
	shakeSegments := message.ConvertStringMessage("[CQ:shake]")
	tests := []struct {
		name string
		send func(pa *PlatformAdapterMilky, ctx *MsgContext)
	}{
		{
			name: "group string shake",
			send: func(pa *PlatformAdapterMilky, ctx *MsgContext) {
				pa.SendToGroup(ctx, "QQ-Group:20020", "[CQ:shake]", "")
			},
		},
		{
			name: "private string shake",
			send: func(pa *PlatformAdapterMilky, ctx *MsgContext) {
				pa.SendToPerson(ctx, "QQ:30030", "[CQ:shake]", "")
			},
		},
		{
			name: "group segment shake",
			send: func(pa *PlatformAdapterMilky, ctx *MsgContext) {
				pa.SendSegmentToGroup(ctx, "QQ-Group:20020", shakeSegments, "")
			},
		},
		{
			name: "private segment shake",
			send: func(pa *PlatformAdapterMilky, ctx *MsgContext) {
				pa.SendSegmentToPerson(ctx, "QQ:30030", shakeSegments, "")
			},
		},
		{
			name: "empty group string",
			send: func(pa *PlatformAdapterMilky, ctx *MsgContext) {
				pa.SendToGroup(ctx, "QQ-Group:20020", "", "")
			},
		},
		{
			name: "invalid group poke",
			send: func(pa *PlatformAdapterMilky, ctx *MsgContext) {
				pa.SendToGroup(ctx, "QQ-Group:20020", "[CQ:poke,qq=invalid]", "")
			},
		},
		{
			name: "invalid at segment",
			send: func(pa *PlatformAdapterMilky, ctx *MsgContext) {
				pa.SendSegmentToPerson(ctx, "QQ:30030", []message.IMessageElement{&message.AtElement{Target: "invalid"}}, "")
			},
		},
		{
			name: "invalid reply segment",
			send: func(pa *PlatformAdapterMilky, ctx *MsgContext) {
				pa.SendSegmentToGroup(ctx, "QQ-Group:20020", []message.IMessageElement{&message.ReplyElement{ReplySeq: "invalid"}}, "")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			session, harness, cleanup := newMilkyForwardHarness(t, "")
			defer cleanup()

			hookCalls := 0
			pa, ctx := newMilkyForwardTestContext(session, func(_ *MsgContext, _ *Message, _ string) {
				hookCalls++
			})

			test.send(pa, ctx)

			if got := harness.requestCount(); got != 0 {
				t.Fatalf("Milky request count = %d, want 0", got)
			}
			if hookCalls != 0 {
				t.Fatalf("OnMessageSend calls = %d, want 0", hookCalls)
			}
		})
	}
}

func TestPlatformAdapterMilkySendsSupportedPartsOfMixedMessage(t *testing.T) {
	const text = "前缀[CQ:shake]后缀"
	tests := []struct {
		name         string
		endpoint     string
		messageType  string
		wantGroupID  string
		wantTargetID int64
		send         func(pa *PlatformAdapterMilky, ctx *MsgContext)
	}{
		{
			name:         "group",
			endpoint:     milky.EndpointSendGroupMessage,
			messageType:  "group",
			wantGroupID:  "QQ-Group:20020",
			wantTargetID: 20020,
			send: func(pa *PlatformAdapterMilky, ctx *MsgContext) {
				pa.SendToGroup(ctx, "QQ-Group:20020", text, "mixed")
			},
		},
		{
			name:         "private",
			endpoint:     milky.EndpointSendPrivateMessage,
			messageType:  "private",
			wantTargetID: 30030,
			send: func(pa *PlatformAdapterMilky, ctx *MsgContext) {
				pa.SendToPerson(ctx, "QQ:30030", text, "mixed")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			session, harness, cleanup := newMilkyForwardHarness(t, "")
			defer cleanup()

			hookCalls := 0
			var hookMessage *Message
			pa, ctx := newMilkyForwardTestContext(session, func(_ *MsgContext, msg *Message, flag string) {
				hookCalls++
				hookMessage = msg
				if flag != "mixed" {
					t.Errorf("OnMessageSend flag = %q, want %q", flag, "mixed")
				}
			})

			test.send(pa, ctx)

			if got := harness.requestCount(); got != 1 {
				t.Fatalf("Milky request count = %d, want 1", got)
			}
			endpoint, request := harness.snapshot()
			if endpoint != test.endpoint {
				t.Fatalf("endpoint = %q, want %q", endpoint, test.endpoint)
			}
			if test.messageType == "group" && request.GroupID != test.wantTargetID {
				t.Errorf("group_id = %d, want %d", request.GroupID, test.wantTargetID)
			}
			if test.messageType == "private" && request.UserID != test.wantTargetID {
				t.Errorf("user_id = %d, want %d", request.UserID, test.wantTargetID)
			}
			if len(request.Message) != 2 {
				t.Fatalf("message segment count = %d, want 2", len(request.Message))
			}
			wantText := []string{"前缀", "后缀"}
			for index, segment := range request.Message {
				if segment.Type != string(milky.Text) || segment.Data.Text != wantText[index] {
					t.Errorf("message segment %d = (%q, %q), want (%q, %q)", index, segment.Type, segment.Data.Text, milky.Text, wantText[index])
				}
			}
			if hookCalls != 1 || hookMessage == nil {
				t.Fatalf("OnMessageSend calls = %d, want 1", hookCalls)
			}
			if hookMessage.MessageType != test.messageType || hookMessage.GroupID != test.wantGroupID {
				t.Errorf("hook destination = (%q, %q), want (%q, %q)", hookMessage.MessageType, hookMessage.GroupID, test.messageType, test.wantGroupID)
			}
			if hookMessage.Message != text || hookMessage.RawID != int64(321) {
				t.Errorf("hook message = (%q, %d), want (%q, 321)", hookMessage.Message, hookMessage.RawID, text)
			}
		})
	}
}

func TestPlatformAdapterMilkyPreservesGroupPoke(t *testing.T) {
	session, harness, cleanup := newMilkyForwardHarness(t, "")
	defer cleanup()

	hookCalls := 0
	var hookMessage *Message
	pa, ctx := newMilkyForwardTestContext(session, func(_ *MsgContext, msg *Message, _ string) {
		hookCalls++
		hookMessage = msg
	})

	pa.SendToGroup(ctx, "QQ-Group:20020", "[CQ:poke,qq=30030]", "")

	request := harness.waitForEndpoint(t, milky.EndpointSendGroupNudge)
	if got := harness.requestCount(); got != 1 {
		t.Fatalf("Milky request count = %d, want 1", got)
	}
	if request.GroupID != int64(20020) || request.UserID != int64(30030) {
		t.Errorf("nudge target = (%d, %d), want (20020, 30030)", request.GroupID, request.UserID)
	}
	if len(request.Message) != 0 {
		t.Errorf("nudge unexpectedly contains message segments: %#v", request.Message)
	}
	if hookCalls != 1 || hookMessage == nil {
		t.Fatalf("OnMessageSend calls = %d, want 1", hookCalls)
	}
	if hookMessage.MessageType != "group" || hookMessage.GroupID != "QQ-Group:20020" || hookMessage.RawID != int64(0) {
		t.Errorf("hook message = (%q, %q, %d), want (%q, %q, 0)", hookMessage.MessageType, hookMessage.GroupID, hookMessage.RawID, "group", "QQ-Group:20020")
	}
}
