package dice_test

import (
	"testing"

	"sealdice-core/dice"
	"sealdice-core/message"
)

func TestNormalizeIncomingMessageBuildsSegmentsFromLegacyMessage(t *testing.T) {
	msg := &dice.Message{
		Platform: "QQ",
		Message:  ".r 1d6",
	}

	dice.NormalizeIncomingMessage(msg)

	if len(msg.Segment) != 1 {
		t.Fatalf("segment count = %d, want 1", len(msg.Segment))
	}
	text, ok := msg.Segment[0].(*message.TextElement)
	if !ok || text.Content != ".r 1d6" {
		t.Fatalf("unexpected segment: %#v", msg.Segment[0])
	}
	if msg.Message != ".r 1d6" {
		t.Fatalf("message view = %q, want original text", msg.Message)
	}
}

func TestNormalizeIncomingMessageDerivesMessageViewFromSegments(t *testing.T) {
	msg := &dice.Message{
		Platform: "QQ",
		Segment: []message.IMessageElement{
			&message.TextElement{Content: ".r "},
			&message.ImageElement{URL: "https://example.invalid/a.png"},
		},
	}

	dice.NormalizeIncomingMessage(msg)

	if msg.Message == "" {
		t.Fatal("message compatibility view should be derived")
	}
	if len(msg.Segment) != 2 {
		t.Fatalf("segment count = %d, want 2", len(msg.Segment))
	}
}
