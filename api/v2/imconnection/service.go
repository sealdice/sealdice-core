package imconnection

import (
	"context"
	"encoding/base64"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	dynamicform "sealdice-core/api/v2/imconnection/dynamic_form"
	imconnm "sealdice-core/api/v2/model/imconnection"
	"sealdice-core/dice"
	"sealdice-core/model/common/request"
	"sealdice-core/model/common/response"
)

type Service struct {
	dice       *dice.Dice
	dm         *dice.DiceManager
	autoServe  bool
	autoSave   bool
	protocols  []*imconnm.ProtocolDefinition
	protocolBy map[string]*imconnm.ProtocolDefinition
}

func NewService(dm *dice.DiceManager) *Service {
	return newService(dm, true, true)
}

func newService(dm *dice.DiceManager, autoServe bool, autoSave bool) *Service {
	_ = loadForms()
	s := &Service{
		dice:       dm.GetDice(),
		dm:         dm,
		autoServe:  autoServe,
		autoSave:   autoSave,
		protocolBy: map[string]*imconnm.ProtocolDefinition{},
	}
	s.protocols = s.buildProtocolDefinitions()
	for _, p := range s.protocols {
		s.protocolBy[p.Key] = p
	}
	return s
}

func loadForms() error {
	paths := []string{
		"api/v2/imconnection/dynamic_form/forms.json",
		"dynamic_form/forms.json",
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return dynamicform.LoadFromFile(path)
		}
	}
	return dynamicform.LoadFromFile(paths[0])
}

func (s *Service) RegisterRoutes(grp *huma.Group) {
	huma.Get(grp, "/protocols", s.GetProtocols, func(o *huma.Operation) {
		o.Description = "获取账号协议能力和配置表单定义"
	})
	huma.Get(grp, "/schemas", s.GetSchemas, func(o *huma.Operation) {
		o.Description = "获取所有平台的配置表单定义"
	})
	huma.Get(grp, "/sign-info", s.GetSignInfo, func(o *huma.Operation) {
		o.Description = "获取 Lagrange 签名服务信息"
	})
	huma.Get(grp, "/{id}/config", s.GetEditableConfig, func(o *huma.Operation) {
		o.Description = "获取连接可编辑配置"
	})
	huma.Get(grp, "/{id}/workflow", s.GetWorkflow, func(o *huma.Operation) {
		o.Description = "获取连接登录工作流状态"
	})
	huma.Get(grp, "/{id}/qrcode", s.GetQRCode, func(o *huma.Operation) {
		o.Description = "获取连接登录二维码"
	})
	huma.Get(grp, "/", s.ListConnections, func(o *huma.Operation) {
		o.Description = "获取当前所有连接列表"
	})
}

func (s *Service) RegisterProtectedRoutes(grp *huma.Group) {
	huma.Post(grp, "/", s.CreateConnection, func(o *huma.Operation) {
		o.Description = "创建连接"
	})
	huma.Delete(grp, "/{id}", s.DeleteConnection, func(o *huma.Operation) {
		o.Description = "按 ID 删除连接"
	})
	huma.Put(grp, "/{id}", s.UpdateConnection, func(o *huma.Operation) {
		o.Description = "按 ID 更新连接配置并重连"
	})
	huma.Put(grp, "/{id}/enable", s.SetEnable, func(o *huma.Operation) {
		o.Description = "启用或禁用连接"
	})
}

