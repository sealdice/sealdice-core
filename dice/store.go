package dice

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/samber/lo"
)

type StoreExtType string

const (
	StoreExtTypePlugin StoreExtType = "plugin"
	StoreExtTypeDeck   StoreExtType = "deck"
	StoreExtTypeReply  StoreExtType = "reply"
)

func (d *Dice) getStoreBackends() []string {
	if d.AdvancedConfig.Enable && d.AdvancedConfig.StoreBackendUrl != "" {
		return []string{d.AdvancedConfig.StoreBackendUrl}
	}
	return lo.Map(BackendUrls, func(backend string, _ int) string {
		return backend + "/dice/api/store"
	})
}

type StoreBackend struct {
	Url string `json:"url"`

	ID               string   `json:"id"`
	Name             string   `json:"name"`
	ProtocolVersions []string `json:"protocolVersions"`
	Announcement     string   `json:"announcement"`
}

func (d *Dice) StoreQueryInfo(backend string) (StoreBackend, error) {
	resp, err := http.Get(backend + "/info")
	if err != nil {
		return StoreBackend{}, err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return StoreBackend{}, err
	}

	respResult := StoreBackend{}
	err = json.Unmarshal(respData, &respResult)
	if err != nil {
		return StoreBackend{}, err
	}
	result := respResult
	result.Url = backend
	return result, nil
}

type StoreExt struct {
	ID        string `json:"id"` // @<namespace>/<key>@<version>, e.g. @seal/example@1.0.0
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

func (d *Dice) StoreQueryRecommend() ([]*StoreExt, error) {
	backends := d.getStoreBackends()
	var result []*StoreExt
	var err error
	for _, backend := range backends {
		result, err = d.getRecommendFromBackend(backend)
		if err == nil {
			return result, nil
		}
	}
	return nil, fmt.Errorf("%w", err)
}

func (d *Dice) getRecommendFromBackend(backend string) ([]*StoreExt, error) {
	resp, err := http.Get(backend + "/recommend")
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

func (d *Dice) StoreQueryPage(params StoreQueryPageParams) (*StoreExtPage, error) {
	backends := d.getStoreBackends()
	var result *StoreExtPage
	var err error
	for _, backend := range backends {
		result, err = d.getStorePageFromBackend(backend, params)
		if err == nil {
			return result, nil
		}
	}
	return nil, fmt.Errorf("%w", err)
}

func (d *Dice) getStorePageFromBackend(backend string, params StoreQueryPageParams) (*StoreExtPage, error) {
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

	u, err := url.Parse(backend + "/page")
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

func (d *Dice) StoreQueryUploadInfo(backend string) (StoreUploadInfo, error) {
	resp, err := http.Get(backend + "/upload/info")
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
