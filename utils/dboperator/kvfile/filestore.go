package kvfile

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cespare/xxhash/v2"
)

// ErrBodyMissing reports that a committed body file does not exist on disk.
var ErrBodyMissing = errors.New("kvfile: body not found")

// DefaultThreshold is the raw-size boundary between database-stored small
// values and file-backed large values. Values at or below this size go to the
// repository database; larger values are stored as files under a per-namespace
// folder and referenced from a pointer row.
const DefaultThreshold int64 = 512 * 1024

// FileStore manages the folder-based body storage for large plugin KV values.
//
// Layout:
//
//	root/<namespace>/<sha1(key)>.bin
//
// Writes are buffered to a <name>.<rand>.tmp file and committed with an atomic
// rename, so a body is never observable half-written. Keys are addressed by
// their SHA-1 hash to avoid path traversal via hostile keys.
type FileStore struct {
	root string
}

// NewFileStore creates a FileStore rooted at root, creating it if needed.
func NewFileStore(root string) (*FileStore, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("kvfile: file store root is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("kvfile: create root: %w", err)
	}
	return &FileStore{root: filepath.Clean(root)}, nil
}

// sanitizeNamespace validates a plugin namespace for safe use as a directory
// name and a ref/row scope.
func sanitizeNamespace(ns string) (string, error) {
	ns = strings.TrimSpace(ns)
	if ns == "" || ns == "." || ns == ".." {
		return "", errors.New("kvfile: invalid namespace")
	}
	if strings.ContainsAny(ns, `/\`) || strings.Contains(ns, "\x00") {
		return "", errors.New("kvfile: namespace contains unsafe characters")
	}
	return ns, nil
}

// hashKey derives the stable body file base name for a key. It mirrors how
// gormkv hashes keys, so both layers address a key the same way.
func hashKey(key string) string {
	return strconv.FormatUint(xxhash.Sum64String(key), 16)
}

func (f *FileStore) bodyPath(ns, fileName string) string {
	return filepath.Join(f.root, ns, fileName)
}

func randomSuffix() string {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return "0"
	}
	return hex.EncodeToString(buf)
}

// PutBody streams body into a temporary file under the namespace and commits
// it atomically (rename). It returns the number of bytes written. On any
// failure the temporary file is removed and no body is visible.
func (f *FileStore) PutBody(_ context.Context, ns, key string, body io.Reader) (int64, error) {
	dir := filepath.Join(f.root, ns)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, fmt.Errorf("kvfile: mkdir namespace: %w", err)
	}
	dst := filepath.Join(dir, hashKey(key)+".bin")
	tmp := fmt.Sprintf("%s.%s.tmp", dst, randomSuffix())

	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return 0, fmt.Errorf("kvfile: create temp body: %w", err)
	}
	abort := func() {
		_ = out.Close()
		_ = os.Remove(tmp)
	}

	size, err := io.Copy(out, body)
	if err != nil {
		abort()
		return 0, fmt.Errorf("kvfile: write body: %w", err)
	}
	if err := out.Sync(); err != nil {
		abort()
		return 0, fmt.Errorf("kvfile: sync body: %w", err)
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return 0, fmt.Errorf("kvfile: close body: %w", err)
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return 0, fmt.Errorf("kvfile: commit body: %w", err)
	}
	return size, nil
}

// PutBodyTemp streams body to a unique temporary file and returns its path and
// size WITHOUT committing it as the value's body. Callers must follow up with
// CommitBody (to publish) or DiscardTemp (on failure); the value's previous
// body stays intact until CommitBody runs.
func (f *FileStore) PutBodyTemp(ns string, body io.Reader) (string, int64, error) {
	dir := filepath.Join(f.root, ns)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", 0, fmt.Errorf("kvfile: mkdir namespace: %w", err)
	}
	tmp := filepath.Join(dir, fmt.Sprintf(".pending-%s.tmp", randomSuffix()))
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return "", 0, fmt.Errorf("kvfile: create temp body: %w", err)
	}
	size, err := io.Copy(out, body)
	if err != nil {
		_ = out.Close()
		_ = os.Remove(tmp)
		return "", 0, fmt.Errorf("kvfile: write body: %w", err)
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return "", 0, fmt.Errorf("kvfile: close body: %w", err)
	}
	return tmp, size, nil
}

// CommitBody atomically publishes a temp body as the committed body of key.
func (f *FileStore) CommitBody(ns, key, temp string) error {
	dst := filepath.Join(f.root, ns, hashKey(key)+".bin")
	if err := os.Rename(temp, dst); err != nil {
		_ = os.Remove(temp)
		return fmt.Errorf("kvfile: commit body: %w", err)
	}
	return nil
}

// DiscardTemp removes an uncommitted temp body. Missing files are not errors.
func (f *FileStore) DiscardTemp(temp string) error {
	err := os.Remove(temp)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// ReadTempBytes reads an uncommitted temp body fully. Used to fold a small
// streamed value back into the database store.
func (f *FileStore) ReadTempBytes(temp string) ([]byte, error) {
	data, err := os.ReadFile(temp)
	if err != nil {
		return nil, fmt.Errorf("kvfile: read temp body: %w", err)
	}
	return data, nil
}

// OpenBody opens the committed body file for a key. Callers must close the
// returned file. A missing body reports ErrBodyMissing.
func (f *FileStore) OpenBody(ns, key string) (*os.File, error) {
	file, err := os.Open(f.bodyPath(ns, hashKey(key)+".bin"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrBodyMissing
		}
		return nil, fmt.Errorf("kvfile: open body: %w", err)
	}
	return file, nil
}

// RemoveNamespaceDir removes the namespace body directory (and everything
// under it). A missing directory is not an error.
func (f *FileStore) RemoveNamespaceDir(ns string) error {
	dir := filepath.Join(f.root, ns)
	err := os.RemoveAll(dir)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// DeleteBody removes the committed body for a key. Missing bodies are not an
// error.
func (f *FileStore) DeleteBody(ns, key string) error {
	err := os.Remove(f.bodyPath(ns, hashKey(key)+".bin"))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
