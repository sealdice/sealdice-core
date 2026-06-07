package ban

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"sealdice-core/dice"
	"sealdice-core/model/common/request"
	"sealdice-core/model/common/response"
	"sealdice-core/utils/paginate"
	"sealdice-core/utils/paginate/slicep"
)

type BanService struct {
	dice *dice.Dice
	dm   *dice.DiceManager
}

func NewBanService(dm *dice.DiceManager) *BanService {
	return &BanService{
		dice: dm.GetDice(),
		dm:   dm,
	}
}

func (s *BanService) RegisterRoutes(grp *huma.Group) {
	huma.Post(grp, "/list", s.GetBanPage, func(o *huma.Operation) {
		o.Description = "分页查询黑白名单"
	})
	huma.Get(grp, "/config", s.GetConfig, func(o *huma.Operation) {
		o.Description = "获取拉黑设置"
	})
}

func (s *BanService) RegisterProtectedRoutes(grp *huma.Group) {
	huma.Get(grp, "/export", s.Export, func(o *huma.Operation) {
		o.Description = "导出黑白名单（JSON 流式下载）"
	})
	huma.Post(grp, "/delete", s.Delete, func(o *huma.Operation) {
		o.Description = "删除黑白名单条目"
	})
	huma.Post(grp, "/batch/delete", s.BatchDelete, func(o *huma.Operation) {
		o.Description = "批量删除黑白名单条目"
	})
	huma.Put(grp, "/config", s.SetConfig, func(o *huma.Operation) {
		o.Description = "保存拉黑设置"
	})
	huma.Post(grp, "/add", s.AddEntry, func(o *huma.Operation) {
		o.Description = "添加黑白名单条目"
	})
	huma.Post(grp, "/import", s.Import, func(o *huma.Operation) {
		o.Description = "导入黑白名单 JSON"
	})
}

func (s *BanService) GetBanPage(_ context.Context, ol *BanPageReq) (*response.ItemResponse[response.HPageResult[*dice.BanListInfoItem]], error) {
	resRaw := s.dice.GetBanList()
	reqBody := ol.Body
	filter := reqBody.Filter

	rankSet := make(map[int]struct{}, len(filter.Ranks))
	for _, r := range filter.Ranks {
		rankSet[r] = struct{}{}
	}
	keyword := strings.TrimSpace(reqBody.Keyword)
	if keyword == "" && filter.KeywordOverride != "" {
		keyword = strings.TrimSpace(filter.KeywordOverride)
	}

	res := make([]*dice.BanListInfoItem, 0, len(resRaw))
	for _, item := range resRaw {
		if item == nil {
			continue
		}
		if len(rankSet) > 0 {
			if _, ok := rankSet[int(item.Rank)]; !ok {
				continue
			}
		}
		if keyword != "" {
			if !strings.Contains(item.ID, keyword) && !strings.Contains(item.Name, keyword) {
				continue
			}
		}
		res = append(res, item)
	}

	if filter.OrderByBanTime {
		sort.SliceStable(res, func(i, j int) bool {
			return res[i].BanUpdatedAt > res[j].BanUpdatedAt
		})
	} else if filter.OrderByScore {
		sort.SliceStable(res, func(i, j int) bool {
			return res[i].Score > res[j].Score
		})
	} else {
		sort.SliceStable(res, func(i, j int) bool {
			return res[i].BanUpdatedAt > res[j].BanUpdatedAt
		})
	}

	var rawResult []*dice.BanListInfoItem
	adapt := slicep.Adapter(res)
	myPage := paginate.SimplePaginate(adapt, int64(reqBody.PageSize), int64(reqBody.Page))
	total, err := myPage.GetTotal()
	if err != nil {
		return nil, huma.Error500InternalServerError("分页统计失败")
	}
	err = myPage.Get(&rawResult)
	if err != nil {
		return nil, huma.Error500InternalServerError("分页数据获取失败")
	}
	out := response.HPageResult[*dice.BanListInfoItem]{
		List:     rawResult,
		Total:    total,
		Page:     int(myPage.GetCurrentPage()),
		PageSize: int(myPage.GetListRows()),
	}
	return response.NewItemResponse[response.HPageResult[*dice.BanListInfoItem]](out), nil
}

func (s *BanService) GetConfig(_ context.Context, _ *request.Empty) (*ConfigItemResponse, error) {
	if s.dice == nil || s.dice.Config.BanList == nil {
		return nil, huma.Error500InternalServerError("黑白名单配置未初始化")
	}
	return response.NewItemResponse(buildBanConfig(s.dice.Config.BanList)), nil
}

