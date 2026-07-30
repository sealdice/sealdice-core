//nolint:testpackage
package dice

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	milky "github.com/Szzrain/Milky-go-sdk"
)

type milkyForwardRequest struct {
	GroupID int64 `json:"group_id"`
	UserID  int64 `json:"user_id"`
	Message []struct {
		Type string `json:"type"`
		Data struct {
			Messages []struct {
				UserID     int64  `json:"user_id"`
				SenderName string `json:"sender_name"`
				LegacyName string `json:"name"`
				Segments   []struct {
					Type string `json:"type"`
					Data struct {
						Text string `json:"text"`
					} `json:"data"`
				} `json:"segments"`
			} `json:"messages"`
		} `json:"data"`
	} `json:"message"`
}

type milkyForwardHarness struct {
	t        *testing.T
	apiError string

	mu       sync.Mutex
	endpoint string
	request  milkyForwardRequest
}

func newMilkyForwardHarness(t *testing.T, apiError string) (*milky.Session, *milkyForwardHarness, func()) {
	t.Helper()
	h := &milkyForwardHarness{t: t, apiError: apiError}
	server := httptest.NewServer(http.HandlerFunc(h.handle))
	session, err := milky.New("ws://127.0.0.1", server.URL, "", noopMilkyLogger{})
	if err != nil {
		server.Close()
		t.Fatalf("failed to create Milky session: %v", err)
	}
	return session, h, server.Close
}

func (h *milkyForwardHarness) handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var request milkyForwardRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.t.Errorf("failed to decode Milky forward request: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.mu.Lock()
	h.endpoint = strings.TrimPrefix(r.URL.Path, "/")
	h.request = request
	h.mu.Unlock()

	if h.apiError != "" {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "failed",
			"retcode": 1,
			"message": h.apiError,
		})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"retcode": 0,
		"data": map[string]any{
			"message_seq": 321,
			"time":        1,
		},
	})
}

func (h *milkyForwardHarness) snapshot() (string, milkyForwardRequest) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.endpoint, h.request
}

func newMilkyForwardTestContext(session *milky.Session, onMessageSend func(ctx *MsgContext, msg *Message, flag string)) (*PlatformAdapterMilky, *MsgContext) {
	d := &Dice{
		Config: Config{
			BaseConfig: BaseConfig{
				MessageDelayRangeStart: 0,
				MessageDelayRangeEnd:   0,
			},
		},
		ExtList: []*ExtInfo{{OnMessageSend: onMessageSend}},
	}
	d.ImSession = &IMSession{Parent: d}
	ep := &EndPointInfo{
		EndPointInfoBase: EndPointInfoBase{
			UserID:       "QQ:10010",
			Nickname:     "MilkyBot",
			Platform:     "QQ",
			ProtocolType: "milky",
		},
	}
	pa := &PlatformAdapterMilky{EndPoint: ep, IntentSession: session}
	ep.Adapter = pa
	ep.BindRuntime(d.ImSession)
	return pa, &MsgContext{Dice: d, EndPoint: ep}
}

