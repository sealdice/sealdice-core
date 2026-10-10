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
