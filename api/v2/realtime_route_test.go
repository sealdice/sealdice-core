package v2_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humaecho"
	"github.com/labstack/echo/v4"

	apiv2 "sealdice-core/api/v2"
	"sealdice-core/dice"
	"sealdice-core/logger"
	"sealdice-core/utils/epoch"
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

func TestRealtimePollRouteReturnsLogDelta(t *testing.T) {
	app := echo.New()
	api := humaecho.New(app, huma.DefaultConfig("Sealdice API", "2.0.0"))
	dm := newRealtimeRouteDiceManager()
	apiv2.InitV2Router(api, dm)

	_, _ = dm.GetDice().LogWriter.Write([]byte(`{"level":"info","module":"core","time":"2026-05-17T12:34:56.000Z+0800","msg":"first"}`))
	_, _ = dm.GetDice().LogWriter.Write([]byte(`{"level":"info","module":"core","time":"2026-05-17T12:34:57.000Z+0800","msg":"second"}`))
	if got := dm.GetDice().LogWriter.Snapshot(); len(got) != 2 {
		t.Fatalf("writer snapshot = %d, want 2", len(got))
	}

	req := httptest.NewRequest(http.MethodGet, "/sd-api/v2/realtime/poll?afterLogSeq=1&afterEpoch="+epoch.Current()+"&token=token-1", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Item struct {
			Logs struct {
				Items []struct {
					Seq uint64 `json:"seq"`
					Msg string `json:"msg"`
				}
				LatestSeq uint64 `json:"latestSeq"`
				Reset     bool   `json:"reset"`
			} `json:"logs"`
		} `json:"item"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	logs := body.Item.Logs
	if logs.Reset || logs.LatestSeq != 2 || len(logs.Items) != 1 || logs.Items[0].Seq != 2 || logs.Items[0].Msg != "second" {
		t.Fatalf("unexpected poll logs: %+v", logs)
	}
}

func TestInitV2RouterRequiresAuthOutsideBootstrap(t *testing.T) {
	app := echo.New()
	api := humaecho.New(app, huma.DefaultConfig("Sealdice API", "2.0.0"))
	apiv2.InitV2Router(api, newRealtimeRouteDiceManager())
	public := map[string]bool{
		"/sd-api/v2/base/health":         true,
		"/sd-api/v2/base/overview":       true,
		"/sd-api/v2/base/login":          true,
		"/sd-api/v2/base/login/salt":     true,
		"/sd-api/v2/base/security-check": true,
	}
	for path, item := range api.OpenAPI().Paths {
		if public[path] {
			continue
		}
		for method, operation := range map[string]*huma.Operation{
			http.MethodGet: item.Get, http.MethodPost: item.Post,
			http.MethodPut: item.Put, http.MethodDelete: item.Delete, http.MethodPatch: item.Patch,
		} {
			if operation == nil {
				continue
			}
			t.Run(method+" "+path, func(t *testing.T) {
				rec := httptest.NewRecorder()
				app.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
				if rec.Code != http.StatusUnauthorized {
					t.Fatalf("status = %d, want 401: %s", rec.Code, rec.Body.String())
				}
			})
		}
	}
}

func TestInitV2RouterProtectsModuleWritesInDemoMode(t *testing.T) {
	app := echo.New()
	api := humaecho.New(app, huma.DefaultConfig("Sealdice API", "2.0.0"))
	dm := newRealtimeRouteDiceManager()
	dm.JustForTest = true
	apiv2.InitV2Router(api, dm)
	for _, route := range []struct{ method, path string }{
		{http.MethodPut, "base-setting/value"},
		{http.MethodPost, "group/modify"},
		{http.MethodPost, "backup/exec"},
		{http.MethodPost, "ban/add"},
		{http.MethodPut, "custom-text/example"},
		{http.MethodPut, "config/advanced"},
		{http.MethodDelete, "custom-reply/files/example"},
		{http.MethodPost, "deck/reload"},
		{http.MethodDelete, "story/log"},
		{http.MethodPost, "js/execute"},
		{http.MethodPost, "helpdoc/reload"},
		{http.MethodPost, "censor/restart"},
		{http.MethodPost, "imconnection/"},
		{http.MethodPost, "tool-test/messages"},
		{http.MethodPost, "resource/delete"},
		{http.MethodPost, "extension/preview-upload"},
	} {
		t.Run(route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, "/sd-api/v2/"+route.path, nil)
			req.Header.Set("Authorization", "Bearer token-1")
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)
			if rec.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403: %s", rec.Code, rec.Body.String())
			}
		})
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
		Dice: d,
	}
	d.Parent = dm
	dm.AccessTokens.Store("token-1", true)
	return dm
}
