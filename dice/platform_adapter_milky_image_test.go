//nolint:testpackage
package dice

import (
	"testing"

	milky "github.com/Szzrain/Milky-go-sdk"

	"sealdice-core/message"
)

func TestParseMessageToMilkyImages(t *testing.T) {
	_, file := newMilkyLocalImage(t)
	localURI := expectedMilkyLocalImageURI(file.URL)
	tests := []struct {
		name    string
		image   *message.ImageElement
		uri     string
		summary string
	}{
		{name: "local image", image: &message.ImageElement{URL: file.URL, File: file}, uri: localURI, summary: file.File},
		{name: "file fallback", image: &message.ImageElement{File: file}, uri: localURI, summary: file.File},
		{name: "local URL only", image: &message.ImageElement{URL: file.URL}, uri: localURI},
		{name: "remote URL only", image: &message.ImageElement{URL: "https://example.com/image.png"}, uri: "https://example.com/image.png"},
		{name: "base64 URL only", image: &message.ImageElement{URL: "base64://aGVsbG8="}, uri: "base64://aGVsbG8="},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			elements := ParseMessageToMilky([]message.IMessageElement{test.image})
			if len(elements) != 1 {
				t.Fatalf("converted %d elements, want one image", len(elements))
			}
			image, ok := elements[0].(*milky.ImageElement)
			if !ok {
				t.Fatalf("converted %T, want *milky.ImageElement", elements[0])
			}
			if image.URI != test.uri || image.Summary != test.summary || image.SubType != "normal" {
				t.Fatalf("converted image = %#v, want URI %q, summary %q and normal subtype", image, test.uri, test.summary)
			}
		})
	}
}

func TestParseMessageToMilkySkipsInvalidImages(t *testing.T) {
	elements := ParseMessageToMilky([]message.IMessageElement{
		&message.TextElement{Content: "before"},
		(*message.ImageElement)(nil),
		&message.ImageElement{},
		&message.ImageElement{File: &message.FileElement{}},
		&message.TextElement{Content: "after"},
	})
	if len(elements) != 2 {
		t.Fatalf("converted %d elements, want two text elements", len(elements))
	}
	for index, text := range []string{"before", "after"} {
		element, ok := elements[index].(*milky.TextElement)
		if !ok || element.Text != text {
			t.Fatalf("element %d = %#v, want text %q", index, elements[index], text)
		}
	}
}

func TestPlatformAdapterMilkySendsLocalImagesWithNativePaths(t *testing.T) {
	filename, file := newMilkyLocalImage(t)
	wantURI := expectedMilkyLocalImageURI(file.URL)
	inputs := []struct {
		name string
		text string
	}{
		{name: "SealCode", text: "before[图:" + filename + "]after"},
		{name: "CQ", text: "before[CQ:image,file=" + message.EscapeCQParam(filename) + "]after"},
	}
	tests := []struct {
		name      string
		endpoint  string
		forwarded bool
		send      func(pa *PlatformAdapterMilky, ctx *MsgContext, text string)
	}{
		{
			name: "group string", endpoint: milky.EndpointSendGroupMessage,
			send: func(pa *PlatformAdapterMilky, ctx *MsgContext, text string) {
				pa.SendToGroup(ctx, "QQ-Group:20020", text, "")
			},
		},
		{
			name: "private string", endpoint: milky.EndpointSendPrivateMessage,
			send: func(pa *PlatformAdapterMilky, ctx *MsgContext, text string) {
				pa.SendToPerson(ctx, "QQ:30030", text, "")
			},
		},
		{
			name: "group segments", endpoint: milky.EndpointSendGroupMessage,
			send: func(pa *PlatformAdapterMilky, ctx *MsgContext, text string) {
				pa.SendSegmentToGroup(ctx, "QQ-Group:20020", message.ConvertStringMessage(text), "")
			},
		},
		{
			name: "private segments", endpoint: milky.EndpointSendPrivateMessage,
			send: func(pa *PlatformAdapterMilky, ctx *MsgContext, text string) {
				pa.SendSegmentToPerson(ctx, "QQ:30030", message.ConvertStringMessage(text), "")
			},
		},
		{
			name: "group forward", endpoint: milky.EndpointSendGroupMessage, forwarded: true,
			send: func(pa *PlatformAdapterMilky, ctx *MsgContext, text string) {
				nodes := []forwardNode{{Data: forwardNodeData{Uin: "10010", Name: "MilkyBot", Content: text}}}
				if !pa.SendGroupForwardMsg(ctx, "QQ-Group:20020", nodes) {
					t.Error("group forward send failed")
				}
			},
		},
		{
			name: "private forward", endpoint: milky.EndpointSendPrivateMessage, forwarded: true,
			send: func(pa *PlatformAdapterMilky, ctx *MsgContext, text string) {
				nodes := []forwardNode{{Data: forwardNodeData{Uin: "10010", Name: "MilkyBot", Content: text}}}
				if !pa.SendPrivateForwardMsg(ctx, "QQ:30030", nodes) {
					t.Error("private forward send failed")
				}
			},
		},
	}

	for _, input := range inputs {
		for _, test := range tests {
			t.Run(input.name+"/"+test.name, func(t *testing.T) {
				session, harness, cleanup := newMilkyForwardHarness(t, "")
				defer cleanup()
				hookCalls := 0
				pa, ctx := newMilkyForwardTestContext(session, func(_ *MsgContext, _ *Message, _ string) {
					hookCalls++
				})
				test.send(pa, ctx, input.text)
				endpoint, request := harness.snapshot()
				if endpoint != test.endpoint || harness.requestCount() != 1 || hookCalls != 1 {
					t.Fatalf("send result: endpoint=%q, requests=%d, hooks=%d", endpoint, harness.requestCount(), hookCalls)
				}

				var imageType, imageURI, before, after string
				if test.forwarded {
					if len(request.Message) != 1 || len(request.Message[0].Data.Messages) != 1 {
						t.Fatalf("expected one forwarded node, got %#v", request.Message)
					}
					segments := request.Message[0].Data.Messages[0].Segments
					if len(segments) != 3 {
						t.Fatalf("forwarded segment count = %d, want 3", len(segments))
					}
					imageType, imageURI = segments[1].Type, segments[1].Data.URI
					before, after = segments[0].Data.Text, segments[2].Data.Text
				} else {
					if len(request.Message) != 3 {
						t.Fatalf("segment count = %d, want 3", len(request.Message))
					}
					imageType, imageURI = request.Message[1].Type, request.Message[1].Data.URI
					before, after = request.Message[0].Data.Text, request.Message[2].Data.Text
				}
				if imageType != string(milky.Image) || imageURI != wantURI {
					t.Errorf("image segment = (%q, %q), want image with URI %q", imageType, imageURI, wantURI)
				}
				if before != "before" || after != "after" {
					t.Errorf("surrounding text = (%q, %q), want (before, after)", before, after)
				}
			})
		}
	}
}
