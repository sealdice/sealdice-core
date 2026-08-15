package uploadcore

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const DefaultChunkSize int64 = 4 * 1024 * 1024

var (
	ErrSessionNotFound = errors.New("upload session not found")
	ErrChunkOutOfRange = errors.New("upload chunk index out of range")
	ErrChunkEmpty      = errors.New("upload chunk body is empty")
	ErrChunkSize       = errors.New("upload chunk size mismatch")
	ErrIncomplete      = errors.New("upload chunks are incomplete")
	ErrFileSize        = errors.New("upload file size mismatch")
	ErrHashMismatch    = errors.New("upload file hash mismatch")
)

type Session struct {
	mu             sync.RWMutex
	closed         bool
	SessionID      string
	Scope          string
	Filename       string
	FileSize       int64
	FileHash       string
	ChunkSize      int64
	ExpectedChunks int
	UploadedChunks map[int]bool
	TempDir        string
}

type Manager struct {
	rootDir  string
	newHash  func() hash.Hash
	mu       sync.Mutex
	sessions map[string]*Session
}

func NewManager(rootDir string) *Manager {
	return &Manager{
		rootDir:  rootDir,
		newHash:  sha256.New,
		sessions: map[string]*Session{},
	}
}

func (m *Manager) Init(filename string, fileSize int64, fileHash string, chunkSize int64) (*Session, error) {
	return m.InitWithScope("", filename, fileSize, fileHash, chunkSize)
}

func (m *Manager) InitWithScope(scope string, filename string, fileSize int64, fileHash string, chunkSize int64) (*Session, error) {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}
	expectedChunks := int((fileSize + chunkSize - 1) / chunkSize)
	sessionID := buildSessionID(scope, filename, fileHash, fileSize)
	m.mu.Lock()
	defer m.mu.Unlock()
	if session := m.sessions[sessionID]; session != nil {
		return session, nil
	}

	tempDir := filepath.Join(m.rootDir, sessionID)
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return nil, err
	}

	session := &Session{
		SessionID:      sessionID,
		Scope:          strings.TrimSpace(scope),
		Filename:       filename,
		FileSize:       fileSize,
		FileHash:       strings.ToLower(strings.TrimSpace(fileHash)),
		ChunkSize:      chunkSize,
		ExpectedChunks: expectedChunks,
		UploadedChunks: map[int]bool{},
		TempDir:        tempDir,
	}
	for index := range expectedChunks {
		chunkPath := filepath.Join(tempDir, ChunkFilename(index))
		if info, err := os.Stat(chunkPath); err == nil && info.Size() == expectedChunkSize(session, index) {
			session.UploadedChunks[index] = true
		}
	}

	m.sessions[sessionID] = session
	return session, nil
}

func (m *Manager) Get(sessionID string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session := m.sessions[sessionID]
	if session == nil {
		return nil, ErrSessionNotFound
	}
	return session, nil
}

