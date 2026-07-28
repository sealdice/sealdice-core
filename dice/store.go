package dice

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver/v3"

	"sealdice-core/dice/sealpack"
	"sealdice-core/static"
	"sealdice-core/utils/crypto"
)

var (
	// OfficialStorePublicKey 官方商店公钥。
	OfficialStorePublicKey = ``

	officialStoreBackendBaseURL = "https://repo.sealdice.com"
)

type StoreBackendType string

const (
	StoreBackendTypeOfficial StoreBackendType = "official"
	StoreBackendTypeTrusted  StoreBackendType = "trusted"
	StoreBackendTypeExtra    StoreBackendType = "extra"
)

const storeBackendInfoCacheTTL = 5 * time.Minute
const maxStoreManifestSize int64 = 1 << 20

type StoreBackend struct {
	Url string `json:"url"`

	ID               string           `json:"id"`
	Name             string           `json:"name"`
	Type             StoreBackendType `json:"type"`
	Builtin          bool             `json:"builtin"`
	Official         bool             `json:"official"`
	Enabled          bool             `json:"enabled"`
	ProtocolVersions []string         `json:"protocolVersions"`
	Announcement     string           `json:"announcement"`
	Health           bool             `json:"health"`
	Sign             string           `json:"sign"`
}

type StorePackageDownload struct {
	URL           string            `json:"url"`
	ZipURL        string            `json:"zipUrl"`
	Hash          map[string]string `json:"hash"`
	Size          uint64            `json:"size"`
	ReleaseTime   uint64            `json:"releaseTime"`
	UpdateTime    uint64            `json:"updateTime"`
	DownloadCount uint64            `json:"downloadCount"`
}

type StorePackage struct {
	ID            string `json:"id"`
	FormatVersion string `json:"formatVersion,omitempty"`
	Version       string `json:"version"`
	FullID        string `json:"-"`

	Name         string                   `json:"name"`
	Authors      []string                 `json:"authors"`
	Description  string                   `json:"description"`
	License      string                   `json:"license"`
	Homepage     string                   `json:"homepage"`
	Repository   string                   `json:"repository"`
	Keywords     []string                 `json:"keywords"`
	Contents     []string                 `json:"contents"`
	Seal         sealpack.SealRequirement `json:"seal"`
	Dependencies map[string]string        `json:"dependencies"`
	StoreAssets  sealpack.StoreInfo       `json:"storeAssets"`

	Download  StorePackageDownload `json:"download"`
	Installed bool                 `json:"installed"`
}

type storeBackendInfoResponse struct {
	FormatVersion    string   `json:"formatVersion"`
	Name             string   `json:"name"`
	ProtocolVersions []string `json:"protocolVersions"`
	Announcement     string   `json:"announcement"`
	Sign             string   `json:"sign"`
}

type storeRecommendResponse struct {
	FormatVersion string          `json:"formatVersion"`
	Result        bool            `json:"result"`
	Data          []*StorePackage `json:"data"`
	Err           string          `json:"err"`
}

type storePageResponse struct {
	FormatVersion string            `json:"formatVersion"`
	Result        bool              `json:"result"`
	Data          *StorePackagePage `json:"data"`
	Err           string            `json:"err"`
}

type StoreQueryPageParams struct {
	Content  string `query:"content"`
	PageNum  int    `query:"pageNum"`
	PageSize int    `query:"pageSize"`
	Author   string `query:"author"`
	Name     string `query:"name"`
	Category string `query:"category"`
	SortBy   string `query:"sortBy"`
	Order    string `query:"order"`
}

type StorePackagePage struct {
	FormatVersion string          `json:"formatVersion"`
	Data          []*StorePackage `json:"data"`
	PageNum       int             `json:"pageNum"`
	PageSize      int             `json:"pageSize"`
	Next          bool            `json:"next"`
}

type StorePackageFileEntry struct {
	Path string `json:"path"`
	Size uint64 `json:"size"`
}

type StoreManager struct {
	lock             *sync.RWMutex
	parent           *Dice
	backend          *StoreBackend
	backendCacheKey  string
	backendFetchedAt time.Time
	packageCache     map[string]*StorePackage

	InstalledPlugins map[string]bool `json:"-" yaml:"-"`
	InstalledDecks   map[string]bool `json:"-" yaml:"-"`
	InstalledReplies map[string]bool `json:"-" yaml:"-"`
}

type storePackageFilesResponse struct {
	FormatVersion string                  `json:"formatVersion"`
	Result        bool                    `json:"result"`
	Data          []StorePackageFileEntry `json:"data"`
	Err           string                  `json:"err"`
}

type storeBackendTarget struct {
	rawURL      string
	id          string
	name        string
	backendType StoreBackendType
	cacheKey    string
}

