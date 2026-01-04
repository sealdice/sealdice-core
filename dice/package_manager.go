package dice

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver/v3"

	"sealdice-core/dice/sealpkg"
)

// PackageManager 扩展包管理器
type PackageManager struct {
	lock   *sync.RWMutex
	parent *Dice

	// 已安装的包
	packages map[string]*sealpkg.Instance

	// 依赖图: A -> [B, C] 表示 A 依赖 B 和 C
	dependencyGraph map[string][]string

	// 反向依赖图: A -> [B, C] 表示 B 和 C 依赖 A
	reverseDependencyGraph map[string][]string
}

// NewPackageManager 创建包管理器
func NewPackageManager(parent *Dice) *PackageManager {
	pm := &PackageManager{
		lock:                   new(sync.RWMutex),
		parent:                 parent,
		packages:               make(map[string]*sealpkg.Instance),
		dependencyGraph:        make(map[string][]string),
		reverseDependencyGraph: make(map[string][]string),
	}
	return pm
}

// Init 初始化包管理器，加载已安装的包
func (pm *PackageManager) Init() error {
	// 确保目录存在
	sourcePath := pm.getSourcePackagesPath()
	cachePath := pm.getCachePackagesPath()
	if err := os.MkdirAll(sourcePath, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(cachePath, 0755); err != nil {
		return err
	}

	// 加载持久化状态
	if err := pm.loadState(); err != nil {
		pm.parent.Logger.Warnf("加载扩展包状态失败: %v", err)
	}

	// 扫描 data/packages/ 目录下的 .sealpkg 文件
	entries, err := os.ReadDir(sourcePath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), sealpkg.Extension) {
			continue
		}

		pkgFilePath := filepath.Join(sourcePath, entry.Name())
		if err := pm.extractAndLoadPackage(pkgFilePath); err != nil {
			pm.parent.Logger.Warnf("加载扩展包 %s 失败: %v", entry.Name(), err)
		}
	}

	// 清理无效的包实例（没有 Manifest 的包）
	// 这可能是因为源文件被删除，但状态文件还保留着
	pm.cleanInvalidPackages()

	// 构建依赖图
	pm.buildDependencyGraph()

	return nil
}

// extractAndLoadPackage 解压并加载扩展包
func (pm *PackageManager) extractAndLoadPackage(pkgFilePath string) error {
	// 打开 zip 文件读取 manifest
	reader, err := zip.OpenReader(pkgFilePath)
	if err != nil {
		return errors.New("无法打开扩展包: " + err.Error())
	}
	defer reader.Close()

	// 查找并解析 manifest.toml
	var manifestData []byte
	for _, f := range reader.File {
		if f.Name == sealpkg.ManifestFile {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			manifestData, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return err
			}
			break
		}
	}

	if manifestData == nil {
		return errors.New("扩展包中未找到 " + sealpkg.ManifestFile)
	}

	manifest, err := sealpkg.ParseManifest(manifestData)
	if err != nil {
		return err
	}

	pkgID := manifest.Package.ID

	// 验证包 ID 格式
	if err := sealpkg.ValidatePackageID(pkgID); err != nil {
		return errors.New("包 ID 格式错误: " + err.Error())
	}

	// 检查文件名和包 ID 是否一致
	expectedFileName := sealpkg.PackageIDToFileName(pkgID)
	actualFileName := filepath.Base(pkgFilePath)
	if expectedFileName != actualFileName {
		pm.parent.Logger.Warnf(
			"扩展包文件名与包内 ID 不一致: 文件名=%s, 期望=%s (ID=%s), 将使用包内 ID 进行解压",
			actualFileName, expectedFileName, pkgID,
		)
	}

	installPath := filepath.Join(pm.getCachePackagesPath(), pkgID)

	// 检查是否需要解压（目录不存在或 manifest 不存在）
	manifestPath := filepath.Join(installPath, sealpkg.ManifestFile)
	needExtract := false
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		needExtract = true
	}

	if needExtract {
		// 清理旧目录（保留用户数据）
		if _, err := os.Stat(installPath); err == nil {
			pm.cleanInstallDir(installPath)
		}

		// 创建目录并解压
		if err := os.MkdirAll(installPath, 0755); err != nil {
			return err
		}

		for _, f := range reader.File {
			destPath := filepath.Join(installPath, f.Name)

			// 防止 Zip Slip 攻击
			if !strings.HasPrefix(destPath, filepath.Clean(installPath)+string(os.PathSeparator)) {
				return errors.New("非法的文件路径: " + f.Name)
			}

			if f.FileInfo().IsDir() {
				os.MkdirAll(destPath, f.Mode())
				continue
			}

			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return err
			}

			destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}

			srcFile, err := f.Open()
			if err != nil {
				destFile.Close()
				return err
			}

			_, err = io.Copy(destFile, srcFile)
			srcFile.Close()
			destFile.Close()

			if err != nil {
				return err
			}
		}

		pm.parent.Logger.Infof("扩展包 %s 已解压到缓存", pkgID)
	}

	// 获取用户数据目录路径（在 data/extensions/<包ID>/_userdata/）
	userDataPath := pm.getUserDataPath(pkgID)
	// 确保用户数据目录存在
	os.MkdirAll(userDataPath, 0755)

	// 注册或更新包实例
	if _, exists := pm.packages[pkgID]; !exists {
		pm.packages[pkgID] = &sealpkg.Instance{
			Manifest:     manifest,
			State:        sealpkg.PackageStateDisabled,
			InstallPath:  installPath,
			SourcePath:   pkgFilePath,
			UserDataPath: userDataPath,
			Config:       make(map[string]interface{}),
		}
	} else {
		pm.packages[pkgID].Manifest = manifest
		pm.packages[pkgID].InstallPath = installPath
		pm.packages[pkgID].SourcePath = pkgFilePath
		pm.packages[pkgID].UserDataPath = userDataPath
	}

	return nil
}

