package dice

import (
	"archive/zip"
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
	"time"

	"sealdice-core/logger"
	"sealdice-core/utils/constant"
)

const (
	restoreDirName           = ".restore"
	restorePendingName       = "pending.json"
	restoreStatusName        = "status.json"
	restoreSourceName        = "source.zip"
	restoreStagingName       = "staging"
	restoreRollbackName      = "rollback"
	restoreJournalName       = "journal.json"
	restoreStatusAuthName    = "status-auth.json"
	restoreLegacyOldDataName = "old-data"
	maxBackupEntries         = 100000
	minimumDiskReserve       = uint64(1 << 30)
)

type BackupArchiveInfo struct {
	Name             string `json:"name"`
	FileSize         int64  `json:"fileSize"`
	Selection        int64  `json:"selection"`
	Version          string `json:"version"`
	VersionCode      int64  `json:"versionCode"`
	Valid            bool   `json:"valid"`
	Restorable       bool   `json:"restorable"`
	Error            string `json:"error,omitempty"`
	Reused           bool   `json:"reused,omitempty"`
	UncompressedSize uint64 `json:"-"`
}

type RestoreStatus struct {
	State            string `json:"state"`
	OperationID      string `json:"operationId,omitempty"`
	SourceName       string `json:"sourceName,omitempty"`
	SafetyBackupName string `json:"safetyBackupName,omitempty"`
	Message          string `json:"message,omitempty"`
	UpdatedAt        int64  `json:"updatedAt"`
}

type restorePending struct {
	Phase            string `json:"phase"`
	OperationID      string `json:"operationId,omitempty"`
	StatusTokenHash  string `json:"statusTokenHash,omitempty"`
	SourceName       string `json:"sourceName"`
	SafetyBackupName string `json:"safetyBackupName"`
}

type RestoreOperation struct {
	OperationID      string `json:"operationId"`
	StatusToken      string `json:"statusToken"`
	SafetyBackupName string `json:"safetyBackupName"`
}

type restoreJournalEntry struct {
	Target      string `json:"target"`
	Rollback    string `json:"rollback"`
	Staged      string `json:"staged,omitempty"`
	HadOriginal bool   `json:"hadOriginal"`
	State       string `json:"state"`
}

type restoreJournal struct {
	OperationID string                `json:"operationId"`
	State       string                `json:"state"`
	Entries     []restoreJournalEntry `json:"entries"`
}

type restoreStatusAuth struct {
	OperationID string `json:"operationId"`
	TokenHash   string `json:"tokenHash"`
	ExpiresAt   int64  `json:"expiresAt"`
}

var restoreRename = os.Rename

type backupManifest struct {
	Config      json.RawMessage `json:"config"`
	Version     json.RawMessage `json:"version"`
	VersionCode int64           `json:"versionCode"`
}

func restoreDir() string { return filepath.Join(BackupDir, restoreDirName) }

func restorePendingPath() string { return filepath.Join(restoreDir(), restorePendingName) }

func restoreStatusPath() string { return filepath.Join(restoreDir(), restoreStatusName) }

func restoreJournalPath() string { return filepath.Join(restoreDir(), restoreJournalName) }

func restoreStatusAuthPath() string { return filepath.Join(restoreDir(), restoreStatusAuthName) }

func HasPendingRestore() bool {
	_, err := os.Stat(restorePendingPath())
	return err == nil
}

func diskReserve(total uint64) uint64 {
	reserve := total / 20
	if reserve < minimumDiskReserve {
		return minimumDiskReserve
	}
	return reserve
}

func ensureDiskSpace(path string, required uint64) error {
	free, total, err := diskSpace(path)
	if err != nil {
		return fmt.Errorf("检查磁盘空间失败: %w", err)
	}
	reserve := diskReserve(total)
	if free < required || free-required < reserve {
		return fmt.Errorf("磁盘空间不足：需要 %d 字节并保留 %d 字节安全余量，当前可用 %d 字节", required, reserve, free)
	}
	return nil
}

func parseManifestVersion(raw json.RawMessage) string {
	var version string
	if json.Unmarshal(raw, &version) == nil {
		return version
	}
	return "旧版"
}

