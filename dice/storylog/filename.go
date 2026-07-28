package storylog

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/cespare/xxhash/v2"
)

const (
	maxExportFilenameBytes = 240
	maxTempPatternBytes    = 128
	fileNameFallbackNotice = "日志名不适合作为文件名，已改用哈希文件名保存/上传，不影响日志内容与查询。"
)

var invalidFilenameCharsRe = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)

func buildLogBackupFilename(groupID, logName string, now time.Time) (string, string) {
	groupPart, ok := sanitizeFilenameComponent(groupID)
	if !ok {
		groupPart = "group"
	}
	groupPart = trimUTF8ByBytes(groupPart, 48)

	logPart, logOK := sanitizeFilenameComponent(logName)
	timestamp := now.Format("060102150405")
	if logOK {
		name := fmt.Sprintf("%s_%s.%s.zip", groupPart, logPart, timestamp)
		if len([]byte(name)) <= maxExportFilenameBytes {
			return name, ""
		}
	}

	hashPart := hashHex(logName)
	name := fmt.Sprintf("%s_%s.%s.zip", groupPart, hashPart, timestamp)
	if len([]byte(name)) <= maxExportFilenameBytes {
		return name, fileNameFallbackNotice
	}

	return fmt.Sprintf("log_%s.%s.zip", hashPart, timestamp), fileNameFallbackNotice
}

func buildTempPattern(prefix string) (string, string) {
	cleanPrefix, ok := sanitizeFilenameComponent(prefix)
	if ok {
		if len([]byte(cleanPrefix+"-*.txt")) <= maxTempPatternBytes {
			pattern := cleanPrefix + "-*.txt"
			return pattern, ""
		}
	}

	pattern := "log-export-" + hashHex(prefix) + "-*.txt"
	if len([]byte(pattern)) > maxTempPatternBytes {
		pattern = trimUTF8ByBytes(pattern, maxTempPatternBytes)
		if !strings.Contains(pattern, "*") {
			pattern = "log-" + hashHex(prefix)[:8] + "-*.txt"
		}
	}
	return pattern, fileNameFallbackNotice
}

func BuildTempPattern(prefix string) (string, string) {
	return buildTempPattern(prefix)
}

func sanitizeFilenameComponent(name string) (string, bool) {
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, name)
	name = invalidFilenameCharsRe.ReplaceAllString(name, "")
	name = strings.Trim(name, ". ")
	if name == "" || name == "." || name == ".." || isWindowsReservedFilename(name) {
		return "", false
	}
	return name, true
}

func isWindowsReservedFilename(name string) bool {
	base := strings.ToUpper(strings.TrimSpace(name))
	switch base {
	case "CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return true
	default:
		return false
	}
}

func trimUTF8ByBytes(s string, maxBytes int) string {
	if len([]byte(s)) <= maxBytes {
		return s
	}
	var builder strings.Builder
	builder.Grow(maxBytes)
	size := 0
	for _, r := range s {
		runeSize := utf8.RuneLen(r)
		if size+runeSize > maxBytes {
			break
		}
		builder.WriteRune(r)
		size += runeSize
	}
	return builder.String()
}

func hashHex(text string) string {
	return fmt.Sprintf("%016x", xxhash.Sum64String(text))
}