// getSourcePackagesPath 获取 .sealpkg 源文件存放目录
// 路径: data/packages/
func (pm *PackageManager) getSourcePackagesPath() string {
	return filepath.Join(".", "data", sealpkg.PackagesDir)
}

// getUserDataPath 获取扩展包用户数据目录
// 路径: data/extensions/<包ID>/_userdata/
// 包ID中的 "/" 会被替换为 "@" 以兼容文件系统
func (pm *PackageManager) getUserDataPath(pkgID string) string {
	// 将包ID转换为文件系统安全的格式
	safePkgID := sealpkg.PackageIDToSafeDir(pkgID)
	return filepath.Join(".", "data", "extensions", safePkgID, sealpkg.UserDataDir)
}

// getCachePackagesPath 获取解压后的包缓存目录
// 路径: cache/packages/
func (pm *PackageManager) getCachePackagesPath() string {
	return filepath.Join(".", "cache", sealpkg.PackagesDir)
}

// getPackagesPath 获取扩展包安装目录（兼容旧代码，返回缓存目录）
func (pm *PackageManager) getPackagesPath() string {
	return pm.getCachePackagesPath()
}

// getStatePath 获取状态文件路径
func (pm *PackageManager) getStatePath() string {
	return filepath.Join(pm.getSourcePackagesPath(), sealpkg.StateFile)
}

// loadState 加载持久化状态
func (pm *PackageManager) loadState() error {
	statePath := pm.getStatePath()
	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var state sealpkg.Persistence
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	for id, persist := range state.Packages {
		// 兼容旧数据：如果没有 UserDataPath，自动计算
		userDataPath := persist.UserDataPath
		if userDataPath == "" {
			userDataPath = pm.getUserDataPath(id)
		}

		pm.packages[id] = &sealpkg.Instance{
			State:        persist.State,
			InstallTime:  persist.InstallTime,
			InstallPath:  persist.InstallPath,
			SourcePath:   persist.SourcePath,
			UserDataPath: userDataPath,
			Config:       persist.Config,
		}
		if pm.packages[id].Config == nil {
			pm.packages[id].Config = make(map[string]interface{})
		}
	}

	return nil
}

