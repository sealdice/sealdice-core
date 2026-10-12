//nolint:testpackage
package dice

import (
	"testing"
	"time"
)

func TestConsumeReplyPrefixOrderAndClear(t *testing.T) {
	ctx := &MsgContext{AliasPrefixText: "A", DelegateText: "D"}

	got := consumeReplyPrefix(ctx, "T")
	const want = "DAT" // Delegate 先加、Alias 后加
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if ctx.AliasPrefixText != "" || ctx.DelegateText != "" {
		t.Fatalf("prefix not cleared: alias=%q delegate=%q", ctx.AliasPrefixText, ctx.DelegateText)
	}
}

func TestReplyToSenderConsumesPrefixSynchronously(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	ctx, msg := newQuitCommandTestContext(t, d, ep, "QQ:9101", "QQ-Group:9101", "G")
	ctx.AliasPrefixText = "A"
	ctx.DelegateText = "D"

	ReplyToSender(ctx, msg, "hi")

	if ctx.AliasPrefixText != "" || ctx.DelegateText != "" {
		t.Fatalf("prefix should be consumed synchronously, got alias=%q delegate=%q",
			ctx.AliasPrefixText, ctx.DelegateText)
	}

	got, ok := adapter.waitForMsg(2 * time.Second)
	if !ok {
		t.Fatal("no message was sent")
	}
	if got != "DAhi" {
		t.Fatalf("sent %q, want %q", got, "DAhi")
	}
}

// 单协程反复设置前缀并调用 ReplyToSender，模拟 handler 连续回复。
func TestReplyToSenderHandlerPrefixRepeatedNoRace(t *testing.T) {
	d, ep, _, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	ctx, msg := newQuitCommandTestContext(t, d, ep, "QQ:9102", "QQ-Group:9102", "G")
	for range 100 {
		ctx.AliasPrefixText = "A"
		ctx.DelegateText = "D"
		ReplyToSender(ctx, msg, "hi")
	}
	time.Sleep(500 * time.Millisecond)
}
