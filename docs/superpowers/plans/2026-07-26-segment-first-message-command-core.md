# Segment-First Message And Command Core Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `Message.Segment` the canonical core representation, remove CQ-string round-trips from command parsing, and preserve existing Go/JS string APIs through compatibility wrappers.

**Architecture:** Normalize messages once at explicit input boundaries, then keep `Segment` as the internal source of truth. Build a segment projection layer for command parsing, wrap a new internal parsed-command model with the existing `CmdArgs` API, and keep legacy string fields as semantically equivalent compatibility views. This borrows `smallseal`'s adapter-normalization and segment/text projection direction, but adds stronger migration boundaries for sealdice-core's existing JS plugin ecosystem.

**Tech Stack:** Go, existing `message` package, existing `dice` package, goja `jsbind` exposure, current Go tests

---

## Decisions Locked By Discussion

- `Segment` is the only canonical message representation inside the core.
- `Message.Message`, `CmdArgs.RawText`, `CmdArgs.RawArgs`, and `CmdArgs.CleanArgs` remain available, but are compatibility views.
- Compatibility means semantic equivalence, not byte-for-byte preservation of CQ parameter order, whitespace normalization, or platform-specific string formatting.
- JS compatibility is strong at input boundaries: JS-created `Message` values that only fill `message` still work when passed into supported Go APIs.
- `CmdArgs` becomes a wrapper over a richer internal command model. Existing top-level fields and helper methods remain usable.
- First implementation wave focuses on unifying the foundation. New JS segment-aware methods are documented as an extension point, not part of the first API surface.

## Relation To smallseal

Use these ideas from `smallseal`:

- Adapter ingress should produce internal message segments.
- `Message.Message` can be derived from segments as a text view.
- Non-text segments can be projected into text with placeholders when a text parser is still needed.

Do not copy these parts directly:

- `smallseal` still has a mostly string-shaped `CommandParse(rawCmd string, ...)`; this plan requires command parsing to start from canonical segments.
- `smallseal` uses simple `$N` placeholders; use internal-only placeholders that cannot be confused with normal user text.
- `smallseal` does not carry the same JS compatibility burden; sealdice-core needs an explicit wrapper and deprecation policy.

## File Structure

- Create: `message/segment_text.go`
  Responsibility: define internal segment projection and rebuilding helpers for text parsers.
- Create: `message/segment_text_test.go`
  Responsibility: verify projection, placeholder collision behavior, and non-text segment preservation.
- Create: `dice/message_normalize.go`
  Responsibility: centralize inbound message normalization and compatibility text projection.
- Create: `dice/message_normalize_test.go`
  Responsibility: verify boundary normalization for adapter-style and JS-created messages.
- Modify: `dice/im_session.go`
  Responsibility: replace broad lazy `ensureSegment` / `ensureMessage` use with explicit normalization at execution boundaries.
- Modify: `dice/utils.go`
  Responsibility: normalize JS-created messages in `CreateTempCtx`.
- Modify: `dice/cmd_parse.go`
  Responsibility: introduce internal parsed-command model, keep `CmdArgs` as wrapper, and remove command parsing via CQ-string serialization.
- Modify: `dice/cmd_parse_test.go`
  Responsibility: extend current command tests to cover segment-first parsing, reparsing, and compatibility views.
- Create: `docs/message-segment-compatibility.md`
  Responsibility: document soft-deprecated fields, compatibility guarantees, and preferred replacement concepts.

## Compatibility And Soft Deprecation Policy

These fields remain available but become compatibility views:

| Field / Method | Current role | New role | Replacement concept |
| --- | --- | --- | --- |
| `Message.Message` / JS `msg.message` | Primary string message body | Derived compatibility text view | `Message.Segment` |
| `Message.Segment` / JS `msg.segment` | Partial segment support | Canonical message body | Same field, with documented element shapes |
| `CmdArgs.RawText` / JS `cmdArgs.rawText` | Original command text | Derived compatibility command text | internal parsed-command projection |
| `CmdArgs.RawArgs` / JS `cmdArgs.rawArgs` | Original argument text | Derived compatibility argument text | internal parsed-command argument span |
| `CmdArgs.CleanArgs` / JS `cmdArgs.cleanArgs` | Normalized argument text | Derived text view over parsed arguments | existing helper methods first, segment-aware APIs in a separate design |
| `CommandParse(rawCmd, ...)` | String parser entry | Legacy parser entry for old callers | segment-first parser used by execution path |
| `CmdArgs.RevokeExecuteTimesParse` | Reparse by converting `RawText` to segments | Reproject and reparse from canonical command data | internal parsed-command model |