// saveState 保存持久化状态
func (pm *PackageManager) saveState() error {
	state := sealpkg.Persistence{
		Packages: make(map[string]*sealpkg.InstancePersist),
	}

	for id, pkg := range pm.packages {
		state.Packages[id] = &sealpkg.InstancePersist{
			State:        pkg.State,
			InstallTime:  pkg.InstallTime,
			InstallPath:  pkg.InstallPath,
			SourcePath:   pkg.SourcePath,
			UserDataPath: pkg.UserDataPath,
			Config:       pkg.Config,
		}
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(pm.getStatePath(), data, 0644)
}

// Install 安装扩展包
// 将 .sealpkg 文件复制到 data/packages/，然后解压到 cache/packages/
func (pm *PackageManager) Install(pkgPath string) error {
	pm.lock.Lock()
	defer pm.lock.Unlock()

	// 打开 zip 文件读取 manifest
	reader, err := zip.OpenReader(pkgPath)
	if err != nil {
		return errors.New("无法打开扩展包: " + err.Error())
	}

	// 查找并解析 manifest.toml
	var manifestData []byte
	for _, f := range reader.File {
		if f.Name == sealpkg.ManifestFile {
			rc, err := f.Open()
			if err != nil {
				reader.Close()
				return err
			}
			manifestData, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				reader.Close()
				return err
			}
			break
		}
	}
	reader.Close()

	if manifestData == nil {
		return errors.New("扩展包中未找到 " + sealpkg.ManifestFile)
	}

	manifest, err := sealpkg.ParseManifest(manifestData)
	if err != nil {
		return err
	}

	pkgID := manifest.Package.ID

	// 验证包 ID 格式
	if err := sealpkg.ValidatePackageID(pkgID); err != nil {
		return errors.New("包 ID 格式错误: " + err.Error())
	}

	// 检查是否已安装
	var oldConfig map[string]interface{}
	if existing, exists := pm.packages[pkgID]; exists {
		existingVer, _ := semver.NewVersion(existing.Manifest.Package.Version)
		newVer, _ := semver.NewVersion(manifest.Package.Version)
		if existingVer != nil && newVer != nil && !newVer.GreaterThan(existingVer) {
			return errors.New("已安装相同或更高版本的扩展包")
		}
		if existing.State == sealpkg.PackageStateEnabled {
			pm.disableInternal(pkgID)
		}
		// 保留旧配置
		oldConfig = existing.Config
	}

	// 检查海豹版本兼容性
	if err := sealpkg.CheckSealVersion(manifest, VERSION.String()); err != nil {
		return err
	}

	// 检查依赖
	if satisfied, missing := pm.CheckDependencies(manifest); !satisfied {
		return &DependencyError{
			PackageID:   pkgID,
			MissingDeps: missing,
		}
	}

	// 生成目标文件名：使用包 ID 直接作为文件名（@ 符号在文件名中合法）
	fileName := sealpkg.PackageIDToFileName(pkgID)
	destPkgPath := filepath.Join(pm.getSourcePackagesPath(), fileName)

	// 复制 .sealpkg 到 data/packages/
	if pkgPath != destPkgPath {
		if err := pm.copyFile(pkgPath, destPkgPath); err != nil {
			return errors.New("复制扩展包失败: " + err.Error())
		}
	}

	// 删除旧的缓存目录（强制重新解压）
	cachePath := filepath.Join(pm.getCachePackagesPath(), pkgID)
	if _, err := os.Stat(cachePath); err == nil {
		pm.cleanInstallDir(cachePath)
		// 删除 manifest 以触发重新解压
		os.Remove(filepath.Join(cachePath, sealpkg.ManifestFile))
	}

	// 解压并加载包
	if err := pm.extractAndLoadPackage(destPkgPath); err != nil {
		return err
	}

	// 恢复旧配置并初始化默认配置
	pkg := pm.packages[pkgID]
	pkg.InstallTime = time.Now()
	if oldConfig != nil {
		pkg.Config = oldConfig
	}
	pkg.Config = sealpkg.MergeConfig(
		sealpkg.InitDefaultConfig(manifest.Config),
		pkg.Config,
	)

	pm.buildDependencyGraph()

	if err := pm.saveState(); err != nil {
		pm.parent.Logger.Warnf("保存扩展包状态失败: %v", err)
	}

	pm.parent.Logger.Infof("扩展包 %s v%s 安装成功", manifest.Package.Name, manifest.Package.Version)
	return nil
}

// cleanInstallDir 清理缓存目录
// 用户数据现在存储在 data/extensions/<包ID>/_userdata/，不在缓存目录中
func (pm *PackageManager) cleanInstallDir(installPath string) {
	entries, err := os.ReadDir(installPath)
	if err != nil {
		return
	}

	for _, entry := range entries {
		os.RemoveAll(filepath.Join(installPath, entry.Name()))
	}
}

// cleanInvalidPackages 清理无效的包实例
// 如果包实例没有 Manifest，尝试从缓存目录恢复，失败则移除
func (pm *PackageManager) cleanInvalidPackages() {
	invalidPackages := make([]string, 0)

	for pkgID, pkg := range pm.packages {
		if pkg.Manifest == nil {
			// 尝试从缓存目录恢复 manifest
			manifestPath := filepath.Join(pkg.InstallPath, sealpkg.ManifestFile)
			if data, err := os.ReadFile(manifestPath); err == nil {
				if manifest, err := sealpkg.ParseManifest(data); err == nil {
					pkg.Manifest = manifest
					pm.parent.Logger.Infof("已从缓存恢复扩展包 %s 的 manifest", pkgID)
					continue
				}
			}

			// 无法恢复，标记为无效
			pm.parent.Logger.Warnf("扩展包 %s 的 manifest 缺失且无法恢复，将被移除", pkgID)
			invalidPackages = append(invalidPackages, pkgID)
		}
	}

	// 移除无效的包
	for _, pkgID := range invalidPackages {
		delete(pm.packages, pkgID)
	}

	// 如果有包被移除，保存状态
	if len(invalidPackages) > 0 {
		pm.saveState()
	}
}

