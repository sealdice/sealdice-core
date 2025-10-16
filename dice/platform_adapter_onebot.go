package dice

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	socketio "github.com/PaienNate/pineutil/evsocket"
	"github.com/bytedance/sonic"
	"github.com/maypok86/otter"
	"github.com/panjf2000/ants/v2"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"

	emitter "sealdice-core/dice/utils/onebot"
	"sealdice-core/logger"
	"sealdice-core/message"
)

// 2025-10-14 首先启用客户端模式

type PlatformAdapterOnebot struct {
	Session          *IMSession    `json:"-"        yaml:"-"`
	EndPoint         *EndPointInfo `json:"-"        yaml:"-"`
	Token            string        `json:"token"    yaml:"token"`        // 正向或者反向时，使用的Token
	ConnectURL       string        `yaml:"connectUrl" json:"connectUrl"` // 正向时 连接地址
	Mode             string        `yaml:"mode" json:"mode"`             // 什么模式 server是反向，client是正向，http
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
		var echo emitter.Response[json.RawMessage]
		if err := sonic.Unmarshal(payload.Data, &echo); err != nil {
			p.logger.Errorf("echo 数据传输异常 %v", err)
		}
		p.emitterChan <- echo
	})
	p.logger = logger.M()
	p.ctx, p.cancel = context.WithCancel(context.Background())
	d := p.Session.Parent
	switch p.Mode {
	case "client":
		options := socketio.ClientOptions{
			UseSSL: strings.Contains(p.ConnectURL, "wss://"),
		}
		client := p.websocketManager.NewClient(p.ConnectURL, options)
		if p.Token != "" {
			client.RequestHeader.Set("Authorization", p.Token)
		}
		client.OnConnectFailed = func(err error) {
			p.logger.Errorf("连接失败原因： %v", err)
		}
		err := client.ClientConnect(func(kws *socketio.WebsocketWrapper) {
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
		})
		if err != nil {
			p.logger.Error(err)
			return 3 // 连接失败
		}
		return 0
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
		p.sendEmitter.Raw(p.ctx, "set_group_leave", map[string]string{
			"group_id": ID,
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
	log := zap.S().Named(logger.LogKeyAdapter)
	groupID := ctx.Group.GroupID
	rawGroupID := ExtractQQGroupID(groupID)
	rawGroupIDInt, err := strconv.ParseInt(rawGroupID, 10, 64)
	if err != nil {
		log.Errorf("Invalid group ID %s: %v", groupID, err)
		return
	}
	userID := ctx.Player.UserID
	rawUserID := ExtractQQUserID(userID)
	rawUserIDInt, err := strconv.ParseInt(rawUserID, 10, 64)
	if err != nil {
		log.Errorf("Invalid user ID %s: %v", userID, err)
		return
	}
	_, err = p.sendEmitter.Raw(p.ctx, "set_group_card", map[string]interface{}{
		"group_id": rawGroupIDInt,
		"user_id":  rawUserIDInt,
		"card":     name,
	})
	if err != nil {
		log.Errorf("Failed to set group card name: %v", err)
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
	// TODO：去掉这个MsgContext的需求？
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
			text := fmt.Sprintf("被群已被拉黑，即将自动退出，解封请联系骰主。打扰各位还请见谅。感谢使用海豹核心:\n当前情况: %s", userResult.BanInfo.toText(ctx.Dice))
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
