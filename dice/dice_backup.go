package dice

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alexmullins/zip"

	"sealdice-core/dice/service"
	"sealdice-core/logger"
	"sealdice-core/utils"
	"sealdice-core/utils/constant"
	"sealdice-core/utils/crypto"
)

const BackupDir = "./backups"

var backupOperationMu sync.Mutex

type BackupCleanStrategy int

const (
	BackupCleanStrategyDisabled BackupCleanStrategy = iota
	BackupCleanStrategyByCount
	BackupCleanStrategyByTime
)

type BackupCleanTrigger int

const (
	// BackupCleanTriggerCron 通过独立定时任务触发
	BackupCleanTriggerCron BackupCleanTrigger = 1 << iota
	// BackupCleanTriggerRotate 通过自动备份触发
	BackupCleanTriggerRotate
)

type backupConfigGlobal struct {
	Global  bool                         `json:"global"`
	Decks   bool                         `json:"decks"`
	HelpDoc bool                         `json:"helpDoc"`
	Censor  bool                         `json:"censor"`
	Names   bool                         `json:"names"`
	Images  bool                         `json:"images"`
	Dices   map[string]*backupConfigDice `json:"dices"`
}

type backupConfigDice struct {
	Accounts    bool `json:"accounts"`    // 帐号
	MiscConfig  bool `json:"miscConfig"`  // 综合设置
	PlayerData  bool `json:"playerData"`  // 用户数据
	CustomReply bool `json:"customReply"` // 文案模板
	CustomText  bool `json:"customText"`  // 自定义回复
	JSScripts   bool `json:"jsScripts"`   // JS脚本
}

type BackupSelection uint64

const (
	BackupSelectionJS BackupSelection = 1 << iota
	BackupSelectionDecks
	BackupSelectionHelpDoc
	BackupSelectionCensor
	BackupSelectionNames
	BackupSelectionImages

	BackupSelectionBasic     BackupSelection = 0
	BackupSelectionResources BackupSelection = BackupSelectionImages
	BackupSelectionAll       BackupSelection = BackupSelectionBasic |
		BackupSelectionJS |
		BackupSelectionDecks |
		BackupSelectionHelpDoc |
		BackupSelectionCensor |
		BackupSelectionNames |
		BackupSelectionResources
)

func (dm *DiceManager) Backup(sel BackupSelection, fromAuto bool) (string, error) {
	backupOperationMu.Lock()
	defer backupOperationMu.Unlock()
	return dm.backup(sel, fromAuto)
}

func (dm *DiceManager) backup(sel BackupSelection, fromAuto bool) (string, error) {
	return dm.backupWithOptions(sel, fromAuto, false)
}

func (dm *DiceManager) backupStrict(sel BackupSelection) (string, error) {
	return dm.backupWithOptions(sel, false, true)
}

