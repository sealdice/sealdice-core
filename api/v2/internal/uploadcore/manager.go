package uploadcore

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
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
	ErrIncomplete      = errors.New("upload chunks are incomplete")
	ErrHashMismatch    = errors.New("upload file hash mismatch")
)

type Session struct {
	SessionID      string
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
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}
	expectedChunks := int((fileSize + chunkSize - 1) / chunkSize)
	sessionID := buildSessionID(filename, fileHash, fileSize)
	tempDir := filepath.Join(m.rootDir, sessionID)
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return nil, err
	}

	session := &Session{
		SessionID:      sessionID,
		Filename:       filename,
		FileSize:       fileSize,
		FileHash:       strings.ToLower(strings.TrimSpace(fileHash)),
		ChunkSize:      chunkSize,
		ExpectedChunks: expectedChunks,
		UploadedChunks: map[int]bool{},
		TempDir:        tempDir,
	}
	for index := 0; index < expectedChunks; index++ {
		chunkPath := filepath.Join(tempDir, ChunkFilename(index))
		if info, err := os.Stat(chunkPath); err == nil && info.Size() > 0 {
			session.UploadedChunks[index] = true
		}
	}

	m.mu.Lock()
	m.sessions[sessionID] = session
	m.mu.Unlock()
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
	if index < 0 || index >= session.ExpectedChunks {
		return nil, ErrChunkOutOfRange
	}
	if len(body) == 0 {
		return nil, ErrChunkEmpty
	}

	chunkPath := filepath.Join(session.TempDir, ChunkFilename(index))
	if err := os.WriteFile(chunkPath, body, 0o644); err != nil {
		return nil, err
	}

	m.mu.Lock()
	session.UploadedChunks[index] = true
	m.mu.Unlock()
	return session, nil
}

func (m *Manager) UploadedBytes(session *Session) int64 {
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
	result := make([]int, 0, len(session.UploadedChunks))
	for index := range session.UploadedChunks {
		result = append(result, index)
	}
	sort.Ints(result)
	return result
}

func (m *Manager) Complete(sessionID string, dstPath string) (*Session, error) {
	session, err := m.Get(sessionID)
	if err != nil {
		return nil, err
	}
	if len(session.UploadedChunks) != session.ExpectedChunks {
		return nil, ErrIncomplete
	}

	dst, err := os.Create(dstPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = dst.Close() }()

	hasher := m.newHash()
	for index := 0; index < session.ExpectedChunks; index++ {
		chunkPath := filepath.Join(session.TempDir, ChunkFilename(index))
		content, readErr := os.ReadFile(chunkPath)
		if readErr != nil {
			return nil, readErr
		}
		if _, writeErr := dst.Write(content); writeErr != nil {
			return nil, writeErr
		}
		if _, hashErr := hasher.Write(content); hashErr != nil {
			return nil, hashErr
		}
	}

	if actual := hex.EncodeToString(hasher.Sum(nil)); actual != session.FileHash {
		return nil, ErrHashMismatch
	}

	m.Cleanup(sessionID)
	return session, nil
}

func (m *Manager) Cleanup(sessionID string) {
	m.mu.Lock()
	session := m.sessions[sessionID]
	delete(m.sessions, sessionID)
	m.mu.Unlock()
	if session != nil {
		_ = os.RemoveAll(session.TempDir)
	}
}

func ChunkFilename(index int) string {
	return fmt.Sprintf("%06d.part", index)
}

func buildSessionID(filename string, fileHash string, fileSize int64) string {
	sum := sha256.Sum256([]byte(filename + ":" + strings.ToLower(fileHash) + ":" + strconv.FormatInt(fileSize, 10)))
	return hex.EncodeToString(sum[:16])
}
