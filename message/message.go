package message

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
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

	"github.com/bytedance/sonic"
)

type CQCommand struct {
	Type      string
	Args      map[string]string
	Overwrite string
}

func (c *CQCommand) Compile() string {
	if c.Overwrite != "" {
		return c.Overwrite
	}
	argsPart := ""
	for k, v := range c.Args {
		argsPart += fmt.Sprintf(",%s=%s", k, v)
	}
	return fmt.Sprintf("[CQ:%s%s]", c.Type, argsPart)
}

type (
	IMessageElement interface {
		Type() ElementType
		FromCQData(dMap map[string]string) error
	}

	ElementType int
)

const (
	Text    ElementType = iota // 文本
	At                         // 艾特
	File                       // 文件
	Image                      // 图片
	TTS                        // 文字转语音
	Reply                      // 回复
	Record                     // 语音
	Face                       // 表情
	Poke                       // 戳一戳
	Default = -1               // 一个兜底的情况，兜底所有不认识的类型
)

const maxFileSize = 1024 * 1024 * 50 // 50MB

// ElementFactory 创建 IMessageElement 实例的工厂函数
type ElementFactory func() IMessageElement

// elementRegistry 全局注册表，存储类型名到工厂函数的映射
var elementRegistry = map[string]ElementFactory{
	"at":     func() IMessageElement { return &AtElement{} },
	"tts":    func() IMessageElement { return &TTSElement{} },
	"reply":  func() IMessageElement { return &ReplyElement{} },
	"poke":   func() IMessageElement { return &PokeElement{} },
	"face":   func() IMessageElement { return &FaceElement{} },
	"file":   func() IMessageElement { return &FileElement{} },
	"image":  func() IMessageElement { return &ImageElement{} },
	"record": func() IMessageElement { return &RecordElement{} },
}

// GetElementFactory 获取指定类型的元素工厂函数
func GetElementFactory(elementType string) ElementFactory {
	factory, exists := elementRegistry[elementType]
	if !exists {
		// 构建成默认类型
		return func() IMessageElement {
			return &DefaultElement{RawType: elementType}
		}
	}
	return factory
}

type DefaultElement struct {
	RawType string                 `jsbind:"type"`
	Data    sonic.NoCopyRawMessage `jsbind:"data"`
}

func (t *DefaultElement) Type() ElementType {
	return Default
}

func (t *DefaultElement) FromCQData(dMap map[string]string) error {
	marshal, err := sonic.Marshal(dMap)
	if err != nil {
		return err
	}
	t.Data = marshal
	return nil
}

type TextElement struct {
	Content string `jsbind:"content"`
}

func (t *TextElement) Type() ElementType {
	return Text
}

func (t *TextElement) FromCQData(dMap map[string]string) error {
	// TextElement 不从 CQ 码创建，这个方法不应该被调用
	return errors.New("TextElement should not be created from CQ data")
}

type AtElement struct {
	Target string `jsbind:"target"`
}

func (t *AtElement) Type() ElementType {
	return At
}

func (t *AtElement) FromCQData(dMap map[string]string) error {
	target := dMap["qq"]
	if dMap["id"] != "" {
		target = dMap["id"]
	}
	t.Target = target
	return nil
}

type ReplyElement struct {
	ReplySeq string            `jsbind:"replySeq"` // 回复的目标消息ID
	Sender   string            `jsbind:"sender"`   // 回复的目标消息发送者ID
	GroupID  string            `jsbind:"groupID"`  // 回复群聊消息时的群号
	Elements []IMessageElement `jsbind:"elements"` // 回复的消息内容
}

func (t *ReplyElement) Type() ElementType {
	return Reply
}

func (t *ReplyElement) FromCQData(dMap map[string]string) error {
	t.ReplySeq = dMap["id"]
	return nil
}

type TTSElement struct {
	Content string `jsbind:"content"`
}

func (t *TTSElement) Type() ElementType {
	return TTS
}

func (t *TTSElement) FromCQData(dMap map[string]string) error {
	t.Content = dMap["text"]
	return nil
}

type FileElement struct {
	ContentType string `jsbind:"contentType"`
	Stream      io.Reader
	File        string `jsbind:"file"`
	URL         string `jsbind:"url"`
}