Guarantees during the soft-deprecation period:

- Existing JS code reading `msg.message`, `cmdArgs.rawText`, `cmdArgs.rawArgs`, and `cmdArgs.cleanArgs` keeps working semantically.
- Existing JS code using `cmdArgs.getArgN`, `cmdArgs.getKwarg`, `cmdArgs.isArgEqual`, and `cmdArgs.chopPrefixToArgsWith` keeps working semantically.
- Existing JS code constructing `seal.newMessage()` and setting only `message` is normalized at supported input boundaries.
- The implementation does not promise exact whitespace, exact CQ argument order, or exact unknown-segment rendering.

Non-goals for this plan:

- No broad JS segment-aware method family in this implementation wave.
- No removal of legacy fields in this implementation wave.
- No full adapter rewrite across every platform in one change. Adapters can continue to supply `Message.Message`; the boundary normalizer converts it once.

## Task 1: Add Internal Segment Text Projection

**Files:**
- Create: `message/segment_text.go`
- Create: `message/segment_text_test.go`

- [ ] **Step 1: Write failing tests for segment projection**

Create `message/segment_text_test.go`:

```go
package message

import "testing"

func TestSegmentProjectionPreservesNonTextSegments(t *testing.T) {
	image := &ImageElement{URL: "https://example.invalid/image.png"}
	segments := []IMessageElement{
		&TextElement{Content: ".foo before "},
		image,
		&TextElement{Content: " after"},
	}

	projection := ProjectSegmentsToText(segments)

	if projection.Text == "" {
		t.Fatal("projection text should not be empty")
	}
	if projection.Text == ".foo before  after" {
		t.Fatal("projection should include a placeholder for non-text segments")
	}
	if len(projection.Placeholders) != 1 {
		t.Fatalf("placeholder count = %d, want 1", len(projection.Placeholders))
	}

	rebuilt := projection.ToSegments()
	if len(rebuilt) != 3 {
		t.Fatalf("rebuilt segment count = %d, want 3", len(rebuilt))
	}
	if rebuilt[1] != image {
		t.Fatalf("non-text segment was not preserved: %#v", rebuilt[1])
	}
}

func TestSegmentProjectionTreatsUserPlaceholderTextAsText(t *testing.T) {
	segments := []IMessageElement{
		&TextElement{Content: ".foo \x1fseg:1\x1f"},
	}

	projection := ProjectSegmentsToText(segments)
	rebuilt := projection.ToSegments()

	if len(rebuilt) != 1 {
		t.Fatalf("rebuilt segment count = %d, want 1", len(rebuilt))
	}
	text, ok := rebuilt[0].(*TextElement)
	if !ok {
		t.Fatalf("rebuilt segment type = %T, want *TextElement", rebuilt[0])
	}
	if text.Content != ".foo \x1fseg:1\x1f" {
		t.Fatalf("text content = %q", text.Content)
	}
}
```

- [ ] **Step 2: Run projection tests and confirm failure**

Run: `go test ./message -run 'TestSegmentProjection' -count=1`

Expected: FAIL with `undefined: ProjectSegmentsToText`.

- [ ] **Step 3: Implement projection helpers**

Create `message/segment_text.go` with this public package API:

