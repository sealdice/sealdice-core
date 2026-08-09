package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"sealdice-core/api"
	"sealdice-core/dice"
	"sealdice-core/logger"
	v2 "sealdice-core/migrate/v2"
	"sealdice-core/utils/constant"
	"sealdice-core/utils/dboperator"
)

type runtimeBuildOptions struct {
	address       string
	containerMode bool
	justForTest   bool
	uiWriter      *logger.UIWriter
}

type runtimeEndpointStart struct {
	dice     *dice.Dice
	endpoint *dice.EndPointInfo
}

type applicationRuntime struct {
	manager  *dice.DiceManager
	ctx      context.Context
	cancel   context.CancelFunc
	startMu  sync.Mutex
	prepared bool
	started  bool
	starts   []runtimeEndpointStart
}

type restoreRuntimePhase string

const (
	restorePhaseMaintenance restoreRuntimePhase = "maintenance"
	restorePhaseQuiesce     restoreRuntimePhase = "quiesce"
	restorePhasePrepare     restoreRuntimePhase = "prepare"
	restorePhaseFinalize    restoreRuntimePhase = "finalize"
	restorePhaseApply       restoreRuntimePhase = "apply"
	restorePhaseBuild       restoreRuntimePhase = "build"
	restorePhaseCommit      restoreRuntimePhase = "commit"
	restorePhasePublish     restoreRuntimePhase = "publish"
	restorePhaseMark        restoreRuntimePhase = "mark"
	restorePhaseRollback    restoreRuntimePhase = "rollback"
)

func (runtime *applicationRuntime) prepareStart() (resultErr error) {
	runtime.startMu.Lock()
	defer runtime.startMu.Unlock()
	if runtime.prepared {
		return nil
	}
	if runtime.manager == nil {
		return errors.New("runtime manager 为空")
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			resultErr = fmt.Errorf("准备 Runtime 启动任务时发生异常: %v", recovered)
		}
	}()
	for _, currentDice := range runtime.manager.DiceSnapshot() {
		if currentDice == nil || currentDice.ImSession == nil {
			continue
		}
		for _, endpoint := range diceServePrepare(currentDice) {
			runtime.starts = append(runtime.starts, runtimeEndpointStart{dice: currentDice, endpoint: endpoint})
		}
	}
	runtime.prepared = true
	return nil
}

// start only publishes already-prepared work. All validation that can fail is
// completed before the restore commit point.
func (runtime *applicationRuntime) start() {
	runtime.startMu.Lock()
	if runtime.started {
		runtime.startMu.Unlock()
		return
	}
	runtime.started = true
	starts := append([]runtimeEndpointStart(nil), runtime.starts...)
	runtime.startMu.Unlock()

	for _, item := range starts {
		currentDice := item.dice
		endpoint := item.endpoint
		runtime.manager.GoRuntime(func(ctx context.Context) {
			diceServeEndpoint(ctx, currentDice, endpoint)
		})
	}
	runtime.manager.GoRuntime(func(ctx context.Context) {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				CheckVersionContext(ctx, runtime.manager)
			}
		}
	})
	runtime.manager.GoRuntime(func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case <-runtime.manager.RebootRequestChan:
				go doReboot(runtime.manager)
				return
			case <-runtime.manager.UpdateCheckRequestChan:
				CheckVersionContext(ctx, runtime.manager)
			case currentDice := <-runtime.manager.UpdateRequestChan:
				updatePack, err := downloadUpdate(runtime.manager, currentDice.Logger)
				if err != nil {
					runtime.manager.UpdateDownloadedChan <- err.Error()
					continue
				}
				runtime.manager.UpdateDownloadedChan <- ""
				if runtime.manager.UpdateSealdiceByFile != nil {
					runtime.manager.UpdateSealdiceByFile(updatePack)
				}
			}
		}
	})
}

func (runtime *applicationRuntime) stop(ctx context.Context) error {
	if runtime == nil || runtime.manager == nil {
		return nil
	}
	return runtime.manager.Stop(ctx)
}

