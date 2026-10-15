package uploadgw

import (
	"net/http"

	tusd "github.com/tus/tusd/v2/pkg/handler"

	"sealdice-core/dice"
)

const (
	ModuleCustomReply = "custom-reply"

	// maxCustomReplyUploadBytes 与 api/v2/customreply 原上限一致（20 MiB）。
	maxCustomReplyUploadBytes int64 = 20 << 20
)

// customReplyDestination 收养原 /sd-api/v2/custom-reply/files/upload 的语义：
// 同名文件拒绝（不覆盖），落盘后重载自定义回复。
type customReplyDestination struct {
	dm *dice.DiceManager
}

func (d *customReplyDestination) Module() string { return ModuleCustomReply }

func (d *customReplyDestination) Validate(meta map[string]string, size int64) error {
	filename := sanitizeFilename(meta[MetaFilename])
	if filename == "" {
		return tusd.NewError("ERR_INVALID_FILENAME", "文件名不能为空", http.StatusBadRequest)
	}
	if size > maxCustomReplyUploadBytes {
		return tusd.NewError("ERR_FILE_TOO_LARGE", "自定义回复文件过大", http.StatusBadRequest)
	}
	inst := d.dm.GetDice()
	if inst != nil && dice.CustomReplyConfigCheckExists(inst, filename) {
		return tusd.NewError("ERR_FILE_EXISTS", "文件已存在", http.StatusConflict)
	}
	return nil
}

func (d *customReplyDestination) Install(stagedPath string, meta map[string]string) error {
	inst := d.dm.GetDice()
	if inst == nil {
		return tusd.NewError("ERR_DICE_NOT_READY", "骰子未初始化", http.StatusInternalServerError)
	}
	filename := sanitizeFilename(meta[MetaFilename])
	if filename == "" {
		return tusd.NewError("ERR_INVALID_FILENAME", "文件名不能为空", http.StatusBadRequest)
	}
	if err := installFileAtomic(stagedPath, inst.GetExtConfigFilePath("reply", filename)); err != nil {
		return tusd.NewError("ERR_INSTALL_FAILED", "写入自定义回复文件失败", http.StatusInternalServerError)
	}
	dice.ReplyReload(inst)
	return nil
}