```go
package message

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

const segmentPlaceholderPrefix = "\x1fseg:"
const segmentPlaceholderSuffix = "\x1f"

type SegmentText struct {
	Text         string
	Placeholders map[int]IMessageElement
}

func ProjectSegmentsToText(segments []IMessageElement) SegmentText {
	var builder strings.Builder
	placeholders := map[int]IMessageElement{}
	for idx, elem := range segments {
		if text, ok := elem.(*TextElement); ok {
			builder.WriteString(text.Content)
			continue
		}
		placeholderID := idx + 1
		placeholders[placeholderID] = elem
		builder.WriteString(fmt.Sprintf("%s%d%s", segmentPlaceholderPrefix, placeholderID, segmentPlaceholderSuffix))
	}
	if len(placeholders) == 0 {
		placeholders = nil
	}
	return SegmentText{Text: builder.String(), Placeholders: placeholders}
}

func (st SegmentText) ToSegments() []IMessageElement {
	if st.Text == "" {
		return nil
	}
	var result []IMessageElement
	var builder strings.Builder
	for i := 0; i < len(st.Text); {
		if !strings.HasPrefix(st.Text[i:], segmentPlaceholderPrefix) {
			r, size := utf8.DecodeRuneInString(st.Text[i:])
			builder.WriteRune(r)
			i += size
			continue
		}
		start := i + len(segmentPlaceholderPrefix)
		end := strings.Index(st.Text[start:], segmentPlaceholderSuffix)
		if end < 0 {
			r, size := utf8.DecodeRuneInString(st.Text[i:])
			builder.WriteRune(r)
			i += size
			continue
		}
		end += start
		id, err := strconv.Atoi(st.Text[start:end])
		elem, ok := st.Placeholders[id]
		if err != nil || !ok || elem == nil {
			builder.WriteString(st.Text[i : end+len(segmentPlaceholderSuffix)])
			i = end + len(segmentPlaceholderSuffix)
			continue
		}
		if builder.Len() > 0 {
			result = append(result, &TextElement{Content: builder.String()})
			builder.Reset()
		}
		result = append(result, elem)
		i = end + len(segmentPlaceholderSuffix)
	}
	if builder.Len() > 0 {
		result = append(result, &TextElement{Content: builder.String()})
	}
	return result
}
```

- [ ] **Step 4: Run projection tests and confirm success**

Run: `go test ./message -run 'TestSegmentProjection' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit projection helper**

```bash
git add message/segment_text.go message/segment_text_test.go
git commit -m "feat(message): add segment text projection"
```

## Task 2: Centralize Inbound Message Normalization

**Files:**
- Create: `dice/message_normalize.go`
- Create: `dice/message_normalize_test.go`
- Modify: `dice/im_session.go`
- Modify: `dice/utils.go`

- [ ] **Step 1: Write failing tests for boundary normalization**

Create `dice/message_normalize_test.go`:

```go
package dice

import (
	"testing"

	"sealdice-core/message"
)

func TestNormalizeIncomingMessageBuildsSegmentsFromLegacyMessage(t *testing.T) {
	msg := &Message{
		Platform: "QQ",
		Message:  ".r 1d6",
	}

	NormalizeIncomingMessage(msg)

	if len(msg.Segment) != 1 {
		t.Fatalf("segment count = %d, want 1", len(msg.Segment))
	}
	text, ok := msg.Segment[0].(*message.TextElement)
	if !ok || text.Content != ".r 1d6" {
		t.Fatalf("unexpected segment: %#v", msg.Segment[0])
	}
	if msg.Message != ".r 1d6" {
		t.Fatalf("message view = %q, want original text", msg.Message)
	}
}

func TestNormalizeIncomingMessageDerivesMessageViewFromSegments(t *testing.T) {
	msg := &Message{
		Platform: "QQ",
		Segment: []message.IMessageElement{
			&message.TextElement{Content: ".r "},
			&message.ImageElement{URL: "https://example.invalid/a.png"},
		},
	}

	NormalizeIncomingMessage(msg)

	if msg.Message == "" {
		t.Fatal("message compatibility view should be derived")
	}
	if len(msg.Segment) != 2 {
		t.Fatalf("segment count = %d, want 2", len(msg.Segment))
	}
}
```

- [ ] **Step 2: Run normalization tests and confirm failure**

Run: `go test ./dice -run 'TestNormalizeIncomingMessage' -count=1`

Expected: FAIL with `undefined: NormalizeIncomingMessage`.

- [ ] **Step 3: Implement normalization helper**

Create `dice/message_normalize.go`:

```go
package dice

import (
	"sealdice-core/message"
)

type MessageNormalizeMode int

const (
	NormalizeModeBoundary MessageNormalizeMode = iota
	NormalizeModeCompatibilityView
)

