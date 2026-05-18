package customreply

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	cmm "sealdice-core/api/v2/model/common"
	customreplym "sealdice-core/api/v2/model/customreply"
	"sealdice-core/dice"
	"sealdice-core/model/common/request"
	"sealdice-core/model/common/response"
	"sealdice-core/utils/paginate"
	"sealdice-core/utils/paginate/slicep"
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
	huma.Get(grp, "/files", s.GetFileList, func(o *huma.Operation) {
		o.Description = "获取自定义回复文件列表"
	})
	huma.Get(grp, "/files/{filename}", s.GetConfig, func(o *huma.Operation) {
		o.Description = "获取自定义回复文件详情"
	})
	huma.Get(grp, "/files/{filename}/conditions", s.GetConditions, func(o *huma.Operation) {
		o.Description = "获取自定义回复前置条件分页"
	})
	huma.Get(grp, "/files/{filename}/rules", s.GetRules, func(o *huma.Operation) {
		o.Description = "获取自定义回复规则分页"
	})
	huma.Get(grp, "/files/{filename}/download", s.Download, func(o *huma.Operation) {
		o.Description = "下载自定义回复文件"
	})
	huma.Get(grp, "/debug-mode", s.GetDebugMode, func(o *huma.Operation) {
		o.Description = "获取自定义回复调试模式"
	})
}

func (s *Service) RegisterProtectedRoutes(grp *huma.Group) {
	huma.Post(grp, "/files", s.CreateFile, func(o *huma.Operation) {
		o.Description = "新建自定义回复文件"
	})
	huma.Delete(grp, "/files/{filename}", s.DeleteFile, func(o *huma.Operation) {
		o.Description = "删除自定义回复文件"
	})
	huma.Put(grp, "/files/{filename}", s.SaveConfig, func(o *huma.Operation) {
		o.Description = "保存自定义回复配置"
	})
	huma.Post(grp, "/files/upload", s.Upload, func(o *huma.Operation) {
		o.Description = "上传自定义回复文件"
	})
	huma.Put(grp, "/debug-mode", s.SetDebugMode, func(o *huma.Operation) {
		o.Description = "设置自定义回复调试模式"
	})
}

func (s *Service) GetFileList(_ context.Context, req *customreplym.FileListQuery) (*response.ItemResponse[customreplym.ReplyFileListResp], error) {
	items := make([]*customreplym.FileInfo, 0, len(s.dice.CustomReplyConfig))
	for _, item := range s.dice.CustomReplyConfig {
		if item == nil {
			continue
		}
		items = append(items, &customreplym.FileInfo{
			Enable:          item.Enable,
			Filename:        item.Filename,
			CreateTimestamp: item.CreateTimestamp,
			UpdateTimestamp: item.UpdateTimestamp,
			ItemCount:       len(item.Items),
		})
	}
	keyword := strings.TrimSpace(req.Keyword)
	if keyword != "" {
		filtered := make([]*customreplym.FileInfo, 0, len(items))
		for _, item := range items {
			if strings.Contains(strings.ToLower(item.Filename), strings.ToLower(keyword)) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	sortBy := strings.TrimSpace(req.SortBy)
	sortOrder := strings.TrimSpace(req.SortOrder)
	if sortOrder == "" {
		sortOrder = "desc"
	}
	sort.SliceStable(items, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "name":
			less = strings.ToLower(items[i].Filename) < strings.ToLower(items[j].Filename)
		default:
			less = items[i].UpdateTimestamp < items[j].UpdateTimestamp
		}
		if sortOrder == "asc" {
			return less
		}
		return !less
	})
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 30
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}
	var paged []*customreplym.FileInfo
	pager := paginate.SimplePaginate(slicep.Adapter(items), int64(pageSize), int64(page))
	total, err := pager.GetTotal()
	if err != nil {
		return nil, huma.Error500InternalServerError("获取文件列表失败")
	}
	if err := pager.Get(&paged); err != nil {
		return nil, huma.Error500InternalServerError("获取文件列表失败")
	}
	return response.NewItemResponse(customreplym.ReplyFileListResp{
		List:     paged,
		Total:    total,
		Page:     int(pager.GetCurrentPage()),
		PageSize: int(pager.GetListRows()),
	}), nil
}

