# 敏感词 Provider 化与脱敏 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复敏感词核心 Bug（大小写、最长匹配），用纯 Go Aho-Corasick 返回命中 span，引入可插拔 Provider 管线（localAC + httpProvider），并在对外发送路径按 span 脱敏。

**Architecture:** 检测与处置分离。`dice/censor` 负责归一化、AC 匹配、span 输出与遮罩；`dice/censor/provider` 定义 Provider/Engine；`dice/dice_censor.go` 的 CensorManager 组装引擎并执行策略（Mask/Block/Warn）；`dice/im_helpers.go` 对外发送路径接入脱敏。归一化时对被丢弃字符（过滤正则、海豹码/CQ码）建立 `norm→orig` 偏移映射，保证 span 可翻译回原文。

**Tech Stack:** Go 1.25，`github.com/BobuSumisu/aho-corasick v1.0.3`（MIT，纯 Go），`golang.org/x/text/unicode/norm`。

**Spec:** `docs/superpowers/specs/2026-10-10-censor-provider-masking-design.md`

---

## File Structure

- `dice/censor/normalize.go`（新建）：归一化 + 偏移映射。
- `dice/censor/censor.go`（修改）：`Censor` 用 AC 匹配；`Check` 返回 `[]Hit`；`addWord` 大小写修复；`Load` 用 `Compile`。
- `dice/censor/mask.go`（新建）：`MaskSpans`。
- `dice/censor/trie.go`（删除）：被 AC 库取代。
- `dice/censor/trie_internal_test.go`（删除）。
- `dice/censor/censor_test.go`（修改）：改为 span 断言。
- `dice/censor/provider/provider.go`（新建）：`Provider`/`Request`/`Result`/`Capability`。
- `dice/censor/provider/localac.go`（新建）：`LocalAC`。
- `dice/censor/provider/http.go`（新建）：`HTTPProvider`。
- `dice/censor/provider/engine.go`（新建）：`Engine`/`Merged`。
- `dice/dice_censor.go`（修改）：Manager 持有 Engine；`CensorMsg` 产出脱敏内容。
- `dice/im_helpers.go`（修改）：对外发送接入 Mask。
- `dice/dice_config.go`（修改）：`CensorConfig` 增加 `CensorProviders`。
- `dice/config.go`（修改）：默认模板 + SubType 注册。

测试文件与实现文件同目录，命名 `*_test.go`。

---

## Task 1: 归一化与偏移映射

**Files:**
- Create: `dice/censor/normalize.go`
- Test: `dice/censor/normalize_test.go`

- [ ] **Step 1: 写失败测试**

```go
// dice/censor/normalize_test.go
package censor

import (
	"regexp"
	"testing"
)

func TestNormalize_LowerAndOffsets(t *testing.T) {
	n := normalize("HeLLo", false, nil)
	if n.text != "hello" {
		t.Fatalf("text=%q want hello", n.text)
	}
	if len(n.runeToOrig) != 5 {
		t.Fatalf("runeToOrig len=%d want 5", len(n.runeToOrig))
	}
	for i, o := range n.runeToOrig {
		if o != i {
			t.Fatalf("runeToOrig[%d]=%d want %d", i, o, i)
		}
	}
}

func TestNormalize_CaseSensitiveKeepsCase(t *testing.T) {
	n := normalize("HeLLo", true, nil)
	if n.text != "HeLLo" {
		t.Fatalf("text=%q want HeLLo", n.text)
	}
}

func TestNormalize_DropsZeroWidth(t *testing.T) {
	n := normalize("a\u200bb", false, nil)
	if n.text != "ab" {
		t.Fatalf("text=%q want ab", n.text)
	}
	if len(n.runeToOrig) != 2 || n.runeToOrig[1] != 2 {
		t.Fatalf("runeToOrig=%v want [0 2]", n.runeToOrig)
	}
}

func TestNormalize_FullWidthFold(t *testing.T) {
	n := normalize("ＡＢ", false, nil)
	if n.text != "ab" {
		t.Fatalf("text=%q want ab", n.text)
	}
}

func TestBuildDropMask(t *testing.T) {
	re := regexp.MustCompile(`\s+`)
	mask := buildDropMask("a b", re)
	if len(mask) != 3 || mask[1] != true {
		t.Fatalf("mask=%v want index1 true", mask)
	}
}

func TestNormalize_DropMaskSkips(t *testing.T) {
	drop := []bool{false, true, false}
	n := normalize("a b", false, drop)
	if n.text != "ab" {
		t.Fatalf("text=%q want ab", n.text)
	}
}

func TestByteToRune(t *testing.T) {
	n := normalize("中文", false, nil)
	// "中" 3 bytes -> rune 0; "文" 3 bytes -> rune 1
	if n.byteToRune[0] != 0 || n.byteToRune[3] != 1 || n.byteToRune[6] != 2 {
		t.Fatalf("byteToRune=%v", n.byteToRune)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./dice/censor/ -run 'TestNormalize|TestBuildDropMask|TestByteToRune' -v`
