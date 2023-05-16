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
	"regexp"
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
	Text  ElementType = iota // 文本
	At                       // 艾特
	File                     // 文件
	Image                    // 图片
	TTS                      // 文字转语音
	Reply                    // 回复
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

type ReplyElement struct {
	Target string
}

func (t *ReplyElement) Type() ElementType {
	return Reply
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

type ImageElement struct {
	file *FileElement
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

func (d *Dice) toElement(t string, dMap map[string]string) (MessageElement, error) {
	switch t {
	case "file":
		p := strings.TrimSpace(dMap["file"])
		if strings.HasPrefix(p, "http") {
			fmt.Println(p)
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
			contenttype := mime.TypeByExtension(filesuffix)
			if len(contenttype) == 0 {
				contenttype = "application/octet-stream"
			}
			r := &FileElement{
				Stream:      bytes.NewReader(content),
				ContentType: contenttype,
				File:        info.Name(),
			}
			return r, nil
		}
	case "at":
		target := dMap["qq"]
		if dMap["id"] != "" {
			target = dMap["id"]
		}
		return &AtElement{Target: target}, nil
	case "image":
		t = "file"
		f, err := d.toElement(t, dMap)
		if err != nil {
			return nil, err
		}
		file := f.(*FileElement)
		return &ImageElement{file: file}, nil
	case "tts":
		content := dMap["text"]
		return &TTSElement{Content: content}, nil
	case "reply":
		target := dMap["id"]
		return &ReplyElement{Target: target}, nil
	}
	return CQToText(t, dMap), nil
}

func (d *Dice) ConvertStringMessage(raw string) (r []MessageElement) {
	var arg, key string
	dMap := map[string]string{}

	text := ImageRewrite(raw, SealCodeToCqCode)

	saveCQCode := func() {
		elem, err := d.toElement(arg, dMap)
		if err != nil {
			d.Logger.Errorf("转换CQ码时出现错误，将原样发送 <%s>", err.Error())
			r = append(r, CQToText(arg, dMap))
		}
		r = append(r, elem)
	}

	for text != "" {
		i := 0
		for i < len(text) && !(text[i] == '[' && i+4 < len(text) && text[i:i+4] == "[CQ:") {
			i++
		}
		if i > 0 {
			r = append(r, newText(text[:i]))
		}

		if i+4 > len(text) {
			return
		}
		text = text[i+4:]
		i = 0
		for i < len(text) && text[i] != ',' && text[i] != ']' {
			i++
		}
		if i+1 > len(text) {
			return
		}
		arg = text[:i]
		for k := range dMap {
			delete(dMap, k)
		}
		text = text[i:]
		i = 0
		for {
			if text[0] == ']' {
				saveCQCode()
				text = text[1:]
				break
			}
			text = text[1:]

			for i < len(text) && text[i] != '=' {
				i++
			}
			if i+1 > len(text) {
				return
			}
			key = text[:i]
			text = text[i+1:]
			i = 0
			for i < len(text) && text[i] != ',' && text[i] != ']' {
				i++
			}

			if i+1 > len(text) {
				return
			}
			dMap[key] = text[:i]
			text = text[i:]
			i = 0
		}
	}
	return
}

func SealCodeToCqCode(text string) string {
	re := regexp.MustCompile(`\[(img|图|文本|text|语音|voice|视频|video):(.+?)]`) // [img:] 或 [图:]
	m := re.FindStringSubmatch(text)
	if m != nil {
		fn := m[2]
		cqType := "image"
		if m[1] == "voice" || m[1] == "语音" {
			cqType = "record"
		}
		if m[1] == "video" || m[1] == "视频" {
			cqType = "video"
		}

		if strings.HasPrefix(fn, "file://") || strings.HasPrefix(fn, "http://") || strings.HasPrefix(fn, "https://") {
			u, err := url.Parse(fn)
			if err != nil {
				return text
			}
			cq := CQCommand{
				Type: cqType,
				Args: map[string]string{"file": u.String()},
			}
			return cq.Compile()
		}

		afn, err := filepath.Abs(fn)
		if err != nil {
			return text // 不是文件路径，不管
		}
		cwd, _ := os.Getwd()
		if strings.HasPrefix(afn, cwd) {
			if _, err := os.Stat(afn); errors.Is(err, os.ErrNotExist) {
				return "[找不到图片/文件]"
			} else {
				// 这里使用绝对路径，windows上gocqhttp会裁掉一个斜杠，所以我这里加一个
				if runtime.GOOS == `windows` {
					afn = "/" + afn
				}
				u := url.URL{
					Scheme: "file",
					Path:   afn,
				}
				cq := CQCommand{
					Type: cqType,
					Args: map[string]string{"file": u.String()},
				}
				return cq.Compile()
			}
		} else {
			return "[图片/文件指向非当前程序目录，已禁止]"
		}
	}
	return text
}
