package main

import (
	"archive/zip"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"go.etcd.io/bbolt"
	"io/ioutil"
	"os"
	"sealdice-core/model"
	"sort"
	"strconv"
	"time"
)

type LogOneItem struct {
	Id uint64 `json:"id"`
	Nickname string `jsno:"nickname"`
	IMUserId int64 `json:"IMUserId"`
	Time int64 `json:"time"`
	Message string `json:"message"`
	IsDice bool `json:"isDice"`
}

// {"data":null,"msg":"SEND_MSG_API_ERROR","retcode":100,"status":"failed","wording":"请参考 go-cqhttp 端输出"}

func (self *Dice) registerBuiltinExtLog() {
	self.extList = append(self.extList, &ExtInfo{
		Name:       "log",
		version:    "0.0.1",
		Brief: "跑团辅助扩展，提供日志、染色等功能",
		Author: "木落",
		autoActive: true,
		OnLoad: func() {
			os.MkdirAll("./data/logs", 0644)
			model.GetDB().Update(func(tx *bbolt.Tx) error {
				_, err := tx.CreateBucketIfNotExists([]byte("logs"))
				return err
			})
		},
		OnMessageSend: func(ctx *MsgContext, messageType string, userId int64, text string, flag string) {
			// 记录骰子发言
			if flag == "skip" {
				return
			}
			if isCurGroupBotOnById(ctx.session, messageType, userId) {
				session := ctx.session
				group := session.ServiceAt[userId]
				if group.LogOn {
					// <2022-02-15 09:54:14.0> [摸鱼king]: 有的 但我不知道
					a := LogOneItem{
						Nickname: session.Nickname,
						IMUserId: session.UserId,
						Time: time.Now().Unix(),
						Message: text,
						IsDice: true,
					}
					LogAppend(group, &a)
				}
			}
		},
		OnMessageReceived: func(ctx *MsgContext, msg *Message) {
			// 处理日志
			if ctx.isCurGroupBotOn {
				if ctx.group.LogOn {
					// <2022-02-15 09:54:14.0> [摸鱼king]: 有的 但我不知道
					a := LogOneItem{
						Nickname: ctx.player.Name,
						IMUserId: ctx.player.UserId,
						Time: msg.Time,
						Message: msg.Message,
						IsDice: false,
					}

					LogAppend(ctx.group, &a)
				}
			}
		},
		GetDescText: func (ei *ExtInfo) string {
			text := "> " + ei.Brief + "\n" + "提供命令:\n"
			keys := make([]string, 0, len(ei.cmdMap))
			for k := range ei.cmdMap {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, i := range keys {
				i := ei.cmdMap[i]
				brief := i.Brief
				if brief != "" {
					brief = " // " + brief
				}
				text += i.name + brief + "\n"
			}

			return text
		},
		cmdMap: CmdMapCls{
			"log": &CmdItemInfo{
				name: ".log",
				solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) struct{ success bool } {
					if ctx.isCurGroupBotOn {
						group := ctx.group

						if len(cmdArgs.Args) == 0 {
							onText := "关闭"
							if group.LogOn {
								onText = "开启"
							}
							text := fmt.Sprintf("记录，当前状态: %s\n已记录文本%d条", onText, LogLinesGet(group))
							replyToSender(ctx, msg, text)
						} else {
							if cmdArgs.isArgEqual(1, "on") {
								group.LogOn = true
								text := fmt.Sprintf("记录已经继续开启，当前已记录文本%d条", LogLinesGet(group))
								replyToSender(ctx, msg, text)
							} else if cmdArgs.isArgEqual(1, "off") {
								group.LogOn = false
								text := fmt.Sprintf("记录已经暂时关闭，当前已记录文本%d条", LogLinesGet(group))
								replyToSender(ctx, msg, text)
							} else if cmdArgs.isArgEqual(1, "save") {
								fn := LogSaveToZip(group)
								replyToSenderRaw(ctx, msg, fmt.Sprintf("已经生成跑团日志，链接如下：\n%s\n着色服务正在开发中，目前请使用公开的着色网站进行着色。", fn), "skip")
							} else if cmdArgs.isArgEqual(1, "end") {
								replyToSender(ctx, msg, "故事落下了帷幕。\n记录已经关闭。")
								group.LogOn = false

								time.Sleep(time.Duration(0.5 * float64(time.Second)))
								fn := LogSaveToZip(group)
								replyToSenderRaw(ctx, msg, fmt.Sprintf("已经生成跑团日志，链接如下：\n%s\n着色服务正在开发中，目前请使用公开的着色网站进行着色。", fn), "skip")
								group.LogCurName = ""
							} else if cmdArgs.isArgEqual(1, "new") {
								if group.LogCurName != "" {
									replyToSender(ctx, msg, "上一段旅程还未结束，请先使用.log end结束故事")
								} else {
									todayTime := time.Now().Format("2006_01_02_15_04_05")
									group.LogCurName = todayTime
									group.LogOn = true
									replyToSender(ctx, msg, "新的故事开始了，祝旅途愉快！\n记录已经开启。")
									//replyToSender(ctx, msg, "log new")
									//fmt.Println("新的故事开始了，祝旅途愉快！\n记录已经开启。")
									//fmt.Println("!!!", err)
									//err := b.Put([]byte("answer"), []byte("42"))
									//replyToSender(ctx, msg, "似乎出了一点问题，与数据库的连接失败了")
								}
							}
						}
					}
					return struct{ success bool }{
						success: true,
					}
				},
			},
		},
	})
}

