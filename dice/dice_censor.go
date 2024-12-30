package dice

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gorm.io/gorm"

	"sealdice-core/dice/censor"
	"sealdice-core/dice/model"
	log "sealdice-core/utils/kratos"
)

type CensorMode int

const (
	OnlyOutputReply CensorMode = iota
	OnlyInputCommand
	AllInput
)

const (
	// SendWarning 发送警告
	SendWarning CensorHandler = iota
	// SendNotice 向通知列表/邮件发送通知
	SendNotice
	// BanUser 拉黑用户
	BanUser
	// BanGroup 拉黑群
	BanGroup
	// BanInviter 拉黑邀请人
	BanInviter
	// AddScore 增加怒气值
	AddScore
)

var CensorHandlerText = map[CensorHandler]string{
	SendWarning: "SendWarning",
	SendNotice:  "SendNotice",
	BanUser:     "BanUser",
	BanGroup:    "BanGroup",
	BanInviter:  "BanInviter",
	AddScore:    "AddScore",
}

type CensorHandler int

type CensorManager struct {
	IsLoading           bool
	Parent              *Dice
	Censor              *censor.Censor
	DB                  *gorm.DB
	SensitiveWordsFiles map[string]*censor.WordFile
}

func (d *Dice) NewCensorManager() {
	db, err := model.CensorDBInit()
	if err != nil {
		panic(err)
	}
	cm := CensorManager{
		Censor: &censor.Censor{
			CaseSensitive:  d.Config.CensorCaseSensitive,
			MatchPinyin:    d.Config.CensorMatchPinyin,
			FilterRegexStr: d.Config.CensorFilterRegexStr,
		},
		DB: db,
	}
	cm.Parent = d
	d.CensorManager = &cm
	if d.Config.CensorThresholds == nil {
		(&d.Config).CensorThresholds = make(map[censor.Level]int)
	}
	if d.Config.CensorHandlers == nil {
		(&d.Config).CensorHandlers = make(map[censor.Level]uint8)
	}
	if d.Config.CensorScores == nil {
		(&d.Config).CensorScores = make(map[censor.Level]int)
	}
	cm.Load(d)
}

// Load 审查加载
func (cm *CensorManager) Load(_ *Dice) {
	fileDir := "./data/censor"
	cm.IsLoading = true
	cm.Censor.SensitiveKeys = make(map[string]censor.WordInfo)
	_ = os.MkdirAll(fileDir, 0o755)
	_ = filepath.Walk(fileDir, func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() && (filepath.Ext(path) == ".txt" || filepath.Ext(path) == ".toml") {
			cm.Parent.Logger.Infof("正在读取敏感词文件：%s\n", path)
			fileInfo, e := cm.Censor.PreloadFile(path)
			if e != nil {
				log.Errorf("censor: unable to read %s, %v", path, e)
			}
			if cm.SensitiveWordsFiles == nil {
				cm.SensitiveWordsFiles = make(map[string]*censor.WordFile)
			}
			cm.SensitiveWordsFiles[fileInfo.Key] = fileInfo
		}
		return nil
	})
	err := cm.Censor.Load()
	if err != nil {
		log.Errorf("censor: load fail, %v", err)
	}
	cm.IsLoading = false
}

func (cm *CensorManager) Check(ctx *MsgContext, msg *Message, checkContent string) (*MsgCheckResult, error) {
	if cm.IsLoading {
		return nil, errors.New("censor is loading")
	}
	res := cm.Censor.Check(checkContent)
	if !ctx.Censored && res.HighestLevel > censor.Ignore {
		// 敏感词命中记录保存
		model.CensorAppend(cm.DB, ctx.MessageType, msg.Sender.UserID, msg.GroupID, msg.Message, res.SensitiveWords, int(res.HighestLevel))
	}
	count := model.CensorCount(cm.DB, msg.Sender.UserID)

	var words []string
	for word := range res.SensitiveWords {
		words = append(words, word)
	}
	return &MsgCheckResult{
		UserID:            msg.Sender.UserID,
		Level:             res.HighestLevel,
		HitCounts:         count,
		CurSensitiveWords: words,
	}, nil
}

type MsgCheckResult struct {
	UserID            string
	Level             censor.Level
	HitCounts         map[censor.Level]int
	CurSensitiveWords []string
}