type runtimeSupervisor struct {
	mu               sync.Mutex
	current          *applicationRuntime
	options          runtimeBuildOptions
	listenAddress    string
	state            atomic.Value
	buildFn          func() (*applicationRuntime, error)
	validateFn       func(*applicationRuntime) error
	prepareFn        func(*dice.DiceManager) error
	applyFn          func() error
	commitFn         func() error
	markSucceededFn  func() error
	rollbackFn       func(string) error
	committedFn      func() (bool, error)
	runnableIDFn     func() (string, error)
	restoreCapableFn func(*dice.DiceManager) error
	getStatusFn      func() dice.RestoreStatus
	statusFn         func(string, string) error
	restoreRunFn     func(context.Context, string) error
	restoreQueueMu   sync.Mutex
	restoreWake      chan struct{}
	restoreQueuedID  string
	restoreActiveID  string
	restoreCancel    context.CancelFunc
	restoreDone      chan struct{}
	restoreOnce      sync.Once
	restoreStopped   bool
}

func newRuntimeSupervisor(options runtimeBuildOptions) *runtimeSupervisor {
	supervisor := &runtimeSupervisor{options: options, restoreWake: make(chan struct{}, 1)}
	supervisor.state.Store("stopped")
	supervisor.buildFn = supervisor.build
	supervisor.validateFn = validateApplicationRuntime
	supervisor.prepareFn = dice.PrepareScheduledRestore
	supervisor.applyFn = dice.ApplyScheduledRestore
	supervisor.commitFn = dice.CommitScheduledRestore
	supervisor.markSucceededFn = dice.MarkScheduledRestoreSucceeded
	supervisor.rollbackFn = dice.RollbackScheduledRestore
	supervisor.committedFn = dice.HasCommittedRestore
	supervisor.runnableIDFn = dice.RunnableScheduledRestoreOperationID
	supervisor.restoreCapableFn = validateRestoreRuntime
	supervisor.getStatusFn = dice.GetRestoreStatus
	supervisor.statusFn = dice.UpdateRestoreStatusState
	supervisor.restoreRunFn = supervisor.restore
	return supervisor
}

func (supervisor *runtimeSupervisor) build() (runtimeResult *applicationRuntime, resultErr error) {
	ctx, cancel := context.WithCancel(context.Background())
	operator, err := dboperator.NewDatabaseOperator(context.Background())
	if err != nil {
		cancel()
		return nil, fmt.Errorf("初始化数据库: %w", err)
	}
	manager := &dice.DiceManager{Operator: operator, ContainerMode: supervisor.options.containerMode}
	manager.SetRuntimeContext(ctx, cancel)
	runtimeResult = &applicationRuntime{manager: manager, ctx: ctx, cancel: cancel}
	defer func() {
		if recovered := recover(); recovered != nil {
			resultErr = fmt.Errorf("初始化 Runtime 时发生异常: %v", recovered)
		}
		if resultErr != nil {
			cleanupErr := discardRuntime(runtimeResult)
			resultErr = errors.Join(resultErr, cleanupErr)
			runtimeResult = nil
		}
	}()

	manager.LoadDice()
	if err = v2.InitUpgrader(operator); err != nil {
		return nil, fmt.Errorf("执行数据库迁移: %w", err)
	}
	manager.TryCreateDefault()
	configs := make([]dice.BaseConfig, 0, len(manager.DiceSnapshot()))
	for _, instance := range manager.DiceSnapshot() {
		if instance == nil {
			return nil, errors.New("runtime 包含空 Dice 实例")
		}
		configs = append(configs, instance.BaseConfig)
	}
	if err = dice.ValidateDiceConfigNames(configs); err != nil {
		return nil, fmt.Errorf("校验 Dice 数据目录名称: %w", err)
	}
	configuredAddress := manager.ServeAddress
	listenAddress := supervisor.listenAddress
	if listenAddress == "" {
		listenAddress = supervisor.options.address
	}
	if listenAddress == "" {
		listenAddress = configuredAddress
	}
	manager.SetRuntimeServeAddress(listenAddress, configuredAddress)
	manager.InitDice(supervisor.options.uiWriter)
	manager.JustForTest = supervisor.options.justForTest
	manager.UpdateSealdiceByFile = func(packName string) bool {
		if checkErr := CheckUpdater(manager); checkErr != nil {
			logger.M().Error("升级程序检查失败: ", checkErr.Error())
			return false
		}
		return UpdateByFile(manager, packName, false)
	}
	manager.IsReady = true
	return runtimeResult, nil
}

