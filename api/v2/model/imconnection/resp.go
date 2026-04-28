package imconnection

import "sealdice-core/dice"

type EndpointListResp struct {
	Items []*dice.EndPointInfo `json:"items"`
}
