package uploadgw

import (
	"context"
	"net/http"
	"os"

	tusd "github.com/tus/tusd/v2/pkg/handler"

	"sealdice-core/dice"
)

const ModuleExtensionInstall = "extension-install"

// extensionInstallDestination 收养原 /sd-api/v2/extension/install-upload 的语义：
// 把暂存的扩展包 zip 流式安装。安装需要 context，故同时实现 contextualDestination。
type extensionInstallDestination struct {
	dm *dice.DiceManager
}

func (d *extensionInstallDestination) Module() string { return ModuleExtensionInstall }

func (d *extensionInstallDestination) Validate(meta map[string]string, _ int64) error {
	if sanitizeFilename(meta[MetaFilename]) == "" {
		return tusd.NewError("ERR_INVALID_FILENAME", "文件名不能为空", http.StatusBadRequest)
	}
	return nil
}

func (d *extensionInstallDestination) Install(stagedPath string, meta map[string]string) error {
	return d.InstallContext(context.Background(), stagedPath, meta)
}

func (d *extensionInstallDestination) InstallContext(ctx context.Context, stagedPath string, _ map[string]string) error {
	inst := d.dm.GetDice()
	if inst == nil || inst.PackageManager == nil {
		return tusd.NewError("ERR_PACKAGE_MANAGER", "package manager unavailable", http.StatusInternalServerError)
	}
	f, err := os.Open(stagedPath)
	if err != nil {
		return tusd.NewError("ERR_STAGED_OPEN", "读取暂存文件失败", http.StatusInternalServerError)
	}
	defer f.Close()
	if err := inst.PackageManager.InstallFromStreamContext(ctx, f); err != nil {
		return tusd.NewError("ERR_INSTALL_FAILED", err.Error(), http.StatusBadRequest)
	}
	return nil
}
