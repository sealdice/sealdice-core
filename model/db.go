package model

import (
	"fmt"
	"go.etcd.io/bbolt"
	"os"
	"sealdice-core/core"
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
		_, err := tx.CreateBucketIfNotExists([]byte("attrs_group")) // 组属性
		_, err = tx.CreateBucketIfNotExists([]byte("attrs_user")) // 用户属性
		_, err = tx.CreateBucketIfNotExists([]byte("attrs_group_user")) // 组_用户_属性
		return err
	})
}

func attrGetAllBase(bucket []byte, key []byte) []byte {
	var data []byte
	db.View(func(tx *bbolt.Tx) error {
		// Retrieve the users bucket.
		// This should be created when the DB is first opened.
		b0 := tx.Bucket(bucket)
		if b0 == nil {
			return nil
		}
		data = b0.Get(key)
		return nil
	})
	return data
}

func attrSave(bucket []byte, key []byte, data []byte) {
	db.Update(func(tx *bbolt.Tx) error {
		// Retrieve the users bucket.
		// This should be created when the DB is first opened.
		b0 := tx.Bucket(bucket)
		if b0 == nil {
			return nil
		}
		err := b0.Put(key, data)
		if err != nil {
			core.GetLogger().Error(err)
		}
		return err
	})
}

func AttrGroupUserGetAll(groupId int64, userId int64) []byte {
	return attrGetAllBase([]byte("attrs_group_user"), []byte(fmt.Sprintf("%d-%d", groupId, userId)))
}

func AttrGroupUserSave(groupId int64, userId int64, data []byte) {
	attrSave([]byte("attrs_group_user"), []byte(fmt.Sprintf("%d-%d", groupId, userId)), data)
}

func AttrGroupGetAll(groupId int64) []byte {
	return attrGetAllBase([]byte("attrs_group"), []byte(fmt.Sprintf("%d", groupId)))
}

func AttrGroupSave(groupId int64, data []byte) {
	attrSave([]byte("attrs_group"), []byte(fmt.Sprintf("%d", groupId)), data)
}

func AttrUserGetAll(userId int64) []byte {
	return attrGetAllBase([]byte("attrs_user"), []byte(fmt.Sprintf("%d", userId)))
}

func AttrUserSave(userId int64, data []byte) {
	attrSave([]byte("attrs_user"), []byte(fmt.Sprintf("%d", userId)), data)
}

func DBInit() {
	BoltDBInit()
}

func GetDB() *bbolt.DB {
	return db
}