func (s *Service) buildProtocolDefinitions() []*imconnm.ProtocolDefinition {
	baseCapabilities := imconnm.ProtocolCapability{
		Create: true,
		Update: true,
		Delete: true,
		Enable: true,
	}
	items := []*imconnm.ProtocolDefinition{
		{Key: "lagrange", Name: "QQ(内置客户端)", Platform: "QQ", SchemaKey: "lagrange", Available: true, Capabilities: withWorkflow(baseCapabilities, true, true, true)},
		{Key: "milky", Name: "QQ(Milky)", Platform: "QQ", SchemaKey: "milky", Available: true, Capabilities: baseCapabilities},
		{Key: "milky-internal", Name: "QQ(内置Milky)", Platform: "QQ", SchemaKey: "milky-internal", Available: true, Capabilities: withWorkflow(baseCapabilities, true, true, false)},
		{Key: "gocq-separate", Name: "QQ(onebot11正向WS)", Platform: "QQ", SchemaKey: "gocq-separate", Available: true, Capabilities: baseCapabilities},
		{Key: "onebot-reverse", Name: "QQ(onebot11反向WS)", Platform: "QQ", SchemaKey: "onebot-reverse", Available: true, Capabilities: baseCapabilities},
		{Key: "officialqq", Name: "QQ(官方机器人)", Platform: "QQ", SchemaKey: "officialqq", Available: true, Capabilities: baseCapabilities},
		{Key: "satori", Name: "[WIP]Satori", Platform: "Satori", SchemaKey: "satori", Available: true, Capabilities: baseCapabilities},
		{Key: "sealchat", Name: "[WIP]SealChat", Platform: "SealChat", SchemaKey: "sealchat", Available: true, Capabilities: baseCapabilities},
		{Key: "discord", Name: "Discord", Platform: "Discord", SchemaKey: "discord", Available: true, Capabilities: baseCapabilities},
		{Key: "kook", Name: "KOOK(开黑啦)", Platform: "KOOK", SchemaKey: "kook", Available: true, Capabilities: baseCapabilities},
		{Key: "telegram", Name: "Telegram", Platform: "Telegram", SchemaKey: "telegram", Available: true, Capabilities: baseCapabilities},
		{Key: "minecraft", Name: "Minecraft服务器", Platform: "Minecraft", SchemaKey: "minecraft", Available: true, Capabilities: baseCapabilities},
		{Key: "dodo", Name: "Dodo语音", Platform: "Dodo", SchemaKey: "dodo", Available: true, Capabilities: baseCapabilities},
		{Key: "dingtalk", Name: "钉钉", Platform: "DingTalk", SchemaKey: "dingtalk", Available: true, Capabilities: baseCapabilities},
		{Key: "slack", Name: "Slack", Platform: "Slack", SchemaKey: "slack", Available: true, Capabilities: baseCapabilities},
		{Key: "red", Name: "[已弃用]QQ(red协议)", Platform: "QQ", SchemaKey: "red", Deprecated: true, Available: false, DisabledReason: "Red 协议已弃用", Capabilities: imconnm.ProtocolCapability{}},
		{Key: "gocq", Name: "[已弃用]QQ(内置 gocq)", Platform: "QQ", SchemaKey: "gocq", Deprecated: true, Available: false, DisabledReason: "内置 gocq 已弃用，请使用内置客户端或分离部署", Capabilities: imconnm.ProtocolCapability{}},
	}
	if s.dm.ContainerMode {
		for _, item := range items {
			if item.Key == "lagrange" || item.Key == "milky-internal" {
				item.Available = false
				item.DisabledReason = "当前为容器模式，内置客户端被禁用"
			}
		}
	}
	return items
}

func withWorkflow(base imconnm.ProtocolCapability, workflow bool, qrcode bool, signInfo bool) imconnm.ProtocolCapability {
	base.Workflow = workflow
	base.QRCode = qrcode
	base.SignInfo = signInfo
	return base
}

func (s *Service) GetProtocols(_ context.Context, _ *request.Empty) (*response.ItemResponse[imconnm.ProtocolListResp], error) {
	return response.NewItemResponse[imconnm.ProtocolListResp](imconnm.ProtocolListResp{Items: s.protocols}), nil
}

func (s *Service) GetSchemas(_ context.Context, _ *request.Empty) (*response.ItemResponse[map[string][]*dynamicform.FormConfigItem], error) {
	return response.NewItemResponse[map[string][]*dynamicform.FormConfigItem](dynamicform.GetAll()), nil
}

func (s *Service) GetSignInfo(_ context.Context, _ *request.Empty) (*response.ItemResponse[imconnm.SignInfoResp], error) {
	infos, err := dice.LagrangeGetSignInfo(s.dice)
	if err != nil {
		return nil, huma.Error500InternalServerError("read sign info failed")
	}
	out := make([]imconnm.SignInfo, 0, len(infos))
	for _, info := range infos {
		servers := make([]*imconnm.SignServer, 0, len(info.Servers))
		for _, server := range info.Servers {
			servers = append(servers, &imconnm.SignServer{
				Name:     server.Name,
				URL:      server.Url,
				Latency:  int64(server.Latency),
				Selected: server.Selected,
				Ignored:  server.Ignored,
				Note:     server.Note,
			})
		}
		out = append(out, imconnm.SignInfo{
			Version:  info.Version,
			AppInfo:  info.Appinfo,
			Servers:  servers,
			Selected: info.Selected,
			Ignored:  info.Ignored,
			Note:     info.Note,
		})
	}
	return response.NewItemResponse[imconnm.SignInfoResp](imconnm.SignInfoResp{Items: out}), nil
}

func (s *Service) ListConnections(_ context.Context, _ *request.Empty) (*response.ItemResponse[imconnm.EndpointListResp], error) {
	return response.NewItemResponse[imconnm.EndpointListResp](imconnm.EndpointListResp{Items: s.dice.ImSession.EndPoints}), nil
}

func (s *Service) GetEditableConfig(_ context.Context, p *imconnm.IDPath) (*response.ItemResponse[imconnm.EditableConfigResp], error) {
	ep := s.findEndpoint(p.ID)
	if ep == nil {
		return nil, huma.Error404NotFound("not found")
	}
	key, err := protocolKeyOfEndpoint(ep)
	if err != nil {
		return nil, err
	}
	protocol, ok := s.protocolBy[key]
	if !ok || !protocol.Capabilities.Update {
		return nil, huma.Error400BadRequest("protocol update unavailable")
	}
	cfg, err := editableConfigOf(ep, key)
	if err != nil {
		return nil, err
	}
	schema := cloneSchemaForUpdate(dynamicform.GetFormConfig(protocol.SchemaKey))
	return response.NewItemResponse[imconnm.EditableConfigResp](
		imconnm.EditableConfigResp{
			ProtocolKey:     key,
			Schema:          schema,
			Config:          cfg,
			RestartRequired: true,
		},
	), nil
}