func (m *Manager) SaveChunk(sessionID string, index int, body []byte) (*Session, error) {
	session, err := m.Get(sessionID)
	if err != nil {
		return nil, err
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.closed {
		return nil, ErrSessionNotFound
	}
	if index < 0 || index >= session.ExpectedChunks {
		return nil, ErrChunkOutOfRange
	}
	if len(body) == 0 {
		return nil, ErrChunkEmpty
	}
	if int64(len(body)) != expectedChunkSize(session, index) {
		return nil, ErrChunkSize
	}

	chunkPath := filepath.Join(session.TempDir, ChunkFilename(index))
	if err := os.WriteFile(chunkPath, body, 0o644); err != nil {
		return nil, err
	}

	session.UploadedChunks[index] = true
	return session, nil
}

func (m *Manager) UploadedBytes(session *Session) int64 {
	session.mu.RLock()
	defer session.mu.RUnlock()
	var total int64
	for index := range session.UploadedChunks {
		chunkPath := filepath.Join(session.TempDir, ChunkFilename(index))
		if info, err := os.Stat(chunkPath); err == nil {
			total += info.Size()
		}
	}
	return total
}

func (m *Manager) SortedUploadedChunks(session *Session) []int {
	session.mu.RLock()
	defer session.mu.RUnlock()
	result := make([]int, 0, len(session.UploadedChunks))
	for index := range session.UploadedChunks {
		result = append(result, index)
	}
	sort.Ints(result)
	return result
}

func (m *Manager) IsChunkUploaded(session *Session, index int) bool {
	session.mu.RLock()
	defer session.mu.RUnlock()
	return session.UploadedChunks[index]
}

func (m *Manager) Complete(sessionID string, dstPath string) (*Session, error) {
	session, err := m.CompleteWithoutCleanup(sessionID, dstPath)
	if err != nil {
		return nil, err
	}
	m.Cleanup(sessionID)
	return session, nil
}

func (m *Manager) CompleteWithoutCleanup(sessionID string, dstPath string) (*Session, error) {
	session, getErr := m.Get(sessionID)
	if getErr != nil {
		return nil, getErr
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.closed {
		return nil, ErrSessionNotFound
	}
	if len(session.UploadedChunks) != session.ExpectedChunks {
		return nil, ErrIncomplete
	}

	dstDir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return nil, err
	}
	staged, err := os.CreateTemp(dstDir, "."+filepath.Base(dstPath)+".upload-*")
	if err != nil {
		return nil, err
	}
	stagedPath := staged.Name()
	keepStaged := false
	defer func() {
		_ = staged.Close()
		if !keepStaged {
			_ = os.Remove(stagedPath)
		}
	}()
	if info, statErr := os.Stat(dstPath); statErr == nil {
		if chmodErr := staged.Chmod(info.Mode().Perm()); chmodErr != nil {
			return nil, chmodErr
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return nil, statErr
	} else if chmodErr := staged.Chmod(0o644); chmodErr != nil {
		return nil, chmodErr
	}

	hasher := m.newHash()
	writer := io.MultiWriter(staged, hasher)
	var written int64
	for index := range session.ExpectedChunks {
		chunkPath := filepath.Join(session.TempDir, ChunkFilename(index))
		chunk, readErr := os.Open(chunkPath)
		if readErr != nil {
			return nil, readErr
		}
		copied, copyErr := io.Copy(writer, chunk)
		closeErr := chunk.Close()
		if copyErr != nil {
			return nil, copyErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		written += copied
	}
	if written != session.FileSize {
		return nil, ErrFileSize
	}

	if actual := hex.EncodeToString(hasher.Sum(nil)); actual != session.FileHash {
		return nil, ErrHashMismatch
	}
	if err := staged.Sync(); err != nil {
		return nil, err
	}
	if err := staged.Close(); err != nil {
		return nil, err
	}
	if err := replaceStagedFile(stagedPath, dstPath); err != nil {
		return nil, err
	}
	keepStaged = true

	return session, nil
}

func (m *Manager) Cleanup(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session := m.sessions[sessionID]
	delete(m.sessions, sessionID)
	if session != nil {
		session.mu.Lock()
		defer session.mu.Unlock()
		session.closed = true
		_ = os.RemoveAll(session.TempDir)
	}
}

func ChunkFilename(index int) string {
	return fmt.Sprintf("%06d.part", index)
}

func expectedChunkSize(session *Session, index int) int64 {
	remaining := session.FileSize - int64(index)*session.ChunkSize
	if remaining < session.ChunkSize {
		return remaining
	}
	return session.ChunkSize
}

// ReplaceFileAtomic 用同目录 staging 文件替换 dstPath：
// 目标不存在时直接 rename；目标存在时先备份、rename 新文件、失败回滚。
func ReplaceFileAtomic(stagedPath string, dstPath string) error {
	return replaceStagedFile(stagedPath, dstPath)
}

func replaceStagedFile(stagedPath string, dstPath string) error {
	if _, err := os.Stat(dstPath); errors.Is(err, os.ErrNotExist) {
		return os.Rename(stagedPath, dstPath)
	} else if err != nil {
		return err
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

func buildSessionID(scope string, filename string, fileHash string, fileSize int64) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(scope) + ":" + filename + ":" + strings.ToLower(strings.TrimSpace(fileHash)) + ":" + strconv.FormatInt(fileSize, 10)))
	return hex.EncodeToString(sum[:16])
}
