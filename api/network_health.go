package api

import (
	"context"
	"net/http"
	"sync"
	"time"
)

const checkTimeout = 5 * time.Second

type networkHealthTarget struct {
	Target   string        `json:"target"`
	Ok       bool          `json:"ok"`
	Duration time.Duration `json:"duration"`
}

type networkHealthResult struct {
	Total     int
	Ok        []string
	Targets   []networkHealthTarget
	Timestamp int64
}

func checkHTTPConnectivity(client *http.Client, targetURL string) (bool, time.Duration) {
	start := time.Now()
	ctx := context.Background()
	if client.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, client.Timeout)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return false, 0
	}
	transport := client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	// Probe one response without parsing or following its Location header.
	resp, _ := transport.RoundTrip(req)
	if resp == nil {
		return false, 0
	}
	if resp.Body != nil {
		_ = resp.Body.Close()
	}
	// An HTTP response, including an authentication error, proves network reachability.
	return true, time.Since(start)
}

func runNetworkHealthCheck() networkHealthResult {
	client := &http.Client{Timeout: checkTimeout}
	return runNetworkHealthCheckWithClient(client)
}

func runNetworkHealthCheckWithClient(client *http.Client) networkHealthResult {
	endpoints := []struct {
		target string
		url    string
	}{
		{"qq", "https://api.sgroup.qq.com/gateway"},
		{"kook", "https://www.kookapp.cn/api/v3/gateway/index"},
		{"discord", "https://discord.com/api/v10/gateway"},
		{"telegram", "https://api.telegram.org"},
		{"dingtalk", "https://api.dingtalk.com/v1.0/gateway/connections/open"},
		{"slack", "https://slack.com/api/api.test"},
	}
	targets := make([]networkHealthTarget, len(endpoints))
	var wg sync.WaitGroup
	for i, endpoint := range endpoints {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, duration := checkHTTPConnectivity(client, endpoint.url)
			targets[i] = networkHealthTarget{
				Target:   endpoint.target,
				Ok:       ok,
				Duration: duration,
			}
		}()
	}
	wg.Wait()

	ok := make([]string, 0, len(targets))
	for _, target := range targets {
		if target.Ok {
			ok = append(ok, target.Target)
		}
	}
	return networkHealthResult{
		Total:     len(targets),
		Ok:        ok,
		Targets:   targets,
		Timestamp: time.Now().Unix(),
	}
}
