package dice

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"

	"sealdice-core/message"
)

var urlWithQueryRe = regexp.MustCompile(`(https?://[^\s\[\]]+?)\?[^\s\[\]]*`)

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
		if platformUsesCQSyntax(msg.Platform) {
			msg.Segment = message.ConvertStringMessage(msg.Message)
		} else {
			// 非 CQ 平台的字符串不含 CQ 码，保持为单个文本段，
			// 用户输入的字面 [CQ:...] 不能被当成真实消息元素。
			msg.Segment = []message.IMessageElement{&message.TextElement{Content: msg.Message}}
		}
		msg.segmentSynthesized = true
	}

	if msg.Message == "" && len(msg.Segment) > 0 {
		msg.Message = MessageSegmentsToCompatibilityText(msg.Segment)
	}
}

// platformUsesCQSyntax 报告入站字符串消息使用 CQ 码语法的平台
// （onebot11 系：gocq / onebot / walleq / lagrange）。
// 其余平台（TG、Discord、KOOK 等）的 @ 与富媒体使用各自的原生语法。
func platformUsesCQSyntax(platform string) bool {
	return platform == "QQ"
}

// MessageSegmentsToCompatibilityText renders segments as legacy CQ text.
// redactURLQueries 去掉 URL 查询串（签名/访问令牌等凭据），用于消息日志脱敏。
func redactURLQueries(s string) string {
	if !strings.ContainsRune(s, '?') {
		return s
	}
	return urlWithQueryRe.ReplaceAllString(s, "$1?...")
}

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
		// 与旧 extractResultFromSegments 一致：本地文件引用优先于 URL。
		file := e.File
		if file == "" {
			file = e.URL
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