func TestPlatformAdapterMilkySendForwardMessage(t *testing.T) {
	nodes := buildForwardNodes("MilkyBot", "10010", "COC7制卡结果", []string{"力量:50", "力量:60"})
	tests := []struct {
		name            string
		endpoint        string
		targetID        int64
		messageType     string
		send            func(pa *PlatformAdapterMilky, ctx *MsgContext) bool
		wantHookGroupID string
	}{
		{
			name:        "group",
			endpoint:    milky.EndpointSendGroupMessage,
			targetID:    20020,
			messageType: "group",
			send: func(pa *PlatformAdapterMilky, ctx *MsgContext) bool {
				return pa.SendGroupForwardMsg(ctx, "QQ-Group:20020", nodes)
			},
			wantHookGroupID: "QQ-Group:20020",
		},
		{
			name:        "private",
			endpoint:    milky.EndpointSendPrivateMessage,
			targetID:    30030,
			messageType: "private",
			send: func(pa *PlatformAdapterMilky, ctx *MsgContext) bool {
				return pa.SendPrivateForwardMsg(ctx, "QQ:30030", nodes)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			session, harness, cleanup := newMilkyForwardHarness(t, "")
			defer cleanup()

			var hookCalls int
			var hookMessage *Message
			pa, ctx := newMilkyForwardTestContext(session, func(_ *MsgContext, msg *Message, flag string) {
				hookCalls++
				hookMessage = msg
				if flag != "" {
					t.Errorf("forward hook flag = %q, want empty", flag)
				}
			})

			if !test.send(pa, ctx) {
				t.Fatal("Milky forward send returned false")
			}

			endpoint, request := harness.snapshot()
			if endpoint != test.endpoint {
				t.Fatalf("endpoint = %q, want %q", endpoint, test.endpoint)
			}
			if test.messageType == "group" && request.GroupID != test.targetID {
				t.Fatalf("group_id = %d, want %d", request.GroupID, test.targetID)
			}
			if test.messageType == "private" && request.UserID != test.targetID {
				t.Fatalf("user_id = %d, want %d", request.UserID, test.targetID)
			}
			if len(request.Message) != 1 || request.Message[0].Type != string(milky.Forward) {
				t.Fatalf("message = %#v, want one forward element", request.Message)
			}

			messages := request.Message[0].Data.Messages
			if len(messages) != len(nodes) {
				t.Fatalf("forward node count = %d, want %d", len(messages), len(nodes))
			}
			for i, node := range messages {
				if node.UserID != 10010 || node.SenderName != "MilkyBot" {
					t.Errorf("node %d sender = (%d, %q), want (10010, %q)", i, node.UserID, node.SenderName, "MilkyBot")
				}
				if node.LegacyName != "" {
					t.Errorf("node %d contains deprecated name field %q", i, node.LegacyName)
				}
				if len(node.Segments) != 1 || node.Segments[0].Type != string(milky.Text) || node.Segments[0].Data.Text != nodes[i].Data.Content {
					t.Errorf("node %d segments = %#v, want text %q", i, node.Segments, nodes[i].Data.Content)
				}
			}

			if hookCalls != 1 || hookMessage == nil {
				t.Fatalf("OnMessageSend calls = %d, want 1", hookCalls)
			}
			if hookMessage.MessageType != test.messageType || hookMessage.GroupID != test.wantHookGroupID {
				t.Errorf("hook destination = (%q, %q), want (%q, %q)", hookMessage.MessageType, hookMessage.GroupID, test.messageType, test.wantHookGroupID)
			}
			if hookMessage.Message != forwardNodesToText(nodes) || hookMessage.RawID != int64(321) {
				t.Errorf("hook message = (%q, %#v), want (%q, 321)", hookMessage.Message, hookMessage.RawID, forwardNodesToText(nodes))
			}
		})
	}
}

func TestPlatformAdapterMilkyForwardMessageFailures(t *testing.T) {
	validNodes := buildForwardNodes("MilkyBot", "10010", "title", []string{"content"})
	session, _, cleanup := newMilkyForwardHarness(t, "forward rejected")
	defer cleanup()
	pa, ctx := newMilkyForwardTestContext(session, func(_ *MsgContext, _ *Message, _ string) {
		t.Error("failed forward send must not trigger OnMessageSend")
	})

	tests := []struct {
		name string
		send func() bool
	}{
		{name: "API error", send: func() bool { return pa.SendGroupForwardMsg(ctx, "QQ-Group:20020", validNodes) }},
		{name: "empty nodes", send: func() bool { return pa.SendGroupForwardMsg(ctx, "QQ-Group:20020", nil) }},
		{name: "invalid group ID", send: func() bool { return pa.SendGroupForwardMsg(ctx, "QQ-Group:invalid", validNodes) }},
		{name: "invalid private ID", send: func() bool { return pa.SendPrivateForwardMsg(ctx, "QQ:invalid", validNodes) }},
		{name: "invalid node user ID", send: func() bool {
			return pa.SendGroupForwardMsg(ctx, "QQ-Group:20020", []forwardNode{{Data: forwardNodeData{Uin: "invalid", Name: "MilkyBot", Content: "content"}}})
		}},
		{name: "missing session", send: func() bool {
			return (&PlatformAdapterMilky{}).SendGroupForwardMsg(ctx, "QQ-Group:20020", validNodes)
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.send() {
				t.Fatal("Milky forward send returned true, want false so the caller can use its text fallback")
			}
		})
	}
}
