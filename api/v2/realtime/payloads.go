package realtime

import (
	"sealdice-core/api/v2/model/imconnection"
	"sealdice-core/dice"
	"sealdice-core/logger"
)

const (
	EventSystemReady          = "system/ready"
	EventLogsSnapshot         = "logs/snapshot"
	EventLogsAppend           = "logs/append"
	EventIMConnectionList     = "imconnection/list"
	EventIMConnectionUpdated  = "imconnection/updated"
	EventIMConnectionWorkflow = "imconnection/workflow"
	EventIMConnectionQRCode   = "imconnection/qrcode"
)

type SystemReadyPayload struct{}

type LogSnapshotPayload struct {
	Items []*logger.LogItem `json:"items"`
}

type LogAppendPayload struct {
	Item *logger.LogItem `json:"item"`
}

type IMConnectionListPayload struct {
	Items []*dice.EndPointInfo `json:"items"`
}

type IMConnectionUpdatedPayload struct {
	Item *dice.EndPointInfo `json:"item"`
}

type IMConnectionWorkflowPayload struct {
	EndpointID string                    `json:"endpointId"`
	Workflow   imconnection.WorkflowResp `json:"workflow"`
}

type IMConnectionQRCodePayload struct {
	EndpointID string `json:"endpointId"`
	Img        string `json:"img"`
}
