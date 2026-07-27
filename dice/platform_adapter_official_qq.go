package dice

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	crand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	qqconstant "github.com/sealdice/botgo/constant"
	"github.com/sealdice/botgo/event"
	"github.com/sealdice/botgo/interaction/signature"
	"github.com/skip2/go-qrcode"

	qqbot "github.com/sealdice/botgo"
	"github.com/sealdice/botgo/dto"
	"github.com/sealdice/botgo/dto/keyboard"
	qqapi "github.com/sealdice/botgo/openapi"
	qqtoken "github.com/sealdice/botgo/token"
	"golang.org/x/oauth2"

	"sealdice-core/message"
	"sealdice-core/utils"
)

var officialQQAtRegex = regexp.MustCompile(`<@!?(\S+?)>`)

type PaginationItem struct {
	Pages     []string
	CreatedAt time.Time
}

type PlatformAdapterOfficialQQ struct {
	EndPoint    *EndPointInfo `json:"-" yaml:"-"`
	DiceServing bool          `yaml:"-"`

	AppID       string `json:"appID"       yaml:"appID"`
	AppSecret   string `json:"-"           yaml:"appSecret"`
	Token       string `json:"-"           yaml:"token,omitempty"` // Deprecated: preserved for old configurations and never used.
	UIN         string `json:"uin"         yaml:"uin,omitempty"`
	OnlyQQGuild bool   `json:"onlyQQGuild" yaml:"onlyQQGuild"`

	// Webhook配置
	UseWebhook  bool   `json:"useWebhook"    yaml:"useWebhook"`  // 是否使用Webhook模式
	WebhookPath string `json:"webhookPath"   yaml:"webhookPath"` // Webhook路径，默认/webhook
	WebhookPort int    `json:"webhookPort"   yaml:"webhookPort"` // Webhook端口，默认8099

	Api            qqapi.OpenAPI        `json:"-" yaml:"-"`
	SessionManager qqbot.SessionManager `json:"-" yaml:"-"`
	Ctx            context.Context      `json:"-" yaml:"-"`
	CancelFunc     context.CancelFunc   `json:"-" yaml:"-"`
	tokenSource    oauth2.TokenSource   `json:"-" yaml:"-"`
	botID          string               `json:"-" yaml:"-"`

	// Webhook服务
	webhookServer *http.Server `json:"-" yaml:"-"`

	paginationCache map[string]*PaginationItem `json:"-" yaml:"-"`
	paginationMu    sync.Mutex                 `json:"-" yaml:"-"`

	// 扫码登录状态（参考 milky 的 BuiltInLoginState / QrCodeData 设计）
	// 当 AppID 为空时，Serve() 会进入扫码登录分支
	QrLoginState OfficialQQLoginState      `json:"qrLoginState" yaml:"-"`
	QrURL        string                    `json:"qrUrl"       yaml:"-"` // 当前二维码 URL
	QrCodeData   []byte                    `json:"-"           yaml:"-"` // 当前二维码图片数据（PNG），参考 milky 的 QrCodeData
	qrSession    *QQOfficialQrLoginSession `json:"-" yaml:"-"`
	qrMu         sync.Mutex                `json:"-" yaml:"-"`
}

type officialQQAdapterJSON struct {
	AppID               string               `json:"appID"`
	UIN                 string               `json:"uin"`
	AppSecretConfigured bool                 `json:"appSecretConfigured"`
	OnlyQQGuild         bool                 `json:"onlyQQGuild"`
	UseWebhook          bool                 `json:"useWebhook"`
	WebhookPath         string               `json:"webhookPath"`
	WebhookPort         int                  `json:"webhookPort"`
	QrLoginState        OfficialQQLoginState `json:"qrLoginState"`
}

func (pa *PlatformAdapterOfficialQQ) MarshalJSON() ([]byte, error) {
	pa.qrMu.Lock()
	qrLoginState := pa.QrLoginState
	pa.qrMu.Unlock()
	return json.Marshal(officialQQAdapterJSON{
		AppID:               pa.AppID,
		UIN:                 pa.UIN,
		AppSecretConfigured: strings.TrimSpace(pa.AppSecret) != "",
		OnlyQQGuild:         pa.OnlyQQGuild,
		UseWebhook:          pa.UseWebhook,
		WebhookPath:         pa.WebhookPath,
		WebhookPort:         pa.WebhookPort,
		QrLoginState:        qrLoginState,
	})
}

type OfficialQQAccountProbeResult struct {
	UIN      string
	BotID    string
	Nickname string
}

type officialQQBotInfoResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	ShareURL string `json:"share_url"`
}

type officialQQTransport interface {
	Transport(ctx context.Context, method, url string, body interface{}) ([]byte, error)
}

func extractOfficialQQBotUIN(link string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(link))
	if err != nil {
		return "", fmt.Errorf("解析机器人分享链接失败: %w", err)
	}
	uin := strings.TrimSpace(parsed.Query().Get("robot_uin"))
	if uin == "" {
		return "", errors.New("机器人分享链接中缺少 robot_uin")
	}
	value, err := strconv.ParseUint(uin, 10, 64)
	if err != nil || value == 0 {
		return "", fmt.Errorf("机器人 UIN 无效: %q", uin)
	}
	return uin, nil
}

