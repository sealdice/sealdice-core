package dice

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	socketio "github.com/PaienNate/pineutil/evsocket"
	"github.com/bytedance/sonic"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"go.uber.org/zap"

	emitter "sealdice-core/dice/utils/onebot"
	"sealdice-core/dice/utils/onebot/schema"
	"sealdice-core/message"
)

// ONEBOT事件对应码表
// OnebotEventPostTypeCode
const (
	OnebotEventPostTypeMessage = "onebot_message"
	// OnebotEventPostTypeCodeMessageSent 是bot发出的消息
	OnebotEventPostTypeMessageSent = "onebot_message_sent"
	OnebotEventPostTypeRequest     = "onebot_request"
	OnebotEventPostTypeNotice      = "onebot_notice"
	OnebotEventPostTypeMetaEvent   = "onebot_meta_event"
)

const (
	OnebotReceiveMessage = "onebot_echo"
)

// serveOnebotEvent 消息分发函数。该函数将所有的传入参数，全部转换为array模式
func (p *PlatformAdapterOnebot) serveOnebotEvent(ep *socketio.EventPayload) {
	// p.logger.Infof("Message event - User: %s - Message: %s", ep.Kws.GetStringAttribute("user_id"), string(ep.Data))
	if !gjson.ValidBytes(ep.Data) {
		return
	}
	resp := gjson.ParseBytes(ep.Data)
	if resp.Get("self_id").Int() != 0 {
		p.once.Do(func() {
			p.emitterChan = make(chan emitter.Response[json.RawMessage], 32)
			p.sendEmitter = emitter.NewEVEmitter(ep.Kws, resp.Get("self_id").Int(), p.emitterChan)
		})
	}

	// 解析是string还是array
	// TODO: 不知道是不是通过这种方式判断是string或者array的
	switch resp.Get("message").Type {
	case gjson.String:
		p.wsmode = "string"
	case gjson.JSON:
		p.wsmode = "array"
	default:
		p.wsmode = "string"
	}
	if p.wsmode == "string" {
		resp2, err := string2array(resp)
		if err != nil {
			p.logger.Warnf("消息转换为array异常 %s 未能正确处理", resp.String())
			return
		}
		resp = resp2
	}

	// 将数据转换为对应的事件event
	eventType := resp.Get("post_type").String()
	if eventType != "" {
		// 分发事件
		eventType = fmt.Sprintf("onebot_%s", eventType)
		ep.Kws.Fire(eventType, []byte(resp.String()))
	} else {
		// 如果没有post_type，说明不是上报信息，而是API的返回信息
		ep.Kws.Fire(OnebotReceiveMessage, []byte(resp.String()))
	}
}

func (p *PlatformAdapterOnebot) onOnebotMessageEvent(ep *socketio.EventPayload) {
	// 收到普通消息的时候：执行ExecuteNew函数
	msg, err := arrayByte2SealdiceMessage(p.logger, ep.Data)
	if err != nil {
		p.logger.Errorf("收到消息但无法进行处理，原因为 %s", err)
		return
	}
	p.Session.ExecuteNew(p.EndPoint, msg)
}

func (p *PlatformAdapterOnebot) onOnebotMetaDataEvent(ep *socketio.EventPayload) {

}

func FormatOnebotDiceIDQQ(diceQQ string) string {
	return fmt.Sprintf("QQ:%s", diceQQ)
}

func FormatOnebotDiceIDQQGroup(diceQQ string) string {
	return fmt.Sprintf("QQ-Group:%s", diceQQ)
}

type MessageQQOBBase struct {
	MessageID     int64           `json:"message_id"`   // QQ信息此类型为int64，频道中为string
	MessageType   string          `json:"message_type"` // Group
	Sender        *Sender         `json:"sender"`       // 发送者
	RawMessage    string          `json:"raw_message"`
	Time          int64           `json:"time"` // 发送时间
	MetaEventType string          `json:"meta_event_type"`
	OperatorID    json.RawMessage `json:"operator_id"`  // 操作者帐号
	GroupID       json.RawMessage `json:"group_id"`     // 群号
	PostType      string          `json:"post_type"`    // 上报类型，如group、notice
	RequestType   string          `json:"request_type"` // 请求类型，如group
	SubType       string          `json:"sub_type"`     // 子类型，如add invite
	Flag          string          `json:"flag"`         // 请求 flag, 在调用处理请求的 API 时需要传入
	NoticeType    string          `json:"notice_type"`
	UserID        json.RawMessage `json:"user_id"`
	SelfID        json.RawMessage `json:"self_id"`
	Duration      int64           `json:"duration"`
	Comment       string          `json:"comment"`
	TargetID      json.RawMessage `json:"target_id"`

	Data *struct {
		// 个人信息
		Nickname string          `json:"nickname"`
		UserID   json.RawMessage `json:"user_id"`

		// 群信息
		GroupID         json.RawMessage `json:"group_id"`          // 群号
		GroupCreateTime uint32          `json:"group_create_time"` // 群号
		MemberCount     int64           `json:"member_count"`
		GroupName       string          `json:"group_name"`
		MaxMemberCount  int32           `json:"max_member_count"`

		// 群成员信息
		Card string `json:"card"`
	} `json:"data"`
	Retcode int64 `json:"retcode"`
	// Status string `json:"status"`
	Echo json.RawMessage `json:"echo"` // 声明类型而不是interface的原因是interface下数字不能正确转换

	Msg string `json:"msg"`
	// Status  interface{} `json:"status"`
	Wording string `json:"wording"`
}

