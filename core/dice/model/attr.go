package model

import (
	"encoding/binary"
	"fmt"
	"go.etcd.io/bbolt"
	"strconv"
	"strings"
)

func attrGetAllBase(db *bbolt.DB, bucket []byte, key []byte) []byte {
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

func attrSave(db *bbolt.DB, bucket []byte, key []byte, data []byte) {
	_ = db.Update(func(tx *bbolt.Tx) error {
		// Retrieve the users bucket.
		// This should be created when the DB is first opened.
		b0 := tx.Bucket(bucket)
		if b0 == nil {
			return nil
		}
		err := b0.Put(key, data)
		if err != nil {
			fmt.Println(err)
		}
		return err
	})
}

func AttrGroupUserGetAll(db *bbolt.DB, groupId string, userId string) []byte {
	return attrGetAllBase(db, []byte("attrs_group_user"), []byte(fmt.Sprintf("%s-%s", groupId, userId)))
}

func AttrGroupUserSave(db *bbolt.DB, groupId string, userId string, data []byte) {
	attrSave(db, []byte("attrs_group_user"), []byte(fmt.Sprintf("%s-%s", groupId, userId)), data)
}

func AttrGroupGetAll(db *bbolt.DB, groupId string) []byte {
	return attrGetAllBase(db, []byte("attrs_group"), []byte(groupId))
}

func AttrGroupSave(db *bbolt.DB, groupId string, data []byte) {
	attrSave(db, []byte("attrs_group"), []byte(groupId), data)
}

func AttrUserGetAll(db *bbolt.DB, userId string) []byte {
	return attrGetAllBase(db, []byte("attrs_user"), []byte(userId))
}

func AttrUserSave(db *bbolt.DB, userId string, data []byte) {
	attrSave(db, []byte("attrs_user"), []byte(userId), data)
}

func AttrTryUpdate(db *bbolt.DB) {
	bucket := "attrs_group_user"

	db.Update(func(tx *bbolt.Tx) error {
		// Retrieve the users bucket.
		// This should be created when the DB is first opened.
		b0 := tx.Bucket([]byte(bucket))
		if b0 == nil {
			return nil
		}

		oldKeys := [][]byte{}
		b0.ForEach(func(k []byte, v []byte) error {
			if !strings.HasPrefix(string(k), "QQ:") && !strings.HasPrefix(string(k), "QQ-Group:") {
				oldKey := string(k)
				if string(k) == "0" {
					return nil
				}
				if string(v) != "{}" {
					keys := strings.Split(oldKey, "-")
					fmt.Println("转换数据格式: 群内个人数据 ", oldKey)

					err := b0.Put([]byte("QQ-Group:"+keys[0]+"-"+"QQ:"+keys[1]), v)
					if err != nil {
						fmt.Println("转换失败", err)
					}
				}
				oldKeys = append(oldKeys, k)
			}
			return nil
		})

		for _, i := range oldKeys {
			b0.Delete(i)
		}

		return nil
	})

	personAttrUpdate(db)
	groupAttrUpdate(db)
	logUpdate(db)
}

func personAttrUpdate(db *bbolt.DB) {
	bucket := "attrs_user"

	db.Update(func(tx *bbolt.Tx) error {
		// Retrieve the users bucket.
		// This should be created when the DB is first opened.
		b0 := tx.Bucket([]byte(bucket))
		if b0 == nil {
			return nil
		}

		oldKeys := [][]byte{}
		b0.ForEach(func(k []byte, v []byte) error {
			if !strings.HasPrefix(string(k), "QQ:") && !strings.HasPrefix(string(k), "PG-QQ:") {
				if string(k) == "0" {
					return nil
				}
				fmt.Println("转换数据格式: 个人变量 ", string(k))

				err := b0.Put([]byte("QQ:"+string(k)), v)
				if err != nil {
					fmt.Println("转换失败", err)
				}
				oldKeys = append(oldKeys, k)
			}
			return nil
		})

		for _, i := range oldKeys {
			b0.Delete(i)
		}

		return nil
	})
}

func groupAttrUpdate(db *bbolt.DB) {
	bucket := "attrs_group"

	db.Update(func(tx *bbolt.Tx) error {
		// Retrieve the users bucket.
		// This should be created when the DB is first opened.
		b0 := tx.Bucket([]byte(bucket))
		if b0 == nil {
			return nil
		}

		oldKeys := [][]byte{}
		b0.ForEach(func(k []byte, v []byte) error {
			if !strings.HasPrefix(string(k), "QQ-Group:") && !strings.HasPrefix(string(k), "PG-QQ:") {
				if string(k) == "0" {
					return nil
				}

				if string(v) != "{}" {
					fmt.Println("转换数据格式: 群变量 ", string(k), string(v))
					err := b0.Put([]byte("QQ-Group:"+string(k)), v)
					if err != nil {
						fmt.Println("转换失败: ", err)
					}
				}
				oldKeys = append(oldKeys, k)
			}
			return nil
		})

		for _, i := range oldKeys {
			b0.Delete(i)
		}

		return nil
	})
}

func FormatDiceIdQQ(diceQQ int64) string {
	return fmt.Sprintf("QQ:%s", strconv.FormatInt(diceQQ, 10))
}

// itob returns an 8-byte big endian representation of v.
func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}
