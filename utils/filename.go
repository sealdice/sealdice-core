package utils

import (
	"fmt"
	"hash/fnv"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var filenameIllegalChars = regexp.MustCompile(`[/:\*\?"<>\|\\]`)

// FilenameClean makes a name legal for file name by removing every
// '/', ':', '*', '?', '"', '<', '>', '|', '\' from the name.
func FilenameClean(name string) string {
	return filenameIllegalChars.ReplaceAllString(name, "")
}

// FilenameSafeReadable returns a readable and bounded filename segment.
// It removes illegal path characters and control characters, folds
// whitespace/symbol runs into underscores, trims dangerous suffixes, and
// appends a short hash when truncation is needed.
func FilenameSafeReadable(name string, fallback string, maxBytes int) string {
	if maxBytes <= 0 {
		maxBytes = 64
	}

	cleaned := sanitizeFilenamePart(name)
	fallbackCleaned := sanitizeFilenamePart(fallback)
	if fallbackCleaned == "" {
		fallbackCleaned = "file"
	}
	if cleaned == "" {
		cleaned = fallbackCleaned
	}
	if isWindowsReservedFilename(cleaned) {
		cleaned = "_" + cleaned
	}
	if len(cleaned) <= maxBytes {
		return cleaned
	}

	hashSuffix := shortFilenameHash(name)
	keepBytes := maxBytes - len(hashSuffix) - 1
	if keepBytes <= 0 {
		if len(fallbackCleaned) > maxBytes {
			return truncateUTF8Bytes(fallbackCleaned, maxBytes)
		}
		return fallbackCleaned
	}

	prefix := truncateUTF8Bytes(cleaned, keepBytes)
	prefix = trimFilenamePart(prefix)
	if prefix == "" {
		prefix = truncateUTF8Bytes(fallbackCleaned, keepBytes)
		prefix = trimFilenamePart(prefix)
		if prefix == "" {
			prefix = "file"
		}
	}

	result := prefix + "-" + hashSuffix
	if isWindowsReservedFilename(result) {
		result = "_" + result
	}
	if len(result) > maxBytes {
		result = truncateUTF8Bytes(result, maxBytes)
		result = trimFilenamePart(result)
	}
	if result == "" {
		return fallbackCleaned
	}
	return result
}

// TempFilePattern returns a safe CreateTemp pattern with a readable prefix.
func TempFilePattern(prefix string, fallback string, ext string, maxBytes int) string {
	cleanExt := sanitizeFilenamePart(strings.TrimPrefix(ext, "."))
	suffixBytes := 2 // "-*"
	if cleanExt != "" {
		suffixBytes += 1 + len(cleanExt)
	}

	prefixLimit := maxBytes - suffixBytes
	if prefixLimit <= 0 {
		prefixLimit = maxBytes
	}
	safePrefix := FilenameSafeReadable(prefix, fallback, prefixLimit)
	if cleanExt == "" {
		return safePrefix + "-*"
	}
	return fmt.Sprintf("%s-*.%s", safePrefix, cleanExt)
}

// TempFilePatternFromName returns a safe CreateTemp pattern while preserving
// the extension from the provided name when possible.
func TempFilePatternFromName(name string, fallback string, maxBytes int) string {
	base := filepath.Base(name)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	return TempFilePattern(stem, fallback, ext, maxBytes)
}

func sanitizeFilenamePart(name string) string {
	var builder strings.Builder
	builder.Grow(len(name))

	lastUnderscore := false
	for _, r := range name {
		switch {
		case r == utf8.RuneError:
			if !lastUnderscore {
				builder.WriteByte('_')
				lastUnderscore = true
			}
		case strings.ContainsRune(`/:\*?"<>|\`, r):
			if !lastUnderscore {
				builder.WriteByte('_')
				lastUnderscore = true
			}
		case unicode.IsControl(r):
			if !lastUnderscore {
				builder.WriteByte('_')
				lastUnderscore = true
			}
		case unicode.IsSpace(r):
			if !lastUnderscore {
				builder.WriteByte('_')
				lastUnderscore = true
			}
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			lastUnderscore = false
		case strings.ContainsRune("._-()[]{}", r):
			builder.WriteRune(r)
			lastUnderscore = false
		default:
			if !lastUnderscore {
				builder.WriteByte('_')
				lastUnderscore = true
			}
		}
	}

	return trimFilenamePart(builder.String())
}

func trimFilenamePart(name string) string {
	name = strings.Trim(name, " ._-")
	for strings.Contains(name, "__") {
		name = strings.ReplaceAll(name, "__", "_")
	}
	return name
}

func truncateUTF8Bytes(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}

	var builder strings.Builder
	builder.Grow(maxBytes)
	size := 0
	for _, r := range s {
		runeBytes := utf8.RuneLen(r)
		if runeBytes < 0 {
			runeBytes = 1
		}
		if size+runeBytes > maxBytes {
			break
		}
		builder.WriteRune(r)
		size += runeBytes
	}
	return builder.String()
}

func shortFilenameHash(name string) string {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(name))
	return fmt.Sprintf("%08x", hasher.Sum32())
}

func isWindowsReservedFilename(name string) bool {
	upper := strings.ToUpper(strings.TrimSpace(name))
	switch upper {
	case "CON", "PRN", "AUX", "NUL":
		return true
	}
	if len(upper) == 4 && (strings.HasPrefix(upper, "COM") || strings.HasPrefix(upper, "LPT")) {
		last := upper[3]
		return last >= '1' && last <= '9'
	}
	return false
}
