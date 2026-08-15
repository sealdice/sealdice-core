package realtime

import (
	"sync"
	"time"
)

const publishTimeout = 2 * time.Second

type Event struct {
	Name    string
	Payload any
}

type Bus struct {
	mu          sync.RWMutex
	nextID      uint64
	subscribers map[uint64]chan Event
}

func NewBus() *Bus {
	return &Bus{
		subscribers: map[uint64]chan Event{},
	}
}

func (b *Bus) Subscribe(buffer int) (<-chan Event, func()) {
	if buffer <= 0 {
		buffer = 1
	}

	ch := make(chan Event, buffer)

	b.mu.Lock()
	id := b.nextID
	b.nextID++
	b.subscribers[id] = ch
	b.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			b.mu.Lock()
			delete(b.subscribers, id)
			close(ch)
			b.mu.Unlock()
		})
	}

	return ch, unsubscribe
}

func (b *Bus) Publish(evt Event) {
	b.mu.RLock()
	subscribers := make([]chan Event, 0, len(b.subscribers))
	for _, ch := range b.subscribers {
		subscribers = append(subscribers, ch)
	}
	b.mu.RUnlock()

	for _, ch := range subscribers {
		if evt.Name == EventLogsAppend {
			// 日志高频，允许背压丢帧，避免拖慢连接状态事件。
			trySend(ch, evt)
			continue
		}

		// 连接状态事件不应因日志刷屏被丢弃；等待一个短超时。
		sendWithTimeout(ch, evt, publishTimeout)
	}
}

func trySend(ch chan Event, evt Event) {
	defer func() { _ = recover() }()
	select {
	case ch <- evt:
	default:
	}
}

func sendWithTimeout(ch chan Event, evt Event, timeout time.Duration) {
	defer func() { _ = recover() }()
	select {
	case ch <- evt:
	case <-time.After(timeout):
	}
}