func NormalizeIncomingMessage(msg *Message) {
	if msg == nil {
		return
	}
	if len(msg.Segment) == 0 && msg.Message != "" {
		msg.Segment = message.ConvertStringMessage(msg.Message)
	}
	if msg.Message == "" && len(msg.Segment) > 0 {
		msg.Message = MessageSegmentsToCompatibilityText(msg.Segment)
	}
}

func MessageSegmentsToCompatibilityText(segments []message.IMessageElement) string {
	_, text := convertSealMsgToMessageChain(segments)
	return text
}
```

- [ ] **Step 4: Replace execution boundary normalization**

Modify `dice/im_session.go`:

- Keep `ensureSegment` and `ensureMessage` only as compatibility wrappers if existing tests or call sites require their names.
- Change `executeCore` to call `NormalizeIncomingMessage(msg)` once at the top.
- Update comments to state that normalization is an input-boundary operation.

Use this shape:

```go
func ensureSegment(msg *Message) {
	NormalizeIncomingMessage(msg)
}

func ensureMessage(msg *Message) {
	NormalizeIncomingMessage(msg)
}

func (s *IMSession) executeCore(ep *EndPointInfo, msg *Message, runInSync bool) {
	d := s.Parent

	NormalizeIncomingMessage(msg)

	mctx := &MsgContext{}
	// existing function body continues unchanged
}
```

- [ ] **Step 5: Normalize JS-created messages in `CreateTempCtx`**

Modify `dice/utils.go`:

```go
func CreateTempCtx(ep *EndPointInfo, msg *Message) *MsgContext {
	if ep == nil {
		panic("CreateTempCtx: endpoint is nil")
	}
	if msg == nil {
		panic("CreateTempCtx: message is nil")
	}

	NormalizeIncomingMessage(msg)

	session := ep.Session
	// existing function body continues unchanged
}
```

- [ ] **Step 6: Run normalization tests**

Run: `go test ./dice -run 'TestNormalizeIncomingMessage|TestCommandParseUnified|TestUnknownCommand|TestComputeAtUID|TestParseAtInfo' -count=1`

Expected: PASS.

- [ ] **Step 7: Commit normalization boundary**

```bash
git add dice/message_normalize.go dice/message_normalize_test.go dice/im_session.go dice/utils.go
git commit -m "refactor(message): normalize inbound messages at boundaries"
```

## Task 3: Introduce Internal Parsed Command Model

**Files:**
- Modify: `dice/cmd_parse.go`
- Modify: `dice/cmd_parse_test.go`

- [ ] **Step 1: Write failing tests for segment-first command parsing**

Append to `dice/cmd_parse_test.go`:

```go
func TestCommandParseNewPreservesNonTextSegmentProjection(t *testing.T) {
	d := &Dice{CommandPrefix: DefaultConfig.CommandPrefix}
	d.CmdMap = CmdMapCls{"img": &CmdItemInfo{}}
	session := &IMSession{Parent: d}
	ep := &EndPointInfo{EndPointInfoBase: EndPointInfoBase{UserID: "QQ:1000", Platform: "QQ"}}
	image := &message.ImageElement{URL: "https://example.invalid/a.png"}
	msg := &Message{
		Platform: "QQ",
		Segment: []message.IMessageElement{
			&message.TextElement{Content: ".img before "},
			image,
			&message.TextElement{Content: " after"},
		},
	}
	ctx := &MsgContext{Session: session, EndPoint: ep, Dice: d}

	cmdArgs := CommandParseNew(ctx, msg)
	if cmdArgs == nil {
		t.Fatal("CommandParseNew returned nil")
	}
	if cmdArgs.Command != "img" {
		t.Fatalf("Command = %q, want img", cmdArgs.Command)
	}
	if cmdArgs.parsed == nil {
		t.Fatal("internal parsed command should be set")
	}
	if len(cmdArgs.parsed.Projection.Placeholders) != 1 {
		t.Fatalf("placeholder count = %d, want 1", len(cmdArgs.parsed.Projection.Placeholders))
	}
}
```

- [ ] **Step 2: Run segment-first parser test and confirm failure**

Run: `go test ./dice -run 'TestCommandParseNewPreservesNonTextSegmentProjection' -count=1`

Expected: FAIL because `CmdArgs.parsed` and the internal model do not exist.

- [ ] **Step 3: Add internal command model**

Modify `dice/cmd_parse.go` near `CmdArgs`:

```go
type ParsedCommand struct {
	Command             string
	Args                []string
	Kwargs              []*Kwarg
	At                  []*AtInfo
	RawArgs             string
	CleanArgs           string
	RawText             string
	Projection          message.SegmentText
	Prefix              string
	PlatformPrefix      string
	SpecialExecuteTimes int
	IsSpaceBeforeArgs   bool
}
```

Add an unexported pointer to `CmdArgs`:

```go
type CmdArgs struct {
	Command                    string    `jsbind:"command"                  json:"command"`
	Args                       []string  `jsbind:"args"                     json:"args"`
	Kwargs                     []*Kwarg  `jsbind:"kwargs"                   json:"kwargs"`
	At                         []*AtInfo `jsbind:"at"                       json:"atInfo"`
	RawArgs                    string    `jsbind:"rawArgs"                  json:"rawArgs"`
	AmIBeMentioned             bool      `jsbind:"amIBeMentioned"           json:"amIBeMentioned"`
	AmIBeMentionedFirst        bool      `jsbind:"amIBeMentionedFirst"      json:"amIBeMentionedFirst"`
	SomeoneBeMentionedButNotMe bool      `json:"someoneBeMentionedButNotMe"`
	IsSpaceBeforeArgs          bool      `json:"isSpaceBeforeArgs"`
	CleanArgs                  string    `jsbind:"cleanArgs"`
	SpecialExecuteTimes        int       `jsbind:"specialExecuteTimes"`
	RawText                    string    `jsbind:"rawText"`
	prefixStr                  string
	platformPrefix             string
	uidForAtInfo               string
	parsed                     *ParsedCommand

	MentionedOtherDice bool
	CleanArgsChopRest  string
}
```

- [ ] **Step 4: Add wrapper synchronization helper**

Add this helper in `dice/cmd_parse.go`:

```go
func (cmdArgs *CmdArgs) applyParsed(parsed *ParsedCommand) *CmdArgs {
	if parsed == nil {
		return nil
	}
	cmdArgs.parsed = parsed
	cmdArgs.Command = parsed.Command
	cmdArgs.Args = append(cmdArgs.Args[:0], parsed.Args...)
	cmdArgs.Kwargs = append(cmdArgs.Kwargs[:0], parsed.Kwargs...)
	cmdArgs.At = parsed.At
	cmdArgs.RawArgs = parsed.RawArgs
	cmdArgs.CleanArgs = parsed.CleanArgs
	cmdArgs.RawText = parsed.RawText
	cmdArgs.IsSpaceBeforeArgs = parsed.IsSpaceBeforeArgs
	cmdArgs.SpecialExecuteTimes = parsed.SpecialExecuteTimes
	cmdArgs.prefixStr = parsed.Prefix
	cmdArgs.platformPrefix = parsed.PlatformPrefix
	return cmdArgs
}
```

- [ ] **Step 5: Run parser test and confirm it still fails**

Run: `go test ./dice -run 'TestCommandParseNewPreservesNonTextSegmentProjection' -count=1`

Expected: FAIL because `commandParseNew` has not populated `parsed`.

- [ ] **Step 6: Commit internal model scaffold**

```bash
git add dice/cmd_parse.go dice/cmd_parse_test.go
git commit -m "refactor(command): add parsed command wrapper model"
```

## Task 4: Replace CQ-String Extraction In Command Parser

**Files:**
- Modify: `dice/cmd_parse.go`
- Modify: `dice/cmd_parse_test.go`

- [ ] **Step 1: Add failing tests for literal CQ text and reparsing**

Append to `dice/cmd_parse_test.go`:

```go
func TestCommandParseNewKeepsLiteralCQTextAsTextWhenSegmentInputIsText(t *testing.T) {
	d := &Dice{CommandPrefix: DefaultConfig.CommandPrefix}
	d.CmdMap = CmdMapCls{"echo": &CmdItemInfo{}}
	session := &IMSession{Parent: d}
	ep := &EndPointInfo{EndPointInfoBase: EndPointInfoBase{UserID: "QQ:1000", Platform: "QQ"}}
	msg := &Message{
		Platform: "QQ",
		Segment: []message.IMessageElement{
			&message.TextElement{Content: ".echo [CQ:at,qq=12345]"},
		},
	}
	ctx := &MsgContext{Session: session, EndPoint: ep, Dice: d}

	cmdArgs := CommandParseNew(ctx, msg)
	if cmdArgs == nil {
		t.Fatal("CommandParseNew returned nil")
	}
	if len(cmdArgs.At) != 0 {
		t.Fatalf("At count = %d, want 0 for literal CQ text", len(cmdArgs.At))
	}
	if cmdArgs.CleanArgs != "[CQ:at,qq=12345]" {
		t.Fatalf("CleanArgs = %q", cmdArgs.CleanArgs)
	}
}