func extractOfficialQQBotUINFromResponse(body []byte) (string, error) {
	var response struct {
		URL     string `json:"url"`
		URLLink string `json:"url_link"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("解析机器人分享链接响应失败: %w", err)
	}
	link := strings.TrimSpace(response.URLLink)
	if link == "" {
		link = strings.TrimSpace(response.URL) // Compatibility with SDKs that expose this field as url.
	}
	if link == "" {
		return "", errors.New("机器人分享链接响应中缺少 url_link")
	}
	return extractOfficialQQBotUIN(link)
}

func getOfficialQQBotUINFromGeneratedLink(ctx context.Context, api officialQQTransport) (string, error) {
	body, err := api.Transport(
		ctx,
		http.MethodPost,
		qqconstant.APIDomain+"/v2/generate_url_link",
		map[string]string{"callback_data": "sealdice"},
	)
	if err != nil {
		return "", fmt.Errorf("生成机器人分享链接失败: %w", err)
	}
	return extractOfficialQQBotUINFromResponse(body)
}

func parseOfficialQQBotInfo(body []byte) (*officialQQBotInfoResponse, error) {
	var response officialQQBotInfoResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("解析机器人信息失败: %w", err)
	}
	response.ID = strings.TrimSpace(response.ID)
	response.Username = strings.TrimSpace(response.Username)
	response.ShareURL = strings.TrimSpace(response.ShareURL)
	if response.ID == "" {
		return nil, errors.New("机器人信息响应中缺少 id")
	}
	return &response, nil
}

func getOfficialQQBotInfo(ctx context.Context, api officialQQTransport) (*OfficialQQAccountProbeResult, error) {
	body, err := api.Transport(ctx, http.MethodGet, qqconstant.APIDomain+"/users/@me", nil)
	if err != nil {
		return nil, fmt.Errorf("获取机器人信息失败: %w", err)
	}
	botInfo, err := parseOfficialQQBotInfo(body)
	if err != nil {
		return nil, err
	}

	var shareURLErr error
	if botInfo.ShareURL != "" {
		if uin, err := extractOfficialQQBotUIN(botInfo.ShareURL); err == nil {
			return &OfficialQQAccountProbeResult{UIN: uin, BotID: botInfo.ID, Nickname: botInfo.Username}, nil
		} else {
			shareURLErr = err
		}
	} else {
		shareURLErr = errors.New("机器人信息响应中缺少 share_url")
	}

	// Older deployments may not include share_url in /users/@me yet.
	uin, generatedLinkErr := getOfficialQQBotUINFromGeneratedLink(ctx, api)
	if generatedLinkErr != nil {
		return nil, fmt.Errorf("无法从机器人信息或生成的分享链接获取 UIN: %w; %w", shareURLErr, generatedLinkErr)
	}
	return &OfficialQQAccountProbeResult{UIN: uin, BotID: botInfo.ID, Nickname: botInfo.Username}, nil
}

func ProbeOfficialQQAccount(ctx context.Context, appID, appSecret string) (*OfficialQQAccountProbeResult, error) {
	appID = strings.TrimSpace(appID)
	appSecret = strings.TrimSpace(appSecret)
	if appID == "" {
		return nil, errors.New("AppID 不能为空")
	}
	if appSecret == "" {
		return nil, errors.New("AppSecret 不能为空")
	}
	qqbot.SetLogger(NewDummyLogger())

	tokenSource := qqtoken.NewQQBotTokenSource(&qqtoken.QQBotCredentials{
		AppID:     appID,
		AppSecret: appSecret,
	})
	if _, err := tokenSource.Token(); err != nil {
		return nil, fmt.Errorf("获取 Access Token 失败: %w", err)
	}
	api := qqbot.NewOpenAPI(appID, tokenSource).WithTimeout(5 * time.Second)
	botInfo, err := getOfficialQQBotInfo(ctx, api)
	if err != nil {
		return nil, err
	}
	ws, err := api.WS(ctx, nil, "")
	if err != nil {
		return nil, fmt.Errorf("获取 QQ 官方机器人网关接入点失败，请检查 IP 白名单: %w", err)
	}
	if ws == nil {
		return nil, errors.New("QQ 官方机器人未返回网关接入点，请检查 IP 白名单")
	}
	return botInfo, nil
}

// FindOfficialQQEndpointByUIN 按 UIN 查找已存在的官方 QQ 连接；excludeID 非空时排除自身
func FindOfficialQQEndpointByUIN(s *IMSession, uin, excludeID string) *EndPointInfo {
	if s == nil {
		return nil
	}
	userID := formatDiceIDOfficialQQ(strings.TrimSpace(uin))
	if userID == formatDiceIDOfficialQQ("") {
		return nil
	}
	for _, endpoint := range s.EndPoints {
		if endpoint == nil {
			continue
		}
		if endpoint.Platform == "QQ" && endpoint.ProtocolType == "official" && endpoint.UserID == userID && endpoint.ID != excludeID {
			return endpoint
		}
	}
	return nil
}

func replaceOfficialQQCredentials(existing *EndPointInfo, source *PlatformAdapterOfficialQQ, probe *OfficialQQAccountProbeResult) (*PlatformAdapterOfficialQQ, error) {
	if existing == nil || source == nil || probe == nil {
		return nil, errors.New("official qq replacement data is incomplete")
	}
	adapter, ok := existing.Adapter.(*PlatformAdapterOfficialQQ)
	if !ok {
		return nil, errors.New("official qq existing endpoint adapter is invalid")
	}

	existingUIN := strings.TrimSpace(adapter.UIN)
	if existingUIN == "" {
		existingUIN, _ = extractOfficialQQUINFromUserID(existing.UserID)
	}
	if existingUIN != "" && existingUIN != strings.TrimSpace(probe.UIN) {
		return nil, fmt.Errorf("official qq existing endpoint UIN mismatch: existing=%s scanned=%s", existingUIN, probe.UIN)
	}

	adapter.AppID = strings.TrimSpace(source.AppID)
	adapter.AppSecret = strings.TrimSpace(source.AppSecret)
	adapter.UIN = strings.TrimSpace(probe.UIN)
	adapter.botID = strings.TrimSpace(probe.BotID)
	existing.UserID = formatDiceIDOfficialQQ(adapter.UIN)
	existing.Nickname = probe.Nickname
	return adapter, nil
}

func (pa *PlatformAdapterOfficialQQ) applyProbeResult(probe *OfficialQQAccountProbeResult) {
	if probe == nil || pa.EndPoint == nil {
		return
	}
	pa.UIN = strings.TrimSpace(probe.UIN)
	pa.botID = strings.TrimSpace(probe.BotID)
	pa.EndPoint.UserID = formatDiceIDOfficialQQ(pa.UIN)
	pa.EndPoint.Nickname = probe.Nickname
}

func (pa *PlatformAdapterOfficialQQ) stopSessionContext() {
	if pa.CancelFunc != nil {
		pa.CancelFunc()
	}
	pa.Ctx = nil
	pa.CancelFunc = nil
}

func (pa *PlatformAdapterOfficialQQ) failConnect() int {
	ep := pa.EndPoint
	if ep != nil {
		ep.State = 3
	}
	if pa.CancelFunc != nil {
		pa.CancelFunc()
	}
	pa.Api = nil
	pa.SessionManager = nil
	pa.Ctx = nil
	pa.CancelFunc = nil
	// 仅扫码流程需要同步更新二维码状态，避免普通重连误标 Failed
	pa.qrMu.Lock()
	isQrFlow := pa.QrLoginState == OfficialQQLoginStateConnecting ||
		pa.QrLoginState == OfficialQQLoginStateQRWaitingForScan ||
		pa.QrLoginState == OfficialQQLoginStateQRScanned
	pa.qrMu.Unlock()
	if isQrFlow {
		pa.markQrLoginFailed()
	}
	return 1
}

func (pa *PlatformAdapterOfficialQQ) Serve() int {
	ep := pa.EndPoint
	log := pa.EndPoint.Session.Parent.Logger

	if pa.Ctx != nil {
		log.Info("official qq session already running, skip Serve")
		return 0
	}

	pa.AppID = strings.TrimSpace(pa.AppID)
	pa.AppSecret = strings.TrimSpace(pa.AppSecret)
	pa.UIN = strings.TrimSpace(pa.UIN)

	// AppID 为空时进入扫码登录：异步生成二维码并轮询，
	// 扫码成功后探测凭据、检查重复账号，再启动连接
	if pa.AppID == "" {
		ctx, cancel := context.WithCancel(context.Background())
		pa.Ctx, pa.CancelFunc = ctx, cancel
		ep.State = 2
		log.Info("official qq AppID 为空，进入扫码登录流程")
		pa.serveQrLogin("sealdice")
		return 0
	}

	return pa.connect(nil)
}

// connect 建立正式连接。probe 非空时复用探测结果，避免重复拉取机器人信息。
// 调用方需确保 AppID/AppSecret 已就绪。
func (pa *PlatformAdapterOfficialQQ) connect(probe *OfficialQQAccountProbeResult) int {
	ep := pa.EndPoint
	log := ep.Session.Parent.Logger
	d := ep.Session.Parent
	pa.AppID = strings.TrimSpace(pa.AppID)
	pa.AppSecret = strings.TrimSpace(pa.AppSecret)
	pa.UIN = strings.TrimSpace(pa.UIN)

	log.Debug("official qq server")
	qqbot.SetLogger(NewDummyLogger())

	// 初始化OAuth2 token source
	pa.tokenSource = qqtoken.NewQQBotTokenSource(&qqtoken.QQBotCredentials{
		AppID:     pa.AppID,
		AppSecret: pa.AppSecret,
	})

	ctx, cancel := context.WithCancel(context.Background())
	pa.Ctx, pa.CancelFunc = ctx, cancel

	// 启动 token 自动刷新
	if err := qqtoken.StartRefreshAccessToken(ctx, pa.tokenSource); err != nil {
		log.Error("official qq 启动 token 刷新失败: ", err)
		return pa.failConnect()
	}

	pa.Api = qqbot.NewOpenAPI(pa.AppID, pa.tokenSource).WithTimeout(3 * time.Second)

	botInfo := probe
	if botInfo == nil {
		var err error
		botInfo, err = getOfficialQQBotInfo(ctx, pa.Api)
		if err != nil {
			log.Error("official qq 获取机器人信息失败: ", err)
			return pa.failConnect()
		}
	}

	uin := botInfo.UIN
	if pa.UIN != "" && pa.UIN != uin {
		log.Errorf("official qq UIN 校验失败: 配置为 %s，远端返回 %s", pa.UIN, uin)
		return pa.failConnect()
	}
	if err := ensureOfficialQQIdentity(d, pa, botInfo.BotID, uin); err != nil {
		var migrationErr *officialQQIdentityMigrationError
		if errors.As(err, &migrationErr) {
			log.Error("official qq 身份数据迁移失败，将继续启动并在下次连接时重试: ", err)
		} else {
			log.Error("official qq 身份初始化失败: ", err)
			return pa.failConnect()
		}
	}

	pa.botID = botInfo.BotID
	ep.Nickname = botInfo.Nickname

	// 身份校验和可选的数据迁移完成后再开始接收事件。
	event.RegisterHandlersByAppID(
		pa.AppID,
		pa.makeHandlers()...,
	)

	// 区分 Webhook 还是 WebSocket 模式
	if pa.UseWebhook {
		// 启动 webhook 服务
		if pa.WebhookPath == "" {
			pa.WebhookPath = "/webhook"
		}
		if pa.WebhookPort == 0 {
			pa.WebhookPort = 8099
		}

		mux := http.NewServeMux()
		mux.HandleFunc(pa.WebhookPath, pa.handleWebhookCallback)

		addr := fmt.Sprintf(":%d", pa.WebhookPort)
		pa.webhookServer = &http.Server{
			Addr:              addr,
			Handler:           mux,
			ReadHeaderTimeout: 3 * time.Second,
		}

		go func() {
			log.Infof("official qq webhook: 监听地址 %s%s", addr, pa.WebhookPath)
			if err := pa.webhookServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Error("official qq webhook服务器启动失败: ", err)
				if pa.Ctx == ctx {
					ep.State = 3
					ep.Enable = false
				}
			}
		}()

		ep.State = 1
		ep.Enable = true
		pa.clearQrLoginState()
		d.LastUpdatedTime = time.Now().Unix()
		d.Save(false)
		log.Info("official qq webhook模式启动成功")
		return 0
	}

	pa.SessionManager = qqbot.NewSessionManager()
	ep.State = 2
	log.Debug("official qq connecting")
	ws, err := pa.Api.WS(ctx, nil, "")
	if err != nil || ws == nil {
		log.Error("official qq 获取 ws 接入点失败? ", err)
		log.Error("official qq 提示：请确认在机器人后台配置 IP 白名单，并检查 AppID/AppSecret 是否正确")
		return pa.failConnect()
	}
	if ws.Shards == 0 {
		ws.Shards = 1
	}
	if ws.Shards > ws.SessionStartLimit.Remaining {
		log.Errorf(
			"official qq session limited: shards=%d remaining=%d resetAfter=%d maxConcurrency=%d",
			ws.Shards, ws.SessionStartLimit.Remaining, ws.SessionStartLimit.ResetAfter, ws.SessionStartLimit.MaxConcurrency,
		)
		return pa.failConnect()
	}

	var intent dto.Intent
	// 文字子频道at消息
	intent |= dto.IntentGuildAtMessage
	// 频道私信
	intent |= dto.IntentDirectMessages
	// 互动事件
	intent |= dto.IntentInteraction

	if !pa.OnlyQQGuild {
		// 群聊@消息、单聊、好友关系事件
		intent |= dto.IntentGroupMessages
	}

	go func() {
		currentCtx := ctx
		defer func() {
			isCurrent := pa.Ctx == currentCtx
			// 防止崩掉进程
			if r := recover(); r != nil {
				log.Error("official qq 启动失败: ", r)
				if isCurrent {
					ep.State = 3
					ep.Enable = false
				}
			}
			if isCurrent {
				pa.Ctx = nil
				pa.CancelFunc = nil
				pa.SessionManager = nil
			}
		}()
		if startErr := pa.SessionManager.Start(currentCtx, ws, pa.tokenSource, &intent); startErr != nil {
			log.Error("official qq session manager 启动失败: ", startErr)
			if pa.Ctx == currentCtx {
				ep.State = 3
				ep.Enable = false
			}
		}
	}()

	ep.State = 1
	ep.Enable = true
	pa.clearQrLoginState()
	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
	log.Info("official qq 连接成功")
	return 0
}

// makeHandlers 构造事件处理器列表
func (pa *PlatformAdapterOfficialQQ) makeHandlers() []interface{} {
	handlers := []interface{}{
		// 频道@消息
		event.ATMessageEventHandler(pa.ChannelAtMessageReceive),
		// 频道私信
		event.DirectMessageEventHandler(pa.GuildDirectMessageReceive),
		// 互动事件 (按钮点击)
		event.InteractionEventHandler(pa.InteractionReceive),
	}

	if !pa.OnlyQQGuild {
		handlers = append(handlers,
			// 群聊@消息
			event.GroupATMessageEventHandler(pa.GroupAtMessageReceive),
			// 群聊普通消息(非@)
			event.GroupMessageEventHandler(pa.GroupMessageReceive),
			// 单聊消息
			event.C2CMessageEventHandler(pa.C2CMessageReceiveFromEvent),
			// 好友关系事件
			event.C2CFriendEventHandler(pa.C2CFriendReceive),
			// 机器人加入群聊
			event.GroupAddRobotEventHandler(pa.GroupAddRobotReceive),
			// 机器人退出群聊
			event.GroupDelRobotEventHandler(pa.GroupDelRobotReceive),
			// 群成员加群
			event.GroupMemberAddEventHandler(pa.GroupMemberAddReceive),
			// 群成员退群
			event.GroupMemberRemoveEventHandler(pa.GroupMemberRemoveReceive),
		)
	}

	return handlers
}

func (pa *PlatformAdapterOfficialQQ) InteractionReceive(eventRaw *dto.WSPayload, data *dto.WSInteractionData) error {
	s := pa.EndPoint.Session
	log := s.Parent.Logger
	log.Debugf("official qq: 收到互动事件: %v, %v", eventRaw, data)

	qctx := context.Background()
	// 首先响应这个 interaction，让客户端停止loading
	if err := pa.Api.PutInteraction(qctx, data.ID, `{"code":0}`); err != nil {
		log.Errorf("official qq 响应互动事件失败: %v", err)
	}

	if data.Data == nil || data.Data.Type != dto.InteractionDataTypeInlineKeyboardClick {
		return nil
	}

	var resolved dto.Resolved
	if err := json.Unmarshal(data.Data.Resolved, &resolved); err != nil {
		log.Errorf("official qq 解析互动事件 Resolved 失败: %v", err)
		return nil
	}

	buttonData := resolved.ButtonData
	if !strings.HasPrefix(buttonData, "pg:") {
		return nil
	}

	// 格式 pg:<cacheID>:<pageIndex>
	parts := strings.Split(buttonData, ":")
	if len(parts) != 3 {
		return nil
	}

	cacheID := parts[1]
	pageIndexVal := parts[2]
	var pageIndex int
	if _, err := fmt.Sscanf(pageIndexVal, "%d", &pageIndex); err != nil {
		return err
	}

	pa.paginationMu.Lock()
	item, ok := pa.paginationCache[cacheID]
	pa.paginationMu.Unlock()

	if !ok {
		log.Warnf("official qq 翻页失败：未找到缓存的翻页消息ID %s", cacheID)
		return nil
	}

	if pageIndex < 0 || pageIndex >= len(item.Pages) {
		log.Warnf("official qq 翻页失败：页码 %d 越界 (总数 %d)", pageIndex, len(item.Pages))
		return nil
	}

	text := item.Pages[pageIndex]

	toCreate := &dto.MessageToCreate{
		MsgSeq: rand.Uint32()%10000000 + 1,
	}

	if eventRaw != nil && eventRaw.EventID != "" {
		toCreate.EventID = eventRaw.EventID
	} else {
		toCreate.EventID = data.ID
	}

	keyboardObj := pa.buildPaginationKeyboard(cacheID, pageIndex, len(item.Pages))

	if pa.EndPoint.Session.Parent.Config.OfficialQQUseMarkdown {
		toCreate.MsgType = 2
		toCreate.Markdown = &dto.Markdown{
			Content: text,
		}
		if keyboardObj != nil {
			toCreate.Keyboard = keyboardObj
		}
	} else {
		toCreate.MsgType = 0
		toCreate.Content = text
	}

	ctx := &MsgContext{
		EndPoint: pa.EndPoint,
		Session:  s,
		Dice:     s.Parent,
	}

	// 根据 chat_type 发送
	switch data.ChatType {
	case 0: // 频道
		msg, err := pa.Api.PostMessage(qctx, data.ChannelID, toCreate)
		if err != nil {
			log.Errorf("official qq 翻页发送频道消息失败：%v", err)
		} else if msg != nil {
			ctx.MessageType = "group"
			pa.EndPoint.Session.OnMessageSend(ctx, &Message{
				Platform:    "QQ",
				MessageType: "group",
				Message:     text,
				GroupID:     data.ChannelID,
				Sender: SenderBase{
					UserID:   pa.EndPoint.UserID,
					Nickname: pa.EndPoint.Nickname,
				},
				RawID: msg.ID,
			}, "")
		}
	case 1: // 群
		msg, err := pa.Api.PostGroupMessage(qctx, data.GroupOpenID, toCreate)
		if err != nil {
			log.Errorf("official qq 翻页发送群聊消息失败：%v", err)
		} else if msg != nil {
			ctx.MessageType = "group"
			groupID := formatDiceIDOfficialQQGroupOpenID(pa.UIN, data.GroupOpenID)
			pa.EndPoint.Session.OnMessageSend(ctx, &Message{
				Platform:    "QQ",
				MessageType: "group",
				Message:     text,
				GroupID:     groupID,
				Sender: SenderBase{
					UserID:   pa.EndPoint.UserID,
					Nickname: pa.EndPoint.Nickname,
				},
				RawID: msg.ID,
			}, "")
		}
	case 2: // C2C
		msg, err := pa.Api.PostC2CMessage(qctx, data.UserOpenID, toCreate)
		if err != nil {
			log.Errorf("official qq 翻页发送私聊消息失败： %v", err)
		} else if msg != nil {
			ctx.MessageType = "private"
			pa.EndPoint.Session.OnMessageSend(ctx, &Message{
				Platform:    "QQ",
				MessageType: "private",
				Message:     text,
				Sender: SenderBase{
					UserID:   pa.EndPoint.UserID,
					Nickname: pa.EndPoint.Nickname,
				},
				RawID: msg.ID,
			}, "")
		}
	default:
		switch data.Scene {
		case "group":
			msg, err := pa.Api.PostGroupMessage(qctx, data.GroupOpenID, toCreate)
			if err != nil {
				log.Errorf("official qq 翻页发送群聊消息失败： %v", err)
			} else if msg != nil {
				ctx.MessageType = "group"
				groupID := formatDiceIDOfficialQQGroupOpenID(pa.UIN, data.GroupOpenID)
				pa.EndPoint.Session.OnMessageSend(ctx, &Message{
					Platform:    "QQ",
					MessageType: "group",
					Message:     text,
					GroupID:     groupID,
					Sender: SenderBase{
						UserID:   pa.EndPoint.UserID,
						Nickname: pa.EndPoint.Nickname,
					},
					RawID: msg.ID,
				}, "")
			}
		case "c2c":
			msg, err := pa.Api.PostC2CMessage(qctx, data.UserOpenID, toCreate)
			if err != nil {
				log.Errorf("official qq 翻页发送私聊消息失败：%v", err)
			} else if msg != nil {
				ctx.MessageType = "private"
				pa.EndPoint.Session.OnMessageSend(ctx, &Message{
					Platform:    "QQ",
					MessageType: "private",
					Message:     text,
					Sender: SenderBase{
						UserID:   pa.EndPoint.UserID,
						Nickname: pa.EndPoint.Nickname,
					},
					RawID: msg.ID,
				}, "")
			}
		default:
			if data.ChannelID != "" {
				msg, err := pa.Api.PostMessage(qctx, data.ChannelID, toCreate)
				if err != nil {
					log.Errorf("official qq 翻页发送频道消息失败： %v", err)
				} else if msg != nil {
					ctx.MessageType = "group"
					pa.EndPoint.Session.OnMessageSend(ctx, &Message{
						Platform:    "QQ",
						MessageType: "group",
						Message:     text,
						GroupID:     data.ChannelID,
						Sender: SenderBase{
							UserID:   pa.EndPoint.UserID,
							Nickname: pa.EndPoint.Nickname,
						},
						RawID: msg.ID,
					}, "")
				}
			}
		}
	}

	return nil
}

func (pa *PlatformAdapterOfficialQQ) ChannelAtMessageReceive(event *dto.WSPayload, data *dto.WSATMessageData) error {
	s := pa.EndPoint.Session
	log := s.Parent.Logger
	log.Debugf("official qq: 收到文字频道消息：%v, %v", event, data)

	s.Execute(pa.EndPoint, pa.channelMsgToStdMsg(data), false)
	return nil
}

func (pa *PlatformAdapterOfficialQQ) channelMsgToStdMsg(msgQQ *dto.WSATMessageData) *Message {
	msg := new(Message)
	timestamp, _ := msgQQ.Timestamp.Time()
	msg.Time = timestamp.Unix()
	msg.MessageType = "group"
	msg.Message = msgQQ.Content
	msg.RawID = msgQQ.ID
	msg.Platform = "OpenQQCH"
	msg.GuildID = formatDiceIDOfficialQQChGuild(msgQQ.GuildID)
	channelID := formatDiceIDOfficialQQChannel(msgQQ.GuildID, msgQQ.ChannelID)
	msg.GroupID = channelID
	msg.ChannelID = channelID
	if msgQQ.Author != nil {
		msg.Sender.Nickname = msgQQ.Author.Username
		msg.Sender.UserID = formatDiceIDOfficialQQCh(msgQQ.Author.ID)
	}
	return msg
}

func (pa *PlatformAdapterOfficialQQ) GuildDirectMessageReceive(event *dto.WSPayload, data *dto.WSDirectMessageData) error {
	s := pa.EndPoint.Session
	log := s.Parent.Logger
	log.Debugf("official qq: 收到频道私信消息：%v, %v", event, data)

	s.Execute(pa.EndPoint, pa.guildDirectMsgToStdMsg(data), false)
	return nil
}

func (pa *PlatformAdapterOfficialQQ) guildDirectMsgToStdMsg(msgQQ *dto.WSDirectMessageData) *Message {
	msg := new(Message)
	timestamp, _ := msgQQ.Timestamp.Time()
	msg.Time = timestamp.Unix()
	msg.MessageType = "private"
	msg.Message = msgQQ.Content
	msg.RawID = msgQQ.ID
	msg.Platform = "OpenQQCH"
	// 频道私信需要私信频道的 guild_id 和channel_id
	channelID := formatDiceIDOfficialQQChannel(msgQQ.GuildID, msgQQ.ChannelID)
	msg.GroupID = channelID
	msg.ChannelID = channelID
	if msgQQ.Author != nil {
		msg.Sender.Nickname = msgQQ.Author.Username
		msg.Sender.UserID = formatDiceIDOfficialQQCh(msgQQ.Author.ID)
	}
	return msg
}

func (pa *PlatformAdapterOfficialQQ) GroupAtMessageReceive(event *dto.WSPayload, data *dto.WSGroupATMessageData) error {
	s := pa.EndPoint.Session
	log := s.Parent.Logger
	log.Debugf("official qq: 收到群聊消息：%v, %v", event, data)

	s.Execute(pa.EndPoint, pa.groupMsgToStdMsg(event, data), false)
	return nil
}

func (pa *PlatformAdapterOfficialQQ) groupMsgToStdMsg(event *dto.WSPayload, msgQQ *dto.WSGroupATMessageData) *Message {
	msg := new(Message)
	timestamp, _ := msgQQ.Timestamp.Time()
	msg.Time = timestamp.Unix()
	msg.MessageType = "group"
	msg.Message = msgQQ.Content
	msg.RawID = msgQQ.ID
	msg.Platform = "OpenQQ"
	msg.GroupID = formatDiceIDOfficialQQGroupOpenID(pa.UIN, msgQQ.GroupOpenID)
	if msgQQ.Author != nil {
		if msgQQ.Author.Username != "" {
			msg.Sender.Nickname = msgQQ.Author.Username
		} else if len(msgQQ.Author.MemberOpenID) >= 4 {
			msg.Sender.Nickname = "用户" + msgQQ.Author.MemberOpenID[len(msgQQ.Author.MemberOpenID)-4:]
		} else {
			msg.Sender.Nickname = "用户"
		}
		msg.Sender.UserID = formatDiceIDOfficialQQMemberOpenID(pa.UIN, msgQQ.GroupOpenID, msgQQ.Author.MemberOpenID)
		msg.Sender.GroupRole = msgQQ.Author.MemberRole
	}

	botSelfOpenID := pa.parseBotSelfOpenID(event)

	if botSelfOpenID == "" {
		botSelfOpenID = pa.botID
	}

	m := officialQQAtRegex.FindStringSubmatch(msgQQ.Content)
	if len(m) == 2 {
		msg.TmpUID = "OpenQQ:" + m[1]
	} else if botSelfOpenID != "" {
		msg.TmpUID = "OpenQQ:" + botSelfOpenID
	}

	appendAttachmentsToMessage(msg, msgQQ.Attachments)

	return msg
}

// GroupMessageReceive 处理群聊普通消息
func (pa *PlatformAdapterOfficialQQ) GroupMessageReceive(event *dto.WSPayload, data *dto.WSGroupMessageData) error {
	s := pa.EndPoint.Session
	log := s.Parent.Logger
	log.Debugf("official qq: 收到群聊普通消息：%v, %v", event, data)

	msg := pa.groupNormalMsgToStdMsg(event, data)
	s.Execute(pa.EndPoint, msg, false)
	return nil
}

// groupNormalMsgToStdMsg 将群聊普通消息转换为标准消息
func (pa *PlatformAdapterOfficialQQ) groupNormalMsgToStdMsg(event *dto.WSPayload, msgQQ *dto.WSGroupMessageData) *Message {
	msg := new(Message)
	timestamp, _ := msgQQ.Timestamp.Time()
	msg.Time = timestamp.Unix()
	msg.MessageType = "group"
	msg.Message = msgQQ.Content
	msg.RawID = msgQQ.ID
	msg.Platform = "OpenQQ"
	msg.GroupID = formatDiceIDOfficialQQGroupOpenID(pa.UIN, msgQQ.GroupOpenID)
	if msgQQ.Author != nil {
		if msgQQ.Author.Username != "" {
			msg.Sender.Nickname = msgQQ.Author.Username
		} else if len(msgQQ.Author.MemberOpenID) >= 4 {
			msg.Sender.Nickname = "用户" + msgQQ.Author.MemberOpenID[len(msgQQ.Author.MemberOpenID)-4:]
		} else {
			msg.Sender.Nickname = "用户"
		}
		msg.Sender.UserID = formatDiceIDOfficialQQMemberOpenID(pa.UIN, msgQQ.GroupOpenID, msgQQ.Author.MemberOpenID)
		msg.Sender.GroupRole = msgQQ.Author.MemberRole
	}

	botSelfOpenID := pa.parseBotSelfOpenID(event)

	if botSelfOpenID == "" {
		botSelfOpenID = pa.botID
	}

	for _, match := range officialQQAtRegex.FindAllStringSubmatch(msgQQ.Content, -1) {
		if len(match) == 2 && match[1] == botSelfOpenID {
			msg.TmpUID = "OpenQQ:" + match[1]
			break
		}
	}

	appendAttachmentsToMessage(msg, msgQQ.Attachments)

	return msg
}

// C2CMessageReceiveFromEvent 处理单聊消息（使用botgo dto 内置类型）
func (pa *PlatformAdapterOfficialQQ) C2CMessageReceiveFromEvent(payload *dto.WSPayload, data *dto.WSC2CMessageData) error {
	s := pa.EndPoint.Session
	log := s.Parent.Logger
	log.Debugf("official qq: 收到单聊消息: %v", data)

	msg := pa.c2cMsgToStdMsg(data)
	s.Execute(pa.EndPoint, msg, false)
	return nil
}

// c2cMsgToStdMsg 将单聊消息转换为标准消息
func (pa *PlatformAdapterOfficialQQ) c2cMsgToStdMsg(msgQQ *dto.WSC2CMessageData) *Message {
	msg := new(Message)
	timestamp, _ := msgQQ.Timestamp.Time()
	msg.Time = timestamp.Unix()
	msg.MessageType = "private"
	msg.Message = msgQQ.Content
	msg.RawID = msgQQ.ID
	msg.Platform = "OpenQQ"
	if msgQQ.Author != nil {
		userOpenID := msgQQ.Author.UserOpenID
		if msgQQ.Author.Username != "" {
			msg.Sender.Nickname = msgQQ.Author.Username
		} else if len(userOpenID) >= 4 {
			msg.Sender.Nickname = "用户" + userOpenID[len(userOpenID)-4:]
		} else {
			msg.Sender.Nickname = "用户"
		}
		msg.Sender.UserID = formatDiceIDOfficialQQUserOpenID(pa.UIN, userOpenID)
	}
	appendAttachmentsToMessage(msg, msgQQ.Attachments)
	return msg
}

// GroupMemberAddReceive 处理群成员增加事件
func (pa *PlatformAdapterOfficialQQ) GroupMemberAddReceive(event *dto.WSPayload, data *dto.WSGroupMemberAddData) error {
	s := pa.EndPoint.Session
	log := s.Parent.Logger
	log.Debugf("official qq: 收到群成员增加事件：%v, %v", event, data)

	groupID := formatDiceIDOfficialQQGroupOpenID(pa.UIN, data.GroupOpenID)
	userID := formatDiceIDOfficialQQMemberOpenID(pa.UIN, data.GroupOpenID, data.MemberOpenID)

	// 如果是机器人自己加入群聊
	if userID == pa.EndPoint.UserID || data.MemberOpenID == "" || data.MemberOpenID == "BOT" {
		ctx := &MsgContext{EndPoint: pa.EndPoint, Session: s, Dice: s.Parent}
		ctx.Group = SetBotOnAtGroup(ctx, groupID)
		ctx.Group.DiceIDExistsMap.Store(ctx.EndPoint.UserID, true)
		ctx.Group.EnteredTime = time.Now().Unix()
		ctx.Group.MarkDirty(ctx.Dice)

		if event != nil && event.EventID != "" {
			VarSetValueStr(ctx, "$tEventID", event.EventID)
		}

		log.Infof("official qq: 机器人加入群 %s", groupID)

		// 发送入群致辞
		go func() {
			time.Sleep(2 * time.Second)
			ctx.Player = &GroupPlayerInfo{}
			text := DiceFormatTmpl(ctx, "核心:骰子进群")
			for _, i := range ctx.SplitText(text) {
				pa.SendToGroup(ctx, groupID, strings.TrimSpace(i), "")
			}
		}()
	} else {
		// 普通成员进群
		ctx := &MsgContext{EndPoint: pa.EndPoint, Session: s, Dice: s.Parent}
		if event != nil && event.EventID != "" {
			VarSetValueStr(ctx, "$tEventID", event.EventID)
		}

		msg := &Message{
			Time:        data.Timestamp,
			MessageType: "group",
			Platform:    "QQ",
			GroupID:     groupID,
			Sender: SenderBase{
				UserID:   userID,
				Nickname: "用户",
			},
		}
		if len(data.MemberOpenID) >= 4 {
			msg.Sender.Nickname = "用户" + data.MemberOpenID[len(data.MemberOpenID)-4:]
		}

		pa.EndPoint.Session.OnGroupMemberJoined(ctx, msg)
	}

	return nil
}

// GroupMemberRemoveReceive 处理群成员减少事件
func (pa *PlatformAdapterOfficialQQ) GroupMemberRemoveReceive(event *dto.WSPayload, data *dto.WSGroupMemberRemoveData) error {
	s := pa.EndPoint.Session
	log := s.Parent.Logger
	log.Debugf("official qq: 收到群成员减少事件：%v, %v", event, data)

	groupID := formatDiceIDOfficialQQGroupOpenID(pa.UIN, data.GroupOpenID)
	userID := formatDiceIDOfficialQQMemberOpenID(pa.UIN, data.GroupOpenID, data.MemberOpenID)

	// 如果是机器人自己被移出群
	if userID == pa.EndPoint.UserID || data.MemberOpenID == "" || data.MemberOpenID == "BOT" {
		groupName := s.Parent.Parent.TryGetGroupName(groupID)

		txt := fmt.Sprintf("official qq: 离开群组: <%s>(%s)", groupName, groupID)
		log.Info(txt)

		group, exists := s.ServiceAtNew.Load(groupID)
		if exists {
			group.DiceIDExistsMap.Delete(pa.EndPoint.UserID)
			group.MarkDirty(s.Parent)
		}
	}

	return nil
}

// GroupAddRobotReceive 处理机器人加入群聊事件
func (pa *PlatformAdapterOfficialQQ) GroupAddRobotReceive(event *dto.WSPayload, data *dto.WSGroupRobotEventData) error {
	s := pa.EndPoint.Session
	log := s.Parent.Logger
	log.Debugf("official qq: 收到机器人加入群聊事件：%v, %v", event, data)

	// 转化为WSGroupMemberAddData
	memberData := &dto.WSGroupMemberAddData{
		GroupOpenID:    data.GroupOpenID,
		MemberOpenID:   "BOT",
		OpMemberOpenID: data.OpMemberOpenID,
		Timestamp:      data.Timestamp,
	}
	return pa.GroupMemberAddReceive(event, memberData)
}

// GroupDelRobotReceive 处理机器人退出群聊事件
func (pa *PlatformAdapterOfficialQQ) GroupDelRobotReceive(event *dto.WSPayload, data *dto.WSGroupRobotEventData) error {
	s := pa.EndPoint.Session
	log := s.Parent.Logger
	log.Debugf("official qq: 收到机器人退出群聊事件：%v, %v", event, data)

	// 转化为WSGroupMemberRemoveData
	memberData := &dto.WSGroupMemberRemoveData{
		GroupOpenID:    data.GroupOpenID,
		MemberOpenID:   "BOT",
		OpMemberOpenID: data.OpMemberOpenID,
		Timestamp:      data.Timestamp,
	}
	return pa.GroupMemberRemoveReceive(event, memberData)
}

// C2CFriendReceive 处理好友关系变动事件
func (pa *PlatformAdapterOfficialQQ) C2CFriendReceive(event *dto.WSPayload, data *dto.WSC2CFriendData) error {
	s := pa.EndPoint.Session
	log := s.Parent.Logger
	log.Debugf("official qq: 收到好友事件: %s, %v, %v", event.Type, event, data)

	switch event.Type {
	case dto.EventC2CFriendAdd:
		userID := formatDiceIDOfficialQQUserOpenID(pa.UIN, data.OpenID)

		ctx := &MsgContext{EndPoint: pa.EndPoint, Session: s, Dice: s.Parent}
		if event != nil && event.EventID != "" {
			VarSetValueStr(ctx, "$tEventID", event.EventID)
		}

		msg := &Message{
			Time:        int64(data.Timestamp),
			MessageType: "private",
			Platform:    "QQ",
			Message:     "",
			Sender: SenderBase{
				UserID:   userID,
				Nickname: "用户",
			},
		}
		if len(data.OpenID) >= 4 {
			msg.Sender.Nickname = "用户" + data.OpenID[len(data.OpenID)-4:]
		}

		ctx.Group, ctx.Player = GetPlayerInfoBySender(ctx, msg)
		welcomeStr := DiceFormatTmpl(ctx, "核心:骰子成为好友")
		log.Infof("official qq: 与%s 成为好友，发送好友致辞 %s", userID, welcomeStr)

		go func() {
			time.Sleep(2 * time.Second)
			for _, i := range ctx.SplitText(welcomeStr) {
				pa.SendToPerson(ctx, userID, strings.TrimSpace(i), "")
			}
			if groupInfo, ok := ctx.Session.ServiceAtNew.Load(msg.GroupID); ok {
				groupInfo.TriggerExtHook(ctx.Dice, func(ext *ExtInfo) func() {
					if ext.OnBecomeFriend == nil {
						return nil
					}
					return func() { ext.OnBecomeFriend(ctx, msg) }
				})
			}
		}()
	case dto.EventC2CFriendDel:
		userID := formatDiceIDOfficialQQUserOpenID(pa.UIN, data.OpenID)
		log.Infof("official qq: 与 %s 解除好友关系", userID)
	default:
		// 忽略其他事件
	}

	return nil
}
func (pa *PlatformAdapterOfficialQQ) initPaginationCache() {
	pa.paginationMu.Lock()
	defer pa.paginationMu.Unlock()
	if pa.paginationCache == nil {
		pa.paginationCache = make(map[string]*PaginationItem)
	}
}

func (pa *PlatformAdapterOfficialQQ) addToPaginationCache(id string, pages []string) {
	pa.initPaginationCache()
	pa.paginationMu.Lock()
	defer pa.paginationMu.Unlock()

	now := time.Now()
	if len(pa.paginationCache) > 1000 {
		for k, v := range pa.paginationCache {
			if now.Sub(v.CreatedAt) > 1*time.Hour || len(pa.paginationCache) > 1000 {
				delete(pa.paginationCache, k)
			}
		}
	}

	pa.paginationCache[id] = &PaginationItem{
		Pages:     pages,
		CreatedAt: now,
	}
}

func generateCacheID() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func (pa *PlatformAdapterOfficialQQ) buildPaginationKeyboard(cacheID string, pageIndex int, totalPages int) *keyboard.MessageKeyboard {
	if totalPages <= 1 {
		return nil
	}

	var buttons []*keyboard.Button

	// 上一页
	if pageIndex > 0 {
		buttons = append(buttons, &keyboard.Button{
			ID: fmt.Sprintf("prev_%s_%d", cacheID, pageIndex-1),
			RenderData: &keyboard.RenderData{
				Label:        fmt.Sprintf("上一页(%d/%d)", pageIndex+1, totalPages),
				VisitedLabel: "跳转中……",
				Style:        0, // 灰色线框
			},
			Action: &keyboard.Action{
				Type: keyboard.ActionTypeCallback, // Callback
				Data: fmt.Sprintf("pg:%s:%d", cacheID, pageIndex-1),
				Permission: &keyboard.Permission{
					Type: keyboard.PermissionTypAll, // 所有人可操作
				},
			},
		})
	}

	// 下一页
	if pageIndex < totalPages-1 {
		buttons = append(buttons, &keyboard.Button{
			ID: fmt.Sprintf("next_%s_%d", cacheID, pageIndex+1),
			RenderData: &keyboard.RenderData{
				Label:        fmt.Sprintf("下一页(%d/%d)", pageIndex+1, totalPages),
				VisitedLabel: "跳转中……",
				Style:        1, // 蓝色线框
			},
			Action: &keyboard.Action{
				Type: keyboard.ActionTypeCallback, // Callback
				Data: fmt.Sprintf("pg:%s:%d", cacheID, pageIndex+1),
				Permission: &keyboard.Permission{
					Type: keyboard.PermissionTypAll, // 所有人可操作
				},
			},
		})
	}

	if len(buttons) == 0 {
		return nil
	}

	return &keyboard.MessageKeyboard{
		Content: &keyboard.CustomKeyboard{
			Rows: []*keyboard.Row{
				{
					Buttons: buttons,
				},
			},
		},
	}
}

func (pa *PlatformAdapterOfficialQQ) shutdownWebhookServer() {
	if pa.webhookServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if err := pa.webhookServer.Shutdown(ctx); err != nil {
			pa.EndPoint.Session.Parent.Logger.Warn("official qq webhook server graceful shutdown failed, forcing close: ", err)
			_ = pa.webhookServer.Close()
		}
		cancel()
		pa.webhookServer = nil
	}
}

func (pa *PlatformAdapterOfficialQQ) DoRelogin() bool {
	if pa.CancelFunc != nil {
		pa.CancelFunc()
	}
	pa.EndPoint.Session.Parent.Logger.Infof("正在启用 official qq 服务")
	pa.EndPoint.State = 0
	pa.EndPoint.Enable = false
	pa.Api = nil
	pa.Ctx = nil
	pa.CancelFunc = nil
	pa.tokenSource = nil
	pa.clearQrLoginState()
	pa.shutdownWebhookServer()
	return pa.Serve() == 0
}

func (pa *PlatformAdapterOfficialQQ) SetEnable(enable bool) {
	d := pa.EndPoint.Session.Parent
	ep := pa.EndPoint
	if enable {
		if pa.Ctx == nil {
			ep.Enable = false
			pa.DiceServing = false
			ep.State = 2
			ServerOfficialQQ(d, ep)
		} else {
			ep.Enable = true
			ep.State = 1
		}
	} else {
		ep.State = 0
		ep.Enable = false
		if pa.CancelFunc != nil {
			pa.CancelFunc()
		}
		pa.shutdownWebhookServer()
		pa.CancelFunc = nil
		pa.Ctx = nil
		pa.tokenSource = nil
		pa.clearQrLoginState()
	}
	d.LastUpdatedTime = time.Now().Unix()
}

func (pa *PlatformAdapterOfficialQQ) SendSegmentToGroup(ctx *MsgContext, groupID string, msg []message.IMessageElement, flag string) {
}

func (pa *PlatformAdapterOfficialQQ) SendSegmentToPerson(ctx *MsgContext, userID string, msg []message.IMessageElement, flag string) {
}

func (pa *PlatformAdapterOfficialQQ) SendToPerson(ctx *MsgContext, uid string, text string, flag string) {
	userID, idType := pa.mustExtractID(uid)

	maxLen := 900
	totalLen := len(text)
	if totalLen > 900*5 {
		maxLen = totalLen/5 + 50
		if maxLen > 2800 {
			maxLen = 2800
		}
	}
	textList := utils.SplitLongText(text, maxLen, utils.DefaultSplitPaginationHint)
	if !pa.EndPoint.Session.Parent.Config.OfficialQQUseMarkdown {
		if len(textList) > 5 {
			textList = textList[:5]
		}
	}

	var activeCtx *MsgContext = nil
	var activeRowID = ""
	if ctx != nil && ctx.MessageType == "private" && ctx.Player != nil && ctx.Player.UserID == uid {
		activeRowID, _ = VarGetValueStr(ctx, "$tMsgID")
		if activeRowID == "" {
			activeRowID, _ = VarGetValueStr(ctx, "$tEventID")
		}
		if activeRowID != "" {
			activeCtx = ctx
		}
	}

	if pa.EndPoint.Session.Parent.Config.OfficialQQUseMarkdown && len(textList) > 1 {
		cacheID := generateCacheID()
		pa.addToPaginationCache(cacheID, textList)

		keyboardObj := pa.buildPaginationKeyboard(cacheID, 0, len(textList))

		if idType == OpenQQUserOpenid {
			msg, err := pa.sendC2CMsgRaw(activeCtx, activeRowID, userID, textList[0], keyboardObj)
			if err == nil && msg != nil {
				pa.EndPoint.Session.OnMessageSend(ctx, &Message{
					Platform:    "QQ",
					MessageType: "private",
					Message:     textList[0],
					Sender: SenderBase{
						UserID:   pa.EndPoint.UserID,
						Nickname: pa.EndPoint.Nickname,
					},
					RawID: msg.ID,
				}, flag)
			}
			return
		}

		if idType != OpenQQCHUser {
			// pa.EndPoint.Session.Parent.Logger.Error("official qq 发送私聊消息失败：不支持该功能")
			return
		}

		if ctx == nil || ctx.Group == nil {
			pa.EndPoint.Session.Parent.Logger.Error("official qq 发送频道私信消息失败：无有效的上下文信息")
			return
		}

		channelID, guildID, _ := pa.mustExtractTwoID(ctx.Group.ChannelID)
		rowID, ok := VarGetValueStr(ctx, "$tMsgID")
		if !ok {
			rowID, ok = VarGetValueStr(ctx, "$tEventID")
		}
		if !ok || ctx.MessageType == "group" {
			g, c, err := pa.createQQGuildDirectChannel(ctx, guildID, userID)
			if err != nil {
				pa.EndPoint.Session.Parent.Logger.Error("official qq 发送频道私信消息失败：", err.Error())
				return
			}
			guildID = g
			channelID = c
		}
		msg, err := pa.sendQQGuildDirectMsgRaw(ctx, rowID, guildID, channelID, textList[0], keyboardObj)
		if err == nil && msg != nil {
			pa.EndPoint.Session.OnMessageSend(ctx, &Message{
				Platform:    "QQ",
				MessageType: "private",
				Message:     textList[0],
				Sender: SenderBase{
					UserID:   pa.EndPoint.UserID,
					Nickname: pa.EndPoint.Nickname,
				},
				RawID: msg.ID,
			}, flag)
		}
		return
	}

	for _, t := range textList {
		if idType == OpenQQUserOpenid {
			msg, err := pa.sendC2CMsgRaw(activeCtx, activeRowID, userID, t, nil)
			if err == nil && msg != nil {
				pa.EndPoint.Session.OnMessageSend(ctx, &Message{
					Platform:    "QQ",
					MessageType: "private",
					Message:     t,
					Sender: SenderBase{
						UserID:   pa.EndPoint.UserID,
						Nickname: pa.EndPoint.Nickname,
					},
					RawID: msg.ID,
				}, flag)
			}
			continue
		}

		if idType != OpenQQCHUser {
			// pa.EndPoint.Session.Parent.Logger.Error("official qq 发送私聊消息失败：不支持该功能")
			return
		}

		if ctx == nil || ctx.Group == nil {
			pa.EndPoint.Session.Parent.Logger.Error("official qq 发送频道私信消息失败：无有效的上下文信息")
			return
		}

		channelID, guildID, _ := pa.mustExtractTwoID(ctx.Group.ChannelID)
		rowID, ok := VarGetValueStr(ctx, "$tMsgID")
		if !ok {
			rowID, ok = VarGetValueStr(ctx, "$tEventID")
		}
		if !ok || ctx.MessageType == "group" {
			// 需要主动发起私信
			g, c, err := pa.createQQGuildDirectChannel(ctx, guildID, userID)
			if err != nil {
				pa.EndPoint.Session.Parent.Logger.Error("official qq 发送频道私信消息失败：", err.Error())
				return
			}
			guildID = g
			channelID = c
		}
		msg, err := pa.sendQQGuildDirectMsgRaw(ctx, rowID, guildID, channelID, t, nil)
		if err == nil && msg != nil {
			pa.EndPoint.Session.OnMessageSend(ctx, &Message{
				Platform:    "QQ",
				MessageType: "private",
				Message:     t,
				Sender: SenderBase{
					UserID:   pa.EndPoint.UserID,
					Nickname: pa.EndPoint.Nickname,
				},
				RawID: msg.ID,
			}, flag)
		}
	}
}

func (pa *PlatformAdapterOfficialQQ) createQQGuildDirectChannel( /* ctx */ _ *MsgContext, guildID, userID string) (string, string, error) {
	if guildID == "" || userID == "" {
		err := errors.New("创建私信频道的参数不全")
		pa.EndPoint.Session.Parent.Logger.Error("official qq 创建私信频道失败：" + err.Error())
		return "", "", err
	}
	qctx := context.Background()
	toCreate := &dto.DirectMessageToCreate{
		SourceGuildID: guildID,
		RecipientID:   userID,
	}
	info, err := pa.Api.CreateDirectMessage(qctx, toCreate)
	if err != nil {
		pa.EndPoint.Session.Parent.Logger.Error("official qq 创建私信频道失败：" + err.Error())
		return "", "", err
	}
	return info.GuildID, info.ChannelID, nil
}

func (pa *PlatformAdapterOfficialQQ) initMessageToCreate(ctx *MsgContext, rowMsgID string) *dto.MessageToCreate {
	toCreate := &dto.MessageToCreate{
		MsgSeq: rand.Uint32()%10000000 + 1,
	}
	if ctx != nil {
		if eventID, ok := VarGetValueStr(ctx, "$tEventID"); ok && eventID != "" {
			toCreate.EventID = eventID
		} else {
			toCreate.MsgID = rowMsgID
		}
	} else {
		toCreate.MsgID = rowMsgID
	}
	return toCreate
}

func (pa *PlatformAdapterOfficialQQ) finalizeMessageToCreate(toCreate *dto.MessageToCreate, content string, keyboardObj *keyboard.MessageKeyboard, isFinal bool) {
	if toCreate.Media == nil && content == "" && toCreate.MessageReference == nil {
		return
	}

	useMarkdown := pa.EndPoint.Session.Parent.Config.OfficialQQUseMarkdown
	if useMarkdown && toCreate.MsgType != 7 {
		toCreate.MsgType = 2
		toCreate.Markdown = &dto.Markdown{
			Content: content,
		}
		if isFinal && keyboardObj != nil {
			toCreate.Keyboard = keyboardObj
		}
	} else {
		toCreate.Content = content
		if toCreate.MsgType != 7 {
			toCreate.MsgType = 0
		}
		if isFinal && keyboardObj != nil {
			toCreate.Keyboard = keyboardObj
		}
	}
}

func (pa *PlatformAdapterOfficialQQ) prepareMediaMessage(file *message.FileElement) (string, []byte, error) {
	url := file.URL
	if pa.EndPoint.Session.Parent.Config.OfficialQQFileSendBase64 || isLocalOrNonPublic(url) {
		data, err := getElementBytes(file)
		if err != nil {
			return "", nil, err
		}
		return url, data, nil
	}
	return url, nil, nil
}

func (pa *PlatformAdapterOfficialQQ) sendQQGuildDirectMsgRaw( /* ctx */ _ *MsgContext, rowMsgID string, guildID, channelID string, text string, keyboardObj *keyboard.MessageKeyboard) (*dto.Message, error) {
	qctx := context.Background()
	elems := message.ConvertStringMessage(text)
	var (
		content string
		msgRef  *dto.MessageReference
	)

	for _, elem := range elems {
		switch e := elem.(type) {
		case *message.TextElement:
			content += e.Content
		case *message.ImageElement:
		case *message.ReplyElement:
			msgRef = &dto.MessageReference{
				MessageID:             e.ReplySeq,
				IgnoreGetMessageError: true,
			}
		}
	}

	dMsg := &dto.DirectMessage{
		GuildID:   guildID,
		ChannelID: channelID,
	}
	toCreate := pa.initMessageToCreate(nil, rowMsgID)
	toCreate.MessageReference = msgRef
	pa.finalizeMessageToCreate(toCreate, content, keyboardObj, true)

	res, err := pa.Api.PostDirectMessage(qctx, dMsg, toCreate)
	if err != nil {
		pa.EndPoint.Session.Parent.Logger.Error("official qq 发送频道私信消息失败：", err.Error())
	}
	return res, err
}

func (pa *PlatformAdapterOfficialQQ) uploadC2CMedia(qctx context.Context, userOpenID string, file *message.FileElement, fileType int) (*dto.MediaInfo, error) {
	url, data, err := pa.prepareMediaMessage(file)
	if err != nil {
		return nil, err
	}
	sendURL := url
	if data != nil {
		sendURL = ""
	}
	fMsg := &C2CRichMediaMessage{
		FileType:   fileType,
		URL:        sendURL,
		FileData:   data,
		SrvSendMsg: false,
	}
	media, err := pa.Api.PostC2CMessage(qctx, userOpenID, fMsg)
	if err != nil {
		return nil, err
	}
	return &dto.MediaInfo{
		FileInfo: media.FileInfo,
	}, nil
}

func (pa *PlatformAdapterOfficialQQ) uploadGroupMedia(qctx context.Context, groupID string, file *message.FileElement, fileType int) (*dto.MediaInfo, error) {
	url, data, err := pa.prepareMediaMessage(file)
	if err != nil {
		return nil, err
	}
	sendURL := url
	if data != nil {
		sendURL = ""
	}
	fMsg := &dto.MessageMediaToCreate{
		FileType:   fileType,
		URL:        sendURL,
		FileData:   data,
		SrvSendMsg: false,
	}
	media, err := pa.Api.PostGroupFile(qctx, groupID, fMsg)
	if err != nil {
		return nil, err
	}
	decodedFileInfo, decodeErr := base64.StdEncoding.DecodeString(media.FileInfo)
	if decodeErr != nil {
		decodedFileInfo = []byte(media.FileInfo)
	}
	return &dto.MediaInfo{
		FileInfo: decodedFileInfo,
	}, nil
}

// sendC2CMsgRaw 发送单聊消息（使用msg_id被动回复）
func (pa *PlatformAdapterOfficialQQ) sendC2CMsgRaw(ctx *MsgContext, rowMsgID, userOpenID string, text string, keyboardObj *keyboard.MessageKeyboard) (*dto.Message, error) {
	qctx := context.Background()
	elems := message.ConvertStringMessage(text)
	var (
		content string
		msgRef  *dto.MessageReference
	)

	toCreate := pa.initMessageToCreate(ctx, rowMsgID)

	var lastRes *dto.Message
	var lastErr error

	sendCurrent := func(isFinal bool) {
		pa.finalizeMessageToCreate(toCreate, content, keyboardObj, isFinal)
		if toCreate.Media == nil && toCreate.Content == "" && toCreate.Markdown == nil && toCreate.MessageReference == nil {
			return
		}
		res, err := pa.Api.PostC2CMessage(qctx, userOpenID, toCreate)
		if err != nil {
			pa.EndPoint.Session.Parent.Logger.Error("official qq 发送单聊消息失败：" + err.Error())
			lastErr = err
		} else {
			lastRes = res
		}
	}

	for _, elem := range elems {
		switch e := elem.(type) {
		case *message.TextElement:
			content += textLinkStrip(e.Content)
		case *message.ReplyElement:
			msgRef = &dto.MessageReference{
				MessageID:             e.ReplySeq,
				IgnoreGetMessageError: true,
			}
			toCreate.MessageReference = msgRef
		case *message.ImageElement:
			media, err := pa.uploadC2CMedia(qctx, userOpenID, e.File, 1)
			if err != nil {
				pa.EndPoint.Session.Parent.Logger.Error("official qq 发送单聊消息时，准备图片信息失败：" + err.Error())
				continue
			}

			if toCreate.Media != nil {
				sendCurrent(false)
				content = ""
				toCreate = pa.initMessageToCreate(ctx, rowMsgID)
				toCreate.MessageReference = msgRef
			}

			toCreate.MsgType = 7
			toCreate.Media = media
		case *message.RecordElement:
			media, err := pa.uploadC2CMedia(qctx, userOpenID, e.File, 3)
			if err != nil {
				pa.EndPoint.Session.Parent.Logger.Error("official qq 发送单聊消息时，准备语音信息失败：" + err.Error())
				continue
			}

			if toCreate.Media != nil {
				sendCurrent(false)
				content = ""
				toCreate = pa.initMessageToCreate(ctx, rowMsgID)
				toCreate.MessageReference = msgRef
			}

			toCreate.MsgType = 7
			toCreate.Media = media
		}
	}

	sendCurrent(true)
	return lastRes, lastErr
}

func (pa *PlatformAdapterOfficialQQ) SendToGroup(ctx *MsgContext, uid string, text string, flag string) {
	groupId, idType := pa.mustExtractID(uid)

	maxLen := 900
	totalLen := len(text)
	if totalLen > 900*5 {
		maxLen = totalLen/5 + 50
		if maxLen > 2800 {
			maxLen = 2800
		}
	}
	textList := utils.SplitLongText(text, maxLen, utils.DefaultSplitPaginationHint)
	if !pa.EndPoint.Session.Parent.Config.OfficialQQUseMarkdown {
		if len(textList) > 5 {
			textList = textList[:5]
		}
	}

	var activeCtx *MsgContext = nil
	var activeRowID = ""
	if ctx != nil && ctx.Group != nil && ctx.Group.GroupID == uid {
		activeRowID, _ = VarGetValueStr(ctx, "$tMsgID")
		if activeRowID == "" {
			activeRowID, _ = VarGetValueStr(ctx, "$tEventID")
		}
		if activeRowID != "" {
			activeCtx = ctx
		}
	}

	if pa.EndPoint.Session.Parent.Config.OfficialQQUseMarkdown && len(textList) > 1 {
		cacheID := generateCacheID()
		pa.addToPaginationCache(cacheID, textList)

		keyboardObj := pa.buildPaginationKeyboard(cacheID, 0, len(textList))

		switch idType {
		case OpenQQGroupOpenid:
			msg, err := pa.sendQQGroupMsgRaw(activeCtx, activeRowID, groupId, textList[0], keyboardObj)
			if err == nil && msg != nil {
				pa.EndPoint.Session.OnMessageSend(ctx, &Message{
					Platform:    "QQ",
					MessageType: "group",
					Message:     textList[0],
					GroupID:     uid,
					Sender: SenderBase{
						UserID:   pa.EndPoint.UserID,
						Nickname: pa.EndPoint.Nickname,
					},
					RawID: msg.ID,
				}, flag)
			}
		case OpenQQCHChannel:
			msg, err := pa.sendQQChannelMsgRaw(activeCtx, activeRowID, groupId, textList[0], keyboardObj)
			if err == nil && msg != nil {
				pa.EndPoint.Session.OnMessageSend(ctx, &Message{
					Platform:    "QQ",
					MessageType: "group",
					Message:     textList[0],
					GroupID:     uid,
					Sender: SenderBase{
						UserID:   pa.EndPoint.UserID,
						Nickname: pa.EndPoint.Nickname,
					},
					RawID: msg.ID,
				}, flag)
			}
		default:
			pa.EndPoint.Session.Parent.Logger.Errorf("official qq 发送群聊消息失败：错误的群聊id[%s]类型-%d", uid, idType)
		}
		return
	}

	for _, t := range textList {
		switch idType {
		case OpenQQGroupOpenid:
			msg, err := pa.sendQQGroupMsgRaw(activeCtx, activeRowID, groupId, t, nil)
			if err == nil && msg != nil {
				pa.EndPoint.Session.OnMessageSend(ctx, &Message{
					Platform:    "QQ",
					MessageType: "group",
					Message:     t,
					GroupID:     uid,
					Sender: SenderBase{
						UserID:   pa.EndPoint.UserID,
						Nickname: pa.EndPoint.Nickname,
					},
					RawID: msg.ID,
				}, flag)
			}
		case OpenQQCHChannel:
			msg, err := pa.sendQQChannelMsgRaw(activeCtx, activeRowID, groupId, t, nil)
			if err == nil && msg != nil {
				pa.EndPoint.Session.OnMessageSend(ctx, &Message{
					Platform:    "QQ",
					MessageType: "group",
					Message:     t,
					GroupID:     uid,
					Sender: SenderBase{
						UserID:   pa.EndPoint.UserID,
						Nickname: pa.EndPoint.Nickname,
					},
					RawID: msg.ID,
				}, flag)
			}
		default:
			pa.EndPoint.Session.Parent.Logger.Errorf("official qq 发送群聊消息失败：错误的群聊id[%s]类型-%d", uid, idType)
			return
		}
	}
}

func (pa *PlatformAdapterOfficialQQ) sendQQGroupMsgRaw(ctx *MsgContext, rowMsgID, groupID string, text string, keyboardObj *keyboard.MessageKeyboard) (*dto.Message, error) {
	qctx := context.Background()
	elems := message.ConvertStringMessage(text)
	var (
		content string
		msgRef  *dto.MessageReference
	)

	toCreate := pa.initMessageToCreate(ctx, rowMsgID)

	var lastRes *dto.Message
	var lastErr error

	sendCurrent := func(isFinal bool) {
		pa.finalizeMessageToCreate(toCreate, content, keyboardObj, isFinal)
		if toCreate.Media == nil && toCreate.Content == "" && toCreate.Markdown == nil && toCreate.MessageReference == nil {
			return
		}
		res, err := pa.Api.PostGroupMessage(qctx, groupID, toCreate)
		if err != nil {
			pa.EndPoint.Session.Parent.Logger.Error("official qq 发送群聊消息失败：" + err.Error())
			lastErr = err
		} else {
			lastRes = res
		}
	}

	for _, element := range elems {
		switch elem := element.(type) {
		case *message.TextElement:
			content += textLinkStrip(elem.Content)
		case *message.ReplyElement:
			msgRef = &dto.MessageReference{
				MessageID:             elem.ReplySeq,
				IgnoreGetMessageError: true,
			}
			toCreate.MessageReference = msgRef
		case *message.AtElement:
			target := strings.TrimPrefix(elem.Target, "OpenQQ:")
			target = strings.TrimPrefix(target, "QQ:")
			if target != "all" {
				content += fmt.Sprintf("<qqbot-at-user id=\"%s\" />", target)
			}
		case *message.ImageElement:
			media, err := pa.uploadGroupMedia(qctx, groupID, elem.File, 1)
			if err != nil {
				pa.EndPoint.Session.Parent.Logger.Error("official qq 发送群聊消息时，准备图片信息失败：" + err.Error())
				continue
			}

			if toCreate.Media != nil {
				sendCurrent(false)
				content = ""
				toCreate = pa.initMessageToCreate(ctx, rowMsgID)
				toCreate.MessageReference = msgRef
			}

			toCreate.MsgType = 7
			toCreate.Media = media
		case *message.RecordElement:
			media, err := pa.uploadGroupMedia(qctx, groupID, elem.File, 3)
			if err != nil {
				pa.EndPoint.Session.Parent.Logger.Error("official qq 发送群聊消息时，准备语音信息失败：" + err.Error())
				continue
			}

			if toCreate.Media != nil {
				sendCurrent(false)
				content = ""
				toCreate = pa.initMessageToCreate(ctx, rowMsgID)
				toCreate.MessageReference = msgRef
			}

			toCreate.MsgType = 7
			toCreate.Media = media
		}
	}

	sendCurrent(true)
	return lastRes, lastErr
}

func (pa *PlatformAdapterOfficialQQ) sendQQChannelMsgRaw( /* ctx */ _ *MsgContext, rowMsgID, channelID string, text string, keyboardObj *keyboard.MessageKeyboard) (*dto.Message, error) {
	qctx := context.Background()
	elems := message.ConvertStringMessage(text)
	var (
		content string
		msgRef  *dto.MessageReference
	)

	for _, elem := range elems {
		switch e := elem.(type) {
		case *message.TextElement:
			content += textLinkStrip(e.Content)
		case *message.AtElement:
			target := strings.TrimPrefix(e.Target, "OpenQQCH:")
			target = strings.TrimPrefix(target, "QQ:")
			if target == "all" {
				content += "<qqbot-at-everyone />"
			} else {
				content += fmt.Sprintf("<qqbot-at-user id=\"%s\" />", target)
			}
		case *message.ImageElement:
		case *message.ReplyElement:
			msgRef = &dto.MessageReference{
				MessageID:             e.ReplySeq,
				IgnoreGetMessageError: true,
			}
		}
	}

	toCreate := pa.initMessageToCreate(nil, rowMsgID)
	toCreate.MessageReference = msgRef
	pa.finalizeMessageToCreate(toCreate, content, keyboardObj, true)

	res, err := pa.Api.PostMessage(qctx, channelID, toCreate)
	if err != nil {
		pa.EndPoint.Session.Parent.Logger.Error("official qq 发送频道消息失败：" + err.Error())
	}
	return res, err
}

func (pa *PlatformAdapterOfficialQQ) GetGroupInfoAsync(groupID string) {
	// 警告太频繁了，拿掉
	// pa.EndPoint.Session.Parent.Logger.Infof("official qq 更新群信息失败：不支持该功能")
}

func formatDiceIDOfficialQQCh(userID string) string {
	return fmt.Sprintf("OpenQQCH:%s", userID)
}

func formatDiceIDOfficialQQChGuild(guildID string) string {
	return fmt.Sprintf("OpenQQCH-Guild:%s", guildID)
}

func formatDiceIDOfficialQQChannel(guildID, channelID string) string {
	return fmt.Sprintf("OpenQQCH-Channel:%s-%s", guildID, channelID)
}

const officialQQUserIDPrefix = "OpenQQ:"

func formatDiceIDOfficialQQ(uin string) string {
	return fmt.Sprintf("%s%s", officialQQUserIDPrefix, uin)
}

func extractOfficialQQUINFromUserID(userID string) (string, bool) {
	raw, ok := strings.CutPrefix(strings.TrimSpace(userID), officialQQUserIDPrefix)
	if !ok {
		return "", false
	}
	raw = strings.TrimSpace(raw)
	return raw, raw != ""
}

func formatDiceIDOfficialQQGroupOpenID(uin, groupOpenID string) string {
	// 官方QQ群ID格式
	return fmt.Sprintf("OpenQQ-Group:%s-%s", uin, groupOpenID)
}

func formatDiceIDOfficialQQMemberOpenID(uin, _ string, memberOpenID string) string {
	// 官方QQ群成员ID格式
	return fmt.Sprintf("%s%s-%s", officialQQUserIDPrefix, uin, memberOpenID)
}

func formatDiceIDOfficialQQUserOpenID(uin, userOpenID string) string {
	// 官方QQ单聊用户ID格式
	return fmt.Sprintf("%s%s-%s", officialQQUserIDPrefix, uin, userOpenID)
}

type OpenQQIDType = int

const (
	OpenQQUnknown OpenQQIDType = iota

	OpenQQUser
	OpenQQGroupOpenid
	OpenQQGroupMemberOpenid
	OpenQQUserOpenid

	OpenQQCHUser
	OpenQQCHGuild
	OpenQQCHChannel
)

func (pa *PlatformAdapterOfficialQQ) mustExtractID(text string) (string, OpenQQIDType) {
	id, _, idType := pa.mustExtractTwoID(text)
	return id, idType
}

func (pa *PlatformAdapterOfficialQQ) mustExtractTwoID(text string) (string, string, OpenQQIDType) {
	if raw, ok := strings.CutPrefix(text, "OpenQQ-Group:"); ok {
		groupOpenID, ok := strings.CutPrefix(raw, pa.UIN+"-")
		if pa.UIN == "" || !ok || groupOpenID == "" {
			return "", "", OpenQQUnknown
		}
		return groupOpenID, "", OpenQQGroupOpenid
	}
	if raw, ok := strings.CutPrefix(text, officialQQUserIDPrefix); ok {
		if pa.UIN != "" && raw == pa.UIN {
			return raw, "", OpenQQUser
		}
		userOpenID, ok := strings.CutPrefix(raw, pa.UIN+"-")
		if pa.UIN == "" || !ok || userOpenID == "" {
			return "", "", OpenQQUnknown
		}
		return userOpenID, "", OpenQQUserOpenid
	}
	if strings.HasPrefix(text, "OpenQQCH:") {
		return text[len("OpenQQCH:"):], "", OpenQQCHUser
	}
	if strings.HasPrefix(text, "OpenQQCH-Guild:") {
		return text[len("OpenQQCH-Guild:"):], "", OpenQQCHGuild
	}
	if strings.HasPrefix(text, "OpenQQCH-Channel:") {
		temp := text[len("OpenQQCH-Channel:"):]
		lst := strings.Split(temp, "-")
		return lst[1], lst[0], OpenQQCHChannel
	}
	return "", "", OpenQQUnknown
}

func (pa *PlatformAdapterOfficialQQ) SendFileToPerson(ctx *MsgContext, uid string, path string, flag string) {
	pa.SendToPerson(ctx, uid, fmt.Sprintf("[尝试发送文件 %s，但不支持]", filepath.Base(path)), flag)
}

func (pa *PlatformAdapterOfficialQQ) SendFileToGroup(ctx *MsgContext, uid string, path string, flag string) {
	pa.SendToGroup(ctx, uid, fmt.Sprintf("[尝试发送文件 %s，但不支持]", filepath.Base(path)), flag)
}

func (pa *PlatformAdapterOfficialQQ) QuitGroup(_ *MsgContext, _ string) {
	pa.EndPoint.Session.Parent.Logger.Error("official qq 退出群组失败：不支持该功能")
}

func (pa *PlatformAdapterOfficialQQ) SetGroupCardName(_ *MsgContext, _ string) {
	pa.EndPoint.Session.Parent.Logger.Error("official qq 修改名片失败：不支持该功能")
}

func (pa *PlatformAdapterOfficialQQ) MemberBan(_ string, _ string, _ int64) {
	pa.EndPoint.Session.Parent.Logger.Error("official qq 禁言用户失败：不支持该功能")
}

func (pa *PlatformAdapterOfficialQQ) MemberKick(_ string, _ string) {
	pa.EndPoint.Session.Parent.Logger.Error("official qq 踢出用户失败：不支持该功能")
}

func (pa *PlatformAdapterOfficialQQ) EditMessage(_ *MsgContext, _, _ string) {}

func (pa *PlatformAdapterOfficialQQ) RecallMessage(_ *MsgContext, _ string) {}

// ServeWebhook 启动Webhook模式（已整合到Serve中，保留作为兼容接口）
func (pa *PlatformAdapterOfficialQQ) ServeWebhook() int {
	return pa.Serve()
}

// handleWebhookCallback 处理Webhook回调
func (pa *PlatformAdapterOfficialQQ) handleWebhookCallback(w http.ResponseWriter, r *http.Request) {
	s := pa.EndPoint.Session
	log := s.Parent.Logger

	// 读取请求
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("official qq webhook: 读取请求失败: ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 签名验证
	if !pa.verifyWebhookSignature(body, r.Header) {
		log.Error("official qq webhook: 签名验证失败")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	log.Debugf("official qq webhook: 收到请求 %s", string(body))

	// 解析事件
	var payload dto.WSPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Error("official qq webhook: 解析事件失败: ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 保存原始消息数据，ParseAndHandle 使用
	payload.RawMessage = body
	payload.Session = &dto.Session{AppID: pa.AppID}

	// 响应确认
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"op": 12, // HTTP回调确认
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error("official qq webhook: 响应确认失败: ", err)
	}

	// 异步处理事件
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				log.Errorf("official qq webhook: 处理事件异常: %v", rec)
			}
		}()
		if err := event.ParseAndHandle(&payload); err != nil {
			log.Errorf("official qq webhook: 事件处理失败: %v", err)
		}
	}()
}

// verifyWebhookSignature 验证Webhook签名
func (pa *PlatformAdapterOfficialQQ) verifyWebhookSignature(body []byte, header http.Header) bool {
	if pa.AppSecret == "" {
		return true
	}
	pass, err := signature.Verify(pa.AppSecret, header, body)
	if err != nil {
		return false
	}
	return pass
}

type C2CRichMediaMessage struct {
	FileType   int    `json:"file_type"`
	URL        string `json:"url,omitempty"`
	SrvSendMsg bool   `json:"srv_send_msg"`
	FileData   []byte `json:"file_data,omitempty"`
}

func (msg *C2CRichMediaMessage) GetEventID() string {
	return ""
}

func (msg *C2CRichMediaMessage) GetSendType() dto.SendType {
	return dto.RichMedia
}

func isLocalOrNonPublic(urlStr string) bool {
	if urlStr == "" {
		return true
	}
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		return true
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		return true
	}
	host := u.Hostname()
	ip := net.ParseIP(host)
	if ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() {
			return true
		}
	} else if host == "localhost" {
		return true
	}
	return false
}

func getElementBytes(elem *message.FileElement) ([]byte, error) {
	if elem == nil {
		return nil, errors.New("nil element")
	}
	// 限制文件大小在30MB以下
	const maxLimit = 30 * 1024 * 1024

	readLimitBytes := func(r io.Reader) ([]byte, error) {
		limitedReader := io.LimitReader(r, maxLimit+1)
		data, err := io.ReadAll(limitedReader)
		if err != nil {
			return nil, err
		}
		if int64(len(data)) > maxLimit {
			return nil, errors.New("file size exceeds the maximum limit of 30MB")
		}
		return data, nil
	}

	if elem.Stream != nil {
		if seeker, ok := elem.Stream.(io.ReadSeeker); ok {
			_, _ = seeker.Seek(0, io.SeekStart)
		}
		return readLimitBytes(elem.Stream)
	}
	pathOrUrl := elem.URL
	if pathOrUrl == "" {
		pathOrUrl = elem.File
	}
	if pathOrUrl == "" {
		return nil, errors.New("no file path or url")
	}
	fileElem, err := message.FilepathToFileElement(pathOrUrl)
	if err != nil {
		return nil, err
	}
	if fileElem.Stream == nil {
		return nil, errors.New("failed to get stream")
	}
	return readLimitBytes(fileElem.Stream)
}

func appendAttachmentsToMessage(msg *Message, attachments []*dto.MessageAttachment) {
	for _, attach := range attachments {
		if attach.URL != "" {
			if strings.HasPrefix(attach.ContentType, "image/") {
				msg.Message += fmt.Sprintf("[CQ:image,file=%s]", attach.URL)
			} else if strings.HasPrefix(attach.ContentType, "voice") || attach.ContentType == "voice" {
				msg.Message += fmt.Sprintf("[CQ:record,file=%s]", attach.URL)
			} else if strings.HasPrefix(attach.ContentType, "video/") {
				msg.Message += fmt.Sprintf("[CQ:video,file=%s]", attach.URL)
			} else {
				ext := strings.ToLower(filepath.Ext(attach.FileName))
				if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".bmp" || ext == ".webp" {
					msg.Message += fmt.Sprintf("[CQ:image,file=%s]", attach.URL)
				} else {
					msg.Message += fmt.Sprintf("[CQ:file,file=%s]", attach.URL)
				}
			}
		}
	}
}

func (pa *PlatformAdapterOfficialQQ) parseBotSelfOpenID(event *dto.WSPayload) string {
	if event == nil || len(event.RawMessage) == 0 {
		return ""
	}
	type rawMention struct {
		ID    string `json:"id"`
		IsYou bool   `json:"is_you"`
	}
	type rawData struct {
		Mentions []rawMention `json:"mentions"`
	}
	type rawPayload struct {
		Data rawData `json:"data"`
		D    rawData `json:"d"`
	}
	var payload rawPayload
	if err := json.Unmarshal(event.RawMessage, &payload); err == nil {
		var mentions []rawMention
		if len(payload.Data.Mentions) > 0 {
			mentions = payload.Data.Mentions
		} else {
			mentions = payload.D.Mentions
		}
		for _, m := range mentions {
			if m.IsYou {
				return m.ID
			}
		}
	}
	return ""
}

// ============================================================
// QQ官方机器人扫码登录（第三方 Agent 接入）
// 参考 https://bot.q.qq.com/wiki/agent-qqbot/#第三方-agent-接入
// ============================================================

// QQ官方机器人扫码登录相关常量
const (
	qqBotCreateBindTaskURL = "https://q.qq.com/lite/create_bind_task"
	qqBotPollBindResultURL = "https://q.qq.com/lite/poll_bind_result"
	qqBotConnectPageURL    = "https://q.qq.com/qqbot/openclaw/connect.html"

	// 扫码绑定任务状态
	qqBotBindStatusNone     = 0 // 未知/初始
	qqBotBindStatusPending  = 1 // 等待扫码确认
	qqBotBindStatusComplete = 2 // 扫码完成
	qqBotBindStatusExpired  = 3 // 二维码已过期

	qqBotBindTaskTimeout = 10 * time.Second // 单次HTTP请求超时
)

// OfficialQQLoginState 扫码登录状态（参考 milky 的 MilkyLoginState）
type OfficialQQLoginState int64

const (
	OfficialQQLoginStateInit             OfficialQQLoginState = iota // 初始
	OfficialQQLoginStateQRWaitingForScan                             // 等待扫码
	OfficialQQLoginStateQRScanned                                    // 已扫码，等待确认（暂未细分使用）
	OfficialQQLoginStateConnecting                                   // 扫码成功，正在连接
	OfficialQQLoginStateFailed                                       // 失败/取消
)

// QQOfficialQrLoginSession 单次扫码登录会话
type QQOfficialQrLoginSession struct {
	ID        string
	TaskID    string
	Key       string // base64 编码的 32 字节随机密钥
	Source    string // 接入平台标识
	QrURL     string // 当前二维码对应的 URL
	CreatedAt time.Time
	mu        sync.Mutex
}

// clearQrLoginState 清理扫码登录相关状态
func (pa *PlatformAdapterOfficialQQ) clearQrLoginState() {
	pa.qrMu.Lock()
	defer pa.qrMu.Unlock()
	pa.QrLoginState = OfficialQQLoginStateInit
	pa.QrURL = ""
	pa.QrCodeData = nil
	pa.qrSession = nil
}

// markQrLoginFailed 标记扫码登录失败并清理会话
func (pa *PlatformAdapterOfficialQQ) markQrLoginFailed() {
	pa.qrMu.Lock()
	defer pa.qrMu.Unlock()
	pa.QrLoginState = OfficialQQLoginStateFailed
	pa.QrURL = ""
	pa.QrCodeData = nil
	pa.qrSession = nil
}

// qqBotGenerateBindKey 生成 32 字节随机密钥并 base64 编码
func qqBotGenerateBindKey() (string, error) {
	buf := make([]byte, 32)
	if _, err := crand.Read(buf); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}

// qqBotBuildConnectURL 构造二维码扫描对应的 URL
func qqBotBuildConnectURL(taskID, source string) string {
	return fmt.Sprintf("%s?task_id=%s&source=%s&_wv=2",
		qqBotConnectPageURL,
		url.QueryEscape(taskID),
		url.QueryEscape(source),
	)
}

// qqBotCreateBindTask 创建绑定任务
func qqBotCreateBindTask() (taskID string, key string, err error) {
	key, err = qqBotGenerateBindKey()
	if err != nil {
		return "", "", fmt.Errorf("生成密钥失败: %w", err)
	}

	reqBody, _ := json.Marshal(map[string]string{"key": key})
	resp, err := qqBotHttpPost(qqBotCreateBindTaskURL, reqBody)
	if err != nil {
		return "", "", fmt.Errorf("创建绑定任务失败: %w", err)
	}

	var result struct {
		RetCode int    `json:"retcode"`
		Msg     string `json:"msg"`
		Data    struct {
			TaskID string `json:"task_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", "", fmt.Errorf("解析创建任务响应失败: %w", err)
	}
	if result.RetCode != 0 {
		msg := result.Msg
		if msg == "" {
			msg = "create_bind_task failed"
		}
		return "", "", errors.New(msg)
	}
	if result.Data.TaskID == "" {
		return "", "", errors.New("create_bind_task: missing task_id")
	}
	return result.Data.TaskID, key, nil
}