func validateApplicationRuntime(runtime *applicationRuntime) error {
	if runtime == nil || runtime.manager == nil {
		return errors.New("runtime 构造结果为空")
	}
	if runtime.manager.Operator == nil {
		return errors.New("runtime 数据库未初始化")
	}
	if !runtime.manager.IsReady {
		return errors.New("runtime manager 尚未就绪")
	}
	instances := runtime.manager.DiceSnapshot()
	if len(instances) == 0 || instances[0] == nil {
		return errors.New("runtime 未包含可发布的 Dice 实例")
	}
	primary := instances[0]
	if primary.Parent != runtime.manager || primary.ImSession == nil || primary.ImSession.Parent != primary {
		return errors.New("runtime Dice 实例关联不完整")
	}
	configs := make([]dice.BaseConfig, 0, len(instances))
	for _, instance := range instances {
		if instance == nil {
			return errors.New("runtime 包含空 Dice 实例")
		}
		configs = append(configs, instance.BaseConfig)
	}
	if err := dice.ValidateDiceConfigNames(configs); err != nil {
		return fmt.Errorf("runtime Dice 名称不安全: %w", err)
	}
	if runtime.manager.IsQuiesced() {
		return errors.New("runtime 在发布前已经进入关闭阶段")
	}
	return nil
}

func (supervisor *runtimeSupervisor) buildValidatedRuntime() (
	runtimeInstance *applicationRuntime,
	resultErr error,
) {
	defer func() {
		if recovered := recover(); recovered != nil {
			resultErr = errors.Join(
				fmt.Errorf("runtime 构造或验证异常: %v", recovered),
				discardRuntime(runtimeInstance),
			)
			runtimeInstance = nil
		}
	}()

	var err error
	runtimeInstance, err = supervisor.buildFn()
	if err != nil {
		return nil, err
	}
	if err = supervisor.validateFn(runtimeInstance); err != nil {
		return nil, errors.Join(err, discardRuntime(runtimeInstance))
	}
	if err = runtimeInstance.prepareStart(); err != nil {
		return nil, errors.Join(err, discardRuntime(runtimeInstance))
	}
	return runtimeInstance, nil
}

func (supervisor *runtimeSupervisor) publish(runtimeInstance *applicationRuntime) {
	if supervisor.listenAddress == "" {
		supervisor.listenAddress = runtimeInstance.manager.ServeAddress
	}
	supervisor.current = runtimeInstance
	api.ReplaceRuntime(runtimeInstance.manager)
	runtimeInstance.start()
}

func (supervisor *runtimeSupervisor) start() (*dice.DiceManager, error) {
	supervisor.mu.Lock()
	defer supervisor.mu.Unlock()
	if supervisor.current != nil {
		return supervisor.current.manager, nil
	}
	supervisor.state.Store("starting")
	runtimeInstance, err := supervisor.buildValidatedRuntime()
	if err != nil {
		api.ReplaceRuntime(nil)
		supervisor.state.Store("degraded")
		_ = supervisor.statusFn("degraded", "Runtime 启动失败，数据未发布: "+err.Error())
		return nil, err
	}
	supervisor.publish(runtimeInstance)
	supervisor.state.Store("running")
	committed, committedErr := supervisor.committedFn()
	if committedErr != nil {
		supervisor.state.Store("degraded")
		_ = supervisor.statusFn("degraded", "Runtime 已启动，但无法确认已提交恢复事务: "+committedErr.Error())
		logger.M().Errorw("[备份恢复] Runtime 已启动，但无法确认已提交事务", "error", committedErr)
	} else if committed {
		if markErr := supervisor.markSucceededFn(); markErr != nil {
			supervisor.state.Store("degraded")
			_ = supervisor.statusFn("degraded", "数据已提交且 Runtime 已启动，但成功状态清理失败: "+markErr.Error())
			logger.M().Errorw("[备份恢复] 已提交事务成功状态清理失败", "error", markErr)
		}
	}
	return runtimeInstance.manager, nil
}

