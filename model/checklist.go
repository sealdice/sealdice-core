package model

import (
	"encoding/hex"
	"encoding/json"
)

type CheckList struct {
	BytePKBaseModel

	Title      string `form:"name" json:"name"  binding:"required"`      // 标题
	UpdateTime int64  `gorm:"index" form:"updateTime" json:"updateTime"` // 更新时间
	Note       string `gorm:"null" form:"note" json:"note"`              // 备注
}

func (m CheckList) MarshalJSON() ([]byte, error) {
	type Alias CheckList
	return json.Marshal(&struct {
		ID string `json:"id"`
		*Alias
	}{
		ID:    hex.EncodeToString(m.ID[:]),
		Alias: (*Alias)(&m),
	})
}
