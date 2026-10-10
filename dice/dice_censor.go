package dice

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"sealdice-core/dice/censor"
	"sealdice-core/dice/service"
	"sealdice-core/utils/dboperator/engine"
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
	// SendEncodedDetails 向当前会话发送 Base64 编码的命中词和上下文片段
	SendEncodedDetails
)

const maxCensorHitContextRunes = 80

var CensorHandlerText = map[CensorHandler]string{
	SendWarning:        "SendWarning",
	SendNotice:         "SendNotice",
	BanUser:            "BanUser",
	BanGroup:           "BanGroup",
	BanInviter:         "BanInviter",
	AddScore:           "AddScore",
	SendEncodedDetails: "SendEncodedDetails",
}

type CensorHandler int

type CensorManager struct {
	IsLoading atomic.Bool
	Parent    *Dice
	Censor    *censor.Censor
	DB        engine.DatabaseOperator

	filesMu   sync.RWMutex
	wordFiles map[string]*censor.WordFile
}

// CensorManager 返回当前审查管理器；未启用或已停止时为 nil（并发安全）。
func (d *Dice) CensorManager() *CensorManager { return d.censorManager.Load() }

// SetCensorManager 原子替换审查管理器；传 nil 表示停止。
func (d *Dice) SetCensorManager(cm *CensorManager) { d.censorManager.Store(cm) }

// WordFiles 返回词库文件快照（并发安全）。
func (cm *CensorManager) WordFiles() map[string]*censor.WordFile {
	cm.filesMu.RLock()
	defer cm.filesMu.RUnlock()
	out := make(map[string]*censor.WordFile, len(cm.wordFiles))
	for k, v := range cm.wordFiles {
		out[k] = v
	}
	return out
}

func (d *Dice) NewCensorManager() {
	cm := CensorManager{
		Censor: &censor.Censor{
			CaseSensitive:  d.Config.CensorCaseSensitive,
			MatchPinyin:    d.Config.CensorMatchPinyin,
			FilterRegexStr: d.Config.CensorFilterRegexStr,
		},
		DB: d.DBOperator,
	}
	cm.Parent = d
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
	// 词表与匹配器就绪后才发布管理器，避免外部读到半初始化状态
	d.SetCensorManager(&cm)
}

// Load 审查加载：在临时 Censor 上读取词库文件，随后原子替换词表，
// 避免重载期间的读写竞争与短暂“词表为空”的窗口。
func (cm *CensorManager) Load(d *Dice) {
	log := d.Logger
	fileDir := "./data/censor"
	cm.IsLoading.Store(true)
	defer cm.IsLoading.Store(false)

	scratch := &censor.Censor{
		CaseSensitive:  cm.Censor.CaseSensitive,
		MatchPinyin:    cm.Censor.MatchPinyin,
		FilterRegexStr: cm.Censor.FilterRegexStr,
	}
	_ = os.MkdirAll(fileDir, 0o755)
	files := make(map[string]*censor.WordFile)
	_ = filepath.Walk(fileDir, func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() && (filepath.Ext(path) == ".txt" || filepath.Ext(path) == ".toml") {
			cm.Parent.Logger.Infof("正在读取敏感词文件：%s\n", path)
			fileInfo, e := scratch.PreloadFile(path)
			if e != nil {
				log.Errorf("censor: unable to read %s, %v", path, e)
			}
			if fileInfo != nil {
				files[fileInfo.Key] = fileInfo
			}
		}
		return nil
	})
	if err := cm.Censor.LoadWords(scratch.SensitiveKeys); err != nil {
		log.Errorf("censor: load fail, %v", err)
	}
	cm.filesMu.Lock()
	cm.wordFiles = files
	cm.filesMu.Unlock()
}

// censorDropRegexes 是匹配前需要剥离的标记（海豹码/CQ码），span 仍映射回原文。
var censorDropRegexes = []*regexp.Regexp{sealCodeRe, cqCodeRe}

func (cm *CensorManager) Check(ctx *MsgContext, msg *Message, text string) (*MsgCheckResult, error) {
	if cm.IsLoading.Load() {
		return nil, errors.New("censor is loading")
	}
	if !cm.Censor.Ready() {
		return nil, errors.New("censor not loaded")
	}
	res := cm.Censor.CheckWithDrops(text, censorDropRegexes)

	spans := make([]censor.Span, 0, len(res.Hits))
	words := res.Words()
	for _, h := range res.Hits {
		spans = append(spans, h.Span)
	}

	if !ctx.Censored && res.HighestLevel > censor.Ignore {
		// 敏感词命中记录保存
		service.CensorAppend(cm.DB, ctx.MessageType, msg.Sender.UserID, msg.GroupID, msg.Message, words, int(res.HighestLevel))
	}
	count := service.CensorCount(cm.DB, msg.Sender.UserID)

	var wordList []string
	for word := range words {
		wordList = append(wordList, word)
	}
	sort.Strings(wordList)
	return &MsgCheckResult{
		UserID:            msg.Sender.UserID,
		Level:             res.HighestLevel,
		HitCounts:         count,
		CurSensitiveWords: wordList,
		Spans:             spans,
	}, nil
}

