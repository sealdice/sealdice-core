package realtime

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/olahol/melody"

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

func RegisterRoutes(e *echo.Echo, dm *dice.DiceManager) *Server {
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

	e.GET(realtimeWSPath, srv.handleWS)
	e.GET(realtimeSSEPath, srv.handleSSE)

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

func (s *Server) handleWS(c echo.Context) error {
	if !isAuthorized(s.dm, apimiddleware.TokenFromHTTPRequest(c.Request())) {
		return c.NoContent(http.StatusUnauthorized)
	}
	return s.ws.HandleRequest(c.Response(), c.Request())
}

func (s *Server) handleSSE(c echo.Context) error {
	if !isAuthorized(s.dm, apimiddleware.TokenFromHTTPRequest(c.Request())) {
		return c.NoContent(http.StatusUnauthorized)
	}

	writer := c.Response().Writer
	flusher, ok := writer.(http.Flusher)
	if !ok {
		return c.NoContent(http.StatusInternalServerError)
	}

	header := c.Response().Header()
	header.Set(echo.HeaderContentType, "text/event-stream")
	header.Set(echo.HeaderCacheControl, "no-cache")
	header.Set(echo.HeaderConnection, "keep-alive")
	header.Set("X-Accel-Buffering", "no")
	c.Response().WriteHeader(http.StatusOK)

	for _, evt := range buildBootstrapEvents(s.dm) {
		if err := writeSSEEvent(writer, evt); err != nil {
			return nil
		}
		flusher.Flush()
	}

	ch, unsubscribe := s.bus.Subscribe(128)
	defer unsubscribe()

	heartbeat := time.NewTicker(sseHeartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case evt, ok := <-ch:
			if !ok {
				return nil
			}
			if err := writeSSEEvent(writer, evt); err != nil {
				return nil
			}
			flusher.Flush()
		case <-heartbeat.C:
			if _, err := io.WriteString(writer, ": ping\n\n"); err != nil {
				return nil
			}
			flusher.Flush()
		}
	}
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

func encodeEnvelope(evt Event) ([]byte, error) {
	return json.Marshal(envelope{
		Event:   evt.Name,
		Payload: evt.Payload,
	})
}

func writeSSEEvent(w io.Writer, evt Event) error {
	data, err := json.Marshal(evt.Payload)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evt.Name, data)
	return err
}

func isAuthorized(dm *dice.DiceManager, token string) bool {
	d := primaryDice(dm)
	if d == nil {
		return false
	}
	return apimiddleware.IsAuthorized(d, token)
}

func primaryDice(dm *dice.DiceManager) *dice.Dice {
	if dm == nil || len(dm.Dice) == 0 {
		return nil
	}
	return dm.Dice[0]
}