func selectionFromManifest(raw json.RawMessage) int64 {
	var cfg struct {
		Decks   bool `json:"decks"`
		HelpDoc bool `json:"helpDoc"`
		Censor  bool `json:"censor"`
		Names   bool `json:"names"`
		Images  bool `json:"images"`
		Dices   map[string]struct {
			JSScripts bool `json:"jsScripts"`
		} `json:"dices"`
	}
	if json.Unmarshal(raw, &cfg) != nil {
		return 0
	}
	var sel BackupSelection
	if cfg.Decks {
		sel |= BackupSelectionDecks
	}
	if cfg.HelpDoc {
		sel |= BackupSelectionHelpDoc
	}
	if cfg.Censor {
		sel |= BackupSelectionCensor
	}
	if cfg.Names {
		sel |= BackupSelectionNames
	}
	if cfg.Images {
		sel |= BackupSelectionImages
	}
	for _, diceCfg := range cfg.Dices {
		if diceCfg.JSScripts {
			sel |= BackupSelectionJS
			break
		}
	}
	return int64(sel)
}

func normalizeBackupEntryName(name string, isDir bool) (string, error) {
	if name == "" {
		return "", errors.New("压缩包包含空路径")
	}
	normalized := strings.ReplaceAll(name, "\\", "/")
	if path.IsAbs(normalized) || strings.HasPrefix(normalized, "//") || filepath.VolumeName(name) != "" ||
		(len(normalized) >= 2 && normalized[1] == ':') {
		return "", fmt.Errorf("压缩包包含绝对路径 %q", name)
	}
	trimmed := strings.TrimSuffix(normalized, "/")
	clean := path.Clean(trimmed)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean != trimmed {
		return "", fmt.Errorf("压缩包包含非法路径 %q", name)
	}
	if strings.Contains(strings.SplitN(clean, "/", 2)[0], ":") {
		return "", fmt.Errorf("压缩包包含非法卷路径 %q", name)
	}
	if isDir && normalized != trimmed+"/" && normalized != trimmed {
		return "", fmt.Errorf("压缩包包含非法目录路径 %q", name)
	}
	return clean, nil
}

func inspectBackupArchive(filename string) (*BackupArchiveInfo, map[string]struct{}, error) {
	stat, err := os.Stat(filename)
	if err != nil {
		return nil, nil, err
	}
	reader, err := zip.OpenReader(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("不是有效的 ZIP 文件: %w", err)
	}
	defer func() { _ = reader.Close() }()
	if len(reader.File) > maxBackupEntries {
		return nil, nil, fmt.Errorf("压缩包条目超过 %d 个", maxBackupEntries)
	}

	entries := make(map[string]struct{}, len(reader.File))
	entriesFolded := make(map[string]struct{}, len(reader.File))
	var manifest backupManifest
	manifestFound := false
	dataConfigFound := false
	var uncompressed uint64
	for _, item := range reader.File {
		name := item.Name
		clean, cleanErr := normalizeBackupEntryName(name, item.FileInfo().IsDir())
		if cleanErr != nil {
			return nil, nil, cleanErr
		}
		if item.Mode()&os.ModeSymlink != 0 || (!item.FileInfo().Mode().IsRegular() && !item.FileInfo().IsDir()) {
			return nil, nil, fmt.Errorf("压缩包包含不支持的文件类型 %q", name)
		}
		if clean != "backup_info.json" && clean != "data" && !strings.HasPrefix(clean, "data/") {
			return nil, nil, fmt.Errorf("压缩包包含 data 目录外的文件 %q", name)
		}
		if _, exists := entries[clean]; exists {
			return nil, nil, fmt.Errorf("压缩包包含重复路径 %q", name)
		}
		folded := strings.ToLower(clean)
		if _, exists := entriesFolded[folded]; exists {
			return nil, nil, fmt.Errorf("压缩包包含大小写冲突路径 %q", name)
		}
		entries[clean] = struct{}{}
		entriesFolded[folded] = struct{}{}
		if ^uint64(0)-uncompressed < item.UncompressedSize64 {
			return nil, nil, errors.New("压缩包解压大小溢出")
		}
		uncompressed += item.UncompressedSize64
		if clean == "data/dice.yaml" {
			dataConfigFound = true
		}
		if clean == "backup_info.json" {
			if item.UncompressedSize64 > 1<<20 {
				return nil, nil, errors.New("backup_info.json 过大")
			}
			rc, openErr := item.Open()
			if openErr != nil {
				return nil, nil, fmt.Errorf("读取 backup_info.json 失败: %w", openErr)
			}
			decodeErr := json.NewDecoder(io.LimitReader(rc, 1<<20)).Decode(&manifest)
			closeErr := rc.Close()
			if decodeErr != nil {
				return nil, nil, fmt.Errorf("解析 backup_info.json 失败: %w", decodeErr)
			}
			if closeErr != nil {
				return nil, nil, fmt.Errorf("校验 backup_info.json 失败: %w", closeErr)
			}
			manifestFound = true
		}
	}
	if !manifestFound || manifest.VersionCode <= 0 {
		return nil, nil, errors.New("缺少有效的 backup_info.json")
	}
	if !dataConfigFound {
		return nil, nil, errors.New("备份缺少 data/dice.yaml")
	}
	if manifest.VersionCode > VERSION_CODE {
		return nil, nil, fmt.Errorf("备份版本 %d 高于当前程序版本 %d", manifest.VersionCode, VERSION_CODE)
	}
	return &BackupArchiveInfo{
		Name:             filepath.Base(filename),
		FileSize:         stat.Size(),
		Selection:        selectionFromManifest(manifest.Config),
		Version:          parseManifestVersion(manifest.Version),
		VersionCode:      manifest.VersionCode,
		Valid:            true,
		Restorable:       true,
		UncompressedSize: uncompressed,
	}, entries, nil
}

