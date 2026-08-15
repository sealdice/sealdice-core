package dice

import (
	"archive/zip"
	"bytes"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"gopkg.in/yaml.v3"

	"sealdice-core/logger"
	"sealdice-core/utils/constant"
)

const (
	restoreDirName                  = ".restore"
	restorePendingName              = "pending.json"
	restoreStatusName               = "status.json"
	restoreSourceName               = "source.zip"
	restoreStagingName              = "staging"
	restoreRollbackName             = "rollback"
	restoreJournalName              = "journal.jsonl"
	restoreLegacyJournalName        = "journal.json"
	restoreStatusAuthName           = "status-auth.json"
	restoreLegacyOldDataName        = "old-data"
	backupManifestFormatVersion     = 2
	backupRestorePolicyOverlay      = "overlay"
	maxBackupEntries                = 100000
	maxBackupCentralDirectorySize   = uint64(256 << 20)
	maxBackupEntryPathBytes         = 4096
	maxImportedBackupSize           = uint64(8 << 30)
	maxBackupEntryUncompressedSize  = uint64(16 << 30)
	maxBackupTotalUncompressedSize  = uint64(64 << 30)
	maxBackupCompressionRatio       = uint64(1000)
	compressionRatioMinimumSize     = uint64(64 << 20)
	maxRestoreJournalSize           = int64(256 << 20)
	maxRestoreDiceConfigSize        = uint64(16 << 20)
	maxRestoreRequestIDBytes        = 128
	minimumDiskReserve              = uint64(1 << 30)
	restoreStatusTokenValidDuration = 24 * time.Hour // 覆盖大型安全备份、恢复应用和 Runtime 重建的完整维护窗口。
	restoreStatusTokenBytes         = 32
)

type BackupArchiveInfo struct {
	Name             string `json:"name"`
	FileSize         int64  `json:"fileSize"`
	Selection        int64  `json:"selection"`
	Version          string `json:"version"`
	VersionCode      int64  `json:"versionCode"`
	FormatVersion    int    `json:"formatVersion,omitempty"`
	DatabaseType     string `json:"databaseType,omitempty"`
	Valid            bool   `json:"valid"`
	Restorable       bool   `json:"restorable"`
	Error            string `json:"error,omitempty"`
	RestoreError     string `json:"restoreError,omitempty"`
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
	OperationID      string `json:"operationId"`
	StatusTokenHash  string `json:"statusTokenHash,omitempty"`
	ExpiresAt        int64  `json:"expiresAt,omitempty"`
	SourceName       string `json:"sourceName"`
	SafetyBackupName string `json:"safetyBackupName,omitempty"`
}

type RestoreOperation struct {
	OperationID      string `json:"operationId"`
	StatusToken      string `json:"statusToken"`
	ExpiresAt        int64  `json:"expiresAt"`
	SafetyBackupName string `json:"safetyBackupName,omitempty"`
	Reused           bool   `json:"reused"`
}

type restoreJournalEntry struct {
	Target      string `json:"target"`
	Rollback    string `json:"rollback"`
	Staged      string `json:"staged,omitempty"`
	HadOriginal bool   `json:"hadOriginal"`
	State       string `json:"-"`

	backupIntent  bool
	backupDone    bool
	installIntent bool
	installDone   bool
	removeIntent  bool
	removeDone    bool
	restoreIntent bool
	restoreDone   bool
}

type restoreJournal struct {
	OperationID string
	State       string
	Entries     []restoreJournalEntry
}

type restoreJournalRecord struct {
	OperationID string                `json:"operationId"`
	Type        string                `json:"type"`
	State       string                `json:"state,omitempty"`
	Index       int                   `json:"index,omitempty"`
	Step        string                `json:"step,omitempty"`
	Phase       string                `json:"phase,omitempty"`
	Entries     []restoreJournalEntry `json:"entries,omitempty"`
}

type restoreStatusAuth struct {
	OperationID string `json:"operationId"`
	Token       string `json:"-"`
	TokenHash   string `json:"tokenHash,omitempty"`
	ExpiresAt   int64  `json:"expiresAt"`
}

type backupManifestFile struct {
	Path   string `json:"path"`
	Size   uint64 `json:"size"`
	SHA256 string `json:"sha256"`
}

type backupManifest struct {
	FormatVersion int                  `json:"formatVersion,omitempty"`
	RestorePolicy string               `json:"restorePolicy,omitempty"`
	DatabaseType  string               `json:"databaseType,omitempty"`
	Files         []backupManifestFile `json:"files,omitempty"`
	Config        json.RawMessage      `json:"config"`
	Version       json.RawMessage      `json:"version"`
	VersionCode   int64                `json:"versionCode"`
}

type managedDatabaseFile struct {
	Name        string
	Path        string
	ArchivePath string
}

var (
	restoreRename              = renameRestorePath
	restoreRemove              = os.Remove
	restoreRemoveAll           = os.RemoveAll
	restoreSyncMutationParents = syncRestoreMutationParents
	probeRestoreDiskSpace      = platformPackageDiskSpace
	errEmptyRestoreJournal     = errors.New("恢复日志不含完整 begin 记录")
	restoreTokenMemory         sync.Map // operationID -> in-memory status token
)

func mustMarshalJSON(value any) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}

func newRestoreStatusToken() (string, error) {
	raw := make([]byte, restoreStatusTokenBytes)
	if _, err := cryptorand.Read(raw); err != nil {
		return "", fmt.Errorf("生成恢复状态凭证: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func restoreStatusTokenMatches(token, encodedHash string) bool {
	digest := sha256.Sum256([]byte(token))
	return subtle.ConstantTimeCompare([]byte(strings.ToLower(encodedHash)), []byte(hex.EncodeToString(digest[:]))) == 1
}

func newRestoreStatusAuth(operationID string) (*restoreStatusAuth, error) {
	token, err := newRestoreStatusToken()
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256([]byte(token))
	auth := &restoreStatusAuth{
		OperationID: operationID,
		Token:       token,
		TokenHash:   hex.EncodeToString(digest[:]),
		ExpiresAt:   time.Now().Add(restoreStatusTokenValidDuration).Unix(),
	}
	restoreTokenMemory.Store(operationID, token)
	return auth, nil
}

func restoreDir() string { return filepath.Join(BackupDir, restoreDirName) }

func restorePendingPath() string { return filepath.Join(restoreDir(), restorePendingName) }

func restoreStatusPath() string { return filepath.Join(restoreDir(), restoreStatusName) }

func restoreJournalPath() string { return filepath.Join(restoreDir(), restoreJournalName) }

func restoreLegacyJournalPath() string { return filepath.Join(restoreDir(), restoreLegacyJournalName) }

func restoreStatusAuthPath() string { return filepath.Join(restoreDir(), restoreStatusAuthName) }

func validateBackupDirectoryLocked() error {
	info, err := os.Lstat(BackupDir)
	if err != nil {
		return err
	}
	if !info.IsDir() || isLinkedRestorePath(info) {
		return errors.New("备份目录不是普通目录或为符号链接")
	}
	return nil
}

func ensureBackupDirectoryLocked() error {
	if err := validateBackupDirectoryLocked(); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(BackupDir, 0o755); err != nil {
		return err
	}
	return validateBackupDirectoryLocked()
}

func validateRestoreStorageLocked() error {
	if err := validateBackupDirectoryLocked(); err != nil {
		return err
	}
	info, err := os.Lstat(restoreDir())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() || isLinkedRestorePath(info) {
		return errors.New("恢复元数据目录不是普通目录或为符号链接")
	}
	return ensureNoLinkedPath(BackupDir, restoreDir(), false)
}

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

func ensureDiskSpace(filename string, required uint64) error {
	space, err := probeRestoreDiskSpace(filename)
	if err != nil {
		return fmt.Errorf("检查磁盘空间失败: %w", err)
	}
	reserve := diskReserve(space.Total)
	if space.Available < required || space.Available-required < reserve {
		return fmt.Errorf("磁盘空间不足：需要 %d 字节并保留 %d 字节安全余量，当前可用 %d 字节", required, reserve, space.Available)
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
	var selection BackupSelection
	if cfg.Decks {
		selection |= BackupSelectionDecks
	}
	if cfg.HelpDoc {
		selection |= BackupSelectionHelpDoc
	}
	if cfg.Censor {
		selection |= BackupSelectionCensor
	}
	if cfg.Names {
		selection |= BackupSelectionNames
	}
	if cfg.Images {
		selection |= BackupSelectionImages
	}
	for _, diceConfig := range cfg.Dices {
		if diceConfig.JSScripts {
			selection |= BackupSelectionJS
			break
		}
	}
	return int64(selection)
}

func pathWithinRoot(root, target string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return "", fmt.Errorf("路径 %s 不在受管目录 %s 内", target, root)
	}
	return relative, nil
}

func ensureNoLinkedPath(root, target string, finalMayBeFile bool) error {
	relative, err := pathWithinRoot(root, target)
	if err != nil {
		return err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	parts := []string{"."}
	if relative != "." {
		parts = strings.Split(relative, string(filepath.Separator))
	}
	current := rootAbs
	for index, part := range parts {
		if part != "." {
			current = filepath.Join(current, part)
		}
		info, statErr := os.Lstat(current)
		if errors.Is(statErr, os.ErrNotExist) {
			return nil
		}
		if statErr != nil {
			return fmt.Errorf("检查路径 %s: %w", current, statErr)
		}
		if isLinkedRestorePath(info) {
			return fmt.Errorf("路径包含符号链接或重解析点: %s", current)
		}
		resolved, resolveErr := filepath.EvalSymlinks(current)
		if resolveErr == nil {
			resolvedInfo, resolvedErr := os.Stat(resolved)
			if resolvedErr != nil {
				return resolvedErr
			}
			if !os.SameFile(info, resolvedInfo) {
				return fmt.Errorf("路径包含符号链接或重解析点: %s", current)
			}
		}
		last := index == len(parts)-1
		if !last && !info.IsDir() {
			return fmt.Errorf("路径父级不是目录: %s", current)
		}
		if last && finalMayBeFile && !info.IsDir() && !info.Mode().IsRegular() {
			return fmt.Errorf("路径不是普通文件: %s", current)
		}
	}
	return nil
}

func managedDataRoot() (string, error) {
	root, err := filepath.Abs("data")
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(root)
	if err != nil {
		return "", fmt.Errorf("读取受管 data 根目录: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("受管 data 根目录不是普通目录或为符号链接")
	}
	if err = ensureNoLinkedPath(root, root, false); err != nil {
		return "", err
	}
	return root, nil
}

func managedArchivePath(filename string) (string, error) {
	root, err := managedDataRoot()
	if err != nil {
		return "", err
	}
	relative, err := pathWithinRoot(root, filename)
	if err != nil {
		return "", err
	}
	if relative == "." {
		return "", errors.New("不能将 data 根目录作为备份文件")
	}
	if err = ensureNoLinkedPath(root, filename, true); err != nil {
		return "", err
	}
	return filepath.ToSlash(filepath.Join("data", relative)), nil
}

func managedSQLiteDatabaseFiles(dm *DiceManager, requireExisting bool) ([]managedDatabaseFile, error) {
	if dm == nil || dm.Operator == nil || len(dm.Dice) == 0 {
		return nil, errors.New("恢复要求非空的 SQLite DiceManager")
	}
	if dm.Operator.Type() != constant.SQLITE {
		return nil, fmt.Errorf("备份恢复仅支持 SQLite，当前数据库类型为 %q", dm.Operator.Type())
	}
	root, err := managedDataRoot()
	if err != nil {
		return nil, err
	}
	for _, d := range dm.Dice {
		if d == nil || d.BaseConfig.Name == "" || d.BaseConfig.DataDir == "" {
			return nil, errors.New("恢复要求所有 Dice 实例都具有非空名称和受管 DataDir")
		}
		if _, pathErr := pathWithinRoot(root, d.BaseConfig.DataDir); pathErr != nil {
			return nil, fmt.Errorf("骰子 %q 的 DataDir 不在受管 data 根目录内: %w", d.BaseConfig.Name, pathErr)
		}
		if pathErr := ensureNoLinkedPath(root, d.BaseConfig.DataDir, false); pathErr != nil {
			return nil, fmt.Errorf("骰子 %q 的 DataDir 不安全: %w", d.BaseConfig.Name, pathErr)
		}
	}
	dataDir := os.Getenv("DATADIR")
	if dataDir == "" {
		dataDir = filepath.Join("data", "default")
	}
	dataDirAbs, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, fmt.Errorf("解析 DATADIR: %w", err)
	}
	if _, err = pathWithinRoot(root, dataDirAbs); err != nil {
		return nil, fmt.Errorf("环境变量 DATADIR 必须位于受管 data 根目录内: %w", err)
	}
	if err = ensureNoLinkedPath(root, dataDirAbs, false); err != nil {
		return nil, fmt.Errorf("环境变量 DATADIR 不安全: %w", err)
	}
	if requireExisting {
		info, statErr := os.Lstat(dataDirAbs)
		if statErr != nil {
			return nil, fmt.Errorf("读取 DATADIR: %w", statErr)
		}
		if !info.IsDir() {
			return nil, errors.New("环境变量 DATADIR 指向的路径不是目录")
		}
	}
	result := make([]managedDatabaseFile, 0, 3)
	for _, name := range []string{"data.db", "data-logs.db", "data-censor.db"} {
		filename := filepath.Join(dataDirAbs, name)
		archiveName, archiveErr := managedArchivePath(filename)
		if archiveErr != nil {
			return nil, fmt.Errorf("数据库文件 %s 不在受管 data 根目录内: %w", name, archiveErr)
		}
		info, statErr := os.Lstat(filename)
		if statErr != nil {
			if requireExisting || !errors.Is(statErr, os.ErrNotExist) {
				return nil, fmt.Errorf("读取数据库文件 %s: %w", name, statErr)
			}
		} else if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("数据库文件 %s 不是普通文件或为符号链接", name)
		}
		result = append(result, managedDatabaseFile{Name: name, Path: filename, ArchivePath: archiveName})
	}
	return result, nil
}

func isWindowsReservedName(segment string) bool {
	base := strings.ToUpper(strings.TrimRight(strings.SplitN(segment, ".", 2)[0], " ."))
	if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || base == "CONIN$" || base == "CONOUT$" || base == "CLOCK$" {
		return true
	}
	if strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT") {
		suffix := base[3:]
		return (len(suffix) == 1 && suffix[0] >= '1' && suffix[0] <= '9') || suffix == "¹" || suffix == "²" || suffix == "³"
	}
	return false
}

func normalizeBackupEntryName(name string, isDir bool) (string, error) {
	if name == "" || !utf8.ValidString(name) {
		return "", errors.New("压缩包包含空路径或非法 UTF-8 路径")
	}
	if len(name) > maxBackupEntryPathBytes {
		return "", fmt.Errorf("压缩包路径超过 %d 字节限制", maxBackupEntryPathBytes)
	}
	normalized := strings.ReplaceAll(name, "\\", "/")
	if strings.HasPrefix(normalized, "/") || strings.HasPrefix(normalized, "//") || path.IsAbs(normalized) || filepath.VolumeName(name) != "" ||
		(len(normalized) >= 2 && normalized[1] == ':') {
		return "", fmt.Errorf("压缩包包含绝对路径 %q", name)
	}
	trimmed := strings.TrimSuffix(normalized, "/")
	clean := path.Clean(trimmed)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean != trimmed {
		return "", fmt.Errorf("压缩包包含非法路径 %q", name)
	}
	for _, segment := range strings.Split(clean, "/") {
		if segment == "" || strings.HasSuffix(segment, ".") || strings.HasSuffix(segment, " ") || strings.ContainsAny(segment, `:<>"|?*`) || isWindowsReservedName(segment) {
			return "", fmt.Errorf("压缩包包含 Windows 不安全路径 %q", name)
		}
		for _, char := range segment {
			if char < 0x20 {
				return "", fmt.Errorf("压缩包路径包含控制字符 %q", name)
			}
		}
	}
	if isDir && normalized != trimmed+"/" && normalized != trimmed {
		return "", fmt.Errorf("压缩包包含非法目录路径 %q", name)
	}
	return clean, nil
}

func isSQLiteSidecarPath(filename string) bool {
	lower := strings.ToLower(filename)
	return strings.HasSuffix(lower, ".db-wal") || strings.HasSuffix(lower, ".db-shm") || strings.HasSuffix(lower, ".db-journal")
}

func checkCompressionRatio(name string, compressed, uncompressed uint64) error {
	if uncompressed <= compressionRatioMinimumSize {
		return nil
	}
	if compressed == 0 || uncompressed/compressed > maxBackupCompressionRatio {
		return fmt.Errorf("压缩包条目 %q 的压缩比异常", name)
	}
	return nil
}

type zipDirectoryMetadata struct {
	records   uint64
	size      uint64
	offset    uint64
	endOffset int64
	zip64     bool
}

func readZIPDirectoryMetadata(file *os.File, size int64) (zipDirectoryMetadata, error) {
	if size < 22 {
		return zipDirectoryMetadata{}, errors.New("zip 文件过短")
	}
	tailSize := int64(22 + 1<<16)
	if tailSize > size {
		tailSize = size
	}
	tail := make([]byte, int(tailSize))
	if _, err := file.ReadAt(tail, size-tailSize); err != nil {
		return zipDirectoryMetadata{}, err
	}
	eocdIndex := -1
	for index := len(tail) - 22; index >= 0; index-- {
		if binary.LittleEndian.Uint32(tail[index:index+4]) != 0x06054b50 {
			continue
		}
		commentLength := int(binary.LittleEndian.Uint16(tail[index+20 : index+22]))
		if index+22+commentLength == len(tail) {
			eocdIndex = index
			break
		}
	}
	if eocdIndex < 0 {
		return zipDirectoryMetadata{}, errors.New("zip 缺少有效中央目录结束记录")
	}
	eocd := tail[eocdIndex : eocdIndex+22]
	disk := binary.LittleEndian.Uint16(eocd[4:6])
	directoryDisk := binary.LittleEndian.Uint16(eocd[6:8])
	recordsOnDisk := binary.LittleEndian.Uint16(eocd[8:10])
	records := binary.LittleEndian.Uint16(eocd[10:12])
	metadata := zipDirectoryMetadata{
		records:   uint64(records),
		size:      uint64(binary.LittleEndian.Uint32(eocd[12:16])),
		offset:    uint64(binary.LittleEndian.Uint32(eocd[16:20])),
		endOffset: size - tailSize + int64(eocdIndex),
	}
	if disk != 0 || directoryDisk != 0 || recordsOnDisk != records {
		return zipDirectoryMetadata{}, errors.New("不支持分卷 zip")
	}
	needsZIP64 := records == math.MaxUint16 || metadata.size == math.MaxUint32 || metadata.offset == math.MaxUint32
	if !needsZIP64 {
		return metadata, nil
	}
	locatorOffset := metadata.endOffset - 20
	if locatorOffset < 0 {
		return zipDirectoryMetadata{}, errors.New("zip64 缺少中央目录定位记录")
	}
	locator := make([]byte, 20)
	if _, err := file.ReadAt(locator, locatorOffset); err != nil {
		return zipDirectoryMetadata{}, err
	}
	if binary.LittleEndian.Uint32(locator[:4]) != 0x07064b50 || binary.LittleEndian.Uint32(locator[4:8]) != 0 ||
		binary.LittleEndian.Uint32(locator[16:20]) != 1 {
		return zipDirectoryMetadata{}, errors.New("zip64 中央目录定位记录非法")
	}
	zip64Offset := binary.LittleEndian.Uint64(locator[8:16])
	if zip64Offset > math.MaxInt64 || int64(zip64Offset) > size-56 {
		return zipDirectoryMetadata{}, errors.New("zip64 中央目录结束记录越界")
	}
	header := make([]byte, 56)
	if _, err := file.ReadAt(header, int64(zip64Offset)); err != nil {
		return zipDirectoryMetadata{}, err
	}
	if binary.LittleEndian.Uint32(header[:4]) != 0x06064b50 || binary.LittleEndian.Uint64(header[4:12]) < 44 {
		return zipDirectoryMetadata{}, errors.New("zip64 中央目录结束记录非法")
	}
	disk64 := binary.LittleEndian.Uint32(header[16:20])
	directoryDisk64 := binary.LittleEndian.Uint32(header[20:24])
	recordsOnDisk64 := binary.LittleEndian.Uint64(header[24:32])
	metadata.records = binary.LittleEndian.Uint64(header[32:40])
	metadata.size = binary.LittleEndian.Uint64(header[40:48])
	metadata.offset = binary.LittleEndian.Uint64(header[48:56])
	metadata.endOffset = int64(zip64Offset)
	metadata.zip64 = true
	if disk64 != 0 || directoryDisk64 != 0 || recordsOnDisk64 != metadata.records {
		return zipDirectoryMetadata{}, errors.New("不支持分卷 zip64")
	}
	return metadata, nil
}

func preflightZIPCentralDirectory(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	return preflightZIPCentralDirectoryFile(file)
}

func preflightZIPCentralDirectoryFile(file *os.File) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	metadata, err := readZIPDirectoryMetadata(file, info.Size())
	if err != nil {
		return err
	}
	if metadata.records > maxBackupEntries {
		return fmt.Errorf("压缩包条目超过 %d 个", maxBackupEntries)
	}
	if metadata.size > maxBackupCentralDirectorySize || metadata.size > math.MaxInt64 {
		return errors.New("zip 中央目录超过大小限制")
	}
	if metadata.offset > math.MaxInt64 {
		return errors.New("zip 中央目录偏移越界")
	}
	baseOffset := metadata.endOffset - int64(metadata.size) - int64(metadata.offset)
	start := baseOffset + int64(metadata.offset)
	if baseOffset > 0 {
		var signature [4]byte
		if _, readErr := file.ReadAt(signature[:], int64(metadata.offset)); readErr == nil && binary.LittleEndian.Uint32(signature[:]) == 0x02014b50 {
			start = int64(metadata.offset)
		}
	}
	if start < 0 || int64(metadata.size) > info.Size()-start {
		return errors.New("zip 中央目录范围越界")
	}
	position := start
	end := start + int64(metadata.size)
	var actualRecords uint64
	var header [46]byte
	for position < end {
		if end-position < int64(len(header)) {
			break
		}
		if _, err = file.ReadAt(header[:], position); err != nil {
			return err
		}
		if binary.LittleEndian.Uint32(header[:4]) != 0x02014b50 {
			break
		}
		nameLength := uint64(binary.LittleEndian.Uint16(header[28:30]))
		extraLength := uint64(binary.LittleEndian.Uint16(header[30:32]))
		commentLength := uint64(binary.LittleEndian.Uint16(header[32:34]))
		if nameLength > maxBackupEntryPathBytes {
			return fmt.Errorf("zip 中央目录路径超过 %d 字节限制", maxBackupEntryPathBytes)
		}
		recordSize, overflow := addUint64Checked(uint64(len(header)), nameLength, extraLength, commentLength)
		if overflow || recordSize > uint64(end-position) {
			return errors.New("zip 中央目录条目越界")
		}
		position += int64(recordSize)
		actualRecords++
		if actualRecords > maxBackupEntries {
			return fmt.Errorf("压缩包条目超过 %d 个", maxBackupEntries)
		}
	}
	if position != end {
		var signatureHeader [6]byte
		if end-position < int64(len(signatureHeader)) {
			return errors.New("zip 中央目录尾部非法")
		}
		if _, err = file.ReadAt(signatureHeader[:], position); err != nil {
			return err
		}
		signatureSize := int64(binary.LittleEndian.Uint16(signatureHeader[4:6]))
		if binary.LittleEndian.Uint32(signatureHeader[:4]) != 0x05054b50 || position+6+signatureSize != end {
			return errors.New("zip 中央目录包含非法尾部记录")
		}
	}
	if metadata.zip64 {
		if actualRecords != metadata.records {
			return errors.New("zip64 中央目录条目数不一致")
		}
	} else if uint16(actualRecords) != uint16(metadata.records) { //nolint:gosec // ZIP32 stores the count modulo 65536.
		return errors.New("zip 中央目录条目数不一致")
	}
	return nil
}

