package utils

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

const DefaultSplitPaginationHint = "[ %d / %d ]\n"

// cqCodePattern 匹配有效的 CQ 码格式
// 格式: [CQ:type] 或 [CQ:type,key=value,...]
// type 由小写字母组成，参数格式为 key=value
var cqCodePattern = regexp.MustCompile(`^\[CQ:[a-z]+(?:,[^,\[\]]+=[^,\[\]]*)*\]`)

// findCQCodeRange 查找包含指定位置的有效 CQ 码范围
// 返回 (start, end)，如果 pos 不在有效 CQ 码内则返回 (-1, -1)
// CQ 码必须符合标准格式: [CQ:type,key=value,...] 或 [CQ:type]
func findCQCodeRange(s string, pos int) (int, int) {
	if pos >= len(s) {
		return -1, -1
	}

	// 遍历所有可能的 CQ 码，找到包含 pos 的那个
	searchStart := 0
	for {
		cqStart := strings.Index(s[searchStart:], "[CQ:")
		if cqStart == -1 {
			break
		}
		cqStart += searchStart

		// 使用正则验证是否为有效的 CQ 码格式
		remaining := s[cqStart:]
		match := cqCodePattern.FindString(remaining)
		if match == "" {
			// 不是有效的 CQ 码格式，跳过继续查找
			searchStart = cqStart + 4 // 跳过 "[CQ:"
			continue
		}

		cqEndAbs := cqStart + len(match) - 1

		// 检查 pos 是否在这个 CQ 码范围内
		if pos >= cqStart && pos <= cqEndAbs {
			return cqStart, cqEndAbs
		}

		// 如果 pos 在当前 CQ 码之前，说明 pos 不在任何 CQ 码内
		if pos < cqStart {
			break
		}

		// 继续查找下一个 CQ 码
		searchStart = cqEndAbs + 1
	}

	return -1, -1
}

// adjustSplitPointForCQCode 调整切分点以避免切断 CQ 码
// 返回安全的切分位置
func adjustSplitPointForCQCode(s string, pos int) int {
	start, _ := findCQCodeRange(s, pos)
	if start == -1 {
		return pos // 不在 CQ 码内，原位置安全
	}
	if start == 0 {
		return 0 // CQ 码从开头开始
	}
	return start
}

func splitFirst(s string, maxLen int) (first string, rest string) {
	// 不足上限不切分
	if len(s) <= maxLen {
		return s, ""
	}

	// 确保子串长度不大于 maxLen 且完整切分 UTF-8 字符
	r := maxLen
	for (!utf8.RuneStart(s[r])) && r > 0 {
		r--
	}

	// 调整切分点以避免切断 CQ 码
	r = adjustSplitPointForCQCode(s, r)
	if r == 0 {
		// CQ 码从开头开始且超过 maxLen，找到 CQ 码结束位置
		match := cqCodePattern.FindString(s)
		if match != "" && len(match) < len(s) {
			return s[:len(match)], s[len(match):]
		}
		// 整个字符串就是一个 CQ 码，不切分
		return s, ""
	}

	// 如果有连续换行符, 直接切分（但要确保不在 CQ 码内）
	multiNL := regexp.MustCompile(`\n{2,}`)
	idxMultiNL := multiNL.FindStringIndex(s[0:r])
	if len(idxMultiNL) == 2 {
		start, _ := findCQCodeRange(s, idxMultiNL[0])
		if start == -1 {
			return s[0:idxMultiNL[0]], s[idxMultiNL[1]:]
		}
	}

	// 如果切分中有换行符, 以最后一个换行符切分（但要确保不在 CQ 码内）
	idxNL := strings.LastIndex(s[0:r], "\n")
	if idxNL >= 0 {
		start, _ := findCQCodeRange(s, idxNL)
		if start == -1 {
			return s[0:idxNL], s[idxNL+1:]
		}
	}

	return s[0:r], s[r:]
}

// SplitLongText 切分长文本
//   - text 要切分的文本.
//   - maxLen 子串长度上限(字节数), <=0 时不切分.
//     **由于分页提示的存在, 实际子串长度可能略大于 maxLen**.
//   - paginationHint 分页提示. 如果结果大于 1 页, 加在每页开头.
//     应该含有 0 或 2 个 "%d", 将被依次替换为当前页数和总页数.
//     为空或 "%d" 数量非法时使用默认值 `DefaultSplitPaginationHint`.
//
// 确保切分后每个子串长度不大于 maxLen.
// 优先以 text 中的换行符为切分点, 总长度小于 maxLen 的连续短行不切分.
//
// 例外: text 中的多个连续换行符会被强制切分.
func SplitLongText(text string, maxLen int, paginationHint string) []string {
	if maxLen <= 0 {
		return []string{text}
	}

	if len(paginationHint) == 0 {
		paginationHint = DefaultSplitPaginationHint
	}
	count := strings.Count(paginationHint, "%d")
	if count != 0 && count != 2 {
		paginationHint = DefaultSplitPaginationHint
		count = 2
	}

	var splits []string

	for len(text) > 0 {
		first, rest := splitFirst(text, maxLen)
		if len(strings.TrimSpace(first)) > 0 {
			splits = append(splits, first)
		}
		text = rest
	}

	if l := len(splits); l > 1 {
		for i := range splits {
			if count == 2 {
				splits[i] = fmt.Sprintf(paginationHint, i+1, l) + splits[i]
			} else {
				splits[i] = paginationHint + splits[i]
			}
		}
	}

	return splits
}
