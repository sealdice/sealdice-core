package helpdoc

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"sealdice-core/api/v2/internal/uploadcore"
	"sealdice-core/dice"
	"sealdice-core/model/common/request"
	"sealdice-core/model/common/response"
)

type Service struct {
	dice          *dice.Dice
	dm            *dice.DiceManager
	uploadManager *uploadcore.Manager
}

func NewService(dm *dice.DiceManager) *Service {
	dataDir := dm.GetDice().BaseConfig.DataDir
	if dataDir == "" {
		dataDir = "."
	}
	return &Service{
		dice:          dm.GetDice(),
		dm:            dm,
		uploadManager: uploadcore.NewManager(filepath.Join(dataDir, "helpdoc", ".uploads")),
	}
}

func (s *Service) RegisterRoutes(grp *huma.Group) {
	huma.Get(grp, "/status", s.GetStatus, func(o *huma.Operation) {
		o.Description = "获取帮助文档加载状态"
	})
	huma.Get(grp, "/tree", s.GetTree, func(o *huma.Operation) {
		o.Description = "获取帮助文档文件树"
	})
	huma.Get(grp, "/items/page", s.GetItemsPage, func(o *huma.Operation) {
		o.Description = "分页查询帮助词条"
	})
	huma.Get(grp, "/config", s.GetConfig, func(o *huma.Operation) {
		o.Description = "获取帮助文档配置"
	})
	huma.Get(grp, "/upload/{sessionId}/{index}", s.GetUploadChunkStatus, func(o *huma.Operation) {
		o.Description = "获取帮助文档上传分块状态"
	})
}

func (s *Service) RegisterProtectedRoutes(grp *huma.Group) {
	huma.Post(grp, "/reload", s.Reload, func(o *huma.Operation) {
		o.Description = "重载帮助文档"
	})
	huma.Post(grp, "/delete", s.Delete, func(o *huma.Operation) {
		o.Description = "删除帮助文档文件"
	})
	huma.Post(grp, "/config", s.SetConfig, func(o *huma.Operation) {
		o.Description = "保存帮助文档配置"
	})
	huma.Post(grp, "/upload/init", s.InitUpload, func(o *huma.Operation) {
		o.Description = "初始化帮助文档上传会话"
	})
	huma.Put(grp, "/upload/{sessionId}/{index}", s.UploadChunk, func(o *huma.Operation) {
		o.Description = "上传帮助文档分块"
	})
	huma.Post(grp, "/upload/complete", s.CompleteUpload, func(o *huma.Operation) {
		o.Description = "完成帮助文档上传"
	})
}

func (s *Service) GetStatus(_ context.Context, _ *request.Empty) (*response.ItemResponse[StatusResp], error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(StatusResp{Loading: false, TestMode: true}), nil
	}
	return response.NewItemResponse(StatusResp{Loading: s.dm.IsHelpReloading}), nil
}

func (s *Service) GetTree(_ context.Context, _ *request.Empty) (*response.ItemResponse[TreeResp], error) {
	if s.dm.JustForTest {
		return nil, huma.Error400BadRequest("展示模式不支持该操作")
	}
	if s.dm.IsHelpReloading {
		return nil, huma.Error409Conflict("帮助文件正在加载")
	}
	if s.dm.Help == nil {
		return response.NewItemResponse(TreeResp{Data: []*dice.HelpDoc{}}), nil
	}
	return response.NewItemResponse(TreeResp{Data: s.dm.Help.HelpDocTree}), nil
}

func (s *Service) Reload(_ context.Context, _ *request.Empty) (*response.ItemResponse[SimpleResp], error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(SimpleResp{Success: false, TestMode: true}), nil
	}
	if s.dm.IsHelpReloading {
		return nil, huma.Error409Conflict("帮助文档正在重新装载")
	}
	s.dm.IsHelpReloading = true
	if s.dm.Help != nil {
		s.dm.Help.Close()
	}
	s.dm.InitHelp()
	return response.NewItemResponse(SimpleResp{Success: true}), nil
}

