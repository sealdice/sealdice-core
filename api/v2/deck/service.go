package deck

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"sealdice-core/api/v2/internal/uploadcore"
	cmm "sealdice-core/api/v2/model/common"
	deckm "sealdice-core/api/v2/model/deck"
	"sealdice-core/dice"
	"sealdice-core/model/common/request"
	"sealdice-core/model/common/response"
	"sealdice-core/utils/paginate"
	"sealdice-core/utils/paginate/slicep"
)

const defaultDeckChunkSize int64 = 4 * 1024 * 1024

type Service struct {
	dice          *dice.Dice
	dm            *dice.DiceManager
	uploadManager *uploadcore.Manager
}

func NewService(dm *dice.DiceManager) *Service {
	return &Service{
		dice:          dm.GetDice(),
		dm:            dm,
		uploadManager: uploadcore.NewManager(filepath.Join("data", "decks", ".uploads")),
	}
}

func (s *Service) RegisterRoutes(grp *huma.Group) {
	huma.Get(grp, "/list", s.GetList, func(o *huma.Operation) {
		o.Description = "获取牌堆列表"
	})
	huma.Get(grp, "/upload/{sessionId}/{index}", s.GetUploadChunkStatus, func(o *huma.Operation) {
		o.Description = "获取牌堆上传分块状态"
	})
}

func (s *Service) RegisterProtectedRoutes(grp *huma.Group) {
	huma.Post(grp, "/reload", s.Reload, func(o *huma.Operation) {
		o.Description = "重载牌堆"
	})
	huma.Post(grp, "/upload", s.Upload, func(o *huma.Operation) {
		o.Description = "上传牌堆"
	})
	huma.Post(grp, "/upload/init", s.InitUpload, func(o *huma.Operation) {
		o.Description = "初始化牌堆上传会话"
	})
	huma.Put(grp, "/upload/{sessionId}/{index}", s.UploadChunk, func(o *huma.Operation) {
		o.Description = "上传牌堆分块"
	})
	huma.Post(grp, "/upload/complete", s.CompleteUpload, func(o *huma.Operation) {
		o.Description = "完成牌堆上传"
	})
	huma.Post(grp, "/delete", s.Delete, func(o *huma.Operation) {
		o.Description = "删除牌堆"
	})
	huma.Post(grp, "/check-update", s.CheckUpdate, func(o *huma.Operation) {
		o.Description = "检查牌堆更新"
	})
	huma.Post(grp, "/update", s.Update, func(o *huma.Operation) {
		o.Description = "更新牌堆"
	})
}

func (s *Service) GetList(_ context.Context, req *deckm.ListQuery) (*response.ItemResponse[deckm.DeckListResp], error) {
	items := make([]*deckm.DeckItem, 0, len(s.dice.DeckList))
	keyword := strings.ToLower(strings.TrimSpace(req.Keyword))
	for _, item := range s.dice.DeckList {
		if item == nil {
			continue
		}
		view := deckm.FromDeckInfo(item)
		if keyword != "" && !matchDeckKeyword(view, keyword) {
			continue
		}
		items = append(items, view)
	}

	sortBy := strings.TrimSpace(req.SortBy)
	sortOrder := strings.TrimSpace(req.SortOrder)
	if sortOrder == "" {
		sortOrder = "asc"
	}
	sort.SliceStable(items, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "updateDate":
			less = items[i].UpdateDate < items[j].UpdateDate
		case "author":
			less = strings.ToLower(items[i].Author) < strings.ToLower(items[j].Author)
		default:
			less = strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
		}
		if sortOrder == "desc" {
			return !less
		}
		return less
	})

	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}

	var paged []*deckm.DeckItem
	pager := paginate.SimplePaginate(slicep.Adapter(items), int64(pageSize), int64(page))
	total, err := pager.GetTotal()
	if err != nil {
		return nil, huma.Error500InternalServerError("获取牌堆列表失败")
	}
	if err := pager.Get(&paged); err != nil {
		return nil, huma.Error500InternalServerError("获取牌堆列表失败")
	}

	return response.NewItemResponse(deckm.DeckListResp{
		List:     paged,
		Total:    total,
		Page:     int(pager.GetCurrentPage()),
		PageSize: int(pager.GetListRows()),
	}), nil
}

func (s *Service) Reload(_ context.Context, _ *request.Empty) (*response.ItemResponse[deckm.ReloadResp], error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(deckm.ReloadResp{
			Success:  false,
			TestMode: true,
		}), nil
	}
	dice.DeckReload(s.dice)
	return response.NewItemResponse(deckm.ReloadResp{
		Success: true,
	}), nil
}

