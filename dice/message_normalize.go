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
		msg.segmentSynthesized = true
	}
	if msg.Message == "" && len(msg.Segment) > 0 {
		msg.Message = MessageSegmentsToCompatibilityText(msg.Segment)
	}
}

func MessageSegmentsToCompatibilityText(segments []message.IMessageElement) string {
	_, text := convertSealMsgToMessageChain(segments)
	return text
}

// parseOutboundText 出站字符串的唯一解析点。
// 之后只以消息段发送，适配器不再把字符串解析成段。
func parseOutboundText(text string) []message.IMessageElement {
	if text == "" {
		return nil
	}
	return message.ConvertStringMessage(text)
}
