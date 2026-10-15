package realtime_test

import (
	"testing"

	realtime "sealdice-core/api/v2/realtime"
	"sealdice-core/dice"
)

func TestBuildBootstrapEventsIncludesCurrentSnapshots(t *testing.T) {
	dm := newTestDiceManager()
	_, _ = dm.GetDice().LogWriter.Write([]byte(`{"level":"info","module":"core","time":"2026-05-17T12:34:56.000Z+0800","msg":"hello"}`))

	ep := dice.NewMilkyConnItem(dice.AddMilkyEcho{BuiltInMode: "yogurt"})
	ep.Enable = true
	ep.BindRuntime(dm.GetDice().ImSession)
	pa := ep.Adapter.(*dice.PlatformAdapterMilky)
	pa.BuiltInLoginState = dice.MilkyLoginStateQRWaitingForScan
	pa.QrCodeData = []byte("fake-png")
	dm.GetDice().ImSession.EndPoints = append(dm.GetDice().ImSession.EndPoints, ep)

	events := realtime.BuildBootstrapEvents(dm)
	if len(events) != 3 {
		t.Fatalf("bootstrap event count = %d, want 3", len(events))
	}

	assertHasEvent(t, events, realtime.EventSystemReady)

	logEvt := findEvent(events, realtime.EventLogsSnapshot)
	logPayload, ok := logEvt.Payload.(realtime.LogSnapshotPayload)
	if !ok {
		t.Fatalf("logs payload type = %T, want LogSnapshotPayload", logEvt.Payload)
	}
	if len(logPayload.Items) != 1 || logPayload.Items[0].Msg != "hello" {
		t.Fatalf("log payload = %#v", logPayload.Items)
	}

	listEvt := findEvent(events, realtime.EventIMConnectionList)
	listPayload := listEvt.Payload.(realtime.IMConnectionListPayload)
	if len(listPayload.Items) != 1 || listPayload.Items[0].ID != ep.ID {
		t.Fatalf("list payload = %#v", listPayload.Items)
	}

	if listPayload.Items[0].Workflow.State != "qrcode" || listPayload.Items[0].QRCode == "" {
		t.Fatalf("snapshot missing login state: %#v", listPayload.Items[0])
	}
}

func assertHasEvent(t *testing.T, events []realtime.Event, name string) {
	t.Helper()
	if findEvent(events, name).Name == "" {
		t.Fatalf("missing event %q", name)
	}
}

func findEvent(events []realtime.Event, name string) realtime.Event {
	for _, evt := range events {
		if evt.Name == name {
			return evt
		}
	}
	return realtime.Event{}
}