func (s *Service) CreateConnection(_ context.Context, req *request.RequestWrapper[imconnm.CreateBody]) (*response.ItemResponse[*dice.EndPointInfo], error) {
	key := req.Body.Platform
	protocol, ok := s.protocolBy[key]
	if !ok {
		return nil, huma.Error404NotFound("protocol not found")
	}
	if protocol.Deprecated || !protocol.Available || !protocol.Capabilities.Create {
		return nil, huma.Error400BadRequest("protocol unavailable")
	}
	params, err := dynamicform.BuildParamsByConfig(dynamicform.GetFormConfig(protocol.SchemaKey), req.Body.Config)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	conn, err := s.newConnection(key, params)
	if err != nil {
		return nil, err
	}
	s.appendAndSave(conn)
	s.serve(key, conn)
	return response.NewItemResponse[*dice.EndPointInfo](conn), nil
}

func (s *Service) newConnection(key string, cfg map[string]interface{}) (*dice.EndPointInfo, error) {
	switch key {
	case "lagrange":
		account := asString(cfg, "account")
		if err := s.checkQQAccount(account); err != nil {
			return nil, err
		}
		conn := dice.NewLagrangeConnectInfoItem(account)
		conn.UserID = dice.FormatDiceIDQQ(account)
		conn.Session = s.dice.ImSession
		pa := conn.Adapter.(*dice.PlatformAdapterGocq)
		pa.Session = s.dice.ImSession
		pa.SignServerName = asString(cfg, "signServerName")
		pa.SignServerVer = asString(cfg, "signServerVersion")
		return conn, nil
	case "gocq-separate":
		account := asString(cfg, "account")
		if err := s.checkQQAccount(account); err != nil {
			return nil, err
		}
		connectURL := normalizeWSURL(asString(cfg, "connectUrl"))
		conn := dice.NewOnebotConnItem(dice.AddOnebotEcho{
			Token:      asString(cfg, "accessToken"),
			ConnectURL: connectURL,
			Mode:       "client",
		})
		conn.UserID = dice.FormatDiceIDQQ(account)
		conn.Session = s.dice.ImSession
		pa := conn.Adapter.(*dice.PlatformAdapterOnebot)
		pa.Session = s.dice.ImSession
		return conn, nil
	case "onebot-reverse":
		account := asString(cfg, "account")
		if err := s.checkQQAccount(account); err != nil {
			return nil, err
		}
		conn := dice.NewOnebotConnItem(dice.AddOnebotEcho{
			ReverseURL:    asString(cfg, "reverseAddr"),
			ReverseSuffix: "/ws",
			Mode:          "server",
		})
		conn.UserID = dice.FormatDiceIDQQ(account)
		conn.Session = s.dice.ImSession
		pa := conn.Adapter.(*dice.PlatformAdapterOnebot)
		pa.Session = s.dice.ImSession
		return conn, nil
	case "milky":
		conn := dice.NewMilkyConnItem(dice.AddMilkyEcho{
			Token:       asString(cfg, "token"),
			WsGateway:   asString(cfg, "wsGateway"),
			RestGateway: asString(cfg, "restGateway"),
		})
		setMilkySession(s.dice, conn)
		return conn, nil
	case "milky-internal":
		account := asString(cfg, "account")
		if err := s.checkQQAccount(account); err != nil {
			return nil, err
		}
		mode := asString(cfg, "builtInMode")
		if mode != "yogurt" && mode != "lagrangeV2" {
			return nil, huma.Error400BadRequest("unsupported builtInMode")
		}
		conn := dice.NewMilkyConnItem(dice.AddMilkyEcho{BuiltInMode: mode})
		conn.UserID = dice.FormatDiceIDQQ(account)
		setMilkySession(s.dice, conn)
		return conn, nil
	case "sealchat":
		conn := dice.NewSealChatConnItem(asString(cfg, "url"), asString(cfg, "token"))
		conn.Session = s.dice.ImSession
		pa := conn.Adapter.(*dice.PlatformAdapterSealChat)
		pa.Session = s.dice.ImSession
		return conn, nil
	case "satori":
		conn := dice.NewSatoriConnItem(asString(cfg, "platform"), asString(cfg, "host"), asInt(cfg, "port"), asString(cfg, "token"))
		conn.Session = s.dice.ImSession
		pa := conn.Adapter.(*dice.PlatformAdapterSatori)
		pa.Session = s.dice.ImSession
		return conn, nil
	case "discord":
		conn := dice.NewDiscordConnItem(dice.AddDiscordEcho{
			Token:              asString(cfg, "token"),
			ProxyURL:           asString(cfg, "proxyURL"),
			ReverseProxyUrl:    asString(cfg, "reverseProxyUrl"),
			ReverseProxyCDNUrl: asString(cfg, "reverseProxyCDNUrl"),
		})
		conn.Session = s.dice.ImSession
		pa := conn.Adapter.(*dice.PlatformAdapterDiscord)
		pa.Session = s.dice.ImSession
		return conn, nil
	case "kook":
		conn := dice.NewKookConnItem(asString(cfg, "token"))
		conn.Session = s.dice.ImSession
		pa := conn.Adapter.(*dice.PlatformAdapterKook)
		pa.Session = s.dice.ImSession
		return conn, nil
	case "telegram":
		conn := dice.NewTelegramConnItem(asString(cfg, "token"), asString(cfg, "proxyURL"))
		conn.Session = s.dice.ImSession
		pa := conn.Adapter.(*dice.PlatformAdapterTelegram)
		pa.Session = s.dice.ImSession
		return conn, nil
	case "minecraft":
		conn := dice.NewMinecraftConnItem(asString(cfg, "url"))
		conn.Session = s.dice.ImSession
		pa := conn.Adapter.(*dice.PlatformAdapterMinecraft)
		pa.Session = s.dice.ImSession
		return conn, nil
	case "dodo":
		conn := dice.NewDodoConnItem(asString(cfg, "clientID"), asString(cfg, "token"))
		conn.Session = s.dice.ImSession
		pa := conn.Adapter.(*dice.PlatformAdapterDodo)
		pa.Session = s.dice.ImSession
		return conn, nil
	case "dingtalk":
		conn := dice.NewDingTalkConnItem(asString(cfg, "clientID"), asString(cfg, "token"), asString(cfg, "nickname"), asString(cfg, "robotCode"))
		conn.Session = s.dice.ImSession
		pa := conn.Adapter.(*dice.PlatformAdapterDingTalk)
		pa.Session = s.dice.ImSession
		return conn, nil
	case "slack":
		conn := dice.NewSlackConnItem(asString(cfg, "appToken"), asString(cfg, "botToken"))
		conn.Session = s.dice.ImSession
		pa := conn.Adapter.(*dice.PlatformAdapterSlack)
		pa.Session = s.dice.ImSession
		return conn, nil
	case "officialqq":
		conn := dice.NewOfficialQQConnItem(uint64(asInt(cfg, "appID")), asString(cfg, "token"), asString(cfg, "appSecret"), asBool(cfg, "onlyQQGuild"))
		conn.Session = s.dice.ImSession
		pa := conn.Adapter.(*dice.PlatformAdapterOfficialQQ)
		pa.Session = s.dice.ImSession
		return conn, nil
	default:
		return nil, huma.Error501NotImplemented("not implemented")
	}
}

