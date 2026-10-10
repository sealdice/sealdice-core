//nolint:testpackage // tests access unexported normalize helpers
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
