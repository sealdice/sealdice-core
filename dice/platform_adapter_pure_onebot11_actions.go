package dice

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/PaienNate/SealSocketIO/socketio"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

// 为什么不直接用WebSocket的事件逻辑，而是非要用另外一个消息队列？
// 答：1. 我正好学习一下消息队列（主因）
// 2. 采用WebSocket的话，处理onebot上报数据的逻辑和我自己定义的逻辑有可能产生混淆。
// 考虑到我自己定义的逻辑也会很多，不如直接分开处理，websocket事件只做事件分发？
// 3. 如果加好友很多，在队列里积压的话，需要持久化这里的加好友，保证消息不会丢失。
// 4. （主因2）消息队列可以很方便的做延时消息，并且能做成顺序添加，从而防止同一时间大量加好友的情况，相比直接塞个Map逻辑看着也好看。
// 事件处理部分
// 这部分用来做事件处理，流程大概描述为下面的方案：
// 注： 考虑到鲁棒性（由于用户可能自己上号操作），我们考虑只校验回应字段，不会重发，转而发送提示。一个例外是发送入群致辞
// 1. 假设有一个加好友请求。
// 2. 收到请求，解析后以生产者的形式存入对应的延时消息队列。
// 3. 程序开始的时候就启一个对应的消费者，准备消费对应的数据
// 4. 数据被消费，此时“加好友”参数发送。注意，此时还要带上一个通用处理的回应参数。
// 5. 收到回应字段，检查retcode是否正确，若不正确，需要提示用户。
// 分析： 理论上就是一个生产者和一个消费者。这样就能保证所有的顺序都能正确执行。
// 另一个情况分析：入群
// 1. 收到一个自己入群的请求
// 2. 获取群信息，带上我是加群获取群信息的标识。
// 3. echo事件收到消息，根据“加群获取群信息”的标识，将群信息封装后，发给“要加群了”的topic。
// 4. 如果失败，那就等待1s重新发送这条指令，并增加计数。如果计数 > 5，说明就是拿不到信息，以未知信息方式发送
// 5. 要加群了的topic消费之后，发送对应的致辞消息。
// 等待处理事件的 routers

// Notice: 这里的缓存现在发现其实不太有必要 因为重启之后 Flag很可能就是不可用的了。 不过，若onebot没有重启过，仍然可以正确执行，故保留
// TODO: 或许可以提供一个手段清空整个加好友列表，if onebot is down
func (p *PlatformAdapterPureOnebot11) messageQueueOnFriendAdd(msg *message.Message) error {
	// Note:(Pinenutn) 收到一条就发送一个加好友的消息。时间完全由外层的插件控制 这样就做到了针对Szz佬的ISSUE所说的 不同意也不拒绝
	var req FriendRequestResponse
	err := sonic.Unmarshal(msg.Payload, &req)
	if err != nil {
		return err
	}
	p.SetFriendAddRequest(req.Flag, req.Approve, req.Remark, req.Reason)
	return nil
}

func (p *PlatformAdapterPureOnebot11) messageQueueOnGroupAdd(msg *message.Message) error {
	// Note:(Pinenutn) 收到一条就加一个群。时间完全由外层的插件控制 这样群添加就是有序不并发 带持久化的情况了。
	var req GroupRequestResponse
	err := sonic.Unmarshal(msg.Payload, &req)
	if err != nil {
		return err
	}
	// 慢慢的发送添加群
	p.SetGroupAddRequest(req.Flag, req.SubType, req.Approve, req.Reason)
	return nil
}

// 结束

// EVENT 消息事件处理

