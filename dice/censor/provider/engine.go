package provider

import (
	"context"

	"sealdice-core/dice/censor"
)

// Merged 是 Engine 合并多个 Provider 后的结果。
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