type MessageOBQQ struct {
	MessageQQOBBase
}

func (msgQQ *MessageOBQQ) toStdMessage() *Message {
	msg := new(Message)
	msg.Time = msgQQ.Time
	msg.MessageType = msgQQ.MessageType
	msg.RawID = msgQQ.MessageID
	msg.Platform = "QQ"

	if msg.MessageType == "" {
		msg.MessageType = "private"
	}

	if msgQQ.Data != nil && len(msgQQ.Data.GroupID) > 0 {
		msg.GroupID = FormatOnebotDiceIDQQGroup(string(msgQQ.Data.GroupID))
	}
	if string(msgQQ.GroupID) != "" {
		if msg.MessageType == "private" {
			msg.MessageType = "group"
		}
		msg.GroupID = FormatOnebotDiceIDQQGroup(string(msgQQ.GroupID))
	}
	if msgQQ.Sender != nil {
		msg.Sender.Nickname = msgQQ.Sender.Nickname
		if msgQQ.Sender.Card != "" {
			msg.Sender.Nickname = msgQQ.Sender.Card
		}
		msg.Sender.GroupRole = msgQQ.Sender.Role
		msg.Sender.UserID = FormatOnebotDiceIDQQ(string(msgQQ.Sender.UserID))
	}
	return msg
}

func arrayByte2SealdiceMessage(log *zap.SugaredLogger, raw []byte) (*Message, error) {
	// 不合法的信息体
	if !gjson.ValidBytes(raw) {
		log.Warn("无法解析 onebot11 字段:", raw)
		return nil, errors.New("解析失败")
	}
	var obMsg MessageOBQQ
	// 原版本转换为gjson对象
	parseContent := gjson.ParseBytes(raw)
	err := sonic.Unmarshal(raw, &obMsg)
	if err != nil {
		return nil, err
	}
	m := obMsg.toStdMessage()
	arrayContent := parseContent.Get("message").Array()
	seg := make([]message.IMessageElement, len(arrayContent))
	cqMessage := strings.Builder{}
	for _, i := range arrayContent {
		// 使用String()方法，如果为空，会自动产生空字符串
		typeStr := i.Get("type").String()
		dataObj := i.Get("data")
		switch typeStr {
		case "text":
			rawTxt := dataObj.Get("text").String()
			seg = append(seg, &message.TextElement{
				Content: rawTxt,
			})
			cqMessage.WriteString(rawTxt)
		case "image":
			rawImg := &message.ImageElement{
				File: &message.FileElement{
					URL: dataObj.Get("file").String(),
				},
			}
			// 兼容NC情况, 此时file字段只有文件名, 完整URL在url字段
			if !hasURLScheme(dataObj.Get("file").String()) && hasURLScheme(dataObj.Get("url").String()) {
				rawImg.File.URL = dataObj.Get("url").String()
				cqMessage.WriteString(fmt.Sprintf("[CQ:image,file=%v]", dataObj.Get("url").String()))
			} else {
				rawImg.File.URL = dataObj.Get("file").String()
				cqMessage.WriteString(fmt.Sprintf("[CQ:image,file=%v]", dataObj.Get("file").String()))
			}
			seg = append(seg, rawImg)
		case "face":
			cqMessage.WriteString(fmt.Sprintf("[CQ:face,id=%v]", dataObj.Get("id").String()))
			seg = append(seg, &message.FaceElement{FaceID: dataObj.Get("id").String()})
		case "record":
			recordRaw := message.RecordElement{File: &message.FileElement{
				URL: "",
			}}
			// 兼容NC情况, 此时file字段只有文件名, 完整路径在path字段
			if !hasURLScheme(dataObj.Get("file").String()) && dataObj.Get("path").String() != "" {
				recordRaw.File.URL = dataObj.Get("path").String()
				cqMessage.WriteString(fmt.Sprintf("[CQ:record,file=%v]", dataObj.Get("path").String()))
			} else {
				recordRaw.File.URL = dataObj.Get("file").String()
				cqMessage.WriteString(fmt.Sprintf("[CQ:record,file=%v]", dataObj.Get("file").String()))
			}
			seg = append(seg, &recordRaw)
		case "at":
			cqMessage.WriteString(fmt.Sprintf("[CQ:at,qq=%v]", dataObj.Get("qq").String()))
			seg = append(seg, &message.AtElement{Target: dataObj.Get("qq").String()})
		case "poke":
			cqMessage.WriteString("[CQ:poke]")
			seg = append(seg, &message.PokeElement{})
		case "reply":
			cqMessage.WriteString(fmt.Sprintf("[CQ:reply,id=%v]", dataObj.Get("id").String()))
			seg = append(seg, &message.ReplyElement{
				ReplySeq: dataObj.Get("id").String(),
			})
		}
	}
	// 获取Message
	m.Message = cqMessage.String()
	m.Message = strings.ReplaceAll(m.Message, "&#91;", "[")
	m.Message = strings.ReplaceAll(m.Message, "&#93;", "]")
	m.Message = strings.ReplaceAll(m.Message, "&amp;", "&")
	// 获取Segment
	m.Segment = seg
	return m, nil
}

