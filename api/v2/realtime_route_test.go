package v2_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humaecho"
	"github.com/labstack/echo/v4"

	apiv2 "sealdice-core/api/v2"
	"sealdice-core/dice"
	"sealdice-core/logger"
)

func TestInitV2RouterDoesNotExposeLegacyRealtimeWebSocketRoute(t *testing.T) {
	app := echo.New()
	api := humaecho.New(app, huma.DefaultConfig("Sealdice API", "2.0.0"))
	dm := newRealtimeRouteDiceManager()

	apiv2.InitV2Router(api, dm)

	req := httptest.NewRequest(http.MethodGet, "/sd-api/v2/realtime/ws?token=token-1", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestInitV2RouterRequiresAuthForGroupList(t *testing.T) {
	app := echo.New()
	api := humaecho.New(app, huma.DefaultConfig("Sealdice API", "2.0.0"))
	dm := newRealtimeRouteDiceManager()

	apiv2.InitV2Router(api, dm)

	req := httptest.NewRequest(http.MethodPost, "/sd-api/v2/group/list", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestInitV2RouterRequiresAuthForNetworkHealth(t *testing.T) {
	app := echo.New()
	api := humaecho.New(app, huma.DefaultConfig("Sealdice API", "2.0.0"))
	dm := newRealtimeRouteDiceManager()

	apiv2.InitV2Router(api, dm)

	req := httptest.NewRequest(http.MethodGet, "/sd-api/v2/base/network-health", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func newRealtimeRouteDiceManager() *dice.DiceManager {
	d := &dice.Dice{
		Logger:    logger.M(),
		LogWriter: logger.NewUIWriter(),
	}
	d.ImSession = &dice.IMSession{
		Parent:       d,
		EndPoints:    []*dice.EndPointInfo{},
		ServiceAtNew: new(dice.SyncMap[string, *dice.GroupInfo]),
		PendingQuits: new(dice.SyncMap[string, *dice.PendingQuitInfo]),
	}
	dm := &dice.DiceManager{
		Dice: []*dice.Dice{d},
	}
	d.Parent = dm
	dm.AccessTokens.Store("token-1", true)
	return dm
}
