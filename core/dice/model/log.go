package model

import (
	"encoding/json"
	"fmt"
	"go.etcd.io/bbolt"
	"strconv"
	"strings"
)

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
