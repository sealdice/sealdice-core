package realtime

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/codecat/melody"
	"github.com/gofiber/fiber/v2"

	apimiddleware "sealdice-core/api/v2/middleware"
	"sealdice-core/dice"
)

const (
	realtimeWSPath          = "/sd-api/v2/realtime/ws"
	realtimeSSEPath         = "/sd-api/v2/realtime/sse"
	sseHeartbeatInterval    = 20 * time.Second
	wsUnsubscribeSessionKey = "realtime-unsubscribe"
)

type envelope struct {
	Event   string `json:"event"`
	Payload any    `json:"payload"`
}

type Server struct {
	dm *dice.DiceManager

	bus     *Bus
	ws      *melody.Melody
	watcher *StateWatcher

	unsubscribeLogs func()
}

func RegisterRoutes(e fiber.Router, dm *dice.DiceManager) *Server {
	if e == nil {
		return nil
	}
	d := primaryDice(dm)
	if d == nil || d.ImSession == nil || d.LogWriter == nil {
		return nil
	}

	srv := NewServer(dm)
	if srv == nil {
		return nil
	}

	srv.Start()

	e.Get(realtimeWSPath, srv.handleWS)
	e.Get(realtimeSSEPath, srv.handleSSE)

	return srv
}

func NewServer(dm *dice.DiceManager) *Server {
	srv := &Server{
		dm:      dm,
		bus:     NewBus(),
		ws:      melody.New(),
		watcher: NewStateWatcher(dm, NewBus()),
	}
	srv.watcher = NewStateWatcher(dm, srv.bus)

	srv.ws.HandleConnect(func(session *melody.Session) {
		unsubscribe := srv.attachWSSession(session)
		session.Set(wsUnsubscribeSessionKey, unsubscribe)
	})
	srv.ws.HandleDisconnect(func(session *melody.Session) {
		srv.detachWSSession(session)
	})
	srv.ws.HandleError(func(session *melody.Session, _ error) {
		srv.detachWSSession(session)
	})

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

func (s *Server) handleWS(c *fiber.Ctx) error {
	if !isAuthorized(s.dm, apimiddleware.TokenFromFiberCtx(c)) {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	return s.ws.HandleRequest(c.Context())
}

func (s *Server) handleSSE(c *fiber.Ctx) error {
	if !isAuthorized(s.dm, apimiddleware.TokenFromFiberCtx(c)) {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	c.Set(fiber.HeaderContentType, "text/event-stream")
	c.Set(fiber.HeaderCacheControl, "no-cache")
	c.Set(fiber.HeaderConnection, "keep-alive")
	c.Set("X-Accel-Buffering", "no")
	c.Status(fiber.StatusOK)

	c.Context().SetBodyStreamWriter(func(writer *bufio.Writer) {
		writeAndFlush := func(fn func() error) bool {
			if err := fn(); err != nil {
				return false
			}
			return writer.Flush() == nil
		}

		for _, evt := range buildBootstrapEvents(s.dm) {
			if !writeAndFlush(func() error { return writeSSEEvent(writer, evt) }) {
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
				if !writeAndFlush(func() error { return writeSSEEvent(writer, evt) }) {
					return
				}
			case <-heartbeat.C:
				if !writeAndFlush(func() error {
					_, err := io.WriteString(writer, ": ping\n\n")
					return err
				}) {
					return
				}
			}
		}
	})
	return nil
}

func (s *Server) attachWSSession(session *melody.Session) func() {
	ch, unsubscribe := s.bus.Subscribe(128)

	for _, evt := range buildBootstrapEvents(s.dm) {
		if data, err := encodeEnvelope(evt); err == nil {
			_ = session.Write(data)
		}
	}

	var once sync.Once
	go func() {
		for evt := range ch {
			data, err := encodeEnvelope(evt)
			if err != nil {
				continue
			}
			if err := session.Write(data); err != nil {
				once.Do(unsubscribe)
				return
			}
		}
	}()

	return func() {
		once.Do(unsubscribe)
	}
}

func (s *Server) detachWSSession(session *melody.Session) {
	value, exists := session.Get(wsUnsubscribeSessionKey)
	if !exists {
		return
	}
	if unsubscribe, ok := value.(func()); ok {
		unsubscribe()
	}
	session.UnSet(wsUnsubscribeSessionKey)
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

func encodeEnvelope(evt Event) ([]byte, error) {
	return json.Marshal(envelope{
		Event:   evt.Name,
		Payload: evt.Payload,
	})
}

func EncodeEnvelope(evt Event) ([]byte, error) {
	return encodeEnvelope(evt)
}

func writeSSEEvent(w io.Writer, evt Event) error {
	data, err := json.Marshal(evt.Payload)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evt.Name, data)
	return err
}

func WriteSSEEvent(w io.Writer, evt Event) error {
	return writeSSEEvent(w, evt)
}

func isAuthorized(dm *dice.DiceManager, token string) bool {
	d := primaryDice(dm)
	if d == nil {
		return false
	}
	return apimiddleware.IsAuthorized(d, token)
}

func IsAuthorized(dm *dice.DiceManager, token string) bool {
	return isAuthorized(dm, token)
}

func primaryDice(dm *dice.DiceManager) *dice.Dice {
	if dm == nil || len(dm.Dice) == 0 {
		return nil
	}
	return dm.Dice[0]
}
