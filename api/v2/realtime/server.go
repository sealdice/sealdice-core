package realtime

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/sse"

	"sealdice-core/dice"
)

const (
	realtimeSSEPath      = "/sd-api/v2/realtime/sse"
	realtimeSSEOpPath    = "/sse"
	sseHeartbeatInterval = 20 * time.Second
)

type Server struct {
	dm *dice.DiceManager

	bus     *Bus
	watcher *StateWatcher

	unsubscribeLogs func()
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
	if srv == nil {
		return nil
	}

	srv.Start()
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

	return srv
}

func NewServer(dm *dice.DiceManager) *Server {
	srv := &Server{
		dm:      dm,
		bus:     NewBus(),
		watcher: NewStateWatcher(dm, NewBus()),
	}
	srv.watcher = NewStateWatcher(dm, srv.bus)
	return srv
}

func (s *Server) Start() {
	if s == nil || s.watcher == nil {
		return
	}

	if s.unsubscribeLogs == nil {
		s.unsubscribeLogs = s.watcher.BindLogs()
	}

	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for range ticker.C {
			s.watcher.Scan()
		}
	}()
}

func (s *Server) Publish(name string, payload any) {
	if s == nil || s.bus == nil {
		return
	}
	s.bus.Publish(Event{Name: name, Payload: payload})
}

func (s *Server) stream(ctx context.Context, _ *struct{}, send sse.Sender) {
	for _, evt := range buildBootstrapEvents(s.dm) {
		if err := sendRealtimeEvent(send, evt); err != nil {
			return
		}
	}

	ch, unsubscribe := s.bus.Subscribe(128)
	defer unsubscribe()

	heartbeat := time.NewTicker(sseHeartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case evt, ok := <-ch:
			if !ok {
				return
			}
			if err := sendRealtimeEvent(send, evt); err != nil {
				return
			}
		case <-heartbeat.C:
			if err := send.Data(SystemHeartbeatPayload{}); err != nil {
				return
			}
		case <-ctx.Done():
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
			Items: d.ImSession.EndPoints,
		},
	})

	for _, ep := range d.ImSession.EndPoints {
		if ep == nil {
			continue
		}

		events = append(events, Event{
			Name: EventIMConnectionWorkflow,
			Payload: IMConnectionWorkflowPayload{
				EndpointID: ep.ID,
				Workflow:   workflowOfEndpoint(ep),
			},
		})

		if qr := qrCodeOfEndpoint(ep); qr != "" {
			events = append(events, Event{
				Name: EventIMConnectionQRCode,
				Payload: IMConnectionQRCodePayload{
					EndpointID: ep.ID,
					Img:        qr,
				},
			})
		}
	}

	return events
}

func BuildBootstrapEvents(dm *dice.DiceManager) []Event {
	return buildBootstrapEvents(dm)
}

func realtimeEventTypeMap() map[string]any {
	return map[string]any{
		EventSystemReady:          SystemReadyPayload{},
		EventSystemHeartbeat:      SystemHeartbeatPayload{},
		EventLogsSnapshot:         LogSnapshotPayload{},
		EventLogsAppend:           LogAppendPayload{},
		EventIMConnectionList:     IMConnectionListPayload{},
		EventIMConnectionUpdated:  IMConnectionUpdatedPayload{},
		EventIMConnectionWorkflow: IMConnectionWorkflowPayload{},
		EventIMConnectionQRCode:   IMConnectionQRCodePayload{},
		EventToolTestMessage:      dice.HTTPTestMessage{},
	}
}

func sendRealtimeEvent(send sse.Sender, evt Event) error {
	switch payload := evt.Payload.(type) {
	case SystemReadyPayload:
		return send.Data(payload)
	case SystemHeartbeatPayload:
		return send.Data(payload)
	case LogSnapshotPayload:
		return send.Data(payload)
	case LogAppendPayload:
		return send.Data(payload)
	case IMConnectionListPayload:
		return send.Data(payload)
	case IMConnectionUpdatedPayload:
		return send.Data(payload)
	case IMConnectionWorkflowPayload:
		return send.Data(payload)
	case IMConnectionQRCodePayload:
		return send.Data(payload)
	case dice.HTTPTestMessage:
		return send.Data(payload)
	default:
		return nil
	}
}

func primaryDice(dm *dice.DiceManager) *dice.Dice {
	if dm == nil || len(dm.Dice) == 0 {
		return nil
	}
	return dm.Dice[0]
}
