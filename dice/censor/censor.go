package censor

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	nanoid "github.com/matoous/go-nanoid/v2"
	"github.com/mozillazg/go-pinyin"
	"github.com/pelletier/go-toml/v2"
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

type WordFile struct {
	Key         string
	Path        string
	FileCounter *FileCounter

	FileType   string
	Name       string
	Authors    []string
	Version    string
	Desc       string
	License    string
	Date       time.Time
	UpdateDate time.Time
}

type FileCounter [5]int

func (c *Censor) PreloadFile(path string) (*WordFile, error) {
	if strings.ToLower(filepath.Ext(path)) == ".toml" {
		return c.tryPreloadTomlFile(path)
	} else {
		return c.tryPreloadTxtFile(path)
	}
}

func (c *Censor) tryPreloadTxtFile(path string) (*WordFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	curLevel := Ignore
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
				c.addWord(word, curLevel, &counter)
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	return &WordFile{
		Key:         generateFileKey(),
		Path:        path,
		FileCounter: &counter,
		FileType:    "txt",
		Name:        filepath.Base(path),
	}, nil
}

type TomlMeta struct {
	Name       string    `toml:"name" comment:"词库名称"`
	Author     string    `toml:"author" comment:"作者，和 authors 存在一个即可"`
	Authors    []string  `toml:"authors" comment:"作者（多个），和 author 存在一个即可，同时存在时优先级高于 author"`
	Version    string    `toml:"version" comment:"版本，建议使用语义化版本号"`
	Desc       string    `toml:"desc" comment:"简介"`
	License    string    `toml:"license" comment:"协议"`
	Date       time.Time `toml:"date" comment:"创建日期，使用 RFC 3339 格式"`
	UpdateDate time.Time `toml:"updateDate" comment:"更新日期，使用 RFC 3339 格式"`
}

type TomlWords struct {
	Ignore  []string `toml:"ignore" comment:"忽略级词表，没有实际作用"`
	Notice  []string `toml:"notice" comment:"提醒级词表"`
	Caution []string `toml:"caution" comment:"注意级词表"`
	Warning []string `toml:"warning" comment:"警告级词表"`
	Danger  []string `toml:"danger" comment:"危险级词表"`
}

type TomlCensorWordFile struct {
	Meta  TomlMeta  `toml:"meta" comment:"元信息，用于填写一些额外的展示内容"`
	Words TomlWords `toml:"words" comment:"词表，出现相同词汇时按最高级别判断"`
}

func (c *Censor) tryPreloadTomlFile(path string) (*WordFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	reader := bufio.NewReader(file)

	var tomlFile *TomlCensorWordFile
	if err = toml.NewDecoder(reader).Decode(&tomlFile); err != nil {
		return nil, err
	}

	var counter FileCounter
	for _, word := range tomlFile.Words.Ignore {
		c.addWord(word, Ignore, &counter)
	}
	for _, word := range tomlFile.Words.Notice {
		c.addWord(word, Notice, &counter)
	}
	for _, word := range tomlFile.Words.Caution {
		c.addWord(word, Caution, &counter)
	}
	for _, word := range tomlFile.Words.Warning {
		c.addWord(word, Warning, &counter)
	}
	for _, word := range tomlFile.Words.Danger {
		c.addWord(word, Danger, &counter)
	}

	meta := tomlFile.Meta
	if meta.Name == "" {
		meta.Name = filepath.Base(path)
	}
	if meta.Author != "" && len(meta.Authors) == 0 {
		meta.Authors = append(meta.Authors, meta.Author)
	}

	return &WordFile{
		Key:         generateFileKey(),
		Path:        path,
		FileCounter: &counter,
		FileType:    "toml",
		Name:        meta.Name,
		Authors:     meta.Authors,
		Version:     meta.Version,
		Desc:        meta.Desc,
		License:     meta.License,
		Date:        meta.Date,
		UpdateDate:  meta.UpdateDate,
	}, nil
}

func (c *Censor) addWord(word string, level Level, counter *FileCounter) {
	key := strings.ToLower(strings.TrimSpace(word))
	counter[level]++
	if c.CaseSensitive {
		c.SensitiveKeys[key] = WordInfo{Level: level}
	} else {
		if c.MatchPinyin {
			// 拼音必须大小写不敏感
			w := strings.ToLower(key)
			c.SensitiveKeys[w] = WordInfo{Level: level, Origin: key, Reason: IgnoreCase}

			pys := pinyin.LazyPinyin(w, pinyin.Args{
				Style: pinyin.Normal,
				Fallback: func(r rune, a pinyin.Args) []string {
					return []string{string(r)}
				},
			})
			pyStr := strings.Join(pys, "")
			c.SensitiveKeys[strings.ToLower(pyStr)] = WordInfo{Level: level, Origin: key, Reason: PinYin}
		} else {
			c.SensitiveKeys[strings.ToLower(key)] = WordInfo{Level: level, Origin: key, Reason: IgnoreCase}
		}
	}
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

func generateFileKey() string {
	key, _ := nanoid.Generate("0123456789abcdef", 16)
	return key
}
