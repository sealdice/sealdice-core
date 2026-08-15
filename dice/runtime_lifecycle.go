package dice

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"

	"sealdice-core/dice/service"
	"sealdice-core/utils/constant"
	"sealdice-core/utils/dboperator/engine"
	sealws "sealdice-core/utils/plugin/websocket"
)

// RuntimeShutdowner is the optional, non-persistent shutdown contract for an
// adapter. It must stop input, reconnect loops, sockets and owned processes
// without changing EndPointInfo.Enable.
type RuntimeShutdowner interface {
	RuntimeShutdown(context.Context) error
}

type lifecyclePhase struct {
	mu   sync.Mutex
	done chan struct{}
	err  error
}

func (phase *lifecyclePhase) run(ctx context.Context, work func(context.Context) error) error {
	if ctx == nil {
		ctx = context.Background()
	}

	for {
		phase.mu.Lock()
		if phase.done == nil {
			attemptCtx := ctx
			phase.done = make(chan struct{})
			go func(done chan struct{}, attemptCtx context.Context) {
				err := runLifecycleWork(attemptCtx, work)
				phase.mu.Lock()
				phase.err = err
				close(done)
				phase.mu.Unlock()
			}(phase.done, attemptCtx)
		}
		done := phase.done
		phase.mu.Unlock()

		select {
		case <-done:
			phase.mu.Lock()
			err := phase.err
			// A phase that failed only because the attempt context expired can
			// be retried by a later caller with a fresh context. Genuine
			// shutdown errors remain sticky.
			retryable := err != nil &&
				(errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) &&
				ctx.Err() == nil
			if retryable {
				phase.done = nil
				phase.err = nil
			}
			phase.mu.Unlock()
			if retryable {
				continue
			}
			return err
		case <-ctx.Done():
			select {
			case <-done:
				phase.mu.Lock()
				err := phase.err
				phase.mu.Unlock()
				return err
			default:
				return ctx.Err()
			}
		}
	}
}

func runLifecycleWork(ctx context.Context, work func(context.Context) error) (resultErr error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			resultErr = fmt.Errorf("runtime 生命周期发生异常: %v\n%s", recovered, debug.Stack())
		}
	}()
	return work(ctx)
}

// Quiesce stops new generation tasks, input sources, reconnect loops and cron
// jobs, then waits for in-flight work and flushes mutable state. The database
// remains open so the restore layer can prepare its transaction.
func (dm *DiceManager) Quiesce(ctx context.Context) error {
	return dm.quiescePhase.run(ctx, dm.quiesce)
}

func (dm *DiceManager) quiesce(ctx context.Context) error {
	dm.runtimeMu.Lock()
	dm.runtimeClosing = true
	dm.CleanupFlag.CompareAndSwap(0, 1)
	cancel := dm.runtimeCancel
	dm.runtimeMu.Unlock()

	wasReady := dm.IsReady
	dm.IsReady = false
	if cancel != nil {
		cancel()
	}

	var shutdownErrors []error
	if dm.Cron != nil {
		if err := waitCronStopped(ctx, dm.Cron.Stop().Done(), "管理器定时任务"); err != nil {
			shutdownErrors = append(shutdownErrors, err)
		}
	}

	for _, d := range dm.DiceSnapshot() {
		if d == nil {
			continue
		}
		if d.Cron != nil {
			if err := waitCronStopped(ctx, d.Cron.Stop().Done(), "骰子定时任务"); err != nil {
				shutdownErrors = append(shutdownErrors, err)
			}
		}
		if d.JsScriptCron != nil {
			if err := waitCronStopped(ctx, d.JsScriptCron.Stop().Done(), "JS 定时任务"); err != nil {
				shutdownErrors = append(shutdownErrors, err)
			}
			d.JsScriptCron = nil
		}
		if err := d.shutdownAdapters(ctx); err != nil {
			shutdownErrors = append(shutdownErrors, err)
		}
	}

	if err := waitRuntimeTasks(ctx, &dm.runtimeWG); err != nil {
		shutdownErrors = append(shutdownErrors, err)
	}

	for _, d := range dm.DiceSnapshot() {
		if d == nil {
			continue
		}
		if d.AttrsManager != nil && d.AttrsManager.cancel != nil {
			if err := stopAttrsManager(ctx, d.AttrsManager); err != nil {
				shutdownErrors = append(shutdownErrors, err)
			}
		}
		if d.IsAlreadyLoadConfig {
			if d.Config.BanList != nil {
				d.Config.BanList.SaveChanged(d)
			}
			d.Save(true)
		}
	}
	if wasReady {
		dm.Save()
	}
	if err := flushRuntimeWAL(dm.Operator); err != nil {
		shutdownErrors = append(shutdownErrors, err)
	}
	return errors.Join(shutdownErrors...)
}

