package uploadgw

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	tusd "github.com/tus/tusd/v2/pkg/handler"

	"sealdice-core/dice"
)

// 模块标识，对应 Upload-Metadata 的 module 字段。
const (
	ModuleJS      = "js"
	ModuleDeck    = "deck"
	ModuleHelpdoc = "helpdoc"
)

// Destination 描述一类上传目标的业务差异：创建前校验 + 完成后落盘。
type Destination interface {
	// Validate 在创建上传会话时校验元数据（文件名、分组、扩展名等）与文件大小。
	Validate(meta map[string]string, size int64) error
	// Install 消费已通过校验的暂存文件，写入业务目录并触发相应重载。
	Install(ctx context.Context, stagedPath string, meta map[string]string) error
}

// ---------- js 脚本 ----------

type scriptDestination struct {
	dm *dice.DiceManager
}

func (d *scriptDestination) Validate(meta map[string]string, _ int64) error {
	if sanitizeFilename(meta[MetaFilename]) == "" {
		return tusd.NewError("ERR_INVALID_FILENAME", "文件名非法", http.StatusBadRequest)
	}
	return nil
}

func (d *scriptDestination) Install(_ context.Context, stagedPath string, meta map[string]string) error {
	filename := sanitizeFilename(meta[MetaFilename])
	if filename == "" {
		return tusd.NewError("ERR_INVALID_FILENAME", "文件名非法", http.StatusBadRequest)
	}
	dst := filepath.Join(dataDirOf(d.dm), "scripts", filename)
	if err := installFileAtomic(stagedPath, dst); err != nil {
		return tusd.NewError("ERR_INSTALL_FAILED", "写入脚本文件失败", http.StatusInternalServerError)
	}
	if d.dm.GetDice() != nil {
		d.dm.GetDice().MarkModified()
	}
	return nil
}

// ---------- 牌堆 ----------

type deckDestination struct {
	dm *dice.DiceManager
}

func (d *deckDestination) Validate(meta map[string]string, _ int64) error {
	if sanitizeFilename(meta[MetaFilename]) == "" {
		return tusd.NewError("ERR_INVALID_FILENAME", "文件名非法", http.StatusBadRequest)
	}
	return nil
}

func (d *deckDestination) Install(_ context.Context, stagedPath string, meta map[string]string) error {
	filename := sanitizeFilename(meta[MetaFilename])
	if filename == "" {
		return tusd.NewError("ERR_INVALID_FILENAME", "文件名非法", http.StatusBadRequest)
	}
	// 与既有实现保持一致：牌堆目录为相对路径 data/decks。
	dst := filepath.Join("data", "decks", filename)
	if err := installFileAtomic(stagedPath, dst); err != nil {
		return tusd.NewError("ERR_INSTALL_FAILED", "写入牌堆文件失败", http.StatusInternalServerError)
	}
	if d.dm.GetDice() != nil {
		dice.DeckReload(d.dm.GetDice())
	}
	return nil
}

// ---------- 帮助文档 ----------

type helpdocDestination struct {
	dm *dice.DiceManager
}

func (d *helpdocDestination) Validate(meta map[string]string, _ int64) error {
	group := sanitizeHelpGroup(meta[MetaGroup])
	if group == "" {
		return tusd.NewError("ERR_INVALID_GROUP", "group不能为空", http.StatusBadRequest)
	}
	if group == dice.HelpBuiltinGroup {
		return tusd.NewError("ERR_BUILTIN_GROUP", "不能为内置分组", http.StatusBadRequest)
	}
	filename := sanitizeFilename(meta[MetaFilename])
	if filename == "" {
		return tusd.NewError("ERR_INVALID_FILENAME", "文件名不能为空", http.StatusBadRequest)
	}
	if !isAllowedHelpDocFile(filename) {
		return tusd.NewError("ERR_INVALID_FILE_TYPE", "仅支持json和xlsx帮助文档", http.StatusBadRequest)
	}
	return nil
}

