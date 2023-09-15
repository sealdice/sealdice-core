package dice

import (
	"fmt"
	"io/fs"
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
	SensitiveWordsFiles []string
}

func NewCensorManager(root string, caseSensitive bool, matchPinyin bool, filterRegexStr string) *CensorManager {
	db, err := model.SQLiteCensorDBInit(root)
	if err != nil {
		panic(err)
	}
	cm := CensorManager{
		Censor: &censor.Censor{
			CaseSensitive:  caseSensitive,
			MatchPinyin:    matchPinyin,
			FilterRegexStr: filterRegexStr,
		},
		DB: db,
	}
	cm.Load(filepath.Join(root, "censor"))
	return &cm
}

// Load 审查加载
func (cm *CensorManager) Load(fileDir string) {
	cm.IsLoading = true
	_ = filepath.Walk(fileDir, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return fs.SkipDir
		}
		cm.SensitiveWordsFiles = append(cm.SensitiveWordsFiles, path)
		e := cm.Censor.PreloadFile(path)
		if e != nil {
			fmt.Printf("censor: unable to read %s, %s\n", path, e.Error())
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
