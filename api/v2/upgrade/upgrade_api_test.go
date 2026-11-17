package upgrade //nolint:testpackage // 测试注入实例私有的版本检测等待时间与备份函数，不增加生产测试接口。

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"sealdice-core/dice"
	"sealdice-core/logger"
	"sealdice-core/model/common/request"
	"sealdice-core/model/common/response"
)

// newTestService 构造一个版本落后的 DiceManager。备份目录是相对工作目录的常量，
// 因此把工作目录和系统临时目录都切到测试目录，避免写入仓库或真实 /tmp。
// 备份函数换成假实现，只模仿真实 Backup 的返回约定（BackupDir 下的相对路径）。
func newTestService(t *testing.T) (*Service, *dice.DiceManager) {
	t.Helper()
	t.Chdir(t.TempDir())
	t.Setenv("TMPDIR", t.TempDir())

	d := &dice.Dice{Logger: logger.M()}
	d.BaseConfig.Name = "upgrade-test"
	d.BaseConfig.DataDir = t.TempDir()
	d.Config = dice.NewConfig(d)
	d.AttrsManager = &dice.AttrsManager{}
	d.ImSession = &dice.IMSession{
		Parent:       d,
		EndPoints:    []*dice.EndPointInfo{},
		ServiceAtNew: new(dice.SyncMap[string, *dice.GroupInfo]),
		PendingQuits: new(dice.SyncMap[string, *dice.PendingQuitInfo]),
	}
	dm := &dice.DiceManager{
		Dice:                   d,
		AppVersionCode:         100,
		AppVersionOnline:       &dice.VersionInfo{VersionLatestCode: 200},
		UpdateRequestChan:      make(chan struct{}, 1),
		UpdateDownloadedChan:   make(chan string, 1),
		UpdateCheckRequestChan: make(chan int, 1),
		UpdateSealdiceByFile:   func(string) bool { return true },
	}
	d.Parent = dm

	svc := NewService(dm)
	svc.versionCheckWait = 0
	svc.backupFn = func(dice.BackupSelection, bool) (string, error) {
		if err := os.MkdirAll(dice.BackupDir, 0o755); err != nil {
			return "", err
		}
		fn := filepath.Join(dice.BackupDir, "bak_test.zip")
		return fn, os.WriteFile(fn, []byte("backup"), 0o644)
	}
	return svc, dm
}

func requireStatus(t *testing.T, err error, want int) {
	t.Helper()
	var statusErr huma.StatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("error = %v, want huma status error %d", err, want)
	}
	if statusErr.GetStatus() != want {
		t.Fatalf("status = %d, want %d", statusErr.GetStatus(), want)
	}
}

func backupFiles(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dice.BackupDir, "bak_*.zip"))
	if err != nil {
		t.Fatalf("glob backups: %v", err)
	}
	return files
}

func tempBackupCopies(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(os.Getenv("TMPDIR"), "backups", "bak_*.zip"))
	if err != nil {
		t.Fatalf("glob temp backups: %v", err)
	}
	return files
}

func TestExecRejectsContainerMode(t *testing.T) {
	svc, dm := newTestService(t)
	dm.ContainerMode = true

	_, err := svc.Exec(t.Context(), &request.Empty{})
	requireStatus(t, err, 400)
	if len(dm.UpdateRequestChan) != 0 || len(dm.UpdateCheckRequestChan) != 0 {
		t.Fatal("container mode must not drive the update channels")
	}
}

func TestExecRequiresUpdateHook(t *testing.T) {
	svc, dm := newTestService(t)
	dm.UpdateSealdiceByFile = nil

	_, err := svc.Exec(t.Context(), &request.Empty{})
	requireStatus(t, err, 500)
}