func (s *Service) Delete(_ context.Context, req *DeleteReq) (*response.ItemResponse[SimpleResp], error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(SimpleResp{Success: false, TestMode: true}), nil
	}
	if s.dm.Help == nil {
		return nil, huma.Error400BadRequest("帮助文档未初始化")
	}
	if err := s.dm.Help.DeleteHelpDoc(req.Body.Keys); err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}
	return response.NewItemResponse(SimpleResp{Success: true}), nil
}

func (s *Service) GetItemsPage(_ context.Context, req *HelpTextItemQuery) (*response.ItemResponse[HelpTextItemPageResp], error) {
	if s.dm.Help == nil {
		return response.NewItemResponse(HelpTextItemPageResp{Data: dice.HelpTextVos{}}), nil
	}
	pageNum := req.PageNum
	if pageNum <= 0 {
		pageNum = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	total, data := s.dm.Help.GetHelpItemPage(pageNum, pageSize, strings.TrimSpace(req.ID), strings.TrimSpace(req.Group), strings.TrimSpace(req.From), strings.TrimSpace(req.Title))
	return response.NewItemResponse(HelpTextItemPageResp{
		Total:    total,
		Data:     data,
		PageNum:  pageNum,
		PageSize: pageSize,
	}), nil
}

func (s *Service) GetConfig(_ context.Context, _ *request.Empty) (*response.ItemResponse[ConfigResp], error) {
	if s.dm.IsHelpReloading {
		return nil, huma.Error409Conflict("帮助文档正在重新装载")
	}
	aliases := map[string][]string{}
	if s.dm.Help != nil && s.dm.Help.Config != nil && s.dm.Help.Config.Aliases != nil {
		aliases = s.dm.Help.Config.Aliases
	}
	return response.NewItemResponse(ConfigResp{Aliases: aliases}), nil
}

func (s *Service) SetConfig(_ context.Context, req *ConfigReq) (*response.ItemResponse[SimpleResp], error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(SimpleResp{Success: false, TestMode: true}), nil
	}
	if s.dm.IsHelpReloading {
		return nil, huma.Error409Conflict("帮助文档正在重新装载")
	}
	if s.dm.Help == nil {
		return nil, huma.Error400BadRequest("帮助文档未初始化")
	}
	aliases := req.Body.Aliases
	if aliases == nil {
		aliases = map[string][]string{}
	}
	if err := s.dm.Help.SaveHelpConfig(&dice.HelpConfig{Aliases: aliases}); err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}
	return response.NewItemResponse(SimpleResp{Success: true}), nil
}

func (s *Service) InitUpload(_ context.Context, req *UploadInitReq) (*response.ItemResponse[HelpDocUploadSessionResp], error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(HelpDocUploadSessionResp{Success: false, ResumeSupported: false}), nil
	}
	body := req.Body
	group := sanitizeHelpGroup(body.Group)
	if group == "" {
		return nil, huma.Error400BadRequest("group不能为空")
	}
	if group == dice.HelpBuiltinGroup {
		return nil, huma.Error400BadRequest("不能为内置分组")
	}
	filename := sanitizeHelpFilename(body.Filename)
	if filename == "" {
		return nil, huma.Error400BadRequest("文件名不能为空")
	}
	if !isAllowedHelpDocFile(filename) {
		return nil, huma.Error400BadRequest("仅支持json和xlsx帮助文档")
	}
	if body.FileSize <= 0 {
		return nil, huma.Error400BadRequest("fileSize必须大于0")
	}
	if strings.TrimSpace(body.FileHash) == "" {
		return nil, huma.Error400BadRequest("fileHash不能为空")
	}
	chunkSize := body.ChunkSize
	if chunkSize <= 0 {
		chunkSize = uploadcore.DefaultChunkSize
	}
	session, err := s.uploadManager.InitWithScope(group, filename, body.FileSize, body.FileHash, chunkSize)
	if err != nil {
		return nil, huma.Error500InternalServerError("创建上传目录失败")
	}
	return response.NewItemResponse(HelpDocUploadSessionResp{
		Success:         true,
		SessionID:       session.SessionID,
		ChunkSize:       session.ChunkSize,
		UploadedChunks:  s.uploadManager.SortedUploadedChunks(session),
		UploadedBytes:   s.uploadManager.UploadedBytes(session),
		ExpectedChunks:  session.ExpectedChunks,
		ResumeSupported: true,
	}), nil
}

