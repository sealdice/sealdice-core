package dice

import (
	"fmt"
	"path/filepath"

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

type PlatformAdapterHTTP struct {
	EndPoint      *EndPointInfo
	RecentMessage []HTTPSimpleMessage
}

func (pa *PlatformAdapterHTTP) SendSegmentToGroup(ctx *MsgContext, groupID string, msg []message.IMessageElement, flag string) {
	logText := MessageSegmentsToCompatibilityText(msg)
	sp := utils.SplitLongText(logText, getUITestReplySplitLen(ctx), utils.DefaultSplitPaginationHint)
	for _, sub := range sp {
		pa.RecentMessage = append(pa.RecentMessage, HTTPSimpleMessage{groupID, sub, "group"})
	}
	pa.EndPoint.Session.OnMessageSend(ctx, &Message{
		MessageType: "group",
		Platform:    "UI",
		Message:     logText,
		Segment:     msg,
		GroupID:     "UI-Group:2001",
		Sender: SenderBase{
			UserID:   pa.EndPoint.UserID,
			Nickname: pa.EndPoint.Nickname,
		},
	}, flag)
}

func (pa *PlatformAdapterHTTP) SendSegmentToPerson(ctx *MsgContext, userID string, msg []message.IMessageElement, flag string) {
	logText := MessageSegmentsToCompatibilityText(msg)
	sp := utils.SplitLongText(logText, getUITestReplySplitLen(ctx), utils.DefaultSplitPaginationHint)
	for _, sub := range sp {
		pa.RecentMessage = append(pa.RecentMessage, HTTPSimpleMessage{userID, sub, "private"})
	}
	pa.EndPoint.Session.OnMessageSend(ctx, &Message{
		MessageType: "private",
		Platform:    "UI",
		Message:     logText,
		Segment:     msg,
		Sender: SenderBase{
			UserID:   pa.EndPoint.UserID,
			Nickname: pa.EndPoint.Nickname,
		},
	}, flag)
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
	pa.SendSegmentToPerson(ctx, uid, parseOutboundText(text), flag)
}

func (pa *PlatformAdapterHTTP) SendToGroup(ctx *MsgContext, uid string, text string, flag string) {
	pa.SendSegmentToGroup(ctx, uid, parseOutboundText(text), flag)
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
