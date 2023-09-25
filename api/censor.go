package api

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sealdice-core/dice"
	"sealdice-core/dice/censor"
	"sealdice-core/dice/model"
	"sort"
	"strings"

	"github.com/labstack/echo/v4"
)

func check(c echo.Context) (bool, error) {
	if !doAuth(c) {
		return false, c.NoContent(http.StatusForbidden)
	}
	if dm.JustForTest {
		return false, Error(&c, "展示模式不支持该操作", Response{"testMode": true})
	}
	if !myDice.EnableCensor {
		return false, Error(&c, "未启用拦截引擎", Response{})
	}
	if myDice.CensorManager.IsLoading {
		return false, Error(&c, "拦截引擎正在加载，请稍候", Response{})
	}
	return true, nil
}

func censorRestart(c echo.Context) error {
	if !doAuth(c) {
		return c.NoContent(http.StatusForbidden)
	}
	if dm.JustForTest {
		return Error(&c, "展示模式不支持该操作", Response{"testMode": true})
	}

	myDice.NewCensorManager()
	myDice.EnableCensor = true

	return Success(&c, Response{
		"enable":    myDice.EnableCensor,
		"isLoading": myDice.CensorManager.IsLoading,
	})
}

func censorGetStatus(c echo.Context) error {
	return Success(&c, Response{
		"enable":    myDice.EnableCensor,
		"isLoading": myDice.CensorManager.IsLoading,
	})
}

func censorGetConfig(c echo.Context) error {
	levelConfig := map[string]LevelConfig{
		"notice":  getLevelConfig(censor.Notice, myDice.CensorThresholds, myDice.CensorHandlers, myDice.CensorScores),
		"caution": getLevelConfig(censor.Caution, myDice.CensorThresholds, myDice.CensorHandlers, myDice.CensorScores),
		"warning": getLevelConfig(censor.Warning, myDice.CensorThresholds, myDice.CensorHandlers, myDice.CensorScores),
		"danger":  getLevelConfig(censor.Danger, myDice.CensorThresholds, myDice.CensorHandlers, myDice.CensorScores),
	}
	return Success(&c, Response{
		"mode":          myDice.CensorMode,
		"caseSensitive": myDice.CensorCaseSensitive,
		"matchPinyin":   myDice.CensorMatchPinyin,
		"filterRegex":   myDice.CensorFilterRegexStr,
		"levelConfig":   levelConfig,
	})
}

type LevelConfig struct {
	Threshold int      `json:"threshold" mapstructure:"threshold"`
	Handlers  []string `json:"handlers" mapstructure:"handlers"`
	Score     int      `json:"score" mapstructure:"score"`
}

func getLevelConfig(
	level censor.Level,
	thresholds map[censor.Level]int,
	handlers map[censor.Level]uint8,
	scores map[censor.Level]int,
) LevelConfig {
	handler := handlers[level]
	h := make([]string, 0)
	if handler&(1<<dice.SendWarning) != 0 {
		// 发送警告
		h = append(h, dice.CensorHandlerText[dice.SendWarning])
	}
	if handler&(1<<dice.SendNotice) != 0 {
		// 向通知列表/邮件发送通知
		h = append(h, dice.CensorHandlerText[dice.SendNotice])
	}
	if handler&(1<<dice.BanUser) != 0 {
		// 拉黑用户
		h = append(h, dice.CensorHandlerText[dice.BanUser])
	}
	if handler&(1<<dice.BanGroup) != 0 {
		// 拉黑群
		h = append(h, dice.CensorHandlerText[dice.BanGroup])
	}
	if handler&(1<<dice.BanInviter) != 0 {
		// 拉黑邀请人
		h = append(h, dice.CensorHandlerText[dice.BanInviter])
	}
	if handler&(1<<dice.AddScore) != 0 {
		// 仅增加怒气值
		h = append(h, dice.CensorHandlerText[dice.AddScore])
	}
	return LevelConfig{
		Threshold: thresholds[level],
		Handlers:  h,
		Score:     scores[level],
	}
}

