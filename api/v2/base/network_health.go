package base

import (
	"context"
	"net/http"
	"net/url"
	"sync"
	"time"

	"sealdice-core/dice"
	"sealdice-core/model/common/request"
	"sealdice-core/model/common/response"
)

const (
	networkHealthTotal        = 5
	networkHealthCheckTimes   = 3
	networkHealthCheckTimeout = 5 * time.Second
)

type networkHealthCheckFunc func(target string, urls []string) (bool, time.Duration)
type networkHealthSignURLsFunc func(*dice.Dice) []string

var (
	networkHealthCheck    networkHealthCheckFunc    = checkNetworkTargetConnectivity
	networkHealthSignURLs networkHealthSignURLsFunc = defaultSignServerPingURLs
)

func SetNetworkHealthTestHooks(check networkHealthCheckFunc, signURLs networkHealthSignURLsFunc) {
	if check != nil {
		networkHealthCheck = check
	}
	if signURLs != nil {
		networkHealthSignURLs = signURLs
	}
}

func ResetNetworkHealthTestHooks() {
	networkHealthCheck = checkNetworkTargetConnectivity
	networkHealthSignURLs = defaultSignServerPingURLs
}

func (s *BaseService) NetworkHealth(_ context.Context, _ *request.Empty) (*response.ItemResponse[NetworkHealthData], error) {
	type targetSpec struct {
		name string
		urls []string
	}

	specs := []targetSpec{
		{name: "baidu", urls: []string{"https://baidu.com"}},
		{name: "seal", urls: dice.BackendUrls},
		{name: "sign", urls: networkHealthSignURLs(s.dice)},
		{name: "google", urls: []string{"https://google.com"}},
		{name: "github", urls: []string{"https://github.com"}},
	}

	targets := make([]NetworkHealthTarget, len(specs))
	var wg sync.WaitGroup
	for index, spec := range specs {
		index, spec := index, spec
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, duration := false, time.Duration(0)
			if len(spec.urls) > 0 {
				ok, duration = networkHealthCheck(spec.name, spec.urls)
			}
			targets[index] = NetworkHealthTarget{
				Target:     spec.name,
				OK:         ok,
				DurationMs: duration.Milliseconds(),
			}
		}()
	}
	wg.Wait()

	okTargets := make([]string, 0, len(targets))
	for _, target := range targets {
		if target.OK {
			okTargets = append(okTargets, target.Target)
		}
	}

	return response.NewItemResponse(NetworkHealthData{
		Total:     networkHealthTotal,
		OK:        okTargets,
		Targets:   targets,
		Timestamp: time.Now().Unix(),
	}), nil
}

func defaultSignServerPingURLs(d *dice.Dice) []string {
	signGroups, err := dice.LagrangeGetSignInfo(d)
	if err != nil || len(signGroups) == 0 {
		return nil
	}
	signServers := signGroups[len(signGroups)-1].Servers
	urls := make([]string, 0, len(signServers))
	for _, signServer := range signServers {
		if signServer == nil || signServer.Url == "" {
			continue
		}
		ping, err := url.JoinPath(signServer.Url, "/ping")
		if err != nil {
			continue
		}
		urls = append(urls, ping)
	}
	return urls
}

func checkNetworkTargetConnectivity(_ string, urls []string) (bool, time.Duration) {
	for _, targetURL := range urls {
		ok, duration := checkHTTPConnectivity(targetURL)
		if ok {
			return true, duration
		}
	}
	return false, 0
}

func checkHTTPConnectivity(targetURL string) (bool, time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), networkHealthCheckTimeout)
	defer cancel()

	type checkResult struct {
		ok       bool
		duration time.Duration
	}
	results := make(chan checkResult, networkHealthCheckTimes)

	var wg sync.WaitGroup
	for range networkHealthCheckTimes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
			if err != nil {
				results <- checkResult{ok: false, duration: time.Since(start)}
				return
			}
			resp, err := http.DefaultClient.Do(req) //nolint:gosec
			duration := time.Since(start)
			if err != nil {
				results <- checkResult{ok: false, duration: duration}
				return
			}
			_ = resp.Body.Close()
			results <- checkResult{ok: true, duration: duration}
		}()
	}
	wg.Wait()
	close(results)

	ok := true
	var totalDuration time.Duration
	var count int64
	for result := range results {
		ok = ok && result.ok
		if result.ok {
			count++
			totalDuration += result.duration
		}
	}
	if count == 0 {
		return ok, 0
	}
	return ok, time.Duration(int64(totalDuration) / count)
}
