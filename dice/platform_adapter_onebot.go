package dice

import (
	"context"
	"encoding/json"
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
	"github.com/tidwall/gjson"
	"go.uber.org/zap"

	emitter "sealdice-core/dice/utils/onebot"
	"sealdice-core/logger"
	"sealdice-core/message"
)

type PlatformAdapterOnebot struct {
	Session          *IMSession    `json:"-"        yaml:"-"`
	EndPoint         *EndPointInfo `json:"-"        yaml:"-"`
	Token            string        `json:"token"    yaml:"token"`              // 正向或者反向时，使用的Token
	ConnectURL       string        `yaml:"connectUrl" json:"connectUrl"`       // 正向时 连接地址
	Mode             string        `yaml:"mode" json:"mode"`                   // 什么模式 server是反向，client是正向，http
	ReverseUrl       string        `yaml:"reverseUrl" json:"reverseUrl"`       // 反向时 监听地址
	ReverseSuffix    string        `yaml:"reverseSuffix" json:"reverseSuffix"` // 反向时 后缀是什么 默认是/ws
	wsmode           string
	websocketManager *socketio.SocketInstance
	ctx              context.Context
	cancel           context.CancelFunc
	logger           *zap.SugaredLogger

	// 保护这几个
	sendEmitter emitter.Emitter
	emitterChan chan emitter.Response[json.RawMessage]
	once        sync.Once

	// 执行用池
	antPool *ants.Pool
	// 群缓存
	groupCache otter.Cache[string, *GroupCache] // 群ID和群信息的缓存
	// 重试相关
	client        *socketio.WebsocketWrapper // WebSocket客户端
	retryAttempts uint                       // 当前重试次数
	isRetrying    bool                       // 是否正在重试
	retryMutex    sync.RWMutex               // 重试状态锁
}

func (p *PlatformAdapterOnebot) Serve() int {
	p.antPool, _ = ants.NewPool(500, ants.WithPanicHandler(func(re any) {
		p.logger.Errorf("执行发送数据任务异常 %v", re)
	}))
	p.groupCache, _ = otter.MustBuilder[string, *GroupCache](1000).
		CollectStats().
		WithTTL(time.Hour). // 一小时后过期
		Build()
	p.websocketManager = socketio.NewSocketInstance()
	// 注册事件
	p.websocketManager.On(socketio.EventMessage, p.serveOnebotEvent)
	p.websocketManager.On(OnebotEventPostTypeMessage, p.onOnebotMessageEvent)
	p.websocketManager.On(OnebotEventPostTypeMetaEvent, p.onOnebotMetaDataEvent)
	p.websocketManager.On(OnebotEventPostTypeRequest, p.onOnebotRequestEvent)
	p.websocketManager.On(OnebotEventPostTypeNotice, p.OnebotNoticeEvent)
	p.websocketManager.On(OnebotReceiveMessage, func(payload *socketio.EventPayload) {
		var echoer emitter.Response[json.RawMessage]
		if err := sonic.Unmarshal(payload.Data, &echoer); err != nil {
			p.logger.Errorf("echo 数据传输异常 %v", err)
		}
		p.emitterChan <- echoer
	})
	p.logger = zap.S().Named(logger.LogKeyAdapter)
	p.ctx, p.cancel = context.WithCancel(context.Background())
	d := p.Session.Parent
	switch p.Mode {
	case "client":
		options := socketio.ClientOptions{
			UseSSL: strings.Contains(p.ConnectURL, "wss://"),
		}
		client := p.websocketManager.NewClient(p.ConnectURL, options)
		p.client = client // 保存client引用以便重连使用
		if p.Token != "" {
			client.RequestHeader.Set("Authorization", p.Token)
		}
		client.OnConnectError = func(err error) {
			p.logger.Errorf("连接失败原因： %v", err)
			p.EndPoint.State = 3
			go p.retryConnect()
		}
		client.OnDisconnected = func(err error) {
			p.logger.Error("连接断开")
			p.EndPoint.State = 3
			go p.retryConnect()
		}
		err := client.ClientConnect(p.onConnected)
		if err != nil {
			p.logger.Error(err)
			// 这个时候要尝试重连
			return 3 // 连接失败
		}
		return 0
	case "server":
		// 反向WebSocket 我是服务器喵
		e := echo.New()
		// 注册Handler
		e.GET("/ws", echo.WrapHandler(
			p.websocketManager.New(func(kws *socketio.WebsocketWrapper) {
				// 连接成功，获取当前登录状态
				if p.emitterChan == nil {
					p.emitterChan = make(chan emitter.Response[json.RawMessage], 32)
				}
				p.sendEmitter = emitter.NewEVEmitter(kws, p.emitterChan)
				info, err := p.sendEmitter.GetLoginInfo(p.ctx)
				if err != nil {
					p.logger.Errorf("获取登录信息异常 %v", err)
					p.EndPoint.State = 3
					return
				}
				p.logger.Infof("PureOnebot 服务连接成功，账号<%s>(%d)", info.NickName, info.UserId)
				p.EndPoint.UserID = fmt.Sprintf("QQ:%d", info.UserId)
				p.EndPoint.Nickname = info.NickName
				// 状态设置
				p.EndPoint.State = 1
				// 启动Endpoint
				p.EndPoint.Enable = true
				// 更新上次时间，并存储
				d.LastUpdatedTime = time.Now().Unix()
				d.Save(false)
			}),
		))
		err := e.Start(p.ReverseUrl)
		if err != nil {
			return 0
		}
	}
	return 0
}

