package provider_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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

func TestHTTPProvider_VerdictOnlyIgnoresSpans(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"level": 3, "reason": "x", "spans": [][2]int{{1, 3}},
		})
	}))
	defer srv.Close()

	p := provider.NewHTTP(provider.HTTPConfig{
		Name: "remote", URL: srv.URL, Capability: provider.VerdictOnly, TimeoutMs: 1000, FailMode: provider.FailOpen,
	})
	r, err := p.Check(context.Background(), provider.Request{Text: "abcde"})
	if err != nil || r == nil {
		t.Fatalf("r=%v err=%v", r, err)
	}
	if len(r.Spans) != 0 {
		t.Fatalf("verdict-only should carry no spans, got %+v", r.Spans)
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
}
