package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestRuntimeMiddlewareOnlyBlocksAPIPaths(t *testing.T) {
	runtimeMaintenance.Store(true)
	t.Cleanup(func() { runtimeMaintenance.Store(false) })
	handler := runtimeMiddleware(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	e := echo.New()
	staticRecorder := httptest.NewRecorder()
	if err := handler(e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), staticRecorder)); err != nil {
		t.Fatal(err)
	}
	if staticRecorder.Code != http.StatusNoContent {
		t.Fatalf("static status = %d, want %d", staticRecorder.Code, http.StatusNoContent)
	}

	apiRecorder := httptest.NewRecorder()
	if err := handler(e.NewContext(httptest.NewRequest(http.MethodGet, "/sd-api/baseInfo", nil), apiRecorder)); err != nil {
		t.Fatal(err)
	}
	if apiRecorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("api status = %d, want %d", apiRecorder.Code, http.StatusServiceUnavailable)
	}
}
