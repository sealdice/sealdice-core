package utils

import (
	"strings"
	"testing"
)

func TestFindCQCodeRange(t *testing.T) {
	tests := []struct {
		name      string
		s         string
		pos       int
		wantStart int
		wantEnd   int
	}{
		{
			name:      "不在CQ码内",
			s:         "Hello World",
			pos:       5,
			wantStart: -1,
			wantEnd:   -1,
		},
		{
			name:      "在CQ码开始位置",
			s:         "Hello [CQ:image,file=test] World",
			pos:       6,
			wantStart: 6,
			wantEnd:   25,
		},
		{
			name:      "在CQ码中间",
			s:         "Hello [CQ:image,file=test] World",
			pos:       15,
			wantStart: 6,
			wantEnd:   25,
		},
		{
			name:      "在CQ码结束位置",
			s:         "Hello [CQ:image,file=test] World",
			pos:       25,
			wantStart: 6,
			wantEnd:   25,
		},
		{
			name:      "在CQ码之后",
			s:         "Hello [CQ:image,file=test] World",
			pos:       27,
			wantStart: -1,
			wantEnd:   -1,
		},
		{
			name:      "多个CQ码-在第一个内",
			s:         "[CQ:at,qq=123] Hello [CQ:image,file=test]",
			pos:       5,
			wantStart: 0,
			wantEnd:   13,
		},
		{
			name:      "多个CQ码-在第二个内",
			s:         "[CQ:at,qq=123] Hello [CQ:image,file=test]",
			pos:       30,
			wantStart: 21,
			wantEnd:   40,
		},
		// 无效 CQ 码格式测试
		{
			name:      "无效CQ码-缺少类型名",
			s:         "Hello [CQ:] World",
			pos:       8,
			wantStart: -1,
			wantEnd:   -1,
		},
		{
			name:      "无效CQ码-类型名含大写",
			s:         "Hello [CQ:Image,file=test] World",
			pos:       15,
			wantStart: -1,
			wantEnd:   -1,
		},
		{
			name:      "无效CQ码-类型名含数字",
			s:         "Hello [CQ:image123,file=test] World",
			pos:       15,
			wantStart: -1,
			wantEnd:   -1,
		},
		{
			name:      "无效CQ码-参数格式错误无等号",
			s:         "Hello [CQ:image,file] World",
			pos:       15,
			wantStart: -1,
			wantEnd:   -1,
		},
		{
			name:      "无效CQ码-未闭合",
			s:         "Hello [CQ:image,file=test World",
			pos:       15,
			wantStart: -1,
			wantEnd:   -1,
		},
		{
			name:      "有效CQ码-无参数",
			s:         "Hello [CQ:face] World",
			pos:       10,
			wantStart: 6,
			wantEnd:   14,
		},
		{
			name:      "有效CQ码-多个参数",
			s:         "Hello [CQ:image,file=test,cache=1] World",
			pos:       20,
			wantStart: 6,
			wantEnd:   33,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStart, gotEnd := findCQCodeRange(tt.s, tt.pos)
			if gotStart != tt.wantStart || gotEnd != tt.wantEnd {
				t.Errorf("findCQCodeRange() = (%v, %v), want (%v, %v)",
					gotStart, gotEnd, tt.wantStart, tt.wantEnd)
			}
		})
	}
}