func (s *Service) GetUploadChunkStatus(_ context.Context, req *UploadChunkQuery) (*response.ItemResponse[HelpDocUploadChunkResp], error) {
	session, err := s.uploadManager.Get(req.SessionID)
	if err != nil {
		return nil, huma.Error404NotFound("上传会话不存在")
	}
	return response.NewItemResponse(HelpDocUploadChunkResp{
		Success:       session.UploadedChunks[req.Index],
		UploadedBytes: s.uploadManager.UploadedBytes(session),
		UploadedChunk: req.Index,
	}), nil
}

func (s *Service) UploadChunk(_ context.Context, req *UploadChunkReq) (*response.ItemResponse[HelpDocUploadChunkResp], error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(HelpDocUploadChunkResp{Success: false, UploadedChunk: req.Index}), nil
	}
	session, err := s.uploadManager.SaveChunk(req.SessionID, req.Index, req.RawBody)
	if err != nil {
		switch {
		case errors.Is(err, uploadcore.ErrSessionNotFound):
			return nil, huma.Error404NotFound("上传会话不存在")
		case errors.Is(err, uploadcore.ErrChunkOutOfRange):
			return nil, huma.Error400BadRequest("chunk index超出范围")
		case errors.Is(err, uploadcore.ErrChunkEmpty):
			return nil, huma.Error400BadRequest("分块内容不能为空")
		default:
			return nil, huma.Error500InternalServerError("写入分块失败")
		}
	}
	return response.NewItemResponse(HelpDocUploadChunkResp{
		Success:       true,
		UploadedBytes: s.uploadManager.UploadedBytes(session),
		UploadedChunk: req.Index,
	}), nil
}

func (s *Service) CompleteUpload(_ context.Context, req *UploadCompleteReq) (*response.ItemResponse[HelpDocUploadCompleteResp], error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(HelpDocUploadCompleteResp{Success: false, TestMode: true}), nil
	}
	if s.dm.Help == nil {
		return nil, huma.Error400BadRequest("帮助文档未初始化")
	}
	sessionMeta, err := s.uploadManager.Get(req.Body.SessionID)
	if err != nil {
		return nil, huma.Error404NotFound("上传会话不存在")
	}
	tmpPath := filepath.Join(sessionMeta.TempDir, "assembled")
	session, err := s.uploadManager.CompleteWithoutCleanup(req.Body.SessionID, tmpPath)
	if err != nil {
		switch {
		case errors.Is(err, uploadcore.ErrIncomplete):
			return nil, huma.Error400BadRequest("上传分块不完整")
		case errors.Is(err, uploadcore.ErrHashMismatch):
			return nil, huma.Error400BadRequest("文件校验失败")
		default:
			return nil, huma.Error500InternalServerError("完成上传失败")
		}
	}
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil, huma.Error500InternalServerError("读取上传文件失败")
	}
	group := session.Scope
	if group == "" {
		group = "default"
	}
	if err := s.dm.Help.UploadHelpDoc(bytes.NewReader(data), group, session.Filename); err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}
	s.uploadManager.Cleanup(req.Body.SessionID)
	return response.NewItemResponse(HelpDocUploadCompleteResp{
		Success:  true,
		Filename: session.Filename,
		Group:    group,
	}), nil
}

func sanitizeHelpFilename(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	return strings.TrimSpace(name)
}

func sanitizeHelpGroup(group string) string {
	group = strings.ReplaceAll(group, "/", "_")
	group = strings.ReplaceAll(group, "\\", "_")
	return strings.TrimSpace(group)
}

func isAllowedHelpDocFile(filename string) bool {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".json", ".xlsx":
		return true
	default:
		return false
	}
}
