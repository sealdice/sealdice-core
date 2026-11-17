package deck

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sort"
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
	huma.Get(grp, "/list", s.GetList, func(o *huma.Operation) {
		o.Description = "获取牌堆列表"
	})
}

func (s *Service) RegisterProtectedRoutes(grp *huma.Group) {
	huma.Post(grp, "/reload", s.Reload, func(o *huma.Operation) {
		o.Description = "重载牌堆"
	})
	huma.Post(grp, "/upload", s.Upload, func(o *huma.Operation) {
		o.Description = "上传牌堆"
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

func (s *Service) GetList(_ context.Context, req *ListQuery) (*response.ItemResponse[DeckListResp], error) {
	items := make([]*DeckItem, 0, len(s.dice.DeckList))
	keyword := strings.ToLower(strings.TrimSpace(req.Keyword))
	for _, item := range s.dice.DeckList {
		if item == nil {
			continue
		}
		view := FromDeckInfo(item)
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

	return response.NewItemResponse(DeckListResp(paginate.Slice(items, req.Page, req.PageSize))), nil
}

func (s *Service) Reload(_ context.Context, _ *request.Empty) (*response.ItemResponse[ReloadResp], error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(ReloadResp{
			Success:  false,
			TestMode: true,
		}), nil
	}
	dice.DeckReload(s.dice)
	return response.NewItemResponse(ReloadResp{
		Success: true,
	}), nil
}

func (s *Service) Upload(_ context.Context, req *UploadReq) (*response.ItemResponse[UploadResp], error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(UploadResp{
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
	return response.NewItemResponse(UploadResp{Success: true}), nil
}

func (s *Service) Delete(_ context.Context, req *FilenameReq) (*response.ItemResponse[response.SimpleOK], error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(response.SimpleOK{Success: false}), nil
	}
	filename := strings.TrimSpace(req.Body.Filename)
	if filename == "" {
		return nil, huma.Error400BadRequest("filename不能为空")
	}

	for _, deck := range s.dice.DeckList {
		if deck != nil && deck.Filename == filename {
			dice.DeckDelete(s.dice, deck)
			dice.DeckReload(s.dice)
			s.dice.MarkModified()
			return response.NewItemResponse(response.SimpleOK{Success: true}), nil
		}
	}

	return response.NewItemResponse(response.SimpleOK{Success: false}), nil
}

func (s *Service) CheckUpdate(_ context.Context, req *FilenameReq) (*response.ItemResponse[UpdateCheckResult], error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(UpdateCheckResult{
			Success: false,
			Err:     "展示模式不支持该操作",
		}), nil
	}
	filename := strings.TrimSpace(req.Body.Filename)
	if filename == "" {
		return nil, huma.Error400BadRequest("filename不能为空")
	}

	updateErr := "未找到牌堆"
	for _, deck := range s.dice.DeckList {
		if deck == nil || deck.Filename != filename {
			continue
		}
		oldDeck, newDeck, tempFileName, err := s.dice.DeckCheckUpdate(deck)
		if err != nil {
			updateErr = err.Error()
			break
		}
		return response.NewItemResponse(UpdateCheckResult{
			Success:      true,
			Old:          oldDeck,
			New:          newDeck,
			Format:       deck.FileFormat,
			Filename:     deck.Filename,
			TempFileName: tempFileName,
		}), nil
	}

	return response.NewItemResponse(UpdateCheckResult{
		Success: false,
		Err:     updateErr,
	}), nil
}

func (s *Service) Update(_ context.Context, req *UpdateReq) (*response.ItemResponse[response.SimpleOK], error) {
	if s.dm.JustForTest {
		return response.NewItemResponse(response.SimpleOK{Success: false}), nil
	}
	filename := strings.TrimSpace(req.Body.Filename)
	tempFileName := strings.TrimSpace(req.Body.TempFileName)
	if filename == "" || tempFileName == "" {
		return nil, huma.Error400BadRequest("filename和tempFileName不能为空")
	}

	for _, deck := range s.dice.DeckList {
		if deck == nil || deck.Filename != filename {
			continue
		}
		if err := s.dice.DeckUpdate(deck, tempFileName); err == nil {
			dice.DeckReload(s.dice)
			s.dice.MarkModified()
			return response.NewItemResponse(response.SimpleOK{Success: true}), nil
		}
		break
	}

	return response.NewItemResponse(response.SimpleOK{Success: false}), nil
}

func matchDeckKeyword(item *DeckItem, keyword string) bool {
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

func extractUploadForm(raw huma.MultipartFormFiles[UploadForm]) (*UploadForm, error) {
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
	return &UploadForm{
		File: huma.FormFile{
			File:        file,
			ContentType: fh.Header.Get("Content-Type"),
			IsSet:       true,
			Size:        fh.Size,
			Filename:    fh.Filename,
		},
	}, nil
}
