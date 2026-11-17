package realtime

import "sync"

type Event struct {
	Name    string
	Payload any
}

type Bus struct {
	mu          sync.Mutex
	nextID      uint64
	subscribers map[uint64]chan Event
}

func NewBus() *Bus {
	return &Bus{subscribers: map[uint64]chan Event{}}
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
	return ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, exists := b.subscribers[id]; exists {
			delete(b.subscribers, id)
			close(ch)
		}
	}
}

func (b *Bus) Publish(evt Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for id, ch := range b.subscribers {
		select {
		case ch <- evt:
		default:
			// Logs may drop under load. A missed state update instead closes
			// the slow stream so SSE reconnects and bootstraps a fresh snapshot.
			// Never block other subscribers or send concurrently with close.
			if evt.Name != EventLogsAppend {
				delete(b.subscribers, id)
				close(ch)
			}
		}
	}
}
