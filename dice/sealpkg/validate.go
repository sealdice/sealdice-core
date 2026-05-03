package sealpkg

import (
	"errors"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

var packageIDSegmentPattern = regexp.MustCompile(`^[\p{L}\p{N}_-]+$`)

// ValidatePackageID validates package IDs in the canonical author/package form.
func ValidatePackageID(id string) error {
	if id == "" {
		return errors.New("包 ID 不能为空")
	}
	if strings.TrimSpace(id) != id {
		return errors.New("包 ID 不能包含首尾空白")
	}
	if strings.Contains(id, `\`) {
		return errors.New("包 ID 只能使用 / 作为分隔符")
	}
	if strings.HasPrefix(id, "/") || strings.HasSuffix(id, "/") {
		return errors.New("包 ID 不能以 / 开头或结尾")
	}

	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		return errors.New("包 ID 格式错误，应为 author/package")
	}

	if err := validatePackageIDSegment("作者", parts[0]); err != nil {
		return err
	}
	if err := validatePackageIDSegment("包名", parts[1]); err != nil {
		return err
	}

	return nil
}

func validatePackageIDSegment(label string, segment string) error {
	if segment == "" {
		return errors.New(label + "不能为空")
	}
	if segment == "." || segment == ".." {
		return errors.New(label + "不能为 . 或 ..")
	}
	if strings.Contains(segment, "/") {
		return errors.New(label + "不能包含 /")
	}
	if !packageIDSegmentPattern.MatchString(segment) {
		return errors.New(label + "只能包含字母、数字、连字符和下划线")
	}
	if utf8.RuneCountInString(segment) > 64 {
		return errors.New(label + "过长（最大 64 个字符）")
	}
	return nil
}

// ParsePackageID splits a package ID into author and package name.
func ParsePackageID(id string) (string, string, error) {
	if err := ValidatePackageID(id); err != nil {
		return "", "", err
	}
	parts := strings.Split(id, "/")
	return parts[0], parts[1], nil
}

// PackageIDToSafePath converts a package ID to a nested filesystem path.
func PackageIDToSafePath(pkgID string) string {
	author, pkg, err := ParsePackageID(pkgID)
	if err != nil {
		return filepath.FromSlash(pkgID)
	}
	return filepath.Join(author, pkg)
}

// PackageIDToSafeDir is kept for existing callers and now returns the nested path.
func PackageIDToSafeDir(pkgID string) string {
	return PackageIDToSafePath(pkgID)
}

// PackageVersionToFileName converts a version to the canonical archive filename.
func PackageVersionToFileName(version string) string {
	return version + Extension
}

// PackageSourceFileName converts a package ID and version to the canonical source archive filename.
func PackageSourceFileName(pkgID, version string) string {
	_, pkg, err := ParsePackageID(pkgID)
	if err != nil {
		return PackageVersionToFileName(version)
	}
	return pkg + "@" + version + Extension
}

// FileNameToPackageVersion converts an archive filename back to its version.
func FileNameToPackageVersion(fileName string) string {
	return strings.TrimSuffix(fileName, Extension)
}
