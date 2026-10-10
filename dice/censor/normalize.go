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

// isCombining 判断组合标记（附加符号等），它们需要与前面的基字符合成后再规范化。
func isCombining(r rune) bool {
	return unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Mc, r) || unicode.Is(unicode.Me, r)
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

// normalizeKey 对词表键做与正文一致的归一化，仅返回匹配文本。
func normalizeKey(key string, caseSensitive bool) string {
	return normalize(key, caseSensitive, nil).text
}

// normalize 构建匹配文本及到原文的偏移映射。
// 基字符与其后的组合标记作为一个簇整体做 NFKC，使分解/预组合形式产生相同匹配文本；
// 簇内所有规范化结果映射到簇首的原文下标。
func normalize(text string, caseSensitive bool, drop []bool) normalized {
	var b []byte
	runeToOrig := make([]int, 0, len(text))
	byteToRune := make([]int, 0, len(text)+1)

	emit := func(nr rune, origIdx int) {
		if !caseSensitive {
			nr = unicode.ToLower(nr)
		}
		idx := len(runeToOrig)
		bs := []byte(string(nr))
		for range bs {
			byteToRune = append(byteToRune, idx)
		}
		b = append(b, bs...)
		runeToOrig = append(runeToOrig, origIdx)
	}

	cluster := make([]rune, 0, 4)
	clusterOrig := 0
	flushCluster := func() {
		if len(cluster) == 0 {
			return
		}
		for _, nr := range norm.NFKC.String(string(cluster)) {
			emit(nr, clusterOrig)
		}
		cluster = cluster[:0]
	}

	for oi, r := range []rune(text) {
		if isDroppable(r) || (oi < len(drop) && drop[oi]) {
			flushCluster()
			continue
		}
		if len(cluster) > 0 && isCombining(r) {
			cluster = append(cluster, r)
			continue
		}
		flushCluster()
		cluster = append(cluster, r)
		clusterOrig = oi
	}
	flushCluster()

	byteToRune = append(byteToRune, len(runeToOrig))
	return normalized{text: string(b), runeToOrig: runeToOrig, byteToRune: byteToRune}
}
