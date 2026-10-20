package realtime_test

import (
	"context"
	"testing"

	realtime "sealdice-core/api/v2/realtime"
	"sealdice-core/dice"
	"sealdice-core/utils/epoch"
)

type fakeToolMessages struct {
	items     []dice.HTTPTestMessage
	latestSeq uint64
	reset     bool
}

func (f fakeToolMessages) MessagesAfter(_ uint64, _ string) ([]dice.HTTPTestMessage, uint64, string, bool) {
	return f.items, f.latestSeq, "test-epoch", f.reset
}

func TestPollReturnsLogDeltaAndConnectionSnapshot(t *testing.T) {
	dm := newTestDiceManager()
	srv := realtime.NewServer(dm)
	defer srv.Close()

	_, _ = dm.GetDice().LogWriter.Write([]byte(`{"level":"info","module":"core","time":"2026-05-17T12:34:56.000Z+0800","msg":"first"}`))
	_, _ = dm.GetDice().LogWriter.Write([]byte(`{"level":"info","module":"core","time":"2026-05-17T12:34:57.000Z+0800","msg":"second"}`))
	ep := dice.NewMilkyConnItem(dice.AddMilkyEcho{BuiltInMode: "yogurt"})
	ep.Enable = true
	ep.BindRuntime(dm.GetDice().ImSession)
	dm.GetDice().ImSession.EndPoints = []*dice.EndPointInfo{ep}
	srv.SetToolMessageProvider(fakeToolMessages{
		items: []dice.HTTPTestMessage{{
			Seq:         2,
			ID:          "second",
			MessageType: "private",
		}},
		latestSeq: 2,
	})

	resp, err := srv.Poll(context.Background(), &realtime.PollRequest{AfterLogSeq: 1, AfterToolSeq: 1, AfterEpoch: epoch.Current()})
	if err != nil {
		t.Fatalf("Poll returned error: %v", err)
	}
	item := resp.Body.Item
	if item.Logs.Reset || item.Logs.LatestSeq != 2 || len(item.Logs.Items) != 1 ||
		item.Logs.Items[0].Seq != 2 || item.Logs.Items[0].Msg != "second" {
		t.Fatalf("unexpected logs: %+v", item.Logs)
	}
	if len(item.Connections.Items) != 1 || item.Connections.Items[0].ID != ep.ID {
		t.Fatalf("unexpected connections: %+v", item.Connections.Items)
	}
	if item.ToolMessages.Reset || item.ToolMessages.LatestSeq != 2 ||
		len(item.ToolMessages.Items) != 1 || item.ToolMessages.Items[0].ID != "second" {
		t.Fatalf("unexpected tool messages: %+v", item.ToolMessages)
	}
}

func TestPollRequestsSnapshotWhenCursorIsUnset(t *testing.T) {
	dm := newTestDiceManager()
	srv := realtime.NewServer(dm)
	defer srv.Close()

	_, _ = dm.GetDice().LogWriter.Write([]byte(`{"level":"info","module":"core","time":"2026-05-17T12:34:56.000Z+0800","msg":"first"}`))

	resp, err := srv.Poll(context.Background(), &realtime.PollRequest{})
	if err != nil {
		t.Fatalf("Poll returned error: %v", err)
	}
	if !resp.Body.Item.Logs.Reset || len(resp.Body.Item.Logs.Items) != 1 ||
		resp.Body.Item.Logs.Items[0].Seq != 1 || resp.Body.Item.Logs.Items[0].Msg != "first" {
		t.Fatalf("unexpected logs: %+v", resp.Body.Item.Logs)
	}
}