func (d *Dice) CensorMsg(mctx *MsgContext, msg *Message, checkContent string, sendContent string) (hit bool, hitWords []string, needToTerminate bool, newContent string) {
	log := d.Logger
	checkResult, err := d.CensorManager.Check(mctx, msg, checkContent)
	if err != nil {
		// FIXME: 尽管这种情况比较少，但是是否要提供一个配置项，用来控制默认是跳过还是拦截吗？
		log.Warnf("拦截系统出错(%s)，来自<%s>(%s)的消息跳过了检查", err.Error(), msg.Sender.Nickname, msg.Sender.UserID)
		return
	}
	newContent = sendContent

	if checkResult.Level <= censor.Ignore {
		return
	}

	hit = true
	hitWords = checkResult.CurSensitiveWords
	// TODO: 替换掉敏感词（先暂时不提供）
	// placeholder := DiceFormatTmpl(mctx, "核心:拦截_替换内容")
	// for _, word := range checkResult.CurSensitiveWords {
	// 	newContent = strings.ReplaceAll(newContent, word, placeholder)
	// }

	if mctx.Censored {
		return
	}

	mctx.Censored = true
	groupInfo, ok := mctx.Session.ServiceAtNew.Load(msg.GroupID)
	if !ok {
		d.Logger.Warn("Dice CenSor获取GroupInfo失败")
	}
	thresholds := d.Config.CensorThresholds

	// 保证按程度依次降低来处理
	var tempLevels censor.Levels
	for level := range checkResult.HitCounts {
		tempLevels = append(tempLevels, level)
	}
	sort.Sort(sort.Reverse(tempLevels))

	for _, level := range tempLevels {
		hitCount := checkResult.HitCounts[level]
		if hitCount > thresholds[level] {
			// 处理完跳出，多个等级超过阈值的处理仅进行最高的处理
			// 需要终止后续动作
			needToTerminate = true
			// 清空此用户该等级计数
			model.CensorClearLevelCount(d.CensorManager.DB, msg.Sender.UserID, level)
			// 该等级敏感词超过阈值，执行操作
			handler := d.Config.CensorHandlers[level]
			levelText := censor.LevelText[level]
			if handler&(1<<SendWarning) != 0 {
				tmplText := fmt.Sprintf("核心:拦截_警告内容_%s级", censor.LevelText[level])
				ReplyToSenderNoCheck(mctx, msg, DiceFormatTmpl(mctx, tmplText))
			}
			if handler&(1<<SendNotice) != 0 {
				// 向通知列表/邮件发送通知
				var text string
				if msg.MessageType == "group" {
					text = fmt.Sprintf(
						"群(%s)内<%s>(%s)触发<%s>敏感词拦截",
						groupInfo.GroupID,
						msg.Sender.Nickname,
						msg.Sender.UserID,
						levelText,
					)
				} else if msg.MessageType == "private" {
					text = fmt.Sprintf(
						"<%s>(%s)触发<%s>敏感词拦截",
						msg.Sender.Nickname,
						msg.Sender.UserID,
						levelText,
					)
				}
				mctx.Notice(text)
			}
			if handler&(1<<BanUser) != 0 {
				// 拉黑用户
				(&d.Config).BanList.AddScoreBase(
					msg.Sender.UserID,
					d.Config.BanList.ThresholdBan,
					"敏感词审查",
					"触发<"+levelText+">敏感词",
					mctx,
				)
			}
			if handler&(1<<BanGroup) != 0 {
				// 拉黑群
				if msg.MessageType == "group" {
					(&d.Config).BanList.AddScoreBase(
						msg.GroupID,
						d.Config.BanList.ThresholdBan,
						"敏感词审查",
						"触发<"+levelText+">敏感词",
						mctx,
					)
				}
			}
			if handler&(1<<BanInviter) != 0 {
				// 拉黑邀请人
				if msg.MessageType == "group" {
					(&d.Config).BanList.AddScoreBase(
						groupInfo.InviteUserID,
						d.Config.BanList.ThresholdBan,
						"敏感词审查",
						"触发<"+levelText+">敏感词",
						mctx,
					)
				}
			}
			if handler&(1<<AddScore) != 0 {
				score, ok := d.Config.CensorScores[level]
				if !ok {
					score = 100
				}
				// 仅增加怒气值
				if msg.MessageType == "group" {
					(&d.Config).BanList.AddScoreByCensor(
						msg.Sender.UserID,
						int64(score),
						groupInfo.GroupID,
						levelText,
						mctx,
					)
				}
			}
			// 只处理一次
			d.Logger.Infof(
				"<%s>(%s)发送的「%s」触发最高<%s>级敏感词（%s），触发次数已经超过阈值，进行处理",
				msg.Sender.Nickname,
				msg.Sender.UserID,
				msg.Message,
				censor.LevelText[level],
				strings.Join(checkResult.CurSensitiveWords, "|"),
			)
			break
		}
	}
	return
}

func (cm *CensorManager) DeleteCensorWordFiles(keys []string) {
	for _, key := range keys {
		file, ok := cm.SensitiveWordsFiles[key]
		if ok {
			_, err := os.Stat(file.Path)
			if !os.IsNotExist(err) {
				_ = os.RemoveAll(file.Path)
			}
			delete(cm.SensitiveWordsFiles, key)
		}
	}
}
