package realtime

import (
	"sealdice-core/api/v2/imconnection"
	"sealdice-core/logger"
)

const (
	EventSystemReady         = "system/ready"
	EventSystemHeartbeat     = "system/heartbeat"
	EventLogsSnapshot        = "logs/snapshot"
	EventLogsAppend          = "logs/append"
	EventIMConnectionList    = "imconnection/list"
	EventIMConnectionUpdated = "imconnection/updated"
	EventToolTestMessage     = "tooltest/message"
)

type SystemReadyPayload struct{}

type SystemHeartbeatPayload struct{}

type LogSnapshotPayload struct {
	Items []*logger.LogItem `json:"items"`
}

type LogAppendPayload struct {
	Item *logger.LogItem `json:"item"`
}

type IMConnectionListPayload struct {
	Items []*imconnection.EndPointInfo `json:"items"`
}

type IMConnectionUpdatedPayload struct {
	Item *imconnection.EndPointInfo `json:"item"`
}
