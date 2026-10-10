package censor

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	ahocorasick "github.com/BobuSumisu/aho-corasick"
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
	}
	return l2
}

type Censor struct {
	CaseSensitive  bool   // 大小写敏感
	MatchPinyin    bool   // 匹配拼音
	FilterRegexStr string // 过滤字符正则

	SensitiveKeys map[string]WordInfo
	matcher       *ahocorasick.Trie
	filterRegex   *regexp.Regexp
	mu            sync.RWMutex
}

// Span 表示命中片段在原文中的 rune 偏移，半开区间 [Start, End)。
type Span struct {
	Start, End int
}

// Hit 表示一次命中。
type Hit struct {
	Span  Span
	Word  string // 原始词（WordInfo.Origin）
	Level Level
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
	}
	return c.tryPreloadTxtFile(path)
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
	Name       string    `comment:"词库名称"                                                                                       toml:"name"`
	Author     string    `comment:"作者，和 authors 存在一个即可"                                                                        toml:"author"`
	Authors    []string  `comment:"作者（多个），和 author 存在一个即可，同时存在时优先级高于 author"                                                   toml:"authors"`
	Version    string    `comment:"版本，建议使用语义化版本号"                                                                              toml:"version"`
	Desc       string    `comment:"简介"                                                                                         toml:"desc"`
	License    string    `comment:"协议"                                                                                         toml:"license"`
	Date       time.Time `comment:"创建日期，使用 RFC 3339 格式"                                                                        toml:"date"`
	UpdateDate time.Time `comment:"更新日期，使用 RFC 3339 格式"                                                                        toml:"updateDate"`
}

type TomlWords struct {
	Ignore  []string `comment:"忽略级词表，没有实际作用"                         toml:"ignore"`
	Notice  []string `comment:"提醒级词表"                                toml:"notice"`
	Caution []string `comment:"注意级词表"                                toml:"caution"`
	Warning []string `comment:"警告级词表"                                toml:"warning"`
	Danger  []string `comment:"危险级词表"                                toml:"danger"`
}

type TomlCensorWordFile struct {
	Meta  TomlMeta  `comment:"元信息，用于填写一些额外的展示内容"                                   toml:"meta"`
	Words TomlWords `comment:"词表，出现相同词汇时按最高级别判断"                                   toml:"words"`
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
	if c.SensitiveKeys == nil {
		c.SensitiveKeys = make(map[string]WordInfo)
	}
	key := strings.TrimSpace(word)
	counter[level]++
	if c.CaseSensitive {
		c.SensitiveKeys[key] = WordInfo{Level: level, Origin: key}
		return
	}
	key = strings.ToLower(key)
	if c.MatchPinyin {
		// 拼音必须大小写不敏感
		c.SensitiveKeys[key] = WordInfo{Level: level, Origin: key, Reason: IgnoreCase}

		pys := pinyin.LazyPinyin(key, pinyin.Args{
			Style: pinyin.Normal,
			Fallback: func(r rune, a pinyin.Args) []string {
				return []string{string(r)}
			},
		})
		pyStr := strings.Join(pys, "")
		c.SensitiveKeys[strings.ToLower(pyStr)] = WordInfo{Level: level, Origin: key, Reason: PinYin}
		return
	}
	c.SensitiveKeys[key] = WordInfo{Level: level, Origin: key, Reason: IgnoreCase}
}

// Load 以当前 SensitiveKeys 重建匹配器，等价于 LoadWords(c.SensitiveKeys)。
func (c *Censor) Load() error {
	c.mu.RLock()
	keys := c.SensitiveKeys
	c.mu.RUnlock()
	return c.LoadWords(keys)
}

// LoadWords 原子地替换词表并重建匹配器（全程持写锁）。
// 键在插入前做与正文一致的归一化（NFKC/小写/去零宽），保证全角等兼容形式可匹配；
// 失败（如过滤正则非法）时保留旧词表与旧匹配器（last-known-good）。
func (c *Censor) LoadWords(words map[string]WordInfo) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.FilterRegexStr != "" {
		re, err := regexp.Compile(c.FilterRegexStr)
		if err != nil {
			return fmt.Errorf("censor: invalid filter regex: %w", err)
		}
		c.filterRegex = re
	} else {
		c.filterRegex = nil
	}

	normalized := make(map[string]WordInfo, len(words))
	for key, info := range words {
		nk := normalizeKey(key, c.CaseSensitive)
		if nk == "" {
			continue
		}
		// 归一化后冲突时保留等级更高者
		if old, ok := normalized[nk]; ok && old.Level >= info.Level {
			continue
		}
		normalized[nk] = info
	}

	builder := ahocorasick.NewTrieBuilder()
	for key := range normalized {
		builder.AddString(key)
	}
	c.SensitiveKeys = normalized
	c.matcher = builder.Build()
	return nil
}

// WordsSnapshot 返回词表快照，供外部遍历，避免与重载竞争。
func (c *Censor) WordsSnapshot() map[string]WordInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string]WordInfo, len(c.SensitiveKeys))
	for k, v := range c.SensitiveKeys {
		out[k] = v
	}
	return out
}

type CheckResult struct {
	HighestLevel Level
	Hits         []Hit
}

// Words 返回 命中词(Origin) -> 最高等级。
func (r CheckResult) Words() map[string]Level {
	out := make(map[string]Level, len(r.Hits))
	for _, h := range r.Hits {
		if l, ok := out[h.Word]; !ok || h.Level > l {
			out[h.Word] = h.Level
		}
	}
	return out
}

func (c *Censor) Check(content string) CheckResult {
	return c.CheckWithDrops(content, nil)
}

// CheckWithDrops 在匹配前额外丢弃 extra 正则覆盖的片段，返回的 span 仍为原文 rune 偏移。
func (c *Censor) CheckWithDrops(content string, extra []*regexp.Regexp) CheckResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	res := CheckResult{HighestLevel: Ignore}
	if c.matcher == nil {
		return res
	}

	drop := buildDropMask(content, c.filterRegex)
	for _, re := range extra {
		m := buildDropMask(content, re)
		for i := range m {
			if m[i] {
				drop[i] = true
			}
		}
	}
	n := normalize(content, c.CaseSensitive, drop)
	matches := c.matcher.MatchString(n.text)
	if len(matches) == 0 {
		return res
	}
	origRunes := []rune(content)
	for _, mt := range matches {
		word := mt.MatchString()
		info, ok := c.SensitiveKeys[word]
		if !ok {
			continue
		}
		bpos := int(mt.Pos())
		blen := len(word)
		if bpos < 0 || bpos+blen >= len(n.byteToRune) {
			continue
		}
		runeStart := n.byteToRune[bpos]
		runeEnd := n.byteToRune[bpos+blen]
		if runeStart < 0 || runeEnd > len(n.runeToOrig) || runeStart >= runeEnd {
			continue
		}
		start := n.runeToOrig[runeStart]
		end := n.runeToOrig[runeEnd-1] + 1
		// 命中末字符若带组合标记（分解形式），一并纳入命中范围
		for end < len(origRunes) && isCombining(origRunes[end]) {
			end++
		}
		res.Hits = append(res.Hits, Hit{Span: Span{Start: start, End: end}, Word: info.Origin, Level: info.Level})
		res.HighestLevel = HigherLevel(res.HighestLevel, info.Level)
	}
	return res
}

func generateFileKey() string {
	key, _ := nanoid.Generate("0123456789abcdef", 16)
	return key
}
