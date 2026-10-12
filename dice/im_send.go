package dice

import (
	"runtime/debug"

	"sealdice-core/dice/sender"
	"sealdice-core/logger"
)

var imSendQueue = sender.New()

func CloseIMSendQueue() {
	imSendQueue.Close()
}

func consumeReplyPrefix(ctx *MsgContext, text string) string {
	prefix := ""
	if ctx.AliasPrefixText != "" {
		prefix = ctx.AliasPrefixText
		ctx.AliasPrefixText = ""
	}
	if ctx.DelegateText != "" {
		prefix = ctx.DelegateText + prefix
		ctx.DelegateText = ""
	}
	if prefix == "" {
		return text
	}
	return prefix + text
}

func enqueueReply(ctx *MsgContext, msg *Message, text string, noCheck bool) {
	text = consumeReplyPrefix(ctx, text)
	snap := ctx.ShallowCopy()
	key := ""
	if ctx.EndPoint != nil {
		key = ctx.EndPoint.UserID
	}
	imSendQueue.Enqueue(key, func() {
		if noCheck {
			safeReplyRawNoCheck(snap, msg, text, "")
		} else {
			safeReplyRaw(snap, msg, text, "")
		}
	})
}

func safeReplyRaw(ctx *MsgContext, msg *Message, text string, flag string) {
	defer recoverReply()
	ReplyToSenderRaw(ctx, msg, text, flag)
}

func safeReplyRawNoCheck(ctx *MsgContext, msg *Message, text string, flag string) {
	defer recoverReply()
	replyToSenderRawNoCheck(ctx, msg, text, flag)
}

func recoverReply() {
	if r := recover(); r != nil {
		logger.M().Errorf("发送消息时崩溃: %v \n堆栈: %s", r, string(debug.Stack()))
	}
}