func (supervisor *runtimeSupervisor) restore(ctx context.Context, expectedOperationID string) (resultErr error) {
	supervisor.mu.Lock()
	defer supervisor.mu.Unlock()
	restoreStatus := supervisor.getStatusFn()
	runnableOperationID, runnableErr := supervisor.runnableIDFn()
	if runnableErr != nil {
		return supervisor.degradeRunning("检查可续跑恢复任务失败", runnableErr)
	}
	if expectedOperationID != "" && runnableOperationID != "" && expectedOperationID != runnableOperationID {
		logger.M().Warnw(
			"[备份恢复] 丢弃过期的 Runtime 恢复排队项",
			"queuedOperationId", expectedOperationID,
			"currentOperationId", runnableOperationID,
		)
		return nil
	}
	if isTerminalRestoreState(restoreStatus.State) && runnableOperationID == "" {
		return nil
	}
	committed, committedErr := supervisor.committedFn()
	if committedErr != nil {
		return fmt.Errorf("确认恢复提交状态: %w", committedErr)
	}
	if committed {
		if supervisor.current == nil {
			return errors.New("恢复数据已提交，但 Runtime 尚未发布")
		}
		return supervisor.markSucceededFn()
	}
	if runnableOperationID == "" {
		return errors.New("没有可执行的恢复任务")
	}
	if expectedOperationID != "" && expectedOperationID != runnableOperationID {
		return nil
	}
	if supervisor.current == nil {
		return errors.New("runtime 尚未启动")
	}
	if err := supervisor.restoreCapableFn(supervisor.current.manager); err != nil {
		return supervisor.degradeRunning("当前 Runtime 不支持执行本地恢复任务", err)
	}
	if ctx == nil {
		ctx = context.Background()
	}

	operationID := runnableOperationID
	sourceName := restoreStatus.SourceName
	logger.M().Infow("[备份恢复] 开始执行 Runtime 恢复", "operationId", operationID, "source", sourceName)
	if err := api.BeginRuntimeMaintenanceContext(ctx); err != nil {
		return fmt.Errorf("等待 Runtime API 请求排空: %w", err)
	}
	defer api.EndRuntimeMaintenance()

	oldRuntime := supervisor.current
	var newRuntime *applicationRuntime
	phase := restorePhaseMaintenance
	defer func() {
		if recovered := recover(); recovered != nil {
			resultErr = supervisor.handleRestorePanic(
				phase,
				operationID,
				oldRuntime,
				newRuntime,
				recovered,
			)
		}
	}()

	phase = restorePhaseQuiesce
	supervisor.state.Store("quiescing")
	_ = supervisor.statusFn("quiescing", "正在停止新输入并刷新 Runtime 数据")
	phaseCtx, phaseCancel := context.WithTimeout(ctx, 30*time.Second)
	quiesceErr := oldRuntime.manager.Quiesce(phaseCtx)
	phaseCancel()
	if quiesceErr != nil {
		return supervisor.degradeStopped("静默当前 Runtime 失败，已拒绝应用备份", quiesceErr)
	}

	phase = restorePhasePrepare
	if err := supervisor.prepareFn(oldRuntime.manager); err != nil {
		phase = restorePhaseFinalize
		phaseCtx, phaseCancel = context.WithTimeout(context.Background(), 30*time.Second)
		finalizeErr := oldRuntime.manager.Finalize(phaseCtx)
		phaseCancel()
		if finalizeErr != nil {
			return supervisor.degradeStopped("准备恢复失败，且当前 Runtime 无法完全释放", errors.Join(err, finalizeErr))
		}
		supervisor.current = nil
		api.ReplaceRuntime(nil)
		phase = restorePhaseRollback
		return supervisor.rollbackAndRestart("准备恢复失败", err)
	}

	phase = restorePhaseFinalize
	phaseCtx, phaseCancel = context.WithTimeout(ctx, 30*time.Second)
	finalizeErr := oldRuntime.manager.Finalize(phaseCtx)
	phaseCancel()
	if finalizeErr != nil {
		return supervisor.degradeStopped("释放当前 Runtime 失败，已拒绝应用备份", finalizeErr)
	}
	supervisor.current = nil
	api.ReplaceRuntime(nil)

	phase = restorePhaseApply
	supervisor.state.Store("applying")
	if err := supervisor.applyFn(); err != nil {
		phase = restorePhaseRollback
		return supervisor.rollbackAndRestart("应用备份失败", err)
	}

	phase = restorePhaseBuild
	supervisor.state.Store("starting")
	_ = supervisor.statusFn("starting", "备份已应用，正在验证新 Runtime")
	var err error
	newRuntime, err = supervisor.buildValidatedRuntime()
	if err != nil {
		phase = restorePhaseRollback
		return supervisor.rollbackAndRestart("恢复后的 Runtime 初始化失败", err)
	}

	phase = restorePhaseCommit
	if err = supervisor.commitFn(); err != nil {
		cleanupErr := discardRuntime(newRuntime)
		return supervisor.degradeStopped(
			"恢复提交结果不确定，数据可能已提交，禁止回滚",
			errors.Join(err, cleanupErr),
		)
	}

	// Commit is the irreversible boundary. From this point onward no rollback is
	// allowed; publish/start consists only of prevalidated, non-failing steps.
	phase = restorePhasePublish
	supervisor.publish(newRuntime)
	phase = restorePhaseMark
	if err = supervisor.markSucceededFn(); err != nil {
		supervisor.state.Store("degraded")
		message := "数据已提交且 Runtime 已发布，但成功状态清理失败: " + err.Error()
		_ = supervisor.statusFn("degraded", message)
		logger.M().Errorw("[备份恢复] 提交后的状态清理失败，禁止回滚", "operationId", operationID, "error", err)
		return errors.New(message)
	}
	supervisor.state.Store("running")
	logger.M().Infow("[备份恢复] 恢复成功，Runtime 已重新运行", "operationId", operationID, "source", sourceName)
	return nil
}

