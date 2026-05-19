package imconnection_test

import (
	"strings"
	"testing"

	imconnection "sealdice-core/api/v2/imconnection"
	dynamicform "sealdice-core/api/v2/imconnection/dynamic_form"
	imconnm "sealdice-core/api/v2/model/imconnection"
	"sealdice-core/dice"
	"sealdice-core/logger"
	"sealdice-core/model/common/request"
)

func newTestService(t *testing.T, containerMode bool) *imconnection.Service {
	t.Helper()
	d := &dice.Dice{
		Logger: logger.M(),
	}
	d.BaseConfig.Name = "test"
	d.BaseConfig.DataDir = t.TempDir()
	d.ImSession = &dice.IMSession{
		Parent:       d,
		EndPoints:    []*dice.EndPointInfo{},
		ServiceAtNew: new(dice.SyncMap[string, *dice.GroupInfo]),
		PendingQuits: new(dice.SyncMap[string, *dice.PendingQuitInfo]),
	}
	dm := &dice.DiceManager{
		Dice:          []*dice.Dice{d},
		ContainerMode: containerMode,
	}
	d.Parent = dm
	return imconnection.NewServiceWithOptions(dm, false, false)
}

func protocolByKey(tree []*imconnm.PlatformTreeNode, key string) *imconnm.ProtocolDefinition {
	for _, platform := range tree {
		for _, method := range platform.Methods {
			for _, item := range method.Protocols {
				if item != nil && item.Key == key {
					return item
				}
			}
		}
	}
	return nil
}

func TestGetProtocolsReturnsCapabilitiesAndContainerAvailability(t *testing.T) {
	svc := newTestService(t, true)

	resp, err := svc.GetProtocols(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("GetProtocols returned error: %v", err)
	}
	items := resp.Body.Item.Items

	lagrange := protocolByKey(items, "lagrange")
	if lagrange == nil {
		t.Fatalf("lagrange protocol missing")
	}
	if lagrange.Available {
		t.Fatalf("lagrange should be unavailable in container mode")
	}
	if !strings.Contains(lagrange.DisabledReason, "容器") {
		t.Fatalf("lagrange disabled reason = %q, want container hint", lagrange.DisabledReason)
	}
	if !lagrange.Capabilities.Workflow || !lagrange.Capabilities.QRCode || !lagrange.Capabilities.SignInfo {
		t.Fatalf("lagrange capabilities should include workflow, qrcode and sign-info")
	}

	gocq := protocolByKey(items, "gocq")
	if gocq == nil || !gocq.Deprecated {
		t.Fatalf("gocq should be listed as deprecated")
	}
	red := protocolByKey(items, "red")
	if red == nil || !red.Deprecated {
		t.Fatalf("red should be listed as deprecated")
	}
}

func TestCreateConnectionSupportsRetainedProtocolsWithoutStartingServers(t *testing.T) {
	svc := newTestService(t, false)

	tests := []struct {
		name         string
		platform     string
		config       map[string]interface{}
		wantPlatform string
		wantProtocol string
		wantUserID   string
	}{
		{"lagrange", "lagrange", map[string]interface{}{"account": "10001", "signServerVersion": "30366", "signServerName": "sealdice"}, "QQ", "onebot", "QQ:10001"},
		{"onebot-forward", "gocq-separate", map[string]interface{}{"account": "10002", "connectUrl": "127.0.0.1:3001", "accessToken": "tok"}, "QQ", "pureonebot", "QQ:10002"},
		{"onebot-reverse", "onebot-reverse", map[string]interface{}{"account": "10003", "reverseAddr": ":4001"}, "QQ", "pureonebot", "QQ:10003"},
		{"milky", "milky", map[string]interface{}{"wsGateway": "ws://127.0.0.1:3000/event", "restGateway": "http://127.0.0.1:3000/api"}, "QQ", "milky", ""},
		{"milky-internal", "milky-internal", map[string]interface{}{"account": "10004", "builtInMode": "yogurt"}, "QQ", "milky", "QQ:10004"},
		{"sealchat", "sealchat", map[string]interface{}{"url": "ws://127.0.0.1:3212/ws/seal", "token": "tok"}, "SEALCHAT", "", ""},
		{"satori", "satori", map[string]interface{}{"platform": "QQ", "host": "127.0.0.1", "port": 5500}, "QQ", "satori", ""},
		{"discord", "discord", map[string]interface{}{"token": "tok"}, "DISCORD", "", ""},
		{"kook", "kook", map[string]interface{}{"token": "tok"}, "KOOK", "", ""},
		{"telegram", "telegram", map[string]interface{}{"token": "tok"}, "TG", "", ""},
		{"minecraft", "minecraft", map[string]interface{}{"url": "127.0.0.1:8887"}, "MC", "", ""},
		{"dodo", "dodo", map[string]interface{}{"clientID": "cid", "token": "tok"}, "DODO", "", ""},
		{"dingtalk", "dingtalk", map[string]interface{}{"clientID": "cid", "token": "tok", "robotCode": "robot"}, "DINGTALK", "", ""},
		{"slack", "slack", map[string]interface{}{"botToken": "bt", "appToken": "at"}, "SLACK", "", ""},
		{"officialqq", "officialqq", map[string]interface{}{"appID": 12345, "appSecret": "sec", "token": "tok", "onlyQQGuild": true}, "QQ", "official", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.CreateConnection(t.Context(), request.NewRequestWrapper(imconnm.CreateBody{
				Platform: tt.platform,
				Config:   tt.config,
			}))
			if err != nil {
				t.Fatalf("CreateConnection returned error: %v", err)
			}
			item := resp.Body.Item
			if item.Platform != tt.wantPlatform {
				t.Fatalf("Platform = %q, want %q", item.Platform, tt.wantPlatform)
			}
			if item.ProtocolType != tt.wantProtocol {
				t.Fatalf("ProtocolType = %q, want %q", item.ProtocolType, tt.wantProtocol)
			}
			if tt.wantUserID != "" && item.UserID != tt.wantUserID {
				t.Fatalf("UserID = %q, want %q", item.UserID, tt.wantUserID)
			}
		})
	}
}