func (l *FileElement) Type() ElementType {
	return File
}

func (l *FileElement) FromCQData(dMap map[string]string) error {
	p := strings.TrimSpace(dMap["file"])
	u := strings.TrimSpace(dMap["url"])
	if u == "" {
		fileElem, err := FilepathToFileElement(p)
		if err != nil {
			return err
		}
		*l = *fileElem
		return nil
	} else {
		// 当 url 不为空时，绕过读取直接发送 url
		l.URL = u
		return nil
	}
}

type ImageElement struct {
	File *FileElement `jsbind:"file"`
	URL  string       `jsbind:"url"`
}

func (l *ImageElement) Type() ElementType {
	return Image
}

func (l *ImageElement) FromCQData(dMap map[string]string) error {
	fileElem := &FileElement{}
	err := fileElem.FromCQData(dMap)
	if err != nil {
		return err
	}
	l.File = fileElem
	l.URL = fileElem.URL
	return nil
}

type RecordElement struct {
	File *FileElement `jsbind:"file"`
}

func (r *RecordElement) Type() ElementType {
	return Record
}

func (r *RecordElement) FromCQData(dMap map[string]string) error {
	fileElem := &FileElement{}
	err := fileElem.FromCQData(dMap)
	if err != nil {
		return err
	}
	r.File = fileElem
	return nil
}

type FaceElement struct {
	FaceID string `jsbind:"faceID"`
}

func (f *FaceElement) Type() ElementType {
	return Face
}

func (f *FaceElement) FromCQData(dMap map[string]string) error {
	f.FaceID = dMap["id"]
	return nil
}

type PokeElement struct {
	Target string `jsbind:"target"` // 戳一戳的目标ID
}

func (p *PokeElement) Type() ElementType {
	return Poke
}

func (p *PokeElement) FromCQData(dMap map[string]string) error {
	p.Target = dMap["qq"]
	return nil
}

func newText(s string) *TextElement {
	return &TextElement{Content: s}
}

func CQToText(t string, d map[string]string) IMessageElement {
	org := "[CQ:" + t
	for k, v := range d {
		org += "," + k + "=" + v
	}
	org += "]"
	return newText(org)
}
func getFileName(header http.Header) string {
	contentDisposition := header.Get("Content-Disposition")
	if contentDisposition == "" {
		contentType := header.Get("Content-Type")
		if contentType == "" {
			return calculateMD5(header)
		}
		filetype, err := mime.ExtensionsByType(contentType)
		if err != nil {
			return calculateMD5(header)
		}
		var suffix string
		if len(filetype) != 0 {
			suffix = filetype[len(filetype)-1]
			return calculateMD5(header) + suffix
		}
		return calculateMD5(header)
	}
	return regexp.MustCompile(`filename=(.+)`).FindStringSubmatch(strings.Split(contentDisposition, ";")[1])[1]
}

