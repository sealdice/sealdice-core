package extension

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/danielgtaylor/huma/v2"

	"sealdice-core/dice"
	"sealdice-core/dice/sealpack"
	"sealdice-core/model/common/request"
	"sealdice-core/model/common/response"
)

const filePreviewContentSecurityPolicy = "sandbox; default-src 'none'; img-src 'self' data:; style-src 'unsafe-inline'"

const maxStoreInstallListPackages = 200

type uploadStreamContextKey struct{}

type uploadStreamResult struct {
	preview *dice.PackageUploadPreview
	done    bool
}

type Service struct {
	dice *dice.Dice
	dm   *dice.DiceManager
}

func NewService(dm *dice.DiceManager) *Service {
	return &Service{
		dice: dm.GetDice(),
		dm:   dm,
	}
}

func (s *Service) RegisterRoutes(grp *huma.Group) {
	huma.Get(grp, "/packages", s.ListPackages)
	huma.Get(grp, "/asset", s.Asset)
	huma.Get(grp, "/config", s.GetConfig)
	huma.Get(grp, "/config-schema", s.GetConfigSchema)

	huma.Get(grp, "/store/backends", s.StoreBackends)
	huma.Get(grp, "/store/recommend", s.StoreRecommend)
	huma.Get(grp, "/store/page", s.StorePage)
	huma.Get(grp, "/store/files/{namespace}/{package}/{version}", s.StorePackageFiles)
	huma.Get(grp, "/store/file/{namespace}/{package}/{version}", s.StorePackageFilePreview)
	huma.Post(grp, "/store/package-info-list", s.StorePackageInfoList)
	huma.Post(grp, "/store/preview-download", s.StorePreviewDownload)
}

func (s *Service) RegisterProtectedRoutes(grp *huma.Group) {
	huma.Post(grp, "/packages/refresh", s.RefreshPackages)
	huma.Post(grp, "/preview-upload", s.PreviewUpload, func(o *huma.Operation) {
		o.Middlewares = append(o.Middlewares, s.streamUploadMiddleware(grp, true))
	})
	huma.Post(grp, "/preview-url", s.PreviewURL)
	huma.Post(grp, "/install-upload", s.InstallUpload, func(o *huma.Operation) {
		o.Middlewares = append(o.Middlewares, s.streamUploadMiddleware(grp, false))
	})
	huma.Post(grp, "/install-url", s.InstallURL)
	huma.Post(grp, "/uninstall", s.Uninstall)
	huma.Post(grp, "/enable", s.Enable)
	huma.Post(grp, "/disable", s.Disable)
	huma.Post(grp, "/reload", s.Reload)
	huma.Post(grp, "/reload-content", s.ReloadContent)
	huma.Post(grp, "/reload-all", s.ReloadAll)
	huma.Put(grp, "/config", s.PutConfig)

	huma.Post(grp, "/store/backends", s.StoreAddBackend)
	huma.Delete(grp, "/store/backends", s.StoreRemoveBackend)
	huma.Post(grp, "/store/backends/enable", s.StoreEnableBackend)
	huma.Post(grp, "/store/backends/disable", s.StoreDisableBackend)
	huma.Post(grp, "/store/download", s.StoreDownload)
	huma.Post(grp, "/store/install-list", s.StoreInstallList)
}

func (s *Service) ListPackages(_ context.Context, _ *request.Empty) (*response.ItemResponse[PackageListResp], error) {
	pm := s.packageManager()
	items := []*sealpack.Instance{}
	if pm != nil {
		items = pm.List()
	}
	return response.NewItemResponse(PackageListResp{Items: items}), nil
}

func (s *Service) RefreshPackages(_ context.Context, _ *request.Empty) (*response.ItemResponse[*dice.PackageRefreshResult], error) {
	pm := s.packageManager()
	if pm == nil {
		return nil, huma.Error500InternalServerError("package manager unavailable")
	}
	result, err := pm.RefreshFromDisk()
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}
	return response.NewItemResponse(result), nil
}

