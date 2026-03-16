//nolint:testpackage
package dice

// Smoke tests for IMSession.ExecuteNew – the central message-dispatch function.
//
// This file builds a minimal but real Dice instance (commands + extensions +
// text map all loaded; SQLite operator pointing at a temp dir) together with a
// mockPlatformAdapter that captures every outbound message.  Each test then
// calls ExecuteNew with a crafted Message and asserts the expected behaviour.
//
// Why package dice (internal test)?
//   - Several functions used for setup (registerCoreCommands,
//     setupBaseTextTemplate, loadTextTemplate, initVerify, NewConfig) are
//     unexported but needed to create a realistic test environment.
//   - Tests in the same package may call them directly without any extra
//     export shims.

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	sealdiceLogger "sealdice-core/logger"
	"sealdice-core/message"
	"sealdice-core/utils/constant"
)

// ---------------------------------------------------------------------------
// mockPlatformAdapter
// ---------------------------------------------------------------------------

// mockPlatformAdapter implements PlatformAdapter, recording every message that
// the bot tries to send.  A buffered channel makes it easy to wait for async
// replies from PreTriggerCommand.
type mockPlatformAdapter struct {
	mu         sync.Mutex
	groupMsgs  []string
	personMsgs []string
	quitGroups []string
	msgCh      chan string
}

// compile-time check
var _ PlatformAdapter = (*mockPlatformAdapter)(nil)

func newMockPlatformAdapter() *mockPlatformAdapter {
	return &mockPlatformAdapter{msgCh: make(chan string, 32)}
}

// waitForMsg blocks until a message arrives or the timeout elapses.
func (m *mockPlatformAdapter) waitForMsg(timeout time.Duration) (string, bool) {
	select {
	case msg := <-m.msgCh:
		return msg, true
	case <-time.After(timeout):
		return "", false
	}
}

func (m *mockPlatformAdapter) SendToGroup(_ *MsgContext, _ string, text string, _ string) {
	m.mu.Lock()
	m.groupMsgs = append(m.groupMsgs, text)
	m.mu.Unlock()
	select {
	case m.msgCh <- text:
	default:
	}
}

func (m *mockPlatformAdapter) SendToPerson(_ *MsgContext, _ string, text string, _ string) {
	m.mu.Lock()
	m.personMsgs = append(m.personMsgs, text)
	m.mu.Unlock()
	select {
	case m.msgCh <- text:
	default:
	}
}

// Remaining PlatformAdapter methods – all no-ops for the smoke test.
func (m *mockPlatformAdapter) Serve() int       { return 0 }
func (m *mockPlatformAdapter) DoRelogin() bool  { return true }
func (m *mockPlatformAdapter) SetEnable(_ bool) {}
func (m *mockPlatformAdapter) QuitGroup(_ *MsgContext, groupID string) {
	m.mu.Lock()
	m.quitGroups = append(m.quitGroups, groupID)
	m.mu.Unlock()
}
func (m *mockPlatformAdapter) SetGroupCardName(_ *MsgContext, _ string) {}
func (m *mockPlatformAdapter) MemberBan(_ string, _ string, _ int64)    {}
func (m *mockPlatformAdapter) MemberKick(_ string, _ string)            {}
func (m *mockPlatformAdapter) GetGroupInfoAsync(_ string)               {}
func (m *mockPlatformAdapter) EditMessage(_ *MsgContext, _, _ string)   {}
func (m *mockPlatformAdapter) RecallMessage(_ *MsgContext, _ string)    {}
func (m *mockPlatformAdapter) SendSegmentToGroup(_ *MsgContext, _ string, _ []message.IMessageElement, _ string) {
}
func (m *mockPlatformAdapter) SendSegmentToPerson(_ *MsgContext, _ string, _ []message.IMessageElement, _ string) {
}
func (m *mockPlatformAdapter) SendFileToPerson(_ *MsgContext, _ string, _ string, _ string) {}
func (m *mockPlatformAdapter) SendFileToGroup(_ *MsgContext, _ string, _ string, _ string)  {}

// ---------------------------------------------------------------------------
// mockDatabaseOperator
// ---------------------------------------------------------------------------

