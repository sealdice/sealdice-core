package sealpkg

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// PermissionError 权限违规错误
type PermissionError struct {
	PackageID  string `json:"packageId"`
	Permission string `json:"permission"`
	Requested  string `json:"requested"`
	Message    string `json:"message"`
}

func (e *PermissionError) Error() string {
	return "包 " + e.PackageID + " 没有 " + e.Permission + " 权限: " + e.Message
}

// Sandbox 扩展包权限沙箱
type Sandbox struct {
	PackageID   string
	Permissions *Permissions
	BasePath    string // 包安装路径
}

// NewSandbox 创建包沙箱
func NewSandbox(pkgID string, permissions *Permissions, basePath string) *Sandbox {
	return &Sandbox{
		PackageID:   pkgID,
		Permissions: permissions,
		BasePath:    basePath,
	}
}

// NewSandboxFromInstance 从包实例创建沙箱
func NewSandboxFromInstance(inst *Instance) *Sandbox {
	return &Sandbox{
		PackageID:   inst.Manifest.Package.ID,
		Permissions: &inst.Manifest.Permissions,
		BasePath:    inst.InstallPath,
	}
}

// CheckNetworkPermission 检查网络访问权限
func (s *Sandbox) CheckNetworkPermission(targetURL string) error {
	if !s.Permissions.Network {
		return &PermissionError{
			PackageID:  s.PackageID,
			Permission: "network",
			Requested:  targetURL,
			Message:    "扩展包未声明网络访问权限",
		}
	}

	// 如果设置了域名白名单，检查目标域名
	if len(s.Permissions.NetworkHosts) > 0 {
		parsedURL, err := url.Parse(targetURL)
		if err != nil {
			return &PermissionError{
				PackageID:  s.PackageID,
				Permission: "network",
				Requested:  targetURL,
				Message:    "无效的URL",
			}
		}

		host := parsedURL.Hostname()
		allowed := false
		for _, allowedHost := range s.Permissions.NetworkHosts {
			if s.matchHost(host, allowedHost) {
				allowed = true
				break
			}
		}

		if !allowed {
			return &PermissionError{
				PackageID:  s.PackageID,
				Permission: "network_hosts",
				Requested:  host,
				Message:    "域名不在白名单中",
			}
		}
	}

	return nil
}

// matchHost 匹配域名（支持通配符）
func (s *Sandbox) matchHost(host, pattern string) bool {
	// 支持 *.example.com 格式
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:] // .example.com
		return strings.HasSuffix(host, suffix) || host == pattern[2:]
	}
	return host == pattern
}

// CheckFileReadPermission 检查文件读取权限
func (s *Sandbox) CheckFileReadPermission(path string) error {
	relPath, err := s.normalizeToRelativePath(path)
	if err != nil {
		return err
	}

	if !s.matchPathPatterns(relPath, s.Permissions.FileRead) {
		return &PermissionError{
			PackageID:  s.PackageID,
			Permission: "file_read",
			Requested:  relPath,
			Message:    "路径不在允许读取的范围内",
		}
	}

	return nil
}

// CheckFileWritePermission 检查文件写入权限
func (s *Sandbox) CheckFileWritePermission(path string) error {
	relPath, err := s.normalizeToRelativePath(path)
	if err != nil {
		return err
	}

	// 默认只允许写入 _userdata 目录
	allowedPaths := s.Permissions.FileWrite
	if len(allowedPaths) == 0 {
		allowedPaths = []string{UserDataDir + "/*"}
	}

	if !s.matchPathPatterns(relPath, allowedPaths) {
		return &PermissionError{
			PackageID:  s.PackageID,
			Permission: "file_write",
			Requested:  relPath,
			Message:    "路径不在允许写入的范围内",
		}
	}

	return nil
}

// normalizeToRelativePath 将路径标准化为相对于包目录的路径
func (s *Sandbox) normalizeToRelativePath(path string) (string, error) {
	cleanPath := filepath.Clean(path)

	if filepath.IsAbs(cleanPath) {
		basePath := filepath.Clean(s.BasePath)
		if !strings.HasPrefix(cleanPath, basePath+string(os.PathSeparator)) &&
			cleanPath != basePath {
			return "", &PermissionError{
				PackageID:  s.PackageID,
				Permission: "file_access",
				Requested:  path,
				Message:    "不允许访问包目录外的路径",
			}
		}
		relPath, err := filepath.Rel(basePath, cleanPath)
		if err != nil {
			return "", err
		}
		cleanPath = relPath
	}

	// 检查路径穿越
	if strings.HasPrefix(cleanPath, "..") ||
		strings.Contains(cleanPath, string(os.PathSeparator)+".."+string(os.PathSeparator)) ||
		strings.HasSuffix(cleanPath, string(os.PathSeparator)+"..") {
		return "", &PermissionError{
			PackageID:  s.PackageID,
			Permission: "file_access",
			Requested:  path,
			Message:    "不允许路径穿越",
		}
	}

	return cleanPath, nil
}

// matchPathPatterns 检查路径是否匹配任一模式
func (s *Sandbox) matchPathPatterns(path string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}

	for _, pattern := range patterns {
		if s.matchPathPattern(path, pattern) {
			return true
		}
	}
	return false
}