// Uninstall 卸载扩展包
func (pm *PackageManager) Uninstall(pkgID string, mode sealpkg.UninstallMode) error {
	pm.lock.Lock()
	defer pm.lock.Unlock()

	pkg, exists := pm.packages[pkgID]
	if !exists {
		return errors.New("扩展包不存在: " + pkgID)
	}

	// 检查反向依赖
	if deps := pm.reverseDependencyGraph[pkgID]; len(deps) > 0 {
		enabledDeps := make([]string, 0)
		for _, depID := range deps {
			if depPkg, ok := pm.packages[depID]; ok && depPkg.State == sealpkg.PackageStateEnabled {
				enabledDeps = append(enabledDeps, depID)
			}
		}
		if len(enabledDeps) > 0 {
			return errors.New("以下扩展包依赖此包: " + strings.Join(enabledDeps, ", "))
		}
	}

	if pkg.State == sealpkg.PackageStateEnabled {
		pm.disableInternal(pkgID)
	}

	switch mode {
	case sealpkg.UninstallModeFull:
		// 删除缓存目录
		if err := os.RemoveAll(pkg.InstallPath); err != nil {
			return err
		}
		// 删除源文件
		if pkg.SourcePath != "" {
			os.Remove(pkg.SourcePath)
		}
		// 删除用户数据目录
		if pkg.UserDataPath != "" {
			os.RemoveAll(pkg.UserDataPath)
			// 尝试删除父目录（如果为空）
			os.Remove(filepath.Dir(pkg.UserDataPath))
		}
		delete(pm.packages, pkgID)

	case sealpkg.UninstallModeKeepData:
		// 删除缓存目录（用户数据在 data/extensions/ 下，不受影响）
		if err := os.RemoveAll(pkg.InstallPath); err != nil {
			return err
		}
		// 删除源文件
		if pkg.SourcePath != "" {
			os.Remove(pkg.SourcePath)
		}
		// 保留 UserDataPath
		delete(pm.packages, pkgID)

	case sealpkg.UninstallModeDisable:
		pkg.State = sealpkg.PackageStateDisabled
	}

	pm.buildDependencyGraph()
	pm.saveState()

	pm.parent.Logger.Infof("扩展包 %s 已卸载 (模式: %s)", pkgID, mode)
	return nil
}

