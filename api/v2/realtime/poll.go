package realtime

import (
	"context"
	"time"

	"sealdice-core/api/v2/imconnection"
	"sealdice-core/dice"
	"sealdice-core/logger"
	"sealdice-core/model/common/response"
	"sealdice-core/utils/epoch"
)

// ToolMessageProvider 提供带序号的指令测试消息，供轮询降级使用。
type ToolMessageProvider interface {
	MessagesAfter(afterSeq uint64, afterEpoch string) (items []dice.HTTPTestMessage, latestSeq uint64, currentEpoch string, reset bool)
}

type PollRequest struct {
	AfterLogSeq  uint64 `query:"afterLogSeq" doc:"已收到的最后一条日志序号"`
	AfterToolSeq uint64 `query:"afterToolSeq" doc:"已收到的最后一条测试消息序号"`
	AfterEpoch   string `query:"afterEpoch" doc:"服务进程标识"`
}

type PollLogs struct {
	Items     []*logger.LogItem `json:"items"`
	LatestSeq uint64            `json:"latestSeq"`
	Reset     bool              `json:"reset"`
}

type PollToolMessages struct {
	Items     []dice.HTTPTestMessage `json:"items"`
	LatestSeq uint64                 `json:"latestSeq"`
	Reset     bool                   `json:"reset"`
}

type PollData struct {
	Logs         PollLogs                `json:"logs"`
	Connections  IMConnectionListPayload `json:"connections"`
	ToolMessages PollToolMessages        `json:"toolMessages"`
	Epoch        string                  `json:"epoch"`
	Timestamp    int64                   `json:"timestamp"`
}

func (s *Server) SetToolMessageProvider(provider ToolMessageProvider) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.toolMessages = provider
}

// Poll 返回 SSE bootstrap 中同类数据的增量版本，供 SSE 不可用时降级使用。
// 日志与测试消息通过序号续传；连接列表直接返回完整快照。
func (s *Server) Poll(_ context.Context, req *PollRequest) (*response.ItemResponse[PollData], error) {
	if req == nil {
		req = &PollRequest{}
	}

	data := PollData{
		Epoch:     epoch.Current(),
		Timestamp: time.Now().Unix(),
	}

	d := primaryDice(s.dm)
	if d != nil && d.LogWriter != nil {
		items, latestSeq, _, reset := d.LogWriter.SnapshotAfter(req.AfterLogSeq, req.AfterEpoch)
		data.Logs = PollLogs{Items: items, LatestSeq: latestSeq, Reset: reset}
	} else {
		data.Logs.Reset = true
	}

	if d != nil && d.ImSession != nil {
		data.Connections = IMConnectionListPayload{
			Items: imconnection.NewEndPointInfoList(d.ImSession.EndPoints),
		}
	} else {
		data.Connections = IMConnectionListPayload{Items: []*imconnection.EndPointInfo{}}
	}

	s.mu.Lock()
	toolMessages := s.toolMessages
	s.mu.Unlock()
	if toolMessages != nil {
		items, latestSeq, _, reset := toolMessages.MessagesAfter(req.AfterToolSeq, req.AfterEpoch)
		data.ToolMessages = PollToolMessages{Items: items, LatestSeq: latestSeq, Reset: reset}
	} else {
		data.ToolMessages.Reset = true
	}

	return response.NewItemResponse(data), nil
}