Expected: FAIL（`undefined: normalize` 等）

- [ ] **Step 3: 实现**

```go
// dice/censor/normalize.go
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
```

- [ ] **Step 4: 运行确认通过**

Run: `go test ./dice/censor/ -run 'TestNormalize|TestBuildDropMask|TestByteToRune' -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add dice/censor/normalize.go dice/censor/normalize_test.go
git commit -m "feat(censor): 新增归一化与 norm->orig 偏移映射"
```

---

## Task 2: AC 匹配 + span 版 Check + 大小写修复

**Files:**
- Modify: `dice/censor/censor.go`
- Delete: `dice/censor/trie.go`, `dice/censor/trie_internal_test.go`
- Modify: `dice/censor/censor_test.go`

- [ ] **Step 1: 加依赖**

Run: `go get github.com/BobuSumisu/aho-corasick@v1.0.3 && go mod tidy`
Expected: `go.mod` 出现该依赖。

- [ ] **Step 2: 改写测试（先失败）**

将 `dice/censor/censor_test.go` 中的 `*_Check_*` 用例改为断言 `Hits`（span 与 Word），并新增最长匹配与大小写用例。删除对 `result.SensitiveWords` 的引用，删除两个 `BenchmarkTrie_*`。

```go
// 追加到 dice/censor/censor_test.go
func TestCensor_Check_LongestMatch(t *testing.T) {
	c := newTestCensor(map[string]Level{
		"色情":  Warning,
		"色情片": Danger,
	})
	res := c.Check("这里有色情片内容")
	if res.HighestLevel != Danger {
		t.Fatalf("want Danger, got %v", res.HighestLevel)
	}
	found := false
	for _, h := range res.Hits {
		if h.Word == "色情片" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected hit 色情片, got %+v", res.Hits)
	}
}

func TestCensor_Check_SpanOffset(t *testing.T) {
	c := newTestCensor(map[string]Level{"bad": Danger})
	res := c.Check("xx bad yy")
	if len(res.Hits) != 1 {
		t.Fatalf("want 1 hit, got %+v", res.Hits)
	}
	h := res.Hits[0]
	if h.Span.Start != 3 || h.Span.End != 6 {
		t.Fatalf("span=%+v want {3 6}", h.Span)
	}
}

func TestCensor_Check_CaseInsensitive(t *testing.T) {
	c := newTestCensor(map[string]Level{"badword": Danger})
	res := c.Check("contains BADWORD here")
	if res.HighestLevel != Danger {
		t.Fatalf("want Danger, got %v", res.HighestLevel)
	}
}
```

`newTestCensor`（已在文件内）保持：向 `SensitiveKeys` 写小写键后 `Load()`。

- [ ] **Step 3: 运行确认失败**

Run: `go test ./dice/censor/ -run TestCensor_Check -v`
Expected: FAIL（`res.SensitiveWords` 不存在 / 类型不匹配 / 漏报）

- [ ] **Step 4: 实现**

编辑 `dice/censor/censor.go`：

删除 `t` 字段与 `newTire` 使用，新增 `Span`、`Hit`，改 `CheckResult`，改 `addWord`/`Load`/`Check`。保留文件顶部常量、`LevelText`、`HigherLevel`、文件解析、`addWord` 的拼音逻辑。

```go
import (
	// 现有 import 基础上：
	"fmt"
	"regexp"
	"sort" // 若已在别处使用则保留
	ahocorasick "github.com/BobuSumisu/aho-corasick"
)

type Span struct {
	Start, End int // 原文 rune 偏移，半开区间
}

type Hit struct {
	Span  Span
	Word  string // 原始词（Origin）
	Level Level
}

type Censor struct {
	CaseSensitive  bool
	MatchPinyin    bool
	FilterRegexStr string

	SensitiveKeys map[string]WordInfo
	matcher       *ahocorasick.Trie
	filterRegex   *regexp.Regexp
}
```

`addWord` 改为：

```go
func (c *Censor) addWord(word string, level Level, counter *FileCounter) {
	key := strings.TrimSpace(word)
	counter[level]++
	if c.CaseSensitive {
		c.SensitiveKeys[key] = WordInfo{Level: level}
		return
	}
	key = strings.ToLower(key)
	if c.MatchPinyin {
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
```

`Load` 改为：