// Enable 启用扩展包
func (pm *PackageManager) Enable(pkgID string) (*sealpkg.OperationResult, error) {
	pm.lock.Lock()
	defer pm.lock.Unlock()

	pkg, exists := pm.packages[pkgID]
	if !exists {
		return nil, errors.New("扩展包不存在: " + pkgID)
	}

	if pkg.Manifest == nil {
		return nil, errors.New("扩展包 manifest 缺失，无法启用")
	}

	if pkg.State == sealpkg.PackageStateEnabled {
		return &sealpkg.OperationResult{
			Success:      true,
			Message:      "扩展包已处于启用状态",
			ReloadNeeded: false,
		}, nil
	}

	if satisfied, missing := pm.CheckDependencies(pkg.Manifest); !satisfied {
		return nil, &DependencyError{
			PackageID:   pkgID,
			MissingDeps: missing,
		}
	}

	// 启用依赖的包
	for depID := range pkg.Manifest.Dependencies {
		if depPkg, ok := pm.packages[depID]; ok && depPkg.State != sealpkg.PackageStateEnabled {
			if _, err := pm.enableInternal(depID); err != nil {
				return nil, errors.New("启用依赖包 " + depID + " 失败: " + err.Error())
			}
		}
	}

	result, err := pm.enableInternal(pkgID)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// enableInternal 内部启用逻辑
func (pm *PackageManager) enableInternal(pkgID string) (*sealpkg.OperationResult, error) {
	pkg := pm.packages[pkgID]
	pkg.State = sealpkg.PackageStateEnabled
	pkg.ErrText = ""

	// 链接包内资源到全局目录
	if err := pm.linkPackageResources(pkg); err != nil {
		pm.parent.Logger.Warnf("链接扩展包 %s 的资源失败: %v", pkgID, err)
	}

	// 生成重载提示并设置待重载状态
	result := pm.generateReloadHints(pkg.Manifest)
	result.Success = true
	result.Message = "扩展包已启用，需要重载相应资源才能生效"

	// 设置待重载状态，供 UI 查询
	if result.ReloadNeeded {
		pkg.PendingReload = result.ReloadHints
	}

	pm.saveState()
	pm.parent.Logger.Infof("扩展包 %s 已启用", pkgID)

	return result, nil
}

// Disable 禁用扩展包
func (pm *PackageManager) Disable(pkgID string) (*sealpkg.OperationResult, error) {
	pm.lock.Lock()
	defer pm.lock.Unlock()

	pkg, exists := pm.packages[pkgID]
	if !exists {
		return nil, errors.New("扩展包不存在: " + pkgID)
	}

	if pkg.State != sealpkg.PackageStateEnabled {
		return &sealpkg.OperationResult{
			Success:      true,
			Message:      "扩展包已处于禁用状态",
			ReloadNeeded: false,
		}, nil
	}

	// 检查反向依赖
	if deps := pm.reverseDependencyGraph[pkgID]; len(deps) > 0 {
		enabledDeps := make([]string, 0)
		for _, depID := range deps {
			if depPkg, ok := pm.packages[depID]; ok && depPkg.State == sealpkg.PackageStateEnabled {
				enabledDeps = append(enabledDeps, depID)
			}
		}
		if len(enabledDeps) > 0 {
			return nil, errors.New("以下已启用的扩展包依赖此包: " + strings.Join(enabledDeps, ", "))
		}
	}

	result, err := pm.disableInternal(pkgID)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// disableInternal 内部禁用逻辑
func (pm *PackageManager) disableInternal(pkgID string) (*sealpkg.OperationResult, error) {
	pkg := pm.packages[pkgID]
	pkg.State = sealpkg.PackageStateDisabled

	// 移除全局目录中的资源链接
	if err := pm.unlinkPackageResources(pkg); err != nil {
		pm.parent.Logger.Warnf("移除扩展包 %s 的资源链接失败: %v", pkgID, err)
	}

	// 生成重载提示并设置待重载状态
	result := pm.generateReloadHints(pkg.Manifest)
	result.Success = true
	result.Message = "扩展包已禁用，需要重载相应资源才能生效"

	// 设置待重载状态，供 UI 查询
	if result.ReloadNeeded {
		pkg.PendingReload = result.ReloadHints
	}

	pm.saveState()
	pm.parent.Logger.Infof("扩展包 %s 已禁用", pkgID)

	return result, nil
}

// CheckDependencies 检查依赖是否满足
func (pm *PackageManager) CheckDependencies(manifest *sealpkg.Manifest) (bool, []string) {
	missing := make([]string, 0)

	for depID, constraint := range manifest.Dependencies {
		depPkg, exists := pm.packages[depID]
		if !exists {
			missing = append(missing, depID)
			continue
		}

		satisfied, err := sealpkg.CheckDependencyConstraint(constraint, depPkg.Manifest.Package.Version)
		if err != nil {
			missing = append(missing, depID+" ("+err.Error()+")")
			continue
		}

		if !satisfied {
			missing = append(missing, depID+" (需要 "+constraint+", 已安装 "+depPkg.Manifest.Package.Version+")")
		}
	}

	return len(missing) == 0, missing
}

// buildDependencyGraph 构建依赖图
func (pm *PackageManager) buildDependencyGraph() {
	pm.dependencyGraph = make(map[string][]string)
	pm.reverseDependencyGraph = make(map[string][]string)

	for pkgID, pkg := range pm.packages {
		if pkg.Manifest == nil {
			continue
		}

		deps := make([]string, 0, len(pkg.Manifest.Dependencies))
		for depID := range pkg.Manifest.Dependencies {
			deps = append(deps, depID)

			if pm.reverseDependencyGraph[depID] == nil {
				pm.reverseDependencyGraph[depID] = make([]string, 0)
			}
			pm.reverseDependencyGraph[depID] = append(pm.reverseDependencyGraph[depID], pkgID)
		}
		pm.dependencyGraph[pkgID] = deps
	}
}

// GetConfig 获取包的用户配置
func (pm *PackageManager) GetConfig(pkgID string) (map[string]interface{}, error) {
	pm.lock.RLock()
	defer pm.lock.RUnlock()

	pkg, exists := pm.packages[pkgID]
	if !exists {
		return nil, errors.New("扩展包不存在: " + pkgID)
	}

	return pkg.Config, nil
}

// SetConfig 设置包的用户配置
func (pm *PackageManager) SetConfig(pkgID string, config map[string]interface{}) error {
	pm.lock.Lock()
	defer pm.lock.Unlock()

	pkg, exists := pm.packages[pkgID]
	if !exists {
		return errors.New("扩展包不存在: " + pkgID)
	}

	if pkg.Manifest == nil {
		return errors.New("扩展包 manifest 缺失，无法设置配置")
	}

	// 验证配置
	if err := sealpkg.ValidateConfig(config, pkg.Manifest.Config); err != nil {
		return err
	}

	pkg.Config = config

	// 保存到用户数据目录
	configPath := filepath.Join(pkg.InstallPath, sealpkg.UserDataDir, sealpkg.ConfigFile)
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return err
	}

	return pm.saveState()
}

// List 列出所有已安装的包
func (pm *PackageManager) List() []*sealpkg.Instance {
	pm.lock.RLock()
	defer pm.lock.RUnlock()

	result := make([]*sealpkg.Instance, 0, len(pm.packages))
	for _, pkg := range pm.packages {
		result = append(result, pkg)
	}
	return result
}

// Get 获取指定包
func (pm *PackageManager) Get(pkgID string) (*sealpkg.Instance, bool) {
	pm.lock.RLock()
	defer pm.lock.RUnlock()

	pkg, exists := pm.packages[pkgID]
	return pkg, exists
}

// GetEnabled 获取所有已启用的包
func (pm *PackageManager) GetEnabled() []*sealpkg.Instance {
	pm.lock.RLock()
	defer pm.lock.RUnlock()

	result := make([]*sealpkg.Instance, 0)
	for _, pkg := range pm.packages {
		if pkg.State == sealpkg.PackageStateEnabled {
			result = append(result, pkg)
		}
	}
	return result
}

// GetEnabledContentDirs 获取所有已启用包中指定类型资源的目录列表
// contentType: "scripts", "decks", "reply", "helpdoc", "templates"
// 返回存在的目录路径列表
func (pm *PackageManager) GetEnabledContentDirs(contentType string) []string {
	pm.lock.RLock()
	defer pm.lock.RUnlock()

	dirs := make([]string, 0)
	for _, pkg := range pm.packages {
		if pkg.State != sealpkg.PackageStateEnabled || pkg.Manifest == nil {
			continue
		}

		// 检查包是否声明了该类型的内容
		hasContent := false
		switch contentType {
		case "scripts":
			hasContent = len(pkg.Manifest.Contents.Scripts) > 0
		case "decks":
			hasContent = len(pkg.Manifest.Contents.Decks) > 0
		case "reply":
			hasContent = len(pkg.Manifest.Contents.Reply) > 0
		case "helpdoc":
			hasContent = len(pkg.Manifest.Contents.Helpdoc) > 0
		case "templates":
			hasContent = len(pkg.Manifest.Contents.Template) > 0
		}

		if !hasContent {
			continue
		}

		// 检查目录是否存在
		dir := filepath.Join(pkg.InstallPath, contentType)
		if _, err := os.Stat(dir); err == nil {
			dirs = append(dirs, dir)
		}
	}

	return dirs
}

// LoadAllEnabled 加载所有已启用包的资源
func (pm *PackageManager) LoadAllEnabled() {
	pm.lock.RLock()
	defer pm.lock.RUnlock()

	for pkgID, pkg := range pm.packages {
		if pkg.State == sealpkg.PackageStateEnabled {
			pm.parent.Logger.Infof("加载扩展包 %s 的资源", pkgID)
			// TODO: 实际的资源加载逻辑
		}
	}
}

// GetSandbox 获取包的沙箱
func (pm *PackageManager) GetSandbox(pkgID string) (*sealpkg.Sandbox, error) {
	pm.lock.RLock()
	defer pm.lock.RUnlock()

	pkg, exists := pm.packages[pkgID]
	if !exists {
		return nil, errors.New("扩展包不存在: " + pkgID)
	}

	return sealpkg.NewSandboxFromInstance(pkg), nil
}

// generateReloadHints 根据包的内容生成重载提示
func (pm *PackageManager) generateReloadHints(manifest *sealpkg.Manifest) *sealpkg.OperationResult {
	hints := make([]string, 0)

	if manifest == nil {
		return &sealpkg.OperationResult{
			ReloadNeeded: false,
			ReloadHints:  hints,
		}
	}

	// 可重载的资源
	if len(manifest.Contents.Scripts) > 0 {
		hints = append(hints, "JS 脚本 - 可通过重载接口生效")
	}

	if len(manifest.Contents.Decks) > 0 {
		hints = append(hints, "牌堆 - 可通过重载接口生效")
	}

	if len(manifest.Contents.Reply) > 0 {
		hints = append(hints, "自动回复 - 可通过重载接口生效")
	}

	if len(manifest.Contents.Helpdoc) > 0 {
		hints = append(hints, "帮助文档 - 可通过重载接口生效")
	}

	if len(manifest.Contents.Template) > 0 {
		hints = append(hints, "游戏系统模板 - 可通过重载接口生效")
	}

	return &sealpkg.OperationResult{
		ReloadNeeded: len(hints) > 0,
		ReloadHints:  hints,
	}
}

// Reload 重载扩展包资源（聚合接口）
func (pm *PackageManager) Reload(pkgID string) (*sealpkg.ReloadResult, error) {
	pm.lock.RLock()
	pkg, exists := pm.packages[pkgID]
	pm.lock.RUnlock()

	if !exists {
		return nil, errors.New("扩展包不存在: " + pkgID)
	}

	if pkg.State != sealpkg.PackageStateEnabled {
		return &sealpkg.ReloadResult{
			Success: false,
			Message: "扩展包未启用，无需重载",
		}, nil
	}

	result := &sealpkg.ReloadResult{
		Success:       true,
		ReloadedItems: make(map[string]string),
		RestartHints:  make([]string, 0),
	}

	manifest := pkg.Manifest
	if manifest == nil {
		return &sealpkg.ReloadResult{
			Success: false,
			Message: "扩展包 manifest 为空",
		}, nil
	}

	// 重载 JS 脚本
	if len(manifest.Contents.Scripts) > 0 {
		pm.parent.JsReload()
		result.ReloadedItems["scripts"] = "JS 脚本已重载"
	}

	// 重载牌堆
	if len(manifest.Contents.Decks) > 0 {
		DeckReload(pm.parent)
		result.ReloadedItems["decks"] = "牌堆已重载"
	}

	// 重载自动回复（自动触发，无需手动调用）
	if len(manifest.Contents.Reply) > 0 {
		ReplyReload(pm.parent)
		result.ReloadedItems["reply"] = "自动回复已重载"
	}

	// 重载帮助文档
	if len(manifest.Contents.Helpdoc) > 0 {
		// 注意：帮助文档重载需要通过 DiceManager
		// 这里简化处理，实际需要调用 dm.InitHelp()
		result.ReloadedItems["helpdoc"] = "帮助文档已标记重载（需通过全局重载接口）"
	}

	// 重载游戏系统模板
	if len(manifest.Contents.Template) > 0 {
		templateDir := filepath.Join(pkg.InstallPath, "templates")
		if err := pm.parent.GameSystemTemplateReload(templateDir); err != nil {
			result.ReloadedItems["template"] = "游戏系统模板重载失败: " + err.Error()
		} else {
			result.ReloadedItems["template"] = "游戏系统模板已重载"
		}
	}

	// 生成消息
	if len(result.ReloadedItems) == 0 && !result.NeedRestart {
		result.Message = "扩展包无需重载的资源"
	} else {
		reloadCount := len(result.ReloadedItems)
		restartCount := len(result.RestartHints)
		if reloadCount > 0 && restartCount > 0 {
			result.Message = "部分资源已重载，部分资源需要重启"
		} else if reloadCount > 0 {
			result.Message = "资源重载成功"
		} else {
			result.Message = "扩展包资源需要重启程序"
		}
	}

	// 清除待重载状态
	if len(result.ReloadedItems) > 0 {
		pm.lock.Lock()
		pkg.PendingReload = nil
		pm.saveState()
		pm.lock.Unlock()
	}

	pm.parent.Logger.Infof("扩展包 %s 重载完成: %s", pkgID, result.Message)
	return result, nil
}

// ReloadAll 重载所有已启用的扩展包资源
func (pm *PackageManager) ReloadAll() (*sealpkg.ReloadResult, error) {
	pm.lock.RLock()
	enabledPackages := make([]*sealpkg.Instance, 0)
	for _, pkg := range pm.packages {
		if pkg.State == sealpkg.PackageStateEnabled {
			enabledPackages = append(enabledPackages, pkg)
		}
	}
	pm.lock.RUnlock()

	result := &sealpkg.ReloadResult{
		Success:       true,
		ReloadedItems: make(map[string]string),
		RestartHints:  make([]string, 0),
	}

	hasScripts := false
	hasDecks := false
	hasReply := false
	hasHelpdoc := false
	hasTemplate := false

	// 汇总所有启用包的资源类型
	for _, pkg := range enabledPackages {
		if pkg.Manifest == nil {
			continue
		}
		if len(pkg.Manifest.Contents.Scripts) > 0 {
			hasScripts = true
		}
		if len(pkg.Manifest.Contents.Decks) > 0 {
			hasDecks = true
		}
		if len(pkg.Manifest.Contents.Reply) > 0 {
			hasReply = true
		}
		if len(pkg.Manifest.Contents.Helpdoc) > 0 {
			hasHelpdoc = true
		}
		if len(pkg.Manifest.Contents.Template) > 0 {
			hasTemplate = true
		}
	}

	// 执行重载（每种资源类型只重载一次）
	if hasScripts {
		pm.parent.JsReload()
		result.ReloadedItems["scripts"] = "JS 脚本已重载"
	}

	if hasDecks {
		DeckReload(pm.parent)
		result.ReloadedItems["decks"] = "牌堆已重载"
	}

	if hasReply {
		ReplyReload(pm.parent)
		result.ReloadedItems["reply"] = "自动回复已重载"
	}

	if hasHelpdoc {
		result.ReloadedItems["helpdoc"] = "帮助文档已标记重载（需通过全局重载接口）"
	}

	if hasTemplate {
		// 重载所有扩展包中的模板
		templateDir := filepath.Join(pm.parent.BaseConfig.DataDir, "data", "game-templates")
		if err := pm.parent.GameSystemTemplateReload(templateDir); err != nil {
			result.ReloadedItems["template"] = "游戏系统模板重载失败: " + err.Error()
		} else {
			result.ReloadedItems["template"] = "游戏系统模板已重载"
		}
	}

	if len(result.ReloadedItems) == 0 && !result.NeedRestart {
		result.Message = "没有需要重载的资源"
	} else {
		result.Message = "所有扩展包资源重载完成"
	}

	// 清除所有包的待重载状态
	if len(result.ReloadedItems) > 0 {
		pm.lock.Lock()
		for _, pkg := range pm.packages {
			pkg.PendingReload = nil
		}
		pm.saveState()
		pm.lock.Unlock()
	}

	pm.parent.Logger.Infof("扩展包全局重载: %s", result.Message)
	return result, nil
}

// linkPackageResources 链接扩展包资源
// 新设计：资源保留在包目录中，由各 Reload 函数直接从包目录加载
// 此函数仅作为启用时的钩子，不再复制文件
func (pm *PackageManager) linkPackageResources(pkg *sealpkg.Instance) error {
	// 资源保留在原目录，不再复制到全局目录
	// 各 Reload 函数会从已启用包的目录中加载资源
	return nil
}

// unlinkPackageResources 取消链接扩展包资源
// 新设计：资源保留在包目录中，禁用后由 Reload 函数跳过加载
// 此函数仅作为禁用时的钩子，不再删除文件
func (pm *PackageManager) unlinkPackageResources(pkg *sealpkg.Instance) error {
	// 资源保留在原目录，不再从全局目录删除
	// 禁用后 Reload 函数会跳过该包的资源
	return nil
}

// copyFile 复制文件
func (pm *PackageManager) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return dstFile.Sync()
}

// InstallFromURL 从 URL 下载并安装扩展包
func (pm *PackageManager) InstallFromURL(url string) error {
	if len(url) == 0 {
		return errors.New("未提供下载链接")
	}

	pm.parent.Logger.Infof("正在从 URL 下载扩展包: %s", url)

	// 下载文件
	statusCode, data, err := GetCloudContent([]string{url}, "")
	if err != nil {
		return errors.New("下载扩展包失败: " + err.Error())
	}
	if statusCode != 200 {
		return errors.New("无法获取扩展包内容，状态码: " + string(rune(statusCode)))
	}

	// 保存到临时文件
	tmpDir := filepath.Join(pm.parent.BaseConfig.DataDir, "temp")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return errors.New("创建临时目录失败: " + err.Error())
	}

	tmpFile := filepath.Join(tmpDir, "package_download_"+time.Now().Format("20060102150405")+".zip")
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return errors.New("保存临时文件失败: " + err.Error())
	}

	// 安装扩展包
	err = pm.Install(tmpFile)

	// 清理临时文件
	os.Remove(tmpFile)

	return err
}
