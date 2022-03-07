package model

import (
	"go.etcd.io/bbolt"
)

func BoltDBInit(path string) *bbolt.DB {
	db, err := bbolt.Open(path, 0644, nil)
	if err != nil {
		panic(err)
	}

	db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("attrs_group"))     // 组属性
		_, err = tx.CreateBucketIfNotExists([]byte("attrs_user"))       // 用户属性
		_, err = tx.CreateBucketIfNotExists([]byte("attrs_group_user")) // 组_用户_属性
		return err
	})

	return db
}
