package message

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// SegmentText is the text projection of a message segment list.
//
// Text segments are concatenated as-is. Every non-text segment is replaced by
// an invisible placeholder, so command parsing can operate on a string without
// losing images, faces, replies, etc. Placeholders can later be resolved back
// to compatibility text (for example CQ codes) or back to segments.
//
// 字面文本中的 \x1f 以成对转义（\x1f -> \x1f\x1f）写入投影，因此用户输入
// 不可能构成合法的占位符令牌，令牌与字面文本不会混淆。
type SegmentText struct {
	Text         string
	Placeholders map[int]IMessageElement
}

const (
	segmentPlaceholderPrefix = "\x1fseg:"
	segmentPlaceholderSuffix = "\x1f"
)

func escapeSegmentText(s string) string {
	if !strings.ContainsRune(s, '\x1f') {
		return s
	}
	return strings.ReplaceAll(s, "\x1f", "\x1f\x1f")
}

func unescapeSegmentText(s string) string {
	if !strings.Contains(s, "\x1f\x1f") {
		return s
	}
	return strings.ReplaceAll(s, "\x1f\x1f", "\x1f")
}

// ProjectSegmentsToText projects segments to text, preserving non-text
// segments in Placeholders.
func ProjectSegmentsToText(segments []IMessageElement) SegmentText {
	var builder strings.Builder
	placeholders := make(map[int]IMessageElement)

	for idx, elem := range segments {
		if text, ok := elem.(*TextElement); ok {
			builder.WriteString(escapeSegmentText(text.Content))
			continue
		}

		id := idx + 1
		placeholders[id] = elem
		_, _ = fmt.Fprintf(&builder, "%s%d%s", segmentPlaceholderPrefix, id, segmentPlaceholderSuffix)
	}

	if len(placeholders) == 0 {
		placeholders = nil
	}
	return SegmentText{Text: builder.String(), Placeholders: placeholders}
}

// ToSegments rebuilds the original segment list. Literal user text that merely
// looks like a placeholder is preserved as text; unknown placeholders are kept
// as-is.
func (st SegmentText) ToSegments() []IMessageElement {
	if st.Text == "" {
		return nil
	}

	var result []IMessageElement
	var text strings.Builder

	flushText := func() {
		if text.Len() == 0 {
			return
		}
		result = append(result, &TextElement{Content: text.String()})
		text.Reset()
	}

	for i := 0; i < len(st.Text); {
		// 成对的 \x1f 是转义后的字面文本，优先于占位符前缀判断。
		if st.Text[i] == '\x1f' && i+1 < len(st.Text) && st.Text[i+1] == '\x1f' {
			text.WriteByte('\x1f')
			i += 2
			continue
		}

		if !strings.HasPrefix(st.Text[i:], segmentPlaceholderPrefix) {
			r, size := utf8.DecodeRuneInString(st.Text[i:])
			text.WriteRune(r)
			i += size
			continue
		}

		start := i + len(segmentPlaceholderPrefix)
		end := strings.Index(st.Text[start:], segmentPlaceholderSuffix)
		if end < 0 {
			text.WriteString(unescapeSegmentText(st.Text[i:]))
			break
		}
		end += start

		id, err := strconv.Atoi(st.Text[start:end])
		elem, ok := st.Placeholders[id]
		if err != nil || !ok || elem == nil {
			text.WriteString(st.Text[i : end+len(segmentPlaceholderSuffix)])
			i = end + len(segmentPlaceholderSuffix)
			continue
		}

		flushText()
		result = append(result, elem)
		i = end + len(segmentPlaceholderSuffix)
	}
	flushText()

	return result
}

// ResolvePlaceholders replaces known placeholders in text with resolve(elem).
// Literal user text that merely looks like a placeholder is preserved as-is.
func (st SegmentText) ResolvePlaceholders(text string, resolve func(IMessageElement) string) string {
	if resolve == nil {
		return text
	}
	// 没有占位符（或文本中不出现前缀）时，所有 \x1f 都是转义的字面文本。
	if len(st.Placeholders) == 0 || !strings.Contains(text, segmentPlaceholderPrefix) {
		return unescapeSegmentText(text)
	}

	var builder strings.Builder
	for i := 0; i < len(text); {
		// 成对的 \x1f 是转义后的字面文本，优先于占位符前缀判断。
		if text[i] == '\x1f' && i+1 < len(text) && text[i+1] == '\x1f' {
			builder.WriteByte('\x1f')
			i += 2
			continue
		}

		if !strings.HasPrefix(text[i:], segmentPlaceholderPrefix) {
			r, size := utf8.DecodeRuneInString(text[i:])
			builder.WriteRune(r)
			i += size
			continue
		}

		start := i + len(segmentPlaceholderPrefix)
		end := strings.Index(text[start:], segmentPlaceholderSuffix)
		if end < 0 {
			builder.WriteString(unescapeSegmentText(text[i:]))
			break
		}
		end += start

		id, err := strconv.Atoi(text[start:end])
		elem, ok := st.Placeholders[id]
		if err != nil || !ok || elem == nil {
			builder.WriteString(text[i : end+len(segmentPlaceholderSuffix)])
			i = end + len(segmentPlaceholderSuffix)
			continue
		}

		builder.WriteString(resolve(elem))
		i = end + len(segmentPlaceholderSuffix)
	}
	return builder.String()
}