func setMilkySession(d *dice.Dice, conn *dice.EndPointInfo) {
	conn.Session = d.ImSession
	pa := conn.Adapter.(*dice.PlatformAdapterMilky)
	pa.Session = d.ImSession
}

func (s *Service) appendAndSave(conn *dice.EndPointInfo) {
	s.dice.ImSession.EndPoints = append(s.dice.ImSession.EndPoints, conn)
	s.dice.LastUpdatedTime = time.Now().Unix()
	s.save()
}

func (s *Service) serve(key string, conn *dice.EndPointInfo) {
	if !s.autoServe {
		return
	}
	switch key {
	case "lagrange":
		pa := conn.Adapter.(*dice.PlatformAdapterGocq)
		uin, _ := strconv.ParseInt(strings.TrimPrefix(conn.UserID, "QQ:"), 10, 64)
		go dice.LagrangeServe(s.dice, conn, dice.LagrangeLoginInfo{
			UIN:               uin,
			SignServerName:    pa.SignServerName,
			SignServerVersion: pa.SignServerVer,
			IsAsyncRun:        true,
		})
	case "gocq-separate", "onebot-reverse":
		go dice.ServePureOnebot(s.dice, conn)
	case "milky":
		go dice.ServeMilky(s.dice, conn)
	case "milky-internal":
		go dice.ServeMilkyBuiltIn(s.dice, conn)
	case "sealchat":
		go dice.ServeSealChat(s.dice, conn)
	case "satori":
		go dice.ServeSatori(s.dice, conn)
	case "discord":
		go dice.ServeDiscord(s.dice, conn)
	case "kook":
		go dice.ServeKook(s.dice, conn)
	case "telegram":
		go dice.ServeTelegram(s.dice, conn)
	case "minecraft":
		go dice.ServeMinecraft(s.dice, conn)
	case "dodo":
		go dice.ServeDodo(s.dice, conn)
	case "dingtalk":
		go dice.ServeDingTalk(s.dice, conn)
	case "slack":
		go dice.ServeSlack(s.dice, conn)
	case "officialqq":
		go dice.ServerOfficialQQ(s.dice, conn)
	}
}

func (s *Service) checkQQAccount(account string) error {
	if account == "" {
		return huma.Error400BadRequest("missing account")
	}
	uid := dice.FormatDiceIDQQ(account)
	for _, ep := range s.dice.ImSession.EndPoints {
		if ep != nil && ep.Enable && ep.UserID == uid {
			return huma.Error409Conflict("account already exists")
		}
	}
	return nil
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
	s.save()
	return response.NewItemResponse[bool](true), nil
}

