package model

import (
	"encoding/binary"
	"encoding/json"
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
	db.Update(func(tx *bbolt.Tx) error {
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

func AttrUserGetAllLegacy(db *bbolt.DB, userId int64) []byte {
	return attrGetAllBase(db, []byte("attrs_user"), []byte(fmt.Sprintf("%d", userId)))
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

type LogOneItemLegacy struct {
	Id        uint64 `json:"id"`
	Nickname  string `json:"nickname"`
	IMUserId  int64  `json:"IMUserId"`
	Time      int64  `json:"time"`
	Message   string `json:"message"`
	IsDice    bool   `json:"isDice"`
	CommandId uint64 `json:"commandId"`

	OldNickname string `json:"Nickname"`
}
type LogOneItem struct {
	Id        uint64 `json:"id"`
	Nickname  string `json:"nickname"`
	IMUserId  string `json:"IMUserId"`
	Time      int64  `json:"time"`
	Message   string `json:"message"`
	IsDice    bool   `json:"isDice"`
	CommandId uint64 `json:"commandId"`

	UniformId string `json:"uniformId"`
	Channel   string `json:"channel"` // 用于秘密团
}

func logUpdate(db *bbolt.DB) {
	bucket := "logs"

	db.Update(func(tx *bbolt.Tx) error {
		// Retrieve the users bucket.
		// This should be created when the DB is first opened.
		b0 := tx.Bucket([]byte(bucket))
		if b0 == nil {
			return nil
		}

		oldKeys := [][]byte{}
		b0.ForEach(func(k []byte, v []byte) error {
			if !strings.HasPrefix(string(k), "QQ-Group:") {
				//b0.Put([]byte("QQ-Group:"+string(k)), v)
				b1 := b0.Bucket(k)
				if b1 != nil {
					fmt.Println("转换数据格式: 跑团记录: ", string(k))
					oldKeys = append(oldKeys, k)
				}
			}
			return nil
		})

		for _, i := range oldKeys {
			newBucketName := "QQ-Group:" + string(i)
			b1 := b0.Bucket(i)
			b2, err := b0.CreateBucketIfNotExists([]byte(newBucketName))
			if err == nil {
				b1.ForEach(func(k []byte, v []byte) error {
					b1New := b1.Bucket(k)

					if b1New != nil {
						b2New, err := b2.CreateBucketIfNotExists(k)
						if err == nil {
							b1New.ForEach(func(k1 []byte, v1 []byte) error {
								id, _ := b2New.NextSequence()
								var item LogOneItemLegacy
								err := json.Unmarshal(v1, &item)
								if err == nil {
									nickname := item.OldNickname
									if nickname == "" {
										nickname = item.Nickname
									}
									newItem := LogOneItem{
										Id:        id,
										Nickname:  nickname,
										IMUserId:  strconv.FormatInt(item.IMUserId, 10),
										Time:      item.Time,
										Message:   item.Message,
										IsDice:    item.IsDice,
										CommandId: item.CommandId,
										UniformId: FormatDiceIdQQ(item.IMUserId),
									}
									nBytes, err := json.Marshal(newItem)
									if err == nil {
										b2New.Put(itob(id), nBytes)
									} else {
										fmt.Println("错误条目: ", string(v1))
									}
								}
								return nil
							})
						}
					}
					return nil
				})
			}
			b0.DeleteBucket(i)
		}

		return nil
	})
}

// itob returns an 8-byte big endian representation of v.
func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}
