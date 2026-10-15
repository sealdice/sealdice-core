package realtime

import (
	"encoding/json"
	"sync"

	imconnm "sealdice-core/api/v2/imconnection"
	"sealdice-core/dice"
)

type StateWatcher struct {
	dm  *dice.DiceManager
	bus *Bus

	mu          sync.Mutex
	listHash    string
	itemHashes  map[string]string
	workflows   map[string]imconnm.WorkflowResp
	qrcodeImage map[string]string
}

func NewStateWatcher(dm *dice.DiceManager, bus *Bus) *StateWatcher {
	return &StateWatcher{
		dm:          dm,
		bus:         bus,
		itemHashes:  map[string]string{},
		workflows:   map[string]imconnm.WorkflowResp{},
		qrcodeImage: map[string]string{},
	}
}

func (w *StateWatcher) BindLogs() func() {
	d := w.dice()
	if d == nil || d.LogWriter == nil {
		return func() {}
	}

	ch, unsubscribe := d.LogWriter.Subscribe()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for item := range ch {
			w.bus.Publish(Event{
				Name: EventLogsAppend,
				Payload: LogAppendPayload{
					Item: item,
				},
			})
		}
	}()

	return func() {
		unsubscribe()
		<-done
	}
}

func (w *StateWatcher) Scan() {
	d := w.dice()
	if d == nil || d.ImSession == nil {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	endpoints := d.ImSession.EndPoints
	currentListHash := marshalString(endpoints)
	if currentListHash != w.listHash {
		w.listHash = currentListHash
		w.bus.Publish(Event{
			Name: EventIMConnectionList,
			Payload: IMConnectionListPayload{
				Items: imconnm.NewEndPointInfoList(endpoints),
			},
		})
	}

	currentIDs := map[string]struct{}{}
	for _, ep := range endpoints {
		if ep == nil {
			continue
		}
		currentIDs[ep.ID] = struct{}{}

		itemHash := marshalString(ep)
		if itemHash != w.itemHashes[ep.ID] {
			w.itemHashes[ep.ID] = itemHash
			w.bus.Publish(Event{
				Name: EventIMConnectionUpdated,
				Payload: IMConnectionUpdatedPayload{
					Item: imconnm.NewEndPointInfo(ep),
				},
			})
		}

		workflow := imconnm.WorkflowOfEndpoint(ep)
		if workflow != w.workflows[ep.ID] {
			w.workflows[ep.ID] = workflow
			w.bus.Publish(Event{
				Name: EventIMConnectionWorkflow,
				Payload: IMConnectionWorkflowPayload{
					EndpointID: ep.ID,
					Workflow:   workflow,
				},
			})
		}

		qr := imconnm.QRCodeOfEndpoint(ep)
		if qr != w.qrcodeImage[ep.ID] {
			w.qrcodeImage[ep.ID] = qr
			w.bus.Publish(Event{
				Name: EventIMConnectionQRCode,
				Payload: IMConnectionQRCodePayload{
					EndpointID: ep.ID,
					Img:        qr,
				},
			})
		}
	}

	for id := range w.itemHashes {
		if _, ok := currentIDs[id]; ok {
			continue
		}
		delete(w.itemHashes, id)
		delete(w.workflows, id)
		delete(w.qrcodeImage, id)
	}
}

func (w *StateWatcher) dice() *dice.Dice {
	if w.dm == nil || len(w.dm.Dice) == 0 {
		return nil
	}
	return w.dm.GetDice()
}

func marshalString(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}