func waitCronStopped(ctx context.Context, done <-chan struct{}, name string) error {
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("等待%s停止: %w", name, ctx.Err())
	}
}

func waitRuntimeTasks(ctx context.Context, wg *sync.WaitGroup) error {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("等待 Runtime 后台任务停止: %w", ctx.Err())
	}
}

func stopAttrsManager(ctx context.Context, manager *AttrsManager) error {
	done := make(chan struct{})
	go func() {
		manager.Stop()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("等待属性存储刷新: %w", ctx.Err())
	}
}

func flushRuntimeWAL(operator engine.DatabaseOperator) error {
	if operator == nil {
		return nil
	}
	dbs := []struct {
		name string
		db   func() error
	}{
		{name: "data", db: func() error {
			db := operator.GetDataDB(constant.WRITE)
			if db == nil {
				return nil
			}
			return service.FlushWAL(db)
		}},
		{name: "logs", db: func() error {
			db := operator.GetLogDB(constant.WRITE)
			if db == nil {
				return nil
			}
			return service.FlushWAL(db)
		}},
		{name: "censor", db: func() error {
			db := operator.GetCensorDB(constant.WRITE)
			if db == nil {
				return nil
			}
			return service.FlushWAL(db)
		}},
	}
	var flushErrors []error
	for _, item := range dbs {
		if err := item.db(); err != nil {
			flushErrors = append(flushErrors, fmt.Errorf("刷新 %s WAL: %w", item.name, err))
		}
	}
	return errors.Join(flushErrors...)
}

func (d *Dice) shutdownAdapters(ctx context.Context) error {
	if d.ImSession == nil {
		return nil
	}
	var shutdownErrors []error
	for _, endpoint := range append([]*EndPointInfo(nil), d.ImSession.EndPoints...) {
		if endpoint == nil || endpoint.Adapter == nil {
			continue
		}
		shutdowner, ok := endpoint.Adapter.(RuntimeShutdowner)
		if !ok {
			shutdownErrors = append(shutdownErrors, fmt.Errorf(
				"适配器 %s/%s 不支持 RuntimeShutdown，无法确认资源已关闭",
				endpoint.Platform,
				endpoint.ProtocolType,
			))
			continue
		}
		if err := shutdownAdapter(ctx, shutdowner); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf(
				"关闭适配器 %s/%s: %w",
				endpoint.Platform,
				endpoint.ProtocolType,
				err,
			))
		}
		endpoint.State = StateDisconnected
	}
	return errors.Join(shutdownErrors...)
}

func shutdownAdapter(ctx context.Context, shutdowner RuntimeShutdowner) (resultErr error) {
	done := make(chan error, 1)
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				done <- fmt.Errorf("%v\n%s", recovered, debug.Stack())
			}
		}()
		done <- shutdowner.RuntimeShutdown(ctx)
	}()
	select {
	case resultErr = <-done:
		return resultErr
	case <-ctx.Done():
		return fmt.Errorf("关闭适配器超时: %w", ctx.Err())
	}
}