func InspectBackupArchive(filename string) *BackupArchiveInfo {
	info, _, err := inspectBackupArchive(filename)
	if err == nil {
		return info
	}
	stat, statErr := os.Stat(filename)
	item := &BackupArchiveInfo{Name: filepath.Base(filename), Selection: -1, Error: err.Error()}
	if statErr == nil {
		item.FileSize = stat.Size()
	}
	return item
}

func validateBackupData(filename string) error {
	reader, err := zip.OpenReader(filename)
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()
	for _, item := range reader.File {
		if item.FileInfo().IsDir() {
			continue
		}
		rc, openErr := item.Open()
		if openErr != nil {
			return openErr
		}
		written, copyErr := io.Copy(io.Discard, rc)
		closeErr := rc.Close()
		if copyErr != nil || closeErr != nil || uint64(written) != item.UncompressedSize64 {
			return fmt.Errorf("校验 %s 失败", item.Name)
		}
	}
	return nil
}

func copyFile(source, target string) error {
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()
	dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err = io.Copy(dst, src); err != nil {
		_ = dst.Close()
		return err
	}
	return dst.Close()
}

func writeJSONAtomic(filename string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(filename), 0o700); err != nil {
		return err
	}
	tmp := filename + ".tmp"
	if err = os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return replaceFileAtomic(tmp, filename)
}

func setRestoreStatus(status RestoreStatus) error {
	status.UpdatedAt = time.Now().Unix()
	return writeJSONAtomic(restoreStatusPath(), status)
}

func GetRestoreStatus() RestoreStatus {
	data, err := os.ReadFile(restoreStatusPath())
	if err != nil {
		return RestoreStatus{State: "idle"}
	}
	var status RestoreStatus
	if json.Unmarshal(data, &status) != nil {
		return RestoreStatus{State: "failed", Message: "恢复状态文件损坏"}
	}
	return status
}

func UpdateRestoreStatusState(state, message string) error {
	pending, err := readPendingRestore()
	if err != nil {
		return err
	}
	return setRestoreStatus(RestoreStatus{
		State:            state,
		OperationID:      pending.OperationID,
		SourceName:       pending.SourceName,
		SafetyBackupName: pending.SafetyBackupName,
		Message:          message,
	})
}

func BackupInUse(name string) bool {
	pending, err := readPendingRestore()
	if err != nil {
		return false
	}
	return name == pending.SourceName || name == pending.SafetyBackupName
}

