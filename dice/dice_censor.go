package dice

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sealdice-core/dice/censor"
	"sealdice-core/dice/model"

	"github.com/jmoiron/sqlx"
)

type CensorManager struct {
	IsLoading           bool
	Parent              *Dice
	Censor              *censor.Censor
	DB                  *sqlx.DB
	SensitiveWordsFiles map[string]*censor.FileCounter
}

func (d *Dice) NewCensorManager() {
	db, err := model.SQLiteCensorDBInit(d.BaseConfig.DataDir)
	if err != nil {
		panic(err)
	}
	cm := CensorManager{
		Censor: &censor.Censor{
			CaseSensitive:  d.CensorCaseSensitive,
			MatchPinyin:    d.CensorMatchPinyin,
			FilterRegexStr: d.CensorFilterRegexStr,
		},
		DB: db,
	}
	cm.Parent = d
	d.CensorManager = &cm
	cm.Load(d)
}

// Load 审查加载
func (cm *CensorManager) Load(d *Dice) {
	fileDir := "./data/censor"
	cm.IsLoading = true
	_ = os.MkdirAll(fileDir, 0755)
	_ = filepath.Walk(fileDir, func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() && (filepath.Ext(path) == ".txt" || filepath.Ext(path) == ".toml") {
			cm.Parent.Logger.Infof("正在读取敏感词文件：%s\n", path)
			counter, e := cm.Censor.PreloadFile(path)
			if e != nil {
				fmt.Printf("censor: unable to read %s, %s\n", path, e.Error())
			}
			if cm.SensitiveWordsFiles == nil {
				cm.SensitiveWordsFiles = make(map[string]*censor.FileCounter)
			}
			cm.SensitiveWordsFiles[path] = counter
		}
		return nil
	})
	err := cm.Censor.Load()
	if err != nil {
		fmt.Printf("censor: load fail, %s\n", err.Error())
	}
	cm.IsLoading = false
}

func (cm *CensorManager) Check(ctx *MsgContext, msg *Message) (*MsgCheckResult, error) {
	if cm.IsLoading {
		return nil, fmt.Errorf("censor is loading")
	}
	res := cm.Censor.Check(msg.Message)
	if res.HighestLevel > censor.Ignore {
		// 敏感词命中记录保存
		model.CensorAppend(cm.DB, ctx.MessageType, msg.Sender.UserId, msg.GroupId, msg.Message, res.SensitiveWords, int(res.HighestLevel))
	}
	return &MsgCheckResult{
		UserId:    msg.Sender.UserId,
		Level:     res.HighestLevel,
		HitCounts: nil,
	}, nil
}

type MsgCheckResult struct {
	UserId            string
	Level             censor.Level
	HitCounts         map[censor.Level]int
	CurSensitiveWords []string
}
