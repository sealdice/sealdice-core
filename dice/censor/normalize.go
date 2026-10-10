package censor

import (
	"regexp"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

type normalized struct {
	text       string
	runeToOrig []int // norm rune 下标 -> 原文 rune 下标
	byteToRune []int // norm byte 偏移 -> norm rune 下标（长度 = len(text)+1）
}

func isDroppable(r rune) bool {
	switch r {
	case '\u200b', '\u200c', '\u200d', '\u2060', '\ufeff':
		return true
	}
	return false
}

// buildDropMask 标记被 regex 匹配覆盖的原文 rune 下标。
func buildDropMask(text string, re *regexp.Regexp) []bool {
	runes := []rune(text)
	mask := make([]bool, len(runes))
	if re == nil {
		return mask
	}
	for _, loc := range re.FindAllStringIndex(text, -1) {
		start := len([]rune(text[:loc[0]]))
		end := len([]rune(text[:loc[1]]))
		for i := start; i < end && i < len(mask); i++ {
			mask[i] = true
		}
	}
	return mask
}

func normalize(text string, caseSensitive bool, drop []bool) normalized {
	var b []byte
	runeToOrig := make([]int, 0, len(text))
	byteToRune := make([]int, 0, len(text)+1)
	for oi, r := range []rune(text) {
		if isDroppable(r) {
			continue
		}
		if oi < len(drop) && drop[oi] {
			continue
		}
		for _, nr := range []rune(norm.NFKC.String(string(r))) {
			if !caseSensitive {
				nr = unicode.ToLower(nr)
			}
			idx := len(runeToOrig)
			bs := []byte(string(nr))
			for range bs {
				byteToRune = append(byteToRune, idx)
			}
			b = append(b, bs...)
			runeToOrig = append(runeToOrig, oi)
		}
	}
	byteToRune = append(byteToRune, len(runeToOrig))
	return normalized{text: string(b), runeToOrig: runeToOrig, byteToRune: byteToRune}
}
