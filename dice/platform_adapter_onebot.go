package dice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	socketio "github.com/PaienNate/pineutil/evsocket"
	"github.com/avast/retry-go"
	"github.com/bytedance/sonic"
	"github.com/labstack/echo/v4"
	"github.com/maypok86/otter"
	"github.com/panjf2000/ants/v2"
	"go.uber.org/zap"

	emitter "sealdice-core/dice/imsdk/onebot"
	"sealdice-core/logger"
	"sealdice-core/message"
)

type PlatformAdapterOnebot struct {
	Session             *IMSession    `json:"-"                     yaml:"-"`
	EndPoint            *EndPointInfo `json:"-"                     yaml:"-"`
	Token               string        `json:"token"                 yaml:"token"`                 // 正向或者反向时，使用的Token
	ConnectURL          string        `json:"connectUrl"            yaml:"connectUrl"`            // 正向时 连接地址
	Mode                string        `json:"mode"                  yaml:"mode"`                  // 什么模式 server是反向，client是正向，http
	ReverseUrl          string        `json:"reverseUrl"            yaml:"reverseUrl"`            // 反向时 监听地址
	ReverseSuffix       string        `json:"reverseSuffix"         yaml:"reverseSuffix"`         // 反向时 后缀是什么 默认是/ws
	IgnoreFriendRequest bool          `json:"ignore_friend_request" yaml:"ignore_friend_request"` // 是否忽略好友请求
	wsmode              string
	websocketManager    *socketio.SocketInstance
	ctx                 context.Context
	cancel              context.CancelFunc
	logger              *zap.SugaredLogger

	// 保护这几个
	sendEmitter emitter.Emitter
	emitterChan chan emitter.Response[sonic.NoCopyRawMessage]
	once        sync.Once

	// 执行用池
	antPool *ants.Pool
	// 群缓存
	groupCache *otter.Cache[string, *GroupCache] // 群ID和群信息的缓存

	retryAttempts  uint         // 当前重试次数
	isRetrying     bool         // 是否正在重试
	retryMutex     sync.RWMutex // 重试状态锁
	isShuttingDown bool         // 是否正在主动关闭连接

	// 连接建立互斥锁，确保同时只有一个连接建立过程
	connectionMutex sync.Mutex
	isConnecting    bool // 是否正在建立连接

	echoServer *echo.Echo
}

func (p *PlatformAdapterOnebot) Serve() int {
    // 设置连接中并保存一次
    p.EndPoint.State = StateConnecting
    d := p.Session.Parent
    d.LastUpdatedTime = time.Now().Unix()
    d.Save(false)

    // 使用统一的连接启动逻辑
    if err := p.startConnection(); err != nil {
        p.logger.Errorf("启动连接失败: %v", err)
        // 连接失败，禁用并保存
        p.EndPoint.State = StateConnectionFailed
        p.EndPoint.Enable = false
        d.LastUpdatedTime = time.Now().Unix()
        d.Save(false)
        return 1
    }
    return 0
}

// DoRelogin 重新登录/重连
func (p *PlatformAdapterOnebot) DoRelogin() bool {
	p.logger.Warn("因协议端不支持所谓重新登录，重新登录功能无实际用途。")
	return true
}

// SetEnable 启用或禁用适配器
func (p *PlatformAdapterOnebot) SetEnable(enable bool) {
    d := p.Session.Parent

    if enable {
        p.logger.Info("正在启用 OneBot 适配器...")
        // 进入连接中态，由 Serve/onConnected 负责最终的 Enable 与 State=1
        p.EndPoint.State = StateConnecting
        d.LastUpdatedTime = time.Now().Unix()
        d.Save(false)
        // 统一走 Serve，避免重复状态逻辑
        if p.Serve() != 0 {
            p.EndPoint.State = StateConnectionFailed
            p.EndPoint.Enable = false
            d.LastUpdatedTime = time.Now().Unix()
            d.Save(false)
        }
    } else {
        p.logger.Info("正在禁用 OneBot 适配器...")
        p.EndPoint.Enable = false
        p.EndPoint.State = StateDisconnected

        // 清理资源
        p.cleanupResources()

        p.logger.Info("OneBot 适配器已禁用")
    }

    // 更新状态并保存
    d.LastUpdatedTime = time.Now().Unix()
    d.Save(false)
}

