package extension

import (
	"sealdice-core/dice"
	"sealdice-core/dice/sealpack"
	"sealdice-core/model/common/response"
)

type PackageListResp struct {
	Items []*sealpack.Instance `json:"items"`
}

type PackageIDQuery struct {
	ID string `query:"id"`
}

type PackageAssetQuery struct {
	ID   string `query:"id"`
	Path string `query:"path"`
}

type UploadReq struct {
	RawBody []byte
}

type ExtensionActionResp struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

type IDBody struct {
	ID string `json:"id"`
}

type IDReq struct {
	Body IDBody `json:"body"`
}

type InstallURLBody struct {
	URL string `json:"url"`
}

type InstallURLReq struct {
	Body InstallURLBody `json:"body"`
}

type UninstallBody struct {
	ID   string                 `json:"id"`
	Mode sealpack.UninstallMode `json:"mode"`
}

type UninstallReq struct {
	Body UninstallBody `json:"body"`
}

type ReloadContentBody struct {
	Content string `json:"content"`
}

type ReloadContentReq struct {
	Body ReloadContentBody `json:"body"`
}

type ConfigReq struct {
	ID   string                 `query:"id"`
	Body map[string]interface{} `json:"body"`
}

type StoreBackendsResp struct {
	Items []*dice.StoreBackend `json:"items"`
}

type StorePackagesResp struct {
	Items []*dice.StorePackage `json:"items"`
}

type StorePageResp struct {
	Items    []*dice.StorePackage `json:"items"`
	PageNum  int                  `json:"pageNum"`
	PageSize int                  `json:"pageSize"`
	Next     bool                 `json:"next"`
}

type StoreBackendAddBody struct {
	Url string `json:"url"`
}

type StoreBackendAddReq struct {
	Body StoreBackendAddBody `json:"body"`
}

type StoreBackendBody struct {
	ID        string `json:"id"`
	BackendID string `json:"backendID"`
	Url       string `json:"url"`
}

type StoreBackendReq struct {
	Body StoreBackendBody `json:"body"`
}

type StorePackageLocator struct {
	Namespace   string `path:"namespace"`
	PackageName string `path:"package"`
	Version     string `path:"version"`
}

type StoreFilePreviewReq struct {
	Namespace   string `path:"namespace"`
	PackageName string `path:"package"`
	Version     string `path:"version"`
	Path        string `query:"path"`
}

type StorePackageFilesResp struct {
	Items []dice.StorePackageFileEntry `json:"items"`
}

type StoreDownloadBody struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type StoreDownloadReq struct {
	Body StoreDownloadBody `json:"body"`
}

type StoreInstallItem struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type StoreInstallListBody struct {
	Packages []StoreInstallItem `json:"packages"`
}

type StoreInstallListReq struct {
	Body StoreInstallListBody `json:"body"`
}

type StoreInstallListItemResult struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type StoreInstallListResp struct {
	Items     []StoreInstallListItemResult `json:"items"`
	Installed int                          `json:"installed"`
	Skipped   int                          `json:"skipped"`
	Failed    int                          `json:"failed"`
}

type StorePackageInfoListReq = StoreInstallListReq

type StorePackageInfoListItemResult struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Name    string `json:"name,omitempty"`
	Error   string `json:"error,omitempty"`
}

type StorePreviewDownloadReq = StoreDownloadReq

type StorePageQuery = dice.StoreQueryPageParams

type PackageConfigSchemaResp map[string]sealpack.ConfigSchema

type PageResult[T any] = response.ItemResponse[T]