var storeAllowedContents = map[string]struct{}{
	"scripts":   {},
	"decks":     {},
	"reply":     {},
	"helpdoc":   {},
	"templates": {},
}

func BuildStorePackageFullID(id, version string) string {
	return id + "@" + version
}

func ParseStorePackageFullID(fullID string) (string, string, error) {
	fullID = strings.TrimSpace(fullID)
	pos := strings.LastIndex(fullID, "@")
	if pos <= 0 || pos == len(fullID)-1 {
		return "", "", errors.New("无效的 fullId，必须满足 author/package@version 格式")
	}

	pkgID := fullID[:pos]
	version := fullID[pos+1:]
	if err := sealpack.ValidatePackageID(pkgID); err != nil {
		return "", "", err
	}
	if _, err := semver.NewVersion(version); err != nil {
		return "", "", fmt.Errorf("无效的版本号: %w", err)
	}
	return pkgID, version, nil
}

func decodeJSONCompatible(data []byte, target interface{}) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("JSON 中存在多余内容")
		}
		return err
	}
	return nil
}

func fetchStoreJSON[T any](requestURL string) (*T, error) {
	// #nosec G107 -- store backend URLs are user/admin-configured extension repository endpoints.
	resp, err := http.Get(requestURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s", string(respData))
	}

	var result T
	if err := decodeJSONCompatible(respData, &result); err != nil {
		return nil, fmt.Errorf("decode store response from %s: %w", requestURL, err)
	}
	return &result, nil
}

func normalizeConfiguredStoreBackendURL(urls []string) string {
	for _, rawURL := range urls {
		normalized := strings.TrimRight(strings.TrimSpace(rawURL), "/")
		if normalized != "" {
			return normalized
		}
	}
	return ""
}

func normalizeConfiguredStoreBackendURLs(urls []string) []string {
	seen := make(map[string]struct{}, len(urls))
	result := make([]string, 0, len(urls))
	for _, rawURL := range urls {
		normalized := strings.TrimRight(strings.TrimSpace(rawURL), "/")
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func containsStoreBackendURL(urls []string, rawURL string) bool {
	normalized := strings.TrimRight(strings.TrimSpace(rawURL), "/")
	for _, urlItem := range urls {
		if urlItem == normalized {
			return true
		}
	}
	return false
}

func removeStoreBackendURL(urls []string, rawURL string) []string {
	normalized := strings.TrimRight(strings.TrimSpace(rawURL), "/")
	result := make([]string, 0, len(urls))
	for _, urlItem := range urls {
		if urlItem == normalized {
			continue
		}
		result = append(result, urlItem)
	}
	return result
}

func officialStoreBackendURL() (string, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(officialStoreBackendBaseURL), "/")
	if baseURL == "" {
		return "", errors.New("官方扩展商店后端地址为空")
	}
	if strings.HasSuffix(baseURL, "/dice/api/store") {
		return baseURL, nil
	}
	return baseURL + "/dice/api/store", nil
}

func validateStoreBackendURL(rawURL string) (string, error) {
	rawURL = strings.TrimRight(strings.TrimSpace(rawURL), "/")
	if rawURL == "" {
		return "", errors.New("后端地址不能为空")
	}
	parsedURL, err := url.Parse(rawURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return "", errors.New("后端地址必须是绝对 URL")
	}
	return rawURL, nil
}

func buildStoreBackendResourceURL(baseURL string, segments ...string) (string, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if _, err := validateStoreBackendURL(baseURL); err != nil {
		return "", err
	}

	var builder strings.Builder
	builder.WriteString(baseURL)
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			return "", errors.New("商店路径参数不能为空")
		}
		builder.WriteByte('/')
		builder.WriteString(url.PathEscape(segment))
	}
	return builder.String(), nil
}

func resolveStoreDownloadURL(rawURL string, backendBaseURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", errors.New("下载地址不能为空")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if parsedURL.IsAbs() && parsedURL.Host != "" {
		return parsedURL.String(), nil
	}
	if backendBaseURL == "" {
		return "", errors.New("download.url 必须是绝对 URL")
	}

	baseURL, err := url.Parse(strings.TrimRight(strings.TrimSpace(backendBaseURL), "/") + "/")
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		return "", errors.New("商店后端地址无效，无法解析相对下载地址")
	}
	resolved := baseURL.ResolveReference(parsedURL)
	if resolved.Scheme == "" || resolved.Host == "" {
		return "", errors.New("download.url 必须是绝对 URL")
	}
	return resolved.String(), nil
}

func normalizeStorePackageLocator(namespace, packageName, version string) (string, string, string, error) {
	namespace = strings.TrimSpace(namespace)
	packageName = strings.TrimSpace(packageName)
	version = strings.TrimSpace(version)
	if err := sealpack.ValidatePackageID(namespace + "/" + packageName); err != nil {
		return "", "", "", err
	}
	if _, err := semver.NewVersion(version); err != nil {
		return "", "", "", fmt.Errorf("无效的版本号: %w", err)
	}
	return namespace, packageName, version, nil
}

