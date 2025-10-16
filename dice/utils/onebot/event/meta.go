package event

import "sealdice-core/dice/utils/onebot/types"

type LifeMeta struct {
	SubType string `json:"sub_type"` //enable disable connect
}

func (e LifeMeta) Type() string {
	return "meta_event:lifecycle"
}

type HeartMeta struct {
	Status   types.Status `json:"status"`
	Interval int64        `json:"interval"`
}

func (e HeartMeta) Type() string {
	return "meta_event:heartbeat"
}
