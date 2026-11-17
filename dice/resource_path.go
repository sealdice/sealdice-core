package dice

import (
	"path/filepath"
	"regexp"
	"strings"
)

var resourceCodePrefixes = []string{
	"[图:", "[img:",
	"[文本:", "[text:",
	"[语音:", "[voice:",
	"[视频:", "[video:",
}

var resourceCodePattern = regexp.MustCompile(`\[(img|图|文本|text|语音|voice|视频|video):(.+?)]`)

func isResourceCode(text string) bool {
	for _, prefix := range resourceCodePrefixes {
		if strings.HasPrefix(text, prefix) {
			return true
		}
	}
	return false
}

func resolveRelativeResourcePath(sourceFilename, resourcePath string) string {
	trimmed := strings.TrimSpace(resourcePath)
	normalized := strings.ReplaceAll(trimmed, `\`, "/")
	if sourceFilename == "" || !strings.HasPrefix(normalized, "./") {
		return resourcePath
	}

	// 相对资源路径以内容来源文件为基准。这里先转成绝对路径，避免包被放入
	// cache/packages 后，后续消息解析又按进程工作目录解释路径。
	resolved, err := filepath.Abs(filepath.Join(filepath.Dir(sourceFilename), filepath.FromSlash(normalized)))
	if err != nil {
		return resourcePath
	}
	return filepath.ToSlash(resolved)
}

func rewriteRelativeResourcePaths(sourceFilename, text string) string {
	// CQ 码可能包含额外参数，使用 CQRewrite 只改写 file 字段并保留其余参数。
	cqSolve := func(cq *CQCommand) {
		if filename, exists := cq.Args["file"]; exists {
			cq.Args["file"] = resolveRelativeResourcePath(sourceFilename, filename)
		}
	}

	sealCodeSolve := func(code string) string {
		match := resourceCodePattern.FindStringSubmatch(code)
		if match == nil {
			return code
		}
		return "[" + match[1] + ":" + resolveRelativeResourcePath(sourceFilename, match[2]) + "]"
	}

	text = CQRewrite(text, cqSolve)
	return ImageRewrite(text, sealCodeSolve)
}