func (s *BanService) SetConfig(_ context.Context, req *ConfigReq) (*ConfigItemResponse, error) {
	if s.dice == nil || s.dice.Config.BanList == nil {
		return nil, huma.Error500InternalServerError("黑白名单配置未初始化")
	}

	body := req.Body
	if body.ThresholdWarn == 0 {
		body.ThresholdWarn = 100
	}
	if body.ThresholdBan == 0 {
		body.ThresholdBan = 200
	}

	banList := s.dice.Config.BanList
	banList.BanBehaviorRefuseReply = body.BanBehaviorRefuseReply
	banList.BanBehaviorRefuseInvite = body.BanBehaviorRefuseInvite
	banList.BanBehaviorQuitLastPlace = body.BanBehaviorQuitLastPlace
	banList.BanBehaviorQuitPlaceImmediately = body.BanBehaviorQuitPlaceImmediately
	banList.BanBehaviorQuitIfAdmin = body.BanBehaviorQuitIfAdmin
	banList.BanBehaviorQuitIfAdminSilentIfNotAdmin = body.BanBehaviorQuitIfAdminSilentIfNotAdmin
	banList.ThresholdWarn = body.ThresholdWarn
	banList.ThresholdBan = body.ThresholdBan
	banList.AutoBanMinutes = body.AutoBanMinutes
	banList.ScoreReducePerMinute = body.ScoreReducePerMinute
	banList.ScoreGroupMuted = body.ScoreGroupMuted
	banList.ScoreGroupKicked = body.ScoreGroupKicked
	banList.ScoreTooManyCommand = body.ScoreTooManyCommand
	banList.JointScorePercentOfGroup = body.JointScorePercentOfGroup
	banList.JointScorePercentOfInviter = body.JointScorePercentOfInviter

	s.dice.MarkModified()
	s.dm.Save()
	return response.NewItemResponse(buildBanConfig(banList)), nil
}

func (s *BanService) Delete(_ context.Context, ol *DeleteEntryReq) (*struct{}, error) {
	id := strings.TrimSpace(ol.Body.ID)
	if id == "" {
		return nil, huma.Error400BadRequest("缺少 id")
	}
	_, existed := s.dice.Config.BanList.GetByID(id)
	if !existed {
		return nil, huma.Error404NotFound("未找到该条目")
	}
	err := s.dice.Config.BanList.DeleteByIDWeb(s.dice, id)
	if err != nil {
		return nil, huma.Error500InternalServerError("删除失败", err)
	}
	return &struct{}{}, nil
}

func (s *BanService) BatchDelete(_ context.Context, ol *BatchDeleteReq) (*response.ItemResponse[response.BatchDeleteResp], error) {
	fails := make([]string, 0, len(ol.Body.IDs))
	for _, id := range ol.Body.IDs {
		id = strings.TrimSpace(id)
		if id == "" {
			fails = append(fails, id)
			continue
		}
		_, existed := s.dice.Config.BanList.GetByID(id)
		s.dice.Config.BanList.DeleteByID(s.dice, id)
		if !existed {
			fails = append(fails, id)
		}
	}
	return response.NewItemResponse[response.BatchDeleteResp](response.BatchDeleteResp{Fails: fails}), nil
}

func (s *BanService) AddEntry(_ context.Context, req *AddReq) (*SimpleItemResponse, error) {
	if s.dm.JustForTest {
		return nil, huma.Error400BadRequest("展示模式不支持该操作")
	}
	if s.dice == nil || s.dice.Config.BanList == nil {
		return nil, huma.Error500InternalServerError("黑白名单配置未初始化")
	}

	body := req.Body
	id := strings.TrimSpace(body.ID)
	if id == "" {
		return nil, huma.Error400BadRequest("id不能为空")
	}

	reason := strings.TrimSpace(body.Reason)
	if reason == "" {
		reason = "骰主后台设置"
	}
	platform := manualBanPlatformOfID(id)
	ctx := &dice.MsgContext{
		Dice:     s.dice,
		EndPoint: &dice.EndPointInfo{EndPointInfoBase: dice.EndPointInfoBase{Platform: platform}},
	}

	switch dice.BanRankType(body.Rank) {
	case dice.BanRankBanned:
		item := s.dice.Config.BanList.AddScoreBase(id, s.dice.Config.BanList.ThresholdBan, "海豹后台", reason, ctx)
		if item != nil && strings.TrimSpace(body.Name) != "" {
			item.Name = strings.TrimSpace(body.Name)
			item.UpdatedAt = time.Now().Unix()
		}
	case dice.BanRankTrusted:
		s.dice.Config.BanList.SetTrustByID(id, "海豹后台", reason)
		item, ok := s.dice.Config.BanList.GetByID(id)
		if ok && item != nil && strings.TrimSpace(body.Name) != "" {
			item.Name = strings.TrimSpace(body.Name)
			item.UpdatedAt = time.Now().Unix()
		}
	default:
		return nil, huma.Error400BadRequest("仅支持添加禁止或信任条目")
	}

	s.dice.Config.BanList.SaveChanged(s.dice)
	return response.NewItemResponse(response.SimpleOK{Success: true}), nil
}