// mockDatabaseOperator wraps a single SQLite file that has no pre-existing
// tables.  All service calls that try to read player / attr data will receive
// "table not found" errors and gracefully fall back to creating in-memory
// objects, which is all a smoke test needs.
type mockDatabaseOperator struct {
	db *gorm.DB
}

func newMockDatabaseOperator(dbPath string) (*mockDatabaseOperator, error) {
	db, err := openTestGormDB(dbPath)
	if err != nil {
		return nil, err
	}
	return &mockDatabaseOperator{db: db}, nil
}

func (m *mockDatabaseOperator) Init(_ context.Context) error           { return nil }
func (m *mockDatabaseOperator) Type() string                           { return "mock-sqlite" }
func (m *mockDatabaseOperator) DBCheck()                               {}
func (m *mockDatabaseOperator) GetDataDB(_ constant.DBMode) *gorm.DB   { return m.db }
func (m *mockDatabaseOperator) GetLogDB(_ constant.DBMode) *gorm.DB    { return m.db }
func (m *mockDatabaseOperator) GetCensorDB(_ constant.DBMode) *gorm.DB { return m.db }
func (m *mockDatabaseOperator) Close() {
	if sqlDB, err := m.db.DB(); err == nil {
		_ = sqlDB.Close()
	}
}

// ---------------------------------------------------------------------------
// newExecuteNewTestDice – minimal Dice setup for smoke testing
// ---------------------------------------------------------------------------

// newExecuteNewTestDice constructs a Dice with enough infrastructure for
// ExecuteNew to run end-to-end: commands are registered, text templates are
// loaded, the ban list is initialised, and the AttrsManager goroutine is
// started.
//
// The returned cleanup function must be deferred by every caller to stop the
// AttrsManager background goroutine and close the database.
func newExecuteNewTestDice(t *testing.T) (*Dice, *EndPointInfo, *mockPlatformAdapter, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	for _, sub := range []string{"configs", "extensions", "scripts", "extra", "log-exports"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, sub), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", sub, err)
		}
	}

	mockDB, err := newMockDatabaseOperator(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("newMockDatabaseOperator: %v", err)
	}

	// Minimal DiceManager – Cron is created but NOT started to avoid background
	// goroutines that would trigger goleak.
	dm := &DiceManager{Cron: cron.New()}

	d := &Dice{
		BaseConfig: BaseConfig{
			DataDir: tmpDir,
			Name:    "test",
		},
		Logger:        sealdiceLogger.M(),
		LogWriter:     sealdiceLogger.NewUIWriter(),
		DBOperator:    mockDB,
		CmdMap:        CmdMapCls{},
		ExtRegistry:   new(SyncMap[string, *ExtInfo]),
		GameSystemMap: new(SyncMap[string, *GameSystemTemplate]),
		DirtyGroups:   new(SyncMap[string, int64]),
		CocExtraRules: map[int]*CocRuleInfo{},
		// Cron is not started to keep the goroutine count predictable.
		Cron:   cron.New(),
		Parent: dm,
	}
	dm.Dice = []*Dice{d}

	// Config – use defaults so CommandCompatibleMode, BanList, rate-limit
	// settings, and other fields are properly initialised.
	d.Config = NewConfig(d)
	// Disable the per-message QQ sleep so tests are fast.
	d.Config.MessageDelayRangeEnd = 0

	// Command prefix used by CommandParseNew.
	d.CommandPrefix = DefaultConfig.CommandPrefix

	// IMSession
	d.ImSession = &IMSession{}
	d.ImSession.Parent = d
	d.ImSession.ServiceAtNew = new(SyncMap[string, *GroupInfo])

	// AttrsManager – starts one background goroutine; stopped in cleanup.
	d.AttrsManager = &AttrsManager{}
	d.AttrsManager.Init(d)

	// ConfigManager
	d.ConfigManager = NewConfigManager(filepath.Join(tmpDir, "configs", "plugin-configs.json"))
	_ = d.ConfigManager.Load()

	// Register all core commands and built-in extensions.
	initVerify()
	d.registerCoreCommands()
	d.RegisterBuiltinExt()

	// Text templates (setupBaseTextTemplate initialises d.TextMapRaw).
	setupBaseTextTemplate(d)
	// loadTextTemplate is a no-op when the file doesn't exist, which is fine.
	loadTextTemplate(d, "configs/text-template.yaml")
	d.GenerateTextMap()

	d.ApplyExtDefaultSettings()
	d.IsAlreadyLoadConfig = true

	// Mock platform adapter + endpoint
	adapter := newMockPlatformAdapter()
	ep := &EndPointInfo{
		EndPointInfoBase: EndPointInfoBase{
			ID:       "test-ep-1",
			UserID:   "QQ:100000",
			Nickname: "TestBot",
			Platform: "QQ",
			Enable:   true,
		},
		Adapter: adapter,
	}
	ep.Session = d.ImSession
	d.ImSession.EndPoints = []*EndPointInfo{ep}

	cleanup := func() {
		// Stop package-owned workers first.
		d.AttrsManager.Stop()

		// Close help search engine if it was initialized (bleve workers).
		if d.Parent != nil && d.Parent.Help != nil {
			d.Parent.Help.Close()
		}

		// Release ants default pool (stops purge/ticktock goroutines).
		ants.Release()

		mockDB.Close()

		// Give async workers a short window to observe cancellation and exit.
		time.Sleep(2 * time.Second)
	}
	return d, ep, adapter, cleanup
}

