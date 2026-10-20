//nolint:testpackage
package dice

import (
	"testing"
	"time"

	"sealdice-core/message"
)

// TestExecuteAndExecuteNewUseSameCore verifies that the legacy string entry
// point and the segment entry point both reach the same command pipeline.
func TestExecuteAndExecuteNewUseSameCore(t *testing.T) {
	t.Run("Execute with string message", func(t *testing.T) {
		d, ep, adapter, cleanup := newExecuteNewTestDice(t)
		defer cleanup()

		d.ImSession.Execute(ep, &Message{
			Platform:    "QQ",
			MessageType: "group",
			GroupID:     "QQ-Group:111",
			GroupName:   "TestGroup",
			Time:        time.Now().Unix(),
			Sender:      SenderBase{UserID: "QQ:999", Nickname: "Tester"},
			Message:     ".r 1d6",
		}, true)

		if _, ok := adapter.waitForMsg(2 * time.Second); !ok {
			t.Fatal("timeout: expected a reply from Execute")
		}
	})

	t.Run("ExecuteNew with segment message", func(t *testing.T) {
		d, ep, adapter, cleanup := newExecuteNewTestDice(t)
		defer cleanup()

		d.ImSession.ExecuteNew(ep, &Message{
			Platform:    "QQ",
			MessageType: "group",
			GroupID:     "QQ-Group:111",
			GroupName:   "TestGroup",
			Time:        time.Now().Unix(),
			Sender:      SenderBase{UserID: "QQ:999", Nickname: "Tester"},
			Segment:     []message.IMessageElement{&message.TextElement{Content: ".r 1d6"}},
		})

		if _, ok := adapter.waitForMsg(2 * time.Second); !ok {
			t.Fatal("timeout: expected a reply from ExecuteNew")
		}
	})
}