func (s *BanService) Import(_ context.Context, req *ImportReq) (*SimpleItemResponse, error) {
	if s.dm.JustForTest {
		return nil, huma.Error400BadRequest("展示模式不支持该操作")
	}
	if s.dice == nil || s.dice.Config.BanList == nil {
		return nil, huma.Error500InternalServerError("黑白名单配置未初始化")
	}

	file, err := extractImportForm(req.RawBody)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.File.Close() }()

	data, err := io.ReadAll(file.File)
	if err != nil {
		return nil, huma.Error500InternalServerError("读取导入文件失败")
	}

	var lst []*dice.BanListInfoItem
	if err := json.Unmarshal(data, &lst); err != nil {
		return nil, huma.Error400BadRequest("导入文件格式不正确")
	}

	now := time.Now().Unix()
	for _, item := range lst {
		if item == nil || strings.TrimSpace(item.ID) == "" {
			continue
		}
		item.ID = strings.TrimSpace(item.ID)
		item.UpdatedAt = now
		item.BanUpdatedAt = now

		newReasons := make([]string, 0, len(item.Reasons))
		for _, reason := range item.Reasons {
			reason = strings.TrimSpace(reason)
			if reason == "" {
				continue
			}
			if !strings.HasSuffix(reason, "（来自导入）") {
				reason += "（来自导入）"
			}
			newReasons = append(newReasons, reason)
		}
		item.Reasons = newReasons
		s.dice.Config.BanList.Map.Store(item.ID, item)
	}

	s.dice.Config.BanList.SaveChanged(s.dice)
	return response.NewItemResponse(response.SimpleOK{Success: true}), nil
}

func (s *BanService) Export(_ context.Context, _ *struct{}) (*huma.StreamResponse, error) {
	if s.dm.JustForTest {
		return &huma.StreamResponse{
			Body: func(ctx huma.Context) {
				ctx.SetHeader("Content-Type", "application/json")
				_, _ = ctx.BodyWriter().Write([]byte(`{"testMode": true}`))
			},
		}, nil
	}
	lst := s.dice.GetBanList()
	data, err := json.Marshal(lst)
	if err != nil {
		return nil, huma.Error500InternalServerError("导出失败")
	}
	return &huma.StreamResponse{
		Body: func(ctx huma.Context) {
			ctx.SetHeader("Cache-Control", "no-store")
			ctx.SetHeader("Content-Type", "application/json")
			ctx.SetHeader("Content-Disposition", "attachment; filename=\"黑白名单.json\"")
			_, _ = ctx.BodyWriter().Write(data)
			if fl, ok := ctx.BodyWriter().(http.Flusher); ok {
				fl.Flush()
			}
		},
	}, nil
}

func buildBanConfig(source *dice.BanListInfo) BanConfig {
	return BanConfig{
		BanBehaviorRefuseReply:                 source.BanBehaviorRefuseReply,
		BanBehaviorRefuseInvite:                source.BanBehaviorRefuseInvite,
		BanBehaviorQuitLastPlace:               source.BanBehaviorQuitLastPlace,
		BanBehaviorQuitPlaceImmediately:        source.BanBehaviorQuitPlaceImmediately,
		BanBehaviorQuitIfAdmin:                 source.BanBehaviorQuitIfAdmin,
		BanBehaviorQuitIfAdminSilentIfNotAdmin: source.BanBehaviorQuitIfAdminSilentIfNotAdmin,
		ThresholdWarn:                          source.ThresholdWarn,
		ThresholdBan:                           source.ThresholdBan,
		AutoBanMinutes:                         source.AutoBanMinutes,
		ScoreReducePerMinute:                   source.ScoreReducePerMinute,
		ScoreGroupMuted:                        source.ScoreGroupMuted,
		ScoreGroupKicked:                       source.ScoreGroupKicked,
		ScoreTooManyCommand:                    source.ScoreTooManyCommand,
		JointScorePercentOfGroup:               source.JointScorePercentOfGroup,
		JointScorePercentOfInviter:             source.JointScorePercentOfInviter,
	}
}

func manualBanPlatformOfID(id string) string {
	prefix := strings.Split(strings.TrimSpace(id), ":")[0]
	if prefix == "" {
		return "UI"
	}
	return strings.Replace(prefix, "-Group", "", 1)
}

func extractImportForm(raw huma.MultipartFormFiles[ImportForm]) (*ImportForm, error) {
	if raw.Form == nil || raw.Form.File == nil {
		return nil, huma.Error400BadRequest("missing file")
	}
	files := raw.Form.File["file"]
	if len(files) == 0 {
		return nil, huma.Error400BadRequest("missing file")
	}
	fileHeader := files[0]
	file, err := fileHeader.Open()
	if err != nil {
		return nil, huma.Error400BadRequest("failed to open file")
	}
	return &ImportForm{
		File: huma.FormFile{
			File:        file,
			Filename:    fileHeader.Filename,
			Size:        fileHeader.Size,
			ContentType: fileHeader.Header.Get("Content-Type"),
			IsSet:       true,
		},
	}, nil
}