func TestSplitLongText_CQCodeProtection(t *testing.T) {
	// 生成一个长的 base64 字符串
	longBase64 := strings.Repeat("A", 3000)

	tests := []struct {
		name        string
		text        string
		maxLen      int
		wantLen     int            // 期望切分后的片段数
		check       func([]string) // 自定义检查函数
		skipCQCheck bool           // 跳过通用 CQ 码完整性检查
	}{
		{
			name:    "普通文本切分",
			text:    strings.Repeat("Hello ", 500),
			maxLen:  100,
			wantLen: -1, // 不检查具体数量
			check:   nil,
		},
		{
			name:    "CQ码不被切断-短文本+长CQ码",
			text:    "Hello [CQ:image,file=base64://" + longBase64 + "] World",
			maxLen:  100,
			wantLen: 1, // 可读文本 "Hello  World" 只有 13 字节，不需要切分
			check: func(splits []string) {
				// 整条消息应该保持完整
				if !strings.Contains(splits[0], "[CQ:image,file=base64://") ||
					!strings.Contains(splits[0], "] World") {
					t.Error("消息应该保持完整不切分")
				}
			},
		},
		{
			name:    "CQ码在开头且超长",
			text:    "[CQ:image,file=base64://" + longBase64 + "]After",
			maxLen:  100,
			wantLen: 1, // 可读文本 "After" 只有 5 字节，不需要切分
			check: func(splits []string) {
				// 整条消息应该保持完整
				if !strings.Contains(splits[0], "[CQ:image,file=base64://") ||
					!strings.Contains(splits[0], "]After") {
					t.Error("消息应该保持完整不切分")
				}
			},
		},
		{
			name:    "多个CQ码",
			text:    "Start [CQ:at,qq=123] Middle [CQ:image,file=base64://" + longBase64 + "] End",
			maxLen:  100,
			wantLen: -1,
			check: func(splits []string) {
				// 检查所有 CQ 码都是完整的
				fullText := strings.Join(splits, "")
				// 移除分页提示后检查
				fullText = strings.ReplaceAll(fullText, "[ ", "")
				for i := 1; i <= 10; i++ {
					fullText = strings.ReplaceAll(fullText, string(rune('0'+i))+" / ", "")
				}
				fullText = strings.ReplaceAll(fullText, " ]\n", "")

				if !strings.Contains(fullText, "[CQ:at,qq=123]") {
					t.Error("第一个CQ码丢失或被切断")
				}
				if !strings.Contains(fullText, "[CQ:image,file=base64://"+longBase64+"]") {
					t.Error("第二个CQ码丢失或被切断")
				}
			},
		},
		{
			name:    "CQ码内有特殊字符",
			text:    "Test [CQ:image,file=base64://" + longBase64 + ",url=http://example.com] Done",
			maxLen:  100,
			wantLen: 1, // 可读文本 "Test  Done" 只有 10 字节，不需要切分
			check: func(splits []string) {
				// 整条消息应该保持完整
				if !strings.Contains(splits[0], "[CQ:image,") ||
					!strings.Contains(splits[0], "url=http://example.com]") {
					t.Error("消息应该保持完整不切分")
				}
			},
		},
		{
			name:        "无效CQ码-当作普通文本切分",
			text:        "Test [CQ:Invalid,file=" + longBase64 + "] Done", // Invalid 含大写，不是有效CQ码
			maxLen:      100,
			wantLen:     -1, // 不检查数量，因为会被正常切分
			skipCQCheck: true,
			check: func(splits []string) {
				// 无效 CQ 码应该被正常切分，不需要保护
				// 检查是否有被切断的情况（这是预期行为）
				fullText := strings.Join(splits, "")
				if !strings.Contains(fullText, "[CQ:Invalid") {
					t.Error("无效CQ码内容丢失")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			splits := SplitLongText(tt.text, tt.maxLen, DefaultSplitPaginationHint)

			if tt.wantLen > 0 && len(splits) != tt.wantLen {
				t.Errorf("切分数量 = %v, want %v", len(splits), tt.wantLen)
			}

			if tt.check != nil {
				tt.check(splits)
			}

			// 通用检查：确保没有片段在有效 CQ 码中间被切断
			if !tt.skipCQCheck {
				for i, s := range splits {
					openCQ := 0
					inCQ := false
					for j := 0; j < len(s); j++ {
						if j+4 <= len(s) && s[j:j+4] == "[CQ:" {
							openCQ++
							inCQ = true
						}
						if inCQ && s[j] == ']' {
							openCQ--
							if openCQ == 0 {
								inCQ = false
							}
						}
					}
					if openCQ > 0 {
						t.Errorf("片段 %d 有未闭合的CQ码: %s...", i, truncateStr(s, 50))
					}
				}
			}
		})
	}
}

func TestSplitFirst_CQCodeProtection(t *testing.T) {
	longContent := strings.Repeat("X", 100)

	tests := []struct {
		name      string
		s         string
		maxLen    int
		wantFirst string
		wantRest  string
	}{
		{
			name:      "普通文本",
			s:         "Hello World Test",
			maxLen:    10,
			wantFirst: "Hello Worl",
			wantRest:  "d Test",
		},
		{
			name:      "短文本+CQ码-不切分",
			s:         "Hi [CQ:image,file=" + longContent + "] End",
			maxLen:    20,
			wantFirst: "Hi [CQ:image,file=" + longContent + "] End", // 可读文本 "Hi  End" 只有 8 字节
			wantRest:  "",
		},
		{
			name:      "CQ码在开头-不切分",
			s:         "[CQ:image,file=" + longContent + "]After",
			maxLen:    20,
			wantFirst: "[CQ:image,file=" + longContent + "]After", // 可读文本 "After" 只有 5 字节
			wantRest:  "",
		},
		{
			name:      "整个字符串是一个CQ码",
			s:         "[CQ:image,file=" + longContent + "]",
			maxLen:    20,
			wantFirst: "[CQ:image,file=" + longContent + "]",
			wantRest:  "",
		},
		{
			name:      "长文本+CQ码+长文本-需要切分",
			s:         strings.Repeat("A", 50) + "[CQ:image,file=" + longContent + "]" + strings.Repeat("B", 50),
			maxLen:    40,
			wantFirst: strings.Repeat("A", 40),
			wantRest:  strings.Repeat("A", 10) + "[CQ:image,file=" + longContent + "]" + strings.Repeat("B", 50),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFirst, gotRest := splitFirst(tt.s, tt.maxLen)
			if gotFirst != tt.wantFirst {
				t.Errorf("splitFirst() first = %q, want %q", truncateStr(gotFirst, 50), truncateStr(tt.wantFirst, 50))
			}
			if gotRest != tt.wantRest {
				t.Errorf("splitFirst() rest = %q, want %q", truncateStr(gotRest, 50), truncateStr(tt.wantRest, 50))
			}
		})
	}
}

// truncateStr 截断字符串用于显示
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
