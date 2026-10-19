package realtime

import (
	"encoding/json"
	"slices"
	"sync"

	imconnm "sealdice-core/api/v2/imconnection"
	"sealdice-core/dice"
)

type StateWatcher struct {
	dm  *dice.DiceManager
	bus *Bus

	mu         sync.Mutex
	ids        []string
	itemHashes map[string]string
}

func NewStateWatcher(dm *dice.DiceManager, bus *Bus) *StateWatcher {
	return &StateWatcher{
		dm:  dm,
		bus: bus,
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

	items := imconnm.NewEndPointInfoList(d.ImSession.EndPoints)
	ids := make([]string, 0, len(items))
	nextHashes := make(map[string]string, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
		nextHashes[item.ID] = marshalString(item)
	}
	listChanged := w.itemHashes == nil || !slices.Equal(ids, w.ids)
	if listChanged {
		w.bus.Publish(Event{Name: EventIMConnectionList, Payload: IMConnectionListPayload{Items: items}})
	} else {
		for _, item := range items {
			if nextHashes[item.ID] != w.itemHashes[item.ID] {
				w.bus.Publish(Event{Name: EventIMConnectionUpdated, Payload: IMConnectionUpdatedPayload{Item: item}})
			}
		}
	}
	w.ids, w.itemHashes = ids, nextHashes
}

func (w *StateWatcher) dice() *dice.Dice {
	if w.dm == nil {
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