```go
func (c *Censor) Load() (err error) {
	if c.FilterRegexStr != "" {
		re, e := regexp.Compile(c.FilterRegexStr)
		if e != nil {
			return fmt.Errorf("censor: invalid filter regex: %w", e)
		}
		c.filterRegex = re
	} else {
		c.filterRegex = nil
	}
	builder := ahocorasick.NewTrieBuilder()
	for key := range c.SensitiveKeys {
		builder.AddString(key)
	}
	c.matcher = builder.Build()
	return nil
}
```

`CheckResult` 与 `Check`：

```go
type CheckResult struct {
	HighestLevel Level
	Hits         []Hit
}

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

func (c *Censor) CheckWithDrops(content string, extra []*regexp.Regexp) CheckResult {
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
	for _, mt := range c.matcher.MatchString(n.text) {
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
		rs := n.byteToRune[bpos]
		re := n.byteToRune[bpos+blen]
		if rs < 0 || re > len(n.runeToOrig) || rs >= re {
			continue
		}
		start := n.runeToOrig[rs]
		end := n.runeToOrig[re-1] + 1
		res.Hits = append(res.Hits, Hit{Span: Span{Start: start, End: end}, Word: info.Origin, Level: info.Level})
		res.HighestLevel = HigherLevel(res.HighestLevel, info.Level)
	}
	return res
}
```

删除 `dice/censor/trie.go` 与 `dice/censor/trie_internal_test.go`。

- [ ] **Step 5: 运行确认通过**

Run: `go test ./dice/censor/ -v`
Expected: PASS

- [ ] **Step 6: 提交**

```bash
git add dice/censor/censor.go dice/censor/censor_test.go dice/censor/trie.go dice/censor/trie_internal_test.go go.mod go.sum
git commit -m "refactor(censor): 改用 Aho-Corasick 返回命中 span 并修复大小写/最长匹配"
```

---

## Task 3: 遮罩 MaskSpans

**Files:**
- Create: `dice/censor/mask.go`
- Test: `dice/censor/mask_test.go`

- [ ] **Step 1: 写失败测试**

```go
// dice/censor/mask_test.go
package censor

import "testing"

func TestMaskSpans_Basic(t *testing.T) {
	got := MaskSpans("黑夜总会来临", []Span{{Start: 1, End: 4}}, "■")
	if got != "黑■■■来临" {
		t.Fatalf("got %q", got)
	}
}

func TestMaskSpans_MergeOverlap(t *testing.T) {
	got := MaskSpans("abcd", []Span{{0, 2}, {1, 3}}, "■")
	if got != "■■■d" {
		t.Fatalf("got %q", got)
	}
}

func TestMaskSpans_EmptyPlaceholderDeletes(t *testing.T) {
	got := MaskSpans("abcd", []Span{{1, 3}}, "")
	if got != "ad" {
		t.Fatalf("got %q", got)
	}
}

func TestMaskSpans_NoSpans(t *testing.T) {
	if got := MaskSpans("abc", nil, "■"); got != "abc" {
		t.Fatalf("got %q", got)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./dice/censor/ -run TestMaskSpans -v`
Expected: FAIL（`undefined: MaskSpans`）

- [ ] **Step 3: 实现**

```go
// dice/censor/mask.go
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
```

- [ ] **Step 4: 运行确认通过**

Run: `go test ./dice/censor/ -run TestMaskSpans -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add dice/censor/mask.go dice/censor/mask_test.go
git commit -m "feat(censor): 新增按 span 遮罩的 MaskSpans"
```

---

## Task 4: Provider 类型、LocalAC 与 Engine

**Files:**
- Create: `dice/censor/provider/provider.go`
- Create: `dice/censor/provider/localac.go`
- Create: `dice/censor/provider/engine.go`
- Test: `dice/censor/provider/provider_test.go`

- [ ] **Step 1: 写失败测试**

