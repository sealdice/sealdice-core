package dice

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"sealdice-core/message"
	"sealdice-core/utils"
)

const (
	UITestReplySplitLenShort = 300
	UITestReplySplitLenQQ    = 2000
	// SplitLongText treats maxLen <= 0 as no split.
	UITestReplySplitLenUnlimited = 0
)

type HTTPSimpleMessage struct {
	UID         string `json:"uid"`
	Message     string `json:"message"`
	MessageType string `json:"messageType"`
}

type HTTPTestSegment struct {
	Type string            `json:"type"`
	Text string            `json:"text,omitempty"`
	Data map[string]string `json:"data,omitempty"`
}

type HTTPTestMessage struct {
	ID             string            `json:"id"`
	MessageType    string            `json:"messageType"`
	ConversationID string            `json:"conversationId"`
	GroupID        string            `json:"groupId,omitempty"`
	GroupName      string            `json:"groupName,omitempty"`
	Sender         SenderBase        `json:"sender"`
	SenderRole     string            `json:"senderRole,omitempty"`
	AvatarKey      string            `json:"avatarKey,omitempty"`
	IsBot          bool              `json:"isBot"`
	Direction      string            `json:"direction"`
	RawMessage     string            `json:"rawMessage"`
	Segments       []HTTPTestSegment `json:"segments"`
	Timestamp      int64             `json:"timestamp"`
	SplitIndex     int               `json:"splitIndex,omitempty"`
	SplitCount     int               `json:"splitCount,omitempty"`
}

type PlatformAdapterHTTP struct {
	EndPoint      *EndPointInfo
	RecentMessage []HTTPSimpleMessage

	mu                      sync.Mutex
	RecentStructuredMessage []HTTPTestMessage
	OnMessage               func(HTTPTestMessage)
	nextMessageID           uint64
}

func (pa *PlatformAdapterHTTP) SendSegmentToGroup(ctx *MsgContext, groupID string, msg []message.IMessageElement, flag string) {
	pa.captureReply(ctx, "group", groupID, httpTestTextFromElements(msg), httpTestSegmentsFromElements(msg), flag)
}

func (pa *PlatformAdapterHTTP) SendSegmentToPerson(ctx *MsgContext, userID string, msg []message.IMessageElement, flag string) {
	pa.captureReply(ctx, "private", userID, httpTestTextFromElements(msg), httpTestSegmentsFromElements(msg), flag)
}

func (pa *PlatformAdapterHTTP) GetGroupInfoAsync(_ string) {}

func (pa *PlatformAdapterHTTP) Serve() int {
	return 0
}

func (pa *PlatformAdapterHTTP) DoRelogin() bool {
	return false
}

func (pa *PlatformAdapterHTTP) SetEnable(_ bool) {}

func getUITestReplySplitLen(ctx *MsgContext) int {
	if ctx == nil || ctx.UITestReplySplitLen == nil {
		return UITestReplySplitLenQQ
	}
	return *ctx.UITestReplySplitLen
}

func (pa *PlatformAdapterHTTP) SendToPerson(ctx *MsgContext, uid string, text string, flag string) {
	sp := utils.SplitLongText(text, getUITestReplySplitLen(ctx), utils.DefaultSplitPaginationHint)
	pa.captureText(ctx, "private", uid, sp, text, flag)
	pa.EndPoint.Session.OnMessageSend(ctx, &Message{
		MessageType: "private",
		Platform:    "UI",
		Message:     text,
		Sender: SenderBase{
			UserID:   pa.EndPoint.UserID,
			Nickname: pa.EndPoint.Nickname,
		},
	}, flag)
}

func (pa *PlatformAdapterHTTP) SendToGroup(ctx *MsgContext, uid string, text string, flag string) {
	sp := utils.SplitLongText(text, getUITestReplySplitLen(ctx), utils.DefaultSplitPaginationHint)
	pa.captureText(ctx, "group", uid, sp, text, flag)
	pa.EndPoint.Session.OnMessageSend(ctx, &Message{
		MessageType: "group",
		Platform:    "UI",
		Message:     text,
		GroupID:     uid,
		Sender: SenderBase{
			UserID:   pa.EndPoint.UserID,
			Nickname: pa.EndPoint.Nickname,
		},
	}, flag)
}

