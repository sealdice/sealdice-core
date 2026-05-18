package realtime_test

import (
	"testing"
	"time"

	imconnm "sealdice-core/api/v2/model/imconnection"
	"sealdice-core/api/v2/realtime"
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

func TestStateWatcherPublishesConnectionSnapshotsAndWorkflowChanges(t *testing.T) {
	dm := newTestDiceManager()
	bus := realtime.NewBus()
	watcher := realtime.NewStateWatcher(dm, bus)

	ep := dice.NewMilkyConnItem(dice.AddMilkyEcho{BuiltInMode: "yogurt"})
	ep.Enable = true
	ep.State = dice.StateConnecting
	ep.Session = dm.GetDice().ImSession
	pa := ep.Adapter.(*dice.PlatformAdapterMilky)
	pa.Session = dm.GetDice().ImSession
	pa.EndPoint = ep
	pa.BuiltInLoginState = dice.MilkyLoginStateQRWaitingForScan
	pa.QrCodeData = []byte("fake-png")
	dm.GetDice().ImSession.EndPoints = append(dm.GetDice().ImSession.EndPoints, ep)

	ch, unsubscribe := bus.Subscribe(8)
	defer unsubscribe()

	watcher.Scan()

	listEvt := waitForEvent(t, ch, realtime.EventIMConnectionList)
	listPayload, ok := listEvt.Payload.(realtime.IMConnectionListPayload)
	if !ok {
		t.Fatalf("payload type = %T, want IMConnectionListPayload", listEvt.Payload)
	}
	if len(listPayload.Items) != 1 || listPayload.Items[0].ID != ep.ID {
		t.Fatalf("list payload = %#v, want endpoint %s", listPayload.Items, ep.ID)
	}

	updatedEvt := waitForEvent(t, ch, realtime.EventIMConnectionUpdated)
	updatedPayload, ok := updatedEvt.Payload.(realtime.IMConnectionUpdatedPayload)
	if !ok {
		t.Fatalf("payload type = %T, want IMConnectionUpdatedPayload", updatedEvt.Payload)
	}
	if updatedPayload.Item == nil || updatedPayload.Item.ID != ep.ID {
		t.Fatalf("updated payload = %#v, want endpoint %s", updatedPayload.Item, ep.ID)
	}

	workflowEvt := waitForEvent(t, ch, realtime.EventIMConnectionWorkflow)
	workflowPayload, ok := workflowEvt.Payload.(realtime.IMConnectionWorkflowPayload)
	if !ok {
		t.Fatalf("payload type = %T, want IMConnectionWorkflowPayload", workflowEvt.Payload)
	}
	if workflowPayload.EndpointID != ep.ID {
		t.Fatalf("workflow endpoint = %q, want %q", workflowPayload.EndpointID, ep.ID)
	}
	if workflowPayload.Workflow.State != "qrcode" || !workflowPayload.Workflow.HasQRCode {
		t.Fatalf("workflow = %#v, want qrcode state with QR code", workflowPayload.Workflow)
	}

	qrEvt := waitForEvent(t, ch, realtime.EventIMConnectionQRCode)
	qrPayload, ok := qrEvt.Payload.(realtime.IMConnectionQRCodePayload)
	if !ok {
		t.Fatalf("payload type = %T, want IMConnectionQRCodePayload", qrEvt.Payload)
	}
	if qrPayload.EndpointID != ep.ID {
		t.Fatalf("qrcode endpoint = %q, want %q", qrPayload.EndpointID, ep.ID)
	}
	if qrPayload.Img == "" {
		t.Fatal("qrcode image should not be empty")
	}

	pa.BuiltInLoginState = dice.MilkyLoginStateQRConnected
	pa.QrCodeData = nil
	ep.State = dice.StateConnected

	watcher.Scan()

	updatedEvt = waitForEvent(t, ch, realtime.EventIMConnectionUpdated)
	updatedPayload = updatedEvt.Payload.(realtime.IMConnectionUpdatedPayload)
	if updatedPayload.Item == nil || updatedPayload.Item.State != dice.StateConnected {
		t.Fatalf("updated payload state = %#v, want connected", updatedPayload.Item)
	}

	workflowEvt = waitForEvent(t, ch, realtime.EventIMConnectionWorkflow)
	workflowPayload = workflowEvt.Payload.(realtime.IMConnectionWorkflowPayload)
	if workflowPayload.Workflow.State != "success" {
		t.Fatalf("workflow state = %q, want success", workflowPayload.Workflow.State)
	}

	qrEvt = waitForEvent(t, ch, realtime.EventIMConnectionQRCode)
	qrPayload = qrEvt.Payload.(realtime.IMConnectionQRCodePayload)
	if qrPayload.Img != "" {
		t.Fatalf("qrcode image = %q, want empty after success", qrPayload.Img)
	}
}

func TestBuildConnectionWorkflowMatchesIMConnectionContract(t *testing.T) {
	ep := dice.NewMilkyConnItem(dice.AddMilkyEcho{BuiltInMode: "yogurt"})
	pa := ep.Adapter.(*dice.PlatformAdapterMilky)
	pa.BuiltInLoginState = dice.MilkyLoginStateFailed

	workflow := realtime.WorkflowOfEndpoint(ep)
	if workflow != (imconnm.WorkflowResp{State: "failed", LoginState: int64(dice.MilkyLoginStateFailed)}) {
		t.Fatalf("workflow = %#v", workflow)
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
		Dice: []*dice.Dice{d},
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
