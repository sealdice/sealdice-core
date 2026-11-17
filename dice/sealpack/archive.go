package sealpack

import (
	"archive/zip"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	MaxManifestSize         int64  = 1 << 20
	maxArchiveEntries              = 10_000
	maxArchiveDirectorySize uint64 = 16 << 20
	maxArchivePathSize             = 1 << 10
	archiveCopyBufferSize          = 32 << 10
)

const (
	directoryHeaderSignature    = 0x02014b50
	directoryEndSignature       = 0x06054b50
	directory64EndSignature     = 0x06064b50
	directory64LocatorSignature = 0x07064b50
	directoryDigitalSignature   = 0x05054b50
)

const (
	directoryHeaderLen    = 46
	directoryEndLen       = 22
	directory64EndLen     = 56
	directory64LocatorLen = 20
)

var allowedArchiveRoots = map[string]struct{}{
	InfoFile:    {},
	"README.md": {},
	"assets":    {},
	"decks":     {},
	"helpdoc":   {},
	"reply":     {},
	"scripts":   {},
	"templates": {},
}

// ExtractProgressFunc is called before the next extracted chunk is written.
type ExtractProgressFunc func(written, total uint64) error

type openedArchive struct {
	file   *os.File
	reader *zip.Reader
	info   *ArchiveInfo
}

func (a *openedArchive) Close() error {
	return a.file.Close()
}

// InspectArchive validates a .sealpack archive and returns its manifest and file list.
func InspectArchive(pkgPath string) (*ArchiveInfo, error) {
	return InspectArchiveContext(context.Background(), pkgPath)
}

// InspectArchiveContext is the cancellable form of InspectArchive.
func InspectArchiveContext(ctx context.Context, pkgPath string) (*ArchiveInfo, error) {
	archive, err := openArchive(ctx, pkgPath)
	if err != nil {
		return nil, err
	}
	defer archive.Close()
	return archive.info, nil
}

// ExtractArchive validates and extracts a .sealpack archive to destDir.
func ExtractArchive(pkgPath, destDir string) (*ArchiveInfo, error) {
	return ExtractArchiveContext(context.Background(), pkgPath, destDir)
}

// ExtractArchiveContext is the cancellable form of ExtractArchive.
func ExtractArchiveContext(ctx context.Context, pkgPath, destDir string) (*ArchiveInfo, error) {
	return ExtractArchiveWithProgress(ctx, pkgPath, destDir, nil)
}

// ExtractArchiveWithProgress extracts an archive and reports bounded-memory progress.
func ExtractArchiveWithProgress(ctx context.Context, pkgPath, destDir string, progress ExtractProgressFunc) (*ArchiveInfo, error) {
	archive, err := openArchive(ctx, pkgPath)
	if err != nil {
		return nil, err
	}
	defer archive.Close()

	root := filepath.Clean(destDir)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}

	buffer := make([]byte, archiveCopyBufferSize)
	var totalWritten uint64
	for _, file := range archive.reader.File {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		normalized, isDir, err := normalizeArchiveEntryName(file.Name)
		if err != nil {
			return nil, err
		}
		if normalized == "" {
			continue
		}

		targetPath := filepath.Join(root, filepath.FromSlash(normalized))
		if targetPath != root && !strings.HasPrefix(targetPath, root+string(os.PathSeparator)) {
			return nil, fmt.Errorf("archive entry escapes install root: %s", file.Name)
		}

		if isDir {
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return nil, err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return nil, err
		}

		if err := extractArchiveFile(ctx, file, targetPath, buffer, &totalWritten, archive.info.UncompressedSize, progress); err != nil {
			return nil, err
		}
	}

	return archive.info, nil
}

func openArchive(ctx context.Context, pkgPath string) (*openedArchive, error) {
	file, err := os.Open(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open extension package: %w", err)
	}
	closeOnError := true
	defer func() {
		if closeOnError {
			_ = file.Close()
		}
	}()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("extension package must be a regular file")
	}
	if preflightErr := preflightArchiveDirectory(ctx, file, info.Size()); preflightErr != nil {
		return nil, preflightErr
	}

	reader, err := zip.NewReader(file, info.Size())
	if err != nil {
		return nil, fmt.Errorf("failed to open extension package: %w", err)
	}
	archiveInfo, err := inspectArchiveFiles(ctx, reader.File)
	if err != nil {
		return nil, err
	}

	closeOnError = false
	return &openedArchive{file: file, reader: reader, info: archiveInfo}, nil
}