// serveOnebotEvent 只要当收到消息时,分发event的函数
func (p *PlatformAdapterPureOnebot11) serveOnebotEvent(ep *socketio.EventPayload) {
	p.Logger.Infof("Message event - User: %s - Message: %s", ep.Kws.GetStringAttribute("user_id"), string(ep.Data))
	var err error
	if !gjson.ValidBytes(ep.Data) {
		// TODO：错误处理
		return
	}
	resp := gjson.ParseBytes(ep.Data)
	// 解析是string还是array
	// TODO: 不知道是不是通过这种方式判断是string或者array的
	switch resp.Get("message").Type {
	case gjson.String:
		p.Mode = LinkerModeString
	case gjson.JSON:
		p.Mode = LinkerModeArray
	default:
		p.Mode = LinkerModeString
	}
	if p.Mode == LinkerModeArray {
		resp, err = p.parseOB11ArrayToStringMessage(resp)
		if err != nil {
			// 日志
			p.Logger.Warnf("解析OB11的Array数据时出错，错误信息为 %v", err)
			return
		}
	}
	// 解析终了，进行分发
	eventType := resp.Get("post_type").String()
	if eventType != "" {
		// 分发事件
		eventType = fmt.Sprintf("onebot_%s", eventType)
		ep.Kws.Fire(eventType, []byte(resp.String()))
	} else {
		// 如果没有post_type，说明不是上报信息，而是API的返回信息
		ep.Kws.Fire(OnebotReceiveMessage, []byte(resp.String()))
	}
	// 完活
}

// onOnebotMessageEvent 当收到Message类型消息时
func (p *PlatformAdapterPureOnebot11) onOnebotMessageEvent(ep *socketio.EventPayload) {
	// 进行进一步的下一层解析，从而获取值
	// 如果是普通消息(群消息/私聊消息)
	// Note(Pinenutn): 我真是草了，这个Execute又是依托答辩，代码长的完全不忍阅读
	// 总之分发到这里的数据肯定都能序列化成Message的罢，对的罢？
	msg := p.convertStringMessage(gjson.ParseBytes(ep.Data))
	if msg.MessageType == "private" || msg.MessageType == "group" {
		msg.UUID = ep.SocketUUID
		p.Session.Execute(p.EndPoint, msg, false)
	}
}

// onOnebotRequestEvent 当收到Request类型消息时
func (p *PlatformAdapterPureOnebot11) onOnebotRequestEvent(ep *socketio.EventPayload) {
	// 请求分为好友请求和加群/邀请请求
	req := gjson.ParseBytes(ep.Data)
	switch req.Get("request_type").String() {
	case "friend":
		_ = p.handleReqFriendAction(req, ep)
	case "group":
		_ = p.handleReqGroupAction(req, ep)
	}
}

// onCustomReplyEvent 当收到由用户发送 接受值是API的消息时
// 如果有@作为分割，则说明@后的数据需要根据某些特定规则，发送给特定的端。
func (p *PlatformAdapterPureOnebot11) onCustomReplyEvent(ep *socketio.EventPayload) {
	req := gjson.ParseBytes(ep.Data)
	// 根据数据的不同，进行转发
	// 采用@进行剪切
	reqList := strings.Split(req.Get("echo").String(), "@")
	if len(reqList) == 0 {
		p.Logger.Warnf("未能获取到echo信息 疑似错误数据 %v", req.String())
		return
	}
	switch reqList[0] {
	case GetVersionInfo:
		// 判断是否是失败了 失败就赋值不知道消息 否则赋值它给的消息
		if req.Get("retcode").String() != "0" {
			p.Logger.Warnf("未能获取到连接信息，赋值未知信息 收到原始信息为 %v", req.String())
			p.AppVersion.AppName = "未知"
			p.AppVersion.ProtocolVersion = "未知"
			p.AppVersion.AppVersion = "未知"
		} else {
			p.AppVersion.AppName = req.Get("data.app_name").String()
			p.AppVersion.ProtocolVersion = req.Get("data.protocol_version").String()
			p.AppVersion.AppVersion = req.Get("data.app_version").String()
		}
	case GetLoginInfo:
		// 读取Data/UserID
		if req.Get("retcode").String() != "0" {
			// TODO: 考虑5S后重发 不知道靠不靠谱
			p.Logger.Warnf("未能获取到登录信息 收到原始信息为 %v，将在 5S 后 重发获取信息!", req.String())
			time.Sleep(5 * time.Second)
			p.GetLoginInfo()
			return
		}
		p.EndPoint.Nickname = req.Get("data.nickname").String()
		p.EndPoint.UserID = FormatDiceIDQQ(req.Get("data.user_id").String())
		p.Logger.Infof("已获取到骰子信息为 %v,%v", p.EndPoint.Nickname, p.EndPoint.UserID)
		p.EndPoint.RefreshGroupNum()
		d := p.Session.Parent
		d.LastUpdatedTime = time.Now().Unix()
		d.Save(false)
	case GetGroupInfo:
		// 获取群信息时，先判断获取到没有
		if req.Get("retcode").String() != "0" {
			p.Logger.Warnf("未能获取到群信息 收到原始信息为 %v", req.String())
			return
		}
		// 将数据放到缓存里
		p.GroupInfoCache.Set(req.Get("data.group_id").String(), req.Get("data"))
		// 查看这个消息分割后是否有对应的值
		if len(reqList) > 1 {
			// 通知消息队列 拼接对应参数
			topic := reqList[1] + "/" + req.Get("data.group_id").String()
			// 将收到的数据直接丢给对应的消息队列
			marshal, _ := sonic.Marshal(req.Get("data").Value())
			err := p.Publisher.Publish(topic, message.NewMessage(uuid.New().String(), marshal))
			if err != nil {
				p.Logger.Errorf("发送群信息到对应消息队列时失败：%v", err)
				return
			}
		}
		// 某些时候 获取群信息 是因为上层逻辑调用 这段逻辑暂时不知道是什么的，先拿过来
		_ = p.handleCustomGroupInfoAction(req, ep)
	}

}