func (s *Service) PreviewUpload(ctx context.Context, req *UploadReq) (*response.ItemResponse[*dice.PackageUploadPreview], error) {
	if streamed, ok := ctx.Value(uploadStreamContextKey{}).(uploadStreamResult); ok && streamed.done {
		return response.NewItemResponse(streamed.preview), nil
	}
	pm := s.packageManager()
	if pm == nil {
		return nil, huma.Error500InternalServerError("package manager unavailable")
	}
	if len(req.RawBody) == 0 {
		return nil, huma.Error400BadRequest("未上传扩展包文件")
	}
	preview, err := pm.PreviewFromStreamContext(ctx, bytes.NewReader(req.RawBody))
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	return response.NewItemResponse(preview), nil
}

func (s *Service) InstallUpload(ctx context.Context, req *UploadReq) (*response.ItemResponse[ExtensionActionResp], error) {
	if streamed, ok := ctx.Value(uploadStreamContextKey{}).(uploadStreamResult); ok && streamed.done {
		return response.NewItemResponse(ExtensionActionResp{Success: true, Message: "扩展包安装成功"}), nil
	}
	pm := s.packageManager()
	if pm == nil {
		return nil, huma.Error500InternalServerError("package manager unavailable")
	}
	if len(req.RawBody) == 0 {
		return nil, huma.Error400BadRequest("未上传扩展包文件")
	}
	if err := pm.InstallFromStreamContext(ctx, bytes.NewReader(req.RawBody)); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	return response.NewItemResponse(ExtensionActionResp{Success: true, Message: "扩展包安装成功"}), nil
}

func (s *Service) InstallURL(ctx context.Context, req *InstallURLReq) (*response.ItemResponse[ExtensionActionResp], error) {
	pm := s.packageManager()
	if pm == nil {
		return nil, huma.Error500InternalServerError("package manager unavailable")
	}
	if err := pm.InstallFromURLContext(ctx, strings.TrimSpace(req.Body.URL), nil); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	return response.NewItemResponse(ExtensionActionResp{Success: true, Message: "扩展包安装成功"}), nil
}

func (s *Service) PreviewURL(ctx context.Context, req *InstallURLReq) (*response.ItemResponse[*dice.PackageUploadPreview], error) {
	pm := s.packageManager()
	if pm == nil {
		return nil, huma.Error500InternalServerError("package manager unavailable")
	}
	url := strings.TrimSpace(req.Body.URL)
	if url == "" {
		return nil, huma.Error400BadRequest("未提供扩展包 URL")
	}
	preview, err := pm.PreviewFromURLWithOptionsContext(ctx, url, dice.PackageDownloadOptions{})
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	return response.NewItemResponse(preview), nil
}

func (s *Service) Uninstall(_ context.Context, req *UninstallReq) (*response.ItemResponse[ExtensionActionResp], error) {
	pm := s.packageManager()
	if pm == nil {
		return nil, huma.Error500InternalServerError("package manager unavailable")
	}
	mode := req.Body.Mode
	if mode == "" {
		mode = sealpack.UninstallModeFull
	}
	if err := pm.Uninstall(req.Body.ID, mode); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	return response.NewItemResponse(ExtensionActionResp{Success: true, Message: "扩展包卸载成功"}), nil
}

func (s *Service) Enable(_ context.Context, req *IDReq) (*response.ItemResponse[*sealpack.OperationResult], error) {
	pm := s.packageManager()
	if pm == nil {
		return nil, huma.Error500InternalServerError("package manager unavailable")
	}
	result, err := pm.Enable(req.Body.ID)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	return response.NewItemResponse(result), nil
}

func (s *Service) Disable(_ context.Context, req *IDReq) (*response.ItemResponse[*sealpack.OperationResult], error) {
	pm := s.packageManager()
	if pm == nil {
		return nil, huma.Error500InternalServerError("package manager unavailable")
	}
	result, err := pm.Disable(req.Body.ID)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	return response.NewItemResponse(result), nil
}

