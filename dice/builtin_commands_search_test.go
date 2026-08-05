//nolint:testpackage // Tests the unexported result-display decision directly.
package dice

import (
	"testing"

	"sealdice-core/dice/docengine"
)

func TestShouldShowBestHelpResult(t *testing.T) {
	newHits := func(scores ...float64) docengine.MatchCollection {
		hits := make(docengine.MatchCollection, 0, len(scores))
		for _, score := range scores {
			hits = append(hits, &docengine.MatchResult{Score: score})
		}
		return hits
	}

	tests := []struct {
		name      string
		hits      docengine.MatchCollection
		bestTitle string
		query     string
		want      bool
	}{
		{name: "no result", want: false},
		{name: "single result", hits: newHits(0.1), want: true},
		{name: "exact title", hits: newHits(10, 9.9), bestTitle: "法术", query: "法术", want: true},
		{name: "relative gap at threshold", hits: newHits(12, 9), want: true},
		{name: "relative gap below threshold", hits: newHits(12, 9.01), want: false},
		{name: "tied scores", hits: newHits(12, 12), want: false},
		{name: "non-positive best score", hits: newHits(0, 0), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldShowBestHelpResult(tt.hits, tt.bestTitle, tt.query, 0.25)
			if got != tt.want {
				t.Fatalf("shouldShowBestHelpResult() = %v, want %v", got, tt.want)
			}
		})
	}
}