func inspectArchiveFiles(ctx context.Context, files []*zip.File) (*ArchiveInfo, error) {
	if len(files) > maxArchiveEntries {
		return nil, fmt.Errorf("archive contains too many entries: %d (maximum %d)", len(files), maxArchiveEntries)
	}

	seen := make(map[string]struct{}, len(files))
	archiveInfo := &ArchiveInfo{Files: make([]string, 0, len(files))}
	var manifestData []byte
	readmePresent := false

	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if len(file.Name) > maxArchivePathSize {
			return nil, fmt.Errorf("archive entry path is too long: %d bytes", len(file.Name))
		}
		normalized, isDir, err := normalizeArchiveEntryName(file.Name)
		if err != nil {
			return nil, err
		}
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			return nil, fmt.Errorf("duplicate archive entry: %s", normalized)
		}
		seen[normalized] = struct{}{}

		if validationErr := validateArchiveTopLevelPath(normalized); validationErr != nil {
			return nil, validationErr
		}
		if normalized == "README.md" {
			readmePresent = true
		}
		if isDir {
			continue
		}
		if math.MaxUint64-archiveInfo.UncompressedSize < file.UncompressedSize64 {
			return nil, errors.New("archive uncompressed size overflows uint64")
		}
		archiveInfo.UncompressedSize += file.UncompressedSize64
		archiveInfo.Files = append(archiveInfo.Files, normalized)

		if normalized != InfoFile {
			continue
		}
		if file.UncompressedSize64 > uint64(MaxManifestSize) {
			return nil, fmt.Errorf("package %s exceeds %d MiB", InfoFile, MaxManifestSize/(1024*1024))
		}
		manifestData, err = readArchiveEntryLimited(file, MaxManifestSize)
		if err != nil {
			return nil, err
		}
	}

	if manifestData == nil {
		return nil, fmt.Errorf("package archive is missing %s", InfoFile)
	}
	manifest, err := ParseManifest(manifestData)
	if err != nil {
		return nil, err
	}
	if manifest.Store.Readme == "" && readmePresent {
		manifest.Store.Readme = "README.md"
	}
	archiveInfo.Manifest = manifest
	return archiveInfo, nil
}

func readArchiveEntryLimited(file *zip.File, limit int64) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	data, readErr := io.ReadAll(io.LimitReader(rc, limit+1))
	closeErr := rc.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("package %s exceeds %d MiB", InfoFile, limit/(1024*1024))
	}
	return data, nil
}

func extractArchiveFile(ctx context.Context, file *zip.File, targetPath string, buffer []byte, totalWritten *uint64, totalSize uint64, progress ExtractProgressFunc) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	mode := file.Mode()
	if mode.Perm() == 0 {
		mode = 0o644
	}
	out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return err
	}
	succeeded := false
	defer func() {
		_ = out.Close()
		if !succeeded {
			_ = os.Remove(targetPath)
		}
	}()

	var entryWritten uint64
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, readErr := rc.Read(buffer)
		if n > 0 {
			chunk := uint64(n)
			if entryWritten > file.UncompressedSize64 || chunk > file.UncompressedSize64-entryWritten {
				return fmt.Errorf("archive entry %s exceeds its declared size", file.Name)
			}
			if *totalWritten > totalSize || chunk > totalSize-*totalWritten {
				return errors.New("archive output exceeds its declared total size")
			}
			if progress != nil {
				if err := progress(*totalWritten+chunk, totalSize); err != nil {
					return err
				}
			}
			written, writeErr := out.Write(buffer[:n])
			if writeErr != nil {
				return writeErr
			}
			if written != n {
				return io.ErrShortWrite
			}
			entryWritten += chunk
			*totalWritten += chunk
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	if entryWritten != file.UncompressedSize64 {
		return fmt.Errorf("archive entry %s size mismatch: expected %d, got %d", file.Name, file.UncompressedSize64, entryWritten)
	}
	if err := out.Close(); err != nil {
		return err
	}
	succeeded = true
	return nil
}

