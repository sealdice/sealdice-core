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
	CaseInsensitive
	PinYin
)

type WordInfo struct {
	Level  Level  // 级别
	Origin string // 附加词对应的原始词，如大小写不敏感指向原单词，拼音为原词汇
	Reason Reason // 添加原因
}

func (c *Censor) PreloadFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	curLevel := Ignore
	c.SensitiveKeys = make(map[string]WordInfo)
	reader := bufio.NewReader(file)
	for {
		word, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		// 处理敏感词库
		if strings.HasPrefix(word, "#") {
			mark := strings.ToLower(word)
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
			if c.CaseSensitive {
				c.SensitiveKeys[word] = WordInfo{Level: curLevel}
			} else {
				if c.MatchPinyin {
					// 拼音必须大小写不敏感
					w := strings.ToLower(word)
					c.SensitiveKeys[w] = WordInfo{Level: curLevel, Origin: word, Reason: CaseInsensitive}
					pys := pinyin.Pinyin(w, pinyin.NewArgs())
					for _, py := range pys {
						pyStr := strings.Join(py, "")
						c.SensitiveKeys[strings.ToLower(pyStr)] = WordInfo{Level: curLevel, Origin: word, Reason: PinYin}
					}
				} else {
					c.SensitiveKeys[strings.ToLower(word)] = WordInfo{Level: curLevel, Origin: word, Reason: CaseInsensitive}
				}
			}
		}

	}

	return nil
}

func (c *Censor) Load() (err error) {
	if c.FilterRegexStr != "" {
		c.filterRegex, err = regexp.Compile(c.FilterRegexStr)
		if err != nil {
			return err
		}
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