func (s *Service) GetConfig(_ context.Context, req *customreplym.FilenamePath) (*response.ItemResponse[customreplym.ReplyFileDetail], error) {
	filename, err := sanitizeFilename(req.Filename)
	if err != nil {
		return nil, err
	}
	rc, err := dice.CustomReplyConfigRead(s.dice, filename)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	return response.NewItemResponse(customreplym.ReplyFileDetail{
		Enable:          rc.Enable,
		Interval:        rc.Interval,
		Name:            rc.Name,
		Author:          rc.Author,
		Version:         rc.Version,
		CreateTimestamp: rc.CreateTimestamp,
		UpdateTimestamp: rc.UpdateTimestamp,
		Desc:            rc.Desc,
		StoreID:         rc.StoreID,
		Conditions:      rc.Conditions,
		Filename:        rc.Filename,
		ItemCount:       len(rc.Items),
	}), nil
}

func (s *Service) GetRules(_ context.Context, req *customreplym.RulePageQuery) (*response.ItemResponse[customreplym.RulePageResp], error) {
	filename, err := sanitizeFilename(req.Filename)
	if err != nil {
		return nil, err
	}
	rc, err := dice.CustomReplyConfigRead(s.dice, filename)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}
	items := make([]*customreplym.RuleInfo, 0, len(rc.Items))
	for index, item := range rc.Items {
		items = append(items, &customreplym.RuleInfo{
			Index: index,
			Item:  item,
		})
	}
	var paged []*customreplym.RuleInfo
	pager := paginate.SimplePaginate(slicep.Adapter(items), int64(pageSize), int64(page))
	total, err := pager.GetTotal()
	if err != nil {
		return nil, huma.Error500InternalServerError("获取规则列表失败")
	}
	if err := pager.Get(&paged); err != nil {
		return nil, huma.Error500InternalServerError("获取规则列表失败")
	}
	return response.NewItemResponse(customreplym.RulePageResp{
		List:     paged,
		Total:    total,
		Page:     int(pager.GetCurrentPage()),
		PageSize: int(pager.GetListRows()),
	}), nil
}

func (s *Service) GetConditions(_ context.Context, req *customreplym.ConditionPageQuery) (*response.ItemResponse[customreplym.ConditionPageResp], error) {
	filename, err := sanitizeFilename(req.Filename)
	if err != nil {
		return nil, err
	}
	rc, err := dice.CustomReplyConfigRead(s.dice, filename)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}
	items := make([]*customreplym.ConditionInfo, 0, len(rc.Conditions))
	for index, item := range rc.Conditions {
		items = append(items, &customreplym.ConditionInfo{
			Index: index,
			Item:  item,
		})
	}
	var paged []*customreplym.ConditionInfo
	pager := paginate.SimplePaginate(slicep.Adapter(items), int64(pageSize), int64(page))
	total, err := pager.GetTotal()
	if err != nil {
		return nil, huma.Error500InternalServerError("获取前置条件列表失败")
	}
	if err := pager.Get(&paged); err != nil {
		return nil, huma.Error500InternalServerError("获取前置条件列表失败")
	}
	return response.NewItemResponse(customreplym.ConditionPageResp{
		List:     paged,
		Total:    total,
		Page:     int(pager.GetCurrentPage()),
		PageSize: int(pager.GetListRows()),
	}), nil
}

func (s *Service) CreateFile(_ context.Context, req *customreplym.FileReq) (*response.ItemResponse[cmm.SimpleOK], error) {
	filename, err := sanitizeFilename(req.Body.Body.Filename)
	if err != nil {
		return nil, err
	}
	if dice.CustomReplyConfigCheckExists(s.dice, filename) {
		return response.NewItemResponse(cmm.SimpleOK{Success: false}), nil
	}
	rc := dice.CustomReplyConfigNew(s.dice, filename)
	dice.ReplyReload(s.dice)
	return response.NewItemResponse(cmm.SimpleOK{Success: rc != nil}), nil
}

func (s *Service) DeleteFile(_ context.Context, req *customreplym.FilenamePath) (*response.ItemResponse[cmm.SimpleOK], error) {
	filename, err := sanitizeFilename(req.Filename)
	if err != nil {
		return nil, err
	}
	success := dice.CustomReplyConfigDelete(s.dice, filename)
	if success {
		dice.ReplyReload(s.dice)
	}
	return response.NewItemResponse(cmm.SimpleOK{Success: success}), nil
}