func (p *PlatformAdapterOnebot) QuitGroup(_ *MsgContext, id string) {
	if p.sendEmitter != nil {
		_ = p.sendEmitter.QuitGroup(p.ctx, ExtractQQEmitterGroupID(id))
	}
}

// SendToPerson 这几个到时候直接调用SendSegment的方法来处理，为以后铺路
func (p *PlatformAdapterOnebot) SendToPerson(ctx *MsgContext, userID string, text string, flag string) {
	msgElement := message.ConvertStringMessage(text)
	p.SendSegmentToPerson(ctx, userID, msgElement, flag)
}

// SendToGroup 这几个到时候直接调用SendSegment的方法来处理，为以后铺路
func (p *PlatformAdapterOnebot) SendToGroup(ctx *MsgContext, groupID string, text string, flag string) {
	msgElement := message.ConvertStringMessage(text)
	p.SendSegmentToGroup(ctx, groupID, msgElement, flag)
}

func (p *PlatformAdapterOnebot) SetGroupCardName(ctx *MsgContext, name string) {
	groupID := ctx.Group.GroupID
	userID := ctx.Player.UserID
	err := p.sendEmitter.SetGroupCard(p.ctx, ExtractQQEmitterGroupID(groupID), ExtractQQEmitterUserID(userID), name)
	if err != nil {
		p.logger.Errorf("Failed to set group card name: %v", err)
		return
	}
}

func (p *PlatformAdapterOnebot) SendSegmentToGroup(ctx *MsgContext, groupID string, msg []message.IMessageElement, flag string) {
	rawMsg, msgText := convertSealMsgToMessageChain(msg)
	rawId, err := p.sendEmitter.SendGrMsg(p.ctx, ExtractQQEmitterGroupID(groupID), rawMsg) // 这里可以获取到发送消息的ID
	if err != nil {
		return
	}
	// 支援插件发送调用
	p.Session.OnMessageSend(ctx, &Message{
		Platform:    "QQ",
		MessageType: "group",
		Segment:     msg,
		Message:     msgText,
		Sender: SenderBase{
			UserID:   p.EndPoint.UserID,
			Nickname: p.EndPoint.Nickname,
		},
		RawID: rawId,
	}, flag)
}

func (p *PlatformAdapterOnebot) SendSegmentToPerson(ctx *MsgContext, userID string, msg []message.IMessageElement, flag string) {
	rawMsg, msgText := convertSealMsgToMessageChain(msg)
	rawId, err := p.sendEmitter.SendPvtMsg(p.ctx, ExtractQQEmitterUserID(userID), rawMsg) // 这里可以获取到发送消息的ID
	if err != nil {
		p.logger.Errorf("发送消息异常 %v", err)
		return
	}
	// 支援插件发送调用
	p.Session.OnMessageSend(ctx, &Message{
		Platform:    "QQ",
		MessageType: "private",
		Segment:     msg,
		Message:     msgText,
		Sender: SenderBase{
			UserID:   p.EndPoint.UserID,
			Nickname: p.EndPoint.Nickname,
		},
		RawID: rawId,
	}, flag)
}

func (p *PlatformAdapterOnebot) SendFileToPerson(ctx *MsgContext, userID string, path string, flag string) {
	msg := []message.IMessageElement{
		&message.FileElement{URL: path},
	}
	p.SendSegmentToPerson(ctx, userID, msg, flag)
}