func fileSHA256(filename string) ([sha256.Size]byte, error) {
	file, err := os.Open(filename)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err = io.Copy(hasher, file); err != nil {
		return [sha256.Size]byte{}, err
	}
	var digest [sha256.Size]byte
	copy(digest[:], hasher.Sum(nil))
	return digest, nil
}

func ImportBackup(reader io.Reader) (*BackupArchiveInfo, error) {
	backupOperationMu.Lock()
	defer backupOperationMu.Unlock()
	if err := os.MkdirAll(BackupDir, 0o755); err != nil {
		return nil, err
	}
	tmp, err := os.CreateTemp(BackupDir, ".upload-*.part")
	if err != nil {
		return nil, err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	hash := sha256.New()
	buffer := make([]byte, 4<<20)
	for {
		if err = ensureDiskSpace(BackupDir, uint64(len(buffer))); err != nil {
			_ = tmp.Close()
			return nil, err
		}
		var count int
		count, err = reader.Read(buffer)
		if count > 0 {
			if _, writeErr := tmp.Write(buffer[:count]); writeErr != nil {
				_ = tmp.Close()
				return nil, writeErr
			}
			_, _ = hash.Write(buffer[:count])
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			_ = tmp.Close()
			return nil, err
		}
	}
	if err = tmp.Close(); err != nil {
		return nil, err
	}
	info, _, err := inspectBackupArchive(tmpName)
	if err != nil {
		return nil, err
	}
	if err = validateBackupData(tmpName); err != nil {
		return nil, fmt.Errorf("备份内容校验失败: %w", err)
	}
	var uploadedDigest [sha256.Size]byte
	copy(uploadedDigest[:], hash.Sum(nil))
	digest := hex.EncodeToString(uploadedDigest[:])[:8]
	existing, _ := filepath.Glob(filepath.Join(BackupDir, "import_*_"+digest+".zip"))
	for _, candidate := range existing {
		candidateDigest, hashErr := fileSHA256(candidate)
		if hashErr != nil || candidateDigest != uploadedDigest {
			continue
		}
		item := InspectBackupArchive(candidate)
		item.Reused = true
		return item, nil
	}
	name := fmt.Sprintf("import_%s_%s.zip", time.Now().Format("060102_150405"), digest)
	target := filepath.Join(BackupDir, name)
	if err = os.Rename(tmpName, target); err != nil {
		return nil, err
	}
	info.Name = name
	return info, nil
}

func isSQLiteManager(dm *DiceManager) bool {
	for _, d := range dm.Dice {
		if d.DBOperator == nil || !strings.Contains(strings.ToLower(d.DBOperator.GetDataDB(constant.WRITE).Name()), "sqlite") {
			return false
		}
	}
	return true
}

func validateSafetyBackup(filename string, dm *DiceManager) error {
	_, entries, err := inspectBackupArchive(filename)
	if err != nil {
		return err
	}
	if err = validateBackupData(filename); err != nil {
		return err
	}
	for _, d := range dm.Dice {
		base := path.Join("data", d.BaseConfig.Name)
		for _, required := range []string{"serve.yaml", "data.db", "data-logs.db"} {
			if _, ok := entries[path.Join(base, required)]; !ok {
				return fmt.Errorf("安全备份缺少 %s", path.Join(base, required))
			}
		}
	}
	return nil
}

func releaseRetryablePendingRestore() error {
	pending, err := readPendingRestore()
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("读取已有恢复任务: %w", err)
	}
	status := GetRestoreStatus()
	terminal := status.State == "failed" || status.State == "rolled_back" || status.State == "degraded"
	if pending.Phase != "pending" || !terminal {
		return errors.New("已有恢复任务正在等待执行")
	}
	legacyOldData := filepath.Join(restoreDir(), restoreLegacyOldDataName)
	if _, err = os.Stat(legacyOldData); err == nil {
		return fmt.Errorf("已有恢复任务保留了旧数据目录 %s，拒绝自动清除", legacyOldData)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("检查旧版恢复遗留目录: %w", err)
	}
	journal, journalErr := readRestoreJournal()
	if journalErr == nil && journal.State != "rolled_back" && journal.State != "committed" {
		return fmt.Errorf("已有恢复事务尚未安全结束，当前状态为 %s", journal.State)
	}
	if journalErr != nil && !errors.Is(journalErr, os.ErrNotExist) {
		return fmt.Errorf("检查已有恢复事务: %w", journalErr)
	}
	if err = os.Remove(restorePendingPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("释放已结束的恢复任务: %w", err)
	}
	return nil
}