func (s *Service) Upload(_ context.Context, req *deckm.UploadReq) (*response.ItemResponse[deckm.UploadResp], error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(deckm.UploadResp{
			Success:  false,
			TestMode: true,
		}), nil
	}

	data, err := extractUploadForm(req.RawBody)
	if err != nil {
		return nil, err
	}
	defer func() { _ = data.File.Close() }()

	filename := sanitizeUploadFilename(data.File.Filename)
	if filename == "" {
		return nil, huma.Error400BadRequest("文件名不能为空")
	}

	dstPath := filepath.Join("data", "decks", filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		return nil, huma.Error500InternalServerError("创建牌堆文件失败")
	}
	defer func() { _ = dst.Close() }()

	if _, err = io.Copy(dst, data.File); err != nil {
		return nil, huma.Error500InternalServerError("写入牌堆文件失败")
	}

	dice.DeckReload(s.dice)
	return response.NewItemResponse(deckm.UploadResp{Success: true}), nil
}

func (s *Service) InitUpload(_ context.Context, req *deckm.UploadInitReq) (*response.ItemResponse[deckm.UploadSessionResp], error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(deckm.UploadSessionResp{
			Success:         false,
			ResumeSupported: false,
		}), nil
	}

	body := req.Body.Body
	filename := sanitizeUploadFilename(body.Filename)
	if filename == "" {
		return nil, huma.Error400BadRequest("文件名不能为空")
	}
	if body.FileSize <= 0 {
		return nil, huma.Error400BadRequest("fileSize必须大于0")
	}
	if strings.TrimSpace(body.FileHash) == "" {
		return nil, huma.Error400BadRequest("fileHash不能为空")
	}

	chunkSize := body.ChunkSize
	if chunkSize <= 0 {
		chunkSize = defaultDeckChunkSize
	}
	session, err := s.uploadManager.Init(filename, body.FileSize, body.FileHash, chunkSize)
	if err != nil {
		return nil, huma.Error500InternalServerError("创建上传目录失败")
	}

	return response.NewItemResponse(deckm.UploadSessionResp{
		Success:         true,
		SessionID:       session.SessionID,
		ChunkSize:       session.ChunkSize,
		UploadedChunks:  s.uploadManager.SortedUploadedChunks(session),
		UploadedBytes:   s.uploadManager.UploadedBytes(session),
		ExpectedChunks:  session.ExpectedChunks,
		ResumeSupported: true,
	}), nil
}

func (s *Service) GetUploadChunkStatus(_ context.Context, req *deckm.UploadChunkQuery) (*response.ItemResponse[deckm.UploadChunkResp], error) {
	session, err := s.uploadManager.Get(req.SessionID)
	if err != nil {
		return nil, huma.Error404NotFound("上传会话不存在")
	}

	uploaded := session.UploadedChunks[req.Index]
	return response.NewItemResponse(deckm.UploadChunkResp{
		Success:       uploaded,
		UploadedBytes: s.uploadManager.UploadedBytes(session),
		UploadedChunk: req.Index,
	}), nil
}

func (s *Service) UploadChunk(_ context.Context, req *deckm.UploadChunkReq) (*response.ItemResponse[deckm.UploadChunkResp], error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(deckm.UploadChunkResp{
			Success:       false,
			UploadedChunk: req.Index,
		}), nil
	}

	session, err := s.uploadManager.SaveChunk(req.SessionID, req.Index, req.RawBody)
	if err != nil {
		switch err {
		case uploadcore.ErrSessionNotFound:
			return nil, huma.Error404NotFound("上传会话不存在")
		case uploadcore.ErrChunkOutOfRange:
			return nil, huma.Error400BadRequest("chunk index超出范围")
		case uploadcore.ErrChunkEmpty:
			return nil, huma.Error400BadRequest("分块内容不能为空")
		default:
			return nil, huma.Error500InternalServerError("写入分块失败")
		}
	}

	return response.NewItemResponse(deckm.UploadChunkResp{
		Success:       true,
		UploadedBytes: s.uploadManager.UploadedBytes(session),
		UploadedChunk: req.Index,
	}), nil
}

func (s *Service) CompleteUpload(_ context.Context, req *deckm.UploadCompleteReq) (*response.ItemResponse[deckm.UploadCompleteResp], error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(deckm.UploadCompleteResp{
			Success:  false,
			TestMode: true,
		}), nil
	}

	sessionMeta, err := s.uploadManager.Get(req.Body.Body.SessionID)
	if err != nil {
		return nil, huma.Error404NotFound("上传会话不存在")
	}
	session, err := s.uploadManager.Complete(req.Body.Body.SessionID, filepath.Join("data", "decks", sessionMeta.Filename))
	if err != nil {
		switch err {
		case uploadcore.ErrIncomplete:
			return nil, huma.Error400BadRequest("上传分块不完整")
		case uploadcore.ErrHashMismatch:
			return nil, huma.Error400BadRequest("文件校验失败")
		default:
			return nil, huma.Error500InternalServerError("完成上传失败")
		}
	}
	dice.DeckReload(s.dice)
	return response.NewItemResponse(deckm.UploadCompleteResp{
		Success:  true,
		Filename: session.Filename,
	}), nil
}