```go
// dice/censor/provider/provider_test.go
package provider_test

import (
	"context"
	"testing"

	"sealdice-core/dice/censor"
	"sealdice-core/dice/censor/provider"
)

func newCensor(words map[string]censor.Level) *censor.Censor {
	c := &censor.Censor{SensitiveKeys: map[string]censor.WordInfo{}}
	for w, lv := range words {
		c.SensitiveKeys[w] = censor.WordInfo{Level: lv, Origin: w}
	}
	_ = c.Load()
	return c
}

func TestLocalAC_SpanResult(t *testing.T) {
	p := provider.NewLocalAC(newCensor(map[string]censor.Level{"bad": censor.Danger}))
	r, err := p.Check(context.Background(), provider.Request{Text: "a bad b"})
	if err != nil || r == nil {
		t.Fatalf("r=%v err=%v", r, err)
	}
	if r.Level != censor.Danger || len(r.Spans) != 1 || r.Spans[0].Start != 2 {
		t.Fatalf("unexpected %+v", r)
	}
}

func TestEngine_Merge(t *testing.T) {
	local := provider.NewLocalAC(newCensor(map[string]censor.Level{"bad": censor.Warning}))
	verdict := &fakeVerdict{}
	e := provider.NewEngine(local, verdict)
	m, err := e.Check(context.Background(), provider.Request{Text: "a bad b"})
	if err != nil {
		t.Fatal(err)
	}
	if !m.SpanHit || len(m.Spans) != 1 {
		t.Fatalf("want span hit, got %+v", m)
	}
	if len(m.Verdicts) != 1 {
		t.Fatalf("want 1 verdict, got %+v", m.Verdicts)
	}
	if m.Level != censor.Danger {
		t.Fatalf("level=%v want Danger", m.Level)
	}
}

type fakeVerdict struct{}

func (f *fakeVerdict) Name() string                { return "fake" }
func (f *fakeVerdict) Capability() provider.Capability { return provider.VerdictOnly }
func (f *fakeVerdict) Reload() error               { return nil }
func (f *fakeVerdict) Check(_ context.Context, _ provider.Request) (*provider.Result, error) {
	return &provider.Result{Level: censor.Danger, Reason: "semantic"}, nil
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./dice/censor/provider/ -v`
Expected: FAIL（包不存在）

- [ ] **Step 3: 实现**

```go
// dice/censor/provider/provider.go
package provider

import (
	"context"

	"sealdice-core/dice/censor"
)

type Capability int

const (
	VerdictOnly Capability = iota
	SpanCapable
)

type Request struct {
	Text string
	Drop []*regexp.Regexp // 附加丢弃正则（如海豹码/CQ码）
}

type Result struct {
	Level  censor.Level
	Reason string
	Spans  []censor.Span
}

type Provider interface {
	Name() string
	Capability() Capability
	Check(ctx context.Context, req Request) (*Result, error)
	Reload() error
}
```

（`provider.go` import 需含 `regexp`。）

```go
// dice/censor/provider/localac.go
package provider

import (
	"context"
	"strings"

	"sealdice-core/dice/censor"
)

type LocalAC struct {
	c *censor.Censor
}

func NewLocalAC(c *censor.Censor) *LocalAC { return &LocalAC{c: c} }

func (l *LocalAC) Name() string             { return "localAC" }
func (l *LocalAC) Capability() Capability   { return SpanCapable }
func (l *LocalAC) Reload() error            { return l.c.Load() }

func (l *LocalAC) Check(_ context.Context, req Request) (*Result, error) {
	res := l.c.CheckWithDrops(req.Text, req.Drop)
	if res.HighestLevel <= censor.Ignore || len(res.Hits) == 0 {
		return nil, nil
	}
	spans := make([]censor.Span, 0, len(res.Hits))
	seen := map[string]bool{}
	var reasons []string
	for _, h := range res.Hits {
		spans = append(spans, h.Span)
		if !seen[h.Word] {
			seen[h.Word] = true
			reasons = append(reasons, h.Word)
		}
	}
	return &Result{Level: res.HighestLevel, Reason: strings.Join(reasons, "|"), Spans: spans}, nil
}
```

```go
// dice/censor/provider/engine.go
package provider

import (
	"context"

	"sealdice-core/dice/censor"
)

type Merged struct {
	Level    censor.Level
	SpanHit  bool
	Spans    []censor.Span
	Verdicts []*Result
	Reasons  []string
}

type Engine struct {
	providers []Provider
}

func NewEngine(ps ...Provider) *Engine { return &Engine{providers: ps} }

func (e *Engine) Providers() []Provider { return e.providers }

func (e *Engine) Check(ctx context.Context, req Request) (*Merged, error) {
	m := &Merged{Level: censor.Ignore}
	for _, p := range e.providers {
		r, err := p.Check(ctx, req)
		if err != nil {
			return m, err
		}
		if r == nil {
			continue
		}
		m.Level = censor.HigherLevel(m.Level, r.Level)
		m.Reasons = append(m.Reasons, r.Reason)
		if p.Capability() == SpanCapable {
			m.SpanHit = true
			m.Spans = append(m.Spans, r.Spans...)
		} else {
			m.Verdicts = append(m.Verdicts, r)
		}
	}
	return m, nil
}
```

- [ ] **Step 4: 运行确认通过**

Run: `go test ./dice/censor/provider/ -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add dice/censor/provider/
git commit -m "feat(censor): 新增 Provider 抽象、LocalAC 与 Engine"
```