func preflightArchiveDirectory(ctx context.Context, file *os.File, size int64) error {
	if size < directoryEndLen {
		return zip.ErrFormat
	}
	tailSize := int64(directoryEndLen + math.MaxUint16)
	if size < tailSize {
		tailSize = size
	}
	tail := make([]byte, tailSize)
	if _, err := file.ReadAt(tail, size-tailSize); err != nil && err != io.EOF {
		return err
	}

	endIndex := -1
	for i := len(tail) - directoryEndLen; i >= 0; i-- {
		if binary.LittleEndian.Uint32(tail[i:i+4]) != directoryEndSignature {
			continue
		}
		commentLen := int(binary.LittleEndian.Uint16(tail[i+20 : i+22]))
		if i+directoryEndLen+commentLen <= len(tail) {
			endIndex = i
			break
		}
	}
	if endIndex < 0 {
		return zip.ErrFormat
	}

	end := tail[endIndex : endIndex+directoryEndLen]
	directoryEndOffset := size - tailSize + int64(endIndex)
	diskNumber := binary.LittleEndian.Uint16(end[4:6])
	directoryDisk := binary.LittleEndian.Uint16(end[6:8])
	recordsOnDisk := uint64(binary.LittleEndian.Uint16(end[8:10]))
	records := uint64(binary.LittleEndian.Uint16(end[10:12]))
	directorySize := uint64(binary.LittleEndian.Uint32(end[12:16]))
	directoryOffset := uint64(binary.LittleEndian.Uint32(end[16:20]))

	// archive/zip also treats 0xffff as a ZIP64 directory-size marker for
	// compatibility, so the preflight must follow the same interpretation.
	if records == math.MaxUint16 || directorySize == math.MaxUint16 || directorySize == math.MaxUint32 || directoryOffset == math.MaxUint32 {
		locatorOffset := directoryEndOffset - directory64LocatorLen
		if locatorOffset < 0 {
			return zip.ErrFormat
		}
		locator := make([]byte, directory64LocatorLen)
		if _, err := file.ReadAt(locator, locatorOffset); err != nil {
			return err
		}
		if binary.LittleEndian.Uint32(locator[:4]) != directory64LocatorSignature ||
			binary.LittleEndian.Uint32(locator[4:8]) != 0 || binary.LittleEndian.Uint32(locator[16:20]) != 1 {
			return zip.ErrFormat
		}
		zip64Offset := binary.LittleEndian.Uint64(locator[8:16])
		if zip64Offset > math.MaxInt64 {
			return zip.ErrFormat
		}
		zip64End := make([]byte, directory64EndLen)
		if _, err := file.ReadAt(zip64End, int64(zip64Offset)); err != nil {
			return err
		}
		if binary.LittleEndian.Uint32(zip64End[:4]) != directory64EndSignature {
			return zip.ErrFormat
		}
		if binary.LittleEndian.Uint32(zip64End[16:20]) != 0 || binary.LittleEndian.Uint32(zip64End[20:24]) != 0 {
			return errors.New("multi-disk ZIP archives are not supported")
		}
		recordsOnDisk = binary.LittleEndian.Uint64(zip64End[24:32])
		records = binary.LittleEndian.Uint64(zip64End[32:40])
		directorySize = binary.LittleEndian.Uint64(zip64End[40:48])
		directoryOffset = binary.LittleEndian.Uint64(zip64End[48:56])
		directoryEndOffset = int64(zip64Offset)
	} else if diskNumber != 0 || directoryDisk != 0 {
		return errors.New("multi-disk ZIP archives are not supported")
	}

	if recordsOnDisk != records {
		return errors.New("multi-disk ZIP archives are not supported")
	}
	if records > maxArchiveEntries {
		return fmt.Errorf("archive contains too many entries: %d (maximum %d)", records, maxArchiveEntries)
	}
	if directorySize > maxArchiveDirectorySize {
		return fmt.Errorf("archive central directory exceeds %d MiB", maxArchiveDirectorySize/(1024*1024))
	}
	if directorySize > math.MaxInt64 || directoryOffset > math.MaxInt64 {
		return zip.ErrFormat
	}

	baseOffset := directoryEndOffset - int64(directorySize) - int64(directoryOffset)
	if baseOffset < 0 {
		return zip.ErrFormat
	}
	directoryStart := baseOffset + int64(directoryOffset)
	if baseOffset > 0 {
		var signature [4]byte
		if _, err := file.ReadAt(signature[:], int64(directoryOffset)); err == nil && binary.LittleEndian.Uint32(signature[:]) == directoryHeaderSignature {
			directoryStart = int64(directoryOffset)
		}
	}
	return scanCentralDirectory(ctx, file, directoryStart, directorySize, records)
}

