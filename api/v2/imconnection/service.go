package imconnection

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"

	dynamicform "sealdice-core/api/v2/imconnection/dynamic_form"
	imconnm "sealdice-core/api/v2/model/imconnection"
	"sealdice-core/dice"
	"sealdice-core/model/common/request"
	"sealdice-core/model/common/response"
)

type Service struct {
	dice *dice.Dice
	dm   *dice.DiceManager
}

func NewService(dm *dice.DiceManager) *Service {
	_ = dynamicform.LoadFromFile("api/v2/imconnection/dynamic_form/forms.json")
	return &Service{
		dice: dm.GetDice(),
		dm:   dm,
	}
}

func (s *Service) RegisterRoutes(grp *huma.Group) {
	huma.Get(grp, "/schemas", s.GetSchemas, func(o *huma.Operation) {
		o.Description = "获取所有平台的配置表单定义"
	})
	huma.Get(grp, "/", s.ListConnections, func(o *huma.Operation) {
		o.Description = "获取当前所有连接列表"
	})
}

func (s *Service) RegisterProtectedRoutes(grp *huma.Group) {
	huma.Post(grp, "/", s.CreateConnection, func(o *huma.Operation) {
		o.Description = "创建连接（仅支持 gocq-separate 最简版）"
	})
	huma.Delete(grp, "/{id}", s.DeleteConnection, func(o *huma.Operation) {
		o.Description = "按 ID 删除连接"
	})
	huma.Put(grp, "/{id}", s.UpdateConnection, func(o *huma.Operation) {
		o.Description = "按 ID 更新连接配置并重连"
	})
}

func (s *Service) GetSchemas(_ context.Context, _ *request.Empty) (*response.ItemResponse[map[string][]*dynamicform.FormConfigItem], error) {
	return response.NewItemResponse[map[string][]*dynamicform.FormConfigItem](dynamicform.GetAll()), nil
}

func (s *Service) ListConnections(_ context.Context, _ *request.Empty) (*response.ItemResponse[imconnm.EndpointListResp], error) {
	return response.NewItemResponse[imconnm.EndpointListResp](imconnm.EndpointListResp{Items: s.dice.ImSession.EndPoints}), nil
}

func (s *Service) CreateConnection(_ context.Context, req *request.RequestWrapper[imconnm.CreateBody]) (*response.ItemResponse[*dice.EndPointInfo], error) {
	switch req.Body.Platform {
	case "gocq-separate":
		cfg := req.Body.Config
		account := asString(cfg, "account")
		connectURL := asString(cfg, "connectUrl")
		accessToken := asString(cfg, "accessToken")
		if account == "" || connectURL == "" {
			return nil, huma.Error400BadRequest("missing params")
		}
		uid := dice.FormatDiceIDQQ(account)
		for _, ep := range s.dice.ImSession.EndPoints {
			if ep.Enable && ep.UserID == uid {
				return nil, huma.Error409Conflict("account already exists")
			}
		}
		conn := dice.NewOnebotConnItem(dice.AddOnebotEcho{
			Token:         accessToken,
			ConnectURL:    connectURL,
			ReverseURL:    "",
			ReverseSuffix: "",
			Mode:          "client",
		})
		conn.UserID = uid
		pa := conn.Adapter.(*dice.PlatformAdapterOnebot)
		pa.Session = s.dice.ImSession
		s.dice.ImSession.EndPoints = append(s.dice.ImSession.EndPoints, conn)
		go dice.ServePureOnebot(s.dice, conn)
		s.dice.LastUpdatedTime = time.Now().Unix()
		s.dice.Save(false)
		return response.NewItemResponse[*dice.EndPointInfo](conn), nil
	default:
		return nil, huma.Error501NotImplemented("not implemented")
	}
}

func (s *Service) DeleteConnection(_ context.Context, p *imconnm.IDPath) (*response.ItemResponse[bool], error) {
	idx := -1
	for i, ep := range s.dice.ImSession.EndPoints {
		if ep != nil && ep.ID == p.ID {
			idx = i
			ep.SetEnable(s.dice, false)
			break
		}
	}
	if idx < 0 {
		return nil, huma.Error404NotFound("not found")
	}
	l := len(s.dice.ImSession.EndPoints)
	cp := make([]*dice.EndPointInfo, 0, l-1)
	cp = append(cp, s.dice.ImSession.EndPoints[:idx]...)
	cp = append(cp, s.dice.ImSession.EndPoints[idx+1:]...)
	s.dice.ImSession.EndPoints = cp
	s.dice.LastUpdatedTime = time.Now().Unix()
	s.dice.Save(false)
	return response.NewItemResponse[bool](true), nil
}

func (s *Service) UpdateConnection(_ context.Context, req *imconnm.UpdateReq) (*response.ItemResponse[*dice.EndPointInfo], error) {
	var ep *dice.EndPointInfo
	for _, e := range s.dice.ImSession.EndPoints {
		if e != nil && e.ID == req.ID {
			ep = e
			break
		}
	}
	if ep == nil {
		return nil, huma.Error404NotFound("not found")
	}
	switch ep.ProtocolType {
	case "pureonebot":
		pa, ok := ep.Adapter.(*dice.PlatformAdapterOnebot)
		if !ok {
			return nil, huma.Error500InternalServerError("adapter error")
		}
		connectURL := asString(req.Body.Body, "connectUrl")
		accessToken := asString(req.Body.Body, "accessToken")
		account := asString(req.Body.Body, "account")
		if connectURL != "" {
			pa.ConnectURL = connectURL
		}
		if accessToken != "" {
			pa.Token = accessToken
		}
		if account != "" {
			ep.UserID = dice.FormatDiceIDQQ(account)
		}
		pa.SetEnable(false)
		pa.SetEnable(true)
		s.dice.LastUpdatedTime = time.Now().Unix()
		s.dice.Save(false)
		return response.NewItemResponse[*dice.EndPointInfo](ep), nil
	default:
		return nil, huma.Error501NotImplemented("not implemented")
	}
}

func asString(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch vv := v.(type) {
	case string:
		return vv
	default:
		return ""
	}
}