---

## Task 5: HTTPProvider

**Files:**
- Create: `dice/censor/provider/http.go`
- Test: `dice/censor/provider/http_test.go`

- [ ] **Step 1: 写失败测试**

```go
// dice/censor/provider/http_test.go
package provider_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"sealdice-core/dice/censor"
	"sealdice-core/dice/censor/provider"
)

func TestHTTPProvider_Spans(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"level":  3,
			"reason": "涉政",
			"spans":  [][2]int{{1, 3}},
		})
	}))
	defer srv.Close()

	p := provider.NewHTTP(provider.HTTPConfig{
		Name: "remote", URL: srv.URL, Capability: provider.SpanCapable, TimeoutMs: 1000, FailMode: provider.FailOpen,
	})
	r, err := p.Check(context.Background(), provider.Request{Text: "abcde"})
	if err != nil || r == nil {
		t.Fatalf("r=%v err=%v", r, err)
	}
	if r.Level != censor.Warning || len(r.Spans) != 1 || r.Spans[0] != (censor.Span{Start: 1, End: 3}) {
		t.Fatalf("unexpected %+v", r)
	}
}

func TestHTTPProvider_FailOpen(t *testing.T) {
	p := provider.NewHTTP(provider.HTTPConfig{
		Name: "remote", URL: "http://127.0.0.1:1", Capability: provider.VerdictOnly, TimeoutMs: 100, FailMode: provider.FailOpen,
	})
	r, err := p.Check(context.Background(), provider.Request{Text: "x"})
	if err != nil || r != nil {
		t.Fatalf("fail-open want nil,nil got r=%v err=%v", r, err)
	}
}

func TestHTTPProvider_FailClosed(t *testing.T) {
	p := provider.NewHTTP(provider.HTTPConfig{
		Name: "remote", URL: "http://127.0.0.1:1", Capability: provider.VerdictOnly, TimeoutMs: 100, FailMode: provider.FailClosed,
	})
	r, err := p.Check(context.Background(), provider.Request{Text: "x"})
	if err != nil || r == nil {
		t.Fatalf("fail-closed want result got r=%v err=%v", r, err)
	}
	if r.Level != censor.Danger {
		t.Fatalf("level=%v want Danger", r.Level)
	}
	_ = time.Now
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./dice/censor/provider/ -run TestHTTPProvider -v`
Expected: FAIL（`undefined: provider.NewHTTP`）

- [ ] **Step 3: 实现**

```go
// dice/censor/provider/http.go
package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"sealdice-core/dice/censor"
)

type FailMode int

const (
	FailOpen FailMode = iota
	FailClosed
)

type HTTPConfig struct {
	Name       string
	URL        string
	Token      string
	TimeoutMs  int
	FailMode   FailMode
	Capability Capability
}

type HTTPProvider struct {
	cfg    HTTPConfig
	client *http.Client
}

func NewHTTP(cfg HTTPConfig) *HTTPProvider {
	timeout := time.Duration(cfg.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = time.Second
	}
	return &HTTPProvider{cfg: cfg, client: &http.Client{Timeout: timeout}}
}

func (h *HTTPProvider) Name() string           { return h.cfg.Name }
func (h *HTTPProvider) Capability() Capability { return h.cfg.Capability }
func (h *HTTPProvider) Reload() error          { return nil }

type httpResp struct {
	Level  int        `json:"level"`
	Reason string     `json:"reason"`
	Spans  [][2]int   `json:"spans"`
}

func (h *HTTPProvider) Check(ctx context.Context, req Request) (*Result, error) {
	body, _ := json.Marshal(map[string]string{"text": req.Text})
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, h.cfg.URL, bytes.NewReader(body))
	if err != nil {
		return h.onErr(err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if h.cfg.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+h.cfg.Token)
	}
	resp, err := h.client.Do(httpReq)
	if err != nil {
		return h.onErr(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return h.onErr(nil)
	}
	var out httpResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return h.onErr(err)
	}
	if out.Level <= int(censor.Ignore) {
		return nil, nil
	}
	r := &Result{Level: censor.Level(out.Level), Reason: out.Reason}
	if h.cfg.Capability == SpanCapable {
		for _, s := range out.Spans {
			r.Spans = append(r.Spans, censor.Span{Start: s[0], End: s[1]})
		}
	}
	return r, nil
}

func (h *HTTPProvider) onErr(_ error) (*Result, error) {
	if h.cfg.FailMode == FailClosed {
		return &Result{Level: censor.Danger, Reason: "provider error"}, nil
	}
	return nil, nil
}
```

- [ ] **Step 4: 运行确认通过**