// ---------------------------------------------------------------------------
// Message helpers
// ---------------------------------------------------------------------------

func newGroupMsg(groupID, senderID, text string) *Message {
	return &Message{
		Platform:    "QQ",
		MessageType: "group",
		GroupID:     groupID,
		GroupName:   "TestGroup",
		Time:        time.Now().Unix(),
		Sender: SenderBase{
			UserID:   senderID,
			Nickname: "Tester",
		},
		Message: text,
		Segment: []message.IMessageElement{
			&message.TextElement{Content: text},
		},
	}
}

func newPrivateMsg(senderID, text string) *Message {
	return &Message{
		Platform:    "QQ",
		MessageType: "private",
		Time:        time.Now().Unix(),
		Sender: SenderBase{
			UserID:   senderID,
			Nickname: "Tester",
		},
		Message: text,
		Segment: []message.IMessageElement{
			&message.TextElement{Content: text},
		},
	}
}

// ---------------------------------------------------------------------------
// Smoke Tests
// ---------------------------------------------------------------------------

// TestExecuteNew_GroupCommand_Roll verifies that a dice-roll command sent to a
// group causes the bot to produce at least one reply via the adapter.
func TestExecuteNew_GroupCommand_Roll(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	d.ImSession.ExecuteNew(ep, newGroupMsg("QQ-Group:111", "QQ:999", ".r 1d6"))

	_, ok := adapter.waitForMsg(2 * time.Second)
	if !ok {
		t.Fatal("timeout: expected a reply to '.r 1d6' but none arrived")
	}
}

// TestExecuteNew_GroupCommand_Roll_NoArgs verifies a bare .r command still
// produces a reply (uses the group's default dice expression).
func TestExecuteNew_GroupCommand_Roll_NoArgs(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	d.ImSession.ExecuteNew(ep, newGroupMsg("QQ-Group:111", "QQ:999", ".r"))

	_, ok := adapter.waitForMsg(2 * time.Second)
	if !ok {
		t.Fatal("timeout: expected a reply to bare '.r' command")
	}
}

// TestExecuteNew_PrivateCommand_Roll verifies that a private message with a
// dice command produces a reply to the sender.
func TestExecuteNew_PrivateCommand_Roll(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	d.ImSession.ExecuteNew(ep, newPrivateMsg("QQ:999", ".r 1d6"))

	_, ok := adapter.waitForMsg(2 * time.Second)
	if !ok {
		t.Fatal("timeout: expected a reply to private '.r 1d6'")
	}
}

// TestExecuteNew_GroupMessage_NonCommand verifies that a plain chat message
// (no command prefix) does NOT produce a bot reply.
func TestExecuteNew_GroupMessage_NonCommand(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	d.ImSession.ExecuteNew(ep, newGroupMsg("QQ-Group:111", "QQ:999", "just chatting"))

	select {
	case unexpected := <-adapter.msgCh:
		t.Errorf("unexpected reply to non-command message: %q", unexpected)
	case <-time.After(400 * time.Millisecond):
		// expected: no reply
	}
}

