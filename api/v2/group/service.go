package group

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	groupm "sealdice-core/api/v2/model/group"
	"sealdice-core/dice"
	"sealdice-core/model/api/req"
	"sealdice-core/model/common/request"
	"sealdice-core/model/common/response"
	"sealdice-core/utils/paginate"
	"sealdice-core/utils/paginate/slicep"
)

// platformOfGroupID
// 基于 GroupID 的“冒号前缀”解析平台标识：
// 例：QQ-Group:12345 -> QQ-Group；DISCORD-CH-Group:xxx -> DISCORD-CH-Group
// 注意：返回统一为大写，便于与筛选条件进行大小写不敏感匹配
func platformOfGroupID(gid string) string {
	i := strings.IndexByte(gid, ':')
	if i < 0 {
		return strings.ToUpper(gid)
	}
	return strings.ToUpper(gid[:i])
}

// GetSupportedPlatforms
// 从当前 ServiceAtNew 中所有群的 GroupID 汇总得到“支持的平台列表”
// 用于前端筛选项的下拉展示；已去重并进行字典序排序
func (b *GroupService) GetSupportedPlatforms() []string {
	resRaw := b.dice.ImSession.ServiceAtNew.ToList()
	set := make(map[string]struct{}, len(resRaw))
	for _, g := range resRaw {
		if g == nil {
			continue
		}
		p := platformOfGroupID(g.GroupID)
		if p != "" {
			set[p] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// 增删改查 - 不能增，不能改
type GroupService struct {
	dice *dice.Dice
	dm   *dice.DiceManager
}

// 现在没办法，SAN锁死了海豹文明，只能用一些特殊的方案了

// GetGroupPage 分页获取群列表
func (b *GroupService) GetGroupPage(_ context.Context, ol *request.RequestWrapper[req.GroupPageRequest]) (*response.
	ItemResponse[response.HPageResult[*dice.GroupInfo]], error) {
	// 获取ServiceAtNew的列表
	res_raw := b.dice.ImSession.ServiceAtNew.ToList()
	// 读取请求体与筛选参数
	reqBody := ol.Body
	filter := reqBody.Filter
	platformSet := make(map[string]struct{}, len(filter.Platform))
	for _, p := range filter.Platform {
		platformSet[strings.ToUpper(strings.TrimSpace(p))] = struct{}{}
	}
	// 关键字（模糊匹配 GroupID / GroupName，大小写敏感保持一致）
	keyword := strings.TrimSpace(reqBody.Keyword)
	// 未使用天数：转换为秒并与 RecentDiceSendTime 比较
	now := time.Now().Unix()
	unusedSecs := int64(filter.UnusedDays) * 86400
	// 逐项过滤
	res := make([]*dice.GroupInfo, 0, len(res_raw))
	for _, g := range res_raw {
		// 日志开关筛选：仅保留 LogOn=true
		if filter.IsLogging && !g.LogOn {
			continue
		}
		// 未使用天数筛选：最近使用时间与当前时间差不足则剔除
		if unusedSecs > 0 {
			last := g.RecentDiceSendTime
			if last != 0 && now-last < unusedSecs {
				continue
			}
		}
		// 关键字模糊匹配：GroupID 或 GroupName 命中其一即保留
		if keyword != "" {
			if !strings.Contains(g.GroupID, keyword) && !strings.Contains(g.GroupName, keyword) {
				continue
			}
		}
		// 平台筛选：基于 GroupID 冒号前缀匹配
		if len(platformSet) > 0 {
			if _, ok := platformSet[platformOfGroupID(g.GroupID)]; !ok {
				continue
			}
		}
		res = append(res, g)
	}
	// 排序：
	// - 若请求指定 OrderByLastTime，则按 RecentDiceSendTime 降序
	// - 否则默认按 GroupName 升序（大小写不敏感）；同名时以 GroupID 升序稳定排序
	if filter.OrderByLastTime {
		sort.SliceStable(res, func(i, j int) bool {
			return res[i].RecentDiceSendTime > res[j].RecentDiceSendTime
		})
	} else {
		sort.SliceStable(res, func(i, j int) bool {
			li := strings.ToLower(res[i].GroupName)
			lj := strings.ToLower(res[j].GroupName)
			if li == lj {
				return res[i].GroupID < res[j].GroupID
			}
			return li < lj
		})
	}
	// 最后使用page的输出
	var rawResult []*dice.GroupInfo
	var adapt = slicep.Adapter(res)
	myPage := paginate.SimplePaginate(adapt, int64(ol.Body.PageSize), int64(ol.Body.Page))
	total, err := myPage.GetTotal()
	if err != nil {
		return nil, err
	}

	err = myPage.Get(&rawResult)
	res2 := response.HPageResult[*dice.GroupInfo]{
		List:     rawResult,
		Total:    total,
		Page:     int(myPage.GetCurrentPage()),
		PageSize: int(myPage.GetListRows()),
	}
	if err != nil {
		return nil, huma.Error500InternalServerError("获取群列表失败")
	}
	result := response.NewItemResponse[response.HPageResult[*dice.GroupInfo]](res2)
	return result, nil
}

func (b *GroupService) RegisterRoutes(grp *huma.Group) {
	huma.Post(grp, "/list", b.GetGroupPage, func(o *huma.Operation) {
		o.Description = "获取群列表"
	})
}

func (b *GroupService) RegisterProtectedRoutes(grp *huma.Group) {
	huma.Post(grp, "/modify", b.ModifyGroup, func(o *huma.Operation) {
		o.Description = "修改群服务状态（开启/关闭）"
	})
	huma.Post(grp, "/quit", b.QuitGroup, func(o *huma.Operation) {
		o.Description = "退出群（支持附加留言与静默）"
	})
	huma.Post(grp, "/batch/quit", b.BatchQuitGroup, func(o *huma.Operation) {
		o.Description = "批量退出群（支持附加留言与静默）"
	})
	huma.Post(grp, "/batch/notify", b.BatchNotifyGroup, func(o *huma.Operation) {
		o.Description = "批量向群发送通知"
	})
}

func NewGroupService(dm *dice.DiceManager) *GroupService {
	return &GroupService{
		dice: dm.GetDice(),
		dm:   dm,
	}
}

// GroupModifyRequest
// 修改群服务状态的请求体：
// - Active: true 表示开启群服务；false 表示关闭
// - GroupID: 目标群ID（如 QQ-Group:12345）
type GroupModifyRequest struct {
	Active  bool   `json:"active"`
	GroupID string `json:"groupId"`
}

// ModifyGroup
// 按照旧版 /api/group.go 的逻辑移植：
// - 遍历当前 dice[0] 的所有 EndPoint，在目标群逐个执行开启/关闭
func (b *GroupService) ModifyGroup(_ context.Context, ol *request.RequestWrapper[groupm.GroupModifyRequest]) (*response.MessageResponse, error) {
	v := ol.Body
	for _, ep := range b.dice.ImSession.EndPoints {
		ctx := &dice.MsgContext{Dice: b.dice, EndPoint: ep, Session: b.dice.ImSession}
		if v.Active {
			dice.SetBotOnAtGroup(ctx, v.GroupID)
		} else {
			dice.SetBotOffAtGroup(ctx, v.GroupID)
		}
	}
	return &response.MessageResponse{Body: struct {
		Message string `json:"message"`
	}{Message: "ok"}}, nil
}

// QuitGroupRequest
// 退出群的请求体：
// - GroupID: 目标群ID
// - Silence: 静默模式，不在群内发告别消息
// - ExtraText: 附加留言文本（非静默时在群内告别消息附加）
type QuitGroupRequest struct {
	GroupID   string `json:"groupId"`
	Silence   bool   `json:"silence"`
	ExtraText string `json:"extraText"`
}

// QuitGroup
// 参考旧版 /api/group.go 的退群逻辑实现：
// - 定位目标群与指定DiceID的端点
// - 记录后台操作与告别文案（非静默时在群内发送）
// - 更新群内Dice存在映射并落库脏标记
// - 调用平台适配器执行退群
func (b *GroupService) QuitGroup(_ context.Context, ol *request.RequestWrapper[groupm.QuitGroupRequest]) (*response.MessageResponse, error) {
	v := ol.Body
	group, exists := b.dice.ImSession.ServiceAtNew.Load(v.GroupID)
	if !exists {
		return &response.MessageResponse{Body: struct {
			Message string `json:"message"`
		}{Message: "group not found"}}, nil
	}
	// 选择在该群内“存在”的第一个端点作为退群执行者；若未找到，则回退到第一个端点
	var chosen *dice.EndPointInfo
	for _, ep := range b.dice.ImSession.EndPoints {
		if group.DiceIDExistsMap.Exists(ep.UserID) {
			chosen = ep
			break
		}
	}
	if chosen == nil && len(b.dice.ImSession.EndPoints) > 0 {
		chosen = b.dice.ImSession.EndPoints[0]
	}
	if chosen == nil {
		return &response.MessageResponse{Body: struct {
			Message string `json:"message"`
		}{Message: "no endpoint available"}}, nil
	}
	txt := fmt.Sprintf("Master后台操作退群: 于群组<%s>(%s)中告别", group.GroupName, group.GroupID)
	ctx := &dice.MsgContext{Dice: b.dice, EndPoint: chosen, Session: b.dice.ImSession}
	b.dice.Logger.Info(txt)
	ctx.Notice(txt)
	if !v.Silence {
		txtPost := dice.DiceFormatTmpl(ctx, "核心:提示_手动退群前缀")
		if v.ExtraText != "" {
			txtPost += "\n骰主留言: " + v.ExtraText
		}
		dice.ReplyGroup(ctx, &dice.Message{GroupID: v.GroupID}, txtPost)
	}
	// 从群的存在映射中删除所选端点
	group.DiceIDExistsMap.Delete(chosen.UserID)
	time.Sleep(6 * time.Second)
	group.MarkDirty(b.dice)
	chosen.Adapter.QuitGroup(ctx, v.GroupID)
	return &response.MessageResponse{Body: struct {
		Message string `json:"message"`
	}{Message: "ok"}}, nil
}

// BatchQuitGroupRequest
// 批量退出群的请求体：
// - GroupIDs: 目标群列表
// - Silence: 静默模式
// - ExtraText: 附加留言
type BatchQuitGroupRequest struct {
	GroupIDs  []string `json:"groupIds"`
	Silence   bool     `json:"silence"`
	ExtraText string   `json:"extraText"`
}

// BatchQuitGroup
// 基于 QuitGroup 逻辑进行批量处理，返回成功退群数量
func (b *GroupService) BatchQuitGroup(_ context.Context, ol *request.RequestWrapper[groupm.BatchQuitGroupRequest]) (*response.ItemResponse[int], error) {
	v := ol.Body
	updated := 0
	for _, gid := range v.GroupIDs {
		group, exists := b.dice.ImSession.ServiceAtNew.Load(gid)
		if !exists {
			continue
		}
		// 为该群选择第一个“在群内存在”的端点，若无则回退到第一个端点
		var chosen *dice.EndPointInfo
		for _, e := range b.dice.ImSession.EndPoints {
			if group.DiceIDExistsMap.Exists(e.UserID) {
				chosen = e
				break
			}
		}
		if chosen == nil && len(b.dice.ImSession.EndPoints) > 0 {
			chosen = b.dice.ImSession.EndPoints[0]
		}
		if chosen == nil {
			continue
		}
		ctx := &dice.MsgContext{Dice: b.dice, EndPoint: chosen, Session: b.dice.ImSession}
		txt := fmt.Sprintf("Master后台操作退群: 于群组<%s>(%s)中告别", group.GroupName, group.GroupID)
		b.dice.Logger.Info(txt)
		ctx.Notice(txt)
		if !v.Silence {
			txtPost := dice.DiceFormatTmpl(ctx, "核心:提示_手动退群前缀")
			if v.ExtraText != "" {
				txtPost += "\n骰主留言: " + v.ExtraText
			}
			dice.ReplyGroup(ctx, &dice.Message{GroupID: gid}, txtPost)
		}
		group.DiceIDExistsMap.Delete(chosen.UserID)
		time.Sleep(6 * time.Second)
		group.MarkDirty(b.dice)
		chosen.Adapter.QuitGroup(ctx, gid)
		updated++
	}
	return response.NewItemResponse[int](updated), nil
}

// BatchNotifyGroupRequest
// 批量通知群的请求体：
// - GroupIDs: 目标群列表
// - Text: 通知内容
type BatchNotifyGroupRequest struct {
	GroupIDs []string `json:"groupIds"`
	Text     string   `json:"text"`
}

// BatchNotifyGroup
// 为每个群发送文本通知：
// - 为每个群选择在群内存在的第一个端点发送，避免重复通知
// - 若没有任何端点在群内存在，则回退至第一个端点尝试发送
// 返回成功发送次数（以群为单位）
func (b *GroupService) BatchNotifyGroup(_ context.Context, ol *request.RequestWrapper[groupm.BatchNotifyGroupRequest]) (*response.ItemResponse[int], error) {
	v := ol.Body
	sent := 0
	for _, gid := range v.GroupIDs {
		group, exists := b.dice.ImSession.ServiceAtNew.Load(gid)
		if !exists {
			continue
		}
		var chosen *dice.EndPointInfo
		for _, ep := range b.dice.ImSession.EndPoints {
			if group.DiceIDExistsMap.Exists(ep.UserID) {
				chosen = ep
				break
			}
		}
		if chosen == nil && len(b.dice.ImSession.EndPoints) > 0 {
			chosen = b.dice.ImSession.EndPoints[0]
		}
		if chosen == nil {
			continue
		}
		ctx := &dice.MsgContext{Dice: b.dice, EndPoint: chosen, Session: b.dice.ImSession}
		dice.ReplyGroup(ctx, &dice.Message{GroupID: gid}, v.Text)
		sent++
	}
	return response.NewItemResponse[int](sent), nil
}