func inspectBackupArchive(filename string) (*BackupArchiveInfo, map[string]struct{}, error) {
	info, entries, _, err := inspectBackupArchiveDetailed(filename)
	return info, entries, err
}

func inspectBackupArchiveDetailed(filename string) (*BackupArchiveInfo, map[string]struct{}, backupManifest, error) {
	stat, err := os.Lstat(filename)
	if err != nil {
		var empty backupManifest
		return nil, nil, empty, err
	}
	if !stat.Mode().IsRegular() {
		var empty backupManifest
		return nil, nil, empty, errors.New("备份文件不是普通文件或为符号链接")
	}
	file, err := os.Open(filename)
	if err != nil {
		var empty backupManifest
		return nil, nil, empty, err
	}
	defer func() { _ = file.Close() }()
	return inspectBackupArchiveDetailedFile(file, stat, filename)
}

func inspectBackupArchiveDetailedFile(file *os.File, stat os.FileInfo, filename string) (*BackupArchiveInfo, map[string]struct{}, backupManifest, error) {
	var emptyManifest backupManifest
	if err := preflightZIPCentralDirectoryFile(file); err != nil {
		return nil, nil, emptyManifest, fmt.Errorf("zip 中央目录预检失败: %w", err)
	}
	reader, err := zip.NewReader(file, stat.Size())
	if err != nil {
		return nil, nil, emptyManifest, fmt.Errorf("不是有效的 ZIP 文件: %w", err)
	}
	if len(reader.File) > maxBackupEntries {
		return nil, nil, emptyManifest, fmt.Errorf("压缩包条目超过 %d 个", maxBackupEntries)
	}

	entries := make(map[string]struct{}, len(reader.File))
	entryKinds := make(map[string]bool, len(reader.File))
	requiredDirs := make(map[string]struct{}, len(reader.File))
	regularSizes := make(map[string]uint64, len(reader.File))
	var manifest backupManifest
	manifestFound := false
	dataConfigFound := false
	processOwnedPath := ""
	var uncompressed uint64
	var compressed uint64
	for _, item := range reader.File {
		clean, cleanErr := normalizeBackupEntryName(item.Name, item.FileInfo().IsDir())
		if cleanErr != nil {
			return nil, nil, emptyManifest, cleanErr
		}
		mode := item.Mode()
		if mode&os.ModeSymlink != 0 || (!item.FileInfo().IsDir() && !mode.IsRegular()) {
			return nil, nil, emptyManifest, fmt.Errorf("压缩包包含不支持的文件类型 %q", item.Name)
		}
		if clean != "backup_info.json" && clean != "data" && !strings.HasPrefix(clean, "data/") {
			return nil, nil, emptyManifest, fmt.Errorf("压缩包包含 data 目录外的文件 %q", item.Name)
		}
		if !item.FileInfo().IsDir() && isSQLiteSidecarPath(clean) {
			return nil, nil, emptyManifest, fmt.Errorf("压缩包不得包含 SQLite 临时文件 %q", item.Name)
		}
		if !item.FileInfo().IsDir() && processOwnedPath == "" && isProcessOwnedRestorePath(clean) {
			processOwnedPath = clean
		}
		folded := strings.ToLower(clean)
		if _, exists := entryKinds[folded]; exists {
			return nil, nil, emptyManifest, fmt.Errorf("压缩包包含重复或大小写冲突路径 %q", item.Name)
		}
		if !item.FileInfo().IsDir() {
			if _, neededAsDir := requiredDirs[folded]; neededAsDir {
				return nil, nil, emptyManifest, fmt.Errorf("压缩包路径同时作为文件和目录 %q", item.Name)
			}
		}
		parts := strings.Split(clean, "/")
		for index := 1; index < len(parts); index++ {
			ancestor := strings.ToLower(strings.Join(parts[:index], "/"))
			if isDir, exists := entryKinds[ancestor]; exists && !isDir {
				return nil, nil, emptyManifest, fmt.Errorf("压缩包路径的父级是文件 %q", item.Name)
			}
			requiredDirs[ancestor] = struct{}{}
		}
		entries[clean] = struct{}{}
		entryKinds[folded] = item.FileInfo().IsDir()
		if item.UncompressedSize64 > maxBackupEntryUncompressedSize {
			return nil, nil, emptyManifest, fmt.Errorf("压缩包条目 %q 解压后过大", item.Name)
		}
		if err = checkCompressionRatio(item.Name, item.CompressedSize64, item.UncompressedSize64); err != nil {
			return nil, nil, emptyManifest, err
		}
		if math.MaxUint64-uncompressed < item.UncompressedSize64 || math.MaxUint64-compressed < item.CompressedSize64 {
			return nil, nil, emptyManifest, errors.New("压缩包大小计算溢出")
		}
		uncompressed += item.UncompressedSize64
		compressed += item.CompressedSize64
		if uncompressed > maxBackupTotalUncompressedSize {
			return nil, nil, emptyManifest, errors.New("压缩包总解压大小超过限制")
		}
		if clean == "data/dice.yaml" && !item.FileInfo().IsDir() {
			dataConfigFound = true
		}
		if !item.FileInfo().IsDir() && clean != "backup_info.json" {
			regularSizes[clean] = item.UncompressedSize64
		}
		if clean == "backup_info.json" {
			if item.FileInfo().IsDir() || item.UncompressedSize64 > 1<<20 {
				return nil, nil, emptyManifest, errors.New("backup_info.json 类型非法或过大")
			}
			rc, openErr := item.Open()
			if openErr != nil {
				return nil, nil, emptyManifest, fmt.Errorf("读取 backup_info.json 失败: %w", openErr)
			}
			decoder := json.NewDecoder(io.LimitReader(rc, 1<<20))
			decodeErr := decoder.Decode(&manifest)
			closeErr := rc.Close()
			if decodeErr != nil {
				return nil, nil, emptyManifest, fmt.Errorf("解析 backup_info.json 失败: %w", decodeErr)
			}
			if closeErr != nil {
				return nil, nil, emptyManifest, fmt.Errorf("校验 backup_info.json 失败: %w", closeErr)
			}
			manifestFound = true
		}
	}
	if err = checkCompressionRatio("整个压缩包", compressed, uncompressed); err != nil {
		return nil, nil, emptyManifest, err
	}
	if !manifestFound {
		return nil, nil, emptyManifest, errors.New("缺少有效的 backup_info.json")
	}
	if manifest.FormatVersion == backupManifestFormatVersion && manifest.VersionCode <= 0 {
		return nil, nil, emptyManifest, errors.New("v2 备份缺少有效的 versionCode")
	}
	if manifest.VersionCode > 0 && manifest.VersionCode > VERSION_CODE {
		return nil, nil, emptyManifest, fmt.Errorf("备份版本 %d 高于当前程序版本 %d", manifest.VersionCode, VERSION_CODE)
	}
	if !dataConfigFound {
		return nil, nil, emptyManifest, errors.New("备份缺少 data/dice.yaml")
	}
	restorable := true
	if manifest.FormatVersion > backupManifestFormatVersion {
		return nil, nil, emptyManifest, fmt.Errorf("备份格式版本 %d 高于支持版本 %d", manifest.FormatVersion, backupManifestFormatVersion)
	}
	if manifest.FormatVersion != 0 && manifest.FormatVersion != backupManifestFormatVersion {
		return nil, nil, emptyManifest, fmt.Errorf("不支持的备份格式版本 %d", manifest.FormatVersion)
	}
	if manifest.FormatVersion == backupManifestFormatVersion {
		if manifest.RestorePolicy != backupRestorePolicyOverlay {
			return nil, nil, emptyManifest, fmt.Errorf("不支持的恢复策略 %q", manifest.RestorePolicy)
		}
		if manifest.DatabaseType == "" {
			return nil, nil, emptyManifest, errors.New("v2 备份缺少 databaseType")
		}
		restorable = manifest.DatabaseType == constant.SQLITE
		manifestPaths := make(map[string]struct{}, len(manifest.Files))
		for _, file := range manifest.Files {
			clean, pathErr := normalizeBackupEntryName(file.Path, false)
			if pathErr != nil || clean != file.Path || strings.Contains(file.Path, "\\") || clean == "backup_info.json" {
				return nil, nil, emptyManifest, fmt.Errorf("v2 清单包含非法文件路径 %q", file.Path)
			}
			folded := strings.ToLower(clean)
			if _, exists := manifestPaths[folded]; exists {
				return nil, nil, emptyManifest, fmt.Errorf("v2 清单包含重复或大小写冲突路径 %q", file.Path)
			}
			digest, hashErr := hex.DecodeString(file.SHA256)
			if hashErr != nil || len(digest) != sha256.Size {
				return nil, nil, emptyManifest, fmt.Errorf("v2 清单包含非法 SHA-256: %s", file.Path)
			}
			archiveSize, exists := regularSizes[clean]
			if !exists || archiveSize != file.Size {
				return nil, nil, emptyManifest, fmt.Errorf("v2 清单与 ZIP 条目不一致: %s", file.Path)
			}
			manifestPaths[folded] = struct{}{}
		}
		if len(manifestPaths) != len(regularSizes) {
			return nil, nil, emptyManifest, errors.New("v2 清单没有完整覆盖 ZIP 文件")
		}
	}
	restoreError := ""
	if processOwnedPath != "" {
		restorable = false
		restoreError = fmt.Sprintf("备份包含进程级占用文件 %s", processOwnedPath)
	}
	return &BackupArchiveInfo{
		Name:             filepath.Base(filename),
		FileSize:         stat.Size(),
		Selection:        selectionFromManifest(manifest.Config),
		Version:          parseManifestVersion(manifest.Version),
		VersionCode:      manifest.VersionCode,
		FormatVersion:    manifest.FormatVersion,
		DatabaseType:     manifest.DatabaseType,
		Valid:            true,
		Restorable:       restorable,
		RestoreError:     restoreError,
		UncompressedSize: uncompressed,
	}, entries, manifest, nil
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
	stat, err := os.Lstat(filename)
	if err != nil {
		return err
	}
	if !stat.Mode().IsRegular() {
		return errors.New("备份文件不是普通文件或为符号链接")
	}
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	return validateBackupDataFile(file, stat, filename)
}

