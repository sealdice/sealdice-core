package model

import (
	"go.etcd.io/bbolt"
)

func BanMapGet(db *bbolt.DB) []byte {
	var data []byte
	db.View(func(tx *bbolt.Tx) error {
		// Retrieve the users bucket.
		// This should be created when the DB is first opened.
		b0 := tx.Bucket([]byte("common"))
		if b0 == nil {
			return nil
		}
		data = b0.Get([]byte("banMap"))
		return nil
	})
	return data
}

func BanMapSet(db *bbolt.DB, data []byte) {
	db.Update(func(tx *bbolt.Tx) error {
		// Retrieve the users bucket.
		// This should be created when the DB is first opened.
		b0, err := tx.CreateBucketIfNotExists([]byte("common"))
		if err != nil {
			return err
		}
		return b0.Put([]byte("banMap"), data)
	})
}
