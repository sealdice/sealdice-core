package message

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
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
	"go.uber.org/zap"
)

// CQFileErrorKind 定义CQ码资源错误类型
type CQFileErrorKind int

const (
	CQFileErrInvalidURL  CQFileErrorKind = iota + 1 // URL格式错误
	CQFileErrUnavailable                            // 资源不可用（网络错误、404等）
	CQFileErrRestricted                             // 路径受限
	CQFileErrInvalidSize                            // 文件大小无效
)

// CQFileError CQ码资源处理错误
type CQFileError struct {
	Kind       CQFileErrorKind
	Raw        string // 原始路径/URL
	Normalized string // 规范化后的路径/URL
	StatusCode int    // HTTP状态码（仅HTTP请求时有效）
	Cause      error  // 底层错误
}

func (e *CQFileError) Error() string {
	switch e.Kind {
	case CQFileErrInvalidURL:
		return fmt.Sprintf("CQ码资源URL格式错误: %s", e.Raw)
	case CQFileErrUnavailable:
		if e.StatusCode > 0 {
			return fmt.Sprintf("CQ码资源不可用(HTTP %d): %s", e.StatusCode, e.Raw)
		}
		return fmt.Sprintf("CQ码资源不可用: %s", e.Raw)
	case CQFileErrRestricted:
		return fmt.Sprintf("CQ码资源路径受限: %s", e.Raw)
	case CQFileErrInvalidSize:
		return fmt.Sprintf("CQ码资源文件大小无效: %s", e.Raw)
	default:
		return fmt.Sprintf("CQ码资源错误: %s", e.Raw)
	}
}

func (e *CQFileError) Unwrap() error {
	return e.Cause
}

type CQCommand struct {
	Type      string
	Args      map[string]string
	Overwrite string
}

func EscapeCQParam(v string) string {
	safeV := strings.ReplaceAll(v, "&", "&amp;")
	safeV = strings.ReplaceAll(safeV, "[", "&#91;")
	safeV = strings.ReplaceAll(safeV, "]", "&#93;")
	safeV = strings.ReplaceAll(safeV, ",", "&#44;")
	return safeV
}

// UnescapeCQParam 还原 EscapeCQParam 的转义，用于解析 CQ 码参数值。
// 还原顺序需与转义顺序相反：其它占位符先还原，最后才还原 &amp;，避免二次解码。
func UnescapeCQParam(v string) string {
	safeV := strings.ReplaceAll(v, "&#44;", ",")
	safeV = strings.ReplaceAll(safeV, "&#93;", "]")
	safeV = strings.ReplaceAll(safeV, "&#91;", "[")
	safeV = strings.ReplaceAll(safeV, "&amp;", "&")
	return safeV
}

func (c *CQCommand) Compile() string {
	if c.Overwrite != "" {
		return c.Overwrite
	}
	var argsPart strings.Builder
	for k, v := range c.Args {
		fmt.Fprintf(&argsPart, ",%s=%s", k, EscapeCQParam(v))
	}
	return fmt.Sprintf("[CQ:%s%s]", c.Type, argsPart.String())
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
	// 未注册的 CQ 资源类型也可能携带本地文件，发送前统一规范化其路径。
	if rawFile, exists := dMap["file"]; exists && strings.TrimSpace(dMap["url"]) == "" {
		normalized, err := NormalizeLocalResourcePath(rawFile)
		if err != nil {
			return err
		}
		if normalized != rawFile {
			cloned := make(map[string]string, len(dMap))
			for key, value := range dMap {
				cloned[key] = value
			}
			cloned["file"] = normalized
			dMap = cloned
		}
	}

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
	Target  string `jsbind:"target"`
	IsRobot bool   `jsbind:"isRobot"`
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
	var org strings.Builder
	org.WriteString("[CQ:")
	org.WriteString(t)
	for k, v := range d {
		org.WriteString(",")
		org.WriteString(k)
		org.WriteString("=")
		org.WriteString(v)
	}
	org.WriteString("]")
	return newText(org.String())
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

func normalizeRemoteURL(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		parsed, err = url.Parse(strings.ReplaceAll(raw, " ", "%20"))
		if err != nil {
			return "", err
		}
	}
	if parsed.Host == "" {
		return "", errors.New("missing host")
	}
	if parsed.Path != "" {
		unescapedPath, err := url.PathUnescape(parsed.Path)
		if err != nil {
			return "", err
		}
		parsed.Path = unescapedPath
	}
	if parsed.RawQuery != "" {
		if q, err := url.ParseQuery(parsed.RawQuery); err == nil {
			parsed.RawQuery = q.Encode()
		}
	}
	return parsed.String(), nil
}

