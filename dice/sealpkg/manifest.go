package sealpkg

import (
	"archive/zip"
	"errors"
	"io"
	"os"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/pelletier/go-toml/v2"
)

// ParseManifestFile 解析 manifest.toml 文件
func ParseManifestFile(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseManifest(data)
}

// ParseManifest 解析 manifest.toml 内容
func ParseManifest(data []byte) (*Manifest, error) {
	var manifest Manifest
	if err := toml.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	// 验证必填字段
	if manifest.Package.ID == "" {
		return nil, errors.New("manifest 缺少 package.id")
	}

	// 验证 package.id 路径安全性
	if err := validatePackageID(manifest.Package.ID); err != nil {
		return nil, err
	}

	if manifest.Package.Name == "" {
		return nil, errors.New("manifest 缺少 package.name")
	}
	if manifest.Package.Version == "" {
		return nil, errors.New("manifest 缺少 package.version")
	}

	// 验证版本号格式
	if _, err := semver.NewVersion(manifest.Package.Version); err != nil {
		return nil, errors.New("manifest 的 package.version 不是有效的语义化版本: " + err.Error())
	}

	// 初始化空字段
	if manifest.Dependencies == nil {
		manifest.Dependencies = make(map[string]string)
	}
	if manifest.Config == nil {
		manifest.Config = make(map[string]ConfigSchema)
	}

	return &manifest, nil
}

// ParseManifestFromZip 从 ZIP 文件中解析 manifest
func ParseManifestFromZip(pkgPath string) (*Manifest, error) {
	reader, err := zip.OpenReader(pkgPath)
	if err != nil {
		return nil, errors.New("无法打开扩展包: " + err.Error())
	}
	defer reader.Close()

	for _, f := range reader.File {
		if f.Name == ManifestFile {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			data, err := io.ReadAll(rc)
			if err != nil {
				return nil, err
			}
			return ParseManifest(data)
		}
	}

	return nil, errors.New("扩展包中未找到 " + ManifestFile)
}

// ValidateManifest 验证 manifest 的完整性
func ValidateManifest(manifest *Manifest) []string {
	var issues []string

	// 检查基本信息
	if manifest.Package.ID == "" {
		issues = append(issues, "缺少 package.id")
	}
	if manifest.Package.Name == "" {
		issues = append(issues, "缺少 package.name")
	}
	if manifest.Package.Version == "" {
		issues = append(issues, "缺少 package.version")
	} else if _, err := semver.NewVersion(manifest.Package.Version); err != nil {
		issues = append(issues, "package.version 不是有效的语义化版本")
	}

	// 检查依赖版本约束格式
	for depID, constraint := range manifest.Dependencies {
		if _, err := semver.NewConstraint(constraint); err != nil {
			issues = append(issues, "依赖 "+depID+" 的版本约束无效: "+constraint)
		}
	}

	// 检查配置 schema
	for key, schema := range manifest.Config {
		if schema.Type == "" {
			issues = append(issues, "配置项 "+key+" 缺少 type 字段")
		}
		switch schema.Type {
		case "string", "integer", "number", "boolean", "array", "object":
			// 有效类型
		default:
			issues = append(issues, "配置项 "+key+" 的类型无效: "+schema.Type)
		}
	}

	return issues
}

// CheckSealVersion 检查海豹版本兼容性
func CheckSealVersion(manifest *Manifest, currentVersion string) error {
	if manifest.Package.Seal.MinVersion == "" && manifest.Package.Seal.MaxVersion == "" {
		return nil // 没有版本要求
	}

	current, err := semver.NewVersion(currentVersion)
	if err != nil {
		return nil // 无法解析当前版本，跳过检查
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

// CheckDependencyConstraint 检查单个依赖版本是否满足约束
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

// validatePackageID 验证 package.id 的格式和安全性
// 要求格式为 "作者/包名"，同时防止路径遍历攻击
func validatePackageID(id string) error {
	// 检查是否为空或仅包含空白字符
	if strings.TrimSpace(id) == "" {
		return errors.New("package.id 不能为空或仅包含空白字符")
	}

	// 检查是否包含反斜杠（Windows 路径分隔符）
	if strings.Contains(id, "\\") {
		return errors.New("package.id 不能包含反斜杠 (\\)")
	}

	// 检查是否包含相对路径符号
	if strings.Contains(id, "..") {
		return errors.New("package.id 不能包含相对路径符号 (..)")
	}

	// 检查是否为绝对路径（Unix 根路径）
	if len(id) > 0 && id[0] == '/' {
		return errors.New("package.id 不能以 / 开头")
	}

	// Windows 盘符检查 (C:, D: 等)
	if len(id) >= 2 && id[1] == ':' {
		return errors.New("package.id 不能包含 Windows 盘符")
	}

	// 检查特殊字符
	if strings.ContainsAny(id, "<>:|?*\"") {
		return errors.New("package.id 包含非法字符")
	}

	// 验证格式为 "作者/包名"
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		return errors.New("package.id 必须为 '作者/包名' 格式")
	}

	author := strings.TrimSpace(parts[0])
	name := strings.TrimSpace(parts[1])

	if author == "" {
		return errors.New("package.id 的作者部分不能为空")
	}
	if name == "" {
		return errors.New("package.id 的包名部分不能为空")
	}

	return nil
}
