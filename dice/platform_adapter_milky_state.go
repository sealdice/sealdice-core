package dice

import (
	"context"
	"fmt"

	milky "github.com/Szzrain/Milky-go-sdk"

	"sealdice-core/logger"
	"sealdice-core/utils/procs"
)

// generation 为 0 时是分离连接；内置连接必须属于当前启动的进程。
func (pa *PlatformAdapterMilky) startMilkySession(session *milky.Session, generation uint64) bool {
	pa.lifecycleMu.Lock()
	if generation != 0 && generation != pa.processGeneration {
		pa.lifecycleMu.Unlock()
		return false
	}
	previous := pa.IntentSession
	previousCancel := pa.sessionCancel
	pa.IntentSession = session
	pa.sessionContext, pa.sessionCancel = context.WithCancel(context.Background())
	pa.sessionActive = true
	pa.sessionReady = false
	pa.accountOffline = false
	pa.transportConnected = false
	pa.EndPoint.State = StateConnecting
	pa.EndPoint.Enable = true
	pa.lifecycleMu.Unlock()
	if previousCancel != nil {
		previousCancel()
	}
	if previous != nil {
		_ = pa.closeMilkyTransport(previous)
	}
	return true
}

func (pa *PlatformAdapterMilky) stopMilkySession() {
	pa.lifecycleMu.Lock()
	session := pa.IntentSession
	cancel := pa.sessionCancel
	pa.sessionActive = false
	pa.sessionReady = false
	pa.transportConnected = false
	pa.lifecycleMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if session != nil {
		_ = pa.closeMilkyTransport(session)
	}
}

func (pa *PlatformAdapterMilky) failMilkySession(session *milky.Session) bool {
	pa.lifecycleMu.Lock()
	if pa.IntentSession != session || !pa.sessionActive {
		pa.lifecycleMu.Unlock()
		return false
	}
	pa.sessionActive = false
	pa.sessionReady = false
	pa.transportConnected = false
	cancel := pa.sessionCancel
	pa.EndPoint.State = StateConnectionFailed
	pa.EndPoint.Enable = false
	pa.lifecycleMu.Unlock()
	if cancel != nil {
		cancel()
	}
	_ = pa.closeMilkyTransport(session)
	return true
}

func (pa *PlatformAdapterMilky) finishMilkySession(session *milky.Session, info *milky.LoginInfo) bool {
	pa.lifecycleMu.Lock()
	defer pa.lifecycleMu.Unlock()
	if pa.IntentSession != session || !pa.sessionActive || info == nil {
		return false
	}
	pa.sessionReady = true
	pa.EndPoint.UserID = fmt.Sprintf("QQ:%d", info.UIN)
	pa.EndPoint.Nickname = info.Nickname
	pa.updateMilkyConnectionState(pa.transportConnected)
	return true
}

// 必须持有 lifecycleMu；自动重连不改变部署者的启用意图。
func (pa *PlatformAdapterMilky) updateMilkyConnectionState(connected bool) {
	switch {
	case !connected || pa.accountOffline:
		pa.EndPoint.State = StateDisconnected
	case !pa.sessionReady:
		pa.EndPoint.State = StateConnecting
	default:
		pa.EndPoint.State = StateConnected
	}
}

func (pa *PlatformAdapterMilky) onMilkyConnectionChange(session *milky.Session, connected bool) {
	pa.lifecycleMu.Lock()
	defer pa.lifecycleMu.Unlock()
	if pa.IntentSession != session || !pa.sessionActive {
		return // 旧连接或过期的通知不能覆盖当前状态。
	}
	pa.transportConnected = connected
	pa.updateMilkyConnectionState(connected)
}

func (pa *PlatformAdapterMilky) onMilkyBotOffline(session *milky.Session, reason string) {
	pa.lifecycleMu.Lock()
	defer pa.lifecycleMu.Unlock()
	if pa.IntentSession != session || !pa.sessionActive {
		return
	}
	pa.accountOffline = true
	pa.EndPoint.State = StateDisconnected
	logger.M().Warnf("Milky QQ 账号离线，账号 %s，原因：%s", pa.EndPoint.UserID, reason)
}

func (pa *PlatformAdapterMilky) onMilkyMessage(session *milky.Session) {
	connected := pa.isMilkyTransportConnected(session)
	pa.lifecycleMu.Lock()
	defer pa.lifecycleMu.Unlock()
	if pa.IntentSession == session && pa.sessionActive && pa.sessionReady && connected {
		// 收到实际 QQ 消息证明账号已经恢复；不能只凭本地 WS 握手清除离线状态。
		pa.accountOffline = false
		pa.transportConnected = true
		pa.EndPoint.State = StateConnected
	}
}

