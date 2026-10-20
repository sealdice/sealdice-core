//nolint:testpackage
package dice

import (
	"testing"
	"time"

	"sealdice-core/message"
)

// 已存在群收到新群名时，必须刷新 GroupName 并标记脏数据，否则保存会跳过该群。
func TestExecuteNewUpdatesExistingGroupNameAndMarksDirty(t *testing.T) {
	d, ep, _, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	group := &GroupInfo{
		GroupID:         "QQ-Group:111",
		GroupName:       "OldName",
		Active:          true,
		System:          "coc7", // 避免 SetTempVars 补默认值时 MarkDirty 干扰断言
		DiceIDActiveMap: new(SyncMap[string, bool]),
		DiceIDExistsMap: new(SyncMap[string, bool]),
	}
	group.DiceIDActiveMap.Store(ep.UserID, true)
	// 预先登记端点，排除 DiceIDExistsMap 分支的 MarkDirty，只考察群名变化。
	group.DiceIDExistsMap.Store(ep.UserID, true)
	d.ImSession.ServiceAtNew.Store(group.GroupID, group)
	// 预热扩展懒初始化（它会 MarkDirty），随后清掉脏标记，隔离目标行为。
	_ = group.GetActivatedExtList(d)
	d.DirtyGroups.Delete(group.GroupID)

	d.ImSession.ExecuteNew(ep, newGroupMsg("QQ-Group:111", "QQ:999", "just chatting"))

	if group.GroupName != "TestGroup" {
		t.Fatalf("GroupName = %q, want TestGroup", group.GroupName)
	}
	if _, ok := d.DirtyGroups.Load("QQ-Group:111"); !ok {
		t.Fatal("group with changed name was not marked dirty; new name would not be saved")
	}
}

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