func (dm *DiceManager) ScheduleRestore(name string) (*RestoreOperation, error) {
	backupOperationMu.Lock()
	defer backupOperationMu.Unlock()
	if !isSQLiteManager(dm) {
		return nil, errors.New("备份恢复仅支持 SQLite，当前外部数据库不会被 ZIP 备份覆盖")
	}
	if filepath.Base(name) != name || strings.ToLower(filepath.Ext(name)) != ".zip" {
		return nil, errors.New("备份文件名非法")
	}
	if err := releaseRetryablePendingRestore(); err != nil {
		return nil, err
	}
	source := filepath.Join(BackupDir, name)
	info, _, err := inspectBackupArchive(source)
	if err != nil {
		return nil, err
	}
	dataSize, err := directorySize("data")
	if err != nil {
		return nil, err
	}
	if err = ensureDiskSpace(BackupDir, dataSize+info.UncompressedSize); err != nil {
		return nil, err
	}
	logger.M().Infow("[备份恢复] 正在创建恢复前安全备份", "source", name)
	safetyPath, err := dm.backup(BackupSelectionAll, false)
	if err != nil {
		return nil, fmt.Errorf("创建恢复前安全备份失败: %w", err)
	}
	if err = validateSafetyBackup(safetyPath, dm); err != nil {
		return nil, fmt.Errorf("恢复前安全备份不完整: %w", err)
	}
	logger.M().Infow("[备份恢复] 恢复前安全备份已创建并校验", "source", name, "safetyBackup", filepath.Base(safetyPath))
	if err = os.RemoveAll(restoreDir()); err != nil {
		return nil, err
	}
	if err = os.MkdirAll(restoreDir(), 0o700); err != nil {
		return nil, err
	}
	if err = copyFile(source, filepath.Join(restoreDir(), restoreSourceName)); err != nil {
		return nil, err
	}
	operationID := RandStringBytesMaskImprSrcSB2(16)
	statusToken := RandStringBytesMaskImprSrcSB2(48)
	tokenDigest := sha256.Sum256([]byte(statusToken))
	pending := restorePending{
		Phase:            "pending",
		OperationID:      operationID,
		StatusTokenHash:  hex.EncodeToString(tokenDigest[:]),
		SourceName:       name,
		SafetyBackupName: filepath.Base(safetyPath),
	}
	if err = writeJSONAtomic(restorePendingPath(), pending); err != nil {
		return nil, err
	}
	if err = writeJSONAtomic(restoreStatusAuthPath(), restoreStatusAuth{
		OperationID: operationID,
		TokenHash:   pending.StatusTokenHash,
		ExpiresAt:   time.Now().Add(15 * time.Minute).Unix(),
	}); err != nil {
		_ = os.Remove(restorePendingPath())
		return nil, err
	}
	_ = setRestoreStatus(RestoreStatus{State: "pending", OperationID: operationID, SourceName: name, SafetyBackupName: pending.SafetyBackupName})
	return &RestoreOperation{OperationID: operationID, StatusToken: statusToken, SafetyBackupName: pending.SafetyBackupName}, nil
}

func ValidateRestoreStatusToken(operationID, token string) bool {
	if operationID == "" || token == "" {
		return false
	}
	var expectedHash string
	pending, err := readPendingRestore()
	if err == nil && pending.OperationID == operationID {
		expectedHash = pending.StatusTokenHash
	} else {
		data, readErr := os.ReadFile(restoreStatusAuthPath())
		if readErr != nil {
			return false
		}
		var auth restoreStatusAuth
		if json.Unmarshal(data, &auth) != nil || auth.OperationID != operationID || time.Now().Unix() > auth.ExpiresAt {
			return false
		}
		expectedHash = auth.TokenHash
	}
	digest := sha256.Sum256([]byte(token))
	return strings.EqualFold(expectedHash, hex.EncodeToString(digest[:]))
}