Run: `go test ./dice/censor/provider/ -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add dice/censor/provider/http.go dice/censor/provider/http_test.go
git commit -m "feat(censor): 新增 HTTPProvider（含 fail-open/closed）"
```

---

## Task 6: Manager 接入 Engine + 策略

**Files:**
- Modify: `dice/dice_censor.go`
- Test: `dice/dice_censor_test.go`（修改/新增）

- [ ] **Step 1: 写失败测试**

```go
// 追加到 dice/dice_censor_test.go
func TestCensorMsg_MasksOutgoing(t *testing.T) {
	d := &Dice{Logger: logger.M(), Config: &DiceConfig{}}
	d.NewCensorManager()
	d.CensorManager.Censor.SensitiveKeys["夜总会"] = censor.WordInfo{Level: censor.Danger, Origin: "夜总会"}
	_ = d.CensorManager.Censor.Load()
	d.TextMap = map[string][]TextTemplateItem{
		"核心:拦截_敏感词过滤_替换占位符": {{Text: "■", Weight: 1}},
	}
	mctx := &MsgContext{Dice: d}
	msg := &Message{MessageType: "group"}
	hit, _, _, masked := d.CensorMsg(mctx, msg, "黑夜总会来临", "黑夜总会来临")
	if !hit {
		t.Fatal("want hit")
	}
	if masked != "黑■■■来临" {
		t.Fatalf("masked=%q", masked)
	}
}
```

注意：`d.NewCensorManager` 目前调用 `cm.Load(d)`，Engine 将在 Step 3 中初始化。`TextMap` 字段名与 `TextTemplateItem` 以现有代码为准（若类型/字段名不同，按实际调整）。

- [ ] **Step 2: 运行确认失败**

Run: `go test ./dice/ -run TestCensorMsg_MasksOutgoing -v`
Expected: FAIL（`CensorMsg` 无语义返回遮罩 / Engine 未接入）

- [ ] **Step 3: 实现**

修改 `dice/dice_censor.go`：

`CensorManager` 增加 `Engine *provider.Engine`。`Load` 末尾构建引擎：

```go
cm.Engine = provider.NewEngine(provider.NewLocalAC(cm.Censor))
```

`CensorManager.Check` 改为调用引擎并把命中写入日志：

```go
func (cm *CensorManager) Check(ctx *MsgContext, msg *Message, text string) (*MsgCheckResult, []censor.Span, bool, error) {
	if cm.IsLoading {
		return nil, nil, false, errors.New("censor is loading")
	}
	drops := []*regexp.Regexp{sealCodeRe, cqCodeRe}
	m, err := cm.Engine.Check(context.Background(), provider.Request{Text: text, Drop: drops})
	if err != nil {
		return nil, nil, false, err
	}
	words := map[string]censor.Level{}
	for _, h := range m.Spans {
		_ = h
	}
	// 用 localAC 的词集写库（保持原语义）
	local := cm.Censor.CheckWithDrops(text, drops)
	if !ctx.Censored && m.Level > censor.Ignore {
		service.CensorAppend(cm.DB, ctx.MessageType, msg.Sender.UserID, msg.GroupID, msg.Message, local.Words(), int(m.Level))
	}
	count := service.CensorCount(cm.DB, msg.Sender.UserID)
	var ws []string
	for w := range local.Words() {
		ws = append(ws, w)
	}
	sort.Strings(ws)
	return &MsgCheckResult{
		UserID:            msg.Sender.UserID,
		Level:             m.Level,
		HitCounts:         count,
		CurSensitiveWords: ws,
	}, m.Spans, m.SpanHit, nil
}
```

说明：保留 `local` 二次调用以复用现有词集合/等级语义（localAC 与 Engine 均基于同一 Censor，成本低）。

`CensorMsg` 调整签名为接收单一文本，产出遮罩内容：

```go
func (d *Dice) CensorMsg(mctx *MsgContext, msg *Message, text string) (hit bool, hitWords []string, needToTerminate bool, newContent string) {
	log := d.Logger
	checkResult, spans, spanHit, err := d.CensorManager.Check(mctx, msg, text)
	if err != nil {
		log.Warnf("拦截系统出错(%s)，来自<%s>(%s)的消息跳过了检查", err.Error(), msg.Sender.Nickname, msg.Sender.UserID)
		return false, nil, false, text
	}
	newContent = text
	if checkResult.Level <= censor.Ignore {
		return false, nil, false, newContent
	}
	hit = true
	hitWords = checkResult.CurSensitiveWords
	if spanHit && len(spans) > 0 {
		placeholder := DiceFormatTmpl(mctx, "核心:拦截_敏感词过滤_替换占位符")
		newContent = censor.MaskSpans(text, spans, placeholder)
	}
	if mctx.Censored {
		return hit, hitWords, false, newContent
	}
	mctx.Censored = true
	// ……以下沿用原 CensorMsg 中阈值 + handler 逻辑（groupInfo / thresholds / handler 循环）……
	return hit, hitWords, needToTerminate, newContent
}
```

