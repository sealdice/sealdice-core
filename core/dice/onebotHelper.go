package dice

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
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

const maxFileSize = 1024 * 1024 * 50 // 50MB

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

func (d *Dice) toElement(t string, dMap map[string]string) (MessageElement, error) {
	switch t {
	case "file":
		p := dMap["file"]
		if strings.HasPrefix(p, "http") {
			resp, err := http.Get(p)
			if err != nil {
				return nil, err
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
				return nil, errors.New("http get failed")
			}
			Sha1Inst := sha1.New()
			filetype, _ := mime.ExtensionsByType(resp.Header.Get("Content-Type"))
			var suffix string
			if filetype != nil {
				suffix = filetype[len(filetype)-1]
			}
			//fmt.Println(filetype)
			if err != nil {
				return nil, err
			}
			//fmt.Println("img size", len(content))
			Sha1Inst.Write(content)
			Result := Sha1Inst.Sum([]byte(""))
			//fmt.Printf("%x\n\n", Result)
			r := &FileElement{
				Stream:      bytes.NewReader(content),
				ContentType: resp.Header.Get("Content-Type"),
				File:        fmt.Sprintf("%x%s", Result, suffix),
			}
			return r, nil
		} else {
			fu, err := url.Parse(p)
			if err != nil {
				return nil, err
			}
			if runtime.GOOS == `windows` && strings.HasPrefix(fu.Path, "/") {
				fu.Path = fu.Path[1:]
			}
			info, err := os.Stat(fu.Path)
			if err != nil {
				return nil, err
			}
			if info.Size() == 0 || info.Size() >= maxFileSize {
				return nil, errors.New("invalid file size")
			}
			afn, err := filepath.Abs(fu.Path)
			if err != nil {
				return nil, err // 不是文件路径，不管
			}
			cwd, _ := os.Getwd()
			if !strings.HasPrefix(afn, cwd) {
				return nil, errors.New("restricted file path")
			}
			filesuffix := path.Ext(fu.Path)
			content, err := os.ReadFile(fu.Path)
			if err != nil {
				return nil, err
			}
			Sha1Inst := sha1.New()
			Sha1Inst.Write(content)
			Result := Sha1Inst.Sum([]byte(""))
			contenttype := mime.TypeByExtension(filesuffix)
			if len(contenttype) == 0 {
				contenttype = "application/octet-stream"
			}
			r := &FileElement{
				Stream:      bytes.NewReader(content),
				ContentType: contenttype,
				File:        fmt.Sprintf("%x%s", Result, filesuffix),
			}
			return r, nil
		}
	case "at":
		target := dMap["qq"]
		return &AtElement{Target: target}, nil
	case "image":
		t = "file"
		return d.toElement(t, dMap)
	case "tts":
		content := dMap["text"]
		return &TTSElement{Content: content}, nil
	}
	return CQToText(t, dMap), nil
}

func (d *Dice) ConvertStringMessage(raw string) (r []MessageElement) {
	var arg, key string
	dMap := map[string]string{}

	saveCQCode := func() {
		elem, err := d.toElement(arg, dMap)
		if err != nil {
			d.Logger.Errorf("转换CQ码时出现错误，将原样发送 <%s>", err.Error())
			r = append(r, CQToText(arg, dMap))
		}
		r = append(r, elem)
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
