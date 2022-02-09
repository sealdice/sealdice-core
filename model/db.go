package model

import (
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

var db *gorm.DB

type BaseModel struct {
	ID        uint       `gorm:"primary_key" json:"id"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `sql:"index" json:"deletedAt"`
}

type BytePKBaseModel struct {
	ID        []byte     `gorm:"primary_key" json:"id"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `sql:"index" json:"deletedAt"`
}

func SqliteDBInit() {
	var err error
	db, err = gorm.Open("sqlite3", "info.db")
	if err != nil {
		panic("连接数据库失败")
	}
}

func DBInit() {
	SqliteDBInit()
}

func GetDB() *gorm.DB {
	return db
}