func (supervisor *runtimeSupervisor) enqueueRunnableScheduledRestore() error {
	operationID, err := supervisor.runnableIDFn()
	if err != nil {
		supervisor.mu.Lock()
		defer supervisor.mu.Unlock()
		return supervisor.degradeRunning("检查启动时待续跑恢复任务失败", err)
	}
	if operationID == "" {
		return nil
	}
	if !supervisor.enqueueRestore(operationID) {
		return errors.New("runtime 恢复队列已经停止")
	}
	return nil
}

func (supervisor *runtimeSupervisor) enqueueRestore(operationID string) bool {
	if operationID == "" {
		return false
	}
	supervisor.startRestoreWorker()
	supervisor.restoreQueueMu.Lock()
	if supervisor.restoreStopped {
		supervisor.restoreQueueMu.Unlock()
		return false
	}
	if supervisor.restoreActiveID == operationID || supervisor.restoreQueuedID == operationID {
		supervisor.restoreQueueMu.Unlock()
		return true
	}
	supervisor.restoreQueuedID = operationID
	supervisor.restoreQueueMu.Unlock()
	select {
	case supervisor.restoreWake <- struct{}{}:
	default:
	}
	return true
}

func (supervisor *runtimeSupervisor) startRestoreWorker() {
	supervisor.restoreOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		supervisor.restoreQueueMu.Lock()
		if supervisor.restoreStopped {
			supervisor.restoreQueueMu.Unlock()
			cancel()
			return
		}
		supervisor.restoreCancel = cancel
		supervisor.restoreDone = done
		supervisor.restoreQueueMu.Unlock()
		go supervisor.restoreWorker(ctx, done)
	})
}

func (supervisor *runtimeSupervisor) restoreWorker(ctx context.Context, done chan<- struct{}) {
	defer close(done)
	for {
		select {
		case <-ctx.Done():
			return
		case <-supervisor.restoreWake:
		}
		for {
			supervisor.restoreQueueMu.Lock()
			operationID := supervisor.restoreQueuedID
			supervisor.restoreQueuedID = ""
			supervisor.restoreActiveID = operationID
			supervisor.restoreQueueMu.Unlock()
			if operationID == "" {
				break
			}

			if err := supervisor.runRestoreQueueItem(ctx, operationID); err != nil {
				logger.M().Errorw(
					"[备份恢复] Runtime 恢复流程结束，但恢复未成功",
					"operationId", operationID,
					"error", err,
				)
			}
			supervisor.restoreQueueMu.Lock()
			supervisor.restoreActiveID = ""
			hasQueued := supervisor.restoreQueuedID != ""
			supervisor.restoreQueueMu.Unlock()
			if !hasQueued {
				break
			}
		}
	}
}

