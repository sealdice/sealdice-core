package sealpkg

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/Masterminds/semver/v3"
	"github.com/pelletier/go-toml/v2"
)

// ParseManifestFile parses an info.toml file from disk.
func ParseManifestFile(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseManifest(data)
}

// ParseManifest parses info.toml contents.
func ParseManifest(data []byte) (*Manifest, error) {
	var manifest Manifest
	decoder := toml.NewDecoder(bytes.NewReader(data)).DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return nil, err
	}

	if manifest.Package.ID == "" {
		return nil, errors.New("info 缺少 package.id")
	}
	if err := ValidatePackageID(manifest.Package.ID); err != nil {
		return nil, err
	}
	if manifest.Package.Name == "" {
		return nil, errors.New("info 缺少 package.name")
	}
	if manifest.Package.Version == "" {
		return nil, errors.New("info 缺少 package.version")
	}
	if _, err := semver.NewVersion(manifest.Package.Version); err != nil {
		return nil, errors.New("info 的 package.version 不是有效的语义化版本: " + err.Error())
	}

	if manifest.Dependencies == nil {
		manifest.Dependencies = make(map[string]string)
	}
	if manifest.Config == nil {
		manifest.Config = make(map[string]ConfigSchema)
	}

	if err := validateDependencies(manifest.Dependencies); err != nil {
		return nil, err
	}
	if err := validateContents(manifest.Contents); err != nil {
		return nil, err
	}
	if err := validateStoreInfo(manifest.Store); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// ParseManifestFromZip parses package metadata from a .sealpkg archive.
func ParseManifestFromZip(pkgPath string) (*Manifest, error) {
	archiveInfo, err := InspectArchive(pkgPath)
	if err != nil {
		return nil, err
	}
	return archiveInfo.Manifest, nil
}

// ValidateManifest returns a list of validation issues for a parsed manifest.
func ValidateManifest(manifest *Manifest) []string {
	var issues []string

	if manifest.Package.ID == "" {
		issues = append(issues, "缺少 package.id")
	} else if err := ValidatePackageID(manifest.Package.ID); err != nil {
		issues = append(issues, err.Error())
	}
	if manifest.Package.Name == "" {
		issues = append(issues, "缺少 package.name")
	}
	if manifest.Package.Version == "" {
		issues = append(issues, "缺少 package.version")
	} else if _, err := semver.NewVersion(manifest.Package.Version); err != nil {
		issues = append(issues, "package.version 不是有效的语义化版本")
	}

	for depID, constraint := range manifest.Dependencies {
		if err := ValidatePackageID(depID); err != nil {
			issues = append(issues, fmt.Sprintf("依赖 %s 的包 ID 无效: %v", depID, err))
			continue
		}
		if _, err := semver.NewConstraint(constraint); err != nil {
			issues = append(issues, "依赖 "+depID+" 的版本约束无效: "+constraint)
		}
	}

	for key, schema := range manifest.Config {
		if schema.Type == "" {
			issues = append(issues, "配置项 "+key+" 缺少 type 字段")
			continue
		}
		switch schema.Type {
		case "string", "integer", "number", "boolean", "array", "object":
		default:
			issues = append(issues, "配置项 "+key+" 的类型无效: "+schema.Type)
		}
	}

	if err := validateContents(manifest.Contents); err != nil {
		issues = append(issues, err.Error())
	}
	if err := validateStoreInfo(manifest.Store); err != nil {
		issues = append(issues, err.Error())
	}

	return issues
}

// CheckSealVersion checks whether the current SealDice version satisfies the package requirement.
func CheckSealVersion(manifest *Manifest, currentVersion string) error {
	if manifest.Package.Seal.MinVersion == "" && manifest.Package.Seal.MaxVersion == "" {
		return nil
	}

	current, err := semver.NewVersion(currentVersion)
	if err != nil {
		return fmt.Errorf("当前海豹版本号无效: %w", err)
	}

	if manifest.Package.Seal.MinVersion != "" {
		minVer, err := semver.NewVersion(manifest.Package.Seal.MinVersion)
		if err == nil && current.LessThan(minVer) {
			return errors.New("此扩展包需要海豹版本 >= " + manifest.Package.Seal.MinVersion)
		}
	}

	if manifest.Package.Seal.MaxVersion != "" {
		maxVer, err := semver.NewVersion(manifest.Package.Seal.MaxVersion)
		if err == nil && current.GreaterThan(maxVer) {
			return errors.New("此扩展包需要海豹版本 <= " + manifest.Package.Seal.MaxVersion)
		}
	}

	return nil
}

// CheckDependencyConstraint checks whether a version satisfies a dependency constraint.
func CheckDependencyConstraint(constraint, version string) (bool, error) {
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return false, errors.New("无效的版本约束: " + constraint)
	}

	v, err := semver.NewVersion(version)
	if err != nil {
		return false, errors.New("无效的版本号: " + version)
	}

	return c.Check(v), nil
}

func validateDependencies(dependencies map[string]string) error {
	for depID, constraint := range dependencies {
		if err := ValidatePackageID(depID); err != nil {
			return fmt.Errorf("依赖 %s 的包 ID 无效: %w", depID, err)
		}
		if _, err := semver.NewConstraint(constraint); err != nil {
			return fmt.Errorf("依赖 %s 的版本约束无效: %w", depID, err)
		}
	}
	return nil
}

func validateContents(contents Contents) error {
	checks := map[string][]string{
		"scripts":   contents.Scripts,
		"decks":     contents.Decks,
		"reply":     contents.Reply,
		"helpdoc":   contents.Helpdoc,
		"templates": contents.Templates,
	}

	for dir, patterns := range checks {
		for _, pattern := range patterns {
			if err := validateContentPattern(dir, pattern); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateContentPattern(dir, pattern string) error {
	if err := validateRelativePackagePath(pattern); err != nil {
		return fmt.Errorf("contents.%s 路径无效: %w", dir, err)
	}
	normalized := filepath.ToSlash(pattern)
	if !strings.HasPrefix(normalized, dir+"/") {
		return fmt.Errorf("contents.%s 只能引用 %s/ 目录下的文件", dir, dir)
	}
	return nil
}

func validateStoreInfo(store StoreInfo) error {
	paths := []string{store.Readme, store.Icon, store.Banner}
	paths = append(paths, store.Screenshots...)
	for _, item := range paths {
		if item == "" {
			continue
		}
		if err := validateRelativePackagePath(item); err != nil {
			return fmt.Errorf("store 路径无效: %w", err)
		}
	}
	return nil
}

func validateRelativePackagePath(path string) error {
	normalized := filepath.ToSlash(strings.TrimSpace(path))
	if normalized == "" {
		return nil
	}
	if strings.HasPrefix(normalized, "/") || strings.HasPrefix(normalized, "//") {
		return errors.New("路径不能以根目录开头")
	}
	if len(normalized) >= 2 && normalized[1] == ':' && unicode.IsLetter(rune(normalized[0])) {
		return errors.New("路径不能使用 Windows 盘符")
	}
	if strings.HasPrefix(normalized, "src/") || normalized == "src" {
		return errors.New("不支持 src/ 目录布局")
	}
	for _, part := range strings.Split(normalized, "/") {
		if part == "" {
			return errors.New("路径不能包含空的目录段")
		}
		if part == "." || part == ".." {
			return errors.New("路径不能包含 . 或 .. 路径段")
		}
	}
	return nil
}
