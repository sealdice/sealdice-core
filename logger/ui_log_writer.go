package logger

import (
	"encoding/json"
	"io"
	"time"

	"go.uber.org/zap/zapcore"
)

const (
	logLimitDefault   = 100
	timeFormatISO8601 = "2006-01-02T15:04:05.000Z0700"
)

type LogItem struct {
	Level  string  `json:"level"`
	Module string  `json:"module"`
	TS     float64 `json:"ts"`
	Caller string  `json:"caller"`
	Msg    string  `json:"msg"`
}

type UIWriter struct {
	LogLimit int64
	Items    []*LogItem
}

var _ io.Writer = (*UIWriter)(nil)

func NewUIWriter() *UIWriter {
	return &UIWriter{
		LogLimit: logLimitDefault,
		Items:    make([]*LogItem, 0),
	}
}

func (l *UIWriter) Write(p []byte) (int, error) {
	var a struct {
		Level  zapcore.Level `json:"level"`
		Module string        `json:"module"`
		Time   string        `json:"time"`
		Msg    string        `json:"msg"`
	}
	err := json.Unmarshal(p, &a)
	if err == nil {
		ts, _ := time.Parse(timeFormatISO8601, a.Time)
		l.Items = append(l.Items, &LogItem{
			Level:  a.Level.String(),
			Module: a.Module,
			TS:     float64(ts.Unix()),
			Caller: "",
			Msg:    a.Msg,
		})
		limit := l.LogLimit
		if limit == 0 {
			l.LogLimit = logLimitDefault
			limit = logLimitDefault
		}
		if len(l.Items) > int(limit) {
			l.Items = l.Items[1:]
		}
	}
	return len(p), nil
}