func (s *Service) Delete(_ context.Context, req *deckm.FilenameReq) (*response.ItemResponse[cmm.SimpleOK], error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(cmm.SimpleOK{Success: false}), nil
	}
	filename := strings.TrimSpace(req.Body.Body.Filename)
	if filename == "" {
		return nil, huma.Error400BadRequest("filename不能为空")
	}

	for _, deck := range s.dice.DeckList {
		if deck != nil && deck.Filename == filename {
			dice.DeckDelete(s.dice, deck)
			dice.DeckReload(s.dice)
			s.dice.MarkModified()
			return response.NewItemResponse(cmm.SimpleOK{Success: true}), nil
		}
	}

	return response.NewItemResponse(cmm.SimpleOK{Success: false}), nil
}

func (s *Service) CheckUpdate(_ context.Context, req *deckm.FilenameReq) (*response.ItemResponse[deckm.UpdateCheckResult], error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(deckm.UpdateCheckResult{
			Success: false,
			Err:     "展示模式不支持该操作",
		}), nil
	}
	filename := strings.TrimSpace(req.Body.Body.Filename)
	if filename == "" {
		return nil, huma.Error400BadRequest("filename不能为空")
	}

	for _, deck := range s.dice.DeckList {
		if deck == nil || deck.Filename != filename {
			continue
		}
		oldDeck, newDeck, tempFileName, err := s.dice.DeckCheckUpdate(deck)
		if err != nil {
			return response.NewItemResponse(deckm.UpdateCheckResult{
				Success: false,
				Err:     err.Error(),
			}), nil
		}
		return response.NewItemResponse(deckm.UpdateCheckResult{
			Success:      true,
			Old:          oldDeck,
			New:          newDeck,
			Format:       deck.FileFormat,
			Filename:     deck.Filename,
			TempFileName: tempFileName,
		}), nil
	}

	return response.NewItemResponse(deckm.UpdateCheckResult{
		Success: false,
		Err:     "未找到牌堆",
	}), nil
}

func (s *Service) Update(_ context.Context, req *deckm.UpdateReq) (*response.ItemResponse[cmm.SimpleOK], error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(cmm.SimpleOK{Success: false}), nil
	}
	filename := strings.TrimSpace(req.Body.Body.Filename)
	tempFileName := strings.TrimSpace(req.Body.Body.TempFileName)
	if filename == "" || tempFileName == "" {
		return nil, huma.Error400BadRequest("filename和tempFileName不能为空")
	}

	for _, deck := range s.dice.DeckList {
		if deck == nil || deck.Filename != filename {
			continue
		}
		if err := s.dice.DeckUpdate(deck, tempFileName); err != nil {
			return response.NewItemResponse(cmm.SimpleOK{Success: false}), nil
		}
		dice.DeckReload(s.dice)
		s.dice.MarkModified()
		return response.NewItemResponse(cmm.SimpleOK{Success: true}), nil
	}

	return response.NewItemResponse(cmm.SimpleOK{Success: false}), nil
}

func matchDeckKeyword(item *deckm.DeckItem, keyword string) bool {
	if strings.Contains(strings.ToLower(item.Name), keyword) {
		return true
	}
	if strings.Contains(strings.ToLower(item.Desc), keyword) {
		return true
	}
	if strings.Contains(strings.ToLower(item.Author), keyword) {
		return true
	}
	if strings.Contains(strings.ToLower(item.Filename), keyword) {
		return true
	}
	for command := range item.Command {
		if strings.Contains(strings.ToLower(command), keyword) {
			return true
		}
	}
	return false
}

func sanitizeUploadFilename(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	return strings.TrimSpace(name)
}

func extractUploadForm(raw huma.MultipartFormFiles[deckm.UploadForm]) (*deckm.UploadForm, error) {
	data := raw.Data()
	if data != nil && data.File.IsSet {
		return data, nil
	}
	if raw.Form == nil || len(raw.Form.File["file"]) == 0 {
		return nil, huma.Error400BadRequest("missing file")
	}
	fh := raw.Form.File["file"][0]
	file, err := fh.Open()
	if err != nil {
		return nil, huma.Error400BadRequest("failed to open file")
	}
	return &deckm.UploadForm{
		File: huma.FormFile{
			File:        file,
			ContentType: fh.Header.Get("Content-Type"),
			IsSet:       true,
			Size:        fh.Size,
			Filename:    fh.Filename,
		},
	}, nil
}
