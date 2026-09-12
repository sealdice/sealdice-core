package storylog

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestProbeCachedLogURL(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       cachedLogProbeResult
	}{
		{name: "alive", statusCode: http.StatusOK, want: cachedLogProbeAlive},
		{name: "missing", statusCode: http.StatusNotFound, want: cachedLogProbeMissing},
		{name: "gone", statusCode: http.StatusGone, want: cachedLogProbeMissing},
		{name: "server error", statusCode: http.StatusInternalServerError, want: cachedLogProbeUnknown},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/custom/backend/load_data" {
					t.Fatalf("path = %q, want /custom/backend/load_data", r.URL.Path)
				}
				if got := r.URL.Query().Get("key"); got != "AbCd" {
					t.Fatalf("key = %q, want AbCd", got)
				}
				if got := r.URL.Query().Get("password"); got != "123456" {
					t.Fatalf("password = %q, want 123456", got)
				}
				if got := r.Header.Get("Authorization"); got != "Bearer token" {
					t.Fatalf("Authorization = %q, want Bearer token", got)
				}
				w.WriteHeader(test.statusCode)
			}))
			defer server.Close()

			env := UploadEnv{
				Backends: []string{server.URL + "/custom/backend/log"},
				Token:    "token",
				Log:      zap.NewNop().Sugar(),
			}
			if got := probeCachedLogURL(env, "https://logs.example/?key=AbCd#123456"); got != test.want {
				t.Fatalf("probeCachedLogURL() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestProbeCachedLogURLNetworkFailureIsUnknown(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	backend := server.URL + "/dice/api/log"
	server.Close()

	env := UploadEnv{
		Backends: []string{backend},
		Log:      zap.NewNop().Sugar(),
	}
	if got := probeCachedLogURL(env, "https://logs.example/?key=AbCd#123456"); got != cachedLogProbeUnknown {
		t.Fatalf("probeCachedLogURL() = %v, want %v", got, cachedLogProbeUnknown)
	}
}

func TestProbeCachedLogURLRejectsUnusableCachedURL(t *testing.T) {
	env := UploadEnv{Log: zap.NewNop().Sugar()}
	if got := probeCachedLogURL(env, "https://logs.example/"); got != cachedLogProbeMissing {
		t.Fatalf("probeCachedLogURL() = %v, want %v", got, cachedLogProbeMissing)
	}
}