func (p *PlatformAdapterOnebot) SendFileToGroup(ctx *MsgContext, groupID string, path string, flag string) {
	msg := []message.IMessageElement{
		&message.FileElement{URL: path},
	}
	p.SendSegmentToGroup(ctx, groupID, msg, flag)
}

func (p *PlatformAdapterOnebot) GetGroupInfoAsync(groupID string) {
	_ = p.antPool.Submit(func() {
		p.GetGroupInfoSync(groupID)
	})
}

func (p *PlatformAdapterOnebot) GetGroupInfoSync(diceGroupID string) *GroupCache {
	// TODO：去掉这个MsgContext的需求 以及这个函数设计的一坨
	ctx := &MsgContext{EndPoint: p.EndPoint, Session: p.Session, Dice: p.Session.Parent}
	rawGroupID := ExtractQQEmitterGroupID(diceGroupID)
	groupInfoResp, err := p.sendEmitter.GetGroupInfo(p.ctx, rawGroupID, true)
	if err != nil {
		p.logger.Errorf("获取群信息异常 %v", err)
		return nil
	}
	// GroupCache里放的ID也设置成Dice的群ID以免混乱
	result := &GroupCache{
		GroupAllShut:   groupInfoResp.GroupAllShut,
		GroupRemark:    groupInfoResp.GroupRemark,
		GroupId:        diceGroupID,
		GroupName:      groupInfoResp.GroupName,
		MemberCount:    groupInfoResp.MemberCount,
		MaxMemberCount: groupInfoResp.MaxMemberCount,
	}
	_ = p.groupCache.Set(diceGroupID, result)
	p.Session.Parent.Parent.GroupNameCache.Store(diceGroupID, &GroupNameCacheItem{
		Name: result.GroupName,
		time: time.Now().Unix(),
	})
	// 存储群组相关信息
	groupInfo, ok := p.Session.ServiceAtNew.Load(diceGroupID)
	if !ok {
		return result
	}
	// 群名有更新的情况
	if result.GroupName != groupInfo.GroupName {
		groupInfo.GroupName = result.GroupName
		groupInfo.UpdatedAtTime = time.Now().Unix()
	}
	// 群信息获取不到，可能退群的情况，删除群信息
	if result.MaxMemberCount == 0 {
		if _, exists := groupInfo.DiceIDExistsMap.Load(p.EndPoint.UserID); exists {
			groupInfo.DiceIDExistsMap.Delete(p.EndPoint.UserID)
			groupInfo.UpdatedAtTime = time.Now().Unix()
		}
	}
	// 发现群情况不对，可能要退群的情况。 放在这里是因为可能这个群已经被邀请进入了
	uid := groupInfo.InviteUserID
	// 邀请人有问题
	userResult := checkBlackList(uid, "user", ctx)
	if !userResult.Passed {
		if groupInfo.EnteredTime > 0 && groupInfo.EnteredTime > userResult.BanInfo.BanTime {
			text := fmt.Sprintf("本次入群为遭遇强制邀请，即将主动退群，因为邀请人%s正处于黑名单上。打扰各位还请见谅。感谢使用海豹核心。", groupInfo.InviteUserID)
			ReplyGroupRaw(ctx, &Message{GroupID: diceGroupID}, text, "")
			time.Sleep(1 * time.Second)
			p.QuitGroup(ctx, diceGroupID)
		}
	}
	// 这群有问题
	groupResult := checkBlackList(diceGroupID, "group", ctx)
	if !groupResult.Passed {
		// 如果是被ban之后拉群，判定为强制拉群
		if groupInfo.EnteredTime > 0 && groupInfo.EnteredTime > userResult.BanInfo.BanTime {
			text := fmt.Sprintf("该群已被拉黑，即将自动退出，解封请联系骰主。打扰各位还请见谅。感谢使用海豹核心:\n当前情况: %s", userResult.BanInfo.toText(ctx.Dice))
			ReplyGroupRaw(ctx, &Message{GroupID: diceGroupID}, text, "")
			time.Sleep(1 * time.Second)
			p.QuitGroup(ctx, diceGroupID)
		}
	}
	return result
}