func (p *PlatformAdapterOnebot) DoRelogin() bool {
	return true
}

func (p *PlatformAdapterOnebot) SetEnable(enable bool) {
}

func (p *PlatformAdapterOnebot) QuitGroup(_ *MsgContext, ID string) {
	if p.sendEmitter != nil {
		_, _ = p.sendEmitter.Raw(p.ctx, "set_group_leave", map[string]interface{}{
			"group_id": ExtractQQEmitterGroupID(ID),
		})
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
	_, err := p.sendEmitter.Raw(p.ctx, "set_group_card", map[string]interface{}{
		"group_id": ExtractQQEmitterGroupID(groupID),
		"user_id":  ExtractQQEmitterUserID(userID),
		"card":     name,
	})
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

func (p *PlatformAdapterOnebot) GetGroupInfoSync(groupID string) *GroupCache {
	// TODO：去掉这个MsgContext的需求 以及这个函数设计的一坨
	ctx := &MsgContext{EndPoint: p.EndPoint, Session: p.Session, Dice: p.Session.Parent}
	rawGroupID := ExtractQQEmitterGroupID(groupID)
	res, err := p.sendEmitter.Raw(p.ctx, "get_group_info", map[string]interface{}{
		"group_id": rawGroupID,
		"no_cache": true,
	})
	if err != nil {
		p.logger.Errorf("获取群信息异常 %v", err)
		return nil
	}
	groupInfoRaw := gjson.ParseBytes(res)
	// GroupCache里放的ID也设置成Dice的群ID以免混乱
	result := &GroupCache{
		GroupAllShut:   int(groupInfoRaw.Get("data.group_all_shut").Int()),
		GroupRemark:    groupInfoRaw.Get("data.group_remark").String(),
		GroupId:        groupID,
		GroupName:      groupInfoRaw.Get("data.group_name").String(),
		MemberCount:    int(groupInfoRaw.Get("data.member_count").Int()),
		MaxMemberCount: int(groupInfoRaw.Get("data.max_member_count").Int()),
	}
	_ = p.groupCache.Set(groupID, result)
	p.Session.Parent.Parent.GroupNameCache.Store(groupID, &GroupNameCacheItem{
		Name: result.GroupName,
		time: time.Now().Unix(),
	})
	// 存储群组相关信息
	groupInfo, ok := p.Session.ServiceAtNew.Load(groupID)
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
			ReplyGroupRaw(ctx, &Message{GroupID: groupID}, text, "")
			time.Sleep(1 * time.Second)
			p.QuitGroup(ctx, groupID)
		}
	}
	// 这群有问题
	groupResult := checkBlackList(groupID, "group", ctx)
	if !groupResult.Passed {
		// 如果是被ban之后拉群，判定为强制拉群
		if groupInfo.EnteredTime > 0 && groupInfo.EnteredTime > userResult.BanInfo.BanTime {
			text := fmt.Sprintf("该群已被拉黑，即将自动退出，解封请联系骰主。打扰各位还请见谅。感谢使用海豹核心:\n当前情况: %s", userResult.BanInfo.toText(ctx.Dice))
			ReplyGroupRaw(ctx, &Message{GroupID: groupID}, text, "")
			time.Sleep(1 * time.Second)
			p.QuitGroup(ctx, groupID)
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

// 全是废弃的，TNND。
func (p *PlatformAdapterOnebot) EditMessage(ctx *MsgContext, msgID, message string) {}

func (p *PlatformAdapterOnebot) RecallMessage(ctx *MsgContext, msgID string) {
}

func (p *PlatformAdapterOnebot) MemberBan(groupID string, userID string, duration int64) {
}

func (p *PlatformAdapterOnebot) MemberKick(groupID string, userID string) {

}

// onConnected 连接成功的回调函数
func (p *PlatformAdapterOnebot) onConnected(kws *socketio.WebsocketWrapper) {
	// 连接成功，获取当前登录状态
	if p.emitterChan == nil {
		p.emitterChan = make(chan emitter.Response[json.RawMessage], 32)
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
	// 启动Endpoint
	p.EndPoint.Enable = true
	// 更新上次时间，并存储
	d := p.Session.Parent
	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
}

// retryConnect 重试连接方法
func (p *PlatformAdapterOnebot) retryConnect() {
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

			// 直接使用接口方法，无需类型断言
			if p.client != nil {
				return p.client.ClientConnect(p.onConnected)
			}
			return fmt.Errorf("client未初始化")
		},
		retry.Attempts(maxRetries),
		retry.Delay(baseDelay),
		retry.DelayType(retry.BackOffDelay),
		retry.OnRetry(func(n uint, err error) {
			nextDelay := time.Duration(1<<n) * baseDelay
			p.logger.Warnf("重连失败 (第%d次): %v，%v后进行下次重试", n+1, err, nextDelay)
		}),
	)

	if err != nil {
		p.logger.Errorf("重连最终失败，已达到最大重试次数 %d: %v", maxRetries, err)
		p.EndPoint.State = 3
	} else {
		p.logger.Infof("OneBot 重连成功")
		// 重置重试状态
		p.retryMutex.Lock()
		p.retryAttempts = 0
		p.retryMutex.Unlock()
	}
}
