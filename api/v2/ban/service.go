package ban

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	banm "sealdice-core/api/v2/model/ban"
	cmm "sealdice-core/api/v2/model/common"
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
}

func (s *BanService) GetBanPage(_ context.Context, ol *request.RequestWrapper[banm.BanPageRequest]) (*response.ItemResponse[response.HPageResult[*dice.BanListInfoItem]], error) {
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

func (s *BanService) Delete(_ context.Context, ol *request.RequestWrapper[banm.DeleteReq]) (*struct{}, error) {
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
	return nil, nil
}

func (s *BanService) BatchDelete(_ context.Context, ol *request.RequestWrapper[cmm.IDListReq]) (*response.ItemResponse[cmm.BatchDeleteResp], error) {
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
	return response.NewItemResponse[cmm.BatchDeleteResp](cmm.BatchDeleteResp{Fails: fails}), nil
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
