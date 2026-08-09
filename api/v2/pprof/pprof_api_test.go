package pprof

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
)

func TestGetCmdlineStreamsText(t *testing.T) {
	svc := NewService()

	stream, err := svc.GetCmdline(t.Context(), &ProfileQuery{})
	if err != nil {
		t.Fatalf("GetCmdline returned error: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx := humatest.NewContext(nil, httptest.NewRequest(http.MethodGet, "/sd-api/v2/pprof/cmdline", nil), recorder)
	stream.Body(ctx)

	if got := recorder.Header().Get("Content-Type"); got == "" {
		t.Fatal("Content-Type is empty")
	}
	if body := strings.TrimSpace(recorder.Body.String()); body == "" {
		t.Fatal("cmdline body is empty")
	}
}

func TestGetHeapSupportsDebugView(t *testing.T) {
	svc := NewService()

	stream, err := svc.GetHeap(t.Context(), &ProfileQuery{Debug: 1})
	if err != nil {
		t.Fatalf("GetHeap returned error: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx := humatest.NewContext(nil, httptest.NewRequest(http.MethodGet, "/sd-api/v2/pprof/heap?debug=1", nil), recorder)
	stream.Body(ctx)

	if got := recorder.Header().Get("Content-Type"); got == "" {
		t.Fatal("Content-Type is empty")
	}
	if body := strings.TrimSpace(recorder.Body.String()); body == "" {
		t.Fatal("heap body is empty")
	}
}

func TestGetProfileSupportsSecondsQuery(t *testing.T) {
	svc := NewService()

	stream, err := svc.GetProfile(t.Context(), &ProfileQuery{Seconds: 1})
	if err != nil {
		t.Fatalf("GetProfile returned error: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx := humatest.NewContext(nil, httptest.NewRequest(http.MethodGet, "/sd-api/v2/pprof/profile?seconds=1", nil), recorder)
	stream.Body(ctx)

	if got := recorder.Header().Get("Content-Type"); got == "" {
		t.Fatal("Content-Type is empty")
	}
	if recorder.Body.Len() == 0 {
		t.Fatal("profile body is empty")
	}
}
