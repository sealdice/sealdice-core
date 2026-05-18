package realtime

import (
	"testing"

	"sealdice-core/dice"
)

func TestBuildBootstrapEventsIncludesCurrentSnapshots(t *testing.T) {
	dm := newTestDiceManager()
	_, _ = dm.GetDice().LogWriter.Write([]byte(`{"level":"info","module":"core","time":"2026-05-17T12:34:56.000Z+0800","msg":"hello"}`))

	ep := dice.NewMilkyConnItem(dice.AddMilkyEcho{BuiltInMode: "yogurt"})
	ep.Enable = true
	ep.Session = dm.GetDice().ImSession
	pa := ep.Adapter.(*dice.PlatformAdapterMilky)
	pa.Session = dm.GetDice().ImSession
	pa.EndPoint = ep
	pa.BuiltInLoginState = dice.MilkyLoginStateQRWaitingForScan
	pa.QrCodeData = []byte("fake-png")
	dm.GetDice().ImSession.EndPoints = append(dm.GetDice().ImSession.EndPoints, ep)

	events := buildBootstrapEvents(dm)
	if len(events) < 5 {
		t.Fatalf("bootstrap event count = %d, want at least 5", len(events))
	}

	assertHasEvent(t, events, EventSystemReady)

	logEvt := findEvent(events, EventLogsSnapshot)
	logPayload, ok := logEvt.Payload.(LogSnapshotPayload)
	if !ok {
		t.Fatalf("logs payload type = %T, want LogSnapshotPayload", logEvt.Payload)
	}
	if len(logPayload.Items) != 1 || logPayload.Items[0].Msg != "hello" {
		t.Fatalf("log payload = %#v", logPayload.Items)
	}

	listEvt := findEvent(events, EventIMConnectionList)
	listPayload := listEvt.Payload.(IMConnectionListPayload)
	if len(listPayload.Items) != 1 || listPayload.Items[0].ID != ep.ID {
		t.Fatalf("list payload = %#v", listPayload.Items)
	}

	workflowEvt := findEvent(events, EventIMConnectionWorkflow)
	workflowPayload := workflowEvt.Payload.(IMConnectionWorkflowPayload)
	if workflowPayload.EndpointID != ep.ID || workflowPayload.Workflow.State != "qrcode" {
		t.Fatalf("workflow payload = %#v", workflowPayload)
	}

	qrEvt := findEvent(events, EventIMConnectionQRCode)
	qrPayload := qrEvt.Payload.(IMConnectionQRCodePayload)
	if qrPayload.EndpointID != ep.ID || qrPayload.Img == "" {
		t.Fatalf("qr payload = %#v", qrPayload)
	}
}

func assertHasEvent(t *testing.T, events []Event, name string) {
	t.Helper()
	if findEvent(events, name).Name == "" {
		t.Fatalf("missing event %q", name)
	}
}

func findEvent(events []Event, name string) Event {
	for _, evt := range events {
		if evt.Name == name {
			return evt
		}
	}
	return Event{}
}
