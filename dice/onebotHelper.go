package dice

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
)

type (
	MessageElement interface {
		Type() ElementType
	}

	ElementType int
)

const (
	Text ElementType = iota // 文本
	At                      // 艾特
	File                    // 文件
	TTS                     // 文字转语音
)

type TextElement struct {
	Content string
}

func (t *TextElement) Type() ElementType {
	return Text
}

type AtElement struct {
	Target string
}

func (t *AtElement) Type() ElementType {
	return At
}

type TTSElement struct {
	Content string
}

func (t *TTSElement) Type() ElementType {
	return TTS
}

type FileElement struct {
	ContentType string
	Stream      io.Reader
	File        string
}

func (l *FileElement) Type() ElementType {
	return File
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
	switch t {
	case "file":
		url := dMap["file"]
		if strings.HasPrefix(url, "http") {
			resp, err := http.Get(url)
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
			filetype, _ := mime.ExtensionsByType(resp.Header.Get("Content-Type"))
			var postfix string
			if filetype != nil {
				postfix = filetype[len(filetype)-1]
			}
			fmt.Println(filetype)
			if err != nil {
				//TODO: logger
				fmt.Println(err)
				return CQToText(t, dMap)
			}
			//fmt.Println("img size", len(content))
			Sha1Inst.Write(content)
			Result := Sha1Inst.Sum([]byte(""))
			//fmt.Printf("%x\n\n", Result)
			r := &FileElement{
				Stream:      bytes.NewReader(content),
				ContentType: resp.Header.Get("Content-Type"),
				File:        fmt.Sprintf("%x%s", Result, postfix),
			}
			return r
		}
	case "at":
		target := dMap["qq"]
		return &AtElement{Target: target}
	case "image":
		t = "file"
		return d.toElement(t, dMap)
	case "tts":
		content := dMap["text"]
		return &TTSElement{Content: content}
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
		}

		if i+4 > len(raw) {
			return
		}
		raw = raw[i+4:]
		i = 0
		for i < len(raw) && raw[i] != ',' && raw[i] != ']' {
			i++
		}
		if i+1 > len(raw) {
			return
		}
		arg = raw[:i]
		for k := range dMap {
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
			raw = raw[i+1:]
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
