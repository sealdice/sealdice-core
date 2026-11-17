package upgrade

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/danielgtaylor/huma/v2"
	cp "github.com/otiai10/copy"

	"sealdice-core/dice"
	"sealdice-core/model/common/request"
	"sealdice-core/model/common/response"
)

// Service 在线升级服务。
// 版本检测、下载与升级由 main 包注册的监听协程和 DiceManager.UpdateSealdiceByFile 完成，
// 这里只负责通过 DiceManager 上的通道驱动它们，与 `.master checkupdate` 走同一条链路。
type Service struct {
	dm               *dice.DiceManager
	versionCheckWait time.Duration
	downloadWait     time.Duration
	backupFn         func(sel dice.BackupSelection, fromAuto bool) (string, error)
	running          atomic.Bool
}

func NewService(dm *dice.DiceManager) *Service {
	return &Service{
		dm:               dm,
		versionCheckWait: 3 * time.Second,
		downloadWait:     10 * time.Minute,
		backupFn:         dm.Backup,
	}
}

func (s *Service) RegisterRoutes(_ *huma.Group) {}

func (s *Service) RegisterProtectedRoutes(grp *huma.Group) {
	huma.Post(grp, "/exec", s.Exec, func(o *huma.Operation) {
		o.Description = "检查新版本并启动在线升级"
	})
}

// Exec 刷新线上版本，有新版本时先备份，再下载并启动升级。
func (s *Service) Exec(ctx context.Context, _ *request.Empty) (*response.ItemResponse[ExecResp], error) {
	dm := s.dm
	if dm.UpdateSealdiceByFile == nil {
		return nil, huma.Error500InternalServerError("骰子没有正确初始化，无法使用此功能")
	}
	if dm.ContainerMode {
		return nil, huma.Error400BadRequest("容器模式下禁止更新，请手动拉取最新镜像")
	}
	d := dm.GetDice()
	if d == nil {
		return nil, huma.Error500InternalServerError("Dice实例不存在")
	}
	// 下载通道一次只服务一个请求，并发进入会互相抢走结果。
	if !s.running.CompareAndSwap(false, true) {
		return failed("升级正在进行中"), nil
	}
	defer s.running.Store(false)

	if err := send(ctx, dm.UpdateCheckRequestChan, 1); err != nil {
		return nil, err
	}
	select {
	case <-time.After(s.versionCheckWait):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	online := dm.AppVersionOnline
	if online == nil {
		return failed("无法获取线上版本，请稍后重试"), nil
	}
	if online.VersionLatestCode == dm.AppVersionCode {
		return failed("当前已是最新版本"), nil
	}

	// 升级进程在下载完成后很快接管并退出，备份必须在触发下载之前完成。
	s.backup(d)

	// 排空上一次遗留的下载结果（例如调用方中途放弃），避免误当成本次结果。
	select {
	case <-dm.UpdateDownloadedChan:
	default:
	}

	if err := send(ctx, dm.UpdateRequestChan, struct{}{}); err != nil {
		return nil, err
	}
	// 监听协程对每个请求都会回复一次，所以请求发出后不随客户端断开而放弃等待：
	// 否则这次的回复会遗留在通道里，被下一次请求误读，并发保护也会在下载仍在进行时提前释放。
	select {
	case reason := <-dm.UpdateDownloadedChan:
		if reason != "" {
			return failed(reason), nil
		}
	case <-time.After(s.downloadWait):
		return failed("等待下载结果超时"), nil
	}
	return response.NewItemResponse(ExecResp{Success: true}), nil
}

// backup 保存配置并做一次全量备份，再复制一份到系统临时目录，避免升级覆盖程序目录后丢失。
// 备份失败只记录日志，不阻止升级。
func (s *Service) backup(d *dice.Dice) {
	d.Save(true)
	bakFn, err := s.backupFn(dice.BackupSelectionAll, false)
	if err != nil {
		d.Logger.Errorf("升级前备份失败: %v", err)
		return
	}
	tmpDir := filepath.Join(os.TempDir(), "backups")
	if err = os.MkdirAll(tmpDir, 0o755); err != nil {
		d.Logger.Errorf("创建备份临时目录失败: %v", err)
		return
	}
	dst := filepath.Join(tmpDir, filepath.Base(bakFn))
	d.Logger.Infof("将备份文件复制到此路径: %s", dst)
	if err = cp.Copy(bakFn, dst); err != nil {
		d.Logger.Errorf("复制备份文件失败: %v", err)
	}
}

func failed(reason string) *response.ItemResponse[ExecResp] {
	return response.NewItemResponse(ExecResp{Err: reason})
}

func send[T any](ctx context.Context, ch chan<- T, v T) error {
	select {
	case ch <- v:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
