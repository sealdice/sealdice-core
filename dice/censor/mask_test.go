//nolint:testpackage // tests access unexported helpers
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