func (m *StoreManager) normalizeStoreConfigLocked() (enabledUrls []string, disabledUrls []string, changed bool) {
	enabledUrls = normalizeConfiguredStoreBackendURLs(m.parent.Config.BackendUrls)
	disabledUrls = normalizeConfiguredStoreBackendURLs(m.parent.Config.DisabledBackendUrls)

	for _, enabledURL := range enabledUrls {
		if containsStoreBackendURL(disabledUrls, enabledURL) {
			disabledUrls = removeStoreBackendURL(disabledUrls, enabledURL)
		}
	}

	changed = strings.Join(m.parent.Config.BackendUrls, "\n") != strings.Join(enabledUrls, "\n") ||
		strings.Join(m.parent.Config.DisabledBackendUrls, "\n") != strings.Join(disabledUrls, "\n")
	if changed {
		m.parent.Config.BackendUrls = enabledUrls
		m.parent.Config.DisabledBackendUrls = disabledUrls
		m.parent.MarkModified()
	}
	return enabledUrls, disabledUrls, changed
}

func (m *StoreManager) invalidateStoreBackendLocked() {
	m.backend = nil
	m.backendCacheKey = ""
	m.backendFetchedAt = time.Time{}
}

func (m *StoreManager) storeConfigSnapshot() (enabledUrls []string, disabledUrls []string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	enabledUrls, disabledUrls, changed := m.normalizeStoreConfigLocked()
	if changed {
		m.invalidateStoreBackendLocked()
	}
	return append([]string(nil), enabledUrls...), append([]string(nil), disabledUrls...)
}

func storeBackendCacheKey(backendType StoreBackendType, rawURL string) string {
	return string(backendType) + "\x00" + rawURL
}

func (m *StoreManager) buildBackend(rawURL, id, name string, backendType StoreBackendType, enabled bool) *StoreBackend {
	backend := &StoreBackend{
		Url:      rawURL,
		ID:       id,
		Name:     name,
		Type:     backendType,
		Builtin:  backendType == StoreBackendTypeOfficial,
		Official: backendType == StoreBackendTypeOfficial,
		Enabled:  enabled,
	}

	if !enabled {
		backend.Health = false
		return backend
	}

	info, err := fetchStoreJSON[storeBackendInfoResponse](backend.Url + "/info")
	if err != nil {
		backend.Health = false
		return backend
	}

	if info.Name != "" {
		backend.Name = info.Name
	}
	if backend.Type != StoreBackendTypeOfficial && info.Sign != "" && OfficialStorePublicKey != "" {
		parsedURL, parseErr := url.Parse(backend.Url)
		if parseErr == nil {
			if verifyErr := crypto.RSAVerify256([]byte(parsedURL.Hostname()), info.Sign, OfficialStorePublicKey); verifyErr == nil {
				backend.Type = StoreBackendTypeTrusted
				backend.Sign = info.Sign
			}
		}
	}
	backend.ProtocolVersions = info.ProtocolVersions
	backend.Announcement = info.Announcement
	backend.Health = true
	if backend.Sign == "" {
		backend.Sign = info.Sign
	}
	return backend
}

func (m *StoreManager) resolveBackendTarget() (*storeBackendTarget, error) {
	enabledCustomBackends, _ := m.storeConfigSnapshot()
	if len(enabledCustomBackends) > 0 {
		backendURL := enabledCustomBackends[0]
		return &storeBackendTarget{
			rawURL:      backendURL,
			id:          "custom",
			name:        "自定义商店",
			backendType: StoreBackendTypeExtra,
			cacheKey:    storeBackendCacheKey(StoreBackendTypeExtra, backendURL),
		}, nil
	}

	officialURL, err := officialStoreBackendURL()
	if err != nil {
		return nil, err
	}
	return &storeBackendTarget{
		rawURL:      officialURL,
		id:          "official",
		name:        "官方商店",
		backendType: StoreBackendTypeOfficial,
		cacheKey:    storeBackendCacheKey(StoreBackendTypeOfficial, officialURL),
	}, nil
}

func (m *StoreManager) cachedBackend(cacheKey string, now time.Time) (*StoreBackend, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	if m.backend == nil || m.backendCacheKey != cacheKey {
		return nil, false
	}
	if m.backendFetchedAt.IsZero() || !now.Before(m.backendFetchedAt.Add(storeBackendInfoCacheTTL)) {
		return nil, false
	}
	backendCopy := *m.backend
	return &backendCopy, true
}

