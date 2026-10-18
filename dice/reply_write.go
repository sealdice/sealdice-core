package dice

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const ReplyPackageWarning = "此文件来自扩展包运行时缓存，重装、刷新缓存或升级扩展包时可能被覆盖"

var ErrReplyPackageNotFound = errors.New("扩展包自定义回复不存在")

func ValidateReplyVMVersion(version string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(version)) {
	case "", ReplyVMVersionV1:
		return "", true
	case ReplyVMVersionV2:
		return ReplyVMVersionV2, true
	default:
		return "", false
	}
}

func (d *Dice) FindReplyPackageFile(packageID, filename string) (PackageContentFile, bool) {
	if d.PackageManager == nil || filename == "" {
		return PackageContentFile{}, false
	}
	for _, file := range d.PackageManager.GetEnabledContentFiles("reply") {
		ext := strings.ToLower(filepath.Ext(file.Path))
		if (ext == ".yaml" || ext == ".yml") && (packageID == "" || strings.EqualFold(file.PackageID, packageID)) && strings.EqualFold(filepath.Base(file.Path), filename) {
			return file, true
		}
	}
	return PackageContentFile{}, false
}

// SaveReplyConfig 统一 VM 版本、包缓存路径、持久化和运行时发布规则。
// 未传版本保留原值；写入失败不修改现有规则，也不重载/重写其它回复文件。
func (d *Dice) SaveReplyConfig(rc *ReplyConfig, version *string) error {
	filename := strings.TrimSpace(rc.Filename)
	if !filepath.IsLocal(filename) || filename == "." || strings.ContainsAny(filename, "/\\") {
		return fmt.Errorf("%w: 文件名无效", ErrInvalidConfig)
	}
	rc.Filename = filename
	index := -1
	for i, old := range d.CustomReplyConfig {
		if old != nil && strings.EqualFold(old.PackageID, rc.PackageID) && strings.EqualFold(old.Filename, filename) {
			index = i
			if version == nil {
				rc.VMVersion = old.VMVersion
			}
			break
		}
	}
	if version != nil {
		value, valid := ValidateReplyVMVersion(*version)
		if !valid {
			return fmt.Errorf("%w: 不支持的自定义回复 VM 版本", ErrInvalidConfig)
		}
		rc.VMVersion = value
	}
	for _, item := range rc.Items {
		if item == nil {
			return fmt.Errorf("%w: 回复规则不能为空", ErrInvalidConfig)
		}
	}
	rc.Clean()
	rc.UpdateTimestamp = time.Now().Unix()
	if rc.CreateTimestamp == 0 {
		rc.CreateTimestamp = rc.UpdateTimestamp
	}
	rc.CacheBacked, rc.Warning = false, ""
	var target string
	if rc.PackageID != "" {
		file, ok := d.FindReplyPackageFile(rc.PackageID, filename)
		if !ok {
			return ErrReplyPackageNotFound
		}
		target = file.Path
		rc.CacheBacked, rc.Warning = true, ReplyPackageWarning
	} else {
		target = d.GetExtConfigFilePath("reply", filename)
	}
	if info, err := os.Lstat(target); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("自定义回复目标不是普通文件: %s", filename)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := rc.SaveToPath(target); err != nil {
		return err
	}
	next := *rc
	configs := slices.Clone(d.CustomReplyConfig)
	if index < 0 {
		configs = append(configs, &next)
	} else {
		configs[index] = &next
	}
	d.CustomReplyConfig = configs
	return nil
}
