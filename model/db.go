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
}

func DBInit() {
	BoltDBInit()
}

func GetDB() *bbolt.DB {
	return db
}