func (supervisor *runtimeSupervisor) runRestoreQueueItem(ctx context.Context, operationID string) (resultErr error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			resultErr = supervisor.failClosedRestorePanic(operationID, recovered)
		}
	}()
	return supervisor.restoreRunFn(ctx, operationID)
}

func (supervisor *runtimeSupervisor) handleRestorePanic(
	phase restoreRuntimePhase,
	operationID string,
	oldRuntime *applicationRuntime,
	newRuntime *applicationRuntime,
	recovered any,
) error {
	panicErr := fmt.Errorf("runtime 恢复在 %s 阶段异常: %v", phase, recovered)
	message := panicErr.Error()
	api.ReplaceRuntime(nil)
	supervisor.state.Store("degraded")
	supervisor.updateRestoreStatusSafely("degraded", message)
	logger.M().Errorw(
		"[备份恢复] Runtime 恢复阶段发生异常",
		"operationId", operationID,
		"phase", phase,
		"error", panicErr,
	)

	switch phase {
	case restorePhasePrepare:
		if oldRuntime == nil || oldRuntime.manager == nil {
			return panicErr
		}
		finalizeCtx, finalizeCancel := context.WithTimeout(context.Background(), 30*time.Second)
		finalizeErr := oldRuntime.manager.Finalize(finalizeCtx)
		finalizeCancel()
		if finalizeErr != nil {
			return errors.Join(panicErr, fmt.Errorf("prepare 异常后释放旧 Runtime: %w", finalizeErr))
		}
		supervisor.current = nil
		return supervisor.rollbackAndRestart("准备恢复时发生异常", panicErr)

	case restorePhaseApply, restorePhaseBuild:
		cleanupErr := discardRuntime(newRuntime)
		supervisor.current = nil
		return supervisor.rollbackAndRestart(
			fmt.Sprintf("恢复 %s 阶段发生异常", phase),
			errors.Join(panicErr, cleanupErr),
		)

	case restorePhaseCommit:
		cleanupErr := discardRuntime(newRuntime)
		supervisor.current = nil
		message = "恢复提交结果不确定，数据可能已提交，禁止回滚"
		supervisor.updateRestoreStatusSafely("degraded", message+": "+panicErr.Error())
		return errors.Join(fmt.Errorf("%s: %w", message, panicErr), cleanupErr)

	case restorePhasePublish, restorePhaseMark:
		failedRuntime := supervisor.current
		if failedRuntime == nil {
			failedRuntime = newRuntime
		}
		supervisor.current = nil
		cleanupErr := discardRuntime(failedRuntime)
		message = "数据已提交，但发布后的 Runtime 发生异常，禁止回滚并保持不可用"
		supervisor.updateRestoreStatusSafely("degraded", message+": "+panicErr.Error())
		return errors.Join(fmt.Errorf("%s: %w", message, panicErr), cleanupErr)

	case restorePhaseRollback:
		failedRuntime := supervisor.current
		supervisor.current = nil
		cleanupErr := discardRuntime(failedRuntime)
		message = "回滚后的 Runtime 发布异常，保持不可用"
		supervisor.updateRestoreStatusSafely("degraded", message+": "+panicErr.Error())
		return errors.Join(fmt.Errorf("%s: %w", message, panicErr), cleanupErr)

	default:
		return panicErr
	}
}

func (supervisor *runtimeSupervisor) failClosedRestorePanic(operationID string, recovered any) error {
	panicErr := fmt.Errorf("runtime 恢复 worker 异常: %v", recovered)
	supervisor.mu.Lock()
	defer supervisor.mu.Unlock()

	api.BeginRuntimeMaintenance()
	defer func() {
		api.ReplaceRuntime(nil)
		api.EndRuntimeMaintenance()
	}()
	api.ReplaceRuntime(nil)

	supervisor.state.Store("degraded")
	supervisor.updateRestoreStatusSafely("degraded", panicErr.Error())
	logger.M().Errorw(
		"[备份恢复] Runtime 恢复 worker 触发 fail-closed",
		"operationId", operationID,
		"error", panicErr,
	)
	return panicErr
}

