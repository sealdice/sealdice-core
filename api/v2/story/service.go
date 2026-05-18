package story

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/danielgtaylor/huma/v2"

	storym "sealdice-core/api/v2/model/story"
	"sealdice-core/dice"
	"sealdice-core/dice/service"
	"sealdice-core/model"
	"sealdice-core/model/common/request"
	"sealdice-core/model/common/response"
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
	huma.Get(grp, "/info", s.GetInfo, func(o *huma.Operation) {
		o.Description = "获取跑团日志统计信息"
	})
	huma.Get(grp, "/logs/page", s.GetLogPage, func(o *huma.Operation) {
		o.Description = "获取跑团日志分页"
	})
	huma.Get(grp, "/items/page", s.GetItemPage, func(o *huma.Operation) {
		o.Description = "获取跑团日志消息分页"
	})
	huma.Get(grp, "/cleanup/preview", s.PreviewCleanup, func(o *huma.Operation) {
		o.Description = "预览跑团日志清理结果"
	})
	huma.Get(grp, "/backup/list", s.GetBackupList, func(o *huma.Operation) {
		o.Description = "获取跑团日志备份列表"
	})
	huma.Get(grp, "/backup/download", s.DownloadBackup, func(o *huma.Operation) {
		o.Description = "下载跑团日志备份"
	})
}

func (s *Service) RegisterProtectedRoutes(grp *huma.Group) {
	huma.Delete(grp, "/log", s.DeleteLog, func(o *huma.Operation) {
		o.Description = "删除跑团日志"
	})
	huma.Post(grp, "/upload-log", s.UploadLog, func(o *huma.Operation) {
		o.Description = "提取跑团日志"
	})
	huma.Post(grp, "/cleanup", s.Cleanup, func(o *huma.Operation) {
		o.Description = "清理超过指定月数未更新的跑团日志"
	})
	huma.Post(grp, "/backup/batch-delete", s.BatchDeleteBackup, func(o *huma.Operation) {
		o.Description = "批量删除跑团日志备份"
	})
}

func (s *Service) GetInfo(_ context.Context, _ *request.Empty) (*response.ItemResponse[storym.StoryInfoResp], error) {
	info, err := service.LogGetInfo(s.dice.DBOperator)
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}
	return response.NewItemResponse(storym.StoryInfoResp{
		TotalLogs:    info[0],
		TotalItems:   info[1],
		CurrentLogs:  info[2],
		CurrentItems: info[3],
	}), nil
}

func (s *Service) GetLogPage(_ context.Context, req *storym.LogPageQuery) (*response.ItemResponse[storym.LogPageResp], error) {
	query := service.QueryLogPage{
		PageNum:  req.PageNum,
		PageSize: req.PageSize,
		Name:     req.Name,
		GroupID:  req.GroupID,
	}
	if query.PageNum < 1 {
		query.PageNum = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if req.CreatedTimeBegin > 0 {
		query.CreatedTimeBegin = strconv.FormatInt(req.CreatedTimeBegin, 10)
	}
	if req.CreatedTimeEnd > 0 {
		query.CreatedTimeEnd = strconv.FormatInt(req.CreatedTimeEnd, 10)
	}

	total, page, err := service.LogGetLogPage(s.dice.DBOperator, &query)
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}

	items := make([]storym.StoryLogView, 0, len(page))
	for _, item := range page {
		if item == nil {
			continue
		}
		items = append(items, storym.StoryLogView{
			ID:         item.ID,
			Name:       item.Name,
			GroupID:    item.GroupID,
			CreatedAt:  item.CreatedAt,
			UpdatedAt:  item.UpdatedAt,
			Size:       item.Size,
			UploadURL:  item.UploadURL,
			UploadTime: int64(item.UploadTime),
			LinkState:  service.BuildLogLinkState(item),
		})
	}

	return response.NewItemResponse(storym.LogPageResp{
		Result:   true,
		Total:    total,
		Data:     items,
		PageNum:  query.PageNum,
		PageSize: len(items),
	}), nil
}

func (s *Service) GetItemPage(_ context.Context, req *storym.ItemPageQuery) (*response.ItemResponse[storym.LogLinePageResp], error) {
	query := service.QueryLogLinePage{
		LogID:    req.LogID,
		PageNum:  req.PageNum,
		PageSize: req.PageSize,
		GroupID:  req.GroupID,
		LogName:  req.LogName,
	}
	if query.PageNum < 1 {
		query.PageNum = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 10
	}

	lines, err := service.LogGetLinePage(s.dice.DBOperator, &query)
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}
	return response.NewItemResponse(lines), nil
}

