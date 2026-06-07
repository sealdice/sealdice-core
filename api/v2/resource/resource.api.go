package resource

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/sunshineplan/imgconv"

	"sealdice-core/dice"
	"sealdice-core/model/common/response"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
	imageRoot       = "data/images"
)

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
	huma.Get(grp, "/list", s.GetList, func(o *huma.Operation) {
		o.Description = "获取资源列表"
		o.Summary = "获取资源列表"
	})
	huma.Get(grp, "/download", s.Download, func(o *huma.Operation) {
		o.Description = "下载资源文件（流式附件下载）"
		o.Summary = "下载资源文件"
	})
	huma.Get(grp, "/data", s.GetData, func(o *huma.Operation) {
		o.Description = "获取资源图片数据，可按需返回缩略图"
		o.Summary = "获取资源图片数据"
	})
}

func (s *Service) RegisterProtectedRoutes(grp *huma.Group) {
	huma.Post(grp, "/upload", s.Upload, func(o *huma.Operation) {
		o.Description = "上传资源文件"
		o.Summary = "上传资源文件"
	})
	huma.Post(grp, "/delete", s.Delete, func(o *huma.Operation) {
		o.Description = "删除资源文件"
		o.Summary = "删除资源文件"
	})
}

func (s *Service) GetList(_ context.Context, req *ListQuery) (*response.ItemResponse[ResourceListResp], error) {
	resourceType := strings.TrimSpace(req.Type)
	if resourceType == "" {
		resourceType = string(ResourceTypeImage)
	}
	if resourceType != string(ResourceTypeImage) {
		return response.NewItemResponse(ResourceListResp{
			List:     []*ResourceItem{},
			Total:    0,
			Page:     normalizePage(req.Page),
			PageSize: normalizePageSize(req.PageSize),
		}), nil
	}

	items, err := scanImageResources()
	if err != nil {
		return nil, huma.Error500InternalServerError("获取资源列表失败")
	}
	items = filterResources(items, req.Keyword)
	sortResources(items, req.SortBy, req.SortOrder)

	page := normalizePage(req.Page)
	pageSize := normalizePageSize(req.PageSize)
	total := int64(len(items))
	paged := paginateResources(items, page, pageSize)

	return response.NewItemResponse(ResourceListResp{
		List:     paged,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}), nil
}

func (s *Service) Upload(_ context.Context, req *UploadReq) (*response.ItemResponse[ResourceUploadResp], error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(ResourceUploadResp{Success: false, TestMode: true}), nil
	}

	files, err := extractUploadFiles(req.RawBody)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.File != nil {
			defer func(file huma.FormFile) {
				_ = file.Close()
			}(file)
		}
	}

	if err = os.MkdirAll(imageRoot, 0o755); err != nil {
		return nil, huma.Error500InternalServerError("创建资源目录失败")
	}

	failed := make([]string, 0)
	for _, file := range files {
		filename := sanitizeUploadFilename(file.Filename)
		if filename == "" || !isAllowedImageFile(filename) {
			failed = append(failed, file.Filename)
			continue
		}

		dst, createErr := os.Create(filepath.Join(imageRoot, filename))
		if createErr != nil {
			failed = append(failed, file.Filename)
			continue
		}
		if _, copyErr := io.Copy(dst, file.File); copyErr != nil {
			failed = append(failed, file.Filename)
		}
		_ = dst.Close()
	}

	if len(failed) > 0 {
		return nil, huma.Error400BadRequest("部分资源上传失败：" + strings.Join(failed, "、"))
	}

	return response.NewItemResponse(ResourceUploadResp{Success: true}), nil
}

func (s *Service) Delete(_ context.Context, req *DeleteReq) (*response.ItemResponse[response.SimpleOK], error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(response.SimpleOK{Success: false}), nil
	}

	path, err := safeImagePath(req.Body.Path)
	if err != nil {
		return nil, huma.Error403Forbidden("invalid path")
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return nil, huma.Error404NotFound("资源不存在")
	}
	if err = os.Remove(path); err != nil {
		return nil, huma.Error500InternalServerError("删除资源失败")
	}
	return response.NewItemResponse(response.SimpleOK{Success: true}), nil
}

func (s *Service) Download(_ context.Context, req *ResourcePathQuery) (*huma.StreamResponse, error) {
	path, err := safeImagePath(req.Path)
	if err != nil {
		return nil, huma.Error403Forbidden("invalid path")
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return nil, huma.Error404NotFound("资源不存在")
	}

	filename := info.Name()
	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(filename)))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return &huma.StreamResponse{
		Body: func(ctx huma.Context) {
			ctx.SetHeader("Content-Type", contentType)
			ctx.SetHeader("Content-Disposition", `attachment; filename="`+filename+`"`)
			ctx.SetHeader("Content-Length", strconv.FormatInt(info.Size(), 10))

			file, openErr := os.Open(path)
			if openErr != nil {
				return
			}
			defer func() {
				_ = file.Close()
			}()
			_, _ = io.Copy(ctx.BodyWriter(), file)
			if flusher, ok := ctx.BodyWriter().(http.Flusher); ok {
				flusher.Flush()
			}
		},
	}, nil
}