func (pa *PlatformAdapterHTTP) TakeRecentMessages() []HTTPSimpleMessage {
	pa.mu.Lock()
	defer pa.mu.Unlock()
	items := append([]HTTPSimpleMessage(nil), pa.RecentMessage...)
	pa.RecentMessage = nil
	return items
}

func (pa *PlatformAdapterHTTP) TakeRecentStructuredMessages() []HTTPTestMessage {
	pa.mu.Lock()
	defer pa.mu.Unlock()
	items := append([]HTTPTestMessage(nil), pa.RecentStructuredMessage...)
	pa.RecentStructuredMessage = nil
	return items
}

func (pa *PlatformAdapterHTTP) captureText(ctx *MsgContext, messageType string, targetID string, parts []string, raw string, flag string) {
	for index, sub := range parts {
		pa.capture(ctx, messageType, targetID, sub, parseHTTPTestSegments(sub), flag, index, len(parts))
	}
	_ = raw
}

func (pa *PlatformAdapterHTTP) captureReply(ctx *MsgContext, messageType string, targetID string, raw string, segments []HTTPTestSegment, flag string) {
	pa.capture(ctx, messageType, targetID, raw, segments, flag, 0, 1)
}

func (pa *PlatformAdapterHTTP) capture(ctx *MsgContext, messageType string, targetID string, raw string, segments []HTTPTestSegment, _ string, splitIndex int, splitCount int) {
	if pa == nil {
		return
	}
	groupID := ""
	groupName := ""
	conversationID := "PG-" + targetID
	if messageType == "group" {
		groupID = targetID
		conversationID = targetID
		if ctx != nil && ctx.Group != nil {
			groupName = ctx.Group.GroupName
		}
	}

	sender := SenderBase{UserID: "UI:1000", Nickname: "海豹核心", IsRobot: true}
	if pa.EndPoint != nil {
		if pa.EndPoint.UserID != "" {
			sender.UserID = pa.EndPoint.UserID
		}
		if strings.TrimSpace(pa.EndPoint.Nickname) != "" {
			sender.Nickname = pa.EndPoint.Nickname
		}
	}
	event := HTTPTestMessage{
		ID:             fmt.Sprintf("ui-test-%d", atomic.AddUint64(&pa.nextMessageID, 1)),
		MessageType:    messageType,
		ConversationID: conversationID,
		GroupID:        groupID,
		GroupName:      groupName,
		Sender:         sender,
		IsBot:          true,
		Direction:      "outgoing",
		RawMessage:     raw,
		Segments:       segments,
		Timestamp:      time.Now().UnixMilli(),
		SplitIndex:     splitIndex,
		SplitCount:     splitCount,
	}

	pa.mu.Lock()
	pa.RecentMessage = append(pa.RecentMessage, HTTPSimpleMessage{UID: targetID, Message: raw, MessageType: messageType})
	pa.RecentStructuredMessage = append(pa.RecentStructuredMessage, event)
	publisher := pa.OnMessage
	pa.mu.Unlock()
	if publisher != nil {
		publisher(event)
	}
}

func httpTestTextFromElements(elements []message.IMessageElement) string {
	var builder strings.Builder
	for _, segment := range httpTestSegmentsFromElements(elements) {
		if segment.Type == "text" {
			builder.WriteString(segment.Text)
			continue
		}
		builder.WriteString("[")
		builder.WriteString(segment.Type)
		builder.WriteString("]")
	}
	return builder.String()
}