func (s *Service) UpdateConnection(_ context.Context, req *imconnm.UpdateReq) (*response.ItemResponse[*dice.EndPointInfo], error) {
	ep := s.findEndpoint(req.ID)
	if ep == nil {
		return nil, huma.Error404NotFound("not found")
	}
	key, err := protocolKeyOfEndpoint(ep)
	if err != nil {
		return nil, err
	}
	protocol, ok := s.protocolBy[key]
	if !ok || !protocol.Capabilities.Update {
		return nil, huma.Error400BadRequest("protocol update unavailable")
	}
	params, err := buildUpdateParams(cloneSchemaForUpdate(dynamicform.GetFormConfig(protocol.SchemaKey)), req.Body.Body)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	if err := s.applyUpdate(ep, key, params); err != nil {
		return nil, err
	}
	s.dice.LastUpdatedTime = time.Now().Unix()
	s.save()
	s.reconnectIfEnabled(ep)
	return response.NewItemResponse[*dice.EndPointInfo](ep), nil
}

func (s *Service) SetEnable(_ context.Context, req *imconnm.EnableReq) (*response.ItemResponse[*dice.EndPointInfo], error) {
	ep := s.findEndpoint(req.ID)
	if ep == nil {
		return nil, huma.Error404NotFound("not found")
	}
	if s.autoServe {
		ep.SetEnable(s.dice, req.Body.Body.Enable)
	} else {
		ep.Enable = req.Body.Body.Enable
	}
	s.dice.LastUpdatedTime = time.Now().Unix()
	s.save()
	return response.NewItemResponse[*dice.EndPointInfo](ep), nil
}

func protocolKeyOfEndpoint(ep *dice.EndPointInfo) (string, error) {
	if ep == nil {
		return "", huma.Error404NotFound("not found")
	}
	switch ep.ProtocolType {
	case "onebot":
		if pa, ok := ep.Adapter.(*dice.PlatformAdapterGocq); ok && pa.UseInPackClient && pa.BuiltinMode == "lagrange" {
			return "lagrange", nil
		}
	case "milky":
		if pa, ok := ep.Adapter.(*dice.PlatformAdapterMilky); ok {
			if pa.BuiltInMode != "" {
				return "milky-internal", nil
			}
			return "milky", nil
		}
	case "pureonebot":
		if pa, ok := ep.Adapter.(*dice.PlatformAdapterOnebot); ok {
			if pa.Mode == "server" {
				return "onebot-reverse", nil
			}
			return "gocq-separate", nil
		}
	case "official":
		return "officialqq", nil
	case "satori":
		return "satori", nil
	}
	switch ep.Platform {
	case "SEALCHAT":
		return "sealchat", nil
	case "DISCORD":
		return "discord", nil
	case "KOOK":
		return "kook", nil
	case "TG":
		return "telegram", nil
	case "MC":
		return "minecraft", nil
	case "DODO":
		return "dodo", nil
	case "DINGTALK":
		return "dingtalk", nil
	case "SLACK":
		return "slack", nil
	default:
		return "", huma.Error501NotImplemented("protocol not supported")
	}
}

func editableConfigOf(ep *dice.EndPointInfo, key string) (map[string]interface{}, error) {
	switch key {
	case "lagrange":
		pa, ok := ep.Adapter.(*dice.PlatformAdapterGocq)
		if !ok {
			return nil, huma.Error500InternalServerError("adapter error")
		}
		return map[string]interface{}{
			"account":           strings.TrimPrefix(ep.UserID, "QQ:"),
			"signServerVersion": pa.SignServerVer,
			"signServerName":    pa.SignServerName,
		}, nil
	case "gocq-separate":
		pa, ok := ep.Adapter.(*dice.PlatformAdapterOnebot)
		if !ok {
			return nil, huma.Error500InternalServerError("adapter error")
		}
		return map[string]interface{}{
			"account":     strings.TrimPrefix(ep.UserID, "QQ:"),
			"connectUrl":  pa.ConnectURL,
			"accessToken": "",
		}, nil
	case "onebot-reverse":
		pa, ok := ep.Adapter.(*dice.PlatformAdapterOnebot)
		if !ok {
			return nil, huma.Error500InternalServerError("adapter error")
		}
		return map[string]interface{}{
			"account":     strings.TrimPrefix(ep.UserID, "QQ:"),
			"reverseAddr": pa.ReverseUrl,
		}, nil
	case "milky":
		pa, ok := ep.Adapter.(*dice.PlatformAdapterMilky)
		if !ok {
			return nil, huma.Error500InternalServerError("adapter error")
		}
		return map[string]interface{}{
			"wsGateway":   pa.WsGateway,
			"restGateway": pa.RestGateway,
			"token":       "",
		}, nil
	case "milky-internal":
		pa, ok := ep.Adapter.(*dice.PlatformAdapterMilky)
		if !ok {
			return nil, huma.Error500InternalServerError("adapter error")
		}
		return map[string]interface{}{
			"account":     strings.TrimPrefix(ep.UserID, "QQ:"),
			"builtInMode": pa.BuiltInMode,
		}, nil
	case "sealchat":
		pa, ok := ep.Adapter.(*dice.PlatformAdapterSealChat)
		if !ok {
			return nil, huma.Error500InternalServerError("adapter error")
		}
		return map[string]interface{}{"url": pa.ConnectURL, "token": ""}, nil
	case "satori":
		pa, ok := ep.Adapter.(*dice.PlatformAdapterSatori)
		if !ok {
			return nil, huma.Error500InternalServerError("adapter error")
		}
		return map[string]interface{}{"platform": pa.Platform, "host": pa.Host, "port": pa.Port, "token": ""}, nil
	case "discord":
		pa, ok := ep.Adapter.(*dice.PlatformAdapterDiscord)
		if !ok {
			return nil, huma.Error500InternalServerError("adapter error")
		}
		return map[string]interface{}{
			"token":              "",
			"proxyURL":           pa.ProxyURL,
			"reverseProxyUrl":    pa.ReverseProxyUrl,
			"reverseProxyCDNUrl": pa.ReverseProxyCDNUrl,
		}, nil
	case "kook":
		return map[string]interface{}{"token": ""}, nil
	case "telegram":
		pa, ok := ep.Adapter.(*dice.PlatformAdapterTelegram)
		if !ok {
			return nil, huma.Error500InternalServerError("adapter error")
		}
		return map[string]interface{}{"token": "", "proxyURL": pa.ProxyURL}, nil
	case "minecraft":
		pa, ok := ep.Adapter.(*dice.PlatformAdapterMinecraft)
		if !ok {
			return nil, huma.Error500InternalServerError("adapter error")
		}
		return map[string]interface{}{"url": pa.ConnectURL}, nil
	case "dodo":
		pa, ok := ep.Adapter.(*dice.PlatformAdapterDodo)
		if !ok {
			return nil, huma.Error500InternalServerError("adapter error")
		}
		return map[string]interface{}{"clientID": pa.ClientID, "token": ""}, nil
	case "dingtalk":
		pa, ok := ep.Adapter.(*dice.PlatformAdapterDingTalk)
		if !ok {
			return nil, huma.Error500InternalServerError("adapter error")
		}
		return map[string]interface{}{
			"clientID":  pa.ClientID,
			"token":     "",
			"nickname":  ep.Nickname,
			"robotCode": pa.RobotCode,
		}, nil
	case "slack":
		return map[string]interface{}{"botToken": "", "appToken": ""}, nil
	case "officialqq":
		pa, ok := ep.Adapter.(*dice.PlatformAdapterOfficialQQ)
		if !ok {
			return nil, huma.Error500InternalServerError("adapter error")
		}
		return map[string]interface{}{
			"appID":       pa.AppID,
			"appSecret":   "",
			"token":       "",
			"onlyQQGuild": pa.OnlyQQGuild,
		}, nil
	default:
		return nil, huma.Error501NotImplemented("not implemented")
	}
}

