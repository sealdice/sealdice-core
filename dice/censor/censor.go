package censor

import (
	"bufio"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/mozillazg/go-pinyin"
)

const (
	Ignore Level = iota
	Notice
	Caution
	Warning
	Danger
)

type Level int

type Levels []Level

func (ls Levels) Len() int { return len(ls) }
func (ls Levels) Less(i, j int) bool {
	return ls[i] < ls[j]
}
func (ls Levels) Swap(i, j int) { ls[i], ls[j] = ls[j], ls[i] }

var LevelText = map[Level]string{
	Ignore:  "忽略",
	Notice:  "提醒",
	Caution: "注意",
	Warning: "警告",
	Danger:  "危险",
}

func HigherLevel(l1 Level, l2 Level) Level {
	if l1 > l2 {
		return l1
	} else {
		return l2
	}
}

type Censor struct {
	CaseSensitive  bool   // 大小写敏感
	MatchPinyin    bool   // 匹配拼音
	FilterRegexStr string // 过滤字符正则

	SensitiveKeys map[string]WordInfo
	t             *trie
	filterRegex   *regexp.Regexp
}

type Reason int

const (
	Origin Reason = iota
	IgnoreCase
	PinYin
)

type WordInfo struct {
	Level  Level  // 级别
	Origin string // 附加词对应的原始词，如大小写不敏感指向原单词，拼音为原词汇
	Reason Reason // 添加原因
}

type FileCounter [5]int

func (c *Censor) PreloadFile(path string) (*FileCounter, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	curLevel := Ignore
	c.SensitiveKeys = make(map[string]WordInfo)
	reader := bufio.NewReader(file)
	var counter FileCounter
	for {
		word, err := reader.ReadString('\n')
		if word != "" {
			// 处理敏感词库
			if strings.HasPrefix(word, "#") {
				mark := strings.ToLower(strings.TrimSpace(word))
				switch mark {
				case "#ignore":
					curLevel = Ignore
				case "#notice":
					curLevel = Notice
				case "#caution":
					curLevel = Caution
				case "#warning":
					curLevel = Warning
				case "#danger":
					curLevel = Danger
				}
			} else {
				key := strings.ToLower(strings.TrimSpace(word))
				counter[curLevel]++
				if c.CaseSensitive {
					c.SensitiveKeys[key] = WordInfo{Level: curLevel}
				} else {
					if c.MatchPinyin {
						// 拼音必须大小写不敏感
						w := strings.ToLower(key)
						c.SensitiveKeys[w] = WordInfo{Level: curLevel, Origin: key, Reason: IgnoreCase}

						pys := pinyin.LazyPinyin(w, pinyin.Args{
							Style: pinyin.Normal,
							Fallback: func(r rune, a pinyin.Args) []string {
								return []string{string(r)}
							},
						})
						pyStr := strings.Join(pys, "")
						c.SensitiveKeys[strings.ToLower(pyStr)] = WordInfo{Level: curLevel, Origin: key, Reason: PinYin}
					} else {
						c.SensitiveKeys[strings.ToLower(key)] = WordInfo{Level: curLevel, Origin: key, Reason: IgnoreCase}
					}
				}
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	return &counter, nil
}

func (c *Censor) Load() (err error) {
	if c.FilterRegexStr != "" {
		c.filterRegex = regexp.MustCompile(c.FilterRegexStr)
	} else {
		c.filterRegex = nil
	}

	c.t = newTire()
	if c.SensitiveKeys != nil {
		for key, wordInfo := range c.SensitiveKeys {
			c.t.Insert(key, wordInfo.Level)
		}
	}
	return nil
}

type CheckResult struct {
	HighestLevel   Level
	SensitiveWords map[string]Level
}

func (c *Censor) Check(content string) CheckResult {
	if c.filterRegex != nil {
		content = c.filterRegex.ReplaceAllString(content, "")
	}
	sensitiveKeys := c.t.Match(content)
	sensitiveWords := make(map[string]Level)
	highestLevel := Ignore
	for key, level := range sensitiveKeys {
		highestLevel = HigherLevel(highestLevel, level)
		wordInfo := c.SensitiveKeys[key]
		sensitiveWords[wordInfo.Origin] = wordInfo.Level
	}
	return CheckResult{
		HighestLevel:   highestLevel,
		SensitiveWords: sensitiveWords,
	}
}