func (m *StoreManager) loadStoreBackend(force bool) (*StoreBackend, error) {
	target, err := m.resolveBackendTarget()
	if err != nil {
		m.lock.Lock()
		m.invalidateStoreBackendLocked()
		m.lock.Unlock()
		return nil, err
	}

	now := time.Now()
	if !force {
		if backend, ok := m.cachedBackend(target.cacheKey, now); ok {
			return backend, nil
		}
	}

	backend := m.buildBackend(target.rawURL, target.id, target.name, target.backendType, true)
	fetchedAt := time.Now()
	m.lock.Lock()
	m.backend = backend
	m.backendCacheKey = target.cacheKey
	m.backendFetchedAt = fetchedAt
	backendCopy := *backend
	m.lock.Unlock()
	return &backendCopy, nil
}

func (m *StoreManager) refreshStoreBackend() {
	_, _ = m.loadStoreBackend(true)
}

func (m *StoreManager) currentBackend() (*StoreBackend, error) {
	backend, err := m.loadStoreBackend(false)
	if err != nil {
		return nil, err
	}
	if backend == nil {
		return nil, errors.New("未配置扩展商店后端")
	}
	return backend, nil
}

func NewStoreManager(parent *Dice) *StoreManager {
	if pub, err := static.Scripts.ReadFile("scripts/seal_store.public.pem"); err == nil && len(pub) > 0 {
		OfficialStorePublicKey = string(pub)
	}

	m := &StoreManager{
		lock:             new(sync.RWMutex),
		parent:           parent,
		packageCache:     make(map[string]*StorePackage),
		InstalledPlugins: map[string]bool{},
		InstalledDecks:   map[string]bool{},
		InstalledReplies: map[string]bool{},
	}
	m.refreshStoreBackend()
	return m
}

func (m *StoreManager) StoreQueryRecommend() ([]*StorePackage, error) {
	backend, err := m.currentBackend()
	if err != nil {
		return nil, err
	}
	if !backend.Health {
		m.refreshStoreBackend()
		backend, err = m.currentBackend()
		if err != nil {
			return nil, err
		}
		if !backend.Health {
			return nil, fmt.Errorf("当前扩展商店后端不可用: %s", backend.Url)
		}
	}

	respResult, err := fetchStoreJSON[storeRecommendResponse](backend.Url + "/recommend")
	if err != nil {
		m.refreshStoreBackend()
		return nil, err
	}
	if !respResult.Result {
		return nil, fmt.Errorf("%s", respResult.Err)
	}
	return sanitizeStorePackages(respResult.Data, backend.Url)
}

func (m *StoreManager) StoreBackendList() []*StoreBackend {
	activeBackend, _ := m.currentBackend()
	enabledUrls, disabledUrls := m.storeConfigSnapshot()
	result := make([]*StoreBackend, 0, 1+len(enabledUrls)+len(disabledUrls))

	if officialURL, err := officialStoreBackendURL(); err == nil {
		if activeBackend != nil && activeBackend.Type == StoreBackendTypeOfficial && activeBackend.Url == officialURL {
			backendCopy := *activeBackend
			backendCopy.Enabled = true
			backendCopy.Builtin = true
			backendCopy.Official = true
			result = append(result, &backendCopy)
		} else {
			result = append(result, m.buildBackend(officialURL, "official", "官方商店", StoreBackendTypeOfficial, true))
		}
	}

	for index, backendURL := range enabledUrls {
		id := "custom"
		if index > 0 {
			id = fmt.Sprintf("custom:%d", index+1)
		}
		if activeBackend != nil && activeBackend.Type != StoreBackendTypeOfficial && activeBackend.Url == backendURL {
			backendCopy := *activeBackend
			backendCopy.ID = id
			backendCopy.Enabled = true
			result = append(result, &backendCopy)
			continue
		}
		result = append(result, m.buildBackend(backendURL, id, "自定义商店", StoreBackendTypeExtra, true))
	}

	for index, backendURL := range disabledUrls {
		result = append(result, m.buildBackend(
			backendURL,
			fmt.Sprintf("disabled:%d", index+1),
			"自定义商店",
			StoreBackendTypeExtra,
			false,
		))
	}

	return result
}

func (m *StoreManager) StoreAddBackend(rawURL string) error {
	normalizedURL, err := validateStoreBackendURL(rawURL)
	if err != nil {
		return err
	}

	m.lock.Lock()
	defer m.lock.Unlock()
	enabledUrls, disabledUrls, _ := m.normalizeStoreConfigLocked()
	if containsStoreBackendURL(enabledUrls, normalizedURL) {
		return fmt.Errorf("后端 `%s` 已经是当前商店来源", normalizedURL)
	}
	if containsStoreBackendURL(disabledUrls, normalizedURL) {
		disabledUrls = removeStoreBackendURL(disabledUrls, normalizedURL)
	}
	enabledUrls = append(enabledUrls, normalizedURL)
	m.parent.Config.BackendUrls = enabledUrls
	m.parent.Config.DisabledBackendUrls = disabledUrls
	m.parent.MarkModified()
	m.invalidateStoreBackendLocked()
	return nil
}