func (s *Service) Reload(_ context.Context, req *IDReq) (*response.ItemResponse[*sealpack.ReloadResult], error) {
	pm := s.packageManager()
	if pm == nil {
		return nil, huma.Error500InternalServerError("package manager unavailable")
	}
	result, err := pm.Reload(req.Body.ID)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	return response.NewItemResponse(result), nil
}

func (s *Service) ReloadContent(_ context.Context, req *ReloadContentReq) (*response.ItemResponse[*sealpack.ReloadResult], error) {
	pm := s.packageManager()
	if pm == nil {
		return nil, huma.Error500InternalServerError("package manager unavailable")
	}
	result, err := pm.ReloadByContent(req.Body.Content)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	return response.NewItemResponse(result), nil
}

func (s *Service) ReloadAll(_ context.Context, _ *request.Empty) (*response.ItemResponse[*sealpack.ReloadResult], error) {
	pm := s.packageManager()
	if pm == nil {
		return nil, huma.Error500InternalServerError("package manager unavailable")
	}
	result, err := pm.ReloadAll()
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}
	return response.NewItemResponse(result), nil
}

func (s *Service) GetConfig(_ context.Context, req *PackageIDQuery) (*response.ItemResponse[map[string]interface{}], error) {
	pm := s.packageManager()
	if pm == nil {
		return nil, huma.Error500InternalServerError("package manager unavailable")
	}
	config, err := pm.GetConfig(req.ID)
	if err != nil {
		return nil, huma.Error404NotFound(err.Error())
	}
	if config == nil {
		config = map[string]interface{}{}
	}
	return response.NewItemResponse(config), nil
}

func (s *Service) PutConfig(_ context.Context, req *ConfigReq) (*response.ItemResponse[ExtensionActionResp], error) {
	pm := s.packageManager()
	if pm == nil {
		return nil, huma.Error500InternalServerError("package manager unavailable")
	}
	if req.Body == nil {
		req.Body = map[string]interface{}{}
	}
	if err := pm.SetConfig(req.ID, req.Body); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	return response.NewItemResponse(ExtensionActionResp{Success: true, Message: "配置更新成功"}), nil
}

func (s *Service) GetConfigSchema(_ context.Context, req *PackageIDQuery) (*response.ItemResponse[PackageConfigSchemaResp], error) {
	pm := s.packageManager()
	if pm == nil {
		return nil, huma.Error500InternalServerError("package manager unavailable")
	}
	pkg, ok := pm.Get(req.ID)
	if !ok || pkg == nil || pkg.Manifest == nil {
		return nil, huma.Error404NotFound("扩展包不存在")
	}
	return response.NewItemResponse(PackageConfigSchemaResp(pkg.Manifest.Config)), nil
}

func (s *Service) Asset(_ context.Context, req *PackageAssetQuery) (*huma.StreamResponse, error) {
	pm := s.packageManager()
	if pm == nil {
		return nil, huma.Error500InternalServerError("package manager unavailable")
	}
	if strings.TrimSpace(req.ID) == "" || strings.TrimSpace(req.Path) == "" {
		return nil, huma.Error400BadRequest("missing id or path")
	}
	if err := sealpack.ValidateRelativePackagePath(req.Path); err != nil {
		return nil, huma.Error400BadRequest("invalid path")
	}

	pkg, ok := pm.Get(req.ID)
	if !ok || pkg == nil || pkg.InstallPath == "" {
		return nil, huma.Error404NotFound("扩展包不存在")
	}
	targetPath, err := resolvePackageAssetPath(pkg.InstallPath, req.Path)
	if err != nil {
		return nil, huma.Error403Forbidden("invalid path")
	}
	info, err := os.Stat(targetPath)
	if err != nil || info.IsDir() {
		return nil, huma.Error404NotFound("资源不存在")
	}
	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(targetPath)))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return &huma.StreamResponse{
		Body: func(ctx huma.Context) {
			ctx.SetHeader("Content-Type", contentType)
			ctx.SetHeader("Content-Length", strconv.FormatInt(info.Size(), 10))
			ctx.SetHeader("X-Content-Type-Options", "nosniff")
			ctx.SetHeader("Content-Security-Policy", filePreviewContentSecurityPolicy)
			file, openErr := os.Open(targetPath)
			if openErr != nil {
				return
			}
			defer func() { _ = file.Close() }()
			_, _ = io.Copy(ctx.BodyWriter(), file)
			if flusher, ok := ctx.BodyWriter().(http.Flusher); ok {
				flusher.Flush()
			}
		},
	}, nil
}