func (s *Service) SaveConfig(_ context.Context, req *customreplym.SaveReq) (*response.ItemResponse[cmm.SimpleOK], error) {
	filename, err := sanitizeFilename(req.Filename)
	if err != nil {
		return nil, err
	}
	rc := req.Body.Body
	rc.Filename = filename
	rc.UpdateTimestamp = time.Now().Unix()
	if rc.CreateTimestamp == 0 {
		rc.CreateTimestamp = rc.UpdateTimestamp
	}
	rc.Clean()
	for index, item := range s.dice.CustomReplyConfig {
		if item != nil && item.Filename == filename {
			s.dice.CustomReplyConfig[index].Enable = rc.Enable
			s.dice.CustomReplyConfig[index].Conditions = rc.Conditions
			s.dice.CustomReplyConfig[index].Items = rc.Items
			s.dice.CustomReplyConfig[index].Interval = rc.Interval
			s.dice.CustomReplyConfig[index].Name = rc.Name
			s.dice.CustomReplyConfig[index].Author = rc.Author
			s.dice.CustomReplyConfig[index].Version = rc.Version
			s.dice.CustomReplyConfig[index].CreateTimestamp = rc.CreateTimestamp
			s.dice.CustomReplyConfig[index].UpdateTimestamp = rc.UpdateTimestamp
			s.dice.CustomReplyConfig[index].Desc = rc.Desc
			s.dice.CustomReplyConfig[index].StoreID = rc.StoreID
			break
		}
	}
	rc.Save(s.dice)
	dice.ReplyReload(s.dice)
	return response.NewItemResponse(cmm.SimpleOK{Success: true}), nil
}

func (s *Service) Download(_ context.Context, req *customreplym.FilenamePath) (*huma.StreamResponse, error) {
	filename, err := sanitizeFilename(req.Filename)
	if err != nil {
		return nil, err
	}
	fp := s.dice.GetExtConfigFilePath("reply", filename)
	info, statErr := os.Stat(fp)
	if statErr != nil || info.IsDir() {
		return nil, huma.Error404NotFound("not found")
	}
	return &huma.StreamResponse{
		Body: func(ctx huma.Context) {
			ctx.SetHeader("Content-Type", "application/x-yaml")
			ctx.SetHeader("Content-Disposition", "attachment; filename=\""+filename+"\"")
			ctx.SetHeader("Content-Length", strconv.FormatInt(info.Size(), 10))
			f, openErr := os.Open(fp)
			if openErr != nil {
				return
			}
			defer func() { _ = f.Close() }()
			w := ctx.BodyWriter()
			_, _ = io.Copy(w, f)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		},
	}, nil
}

func (s *Service) Upload(_ context.Context, req *customreplym.UploadReq) (*response.ItemResponse[cmm.SimpleOK], error) {
	data := req.RawBody.Data()
	if data == nil || !data.File.IsSet {
		if req.RawBody.Form == nil || len(req.RawBody.Form.File["file"]) == 0 {
			return nil, huma.Error400BadRequest("missing file")
		}
		fh := req.RawBody.Form.File["file"][0]
		file, openErr := fh.Open()
		if openErr != nil {
			return nil, huma.Error400BadRequest("failed to open file")
		}
		defer func() { _ = file.Close() }()
		data = &customreplym.UploadForm{
			File: huma.FormFile{
				File:        file,
				ContentType: fh.Header.Get("Content-Type"),
				IsSet:       true,
				Size:        fh.Size,
				Filename:    fh.Filename,
			},
		}
	}
	if s.dm.JustForTest {
		return response.NewItemResponse(cmm.SimpleOK{Success: true}), nil
	}

	filename, err := sanitizeFilename(data.File.Filename)
	if err != nil {
		return nil, err
	}
	if dice.CustomReplyConfigCheckExists(s.dice, filename) {
		return nil, huma.Error409Conflict("file already exists")
	}
	fp := s.dice.GetExtConfigFilePath("reply", filename)
	dst, err := os.Create(fp)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to create file")
	}
	defer func() { _ = dst.Close() }()
	if _, err := io.Copy(dst, data.File); err != nil {
		return nil, huma.Error500InternalServerError("failed to write file")
	}
	dice.ReplyReload(s.dice)
	return response.NewItemResponse(cmm.SimpleOK{Success: true}), nil
}

func (s *Service) GetDebugMode(_ context.Context, _ *request.Empty) (*response.ItemResponse[customreplym.DebugModeResp], error) {
	return response.NewItemResponse(customreplym.DebugModeResp{
		Value: s.dice.Config.ReplyDebugMode,
	}), nil
}

func (s *Service) SetDebugMode(_ context.Context, req *customreplym.DebugModeReq) (*response.ItemResponse[customreplym.DebugModeResp], error) {
	s.dice.Config.ReplyDebugMode = req.Body.Body.Value
	s.dice.MarkModified()
	return response.NewItemResponse(customreplym.DebugModeResp{
		Value: s.dice.Config.ReplyDebugMode,
	}), nil
}

func sanitizeFilename(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", huma.Error400BadRequest("missing filename")
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return "", huma.Error400BadRequest("invalid filename")
	}
	base := filepath.Base(name)
	if base != name || base == "." {
		return "", huma.Error400BadRequest("invalid filename")
	}
	return name, nil
}