func TestCreateConnectionRejectsDeprecatedAndInvalidConfig(t *testing.T) {
	svc := newTestService(t, false)

	if _, err := svc.CreateConnection(t.Context(), request.NewRequestWrapper(imconnm.CreateBody{
		Platform: "gocq",
		Config:   map[string]interface{}{"account": "10001"},
	})); err == nil {
		t.Fatalf("expected deprecated gocq create to fail")
	}

	if _, err := svc.CreateConnection(t.Context(), request.NewRequestWrapper(imconnm.CreateBody{
		Platform: "gocq-separate",
		Config:   map[string]interface{}{"account": "10001"},
	})); err == nil {
		t.Fatalf("expected missing connectUrl to fail")
	}
}

func TestEnableWorkflowAndQRCode(t *testing.T) {
	svc := newTestService(t, false)
	resp, err := svc.CreateConnection(t.Context(), request.NewRequestWrapper(imconnm.CreateBody{
		Platform: "milky-internal",
		Config:   map[string]interface{}{"account": "10001", "builtInMode": "yogurt"},
	}))
	if err != nil {
		t.Fatalf("CreateConnection returned error: %v", err)
	}
	ep := resp.Body.Item

	if _, enableErr := svc.SetEnable(t.Context(), &imconnm.EnableReq{
		ID: ep.ID,
		Body: request.RequestWrapper[imconnm.EnableBody]{
			Body: imconnm.EnableBody{Enable: true},
		},
	}); enableErr != nil {
		t.Fatalf("SetEnable returned error: %v", enableErr)
	}
	if !ep.Enable {
		t.Fatalf("endpoint should be enabled")
	}
	if _, disableErr := svc.SetEnable(t.Context(), &imconnm.EnableReq{
		ID: ep.ID,
		Body: request.RequestWrapper[imconnm.EnableBody]{
			Body: imconnm.EnableBody{Enable: false},
		},
	}); disableErr != nil {
		t.Fatalf("SetEnable disable returned error: %v", disableErr)
	}
	if ep.Enable {
		t.Fatalf("endpoint should be disabled")
	}

	pa := ep.Adapter.(*dice.PlatformAdapterMilky)
	pa.BuiltInLoginState = dice.MilkyLoginStateQRWaitingForScan
	pa.QrCodeData = []byte("fake-png")

	workflow, err := svc.GetWorkflow(t.Context(), &imconnm.IDPath{ID: ep.ID})
	if err != nil {
		t.Fatalf("GetWorkflow returned error: %v", err)
	}
	if workflow.Body.Item.State != "qrcode" || !workflow.Body.Item.HasQRCode {
		t.Fatalf("workflow = %+v, want qrcode with QRCode", workflow.Body.Item)
	}

	qrcode, err := svc.GetQRCode(t.Context(), &imconnm.IDPath{ID: ep.ID})
	if err != nil {
		t.Fatalf("GetQRCode returned error: %v", err)
	}
	if !strings.HasPrefix(qrcode.Body.Item.Img, "data:image/png;base64,") {
		t.Fatalf("qrcode img should be a data URL, got %q", qrcode.Body.Item.Img)
	}
}

