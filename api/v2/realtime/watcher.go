package realtime

import (
	"encoding/base64"
	"encoding/json"
	"sync"

	imconnm "sealdice-core/api/v2/model/imconnection"
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
	go func() {
		for item := range ch {
			w.bus.Publish(Event{
				Name: EventLogsAppend,
				Payload: LogAppendPayload{
					Item: item,
				},
			})
		}
	}()

	return unsubscribe
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
				Items: endpoints,
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
					Item: ep,
				},
			})
		}

		workflow := WorkflowOfEndpoint(ep)
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

		qr := qrCodeOfEndpoint(ep)
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

func WorkflowOfEndpoint(ep *dice.EndPointInfo) imconnm.WorkflowResp {
	switch pa := ep.Adapter.(type) {
	case *dice.PlatformAdapterGocq:
		state, hasQR := mapGocqWorkflow(pa.GoCqhttpState, len(pa.GoCqhttpQrcodeData) > 0)
		return imconnm.WorkflowResp{
			State:        state,
			HasQRCode:    hasQR,
			LoginState:   int64(pa.GoCqhttpState),
			FailedReason: pa.GocqhttpLoginFailedReason,
		}
	case *dice.PlatformAdapterMilky:
		state, hasQR := mapMilkyWorkflow(pa.BuiltInLoginState, len(pa.QrCodeData) > 0)
		return imconnm.WorkflowResp{
			State:      state,
			HasQRCode:  hasQR,
			LoginState: int64(pa.BuiltInLoginState),
		}
	default:
		return imconnm.WorkflowResp{State: "none"}
	}
}

func qrCodeOfEndpoint(ep *dice.EndPointInfo) string {
	switch pa := ep.Adapter.(type) {
	case *dice.PlatformAdapterGocq:
		if pa.GoCqhttpState == dice.StateCodeInLoginQrCode && len(pa.GoCqhttpQrcodeData) > 0 {
			return "data:image/png;base64," + base64.StdEncoding.EncodeToString(pa.GoCqhttpQrcodeData)
		}
	case *dice.PlatformAdapterMilky:
		if pa.BuiltInLoginState == dice.MilkyLoginStateQRWaitingForScan && len(pa.QrCodeData) > 0 {
			return "data:image/png;base64," + base64.StdEncoding.EncodeToString(pa.QrCodeData)
		}
	}
	return ""
}

func mapGocqWorkflow(state int, hasQR bool) (string, bool) {
	switch state {
	case dice.StateCodeInLoginQrCode:
		return "qrcode", hasQR
	case dice.StateCodeInLogin:
		return "pending", false
	case dice.StateCodeLoginSuccessed:
		return "success", false
	case dice.StateCodeLoginFailed:
		return "failed", false
	default:
		return "idle", false
	}
}

func mapMilkyWorkflow(state dice.MilkyLoginState, hasQR bool) (string, bool) {
	switch state {
	case dice.MilkyLoginStateQRWaitingForScan:
		return "qrcode", hasQR
	case dice.MilkyLoginStateConnecting:
		return "pending", false
	case dice.MilkyLoginStateQRConnected:
		return "success", false
	case dice.MilkyLoginStateFailed:
		return "failed", false
	default:
		return "idle", false
	}
}

func marshalString(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}