保留原阈值/handler 循环体（`dice_censor.go:237-347`）不变，仅前置了 span 脱敏与去重。

- [ ] **Step 4: 更新调用方签名**

`dice/im_session.go` 四处（`:1134`、`:1184`、`:1504`、`:1605`）调用：
```go
hit, words, needToTerminate, _ := d.CensorMsg(mctx, msg, msg.Message)
```
（删除多余的第二个文本参数。）

`dice/im_helpers.go` 三处收尾时在 Task 7 处理。

- [ ] **Step 5: 运行确认通过**

Run: `go test ./dice/ -run 'TestCensor' -v`
Expected: PASS（若既有调用方编译失败，先按 Step 4 修正）

- [ ] **Step 6: 提交**

```bash
git add dice/dice_censor.go dice/im_session.go dice/dice_censor_test.go
git commit -m "feat(censor): Manager 接入 Engine 与策略，CensorMsg 支持脱敏输出"
```

---

## Task 7: 对外发送路径接入脱敏

**Files:**
- Modify: `dice/im_helpers.go:133-152, 419-439, 491-510`

- [ ] **Step 1: 修改合并转发检查**

`TryReplyToSenderMergedForward` 内（`:133` 起）改为：

```go
if ctx.Dice.Config.EnableCensor && ctx.Dice.Config.CensorMode == OnlyOutputReply {
	for i, content := range contents {
		hit, words, needToTerminate, masked := ctx.Dice.CensorMsg(ctx, msg, content)
		if needToTerminate {
			return true
		}
		if hit {
			if masked != content {
				contents[i] = masked
			} else {
				contents[i] = DiceFormatTmpl(ctx, "核心:拦截_完全拦截_发出的消息")
			}
			ctx.Dice.Logger.Infof("拒绝回复命中敏感词「%s」的内容（合并转发）- 来自<%s>(%s)", strings.Join(words, "|"), msg.Sender.Nickname, msg.Sender.UserID)
		}
	}
}
```

- [ ] **Step 2: 修改群/个人回复检查**

`ReplyGroupRaw`（`:419` 附近）与 `ReplyPersonRaw`（`:491` 附近）改为：

```go
if d.Config.EnableCensor && d.Config.CensorMode == OnlyOutputReply {
	hit, words, needToTerminate, masked := d.CensorMsg(ctx, msg, text)
	if needToTerminate {
		return
	}
	if hit {
		if masked != text {
			text = masked
		} else {
			text = DiceFormatTmpl(ctx, "核心:拦截_完全拦截_发出的消息")
		}
		d.Logger.Infof("拒绝回复命中敏感词「%s」的内容「%s」，原消息「%s」- 来自群(%s)内<%s>(%s)", strings.Join(words, "|"), text, msg.Message, msg.GroupID, msg.Sender.Nickname, msg.Sender.UserID)
	}
}
```

（个人回复日志用 `- 来自<%s>(%s)` 版本，保持原格式。）

- [ ] **Step 3: 编译**

Run: `go build ./...`
Expected: 成功（无 `sealCodeRe`/`cqCodeRe` 未使用等错误；若这两个正则不再在 im_helpers 使用，保留其定义，移除无用 import）

- [ ] **Step 4: 运行相关测试**

Run: `go test ./dice/ -run 'TestCensor|Reply' -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add dice/im_helpers.go
git commit -m "feat(censor): 对外发送路径按 span 脱敏，无 span 时回退整条替换"
```

---

## Task 8: 配置字段与默认模板

**Files:**
- Modify: `dice/dice_config.go:248-256`
- Modify: `dice/config.go`（默认模板表 `:969`、SubType 注册表 `:1714`）

- [ ] **Step 1: 增加配置字段**

在 `CensorConfig` 末尾加：

```go
	CensorProviders []ProviderConfig `json:"censorProviders" yaml:"censorProviders"` // 外部检测 Provider（v1 不开放编辑）
```

新增类型（同文件或 `dice/dice_config.go` 内）：

```go
type ProviderConfig struct {
	Name       string `json:"name"       yaml:"name"`
	URL        string `json:"url"        yaml:"url"`
	Token      string `json:"token"      yaml:"token"`
	TimeoutMs  int    `json:"timeoutMs"  yaml:"timeoutMs"`
	FailMode   int    `json:"failMode"   yaml:"failMode"` // 0=open 1=closed
	Capability int    `json:"capability" yaml:"capability"` // 0=verdict 1=span
}
```