func (p *PlatformAdapterOnebot) GetGroupCacheInfo(groupID string) *GroupCache {
	res, ok := p.groupCache.Get(groupID)
	if ok {
		return res
	}
	return p.GetGroupInfoSync(groupID)
}

// 废弃代码

func (p *PlatformAdapterOnebot) EditMessage(_ *MsgContext, _, _ string) {}

func (p *PlatformAdapterOnebot) RecallMessage(_ *MsgContext, _ string) {
}

func (p *PlatformAdapterOnebot) MemberBan(_ string, _ string, _ int64) {
}

func (p *PlatformAdapterOnebot) MemberKick(_ string, _ string) {

}

// onConnected 连接成功的回调函数
func (p *PlatformAdapterOnebot) onConnected(kws *socketio.WebsocketWrapper) {
	// 连接成功，获取当前登录状态
	if p.emitterChan == nil {
		p.emitterChan = make(chan emitter.Response[sonic.NoCopyRawMessage], 32)
	}
	p.sendEmitter = emitter.NewEVEmitter(kws, p.emitterChan)
	info, err := p.sendEmitter.GetLoginInfo(p.ctx)
	if err != nil {
		p.logger.Errorf("获取登录信息异常 %v", err)
		p.EndPoint.State = 3
		return
	}
	p.logger.Infof("OneBot 连接成功，账号<%s>(%d)", info.NickName, info.UserId)
	p.EndPoint.UserID = fmt.Sprintf("QQ:%d", info.UserId)
	p.EndPoint.Nickname = info.NickName
    // 状态设置
    p.EndPoint.State = 1
    p.EndPoint.Enable = true
    // 更新上次时间，并存储
    d := p.Session.Parent
    d.LastUpdatedTime = time.Now().Unix()
    d.Save(false)
}

// initializeCommonResources 初始化公共资源（移除sync.Once限制，允许重新初始化）
func (p *PlatformAdapterOnebot) initializeCommonResources() {
	// 注意：这个方法已经在startConnection中被connectionMutex保护，所以不需要额外的锁

	// 条件性初始化websocketManager
	if p.websocketManager == nil {
		p.websocketManager = socketio.NewSocketInstance()

		// 注册事件处理器
		p.websocketManager.On(socketio.EventMessage, p.serveOnebotEvent)
		p.websocketManager.On(OnebotEventPostTypeMessage, p.onOnebotMessageEvent)
		p.websocketManager.On(OnebotEventPostTypeMetaEvent, p.onOnebotMetaDataEvent)
		p.websocketManager.On(OnebotEventPostTypeRequest, p.onOnebotRequestEvent)
		p.websocketManager.On(OnebotEventPostTypeNotice, p.OnebotNoticeEvent)
		p.websocketManager.On(OnebotReceiveMessage, func(payload *socketio.EventPayload) {
			var echoer emitter.Response[sonic.NoCopyRawMessage]
			if err := sonic.Unmarshal(payload.Data, &echoer); err != nil {
				p.logger.Errorf("echo 数据传输异常 %v", err)
			}
			p.emitterChan <- echoer
		})
	}

	// 每次都重新创建上下文
	p.ctx, p.cancel = context.WithCancel(context.Background())

	// 条件性初始化antPool
	if p.antPool == nil {
		p.antPool, _ = ants.NewPool(500, ants.WithPanicHandler(func(re any) {
			p.logger.Errorf("执行发送数据任务异常 %v", re)
		}))
	}

	// 条件性初始化groupCache
	if p.groupCache == nil {
		cacher, _ := otter.MustBuilder[string, *GroupCache](1000).
			CollectStats().
			WithTTL(time.Hour). // 一小时后过期
			Build()
		p.groupCache = &cacher
	}
}