// qqBotBindResult 轮询绑定结果
type qqBotBindResult struct {
	Status           int    // 绑定状态
	BotAppID         string // 机器人 AppID
	BotEncryptSecret string // 加密的 AppSecret
	UserOpenID       string // 扫码用户 openid
}

func qqBotPollBindResult(taskID string) (*qqBotBindResult, error) {
	reqBody, _ := json.Marshal(map[string]string{"task_id": taskID})
	resp, err := qqBotHttpPost(qqBotPollBindResultURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("轮询绑定结果失败: %w", err)
	}

	var result struct {
		RetCode int    `json:"retcode"`
		Msg     string `json:"msg"`
		Data    struct {
			Status           int    `json:"status"`
			BotAppID         any    `json:"bot_appid"`
			BotEncryptSecret string `json:"bot_encrypt_secret"`
			UserOpenID       string `json:"user_openid"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("解析轮询响应失败: %w", err)
	}
	if result.RetCode != 0 {
		msg := result.Msg
		if msg == "" {
			msg = "poll_bind_result failed"
		}
		return nil, errors.New(msg)
	}

	return &qqBotBindResult{
		Status:           result.Data.Status,
		BotAppID:         fmt.Sprintf("%v", result.Data.BotAppID),
		BotEncryptSecret: result.Data.BotEncryptSecret,
		UserOpenID:       result.Data.UserOpenID,
	}, nil
}

// qqBotDecryptSecret 使用 AES-256-GCM 解密 AppSecret
// key 为 base64 编码的 32 字节密钥；encryptedSecret 为 base64 编码的
// nonce(12B) || ciphertext || tag(16B)
func qqBotDecryptSecret(encryptedSecret, key string) (string, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return "", fmt.Errorf("解码密钥失败: %w", err)
	}
	if len(keyBytes) != 32 {
		return "", fmt.Errorf("密钥长度非法: %d", len(keyBytes))
	}

	data, err := base64.StdEncoding.DecodeString(encryptedSecret)
	if err != nil {
		return "", fmt.Errorf("解码密文失败: %w", err)
	}
	if len(data) < 12+16 {
		return "", errors.New("密文长度不足")
	}

	nonce := data[:12]
	tag := data[len(data)-16:]
	ciphertext := data[12 : len(data)-16]

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", fmt.Errorf("创建 AES cipher 失败: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建 GCM 失败: %w", err)
	}

	// Go 的 GCM.Open 期望 ciphertext || tag
	ct := make([]byte, 0, len(ciphertext)+len(tag))
	ct = append(ct, ciphertext...)
	ct = append(ct, tag...)

	plaintext, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("解密失败: %w", err)
	}
	return string(plaintext), nil
}

// qqBotHttpPost 发送 JSON POST 请求
func qqBotHttpPost(url string, body []byte) ([]byte, error) {
	client := &http.Client{Timeout: qqBotBindTaskTimeout}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	return io.ReadAll(resp.Body)
}

func (pa *PlatformAdapterOfficialQQ) failQrLogin(logMsg string, err error) {
	ep := pa.EndPoint
	d := ep.Session.Parent
	log := d.Logger
	if err != nil {
		log.Error(logMsg, err)
	} else {
		log.Error(logMsg)
	}
	pa.markQrLoginFailed()
	ep.State = 3
	ep.Enable = false
	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
}

// serveQrLogin 在 adapter 内执行扫码登录全流程：
//   - 生成二维码后写入 QrCodeData，状态置为 QRWaitingForScan
//   - 二维码过期自动刷新
//   - 扫码成功后落盘凭据、显式探测、检查重复账号，再启动连接
//   - 失败/取消置状态为 Failed
func (pa *PlatformAdapterOfficialQQ) serveQrLogin(source string) {
	ep := pa.EndPoint
	d := ep.Session.Parent
	log := d.Logger

	pa.qrMu.Lock()
	pa.QrLoginState = OfficialQQLoginStateInit
	pa.qrMu.Unlock()

	// 启动扫码会话
	taskID, key, err := qqBotCreateBindTask()
	if err != nil {
		pa.failQrLogin("official qq 创建扫码任务失败: ", err)
		return
	}

	session := &QQOfficialQrLoginSession{
		ID:        uuid.New().String(),
		TaskID:    taskID,
		Key:       key,
		Source:    source,
		QrURL:     qqBotBuildConnectURL(taskID, source),
		CreatedAt: time.Now(),
	}

	qrCodeData, err := qrcode.Encode(session.QrURL, qrcode.Medium, 256)
	if err != nil {
		pa.failQrLogin("official qq 生成二维码失败: ", err)
		return
	}

	pa.qrMu.Lock()
	pa.qrSession = session
	pa.QrURL = session.QrURL
	pa.QrCodeData = qrCodeData
	pa.QrLoginState = OfficialQQLoginStateQRWaitingForScan
	pa.qrMu.Unlock()
	log.Info("official qq 扫码二维码已就绪")

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-pa.Ctx.Done():
				return
			case <-ticker.C:
			}

			pa.qrMu.Lock()
			curSession := pa.qrSession
			pa.qrMu.Unlock()
			if curSession == nil {
				return
			}

			curSession.mu.Lock()
			result, perr := qqBotPollBindResult(curSession.TaskID)
			if perr != nil {
				curSession.mu.Unlock()
				log.Debugf("official qq 轮询扫码状态失败: %v", perr)
				continue
			}

			switch result.Status {
			case qqBotBindStatusComplete:
				pa.AppID = strings.TrimSpace(result.BotAppID)
				curSession.mu.Unlock()
				if pa.AppID == "" {
					pa.failQrLogin("official qq 扫码结果缺少 AppID: ", nil)
					return
				}

				secret, derr := qqBotDecryptSecret(result.BotEncryptSecret, curSession.Key)
				if derr != nil {
					pa.failQrLogin("official qq 解密 AppSecret 失败: ", derr)
					return
				}
				pa.AppSecret = secret
				pa.qrMu.Lock()
				pa.QrLoginState = OfficialQQLoginStateConnecting
				pa.QrURL = ""
				pa.QrCodeData = nil
				pa.qrSession = nil
				pa.qrMu.Unlock()
				d.LastUpdatedTime = time.Now().Unix()
				d.Save(false)
				log.Infof("official qq 扫码成功，AppID=%s，开始探测凭据", pa.AppID)

				// 停止扫码 context；后续探测与 connect 使用独立生命周期
				pa.stopSessionContext()

				probeCtx, probeCancel := context.WithTimeout(context.Background(), 15*time.Second)
				probe, probeErr := ProbeOfficialQQAccount(probeCtx, pa.AppID, pa.AppSecret)
				probeCancel()
				if probeErr != nil {
					pa.failQrLogin("official qq 扫码后凭据探测失败: ", probeErr)
					return
				}

				existing := FindOfficialQQEndpointByUIN(d.ImSession, probe.UIN, ep.ID)
				if existing != nil {
					log.Infof("official qq 扫码账号重复，准备替换已有端点: UIN=%s temporaryID=%s", probe.UIN, ep.ID)
					existingAdapter, replaceErr := replaceOfficialQQCredentials(existing, pa, probe)
					if replaceErr != nil {
						pa.failQrLogin("official qq 替换已有账号失败: ", replaceErr)
						return
					}
					reloginOK := existingAdapter.DoRelogin()
					if !reloginOK {
						log.Errorf("official qq 替换后重连已有端点失败: UIN=%s", probe.UIN)
					} else {
						log.Infof("official qq 重复账号替换成功，已有端点已开始重连: UIN=%s", probe.UIN)
					}
					pa.AppID = ""
					pa.AppSecret = ""
					pa.UIN = ""
					ep.UserID = ""
					ep.Nickname = ""
					pa.markQrLoginFailed()
					ep.State = 3
					ep.Enable = false
					d.LastUpdatedTime = time.Now().Unix()
					d.Save(false)
					log.Infof("official qq 临时扫码端点已禁用: UIN=%s temporaryID=%s", probe.UIN, ep.ID)
					return
				}

				pa.applyProbeResult(probe)
				d.LastUpdatedTime = time.Now().Unix()
				d.Save(false)
				log.Infof("official qq 扫码探测完成，UIN=%s，开始连接", probe.UIN)
				if pa.connect(probe) != 0 {
					ep.Enable = false
					d.LastUpdatedTime = time.Now().Unix()
					d.Save(false)
				}
				return

			case qqBotBindStatusExpired:
				newTaskID, newKey, cerr := qqBotCreateBindTask()
				if cerr != nil {
					curSession.mu.Unlock()
					pa.failQrLogin("official qq 刷新扫码任务失败: ", cerr)
					return
				}
				newQrURL := qqBotBuildConnectURL(newTaskID, curSession.Source)
				newQrCodeData, qerr := qrcode.Encode(newQrURL, qrcode.Medium, 256)
				if qerr != nil {
					curSession.mu.Unlock()
					pa.failQrLogin("official qq 生成新二维码失败: ", qerr)
					return
				}
				curSession.TaskID = newTaskID
				curSession.Key = newKey
				curSession.QrURL = newQrURL
				curSession.mu.Unlock()

				pa.qrMu.Lock()
				pa.QrURL = newQrURL
				pa.QrCodeData = newQrCodeData
				pa.QrLoginState = OfficialQQLoginStateQRWaitingForScan
				pa.qrMu.Unlock()
				log.Info("official qq 二维码已刷新")

			default:
				curSession.mu.Unlock()
			}
		}
	}()
}