// EVENT 消息事件处理函数结束

// GetLoginInfo 获取登录信息
func (pa *PlatformAdapterPureOnebot11) GetLoginInfo() {
	a, _ := sonic.Marshal(struct {
		Action string `json:"action"`
		Echo   string `json:"echo"`
	}{
		Action: "get_login_info",
		Echo:   "get_login_info",
	})
	err := pa.Instance.EmitTo(pa.UUID, a, socketio.TextMessage)
	if err != nil {
		pa.Logger.Error("发送登录消息出现异常: %v", err)
		return
	}
}

// GetVersionInfo 获取连接的Onebot的信息
func (pa *PlatformAdapterPureOnebot11) GetVersionInfo() {
	a, _ := sonic.Marshal(struct {
		Action string `json:"action"`
		Echo   string `json:"echo"`
	}{
		Action: "get_version_info",
		Echo:   "get_version_info",
	})
	err := pa.Instance.EmitTo(pa.UUID, a, socketio.TextMessage)
	if err != nil {
		pa.Logger.Error("发送连接Onebot消息出现异常: %v", err)
		return
	}
}

// SetFriendAddRequest 添加好友请求
func (p *PlatformAdapterPureOnebot11) SetFriendAddRequest(flag string, approve bool, remark string, reason string) {
	type DetailParams struct {
		Flag    string `json:"flag"`
		Remark  string `json:"remark"` // 备注名
		Approve bool   `json:"approve"`
		Reason  string `json:"reason"`
	}

	msg, _ := sonic.Marshal(oneBotCommand{
		Action: "set_friend_add_request",
		Params: DetailParams{
			Flag:    flag,
			Approve: approve,
			Remark:  remark,
			Reason:  reason,
		},
		Echo: "set_friend_add_request",
	})
	// 发送数据
	err := p.Instance.EmitTo(p.UUID, msg, socketio.TextMessage)
	if err != nil {
		p.Logger.Error("发送添加好友请求出现异常: %v", err)
		return
	}
}

func (p *PlatformAdapterPureOnebot11) SetGroupAddRequest(flag string, subType string, approve bool, reason string) {
	type DetailParams struct {
		Flag    string `json:"flag"`
		SubType string `json:"sub_type"`
		Approve bool   `json:"approve"`
		Reason  string `json:"reason"`
	}

	msg, _ := json.Marshal(oneBotCommand{
		Action: "set_group_add_request",
		Params: DetailParams{
			Flag:    flag,
			SubType: subType,
			Approve: approve,
			Reason:  reason,
		},
		Echo: "set_group_add_request",
	})

	// 发送数据
	err := p.Instance.EmitTo(p.UUID, msg, socketio.TextMessage)
	if err != nil {
		p.Logger.Error("发送添加好友请求出现异常: %v", err)
		return
	}
}