func validateBackupDataFile(file *os.File, stat os.FileInfo, filename string) error {
	_, _, manifest, err := inspectBackupArchiveDetailedFile(file, stat, filename)
	if err != nil {
		return err
	}
	expected := make(map[string]backupManifestFile, len(manifest.Files))
	if manifest.FormatVersion == backupManifestFormatVersion {
		for _, file := range manifest.Files {
			expected[file.Path] = file
		}
	}
	reader, err := zip.NewReader(file, stat.Size())
	if err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(expected))
	var archivedDiceConfig []byte
	for _, item := range reader.File {
		if item.FileInfo().IsDir() {
			continue
		}
		clean, cleanErr := normalizeBackupEntryName(item.Name, false)
		if cleanErr != nil {
			return cleanErr
		}
		rc, openErr := item.Open()
		if openErr != nil {
			return openErr
		}
		hasher := sha256.New()
		destination := io.Writer(hasher)
		var diceConfig bytes.Buffer
		if clean == "data/dice.yaml" {
			if item.UncompressedSize64 > maxRestoreDiceConfigSize {
				_ = rc.Close()
				return fmt.Errorf("data/dice.yaml 超过 %d 字节限制", maxRestoreDiceConfigSize)
			}
			destination = io.MultiWriter(hasher, &diceConfig)
		}
		written, copyErr := io.Copy(destination, rc)
		closeErr := rc.Close()
		if copyErr != nil || closeErr != nil || uint64(written) != item.UncompressedSize64 {
			return fmt.Errorf("校验 %s 失败", item.Name)
		}
		if clean == "backup_info.json" || manifest.FormatVersion != backupManifestFormatVersion {
			if clean == "data/dice.yaml" {
				archivedDiceConfig = append([]byte(nil), diceConfig.Bytes()...)
			}
			continue
		}
		expectedFile, exists := expected[clean]
		if !exists || expectedFile.Size != uint64(written) || !strings.EqualFold(expectedFile.SHA256, hex.EncodeToString(hasher.Sum(nil))) {
			return fmt.Errorf("v2 文件校验失败: %s", clean)
		}
		seen[clean] = struct{}{}
		if clean == "data/dice.yaml" {
			archivedDiceConfig = append([]byte(nil), diceConfig.Bytes()...)
		}
	}
	if len(seen) != len(expected) {
		return errors.New("v2 文件清单校验不完整")
	}
	var config struct {
		DiceConfigs []BaseConfig `yaml:"diceConfigs"`
	}
	if err = yaml.Unmarshal(archivedDiceConfig, &config); err != nil {
		return fmt.Errorf("解析 data/dice.yaml 失败: %w", err)
	}
	if err = ValidateDiceConfigNamesPortable(config.DiceConfigs); err != nil {
		return fmt.Errorf("data/dice.yaml 包含不安全的骰子名称: %w", err)
	}
	return nil
}

func copyFileSynced(source, target string) error {
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()
	info, err := src.Stat()
	if err != nil {
		return err
	}
	return copyFileSyncedFile(src, info, target)
}

func copyFileSyncedFile(src *os.File, info os.FileInfo, target string) error {
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return err
	}
	dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err = io.Copy(dst, src); err != nil {
		_ = dst.Close()
		return err
	}
	if err = dst.Sync(); err != nil {
		_ = dst.Close()
		return err
	}
	return dst.Close()
}

func syncDirectory(directory string) error {
	return syncRestoreDirectoryPath(directory)
}

func syncRestoreMutationParents(rootAndFiles ...string) error {
	if len(rootAndFiles)%2 != 0 {
		return errors.New("恢复目录同步参数不成对")
	}
	seen := make(map[string]struct{}, len(rootAndFiles))
	for index := 0; index < len(rootAndFiles); index += 2 {
		rootAbs, err := filepath.Abs(rootAndFiles[index])
		if err != nil {
			return err
		}
		filenameAbs, err := filepath.Abs(rootAndFiles[index+1])
		if err != nil {
			return err
		}
		if _, err = pathWithinRoot(rootAbs, filenameAbs); err != nil {
			return err
		}
		current := filepath.Dir(filenameAbs)
		for {
			key := strings.ToLower(filepath.Clean(current))
			if _, exists := seen[key]; !exists {
				if err = syncRestoreDirectoryPath(current); err != nil {
					return fmt.Errorf("同步恢复目录 %s: %w", current, err)
				}
				seen[key] = struct{}{}
			}
			if filepath.Clean(current) == filepath.Clean(rootAbs) {
				break
			}
			parent := filepath.Dir(current)
			if parent == current {
				return fmt.Errorf("同步路径 %s 未到达受管根 %s", filenameAbs, rootAbs)
			}
			current = parent
		}
	}
	return nil
}

func writeJSONAtomic(filename string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(filename), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(filename), "."+filepath.Base(filename)+"-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if err = tmp.Chmod(0o600); err == nil {
		_, err = tmp.Write(data)
	}
	if err == nil {
		err = tmp.Sync()
	}
	closeErr := tmp.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err = replaceFileAtomic(tmpName, filename); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(filename))
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
	backupOperationMu.Lock()
	defer backupOperationMu.Unlock()
	if err := validateRestoreStorageLocked(); err != nil {
		return err
	}
	pending, err := readPendingRestore()
	if err != nil {
		return err
	}
	if pending.OperationID == "" {
		return errors.New("恢复任务缺少 operationID")
	}
	return setRestoreStatus(RestoreStatus{
		State:            state,
		OperationID:      pending.OperationID,
		SourceName:       pending.SourceName,
		SafetyBackupName: pending.SafetyBackupName,
		Message:          message,
	})
}

func backupInUseLocked(name string) (bool, error) {
	pending, err := readPendingRestore()
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("读取恢复任务以确认备份占用: %w", err)
	}
	return strings.EqualFold(name, pending.SourceName) || strings.EqualFold(name, pending.SafetyBackupName), nil
}

func BackupInUse(name string) bool {
	backupOperationMu.Lock()
	defer backupOperationMu.Unlock()
	inUse, err := backupInUseLocked(name)
	return err != nil || inUse
}

func validBackupFilename(name string) bool {
	if filepath.Base(name) != name || strings.ToLower(filepath.Ext(name)) != ".zip" || strings.HasSuffix(name, ".") || strings.HasSuffix(name, " ") {
		return false
	}
	_, err := normalizeBackupEntryName("data/"+name, false)
	return err == nil
}

func backupArchiveFileLocked(name string) (string, os.FileInfo, error) {
	if !validBackupFilename(name) {
		return "", nil, errors.New("备份文件名非法")
	}
	if err := validateBackupDirectoryLocked(); err != nil {
		return "", nil, err
	}
	target := filepath.Join(BackupDir, name)
	if _, err := pathWithinRoot(BackupDir, target); err != nil {
		return "", nil, err
	}
	info, err := os.Lstat(target)
	if err != nil {
		return "", nil, err
	}
	if !info.Mode().IsRegular() {
		return "", nil, errors.New("备份目标不是普通文件或为符号链接")
	}
	return target, info, nil
}

func backupArchivePathLocked(name string) (string, error) {
	target, _, err := backupArchiveFileLocked(name)
	return target, err
}

// BackupArchivePath 返回经过文件名、边界和普通文件校验的备份路径。
func BackupArchivePath(name string) (string, error) {
	backupOperationMu.Lock()
	defer backupOperationMu.Unlock()
	return backupArchivePathLocked(name)
}

// OpenBackupArchive 在同一临界区内校验并打开备份，返回的句柄不受后续路径替换影响。
func OpenBackupArchive(name string) (*os.File, os.FileInfo, error) {
	backupOperationMu.Lock()
	defer backupOperationMu.Unlock()
	return openBackupArchiveLocked(name)
}

func openBackupArchiveLocked(name string) (*os.File, os.FileInfo, error) {
	target, expected, err := backupArchiveFileLocked(name)
	if err != nil {
		return nil, nil, err
	}
	file, err := os.Open(target)
	if err != nil {
		return nil, nil, err
	}
	actual, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, nil, err
	}
	if !actual.Mode().IsRegular() || !os.SameFile(expected, actual) || actual.Size() != expected.Size() || !actual.ModTime().Equal(expected.ModTime()) {
		_ = file.Close()
		return nil, nil, errors.New("打开的备份文件与已校验目标不一致")
	}
	return file, actual, nil
}

func deleteBackupLocked(name string) error {
	inUse, err := backupInUseLocked(name)
	if err != nil {
		return err
	}
	if inUse {
		return fmt.Errorf("备份 %s 正被恢复事务使用，不能删除", name)
	}
	target, err := backupArchivePathLocked(name)
	if err != nil {
		return err
	}
	return os.Remove(target)
}

func DeleteBackup(name string) error {
	backupOperationMu.Lock()
	defer backupOperationMu.Unlock()
	return deleteBackupLocked(name)
}

func DeleteBackups(names []string) error {
	backupOperationMu.Lock()
	defer backupOperationMu.Unlock()
	var resultErr error
	for _, name := range names {
		if err := deleteBackupLocked(name); err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("删除 %s: %w", name, err))
		}
	}
	return resultErr
}

func fileSHA256(filename string) ([sha256.Size]byte, error) {
	info, err := os.Lstat(filename)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	if !info.Mode().IsRegular() {
		return [sha256.Size]byte{}, errors.New("哈希目标不是普通文件或为符号链接")
	}
	file, err := os.Open(filename)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	defer func() { _ = file.Close() }()
	return fileSHA256File(file)
}

func fileSHA256File(file *os.File) ([sha256.Size]byte, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return [sha256.Size]byte{}, err
	}
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return [sha256.Size]byte{}, err
	}
	var digest [sha256.Size]byte
	copy(digest[:], hasher.Sum(nil))
	return digest, nil
}

