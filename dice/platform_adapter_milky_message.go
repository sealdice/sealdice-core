package dice

import (
	"sealdice-core/message"
)

// milkySentSegments keeps only the source segments represented in the request.
// The original elements are recorded, so history keeps standard resource
// references instead of Milky-specific wire URIs.
func milkySentSegments(source []message.IMessageElement, included []bool) []message.IMessageElement {
	sent := make([]message.IMessageElement, 0, len(source))
	for index, element := range source {
		if index < len(included) && included[index] {
			sent = append(sent, element)
		}
	}
	return sent
}

// milkySentText keeps the original text when every element was sent, which
// preserves SealCode and formatting. Otherwise it renders only sent segments.
func milkySentText(original string, source []message.IMessageElement, included []bool) string {
	sent := milkySentSegments(source, included)
	if len(sent) == len(source) {
		return original
	}
	_, text := convertSealMsgToMessageChain(sent)
	return text
}