func (d *helpdocDestination) Install(_ context.Context, stagedPath string, meta map[string]string) error {
	help := d.dm.Help
	if help == nil {
		return tusd.NewError("ERR_HELPDOC_NOT_READY", "帮助文档未初始化", http.StatusBadRequest)
	}
	group := sanitizeHelpGroup(meta[MetaGroup])
	filename := sanitizeFilename(meta[MetaFilename])
	f, err := os.Open(stagedPath)
	if err != nil {
		return tusd.NewError("ERR_STAGED_OPEN", "读取暂存文件失败", http.StatusInternalServerError)
	}
	defer f.Close()
	if err := help.UploadHelpDoc(f, group, filename); err != nil {
		return tusd.NewError("ERR_INSTALL_FAILED", err.Error(), http.StatusInternalServerError)
	}
	return nil
}

// ---------- helpers ----------

func dataDirOf(dm *dice.DiceManager) string {
	if dm == nil {
		return "."
	}
	d := dm.GetDice()
	if d == nil || d.BaseConfig.DataDir == "" {
		return "."
	}
	return d.BaseConfig.DataDir
}

func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.TrimSpace(name)
	if name == "." || name == ".." || !filepath.IsLocal(name) {
		return ""
	}
	return name
}

func sanitizeHelpGroup(group string) string {
	group = strings.ReplaceAll(group, "/", "_")
	group = strings.ReplaceAll(group, "\\", "_")
	group = strings.TrimSpace(group)
	if group == "." || group == ".." {
		return ""
	}
	return group
}

func isAllowedHelpDocFile(filename string) bool {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".json", ".xlsx":
		return true
	default:
		return false
	}
}

// installFileAtomic 把 stagedPath 的内容原子地写入 dstPath：
// 先在目标目录建临时文件写入并 fsync，再走带备份/回滚的替换流程。
func installFileAtomic(stagedPath, dstPath string) error {
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return err
	}
	src, err := os.Open(stagedPath)
	if err != nil {
		return err
	}
	defer src.Close()

	tmp, err := os.CreateTemp(filepath.Dir(dstPath), "."+filepath.Base(dstPath)+".upload-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	keepTmp := false
	defer func() {
		_ = tmp.Close()
		if !keepTmp {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := io.Copy(tmp, src); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		return err
	}
	if err := replaceFileAtomic(tmpPath, dstPath); err != nil {
		return err
	}
	keepTmp = true
	return nil
}

// replaceFileAtomic 用同目录 staging 文件替换 dstPath：
// 目标不存在时直接 rename；目标存在时先备份、rename 新文件、失败回滚。
func replaceFileAtomic(stagedPath string, dstPath string) error {
	staged, err := os.Lstat(stagedPath)
	if err != nil {
		return err
	}
	if !staged.Mode().IsRegular() {
		return fmt.Errorf("upload source is not a regular file: %s", stagedPath)
	}
	// Lstat also rejects symlinks: only regular files may be moved out of the way.
	target, err := os.Lstat(dstPath)
	if errors.Is(err, os.ErrNotExist) {
		return os.Rename(stagedPath, dstPath)
	} else if err != nil {
		return err
	} else if !target.Mode().IsRegular() {
		return fmt.Errorf("upload target is not a regular file: %s", dstPath)
	}

	backup, err := os.CreateTemp(filepath.Dir(dstPath), "."+filepath.Base(dstPath)+".backup-*")
	if err != nil {
		return err
	}
	backupPath := backup.Name()
	if err := backup.Close(); err != nil {
		_ = os.Remove(backupPath)
		return err
	}
	if err := os.Remove(backupPath); err != nil {
		return err
	}
	if err := os.Rename(dstPath, backupPath); err != nil {
		return err
	}
	if err := os.Rename(stagedPath, dstPath); err != nil {
		if rollbackErr := os.Rename(backupPath, dstPath); rollbackErr != nil {
			return errors.Join(
				fmt.Errorf("replace upload target: %w", err),
				fmt.Errorf("rollback upload target: %w", rollbackErr),
			)
		}
		return err
	}
	_ = os.Remove(backupPath)
	return nil
}