func ImportBackup(reader io.Reader) (*BackupArchiveInfo, error) {
	if err := ensureBackupDirectoryLocked(); err != nil {
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
	var uploaded uint64
	emptyReads := 0
	for {
		count, readErr := reader.Read(buffer)
		if count > 0 {
			emptyReads = 0
			if uint64(count) > maxImportedBackupSize-uploaded {
				_ = tmp.Close()
				return nil, fmt.Errorf("上传备份超过 %d 字节限制", maxImportedBackupSize)
			}
			if err = ensureDiskSpace(BackupDir, uint64(count)); err != nil {
				_ = tmp.Close()
				return nil, err
			}
			if _, err = tmp.Write(buffer[:count]); err != nil {
				_ = tmp.Close()
				return nil, err
			}
			_, _ = hash.Write(buffer[:count])
			uploaded += uint64(count)
		} else if readErr == nil {
			emptyReads++
			if emptyReads >= 100 {
				_ = tmp.Close()
				return nil, io.ErrNoProgress
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			_ = tmp.Close()
			return nil, readErr
		}
	}
	if err = tmp.Sync(); err == nil {
		err = tmp.Close()
	} else {
		_ = tmp.Close()
	}
	if err != nil {
		return nil, err
	}

	// The network upload itself must not hold the global backup lock; only
	// validation, deduplication and the atomic publish step are serialized.
	backupOperationMu.Lock()
	defer backupOperationMu.Unlock()
	if err = validateBackupDirectoryLocked(); err != nil {
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
	digest := hex.EncodeToString(uploadedDigest[:])
	existing, globErr := filepath.Glob(filepath.Join(BackupDir, "import_*_"+digest+".zip"))
	if globErr != nil {
		return nil, fmt.Errorf("查找同内容备份: %w", globErr)
	}
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
	if err = renameRestorePath(tmpName, target); err != nil {
		return nil, err
	}
	if err = syncDirectory(BackupDir); err != nil {
		_ = os.Remove(target)
		return nil, err
	}
	info.Name = name
	return info, nil
}

func validateSafetyBackup(filename string, dm *DiceManager) error {
	info, entries, err := inspectBackupArchive(filename)
	if err != nil {
		return err
	}
	if info.FormatVersion != backupManifestFormatVersion || info.DatabaseType != constant.SQLITE {
		return errors.New("安全备份不是 SQLite v2 备份")
	}
	if err = validateBackupData(filename); err != nil {
		return err
	}
	databaseFiles, err := managedSQLiteDatabaseFiles(dm, true)
	if err != nil {
		return err
	}
	required := []string{"data/dice.yaml"}
	for _, d := range dm.Dice {
		if d == nil || d.BaseConfig.Name == "" {
			return errors.New("安全备份包含空骰子实例")
		}
		archiveName, pathErr := managedArchivePath(filepath.Join(d.BaseConfig.DataDir, "serve.yaml"))
		if pathErr != nil {
			return fmt.Errorf("安全备份实例配置路径不安全: %w", pathErr)
		}
		required = append(required, archiveName)
	}
	for _, databaseFile := range databaseFiles {
		required = append(required, databaseFile.ArchivePath)
	}
	for _, filename := range required {
		if _, ok := entries[filepath.ToSlash(filename)]; !ok {
			return fmt.Errorf("安全备份缺少 %s", filename)
		}
	}
	return nil
}

func directorySize(root string) (uint64, error) {
	var total uint64
	err := filepath.WalkDir(root, func(filename string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		if entry.Type()&os.ModeSymlink != 0 || isLinkedRestorePath(info) {
			return fmt.Errorf("目录包含符号链接或重解析点: %s", filename)
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if info.Size() < 0 || math.MaxUint64-total < uint64(info.Size()) {
			return errors.New("目录大小计算溢出")
		}
		total += uint64(info.Size())
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

func readRestoreStatusAuth() (*restoreStatusAuth, error) {
	data, err := os.ReadFile(restoreStatusAuthPath())
	if err != nil {
		return nil, err
	}
	var auth restoreStatusAuth
	if err = json.Unmarshal(data, &auth); err != nil {
		return nil, err
	}
	if auth.Token == "" && auth.TokenHash != "" {
		if cached, ok := restoreTokenMemory.Load(auth.OperationID); ok {
			if token, ok := cached.(string); ok && restoreStatusTokenMatches(token, auth.TokenHash) {
				auth.Token = token
			}
		}
	}
	return &auth, nil
}

func rebuildMissingRestoreAuth(pending *restorePending) (*restoreStatusAuth, error) {
	if pending.Phase != "scheduled" && pending.Phase != "prepared" {
		return nil, fmt.Errorf("恢复授权缺失且 pending 处于不安全阶段 %q", pending.Phase)
	}
	if _, journalErr := os.Stat(restoreJournalPath()); journalErr == nil {
		return nil, errors.New("恢复授权缺失但 journal 已存在，无法无歧义重建")
	} else if !errors.Is(journalErr, os.ErrNotExist) {
		return nil, journalErr
	}
	if _, _, err := inspectBackupArchive(filepath.Join(restoreDir(), restoreSourceName)); err != nil {
		return nil, fmt.Errorf("恢复授权缺失且排队源文件无法确认: %w", err)
	}
	auth, err := newRestoreStatusAuth(pending.OperationID)
	if err != nil {
		return nil, err
	}
	if err := writeJSONAtomic(restoreStatusAuthPath(), auth); err != nil {
		return nil, err
	}
	pending.StatusTokenHash = auth.TokenHash
	pending.ExpiresAt = auth.ExpiresAt
	if err := writeJSONAtomic(restorePendingPath(), pending); err != nil {
		return nil, err
	}
	return auth, nil
}

func refreshRestoreStatusAuth(pending *restorePending, auth *restoreStatusAuth) (*restoreStatusAuth, error) {
	if auth == nil || auth.OperationID == "" {
		return nil, errors.New("恢复授权不完整")
	}
	now := time.Now().Unix()
	if auth.ExpiresAt > now && auth.Token != "" {
		if !restoreStatusTokenMatches(auth.Token, auth.TokenHash) {
			return nil, errors.New("恢复授权 token 与哈希不一致")
		}
		if pending != nil && (pending.ExpiresAt != auth.ExpiresAt ||
			subtle.ConstantTimeCompare([]byte(strings.ToLower(pending.StatusTokenHash)), []byte(strings.ToLower(auth.TokenHash))) != 1) {
			pending.StatusTokenHash = auth.TokenHash
			pending.ExpiresAt = auth.ExpiresAt
			if err := writeJSONAtomic(restorePendingPath(), pending); err != nil {
				return nil, fmt.Errorf("修复恢复任务授权元数据: %w", err)
			}
		}
		return auth, nil
	}

	refreshed, err := newRestoreStatusAuth(auth.OperationID)
	if err != nil {
		return nil, err
	}
	if err = writeJSONAtomic(restoreStatusAuthPath(), refreshed); err != nil {
		return nil, err
	}
	if pending != nil {
		pending.StatusTokenHash = refreshed.TokenHash
		pending.ExpiresAt = refreshed.ExpiresAt
		if err = writeJSONAtomic(restorePendingPath(), pending); err != nil {
			return nil, err
		}
	}
	return refreshed, nil
}

func safePendingRestoreForReuse(pending *restorePending) (bool, error) {
	if pending.Phase != "scheduled" && pending.Phase != "prepared" {
		return false, nil
	}
	journal, err := readRestoreJournal()
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}
	if errors.Is(err, errEmptyRestoreJournal) {
		if discardErr := discardEmptyRestoreJournal(); discardErr != nil {
			return false, discardErr
		}
		return true, nil
	}
	if err != nil {
		return false, err
	}
	if journal.OperationID != pending.OperationID {
		return false, errors.New("可重试任务的 pending/journal operationID 不一致")
	}
	return journal.State == "rolled_back", nil
}

func restoreStatusForPending(pending *restorePending) RestoreStatus {
	status := RestoreStatus{
		State:            "pending",
		OperationID:      pending.OperationID,
		SourceName:       pending.SourceName,
		SafetyBackupName: pending.SafetyBackupName,
		Message:          "恢复任务已重新确认",
	}
	if pending.Phase == "prepared" {
		status.State = "quiescing"
		status.Message = "恢复前安全备份已完成，等待关闭数据库"
	}
	return status
}

func restoreOperationFromPending(name, requestID string, pending *restorePending, status RestoreStatus) (*RestoreOperation, bool, error) {
	if pending.OperationID != requestID {
		auth, authErr := readRestoreStatusAuth()
		if authErr == nil && auth.OperationID == requestID {
			return nil, false, errors.New("恢复授权与当前 pending 指向不同 operationID")
		}
		return nil, false, nil
	}
	if pending.SourceName != name {
		return nil, false, fmt.Errorf("requestId %q 已用于另一个备份 %q", requestID, pending.SourceName)
	}
	if status.OperationID != "" && status.OperationID != requestID {
		return nil, false, errors.New("已有恢复任务的 pending/status operationID 不一致")
	}
	if pending.Phase == "prepared" {
		if pending.SafetyBackupName == "" {
			return nil, false, errors.New("prepared 恢复任务缺少安全备份名")
		}
		if _, err := backupArchivePathLocked(pending.SafetyBackupName); err != nil {
			return nil, false, fmt.Errorf("prepared 恢复任务的安全备份无效: %w", err)
		}
	}
	auth, authErr := readRestoreStatusAuth()
	if errors.Is(authErr, os.ErrNotExist) {
		auth, authErr = rebuildMissingRestoreAuth(pending)
	}
	if authErr != nil {
		return nil, false, fmt.Errorf("读取或重建恢复授权失败: %w", authErr)
	}
	if auth.OperationID != requestID {
		return nil, false, errors.New("同一 requestId 的恢复授权不完整或 operationID 不一致")
	}
	auth, authErr = refreshRestoreStatusAuth(pending, auth)
	if authErr != nil {
		return nil, false, fmt.Errorf("刷新恢复授权失败: %w", authErr)
	}
	resetStatus := status.OperationID == "" || status.State == "failed" || status.State == "rolled_back" || status.State == "degraded"
	if resetStatus {
		safe, safeErr := safePendingRestoreForReuse(pending)
		if safeErr != nil {
			return nil, false, fmt.Errorf("确认恢复任务可重试: %w", safeErr)
		}
		if !safe {
			return nil, false, fmt.Errorf("恢复状态为 %q，但 pending 阶段 %q 不能安全重置", status.State, pending.Phase)
		}
		if err := setRestoreStatus(restoreStatusForPending(pending)); err != nil {
			return nil, false, err
		}
	}
	return &RestoreOperation{
		OperationID: requestID, StatusToken: auth.Token, ExpiresAt: auth.ExpiresAt,
		SafetyBackupName: pending.SafetyBackupName, Reused: true,
	}, true, nil
}

func restoreOperationFromExisting(name, requestID string) (*RestoreOperation, bool, error) {
	pending, pendingErr := readPendingRestore()
	status := GetRestoreStatus()
	if pendingErr == nil {
		return restoreOperationFromPending(name, requestID, pending, status)
	}
	if !errors.Is(pendingErr, os.ErrNotExist) {
		return nil, false, fmt.Errorf("读取已有恢复任务失败: %w", pendingErr)
	}
	// A finished restore keeps its terminal status after metadata cleanup, so
	// an idempotent replay can be answered without re-scheduling anything.
	if status.State == "succeeded" && status.OperationID == requestID && status.SourceName == name {
		auth, authErr := readRestoreStatusAuth()
		if authErr == nil && auth.OperationID == requestID {
			auth, authErr = refreshRestoreStatusAuth(nil, auth)
			if authErr != nil {
				return nil, false, fmt.Errorf("刷新终态恢复授权失败: %w", authErr)
			}
			return &RestoreOperation{OperationID: requestID, StatusToken: auth.Token, ExpiresAt: auth.ExpiresAt, Reused: true}, true, nil
		}
		if authErr != nil && !errors.Is(authErr, os.ErrNotExist) {
			return nil, false, fmt.Errorf("读取终态恢复授权失败: %w", authErr)
		}
		return &RestoreOperation{OperationID: requestID, Reused: true}, true, nil
	}
	auth, authErr := readRestoreStatusAuth()
	if errors.Is(authErr, os.ErrNotExist) || (authErr == nil && auth.OperationID != requestID) {
		return nil, false, nil
	}
	if authErr != nil {
		return nil, false, fmt.Errorf("读取已有恢复授权失败: %w", authErr)
	}
	if status.OperationID != requestID || status.SourceName != name {
		return nil, false, errors.New("恢复授权存在但 pending/status 无法确认同一 requestId 和备份源")
	}
	if status.State != "succeeded" {
		return nil, false, fmt.Errorf("pending 缺失且恢复状态为 %q，不能当作可复用操作", status.State)
	}
	auth, authErr = refreshRestoreStatusAuth(nil, auth)
	if authErr != nil {
		return nil, false, fmt.Errorf("刷新终态恢复授权失败: %w", authErr)
	}
	return &RestoreOperation{
		OperationID:      requestID,
		StatusToken:      auth.Token,
		ExpiresAt:        auth.ExpiresAt,
		SafetyBackupName: status.SafetyBackupName,
		Reused:           true,
	}, true, nil
}

func releaseRetryablePendingRestore() error {
	pending, err := readPendingRestore()
	if errors.Is(err, os.ErrNotExist) {
		if _, journalErr := os.Stat(restoreJournalPath()); journalErr == nil {
			return errors.New("恢复日志存在但 pending 缺失，无法无歧义释放")
		} else if !errors.Is(journalErr, os.ErrNotExist) {
			return journalErr
		}
		if _, legacyJournalErr := os.Stat(restoreLegacyJournalPath()); legacyJournalErr == nil {
			return errors.New("检测到旧版恢复日志 journal.json 且 pending 缺失，拒绝自动清除")
		} else if !errors.Is(legacyJournalErr, os.ErrNotExist) {
			return legacyJournalErr
		}
		legacyOldData := filepath.Join(restoreDir(), restoreLegacyOldDataName)
		if _, oldDataErr := os.Stat(legacyOldData); oldDataErr == nil {
			return fmt.Errorf("检测到旧版恢复遗留目录 %s 且 pending 缺失，拒绝自动清除", restoreLegacyOldDataName)
		} else if !errors.Is(oldDataErr, os.ErrNotExist) {
			return oldDataErr
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("读取已有恢复任务: %w", err)
	}
	status := GetRestoreStatus()
	if pending.OperationID == "" || (status.OperationID != "" && status.OperationID != pending.OperationID) {
		return errors.New("已有恢复任务的 pending/status operationID 不一致")
	}
	terminal := status.State == "failed" || status.State == "rolled_back" || status.State == "degraded" || status.State == "succeeded" ||
		(status.State == "idle" && (pending.Phase == "scheduled" || pending.Phase == "prepared"))
	if !terminal {
		return errors.New("已有恢复任务正在等待执行")
	}
	legacyOldData := filepath.Join(restoreDir(), restoreLegacyOldDataName)
	if _, err = os.Stat(legacyOldData); err == nil {
		return fmt.Errorf("已有恢复任务保留了旧数据目录 %s，拒绝自动清除", legacyOldData)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("检查旧版恢复遗留目录: %w", err)
	}
	journal, journalErr := readRestoreJournal()
	if journalErr == nil && journal.OperationID != pending.OperationID {
		return errors.New("已有恢复任务的 pending/journal operationID 不一致")
	}
	if journalErr == nil {
		switch journal.State {
		case "rolled_back":
			return nil
		case "committed":
			if status.State == "succeeded" {
				return nil
			}
			return errors.New("已有恢复事务已经提交但尚未发布成功，不能覆盖")
		default:
			return fmt.Errorf("已有恢复事务尚未安全结束，当前状态为 %s", journal.State)
		}
	}
	if errors.Is(journalErr, errEmptyRestoreJournal) {
		if pending.Phase != "scheduled" && pending.Phase != "prepared" {
			return fmt.Errorf("空恢复日志对应不安全阶段 %q", pending.Phase)
		}
		return discardEmptyRestoreJournal()
	}
	if !errors.Is(journalErr, os.ErrNotExist) {
		return fmt.Errorf("检查已有恢复事务: %w", journalErr)
	}
	if pending.Phase != "scheduled" && pending.Phase != "prepared" {
		return fmt.Errorf("无日志恢复任务处于不安全阶段 %q，不能覆盖", pending.Phase)
	}
	return nil
}

func validRestoreRequestID(requestID string) bool {
	return requestID != "" && requestID == strings.TrimSpace(requestID) && len(requestID) <= maxRestoreRequestIDBytes && utf8.ValidString(requestID) &&
		!strings.ContainsAny(requestID, "\r\n\x00")
}

func ensureRestoreVolumesMatch() error {
	dataSpace, err := probeRestoreDiskSpace("data")
	if err != nil {
		return fmt.Errorf("检查 data 所在卷: %w", err)
	}
	restoreSpace, err := probeRestoreDiskSpace(BackupDir)
	if err != nil {
		return fmt.Errorf("检查 backups 所在卷: %w", err)
	}
	if dataSpace.Volume != restoreSpace.Volume {
		return errors.New("data 与 backups 必须位于同一文件系统，才能保证恢复事务使用原子 rename")
	}
	return nil
}

func (dm *DiceManager) ScheduleRestore(name, requestID string) (*RestoreOperation, error) {
	backupOperationMu.Lock()
	defer backupOperationMu.Unlock()
	if !validBackupFilename(name) {
		return nil, errors.New("备份文件名非法")
	}
	if !validRestoreRequestID(requestID) {
		return nil, errors.New("requestId 为空、过长或包含非法字符")
	}
	if err := validateRestoreStorageLocked(); err != nil {
		return nil, err
	}
	if _, err := managedSQLiteDatabaseFiles(dm, true); err != nil {
		return nil, err
	}
	if operation, reused, err := restoreOperationFromExisting(name, requestID); err != nil {
		return nil, err
	} else if reused {
		return operation, nil
	}
	if err := releaseRetryablePendingRestore(); err != nil {
		return nil, err
	}
	if err := restoreRemoveAll(restoreDir()); err != nil {
		return nil, fmt.Errorf("清理旧恢复目录: %w", err)
	}
	if err := os.MkdirAll(restoreDir(), 0o700); err != nil {
		return nil, err
	}
	if err := syncDirectory(BackupDir); err != nil {
		return nil, err
	}
	if err := ensureRestoreVolumesMatch(); err != nil {
		return nil, err
	}
	sourceFile, sourceInfo, err := openBackupArchiveLocked(name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = sourceFile.Close() }()
	info, entries, _, err := inspectBackupArchiveDetailedFile(sourceFile, sourceInfo, filepath.Join(BackupDir, name))
	if err != nil {
		return nil, err
	}
	if !info.Restorable {
		if info.RestoreError != "" {
			return nil, errors.New(info.RestoreError)
		}
		if info.DatabaseType != "" && info.DatabaseType != constant.SQLITE {
			return nil, fmt.Errorf("备份数据库类型为 %q，不能恢复到 SQLite", info.DatabaseType)
		}
		return nil, errors.New("该备份包含不能由 Runtime 恢复的进程级文件")
	}
	if processPath := firstProcessOwnedRestorePath(entries); processPath != "" {
		return nil, fmt.Errorf("备份包含进程级占用文件 %s，不能安全恢复", processPath)
	}
	if err = validateBackupDataFile(sourceFile, sourceInfo, filepath.Join(BackupDir, name)); err != nil {
		return nil, fmt.Errorf("备份内容校验失败: %w", err)
	}
	dataSize, err := directorySize("data")
	if err != nil {
		return nil, err
	}
	required, overflow := addUint64Checked(dataSize, dataSize, info.UncompressedSize, uint64(info.FileSize))
	if overflow {
		return nil, errors.New("恢复所需磁盘空间计算溢出")
	}
	if err = ensureDiskSpace(BackupDir, required); err != nil {
		return nil, err
	}
	sourceDigest, err := fileSHA256File(sourceFile)
	if err != nil {
		return nil, err
	}
	queuedSource := filepath.Join(restoreDir(), restoreSourceName)
	if err = copyFileSyncedFile(sourceFile, sourceInfo, queuedSource); err != nil {
		return nil, err
	}
	queuedDigest, err := fileSHA256(queuedSource)
	if err != nil || queuedDigest != sourceDigest {
		return nil, errors.New("排队恢复源文件复制校验失败")
	}
	auth, err := newRestoreStatusAuth(requestID)
	if err != nil {
		return nil, err
	}
	pending := restorePending{
		Phase:           "scheduled",
		OperationID:     requestID,
		StatusTokenHash: auth.TokenHash,
		ExpiresAt:       auth.ExpiresAt,
		SourceName:      name,
	}
	if err = writeJSONAtomic(restorePendingPath(), pending); err != nil {
		return nil, err
	}
	if err = writeJSONAtomic(restoreStatusAuthPath(), auth); err != nil {
		return nil, err
	}
	if err = setRestoreStatus(RestoreStatus{State: "pending", OperationID: requestID, SourceName: name}); err != nil {
		return nil, err
	}
	return &RestoreOperation{OperationID: requestID, StatusToken: auth.Token, ExpiresAt: auth.ExpiresAt}, nil
}

func addUint64Checked(values ...uint64) (uint64, bool) {
	var result uint64
	for _, value := range values {
		if math.MaxUint64-result < value {
			return 0, true
		}
		result += value
	}
	return result, false
}

func validateRestoreStatusTokenLocked(operationID, token string) bool {
	if operationID == "" || token == "" {
		return false
	}
	auth, err := readRestoreStatusAuth()
	if err != nil || auth.OperationID != operationID || time.Now().Unix() > auth.ExpiresAt {
		return false
	}
	return restoreStatusTokenMatches(token, auth.TokenHash)
}

func ValidateRestoreStatusToken(operationID, token string) bool {
	backupOperationMu.Lock()
	defer backupOperationMu.Unlock()
	if validateRestoreStorageLocked() != nil {
		return false
	}
	return validateRestoreStatusTokenLocked(operationID, token)
}

// GetRestoreStatusAuthorized 在同一临界区内验证 bearer 并读取其绑定的恢复状态。
func GetRestoreStatusAuthorized(operationID, token string) (RestoreStatus, bool) {
	backupOperationMu.Lock()
	defer backupOperationMu.Unlock()
	if validateRestoreStorageLocked() != nil {
		return RestoreStatus{}, false
	}
	if !validateRestoreStatusTokenLocked(operationID, token) {
		return RestoreStatus{}, false
	}
	status := GetRestoreStatus()
	if status.OperationID != operationID {
		return RestoreStatus{}, false
	}
	return status, true
}

func PrepareScheduledRestore(dm *DiceManager) error {
	backupOperationMu.Lock()
	defer backupOperationMu.Unlock()
	if err := validateRestoreStorageLocked(); err != nil {
		return err
	}
	pending, err := readPendingRestore()
	if err != nil {
		return fmt.Errorf("读取恢复任务: %w", err)
	}
	if pending.OperationID == "" {
		return errors.New("恢复任务缺少 operationID")
	}
	if _, err = managedSQLiteDatabaseFiles(dm, true); err != nil {
		return err
	}
	if pending.Phase == "prepared" {
		if pending.SafetyBackupName == "" {
			return errors.New("prepared 恢复任务缺少安全备份名")
		}
		if err = validateSafetyBackup(filepath.Join(BackupDir, pending.SafetyBackupName), dm); err != nil {
			return fmt.Errorf("复用安全备份失败: %w", err)
		}
		return setRestoreStatus(RestoreStatus{State: "quiescing", OperationID: pending.OperationID, SourceName: pending.SourceName, SafetyBackupName: pending.SafetyBackupName, Message: "恢复前安全备份已完成，等待关闭数据库"})
	}
	if pending.Phase != "scheduled" {
		return fmt.Errorf("恢复任务不能从阶段 %q 准备", pending.Phase)
	}
	status := GetRestoreStatus()
	if status.OperationID != "" && status.OperationID != pending.OperationID {
		return errors.New("pending 与 status 的 operationID 不一致")
	}
	source := filepath.Join(restoreDir(), restoreSourceName)
	info, _, err := inspectBackupArchive(source)
	if err != nil {
		return fmt.Errorf("检查排队恢复源: %w", err)
	}
	if err = validateBackupData(source); err != nil {
		return fmt.Errorf("校验排队恢复源: %w", err)
	}
	dataSize, err := directorySize("data")
	if err != nil {
		return err
	}
	required, overflow := addUint64Checked(dataSize, dataSize, info.UncompressedSize, uint64(info.FileSize))
	if overflow {
		return errors.New("恢复所需磁盘空间计算溢出")
	}
	if err = ensureDiskSpace(BackupDir, required); err != nil {
		return err
	}
	logger.M().Infow("[备份恢复] 正在创建恢复前安全备份", "operationId", pending.OperationID, "source", pending.SourceName)
	safetyPath, err := dm.backupStrict(BackupSelectionAll)
	if err != nil {
		return fmt.Errorf("创建恢复前安全备份失败: %w", err)
	}
	if err = validateSafetyBackup(safetyPath, dm); err != nil {
		return fmt.Errorf("恢复前安全备份不完整: %w", err)
	}
	pending.SafetyBackupName = filepath.Base(safetyPath)
	pending.Phase = "prepared"
	if err = writeJSONAtomic(restorePendingPath(), pending); err != nil {
		return err
	}
	if err = setRestoreStatus(RestoreStatus{
		State:            "quiescing",
		OperationID:      pending.OperationID,
		SourceName:       pending.SourceName,
		SafetyBackupName: pending.SafetyBackupName,
		Message:          "恢复前安全备份已创建并校验",
	}); err != nil {
		return err
	}
	logger.M().Infow("[备份恢复] 恢复前安全备份已创建并校验", "operationId", pending.OperationID, "safetyBackup", pending.SafetyBackupName)
	return nil
}

func isProcessOwnedRestorePath(filename string) bool {
	lower := strings.ToLower(filepath.ToSlash(filename))
	if lower == "data/main.log" || lower == "data/panic.log" {
		return true
	}
	base := path.Base(lower)
	return strings.HasSuffix(base, ".lock") || base == "sealdice-lock.lock"
}

func firstProcessOwnedRestorePath(entries map[string]struct{}) string {
	paths := make([]string, 0)
	for filename := range entries {
		if isProcessOwnedRestorePath(filename) {
			paths = append(paths, filename)
		}
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
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
		if isProcessOwnedRestorePath(clean) {
			return fmt.Errorf("不能恢复进程级占用文件 %s", clean)
		}
		target := filepath.Join(destination, filepath.FromSlash(clean))
		if _, err = pathWithinRoot(destination, target); err != nil {
			return err
		}
		if item.FileInfo().IsDir() {
			if err = os.MkdirAll(target, 0o700); err != nil {
				return err
			}
			if err = ensureNoLinkedPath(destination, target, false); err != nil {
				return err
			}
			continue
		}
		if err = os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		if err = ensureNoLinkedPath(destination, filepath.Dir(target), false); err != nil {
			return err
		}
		rc, openErr := item.Open()
		if openErr != nil {
			return openErr
		}
		dst, createErr := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
		if createErr != nil {
			_ = rc.Close()
			return createErr
		}
		written, copyErr := io.Copy(dst, rc)
		if copyErr == nil {
			copyErr = dst.Sync()
		}
		closeDstErr := dst.Close()
		closeSrcErr := rc.Close()
		if copyErr != nil || closeDstErr != nil || closeSrcErr != nil || uint64(written) != item.UncompressedSize64 {
			return fmt.Errorf("解压 %s 失败或大小不匹配", item.Name)
		}
	}
	return nil
}

func appendJournalRecordFile(file *os.File, record restoreJournalRecord) error {
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.Size() < 0 || int64(len(data)) > maxRestoreJournalSize-info.Size() {
		return errors.New("恢复日志将超过大小限制")
	}
	written, err := file.Write(data)
	if err != nil {
		return err
	}
	if written != len(data) {
		return io.ErrShortWrite
	}
	return file.Sync()
}

func estimateRestoreJournalMaximum(journal *restoreJournal) (uint64, bool) {
	estimate := uint64(1024 + len(journal.OperationID)*12)
	recordSize := uint64(256 + len(journal.OperationID)*6)
	for _, entry := range journal.Entries {
		pathBytes, overflow := addUint64Checked(uint64(len(entry.Target)), uint64(len(entry.Rollback)), uint64(len(entry.Staged)))
		if overflow || pathBytes > (math.MaxUint64-512)/6 {
			return 0, true
		}
		entrySize := uint64(512) + pathBytes*6
		records := uint64(4) // backup intent/done and conservative state overhead.
		if entry.Staged != "" {
			records += 4 // install and remove intent/done.
		} else {
			records += 2 // rollback-only cleanup intent/done.
		}
		if entry.HadOriginal {
			records += 2 // restore-original intent/done.
		}
		recordBytes, recordOverflow := addUint64Checked(entrySize, records*recordSize)
		if recordOverflow || math.MaxUint64-estimate < recordBytes {
			return 0, true
		}
		estimate += recordBytes
	}
	return estimate, false
}

func appendRestoreJournalRecord(record restoreJournalRecord) error {
	file, err := os.OpenFile(restoreJournalPath(), os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	writeErr := appendJournalRecordFile(file, record)
	closeErr := file.Close()
	return errors.Join(writeErr, closeErr)
}

func createRestoreJournal(journal *restoreJournal) error {
	if journal == nil || journal.OperationID == "" {
		return errors.New("不能创建缺少 operationID 的恢复日志")
	}
	if journal.State == "" {
		journal.State = "planned"
	}
	if journal.State != "planned" {
		return fmt.Errorf("不能从状态 %q 创建恢复日志", journal.State)
	}
	estimatedSize, overflow := estimateRestoreJournalMaximum(journal)
	if overflow || estimatedSize > uint64(maxRestoreJournalSize) {
		return fmt.Errorf("恢复日志最坏情况将超过 %d 字节限制", maxRestoreJournalSize)
	}
	file, err := os.OpenFile(restoreJournalPath(), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	record := restoreJournalRecord{OperationID: journal.OperationID, Type: "begin", Entries: journal.Entries}
	writeErr := appendJournalRecordFile(file, record)
	closeErr := file.Close()
	if err = errors.Join(writeErr, closeErr); err != nil {
		return err
	}
	if err = syncDirectory(filepath.Dir(restoreJournalPath())); err != nil {
		return err
	}
	return appendJournalState(journal, "applying")
}

func applyJournalRecordStep(entry *restoreJournalEntry, record restoreJournalRecord) error {
	if record.Phase != "intent" && record.Phase != "done" {
		return fmt.Errorf("非法日志阶段 %q", record.Phase)
	}
	intent := record.Phase == "intent"
	switch record.Step {
	case "backup-original":
		if intent {
			entry.backupIntent = true
			entry.State = "backing_up"
		} else {
			if !entry.backupIntent {
				return errors.New("backup-original done 缺少 intent")
			}
			entry.backupDone = true
			entry.State = "backed_up"
		}
	case "install-staged":
		if intent {
			if !entry.backupDone {
				return errors.New("install-staged intent 早于 backup-original done")
			}
			entry.installIntent = true
			entry.State = "installing"
		} else {
			if !entry.installIntent {
				return errors.New("install-staged done 缺少 intent")
			}
			entry.installDone = true
			entry.State = "applied"
		}
	case "remove-installed":
		if intent {
			entry.removeIntent = true
		} else {
			if !entry.removeIntent {
				return errors.New("remove-installed done 缺少 intent")
			}
			entry.removeDone = true
		}
	case "restore-original":
		if intent {
			entry.restoreIntent = true
		} else {
			if !entry.restoreIntent {
				return errors.New("restore-original done 缺少 intent")
			}
			entry.restoreDone = true
			entry.State = "rolled_back"
		}
	default:
		return fmt.Errorf("非法日志步骤 %q", record.Step)
	}
	return nil
}

func validJournalStateTransition(current, next string) bool {
	switch current {
	case "planned":
		return next == "applying" || next == "rolling_back"
	case "applying":
		return next == "applied" || next == "rolling_back"
	case "applied":
		return next == "committed" || next == "rolling_back"
	case "rolling_back":
		return next == "rolled_back" || next == "rollback_failed"
	case "rollback_failed":
		return next == "rolling_back"
	default:
		return false
	}
}

func truncateTornRestoreJournal(expected os.FileInfo, size int64) error {
	file, err := os.OpenFile(restoreJournalPath(), os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	actual, statErr := file.Stat()
	if statErr != nil || !os.SameFile(expected, actual) {
		_ = file.Close()
		if statErr != nil {
			return statErr
		}
		return errors.New("截断恢复日志时目标文件已变化")
	}
	truncateErr := file.Truncate(size)
	var syncErr error
	if truncateErr == nil {
		syncErr = file.Sync()
	}
	closeErr := file.Close()
	return errors.Join(truncateErr, syncErr, closeErr)
}

func readRestoreJournal() (*restoreJournal, error) {
	info, err := os.Lstat(restoreJournalPath())
	if errors.Is(err, os.ErrNotExist) {
		if _, legacyErr := os.Stat(restoreLegacyJournalPath()); legacyErr == nil {
			return nil, errors.New("检测到旧版可覆写恢复日志 journal.json，无法无歧义自动恢复")
		} else if !errors.Is(legacyErr, os.ErrNotExist) {
			return nil, legacyErr
		}
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("恢复日志不是普通文件或为符号链接")
	}
	if info.Size() < 0 || info.Size() > maxRestoreJournalSize {
		return nil, errors.New("恢复日志超过大小限制")
	}
	file, err := os.Open(restoreJournalPath())
	if err != nil {
		return nil, err
	}
	data, readErr := io.ReadAll(io.LimitReader(file, maxRestoreJournalSize+1))
	closeErr := file.Close()
	if err = errors.Join(readErr, closeErr); err != nil {
		return nil, err
	}
	if int64(len(data)) > maxRestoreJournalSize {
		return nil, errors.New("恢复日志超过大小限制")
	}
	var journal *restoreJournal
	lineNumber := 0
	position := 0
	truncatedTail := false
	for position < len(data) {
		lineNumber++
		relativeEnd := bytes.IndexByte(data[position:], '\n')
		if relativeEnd < 0 {
			if err = truncateTornRestoreJournal(info, int64(position)); err != nil {
				return nil, fmt.Errorf("截断恢复日志残片: %w", err)
			}
			truncatedTail = true
			break
		}
		line := data[position : position+relativeEnd]
		position += relativeEnd + 1
		if len(line) == 0 {
			continue
		}
		var record restoreJournalRecord
		if err = json.Unmarshal(line, &record); err != nil {
			return nil, fmt.Errorf("解析恢复日志第 %d 行: %w", lineNumber, err)
		}
		if record.OperationID == "" {
			return nil, fmt.Errorf("恢复日志第 %d 行缺少 operationID", lineNumber)
		}
		if journal == nil {
			if record.Type != "begin" || len(record.Entries) == 0 {
				return nil, errors.New("恢复日志缺少有效 begin 记录")
			}
			journal = &restoreJournal{OperationID: record.OperationID, State: "planned", Entries: record.Entries}
			for index := range journal.Entries {
				if err := validateRestoreJournalEntryPaths(&journal.Entries[index]); err != nil {
					return nil, fmt.Errorf("恢复日志第 %d 条路径校验失败: %w", lineNumber, err)
				}
			}
			continue
		}
		if record.OperationID != journal.OperationID {
			return nil, errors.New("恢复日志包含多个 operationID")
		}
		switch record.Type {
		case "state":
			if !validJournalStateTransition(journal.State, record.State) {
				return nil, fmt.Errorf("恢复日志包含非法状态转换 %s -> %s", journal.State, record.State)
			}
			journal.State = record.State
		case "step":
			if journal.State == "committed" || journal.State == "rolled_back" {
				return nil, fmt.Errorf("恢复日志在终态 %s 后仍包含文件步骤", journal.State)
			}
			if record.Index < 0 || record.Index >= len(journal.Entries) {
				return nil, errors.New("恢复日志包含越界条目索引")
			}
			if err = applyJournalRecordStep(&journal.Entries[record.Index], record); err != nil {
				return nil, fmt.Errorf("恢复日志第 %d 行: %w", lineNumber, err)
			}
		default:
			return nil, fmt.Errorf("恢复日志包含非法记录类型 %q", record.Type)
		}
	}
	if journal == nil {
		if len(data) == 0 || truncatedTail {
			return nil, errEmptyRestoreJournal
		}
		return nil, errors.New("恢复日志为空或只包含不完整记录")
	}
	return journal, nil
}

func discardEmptyRestoreJournal() error {
	if err := restoreRemove(restoreJournalPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return syncDirectory(restoreDir())
}

func appendJournalState(journal *restoreJournal, state string) error {
	if !validJournalStateTransition(journal.State, state) {
		return fmt.Errorf("非法恢复日志状态转换 %s -> %s", journal.State, state)
	}
	record := restoreJournalRecord{OperationID: journal.OperationID, Type: "state", State: state}
	if err := appendRestoreJournalRecord(record); err != nil {
		return err
	}
	journal.State = state
	return nil
}

func appendJournalStep(journal *restoreJournal, index int, step, phase string) error {
	if index < 0 || index >= len(journal.Entries) {
		return errors.New("恢复日志条目索引越界")
	}
	record := restoreJournalRecord{OperationID: journal.OperationID, Type: "step", Index: index, Step: step, Phase: phase}
	if err := appendRestoreJournalRecord(record); err != nil {
		return err
	}
	return applyJournalRecordStep(&journal.Entries[index], record)
}

func ensureRestoreTargetSafe(target string) error {
	root, err := managedDataRoot()
	if err != nil {
		return err
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	if _, err = pathWithinRoot(root, targetAbs); err != nil {
		return fmt.Errorf("恢复目标不在 data 目录内: %s", target)
	}
	if err = ensureNoLinkedPath(root, targetAbs, true); err != nil {
		return fmt.Errorf("恢复目标不安全 %s: %w", target, err)
	}
	if info, statErr := os.Lstat(targetAbs); statErr == nil && info.IsDir() {
		return fmt.Errorf("恢复目标是目录: %s", target)
	} else if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return statErr
	}
	return nil
}

func ensureRestorePrivatePath(root, target string) error {
	if _, err := pathWithinRoot(root, target); err != nil {
		return err
	}
	return ensureNoLinkedPath(root, target, true)
}

func validateRestoreJournalEntryPaths(entry *restoreJournalEntry) error {
	if entry == nil {
		return errors.New("恢复日志包含空条目")
	}
	if err := ensureRestoreTargetSafe(entry.Target); err != nil {
		return fmt.Errorf("恢复日志目标不安全: %w", err)
	}
	rollbackRoot := filepath.Join(restoreDir(), restoreRollbackName)
	if err := ensureRestorePrivatePath(rollbackRoot, entry.Rollback); err != nil {
		return fmt.Errorf("恢复日志回滚路径不安全: %w", err)
	}
	if entry.Staged != "" {
		stagingRoot := filepath.Join(restoreDir(), restoreStagingName)
		if err := ensureRestorePrivatePath(stagingRoot, entry.Staged); err != nil {
			return fmt.Errorf("恢复日志暂存路径不安全: %w", err)
		}
		if info, statErr := os.Lstat(entry.Staged); statErr == nil && !info.Mode().IsRegular() {
			return fmt.Errorf("恢复日志暂存路径不是普通文件: %s", entry.Staged)
		} else if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
			return statErr
		}
	}
	return nil
}

func buildRestoreJournal(operationID, staging string) (*restoreJournal, error) {
	if operationID == "" {
		return nil, errors.New("恢复事务缺少 operationID")
	}
	journal := &restoreJournal{OperationID: operationID, State: "planned"}
	targets := map[string]struct{}{}
	stagingData := filepath.Join(staging, "data")
	err := filepath.WalkDir(stagingData, func(stagedPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("暂存目录包含符号链接: %s", stagedPath)
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("暂存目录包含特殊文件: %s", stagedPath)
		}
		relative, relErr := filepath.Rel(staging, stagedPath)
		if relErr != nil {
			return relErr
		}
		target := filepath.Clean(relative)
		if isProcessOwnedRestorePath(filepath.ToSlash(target)) {
			return fmt.Errorf("不能恢复进程级占用文件 %s", target)
		}
		if targetErr := ensureRestoreTargetSafe(target); targetErr != nil {
			return targetErr
		}
		rollback := filepath.Join(restoreDir(), restoreRollbackName, relative)
		if rollbackErr := ensureRestorePrivatePath(filepath.Join(restoreDir(), restoreRollbackName), rollback); rollbackErr != nil {
			return rollbackErr
		}
		_, statErr := os.Lstat(target)
		if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
			return statErr
		}
		targets[strings.ToLower(filepath.ToSlash(target))] = struct{}{}
		journal.Entries = append(journal.Entries, restoreJournalEntry{
			Target:      target,
			Rollback:    rollback,
			Staged:      stagedPath,
			HadOriginal: statErr == nil,
			State:       "planned",
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	entries := append([]restoreJournalEntry(nil), journal.Entries...)
	for _, entry := range entries {
		if !strings.HasSuffix(strings.ToLower(entry.Target), ".db") {
			continue
		}
		for _, suffix := range []string{"-wal", "-shm", "-journal"} {
			sidecar := entry.Target + suffix
			if _, exists := targets[strings.ToLower(filepath.ToSlash(sidecar))]; exists {
				continue
			}
			if err = ensureRestoreTargetSafe(sidecar); err != nil {
				return nil, err
			}
			sidecarRollback := filepath.Join(restoreDir(), restoreRollbackName, sidecar)
			if err = ensureRestorePrivatePath(filepath.Join(restoreDir(), restoreRollbackName), sidecarRollback); err != nil {
				return nil, fmt.Errorf("sqlite sidecar 回滚路径不安全: %w", err)
			}
			hadOriginal := false
			if info, statErr := os.Lstat(sidecar); statErr == nil {
				if !info.Mode().IsRegular() {
					return nil, fmt.Errorf("sqlite sidecar 不是普通文件: %s", sidecar)
				}
				hadOriginal = true
			} else if !errors.Is(statErr, os.ErrNotExist) {
				return nil, statErr
			}
			journal.Entries = append(journal.Entries, restoreJournalEntry{
				Target:      sidecar,
				Rollback:    filepath.Join(restoreDir(), restoreRollbackName, sidecar),
				HadOriginal: hadOriginal,
				State:       "planned",
			})
		}
	}
	sort.Slice(journal.Entries, func(i, j int) bool {
		return strings.ToLower(filepath.ToSlash(journal.Entries[i].Target)) < strings.ToLower(filepath.ToSlash(journal.Entries[j].Target))
	})
	return journal, nil
}

func pathExists(filename string) (bool, error) {
	_, err := os.Lstat(filename)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func applyRestoreJournal(journal *restoreJournal) (resultErr error) {
	defer func() {
		if resultErr != nil {
			if reloaded, reloadErr := readRestoreJournal(); reloadErr == nil {
				if reloaded.OperationID != journal.OperationID {
					resultErr = errors.Join(resultErr, errors.New("失败后的 journal operationID 发生变化"))
					return
				}
				journal = reloaded
			} else {
				resultErr = errors.Join(resultErr, fmt.Errorf("失败后重读 journal: %w", reloadErr))
			}
			resultErr = errors.Join(resultErr, rollbackRestoreJournal(journal))
		}
	}()
	for index := range journal.Entries {
		entry := &journal.Entries[index]
		if err := ensureRestoreTargetSafe(entry.Target); err != nil {
			return err
		}
		if entry.Staged != "" {
			stagingRoot := filepath.Join(restoreDir(), restoreStagingName)
			if err := ensureRestorePrivatePath(stagingRoot, entry.Staged); err != nil {
				return fmt.Errorf("恢复暂存路径不安全 %s: %w", entry.Staged, err)
			}
		}
		if !entry.backupIntent {
			if err := appendJournalStep(journal, index, "backup-original", "intent"); err != nil {
				return err
			}
		}
		if !entry.backupDone {
			if entry.HadOriginal {
				if err := os.MkdirAll(filepath.Dir(entry.Rollback), 0o700); err != nil {
					return err
				}
				if err := ensureRestorePrivatePath(filepath.Join(restoreDir(), restoreRollbackName), entry.Rollback); err != nil {
					return err
				}
				if err := restoreRename(entry.Target, entry.Rollback); err != nil {
					return fmt.Errorf("暂存原文件 %s: %w", entry.Target, err)
				}
				if err := restoreSyncMutationParents("data", entry.Target, restoreDir(), entry.Rollback); err != nil {
					return err
				}
			}
			if err := appendJournalStep(journal, index, "backup-original", "done"); err != nil {
				return err
			}
		}
		if entry.Staged == "" {
			entry.State = "applied"
			continue
		}
		if !entry.installIntent {
			if err := appendJournalStep(journal, index, "install-staged", "intent"); err != nil {
				return err
			}
		}
		if !entry.installDone {
			if err := os.MkdirAll(filepath.Dir(entry.Target), 0o700); err != nil {
				return err
			}
			if err := restoreRename(entry.Staged, entry.Target); err != nil {
				return fmt.Errorf("应用恢复文件 %s: %w", entry.Target, err)
			}
			if err := restoreSyncMutationParents(restoreDir(), entry.Staged, "data", entry.Target); err != nil {
				return err
			}
			if err := appendJournalStep(journal, index, "install-staged", "done"); err != nil {
				return err
			}
		}
	}
	return appendJournalState(journal, "applied")
}

func rollbackRestoreOnlyJournalEntry(journal *restoreJournal, index int, entry *restoreJournalEntry) error {
	if entry.restoreDone {
		entry.State = "rolled_back"
		return nil
	}
	backedUp := entry.backupDone
	if entry.HadOriginal && entry.backupIntent && !entry.backupDone {
		rollbackExists, rollbackErr := pathExists(entry.Rollback)
		targetExists, targetErr := pathExists(entry.Target)
		if rollbackErr != nil || targetErr != nil {
			return errors.Join(rollbackErr, targetErr)
		}
		switch {
		case rollbackExists && !targetExists:
			if err := restoreSyncMutationParents("data", entry.Target, restoreDir(), entry.Rollback); err != nil {
				return err
			}
			backedUp = true
		case !rollbackExists && targetExists:
			backedUp = false
		default:
			return fmt.Errorf("无法判断原文件 %s 是否已移入 rollback", entry.Target)
		}
	}
	if !backedUp {
		entry.State = "rolled_back"
		return nil
	}
	if entry.restoreIntent && !entry.restoreDone {
		rollbackExists, rollbackErr := pathExists(entry.Rollback)
		targetExists, targetErr := pathExists(entry.Target)
		if rollbackErr != nil || targetErr != nil {
			return errors.Join(rollbackErr, targetErr)
		}
		if !rollbackExists && targetExists {
			if err := restoreSyncMutationParents(restoreDir(), entry.Rollback, "data", entry.Target); err != nil {
				return err
			}
			return appendJournalStep(journal, index, "restore-original", "done")
		}
		if !rollbackExists || targetExists {
			return fmt.Errorf("无法判断原文件 %s 是否已恢复", entry.Target)
		}
	}

	if !entry.removeIntent {
		if err := appendJournalStep(journal, index, "remove-installed", "intent"); err != nil {
			return err
		}
	}
	if err := restoreRemove(entry.Target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("移除恢复期间生成的 SQLite sidecar %s: %w", entry.Target, err)
	}
	if err := restoreSyncMutationParents("data", entry.Target); err != nil {
		return err
	}
	if !entry.removeDone {
		if err := appendJournalStep(journal, index, "remove-installed", "done"); err != nil {
			return err
		}
	}
	if !entry.HadOriginal {
		entry.State = "rolled_back"
		return nil
	}

	if !entry.restoreIntent {
		if err := appendJournalStep(journal, index, "restore-original", "intent"); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(entry.Target), 0o700); err != nil {
		return err
	}
	if err := restoreRename(entry.Rollback, entry.Target); err != nil {
		return fmt.Errorf("恢复原文件 %s: %w", entry.Target, err)
	}
	if err := restoreSyncMutationParents(restoreDir(), entry.Rollback, "data", entry.Target); err != nil {
		return err
	}
	return appendJournalStep(journal, index, "restore-original", "done")
}

func rollbackRestoreJournal(journal *restoreJournal) error {
	if journal == nil || journal.OperationID == "" {
		return errors.New("无法回滚缺少 operationID 的恢复日志")
	}
	if journal.State == "committed" {
		return errors.New("恢复事务已经提交，拒绝回滚")
	}
	if journal.State != "rolling_back" {
		if err := appendJournalState(journal, "rolling_back"); err != nil {
			return err
		}
	}
	var rollbackErr error
	for index := len(journal.Entries) - 1; index >= 0; index-- {
		entry := &journal.Entries[index]
		if err := ensureRestoreTargetSafe(entry.Target); err != nil {
			rollbackErr = errors.Join(rollbackErr, err)
			continue
		}
		if err := ensureRestorePrivatePath(filepath.Join(restoreDir(), restoreRollbackName), entry.Rollback); err != nil {
			rollbackErr = errors.Join(rollbackErr, err)
			continue
		}
		if entry.Staged == "" {
			if err := rollbackRestoreOnlyJournalEntry(journal, index, entry); err != nil {
				rollbackErr = errors.Join(rollbackErr, err)
			}
			continue
		}
		if entry.restoreDone {
			entry.State = "rolled_back"
			continue
		}
		if entry.restoreIntent && !entry.restoreDone {
			rollbackExists, rollbackStatErr := pathExists(entry.Rollback)
			targetExists, targetStatErr := pathExists(entry.Target)
			if rollbackStatErr != nil || targetStatErr != nil {
				rollbackErr = errors.Join(rollbackErr, rollbackStatErr, targetStatErr)
				continue
			}
			if !rollbackExists && targetExists {
				if err := restoreSyncMutationParents(restoreDir(), entry.Rollback, "data", entry.Target); err != nil {
					rollbackErr = errors.Join(rollbackErr, err)
					continue
				}
				if err := appendJournalStep(journal, index, "restore-original", "done"); err != nil {
					rollbackErr = errors.Join(rollbackErr, err)
				}
				continue
			}
			if !rollbackExists || targetExists {
				rollbackErr = errors.Join(rollbackErr, fmt.Errorf("无法判断原文件 %s 是否已恢复", entry.Target))
				continue
			}
		}

		installed := entry.installDone
		if entry.Staged != "" && entry.installIntent && !entry.installDone && !entry.removeDone {
			stagedExists, stagedErr := pathExists(entry.Staged)
			targetExists, targetErr := pathExists(entry.Target)
			if stagedErr != nil || targetErr != nil {
				rollbackErr = errors.Join(rollbackErr, stagedErr, targetErr)
				continue
			}
			switch {
			case !stagedExists && targetExists:
				if err := restoreSyncMutationParents(restoreDir(), entry.Staged, "data", entry.Target); err != nil {
					rollbackErr = errors.Join(rollbackErr, err)
					continue
				}
				installed = true
			case !stagedExists && !targetExists && entry.removeIntent:
				installed = false
			case stagedExists && !targetExists:
				installed = false
			default:
				rollbackErr = errors.Join(rollbackErr, fmt.Errorf("无法判断 staged 文件 %s 是否已安装", entry.Target))
				continue
			}
		}
		if entry.removeDone {
			if err := restoreRemove(entry.Target); err != nil && !errors.Is(err, os.ErrNotExist) {
				rollbackErr = errors.Join(rollbackErr, fmt.Errorf("再次移除已恢复文件 %s: %w", entry.Target, err))
				continue
			}
			if err := restoreSyncMutationParents("data", entry.Target); err != nil {
				rollbackErr = errors.Join(rollbackErr, err)
				continue
			}
			installed = false
		} else if entry.removeIntent && !installed {
			if err := restoreSyncMutationParents("data", entry.Target); err != nil {
				rollbackErr = errors.Join(rollbackErr, err)
				continue
			}
			if err := appendJournalStep(journal, index, "remove-installed", "done"); err != nil {
				rollbackErr = errors.Join(rollbackErr, err)
				continue
			}
		}
		if installed && !entry.removeDone {
			if !entry.removeIntent {
				if err := appendJournalStep(journal, index, "remove-installed", "intent"); err != nil {
					rollbackErr = errors.Join(rollbackErr, err)
					continue
				}
			}
			if err := restoreRemove(entry.Target); err != nil && !errors.Is(err, os.ErrNotExist) {
				rollbackErr = errors.Join(rollbackErr, fmt.Errorf("移除已恢复文件 %s: %w", entry.Target, err))
				continue
			}
			if err := restoreSyncMutationParents("data", entry.Target); err != nil {
				rollbackErr = errors.Join(rollbackErr, err)
				continue
			}
			if err := appendJournalStep(journal, index, "remove-installed", "done"); err != nil {
				rollbackErr = errors.Join(rollbackErr, err)
				continue
			}
		}

		if !entry.HadOriginal || entry.restoreDone {
			entry.State = "rolled_back"
			continue
		}
		backedUp := entry.backupDone
		if entry.backupIntent && !entry.backupDone {
			rollbackExists, rollbackStatErr := pathExists(entry.Rollback)
			targetExists, targetStatErr := pathExists(entry.Target)
			if rollbackStatErr != nil || targetStatErr != nil {
				rollbackErr = errors.Join(rollbackErr, rollbackStatErr, targetStatErr)
				continue
			}
			switch {
			case rollbackExists && !targetExists:
				if err := restoreSyncMutationParents("data", entry.Target, restoreDir(), entry.Rollback); err != nil {
					rollbackErr = errors.Join(rollbackErr, err)
					continue
				}
				backedUp = true
			case !rollbackExists && targetExists && !installed:
				backedUp = false
			default:
				rollbackErr = errors.Join(rollbackErr, fmt.Errorf("无法判断原文件 %s 是否已移入 rollback", entry.Target))
				continue
			}
		}
		if !backedUp {
			entry.State = "rolled_back"
			continue
		}
		if entry.restoreIntent && !entry.restoreDone {
			rollbackExists, rollbackStatErr := pathExists(entry.Rollback)
			targetExists, targetStatErr := pathExists(entry.Target)
			if rollbackStatErr != nil || targetStatErr != nil {
				rollbackErr = errors.Join(rollbackErr, rollbackStatErr, targetStatErr)
				continue
			}
			if !rollbackExists && targetExists {
				if err := restoreSyncMutationParents(restoreDir(), entry.Rollback, "data", entry.Target); err != nil {
					rollbackErr = errors.Join(rollbackErr, err)
					continue
				}
				if err := appendJournalStep(journal, index, "restore-original", "done"); err != nil {
					rollbackErr = errors.Join(rollbackErr, err)
				}
				continue
			}
			if !rollbackExists || targetExists {
				rollbackErr = errors.Join(rollbackErr, fmt.Errorf("无法判断原文件 %s 是否已恢复", entry.Target))
				continue
			}
		}
		if !entry.restoreIntent {
			if err := appendJournalStep(journal, index, "restore-original", "intent"); err != nil {
				rollbackErr = errors.Join(rollbackErr, err)
				continue
			}
		}
		if err := os.MkdirAll(filepath.Dir(entry.Target), 0o700); err != nil {
			rollbackErr = errors.Join(rollbackErr, err)
			continue
		}
		if err := restoreRename(entry.Rollback, entry.Target); err != nil {
			rollbackErr = errors.Join(rollbackErr, fmt.Errorf("恢复原文件 %s: %w", entry.Target, err))
			continue
		}
		if err := restoreSyncMutationParents(restoreDir(), entry.Rollback, "data", entry.Target); err != nil {
			rollbackErr = errors.Join(rollbackErr, err)
			continue
		}
		if err := appendJournalStep(journal, index, "restore-original", "done"); err != nil {
			rollbackErr = errors.Join(rollbackErr, err)
		}
	}
	if rollbackErr != nil {
		stateErr := appendJournalState(journal, "rollback_failed")
		return errors.Join(rollbackErr, stateErr)
	}
	return appendJournalState(journal, "rolled_back")
}

func resetRolledBackPending() error {
	pending, err := readPendingRestore()
	if errors.Is(err, os.ErrNotExist) {
		return errors.New("恢复已回滚，但 pending 缺失，无法确认 operationID")
	}
	if err != nil {
		return fmt.Errorf("读取已回滚恢复任务: %w", err)
	}
	if pending.OperationID == "" {
		return errors.New("已回滚恢复任务缺少 operationID")
	}
	if pending.SafetyBackupName == "" {
		pending.Phase = "scheduled"
	} else {
		pending.Phase = "prepared"
	}
	if err = writeJSONAtomic(restorePendingPath(), pending); err != nil {
		return fmt.Errorf("重置已回滚恢复任务: %w", err)
	}
	return setRestoreStatus(RestoreStatus{
		State:            "rolled_back",
		OperationID:      pending.OperationID,
		SourceName:       pending.SourceName,
		SafetyBackupName: pending.SafetyBackupName,
		Message:          "检测到未提交恢复，原数据已回滚；可使用同一 requestId 重试",
	})
}

func cleanupCommittedArtifacts(removeMetadata bool) error {
	var cleanupErr error
	for _, directory := range []string{restoreRollbackName, restoreStagingName} {
		if err := restoreRemoveAll(filepath.Join(restoreDir(), directory)); err != nil {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("清理 %s: %w", directory, err))
		}
	}
	if err := restoreRemove(filepath.Join(restoreDir(), restoreSourceName)); err != nil && !errors.Is(err, os.ErrNotExist) {
		cleanupErr = errors.Join(cleanupErr, fmt.Errorf("清理恢复源文件: %w", err))
	}
	if cleanupErr == nil {
		cleanupErr = syncDirectory(restoreDir())
	}
	if !removeMetadata || cleanupErr != nil {
		return cleanupErr
	}
	if err := restoreRemove(restorePendingPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		cleanupErr = fmt.Errorf("清理恢复任务: %w", err)
	}
	if cleanupErr == nil {
		cleanupErr = syncDirectory(restoreDir())
	}
	if cleanupErr == nil {
		if err := restoreRemove(restoreJournalPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
			cleanupErr = fmt.Errorf("清理恢复日志: %w", err)
		}
	}
	if cleanupErr == nil {
		if err := restoreRemove(restoreStatusAuthPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
			cleanupErr = fmt.Errorf("清理恢复授权: %w", err)
		}
	}
	if cleanupErr == nil {
		cleanupErr = syncDirectory(restoreDir())
	}
	return cleanupErr
}

func cleanupRolledBackArtifactsWithoutPending(journal *restoreJournal) error {
	if err := setRestoreStatus(RestoreStatus{
		State:       "rolled_back",
		OperationID: journal.OperationID,
		Message:     "检测到已回滚恢复但 pending 缺失，事务已清理",
	}); err != nil {
		return err
	}
	var cleanupErr error
	for _, directory := range []string{restoreRollbackName, restoreStagingName} {
		if err := restoreRemoveAll(filepath.Join(restoreDir(), directory)); err != nil {
			cleanupErr = errors.Join(cleanupErr, err)
		}
	}
	if err := restoreRemove(filepath.Join(restoreDir(), restoreSourceName)); err != nil && !errors.Is(err, os.ErrNotExist) {
		cleanupErr = errors.Join(cleanupErr, err)
	}
	if cleanupErr == nil {
		cleanupErr = syncDirectory(restoreDir())
	}
	if cleanupErr == nil {
		if err := restoreRemove(restoreJournalPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
			cleanupErr = errors.Join(cleanupErr, err)
		}
	}
	if cleanupErr == nil {
		cleanupErr = syncDirectory(restoreDir())
	}
	return cleanupErr
}

func recoverInterruptedRestore() error {
	journal, journalErr := readRestoreJournal()
	journalEmpty := errors.Is(journalErr, errEmptyRestoreJournal)
	if errors.Is(journalErr, os.ErrNotExist) || journalEmpty {
		pending, pendingErr := readPendingRestore()
		if errors.Is(pendingErr, os.ErrNotExist) {
			return nil
		}
		if pendingErr != nil {
			return fmt.Errorf("读取无日志恢复任务: %w", pendingErr)
		}
		if pending.OperationID == "" {
			return errors.New("无日志恢复任务缺少 operationID")
		}
		status := GetRestoreStatus()
		if status.OperationID != "" && status.OperationID != pending.OperationID {
			return errors.New("无日志恢复任务的 pending/status operationID 不一致")
		}
		switch pending.Phase {
		case "scheduled", "prepared":
			if journalEmpty {
				return discardEmptyRestoreJournal()
			}
			return nil
		case "applied", "committed":
			if status.State != "succeeded" || status.OperationID != pending.OperationID {
				return fmt.Errorf("无日志 %s 任务缺少同 operationID 的 succeeded 状态", pending.Phase)
			}
			if journalEmpty {
				if err := discardEmptyRestoreJournal(); err != nil {
					return err
				}
			}
			if err := restoreRemove(restorePendingPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			return syncDirectory(restoreDir())
		case "switching", "swapped":
			legacyOldData := filepath.Join(restoreDir(), restoreLegacyOldDataName)
			if _, oldDataErr := os.Stat(legacyOldData); oldDataErr == nil {
				return fmt.Errorf("检测到旧版恢复遗留目录 %s，无法无歧义自动恢复", legacyOldData)
			} else if !errors.Is(oldDataErr, os.ErrNotExist) {
				return oldDataErr
			}
			pending.Phase = "scheduled"
			if err := writeJSONAtomic(restorePendingPath(), pending); err != nil {
				return err
			}
			return setRestoreStatus(RestoreStatus{State: "failed", OperationID: pending.OperationID, SourceName: pending.SourceName, SafetyBackupName: pending.SafetyBackupName, Message: "旧版恢复未移动数据，任务已重置"})
		default:
			return fmt.Errorf("恢复任务处于 %q 但 journal.jsonl 缺失，无法无歧义恢复", pending.Phase)
		}
	}
	if journalErr != nil {
		return fmt.Errorf("读取恢复事务日志: %w", journalErr)
	}
	pending, pendingErr := readPendingRestore()
	if pendingErr != nil && !errors.Is(pendingErr, os.ErrNotExist) {
		return pendingErr
	}
	if pendingErr == nil && pending.OperationID != journal.OperationID {
		return errors.New("pending 与 journal 的 operationID 不一致")
	}
	status := GetRestoreStatus()
	if status.OperationID != "" && status.OperationID != journal.OperationID {
		return errors.New("status 与 journal 的 operationID 不一致")
	}
	if journal.State == "committed" {
		removeMetadata := status.State == "succeeded" && status.OperationID == journal.OperationID
		if err := cleanupCommittedArtifacts(removeMetadata); err != nil {
			logger.M().Warnw("[备份恢复] 已提交事务清理未完成，将在下次启动重试", "operationId", journal.OperationID, "error", err)
		}
		return nil
	}
	if pendingErr != nil {
		if journal.State == "rolled_back" {
			if err := cleanupRolledBackArtifactsWithoutPending(journal); err != nil {
				return err
			}
			return nil
		}
		rollbackErr := rollbackRestoreJournal(journal)
		return errors.Join(errors.New("恢复日志存在但 pending 缺失，已尝试回滚，仍需进入 degraded"), rollbackErr)
	}
	if journal.State != "rolled_back" {
		if err := rollbackRestoreJournal(journal); err != nil {
			return err
		}
	}
	return resetRolledBackPending()
}

// RecoverInterruptedRestore 在 Runtime 启动前回滚上次未提交的逐文件事务。
func RecoverInterruptedRestore() error {
	backupOperationMu.Lock()
	defer backupOperationMu.Unlock()
	if err := validateRestoreStorageLocked(); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return recoverInterruptedRestore()
}

// HasCommittedRestore 报告是否存在已越过提交点、等待 Runtime 发布成功的恢复事务。
func HasCommittedRestore() (bool, error) {
	backupOperationMu.Lock()
	defer backupOperationMu.Unlock()
	if err := validateRestoreStorageLocked(); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	journal, err := readRestoreJournal()
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if journal.State != "committed" {
		return false, nil
	}
	pending, pendingErr := readPendingRestore()
	if pendingErr == nil && pending.OperationID != journal.OperationID {
		return false, errors.New("已提交 journal 与 pending 的 operationID 不一致")
	}
	if pendingErr != nil && !errors.Is(pendingErr, os.ErrNotExist) {
		return false, pendingErr
	}
	return true, nil
}

// HasRunnableScheduledRestore 报告是否存在尚未改动 data、可由 Runtime 安全续跑的恢复任务。
func runnableScheduledRestoreOperationIDLocked() (string, error) {
	pending, err := readPendingRestore()
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if pending.OperationID == "" || (pending.Phase != "scheduled" && pending.Phase != "prepared") {
		return "", nil
	}
	status := GetRestoreStatus()
	if status.OperationID != "" && status.OperationID != pending.OperationID {
		return "", errors.New("可续跑恢复任务的 pending/status operationID 不一致")
	}
	if _, _, err = inspectBackupArchive(filepath.Join(restoreDir(), restoreSourceName)); err != nil {
		return "", fmt.Errorf("可续跑恢复任务的源文件无效: %w", err)
	}
	if pending.Phase == "prepared" {
		if pending.SafetyBackupName == "" {
			return "", errors.New("prepared 恢复任务缺少安全备份名")
		}
		if _, err = backupArchivePathLocked(pending.SafetyBackupName); err != nil {
			return "", fmt.Errorf("prepared 恢复任务的安全备份无效: %w", err)
		}
	}
	journal, journalErr := readRestoreJournal()
	if errors.Is(journalErr, os.ErrNotExist) {
		return pending.OperationID, nil
	}
	if errors.Is(journalErr, errEmptyRestoreJournal) {
		if err = discardEmptyRestoreJournal(); err != nil {
			return "", err
		}
		return pending.OperationID, nil
	}
	if journalErr != nil {
		return "", journalErr
	}
	if journal.OperationID != pending.OperationID {
		return "", errors.New("可续跑恢复任务的 pending/journal operationID 不一致")
	}
	// A fully rolled back transaction must only run again after an explicit
	// user retry; never re-enqueue it automatically at startup.
	if journal.State == "rolled_back" {
		return "", nil
	}
	return pending.OperationID, nil
}

// RunnableScheduledRestoreOperationID 返回经完整校验、可安全续跑的恢复 operationID。
func RunnableScheduledRestoreOperationID() (string, error) {
	backupOperationMu.Lock()
	defer backupOperationMu.Unlock()
	if err := validateRestoreStorageLocked(); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return runnableScheduledRestoreOperationIDLocked()
}

// HasRunnableScheduledRestore 报告是否存在尚未改动 data、可由 Runtime 安全续跑的恢复任务。
func HasRunnableScheduledRestore() (bool, error) {
	backupOperationMu.Lock()
	defer backupOperationMu.Unlock()
	if err := validateRestoreStorageLocked(); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	operationID, err := runnableScheduledRestoreOperationIDLocked()
	return operationID != "", err
}

func prepareRolledBackRetry(operationID string) error {
	journal, err := readRestoreJournal()
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if journal.OperationID != operationID {
		return errors.New("重试任务与旧 journal 的 operationID 不一致")
	}
	if journal.State != "rolled_back" {
		return fmt.Errorf("旧 journal 状态为 %q，不能开始重试", journal.State)
	}
	if err = restoreRemoveAll(filepath.Join(restoreDir(), restoreRollbackName)); err != nil {
		return err
	}
	if err = restoreRemoveAll(filepath.Join(restoreDir(), restoreStagingName)); err != nil {
		return err
	}
	if err = restoreRemove(restoreJournalPath()); err != nil {
		return err
	}
	return nil
}

// ApplyScheduledRestore 在数据库关闭后应用已准备的待恢复文件。
func ApplyScheduledRestore() (resultErr error) {
	backupOperationMu.Lock()
	defer backupOperationMu.Unlock()
	if err := validateRestoreStorageLocked(); err != nil {
		return err
	}
	if err := recoverInterruptedRestore(); err != nil {
		return err
	}
	pending, err := readPendingRestore()
	if errors.Is(err, os.ErrNotExist) {
		return errors.New("没有待执行的恢复任务")
	}
	if err != nil {
		return err
	}
	if pending.OperationID == "" {
		return errors.New("恢复任务缺少 operationID")
	}
	if pending.Phase == "applied" {
		journal, journalErr := readRestoreJournal()
		if journalErr == nil && journal.OperationID == pending.OperationID && journal.State == "applied" {
			return nil
		}
		return errors.New("pending 显示 applied，但 journal 无法确认同一事务")
	}
	if pending.Phase != "prepared" || pending.SafetyBackupName == "" {
		return fmt.Errorf("恢复任务尚未完成安全备份，当前阶段为 %q", pending.Phase)
	}
	if err = prepareRolledBackRetry(pending.OperationID); err != nil {
		return err
	}
	var journal *restoreJournal
	defer func() {
		if resultErr != nil {
			state := "rolling_back"
			message := "应用备份失败，等待回滚: " + resultErr.Error()
			if journal != nil && journal.State == "rolled_back" {
				state = "rolled_back"
				message = "应用备份失败，已回滚: " + resultErr.Error()
			}
			_ = setRestoreStatus(RestoreStatus{State: state, OperationID: pending.OperationID, SourceName: pending.SourceName, SafetyBackupName: pending.SafetyBackupName, Message: message})
		}
	}()
	source := filepath.Join(restoreDir(), restoreSourceName)
	info, entries, err := inspectBackupArchive(source)
	if err != nil {
		return err
	}
	if !info.Restorable {
		return errors.New("排队备份已不再满足恢复条件")
	}
	if processPath := firstProcessOwnedRestorePath(entries); processPath != "" {
		return fmt.Errorf("备份包含进程级占用文件 %s", processPath)
	}
	if err = validateBackupData(source); err != nil {
		return err
	}
	if err = ensureDiskSpace(restoreDir(), info.UncompressedSize); err != nil {
		return err
	}
	staging := filepath.Join(restoreDir(), restoreStagingName)
	if err = restoreRemoveAll(staging); err != nil {
		return fmt.Errorf("清理暂存目录: %w", err)
	}
	if err = restoreRemoveAll(filepath.Join(restoreDir(), restoreRollbackName)); err != nil {
		return fmt.Errorf("清理回滚目录: %w", err)
	}
	if err = os.MkdirAll(staging, 0o700); err != nil {
		return err
	}
	if err = extractBackupTo(source, staging); err != nil {
		return err
	}
	journal, err = buildRestoreJournal(pending.OperationID, staging)
	if err != nil {
		return err
	}
	if err = createRestoreJournal(journal); err != nil {
		return err
	}
	pending.Phase = "applying"
	if err = writeJSONAtomic(restorePendingPath(), pending); err != nil {
		return err
	}
	_ = setRestoreStatus(RestoreStatus{State: "applying", OperationID: pending.OperationID, SourceName: pending.SourceName, SafetyBackupName: pending.SafetyBackupName})
	if err = applyRestoreJournal(journal); err != nil {
		if journal.State == "rolled_back" {
			_ = resetRolledBackPending()
		}
		return err
	}
	pending.Phase = "applied"
	if err = writeJSONAtomic(restorePendingPath(), pending); err != nil {
		return err
	}
	return nil
}

// CommitScheduledRestore 写入唯一不可逆的提交记录，不负责发布成功状态。
func CommitScheduledRestore() error {
	backupOperationMu.Lock()
	defer backupOperationMu.Unlock()
	if err := validateRestoreStorageLocked(); err != nil {
		return err
	}
	pending, err := readPendingRestore()
	if errors.Is(err, os.ErrNotExist) {
		status := GetRestoreStatus()
		if status.State == "succeeded" {
			return nil
		}
		return errors.New("提交恢复时 pending 缺失")
	}
	if err != nil {
		return err
	}
	journal, err := readRestoreJournal()
	if err != nil {
		return err
	}
	if pending.OperationID == "" || pending.OperationID != journal.OperationID {
		return errors.New("提交恢复时 operationID 不一致")
	}
	if journal.State == "committed" {
		return nil
	}
	if pending.Phase != "applied" || journal.State != "applied" {
		return fmt.Errorf("恢复任务尚未应用完成，pending=%s journal=%s", pending.Phase, journal.State)
	}
	if err = appendJournalState(journal, "committed"); err != nil {
		return fmt.Errorf("提交记录持久化结果不确定，拒绝继续发布: %w", err)
	}
	pending.Phase = "committed"
	if updateErr := writeJSONAtomic(restorePendingPath(), pending); updateErr != nil {
		logger.M().Warnw("[备份恢复] 恢复已提交，但 pending 更新失败", "operationId", pending.OperationID, "error", updateErr)
	}
	if cleanupErr := cleanupCommittedArtifacts(false); cleanupErr != nil {
		logger.M().Warnw("[备份恢复] 恢复已提交，但事务清理未完成，将在下次启动重试", "operationId", pending.OperationID, "error", cleanupErr)
	}
	return nil
}

// MarkScheduledRestoreSucceeded 在新 Runtime 发布后写入成功状态并清理事务元数据。
func MarkScheduledRestoreSucceeded() error {
	backupOperationMu.Lock()
	defer backupOperationMu.Unlock()
	if err := validateRestoreStorageLocked(); err != nil {
		return err
	}
	journal, err := readRestoreJournal()
	if errors.Is(err, os.ErrNotExist) {
		if GetRestoreStatus().State == "succeeded" {
			return nil
		}
		return errors.New("标记恢复成功时 journal 缺失")
	}
	if err != nil {
		return err
	}
	if journal.State != "committed" {
		return fmt.Errorf("标记恢复成功时 journal 状态为 %q", journal.State)
	}
	pending, pendingErr := readPendingRestore()
	if pendingErr != nil && !errors.Is(pendingErr, os.ErrNotExist) {
		return pendingErr
	}
	status := GetRestoreStatus()
	if pendingErr == nil {
		if pending.OperationID != journal.OperationID {
			return errors.New("标记恢复成功时 operationID 不一致")
		}
		status = RestoreStatus{
			State:            "succeeded",
			OperationID:      pending.OperationID,
			SourceName:       pending.SourceName,
			SafetyBackupName: pending.SafetyBackupName,
		}
	} else {
		if status.OperationID != journal.OperationID {
			return errors.New("pending 缺失且 status 无法确认已提交 operationID")
		}
		status.State = "succeeded"
		status.Message = ""
	}
	if err = setRestoreStatus(status); err != nil {
		return err
	}
	if cleanupErr := cleanupCommittedArtifacts(true); cleanupErr != nil {
		logger.M().Warnw("[备份恢复] 成功状态已发布，但事务清理未完成，将在下次启动重试", "operationId", journal.OperationID, "error", cleanupErr)
	}
	return nil
}

func RollbackScheduledRestore(message string) error {
	backupOperationMu.Lock()
	defer backupOperationMu.Unlock()
	if err := validateRestoreStorageLocked(); err != nil {
		return err
	}
	pending, pendingErr := readPendingRestore()
	journal, journalErr := readRestoreJournal()
	if journalErr == nil {
		if pendingErr == nil && pending.OperationID != journal.OperationID {
			return errors.New("回滚恢复时 operationID 不一致")
		}
		if journal.State == "committed" {
			return errors.New("恢复事务已经提交，拒绝回滚")
		}
		if journal.State != "rolled_back" {
			if err := rollbackRestoreJournal(journal); err != nil {
				return err
			}
		}
	} else if errors.Is(journalErr, errEmptyRestoreJournal) {
		if pendingErr != nil || (pending.Phase != "scheduled" && pending.Phase != "prepared") {
			return errors.New("恢复日志不含完整 begin，但 pending 阶段无法确认未改动 data")
		}
		if err := discardEmptyRestoreJournal(); err != nil {
			return err
		}
	} else if !errors.Is(journalErr, os.ErrNotExist) {
		return journalErr
	} else if pendingErr == nil && pending.Phase != "scheduled" && pending.Phase != "prepared" {
		return fmt.Errorf("恢复任务处于 %q 但 journal 缺失，拒绝宣称已回滚", pending.Phase)
	}
	if pendingErr != nil {
		return pendingErr
	}
	if pending.OperationID == "" {
		return errors.New("回滚恢复任务缺少 operationID")
	}
	if pending.SafetyBackupName == "" {
		pending.Phase = "scheduled"
	} else {
		pending.Phase = "prepared"
	}
	if err := writeJSONAtomic(restorePendingPath(), pending); err != nil {
		return err
	}
	return setRestoreStatus(RestoreStatus{
		State:            "rolled_back",
		OperationID:      pending.OperationID,
		SourceName:       pending.SourceName,
		SafetyBackupName: pending.SafetyBackupName,
		Message:          message,
	})
}
