package ban

import (
	"github.com/danielgtaylor/huma/v2"

	"sealdice-core/model/common/request"
	"sealdice-core/model/common/response"
)

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

type BanPageReq struct {
	Body BanPageRequest `json:"body"`
}

type DeleteReq struct {
	ID string `json:"id"`
}

type DeleteEntryReq struct {
	Body DeleteReq `json:"body"`
}

type BatchDeleteReq struct {
	Body request.IDListReq `json:"body"`
}

type BanConfig struct {
	BanBehaviorRefuseReply                 bool    `json:"banBehaviorRefuseReply"`
	BanBehaviorRefuseInvite                bool    `json:"banBehaviorRefuseInvite"`
	BanBehaviorQuitLastPlace               bool    `json:"banBehaviorQuitLastPlace"`
	BanBehaviorQuitPlaceImmediately        bool    `json:"banBehaviorQuitPlaceImmediately"`
	BanBehaviorQuitIfAdmin                 bool    `json:"banBehaviorQuitIfAdmin"`
	BanBehaviorQuitIfAdminSilentIfNotAdmin bool    `json:"banBehaviorQuitIfAdminSilentIfNotAdmin"`
	ThresholdWarn                          int64   `json:"thresholdWarn"`
	ThresholdBan                           int64   `json:"thresholdBan"`
	AutoBanMinutes                         int64   `json:"autoBanMinutes"`
	ScoreReducePerMinute                   int64   `json:"scoreReducePerMinute"`
	ScoreGroupMuted                        int64   `json:"scoreGroupMuted"`
	ScoreGroupKicked                       int64   `json:"scoreGroupKicked"`
	ScoreTooManyCommand                    int64   `json:"scoreTooManyCommand"`
	JointScorePercentOfGroup               float64 `json:"jointScorePercentOfGroup"`
	JointScorePercentOfInviter             float64 `json:"jointScorePercentOfInviter"`
}

type ConfigReq struct {
	Body BanConfig `json:"body"`
}

type AddReqBody struct {
	ID     string `json:"id"`
	Rank   int    `json:"rank"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type AddReq struct {
	Body AddReqBody `json:"body"`
}

type ImportForm struct {
	File huma.FormFile `form:"file" required:"true"`
}

type ImportReq struct {
	RawBody huma.MultipartFormFiles[ImportForm]
}

type ConfigItemResponse = response.ItemResponse[BanConfig]
type SimpleItemResponse = response.ItemResponse[response.SimpleOK]
