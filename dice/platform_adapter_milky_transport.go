package dice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	milky "github.com/Szzrain/Milky-go-sdk"
)

const (
	milkyHealthInterval = 3 * time.Second
	milkyHealthTimeout  = 3 * time.Second
	milkyRetryMaxDelay  = 30 * time.Second
)

var errMilkyPermanentFailure = errors.New("milky configuration or authentication failure")

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
	active := pa.IntentSession == session && pa.sessionActive
	pa.lifecycleMu.Unlock()
	if active {
		go pa.watchMilkyHealth(ctx, session, milkyHealthInterval, milkyHealthTimeout)
	}
}

func (pa *PlatformAdapterMilky) watchMilkyHealth(ctx context.Context, session *milky.Session, interval, timeout time.Duration) {
	client := newMilkyHealthClient(timeout)
	delay := interval
	timer := time.NewTimer(delay)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		err := pa.refreshMilkyTransport(session, client)
		if ctx.Err() != nil {
			return
		}
		if errors.Is(err, errMilkyPermanentFailure) {
			session.Logger.Errorf("Milky 连接配置或鉴权失败，停止重试: %v", err)
			if pa.failMilkySession(session) {
				d := pa.EndPoint.Session.Parent
				d.LastUpdatedTime = time.Now().Unix()
				d.Save(false)
			}
			return
		}
		if err != nil {
			delay = min(delay*2, milkyRetryMaxDelay)
		} else {
			delay = interval
		}
		timer.Reset(delay)
	}
}

func newMilkyHealthClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:       timeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}
}

func (pa *PlatformAdapterMilky) refreshMilkyTransport(session *milky.Session, client *http.Client) error {
	pa.lifecycleMu.Lock()
	ctx := pa.sessionContext
	active := pa.IntentSession == session && pa.sessionActive
	expectedUserID := pa.EndPoint.UserID
	offlineGeneration := pa.offlineGeneration
	wasReady := pa.sessionReady
	pa.lifecycleMu.Unlock()
	if !active {
		return context.Canceled
	}
	if err := validateMilkyGateway(session.WSGateway, "ws", "wss"); err != nil {
		return err
	}
	if err := validateMilkyGateway(session.RestGateway, "http", "https"); err != nil {
		return err
	}
	info, err := probeMilkyHealth(ctx, client, session.RestGateway, session.Token, expectedUserID)
	if err != nil {
		pa.onMilkyConnectionChange(session, false)
		// API 不可用时关闭事件连接，恢复后重新建立，避免显示虚假的已连接。
		_ = pa.closeMilkyTransport(session)
		return err
	}
	if err := pa.openMilkyTransport(session); err != nil {
		return err
	}
	if !pa.finishMilkySession(session, info) {
		return context.Canceled
	}
	// 登录 API 及 WS 都正常才能清除离线状态，无需等待下一条 QQ 消息。
	pa.onMilkyAccountOnline(session, offlineGeneration)
	if !wasReady {
		session.Logger.Infof("Milky 服务连接成功，账号<%s>(%d)", info.Nickname, info.UIN)
	}
	return nil
}

func validateMilkyGateway(gateway string, scheme, secureScheme string) error {
	parsed, err := url.Parse(gateway)
	if err != nil || parsed.Hostname() == "" || (parsed.Scheme != scheme && parsed.Scheme != secureScheme) {
		return fmt.Errorf("%w: invalid %s gateway", errMilkyPermanentFailure, scheme)
	}
	return nil
}

// 使用标准 HTTP 检查登录信息，不依赖 SDK 新接口或内部连接字段。
func probeMilkyHealth(ctx context.Context, client *http.Client, gateway, token, expectedUserID string) (*milky.LoginInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(gateway, "/")+"/get_login_info", strings.NewReader("{}"))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		switch response.StatusCode {
		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusMethodNotAllowed, http.StatusUnsupportedMediaType:
			return nil, fmt.Errorf("%w: HTTP status %d", errMilkyPermanentFailure, response.StatusCode)
		default:
			return nil, fmt.Errorf("milky health HTTP status %d", response.StatusCode)
		}
	}
	var result struct {
		Status  string           `json:"status"`
		RetCode int              `json:"retcode"`
		Data    *milky.LoginInfo `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 64*1024)).Decode(&result); err != nil {
		return nil, err
	}
	if result.Status != "ok" || result.RetCode != 0 || result.Data == nil || result.Data.UIN <= 0 {
		return nil, errors.New("milky login info unavailable")
	}
	if expectedUserID != "" && fmt.Sprintf("QQ:%d", result.Data.UIN) != expectedUserID {
		return nil, fmt.Errorf("%w: milky account changed", errMilkyPermanentFailure)
	}
	return result.Data, nil
}
