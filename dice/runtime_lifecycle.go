package dice

import (
	"context"
	"fmt"
	"runtime/debug"
)

// Stop 停止当前 DiceManager 拥有的任务、适配器和数据库。
// 它不处理进程锁、日志或 WebUI，这些资源属于进程级控制平面。
func (dm *DiceManager) Stop(ctx context.Context) error {
	if !dm.CleanupFlag.CompareAndSwap(0, 1) {
		return nil
	}

	if dm.runtimeCancel != nil {
		dm.runtimeCancel()
	}
	if dm.Cron != nil {
		stopCtx := dm.Cron.Stop()
		select {
		case <-stopCtx.Done():
		case <-ctx.Done():
			return fmt.Errorf("等待管理器定时任务停止: %w", ctx.Err())
		}
	}

	for _, d := range dm.Dice {
		if d == nil {
			continue
		}
		d.stopAdapters()
		if d.Cron != nil {
			stopCtx := d.Cron.Stop()
			select {
			case <-stopCtx.Done():
			case <-ctx.Done():
				return fmt.Errorf("等待骰子定时任务停止: %w", ctx.Err())
			}
		}
		if d.JsScriptCron != nil {
			d.JsScriptCron.Stop()
		}
		if d.ExtLoopManager != nil {
			d.jsClear()
		}
		if d.AttrsManager != nil {
			d.AttrsManager.Stop()
		}
		for _, ext := range d.ExtList {
			if ext != nil && ext.Storage != nil {
				if err := ext.StorageClose(); err != nil {
					d.Logger.Errorf("关闭扩展存储失败: %v", err)
				}
			}
		}
		d.IsAlreadyLoadConfig = false
	}

	waitDone := make(chan struct{})
	go func() {
		dm.runtimeWG.Wait()
		close(waitDone)
	}()
	select {
	case <-waitDone:
	case <-ctx.Done():
		return fmt.Errorf("等待 Runtime 后台任务停止: %w", ctx.Err())
	}

	if dm.Help != nil {
		dm.Help.Close()
	}
	if dm.Operator != nil {
		dm.Operator.Close()
	}
	dm.IsReady = false
	return nil
}

func (d *Dice) stopAdapters() {
	if d.IsAlreadyLoadConfig {
		d.Config.BanList.SaveChanged(d)
		d.Save(true)
	}
	if d.ImSession == nil {
		return
	}
	for _, endpoint := range d.ImSession.EndPoints {
		if endpoint == nil {
			continue
		}
		wasEnabled := endpoint.Enable
		if endpoint.Adapter != nil {
			func() {
				defer func() {
					if recovered := recover(); recovered != nil && d.Logger != nil {
						d.Logger.Errorf("关闭适配器失败: %v\n%s", recovered, debug.Stack())
					}
				}()
				endpoint.Adapter.SetEnable(false)
			}()
		}
		BuiltinQQServeProcessKill(d, endpoint)
		endpoint.Enable = wasEnabled
	}
	// SetEnable(false) 可能会保存禁用状态，恢复原来的启用配置后再次落盘。
	if d.IsAlreadyLoadConfig {
		d.Save(true)
	}
}

// IsStopped 供生命周期测试和健康检查使用。
func (dm *DiceManager) IsStopped() bool {
	return dm.CleanupFlag.Load() != 0
}