func LogSaveToZip(group *ServiceAtItem) string {
	lines, err := LogGetAllLines(group)
	if err == nil {
		os.MkdirAll("./data/logs", 0644)
		fzip, _ := ioutil.TempFile("./data/logs", group.LogCurName + ".*.zip")
		writer := zip.NewWriter(fzip)
		defer writer.Close()

		text := ""
		for _, i := range lines {
			timeTxt := time.Now().Format("2006-01-02 15:04:05")
			if i.IsDice {
				text += fmt.Sprintf("[%s] %s(骰子): %s\n", timeTxt, i.Nickname, i.Message)
			} else {
				text += fmt.Sprintf("[%s] <%s>: %s\n", timeTxt, i.Nickname, i.Message)
			}
		}

		//f, _ := ioutil.TempFile("./temp", "log.*.txt")
		//f.WriteString(text)
		//defer f.Close()

		fileWriter, _ := writer.Create("跑团日志(标准格式).txt")
		fileWriter.Write([]byte(text))

		// 第二份，QQ格式
		text = ""
		for _, i := range lines {
			timeTxt := time.Now().Format("2006-01-02 15:04:05")
			text += fmt.Sprintf("%s(%d) %s\n%s\n\n", i.Nickname, i.IMUserId, timeTxt, i.Message)
		}

		fileWriter, _ = writer.Create("跑团日志(类QQ格式).txt")
		fileWriter.Write([]byte(text))
		writer.Close()

		// 回到开头上传
		fzip.Seek(0, 0)
		fn := UploadFileToTransferSh(group.LogCurName + ".zip", fzip)
		//fn := UploadFileToFileIo(group.LogCurName + ".zip", fzip)

		return fn
	}
	return ""
}

func LogLinesGet(group *ServiceAtItem) int {
	var size int
	model.GetDB().View(func(tx *bbolt.Tx) error {
		// Retrieve the users bucket.
		// This should be created when the DB is first opened.
		b0 := tx.Bucket([]byte("logs"))
		if b0 == nil {
			return nil
		}
		b1 := b0.Bucket([]byte(strconv.FormatInt(group.GroupId, 10)))
		if b1 == nil {
			return nil
		}
		b := b1.Bucket([]byte(group.LogCurName))
		if b == nil {
			return nil
		}
		size = b.Stats().KeyN
		return nil
	})
	return size
}

func LogGetAllLines(group *ServiceAtItem) ([]*LogOneItem, error) {
	ret := []*LogOneItem{}
	return ret, model.GetDB().View(func(tx *bbolt.Tx) error {
		b0 := tx.Bucket([]byte("logs"))
		if b0 == nil {
			return nil
		}
		b1 := b0.Bucket([]byte(strconv.FormatInt(group.GroupId, 10)))
		if b1 == nil {
			return nil
		}

		b := b1.Bucket([]byte(group.LogCurName))
		if b == nil {
			return nil
		}

		return b.ForEach(func(k, v []byte) error {
			logItem := LogOneItem{}
			err := json.Unmarshal(v, &logItem)
			if err == nil {
				ret = append(ret, &logItem)
			}

			return nil
		})
	})
}

func LogAppend(group *ServiceAtItem, l *LogOneItem) error {
	return model.GetDB().Update(func(tx *bbolt.Tx) error {
		// Retrieve the users bucket.
		// This should be created when the DB is first opened.
		b0 := tx.Bucket([]byte("logs"))
		b1, err := b0.CreateBucketIfNotExists([]byte(strconv.FormatInt(group.GroupId, 10)))
		if err != nil {
			return err
		}

		b, err := b1.CreateBucketIfNotExists([]byte(group.LogCurName))
		if err == nil {
			l.Id, _ = b.NextSequence()
			buf, err := json.Marshal(l)
			if err != nil {
				return err
			}

			return b.Put(itob(l.Id), buf)
		}
		return err
	})
}

// itob returns an 8-byte big endian representation of v.
func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}