func TestRevokeExecuteTimesParseDoesNotOverwriteSegmentsFromRawText(t *testing.T) {
	d := &Dice{CommandPrefix: DefaultConfig.CommandPrefix}
	d.CmdMap = CmdMapCls{"r": &CmdItemInfo{}}
	session := &IMSession{Parent: d}
	ep := &EndPointInfo{EndPointInfoBase: EndPointInfoBase{UserID: "QQ:1000", Platform: "QQ"}}
	image := &message.ImageElement{URL: "https://example.invalid/a.png"}
	msg := &Message{
		Platform: "QQ",
		Segment: []message.IMessageElement{
			&message.TextElement{Content: "3#.r "},
			image,
		},
	}
	ctx := &MsgContext{Session: session, EndPoint: ep, Dice: d}

	cmdArgs := new(CmdArgs).commandParseNew(ctx, msg, false)
	if cmdArgs == nil {
		t.Fatal("commandParseNew returned nil")
	}
	cmdArgs.RevokeExecuteTimesParse(ctx, msg)

	if len(msg.Segment) != 2 || msg.Segment[1] != image {
		t.Fatalf("message segments were overwritten: %#v", msg.Segment)
	}
	if cmdArgs.SpecialExecuteTimes != 3 {
		t.Fatalf("SpecialExecuteTimes = %d, want 3", cmdArgs.SpecialExecuteTimes)
	}
}
```

- [ ] **Step 2: Run tests and confirm failure**

Run: `go test ./dice -run 'TestCommandParseNewKeepsLiteralCQText|TestRevokeExecuteTimesParseDoesNotOverwriteSegmentsFromRawText' -count=1`

Expected: FAIL because current parsing still relies on CQ-string conversion and `RevokeExecuteTimesParse` overwrites `msg.Segment`.

- [ ] **Step 3: Replace segment extraction with projection**

Modify `commandParseNew` to use:

```go
projection := message.ProjectSegmentsToText(msg.Segment)
rawCmd := strings.ReplaceAll(projection.Text, "\r\n", "\n")
```

Remove the command parser's dependency on `extractResultFromSegments` for the new path. Keep `extractResultFromSegments` temporarily only if old call sites still compile, and mark it unexported legacy inside comments without using it in `commandParseNew`.

- [ ] **Step 4: Build `ParsedCommand` in `buildCmdArgs`**

Change `buildCmdArgs` signature to accept `projection message.SegmentText`:

```go
func buildCmdArgs(cmdArgs *CmdArgs, matched, restText, rawCmd string, projection message.SegmentText,
	specialExecuteTimes int, prefixStr, platform string, isSpaceBeforeArgs bool) *CmdArgs