func TestExecDoesNothingWhenAlreadyLatest(t *testing.T) {
	svc, dm := newTestService(t)
	dm.AppVersionOnline = &dice.VersionInfo{VersionLatestCode: dm.AppVersionCode}

	resp, err := svc.Exec(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if resp.Body.Item.Success || resp.Body.Item.Err == "" {
		t.Fatalf("item = %+v, want failure with reason", resp.Body.Item)
	}
	if len(dm.UpdateRequestChan) != 0 {
		t.Fatal("already latest must not trigger a download")
	}
	if got := backupFiles(t); len(got) != 0 {
		t.Fatalf("already latest must not create backups, got %v", got)
	}
}

func TestExecReportsUnknownOnlineVersion(t *testing.T) {
	svc, dm := newTestService(t)
	dm.AppVersionOnline = nil

	resp, err := svc.Exec(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if resp.Body.Item.Success || resp.Body.Item.Err == "" {
		t.Fatalf("item = %+v, want failure with reason", resp.Body.Item)
	}
	if len(dm.UpdateRequestChan) != 0 {
		t.Fatal("unknown online version must not trigger a download")
	}
}

// 升级进程在下载完成后很快接管并退出，所以备份必须先于下载请求完成。
func TestExecBacksUpBeforeTriggeringDownload(t *testing.T) {
	svc, dm := newTestService(t)

	backupsAtTrigger := make(chan int, 1)
	go func() {
		<-dm.UpdateRequestChan
		backupsAtTrigger <- len(backupFiles(t))
		dm.UpdateDownloadedChan <- ""
	}()

	resp, err := svc.Exec(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if !resp.Body.Item.Success {
		t.Fatalf("item = %+v, want success", resp.Body.Item)
	}
	if got := <-backupsAtTrigger; got != 1 {
		t.Fatalf("backups present when download was triggered = %d, want 1", got)
	}
	if got := tempBackupCopies(t); len(got) != 1 {
		t.Fatalf("temp backup copies = %v, want exactly one", got)
	}
}

// 备份失败只记录日志：不能因为备份出错就让升级无法进行。
func TestExecContinuesWhenBackupFails(t *testing.T) {
	svc, dm := newTestService(t)
	svc.backupFn = func(dice.BackupSelection, bool) (string, error) {
		return "", errors.New("disk full")
	}

	go func() {
		<-dm.UpdateRequestChan
		dm.UpdateDownloadedChan <- ""
	}()

	resp, err := svc.Exec(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if !resp.Body.Item.Success {
		t.Fatalf("item = %+v, want success despite backup failure", resp.Body.Item)
	}
	if got := tempBackupCopies(t); len(got) != 0 {
		t.Fatalf("no backup was made, temp copies = %v", got)
	}
}

func TestExecSurfacesDownloadFailure(t *testing.T) {
	svc, dm := newTestService(t)

	go func() {
		<-dm.UpdateRequestChan
		dm.UpdateDownloadedChan <- "更新: 下载更新文件失败"
	}()

	resp, err := svc.Exec(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if resp.Body.Item.Success {
		t.Fatalf("item = %+v, want failure", resp.Body.Item)
	}
	if resp.Body.Item.Err != "更新: 下载更新文件失败" {
		t.Fatalf("Err = %q, want the download failure reason", resp.Body.Item.Err)
	}
}

// 下载通道一次只服务一个请求：第二个并发请求必须立即被拒绝，
// 而不是和第一个抢同一个下载结果；第一个拿到下载结果后正常完成。
func TestExecRejectsConcurrentRun(t *testing.T) {
	svc, dm := newTestService(t)

	type result struct {
		resp *response.ItemResponse[ExecResp]
		err  error
	}
	first := make(chan result, 1)
	go func() {
		resp, err := svc.Exec(t.Context(), &request.Empty{})
		first <- result{resp, err}
	}()
	deadline := time.After(10 * time.Second)
	for len(dm.UpdateRequestChan) == 0 { // 第一个请求已进入等待下载结果
		select {
		case <-deadline:
			t.Fatal("first request never reached the download step")
		case <-time.After(5 * time.Millisecond):
		}
	}

	resp, err := svc.Exec(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("second Exec returned error: %v", err)
	}
	if resp.Body.Item.Success || resp.Body.Item.Err == "" {
		t.Fatalf("second item = %+v, want rejection", resp.Body.Item)
	}

	dm.UpdateDownloadedChan <- ""
	got := <-first
	if got.err != nil || !got.resp.Body.Item.Success {
		t.Fatalf("first Exec = %+v, %v, want success once the download result arrives", got.resp, got.err)
	}
}

// 请求发出后即使客户端断开也要等到下载结果，不能把它遗留在通道里：
// 否则下一次请求会把上一次的结果当成自己的。
func TestExecDoesNotAbandonDownloadWhenClientDisconnects(t *testing.T) {
	svc, dm := newTestService(t)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = svc.Exec(ctx, &request.Empty{})
	}()
	deadline := time.After(10 * time.Second)
	for len(dm.UpdateRequestChan) == 0 {
		select {
		case <-deadline:
			t.Fatal("request never reached the download step")
		case <-time.After(5 * time.Millisecond):
		}
	}

	cancel()
	select {
	case <-done:
		t.Fatal("Exec returned on client disconnect while the download was still pending")
	case <-time.After(100 * time.Millisecond):
	}

	dm.UpdateDownloadedChan <- ""
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Exec did not return after the download result arrived")
	}
}

// 上一次遗留的下载结果不能被当成本次结果。
func TestExecIgnoresStaleDownloadResult(t *testing.T) {
	svc, dm := newTestService(t)
	dm.UpdateDownloadedChan <- "" // 遗留的“成功”

	go func() {
		<-dm.UpdateRequestChan
		dm.UpdateDownloadedChan <- "更新: 下载更新文件失败"
	}()

	resp, err := svc.Exec(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if resp.Body.Item.Success || resp.Body.Item.Err != "更新: 下载更新文件失败" {
		t.Fatalf("item = %+v, want this attempt's failure, not the stale success", resp.Body.Item)
	}
}

// 监听协程没有回复时不能永远占着并发保护。
func TestExecTimesOutWaitingForDownload(t *testing.T) {
	svc, dm := newTestService(t)
	svc.downloadWait = 50 * time.Millisecond

	resp, err := svc.Exec(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("Exec returned error: %v", err)
	}
	if resp.Body.Item.Success || resp.Body.Item.Err == "" {
		t.Fatalf("item = %+v, want timeout failure", resp.Body.Item)
	}

	// 保护已释放：再次请求会重新进入下载步骤，而不是被“正在进行中”拒绝。
	// 先取走第一次请求遗留的两个信号（测试里没有常驻监听协程消费它们）。
	<-dm.UpdateCheckRequestChan
	<-dm.UpdateRequestChan
	go func() {
		<-dm.UpdateRequestChan
		dm.UpdateDownloadedChan <- ""
	}()
	resp, err = svc.Exec(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("second Exec returned error: %v", err)
	}
	if !resp.Body.Item.Success {
		t.Fatalf("second item = %+v, want success after the timeout released the guard", resp.Body.Item)
	}
}
