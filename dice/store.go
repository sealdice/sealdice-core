package dice

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

type StoreExtType string

const (
	StoreExtTypePlugin StoreExtType = "plugin"
	StoreExtTypeDeck   StoreExtType = "deck"
	StoreExtTypeReply  StoreExtType = "reply"
)

type StoreBackendType string

const (
	StoreBackendTypeOfficial StoreBackendType = "official"
	StoreBackendTypeTrusted  StoreBackendType = "trusted"
	StoreBackendTypeExtra    StoreBackendType = "extra"
)

type StoreBackend struct {
	Url string `json:"url"`

	ID               string           `json:"id"`
	Name             string           `json:"name"`
	Type             StoreBackendType `json:"type"`
	ProtocolVersions []string         `json:"protocolVersions"`
	Announcement     string           `json:"announcement"`
	Health           bool
}

type StoreManager struct {
	lock       *sync.RWMutex
	parent     *Dice
	backends   []*StoreBackend
	storeCache map[string]*StoreExt

	InstalledPlugins map[string]bool `yaml:"-" json:"-"`
	InstalledDecks   map[string]bool `yaml:"-" json:"-"`
	InstalledReplies map[string]bool `yaml:"-" json:"-"`
}

func NewStoreManager(parent *Dice) *StoreManager {
	m := &StoreManager{
		lock:       new(sync.RWMutex),
		parent:     parent,
		storeCache: make(map[string]*StoreExt),
	}
	m.refreshStoreBackends()
	return m
}

func (m *StoreManager) refreshStoreBackends() {
	backends := make([]*StoreBackend, 0, len(BackendUrls))
	backendSet := make(map[string]bool)

	official := 0
	for i, backend := range BackendUrls {
		if backendSet[backend] {
			continue
		}
		id := "official"
		name := "官方仓库"
		if i > 0 {
			id += fmt.Sprintf(":%d", official+1)
			name += fmt.Sprintf("[线路%d]", official+1)
		}
		backends = append(backends, &StoreBackend{
			Url:  backend + "/dice/api/store",
			ID:   id,
			Name: name,
			Type: StoreBackendTypeOfficial,
		})
		backendSet[id] = true
		official++
	}

	extraBackends := m.parent.Config.StoreConfig.BackendUrls
	if len(extraBackends) > 0 {
		for _, backend := range extraBackends {
			if backendSet[backend] {
				continue
			}
			id := "extra:" + base64.StdEncoding.EncodeToString([]byte(backend))
			backends = append(backends, &StoreBackend{
				Url:  backend,
				ID:   id,
				Type: StoreBackendTypeExtra,
			})
			backendSet[id] = true
		}
	}
	var wg sync.WaitGroup
	wg.Add(len(backends))
	for _, backend := range backends {
		go func(wg *sync.WaitGroup, backend *StoreBackend) {
			defer wg.Done()
			info, err := m.storeQueryInfo(*backend)
			if err != nil {
				backend.Health = false
				return
			}
			if backend.Type != StoreBackendTypeOfficial {
				backend.Name = info.Name
			}
			backend.ProtocolVersions = info.ProtocolVersions
			backend.Announcement = info.Announcement
			backend.Health = true
		}(&wg, backend)
	}
	wg.Wait()

	m.lock.Lock()
	defer m.lock.Unlock()
	m.backends = backends
}

func (m *StoreManager) storeQueryInfo(backend StoreBackend) (StoreBackend, error) {
	resp, err := http.Get(backend.Url + "/info")
	if err != nil {
		return backend, err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return backend, err
	}
	if resp.StatusCode != 200 {
		return backend, fmt.Errorf("%s", string(respData))
	}

	respResult := StoreBackend{}
	err = json.Unmarshal(respData, &respResult)
	if err != nil {
		return backend, err
	}
	backend.Name = respResult.Name
	backend.ProtocolVersions = respResult.ProtocolVersions
	backend.Announcement = respResult.Announcement
	backend.Health = true
	return backend, nil
}

type StoreExt struct {
	ID        string `json:"id"` // <namespace>@<key>@<version>, e.g. seal@example@1.0.0
	Key       string `json:"key"`
	Namespace string `json:"namespace"`
	Version   string `json:"version"`
	Installed bool   `json:"installed"`

	Source string       `json:"source"` // official
	Type   StoreExtType `json:"type"`
	Ext    string       `json:"ext"` // .js | .json |...

	Name         string            `json:"name"`
	Authors      []string          `json:"authors"`
	Desc         string            `json:"desc"`
	License      string            `json:"license"`
	ReleaseTime  uint64            `json:"releaseTime"`
	UpdateTime   uint64            `json:"updateTime"`
	Tags         []string          `json:"tags"`
	Rate         float64           `json:"rate"`
	Extra        map[string]string `json:"extra"`
	DownloadNum  uint64            `json:"downloadNum"`
	DownloadUrl  string            `json:"downloadUrl"`
	Hash         map[string]string `json:"hash"`
	HomePage     string            `json:"homePage"`
	SealVersion  string            `json:"sealVersion"`
	Dependencies map[string]string `json:"dependencies"`
}

