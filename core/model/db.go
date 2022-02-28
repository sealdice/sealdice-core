package model

import (
	"go.etcd.io/bbolt"
	"os"
)

var db *bbolt.DB

func BoltDBInit() {
	os.MkdirAll("./data", 0644)
	var err error
	db, err = bbolt.Open("./data/data.bdb", 0644, nil)
	if err != nil {
		panic(err)
	}

	db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("attrs_group"))     // 组属性
		_, err = tx.CreateBucketIfNotExists([]byte("attrs_user"))       // 用户属性
		_, err = tx.CreateBucketIfNotExists([]byte("attrs_group_user")) // 组_用户_属性
		return err
	})
}

func DBInit() {
	BoltDBInit()
}

func GetDB() *bbolt.DB {
	return db
}
