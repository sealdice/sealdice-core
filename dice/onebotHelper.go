package dice

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
)

type (
	MessageElement interface {
		Type() ElementType
	}

	ElementType int
)

//go:generate stringer -type ElementType -linecomment
const (
	Text  ElementType = iota // 文本
	Image                    // 图片
	At                       // 艾特
	File                     // 文件
	Voice                    // 语音
	Video                    // 视频
)

type TextElement struct {
	Content string
}

func (t *TextElement) Type() ElementType {
	return Text
}

type ImageElement struct {
	ContentType string
	Stream      io.Reader
	File        string
}

func (l *ImageElement) Type() ElementType {
	return Image
}

func newText(s string) *TextElement {
	return &TextElement{Content: s}
}

func CQToText(t string, d map[string]string) MessageElement {
	org := "[CQ:" + t
	for k, v := range d {
		org += "," + k + "=" + v
	}
	org += "]"
	return newText(org)
}

func (d *Dice) toElement(t string, dMap map[string]string) MessageElement {
	if t == "image" {
		resp, err := http.Get(dMap["file"])
		if err != nil {
			//TODO: logger
			fmt.Println(err)
			return CQToText(t, dMap)
		}
		content, err := io.ReadAll(resp.Body)
		//fmt.Println(string(body))
		//fmt.Println(resp.StatusCode)
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)
		if resp.StatusCode == 200 {
			//fmt.Println("ok")
		} else {
			return CQToText(t, dMap)
		}
		Sha1Inst := sha1.New()
		if err != nil {
			//TODO: logger
			fmt.Println(err)
			return CQToText(t, dMap)
		}
		//fmt.Println("img size", len(content))
		Sha1Inst.Write(content)
		Result := Sha1Inst.Sum([]byte(""))
		//fmt.Printf("%x\n\n", Result)
		r := &ImageElement{
			Stream:      bytes.NewReader(content),
			ContentType: resp.Header.Get("Content-Type"),
			File:        fmt.Sprintf("%x.jpg", Result),
		}
		return r
	}
	return CQToText(t, dMap)
}

func (d *Dice) ConvertStringMessage(raw string) (r []MessageElement) {
	var arg, key string
	dMap := map[string]string{}

	saveCQCode := func() {
		r = append(r, d.toElement(arg, dMap))
	}

	for raw != "" {
		i := 0
		for i < len(raw) && !(raw[i] == '[' && i+4 < len(raw) && raw[i:i+4] == "[CQ:") {
			i++
		}
		if i > 0 {
			r = append(r, newText(raw[:i]))
			//if base.SplitURL {
			//	for _, txt := range param.SplitURL(cqcode.UnescapeText(raw[:i])) {
			//		r = append(r, message.newText(txt))
			//	}
			//} else {
			//	r = append(r, message.newText(cqcode.UnescapeText(raw[:i])))
			//}
		}

		if i+4 > len(raw) {
			return
		}
		raw = raw[i+4:] // skip "[CQ:"
		i = 0
		for i < len(raw) && raw[i] != ',' && raw[i] != ']' {
			i++
		}
		if i+1 > len(raw) {
			return
		}
		arg = raw[:i]
		for k := range dMap { // clear the map, reuse it
			delete(dMap, k)
		}
		raw = raw[i:]
		i = 0
		for {
			if raw[0] == ']' {
				saveCQCode()
				raw = raw[1:]
				break
			}
			raw = raw[1:]

			for i < len(raw) && raw[i] != '=' {
				i++
			}
			if i+1 > len(raw) {
				return
			}
			key = raw[:i]
			raw = raw[i+1:] // skip "="
			i = 0
			for i < len(raw) && raw[i] != ',' && raw[i] != ']' {
				i++
			}

			if i+1 > len(raw) {
				return
			}
			dMap[key] = raw[:i]
			raw = raw[i:]
			i = 0
		}
	}
	return
}
