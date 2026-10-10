package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"sealdice-core/dice/censor"
)

type FailMode int

const (
	FailOpen FailMode = iota
	FailClosed
)

type HTTPConfig struct {
	Name       string
	URL        string
	Token      string
	TimeoutMs  int
	FailMode   FailMode
	Capability Capability
}

type HTTPProvider struct {
	cfg    HTTPConfig
	client *http.Client
}

func NewHTTP(cfg HTTPConfig) *HTTPProvider {
	timeout := time.Duration(cfg.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = time.Second
	}
	return &HTTPProvider{cfg: cfg, client: &http.Client{Timeout: timeout}}
}

func (h *HTTPProvider) Name() string           { return h.cfg.Name }
func (h *HTTPProvider) Capability() Capability { return h.cfg.Capability }
func (h *HTTPProvider) Reload() error          { return nil }

type httpResp struct {
	Level  int      `json:"level"`
	Reason string   `json:"reason"`
	Spans  [][2]int `json:"spans"`
}

func (h *HTTPProvider) Check(ctx context.Context, req Request) (*Result, error) {
	body, _ := json.Marshal(map[string]string{"text": req.Text})
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, h.cfg.URL, bytes.NewReader(body))
	if err != nil {
		return h.onErr()
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if h.cfg.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+h.cfg.Token)
	}
	resp, err := h.client.Do(httpReq)
	if err != nil {
		return h.onErr()
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return h.onErr()
	}
	var out httpResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return h.onErr()
	}
	if out.Level <= int(censor.Ignore) {
		return nil, nil
	}
	r := &Result{Level: censor.Level(out.Level), Reason: out.Reason}
	if h.cfg.Capability == SpanCapable {
		for _, s := range out.Spans {
			r.Spans = append(r.Spans, censor.Span{Start: s[0], End: s[1]})
		}
	}
	return r, nil
}

func (h *HTTPProvider) onErr() (*Result, error) {
	if h.cfg.FailMode == FailClosed {
		return &Result{Level: censor.Danger, Reason: "provider error"}, nil
	}
	return nil, nil
}