func httpTestSegmentsFromElements(elements []message.IMessageElement) []HTTPTestSegment {
	segments := make([]HTTPTestSegment, 0, len(elements))
	for _, element := range elements {
		if element == nil {
			continue
		}
		segment := HTTPTestSegment{Data: map[string]string{}}
		switch value := element.(type) {
		case *message.TextElement:
			segment.Type = "text"
			segment.Text = value.Content
		case *message.AtElement:
			segment.Type = "at"
			segment.Data["target"] = value.Target
			segment.Data["isRobot"] = strconv.FormatBool(value.IsRobot)
		case *message.ImageElement:
			segment.Type = "image"
			segment.Data["url"] = value.URL
			if value.File != nil {
				segment.Data["file"] = value.File.File
				if segment.Data["url"] == "" {
					segment.Data["url"] = value.File.URL
				}
			}
		case *message.FileElement:
			segment.Type = "file"
			segment.Data["file"] = value.File
			segment.Data["url"] = value.URL
			segment.Data["contentType"] = value.ContentType
		case *message.RecordElement:
			segment.Type = "record"
			if value.File != nil {
				segment.Data["file"] = value.File.File
				segment.Data["url"] = value.File.URL
			}
		case *message.TTSElement:
			segment.Type = "tts"
			segment.Text = value.Content
		case *message.ReplyElement:
			segment.Type = "reply"
			segment.Data["id"] = value.ReplySeq
			segment.Data["sender"] = value.Sender
			segment.Data["groupId"] = value.GroupID
		case *message.FaceElement:
			segment.Type = "face"
			segment.Data["id"] = value.FaceID
		case *message.PokeElement:
			segment.Type = "poke"
			segment.Data["target"] = value.Target
		case *message.DefaultElement:
			segment.Type = "unsupported"
			segment.Data["rawType"] = value.RawType
		default:
			segment.Type = "unsupported"
		}
		if len(segment.Data) == 0 {
			segment.Data = nil
		}
		segments = append(segments, segment)
	}
	return segments
}

var httpTestCodePattern = regexp.MustCompile(`\[(CQ:[^\]]+|(?:img|图|文本|text|语音|voice|视频|video):[^\]]+)]`)

func parseHTTPTestSegments(raw string) []HTTPTestSegment {
	matches := httpTestCodePattern.FindAllStringIndex(raw, -1)
	if len(matches) == 0 {
		return []HTTPTestSegment{{Type: "text", Text: raw}}
	}

	segments := make([]HTTPTestSegment, 0, len(matches)*2+1)
	last := 0
	for _, match := range matches {
		if match[0] > last {
			segments = append(segments, HTTPTestSegment{Type: "text", Text: raw[last:match[0]]})
		}
		token := raw[match[0]:match[1]]
		segments = append(segments, parseHTTPTestCode(token))
		last = match[1]
	}
	if last < len(raw) {
		segments = append(segments, HTTPTestSegment{Type: "text", Text: raw[last:]})
	}
	return segments
}

func parseHTTPTestCode(token string) HTTPTestSegment {
	if strings.HasPrefix(token, "[CQ:") {
		inner := strings.TrimSuffix(strings.TrimPrefix(token, "[CQ:"), "]")
		parts := strings.Split(inner, ",")
		segment := HTTPTestSegment{Type: "unsupported", Data: map[string]string{}}
		if len(parts) > 0 {
			segment.Type = parts[0]
		}
		for _, item := range parts[1:] {
			key, value, ok := strings.Cut(item, "=")
			if ok {
				segment.Data[key] = value
			}
		}
		return segment
	}

	inner := strings.TrimSuffix(strings.TrimPrefix(token, "["), "]")
	key, value, _ := strings.Cut(inner, ":")
	segmentType := "text"
	switch key {
	case "img", "图":
		segmentType = "image"
	case "语音", "voice":
		segmentType = "record"
	case "视频", "video":
		segmentType = "video"
	}
	return HTTPTestSegment{Type: segmentType, Data: map[string]string{"file": strings.TrimSpace(value)}}
}

func (pa *PlatformAdapterHTTP) SendFileToPerson(ctx *MsgContext, uid string, path string, flag string) {
	pa.SendToPerson(ctx, uid, fmt.Sprintf("[尝试发送文件: %s，但不支持]", filepath.Base(path)), flag)
}

func (pa *PlatformAdapterHTTP) SendFileToGroup(ctx *MsgContext, uid string, path string, flag string) {
	pa.SendToGroup(ctx, uid, fmt.Sprintf("[尝试发送文件: %s，但不支持]", filepath.Base(path)), flag)
}

func (pa *PlatformAdapterHTTP) QuitGroup(_ *MsgContext, _ string) {}

func (pa *PlatformAdapterHTTP) SetGroupCardName(_ *MsgContext, _ string) {}

func (pa *PlatformAdapterHTTP) MemberBan(_ string, _ string, _ int64) {}

func (pa *PlatformAdapterHTTP) MemberKick(_ string, _ string) {}

func (pa *PlatformAdapterHTTP) EditMessage(_ *MsgContext, _, _ string) {}

func (pa *PlatformAdapterHTTP) RecallMessage(_ *MsgContext, _ string) {}