// Finalize releases JS runtimes, extension storage, help indexes and the
// database after a successful Quiesce.
func (dm *DiceManager) Finalize(ctx context.Context) error {
	if err := dm.Quiesce(ctx); err != nil {
		return err
	}
	return dm.finalizePhase.run(ctx, dm.finalize)
}

func (dm *DiceManager) finalize(ctx context.Context) error {
	var finalizeErrors []error
	for _, d := range dm.DiceSnapshot() {
		if d == nil {
			continue
		}
		if d.ExtLoopManager != nil {
			d.jsClear()
			if err := waitJSLoopStopped(ctx, d); err != nil {
				finalizeErrors = append(finalizeErrors, err)
			}
		}
		for _, ext := range d.ExtList {
			if ext != nil && ext.Storage != nil {
				if err := ext.StorageClose(); err != nil {
					finalizeErrors = append(finalizeErrors, fmt.Errorf("关闭扩展存储: %w", err))
				}
			}
		}
		d.IsAlreadyLoadConfig = false
		d.DBOperator = nil
	}

	// Close plugin WebSocket connections once for the whole Runtime instead
	// of during each Dice's jsClear, which would disconnect sibling Dice.
	sealws.GlobalConnManager.CloseAll()

	if dm.Help != nil {
		dm.Help.Close()
		dm.Help = nil
	}
	if err := errors.Join(finalizeErrors...); err != nil {
		return err
	}
	if dm.progressExitGroupWin != 0 {
		_ = dm.progressExitGroupWin.Dispose()
	}
	if dm.Operator != nil {
		dm.Operator.Close()
		dm.Operator = nil
	}
	dm.CleanupFlag.Store(2)
	return nil
}

func waitJSLoopStopped(ctx context.Context, d *Dice) error {
	d.jsLoopMu.Lock()
	done := d.jsLoopDone
	d.jsLoopMu.Unlock()
	if done == nil {
		return nil
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("等待 JS Runtime 停止: %w", ctx.Err())
	}
}

// Stop performs the two shutdown phases and shares each phase result across
// all callers. A timed-out first call cannot make a later call return success
// merely because shutdown had already started.
func (dm *DiceManager) Stop(ctx context.Context) error {
	if err := dm.Quiesce(ctx); err != nil {
		return err
	}
	return dm.Finalize(ctx)
}

func (dm *DiceManager) IsQuiesced() bool {
	return dm.CleanupFlag.Load() >= 1
}

// IsStopped reports whether Finalize completed.
func (dm *DiceManager) IsStopped() bool {
	return dm.CleanupFlag.Load() >= 2
}

func (*PlatformAdapterHTTP) RuntimeShutdown(context.Context) error { return nil }

func (pa *PlatformAdapterDiscord) RuntimeShutdown(_ context.Context) error {
	session := pa.IntentSession
	if session == nil {
		return nil
	}
	return session.Close()
}

func (pa *PlatformAdapterDingTalk) RuntimeShutdown(_ context.Context) error {
	return pa.closeSessionLocked()
}

