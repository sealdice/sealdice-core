package networkhealth

import (
	"context"
	"net/http"
	"sync"
	"time"
)

const CheckTimeout = 5 * time.Second

// Target 描述一个网络检测目标。Key 是稳定的机器标识，Name 与 Label 用于展示，
// 新增或调整检测目标时无需同步修改前端。
type Target struct {
	Key   string
	Name  string
	Label string
	URL   string
}

// TargetResult 是单个目标的检测结果。
type TargetResult struct {
	Target   string        `json:"target"`
	Name     string        `json:"name"`
	Label    string        `json:"label"`
	OK       bool          `json:"ok"`
	Duration time.Duration `json:"duration"`
}

// Result 是一次网络检测的完整结果。
type Result struct {
	Total     int            `json:"total"`
	OK        []string       `json:"ok"`
	Targets   []TargetResult `json:"targets"`
	Timestamp int64          `json:"timestamp"`
}

// BotAPITargets 是机器人平台的 API 端点。检测这些端点能直接反映
// 各机器人平台在当前网络环境下的可达性。
var BotAPITargets = []Target{
	{Key: "qq", Name: "QQ", Label: "QQ 机器人 API", URL: "https://api.sgroup.qq.com/gateway"},
	{Key: "kook", Name: "KOOK", Label: "KOOK 机器人 API", URL: "https://www.kookapp.cn/api/v3/gateway/index"},
	{Key: "discord", Name: "Discord", Label: "Discord 机器人 API", URL: "https://discord.com/api/v10/gateway"},
	{Key: "telegram", Name: "Telegram", Label: "Telegram 机器人 API", URL: "https://api.telegram.org"},
	{Key: "dingtalk", Name: "钉钉", Label: "钉钉机器人 API", URL: "https://api.dingtalk.com/v1.0/gateway/connections/open"},
	{Key: "slack", Name: "Slack", Label: "Slack 机器人 API", URL: "https://slack.com/api/api.test"},
}

// Run 使用默认超时检测全部机器人平台 API。
func Run() Result {
	return RunWithClient(&http.Client{Timeout: CheckTimeout})
}

// RunWithClient 使用给定客户端检测全部机器人平台 API。
func RunWithClient(client *http.Client) Result {
	if client == nil {
		client = &http.Client{Timeout: CheckTimeout}
	}

	targets := make([]TargetResult, len(BotAPITargets))
	var wg sync.WaitGroup
	for index, target := range BotAPITargets {
		wg.Add(1)
		go func(index int, target Target) {
			defer wg.Done()
			ok, duration := checkHTTPConnectivity(client, target.URL)
			targets[index] = TargetResult{
				Target:   target.Key,
				Name:     target.Name,
				Label:    target.Label,
				OK:       ok,
				Duration: duration,
			}
		}(index, target)
	}
	wg.Wait()

	ok := make([]string, 0, len(targets))
	for _, target := range targets {
		if target.OK {
			ok = append(ok, target.Target)
		}
	}

	return Result{
		Total:     len(targets),
		OK:        ok,
		Targets:   targets,
		Timestamp: time.Now().Unix(),
	}
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
	// 只探测一次响应，不解析也不跟随 Location 头。
	resp, _ := transport.RoundTrip(req)
	if resp == nil {
		return false, 0
	}
	if resp.Body != nil {
		_ = resp.Body.Close()
	}
	// 收到任意 HTTP 响应（包括鉴权失败）都说明网络可达。
	return true, time.Since(start)
}