func (m *StoreManager) StoreQueryRecommend() ([]*StoreExt, error) {
	m.refreshStoreBackends()

	m.lock.RLock()
	defer m.lock.RUnlock()

	backends := m.backends
	if len(backends) == 0 {
		return []*StoreExt{}, nil
	}

	var result []*StoreExt
	var err error
	for _, backend := range backends {
		if !backend.Health {
			continue
		}
		result, err = m.getRecommendFromBackend(*backend)
		if err == nil {
			return result, nil
		}
	}
	return nil, fmt.Errorf("%w", err)
}

func (m *StoreManager) getRecommendFromBackend(backend StoreBackend) ([]*StoreExt, error) {
	resp, err := http.Get(backend.Url + "/recommend")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var respResult struct {
		Result bool
		Data   []*StoreExt
		Err    string
	}
	err = json.Unmarshal(respData, &respResult)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%s", string(respData))
	}
	if !respResult.Result {
		return nil, fmt.Errorf("%s", respResult.Err)
	}
	return respResult.Data, nil
}

type StoreQueryPageParams struct {
	Type     string `query:"type"`
	PageNum  int    `query:"pageNum"`
	PageSize int    `query:"pageSize"`
	Author   string `query:"author"`
	Name     string `query:"name"`
	SortBy   string `query:"sortBy"`
	Order    string `query:"order"`
}

type StoreExtPage struct {
	Data     []*StoreExt
	PageNum  int
	PageSize int
	Next     bool
}

func (m *StoreManager) StoreBackendList() []*StoreBackend {
	m.refreshStoreBackends()
	return m.backends
}

func (m *StoreManager) StoreAddBackend(url string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	backends := m.parent.Config.StoreConfig.BackendUrls
	for _, backend := range backends {
		if backend == url {
			return fmt.Errorf("backend `%s` already exists", backend)
		}
	}
	backends = append(backends, url)

	m.parent.Config.StoreConfig = StoreConfig{
		BackendUrls: backends,
	}
	return nil
}

func (m *StoreManager) StoreRemoveBackend(id string) error {
	if strings.HasPrefix("official:", id) {
		return fmt.Errorf("cannot remove official backend")
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	backends := []string{}
	for _, backend := range m.backends {
		if backend.Type != StoreBackendTypeOfficial && backend.ID != id {
			backends = append(backends, backend.Url)
		}
	}

	m.parent.Config.StoreConfig = StoreConfig{
		BackendUrls: backends,
	}
	return nil
}

func (m *StoreManager) StoreQueryPage(params StoreQueryPageParams) (*StoreExtPage, error) {
	m.refreshStoreBackends()

	m.lock.RLock()
	defer m.lock.RUnlock()

	backends := m.backends
	if len(backends) == 0 {
		return &StoreExtPage{}, nil
	}

	var result *StoreExtPage
	var err error
	for _, backend := range backends {
		if !backend.Health {
			continue
		}
		result, err = m.getStorePageFromBackend(*backend, params)
		if err == nil {
			return result, nil
		}
	}
	return nil, fmt.Errorf("%w", err)
}

func (m *StoreManager) getStorePageFromBackend(backend StoreBackend, params StoreQueryPageParams) (*StoreExtPage, error) {
	reqParams := url.Values{}
	if params.Type != "" {
		reqParams.Set("type", params.Type)
	}
	if params.Author != "" {
		reqParams.Set("author", params.Author)
	}
	if params.Name != "" {
		reqParams.Set("name", params.Name)
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

	u, err := url.Parse(backend.Url + "/page")
	if err != nil {
		return nil, err
	}
	u.RawQuery = reqParams.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%s", string(respData))
	}

	var respResult struct {
		Result bool
		Data   *StoreExtPage
		Err    string
	}
	err = json.Unmarshal(respData, &respResult)
	if err != nil {
		return nil, err
	}
	if !respResult.Result {
		return nil, fmt.Errorf("%s", respResult.Err)
	}
	return respResult.Data, nil
}

func (m *StoreManager) RefreshInstalled(exts []*StoreExt) {
	m.lock.Lock()
	defer m.lock.Unlock()
	for _, ext := range exts {
		switch ext.Type {
		case StoreExtTypeDeck:
			ext.Installed = m.InstalledDecks[ext.ID]
		case StoreExtTypePlugin:
			ext.Installed = m.InstalledPlugins[ext.ID]
		default:
			// pass
		}
		if len(ext.ID) > 0 {
			m.storeCache[ext.ID] = ext
		}
	}
}

func (m *StoreManager) FindExt(id string) (*StoreExt, bool) {
	ext, ok := m.storeCache[id]
	return ext, ok
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

func (m *StoreManager) StoreQueryUploadInfo(backend StoreBackend) (StoreUploadInfo, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	resp, err := http.Get(backend.Url + "/upload/info")
	if err != nil {
		return StoreUploadInfo{}, err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return StoreUploadInfo{}, err
	}

	err = json.Unmarshal(respData, &StoreUploadInfo{})
	if err != nil {
		return StoreUploadInfo{}, err
	}
	result := StoreUploadInfo{}
	return result, nil
}