// 将OB11的Array数据转换为string字符串
func array2string(parseContent gjson.Result) (gjson.Result, error) {
	arrayContent := parseContent.Get("message").Array()
	cqMessage := strings.Builder{}

	for _, i := range arrayContent {
		typeStr := i.Get("type").String()
		dataObj := i.Get("data")
		switch typeStr {
		case "text":
			cqMessage.WriteString(dataObj.Get("text").String())
		case "image":
			// 兼容NC情况, 此时file字段只有文件名, 完整URL在url字段
			if !hasURLScheme(dataObj.Get("file").String()) && hasURLScheme(dataObj.Get("url").String()) {
				cqMessage.WriteString(fmt.Sprintf("[CQ:image,file=%v]", dataObj.Get("url").String()))
			} else {
				cqMessage.WriteString(fmt.Sprintf("[CQ:image,file=%v]", dataObj.Get("file").String()))
			}
		case "face":
			// 兼容四叶草，移除 .(string)。自动获取的信息表示此类型为 float64，这是go解析的问题
			cqMessage.WriteString(fmt.Sprintf("[CQ:face,id=%v]", dataObj.Get("id").String()))
		case "record":
			// 兼容NC情况, 此时file字段只有文件名, 完整路径在path字段
			if !hasURLScheme(dataObj.Get("file").String()) && dataObj.Get("path").String() != "" {
				cqMessage.WriteString(fmt.Sprintf("[CQ:record,file=%v]", dataObj.Get("path").String()))
			} else {
				cqMessage.WriteString(fmt.Sprintf("[CQ:record,file=%v]", dataObj.Get("file").String()))
			}
		case "at":
			cqMessage.WriteString(fmt.Sprintf("[CQ:at,qq=%v]", dataObj.Get("qq").String()))
		case "poke":
			cqMessage.WriteString(fmt.Sprintf("[CQ:poke,qq=%v]", dataObj.Get("qq").String()))
		case "reply":
			cqMessage.WriteString(fmt.Sprintf("[CQ:reply,id=%v]", dataObj.Get("id").String()))
		}
	}
	// 赋值对应的Message
	tempStr, err := sjson.Set(parseContent.String(), "message", cqMessage.String())
	if err != nil {
		return gjson.Result{}, err
	}
	return gjson.Parse(tempStr), nil
}

