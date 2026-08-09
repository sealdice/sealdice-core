package api //nolint:testpackage // Tests exercise the unexported Runtime maintenance gate.

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"sealdice-core/dice"
)

func TestBeginRuntimeMaintenanceContextRestoresPublishedStateAfterTimeout(t *testing.T) {
	manager := &dice.DiceManager{}
	manager.Dice = []*dice.Dice{{Parent: manager}}
	ReplaceRuntime(manager)
	runtimeState.Store(runtimeStateRunning)
	t.Cleanup(func() {
		ReplaceRuntime(nil)
		runtimeState.Store(runtimeStateRunning)
	})

	runtimeGate.RLock()
	readerHeld := true
	defer func() {
		if readerHeld {
			runtimeGate.RUnlock()
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if err := BeginRuntimeMaintenanceContext(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("maintenance error = %v, want context deadline exceeded", err)
	}
	if runtimeState.Load() != runtimeStateRunning || dm != manager || myDice != manager.Dice[0] {
		t.Fatal("timed-out maintenance did not restore the published Runtime state")
	}

	runtimeGate.RUnlock()
	readerHeld = false
	if !runtimeMaintenanceMu.TryLock() {
		t.Fatal("timed-out maintenance retained its acquisition lock")
	}
	runtimeMaintenanceMu.Unlock()
	if !runtimeGate.TryLock() {
		t.Fatal("timed-out maintenance retained the Runtime gate")
	}
	runtimeGate.Unlock()
}

func TestRuntimeMiddlewareOnlyBlocksAPIPaths(t *testing.T) {
	runtimeState.Store(runtimeStateReloading)
	t.Cleanup(func() { runtimeState.Store(runtimeStateRunning) })
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
	if !strings.Contains(apiRecorder.Body.String(), "RUNTIME_RELOADING") {
		t.Fatalf("api response = %s, want maintenance code", apiRecorder.Body.String())
	}

	for _, controlPath := range []string{"/sd-api/backup/restore/status", "/sd-api/signin/salt"} {
		recorder := httptest.NewRecorder()
		if err := handler(e.NewContext(httptest.NewRequest(http.MethodGet, controlPath, nil), recorder)); err != nil {
			t.Fatal(err)
		}
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("control path %s status = %d, want %d", controlPath, recorder.Code, http.StatusNoContent)
		}
	}
}

func TestRuntimeMiddlewareReportsUnavailable(t *testing.T) {
	runtimeState.Store(runtimeStateUnavailable)
	t.Cleanup(func() { runtimeState.Store(runtimeStateRunning) })
	handler := runtimeMiddleware(func(c echo.Context) error { return c.NoContent(http.StatusNoContent) })
	e := echo.New()
	recorder := httptest.NewRecorder()
	if err := handler(e.NewContext(httptest.NewRequest(http.MethodGet, "/sd-api/baseInfo", nil), recorder)); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusServiceUnavailable || !strings.Contains(recorder.Body.String(), "RUNTIME_UNAVAILABLE") {
		t.Fatalf("response = %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestRuntimeMiddlewareRechecksStateAfterWaitingForGate(t *testing.T) {
	manager := &dice.DiceManager{}
	manager.Dice = []*dice.Dice{{Parent: manager}}
	ReplaceRuntime(manager)
	runtimeState.Store(runtimeStateRunning)
	t.Cleanup(func() {
		ReplaceRuntime(nil)
		runtimeState.Store(runtimeStateRunning)
	})

	handlerCalled := make(chan struct{}, 1)
	handler := runtimeMiddleware(func(c echo.Context) error {
		handlerCalled <- struct{}{}
		return c.NoContent(http.StatusNoContent)
	})
	e := echo.New()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/sd-api/config", nil)
	requestDone := make(chan error, 1)

	runtimeGate.Lock()
	go func() {
		requestDone <- handler(e.NewContext(request, recorder))
	}()
	// Give the request time to pass the optimistic state check and wait for
	// the gate. The post-lock check remains correct if it has not run yet.
	time.Sleep(20 * time.Millisecond)
	runtimeState.Store(runtimeStateReloading)
	runtimeGate.Unlock()

	if err := <-requestDone; err != nil {
		t.Fatal(err)
	}
	select {
	case <-handlerCalled:
		t.Fatal("request entered handler after Runtime maintenance started")
	default:
	}
	if recorder.Code != http.StatusServiceUnavailable || !strings.Contains(recorder.Body.String(), "RUNTIME_RELOADING") {
		t.Fatalf("response = %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestReplaceRuntimeRejectsManagerWithoutDice(t *testing.T) {
	runtimeState.Store(runtimeStateRunning)
	ReplaceRuntime(&dice.DiceManager{})
	t.Cleanup(func() {
		ReplaceRuntime(nil)
		runtimeState.Store(runtimeStateRunning)
	})

	if dm != nil || myDice != nil || runtimeState.Load() != runtimeStateUnavailable {
		t.Fatalf("empty manager was published: dm=%p dice=%p state=%d", dm, myDice, runtimeState.Load())
	}
}

func TestSignInSaltDeclaresWhetherPasswordIsRequired(t *testing.T) {
	t.Cleanup(func() {
		ReplaceRuntime(nil)
		runtimeAuth.Store(nil)
		runtimeState.Store(runtimeStateRunning)
	})
	for _, test := range []struct {
		name     string
		password string
		want     string
	}{
		{name: "empty password", want: `"passwordRequired":false`},
		{name: "configured password", password: "hash", want: `"passwordRequired":true`},
	} {
		t.Run(test.name, func(t *testing.T) {
			manager := &dice.DiceManager{UIPasswordSalt: "salt", UIPasswordHash: test.password}
			manager.Dice = []*dice.Dice{{Parent: manager}}
			ReplaceRuntime(manager)
			e := echo.New()
			recorder := httptest.NewRecorder()
			ctx := e.NewContext(httptest.NewRequest(http.MethodGet, "/sd-api/signin/salt", nil), recorder)
			if err := doSignInGetSalt(ctx); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(recorder.Body.String(), test.want) {
				t.Fatalf("response = %s, want %s", recorder.Body.String(), test.want)
			}
			if recorder.Header().Get(echo.HeaderCacheControl) != "no-store" {
				t.Fatalf("salt cache-control = %q", recorder.Header().Get(echo.HeaderCacheControl))
			}
		})
	}
}

func installAPITestRuntime(t *testing.T) string {
	t.Helper()
	manager := &dice.DiceManager{}
	manager.Dice = []*dice.Dice{{Parent: manager}}
	token := "test-token"
	manager.AccessTokens.Store(token, true)
	ReplaceRuntime(manager)
	runtimeState.Store(runtimeStateRunning)
	t.Cleanup(func() {
		ReplaceRuntime(nil)
		runtimeAuth.Store(nil)
		runtimeRestoreFn = nil
		runtimeState.Store(runtimeStateRunning)
	})
	return token
}

func TestBackupListOnlyReadsRootArchives(t *testing.T) {
	t.Chdir(t.TempDir())
	token := installAPITestRuntime(t)
	if err := os.MkdirAll(filepath.Join(dice.BackupDir, ".restore"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dice.BackupDir, "root.zip"), []byte("invalid"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dice.BackupDir, ".restore", "source.zip"), []byte("invalid"), 0o600); err != nil {
		t.Fatal(err)
	}

	e := echo.New()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/sd-api/backup/list", nil)
	request.Header.Set("token", token) //nolint:canonicalheader // Private API compatibility header.
	if err := backupGetList(e.NewContext(request, recorder)); err != nil {
		t.Fatal(err)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "root.zip") || strings.Contains(body, "source.zip") {
		t.Fatalf("unexpected backup list: %s", body)
	}
}

func TestBackupRestoreRejectsInvalidRequestIDBeforeScheduling(t *testing.T) {
	token := installAPITestRuntime(t)
	runtimeRestoreFn = func(_ string) bool {
		t.Fatal("restore callback must not run")
		return false
	}
	e := echo.New()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/sd-api/backup/restore", bytes.NewBufferString(`{"name":"backup.zip","requestId":"short"}`))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	request.Header.Set("token", token) //nolint:canonicalheader // Private API compatibility header.
	if err := backupRestore(e.NewContext(request, recorder)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(recorder.Body.String(), "requestId") || !strings.Contains(recorder.Body.String(), `"result":false`) {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
	if recorder.Header().Get(echo.HeaderCacheControl) != "no-store" {
		t.Fatalf("restore cache-control = %q", recorder.Header().Get(echo.HeaderCacheControl))
	}
}
