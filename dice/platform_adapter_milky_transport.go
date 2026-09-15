package dice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	milky "github.com/Szzrain/Milky-go-sdk"
)

const (
	milkyHealthInterval = 3 * time.Second
	milkyHealthTimeout  = 3 * time.Second
)

// 原版 SDK 没有断开通知。临时由核心检查 API 并管理重连，避免两套重连并行。
// 与 SDK 通知方案共用状态和进程管理；上游支持通知后，只需替换本文件。
func (pa *PlatformAdapterMilky) prepareMilkyTransport(session *milky.Session) {
	session.ShouldReconnectOnError = false
	pa.lifecycleMu.Lock()
	ctx := pa.sessionContext
	pa.lifecycleMu.Unlock()
	dialer := *session.Dialer
	dialer.HandshakeTimeout = milkyHealthTimeout
	dialer.NetDialContext = func(_ context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{Timeout: milkyHealthTimeout}).DialContext(ctx, network, address)
	}
	session.Dialer = &dialer
}

func (pa *PlatformAdapterMilky) openMilkyTransport(session *milky.Session) error {
	err := session.Open()
	if err == nil || errors.Is(err, milky.ErrWSAlreadyOpen) {
		pa.onMilkyConnectionChange(session, true)
		return nil
	}
	pa.onMilkyConnectionChange(session, false)
	return err
}

func (pa *PlatformAdapterMilky) closeMilkyTransport(session *milky.Session) error {
	return session.Close()
}

func (pa *PlatformAdapterMilky) isMilkyTransportConnected(session *milky.Session) bool {
	pa.lifecycleMu.Lock()
	defer pa.lifecycleMu.Unlock()
	return pa.IntentSession == session && pa.sessionActive && pa.transportConnected
}

func (pa *PlatformAdapterMilky) startMilkyConnectionMonitor(session *milky.Session) {
	pa.lifecycleMu.Lock()
	ctx := pa.sessionContext
	active := pa.IntentSession == session && pa.sessionActive && pa.sessionReady
	pa.lifecycleMu.Unlock()
	if active {
		go pa.watchMilkyHealth(ctx, session, milkyHealthInterval, milkyHealthTimeout)
	}
}

func (pa *PlatformAdapterMilky) watchMilkyHealth(ctx context.Context, session *milky.Session, interval, timeout time.Duration) {
	client := &http.Client{
		Timeout:       timeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		pa.lifecycleMu.Lock()
		active := pa.IntentSession == session && pa.sessionActive && pa.sessionReady
		expectedUserID := pa.EndPoint.UserID
		pa.lifecycleMu.Unlock()
		if !active {
			return
		}
		err := probeMilkyHealth(ctx, client, session.RestGateway, session.Token, expectedUserID)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			pa.onMilkyConnectionChange(session, false)
			// API 不可用时关闭事件连接，恢复后重新建立，避免显示虚假的已连接。
			_ = pa.closeMilkyTransport(session)
			continue
		}
		// Open 返回 ErrWSAlreadyOpen 表示连接仍在；否则尝试连接当前网关。
		// 不能仅凭 REST 恢复就显示已连接，WS 握手也必须成功。
		_ = pa.openMilkyTransport(session)
	}
}

// 使用标准 HTTP 检查登录信息，不依赖 SDK 新接口或内部连接字段。
func probeMilkyHealth(ctx context.Context, client *http.Client, gateway, token, expectedUserID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(gateway, "/")+"/get_login_info", strings.NewReader("{}"))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("milky health HTTP status %d", response.StatusCode)
	}
	var result struct {
		Status  string `json:"status"`
		RetCode int    `json:"retcode"`
		Data    *struct {
			UIN int64 `json:"uin"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 64*1024)).Decode(&result); err != nil {
		return err
	}
	if result.Status != "ok" || result.RetCode != 0 || result.Data == nil || result.Data.UIN <= 0 {
		return errors.New("milky login info unavailable")
	}
	if fmt.Sprintf("QQ:%d", result.Data.UIN) != expectedUserID {
		return errors.New("milky account changed")
	}
	return nil
}
