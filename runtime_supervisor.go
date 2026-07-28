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
	"sealdice-core/utils/dboperator"
)

type runtimeBuildOptions struct {
	address       string
	containerMode bool
	justForTest   bool
	uiWriter      *logger.UIWriter
}

type applicationRuntime struct {
	manager *dice.DiceManager
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

func (runtime *applicationRuntime) goTask(task func(context.Context)) {
	runtime.wg.Add(1)
	go func() {
		defer runtime.wg.Done()
		task(runtime.ctx)
	}()
}

func (runtime *applicationRuntime) start() {
	for _, currentDice := range runtime.manager.Dice {
		d := currentDice
		for _, currentEndpoint := range diceServePrepare(d) {
			endpoint := currentEndpoint
			runtime.goTask(func(ctx context.Context) { diceServeEndpoint(ctx, d, endpoint) })
		}
	}
	runtime.goTask(func(ctx context.Context) {
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
	runtime.goTask(func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case <-runtime.manager.RebootRequestChan:
				doReboot(runtime.manager)
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
	runtime.cancel()
	if err := runtime.manager.Stop(ctx); err != nil {
		return err
	}
	done := make(chan struct{})
	go func() {
		runtime.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type runtimeSupervisor struct {
	mu         sync.Mutex
	current    *applicationRuntime
	options    runtimeBuildOptions
	state      atomic.Value
	buildFn    func() (*applicationRuntime, error)
	applyFn    func() error
	commitFn   func() error
	rollbackFn func(string) error
	statusFn   func(string, string) error
}

func newRuntimeSupervisor(options runtimeBuildOptions) *runtimeSupervisor {
	supervisor := &runtimeSupervisor{options: options}
	supervisor.state.Store("stopped")
	supervisor.buildFn = supervisor.build
	supervisor.applyFn = dice.ApplyScheduledRestore
	supervisor.commitFn = dice.CommitScheduledRestore
	supervisor.rollbackFn = dice.RollbackScheduledRestore
	supervisor.statusFn = dice.UpdateRestoreStatusState
	return supervisor
}

func (supervisor *runtimeSupervisor) build() (runtimeResult *applicationRuntime, resultErr error) {
	ctx, cancel := context.WithCancel(context.Background())
	operator, err := dboperator.NewDatabaseOperator(ctx)
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
			stopCtx, stopCancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer stopCancel()
			_ = runtimeResult.stop(stopCtx)
			runtimeResult = nil
		}
	}()

	manager.LoadDice()
	manager.IsReady = true
	if supervisor.options.address != "" {
		manager.ServeAddress = supervisor.options.address
	}
	if err = v2.InitUpgrader(operator); err != nil {
		return nil, fmt.Errorf("执行数据库迁移: %w", err)
	}
	manager.TryCreateDefault()
	manager.InitDice(supervisor.options.uiWriter)
	manager.JustForTest = supervisor.options.justForTest
	manager.UpdateSealdiceByFile = func(packName string) bool {
		if checkErr := CheckUpdater(manager); checkErr != nil {
			logger.M().Error("升级程序检查失败: ", checkErr.Error())
			return false
		}
		return UpdateByFile(manager, packName, false)
	}
	runtimeResult.start()
	return runtimeResult, nil
}

func (supervisor *runtimeSupervisor) start() (*dice.DiceManager, error) {
	supervisor.mu.Lock()
	defer supervisor.mu.Unlock()
	if supervisor.current != nil {
		return supervisor.current.manager, nil
	}
	supervisor.state.Store("starting")
	runtimeInstance, err := supervisor.buildFn()
	if err != nil {
		supervisor.state.Store("degraded")
		return nil, err
	}
	supervisor.current = runtimeInstance
	supervisor.state.Store("running")
	return runtimeInstance.manager, nil
}

func (supervisor *runtimeSupervisor) restore(_ context.Context) (resultErr error) {
	supervisor.mu.Lock()
	defer supervisor.mu.Unlock()
	if supervisor.current == nil {
		return errors.New("runtime 尚未启动")
	}
	restoreStatus := dice.GetRestoreStatus()
	operationID := restoreStatus.OperationID
	sourceName := restoreStatus.SourceName
	logger.M().Infow("[备份恢复] 开始执行 Runtime 恢复", "operationId", operationID, "source", sourceName)

	api.BeginRuntimeMaintenance()
	defer api.EndRuntimeMaintenance()
	oldRuntime := supervisor.current
	supervisor.state.Store("quiescing")
	_ = supervisor.statusFn("quiescing", "正在停止数据库、适配器和后台任务")
	logger.M().Infow("[备份恢复] 正在停止数据库、适配器和后台任务", "operationId", operationID)
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
	stopErr := oldRuntime.stop(stopCtx)
	stopCancel()
	if stopErr != nil {
		supervisor.current = nil
		api.ReplaceRuntime(nil)
		supervisor.state.Store("degraded")
		_ = supervisor.statusFn("degraded", "停止当前 Runtime 失败，已拒绝继续恢复: "+stopErr.Error())
		logger.M().Errorw("[备份恢复] 停止 Runtime 失败，已拒绝应用备份", "operationId", operationID, "error", stopErr)
		return stopErr
	}
	supervisor.current = nil
	api.ReplaceRuntime(nil)
	logger.M().Infow("[备份恢复] 原 Runtime 已完全停止", "operationId", operationID)

	supervisor.state.Store("applying")
	logger.M().Infow("[备份恢复] 正在应用备份文件", "operationId", operationID, "source", sourceName)
	if err := supervisor.applyFn(); err != nil {
		return supervisor.rebuildAfterFailure("应用备份失败", err)
	}
	logger.M().Infow("[备份恢复] 备份文件已应用", "operationId", operationID)

	supervisor.state.Store("starting")
	_ = supervisor.statusFn("starting", "备份已应用，正在初始化新 Runtime")
	logger.M().Infow("[备份恢复] 正在初始化恢复后的 Runtime", "operationId", operationID)
	newRuntime, err := supervisor.buildFn()
	if err != nil {
		return supervisor.rebuildAfterFailure("恢复后的 Runtime 初始化失败", err)
	}
	logger.M().Infow("[备份恢复] 恢复后的 Runtime 已初始化", "operationId", operationID)
	logger.M().Infow("[备份恢复] 正在提交恢复事务", "operationId", operationID)
	if err = supervisor.commitFn(); err != nil {
		stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		_ = newRuntime.stop(stopCtx)
		cancel()
		return supervisor.rebuildAfterFailure("提交恢复事务失败", err)
	}
	supervisor.current = newRuntime
	api.ReplaceRuntime(newRuntime.manager)
	supervisor.state.Store("running")
	logger.M().Infow("[备份恢复] 恢复成功，Runtime 已重新运行", "operationId", operationID, "source", sourceName)
	return nil
}

func (supervisor *runtimeSupervisor) rebuildAfterFailure(prefix string, cause error) error {
	message := fmt.Sprintf("%s: %v", prefix, cause)
	restoreStatus := dice.GetRestoreStatus()
	operationID := restoreStatus.OperationID
	logger.M().Errorw("[备份恢复] 恢复阶段失败，开始回滚", "operationId", operationID, "stage", prefix, "error", cause)
	supervisor.state.Store("rolling_back")
	_ = supervisor.statusFn("rolling_back", message)
	rollbackErr := supervisor.rollbackFn(message)
	if rollbackErr != nil {
		logger.M().Errorw("[备份恢复] 回滚恢复事务失败", "operationId", operationID, "error", rollbackErr)
	} else {
		logger.M().Infow("[备份恢复] 原数据已回滚", "operationId", operationID)
	}
	logger.M().Infow("[备份恢复] 正在重新初始化回滚后的 Runtime", "operationId", operationID)
	oldRuntime, rebuildErr := supervisor.buildFn()
	if rebuildErr != nil {
		supervisor.state.Store("degraded")
		_ = supervisor.statusFn("degraded", message+"；回滚后的 Runtime 也无法启动: "+rebuildErr.Error())
		logger.M().Errorw("[备份恢复] 回滚后的 Runtime 初始化失败", "operationId", operationID, "error", rebuildErr)
		return errors.Join(cause, rollbackErr, rebuildErr)
	}
	supervisor.current = oldRuntime
	api.ReplaceRuntime(oldRuntime.manager)
	supervisor.state.Store("running")
	logger.M().Warnw("[备份恢复] 恢复未完成，已回滚原数据并恢复 Runtime", "operationId", operationID, "stage", prefix)
	return errors.Join(cause, rollbackErr)
}

func (supervisor *runtimeSupervisor) stop(ctx context.Context) error {
	supervisor.mu.Lock()
	defer supervisor.mu.Unlock()
	if supervisor.current == nil {
		return nil
	}
	supervisor.state.Store("stopping")
	err := supervisor.current.stop(ctx)
	supervisor.current = nil
	supervisor.state.Store("stopped")
	return err
}