func cloneSchemaForUpdate(items []*dynamicform.FormConfigItem) []*dynamicform.FormConfigItem {
	out := make([]*dynamicform.FormConfigItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		cp := *item
		if cp.Sensitive {
			cp.IsRequired = dynamicform.RequiredFalse
			if cp.Placeholder == "" {
				cp.Placeholder = "留空则保持原值"
			} else if !strings.Contains(cp.Placeholder, "留空") {
				cp.Placeholder += "；留空则保持原值"
			}
		}
		if cp.FieldName == "account" {
			cp.IsRequired = dynamicform.RequiredFalse
			cp.Readonly = true
		}
		out = append(out, &cp)
	}
	return out
}

func buildUpdateParams(schema []*dynamicform.FormConfigItem, body map[string]interface{}) (map[string]interface{}, error) {
	params, err := dynamicform.BuildParamsByConfig(schema, body)
	if err != nil {
		return nil, err
	}
	for _, item := range schema {
		if item == nil || item.Sensitive || item.IsRequired == dynamicform.RequiredTrue {
			continue
		}
		key := item.FieldName
		if key == "" {
			key = strconv.FormatUint(item.ID, 10)
		}
		value, ok := body[key]
		if !ok || value != "" {
			continue
		}
		params[key] = ""
	}
	return params, nil
}