func (dm *DiceManager) backupWithOptions(sel BackupSelection, fromAuto, strict bool) (result string, resultErr error) {
	if dm == nil || len(dm.Dice) == 0 {
		return "", errors.New("没有可备份的骰子实例")
	}
	if err := ensureBackupDirectoryLocked(); err != nil {
		return "", err
	}
	log := logger.M()

	cfgGlb := backupConfigGlobal{
		Global: true,
		Dices:  map[string]*backupConfigDice{},
	}
	cfgDice := backupConfigDice{
		Accounts:    true,
		MiscConfig:  true,
		PlayerData:  true,
		CustomReply: true,
		CustomText:  true,
	}

	bakFn := "bak_" + time.Now().Format("060102_150405")
	if fromAuto {
		bakFn += "_auto"
	}
	bakFn += "_r" + strconv.FormatUint(uint64(sel), 16)
	fnHashed := crypto.CalculateSHA512Str([]byte(bakFn + strconv.FormatInt(time.Now().UnixNano(), 16)))[:8]
	bakFn += "_" + fnHashed + ".zip"
	target := filepath.Join(BackupDir, bakFn)

	fzip, err := os.CreateTemp(BackupDir, ".bak-*.tmp")
	if err != nil {
		return "", err
	}
	tmpName := fzip.Name()
	published := false
	defer func() {
		if !published {
			_ = fzip.Close()
			_ = os.Remove(tmpName)
		}
	}()
	writer := zip.NewWriter(fzip)
	defer func() {
		if !published {
			_ = writer.Close()
		}
	}()

	fileOK := func(fn string) bool {
		stat, statErr := os.Stat(fn)
		return statErr == nil && !stat.IsDir()
	}
	dirOK := func(fn string) bool {
		stat, statErr := os.Stat(fn)
		return statErr == nil && stat.IsDir()
	}

	manifestFiles := make([]backupManifestFile, 0, 32)
	seen := map[string]struct{}{}
	var fatalErr error
	var totalUncompressed uint64
	logBackupError := func(d *Dice, fn string, err error) {
		if d != nil && d.Logger != nil {
			d.Logger.Errorf("备份文件失败: %s, 原因: %s", fn, err.Error())
			return
		}
		log.Errorf("备份文件失败: %s, 原因: %s", fn, err.Error())
	}
	// recordBackupError keeps strict backups all-or-nothing. Best-effort
	// backups only abort for required files, so one broken optional file or a
	// symlinked directory can no longer stop automatic backups entirely.
	recordBackupError := func(d *Dice, fn string, err error, required bool) {
		logBackupError(d, fn, err)
		if required || strict {
			fatalErr = errors.Join(fatalErr, err)
		}
	}
	backupArchiveName := func(fn string) (string, error) {
		if archiveName, pathErr := managedArchivePath(fn); pathErr == nil || strict {
			return archiveName, pathErr
		}
		// Keep legacy deployments working in best-effort mode even when the
		// data directory or one of its parents is a symlink. Strict safety
		// backups keep the managed-path requirement.
		root, rootErr := filepath.Abs("data")
		if rootErr != nil {
			return "", rootErr
		}
		targetAbs, targetErr := filepath.Abs(fn)
		if targetErr != nil {
			return "", targetErr
		}
		relative, relErr := filepath.Rel(root, targetAbs)
		if relErr != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
			return "", fmt.Errorf("备份文件 %s 不在 data 目录内", fn)
		}
		return filepath.ToSlash(filepath.Join("data", relative)), nil
	}
	backupFile := func(d *Dice, fn string, required bool) {
		if strict && fatalErr != nil {
			return
		}
		info, statErr := os.Lstat(fn)
		if statErr != nil {
			if errors.Is(statErr, os.ErrNotExist) && !required {
				return
			}
			recordBackupError(d, fn, fmt.Errorf("备份 %s: %w", fn, statErr), required)
			return
		}
		if !info.Mode().IsRegular() {
			recordBackupError(d, fn, fmt.Errorf("备份 %s: 不是普通文件或为符号链接", fn), required)
			return
		}
		archiveName, pathErr := backupArchiveName(fn)
		if pathErr != nil {
			recordBackupError(d, fn, pathErr, required)
			return
		}
		archiveName, nameErr := normalizeBackupEntryName(filepath.ToSlash(archiveName), false)
		if nameErr != nil {
			recordBackupError(d, fn, fmt.Errorf("备份条目 %s: %w", fn, nameErr), required)
			return
		}
		if isProcessOwnedRestorePath(archiveName) {
			log.Warnf("跳过进程级占用文件: %s", archiveName)
			return
		}
		if isSQLiteSidecarPath(archiveName) {
			log.Warnf("跳过 SQLite 临时文件: %s", archiveName)
			return
		}
		if uint64(info.Size()) > maxBackupEntryUncompressedSize {
			recordBackupError(d, fn, fmt.Errorf("备份条目 %s 超过单文件大小限制", archiveName), required)
			return
		}
		if math.MaxUint64-totalUncompressed < uint64(info.Size()) ||
			totalUncompressed+uint64(info.Size()) > maxBackupTotalUncompressedSize {
			recordBackupError(d, fn, fmt.Errorf("备份总大小超过限制: %s", archiveName), required)
			return
		}
		key := strings.ToLower(archiveName)
		if _, exists := seen[key]; exists {
			recordBackupError(d, fn, fmt.Errorf("备份路径重复: %s", archiveName), required)
			return
		}
		if int64(len(manifestFiles))+1 > maxBackupEntries {
			recordBackupError(d, fn, fmt.Errorf("备份条目超过 %d 个", maxBackupEntries), required)
			return
		}
		file, openErr := os.Open(fn)
		if openErr != nil {
			recordBackupError(d, fn, fmt.Errorf("打开 %s: %w", fn, openErr), required)
			return
		}
		h := &zip.FileHeader{Name: filepath.ToSlash(archiveName), Method: zip.Deflate, Flags: 0x800}
		h.SetMode(0o600)
		fileWriter, createErr := writer.CreateHeader(h)
		if createErr != nil {
			_ = file.Close()
			recordBackupError(d, fn, fmt.Errorf("创建 ZIP 条目 %s: %w", archiveName, createErr), required)
			return
		}
		hasher := sha256.New()
		written, copyErr := io.Copy(io.MultiWriter(fileWriter, hasher), file)
		closeErr := file.Close()
		if copyErr != nil || closeErr != nil || written != info.Size() {
			if copyErr == nil {
				if closeErr != nil {
					copyErr = closeErr
				} else {
					copyErr = fmt.Errorf("读取大小变化: 预期 %d，实际 %d", info.Size(), written)
				}
			}
			recordBackupError(d, fn, fmt.Errorf("写入 ZIP 条目 %s: %w", archiveName, copyErr), required)
			return
		}
		seen[key] = struct{}{}
		totalUncompressed += uint64(written)
		manifestFiles = append(manifestFiles, backupManifestFile{
			Path:   filepath.ToSlash(archiveName),
			Size:   uint64(written),
			SHA256: hex.EncodeToString(hasher.Sum(nil)),
		})
	}
	walkFiles := func(root string, visit func(string, fs.DirEntry) error) {
		if strict && fatalErr != nil {
			return
		}
		walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				if errors.Is(walkErr, os.ErrNotExist) && path == root {
					return filepath.SkipDir
				}
				if !strict {
					log.Warnf("跳过无法读取的备份路径: %s, 原因: %s", path, walkErr.Error())
					return filepath.SkipDir
				}
				return walkErr
			}
			if entry.Type()&os.ModeSymlink != 0 {
				if strict {
					return fmt.Errorf("备份目录包含符号链接: %s", path)
				}
				log.Warnf("跳过符号链接: %s", path)
				return nil
			}
			return visit(path, entry)
		})
		if walkErr != nil {
			logBackupError(nil, root, walkErr)
			if strict {
				fatalErr = errors.Join(fatalErr, walkErr)
			}
		}
	}

	backupFile(nil, "data/dice.yaml", true)

	if sel&BackupSelectionDecks != 0 {
		cfgGlb.Decks = true
		walkFiles("data/decks", func(path string, info fs.DirEntry) error {
			if !info.IsDir() {
				backupFile(nil, path, strict)
				return nil
			}
			base := filepath.Base(path)
			// 跳过 deck 压缩包解压出的目录
			if strings.HasPrefix(base, "_") && strings.HasSuffix(base, ".deck") {
				if fileOK(filepath.Join(filepath.Dir(path), base[1:])) {
					return filepath.SkipDir
				}
			}
			return nil
		})
	}

	if sel&BackupSelectionHelpDoc != 0 {
		if !dirOK("data/helpdoc") {
			log.Warn("备份 helpdoc 失败: 不存在或不是目录")
		} else {
			cfgGlb.HelpDoc = true
			walkFiles("data/helpdoc", func(path string, info fs.DirEntry) error {
				if !info.IsDir() {
					backupFile(nil, path, strict)
				}
				return nil
			})
		}
	}

	if sel&BackupSelectionCensor != 0 {
		if !dirOK("data/censor") {
			log.Warn("备份 censor 失败: 不存在或不是目录")
		} else {
			cfgGlb.Censor = true
			walkFiles("data/censor", func(path string, info fs.DirEntry) error {
				if !info.IsDir() {
					backupFile(nil, path, strict)
				}
				return nil
			})
		}
	}

	if sel&BackupSelectionNames != 0 {
		if !dirOK("data/names") {
			log.Warn("备份 names 失败: 不存在或不是目录")
		} else {
			cfgGlb.Names = true
			walkFiles("data/names", func(path string, info fs.DirEntry) error {
				if !info.IsDir() {
					backupFile(nil, path, strict)
				}
				return nil
			})
		}
	}

	if sel&BackupSelectionImages != 0 {
		if !dirOK("data/images") {
			log.Warn("备份 images 失败: 不存在或不是目录")
		} else {
			cfgGlb.Images = true
			walkFiles("data/images", func(path string, info fs.DirEntry) error {
				if !info.IsDir() {
					backupFile(nil, path, strict)
				}
				return nil
			})
		}
	}

	withJS := sel&BackupSelectionJS != 0
	cfgDice.JSScripts = withJS

	for _, d := range dm.Dice {
		if d == nil {
			if strict {
				fatalErr = errors.Join(fatalErr, errors.New("备份列表包含空 Dice 实例"))
				break
			}
			log.Warn("跳过空 Dice 实例")
			continue
		}
		cfgGlb.Dices[d.BaseConfig.Name] = &cfgDice
		dataDir := d.BaseConfig.DataDir

		backupFile(d, filepath.Join(dataDir, "serve.yaml"), true)
		if fn := filepath.Join(dataDir, "advanced.yaml"); fileOK(fn) {
			backupFile(d, fn, true)
		}
		if fn := filepath.Join(dataDir, "configs", "plugin-configs.json"); fileOK(fn) {
			backupFile(d, fn, true)
		}

		backupFile(d, filepath.Join(dataDir, "configs/text-template.yaml"), false)

		walkFiles(filepath.Join(dataDir, "extensions/reply"), func(path string, info fs.DirEntry) error {
			// NOTE(Xiangze Li): copied from dice.ReplyReload. Should extract as function, but I'm lazy
			if info.IsDir() {
				if strings.EqualFold(info.Name(), "assets") || strings.EqualFold(info.Name(), "images") {
					return fs.SkipDir
				}
				return nil
			}
			if strings.HasPrefix(info.Name(), ".reply") || info.Name() == "info.yaml" {
				return nil
			}

			ext := filepath.Ext(path)
			if ext == ".yaml" || ext == "" {
				backupFile(d, path, strict)
			}
			return nil
		})

		if d.ImSession != nil {
			for _, i := range d.ImSession.EndPoints {
				if i == nil {
					continue
				}
				if i.Platform == "QQ" {
					if pa, ok := i.Adapter.(*PlatformAdapterGocq); ok && pa.UseInPackClient {
						workDir := i.RelWorkDir
						if pa.BuiltinMode == "lagrange" {
							backupFile(d, filepath.Join(dataDir, workDir, "appsettings.json"), false)
							backupFile(d, filepath.Join(dataDir, workDir, "device.json"), false)
							backupFile(d, filepath.Join(dataDir, workDir, "keystore.json"), false)
						} else {
							backupFile(d, filepath.Join(dataDir, workDir, "config.yml"), false)
							backupFile(d, filepath.Join(dataDir, workDir, "device.json"), false)
							backupFile(d, filepath.Join(dataDir, workDir, "session.token"), false)
						}
					}
				}
			}
		}

		if withJS {
			walkFiles(filepath.Join(dataDir, "scripts"), func(path string, info fs.DirEntry) error {
				if info.IsDir() {
					if info.Name() == "_builtin" {
						return filepath.SkipDir
					}
					return nil
				}
				if filepath.Ext(info.Name()) == ".js" {
					backupFile(d, path, strict)
				}
				return nil
			})
			extDataDir := filepath.Join(dataDir, "extensions")
			walkFiles(extDataDir, func(path string, info fs.DirEntry) error {
				if info.IsDir() {
					if filepath.Dir(path) == extDataDir {
						if ext := d.ExtFind(info.Name(), false); ext == nil || !ext.IsJsExt {
							return filepath.SkipDir
						}
					}
					return nil
				}
				backupFile(d, path, strict)
				return nil
			})
		}
	}

	databaseType := ""
	if dm.Operator != nil {
		databaseType = dm.Operator.Type()
	}
	if databaseType == "" {
		fatalErr = errors.Join(fatalErr, errors.New("数据库类型为空，无法创建可验证备份"))
	}
	if databaseType == constant.SQLITE {
		databaseFiles, pathErr := managedSQLiteDatabaseFiles(dm, strict)
		if pathErr != nil {
			fatalErr = errors.Join(fatalErr, pathErr)
			if !strict {
				log.Errorf("备份 SQLite 数据库失败: %v", pathErr)
			}
		} else {
			dbHandles := []struct {
				name string
				db   func() error
			}{
				{name: "data", db: func() error {
					database := dm.Operator.GetDataDB(constant.WRITE)
					if database == nil {
						return errors.New("data 数据库句柄为空")
					}
					return service.FlushWAL(database)
				}},
				{name: "logs", db: func() error {
					database := dm.Operator.GetLogDB(constant.WRITE)
					if database == nil {
						return errors.New("logs 数据库句柄为空")
					}
					return service.FlushWAL(database)
				}},
				{name: "censor", db: func() error {
					database := dm.Operator.GetCensorDB(constant.WRITE)
					if database == nil {
						return errors.New("censor 数据库句柄为空")
					}
					return service.FlushWAL(database)
				}},
			}
			for index, databaseFile := range databaseFiles {
				flushErr := dbHandles[index].db()
				if flushErr != nil {
					log.Errorf("备份时 %s 数据库 flush 出错: %v", dbHandles[index].name, flushErr)
					if strict || databaseFile.Name != "data-censor.db" {
						fatalErr = errors.Join(fatalErr, fmt.Errorf("flush %s 数据库: %w", dbHandles[index].name, flushErr))
					}
					continue
				}
				backupFile(nil, databaseFile.Path, strict || databaseFile.Name != "data-censor.db")
			}
		}
	} else if strict {
		fatalErr = errors.Join(fatalErr, fmt.Errorf("安全备份仅支持 SQLite，当前数据库类型为 %q", databaseType))
	}
	if fatalErr != nil {
		return "", fatalErr
	}
	sort.Slice(manifestFiles, func(i, j int) bool { return manifestFiles[i].Path < manifestFiles[j].Path })

	// 写入文件信息
	data, err := json.Marshal(backupManifest{
		FormatVersion: backupManifestFormatVersion,
		RestorePolicy: backupRestorePolicyOverlay,
		DatabaseType:  databaseType,
		Files:         manifestFiles,
		Config:        mustMarshalJSON(cfgGlb),
		Version:       mustMarshalJSON(VERSION.String()),
		VersionCode:   VERSION_CODE,
	})
	if err != nil {
		return "", err
	}

	h := &zip.FileHeader{Name: "backup_info.json", Method: zip.Deflate, Flags: 0x800}
	h.SetMode(0o600)
	fileWriter, err := writer.CreateHeader(h)
	if err != nil {
		return "", err
	}
	if _, err = fileWriter.Write(data); err != nil {
		return "", err
	}
	if err = writer.Close(); err != nil {
		return "", err
	}
	if err = fzip.Sync(); err != nil {
		return "", err
	}
	if err = fzip.Close(); err != nil {
		return "", err
	}
	if _, statErr := os.Lstat(target); statErr == nil {
		return "", fmt.Errorf("备份文件已存在: %s", bakFn)
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return "", statErr
	}
	if err = renameRestorePath(tmpName, target); err != nil {
		return "", err
	}
	if err = syncDirectory(BackupDir); err != nil {
		return "", err
	}
	published = true
	return target, nil
}

