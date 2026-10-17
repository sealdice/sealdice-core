//nolint:testpackage // Identical business probes run on master, mmap and PR 1837.
package dice

import (
	"os"
	"testing"

	"go.uber.org/zap"
)

func comparisonDice(t *testing.T, body string) *Dice {
	t.Helper()
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("data/decks", 0o755); err != nil {
		t.Fatal(err)
	}
	comparisonWrite(t, "data/decks/probe.json", body)
	d := &Dice{Logger: zap.NewNop().Sugar()}
	d.Config = NewConfig(d)
	if closer, ok := any(d).(interface{ CloseDecks() }); ok {
		t.Cleanup(closer.CloseDecks)
	}
	DeckReload(d)
	if len(d.DeckList) != 1 || !d.DeckList[0].Enable {
		t.Fatalf("load rejected: %+v", d.DeckList)
	}
	return d
}

func comparisonWrite(t *testing.T, filename, body string) {
	t.Helper()
	if err := os.WriteFile(filename, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func comparisonDraw(t *testing.T, ctx *MsgContext, shuffle bool) string {
	t.Helper()
	ctx.deckDepth = 0
	ok, text, err := deckDraw(ctx, "g", shuffle)
	if !ok || err != nil {
		t.Fatalf("draw ok=%v error=%v", ok, err)
	}
	return text
}

func TestDeckComparisonProbe(t *testing.T) {
	t.Run("filename-fallback", func(t *testing.T) {
		d := comparisonDice(t, `{"g":["ok"]}`)
		if d.DeckList[0].Name != "probe.json" {
			t.Fatalf("name=%q; want filename", d.DeckList[0].Name)
		}
	})
	t.Run("object-extension", func(t *testing.T) {
		d := comparisonDice(t, `{"extension":{"normal":1},"g":["ok"]}`)
		if got := comparisonDraw(t, &MsgContext{Dice: d}, false); got != "ok" {
			t.Fatal(got)
		}
	})
	t.Run("mixed-array", func(t *testing.T) {
		d := comparisonDice(t, `{"g":[1,null,{},"ok",[2]]}`)
		if got := comparisonDraw(t, &MsgContext{Dice: d}, false); got != "ok" {
			t.Fatal(got)
		}
	})
	t.Run("zero-and-default", func(t *testing.T) {
		d := comparisonDice(t, `{"g":["::0::BAD","ok"]}`)
		ctx := &MsgContext{Dice: d, diceRandSrc: newPCGDiceSource(41)}
		for range 32 {
			if got := comparisonDraw(t, ctx, false); got != "ok" {
				t.Fatalf("zero weight selected: %q", got)
			}
		}
	})
	t.Run("wide-weight", func(t *testing.T) {
		d := comparisonDice(t, `{"g":["::4294967296::WIDE","::1::small"]}`)
		ctx := &MsgContext{Dice: d, diceRandSrc: newPCGDiceSource(41)}
		for range 32 {
			if got := comparisonDraw(t, ctx, false); got != "WIDE" {
				t.Fatalf("wide weight draw: %q", got)
			}
		}
	})
	t.Run("online-update", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("online update panicked: %v", r)
			}
		}()
		d := comparisonDice(t, `{"g":["old"]}`)
		comparisonWrite(t, "update.json", `{"g":["new"]}`)
		if err := d.DeckUpdate(d.DeckList[0], "update.json"); err != nil {
			t.Fatal(err)
		}
		if got := comparisonDraw(t, &MsgContext{Dice: d}, false); got != "new" {
			t.Fatalf("stale update: %q", got)
		}
	})
	t.Run("reload-context-pool", func(t *testing.T) {
		d := comparisonDice(t, `{"g":["o0","o1","o2","o3","o4","o5","o6","o7"]}`)
		ctx := &MsgContext{Dice: d, diceRandSrc: newPCGDiceSource(41)}
		_ = comparisonDraw(t, ctx, true)
		comparisonWrite(t, "data/decks/probe.json", `{"g":["new"]}`)
		DeckReload(d)
		for i := range 8 {
			if got := comparisonDraw(t, ctx, true); got != "new" {
				t.Fatalf("draw %d stale pool: %q", i, got)
			}
		}
	})
}
