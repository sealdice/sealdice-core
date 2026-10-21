package networkhealth //nolint:testpackage // 测试内部探测行为

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestBotAPITargetsCoverBotPlatforms(t *testing.T) {
	want := []struct {
		key   string
		name  string
		label string
		url   string
	}{
		{"qq", "QQ", "QQ 机器人 API", "https://api.sgroup.qq.com/gateway"},
		{"kook", "KOOK", "KOOK 机器人 API", "https://www.kookapp.cn/api/v3/gateway/index"},
		{"discord", "Discord", "Discord 机器人 API", "https://discord.com/api/v10/gateway"},
		{"telegram", "Telegram", "Telegram 机器人 API", "https://api.telegram.org"},
		{"dingtalk", "钉钉", "钉钉机器人 API", "https://api.dingtalk.com/v1.0/gateway/connections/open"},
		{"slack", "Slack", "Slack 机器人 API", "https://slack.com/api/api.test"},
	}
	if len(BotAPITargets) != len(want) {
		t.Fatalf("got %d targets, want %d", len(BotAPITargets), len(want))
	}
	for i, target := range BotAPITargets {
		if target.Key != want[i].key || target.Name != want[i].name || target.Label != want[i].label || target.URL != want[i].url {
			t.Fatalf("target %d = %+v, want %+v", i, target, want[i])
		}
	}
}

func TestRunWithClientReportsTargetMetadataAndReachability(t *testing.T) {
	var mu sync.Mutex
	requested := map[string]bool{}
	client := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		mu.Lock()
		requested[req.URL.String()] = true
		mu.Unlock()
		if req.URL.Host == "slack.com" {
			return nil, errors.New("blocked API")
		}
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("{}")),
			Request:    req,
		}, nil
	})}

	result := RunWithClient(client)
	if result.Total != len(BotAPITargets) || len(result.Targets) != len(BotAPITargets) || result.Timestamp == 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	for i, target := range result.Targets {
		want := BotAPITargets[i]
		if target.Target != want.Key || target.Name != want.Name || target.Label != want.Label {
			t.Fatalf("target result %d = %+v, want metadata from %+v", i, target, want)
		}
		if target.OK != (want.Key != "slack") {
			t.Fatalf("target %s ok = %v", want.Key, target.OK)
		}
	}
	if len(result.OK) != len(BotAPITargets)-1 {
		t.Fatalf("ok targets = %v", result.OK)
	}
	for _, target := range BotAPITargets {
		mu.Lock()
		got := requested[target.URL]
		mu.Unlock()
		if !got {
			t.Fatalf("target %s was not requested", target.Key)
		}
	}
}

func TestCheckHTTPConnectivityAcceptsAnyHTTPResponse(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusTooManyRequests, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			client := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: status,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader("")),
					Request:    req,
				}, nil
			})}
			ok, duration := checkHTTPConnectivity(client, "https://api.example.invalid")
			if !ok || duration < 0 {
				t.Fatalf("status %d: ok=%v, duration=%v", status, ok, duration)
			}
		})
	}
}

func TestCheckHTTPConnectivityRejectsTransportFailures(t *testing.T) {
	client := &http.Client{Transport: roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
		return nil, errors.New("network unavailable")
	})}
	for _, targetURL := range []string{"https://unreachable.invalid", ":invalid-url"} {
		ok, duration := checkHTTPConnectivity(client, targetURL)
		if ok || duration != 0 {
			t.Fatalf("%q: ok=%v, duration=%v; want false, 0", targetURL, ok, duration)
		}
	}
}

func TestCheckHTTPConnectivityDoesNotFollowRedirects(t *testing.T) {
	var requests int
	client := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		return &http.Response{
			StatusCode: http.StatusFound,
			Header:     http.Header{"Location": []string{"https://redirect.invalid"}},
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    req,
		}, nil
	})}
	ok, _ := checkHTTPConnectivity(client, "https://api.example.invalid")
	if !ok || requests != 1 {
		t.Fatalf("ok=%v, requests=%d; a probe must accept the first response without following redirects", ok, requests)
	}
}

func TestCheckHTTPConnectivityTimesOut(t *testing.T) {
	client := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	}), Timeout: 20 * time.Millisecond}
	start := time.Now()
	ok, _ := checkHTTPConnectivity(client, "https://api.example.invalid")
	if ok || time.Since(start) > time.Second {
		t.Fatal("unresponsive API must fail within the client timeout")
	}
}