func (pa *PlatformAdapterMilky) beginMilkyProcess() (uint64, chan struct{}) {
	pa.lifecycleMu.Lock()
	defer pa.lifecycleMu.Unlock()
	pa.processGeneration++
	done := make(chan struct{})
	pa.processDone = done
	pa.BuiltInLoginState = MilkyLoginStateInit
	pa.EndPoint.State = StateConnecting
	pa.EndPoint.Enable = true
	return pa.processGeneration, done
}

func (pa *PlatformAdapterMilky) isCurrentMilkyProcess(generation uint64) bool {
	pa.lifecycleMu.Lock()
	defer pa.lifecycleMu.Unlock()
	return generation == pa.processGeneration
}

func (pa *PlatformAdapterMilky) milkyProcessLoginState(generation uint64) (MilkyLoginState, bool) {
	pa.lifecycleMu.Lock()
	defer pa.lifecycleMu.Unlock()
	return pa.BuiltInLoginState, generation == pa.processGeneration
}

func (pa *PlatformAdapterMilky) setMilkyProcessLoginState(generation uint64, state MilkyLoginState) bool {
	pa.lifecycleMu.Lock()
	defer pa.lifecycleMu.Unlock()
	if generation != pa.processGeneration {
		return false
	}
	pa.BuiltInLoginState = state
	return true
}

func (pa *PlatformAdapterMilky) setMilkyProcessQRCode(generation uint64, data []byte) bool {
	pa.lifecycleMu.Lock()
	defer pa.lifecycleMu.Unlock()
	if generation != pa.processGeneration || pa.BuiltInLoginState >= MilkyLoginStateConnecting {
		return false
	}
	pa.QrCodeData = data
	if data == nil {
		pa.BuiltInLoginState = MilkyLoginStateFailed
	} else {
		pa.BuiltInLoginState = MilkyLoginStateQRWaitingForScan
	}
	return true
}

func (pa *PlatformAdapterMilky) registerMilkyProcess(generation uint64, process *procs.Process) bool {
	pa.lifecycleMu.Lock()
	defer pa.lifecycleMu.Unlock()
	if generation != pa.processGeneration {
		return false
	}
	pa.MilkyProcess = process
	return true
}

func (pa *PlatformAdapterMilky) detachMilkyProcess() (*procs.Process, chan struct{}) {
	pa.lifecycleMu.Lock()
	defer pa.lifecycleMu.Unlock()
	pa.processGeneration++
	process, done := pa.MilkyProcess, pa.processDone
	pa.MilkyProcess = nil
	pa.processDone = nil
	return process, done
}

func (pa *PlatformAdapterMilky) failMilkyProcessStart(generation uint64) bool {
	pa.lifecycleMu.Lock()
	defer pa.lifecycleMu.Unlock()
	if generation != pa.processGeneration {
		return false
	}
	pa.BuiltInLoginState = MilkyLoginStateFailed
	pa.EndPoint.State = StateConnectionFailed
	return true
}

// 仅用于启动协程接管前的失败。调用方仍拥有 done，旧尝试只关闭自己的通道。
func (pa *PlatformAdapterMilky) abortMilkyProcessSetup(generation uint64, done chan struct{}) bool {
	defer close(done)
	pa.lifecycleMu.Lock()
	defer pa.lifecycleMu.Unlock()
	if generation != pa.processGeneration || pa.processDone != done {
		return false
	}
	pa.processGeneration++
	pa.processDone = nil
	pa.BuiltInLoginState = MilkyLoginStateFailed
	pa.EndPoint.State = StateConnectionFailed
	return true
}

func (pa *PlatformAdapterMilky) finishMilkyProcess(generation uint64, process *procs.Process) {
	pa.lifecycleMu.Lock()
	if generation != pa.processGeneration || pa.MilkyProcess != process {
		pa.lifecycleMu.Unlock()
		return
	}
	pa.processGeneration++
	pa.MilkyProcess = nil
	pa.processDone = nil
	pa.BuiltInLoginState = MilkyLoginStateFailed
	if pa.EndPoint.State != StateConnectionFailed {
		pa.EndPoint.State = StateDisconnected
	}
	session := pa.IntentSession
	cancel := pa.sessionCancel
	pa.sessionActive = false
	pa.sessionReady = false
	pa.transportConnected = false
	pa.lifecycleMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if session != nil {
		_ = pa.closeMilkyTransport(session)
	}
}
