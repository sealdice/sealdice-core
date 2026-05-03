package dice //nolint:testpackage

import (
	"context"
	"testing"
)

func TestPlatformAdapterOnebot_EnableNotClearedOnConnectFailure(t *testing.T) {
	ep := &EndPointInfo{}
	s := &IMSession{}

	pa := &PlatformAdapterOnebot{
		Session:  s,
		EndPoint: ep,
		Mode:     "unknown-mode", // 避免真实连接，确保 startConnection 立即失败
	}

	pa.ensureFSM()
	pa.desiredEnabled = true

	// 避免进入失败态后启动耗时的重试循环（仍会起 goroutine，但会立刻返回）
	pa.retryMutex.Lock()
	pa.isRetrying = true
	pa.retryMutex.Unlock()

	err := pa.sm.Event(context.Background(), "enable")
	if err != nil {
		t.Fatalf("expected enable event to be accepted, got error: %v", err)
	}

	if !ep.Enable {
		t.Fatalf("expected EndPoint.Enable to remain true after connection failure")
	}
	if ep.State != StateConnectionFailed {
		t.Fatalf("expected EndPoint.State=%v, got %v", StateConnectionFailed, ep.State)
	}

	pa.cleanupResources()
}