func (s *Service) GetData(_ context.Context, req *ResourceDataQuery) (*huma.StreamResponse, error) {
	path, err := safeImagePath(req.Path)
	if err != nil {
		return nil, huma.Error403Forbidden("invalid path")
	}
	if _, err = os.Stat(path); err != nil {
		return nil, huma.Error404NotFound("资源不存在")
	}

	img, err := imgconv.Open(path)
	if err != nil {
		return nil, huma.Error400BadRequest("图片资源无法读取")
	}
	if req.Thumbnail {
		img = imgconv.Resize(img, &imgconv.ResizeOption{Width: 96})
	}

	buf := new(bytes.Buffer)
	if err = imgconv.Write(buf, img, &imgconv.FormatOption{Format: imgconv.PNG}); err != nil {
		return nil, huma.Error500InternalServerError("图片资源转换失败")
	}
	data := buf.Bytes()

	return &huma.StreamResponse{
		Body: func(ctx huma.Context) {
			ctx.SetHeader("Content-Type", "image/png")
			ctx.SetHeader("Content-Length", strconv.Itoa(len(data)))
			_, _ = ctx.BodyWriter().Write(data)
		},
	}, nil
}

func scanImageResources() ([]*ResourceItem, error) {
	items := make([]*ResourceItem, 0)
	if _, err := os.Stat(imageRoot); errors.Is(err, os.ErrNotExist) {
		return items, nil
	}

	err := filepath.Walk(imageRoot, func(path string, info fs.FileInfo, err error) error {
		if err != nil || info == nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		resourceType, ext := getResourceType(path)
		if resourceType != ResourceTypeImage {
			return nil
		}
		items = append(items, &ResourceItem{
			Type: resourceType,
			Name: info.Name(),
			Ext:  ext,
			Path: filepath.ToSlash(path),
			Size: info.Size(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return items, nil
}

func getResourceType(filename string) (ResourceType, string) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif":
		return ResourceTypeImage, ext
	default:
		return ResourceTypeUnknown, ext
	}
}

func filterResources(items []*ResourceItem, keyword string) []*ResourceItem {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" {
		return items
	}

	filtered := make([]*ResourceItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		if strings.Contains(strings.ToLower(item.Name), keyword) ||
			strings.Contains(strings.ToLower(item.Path), keyword) ||
			strings.Contains(strings.ToLower(item.Ext), keyword) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func sortResources(items []*ResourceItem, sortBy string, sortOrder string) {
	sortBy = strings.TrimSpace(sortBy)
	sortOrder = strings.TrimSpace(sortOrder)
	if sortOrder == "" {
		sortOrder = "asc"
	}

	sort.SliceStable(items, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "size":
			less = items[i].Size < items[j].Size
		case "ext":
			less = items[i].Ext < items[j].Ext
		case "path":
			less = strings.ToLower(items[i].Path) < strings.ToLower(items[j].Path)
		default:
			less = strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
		}
		if sortOrder == "desc" {
			return !less
		}
		return less
	})
}

func paginateResources(items []*ResourceItem, page int, pageSize int) []*ResourceItem {
	if len(items) == 0 {
		return []*ResourceItem{}
	}
	start := (page - 1) * pageSize
	if start >= len(items) {
		return []*ResourceItem{}
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}

func normalizePage(page int) int {
	if page <= 0 {
		return 1
	}
	return page
}

func normalizePageSize(pageSize int) int {
	if pageSize <= 0 {
		return defaultPageSize
	}
	if pageSize > maxPageSize {
		return maxPageSize
	}
	return pageSize
}

func extractUploadFiles(raw huma.MultipartFormFiles[UploadForm]) ([]huma.FormFile, error) {
	data := raw.Data()
	if data != nil && len(data.Files) > 0 {
		return data.Files, nil
	}
	if raw.Form == nil {
		return nil, huma.Error400BadRequest("missing files")
	}

	headers := raw.Form.File["files"]
	if len(headers) == 0 {
		headers = raw.Form.File["file"]
	}
	if len(headers) == 0 {
		return nil, huma.Error400BadRequest("missing files")
	}

	files := make([]huma.FormFile, 0, len(headers))
	for _, fh := range headers {
		file, err := fh.Open()
		if err != nil {
			return nil, huma.Error400BadRequest("failed to open file")
		}
		files = append(files, huma.FormFile{
			File:        file,
			ContentType: fh.Header.Get("Content-Type"),
			IsSet:       true,
			Size:        fh.Size,
			Filename:    fh.Filename,
		})
	}
	return files, nil
}

func sanitizeUploadFilename(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "\x00", "")
	return strings.TrimSpace(name)
}

func isAllowedImageFile(filename string) bool {
	resourceType, _ := getResourceType(filename)
	return resourceType == ResourceTypeImage
}

func safeImagePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("empty path")
	}

	rootAbs, err := filepath.Abs(imageRoot)
	if err != nil {
		return "", err
	}
	targetAbs, err := filepath.Abs(filepath.FromSlash(path))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return "", err
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", errors.New("invalid path")
	}
	return targetAbs, nil
}
