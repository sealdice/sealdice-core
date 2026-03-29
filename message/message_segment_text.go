package message

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// SegmentText 表示带占位符的文本和占位符映射，用于将 segment 投影到文本视图。
type SegmentText struct {
	Text         string
	Placeholders map[int]IMessageElement
}

// ToSegmentText 将消息元素转换为带占位符的文本表示。
func ToSegmentText(segments []IMessageElement) SegmentText {
	var placeholders map[int]IMessageElement
	var builder strings.Builder
	for idx, elem := range segments {
		if textElem, ok := elem.(*TextElement); ok {
			builder.WriteString(textElem.Content)
			continue
		}
		placeholderIndex := idx + 1
		if placeholders == nil {
			placeholders = make(map[int]IMessageElement)
		}
		placeholders[placeholderIndex] = elem
		builder.WriteByte('$')
		builder.WriteString(strconv.Itoa(placeholderIndex))
	}
	return SegmentText{
		Text:         builder.String(),
		Placeholders: placeholders,
	}
}

// ToMessageElements 根据占位符映射还原消息元素切片。
func (st SegmentText) ToMessageElements() []IMessageElement {
	if st.Text == "" {
		return nil
	}
	var result []IMessageElement
	var builder strings.Builder
	for i := 0; i < len(st.Text); {
		if st.Text[i] != '$' {
			r, size := utf8.DecodeRuneInString(st.Text[i:])
			builder.WriteRune(r)
			i += size
			continue
		}
		if i+1 >= len(st.Text) || st.Text[i+1] < '0' || st.Text[i+1] > '9' {
			builder.WriteByte('$')
			i++
			continue
		}
		j := i + 1
		for j < len(st.Text) && st.Text[j] >= '0' && st.Text[j] <= '9' {
			j++
		}
		idxStr := st.Text[i+1 : j]
		placeholderIndex, err := strconv.Atoi(idxStr)
		if err != nil {
			builder.WriteByte('$')
			builder.WriteString(idxStr)
			i = j
			continue
		}
		elem, ok := st.Placeholders[placeholderIndex]
		if !ok || elem == nil {
			builder.WriteByte('$')
			builder.WriteString(idxStr)
			i = j
			continue
		}
		if builder.Len() > 0 {
			result = append(result, &TextElement{Content: builder.String()})
			builder.Reset()
		}
		result = append(result, elem)
		i = j
	}
	if builder.Len() > 0 {
		result = append(result, &TextElement{Content: builder.String()})
	}
	return result
}

// SegmentsToText 返回用于命令解析/兼容文本的 segment 文本投影。
func SegmentsToText(segments []IMessageElement) string {
	return ToSegmentText(segments).Text
}

// ParseSegmentText 根据文本和占位符映射重建 segment。
func ParseSegmentText(text string, placeholders map[int]IMessageElement) []IMessageElement {
	return SegmentText{
		Text:         text,
		Placeholders: placeholders,
	}.ToMessageElements()
}

// SegmentsToLegacyCQText 将内部消息元素转换为历史 CQ 文本视图，用于兼容旧命令解析流程。
func SegmentsToLegacyCQText(segments []IMessageElement) string {
	var cqMessage strings.Builder
	var foundFirstText bool
	for _, v := range segments {
		if v.Type() == Text {
			foundFirstText = true
		}
		if !foundFirstText {
			continue
		}
		switch v.Type() {
		case At:
			// 旧流程在命令解析阶段单独处理 @，这里保持一致。
			continue
		case Text:
			res, ok := v.(*TextElement)
			if ok {
				cqMessage.WriteString(res.Content)
			}
		case Face:
			res, ok := v.(*FaceElement)
			if ok {
				_, _ = fmt.Fprintf(&cqMessage, "[CQ:face,id=%v]", res.FaceID)
			}
		case File:
			res, ok := v.(*FileElement)
			if !ok {
				continue
			}
			fileVal := res.File
			if fileVal == "" {
				fileVal = res.URL
			}
			if fileVal != "" {
				_, _ = fmt.Fprintf(&cqMessage, "[CQ:file,file=%v]", fileVal)
			}
		case Image:
			res, ok := v.(*ImageElement)
			if !ok {
				continue
			}
			urlVal := res.URL
			if urlVal == "" && res.File != nil {
				urlVal = res.File.URL
				if urlVal == "" {
					urlVal = res.File.File
				}
			}
			if urlVal != "" {
				_, _ = fmt.Fprintf(&cqMessage, "[CQ:image,file=%v]", urlVal)
			}
		case Record:
			res, ok := v.(*RecordElement)
			if !ok {
				continue
			}
			var recordFile string
			if res.File != nil {
				recordFile = res.File.URL
				if recordFile == "" {
					recordFile = res.File.File
				}
			}
			if recordFile != "" {
				_, _ = fmt.Fprintf(&cqMessage, "[CQ:record,file=%v]", recordFile)
			}
		case Reply:
			res, ok := v.(*ReplyElement)
			if !ok {
				continue
			}
			parseInt, err := strconv.Atoi(res.ReplySeq)
			if err != nil {
				continue
			}
			_, _ = fmt.Fprintf(&cqMessage, "[CQ:reply,id=%v]", parseInt)
		case TTS:
			res, ok := v.(*TTSElement)
			if ok {
				_, _ = fmt.Fprintf(&cqMessage, "[CQ:tts,text=%v]", res.Content)
			}
		case Poke:
			res, ok := v.(*PokeElement)
			if ok {
				_, _ = fmt.Fprintf(&cqMessage, "[CQ:poke,qq=%v]", res.Target)
			}
		default:
			res, ok := v.(*DefaultElement)
			if !ok {
				continue
			}
			dMap := map[string]interface{}{}
			if len(res.Data) > 0 {
				_ = json.Unmarshal(res.Data, &dMap)
			}
			var cqParamParts []string
			for paramStr, paramValue := range dMap {
				cqParamParts = append(cqParamParts, fmt.Sprintf("%s=%v", paramStr, paramValue))
			}
			cqParam := strings.Join(cqParamParts, ",")
			if cqParam == "" {
				_, _ = fmt.Fprintf(&cqMessage, "[CQ:%s]", res.RawType)
			} else {
				_, _ = fmt.Fprintf(&cqMessage, "[CQ:%s,%s]", res.RawType, cqParam)
			}
		}
	}
	return cqMessage.String()
}