func calculateMD5(header http.Header) string {
	hash := md5.New() //nolint:gosec
	for _, value := range header["Content-Type"] {
		hash.Write([]byte(value))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// ExtractLocalTempFile 按路径提取临时文件，路径可以是 http/base64/本地路径
func ExtractLocalTempFile(path string) (string, string, error) {
	fileElement, err := FilepathToFileElement(path)
	if err != nil {
		return "", "", err
	}
	temp, err := os.CreateTemp("", "temp-")
	defer func(temp *os.File) {
		_ = temp.Close()
	}(temp)
	if err != nil {
		return "", "", err
	}
	data, err := io.ReadAll(fileElement.Stream)
	if err != nil {
		return "", "", err
	}
	_, err = temp.Write(data)
	if err != nil {
		return "", "", err
	}
	return fileElement.File, temp.Name(), nil
}

func FilepathToFileElement(fp string) (*FileElement, error) {
	if strings.HasPrefix(fp, "http") {
		resp, err := http.Get(fp) //nolint:gosec
		if err != nil {
			return nil, err
		}
		header := resp.Header
		content, err := io.ReadAll(resp.Body)
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			return nil, errors.New("http get failed")
		}
		filename := getFileName(header)
		if err != nil {
			return nil, err
		}
		r := &FileElement{
			Stream:      bytes.NewReader(content),
			ContentType: resp.Header.Get("Content-Type"),
			File:        filename,
			URL:         fp,
		}
		return r, nil
	} else if strings.HasPrefix(fp, "base64://") {
		content, err := base64.StdEncoding.DecodeString(fp[9:])
		if err != nil {
			return nil, err
		}
		sha1Inst := sha1.New() //nolint:gosec
		filetype, _ := mime.ExtensionsByType(http.DetectContentType(content))
		var suffix string
		if filetype != nil {
			suffix = filetype[len(filetype)-1]
		}
		sha1Inst.Write(content)
		result := sha1Inst.Sum([]byte(""))
		r := &FileElement{
			Stream:      bytes.NewReader(content),
			ContentType: http.DetectContentType(content),
			File:        fmt.Sprintf("%x%s", result, suffix),
			URL:         fp,
		}
		return r, nil
	} else {
		fu, err := url.Parse(fp)
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
		if !strings.HasPrefix(afn, cwd) && !strings.HasPrefix(afn, os.TempDir()) {
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
			URL:         "file://" + afn,
		}
		return r, nil
	}
}

func toElement(t string, dMap map[string]string) (IMessageElement, error) {
	// 从注册表查找工厂函数
	elemFactory := GetElementFactory(t)
	elem := elemFactory()
	// 调用实例的转换方法
	err := elem.FromCQData(dMap)
	if err != nil {
		return nil, err
	}

	return elem, nil
}

func ImageRewrite(longText string, solve func(text string) string) string {
	re := regexp.MustCompile(`\[(img|图|文本|text|语音|voice|视频|video):(.+?)]`) // [img:] 或 [图:]
	m := re.FindAllStringIndex(longText, -1)

	newText := longText
	for i := len(m) - 1; i >= 0; i-- {
		p := m[i]
		text := solve(longText[p[0]:p[1]])
		newText = newText[:p[0]] + text + newText[p[1]:]
	}

	return newText
}

func SealCodeToCqCode(text string) string {
	text = strings.ReplaceAll(text, " ", "")
	re := regexp.MustCompile(`\[(img|图|文本|text|语音|voice|视频|video):(.+?)]`) // [img:] 或 [图:]
	m := re.FindStringSubmatch(text)
	if len(m) == 0 {
		return text
	}

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
		}
		// 这里使用绝对路径，windows上gocqhttp会裁掉一个斜杠，所以我这里加一个
		if runtime.GOOS == `windows` {
			afn = "/" + afn
		}
		u := url.URL{
			Scheme: "file",
			Path:   filepath.ToSlash(afn),
		}
		cq := CQCommand{
			Type: cqType,
			Args: map[string]string{"file": u.String()},
		}
		return cq.Compile()
	}
	return "[图片/文件指向非当前程序目录，已禁止]"
}

func ConvertStringMessage(raw string) (r []IMessageElement) {
	var arg, key string
	dMap := map[string]string{}

	text := ImageRewrite(raw, SealCodeToCqCode)

	saveCQCode := func() {
		elem, err := toElement(arg, dMap)
		if err != nil {
			// d.Logger.Errorf("转换CQ码时出现错误，将原样发送 <%s>", err.Error())
			r = append(r, CQToText(arg, dMap))
			return
		}
		r = append(r, elem)
	}

	for text != "" {
		i := 0
		for i < len(text) && (text[i] != '[' || i+4 >= len(text) || text[i:i+4] != "[CQ:") {
			i++
		}
		if i > 0 {
			r = append(r, newText(text[:i]))
		}

		if i+4 > len(text) {
			return r
		}
		text = text[i+4:]
		i = 0
		for i < len(text) && text[i] != ',' && text[i] != ']' {
			i++
		}
		if i+1 > len(text) {
			return r
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
				return r
			}
			key = text[:i]
			text = text[i+1:]
			i = 0
			for i < len(text) && text[i] != ',' && text[i] != ']' {
				i++
			}

			if i+1 > len(text) {
				return r
			}
			dMap[key] = text[:i]
			text = text[i:]
			i = 0
		}
	}
	return r
}
