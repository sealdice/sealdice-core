package base

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/olahol/melody"

	"sealdice-core/api/v2/middleware"
	"sealdice-core/dice"
	"sealdice-core/logger"
)

const logStreamPath = "/sd-api/v2/base/logs/ws"

type logStreamMessage struct {
	Type  string            `json:"type"`
	Items []*logger.LogItem `json:"items,omitempty"`
	Item  *logger.LogItem   `json:"item,omitempty"`
}

func RegisterLogStreamRoute(e *echo.Echo, dm *dice.DiceManager) {
	if e == nil || dm == nil || len(dm.Dice) == 0 || dm.Dice[0] == nil || dm.Dice[0].LogWriter == nil {
		return
	}
	hub := newLogStreamHub(dm.Dice[0])
	e.GET(logStreamPath, hub.handle)
}

type logStreamHub struct {
	dice   *dice.Dice
	melody *melody.Melody
}

func newLogStreamHub(d *dice.Dice) *logStreamHub {
	m := melody.New()
	h := &logStreamHub{
		dice:   d,
		melody: m,
	}
	m.HandleConnect(func(s *melody.Session) {
		writeLogStreamMessage(s, logStreamMessage{
			Type:  "snapshot",
			Items: d.LogWriter.Snapshot(),
		})
	})
	logCh, _ := d.LogWriter.Subscribe()
	go func() {
		for item := range logCh {
			data, err := json.Marshal(logStreamMessage{Type: "append", Item: item})
			if err == nil {
				_ = m.Broadcast(data)
			}
		}
	}()
	return h
}

func (h *logStreamHub) handle(c echo.Context) error {
	token := middleware.TokenFromHTTPRequest(c.Request())
	if !middleware.IsAuthorized(h.dice, token) {
		return c.NoContent(http.StatusUnauthorized)
	}
	return h.melody.HandleRequest(c.Response(), c.Request())
}

func writeLogStreamMessage(s *melody.Session, msg logStreamMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	_ = s.Write(data)
}
