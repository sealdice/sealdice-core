package realtime_test

import (
	"testing"
	"time"

	realtime "sealdice-core/api/v2/realtime"
	"sealdice-core/dice"
	"sealdice-core/logger"
)

func TestStateWatcherPublishesLogAppendEvents(t *testing.T) {
	dm := newTestDiceManager()
	bus := realtime.NewBus()
	watcher := realtime.NewStateWatcher(dm, bus)

	unsubscribeLogs := watcher.BindLogs()
	defer unsubscribeLogs()

	ch, unsubscribe := bus.Subscribe(1)
	defer unsubscribe()

	_, _ = dm.GetDice().LogWriter.Write([]byte(`{"level":"info","module":"core","time":"2026-05-17T12:34:56.000Z+0800","msg":"hello"}`))

	evt := waitForEvent(t, ch, realtime.EventLogsAppend)
	payload, ok := evt.Payload.(realtime.LogAppendPayload)
	if !ok {
		t.Fatalf("payload type = %T, want LogAppendPayload", evt.Payload)
	}
	if payload.Item == nil || payload.Item.Msg != "hello" {
		t.Fatalf("payload item = %#v, want message hello", payload.Item)
	}
}

func TestStateWatcherPublishesCompleteSnapshotsAndOnlyChangedItems(t *testing.T) {
	dm := newTestDiceManager()
	bus := realtime.NewBus()
	watcher := realtime.NewStateWatcher(dm, bus)
	ep := dice.NewMilkyConnItem(dice.AddMilkyEcho{BuiltInMode: "yogurt"})
	ep.Enable = true
	ep.State = dice.StateConnecting
	ep.BindRuntime(dm.GetDice().ImSession)
	pa := ep.Adapter.(*dice.PlatformAdapterMilky)
	pa.BuiltInLoginState = dice.MilkyLoginStateQRWaitingForScan
	pa.QrCodeData = []byte("fake-png")
	dm.GetDice().ImSession.EndPoints = []*dice.EndPointInfo{ep}
	ch, unsubscribe := bus.Subscribe(8)
	defer unsubscribe()

	watcher.Scan()
	list := waitForEvent(t, ch, realtime.EventIMConnectionList).Payload.(realtime.IMConnectionListPayload)
	if len(list.Items) != 1 || list.Items[0].Workflow.State != "qrcode" || list.Items[0].QRCode == "" {
		t.Fatalf("incomplete snapshot: %#v", list.Items)
	}
	assertNoEvents(t, ch) // The list already includes the changed item and login state.
	watcher.Scan()
	assertNoEvents(t, ch)

	pa.BuiltInLoginState = dice.MilkyLoginStateQRConnected
	pa.QrCodeData = nil
	ep.State = dice.StateConnected
	watcher.Scan()
	update := waitForEvent(t, ch, realtime.EventIMConnectionUpdated).Payload.(realtime.IMConnectionUpdatedPayload)
	if update.Item.State != dice.StateConnected || update.Item.Workflow.State != "success" || update.Item.QRCode != "" {
		t.Fatalf("incomplete update: %#v", update.Item)
	}
	assertNoEvents(t, ch)

	// A retained, non-scannable image is private adapter state, not a UI change.
	pa.QrCodeData = []byte("old-image")
	watcher.Scan()
	assertNoEvents(t, ch)

	dm.GetDice().ImSession.EndPoints = nil
	watcher.Scan()
	list = waitForEvent(t, ch, realtime.EventIMConnectionList).Payload.(realtime.IMConnectionListPayload)
	if len(list.Items) != 0 {
		t.Fatalf("deleted connection remained: %#v", list.Items)
	}
	assertNoEvents(t, ch)
}

func assertNoEvents(t *testing.T, ch <-chan realtime.Event) {
	t.Helper()
	select {
	case evt := <-ch:
		t.Fatalf("unexpected event: %s", evt.Name)
	default:
	}
}

func newTestDiceManager() *dice.DiceManager {
	d := &dice.Dice{
		Logger:    logger.M(),
		LogWriter: logger.NewUIWriter(),
	}
	d.ImSession = &dice.IMSession{
		Parent:       d,
		EndPoints:    []*dice.EndPointInfo{},
		ServiceAtNew: new(dice.SyncMap[string, *dice.GroupInfo]),
		PendingQuits: new(dice.SyncMap[string, *dice.PendingQuitInfo]),
	}
	dm := &dice.DiceManager{
		Dice: d,
	}
	d.Parent = dm
	return dm
}

func waitForEvent(t *testing.T, ch <-chan realtime.Event, name string) realtime.Event {
	t.Helper()

	deadline := time.After(time.Second)
	for {
		select {
		case evt := <-ch:
			if evt.Name == name {
				return evt
			}
		case <-deadline:
			t.Fatalf("timed out waiting for event %q", name)
		}
	}
}