func (m *StoreManager) StoreRemoveBackend(id, rawURL string) error {
	id = strings.TrimSpace(id)
	m.lock.Lock()
	defer m.lock.Unlock()
	enabledUrls, disabledUrls, _ := m.normalizeStoreConfigLocked()
	targetURL, err := m.resolveStoreBackendActionURL(id, rawURL, enabledUrls, disabledUrls)
	if err != nil {
		return err
	}
	if targetURL == "" {
		return errors.New("cannot remove official backend")
	}
	enabledUrls = removeStoreBackendURL(enabledUrls, targetURL)
	disabledUrls = removeStoreBackendURL(disabledUrls, targetURL)
	m.parent.Config.BackendUrls = enabledUrls
	m.parent.Config.DisabledBackendUrls = disabledUrls
	m.parent.MarkModified()
	m.invalidateStoreBackendLocked()
	return nil
}

func (m *StoreManager) StoreSetBackendEnabled(id, rawURL string, enabled bool) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	enabledUrls, disabledUrls, _ := m.normalizeStoreConfigLocked()
	targetURL, err := m.resolveStoreBackendActionURL(strings.TrimSpace(id), rawURL, enabledUrls, disabledUrls)
	if err != nil {
		return err
	}
	if targetURL == "" {
		return errors.New("官方商店后端不支持禁用")
	}
	if enabled {
		if !containsStoreBackendURL(enabledUrls, targetURL) {
			enabledUrls = append(enabledUrls, targetURL)
		}
		disabledUrls = removeStoreBackendURL(disabledUrls, targetURL)
	} else {
		_, officialErr := officialStoreBackendURL()
		if officialErr != nil && len(enabledUrls) == 1 && containsStoreBackendURL(enabledUrls, targetURL) {
			return errors.New("至少需要保留一个启用的自定义商店后端；可删除该后端以恢复默认官方商店")
		}
		enabledUrls = removeStoreBackendURL(enabledUrls, targetURL)
		if !containsStoreBackendURL(disabledUrls, targetURL) {
			disabledUrls = append(disabledUrls, targetURL)
		}
	}
	m.parent.Config.BackendUrls = enabledUrls
	m.parent.Config.DisabledBackendUrls = disabledUrls
	m.parent.MarkModified()
	m.invalidateStoreBackendLocked()
	return nil
}

func (m *StoreManager) resolveStoreBackendActionURL(id, rawURL string, enabledUrls, disabledUrls []string) (string, error) {
	if rawURL != "" {
		return validateStoreBackendURL(rawURL)
	}
	switch {
	case id == "" || id == "official":
		return "", nil
	case id == "custom":
		if len(enabledUrls) > 0 {
			return enabledUrls[0], nil
		}
		if len(disabledUrls) > 0 {
			return disabledUrls[0], nil
		}
	case strings.HasPrefix(id, "custom:"):
		index, err := strconv.Atoi(strings.TrimPrefix(id, "custom:"))
		if err == nil && index >= 1 && index <= len(enabledUrls) {
			return enabledUrls[index-1], nil
		}
	case strings.HasPrefix(id, "disabled:"):
		index, err := strconv.Atoi(strings.TrimPrefix(id, "disabled:"))
		if err == nil && index >= 1 && index <= len(disabledUrls) {
			return disabledUrls[index-1], nil
		}
	}
	return "", fmt.Errorf("backend `%s` not found", id)
}

func (m *StoreManager) StoreQueryPage(params StoreQueryPageParams) (*StorePackagePage, error) {
	backend, err := m.currentBackend()
	if err != nil {
		return nil, err
	}
	if !backend.Health {
		m.refreshStoreBackend()
		backend, err = m.currentBackend()
		if err != nil {
			return nil, err
		}
		if !backend.Health {
			return nil, fmt.Errorf("当前扩展商店后端不可用: %s", backend.Url)
		}
	}

	reqParams := url.Values{}
	if params.Content != "" {
		reqParams.Set("content", params.Content)
	}
	if params.Author != "" {
		reqParams.Set("author", params.Author)
	}
	if params.Name != "" {
		reqParams.Set("name", params.Name)
	}
	if params.Category != "" {
		reqParams.Set("category", params.Category)
	}
	if params.SortBy != "" {
		reqParams.Set("sortBy", params.SortBy)
	}
	if params.Order != "" {
		reqParams.Set("order", params.Order)
	}
	if params.PageNum != 0 {
		reqParams.Set("pageNum", strconv.Itoa(params.PageNum))
	}
	if params.PageSize != 0 {
		reqParams.Set("pageSize", strconv.Itoa(params.PageSize))
	}

	requestURL, err := url.Parse(backend.Url + "/page")
	if err != nil {
		return nil, err
	}
	requestURL.RawQuery = reqParams.Encode()

	respResult, err := fetchStoreJSON[storePageResponse](requestURL.String())
	if err != nil {
		m.refreshStoreBackend()
		return nil, err
	}
	if !respResult.Result {
		return nil, fmt.Errorf("%s", respResult.Err)
	}
	if respResult.Data == nil {
		return nil, errors.New("扩展商店返回了空分页数据")
	}

	sanitized, err := sanitizeStorePackages(respResult.Data.Data, backend.Url)
	if err != nil {
		return nil, err
	}
	respResult.Data.Data = sanitized
	return respResult.Data, nil
}

