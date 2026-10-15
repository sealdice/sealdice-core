package helpdoc

import (
	"sealdice-core/dice"
)

type StatusResp struct {
	Loading  bool `json:"loading"`
	TestMode bool `json:"testMode,omitempty"`
}

type TreeResp struct {
	Data []*dice.HelpDoc `json:"data"`
}

type HelpTextItemQuery struct {
	PageNum  int    `query:"pageNum"`
	PageSize int    `query:"pageSize"`
	ID       string `query:"id"`
	Group    string `query:"group"`
	From     string `query:"from"`
	Title    string `query:"title"`
}

type HelpTextItemPageResp struct {
	Total    int              `json:"total"`
	Data     dice.HelpTextVos `json:"data"`
	PageNum  int              `json:"pageNum"`
	PageSize int              `json:"pageSize"`
}

type HelpConfigBody struct {
	Aliases map[string][]string `json:"aliases"`
}

type ConfigReq struct {
	Body HelpConfigBody `json:"body"`
}

type ConfigResp = HelpConfigBody

type DeleteReqBody struct {
	Keys []string `json:"keys"`
}

type DeleteReq struct {
	Body DeleteReqBody `json:"body"`
}

type SimpleResp struct {
	Success  bool   `json:"success"`
	TestMode bool   `json:"testMode,omitempty"`
	Err      string `json:"err,omitempty"`
}
