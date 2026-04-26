package dice

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
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

type packageArtifactCandidate struct {
	Manifest   *sealpkg.Manifest
	SourcePath string
	Version    *semver.Version
}

type PackageContentFile struct {
	PackageID   string
	Path        string
	PackagePath string
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

// Init 初始化包管理器，并恢复已安装的包。
func (pm *PackageManager) Init() error {
	// 确保核心目录存在
	sourcePath := pm.getSourcePackagesPath()
	cachePath := pm.getCachePackagesPath()
	if err := os.MkdirAll(sourcePath, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(cachePath, 0o755); err != nil {
		return err
	}

	if err := pm.loadState(); err != nil {
		pm.parent.Logger.Warnf("加载扩展包状态失败: %v", err)
	}

	candidates, err := pm.scanSourceArtifacts()
	if err != nil {
		return err
	}

	persisted := pm.packages
	loaded := make(map[string]*sealpkg.Instance, len(persisted))

	for pkgID, pkg := range persisted {
		candidate := findCandidateBySourcePath(candidates[pkgID], pkg.SourcePath)
		if candidate == nil {
			candidate = pickHighestCandidate(candidates[pkgID])
			if candidate != nil {
				pm.parent.Logger.Warnf("扩展包 %s 的源文件已变更，将使用最新可用版本 %s", pkgID, candidate.Manifest.Package.Version)
			}
		}
		if candidate == nil {
			pm.parent.Logger.Warnf("扩展包 %s 的源文件丢失，已跳过加载", pkgID)
			continue
		}

		instance, err := pm.materializeCandidate(candidate, pkg)
		if err != nil {
			pm.parent.Logger.Warnf("恢复扩展包 %s 失败: %v", pkgID, err)
			continue
		}
		loaded[pkgID] = instance
		delete(candidates, pkgID)
	}

	for pkgID, pkgCandidates := range candidates {
		candidate := pickHighestCandidate(pkgCandidates)
		if candidate == nil {
			continue
		}
		instance, err := pm.materializeCandidate(candidate, nil)
		if err != nil {
			pm.parent.Logger.Warnf("初始化新扩展包 %s 失败: %v", pkgID, err)
			continue
		}
		loaded[pkgID] = instance
	}

	pm.packages = loaded
	pm.buildDependencyGraph()
	if err := pm.saveState(); err != nil {
		pm.parent.Logger.Warnf("failed to save package state: %v", err)
	}

	return nil
}

func (pm *PackageManager) scanSourceArtifacts() (map[string][]*packageArtifactCandidate, error) {
	candidates := make(map[string][]*packageArtifactCandidate)
	err := filepath.WalkDir(pm.getSourcePackagesPath(), func(pkgFilePath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			pm.parent.Logger.Warnf("扫描扩展包目录时发生错误: %v", walkErr)
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), sealpkg.Extension) {
			return nil
		}

		archiveInfo, err := sealpkg.InspectArchive(pkgFilePath)
		if err != nil {
			pm.parent.Logger.Warnf("检查扩展包失败 %s: %v", d.Name(), err)
			return nil
		}
		version, err := semver.NewVersion(archiveInfo.Manifest.Package.Version)
		if err != nil {
			pm.parent.Logger.Warnf("解析扩展包版本失败 %s: %v", d.Name(), err)
			return nil
		}
		pkgID := archiveInfo.Manifest.Package.ID
		candidates[pkgID] = append(candidates[pkgID], &packageArtifactCandidate{
			Manifest:   archiveInfo.Manifest,
			SourcePath: pkgFilePath,
			Version:    version,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return candidates, nil
}

func (pm *PackageManager) materializeCandidate(candidate *packageArtifactCandidate, persisted *sealpkg.Instance) (*sealpkg.Instance, error) {
	pkgID := candidate.Manifest.Package.ID
	installPath := pm.getPackageInstallPath(pkgID)
	if err := pm.ensureInstallCache(candidate, installPath); err != nil {
		return nil, err
	}

	userDataPath := pm.getUserDataPath(pkgID)
	if err := os.MkdirAll(userDataPath, 0o755); err != nil {
		return nil, err
	}

	config := sealpkg.InitDefaultConfig(candidate.Manifest.Config)
	state := sealpkg.PackageStateDisabled
	installTime := time.Now()
	var pendingReload []string
	if persisted != nil {
		if persisted.State != "" {
			state = persisted.State
		}
		if !persisted.InstallTime.IsZero() {
			installTime = persisted.InstallTime
		}
		config = sealpkg.MergeConfig(config, persisted.Config)
		pendingReload = append([]string(nil), persisted.PendingReload...)
	} else if info, err := os.Stat(candidate.SourcePath); err == nil {
		installTime = info.ModTime()
	}

	return &sealpkg.Instance{
		Manifest:      candidate.Manifest,
		State:         state,
		InstallTime:   installTime,
		InstallPath:   installPath,
		SourcePath:    candidate.SourcePath,
		UserDataPath:  userDataPath,
		Config:        config,
		PendingReload: pendingReload,
	}, nil
}

func (pm *PackageManager) ensureInstallCache(candidate *packageArtifactCandidate, installPath string) error {
	infoPath := filepath.Join(installPath, sealpkg.InfoFile)
	if data, err := os.ReadFile(infoPath); err == nil {
		manifest, parseErr := sealpkg.ParseManifest(data)
		if parseErr == nil && manifest.Package.ID == candidate.Manifest.Package.ID && manifest.Package.Version == candidate.Manifest.Package.Version {
			return nil
		}
	}

	stagedDir, err := pm.stageExtractPackage(candidate.SourcePath, candidate.Manifest.Package.ID)
	if err != nil {
		return err
	}
	return pm.swapInstallDir(stagedDir, installPath)
}

func (pm *PackageManager) stageExtractPackage(pkgPath, pkgID string) (string, error) {
	tempDir, err := os.MkdirTemp(pm.getCachePackagesPath(), strings.ReplaceAll(sealpkg.PackageIDToSafePath(pkgID), string(os.PathSeparator), "-")+"-")
	if err != nil {
		return "", err
	}
	if _, err := sealpkg.ExtractArchive(pkgPath, tempDir); err != nil {
		_ = os.RemoveAll(tempDir)
		return "", err
	}
	return tempDir, nil
}

func (pm *PackageManager) swapInstallDir(stagedDir, finalDir string) error {
	stagedDir = filepath.Clean(stagedDir)
	finalDir = filepath.Clean(finalDir)
	if err := os.MkdirAll(filepath.Dir(finalDir), 0o755); err != nil {
		_ = os.RemoveAll(stagedDir)
		return err
	}

	backupDir := ""
	if _, err := os.Stat(finalDir); err == nil {
		backupDir = finalDir + ".bak-" + time.Now().Format("20060102150405.000000000")
		if err := os.Rename(finalDir, backupDir); err != nil {
			_ = os.RemoveAll(stagedDir)
			return err
		}
	}

	if err := os.Rename(stagedDir, finalDir); err != nil {
		if backupDir != "" {
			_ = os.Rename(backupDir, finalDir)
		}
		_ = os.RemoveAll(stagedDir)
		return err
	}
	if backupDir != "" {
		_ = os.RemoveAll(backupDir)
	}
	return nil
}

func findCandidateBySourcePath(candidates []*packageArtifactCandidate, sourcePath string) *packageArtifactCandidate {
	for _, candidate := range candidates {
		if samePackagePath(candidate.SourcePath, sourcePath) {
			return candidate
		}
	}
	return nil
}

func pickHighestCandidate(candidates []*packageArtifactCandidate) *packageArtifactCandidate {
	var best *packageArtifactCandidate
	for _, candidate := range candidates {
		if best == nil || candidate.Version.GreaterThan(best.Version) {
			best = candidate
		}
	}
	return best
}

func samePackagePath(left, right string) bool {
	if left == "" || right == "" {
		return false
	}
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if os.PathSeparator == '\\' {
		return strings.EqualFold(left, right)
	}
	return left == right
}

// getSourcePackagesPath 获取 .sealpkg 源文件存放目录
// 路径: data/packages/
func (pm *PackageManager) getSourcePackagesPath() string {
	return filepath.Join(".", "data", sealpkg.PackagesDir)
}

// getUserDataPath 获取包的用户数据目录
// 路径: data/extensions/<author>/<package>/_userdata/
func (pm *PackageManager) getUserDataPath(pkgID string) string {
	return filepath.Join(".", "data", "extensions", sealpkg.PackageIDToSafePath(pkgID), sealpkg.UserDataDir)
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

func (pm *PackageManager) getPackageSourceDir(pkgID string) string {
	return filepath.Join(pm.getSourcePackagesPath(), sealpkg.PackageIDToSafePath(pkgID))
}

func (pm *PackageManager) getPackageSourcePath(pkgID, version string) string {
	return filepath.Join(pm.getPackageSourceDir(pkgID), sealpkg.PackageVersionToFileName(version))
}

func (pm *PackageManager) getPackageInstallPath(pkgID string) string {
	return filepath.Join(pm.getCachePackagesPath(), sealpkg.PackageIDToSafePath(pkgID))
}

func (pm *PackageManager) removeEmptyParents(path string, stop string) {
	current := filepath.Clean(path)
	stop = filepath.Clean(stop)
	for current != stop && current != "." {
		if err := os.Remove(current); err != nil {
			break
		}
		current = filepath.Dir(current)
	}
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
			State:         persist.State,
			InstallTime:   persist.InstallTime,
			InstallPath:   persist.InstallPath,
			SourcePath:    persist.SourcePath,
			UserDataPath:  userDataPath,
			Config:        persist.Config,
			PendingReload: append([]string(nil), persist.PendingReload...),
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
			Version:       packageVersionOf(pkg),
			State:         pkg.State,
			InstallTime:   pkg.InstallTime,
			InstallPath:   pkg.InstallPath,
			SourcePath:    pkg.SourcePath,
			UserDataPath:  pkg.UserDataPath,
			Config:        pkg.Config,
			PendingReload: append([]string(nil), pkg.PendingReload...),
		}
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(pm.getStatePath(), data, 0644)
}

// Install 安装扩展包
// 将 .sealpkg 复制到 data/packages/，并解压到 cache/packages/
func (pm *PackageManager) Install(pkgPath string) error {
	pm.lock.Lock()
	defer pm.lock.Unlock()

	archiveInfo, err := sealpkg.InspectArchive(pkgPath)
	if err != nil {
		return err
	}
	manifest := archiveInfo.Manifest
	pkgID := manifest.Package.ID
	newVersion, err := semver.NewVersion(manifest.Package.Version)
	if err != nil {
		return err
	}

	if err := sealpkg.CheckSealVersion(manifest, VERSION.String()); err != nil {
		return err
	}
	if satisfied, missing := pm.CheckDependencies(manifest); !satisfied {
		return &DependencyError{
			PackageID:   pkgID,
			MissingDeps: missing,
		}
	}

	destDir := pm.getPackageSourceDir(pkgID)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	destPkgPath := pm.getPackageSourcePath(pkgID, manifest.Package.Version)
	if _, err := os.Stat(destPkgPath); err == nil {
		return errors.New("the package version is already installed")
	}

	var existing *sealpkg.Instance
	if current, exists := pm.packages[pkgID]; exists {
		existing = current
		if current.Manifest != nil {
			existingVersion, parseErr := semver.NewVersion(current.Manifest.Package.Version)
			if parseErr == nil && !newVersion.GreaterThan(existingVersion) {
				return errors.New("the same or a newer package version is already installed")
			}
		}
	}

	stagedSourcePath, err := pm.stageSourceArtifact(pkgPath, destDir)
	if err != nil {
		return err
	}
	stagedCachePath, err := pm.stageExtractPackage(pkgPath, pkgID)
	if err != nil {
		_ = os.Remove(stagedSourcePath)
		return err
	}

	if err := os.Rename(stagedSourcePath, destPkgPath); err != nil {
		_ = os.Remove(stagedSourcePath)
		_ = os.RemoveAll(stagedCachePath)
		return err
	}
	if err := pm.swapInstallDir(stagedCachePath, pm.getPackageInstallPath(pkgID)); err != nil {
		_ = os.Remove(destPkgPath)
		pm.removeEmptyParents(filepath.Dir(destPkgPath), pm.getSourcePackagesPath())
		return err
	}

	userDataPath := pm.getUserDataPath(pkgID)
	if err := os.MkdirAll(userDataPath, 0o755); err != nil {
		return err
	}

	config := sealpkg.InitDefaultConfig(manifest.Config)
	state := sealpkg.PackageStateDisabled
	pendingReload := []string(nil)
	if existing != nil {
		state = existing.State
		config = sealpkg.MergeConfig(config, existing.Config)
		if state == sealpkg.PackageStateEnabled {
			pendingReload = append(pendingReload, pm.generateReloadHints(manifest).ReloadHints...)
		}
	}

	pm.packages[pkgID] = &sealpkg.Instance{
		Manifest:      manifest,
		State:         state,
		InstallTime:   time.Now(),
		InstallPath:   pm.getPackageInstallPath(pkgID),
		SourcePath:    destPkgPath,
		UserDataPath:  userDataPath,
		Config:        config,
		PendingReload: pendingReload,
	}

	if existing != nil && existing.SourcePath != "" && !samePackagePath(existing.SourcePath, destPkgPath) {
		_ = os.Remove(existing.SourcePath)
		pm.removeEmptyParents(filepath.Dir(existing.SourcePath), pm.getSourcePackagesPath())
	}

	pm.buildDependencyGraph()
	if err := pm.saveState(); err != nil {
		pm.parent.Logger.Warnf("failed to save package state: %v", err)
	}

	pm.parent.Logger.Infof("package %s v%s installed", manifest.Package.Name, manifest.Package.Version)
	return nil
}

func (pm *PackageManager) stageSourceArtifact(srcPath, destDir string) (string, error) {
	tempFile, err := os.CreateTemp(destDir, "staged-*.sealpkg")
	if err != nil {
		return "", err
	}
	tempPath := tempFile.Name()
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return "", err
	}
	if err := pm.copyFile(srcPath, tempPath); err != nil {
		_ = os.Remove(tempPath)
		return "", errors.New("failed to copy extension package: " + err.Error())
	}
	return tempPath, nil
}

func packageVersionOf(pkg *sealpkg.Instance) string {
	if pkg == nil || pkg.Manifest == nil {
		return ""
	}
	return pkg.Manifest.Package.Version
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
			manifestPath := filepath.Join(pkg.InstallPath, sealpkg.InfoFile)
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
		if err := os.RemoveAll(pkg.InstallPath); err != nil {
			return err
		}
		pm.removeEmptyParents(filepath.Dir(pkg.InstallPath), pm.getCachePackagesPath())
		if pkg.SourcePath != "" {
			_ = os.Remove(pkg.SourcePath)
			pm.removeEmptyParents(filepath.Dir(pkg.SourcePath), pm.getSourcePackagesPath())
		}
		if pkg.UserDataPath != "" {
			if err := os.RemoveAll(pkg.UserDataPath); err != nil {
				return err
			}
			pm.removeEmptyParents(filepath.Dir(pkg.UserDataPath), filepath.Join(".", "data", "extensions"))
		}
		delete(pm.packages, pkgID)

	case sealpkg.UninstallModeKeepData:
		if err := os.RemoveAll(pkg.InstallPath); err != nil {
			return err
		}
		pm.removeEmptyParents(filepath.Dir(pkg.InstallPath), pm.getCachePackagesPath())
		if pkg.SourcePath != "" {
			_ = os.Remove(pkg.SourcePath)
			pm.removeEmptyParents(filepath.Dir(pkg.SourcePath), pm.getSourcePackagesPath())
		}
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
	// 禁用会移除运行时缓存，重新启用时先恢复缓存目录。
	if err := pm.linkPackageResources(pkg); err != nil {
		pkg.ErrText = err.Error()
		pm.saveState()
		return nil, err
	}
	pkg.State = sealpkg.PackageStateEnabled
	pkg.ErrText = ""

	// 生成重载提示并设置待重载状态
	result := pm.generateReloadHints(pkg.Manifest)
	result.Success = true
	result.Message = "扩展包已启用，需要重载相应资源才能生效"

	// 设置待重载状态，供 UI 查询
	pkg.PendingReload = nil
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
	pkg.PendingReload = nil
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
	if err := os.MkdirAll(pkg.UserDataPath, 0o755); err != nil {
		return err
	}
	configPath := filepath.Join(pkg.UserDataPath, sealpkg.ConfigFile)
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
func packageInstanceSortKey(pkg *sealpkg.Instance) string {
	if pkg == nil {
		return ""
	}
	if pkg.Manifest != nil && pkg.Manifest.Package.ID != "" {
		return pkg.Manifest.Package.ID
	}
	if pkg.SourcePath != "" {
		return filepath.ToSlash(pkg.SourcePath)
	}
	return filepath.ToSlash(pkg.InstallPath)
}

func (pm *PackageManager) List() []*sealpkg.Instance {
	pm.lock.RLock()
	defer pm.lock.RUnlock()

	result := make([]*sealpkg.Instance, 0, len(pm.packages))
	for _, pkg := range pm.packages {
		result = append(result, pkg)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return packageInstanceSortKey(result[i]) < packageInstanceSortKey(result[j])
	})
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
	sort.SliceStable(result, func(i, j int) bool {
		return packageInstanceSortKey(result[i]) < packageInstanceSortKey(result[j])
	})
	return result
}

// GetEnabledContentFiles 获取所有已启用包中某种资源类型的文件列表
func (pm *PackageManager) GetEnabledContentFiles(contentType string) []PackageContentFile {
	pm.lock.RLock()
	defer pm.lock.RUnlock()
	return pm.getEnabledContentFilesLocked(contentType)
}

func (pm *PackageManager) getEnabledContentFilesLocked(contentType string) []PackageContentFile {
	pkgIDs := make([]string, 0, len(pm.packages))
	for pkgID, pkg := range pm.packages {
		if pkg != nil && pkg.State == sealpkg.PackageStateEnabled && pkg.Manifest != nil {
			pkgIDs = append(pkgIDs, pkgID)
		}
	}
	sort.Strings(pkgIDs)

	result := make([]PackageContentFile, 0)
	for _, pkgID := range pkgIDs {
		result = append(result, pm.collectPackageContentFiles(pm.packages[pkgID], contentType)...)
	}
	return result
}

func (pm *PackageManager) collectPackageContentFiles(pkg *sealpkg.Instance, contentType string) []PackageContentFile {
	patterns := packageContentPatterns(pkg.Manifest, contentType)
	if len(patterns) == 0 {
		return nil
	}

	baseDir := filepath.Join(pkg.InstallPath, contentType)
	if _, err := os.Stat(baseDir); err != nil {
		return nil
	}

	files := make([]PackageContentFile, 0)
	seen := make(map[string]struct{})
	_ = filepath.WalkDir(baseDir, func(currentPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(pkg.InstallPath, currentPath)
		if err != nil {
			return nil
		}
		packagePath := filepath.ToSlash(relPath)
		if !matchesPackageContentPatterns(patterns, packagePath) {
			return nil
		}
		if _, exists := seen[packagePath]; exists {
			return nil
		}
		seen[packagePath] = struct{}{}
		files = append(files, PackageContentFile{
			PackageID:   pkg.Manifest.Package.ID,
			Path:        currentPath,
			PackagePath: packagePath,
		})
		return nil
	})
	sort.Slice(files, func(i, j int) bool {
		return files[i].PackagePath < files[j].PackagePath
	})
	return files
}

func packageContentPatterns(manifest *sealpkg.Manifest, contentType string) []string {
	if manifest == nil {
		return nil
	}
	switch contentType {
	case "scripts":
		return manifest.Contents.Scripts
	case "decks":
		return manifest.Contents.Decks
	case "reply":
		return manifest.Contents.Reply
	case "helpdoc":
		return manifest.Contents.Helpdoc
	case "templates":
		return manifest.Contents.Templates
	default:
		return nil
	}
}

func matchesPackageContentPatterns(patterns []string, packagePath string) bool {
	for _, pattern := range patterns {
		if matchPackageContentPattern(pattern, packagePath) {
			return true
		}
	}
	return false
}

func matchPackageContentPattern(pattern, packagePath string) bool {
	patternSegments := strings.Split(filepath.ToSlash(pattern), "/")
	pathSegments := strings.Split(filepath.ToSlash(packagePath), "/")
	return matchPackageContentPatternSegments(patternSegments, pathSegments)
}

func matchPackageContentPatternSegments(patternSegments, pathSegments []string) bool {
	if len(patternSegments) == 0 {
		return len(pathSegments) == 0
	}
	if patternSegments[0] == "**" {
		if matchPackageContentPatternSegments(patternSegments[1:], pathSegments) {
			return true
		}
		for idx := range pathSegments {
			if matchPackageContentPatternSegments(patternSegments[1:], pathSegments[idx+1:]) {
				return true
			}
		}
		return false
	}
	if len(pathSegments) == 0 {
		return false
	}
	matched, err := path.Match(patternSegments[0], pathSegments[0])
	if err != nil || !matched {
		return false
	}
	return matchPackageContentPatternSegments(patternSegments[1:], pathSegments[1:])
}

// GetEnabledContentDirs 获取已启用包中包含该资源的目录列表
func (pm *PackageManager) GetEnabledContentDirs(contentType string) []string {
	files := pm.GetEnabledContentFiles(contentType)
	dirSet := make(map[string]struct{}, len(files))
	dirs := make([]string, 0, len(files))
	for _, file := range files {
		dir := filepath.Dir(file.Path)
		if _, exists := dirSet[dir]; exists {
			continue
		}
		dirSet[dir] = struct{}{}
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
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

	if len(manifest.Contents.Templates) > 0 {
		hints = append(hints, "游戏系统模板 - 可通过重载接口生效")
	}

	return &sealpkg.OperationResult{
		ReloadNeeded: len(hints) > 0,
		ReloadHints:  hints,
	}
}

// Reload 按指定扩展包声明的资源类型触发重载。
// 对已禁用的扩展包也允许执行，以便把内存中的旧资源清掉。
func (pm *PackageManager) Reload(pkgID string) (*sealpkg.ReloadResult, error) {
	pm.lock.RLock()
	pkg, exists := pm.packages[pkgID]
	pm.lock.RUnlock()

	if !exists {
		return nil, errors.New("扩展包不存在: " + pkgID)
	}
	manifest := pkg.Manifest
	if manifest == nil {
		return &sealpkg.ReloadResult{Success: false, Message: "扩展包 manifest 缺失"}, nil
	}

	exec := pm.reloadPackageContent(packageReloadContentFlagsFromManifest(manifest))
	result := exec.result

	pm.lock.Lock()
	pendingChanged := pm.clearPendingReloadForAllLocked(exec.succeeded)
	if pendingChanged {
		_ = pm.saveState()
	}
	pm.lock.Unlock()

	pm.parent.Logger.Infof("扩展包 %s 重载完成: %s", pkgID, result.Message)
	return result, nil
}

// ReloadAll 重载所有已启用扩展包的资源，并应用待重载的禁用变更。
// ReloadByContent reloads package resources globally by content type and clears matching pending flags.
func (pm *PackageManager) ReloadByContent(contentType string) (*sealpkg.ReloadResult, error) {
	flags, err := packageReloadContentFlagsFromContentType(contentType)
	if err != nil {
		return nil, err
	}

	exec := pm.reloadPackageContent(flags)
	result := exec.result

	pm.lock.Lock()
	pendingChanged := pm.clearPendingReloadForAllLocked(exec.succeeded)
	if pendingChanged {
		_ = pm.saveState()
	}
	pm.lock.Unlock()

	pm.parent.Logger.Infof("package reload by content finished (%s): %s", strings.TrimSpace(contentType), result.Message)
	return result, nil
}

func (pm *PackageManager) ReloadAll() (*sealpkg.ReloadResult, error) {
	pm.lock.RLock()
	flags := packageReloadContentFlags{}
	for _, pkg := range pm.packages {
		if pkg == nil || pkg.Manifest == nil {
			continue
		}
		if pkg.State != sealpkg.PackageStateEnabled && len(pkg.PendingReload) == 0 {
			continue
		}
		flags.merge(packageReloadContentFlagsFromManifest(pkg.Manifest))
	}
	pm.lock.RUnlock()

	exec := pm.reloadPackageContent(flags)
	result := exec.result

	pm.lock.Lock()
	pendingChanged := false
	for _, pkg := range pm.packages {
		if pm.clearPendingReloadLocked(pkg, exec.succeeded) {
			pendingChanged = true
		}
	}
	if pendingChanged {
		_ = pm.saveState()
	}
	pm.lock.Unlock()

	pm.parent.Logger.Infof("扩展包全局重载完成: %s", result.Message)
	return result, nil
}

type packageReloadContentFlags struct {
	scripts   bool
	decks     bool
	reply     bool
	helpdoc   bool
	templates bool
}

type packageReloadExecution struct {
	result    *sealpkg.ReloadResult
	succeeded packageReloadContentFlags
	failed    packageReloadContentFlags
}

func packageReloadContentFlagsFromManifest(manifest *sealpkg.Manifest) packageReloadContentFlags {
	if manifest == nil {
		return packageReloadContentFlags{}
	}
	return packageReloadContentFlags{
		scripts:   len(manifest.Contents.Scripts) > 0,
		decks:     len(manifest.Contents.Decks) > 0,
		reply:     len(manifest.Contents.Reply) > 0,
		helpdoc:   len(manifest.Contents.Helpdoc) > 0,
		templates: len(manifest.Contents.Templates) > 0,
	}
}

func packageReloadContentFlagsFromContentType(contentType string) (packageReloadContentFlags, error) {
	switch strings.TrimSpace(strings.ToLower(contentType)) {
	case "scripts":
		return packageReloadContentFlags{scripts: true}, nil
	case "decks":
		return packageReloadContentFlags{decks: true}, nil
	case "reply":
		return packageReloadContentFlags{reply: true}, nil
	case "helpdoc":
		return packageReloadContentFlags{helpdoc: true}, nil
	case "templates":
		return packageReloadContentFlags{templates: true}, nil
	default:
		return packageReloadContentFlags{}, errors.New("unsupported reload content type: " + contentType)
	}
}

func (flags *packageReloadContentFlags) merge(other packageReloadContentFlags) {
	flags.scripts = flags.scripts || other.scripts
	flags.decks = flags.decks || other.decks
	flags.reply = flags.reply || other.reply
	flags.helpdoc = flags.helpdoc || other.helpdoc
	flags.templates = flags.templates || other.templates
}

func (flags packageReloadContentFlags) contains(kind string) bool {
	switch kind {
	case "scripts":
		return flags.scripts
	case "decks":
		return flags.decks
	case "reply":
		return flags.reply
	case "helpdoc":
		return flags.helpdoc
	case "templates":
		return flags.templates
	default:
		return false
	}
}

func (flags packageReloadContentFlags) count() int {
	count := 0
	if flags.scripts {
		count++
	}
	if flags.decks {
		count++
	}
	if flags.reply {
		count++
	}
	if flags.helpdoc {
		count++
	}
	if flags.templates {
		count++
	}
	return count
}

func buildReloadResultMessage(successCount, failedCount int, needRestart bool) string {
	switch {
	case successCount == 0 && failedCount == 0 && !needRestart:
		return "没有需要重载的资源"
	case successCount == 0 && failedCount == 0:
		return "需要重启后才能生效"
	case successCount > 0 && failedCount == 0 && needRestart:
		return "资源重载完成，部分内容仍需重启才能生效"
	case successCount > 0 && failedCount == 0:
		return "资源重载完成"
	case successCount == 0 && failedCount > 0 && needRestart:
		return "资源重载失败，部分内容仍需重启才能生效"
	case successCount == 0 && failedCount > 0:
		return "资源重载失败"
	case needRestart:
		return "部分资源重载成功，部分失败，另有内容仍需重启"
	default:
		return "部分资源重载成功，部分失败"
	}
}

func reloadHintMatchesContentType(hint, kind string) bool {
	hint = strings.TrimSpace(hint)
	switch kind {
	case "scripts":
		return hint == "scripts" || strings.HasPrefix(hint, "JS 脚本")
	case "decks":
		return hint == "decks" || strings.HasPrefix(hint, "牌堆")
	case "reply":
		return hint == "reply" || strings.HasPrefix(hint, "自动回复") || strings.HasPrefix(hint, "自定义回复")
	case "helpdoc":
		return hint == "helpdoc" || strings.HasPrefix(hint, "帮助文档")
	case "templates":
		return hint == "templates" || strings.HasPrefix(hint, "游戏系统模板")
	default:
		return false
	}
}

func (pm *PackageManager) clearPendingReloadLocked(pkg *sealpkg.Instance, succeeded packageReloadContentFlags) bool {
	if pkg == nil || len(pkg.PendingReload) == 0 {
		return false
	}

	remaining := make([]string, 0, len(pkg.PendingReload))
	changed := false
	for _, hint := range pkg.PendingReload {
		shouldClear := false
		for _, kind := range []string{"scripts", "decks", "reply", "helpdoc", "templates"} {
			if succeeded.contains(kind) && reloadHintMatchesContentType(hint, kind) {
				shouldClear = true
				break
			}
		}
		if shouldClear {
			changed = true
			continue
		}
		remaining = append(remaining, hint)
	}
	if !changed {
		return false
	}
	if len(remaining) == 0 {
		pkg.PendingReload = nil
	} else {
		pkg.PendingReload = remaining
	}
	return true
}

func (pm *PackageManager) clearPendingReloadForAllLocked(succeeded packageReloadContentFlags) bool {
	changed := false
	for _, pkg := range pm.packages {
		if pm.clearPendingReloadLocked(pkg, succeeded) {
			changed = true
		}
	}
	return changed
}

func (pm *PackageManager) reloadPackageContent(flags packageReloadContentFlags) packageReloadExecution {
	result := &sealpkg.ReloadResult{
		Success:       true,
		ReloadedItems: make(map[string]string),
		RestartHints:  make([]string, 0),
	}
	exec := packageReloadExecution{result: result}

	if flags.scripts {
		pm.parent.JsReload()
		result.ReloadedItems["scripts"] = "JS 脚本已重载"
		exec.succeeded.scripts = true
	}
	if flags.decks {
		DeckReload(pm.parent)
		result.ReloadedItems["decks"] = "牌堆已重载"
		exec.succeeded.decks = true
	}
	if flags.reply {
		ReplyReload(pm.parent)
		result.ReloadedItems["reply"] = "自定义回复已重载"
		exec.succeeded.reply = true
	}
	if flags.helpdoc {
		if err := pm.reloadHelpdocs(); err != nil {
			result.ReloadedItems["helpdoc"] = "帮助文档重载失败: " + err.Error()
			exec.failed.helpdoc = true
			result.Success = false
		} else {
			result.ReloadedItems["helpdoc"] = "帮助文档已重载"
			exec.succeeded.helpdoc = true
		}
	}
	if flags.templates {
		if err := pm.reloadTemplates(); err != nil {
			result.ReloadedItems["templates"] = "游戏系统模板重载失败: " + err.Error()
			exec.failed.templates = true
			result.Success = false
		} else {
			result.ReloadedItems["templates"] = "游戏系统模板已重载"
			exec.succeeded.templates = true
		}
	}

	result.Message = buildReloadResultMessage(exec.succeeded.count(), exec.failed.count(), result.NeedRestart)
	return exec
}

func (pm *PackageManager) reloadHelpdocs() error {
	if pm.parent == nil || pm.parent.Parent == nil {
		return errors.New("help manager is unavailable")
	}
	pm.parent.Parent.InitHelp()
	return nil
}

func (pm *PackageManager) reloadTemplates() error {
	templateFiles := pm.GetEnabledContentFiles("templates")
	paths := make([]string, 0, len(templateFiles))
	for _, file := range templateFiles {
		paths = append(paths, file.Path)
	}
	return pm.parent.GameSystemTemplateReloadFiles(paths)
}

// linkPackageResources 恢复扩展包运行时缓存目录。
func (pm *PackageManager) linkPackageResources(pkg *sealpkg.Instance) error {
	if pkg == nil || pkg.Manifest == nil {
		return errors.New("扩展包 manifest 缺失")
	}
	if pkg.SourcePath == "" {
		return errors.New("扩展包源文件路径缺失")
	}
	if pkg.InstallPath == "" {
		pkg.InstallPath = pm.getPackageInstallPath(pkg.Manifest.Package.ID)
	}
	return pm.ensureInstallCache(&packageArtifactCandidate{
		Manifest:   pkg.Manifest,
		SourcePath: pkg.SourcePath,
	}, pkg.InstallPath)
}

// unlinkPackageResources 移除运行时缓存目录，等待重载后从内存中清理资源。
func (pm *PackageManager) unlinkPackageResources(pkg *sealpkg.Instance) error {
	if pkg == nil || pkg.InstallPath == "" {
		return nil
	}
	if err := os.RemoveAll(pkg.InstallPath); err != nil {
		return err
	}
	pm.removeEmptyParents(filepath.Dir(pkg.InstallPath), pm.getCachePackagesPath())
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

func verifyDownloadedPackageHash(data []byte, hashes map[string]string) error {
	if len(hashes) == 0 {
		return nil
	}
	for algorithm, expected := range hashes {
		if !strings.EqualFold(algorithm, "sha256") {
			continue
		}
		expected = strings.TrimSpace(expected)
		if expected == "" {
			return errors.New("download.hash.sha256 不能为空")
		}
		sum := sha256.Sum256(data)
		actual := hex.EncodeToString(sum[:])
		if !strings.EqualFold(actual, expected) {
			return fmt.Errorf("扩展包 SHA-256 校验失败，期望 %s，实际 %s", strings.ToLower(expected), actual)
		}
		return nil
	}
	return nil
}

func unsupportedPackageHashAlgorithms(hashes map[string]string) []string {
	if len(hashes) == 0 {
		return nil
	}
	result := make([]string, 0, len(hashes))
	for algorithm := range hashes {
		if strings.EqualFold(algorithm, "sha256") {
			continue
		}
		result = append(result, algorithm)
	}
	sort.Strings(result)
	return result
}

// InstallFromURL 从 URL 下载并安装扩展包
func (pm *PackageManager) InstallFromURL(url string, hashes map[string]string) error {
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
		return fmt.Errorf("无法获取扩展包内容，状态码: %d", statusCode)
	}
	if err := verifyDownloadedPackageHash(data, hashes); err != nil {
		return err
	}
	if unsupported := unsupportedPackageHashAlgorithms(hashes); len(unsupported) > 0 && pm.parent != nil && pm.parent.Logger != nil {
		pm.parent.Logger.Warnf("下载校验目前仅支持 sha256，已忽略以下算法: %s", strings.Join(unsupported, ", "))
	}

	// 保存到临时文件
	tmpDir := filepath.Join(pm.parent.BaseConfig.DataDir, "temp")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return errors.New("创建临时目录失败: " + err.Error())
	}

	tmpFile := filepath.Join(tmpDir, "package_download_"+time.Now().Format("20060102150405")+".sealpkg")
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return errors.New("保存临时文件失败: " + err.Error())
	}

	// 安装扩展包
	err = pm.Install(tmpFile)

	// 清理临时文件
	os.Remove(tmpFile)

	return err
}