func (supervisor *runtimeSupervisor) updateRestoreStatusSafely(state string, message string) {
	defer func() {
		if recovered := recover(); recovered != nil {
			logger.M().Errorw(
				"[备份恢复] 写入恢复状态时发生异常",
				"state", state,
				"error", recovered,
			)
		}
	}()
	if err := supervisor.statusFn(state, message); err != nil {
		logger.M().Errorw("[备份恢复] 写入恢复状态失败", "state", state, "error", err)
	}
}

func (supervisor *runtimeSupervisor) stopRestoreWorker(ctx context.Context) error {
	supervisor.restoreQueueMu.Lock()
	supervisor.restoreStopped = true
	cancel := supervisor.restoreCancel
	done := supervisor.restoreDone
	supervisor.restoreQueueMu.Unlock()
	if cancel == nil || done == nil {
		return nil
	}
	cancel()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("等待 runtime 恢复 worker 停止: %w", ctx.Err())
	}
}

func isTerminalRestoreState(state string) bool {
	switch state {
	case "succeeded", "failed", "rolled_back", "degraded":
		return true
	default:
		return false
	}
}

func validateRestoreRuntime(manager *dice.DiceManager) error {
	if manager == nil || manager.Operator == nil {
		return errors.New("runtime 数据库未初始化")
	}
	databaseType := manager.Operator.Type()
	if databaseType != constant.SQLITE {
		return fmt.Errorf("恢复仅支持 SQLite，当前数据库类型为 %q", databaseType)
	}
	return nil
}

func (supervisor *runtimeSupervisor) rollbackAndRestart(prefix string, cause error) error {
	message := fmt.Sprintf("%s: %v", prefix, cause)
	operationID := supervisor.getStatusFn().OperationID
	supervisor.state.Store("rolling_back")
	_ = supervisor.statusFn("rolling_back", message)
	rollbackErr := supervisor.rollbackFn(message)
	if rollbackErr != nil {
		return supervisor.degradeStopped(
			message+"；回滚失败，禁止构造 Runtime",
			errors.Join(cause, rollbackErr),
		)
	}

	fallback, rebuildErr := supervisor.buildValidatedRuntime()
	if rebuildErr != nil {
		return supervisor.degradeStopped(message+"；回滚后的 Runtime 无法启动", errors.Join(cause, rebuildErr))
	}
	supervisor.publish(fallback)
	supervisor.state.Store("running")
	logger.M().Warnw("[备份恢复] 恢复未完成，已回滚并重新发布原 Runtime", "operationId", operationID, "stage", prefix)
	return cause
}

func (supervisor *runtimeSupervisor) degradeStopped(message string, cause error) error {
	api.ReplaceRuntime(nil)
	supervisor.state.Store("degraded")
	_ = supervisor.statusFn("degraded", message+": "+cause.Error())
	logger.M().Errorw("[备份恢复] Runtime 进入 degraded", "message", message, "error", cause)
	return fmt.Errorf("%s: %w", message, cause)
}

func (supervisor *runtimeSupervisor) degradeRunning(message string, cause error) error {
	supervisor.state.Store("degraded")
	_ = supervisor.statusFn("degraded", message+": "+cause.Error())
	logger.M().Errorw("[备份恢复] Runtime 保持运行但恢复状态进入 degraded", "message", message, "error", cause)
	return fmt.Errorf("%s: %w", message, cause)
}

func discardRuntime(runtimeInstance *applicationRuntime) error {
	if runtimeInstance == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return runtimeInstance.stop(ctx)
}

func (supervisor *runtimeSupervisor) stop(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := supervisor.stopRestoreWorker(ctx); err != nil {
		return err
	}
	supervisor.mu.Lock()
	defer supervisor.mu.Unlock()

	if err := api.BeginRuntimeMaintenanceContext(ctx); err != nil {
		return fmt.Errorf("等待 Runtime API 请求排空: %w", err)
	}
	defer func() {
		api.ReplaceRuntime(nil)
		api.EndRuntimeMaintenance()
	}()
	api.ReplaceRuntime(nil)

	if supervisor.current == nil {
		supervisor.state.Store("stopped")
		return nil
	}
	supervisor.state.Store("stopping")
	err := supervisor.current.stop(ctx)
	if err != nil {
		supervisor.state.Store("degraded")
		return err
	}
	supervisor.current = nil
	supervisor.state.Store("stopped")
	return nil
}