func (pa *PlatformAdapterDodo) RuntimeShutdown(ctx context.Context) error {
	pa.runtimeMu.Lock()
	pa.runtimeStopping.Store(true)
	stop := pa.runtimeStop
	if stop != nil {
		select {
		case <-stop:
		default:
			close(stop)
		}
	}
	ws := pa.WebSocket
	done := pa.listenDone
	pa.WebSocket = nil
	pa.Client = nil
	pa.runtimeMu.Unlock()
	if ws != nil {
		ws.Close()
	}
	if done == nil {
		return nil
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (pa *PlatformAdapterKook) RuntimeShutdown(_ context.Context) error {
	session := pa.IntentSession
	if session == nil {
		return nil
	}
	return session.Close()
}

func (pa *PlatformAdapterMinecraft) RuntimeShutdown(_ context.Context) error {
	pa.runtimeStopping.Store(true)
	pa.Reconnecting.Store(true)
	pa.runtimeMu.Lock()
	socket := pa.Socket
	pa.Socket = nil
	pa.runtimeMu.Unlock()
	if socket != nil {
		socket.Close()
	}
	return nil
}

func (pa *PlatformAdapterSealChat) RuntimeShutdown(_ context.Context) error {
	pa.runtimeStopping.Store(true)
	pa.Reconnecting.Store(true)
	pa.stopHeartbeat()
	pa.runtimeMu.Lock()
	socket := pa.Socket
	pa.Socket = nil
	pa.runtimeMu.Unlock()
	if socket != nil {
		socket.Close()
	}
	return nil
}

func (pa *PlatformAdapterSlack) RuntimeShutdown(_ context.Context) error {
	cancel := pa.cancel
	if cancel != nil {
		cancel()
	}
	return nil
}

func (pa *PlatformAdapterTelegram) RuntimeShutdown(_ context.Context) error {
	session := pa.IntentSession
	if session != nil {
		session.StopReceivingUpdates()
	}
	return nil
}

func (pa *PlatformAdapterSatori) RuntimeShutdown(_ context.Context) error {
	if pa.CancelFunc != nil {
		pa.CancelFunc()
	}
	if pa.conn != nil {
		return pa.conn.Close()
	}
	return nil
}

func (pa *PlatformAdapterMilky) RuntimeShutdown(_ context.Context) error {
	var shutdownErrors []error
	if err := pa.stopMilkyRuntime(); err != nil {
		shutdownErrors = append(shutdownErrors, fmt.Errorf("停止内置 Milky 进程: %w", err))
	}
	pa.runtimeMu.Lock()
	session := pa.IntentSession
	pa.runtimeMu.Unlock()
	if session != nil {
		if err := session.Close(); err != nil {
			shutdownErrors = append(shutdownErrors, err)
		}
	}
	return errors.Join(shutdownErrors...)
}

func (pa *PlatformAdapterGocq) RuntimeShutdown(ctx context.Context) error {
	var shutdownErrors []error
	if err := pa.stopGoCqhttpRuntime(); err != nil {
		shutdownErrors = append(shutdownErrors, fmt.Errorf("停止内置 OneBot 进程: %w", err))
	}
	pa.runtimeMu.Lock()
	pa.bumpLoginIndexLocked()
	pa.diceServing = false
	disconnected := pa.InPackGoCqhttpDisconnectedCH
	pa.InPackGoCqhttpDisconnectedCH = nil
	reverseApp := pa.reverseApp
	pa.reverseApp = nil
	socket := pa.Socket
	pa.Socket = nil
	pa.runtimeMu.Unlock()
	signalAdapterStop(disconnected, 0)
	if reverseApp != nil {
		if err := reverseApp.Shutdown(ctx); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("关闭反向 OneBot 服务: %w", err))
		}
	}
	if socket != nil {
		socket.Close()
	}
	return errors.Join(shutdownErrors...)
}

func (pa *PlatformAdapterWalleQ) RuntimeShutdown(_ context.Context) error {
	var shutdownErrors []error
	if err := pa.stopWalleQRuntime(); err != nil {
		shutdownErrors = append(shutdownErrors, fmt.Errorf("停止内置 WalleQ 进程: %w", err))
	}
	pa.runtimeMu.Lock()
	pa.bumpLoginIndexLocked()
	pa.DiceServing = false
	disconnected := pa.InPackWalleQDisconnectedCH
	pa.InPackWalleQDisconnectedCH = nil
	socket := pa.Socket
	pa.Socket = nil
	pa.runtimeMu.Unlock()
	signalAdapterStop(disconnected, 0)
	if socket != nil {
		socket.Close()
	}
	return errors.Join(shutdownErrors...)
}

