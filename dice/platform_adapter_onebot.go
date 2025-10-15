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
}

func (p *PlatformAdapterOnebot) Serve() int {
	p.websocketManager = socketio.NewSocketInstance()
	// 注册事件
	p.websocketManager.On(socketio.EventMessage, p.serveOnebotEvent)
	p.websocketManager.On(OnebotEventPostTypeMessage, p.onOnebotMessageEvent)
	p.websocketManager.On(OnebotEventPostTypeMetaEvent, p.onOnebotMetaDataEvent)
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
	res, err := p.sendEmitter.Raw(p.ctx, "get_group_info", map[string]interface{}{
		"group_id": groupID,
		"no_cache": true,
	})
	if err != nil {
		p.logger.Errorf("获取群信息异常 %v", err)
	}
	name := gjson.ParseBytes(res).Get("group_name").String()
	p.Session.Parent.Parent.GroupNameCache.Store(groupID, &GroupNameCacheItem{
		Name: name,
		time: time.Now().Unix(),
	})
}

// 全是废弃的，TNND。
func (p *PlatformAdapterOnebot) EditMessage(ctx *MsgContext, msgID, message string) {}

func (p *PlatformAdapterOnebot) RecallMessage(ctx *MsgContext, msgID string) {
}

func (p *PlatformAdapterOnebot) MemberBan(groupID string, userID string, duration int64) {
}

func (p *PlatformAdapterOnebot) MemberKick(groupID string, userID string) {

}
