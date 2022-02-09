package model

import (
	"encoding/hex"
	"encoding/json"
)

type Todo struct {
	BytePKBaseModel

	Title  string `form:"name" json:"name"  binding:"required"` // 标题
	Parent []byte `gorm:"index;null" json:"parent"`             // 所属checklist

	UpdateTime   int64  `gorm:"index" form:"updateTime" json:"updateTime"`     // 更新时间
	DeadlineTime int64  `gorm:"index" form:"deadlineTime" json:"deadlineTime"` // 截止时间
	IsDone       bool   `gorm:"index" json:"isDone"`                           // 是否完成
	Note         string `gorm:"null" form:"note" json:"note"`                  // 备注
}

func (m Todo) MarshalJSON() ([]byte, error) {
	type Alias Todo
	return json.Marshal(&struct {
		ID     string `json:"id"`
		Parent string `json:"parent"`
		*Alias
	}{
		ID:     hex.EncodeToString(m.ID[:]),
		Parent: hex.EncodeToString(m.Parent[:]),
		Alias:  (*Alias)(&m),
	})
}

func TodoGetList(parent []byte, all bool) []*Todo {
	q := Todo{Parent: parent}

	ret := []*Todo{}
	items := []Todo{}
	db.Where(&q).Order("created_at desc").Find(&items)

	return ret
}