func isPassthroughResourceReference(raw string) bool {
	lower := strings.ToLower(raw)
	return strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "base64://")
}

func localPathFromResourceReference(raw string) (string, error) {
	if !strings.HasPrefix(strings.ToLower(raw), "file://") {
		return raw, nil
	}

	parsed, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(parsed.Scheme, "file") {
		return "", &CQFileError{Kind: CQFileErrInvalidURL, Raw: raw, Cause: err}
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", &CQFileError{Kind: CQFileErrInvalidURL, Raw: raw, Cause: errors.New("file URL cannot contain query or fragment")}
	}

	localPath := parsed.Path
	if parsed.Host != "" && !strings.EqualFold(parsed.Host, "localhost") {
		localPath = "//" + parsed.Host + parsed.Path
	}
	// net/url 已完成路径解码；Windows 盘符还需要去掉 URL 路径的前导斜杠。
	if runtime.GOOS == "windows" && len(localPath) >= 3 && localPath[0] == '/' && localPath[2] == ':' {
		localPath = localPath[1:]
	}
	return filepath.FromSlash(localPath), nil
}

func pathWithinRoot(candidate, root string) bool {
	rel, err := filepath.Rel(root, candidate)
	if err != nil || filepath.IsAbs(rel) {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func localPathToFileURL(absolutePath string) string {
	fileURLPath := filepath.ToSlash(absolutePath)
	// Windows 盘符必须写成 /C:/...，url.URL 才会生成标准的 file:///C:/...。
	if runtime.GOOS == "windows" && !strings.HasPrefix(fileURLPath, "/") {
		fileURLPath = "/" + fileURLPath
	}
	return (&url.URL{Scheme: "file", Path: fileURLPath}).String()
}

func normalizeLocalFileReference(raw string) (string, string, os.FileInfo, error) {
	localPath, err := localPathFromResourceReference(raw)
	if err != nil {
		return "", "", nil, err
	}

	absolutePath, err := filepath.Abs(localPath)
	if err != nil {
		return "", "", nil, &CQFileError{Kind: CQFileErrInvalidURL, Raw: raw, Normalized: localPath, Cause: err}
	}
	absolutePath = filepath.Clean(absolutePath)

	// 本地资源仍只允许来自程序目录或系统临时目录。filepath.Rel 能正确区分
	// 同名前缀的相邻目录，避免简单字符串前缀判断造成越界。
	cwd, cwdErr := os.Getwd()
	tempDir, tempErr := filepath.Abs(os.TempDir())
	allowed := cwdErr == nil && pathWithinRoot(absolutePath, filepath.Clean(cwd))
	allowed = allowed || tempErr == nil && pathWithinRoot(absolutePath, filepath.Clean(tempDir))
	if !allowed {
		return "", "", nil, &CQFileError{Kind: CQFileErrRestricted, Raw: raw, Normalized: absolutePath}
	}

	info, err := os.Stat(absolutePath) //nolint:gosec
	if err != nil {
		return "", "", nil, &CQFileError{Kind: CQFileErrUnavailable, Raw: raw, Normalized: absolutePath, Cause: err}
	}
	return localPathToFileURL(absolutePath), absolutePath, info, nil
}

// NormalizeLocalResourcePath 将本地资源引用规范化为绝对 file URL。
// 此过程只检查路径和文件状态，不读取文件内容，也不会转换为 Base64；远程 URL、
// Base64 数据及适配器使用的不透明资源 ID 均保持原样。
func NormalizeLocalResourcePath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || isPassthroughResourceReference(raw) {
		return raw, nil
	}
	if strings.Contains(raw, "://") && !strings.HasPrefix(strings.ToLower(raw), "file://") {
		return raw, nil
	}

	isLocal := strings.HasPrefix(strings.ToLower(raw), "file://")
	if !isLocal {
		_, err := os.Stat(raw)
		isLocal = err == nil
	}
	if !isLocal {
		return raw, nil
	}

	normalized, _, _, err := normalizeLocalFileReference(raw)
	return normalized, err
}

func FilepathToFileElement(fp string) (*FileElement, error) {
	fp = strings.TrimSpace(fp)

	if strings.HasPrefix(fp, "http://") || strings.HasPrefix(fp, "https://") {
		normalizedURL, err := normalizeRemoteURL(fp)
		if err != nil {
			return nil, &CQFileError{Kind: CQFileErrInvalidURL, Raw: fp, Cause: err}
		}
		fileName := ""
		if u, errr := url.Parse(normalizedURL); errr == nil {
			fileName = path.Base(u.Path)
			if fileName == "." || fileName == "/" {
				fileName = ""
			}
		}
		resp, err := http.Get(normalizedURL) //nolint:gosec
		defer func() {
			_ = resp.Body.Close()
		}()
		if err != nil {
			return nil, &CQFileError{Kind: CQFileErrUnavailable, Raw: fp, Normalized: normalizedURL, Cause: err}
		}
		if resp.StatusCode != http.StatusOK {
			return nil, &CQFileError{Kind: CQFileErrUnavailable, Raw: fp, Normalized: normalizedURL, StatusCode: resp.StatusCode}
		}
		// 限制从响应体中读取的数据量，以避免过度的内存使用。
		limitedBody := io.LimitReader(resp.Body, maxFileSize+100) // 允许比 maxFileSize 大一点的读取，以便检测超过限制的情况
		content, err := io.ReadAll(limitedBody)
		if err != nil {
			return nil, &CQFileError{Kind: CQFileErrUnavailable, Raw: fp, Normalized: normalizedURL, Cause: err}
		}
		if int64(len(content)) == 0 || int64(len(content)) >= maxFileSize {
			return nil, &CQFileError{Kind: CQFileErrInvalidSize, Raw: fp, Normalized: normalizedURL}
		}
		contentType := resp.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		return &FileElement{
			File:        fileName,
			URL:         normalizedURL,
			Stream:      bytes.NewReader(content),
			ContentType: contentType,
		}, nil
	} else if strings.HasPrefix(fp, "base64://") {
		content, err := base64.StdEncoding.DecodeString(fp[9:])
		if err != nil {
			return nil, &CQFileError{Kind: CQFileErrInvalidURL, Raw: fp, Cause: err}
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
		// 本地文件路径处理。路径解析和访问范围校验与其他本地资源保持一致。
		fileURL, absolutePath, info, err := normalizeLocalFileReference(fp)
		if err != nil {
			return nil, err
		}
		if info.Size() == 0 || info.Size() >= maxFileSize {
			return nil, &CQFileError{Kind: CQFileErrInvalidSize, Raw: fp, Normalized: absolutePath}
		}
		filesuffix := path.Ext(absolutePath)
		content, err := os.ReadFile(absolutePath) //nolint:gosec
		if err != nil {
			return nil, &CQFileError{Kind: CQFileErrUnavailable, Raw: fp, Normalized: absolutePath, Cause: err}
		}
		contenttype := mime.TypeByExtension(filesuffix)
		if len(contenttype) == 0 {
			contenttype = "application/octet-stream"
		}
		r := &FileElement{
			Stream:      bytes.NewReader(content),
			ContentType: contenttype,
			File:        info.Name(),
			URL:         fileURL,
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

	fn = strings.TrimSpace(fn)

	if isPassthroughResourceReference(fn) {
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

	normalized, _, _, err := normalizeLocalFileReference(fn)
	if err != nil {
		var fileErr *CQFileError
		if errors.As(err, &fileErr) {
			if fileErr.Kind == CQFileErrUnavailable && errors.Is(fileErr.Cause, os.ErrNotExist) {
				return "[找不到图片/文件]"
			}
			if fileErr.Kind == CQFileErrRestricted {
				return "[图片/文件指向非当前程序目录，已禁止]"
			}
		}
		return text
	}
	cq := CQCommand{
		Type: cqType,
		Args: map[string]string{"file": normalized},
	}
	return cq.Compile()
}

// convertConfig ConvertStringMessage的配置
type convertConfig struct {
	logger  *zap.SugaredLogger
	onError func(err error, cqType string, cqArgs map[string]string)
}

// ConvertOption ConvertStringMessage的选项函数
type ConvertOption func(*convertConfig)

// WithLogger 设置日志记录器
func WithLogger(l *zap.SugaredLogger) ConvertOption {
	return func(c *convertConfig) {
		if l != nil {
			c.logger = l
		}
	}
}

// WithOnError 设置错误回调
func WithOnError(fn func(err error, cqType string, cqArgs map[string]string)) ConvertOption {
	return func(c *convertConfig) { c.onError = fn }
}

func ConvertStringMessage(raw string, opts ...ConvertOption) (r []IMessageElement) {
	cfg := &convertConfig{
		// 默认使用全局logger，确保控制台+前端日志可见
		logger: zap.S().Named("message"),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	var arg, key string
	dMap := map[string]string{}

	text := ImageRewrite(raw, SealCodeToCqCode)

	resourceName := func(cqType string) string {
		switch cqType {
		case "image":
			return "图片"
		case "record":
			return "语音"
		case "video":
			return "视频"
		case "file":
			return "文件"
		default:
			return "资源"
		}
	}
	placeholderForError := func(err error, cqType string, cqArgs map[string]string) string {
		name := resourceName(cqType)
		var fe *CQFileError
		if errors.As(err, &fe) {
			switch fe.Kind {
			case CQFileErrInvalidURL:
				return fmt.Sprintf("[%sURL无效]", name)
			case CQFileErrUnavailable:
				if fe.StatusCode == http.StatusNotFound || errors.Is(fe.Cause, os.ErrNotExist) {
					return fmt.Sprintf("[找不到%s]", name)
				}
				return fmt.Sprintf("[%s不可用]", name)
			case CQFileErrRestricted:
				return fmt.Sprintf("[%s路径受限]", name)
			case CQFileErrInvalidSize:
				return fmt.Sprintf("[%s大小无效]", name)
			default:
				return fmt.Sprintf("[%s处理失败]", name)
			}
		}
		return "[消息解析失败]"
	}

	saveCQCode := func() {
		elem, err := toElement(arg, dMap)
		if err != nil {
			// 错误时跳过该CQ码，不原样发出，但记录日志
			var fe *CQFileError
			if errors.As(err, &fe) {
				switch fe.Kind {
				case CQFileErrInvalidURL:
					cfg.logger.Warnf("CQ码资源URL格式错误，已跳过: type=%s raw=%q err=%v", arg, fe.Raw, fe)
				case CQFileErrUnavailable:
					if fe.StatusCode > 0 {
						cfg.logger.Warnf("CQ码资源不可用(HTTP %d)，已跳过: type=%s raw=%q", fe.StatusCode, arg, fe.Raw)
					} else {
						cfg.logger.Warnf("CQ码资源不可用，已跳过: type=%s raw=%q err=%v", arg, fe.Raw, fe.Cause)
					}
				case CQFileErrRestricted:
					cfg.logger.Warnf("CQ码资源路径受限，已跳过: type=%s raw=%q", arg, fe.Raw)
				case CQFileErrInvalidSize:
					cfg.logger.Warnf("CQ码资源文件大小无效，已跳过: type=%s raw=%q", arg, fe.Raw)
				default:
					cfg.logger.Warnf("CQ码资源处理失败，已跳过: type=%s raw=%q err=%v", arg, fe.Raw, fe)
				}
			} else {
				cfg.logger.Warnf("转换CQ码失败，已跳过: type=%s args=%v err=%v", arg, dMap, err)
			}
			// 调用错误回调（如果设置了的话）
			if cfg.onError != nil {
				// 复制一份dMap防止后续修改影响回调
				argsCopy := make(map[string]string, len(dMap))
				for k, v := range dMap {
					argsCopy[k] = v
				}
				cfg.onError(err, arg, argsCopy)
			}
			if fe != nil {
				return
			}
			r = append(r, newText(placeholderForError(err, arg, dMap)))
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
			dMap[key] = UnescapeCQParam(text[:i])
			text = text[i:]
			i = 0
		}
	}
	return r
}