// matchPathPattern 检查路径是否匹配模式
func (s *Sandbox) matchPathPattern(path, pattern string) bool {
	matched, err := filepath.Match(pattern, path)
	if err == nil && matched {
		return true
	}

	// 处理 ** 通配符
	if strings.Contains(pattern, "**") {
		parts := strings.Split(pattern, "**")
		if len(parts) == 2 {
			prefix := parts[0]
			suffix := parts[1]

			if prefix != "" && !strings.HasPrefix(path, prefix) {
				return false
			}

			if suffix != "" {
				suffix = strings.TrimPrefix(suffix, "/")
				suffix = strings.TrimPrefix(suffix, string(os.PathSeparator))
				if suffix != "" && !strings.HasSuffix(path, suffix) {
					return false
				}
			}

			return true
		}
	}

	// 处理目录通配符 "dir/*"
	if strings.HasSuffix(pattern, "/*") {
		dir := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(path, dir+string(os.PathSeparator)) ||
			strings.HasPrefix(path, dir+"/")
	}

	return false
}

// CheckDangerousPermission 检查危险操作权限
func (s *Sandbox) CheckDangerousPermission(operation string) error {
	if !s.Permissions.Dangerous {
		return &PermissionError{
			PackageID:  s.PackageID,
			Permission: "dangerous",
			Requested:  operation,
			Message:    "扩展包未声明危险操作权限",
		}
	}
	return nil
}

// CheckHTTPServerPermission 检查HTTP服务权限
func (s *Sandbox) CheckHTTPServerPermission() error {
	if !s.Permissions.HTTPServer {
		return &PermissionError{
			PackageID:  s.PackageID,
			Permission: "http_server",
			Requested:  "start_server",
			Message:    "扩展包未声明HTTP服务权限",
		}
	}
	return nil
}

// CheckIPCPermission 检查扩展间通信权限
func (s *Sandbox) CheckIPCPermission(targetPackageID string) error {
	if len(s.Permissions.IPC) == 0 {
		return &PermissionError{
			PackageID:  s.PackageID,
			Permission: "ipc",
			Requested:  targetPackageID,
			Message:    "扩展包未声明IPC权限",
		}
	}

	for _, allowedPkg := range s.Permissions.IPC {
		if allowedPkg == targetPackageID || allowedPkg == "*" {
			return nil
		}
	}

	return &PermissionError{
		PackageID:  s.PackageID,
		Permission: "ipc",
		Requested:  targetPackageID,
		Message:    "不允许与目标扩展包通信",
	}
}

// SandboxedFS 沙箱化文件系统访问
type SandboxedFS struct {
	sandbox *Sandbox
}

// NewSandboxedFS 创建沙箱化文件系统
func NewSandboxedFS(sandbox *Sandbox) *SandboxedFS {
	return &SandboxedFS{sandbox: sandbox}
}

// ReadFile 沙箱化的文件读取
func (fs *SandboxedFS) ReadFile(path string) ([]byte, error) {
	if err := fs.sandbox.CheckFileReadPermission(path); err != nil {
		return nil, err
	}
	return os.ReadFile(fs.resolvePath(path))
}

// WriteFile 沙箱化的文件写入
func (fs *SandboxedFS) WriteFile(path string, data []byte, perm os.FileMode) error {
	if err := fs.sandbox.CheckFileWritePermission(path); err != nil {
		return err
	}

	fullPath := fs.resolvePath(path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(fullPath, data, perm)
}

// Stat 沙箱化的文件状态查询
func (fs *SandboxedFS) Stat(path string) (os.FileInfo, error) {
	if err := fs.sandbox.CheckFileReadPermission(path); err != nil {
		return nil, err
	}
	return os.Stat(fs.resolvePath(path))
}

// ReadDir 沙箱化的目录读取
func (fs *SandboxedFS) ReadDir(path string) ([]os.DirEntry, error) {
	if err := fs.sandbox.CheckFileReadPermission(path); err != nil {
		return nil, err
	}
	return os.ReadDir(fs.resolvePath(path))
}

// Mkdir 沙箱化的目录创建
func (fs *SandboxedFS) Mkdir(path string, perm os.FileMode) error {
	if err := fs.sandbox.CheckFileWritePermission(path); err != nil {
		return err
	}
	return os.MkdirAll(fs.resolvePath(path), perm)
}

// Remove 沙箱化的文件删除
func (fs *SandboxedFS) Remove(path string) error {
	if err := fs.sandbox.CheckFileWritePermission(path); err != nil {
		return err
	}
	return os.Remove(fs.resolvePath(path))
}

// resolvePath 解析路径到完整路径
func (fs *SandboxedFS) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(fs.sandbox.BasePath, path)
}

// SandboxedHTTP 沙箱化HTTP客户端
type SandboxedHTTP struct {
	sandbox *Sandbox
	client  *http.Client
}

// NewSandboxedHTTP 创建沙箱化HTTP客户端
func NewSandboxedHTTP(sandbox *Sandbox) *SandboxedHTTP {
	return &SandboxedHTTP{
		sandbox: sandbox,
		client:  &http.Client{},
	}
}

// Get 沙箱化的HTTP GET请求
func (h *SandboxedHTTP) Get(url string) (*http.Response, error) {
	if err := h.sandbox.CheckNetworkPermission(url); err != nil {
		return nil, err
	}
	return h.client.Get(url)
}

// Post 沙箱化的HTTP POST请求
func (h *SandboxedHTTP) Post(url, contentType string, body []byte) (*http.Response, error) {
	if err := h.sandbox.CheckNetworkPermission(url); err != nil {
		return nil, err
	}
	return h.client.Post(url, contentType, strings.NewReader(string(body)))
}

// Do 沙箱化的HTTP请求
func (h *SandboxedHTTP) Do(req *http.Request) (*http.Response, error) {
	if err := h.sandbox.CheckNetworkPermission(req.URL.String()); err != nil {
		return nil, err
	}
	return h.client.Do(req)
}
