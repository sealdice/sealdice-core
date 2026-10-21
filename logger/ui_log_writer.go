package logger

import (
	"encoding/json"
	"io"
	"sync"
	"time"

	"go.uber.org/zap/zapcore"

	"sealdice-core/utils/epoch"
)

const (
	logLimitDefault   = 100
	timeFormatISO8601 = "2006-01-02T15:04:05.000Z0700"
)

type LogItem struct {
	Seq    uint64  `json:"seq"`
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
	nextSeq     uint64
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
	return copyLogItems(l.Items)
}

// SnapshotAfter 返回 Seq 大于 afterSeq 的日志。
// reset 为 true 时表示 afterSeq 已不在保留窗口内、序号来自旧进程，
// 或客户端尚未建立游标；调用方应使用返回的完整保留日志替换本地状态。
func (l *UIWriter) SnapshotAfter(afterSeq uint64, afterEpoch string) (items []*LogItem, latestSeq uint64, currentEpoch string, reset bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	latestSeq = l.nextSeq
	currentEpoch = epoch.Current()
	if afterSeq == 0 || afterEpoch != currentEpoch {
		return copyLogItems(l.Items), latestSeq, currentEpoch, true
	}
	if afterSeq > latestSeq {
		return copyLogItems(l.Items), latestSeq, currentEpoch, true
	}
	if len(l.Items) > 0 && afterSeq+1 < l.Items[0].Seq {
		return copyLogItems(l.Items), latestSeq, currentEpoch, true
	}

	start := 0
	for start < len(l.Items) && l.Items[start].Seq <= afterSeq {
		start++
	}
	return copyLogItems(l.Items[start:]), latestSeq, currentEpoch, false
}

func copyLogItems(items []*LogItem) []*LogItem {
	result := make([]*LogItem, len(items))
	for i, item := range items {
		if item == nil {
			continue
		}
		itemCopy := *item
		result[i] = &itemCopy
	}
	return result
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
		l.nextSeq++
		item.Seq = l.nextSeq
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
