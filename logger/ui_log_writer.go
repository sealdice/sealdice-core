package logger

import (
	"encoding/json"
	"io"
	"sync"
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
	LogLimit int
	Items    []*LogItem

	mu          sync.RWMutex
	subscribers map[chan *LogItem]struct{}
}

var _ io.Writer = (*UIWriter)(nil)

func NewUIWriter() *UIWriter {
	return &UIWriter{
		LogLimit:    logLimitDefault,
		Items:       make([]*LogItem, 0),
		subscribers: make(map[chan *LogItem]struct{}),
	}
}

func (l *UIWriter) Snapshot() []*LogItem {
	l.mu.RLock()
	defer l.mu.RUnlock()
	items := make([]*LogItem, len(l.Items))
	for i, item := range l.Items {
		if item == nil {
			continue
		}
		itemCopy := *item
		items[i] = &itemCopy
	}
	return items
}

func (l *UIWriter) Subscribe() (<-chan *LogItem, func()) {
	ch := make(chan *LogItem, 64)
	l.mu.Lock()
	if l.subscribers == nil {
		l.subscribers = make(map[chan *LogItem]struct{})
	}
	l.subscribers[ch] = struct{}{}
	l.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			l.mu.Lock()
			delete(l.subscribers, ch)
			close(ch)
			l.mu.Unlock()
		})
	}
	return ch, unsubscribe
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
		item := &LogItem{
			Level:  a.Level.String(),
			Module: a.Module,
			TS:     float64(ts.Unix()),
			Caller: "",
			Msg:    a.Msg,
		}
		l.mu.Lock()
		l.Items = append(l.Items, item)
		if l.LogLimit == 0 {
			l.LogLimit = logLimitDefault
		}
		if len(l.Items) > l.LogLimit {
			l.Items = l.Items[1:]
		}
		for ch := range l.subscribers {
			select {
			case ch <- item:
			default:
			}
		}
		l.mu.Unlock()
	}
	return len(p), nil
}
