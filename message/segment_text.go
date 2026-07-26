package message

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

const segmentPlaceholderPrefix = "\x1fseg:"
const segmentPlaceholderSuffix = "\x1f"

type SegmentText struct {
	Text         string
	Placeholders map[int]IMessageElement
}

func ProjectSegmentsToText(segments []IMessageElement) SegmentText {
	var builder strings.Builder
	placeholders := map[int]IMessageElement{}
	for idx, elem := range segments {
		if text, ok := elem.(*TextElement); ok {
			builder.WriteString(text.Content)
			continue
		}
		placeholderID := idx + 1
		placeholders[placeholderID] = elem
		builder.WriteString(fmt.Sprintf("%s%d%s", segmentPlaceholderPrefix, placeholderID, segmentPlaceholderSuffix))
	}
	if len(placeholders) == 0 {
		placeholders = nil
	}
	return SegmentText{Text: builder.String(), Placeholders: placeholders}
}

func (st SegmentText) ToSegments() []IMessageElement {
	if st.Text == "" {
		return nil
	}
	var result []IMessageElement
	var builder strings.Builder
	for i := 0; i < len(st.Text); {
		if !strings.HasPrefix(st.Text[i:], segmentPlaceholderPrefix) {
			r, size := utf8.DecodeRuneInString(st.Text[i:])
			builder.WriteRune(r)
			i += size
			continue
		}
		start := i + len(segmentPlaceholderPrefix)
		end := strings.Index(st.Text[start:], segmentPlaceholderSuffix)
		if end < 0 {
			r, size := utf8.DecodeRuneInString(st.Text[i:])
			builder.WriteRune(r)
			i += size
			continue
		}
		end += start
		id, err := strconv.Atoi(st.Text[start:end])
		elem, ok := st.Placeholders[id]
		if err != nil || !ok || elem == nil {
			builder.WriteString(st.Text[i : end+len(segmentPlaceholderSuffix)])
			i = end + len(segmentPlaceholderSuffix)
			continue
		}
		if builder.Len() > 0 {
			result = append(result, &TextElement{Content: builder.String()})
			builder.Reset()
		}
		result = append(result, elem)
		i = end + len(segmentPlaceholderSuffix)
	}
	if builder.Len() > 0 {
		result = append(result, &TextElement{Content: builder.String()})
	}
	return result
}