// setupClientConnection 设置客户端连接
func (p *PlatformAdapterOnebot) setupClientConnection() error {
	options := socketio.ClientOptions{
		UseSSL: strings.Contains(p.ConnectURL, "wss://"),
	}
	client := p.websocketManager.NewClient(p.ConnectURL, options)

	if p.Token != "" {
		client.RequestHeader.Set("Authorization", p.Token)
	}

	client.OnConnectError = func(err error) {
		p.logger.Errorf("连接失败: %v", err)
		p.EndPoint.State = StateConnecting
		go p.retryConnect()
	}

	client.OnDisconnected = func(err error) {
		// 特判：只有在非主动关闭的情况下才记录为异常断开
		if !p.isShuttingDown && p.EndPoint.Enable {
			if err != nil {
				p.logger.Errorf("连接异常断开: %v", err)
			} else {
				p.logger.Warn("连接意外断开")
			}
			p.EndPoint.State = StateConnecting
			go p.retryConnect()
		} else {
			// 主动关闭或适配器已禁用的情况
			if p.isShuttingDown {
				p.logger.Info("连接已主动关闭")
			} else {
				p.logger.Info("适配器已禁用，连接断开")
			}
			p.EndPoint.State = StateDisconnected
		}
	}

	return client.ClientConnect(p.onConnected)
}

// setupServerConnection 设置服务器连接
func (p *PlatformAdapterOnebot) setupServerConnection() error {
	p.echoServer = echo.New()

	// 注册Handler
	p.echoServer.GET(p.ReverseSuffix, echo.WrapHandler(
		p.websocketManager.New(func(kws *socketio.WebsocketWrapper) {
			// 先检查是否允许
			if p.Token != "" {
				token := kws.RequestHeader.Get("Authorization")
				token = strings.TrimPrefix(token, "Bearer ")
				if p.Token != token {
					kws.Emit([]byte(`{
						"status": "failed",
						"retcode": 1403,
						"data": null,
						"message": "token验证失败",
						"wording": "token验证失败",
						"echo": null,
						"stream": "normal-action"
					}`))
					kws.Close()
					return
				}
			}
			p.onConnected(kws)
		}),
	))

	return p.echoServer.Start(p.ReverseUrl)
}

// cleanupResources 清理资源
func (p *PlatformAdapterOnebot) cleanupResources() {
	// 设置主动关闭标志
	p.isShuttingDown = true

	// 1. 首先关闭服务器，停止接收新的请求（避免在清理过程中崩溃）
	if p.Mode == "server" && p.echoServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := p.echoServer.Shutdown(ctx); err != nil {
			p.logger.Errorf("关闭服务器失败: %v", err)
		}
		p.echoServer = nil
	}

	// 2. 取消上下文，停止所有相关的goroutine
	if p.cancel != nil {
		p.cancel()
	}

	// 3. 关闭WebSocket管理器（生命周期控制移到SocketInstance层）
	if p.websocketManager != nil {
		err := p.websocketManager.Shutdown()
		if err != nil {
			p.logger.Errorf("关闭WebSocket管理器失败: %v", err)
		}
		// 重置WebSocket管理器，为重新初始化做准备
		p.websocketManager = nil
	}

	// 4. 关闭资源池
	if p.antPool != nil {
		p.antPool.Release()
		p.antPool = nil
	}

	// 5. 重置连接状态标志
	p.connectionMutex.Lock()
	p.isConnecting = false
	p.EndPoint.State = StateDisconnected
	p.connectionMutex.Unlock()

	// 6. 重置主动关闭标志
	p.isShuttingDown = false
}