```

Inside it, build `ParsedCommand` and call `cmdArgs.applyParsed(parsed)`:

```go
parsed := &ParsedCommand{
	Command:             m[1],
	RawArgs:             m[2],
	Args:                a.Args,
	Kwargs:              a.Kwargs,
	RawText:             rawCmd,
	CleanArgs:           strings.TrimSpace(strings.Join(a.Args, " ")),
	Projection:          projection,
	IsSpaceBeforeArgs:   isSpaceBeforeArgs,
	SpecialExecuteTimes: specialExecuteTimes,
	Prefix:              prefixStr,
	PlatformPrefix:      platform,
}
return cmdArgs.applyParsed(parsed)
```

- [ ] **Step 5: Update `commandParseNew` caller**

Use this return call:

```go
return buildCmdArgs(cmdArgs, matched, restText, rawCmd, projection, specialExecuteTimes, prefixStr, msg.Platform, isSpaceBeforeArgs)
```

- [ ] **Step 6: Rework `RevokeExecuteTimesParse`**

Replace the body with:

```go
func (cmdArgs *CmdArgs) RevokeExecuteTimesParse(ctx *MsgContext, msg *Message) {
	NormalizeIncomingMessage(msg)
	cmdArgs.commandParseNew(ctx, msg, true)
}
```

This keeps canonical segments intact and reparses from projection.

- [ ] **Step 7: Run segment parser tests**

Run: `go test ./dice -run 'TestCommandParseNewPreservesNonTextSegmentProjection|TestCommandParseNewKeepsLiteralCQText|TestRevokeExecuteTimesParseDoesNotOverwriteSegmentsFromRawText|TestCommandParseUnified|TestUnknownCommand' -count=1`

Expected: PASS.

- [ ] **Step 8: Commit segment-first parser**

```bash
git add dice/cmd_parse.go dice/cmd_parse_test.go
git commit -m "refactor(command): parse commands from segment projection"
```

## Task 5: Preserve Legacy CmdArgs Helper Semantics

**Files:**
- Modify: `dice/cmd_parse.go`
- Modify: `dice/cmd_parse_test.go`

- [ ] **Step 1: Add wrapper compatibility tests**

Append to `dice/cmd_parse_test.go`:

```go
func TestCmdArgsLegacyMethodsDelegateToParsedModel(t *testing.T) {
	cmdArgs := &CmdArgs{}
	cmdArgs.applyParsed(&ParsedCommand{
		Command:   "test",
		Args:      []string{"on", "target"},
		Kwargs:    []*Kwarg{{Name: "flag", Value: "true", ValueExists: true, AsBool: true}},
		RawArgs:   "on target --flag=true",
		CleanArgs: "on target",
		RawText:   ".test on target --flag=true",
	})

	if !cmdArgs.IsArgEqual(1, "on") {
		t.Fatal("IsArgEqual should read wrapper args")
	}
	if cmdArgs.GetArgN(2) != "target" {
		t.Fatalf("GetArgN(2) = %q", cmdArgs.GetArgN(2))
	}
	if cmdArgs.GetKwarg("flag") == nil {
		t.Fatal("GetKwarg should find flag")
	}
	if cmdArgs.GetRestArgsFrom(1) != "on target" {
		t.Fatalf("GetRestArgsFrom(1) = %q", cmdArgs.GetRestArgsFrom(1))
	}
}
```

- [ ] **Step 2: Run wrapper test**

Run: `go test ./dice -run 'TestCmdArgsLegacyMethodsDelegateToParsedModel' -count=1`

Expected: PASS after Task 4. If it fails, keep the existing helper method outputs semantically unchanged by synchronizing wrapper fields in `applyParsed`.

- [ ] **Step 3: Add a compatibility guard for `ChopPrefixToArgsWith`**

Append to `dice/cmd_parse_test.go`:

```go
func TestCmdArgsChopPrefixToArgsWithMaintainsLegacyFields(t *testing.T) {
	cmdArgs := &CmdArgs{}
	cmdArgs.applyParsed(&ParsedCommand{
		Command:   "bot",
		Args:      []string{"on123", "extra"},
		RawArgs:   "on123 extra",
		CleanArgs: "on123 extra",
		RawText:   ".bot on123 extra",
	})

	if !cmdArgs.ChopPrefixToArgsWith("on", "off") {
		t.Fatal("ChopPrefixToArgsWith should match prefix")
	}
	if cmdArgs.GetArgN(1) != "on" || cmdArgs.GetArgN(2) != "123" {
		t.Fatalf("unexpected args after chop: %#v", cmdArgs.Args)
	}
	if cmdArgs.CleanArgsChopRest != "123 extra" {
		t.Fatalf("CleanArgsChopRest = %q", cmdArgs.CleanArgsChopRest)
	}
}
```

- [ ] **Step 4: Run compatibility tests**

Run: `go test ./dice -run 'TestCmdArgsLegacyMethodsDelegateToParsedModel|TestCmdArgsChopPrefixToArgsWithMaintainsLegacyFields' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit wrapper compatibility**

