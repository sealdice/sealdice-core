package provider

import (
	"context"
	"regexp"

	"sealdice-core/dice/censor"
)

type Capability int

const (
	// VerdictOnly 只给结论，不提供脱敏位置。
	VerdictOnly Capability = iota
	// SpanCapable 给出结论并支持脱敏位置。
	SpanCapable
)

// Request 是提供给 Provider 的检测输入。
type Request struct {
	Text string
	Drop []*regexp.Regexp // 附加丢弃正则（如海豹码/CQ码）
}

// Result 是一次检测命中。Spans 仅在 Capability==SpanCapable 时有效。
type Result struct {
	Level  censor.Level
	Reason string
	Spans  []censor.Span
}

type Provider interface {
	Name() string
	Capability() Capability
	Check(ctx context.Context, req Request) (*Result, error)
	Reload() error
}
