package censor

import "sort"

// MaskSpans 用 placeholder 逐 rune 替换 spans（原文 rune 偏移），重叠区间合并。
func MaskSpans(orig string, spans []Span, placeholder string) string {
	if len(spans) == 0 {
		return orig
	}
	runes := []rune(orig)
	ss := make([]Span, 0, len(spans))
	for _, s := range spans {
		if s.Start < s.End {
			ss = append(ss, s)
		}
	}
	if len(ss) == 0 {
		return orig
	}
	sort.Slice(ss, func(i, j int) bool { return ss[i].Start < ss[j].Start })
	var out []rune
	last := 0
	for _, s := range ss {
		start, end := s.Start, s.End
		if start < last {
			start = last
		}
		if end > len(runes) {
			end = len(runes)
		}
		if start >= end {
			continue
		}
		out = append(out, runes[last:start]...)
		if placeholder != "" {
			ph := []rune(placeholder)
			for i := start; i < end; i++ {
				out = append(out, ph...)
			}
		}
		last = end
	}
	out = append(out, runes[last:]...)
	return string(out)
}