func (s *Service) DeleteLog(_ context.Context, req *storym.DeleteLogReq) (*response.ItemResponse[storym.DeleteLogResp], error) {
	if req.Body.Body.ID == 0 {
		return nil, huma.Error400BadRequest("缺少 id")
	}
	err := service.LogDeleteByID(s.dice.DBOperator, req.Body.Body.ID)
	if err != nil {
		if errors.Is(err, service.ErrLogNotFound) {
			return response.NewItemResponse(storym.DeleteLogResp{Success: false}), nil
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}
	return response.NewItemResponse(storym.DeleteLogResp{Success: true}), nil
}

func (s *Service) UploadLog(_ context.Context, req *storym.UploadLogReq) (*response.ItemResponse[storym.UploadLogResp], error) {
	if req.Body.Body.ID == 0 {
		return nil, huma.Error400BadRequest("缺少 id")
	}
	logInfo, err := service.LogGetByID(s.dice.DBOperator, req.Body.Body.ID)
	if err != nil {
		if errors.Is(err, service.ErrLogNotFound) {
			return nil, huma.Error404NotFound("未找到该条日志")
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}
	url0, uploadTime, updateTime, err := service.LogGetUploadInfoByID(s.dice.DBOperator, req.Body.Body.ID)
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}
	if !req.Body.Body.Force && url0 != "" && uploadTime > updateTime {
		return response.NewItemResponse(storym.UploadLogResp{
			URL:    url0,
			Reused: true,
		}), nil
	}
	unofficial, url, err := logSendToBackend(s.dice, logInfo, req.Body.Body.Force)
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}
	return response.NewItemResponse(storym.UploadLogResp{
		URL:        url,
		Unofficial: unofficial,
		Reused:     false,
		Forced:     req.Body.Body.Force,
	}), nil
}

func (s *Service) GetBackupList(_ context.Context, _ *request.Empty) (*response.ItemResponse[storym.BackupListResp], error) {
	list, err := dice.StoryLogBackupList(s.dice)
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}
	return response.NewItemResponse(storym.BackupListResp{
		Result: true,
		Data:   list,
	}), nil
}

func (s *Service) DownloadBackup(ctx context.Context, req *storym.BackupDownloadQuery) (*huma.StreamResponse, error) {
	path, err := dice.StoryLogBackupDownloadPath(s.dice, req.Name)
	if err != nil {
		return nil, huma.Error404NotFound(err.Error())
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}
	return &huma.StreamResponse{
		Body: func(w huma.Context) {
			defer f.Close()
			w.SetHeader("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, req.Name))
			w.SetHeader("Content-Type", "application/octet-stream")
			_, _ = io.Copy(w.BodyWriter(), f)
		},
	}, nil
}

func (s *Service) BatchDeleteBackup(_ context.Context, req *storym.BackupBatchDeleteReq) (*response.ItemResponse[storym.BackupBatchDeleteResp], error) {
	fails := dice.StoryLogBackupBatchDelete(s.dice, req.Body.Body.Names)
	return response.NewItemResponse(storym.BackupBatchDeleteResp{
		Result: len(fails) == 0,
		Fails:  fails,
	}), nil
}

func (s *Service) PreviewCleanup(_ context.Context, req *storym.CleanupPreviewQuery) (*response.ItemResponse[storym.CleanupPreviewResp], error) {
	if req.Months < 0 {
		return nil, huma.Error400BadRequest("months 不能小于 0")
	}
	preview, err := service.LogGetCleanupPreviewByMonths(s.dice.DBOperator, req.Months)
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}
	return response.NewItemResponse(storym.CleanupPreviewResp{
		Logs:          preview.Logs,
		Items:         preview.Items,
		OldestUpdated: preview.OldestUpdated,
		NewestUpdated: preview.NewestUpdated,
		CanVacuum:     preview.CanVacuum,
	}), nil
}

func (s *Service) Cleanup(_ context.Context, req *storym.CleanupReq) (*response.ItemResponse[storym.CleanupResp], error) {
	if req.Body.Body.Months < 0 {
		return nil, huma.Error400BadRequest("months 不能小于 0")
	}
	result, err := service.LogCleanupByMonths(s.dice.DBOperator, req.Body.Body.Months, req.Body.Body.Vacuum)
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}
	return response.NewItemResponse(storym.CleanupResp{
		Logs:     result.Logs,
		Items:    result.Items,
		Vacuumed: result.Vacuumed,
	}), nil
}

func logSendToBackend(d *dice.Dice, logInfo *model.LogInfo, force bool) (bool, string, error) {
	ctx := &dice.MsgContext{
		Dice:     d,
		EndPoint: d.UIEndpoint,
	}
	return dice.LogSendToBackendByInfo(ctx, logInfo, force)
}
