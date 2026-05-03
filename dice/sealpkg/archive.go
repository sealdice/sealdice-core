package sealpkg

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
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

// InspectArchive validates a .sealpkg archive and returns its manifest and file list.
func InspectArchive(pkgPath string) (*ArchiveInfo, error) {
	reader, err := zip.OpenReader(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open extension package: %w", err)
	}
	defer reader.Close()

	return inspectArchiveFiles(reader.File)
}

// ExtractArchive validates and extracts a .sealpkg archive to destDir.
func ExtractArchive(pkgPath, destDir string) (*ArchiveInfo, error) {
	archiveInfo, err := InspectArchive(pkgPath)
	if err != nil {
		return nil, err
	}

	reader, err := zip.OpenReader(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open extension package: %w", err)
	}
	defer reader.Close()

	root := filepath.Clean(destDir)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}

	for _, file := range reader.File {
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
			if mkdirErr := os.MkdirAll(targetPath, 0o755); mkdirErr != nil {
				return nil, mkdirErr
			}
			continue
		}

		if mkdirErr := os.MkdirAll(filepath.Dir(targetPath), 0o755); mkdirErr != nil {
			return nil, mkdirErr
		}

		rc, err := file.Open()
		if err != nil {
			return nil, err
		}

		mode := file.Mode()
		if mode.Perm() == 0 {
			mode = 0o644
		}
		out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
		if err != nil {
			rc.Close()
			return nil, err
		}
		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			rc.Close()
			return nil, err
		}
		if err := out.Close(); err != nil {
			rc.Close()
			return nil, err
		}
		if err := rc.Close(); err != nil {
			return nil, err
		}
	}

	return archiveInfo, nil
}

func inspectArchiveFiles(files []*zip.File) (*ArchiveInfo, error) {
	seen := make(map[string]struct{}, len(files))
	archiveInfo := &ArchiveInfo{Files: make([]string, 0, len(files))}
	var manifestData []byte
	readmePresent := false

	for _, file := range files {
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

		if validateErr := validateArchiveTopLevelPath(normalized); validateErr != nil {
			return nil, validateErr
		}

		if normalized == "README.md" {
			readmePresent = true
		}
		if isDir {
			continue
		}
		archiveInfo.Files = append(archiveInfo.Files, normalized)

		if normalized != InfoFile {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, err
		}
		manifestData, err = io.ReadAll(rc)
		if closeErr := rc.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
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