func (m *StoreManager) StoreQueryPackageFiles(namespace, packageName, version string) ([]StorePackageFileEntry, error) {
	backend, err := m.currentBackend()
	if err != nil {
		return nil, err
	}
	if !backend.Health {
		m.refreshStoreBackend()
		backend, err = m.currentBackend()
		if err != nil {
			return nil, err
		}
		if !backend.Health {
			return nil, fmt.Errorf("当前扩展商店后端不可用: %s", backend.Url)
		}
	}

	namespace, packageName, version, err = normalizeStorePackageLocator(namespace, packageName, version)
	if err != nil {
		return nil, err
	}
	requestURL, err := buildStoreBackendResourceURL(backend.Url, "files", namespace, packageName, version)
	if err != nil {
		return nil, err
	}

	respResult, err := fetchStoreJSON[storePackageFilesResponse](requestURL)
	if err != nil {
		m.refreshStoreBackend()
		return nil, err
	}
	if !respResult.Result {
		return nil, fmt.Errorf("%s", respResult.Err)
	}
	return sanitizeStorePackageFileEntries(respResult.Data)
}

func (m *StoreManager) StorePreviewPackageFile(ctx context.Context, namespace, packageName, version, filePath string) (*http.Response, error) {
	backend, err := m.currentBackend()
	if err != nil {
		return nil, err
	}
	if !backend.Health {
		m.refreshStoreBackend()
		backend, err = m.currentBackend()
		if err != nil {
			return nil, err
		}
		if !backend.Health {
			return nil, fmt.Errorf("当前扩展商店后端不可用: %s", backend.Url)
		}
	}

	namespace, packageName, version, err = normalizeStorePackageLocator(namespace, packageName, version)
	if err != nil {
		return nil, err
	}
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return nil, errors.New("文件路径不能为空")
	}
	if pathErr := sealpack.ValidateRelativePackagePath(filePath); pathErr != nil {
		return nil, pathErr
	}

	requestURL, err := buildStoreBackendResourceURL(backend.Url, "file", namespace, packageName, version)
	if err != nil {
		return nil, err
	}
	parsedURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, err
	}
	query := parsedURL.Query()
	query.Set("path", filepath.ToSlash(filePath))
	parsedURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return nil, err
	}
	// #nosec G107 -- store backend URLs are user/admin-configured extension repository endpoints.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		m.refreshStoreBackend()
		return nil, err
	}
	return resp, nil
}

func (m *StoreManager) StoreQueryPackageManifest(ctx context.Context, id, version string) (*sealpack.Manifest, error) {
	id = strings.TrimSpace(id)
	version = strings.TrimSpace(version)
	namespace, packageName, err := sealpack.ParsePackageID(id)
	if err != nil {
		return nil, err
	}
	targetVersion, err := semver.NewVersion(version)
	if err != nil {
		return nil, fmt.Errorf("无效的版本号: %w", err)
	}

	resp, err := m.StorePreviewPackageFile(ctx, namespace, packageName, version, sealpack.InfoFile)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if len(message) == 0 {
			message = []byte(http.StatusText(resp.StatusCode))
		}
		return nil, fmt.Errorf("读取扩展包清单失败: %s", strings.TrimSpace(string(message)))
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxStoreManifestSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxStoreManifestSize {
		return nil, errors.New("扩展包 info.toml 不能超过 1 MB")
	}
	manifest, err := sealpack.ParseManifest(data)
	if err != nil {
		return nil, err
	}
	if manifest.Package.ID != id {
		return nil, fmt.Errorf("扩展包 ID 不匹配，期望 %s，实际 %s", id, manifest.Package.ID)
	}
	manifestVersion, err := semver.NewVersion(manifest.Package.Version)
	if err != nil || !manifestVersion.Equal(targetVersion) {
		return nil, fmt.Errorf("扩展包版本不匹配，期望 %s，实际 %s", version, manifest.Package.Version)
	}
	return manifest, nil
}