func (s *Service) applyUpdate(ep *dice.EndPointInfo, key string, cfg map[string]interface{}) error {
	switch key {
	case "lagrange":
		pa, ok := ep.Adapter.(*dice.PlatformAdapterGocq)
		if !ok {
			return huma.Error500InternalServerError("adapter error")
		}
		pa.SignServerVer = asStringOrKeep(cfg, "signServerVersion", pa.SignServerVer)
		pa.SignServerName = asStringOrKeep(cfg, "signServerName", pa.SignServerName)
	case "gocq-separate":
		pa, ok := ep.Adapter.(*dice.PlatformAdapterOnebot)
		if !ok {
			return huma.Error500InternalServerError("adapter error")
		}
		if connectURL := normalizeWSURL(asString(cfg, "connectUrl")); connectURL != "" {
			pa.ConnectURL = connectURL
		}
		pa.Token = asStringOrKeep(cfg, "accessToken", pa.Token)
	case "onebot-reverse":
		pa, ok := ep.Adapter.(*dice.PlatformAdapterOnebot)
		if !ok {
			return huma.Error500InternalServerError("adapter error")
		}
		if reverseAddr := asString(cfg, "reverseAddr"); reverseAddr != "" {
			pa.ReverseUrl = reverseAddr
		}
	case "milky":
		pa, ok := ep.Adapter.(*dice.PlatformAdapterMilky)
		if !ok {
			return huma.Error500InternalServerError("adapter error")
		}
		pa.WsGateway = asStringOrKeep(cfg, "wsGateway", pa.WsGateway)
		pa.RestGateway = asStringOrKeep(cfg, "restGateway", pa.RestGateway)
		pa.Token = asStringOrKeep(cfg, "token", pa.Token)
	case "milky-internal":
		pa, ok := ep.Adapter.(*dice.PlatformAdapterMilky)
		if !ok {
			return huma.Error500InternalServerError("adapter error")
		}
		if mode := asString(cfg, "builtInMode"); mode == "yogurt" || mode == "lagrangeV2" {
			pa.BuiltInMode = mode
		}
	case "sealchat":
		pa, ok := ep.Adapter.(*dice.PlatformAdapterSealChat)
		if !ok {
			return huma.Error500InternalServerError("adapter error")
		}
		pa.ConnectURL = asStringOrKeep(cfg, "url", pa.ConnectURL)
		pa.Token = asStringOrKeep(cfg, "token", pa.Token)
	case "satori":
		pa, ok := ep.Adapter.(*dice.PlatformAdapterSatori)
		if !ok {
			return huma.Error500InternalServerError("adapter error")
		}
		pa.Platform = asStringOrKeep(cfg, "platform", pa.Platform)
		ep.Platform = pa.Platform
		pa.Host = asStringOrKeep(cfg, "host", pa.Host)
		if port := asInt(cfg, "port"); port != 0 {
			pa.Port = port
		}
		pa.Token = asStringOrKeep(cfg, "token", pa.Token)
	case "discord":
		pa, ok := ep.Adapter.(*dice.PlatformAdapterDiscord)
		if !ok {
			return huma.Error500InternalServerError("adapter error")
		}
		pa.Token = asStringOrKeep(cfg, "token", pa.Token)
		pa.ProxyURL = asStringIfPresent(cfg, "proxyURL", pa.ProxyURL)
		pa.ReverseProxyUrl = asStringIfPresent(cfg, "reverseProxyUrl", pa.ReverseProxyUrl)
		pa.ReverseProxyCDNUrl = asStringIfPresent(cfg, "reverseProxyCDNUrl", pa.ReverseProxyCDNUrl)
	case "kook":
		pa, ok := ep.Adapter.(*dice.PlatformAdapterKook)
		if !ok {
			return huma.Error500InternalServerError("adapter error")
		}
		pa.Token = asStringOrKeep(cfg, "token", pa.Token)
	case "telegram":
		pa, ok := ep.Adapter.(*dice.PlatformAdapterTelegram)
		if !ok {
			return huma.Error500InternalServerError("adapter error")
		}
		pa.Token = asStringOrKeep(cfg, "token", pa.Token)
		pa.ProxyURL = asStringIfPresent(cfg, "proxyURL", pa.ProxyURL)
	case "minecraft":
		pa, ok := ep.Adapter.(*dice.PlatformAdapterMinecraft)
		if !ok {
			return huma.Error500InternalServerError("adapter error")
		}
		pa.ConnectURL = asStringOrKeep(cfg, "url", pa.ConnectURL)
	case "dodo":
		pa, ok := ep.Adapter.(*dice.PlatformAdapterDodo)
		if !ok {
			return huma.Error500InternalServerError("adapter error")
		}
		pa.ClientID = asStringOrKeep(cfg, "clientID", pa.ClientID)
		pa.Token = asStringOrKeep(cfg, "token", pa.Token)
	case "dingtalk":
		pa, ok := ep.Adapter.(*dice.PlatformAdapterDingTalk)
		if !ok {
			return huma.Error500InternalServerError("adapter error")
		}
		pa.ClientID = asStringOrKeep(cfg, "clientID", pa.ClientID)
		pa.Token = asStringOrKeep(cfg, "token", pa.Token)
		ep.Nickname = asStringIfPresent(cfg, "nickname", ep.Nickname)
		pa.RobotCode = asStringOrKeep(cfg, "robotCode", pa.RobotCode)
	case "slack":
		pa, ok := ep.Adapter.(*dice.PlatformAdapterSlack)
		if !ok {
			return huma.Error500InternalServerError("adapter error")
		}
		pa.BotToken = asStringOrKeep(cfg, "botToken", pa.BotToken)
		pa.AppToken = asStringOrKeep(cfg, "appToken", pa.AppToken)
	case "officialqq":
		pa, ok := ep.Adapter.(*dice.PlatformAdapterOfficialQQ)
		if !ok {
			return huma.Error500InternalServerError("adapter error")
		}
		if appID := asInt(cfg, "appID"); appID != 0 {
			pa.AppID = uint64(appID)
		}
		pa.AppSecret = asStringOrKeep(cfg, "appSecret", pa.AppSecret)
		pa.Token = asStringOrKeep(cfg, "token", pa.Token)
		if _, ok := cfg["onlyQQGuild"]; ok {
			pa.OnlyQQGuild = asBool(cfg, "onlyQQGuild")
		}
	default:
		return huma.Error501NotImplemented("not implemented")
	}
	return nil
}

func (s *Service) reconnectIfEnabled(ep *dice.EndPointInfo) {
	if ep == nil || !ep.Enable || !s.autoServe {
		return
	}
	ep.SetEnable(s.dice, false)
	ep.SetEnable(s.dice, true)
}