func directorySize(root string) (uint64, error) {
	var total uint64
	err := filepath.WalkDir(root, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type().IsRegular() {
			info, infoErr := entry.Info()
			if infoErr != nil {
				return infoErr
			}
			total += uint64(info.Size())
		}
		return nil
	})
	return total, err
}

func readPendingRestore() (*restorePending, error) {
	data, err := os.ReadFile(restorePendingPath())
	if err != nil {
		return nil, err
	}
	var pending restorePending
	if err = json.Unmarshal(data, &pending); err != nil {
		return nil, err
	}
	return &pending, nil
}

func extractBackupTo(source, destination string) error {
	reader, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()
	for _, item := range reader.File {
		clean, cleanErr := normalizeBackupEntryName(item.Name, item.FileInfo().IsDir())
		if cleanErr != nil {
			return cleanErr
		}
		if clean == "backup_info.json" || clean == "data" {
			continue
		}
		target := filepath.Join(destination, filepath.FromSlash(clean))
		if item.FileInfo().IsDir() {
			if err = os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err = os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, openErr := item.Open()
		if openErr != nil {
			return openErr
		}
		mode := item.Mode().Perm()
		if mode == 0 {
			mode = 0o600
		}
		dst, createErr := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
		if createErr != nil {
			_ = rc.Close()
			return createErr
		}
		written, copyErr := io.Copy(dst, rc)
		closeDstErr := dst.Close()
		closeSrcErr := rc.Close()
		if copyErr != nil || closeDstErr != nil || closeSrcErr != nil || uint64(written) != item.UncompressedSize64 {
			return fmt.Errorf("解压 %s 失败或大小不匹配", item.Name)
		}
	}
	return nil
}

func readRestoreJournal() (*restoreJournal, error) {
	data, err := os.ReadFile(restoreJournalPath())
	if err != nil {
		return nil, err
	}
	var journal restoreJournal
	if err = json.Unmarshal(data, &journal); err != nil {
		return nil, err
	}
	return &journal, nil
}

func writeRestoreJournal(journal *restoreJournal) error {
	return writeJSONAtomic(restoreJournalPath(), journal)
}

func rollbackRestoreJournal(journal *restoreJournal) error {
	journal.State = "rolling_back"
	_ = writeRestoreJournal(journal)
	var rollbackErr error
	for index := len(journal.Entries) - 1; index >= 0; index-- {
		entry := &journal.Entries[index]
		if entry.State == "installing" || entry.State == "applied" {
			if err := os.Remove(entry.Target); err != nil && !errors.Is(err, os.ErrNotExist) {
				rollbackErr = errors.Join(rollbackErr, fmt.Errorf("移除已恢复文件 %s: %w", entry.Target, err))
				continue
			}
		}
		if entry.HadOriginal && entry.State != "planned" {
			if _, err := os.Stat(entry.Rollback); errors.Is(err, os.ErrNotExist) {
				// backing_up 可能在 rename 之前中断，此时原文件仍在目标位置。
				entry.State = "rolled_back"
				_ = writeRestoreJournal(journal)
				continue
			} else if err != nil {
				rollbackErr = errors.Join(rollbackErr, fmt.Errorf("检查回滚文件 %s: %w", entry.Rollback, err))
				continue
			}
			if err := os.MkdirAll(filepath.Dir(entry.Target), 0o755); err != nil {
				rollbackErr = errors.Join(rollbackErr, err)
				continue
			}
			if err := restoreRename(entry.Rollback, entry.Target); err != nil {
				rollbackErr = errors.Join(rollbackErr, fmt.Errorf("恢复原文件 %s: %w", entry.Target, err))
				continue
			}
		}
		entry.State = "rolled_back"
		_ = writeRestoreJournal(journal)
	}
	if rollbackErr != nil {
		journal.State = "rollback_failed"
		_ = writeRestoreJournal(journal)
		return rollbackErr
	}
	journal.State = "rolled_back"
	return writeRestoreJournal(journal)
}

func buildRestoreJournal(operationID, staging string) (*restoreJournal, error) {
	journal := &restoreJournal{OperationID: operationID, State: "applying"}
	err := filepath.WalkDir(filepath.Join(staging, "data"), func(stagedPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(staging, stagedPath)
		if err != nil {
			return err
		}
		target := filepath.Clean(relative)
		rollback := filepath.Join(restoreDir(), restoreRollbackName, relative)
		_, statErr := os.Stat(target)
		journal.Entries = append(journal.Entries, restoreJournalEntry{
			Target:      target,
			Rollback:    rollback,
			Staged:      stagedPath,
			HadOriginal: statErr == nil,
			State:       "planned",
		})
		if strings.HasSuffix(strings.ToLower(target), ".db") {
			for _, suffix := range []string{"-wal", "-shm"} {
				sidecar := target + suffix
				if _, err = os.Stat(sidecar); err == nil {
					journal.Entries = append(journal.Entries, restoreJournalEntry{
						Target:      sidecar,
						Rollback:    filepath.Join(restoreDir(), restoreRollbackName, sidecar),
						HadOriginal: true,
						State:       "planned",
					})
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(journal.Entries, func(i, j int) bool { return journal.Entries[i].Target < journal.Entries[j].Target })
	return journal, nil
}

func applyRestoreJournal(journal *restoreJournal) (resultErr error) {
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, rollbackRestoreJournal(journal))
		}
	}()
	for index := range journal.Entries {
		entry := &journal.Entries[index]
		if entry.HadOriginal {
			if err := os.MkdirAll(filepath.Dir(entry.Rollback), 0o700); err != nil {
				return err
			}
			entry.State = "backing_up"
			if err := writeRestoreJournal(journal); err != nil {
				return err
			}
			if err := restoreRename(entry.Target, entry.Rollback); err != nil {
				return fmt.Errorf("暂存原文件 %s: %w", entry.Target, err)
			}
		}
		entry.State = "backed_up"
		if err := writeRestoreJournal(journal); err != nil {
			return err
		}
		if entry.Staged != "" {
			if err := os.MkdirAll(filepath.Dir(entry.Target), 0o755); err != nil {
				return err
			}
			entry.State = "installing"
			if err := writeRestoreJournal(journal); err != nil {
				return err
			}
			if err := restoreRename(entry.Staged, entry.Target); err != nil {
				return fmt.Errorf("应用恢复文件 %s: %w", entry.Target, err)
			}
		}
		entry.State = "applied"
		if err := writeRestoreJournal(journal); err != nil {
			return err
		}
	}
	journal.State = "applied"
	return writeRestoreJournal(journal)
}

// RecoverInterruptedRestore 在 Runtime 启动前回滚上次未提交的逐文件事务。
func RecoverInterruptedRestore() error {
	journal, err := readRestoreJournal()
	if errors.Is(err, os.ErrNotExist) {
		// 兼容旧实现留下的 switching 状态。若 old-data 不存在，原数据没有被移动。
		pending, pendingErr := readPendingRestore()
		if pendingErr == nil && (pending.Phase == "switching" || pending.Phase == "swapped") {
			legacyOldData := filepath.Join(restoreDir(), restoreLegacyOldDataName)
			if _, oldDataErr := os.Stat(legacyOldData); oldDataErr == nil {
				return fmt.Errorf("检测到旧版恢复遗留目录 %s；为避免覆盖仍被日志占用的数据，已停止自动恢复，请保留现场并使用安全备份 %s", legacyOldData, pending.SafetyBackupName)
			} else if !errors.Is(oldDataErr, os.ErrNotExist) {
				return fmt.Errorf("检查旧版恢复遗留目录: %w", oldDataErr)
			}
			pending.Phase = "pending"
			if writeErr := writeJSONAtomic(restorePendingPath(), pending); writeErr != nil {
				return writeErr
			}
			return setRestoreStatus(RestoreStatus{State: "failed", SourceName: pending.SourceName, SafetyBackupName: pending.SafetyBackupName, Message: "检测到旧版未完成恢复，已保留原数据；请重新发起恢复"})
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("读取恢复事务日志: %w", err)
	}
	if journal.State == "committed" || journal.State == "rolled_back" {
		return nil
	}
	return rollbackRestoreJournal(journal)
}

// ApplyScheduledRestore 在 Runtime 已完全停止后应用待恢复文件。
func ApplyScheduledRestore() (resultErr error) {
	pending, err := readPendingRestore()
	if errors.Is(err, os.ErrNotExist) {
		return errors.New("没有待执行的恢复任务")
	}
	if err != nil {
		_ = setRestoreStatus(RestoreStatus{State: "failed", Message: "读取恢复任务失败: " + err.Error()})
		return err
	}
	defer func() {
		if resultErr != nil {
			_ = setRestoreStatus(RestoreStatus{State: "failed", SourceName: pending.SourceName, SafetyBackupName: pending.SafetyBackupName, Message: resultErr.Error()})
		}
	}()
	if err = RecoverInterruptedRestore(); err != nil {
		return err
	}
	source := filepath.Join(restoreDir(), restoreSourceName)
	info, _, err := inspectBackupArchive(source)
	if err != nil {
		_ = setRestoreStatus(RestoreStatus{State: "failed", SourceName: pending.SourceName, SafetyBackupName: pending.SafetyBackupName, Message: err.Error()})
		_ = os.Remove(restorePendingPath())
		return err
	}
	if err = ensureDiskSpace(restoreDir(), info.UncompressedSize); err != nil {
		return err
	}
	_ = setRestoreStatus(RestoreStatus{State: "applying", OperationID: pending.OperationID, SourceName: pending.SourceName, SafetyBackupName: pending.SafetyBackupName})
	staging := filepath.Join(restoreDir(), restoreStagingName)
	_ = os.RemoveAll(staging)
	_ = os.RemoveAll(filepath.Join(restoreDir(), restoreRollbackName))
	_ = os.Remove(restoreJournalPath())
	if err = os.MkdirAll(staging, 0o700); err != nil {
		return err
	}
	if err = extractBackupTo(source, staging); err != nil {
		return err
	}
	journal, err := buildRestoreJournal(pending.OperationID, staging)
	if err != nil {
		return err
	}
	if err = writeRestoreJournal(journal); err != nil {
		return err
	}
	pending.Phase = "applying"
	if err = writeJSONAtomic(restorePendingPath(), pending); err != nil {
		return err
	}
	if err = applyRestoreJournal(journal); err != nil {
		return err
	}
	pending.Phase = "applied"
	if err = writeJSONAtomic(restorePendingPath(), pending); err != nil {
		return err
	}
	return nil
}

func CommitScheduledRestore() error {
	pending, err := readPendingRestore()
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil || pending.Phase != "applied" {
		return err
	}
	journal, err := readRestoreJournal()
	if err != nil {
		return err
	}
	journal.State = "committed"
	if err = writeRestoreJournal(journal); err != nil {
		return err
	}
	if err = os.RemoveAll(filepath.Join(restoreDir(), restoreRollbackName)); err != nil {
		return err
	}
	_ = os.RemoveAll(filepath.Join(restoreDir(), restoreStagingName))
	_ = os.Remove(filepath.Join(restoreDir(), restoreSourceName))
	_ = os.Remove(restorePendingPath())
	_ = os.Remove(restoreJournalPath())
	return setRestoreStatus(RestoreStatus{State: "succeeded", OperationID: pending.OperationID, SourceName: pending.SourceName, SafetyBackupName: pending.SafetyBackupName})
}

func RollbackScheduledRestore(message string) error {
	pending, pendingErr := readPendingRestore()
	journal, journalErr := readRestoreJournal()
	if journalErr == nil && journal.State != "rolled_back" {
		if err := rollbackRestoreJournal(journal); err != nil {
			return err
		}
	}
	if pendingErr == nil {
		pending.Phase = "pending"
		if err := writeJSONAtomic(restorePendingPath(), pending); err != nil {
			return err
		}
		return setRestoreStatus(RestoreStatus{State: "rolled_back", OperationID: pending.OperationID, SourceName: pending.SourceName, SafetyBackupName: pending.SafetyBackupName, Message: message})
	}
	return pendingErr
}