func censorSetConfig(c echo.Context) error {
	ok, err := check(c)
	if !ok {
		return err
	}

	jsonMap := make(map[string]interface{})
	err = json.NewDecoder(c.Request().Body).Decode(&jsonMap)
	if err != nil {
		fmt.Println(err)
		return c.JSON(http.StatusInternalServerError, err)
	}

	if val, ok := jsonMap["filterRegex"]; ok {
		filterRegex, ok := val.(string)
		if ok {
			_, err := regexp.Compile(filterRegex)
			if err != nil {
				return Error(&c, "过滤字符正则不是合法的正则表达式", Response{})
			}
			myDice.CensorFilterRegexStr = filterRegex
		}
	}
	if val, ok := jsonMap["mode"]; ok {
		mode, ok := val.(float64)
		if ok {
			myDice.CensorMode = dice.CensorMode(mode)
		}
	}
	if val, ok := jsonMap["caseSensitive"]; ok {
		caseSensitive, ok := val.(bool)
		if ok {
			myDice.CensorCaseSensitive = caseSensitive
		}
	}
	if val, ok := jsonMap["matchPinyin"]; ok {
		matchPinyin, ok := val.(bool)
		if ok {
			myDice.CensorMatchPinyin = matchPinyin
		}
	}
	if val, ok := jsonMap["levelConfig"]; ok {
		levelConfig, ok := val.(map[string]interface{})

		stringConvert := func(val interface{}) []string {
			var lst []string
			for _, i := range val.([]interface{}) {
				t := i.(string)
				if t != "" {
					lst = append(lst, t)
				}
			}
			return lst
		}

		if ok {
			for levelStr, confVal := range levelConfig {
				var level censor.Level
				switch levelStr {
				case "notice":
					level = censor.Notice
				case "caution":
					level = censor.Caution
				case "warning":
					level = censor.Warning
				case "danger":
					level = censor.Danger
				}
				confMap, ok := confVal.(map[string]interface{})
				if ok {
					if val, ok = confMap["threshold"]; ok {
						threshold := val.(float64)
						myDice.CensorThresholds[level] = int(threshold)
					}
					if val, ok = confMap["handlers"]; ok {
						handlers := stringConvert(val)
						setLevelHandlers(level, handlers)
					}
					if val, ok = confMap["score"]; ok {
						score := val.(float64)
						myDice.CensorScores[level] = int(score)
					}
				}
			}
		}
	}
	myDice.MarkModified()
	myDice.Parent.Save()

	return Success(&c, Response{})
}

func setLevelHandlers(level censor.Level, handlers []string) {
	newHandlers := map[dice.CensorHandler]bool{}
	for _, newH := range handlers {
		switch newH {
		case "SendWarning":
			newHandlers[dice.SendWarning] = true
		case "SendNotice":
			newHandlers[dice.SendNotice] = true
		case "BanUser":
			newHandlers[dice.BanUser] = true
		case "BanGroup":
			newHandlers[dice.BanGroup] = true
		case "BanInviter":
			newHandlers[dice.BanInviter] = true
		case "AddScore":
			newHandlers[dice.AddScore] = true
		}
	}

	var handlerVal uint8
	handlerVal = newHandlerVal(handlerVal, dice.SendWarning, newHandlers)
	handlerVal = newHandlerVal(handlerVal, dice.SendNotice, newHandlers)
	handlerVal = newHandlerVal(handlerVal, dice.BanUser, newHandlers)
	handlerVal = newHandlerVal(handlerVal, dice.BanGroup, newHandlers)
	handlerVal = newHandlerVal(handlerVal, dice.BanInviter, newHandlers)
	handlerVal = newHandlerVal(handlerVal, dice.AddScore, newHandlers)

	myDice.CensorHandlers[level] = handlerVal
}

func newHandlerVal(val uint8, handle dice.CensorHandler, newHandlers map[dice.CensorHandler]bool) uint8 {
	if _, ok := newHandlers[handle]; ok {
		val |= 1 << handle
	} else {
		val &^= 1 << handle
	}
	return val
}

