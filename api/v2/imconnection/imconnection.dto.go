package imconnection

import (
	dynamicform "sealdice-core/api/v2/imconnection/dynamic_form"
	"sealdice-core/dice"
)

type IDPath struct {
	ID string `path:"id"`
}

type CreateBody struct {
	Platform string                 `json:"platform"`
	Config   map[string]interface{} `json:"config"`
}

type CreateReq struct {
	Body CreateBody `json:"body"`
}

type OfficialQQTestBody struct {
	Config map[string]interface{} `json:"config"`
}

type OfficialQQTestReq struct {
	Body OfficialQQTestBody `json:"body"`
}

type EnableBody struct {
	Enable bool `json:"enable"`
}

type UpdateReq struct {
	ID   string                 `path:"id"`
	Body map[string]interface{} `json:"body"`
}

type EnableReq struct {
	ID   string     `path:"id"`
	Body EnableBody `json:"body"`
}

type EndpointListResp struct {
	Items []*EndPointInfo `json:"items"`
}

type EndPointInfo struct {
	ID                  string                 `json:"id"`
	Nickname            string                 `json:"nickname"`
	State               dice.EndpointState     `json:"state"`
	UserID              string                 `json:"userId"`
	GroupNum            int64                  `json:"groupNum"`
	CmdExecutedNum      int64                  `json:"cmdExecutedNum"`
	CmdExecutedLastTime int64                  `json:"cmdExecutedLastTime"`
	OnlineTotalTime     int64                  `json:"onlineTotalTime"`
	Platform            string                 `json:"platform"`
	RelWorkDir          string                 `json:"relWorkDir"`
	Enable              bool                   `json:"enable"`
	ProtocolType        string                 `json:"protocolType"`
	IsPublic            bool                   `json:"isPublic"`
	Adapter             map[string]interface{} `json:"adapter"`
}

func NewEndPointInfo(ep *dice.EndPointInfo) *EndPointInfo {
	if ep == nil {
		return nil
	}
	return &EndPointInfo{
		ID:                  ep.ID,
		Nickname:            ep.Nickname,
		State:               ep.State,
		UserID:              ep.UserID,
		GroupNum:            ep.GroupNum,
		CmdExecutedNum:      ep.CmdExecutedNum,
		CmdExecutedLastTime: ep.CmdExecutedLastTime,
		OnlineTotalTime:     ep.OnlineTotalTime,
		Platform:            ep.Platform,
		RelWorkDir:          ep.RelWorkDir,
		Enable:              ep.Enable,
		ProtocolType:        ep.ProtocolType,
		IsPublic:            ep.IsPublic,
		Adapter:             publicAdapterFields(ep.Adapter),
	}
}

func NewEndPointInfoList(endpoints []*dice.EndPointInfo) []*EndPointInfo {
	items := make([]*EndPointInfo, 0, len(endpoints))
	for _, ep := range endpoints {
		if item := NewEndPointInfo(ep); item != nil {
			items = append(items, item)
		}
	}
	return items
}

func publicAdapterFields(adapter dice.PlatformAdapter) map[string]interface{} {
	fields := map[string]interface{}{}
	switch pa := adapter.(type) {
	case *dice.PlatformAdapterGocq:
		fields["reverseAddr"] = pa.ReverseAddr
		fields["connectUrl"] = pa.ConnectURL
		fields["builtinMode"] = pa.BuiltinMode
		fields["signServerVer"] = pa.SignServerVer
		fields["signServerName"] = pa.SignServerName
	case *dice.PlatformAdapterOnebot:
		fields["connectUrl"] = pa.ConnectURL
		fields["reverseAddr"] = pa.ReverseUrl
	case *dice.PlatformAdapterMilky:
		fields["ws_gateway"] = pa.WsGateway
		fields["built_in_mode"] = pa.BuiltInMode
	case *dice.PlatformAdapterOfficialQQ:
		fields["useWebhook"] = pa.UseWebhook
		fields["webhookPath"] = pa.WebhookPath
		fields["webhookPort"] = pa.WebhookPort
	case *dice.PlatformAdapterSatori:
		fields["host"] = pa.Host
		fields["port"] = pa.Port
	case *dice.PlatformAdapterSealChat:
		fields["connectUrl"] = pa.ConnectURL
	}
	return fields
}

type ProtocolCapability struct {
	Create   bool `json:"create"`
	Update   bool `json:"update"`
	Delete   bool `json:"delete"`
	Enable   bool `json:"enable"`
	Workflow bool `json:"workflow"`
	QRCode   bool `json:"qrcode"`
	SignInfo bool `json:"signInfo"`
}

type ProtocolDefinition struct {
	Key            string             `json:"key"`
	Name           string             `json:"name"`
	Platform       string             `json:"platform"`
	SchemaKey      string             `json:"schemaKey"`
	Deprecated     bool               `json:"deprecated"`
	Available      bool               `json:"available"`
	DisabledReason string             `json:"disabledReason,omitempty"`
	Capabilities   ProtocolCapability `json:"capabilities"`
	Description    string             `json:"description,omitempty"`
}

type MethodTreeNode struct {
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Protocols   []*ProtocolDefinition `json:"protocols"`
}

type PlatformTreeNode struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Methods     []*MethodTreeNode `json:"methods"`
}

type ProtocolListResp struct {
	Items []*PlatformTreeNode `json:"items"`
}

type EditableConfigResp struct {
	ProtocolKey     string                        `json:"protocolKey"`
	Schema          []*dynamicform.FormConfigItem `json:"schema"`
	Config          map[string]interface{}        `json:"config"`
	RestartRequired bool                          `json:"restartRequired"`
}

type SignServer struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Latency  int64  `json:"latency"`
	Selected bool   `json:"selected"`
	Ignored  bool   `json:"ignored"`
	Note     string `json:"note"`
}

type SignInfo struct {
	Version  string                 `json:"version"`
	AppInfo  map[string]interface{} `json:"appinfo"`
	Servers  []*SignServer          `json:"servers"`
	Selected bool                   `json:"selected"`
	Ignored  bool                   `json:"ignored"`
	Note     string                 `json:"note"`
}

type SignInfoResp struct {
	Items []SignInfo `json:"items"`
}

type WorkflowResp struct {
	State        string `json:"state"`
	Message      string `json:"message,omitempty"`
	HasQRCode    bool   `json:"hasQRCode"`
	LoginState   int64  `json:"loginState"`
	FailedReason string `json:"failedReason,omitempty"`
}

type QRCodeResp struct {
	Img string `json:"img"`
}

type OfficialQQTestResp struct {
	TestOnly bool   `json:"testOnly"`
	ID       string `json:"id,omitempty"`
	UserID   string `json:"userId"`
	UIN      string `json:"uin"`
	Nickname string `json:"nickname"`
	Exists   bool   `json:"exists"`
}
