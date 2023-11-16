package utils

import (
	"strings"
	"unicode/utf8"
)

func splitFirst(s string, maxLen int) (first string, rest string) {
	// 去除首尾空白
	s = strings.TrimSpace(s)

	// 不足上限不切分
	if len(s) <= maxLen {
		return s, ""
	}

	// 确保子串长度不大于 maxLen 且完整切分 UTF-8 字符
	r := maxLen
	for (!utf8.RuneStart(s[r])) && r > 0 {
		r--
	}

	// 如果有连续换行符, 直接切分
	idxTwoNL := strings.Index(s[0:r], "\n\n")
	if idxTwoNL >= 0 {
		return s[0:idxTwoNL], s[idxTwoNL+2:]
	}

	// 如果切分中有换行符, 以最后一个换行符切分, 增强可读性
	idxNL := strings.LastIndex(s[0:r], "\n")
	if idxNL >= 0 {
		return s[0:idxNL], s[idxNL+1:]
	}

	return s[0:r], s[r:]
}

// SplitLongText 切分长文本
//   - text 要切分的文本
//   - maxLen 子串长度上限(字节数), <=0 时不切分
//
// 确保切分后每个子串长度不大于 maxLen.
// 优先以 text 中的换行符为切分点, 总长度小于 maxLen 的连续短行不切分.
//
// 例外: text 中的一个完全空行 (2 个连续换行符) 强制切分.
func SplitLongText(text string, maxLen int) []string {
	if maxLen <= 0 {
		return []string{text}
	}

	var splits []string

	for len(text) > 0 {
		first, rest := splitFirst(text, maxLen)
		splits = append(splits, first)
		text = rest
	}

	return splits
}
