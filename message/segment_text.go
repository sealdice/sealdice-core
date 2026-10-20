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
type SegmentText struct {
	Text         string
	Placeholders map[int]IMessageElement
}

const (
	segmentPlaceholderPrefix = "\x1fseg:"
	segmentPlaceholderSuffix = "\x1f"
)

// ProjectSegmentsToText projects segments to text, preserving non-text
// segments in Placeholders.
func ProjectSegmentsToText(segments []IMessageElement) SegmentText {
	var builder strings.Builder
	placeholders := make(map[int]IMessageElement)

	for idx, elem := range segments {
		if text, ok := elem.(*TextElement); ok {
			builder.WriteString(text.Content)
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

// ToSegments rebuilds the original segment list. Unknown placeholders are kept
// as literal text, so user-authored placeholder-looking text is not consumed.
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
		if !strings.HasPrefix(st.Text[i:], segmentPlaceholderPrefix) {
			r, size := utf8.DecodeRuneInString(st.Text[i:])
			text.WriteRune(r)
			i += size
			continue
		}

		start := i + len(segmentPlaceholderPrefix)
		end := strings.Index(st.Text[start:], segmentPlaceholderSuffix)
		if end < 0 {
			r, size := utf8.DecodeRuneInString(st.Text[i:])
			text.WriteRune(r)
			i += size
			continue
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
// Unknown placeholders are kept as-is.
func (st SegmentText) ResolvePlaceholders(text string, resolve func(IMessageElement) string) string {
	if len(st.Placeholders) == 0 || resolve == nil || !strings.Contains(text, segmentPlaceholderPrefix) {
		return text
	}

	var builder strings.Builder
	for i := 0; i < len(text); {
		if !strings.HasPrefix(text[i:], segmentPlaceholderPrefix) {
			r, size := utf8.DecodeRuneInString(text[i:])
			builder.WriteRune(r)
			i += size
			continue
		}

		start := i + len(segmentPlaceholderPrefix)
		end := strings.Index(text[start:], segmentPlaceholderSuffix)
		if end < 0 {
			builder.WriteString(text[i:])
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