func (dm *DiceManager) BackupAuto() error {
	_, err := dm.Backup(dm.AutoBackupSelection, true)
	return err
}

func (dm *DiceManager) BackupClean(fromAuto bool) (err error) {
	if dm.BackupCleanStrategy == BackupCleanStrategyDisabled {
		return nil
	}

	if fromAuto && (dm.BackupCleanTrigger&BackupCleanTriggerRotate == 0) {
		return nil
	}
	backupOperationMu.Lock()
	defer backupOperationMu.Unlock()
	if err = validateBackupDirectoryLocked(); err != nil {
		return err
	}

	log := logger.M()
	log.Info("开始清理备份文件")

	backupDir, err := os.Open(BackupDir)
	if err != nil {
		return err
	}
	defer func() { _ = backupDir.Close() }()
	if i, _ := backupDir.Stat(); !i.IsDir() {
		return fmt.Errorf("backup directory %q is not a directory", BackupDir)
	}

	files, err := backupDir.ReadDir(-1)
	if err != nil {
		return err
	}

	fileInfos := make([]os.FileInfo, 0, len(files))
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if strings.HasPrefix(f.Name(), ".bak-") || strings.HasSuffix(f.Name(), ".part") {
			continue
		}
		if !strings.EqualFold(filepath.Ext(f.Name()), ".zip") {
			continue
		}
		if fi, err := f.Info(); err == nil && fi.Mode().IsRegular() {
			fileInfos = append(fileInfos, fi)
		}
	}

	sort.Sort(utils.ByModtime(fileInfos))

	var fileInfoOld []os.FileInfo

	logMsg := strings.Builder{}
	_, _ = fmt.Fprintf(&logMsg, "现有备份文件 %d 个, 清理模式为 ", len(fileInfos)) //nolint:gosec

	switch dm.BackupCleanStrategy {
	case BackupCleanStrategyByCount:
		_, _ = fmt.Fprintf(&logMsg, "保留一定数量(%d)", dm.BackupCleanKeepCount)
		if len(fileInfos) > dm.BackupCleanKeepCount {
			fileInfoOld = fileInfos[:len(fileInfos)-dm.BackupCleanKeepCount]
		}
	case BackupCleanStrategyByTime:
		threshold := time.Now().Add(-dm.BackupCleanKeepDur)
		_, _ = fmt.Fprintf(&logMsg, "保留一定时间(%v, %s)", dm.BackupCleanKeepDur, threshold.Format(time.DateTime))
		idx, _ := sort.Find(len(fileInfos), func(i int) int {
			return threshold.Compare(fileInfos[i].ModTime())
		})
		fileInfoOld = fileInfos[:idx]
	default:
		// no-op
	}

	_, _ = fmt.Fprintf(&logMsg, ", 有以下 %d 个将要被删除", len(fileInfoOld)) //nolint:gosec

	errDel := []string{}
	deletedCount := 0
	skippedCount := 0
	for i, fi := range fileInfoOld {
		_, _ = fmt.Fprintf(&logMsg, "\n%d. %s", i+1, fi.Name()) //nolint:gosec
		inUse, useErr := backupInUseLocked(fi.Name())
		if useErr != nil {
			skippedCount++
			_, _ = fmt.Fprintf(&logMsg, "（无法读取恢复事务元数据，已跳过）") //nolint:gosec
			log.Warnf("清理备份时无法读取恢复事务元数据: %v", useErr)
			continue
		}
		if inUse {
			skippedCount++
			_, _ = fmt.Fprintf(&logMsg, "（恢复事务正在使用，已跳过）") //nolint:gosec
			continue
		}
		errDelete := os.Remove(filepath.Join(BackupDir, fi.Name())) //nolint:gosec
		if errDelete != nil {
			errDel = append(errDel, errDelete.Error())
		} else {
			deletedCount++
		}
	}
	_, _ = fmt.Fprintf(&logMsg, "\n实际删除 %d 个，跳过 %d 个", deletedCount, skippedCount) //nolint:gosec

	log.Info(logMsg.String())

	if len(errDel) > 0 {
		return errors.New("error(s) occured when deleting files:\n" + strings.Join(errDel, "\n"))
	}
	return nil
}
