package provider

import (
	"context"
	"strings"

	"sealdice-core/dice/censor"
)

// LocalAC 是内置的本地 Aho-Corasick Provider，离线可用，支持脱敏。
type LocalAC struct {
	c *censor.Censor
}

func NewLocalAC(c *censor.Censor) *LocalAC { return &LocalAC{c: c} }

func (l *LocalAC) Name() string           { return "localAC" }
func (l *LocalAC) Capability() Capability { return SpanCapable }
func (l *LocalAC) Reload() error          { return l.c.Load() }

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