// 将CQ码字符串消息转换为OB11 Array格式
func string2array(parseContent gjson.Result) (gjson.Result, error) {
	messageStr := parseContent.Get("message").String()
	var messageArray []map[string]interface{}

	// 使用正则表达式匹配CQ码和普通文本
	re := regexp.MustCompile(`(\[CQ:\w+,[^\]]+\])|([^\[\]]+)`)
	matches := re.FindAllStringSubmatch(messageStr, -1)

	for _, match := range matches {
		if match[1] != "" {
			// 处理CQ码
			cqCode := match[1]
			cqType := strings.TrimPrefix(strings.Split(cqCode, ",")[0], "[CQ:")
			cqType = strings.TrimSuffix(cqType, "]")

			// 解析CQ码参数
			params := make(map[string]string)
			paramPairs := strings.Split(strings.TrimSuffix(strings.Split(cqCode, ",")[1], "]"), ",")
			for _, pair := range paramPairs {
				kv := strings.SplitN(pair, "=", 2)
				if len(kv) == 2 {
					params[kv[0]] = kv[1]
				}
			}

			// 转换为OB11格式
			item := map[string]interface{}{
				"type": cqType,
				"data": params,
			}
			messageArray = append(messageArray, item)
		} else if match[2] != "" {
			// 处理普通文本
			item := map[string]interface{}{
				"type": "text",
				"data": map[string]string{
					"text": match[2],
				},
			}
			messageArray = append(messageArray, item)
		}
	}

	// 赋值对应的Message
	tempStr, err := sjson.Set(parseContent.String(), "message", messageArray)
	if err != nil {
		return gjson.Result{}, err
	}
	return gjson.Parse(tempStr), nil
}

func convertSealMsgToMessageChain(msg []message.IMessageElement) (schema.MessageChain, string) {
	cqMessage := strings.Builder{}
	rawMsg := schema.MessageChain{}
	for _, v := range msg {
		switch v.Type() {
		case message.At:
			res, ok := v.(*message.AtElement)
			if !ok {
				continue
			}
			rawMsg.At(res.Target)
			cqMessage.WriteString(fmt.Sprintf("[CQ:at,qq=%v]", res.Target))
		case message.Text:
			res, ok := v.(*message.TextElement)
			if !ok {
				continue
			}
			rawMsg = rawMsg.Text(res.Content)
			cqMessage.WriteString(res.Content)
		case message.Face:
			res, ok := v.(*message.FaceElement)
			if !ok {
				continue
			}
			rawMsg = rawMsg.Face(res.FaceID)
			cqMessage.WriteString(fmt.Sprintf("[CQ:face,id=%v]", res.FaceID))
		case message.File:
			res, ok := v.(*message.FileElement)
			if !ok {
				continue
			}
			rawMsg = rawMsg.File(res.File)
			cqMessage.WriteString(fmt.Sprintf("[CQ:file,file=%v]", res.File))
		case message.Image:
			res, ok := v.(*message.ImageElement)
			if !ok {
				continue
			}
			if res.URL != "" {
				rawMsg = rawMsg.Image(res.URL)
			} else {
				// 对，对吗？
				rawMsg = rawMsg.Image(res.File.URL)
			}
			cqMessage.WriteString(fmt.Sprintf("[CQ:image,file=%v]", res.URL))
		case message.Record:
			res, ok := v.(*message.RecordElement)
			if !ok {
				continue
			}
			rawMsg = rawMsg.Record(res.File.URL)
			cqMessage.WriteString(fmt.Sprintf("[CQ:record,file=%v]", res.File.URL))
		case message.Reply:
			res, ok := v.(*message.ReplyElement)
			if !ok {
				continue
			}
			parseInt, err := strconv.Atoi(res.ReplySeq)
			if err != nil {
				continue
			}
			rawMsg = rawMsg.Reply(parseInt)
			cqMessage.WriteString(fmt.Sprintf("[CQ:reply,id=%v]", parseInt))
		case message.TTS:
			res, ok := v.(*message.TTSElement)
			if !ok {
				continue
			}
			m := map[string]string{
				"text": res.Content,
			}
			marshal, err := sonic.Marshal(m)
			if err != nil {
				continue
			}
			rawMsg = rawMsg.Append(schema.Message{
				Type: "tts",
				Data: marshal,
			})
			cqMessage.WriteString(fmt.Sprintf("[CQ:tts,text=%v]", res.Content))
		case message.Poke:
			res, ok := v.(*message.PokeElement)
			if !ok {
				continue
			}
			m := map[string]string{
				"qq": res.Target,
			}
			marshal, err := sonic.Marshal(m)
			if err != nil {
				continue
			}
			rawMsg = rawMsg.Append(schema.Message{
				Type: "poke",
				Data: marshal,
			})
			cqMessage.WriteString(fmt.Sprintf("[CQ:poke,qq=%v]", res.Target))
		}
	}
	return rawMsg, cqMessage.String()
}

func ExtractQQEmitterUserID(id string) int64 {
	if strings.HasPrefix(id, "QQ:") {
		atoi, _ := strconv.ParseInt(id[len("QQ:"):], 10, 64)
		return atoi
	}
	return 0
}

func ExtractQQEmitterGroupID(id string) int64 {
	if strings.HasPrefix(id, "QQ-Group:") {
		atoi, _ := strconv.ParseInt(id[len("QQ-Group:"):], 10, 64)
		return atoi
	}
	return 0
}