func (s *Service) StoreBackends(_ context.Context, _ *request.Empty) (*response.ItemResponse[StoreBackendsResp], error) {
	sm := s.storeManager()
	if sm == nil {
		return nil, huma.Error500InternalServerError("store manager unavailable")
	}
	return response.NewItemResponse(StoreBackendsResp{Items: sm.StoreBackendList()}), nil
}

func (s *Service) StoreAddBackend(_ context.Context, req *StoreBackendAddReq) (*response.ItemResponse[ExtensionActionResp], error) {
	sm := s.storeManager()
	if sm == nil {
		return nil, huma.Error500InternalServerError("store manager unavailable")
	}
	if err := sm.StoreAddBackend(req.Body.Url); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	return response.NewItemResponse(ExtensionActionResp{Success: true}), nil
}

func (s *Service) StoreRemoveBackend(_ context.Context, req *StoreBackendReq) (*response.ItemResponse[ExtensionActionResp], error) {
	sm := s.storeManager()
	if sm == nil {
		return nil, huma.Error500InternalServerError("store manager unavailable")
	}
	if err := sm.StoreRemoveBackend(req.Body.ID, req.Body.Url); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	return response.NewItemResponse(ExtensionActionResp{Success: true}), nil
}

func (s *Service) StoreEnableBackend(_ context.Context, req *StoreBackendReq) (*response.ItemResponse[ExtensionActionResp], error) {
	return s.setStoreBackendEnabled(req, true)
}

func (s *Service) StoreDisableBackend(_ context.Context, req *StoreBackendReq) (*response.ItemResponse[ExtensionActionResp], error) {
	return s.setStoreBackendEnabled(req, false)
}

func (s *Service) setStoreBackendEnabled(req *StoreBackendReq, enabled bool) (*response.ItemResponse[ExtensionActionResp], error) {
	sm := s.storeManager()
	if sm == nil {
		return nil, huma.Error500InternalServerError("store manager unavailable")
	}
	id := req.Body.ID
	if id == "" {
		id = req.Body.BackendID
	}
	if err := sm.StoreSetBackendEnabled(id, req.Body.Url, enabled); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	return response.NewItemResponse(ExtensionActionResp{Success: true}), nil
}

func (s *Service) StoreRecommend(ctx context.Context, _ *request.Empty) (*response.ItemResponse[StorePackagesResp], error) {
	sm := s.storeManager()
	if sm == nil {
		return nil, huma.Error500InternalServerError("store manager unavailable")
	}
	items, err := sm.StoreQueryRecommendContext(ctx)
	if err != nil {
		return nil, huma.Error502BadGateway(err.Error())
	}
	sm.RefreshInstalled(items)
	return response.NewItemResponse(StorePackagesResp{Items: items}), nil
}

func (s *Service) StorePage(ctx context.Context, req *StorePageQuery) (*response.ItemResponse[StorePageResp], error) {
	sm := s.storeManager()
	if sm == nil {
		return nil, huma.Error500InternalServerError("store manager unavailable")
	}
	page, err := sm.StoreQueryPageContext(ctx, *req)
	if err != nil {
		return nil, huma.Error502BadGateway(err.Error())
	}
	sm.RefreshInstalled(page.Data)
	return response.NewItemResponse(StorePageResp{
		Items:    page.Data,
		PageNum:  page.PageNum,
		PageSize: page.PageSize,
		Next:     page.Next,
	}), nil
}

