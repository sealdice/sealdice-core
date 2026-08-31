package dice

import "sealdice-core/message"

type MessageNormalizeMode int

const (
	NormalizeModeBoundary MessageNormalizeMode = iota
	NormalizeModeCompatibilityView
)

func NormalizeIncomingMessage(msg *Message) {
	if msg == nil {
		return
	}
	if len(msg.Segment) == 0 && msg.Message != "" {
		msg.Segment = message.ConvertStringMessage(msg.Message)
	}
	if msg.Message == "" && len(msg.Segment) > 0 {
		msg.Message = MessageSegmentsToCompatibilityText(msg.Segment)
	}
}

func MessageSegmentsToCompatibilityText(segments []message.IMessageElement) string {
	_, text := convertSealMsgToMessageChain(segments)
	return text
}