```bash
git add dice/cmd_parse.go dice/cmd_parse_test.go
git commit -m "test(command): preserve legacy cmd args wrapper behavior"
```

## Task 6: Document Compatibility And Soft Deprecations

**Files:**
- Create: `docs/message-segment-compatibility.md`
- Modify if present in a future branch: JS API declaration docs for `Message` and `CmdArgs`

- [ ] **Step 1: Create compatibility document**

Create `docs/message-segment-compatibility.md`:

```markdown
# Message Segment Compatibility

## Canonical Model

`Message.Segment` is the canonical message body inside the core. Command parsing, reparsing, and future mixed-content features should use segment-backed data.

`Message.Message` remains available for compatibility with existing Go call sites and JS plugins. It is a derived text view and should not be treated as the source of truth inside new core code.

## Soft-Deprecated Compatibility Views

The following fields remain available but are compatibility views:

| API | Compatibility status | Preferred direction |
| --- | --- | --- |
| `msg.message` | Soft-deprecated text view | `msg.segment` |
| `cmdArgs.rawText` | Soft-deprecated text view | parsed command data |
| `cmdArgs.rawArgs` | Soft-deprecated text view | parsed command data |
| `cmdArgs.cleanArgs` | Soft-deprecated text view | existing helper methods first, segment-aware APIs when introduced |

Compatibility is semantic. The project does not guarantee exact whitespace, exact CQ parameter order, or exact formatting for unknown segment text.

## Supported Legacy Behavior

JS plugins may continue to read `msg.message` and the existing `cmdArgs` string fields.

JS plugins may continue to construct `seal.newMessage()` with only `message` filled when passing the message into supported boundary APIs such as `createTempCtx`.

Existing `CmdArgs` helper methods remain available:

- `getArgN`
- `getKwarg`
- `isArgEqual`
- `eatPrefixWith`
- `chopPrefixToArgsWith`
- `getRestArgsFrom`

## New Code Guidance

New core code should consume `Message.Segment` and avoid reparsing compatibility text.

New command parser code should use the segment projection layer rather than constructing CQ strings.

New JS-facing segment-aware APIs should be designed as wrapper methods so the internal parsed-command model can evolve without exposing implementation details.
```

