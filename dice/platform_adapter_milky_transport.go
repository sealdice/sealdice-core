package dice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	milky "github.com/Szzrain/Milky-go-sdk"
)

const (
	milkyLoginRetryInterval = 3 * time.Second
	milkyLoginTimeout       = 3 * time.Second
	milkyRetryMaxDelay      = 30 * time.Second
)

var errMilkyPermanentFailure = errors.New("milky configuration or authentication failure")

// SDK 管理 WebSocket 连接和重连，核心仅在启动或收到状态事件后验证 QQ 登录信息。
func (pa *PlatformAdapterMilky) prepareMilkyTransport(session *milky.Session) {
	dialer := *session.Dialer
	dialer.HandshakeTimeout = milkyLoginTimeout
	session.Dialer = &dialer
	session.OnConnectionChange = pa.onMilkyTransportChange
}

func (pa *PlatformAdapterMilky) onMilkyTransportChange(session *milky.Session, connected bool) {
	currentConnected := session.IsConnected()
	pa.lifecycleMu.Lock()
	defer pa.lifecycleMu.Unlock()
	if pa.IntentSession != session || !pa.sessionActive {
		return
	}
	if connected {
		pa.sessionOpened = true
	}
	if currentConnected != connected {
		return // 过期通知仍可证明 SDK 已接管重连，但不能覆盖当前状态。
	}
	// WS 重连后也需要验证 QQ 登录，不能仅凭握手恢复账号状态。
	pa.transportConnected = false
	pa.updateMilkyConnectionState(false)
	pa.signalMilkyRecoveryLocked()
}

// 必须持有 lifecycleMu；合并重复的恢复请求，回调中不执行网络请求。
func (pa *PlatformAdapterMilky) signalMilkyRecoveryLocked() {
	select {
	case pa.sessionWake <- struct{}{}:
	default:
	}
}

func (pa *PlatformAdapterMilky) openMilkyTransport(session *milky.Session) error {
	if session.IsConnected() {
		return nil
	}
	pa.lifecycleMu.Lock()
	opened := pa.IntentSession == session && pa.sessionOpened
	pa.lifecycleMu.Unlock()
	if opened {
		return errors.New("waiting for SDK websocket reconnect")
	}
	err := session.Open()
	if err == nil || errors.Is(err, milky.ErrWSAlreadyOpen) {
		return nil
	}
	pa.onMilkyConnectionChange(session, false)
	return err
}

func (pa *PlatformAdapterMilky) closeMilkyTransport(session *milky.Session) error {
	return session.Shutdown()
}

func (pa *PlatformAdapterMilky) isMilkyTransportConnected(session *milky.Session) bool {
	return session.IsConnected()
}

func (pa *PlatformAdapterMilky) startMilkyRecovery(session *milky.Session) {
	pa.lifecycleMu.Lock()
	ctx := pa.sessionContext
	active := pa.IntentSession == session && pa.sessionActive
	pa.lifecycleMu.Unlock()
	if active {
		go pa.watchMilkyRecovery(ctx, session, milkyLoginRetryInterval, milkyLoginTimeout)
	}
}

func (pa *PlatformAdapterMilky) needsMilkyRecovery(session *milky.Session) (bool, bool) {
	pa.lifecycleMu.Lock()
	active := pa.IntentSession == session && pa.sessionActive
	needed := !pa.sessionReady || pa.accountOffline || !pa.transportConnected
	opened := pa.sessionOpened
	pa.lifecycleMu.Unlock()
	// 已建立过的 WS 由 SDK 重连；等待 SDK 通知，不轮询 API 或争抢 Open。
	return active && needed && (!opened || session.IsConnected()), active
}

func (pa *PlatformAdapterMilky) watchMilkyRecovery(ctx context.Context, session *milky.Session, interval, timeout time.Duration) {
	client := newMilkyLoginClient(timeout)
	pa.lifecycleMu.Lock()
	wake := pa.sessionWake
	pa.lifecycleMu.Unlock()
	delay := interval
	timer := time.NewTimer(delay)
	defer timer.Stop()
	for {
		needed, active := pa.needsMilkyRecovery(session)
		if !active {
			return
		}
		var retry <-chan time.Time
		if needed {
			retry = timer.C
		}
		select {
		case <-ctx.Done():
			return
		case <-wake:
		case <-retry:
		}
		if needed, _ := pa.needsMilkyRecovery(session); !needed {
			continue
		}
		pa.lifecycleMu.Lock()
		wasReady := pa.sessionReady
		pa.lifecycleMu.Unlock()
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
			if !wasReady {
				// Serve 已保存过首次失败的配置；后台首次登录成功后保存账号信息。
				d := pa.EndPoint.Session.Parent
				d.MarkModified()
				d.Save(false)
			}
			delay = interval
		}
		timer.Reset(delay)
	}
}

func newMilkyLoginClient(timeout time.Duration) *http.Client {
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
	info, err := probeMilkyLogin(ctx, client, session.RestGateway, session.Token, expectedUserID)
	if err != nil {
		pa.onMilkyConnectionChange(session, false)
		return err
	}
	if err := pa.openMilkyTransport(session); err != nil {
		return err
	}
	pa.onMilkyConnectionChange(session, session.IsConnected())
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
func probeMilkyLogin(ctx context.Context, client *http.Client, gateway, token, expectedUserID string) (*milky.LoginInfo, error) {
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
