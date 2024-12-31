package public_dice

import (
	"github.com/monaco-io/request"
)

// PublicDiceClient SDK客户端
type PublicDiceClient struct {
	baseURL string
	token   string
}

// NewClient 创建新的SDK客户端
func NewClient(baseURL string, token string) *PublicDiceClient {
	return &PublicDiceClient{
		baseURL: baseURL,
		token:   token,
	}
}

// doReq 发送HTTP请求
func doReq[T any](c *PublicDiceClient, method string, path string, data any, params map[string]string) (*T, int) {
	req := request.Client{
		URL:    c.baseURL + path,
		Method: method,
		JSON:   data,
		Query:  params,
	}

	var result T
	resp := req.Send()
	resp.Scan(&result)

	return &result, resp.Code()
}

// Endpoint 公骰终端信息
type Endpoint struct {
	Platform  string `json:"platform" msgpack:",omitempty"`
	UID       string `json:"uid" msgpack:",omitempty"`
	InviteURL string `json:"inviteUrl" msgpack:",omitempty"`
	IsOnline  bool   `json:"isOnline" msgpack:",omitempty"`

	ID           string `json:"id" msgpack:",omitempty"`
	CreatedAt    string `json:"createdAt" msgpack:",omitempty"`
	UpdatedAt    string `json:"updatedAt" msgpack:",omitempty"`
	LastTickTime int64  `json:"lastTickTime" msgpack:",omitempty"`
}

// DiceInfo 公骰信息
type DiceInfo struct {
	ID                string      `json:"id" msgpack:",omitempty"`
	CreatedAt         string      `json:"createdAt" msgpack:",omitempty"`
	UpdatedAt         string      `json:"updatedAt" msgpack:",omitempty"`
	Name              string      `json:"name" msgpack:",omitempty"`
	Brief             string      `json:"brief" msgpack:",omitempty"`
	Note              string      `json:"note" msgpack:",omitempty"`
	Avatar            string      `json:"avatar" msgpack:",omitempty"`
	Version           string      `json:"version" msgpack:",omitempty"`
	IsOfficialVersion bool        `json:"isOfficialVersion" msgpack:",omitempty"`
	UpdateTickCount   int         `json:"updateTickCount" msgpack:",omitempty"`
	LastTickTime      int64       `json:"lastTickTime" msgpack:",omitempty"`
	Endpoints         []*Endpoint `json:"endpoints" msgpack:",omitempty"`
}

// ListResponse 公骰列表响应
type ListResponse struct {
	Items []*DiceInfo `json:"items"`
}

// ListGet 获取公骰列表
func (c *PublicDiceClient) ListGet(keyFunc func(data any) string) (*ListResponse, int) {
	if keyFunc != nil {
		data := keyFunc(nil)
		return doReq[ListResponse](c, "GET", "/dice/api/public-dice/list", data, nil)
	}
	return doReq[ListResponse](c, "GET", "/dice/api/public-dice/list", nil, nil)
}

// RegisterRequest 注册公骰请求
type RegisterRequest struct {
	ID     string `json:"ID,omitempty" msgpack:",omitempty"`
	Name   string `json:"name,omitempty" msgpack:",omitempty"` // 15字
	Brief  string `json:"brief,omitempty" msgpack:",omitempty"`
	Note   string `json:"note,omitempty" msgpack:",omitempty"`
	Avatar string `json:"avatar,omitempty" msgpack:",omitempty"` // 头像？还是用另一个api进行注册比较好？可以省略
	Key    string `json:"key,omitempty" msgpack:",omitempty"`
}

// RegisterResponse 注册公骰响应
type RegisterResponse struct {
	Item DiceInfo `json:"item"`
}

// Register 注册公骰
func (c *PublicDiceClient) Register(req *RegisterRequest, keyFunc func(data any) string) (*RegisterResponse, int) {
	if keyFunc != nil {
		req.Key = keyFunc(req)
	}
	return doReq[RegisterResponse](c, "POST", "/dice/api/public-dice/register", req, nil)
}

// DiceUpdateRequest 更新公骰请求
type DiceUpdateRequest struct {
	ID    string `json:"id" msgpack:",omitempty"`
	Name  string `json:"name" msgpack:",omitempty"`
	Brief string `json:"brief" msgpack:",omitempty"`
	Note  string `json:"note" msgpack:",omitempty"`
	Key   string `json:"key" msgpack:",omitempty"`
}

// DiceUpdateResponse 更新公骰响应
type DiceUpdateResponse struct {
	Updated int `json:"updated"`
}

// DiceUpdate 更新公骰
func (c *PublicDiceClient) DiceUpdate(req *DiceUpdateRequest, keyFunc func(data any) string) (*DiceUpdateResponse, int) {
	if keyFunc != nil {
		req.Key = keyFunc(req)
	}
	return doReq[DiceUpdateResponse](c, "POST", "/dice/api/public-dice/register?update=1", req, nil)
}

// EndpointUpdateRequest 更新公骰SNS账号信息请求
type EndpointUpdateRequest struct {
	DiceID    string      `json:"diceId" msgpack:",omitempty"`
	Key       string      `json:"key" msgpack:",omitempty"`
	Endpoints []*Endpoint `json:"endpoints" msgpack:",omitempty"`
}

// EndpointUpdateResponse 更新公骰SNS账号信息响应
type EndpointUpdateResponse struct{}

// EndpointUpdate 更新公骰SNS账号信息
func (c *PublicDiceClient) EndpointUpdate(req *EndpointUpdateRequest, keyFunc func(data any) string) (*EndpointUpdateResponse, int) {
	if keyFunc != nil {
		req.Key = keyFunc(req)
	}
	return doReq[EndpointUpdateResponse](c, "POST", "/dice/api/public-dice/endpoint-update", req, nil)
}

// TickUpdateRequest 更新公骰心跳请求
type TickUpdateRequest struct {
	ID        string          `json:"ID" msgpack:",omitempty"`
	Key       string          `json:"key" msgpack:",omitempty"`
	Endpoints []*TickEndpoint `json:"Endpoints" msgpack:",omitempty"`
}

// TickEndpoint 公骰心跳端点信息
type TickEndpoint struct {
	UID      string `json:"uid" msgpack:",omitempty"`
	IsOnline bool   `json:"isOnline" msgpack:",omitempty"`
}

// TickUpdateResponse 更新公骰心跳响应
type TickUpdateResponse struct{}

// TickUpdate 更新公骰心跳
func (c *PublicDiceClient) TickUpdate(req *TickUpdateRequest, keyFunc func(data any) string) (*TickUpdateResponse, int) {
	if keyFunc != nil {
		req.Key = keyFunc(req)
	}
	return doReq[TickUpdateResponse](c, "POST", "/dice/api/public-dice/tick-update", req, nil)
}