func (s *Service) save() {
	if s.autoSave {
		s.dice.Save(false)
	}
}

func (s *Service) GetWorkflow(_ context.Context, p *imconnm.IDPath) (*response.ItemResponse[imconnm.WorkflowResp], error) {
	ep := s.findEndpoint(p.ID)
	if ep == nil {
		return nil, huma.Error404NotFound("not found")
	}
	item := workflowOf(ep)
	return response.NewItemResponse[imconnm.WorkflowResp](item), nil
}

func workflowOf(ep *dice.EndPointInfo) imconnm.WorkflowResp {
	switch pa := ep.Adapter.(type) {
	case *dice.PlatformAdapterGocq:
		state, hasQR := mapGocqWorkflow(pa.GoCqhttpState, len(pa.GoCqhttpQrcodeData) > 0)
		return imconnm.WorkflowResp{
			State:        state,
			HasQRCode:    hasQR,
			LoginState:   int64(pa.GoCqhttpState),
			FailedReason: pa.GocqhttpLoginFailedReason,
		}
	case *dice.PlatformAdapterMilky:
		state, hasQR := mapMilkyWorkflow(pa.BuiltInLoginState, len(pa.QrCodeData) > 0)
		return imconnm.WorkflowResp{
			State:      state,
			HasQRCode:  hasQR,
			LoginState: int64(pa.BuiltInLoginState),
		}
	default:
		return imconnm.WorkflowResp{State: "none"}
	}
}

func mapGocqWorkflow(state int, hasQR bool) (string, bool) {
	switch state {
	case dice.StateCodeInLoginQrCode:
		return "qrcode", hasQR
	case dice.StateCodeInLogin:
		return "pending", false
	case dice.StateCodeLoginSuccessed:
		return "success", false
	case dice.StateCodeLoginFailed:
		return "failed", false
	default:
		return "idle", false
	}
}

func mapMilkyWorkflow(state dice.MilkyLoginState, hasQR bool) (string, bool) {
	switch state {
	case dice.MilkyLoginStateQRWaitingForScan:
		return "qrcode", hasQR
	case dice.MilkyLoginStateConnecting:
		return "pending", false
	case dice.MilkyLoginStateQRConnected:
		return "success", false
	case dice.MilkyLoginStateFailed:
		return "failed", false
	default:
		return "idle", false
	}
}

func (s *Service) GetQRCode(_ context.Context, p *imconnm.IDPath) (*response.ItemResponse[imconnm.QRCodeResp], error) {
	ep := s.findEndpoint(p.ID)
	if ep == nil {
		return nil, huma.Error404NotFound("not found")
	}
	img := ""
	switch pa := ep.Adapter.(type) {
	case *dice.PlatformAdapterGocq:
		if pa.GoCqhttpState == dice.StateCodeInLoginQrCode && len(pa.GoCqhttpQrcodeData) > 0 {
			img = "data:image/png;base64," + base64.StdEncoding.EncodeToString(pa.GoCqhttpQrcodeData)
		}
	case *dice.PlatformAdapterMilky:
		if pa.BuiltInLoginState == dice.MilkyLoginStateQRWaitingForScan && len(pa.QrCodeData) > 0 {
			img = "data:image/png;base64," + base64.StdEncoding.EncodeToString(pa.QrCodeData)
		}
	}
	if img == "" {
		return nil, huma.Error404NotFound("qrcode not found")
	}
	return response.NewItemResponse[imconnm.QRCodeResp](imconnm.QRCodeResp{Img: img}), nil
}

func (s *Service) findEndpoint(id string) *dice.EndPointInfo {
	for _, ep := range s.dice.ImSession.EndPoints {
		if ep != nil && ep.ID == id {
			return ep
		}
	}
	return nil
}

func normalizeWSURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "ws://") || strings.HasPrefix(value, "wss://") {
		return value
	}
	if strings.HasPrefix(value, "http://") {
		return "ws://" + strings.TrimPrefix(value, "http://")
	}
	if strings.HasPrefix(value, "https://") {
		return "wss://" + strings.TrimPrefix(value, "https://")
	}
	return "ws://" + value
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

func asStringOrKeep(m map[string]interface{}, key string, old string) string {
	if _, ok := m[key]; !ok {
		return old
	}
	value := asString(m, key)
	if value == "" {
		return old
	}
	return value
}

func asStringIfPresent(m map[string]interface{}, key string, old string) string {
	if _, ok := m[key]; !ok {
		return old
	}
	return asString(m, key)
}

func asInt(m map[string]interface{}, key string) int {
	v, ok := m[key]
	if !ok || v == nil {
		return 0
	}
	switch vv := v.(type) {
	case int:
		return vv
	case int64:
		return int(vv)
	case uint64:
		return int(vv)
	case float64:
		return int(vv)
	case string:
		n, _ := strconv.Atoi(vv)
		return n
	default:
		return 0
	}
}

func asBool(m map[string]interface{}, key string) bool {
	v, ok := m[key]
	if !ok || v == nil {
		return false
	}
	switch vv := v.(type) {
	case bool:
		return vv
	case string:
		return vv == "true" || vv == "1"
	default:
		return false
	}
}