func (pa *PlatformAdapterRed) RuntimeShutdown(_ context.Context) error {
	pa.runtimeMu.Lock()
	pa.runtimeStopping.Store(true)
	pa.DiceServing = false
	conn := pa.conn
	pa.conn = nil
	pa.runtimeMu.Unlock()
	if conn == nil {
		return nil
	}
	return conn.Close()
}

func (pa *PlatformAdapterOfficialQQ) RuntimeShutdown(ctx context.Context) error {
	pa.runtimeStopping.Store(true)
	pa.runtimeMu.Lock()
	cancel := pa.CancelFunc
	server := pa.webhookServer
	qrDone := pa.qrDone
	pa.runtimeMu.Unlock()
	if cancel != nil {
		cancel()
	}
	var shutdownErrors []error
	if server != nil {
		if err := shutdownRuntimeHTTPServer(ctx, server.Shutdown, server.Close); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("关闭 Official QQ webhook: %w", err))
		}
	}
	if err := waitRuntimeDone(ctx, "Official QQ 扫码任务", qrDone); err != nil {
		shutdownErrors = append(shutdownErrors, err)
	}

	pa.runtimeMu.Lock()
	cancel = pa.CancelFunc
	server = pa.webhookServer
	sessionDone := pa.sessionDone
	webhookDone := pa.webhookDone
	pa.runtimeMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if server != nil {
		if err := shutdownRuntimeHTTPServer(ctx, server.Shutdown, server.Close); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("关闭 Official QQ webhook: %w", err))
		}
	}
	if err := waitRuntimeDone(ctx, "Official QQ 会话", sessionDone); err != nil {
		shutdownErrors = append(shutdownErrors, err)
	}
	if err := waitRuntimeDone(ctx, "Official QQ webhook", webhookDone); err != nil {
		shutdownErrors = append(shutdownErrors, err)
	}
	pa.clearQrLoginState()
	pa.runtimeMu.Lock()
	pa.Ctx = nil
	pa.CancelFunc = nil
	pa.SessionManager = nil
	pa.webhookServer = nil
	pa.DiceServing = false
	pa.runtimeMu.Unlock()
	return errors.Join(shutdownErrors...)
}

func (pa *PlatformAdapterOnebot) RuntimeShutdown(ctx context.Context) error {
	pa.runtimeStopping.Store(true)
	pa.isShuttingDown.Store(true)
	var shutdownErrors []error
	if pa.echoServer != nil {
		if err := shutdownRuntimeHTTPServer(ctx, pa.echoServer.Shutdown, pa.echoServer.Close); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("关闭 OneBot 反向服务: %w", err))
		}
	}
	pa.echoServer = nil
	if pa.cancel != nil {
		pa.cancel()
		pa.cancel = nil
	}
	if pa.websocketManager != nil {
		if err := pa.websocketManager.Shutdown(); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("关闭 OneBot WebSocket: %w", err))
		}
		pa.websocketManager = nil
	}
	pa.releaseAntPool()
	if pa.groupCache != nil {
		pa.groupCache.Close()
		pa.groupCache = nil
	}
	if pa.friendRequestDedupeCache != nil {
		pa.friendRequestDedupeCache.Close()
		pa.friendRequestDedupeCache = nil
	}
	pa.sendEmitter = nil
	pa.connectionMutex.Lock()
	pa.isConnecting = false
	pa.connectionMutex.Unlock()
	return errors.Join(shutdownErrors...)
}

func shutdownRuntimeHTTPServer(
	ctx context.Context,
	shutdown func(context.Context) error,
	closeServer func() error,
) error {
	if err := shutdown(ctx); err != nil {
		return closeServer()
	}
	return nil
}

func waitRuntimeDone(ctx context.Context, name string, done <-chan struct{}) error {
	if done == nil {
		return nil
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("等待%s停止: %w", name, ctx.Err())
	}
}

func signalAdapterStop(ch chan int, value int) {
	if ch == nil {
		return
	}
	select {
	case ch <- value:
	default:
	}
}