type SensitiveRelatedWord struct {
	Word   string `json:"word"`
	Reason int    `json:"reason"`
}

type SensitiveRelatedWords []SensitiveRelatedWord

func (srs SensitiveRelatedWords) Len() int { return len(srs) }
func (srs SensitiveRelatedWords) Less(i, j int) bool {
	if srs[i].Reason == srs[j].Reason {
		return srs[i].Word < srs[j].Word
	} else {
		return srs[i].Reason < srs[j].Reason
	}
}
func (srs SensitiveRelatedWords) Swap(i, j int) { srs[i], srs[j] = srs[j], srs[i] }

type SensitiveWord struct {
	Main    string                `json:"main"`
	Level   censor.Level          `json:"level"`
	Related SensitiveRelatedWords `json:"related"`
}

type SensitiveWords []*SensitiveWord

func (sws SensitiveWords) Len() int { return len(sws) }
func (sws SensitiveWords) Less(i, j int) bool {
	if sws[i].Level == sws[j].Level {
		return sws[i].Main < sws[j].Main
	} else {
		return sws[i].Level < sws[j].Level
	}
}
func (sws SensitiveWords) Swap(i, j int) { sws[i], sws[j] = sws[j], sws[i] }

func censorGetWords(c echo.Context) error {
	ok, err := check(c)
	if !ok {
		return err
	}

	temp := map[string]*SensitiveWord{}
	for word, info := range myDice.CensorManager.Censor.SensitiveKeys {
		switch info.Reason {
		case censor.Origin:
			_, ok := temp[word]
			if !ok {
				temp[word] = &SensitiveWord{
					Main:  word,
					Level: info.Level,
				}
			}
		case censor.IgnoreCase:
			fallthrough
		case censor.PinYin:
			sensitiveWord, ok := temp[info.Origin]
			if !ok {
				temp[info.Origin] = &SensitiveWord{
					Main:  info.Origin,
					Level: info.Level,
				}
				sensitiveWord = temp[info.Origin]
			}
			sensitiveWord.Related = append(sensitiveWord.Related, SensitiveRelatedWord{
				Word:   word,
				Reason: int(info.Reason),
			})
		}
	}

	data := make(SensitiveWords, 0, len(temp))
	for _, word := range temp {
		sort.Sort(word.Related)
		data = append(data, word)
	}
	sort.Sort(data)
	return Success(&c, Response{
		"data": data,
	})
}

func censorGetWordFiles(c echo.Context) error {
	ok, err := check(c)
	if !ok {
		return err
	}

	files := myDice.CensorManager.SensitiveWordsFiles

	type file struct {
		Name  string              `json:"name"`
		Count *censor.FileCounter `json:"count"`
	}
	var res []file
	for path, counter := range files {
		res = append(res, file{
			Name:  filepath.Base(path),
			Count: counter,
		})
	}

	return Success(&c, Response{
		"data": res,
	})
}

func censorUploadWordFiles(c echo.Context) error {
	ok, err := check(c)
	if !ok {
		return err
	}

	file, err := c.FormFile("file")
	if err != nil {
		return err
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer func(src multipart.File) {
		_ = src.Close()
	}(src)

	file.Filename = strings.ReplaceAll(file.Filename, "/", "_")
	file.Filename = strings.ReplaceAll(file.Filename, "\\", "_")
	dst, err := os.Create(filepath.Join("./data/censor", file.Filename))
	if err != nil {
		return err
	}
	defer func(dst *os.File) {
		_ = dst.Close()
	}(dst)

	if _, err = io.Copy(dst, src); err != nil {
		return err
	}

	return Success(&c, Response{})
}

func censorGetLogPage(c echo.Context) error {
	ok, err := check(c)
	if !ok {
		return err
	}

	v := model.QueryCensorLog{}
	err = c.Bind(&v)
	if err != nil {
		fmt.Println(err)
		return c.JSON(http.StatusInternalServerError, err)
	}

	page, err := model.CensorGetLogPage(myDice.CensorManager.DB, &v)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}
	return Success(&c, Response{
		"data": page,
	})
}