func (m *StoreManager) RefreshInstalled(packages []*StorePackage) {
	installed := map[string]bool{}
	if m.parent != nil && m.parent.PackageManager != nil {
		for _, pkg := range m.parent.PackageManager.List() {
			if pkg == nil || pkg.Manifest == nil {
				continue
			}
			installed[pkg.Manifest.Package.ID] = true
		}
	}

	m.lock.Lock()
	defer m.lock.Unlock()
	for _, pkg := range packages {
		if pkg == nil {
			continue
		}
		pkg.Installed = installed[pkg.ID]
		if pkg.FullID == "" {
			pkg.FullID = BuildStorePackageFullID(pkg.ID, pkg.Version)
		}
		if pkg.FullID != "" {
			m.packageCache[pkg.FullID] = pkg
		}
	}
}

func (m *StoreManager) FindPackage(id, version string) (*StorePackage, bool) {
	fullID := BuildStorePackageFullID(strings.TrimSpace(id), strings.TrimSpace(version))
	m.lock.RLock()
	defer m.lock.RUnlock()
	pkg, ok := m.packageCache[fullID]
	if !ok || pkg == nil {
		return nil, false
	}
	return pkg, true
}

// ResolvePackage resolves an exact package coordinate against the active store.
// List queries populate richer metadata in the cache, while list installation can
// still use the protocol's canonical artifact URL for versions outside those pages.
func (m *StoreManager) ResolvePackage(id, version string) (*StorePackage, error) {
	id = strings.TrimSpace(id)
	version = strings.TrimSpace(version)
	if cached, ok := m.FindPackage(id, version); ok {
		return cached, nil
	}

	namespace, packageName, err := sealpack.ParsePackageID(id)
	if err != nil {
		return nil, err
	}
	if _, versionErr := semver.NewVersion(version); versionErr != nil {
		return nil, fmt.Errorf("无效的版本号: %w", versionErr)
	}

	backend, err := m.currentBackend()
	if err != nil {
		return nil, err
	}
	if !backend.Health {
		m.refreshStoreBackend()
		backend, err = m.currentBackend()
		if err != nil {
			return nil, err
		}
		if !backend.Health {
			return nil, fmt.Errorf("当前扩展商店后端不可用: %s", backend.Url)
		}
	}

	downloadURL, err := buildStoreBackendResourceURL(
		backend.Url,
		"packages",
		namespace,
		packageName,
		version,
		sealpack.PackageSourceFileName(id, version),
	)
	if err != nil {
		return nil, err
	}
	return &StorePackage{
		ID:      id,
		Version: version,
		FullID:  BuildStorePackageFullID(id, version),
		Download: StorePackageDownload{
			URL: downloadURL,
		},
	}, nil
}

type StoreUploadFormOption struct {
	Key  string `json:"key"`
	Desc string `json:"desc"`
}

type StoreUploadFormElem struct {
	Key      string                  `json:"key"`
	Desc     string                  `json:"desc"`
	Required bool                    `json:"required"`
	Default  string                  `json:"default"`
	Options  []StoreUploadFormOption `json:"options"`
}

type StoreUploadInfo struct {
	UploadNotice string                `json:"uploadNotice"`
	UploadForm   []StoreUploadFormElem `json:"uploadForm"`
}

func (m *StoreManager) StoreQueryUploadInfo() (StoreUploadInfo, error) {
	backend, err := m.currentBackend()
	if err != nil {
		return StoreUploadInfo{}, err
	}
	if !backend.Health {
		m.refreshStoreBackend()
		backend, err = m.currentBackend()
		if err != nil {
			return StoreUploadInfo{}, err
		}
		if !backend.Health {
			return StoreUploadInfo{}, fmt.Errorf("当前扩展商店后端不可用: %s", backend.Url)
		}
	}
	result, err := fetchStoreJSON[StoreUploadInfo](backend.Url + "/upload/info")
	if err != nil {
		m.refreshStoreBackend()
		return StoreUploadInfo{}, err
	}
	return *result, nil
}

func sanitizeStorePackages(packages []*StorePackage, backendBaseURL string) ([]*StorePackage, error) {
	if len(packages) == 0 {
		return []*StorePackage{}, nil
	}

	result := make([]*StorePackage, 0, len(packages))
	for _, pkg := range packages {
		sanitized, err := sanitizeStorePackage(pkg, backendBaseURL)
		if err != nil {
			return nil, err
		}
		result = append(result, sanitized)
	}
	return result, nil
}

