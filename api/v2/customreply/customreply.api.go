package customreply

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"sealdice-core/dice"
	"sealdice-core/model/common/request"
	"sealdice-core/model/common/response"
	"sealdice-core/utils/paginate"
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

func (s *Service) Dice() *dice.Dice {
	return s.dice
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
		// 导入允许 20 MiB；完整 JSON 保存还需容纳字符串转义和字段开销。
		o.MaxBodyBytes = 128 << 20
	})
	huma.Put(grp, "/debug-mode", s.SetDebugMode, func(o *huma.Operation) {
		o.Description = "设置自定义回复调试模式"
	})
}

func (s *Service) GetFileList(_ context.Context, req *FileListQuery) (*response.ItemResponse[ReplyFileListResp], error) {
	items := make([]*FileInfo, 0, len(s.dice.CustomReplyConfig))
	for _, item := range s.dice.CustomReplyConfig {
		if item == nil {
			continue
		}
		items = append(items, &FileInfo{
			Enable:          item.Enable,
			Filename:        item.Filename,
			CreateTimestamp: item.CreateTimestamp,
			UpdateTimestamp: item.UpdateTimestamp,
			ItemCount:       len(item.Items),
			PackageID:       item.PackageID,
		})
	}
	keyword := strings.TrimSpace(req.Keyword)
	if keyword != "" {
		filtered := make([]*FileInfo, 0, len(items))
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
	return response.NewItemResponse(ReplyFileListResp(paginate.Slice(items, req.Page, pageSize))), nil
}

func (s *Service) GetConfig(_ context.Context, req *FilenamePath) (*response.ItemResponse[ReplyFileDetail], error) {
	filename, err := sanitizeFilename(req.Filename)
	if err != nil {
		return nil, err
	}
	rc, err := dice.CustomReplyConfigRead(s.dice, filename)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	for _, item := range s.dice.CustomReplyConfig {
		if item != nil && item.Filename == rc.Filename && rc.PackageID == "" {
			rc.PackageID = item.PackageID
			break
		}
	}
	return response.NewItemResponse(ReplyFileDetail{
		Enable:          rc.Enable,
		VMVersion:       displayReplyVMVersion(rc.VMVersion),
		Interval:        rc.Interval,
		Name:            rc.Name,
		Author:          rc.Author,
		Version:         rc.Version,
		CreateTimestamp: rc.CreateTimestamp,
		UpdateTimestamp: rc.UpdateTimestamp,
		Desc:            rc.Desc,
		StoreID:         rc.StoreID,
		PackageID:       rc.PackageID,
		Conditions:      rc.Conditions,
		Filename:        rc.Filename,
		ItemCount:       len(rc.Items),
	}), nil
}

func (s *Service) GetRules(_ context.Context, req *RulePageQuery) (*response.ItemResponse[RulePageResp], error) {
	filename, err := sanitizeFilename(req.Filename)
	if err != nil {
		return nil, err
	}
	rc, err := dice.CustomReplyConfigRead(s.dice, filename)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	items := make([]*RuleInfo, 0, len(rc.Items))
	for index, item := range rc.Items {
		items = append(items, &RuleInfo{
			Index: index,
			Item:  item,
		})
	}
	return response.NewItemResponse(RulePageResp(paginate.Slice(items, req.Page, req.PageSize))), nil
}

func (s *Service) GetConditions(_ context.Context, req *ConditionPageQuery) (*response.ItemResponse[ConditionPageResp], error) {
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
	items := make([]*ConditionInfo, 0, len(rc.Conditions))
	for index, item := range rc.Conditions {
		items = append(items, &ConditionInfo{
			Index: index,
			Item:  item,
		})
	}
	return response.NewItemResponse(ConditionPageResp(paginate.Slice(items, req.Page, pageSize))), nil
}

func (s *Service) CreateFile(_ context.Context, req *FileReq) (*response.ItemResponse[response.SimpleOK], error) {
	filename, err := sanitizeFilename(req.Body.Filename)
	if err != nil {
		return nil, err
	}
	if dice.CustomReplyConfigCheckExists(s.dice, filename) {
		return response.NewItemResponse(response.SimpleOK{Success: false}), nil
	}
	vmVersion := req.Body.VMVersion
	if vmVersion == "" {
		vmVersion = dice.ReplyVMVersionV2
	}
	vmVersion, ok := dice.ValidateReplyVMVersion(vmVersion)
	if !ok {
		return nil, huma.Error400BadRequest("不支持的自定义回复 VM 版本")
	}
	rc := dice.CustomReplyConfigNewWithVMVersion(s.dice, filename, vmVersion)
	dice.ReplyReload(s.dice)
	return response.NewItemResponse(response.SimpleOK{Success: rc != nil}), nil
}

func (s *Service) DeleteFile(_ context.Context, req *FilenamePath) (*response.ItemResponse[response.SimpleOK], error) {
	filename, err := sanitizeFilename(req.Filename)
	if err != nil {
		return nil, err
	}
	success := dice.CustomReplyConfigDelete(s.dice, filename)
	if success {
		dice.ReplyReload(s.dice)
	}
	return response.NewItemResponse(response.SimpleOK{Success: success}), nil
}

func (s *Service) SaveConfig(_ context.Context, req *SaveReq) (*response.ItemResponse[response.SimpleOK], error) {
	rc := req.Body.ReplyConfig
	rc.Filename = req.Filename
	if err := s.dice.SaveReplyConfig(&rc, req.Body.VMVersion); err != nil {
		if errors.Is(err, dice.ErrInvalidConfig) {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if errors.Is(err, dice.ErrReplyPackageNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		s.dice.Logger.Errorf("保存自定义回复文件失败: %v", err)
		return nil, huma.Error500InternalServerError("保存自定义回复文件失败")
	}
	return response.NewItemResponse(response.SimpleOK{Success: true}), nil
}

func displayReplyVMVersion(version string) string {
	if strings.EqualFold(strings.TrimSpace(version), dice.ReplyVMVersionV2) {
		return dice.ReplyVMVersionV2
	}
	return dice.ReplyVMVersionV1
}

func (s *Service) Download(_ context.Context, req *FilenamePath) (*huma.StreamResponse, error) {
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

func (s *Service) GetDebugMode(_ context.Context, _ *request.Empty) (*response.ItemResponse[DebugModeResp], error) {
	return response.NewItemResponse(DebugModeResp{
		Value: s.dice.Config.ReplyDebugMode,
	}), nil
}

func (s *Service) SetDebugMode(_ context.Context, req *DebugModeReq) (*response.ItemResponse[DebugModeResp], error) {
	s.dice.Config.ReplyDebugMode = req.Body.Value
	s.dice.MarkModified()
	return response.NewItemResponse(DebugModeResp{
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
	if base != name || base == "." || base == ".." {
		return "", huma.Error400BadRequest("invalid filename")
	}
	return name, nil
}
