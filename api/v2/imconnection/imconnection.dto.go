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

type EnableBody struct {
	Enable bool `json:"enable"`
}

type createConnectionInput struct {
	Body CreateBody
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
	Items []*dice.EndPointInfo `json:"items"`
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