- [ ] **Step 2: 注册默认模板**

在 `dice/config.go:975` 的 `"拦截_完全拦截_发出的消息"` 之后，插入：

```go
			"拦截_敏感词过滤_替换占位符": {
				{"■", 1},
			},
```

在 SubType 注册表 `:1714` 对应位置插入：

```go
			"拦截_敏感词过滤_替换占位符": {
				SubType: "拦截",
			},
```

- [ ] **Step 3: 用配置构建 httpProvider**

修改 `CensorManager.Load`（或 `NewCensorManager`）在构建引擎时，按 `cm.Parent.Config.CensorProviders` 追加 HTTPProvider：

```go
var ps []provider.Provider
ps = append(ps, provider.NewLocalAC(cm.Censor))
for _, pc := range cm.Parent.Config.CensorProviders {
	ps = append(ps, provider.NewHTTP(provider.HTTPConfig{
		Name: pc.Name, URL: pc.URL, Token: pc.Token, TimeoutMs: pc.TimeoutMs,
		FailMode: provider.FailMode(pc.FailMode), Capability: provider.Capability(pc.Capability),
	}))
}
cm.Engine = provider.NewEngine(ps...)
```

- [ ] **Step 4: 编译并测试**

Run: `go build ./... && go test ./dice/... ./dice/censor/...`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add dice/dice_config.go dice/config.go dice/dice_censor.go
git commit -m "feat(censor): 新增 censorProviders 配置与脱敏占位符默认模板"
```

---

## Task 9: 端到端与并发测试

**Files:**
- Modify: `dice/dice_censor_test.go`
- Create/Modify: `dice/censor/provider/engine_race_test.go`

- [ ] **Step 1: Engine 并发重载测试**

```go
// dice/censor/provider/engine_race_test.go
package provider_test

import (
	"context"
	"sync"
	"testing"

	"sealdice-core/dice/censor"
	"sealdice-core/dice/censor/provider"
)

func TestEngine_ConcurrentCheckAndReload(t *testing.T) {
	c := newCensor(map[string]censor.Level{"bad": censor.Danger})
	p := provider.NewLocalAC(c)
	e := provider.NewEngine(p)
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 200 {
				_, _ = e.Check(context.Background(), provider.Request{Text: "a bad b"})
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 50 {
			_ = p.Reload()
		}
	}()
	wg.Wait()
}
```

注意：`Reload` 若在匹配进行时重建 `matcher` 存在数据竞争，需在 Task 4/2 中为 `Censor.Load` 与匹配加 `sync.RWMutex`（读锁匹配、写锁重建），否则 `-race` 会报错。

- [ ] **Step 2: 运行 -race**

Run: `go test -race ./dice/censor/... -run 'TestEngine_Concurrent' -v`
Expected: PASS（无 DATA RACE）。若报竞态，按 Step 1 注记为 `Censor` 加读写锁后重跑。

- [ ] **Step 3: 端到端脱敏场景**

在 `dice/dice_censor_test.go` 增加：设置词库含 `色情片`，发送文本 `这是色情片`，断言 `CensorMsg` 返回 `这是■■■`；再设置纯语义 fake provider（经 `CensorProviders`）无 spans，断言返回原文且 `hit=true`（走 Block/Warn）。

- [ ] **Step 4: 全量测试**

Run: `go test ./dice/... && go test -race ./dice/censor/...`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add dice/dice_censor_test.go dice/censor/provider/engine_race_test.go dice/censor/censor.go
git commit -m "test(censor): 端到端脱敏与并发重载测试"
```

---

## Self-Review 记录

- **Spec 覆盖**：归一化/偏移映射→Task 1；AC+span+大小写→Task 2；Masker→Task 3；Provider/Engine→Task 4；httpProvider→Task 5；策略/Masker 接入与 VerdictOnly 降级→Task 6；对外发送接入（mode A）→Task 7；配置与占位符模板→Task 8；并发与端到端→Task 9。log 脱敏为 non-goal，未建任务（符合 spec）。
- **占位符扫描**：无 TBD/TODO；每个代码步骤含完整代码。
- **类型一致性**：`provider.Provider.Check(ctx, Request)`、`provider.Result{Level,Reason,Spans}`、`censor.Span`、`censor.MaskSpans`、`provider.NewHTTP/HTTPConfig/FailMode/Capability` 在 Task 4/5/6/8 间一致。
- **已知需执行时适配点**：`TextMap`/模板类型字段名、`logger.M()` 用法、`sealCodeRe`/`cqCodeRe` 是否仍被引用——执行时以实际代码为准。