func TestProtocolSchemasUseSensitiveMetadata(t *testing.T) {
	svc := newTestService(t, false)
	resp, err := svc.GetSchemas(t.Context(), &request.Empty{})
	if err != nil {
		t.Fatalf("GetSchemas returned error: %v", err)
	}
	tokenID := uint64(0)
	for _, it := range resp.Body.Item["discord"] {
		if it.FieldName == "token" {
			tokenID = it.ID
			if !it.Sensitive {
				t.Fatalf("discord token should be sensitive")
			}
		}
	}
	if tokenID == 0 {
		t.Fatalf("discord token field missing")
	}
	if _, err := dynamicform.BuildParamsByConfig(resp.Body.Item["discord"], map[string]interface{}{}); err == nil {
		t.Fatalf("missing required token should fail")
	}
}

func TestEditableConfigMasksSensitiveFieldsAndIncludesPlaceholders(t *testing.T) {
	svc := newTestService(t, false)
	createResp, err := svc.CreateConnection(t.Context(), request.NewRequestWrapper(imconnm.CreateBody{
		Platform: "discord",
		Config: map[string]interface{}{
			"token":              "old-token",
			"proxyURL":           "http://127.0.0.1:7890",
			"reverseProxyUrl":    "https://discord-proxy.example",
			"reverseProxyCDNUrl": "https://cdn-proxy.example",
		},
	}))
	if err != nil {
		t.Fatalf("CreateConnection returned error: %v", err)
	}

	resp, err := svc.GetEditableConfig(t.Context(), &imconnm.IDPath{ID: createResp.Body.Item.ID})
	if err != nil {
		t.Fatalf("GetEditableConfig returned error: %v", err)
	}
	if resp.Body.Item.ProtocolKey != "discord" {
		t.Fatalf("ProtocolKey = %q, want discord", resp.Body.Item.ProtocolKey)
	}
	if got := resp.Body.Item.Config["token"]; got != "" {
		t.Fatalf("sensitive token should be masked, got %q", got)
	}
	if got := resp.Body.Item.Config["proxyURL"]; got != "http://127.0.0.1:7890" {
		t.Fatalf("proxyURL = %q, want existing value", got)
	}
	if !resp.Body.Item.RestartRequired {
		t.Fatalf("editing connection config should require reconnect")
	}
	for _, item := range resp.Body.Item.Schema {
		if item.FieldName != "" && item.Placeholder == "" {
			t.Fatalf("schema field %s should include placeholder", item.FieldName)
		}
		if item.FieldName == "account" && !item.Readonly {
			t.Fatalf("identity field account should be readonly in edit schema")
		}
	}
}

func TestUpdateConnectionPreservesSensitiveFieldsAndRejectsIdentityChange(t *testing.T) {
	svc := newTestService(t, false)
	createResp, err := svc.CreateConnection(t.Context(), request.NewRequestWrapper(imconnm.CreateBody{
		Platform: "gocq-separate",
		Config: map[string]interface{}{
			"account":     "10002",
			"connectUrl":  "127.0.0.1:3001",
			"accessToken": "old-token",
		},
	}))
	if err != nil {
		t.Fatalf("CreateConnection returned error: %v", err)
	}
	ep := createResp.Body.Item
	pa := ep.Adapter.(*dice.PlatformAdapterOnebot)

	if _, err := svc.UpdateConnection(t.Context(), &imconnm.UpdateReq{
		ID: ep.ID,
		Body: request.RequestWrapper[map[string]interface{}]{
			Body: map[string]interface{}{
				"connectUrl":  "127.0.0.1:3002",
				"accessToken": "",
				"account":     "10003",
			},
		},
	}); err != nil {
		t.Fatalf("UpdateConnection returned error: %v", err)
	}
	if ep.UserID != "QQ:10002" {
		t.Fatalf("UserID should not be editable, got %q", ep.UserID)
	}
	if pa.ConnectURL != "ws://127.0.0.1:3002" {
		t.Fatalf("ConnectURL = %q, want normalized updated URL", pa.ConnectURL)
	}
	if pa.Token != "old-token" {
		t.Fatalf("empty sensitive accessToken should preserve old value, got %q", pa.Token)
	}
}
