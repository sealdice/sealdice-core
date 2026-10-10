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
	r, err := p.Check(t.Context(), provider.Request{Text: "a bad b"})
	if err != nil || r == nil {
		t.Fatalf("r=%v err=%v", r, err)
	}
	if r.Level != censor.Danger || len(r.Spans) != 1 || r.Spans[0].Start != 2 {
		t.Fatalf("unexpected %+v", r)
	}
}

func TestLocalAC_NoHit(t *testing.T) {
	p := provider.NewLocalAC(newCensor(map[string]censor.Level{"bad": censor.Danger}))
	r, err := p.Check(t.Context(), provider.Request{Text: "clean"})
	if err != nil || r != nil {
		t.Fatalf("want nil,nil got r=%v err=%v", r, err)
	}
}

func TestEngine_Merge(t *testing.T) {
	local := provider.NewLocalAC(newCensor(map[string]censor.Level{"bad": censor.Warning}))
	verdict := &fakeVerdict{}
	e := provider.NewEngine(local, verdict)
	m, err := e.Check(t.Context(), provider.Request{Text: "a bad b"})
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

func (f *fakeVerdict) Name() string                    { return "fake" }
func (f *fakeVerdict) Capability() provider.Capability { return provider.VerdictOnly }
func (f *fakeVerdict) Reload() error                   { return nil }
func (f *fakeVerdict) Check(_ context.Context, _ provider.Request) (*provider.Result, error) {
	return &provider.Result{Level: censor.Danger, Reason: "semantic"}, nil
}
