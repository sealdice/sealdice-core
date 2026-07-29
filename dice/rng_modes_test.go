//nolint:testpackage
package dice

import (
	"strings"
	"testing"
	"time"

	wr "github.com/mroth/weightedrand"
	ds "github.com/sealdice/dicescript"
)

type stubDiceSource struct {
	values []uint64
	index  int
}

func (s *stubDiceSource) Uint64() uint64 {
	if len(s.values) == 0 {
		return 0
	}
	if s.index >= len(s.values) {
		return s.values[len(s.values)-1]
	}
	v := s.values[s.index]
	s.index++
	return v
}

func newRandomModeTestCtx(t *testing.T) (*MsgContext, func()) {
	t.Helper()

	ctx, cleanup := newPlayerObjectTestCtx(t, "coc7", nil)
	ctx.CreateVmIfNotExists()
	ctx.vm = nil
	return ctx, cleanup
}

func TestCreateVmIfNotExists_UsesContextDiceSource(t *testing.T) {
	ctx, cleanup := newRandomModeTestCtx(t)
	defer cleanup()

	src := &stubDiceSource{values: []uint64{0}}
	ctx.diceRandSrc = src

	ctx.CreateVmIfNotExists()

	if ctx.vm == nil {
		t.Fatal("expected vm to be created")
	}
	if ctx.vm.RandSrc != src {
		t.Fatalf("expected vm.RandSrc to reuse bound dice source, got %#v", ctx.vm.RandSrc)
	}
	if ctx._v1Rand != src {
		t.Fatalf("expected legacy source to mirror bound dice source, got %#v", ctx._v1Rand)
	}
}

func TestMsgContextRoll64_UsesBoundDiceSource(t *testing.T) {
	ctx, cleanup := newRandomModeTestCtx(t)
	defer cleanup()

	src := &stubDiceSource{values: []uint64{5}}
	ctx.diceRandSrc = src

	got := ctx.Roll64(6)
	if got != 6 {
		t.Fatalf("Roll64() = %d, want 6", got)
	}
}

func TestNewPCGDiceSource_IsStateful(t *testing.T) {
	src, err := newDiceSourceForMode(DiceRandomModePCG)
	if err != nil {
		t.Fatalf("newDiceSourceForMode(pcq) error = %v", err)
	}
	if _, ok := src.(ds.StatefulDiceSource); !ok {
		t.Fatalf("expected pcg source to implement StatefulDiceSource, got %T", src)
	}
}

func TestCmdTi_UsesContextDiceSource(t *testing.T) {
	ctx, cleanup := newRandomModeTestCtx(t)
	defer cleanup()

	src := &stubDiceSource{values: []uint64{0, 0}}
	ctx.diceRandSrc = src

	ext := ctx.Dice.ExtFind("coc7", false)
	if ext == nil {
		t.Fatal("expected coc7 extension to exist")
	}
	cmd := ext.CmdMap["ti"]
	if cmd == nil {
		t.Fatal("expected ti command to exist")
	}

	ret := cmd.Solve(ctx, newGroupMsg(ctx.Group.GroupID, ctx.Player.UserID, ".ti"), &CmdArgs{Command: "ti"})
	if !ret.Solved {
		t.Fatal("expected ti command to solve")
	}

	got, ok := VarGetValueStr(ctx, "$t表达式文本")
	if !ok || got != "1D10=1" {
		t.Fatalf("expected expression text from bound dice source, got %q (exists=%v)", got, ok)
	}
	if src.index != 2 {
		t.Fatalf("expected ti command to consume 2 values from bound source, got %d", src.index)
	}
}

func TestCmdRsr_UsesContextDiceSource(t *testing.T) {
	ctx, cleanup := newRandomModeTestCtx(t)
	defer cleanup()

	src := &stubDiceSource{values: []uint64{0}}
	ctx.diceRandSrc = src

	ext := ctx.Dice.ExtFind("fun", false)
	if ext == nil {
		t.Fatal("expected fun extension to exist")
	}
	cmd := ext.CmdMap["rsr"]
	if cmd == nil {
		t.Fatal("expected rsr command to exist")
	}

	ret := cmd.Solve(ctx, newGroupMsg(ctx.Group.GroupID, ctx.Player.UserID, ".rsr 1"), &CmdArgs{
		Command: "rsr",
		Args:    []string{"1"},
	})
	if !ret.Solved {
		t.Fatal("expected rsr command to solve")
	}

	adapter, ok := ctx.EndPoint.Adapter.(*mockPlatformAdapter)
	if !ok {
		t.Fatalf("unexpected adapter type %T", ctx.EndPoint.Adapter)
	}
	reply, ok := adapter.waitForMsg(time.Second)
	if !ok {
		t.Fatal("expected rsr command to emit a reply")
	}
	if !strings.Contains(reply, "成功度:0/1") {
		t.Fatalf("expected rsr reply to reflect bound source, got %q", reply)
	}
	if src.index != 1 {
		t.Fatalf("expected rsr command to consume 1 value from bound source, got %d", src.index)
	}
}

func TestDiceFormatTmpl_UsesContextDiceSource(t *testing.T) {
	ctx, cleanup := newRandomModeTestCtx(t)
	defer cleanup()

	src := &stubDiceSource{values: []uint64{0}}
	ctx.diceRandSrc = src

	chooser, err := wr.NewChooser(
		wr.Choice{Item: "A", Weight: 1},
		wr.Choice{Item: "B", Weight: 1},
	)
	if err != nil {
		t.Fatalf("NewChooser() error = %v", err)
	}
	ctx.Dice.TextMap["测试:随机"] = chooser

	got := DiceFormatTmpl(ctx, "测试:随机")
	if got != "A" {
		t.Fatalf("DiceFormatTmpl() = %q, want %q", got, "A")
	}
	if src.index != 1 {
		t.Fatalf("expected template selection to consume 1 value from bound source, got %d", src.index)
	}
}

func TestExecuteDeck_UsesContextDiceSource(t *testing.T) {
	ctx, cleanup := newRandomModeTestCtx(t)
	defer cleanup()

	src := &stubDiceSource{values: []uint64{0, 0, 0, 0}}
	ctx.diceRandSrc = src

	deckInfo := &DeckInfo{
		DeckItems: map[string][]string{
			"test": {"A", "B"},
		},
	}

	got, err := executeDeck(ctx, deckInfo, "test", false)
	if err != nil {
		t.Fatalf("executeDeck() error = %v", err)
	}
	if got != "A" {
		t.Fatalf("executeDeck() = %q, want %q", got, "A")
	}
	if src.index != 1 {
		t.Fatalf("expected normal deck draw to consume 1 value from bound source, got %d", src.index)
	}

	src.index = 0
	got, err = executeDeck(ctx, deckInfo, "test", true)
	if err != nil {
		t.Fatalf("executeDeck(shuffle) error = %v", err)
	}
	if got != "A" && got != "B" {
		t.Fatalf("executeDeck(shuffle) returned unexpected item %q", got)
	}
	if src.index == 0 {
		t.Fatal("expected shuffle deck draw to consume values from bound source")
	}
}