// startConnection 启动连接（统一的连接启动逻辑）
func (p *PlatformAdapterOnebot) startConnection() error {
	// 使用互斥锁确保同时只有一个连接建立过程
	p.connectionMutex.Lock()
	defer p.connectionMutex.Unlock()
	// 初始化logger 必须在最前初始化
	if p.logger == nil {
		p.logger = zap.S().Named(logger.LogKeyAdapter)
	}

	// 检查是否已经在连接中
	if p.isConnecting {
		p.logger.Info("连接建立过程已在进行中，跳过重复连接")
		return errors.New("连接建立过程已在进行中")
	}

	// 设置连接中状态
	p.isConnecting = true
	defer func() {
		p.isConnecting = false
	}()

	p.logger.Info("开始建立连接...")

	// 确保公共资源已初始化（包括上下文创建）
	p.initializeCommonResources()

	switch p.Mode {
	case "client":
		return p.setupClientConnection()
	case "server":
		// 服务器模式在 goroutine 中启动，避免阻塞
		go func() {
			if err := p.setupServerConnection(); err != nil {
				p.logger.Errorf("启动服务器失败: %v", err)
				p.EndPoint.State = StateConnecting
			}
		}()
		return nil
	default:
		return fmt.Errorf("未知的连接模式: %s", p.Mode)
	}
}

// retryConnect 重试连接方法
func (p *PlatformAdapterOnebot) retryConnect() {
	// 检查适配器是否已被禁用
	if !p.EndPoint.Enable {
		p.logger.Info("适配器已被禁用，取消重连")
		return
	}

	p.retryMutex.Lock()
	if p.isRetrying {
		p.retryMutex.Unlock()
		return // 已经在重试中，避免重复重试
	}
	p.isRetrying = true
	p.retryMutex.Unlock()

	defer func() {
		p.retryMutex.Lock()
		p.isRetrying = false
		p.retryAttempts = 0
		p.retryMutex.Unlock()
	}()

	const maxRetries = 5
	const baseDelay = 2 * time.Second

	err := retry.Do(
		func() error {
			// 在每次重试前再次检查适配器是否已被禁用
			if !p.EndPoint.Enable {
				return errors.New("适配器已被禁用，停止重连")
			}

			p.retryMutex.Lock()
			p.retryAttempts++
			currentAttempt := p.retryAttempts
			p.retryMutex.Unlock()

			// 计算下次重试时间（指数退避）
			nextRetryDelay := time.Duration(1<<(currentAttempt-1)) * baseDelay
			if currentAttempt < maxRetries {
				p.logger.Infof("尝试重新连接 OneBot [%d/%d]，下次重试间隔: %v", currentAttempt, maxRetries, nextRetryDelay)
			} else {
				p.logger.Infof("尝试重新连接 OneBot [%d/%d]，最后一次尝试", currentAttempt, maxRetries)
			}

			// 调用startConnection重新建立连接，确保遵循连接互斥锁机制
			return p.startConnection()
		},
		retry.Attempts(maxRetries),
		retry.Delay(baseDelay),
		retry.DelayType(retry.BackOffDelay),
		retry.OnRetry(func(n uint, err error) {
			// 在重试前再次检查适配器是否已被禁用
			if !p.EndPoint.Enable {
				p.logger.Info("适配器已被禁用，停止重试")
				return
			}
			nextDelay := time.Duration(1<<n) * baseDelay
			p.logger.Warnf("重连失败 (第%d次): %v，%v后进行下次重试", n+1, err, nextDelay)
		}),
	)

	if err != nil {
		// 检查是否是因为适配器被禁用而失败
		if strings.Contains(err.Error(), "适配器已被禁用") {
			p.logger.Info("重连已停止：适配器被禁用")
		} else {
			p.logger.Errorf("重连最终失败，已达到最大重试次数 %d: %v", maxRetries, err)
			p.EndPoint.State = 3
		}
	} else {
		p.logger.Infof("OneBot 重连成功")
		// 重置重试状态
		p.retryMutex.Lock()
		p.retryAttempts = 0
		p.retryMutex.Unlock()
	}
}