func sanitizeStorePackage(pkg *StorePackage, backendBaseURL string) (*StorePackage, error) {
	if pkg == nil {
		return nil, errors.New("商店返回了空包数据")
	}

	copyPkg := *pkg
	copyPkg.ID = strings.TrimSpace(copyPkg.ID)
	copyPkg.FormatVersion = strings.TrimSpace(copyPkg.FormatVersion)
	copyPkg.Version = strings.TrimSpace(copyPkg.Version)
	copyPkg.FullID = strings.TrimSpace(copyPkg.FullID)
	copyPkg.Name = strings.TrimSpace(copyPkg.Name)
	copyPkg.Download.URL = strings.TrimSpace(copyPkg.Download.URL)
	copyPkg.Download.ZipURL = strings.TrimSpace(copyPkg.Download.ZipURL)

	if err := sealpack.ValidatePackageID(copyPkg.ID); err != nil {
		return nil, fmt.Errorf("无效的包 ID: %w", err)
	}
	if copyPkg.Version == "" {
		return nil, errors.New("商店包缺少 version")
	}
	if _, err := semver.NewVersion(copyPkg.Version); err != nil {
		return nil, fmt.Errorf("无效的版本号: %w", err)
	}
	if copyPkg.Name == "" {
		return nil, errors.New("商店包缺少 name")
	}
	if copyPkg.Download.URL == "" {
		return nil, errors.New("商店包缺少 download.url")
	}

	resolvedDownloadURL, err := resolveStoreDownloadURL(copyPkg.Download.URL, backendBaseURL)
	if err != nil {
		return nil, fmt.Errorf("download.url 无效: %w", err)
	}
	copyPkg.Download.URL = resolvedDownloadURL
	parsedDownloadURL, err := url.Parse(copyPkg.Download.URL)
	if err != nil {
		return nil, fmt.Errorf("download.url 无效: %w", err)
	}
	if !strings.HasSuffix(strings.ToLower(parsedDownloadURL.Path), sealpack.Extension) {
		return nil, fmt.Errorf("download.url 必须指向 %s 文件", sealpack.Extension)
	}
	if copyPkg.Download.ZipURL != "" {
		resolvedZipURL, zipErr := resolveStoreDownloadURL(copyPkg.Download.ZipURL, backendBaseURL)
		if zipErr != nil {
			return nil, fmt.Errorf("download.zipUrl 无效: %w", zipErr)
		}
		copyPkg.Download.ZipURL = resolvedZipURL
	}

	expectedFullID := BuildStorePackageFullID(copyPkg.ID, copyPkg.Version)
	if copyPkg.FullID == "" {
		copyPkg.FullID = expectedFullID
	} else if copyPkg.FullID != expectedFullID {
		return nil, fmt.Errorf("fullId 与当前包信息不一致，期望 %s", expectedFullID)
	}

	contents, err := normalizeStoreContents(copyPkg.Contents)
	if err != nil {
		return nil, err
	}
	copyPkg.Contents = contents

	if copyPkg.Dependencies == nil {
		copyPkg.Dependencies = map[string]string{}
	}
	for depID := range copyPkg.Dependencies {
		if err := sealpack.ValidatePackageID(depID); err != nil {
			return nil, fmt.Errorf("依赖 %s 的包 ID 无效: %w", depID, err)
		}
	}
	if copyPkg.Download.Hash == nil {
		copyPkg.Download.Hash = map[string]string{}
	}
	if copyPkg.Authors == nil {
		copyPkg.Authors = []string{}
	}
	if copyPkg.Keywords == nil {
		copyPkg.Keywords = []string{}
	}
	if copyPkg.StoreAssets.Screenshots == nil {
		copyPkg.StoreAssets.Screenshots = []string{}
	}

	return &copyPkg, nil
}

func sanitizeStorePackageFileEntries(entries []StorePackageFileEntry) ([]StorePackageFileEntry, error) {
	if len(entries) == 0 {
		return []StorePackageFileEntry{}, nil
	}

	result := make([]StorePackageFileEntry, 0, len(entries))
	for _, entry := range entries {
		entry.Path = filepath.ToSlash(strings.TrimSpace(entry.Path))
		if entry.Path == "" {
			return nil, errors.New("商店返回了空文件路径")
		}
		if err := sealpack.ValidateRelativePackagePath(entry.Path); err != nil {
			return nil, fmt.Errorf("商店返回了无效文件路径 %s: %w", entry.Path, err)
		}
		result = append(result, entry)
	}
	return result, nil
}

func normalizeStoreContents(contents []string) ([]string, error) {
	if len(contents) == 0 {
		return []string{}, nil
	}

	seen := map[string]bool{}
	result := make([]string, 0, len(contents))
	for _, content := range contents {
		content = strings.TrimSpace(content)
		if content == "" {
			continue
		}
		if _, ok := storeAllowedContents[content]; !ok {
			return nil, fmt.Errorf("contents 包含不支持的资源类型: %s", content)
		}
		if !seen[content] {
			seen[content] = true
			result = append(result, content)
		}
	}
	return result, nil
}