func scanCentralDirectory(ctx context.Context, file *os.File, start int64, size, expectedRecords uint64) error {
	var offset uint64
	var records uint64
	header := make([]byte, directoryHeaderLen)
	for offset < size {
		if err := ctx.Err(); err != nil {
			return err
		}
		if size-offset < 4 {
			return zip.ErrFormat
		}
		if _, err := file.ReadAt(header[:4], start+int64(offset)); err != nil {
			return err
		}
		signature := binary.LittleEndian.Uint32(header[:4])
		if signature == directoryDigitalSignature {
			var lengthBytes [2]byte
			if size-offset < 6 {
				return zip.ErrFormat
			}
			if _, err := file.ReadAt(lengthBytes[:], start+int64(offset)+4); err != nil {
				return err
			}
			offset += 6 + uint64(binary.LittleEndian.Uint16(lengthBytes[:]))
			break
		}
		if signature != directoryHeaderSignature || size-offset < directoryHeaderLen {
			return zip.ErrFormat
		}
		if _, err := file.ReadAt(header, start+int64(offset)); err != nil {
			return err
		}
		nameLen := uint64(binary.LittleEndian.Uint16(header[28:30]))
		extraLen := uint64(binary.LittleEndian.Uint16(header[30:32]))
		commentLen := uint64(binary.LittleEndian.Uint16(header[32:34]))
		if nameLen > maxArchivePathSize {
			return fmt.Errorf("archive entry path is too long: %d bytes", nameLen)
		}
		recordSize := uint64(directoryHeaderLen) + nameLen + extraLen + commentLen
		if recordSize > size-offset {
			return zip.ErrFormat
		}
		offset += recordSize
		records++
		if records > maxArchiveEntries {
			return fmt.Errorf("archive contains too many entries: %d (maximum %d)", records, maxArchiveEntries)
		}
	}
	if offset != size || records != expectedRecords {
		return zip.ErrFormat
	}
	return nil
}

func normalizeArchiveEntryName(name string) (string, bool, error) {
	trimmed := strings.TrimSpace(strings.ReplaceAll(name, `\\`, `/`))
	if trimmed == "" || trimmed == "." {
		return "", false, nil
	}
	isDir := strings.HasSuffix(trimmed, "/")
	trimmed = strings.TrimSuffix(trimmed, "/")
	if trimmed == "" || trimmed == "." {
		return "", false, nil
	}
	if err := validateRelativePackagePath(trimmed); err != nil {
		return "", false, fmt.Errorf("invalid archive entry %q: %w", name, err)
	}
	if path.Base(trimmed) == ManifestFile && trimmed != InfoFile {
		return "", false, fmt.Errorf("legacy or nested manifest is not supported: %s", name)
	}
	return trimmed, isDir, nil
}

func validateArchiveTopLevelPath(normalized string) error {
	root := normalized
	if idx := strings.IndexByte(normalized, '/'); idx >= 0 {
		root = normalized[:idx]
	}
	if _, ok := allowedArchiveRoots[root]; !ok {
		return fmt.Errorf("archive entry is outside the package root layout: %s", normalized)
	}
	return nil
}
