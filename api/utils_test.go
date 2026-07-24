package api_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"sealdice-core/api"
	"sealdice-core/dice"
)

type blockingRoundTripper struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
	calls   atomic.Int32
}

func newBlockingRoundTripper() *blockingRoundTripper {
	return &blockingRoundTripper{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
}

func (transport *blockingRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	transport.calls.Add(1)
	transport.once.Do(func() { close(transport.started) })

	select {
	case <-transport.release:
	case <-request.Context().Done():
		return nil, request.Context().Err()
	}

	body := "{}"
	if strings.HasSuffix(request.URL.Path, "/signinfo.json") {
		body = `[{"version":"test","servers":[{"name":"test","url":"https://sign.test"}]}]`
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    request,
	}, nil
}

func setupHealthAPI(t *testing.T, transport http.RoundTripper) *echo.Echo {
	t.Helper()

	dataDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dataDir, "extra"), 0o755); err != nil {
		t.Fatalf("create test data directory: %v", err)
	}

	manager := &dice.DiceManager{}
	testDice := &dice.Dice{
		BaseConfig: dice.BaseConfig{DataDir: dataDir},
		Logger:     zap.NewNop().Sugar(),
		Parent:     manager,
	}
	manager.Dice = []*dice.Dice{testDice}

	originalTransport := http.DefaultTransport
	originalClientTransport := http.DefaultClient.Transport
	http.DefaultTransport = transport
	http.DefaultClient.Transport = transport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
		http.DefaultClient.Transport = originalClientTransport
	})

	e := echo.New()
	api.Bind(e, manager)
	return e
}

func performHealthRequest(e *echo.Echo, ctx context.Context) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/sd-api/utils/check_network_health", nil)
	request = request.WithContext(ctx)
	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, request)
	return recorder
}

func TestCheckNetworkHealthSharesInFlightResult(t *testing.T) {
	transport := newBlockingRoundTripper()
	e := setupHealthAPI(t, transport)

	responses := make(chan *httptest.ResponseRecorder, 2)
	go func() { responses <- performHealthRequest(e, context.Background()) }()
	<-transport.started

	secondStarted := make(chan struct{})
	go func() {
		close(secondStarted)
		responses <- performHealthRequest(e, context.Background())
	}()
	<-secondStarted
	time.Sleep(20 * time.Millisecond)
	close(transport.release)

	first := <-responses
	second := <-responses
	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("response codes = %d and %d, want 200", first.Code, second.Code)
	}
	if !bytes.Equal(first.Body.Bytes(), second.Body.Bytes()) {
		t.Fatalf("shared responses differ:\nfirst: %s\nsecond: %s", first.Body, second.Body)
	}

	concurrentCalls := transport.calls.Load()
	beforeSingle := transport.calls.Load()
	single := performHealthRequest(e, context.Background())
	if single.Code != http.StatusOK {
		t.Fatalf("single response code = %d, want 200", single.Code)
	}
	singleCalls := transport.calls.Load() - beforeSingle
	if concurrentCalls != singleCalls {
		t.Fatalf("concurrent requests made %d outbound calls, one request made %d", concurrentCalls, singleCalls)
	}
}

func TestCheckNetworkHealthCanceledWaiterDoesNotStartAnotherCheck(t *testing.T) {
	transport := newBlockingRoundTripper()
	e := setupHealthAPI(t, transport)

	firstResponse := make(chan *httptest.ResponseRecorder, 1)
	go func() { firstResponse <- performHealthRequest(e, context.Background()) }()
	<-transport.started

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	_ = performHealthRequest(e, canceledCtx)
	close(transport.release)

	if response := <-firstResponse; response.Code != http.StatusOK {
		t.Fatalf("shared response code = %d, want 200", response.Code)
	}
	sharedCalls := transport.calls.Load()

	beforeSingle := transport.calls.Load()
	if response := performHealthRequest(e, context.Background()); response.Code != http.StatusOK {
		t.Fatalf("single response code = %d, want 200", response.Code)
	}
	singleCalls := transport.calls.Load() - beforeSingle
	if sharedCalls != singleCalls {
		t.Fatalf("check with canceled waiter made %d outbound calls, one request made %d", sharedCalls, singleCalls)
	}
}