type MsgCheckResult struct {
	UserID            string
	Level             censor.Level
	HitCounts         map[censor.Level]int
	CurSensitiveWords []string
	Spans             []censor.Span // 命中片段（原文 rune 偏移），用于脱敏
}

func censorHitContext(content string, words []string) string {
	contentRunes := []rune(content)
	if len(contentRunes) <= maxCensorHitContextRunes {
		return content
	}

	contentLower := strings.ToLower(content)
	hitStart := -1
	hitLen := 0
	for _, word := range words {
		if word == "" {
			continue
		}
		byteIndex := strings.Index(contentLower, strings.ToLower(word))
		if byteIndex < 0 {
			continue
		}
		start := len([]rune(contentLower[:byteIndex]))
		if hitStart < 0 || start < hitStart {
			hitStart = start
			hitLen = len([]rune(word))
		}
	}
	if hitStart < 0 {
		return "..."
	}

	start := hitStart
	if hitLen < maxCensorHitContextRunes {
		start -= (maxCensorHitContextRunes - hitLen) / 2
	}
	start = max(0, min(start, len(contentRunes)-maxCensorHitContextRunes))
	end := start + maxCensorHitContextRunes

	context := string(contentRunes[start:end])
	if start > 0 {
		context = "..." + context
	}
	if end < len(contentRunes) {
		context += "..."
	}
	return context
}

func formatCensorHitDetails(levelText string, words []string, content string) string {
	encodedWords := make([]string, 0, len(words))
	for _, word := range words {
		encodedWords = append(encodedWords, base64.StdEncoding.EncodeToString([]byte(word)))
	}
	context := censorHitContext(content, words)

	return fmt.Sprintf(
		"检测到<%s>级敏感词。\n命中词(Base64): %s\n上下文片段(Base64): %s",
		levelText,
		strings.Join(encodedWords, " | "),
		base64.StdEncoding.EncodeToString([]byte(context)),
	)
}

func (d *Dice) CensorMsg(mctx *MsgContext, msg *Message, text string) (hit bool, hitWords []string, needToTerminate bool, newContent string) {
	log := d.Logger
	newContent = text
	cm := d.CensorManager()
	if cm == nil {
		return false, nil, false, newContent
	}
	checkResult, err := cm.Check(mctx, msg, text)
	if err != nil {
		// FIXME: 尽管这种情况比较少，但是是否要提供一个配置项，用来控制默认是跳过还是拦截吗？
		log.Warnf("拦截系统出错(%s)，来自<%s>(%s)的消息跳过了检查", err.Error(), msg.Sender.Nickname, msg.Sender.UserID)
		return false, nil, false, newContent
	}

	if checkResult.Level <= censor.Ignore {
		return false, nil, false, newContent
	}

	hit = true
	hitWords = checkResult.CurSensitiveWords

	// 脱敏：仅当命中带有可用 span 时按 span 打码，否则由调用方回退为整条拦截模板
	if len(checkResult.Spans) > 0 {
		newContent = censorMaskContent(mctx, text, checkResult.Spans)
	}

	if mctx.Censored {
		return hit, hitWords, needToTerminate, newContent
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
			service.CensorClearLevelCount(cm.DB, msg.Sender.UserID, level)
			// 该等级敏感词超过阈值，执行操作
			handler := d.Config.CensorHandlers[level]
			levelText := censor.LevelText[level]
			if handler&(1<<SendWarning) != 0 {
				tmplText := fmt.Sprintf("核心:拦截_警告内容_%s级", censor.LevelText[level])
				ReplyToSenderNoCheck(mctx, msg, DiceFormatTmpl(mctx, tmplText))
			}
			if handler&(1<<SendEncodedDetails) != 0 {
				ReplyToSenderNoCheck(mctx, msg, formatCensorHitDetails(levelText, checkResult.CurSensitiveWords, text))
			}
			if handler&(1<<SendNotice) != 0 {
				// 向通知列表/邮件发送通知
				var text string
				switch msg.MessageType {
				case "group":
					text = fmt.Sprintf(
						"群(%s)内<%s>(%s)触发<%s>敏感词拦截",
						groupInfo.GroupID,
						msg.Sender.Nickname,
						msg.Sender.UserID,
						levelText,
					)
				case "private":
					text = fmt.Sprintf(
						"<%s>(%s)触发<%s>敏感词拦截",
						msg.Sender.Nickname,
						msg.Sender.UserID,
						levelText,
					)
				}
				mctx.Notice(text, NoticeTypeCensor)
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
	return hit, hitWords, needToTerminate, newContent
}

// censorMaskContent 用配置的占位符模板，按 span 对原文打码。
func censorMaskContent(mctx *MsgContext, text string, spans []censor.Span) string {
	placeholder := DiceFormatTmpl(mctx, "核心:拦截_敏感词过滤_替换占位符")
	return censor.MaskSpans(text, spans, placeholder)
}

func (cm *CensorManager) DeleteCensorWordFiles(keys []string) {
	cm.filesMu.Lock()
	defer cm.filesMu.Unlock()
	for _, key := range keys {
		file, ok := cm.wordFiles[key]
		if ok {
			_, err := os.Stat(file.Path)
			if !os.IsNotExist(err) {
				_ = os.RemoveAll(file.Path)
			}
			delete(cm.wordFiles, key)
		}
	}
}
