package ban

import "sealdice-core/model/common/request"

type BanFilter struct {
	Ranks           []int  `json:"ranks" required:"false"`
	OrderByBanTime  bool   `json:"orderByBanTime" required:"false"`
	OrderByScore    bool   `json:"orderByScore" required:"false"`
	KeywordOverride string `json:"keyword,omitempty" required:"false"`
}

type BanPageRequest struct {
	request.PageInfo
	Filter BanFilter `json:"filter" required:"false"`
}

type DeleteReq struct {
	ID string `json:"id"`
}