func (s *Service) StorePackageFiles(ctx context.Context, req *StorePackageLocator) (*response.ItemResponse[StorePackageFilesResp], error) {
	sm := s.storeManager()
	if sm == nil {
		return nil, huma.Error500InternalServerError("store manager unavailable")
	}
	items, err := sm.StoreQueryPackageFilesContext(ctx, req.Namespace, req.PackageName, req.Version)
	if err != nil {
		return nil, huma.Error502BadGateway(err.Error())
	}
	return response.NewItemResponse(StorePackageFilesResp{Items: items}), nil
}

func (s *Service) StorePackageFilePreview(ctx context.Context, req *StoreFilePreviewReq) (*huma.StreamResponse, error) {
	sm := s.storeManager()
	if sm == nil {
		return nil, huma.Error500InternalServerError("store manager unavailable")
	}
	resp, err := sm.StorePreviewPackageFile(ctx, req.Namespace, req.PackageName, req.Version, req.Path)
	if err != nil {
		return nil, huma.Error502BadGateway(err.Error())
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()
	if resp.StatusCode != http.StatusOK {
		errText := strings.TrimSpace(http.StatusText(resp.StatusCode))
		if errText == "" {
			errText = "商店文件预览失败"
		}
		switch resp.StatusCode {
		case http.StatusNotFound:
			return nil, huma.Error404NotFound(errText)
		default:
			return nil, huma.Error502BadGateway(errText)
		}
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, huma.Error502BadGateway("读取商店文件失败")
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return &huma.StreamResponse{
		Body: func(hctx huma.Context) {
			hctx.SetHeader("Content-Type", contentType)
			hctx.SetHeader("Content-Length", strconv.Itoa(len(body)))
			hctx.SetHeader("Cache-Control", "public, max-age=31536000, immutable")
			hctx.SetHeader("X-Content-Type-Options", "nosniff")
			hctx.SetHeader("Content-Security-Policy", filePreviewContentSecurityPolicy)
			_, _ = hctx.BodyWriter().Write(body)
		},
	}, nil
}

func (s *Service) StorePackageInfoList(ctx context.Context, req *StorePackageInfoListReq) (*response.ItemResponse[[]StorePackageInfoListItemResult], error) {
	sm := s.storeManager()
	pm := s.packageManager()
	if sm == nil || pm == nil {
		return nil, huma.Error500InternalServerError("extension managers unavailable")
	}
	packages := req.Body.Packages
	if len(packages) == 0 {
		return nil, huma.Error400BadRequest("清单中没有可查询的扩展包")
	}
	if len(packages) > maxStoreInstallListPackages {
		return nil, huma.Error400BadRequest("清单中的扩展包不能超过 200 个")
	}
	results := make([]StorePackageInfoListItemResult, 0, len(packages))
	seen := make(map[string]struct{}, len(packages))
	for _, item := range packages {
		result := StorePackageInfoListItemResult{ID: item.ID, Version: item.Version}
		coordinate := dice.BuildStorePackageFullID(strings.TrimSpace(item.ID), strings.TrimSpace(item.Version))
		if _, exists := seen[coordinate]; exists {
			return nil, huma.Error400BadRequest("清单中存在重复的扩展包版本: " + coordinate)
		}
		seen[coordinate] = struct{}{}

		if installed, exists := pm.Get(strings.TrimSpace(item.ID)); exists && installed != nil && installed.Manifest != nil &&
			sameStorePackageVersion(installed.Manifest.Package.Version, item.Version) {
			result.Name = installed.Manifest.Package.Name
			results = append(results, result)
			continue
		}

		manifest, err := sm.StoreQueryPackageManifest(ctx, item.ID, item.Version)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.ID = manifest.Package.ID
			result.Version = manifest.Package.Version
			result.Name = manifest.Package.Name
		}
		results = append(results, result)
	}
	return response.NewItemResponse(results), nil
}

func (s *Service) StorePreviewDownload(ctx context.Context, req *StorePreviewDownloadReq) (*response.ItemResponse[*dice.PackageUploadPreview], error) {
	sm := s.storeManager()
	pm := s.packageManager()
	if sm == nil || pm == nil {
		return nil, huma.Error500InternalServerError("extension managers unavailable")
	}
	target, ok := sm.FindPackage(req.Body.ID, req.Body.Version)
	if !ok {
		return nil, huma.Error400BadRequest("未找到已缓存的商店包，请先刷新商店列表后重试")
	}
	preview, err := pm.PreviewFromURLWithOptionsContext(ctx, target.Download.URL, dice.PackageDownloadOptions{
		Hashes:       target.Download.Hash,
		ExpectedSize: target.Download.Size,
	})
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	return response.NewItemResponse(preview), nil
}

func (s *Service) StoreDownload(ctx context.Context, req *StoreDownloadReq) (*response.ItemResponse[StoreInstallListItemResult], error) {
	sm := s.storeManager()
	pm := s.packageManager()
	if sm == nil || pm == nil {
		return nil, huma.Error500InternalServerError("extension managers unavailable")
	}
	target, err := sm.ResolvePackage(req.Body.ID, req.Body.Version)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	status, err := s.installStorePackage(ctx, target, true)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	result := StoreInstallListItemResult{
		ID:      target.ID,
		Version: target.Version,
		Status:  status,
	}
	if status == "skipped" {
		result.Message = "已安装目标版本"
	}
	return response.NewItemResponse(result), nil
}

func (s *Service) StoreInstallList(ctx context.Context, req *StoreInstallListReq) (*response.ItemResponse[StoreInstallListResp], error) {
	sm := s.storeManager()
	if sm == nil {
		return nil, huma.Error500InternalServerError("store manager unavailable")
	}
	packages := req.Body.Packages
	if len(packages) == 0 {
		return nil, huma.Error400BadRequest("清单中没有可安装的扩展包")
	}
	if len(packages) > maxStoreInstallListPackages {
		return nil, huma.Error400BadRequest("清单中的扩展包不能超过 200 个")
	}

	results := make([]StoreInstallListItemResult, len(packages))
	pending := make([]pendingStoreInstall, 0, len(packages))
	seen := make(map[string]struct{}, len(packages))
	for index, item := range packages {
		target, err := sm.ResolvePackage(item.ID, item.Version)
		if err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if _, exists := seen[target.ID]; exists {
			return nil, huma.Error400BadRequest("清单中存在重复的扩展包: " + target.ID)
		}
		seen[target.ID] = struct{}{}
		results[index] = StoreInstallListItemResult{ID: target.ID, Version: target.Version}
		pending = append(pending, pendingStoreInstall{target: target, resultIndex: index})
	}

	s.installStorePackageBatch(ctx, results, pending)

	summary := StoreInstallListResp{Items: results}
	for _, item := range results {
		switch item.Status {
		case "installed":
			summary.Installed++
		case "skipped":
			summary.Skipped++
		case "failed":
			summary.Failed++
		}
	}
	return response.NewItemResponse(summary), nil
}

type pendingStoreInstall struct {
	target      *dice.StorePackage
	resultIndex int
	lastError   error
}

func (s *Service) installStorePackageBatch(ctx context.Context, results []StoreInstallListItemResult, pending []pendingStoreInstall) {
	for len(pending) > 0 {
		nextPending := make([]pendingStoreInstall, 0, len(pending))
		installedThisPass := 0
		for _, item := range pending {
			status, err := s.installStorePackage(ctx, item.target, false)
			if err == nil {
				results[item.resultIndex].Status = status
				if status == "skipped" {
					results[item.resultIndex].Message = "已安装目标版本"
				} else {
					installedThisPass++
				}
				continue
			}

			var dependencyErr *dice.DependencyError
			if errors.As(err, &dependencyErr) {
				item.lastError = err
				nextPending = append(nextPending, item)
				continue
			}
			results[item.resultIndex].Status = "failed"
			results[item.resultIndex].Message = err.Error()
		}

		if len(nextPending) == 0 {
			break
		}
		if installedThisPass == 0 {
			for _, item := range nextPending {
				results[item.resultIndex].Status = "failed"
				results[item.resultIndex].Message = item.lastError.Error()
			}
			break
		}
		pending = nextPending
	}
}

func (s *Service) installStorePackage(ctx context.Context, target *dice.StorePackage, reinstallExactVersion bool) (string, error) {
	pm := s.packageManager()
	sm := s.storeManager()
	if pm == nil || sm == nil {
		return "", errors.New("extension managers unavailable")
	}
	if installedPkg, exists := pm.Get(target.ID); exists && installedPkg != nil && installedPkg.Manifest != nil {
		existingVer, existingErr := semver.NewVersion(installedPkg.Manifest.Package.Version)
		targetVer, targetErr := semver.NewVersion(target.Version)
		if existingErr == nil && targetErr == nil && targetVer.LessThan(existingVer) {
			return "", errors.New("当前已安装更高版本的扩展包")
		}
		if existingErr == nil && targetErr == nil && targetVer.Equal(existingVer) {
			if !reinstallExactVersion {
				return "skipped", nil
			}
			if err := pm.Uninstall(target.ID, sealpack.UninstallModeKeepData); err != nil {
				return "", err
			}
		}
	}

	if err := pm.InstallFromURLWithOptionsContext(ctx, target.Download.URL, dice.PackageDownloadOptions{
		Hashes:       target.Download.Hash,
		ExpectedSize: target.Download.Size,
	}); err != nil {
		return "", err
	}
	sm.RefreshInstalled([]*dice.StorePackage{target})
	return "installed", nil
}

func (s *Service) streamUploadMiddleware(api huma.API, preview bool) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		pm := s.packageManager()
		if pm == nil {
			_ = huma.WriteErr(api, ctx, http.StatusInternalServerError, "package manager unavailable")
			return
		}

		result := uploadStreamResult{done: true}
		var err error
		if preview {
			result.preview, err = pm.PreviewFromStreamContext(ctx.Context(), ctx.BodyReader())
		} else {
			err = pm.InstallFromStreamContext(ctx.Context(), ctx.BodyReader())
		}
		if err != nil {
			_ = huma.WriteErr(api, ctx, http.StatusBadRequest, err.Error())
			return
		}
		next(huma.WithValue(ctx, uploadStreamContextKey{}, result))
	}
}

func sameStorePackageVersion(left, right string) bool {
	leftVersion, leftErr := semver.NewVersion(strings.TrimSpace(left))
	rightVersion, rightErr := semver.NewVersion(strings.TrimSpace(right))
	return leftErr == nil && rightErr == nil && leftVersion.Equal(rightVersion)
}

func resolvePackageAssetPath(installPath string, assetPath string) (string, error) {
	root, err := filepath.Abs(filepath.Clean(installPath))
	if err != nil {
		return "", err
	}
	target, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(assetPath)))
	if err != nil {
		return "", err
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	resolvedTarget, err := filepath.EvalSymlinks(target)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(resolvedRoot, resolvedTarget)
	if err != nil || rel == "." || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", os.ErrPermission
	}
	return resolvedTarget, nil
}

func (s *Service) packageManager() *dice.PackageManager {
	if s == nil || s.dice == nil {
		return nil
	}
	return s.dice.PackageManager
}

func (s *Service) storeManager() *dice.StoreManager {
	if s == nil || s.dice == nil {
		return nil
	}
	return s.dice.StoreManager
}