// TestExecuteNew_GroupInfo_Created verifies that ExecuteNew automatically
// creates and stores a GroupInfo entry for a previously-unseen group.
func TestExecuteNew_GroupInfo_Created(t *testing.T) {
	d, ep, _, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	const groupID = "QQ-Group:654321"
	d.ImSession.ExecuteNew(ep, newGroupMsg(groupID, "QQ:111", ".r"))

	// Give the async goroutine time to complete.
	time.Sleep(200 * time.Millisecond)

	groupInfo, ok := d.ImSession.ServiceAtNew.Load(groupID)
	if !ok {
		t.Fatal("GroupInfo was not created for the new group")
	}
	if groupInfo.GroupID != groupID {
		t.Errorf("GroupID = %q, want %q", groupInfo.GroupID, groupID)
	}
	if !groupInfo.Active {
		t.Error("new GroupInfo should have Active=true")
	}
}

// TestExecuteNew_Segment_TextReconstructed verifies that when Message.Message
// is empty, ExecuteNew reconstructs the text from Segment elements and still
// handles the command.
func TestExecuteNew_Segment_TextReconstructed(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	msg := &Message{
		Platform:    "QQ",
		MessageType: "group",
		GroupID:     "QQ-Group:789",
		GroupName:   "TestGroup",
		Time:        time.Now().Unix(),
		Sender:      SenderBase{UserID: "QQ:222", Nickname: "Tester"},
		Message:     "", // intentionally empty – should be filled from Segment
		Segment: []message.IMessageElement{
			&message.TextElement{Content: ".r 2d6"},
		},
	}

	d.ImSession.ExecuteNew(ep, msg)

	_, ok := adapter.waitForMsg(2 * time.Second)
	if !ok {
		t.Fatal("timeout: command in text segment was not processed")
	}
}

// TestExecuteNew_InvalidMessageType verifies that messages with an unsupported
// MessageType (neither "group" nor "private") are silently discarded.
func TestExecuteNew_InvalidMessageType(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	msg := &Message{
		Platform:    "QQ",
		MessageType: "channel", // unsupported
		GroupID:     "QQ-Group:111",
		Time:        time.Now().Unix(),
		Sender:      SenderBase{UserID: "QQ:999", Nickname: "Tester"},
		Message:     ".r",
		Segment:     []message.IMessageElement{&message.TextElement{Content: ".r"}},
	}

	d.ImSession.ExecuteNew(ep, msg)

	select {
	case got := <-adapter.msgCh:
		t.Errorf("unexpected reply for unsupported MessageType: %q", got)
	case <-time.After(300 * time.Millisecond):
		// expected: silently ignored
	}
}

// TestExecuteNew_MultipleGroups verifies that two independent groups each get
// their own GroupInfo and that commands in one group do not bleed into the
// other.
func TestExecuteNew_MultipleGroups(t *testing.T) {
	d, ep, adapter, cleanup := newExecuteNewTestDice(t)
	defer cleanup()

	const (
		groupA = "QQ-Group:AAAA"
		groupB = "QQ-Group:BBBB"
	)

	d.ImSession.ExecuteNew(ep, newGroupMsg(groupA, "QQ:001", ".r"))
	_, okA := adapter.waitForMsg(2 * time.Second)

	d.ImSession.ExecuteNew(ep, newGroupMsg(groupB, "QQ:002", ".r"))
	_, okB := adapter.waitForMsg(2 * time.Second)

	if !okA {
		t.Error("group A: timeout waiting for reply")
	}
	if !okB {
		t.Error("group B: timeout waiting for reply")
	}

	_, existsA := d.ImSession.ServiceAtNew.Load(groupA)
	_, existsB := d.ImSession.ServiceAtNew.Load(groupB)
	if !existsA {
		t.Error("GroupInfo not created for group A")
	}
	if !existsB {
		t.Error("GroupInfo not created for group B")
	}
}
