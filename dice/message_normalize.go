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
			// 入站只做语法解析：协议端负责资源获取，不能在这里 Stat/下载图片。
			msg.Segment = message.ConvertStringMessage(msg.Message, message.WithResourceMode(message.ResourceInbound))
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
		return mediaCompatibilityText("file", e, "")
	case *message.ImageElement:
		return mediaCompatibilityText("image", e.File, e.URL)
	case *message.RecordElement:
		return mediaCompatibilityText("record", e.File, "")
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
		args := map[string]string{}
		for key, value := range gjson.ParseBytes(e.Data).Map() {
			args[key] = value.String()
		}
		return (&message.CQCommand{Type: e.RawType, Args: args}).Compile()
	default:
		return ""
	}
}

// mediaCompatibilityText 还原媒体 CQ 码。优先使用入站保存的原始参数，
// 保证 file/url/subType/cache 等参数不丢失；没有原始参数时按 File → URL
// 的顺序选择可发送引用（与旧行为一致）。
func mediaCompatibilityText(cqType string, file *message.FileElement, fallbackURL string) string {
	if file != nil && len(file.RawParams) > 0 {
		return (&message.CQCommand{Type: cqType, Args: file.RawParams}).Compile()
	}

	args := map[string]string{}
	ref := ""
	if file != nil {
		ref = file.File
		if ref == "" {
			ref = file.URL
		}
	}
	if ref == "" {
		ref = fallbackURL
	}
	if ref != "" {
		args["file"] = ref
	}
	if fallbackURL != "" && fallbackURL != ref {
		args["url"] = fallbackURL
	}
	if len(args) == 0 {
		return ""
	}
	return (&message.CQCommand{Type: cqType, Args: args}).Compile()
}
