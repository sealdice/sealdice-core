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
	if d.CensorThresholds == nil {
		d.CensorThresholds = make(map[censor.Level]int)
	}
	if d.CensorHandlers == nil {
		d.CensorHandlers = make(map[censor.Level]uint8)
	}
	if d.CensorScores == nil {
		d.CensorScores = make(map[censor.Level]int)
	}
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
	if !ctx.Censored && res.HighestLevel > censor.Ignore {
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

func (d *Dice) CensorMsg(mctx *MsgContext, msg *Message, sendContent string) string {
	var newContent string
	// TODO: 替换掉敏感词
	checkResult, err := d.CensorManager.Check(mctx, msg)
	newContent = sendContent

	if !mctx.Censored {
		mctx.Censored = true
		group := mctx.Session.ServiceAtNew[msg.GroupId]
		log := d.Logger
		if err != nil {
			// FIXME: 尽管这种情况比较少，但是是否要提供一个配置项，用来控制默认是跳过还是拦截吗？
			log.Warnf("审查系统出错(%s)，来自<%s>(%s)的消息跳过了检查", err.Error(), msg.Sender.Nickname, msg.Sender.UserId)
		}
		thresholds := d.CensorThresholds
		for level, hitCount := range checkResult.HitCounts {
			if hitCount > thresholds[level] {
				// 该等级敏感词超过阈值，执行操作
				handler := d.CensorHandlers[level]
				levelText := censor.LevelText[level]
				if (handler << SendWarning) != 0 {
					// FIXME: 发送警告
					ReplyToSender(mctx, msg, "")
				}
				if handler&(1<<SendNotice) != 0 {
					// 向通知列表/邮件发送通知
					var text string
					if msg.MessageType == "group" {
						text = fmt.Sprintf(
							"群(%s)内<%s>(%s)的消息「%s」触发<%s>敏感词：",
							group.GroupId,
							msg.Sender.Nickname,
							msg.Sender.UserId,
							msg.Message,
							levelText,
						)
					} else if msg.MessageType == "private" {
						text = fmt.Sprintf(
							"<%s>(%s)的私聊消息「%s」触发<%s>敏感词：",
							msg.Sender.Nickname,
							msg.Sender.UserId,
							msg.Message,
							levelText,
						)
					}
					mctx.Notice(text)
				}
				if handler&(1<<BanUser) != 0 {
					// 拉黑用户
					d.BanList.AddScoreBase(
						msg.Sender.UserId,
						d.BanList.ThresholdBan,
						"敏感词审查",
						"触发<"+levelText+">敏感词",
						mctx,
					)
				}
				if handler&(1<<BanGroup) != 0 {
					// 拉黑群
					if msg.MessageType == "group" {
						d.BanList.AddScoreBase(
							msg.GroupId,
							d.BanList.ThresholdBan,
							"敏感词审查",
							"触发<"+levelText+">敏感词",
							mctx,
						)
					}
				}
				if handler&(1<<BanInviter) != 0 {
					// 拉黑邀请人
					if msg.MessageType == "group" {
						d.BanList.AddScoreBase(
							group.InviteUserId,
							d.BanList.ThresholdBan,
							"敏感词审查",
							"触发<"+levelText+">敏感词",
							mctx,
						)
					}
				}
				if handler&(1<<AddScore) != 0 {
					score, ok := d.CensorScores[level]
					if !ok {
						score = 100
					}
					// 仅增加怒气值
					if msg.MessageType == "group" {
						d.BanList.AddScoreByCensor(
							msg.Sender.UserId,
							int64(score),
							group.GroupId,
							levelText,
							mctx,
						)
					}
				}
			}
		}
	}
	return newContent
}
