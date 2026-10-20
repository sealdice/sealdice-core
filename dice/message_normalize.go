package dice

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"

	"sealdice-core/message"
)

// NormalizeIncomingMessage makes Message.Segment the canonical representation
// and keeps Message as a compatibility view.
//
//   - String-only input is converted to segments once, at the input boundary.
//     segmentSynthesized records that the segments came from the legacy string,
//     which lets non-segment-native platforms fall back to their own @ syntax.
//   - Segment-only input is rendered back to compatibility text for legacy
//     consumers that still read Message.Message.
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

// MessageSegmentsToCompatibilityText renders segments as legacy CQ text.
func MessageSegmentsToCompatibilityText(segments []message.IMessageElement) string {
	var builder strings.Builder
	for _, elem := range segments {
		builder.WriteString(segmentToCompatibilityText(elem))
	}
	return builder.String()
}

func segmentToCompatibilityText(elem message.IMessageElement) string {
	switch e := elem.(type) {
	case *message.TextElement:
		return e.Content
	case *message.AtElement:
		return fmt.Sprintf("[CQ:at,qq=%s]", e.Target)
	case *message.FaceElement:
		return fmt.Sprintf("[CQ:face,id=%s]", e.FaceID)
	case *message.FileElement:
		file := e.URL
		if file == "" {
			file = e.File
		}
		if file == "" {
			return ""
		}
		return fmt.Sprintf("[CQ:file,file=%s]", file)
	case *message.ImageElement:
		url := e.URL
		if url == "" && e.File != nil {
			url = e.File.URL
			if url == "" {
				url = e.File.File
			}
		}
		if url == "" {
			return ""
		}
		return fmt.Sprintf("[CQ:image,file=%s]", url)
	case *message.RecordElement:
		if e.File == nil {
			return ""
		}
		file := e.File.URL
		if file == "" {
			file = e.File.File
		}
		if file == "" {
			return ""
		}
		return fmt.Sprintf("[CQ:record,file=%s]", file)
	case *message.ReplyElement:
		if _, err := strconv.Atoi(e.ReplySeq); err != nil {
			return ""
		}
		return fmt.Sprintf("[CQ:reply,id=%s]", e.ReplySeq)
	case *message.TTSElement:
		return fmt.Sprintf("[CQ:tts,text=%s]", e.Content)
	case *message.PokeElement:
		return fmt.Sprintf("[CQ:poke,qq=%s]", e.Target)
	case *message.DefaultElement:
		var params []string
		for key, value := range gjson.ParseBytes(e.Data).Map() {
			params = append(params, fmt.Sprintf("%s=%s", key, value))
		}
		return fmt.Sprintf("[CQ:%s,%s]", e.RawType, strings.Join(params, ","))
	default:
		return ""
	}
}