- [ ] **Step 2: Search docs for forbidden placeholders**

Run: `rg -n "T[B]D|TO[D]O|f[i]ll in|implement [l]ater|待[定]" docs/message-segment-compatibility.md docs/superpowers/plans/2026-07-26-segment-first-message-command-core.md`

Expected: no matches.

- [ ] **Step 3: Commit compatibility docs**

```bash
git add docs/message-segment-compatibility.md docs/superpowers/plans/2026-07-26-segment-first-message-command-core.md
git commit -m "docs(message): document segment compatibility migration"
```

## Task 7: Run Focused Regression Suite

**Files:**
- No source edits expected.

- [ ] **Step 1: Run message package tests**

Run: `go test ./message -count=1`

Expected: PASS.

- [ ] **Step 2: Run command parser tests**

Run: `go test ./dice -run 'TestCommandParse|TestUnknownCommand|TestIsAtMe|TestComputeAtUID|TestParseAtInfo|TestCmdArgs|TestRevokeExecuteTimesParse' -count=1`

Expected: PASS.

- [ ] **Step 3: Run execution path smoke tests**

Run: `go test ./dice -run 'TestExecuteNew' -count=1`

Expected: PASS.

- [ ] **Step 4: Run lint if available**

Run: `golangci-lint run ./dice ./message`

Expected: PASS. If the command is unavailable in the environment, record that it was not run and include the shell error in the handoff.

- [ ] **Step 5: Final commit for any regression fixes**

If Task 7 required code changes:

```bash
git add dice message docs
git commit -m "fix(command): preserve segment-first compatibility"
```

If Task 7 required no code changes, no commit is needed.

## Acceptance Criteria

- `CommandParseNew` no longer serializes segments into CQ text before parsing.
- `RevokeExecuteTimesParse` does not overwrite canonical `msg.Segment`.
- `Message.Message` remains populated where logs, censoring, and JS compatibility need a text view.
- JS-created string-only messages are normalized at supported boundaries.
- Existing `CmdArgs` fields and helper methods keep semantic compatibility.
- The compatibility document clearly marks old string fields as soft-deprecated views.
- Focused `message` and `dice` tests pass.
