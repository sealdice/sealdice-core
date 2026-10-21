package realtime

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/sse"

	"sealdice-core/api/v2/imconnection"
	"sealdice-core/dice"
)

const (
	realtimeSSEPath      = "/sd-api/v2/realtime/sse"
	realtimeSSEOpPath    = "/sse"
	sseHeartbeatInterval = 20 * time.Second
)

type Server struct {
	dm *dice.DiceManager

	bus          *Bus
	watcher      *StateWatcher
	toolMessages ToolMessageProvider

	mu      sync.Mutex
	cancel  context.CancelFunc
	done    chan struct{}
	stopped chan struct{}
	closed  bool
}

func RegisterRoutes(api huma.API, dm *dice.DiceManager) *Server {
	if api == nil {
		return nil
	}
	d := primaryDice(dm)
	if d == nil || d.ImSession == nil {
		return nil
	}

	srv := NewServer(dm)
	sse.Register(api, huma.Operation{
		OperationID: "realtime-sse",
		Method:      http.MethodGet,
		Path:        realtimeSSEOpPath,
		Middlewares: huma.Middlewares{
			func(ctx huma.Context, next func(huma.Context)) {
				ctx.SetHeader("Cache-Control", "no-cache")
				ctx.SetHeader("X-Accel-Buffering", "no")
				next(ctx)
			},
		},
	}, realtimeEventTypeMap(), srv.stream)

	huma.Get(api, "/poll", srv.Poll, func(o *huma.Operation) {
		o.Description = "SSE 不可用时的增量轮询接口"
		o.Summary = "轮询实时事件"
	})

	return srv
}

func NewServer(dm *dice.DiceManager) *Server {
	srv := &Server{dm: dm, bus: NewBus(), stopped: make(chan struct{})}
	srv.watcher = NewStateWatcher(dm, srv.bus)
	return srv
}

// Start runs once, under the application's lifetime rather than route registration.
func (s *Server) Start(ctx context.Context) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.cancel != nil {
		return
	}
	ctx, s.cancel = context.WithCancel(ctx)
	s.done = make(chan struct{})
	unsubscribe := s.watcher.BindLogs()
	go func() {
		defer close(s.done)
		defer unsubscribe()
		defer s.stop()
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.watcher.Scan()
			}
		}
	}()
}

func (s *Server) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		s.closed = true
		close(s.stopped)
	}
	if s.cancel != nil {
		s.cancel()
	}
}

// Close is idempotent and waits for the scanner and log forwarding to finish.
func (s *Server) Close() {
	if s == nil {
		return
	}
	s.stop()
	s.mu.Lock()
	done := s.done
	s.mu.Unlock()
	if done != nil {
		<-done
	}
}

func (s *Server) Publish(name string, payload any) {
	if s == nil || s.bus == nil {
		return
	}
	s.bus.Publish(Event{Name: name, Payload: payload})
}

func (s *Server) stream(ctx context.Context, _ *struct{}, send sse.Sender) {
	s.watcher.mu.Lock()
	ch, unsubscribe := s.bus.Subscribe(128)
	bootstrapEvents := buildBootstrapEvents(s.dm)
	s.watcher.mu.Unlock()
	defer unsubscribe()

	for _, evt := range bootstrapEvents {
		if err := send.Data(evt.Payload); err != nil {
			return
		}
	}

	heartbeat := time.NewTicker(sseHeartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case evt, ok := <-ch:
			if !ok {
				return
			}
			if err := send.Data(evt.Payload); err != nil {
				return
			}
		case <-heartbeat.C:
			if err := send.Data(SystemHeartbeatPayload{}); err != nil {
				return
			}
		case <-ctx.Done():
			return
		case <-s.stopped:
			return
		}
	}
}

func buildBootstrapEvents(dm *dice.DiceManager) []Event {
	events := []Event{
		{
			Name:    EventSystemReady,
			Payload: SystemReadyPayload{},
		},
	}

	d := primaryDice(dm)
	if d == nil {
		return events
	}

	if d.LogWriter != nil {
		events = append(events, Event{
			Name: EventLogsSnapshot,
			Payload: LogSnapshotPayload{
				Items: d.LogWriter.Snapshot(),
			},
		})
	}

	if d.ImSession == nil {
		return events
	}

	events = append(events, Event{
		Name: EventIMConnectionList,
		Payload: IMConnectionListPayload{
			Items: imconnection.NewEndPointInfoList(d.ImSession.EndPoints),
		},
	})

	return events
}

func realtimeEventTypeMap() map[string]any {
	return map[string]any{
		EventSystemReady:         SystemReadyPayload{},
		EventSystemHeartbeat:     SystemHeartbeatPayload{},
		EventLogsSnapshot:        LogSnapshotPayload{},
		EventLogsAppend:          LogAppendPayload{},
		EventIMConnectionList:    IMConnectionListPayload{},
		EventIMConnectionUpdated: IMConnectionUpdatedPayload{},
		EventToolTestMessage:     dice.HTTPTestMessage{},
	}
}

func primaryDice(dm *dice.DiceManager) *dice.Dice {
	if dm == nil {
		return nil
	}
	return dm.GetDice()
}
