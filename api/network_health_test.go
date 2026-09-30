package api //nolint:testpackage

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type networkHealthRoundTripper func(*http.Request) (*http.Response, error)

func (f networkHealthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestCheckHTTPConnectivityAcceptsHTTPResponses(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusTooManyRequests, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
			}))
			defer server.Close()
			ok, duration := checkHTTPConnectivity(server.Client(), server.URL)
			if !ok || duration < 0 {
				t.Fatalf("status %d: ok=%v, duration=%v; an HTTP response proves reachability", status, ok, duration)
			}
		})
	}
}

func TestCheckHTTPConnectivityRejectsTransportFailures(t *testing.T) {
	client := &http.Client{Transport: networkHealthRoundTripper(func(_ *http.Request) (*http.Response, error) {
		return nil, errors.New("network unavailable")
	})}
	for _, targetURL := range []string{"https://unreachable.invalid", ":invalid-url"} {
		ok, duration := checkHTTPConnectivity(client, targetURL)
		if ok || duration != 0 {
			t.Fatalf("%q: ok=%v, duration=%v; want false, 0", targetURL, ok, duration)
		}
	}
}

func TestCheckHTTPConnectivityTimesOut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
		<-req.Context().Done()
	}))
	defer server.Close()
	client := server.Client()
	client.Timeout = 30 * time.Millisecond
	start := time.Now()
	ok, _ := checkHTTPConnectivity(client, server.URL)
	if ok || time.Since(start) > time.Second {
		t.Fatal("unresponsive API must fail within the client timeout")
	}
}

func TestRunNetworkHealthCheckReportsBotAPIs(t *testing.T) {
	previousTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = previousTransport })
	requests := make(chan string, 12)
	http.DefaultTransport = networkHealthRoundTripper(func(req *http.Request) (*http.Response, error) {
		requests <- req.URL.String()
		if req.URL.Host == "slack.com" {
			return nil, errors.New("blocked API")
		}
		status := http.StatusUnauthorized
		headers := make(http.Header)
		if req.URL.Host == "api.telegram.org" {
			status = http.StatusFound
			headers.Set("Location", "https://core.telegram.org/bots")
		}
		return &http.Response{
			StatusCode: status,
			Header:     headers,
			Body:       io.NopCloser(strings.NewReader("{}")),
			Request:    req,
		}, nil
	})

	result := runNetworkHealthCheck()
	if result.Total != 6 || len(result.Targets) != 6 || len(result.Ok) != 5 || result.Timestamp == 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	expected := []string{"qq", "kook", "discord", "telegram", "dingtalk", "slack"}
	for i, target := range result.Targets {
		if target.Target != expected[i] || target.Ok != (target.Target != "slack") {
			t.Fatalf("unexpected target at %d: %+v", i, target)
		}
		if i < len(result.Ok) && result.Ok[i] != expected[i] {
			t.Fatalf("compatibility ok list: %v", result.Ok)
		}
	}
	close(requests)
	count := 0
	for targetURL := range requests {
		count++
		if strings.Contains(targetURL, "core.telegram.org") || strings.Contains(targetURL, "google.com") || strings.Contains(targetURL, "github.com") {
			t.Errorf("must probe bot APIs, requested %s", targetURL)
		}
	}
	if count != result.Total {
		t.Fatalf("sent %d requests, want %d API probes without following redirects", count, result.Total)
	}
}

func TestCheckHTTPConnectivityAcceptsRedirectsWithoutFollowingThem(t *testing.T) {
	for _, location := range []string{"/redirected", "https://redirect.invalid", "unsupported://redirect", "http://[invalid"} {
		t.Run(location, func(t *testing.T) {
			requests := make(chan string, 12)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				requests <- req.URL.Path
				w.Header().Set("Location", location)
				w.WriteHeader(http.StatusFound)
			}))
			defer server.Close()

			client := server.Client()
			client.Timeout = time.Second
			client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			}
			ok, duration := checkHTTPConnectivity(client, server.URL)
			if !ok || duration < 0 {
				t.Fatalf("redirect to %q: ok=%v, duration=%v; the initial HTTP response proves reachability", location, ok, duration)
			}
			if len(requests) != 1 {
				t.Fatalf("sent %d requests; a probe must not follow redirects", len(requests))
			}
			if requestedPath := <-requests; requestedPath != "/" {
				t.Fatalf("requested %q; want the API endpoint", requestedPath)
			}
		})
	}
}

type networkHealthResponseBody struct {
	io.Reader
	closed bool
}

func (body *networkHealthResponseBody) Close() error {
	body.closed = true
	return nil
}

func TestCheckHTTPConnectivityClosesResponseWithTransportError(t *testing.T) {
	body := &networkHealthResponseBody{Reader: strings.NewReader("response")}
	client := &http.Client{Transport: networkHealthRoundTripper(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusFound,
			Header:     make(http.Header),
			Body:       body,
			Request:    req,
		}, errors.New("response accompanied by a transport error")
	})}
	ok, _ := checkHTTPConnectivity(client, "https://api.example.invalid")
	if !ok || !body.closed {
		t.Fatalf("ok=%v, body.closed=%v; an HTTP response must be counted and closed", ok, body.closed)
	}
}
