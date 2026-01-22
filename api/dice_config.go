package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"

	"sealdice-core/dice"
	"sealdice-core/utils"
)

type DiceConfigInfo struct {
	dice.Config

	CommandPrefix       []string `form:"commandPrefix"       json:"commandPrefix"`
	DiceMasters         []string `form:"diceMasters"         json:"diceMasters"`
	UIPassword          string   `form:"uiPassword"          json:"uiPassword"`
	LogPageItemLimit    int64    `json:"logPageItemLimit"`
	DefaultCocRuleIndex string   `json:"defaultCocRuleIndex"` // 默认coc index
	MaxExecuteTime      string   `json:"maxExecuteTime"`      // 最大骰点次数
	MaxCocCardGen       string   `json:"maxCocCardGen"`       // 最大coc制卡数
	ServerAddress       string   `form:"serveAddress"        json:"serveAddress"`
	HelpDocEngineType   int      `json:"helpDocEngineType"`
}

func DiceConfig(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	password := ""
	if myDice.Parent.UIPasswordHash != "" {
		password = "------"
	}

	limit := myDice.Config.UILogLimit
	if limit == 0 {
		limit = 100
	}
	myDice.UnlockCodeUpdate(false)

	cocRule := strconv.FormatInt(myDice.Config.DefaultCocRuleIndex, 10)
	if myDice.Config.DefaultCocRuleIndex == 11 {
		cocRule = "dg"
	}

	maxExec := strconv.FormatInt(myDice.Config.MaxExecuteTime, 10)

	maxCard := strconv.FormatInt(myDice.Config.MaxCocCardGen, 10)

	emailPasswordMasked := ""
	if myDice.Config.MailPassword != "" {
		emailPasswordMasked = "******"
	}

	// 过滤掉未加载的: 包括关闭的和已经删除的
	// TODO(Xiangze Li): 如果前端能支持区分显示未加载插件的配置(.loaded字段), 这里就的过滤就可以去掉
	extDefaultSettings := make([]*dice.ExtDefaultSettingItem, 0, len(myDice.Config.ExtDefaultSettings))
	for _, i := range myDice.Config.ExtDefaultSettings {
		if i.Loaded {
			extDefaultSettings = append(extDefaultSettings, i)
		}
	}

	info := DiceConfigInfo{
		Config: myDice.Config,

		CommandPrefix:       myDice.CommandPrefix,
		DiceMasters:         myDice.DiceMasters,
		UIPassword:          password,
		LogPageItemLimit:    limit,
		DefaultCocRuleIndex: cocRule,
		ServerAddress:       myDice.Parent.ServeAddress,
		HelpDocEngineType:   myDice.Parent.HelpDocEngineType,
		MaxExecuteTime:      maxExec,
		MaxCocCardGen:       maxCard,
	}
	info.ExtDefaultSettings = extDefaultSettings
	info.DefaultCocRuleIndex = cocRule
	info.MailPassword = emailPasswordMasked
	info.QuitInactiveThresholdDays = info.QuitInactiveThreshold.Hours() / 24

	return c.JSON(http.StatusOK, info)
}

func DiceConfigSet(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	jsonMap := make(map[string]interface{})
	err := json.NewDecoder(c.Request().Body).Decode(&jsonMap)

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

	if err != nil {
		myDice.Logger.Error("DiceConfigSet", err)
		return c.JSON(http.StatusOK, nil)
	}
	if val, ok := jsonMap["commandPrefix"]; ok {
		myDice.CommandPrefix = stringConvert(val)
	}

	if val, ok := jsonMap["diceMasters"]; ok {
		data := stringConvert(val)
		var masters []string
		// 自动修复部分不正确的格式
		for _, i := range data {
			i = strings.ReplaceAll(i, "qq：", "QQ:")
			i = strings.ReplaceAll(i, "QQ：", "QQ:")

			if _, errConv := strconv.Atoi(i); errConv == nil {
				i = "QQ:" + i
			}

			masters = append(masters, i)
		}
		myDice.DiceMasters = masters
	}

	config := &myDice.Config
	if val, ok := jsonMap["noticeIds"]; ok {
		config.NoticeIDs = stringConvert(val)
	}

	if val, ok := jsonMap["defaultCocRuleIndex"]; ok { //nolint:nestif
		valStr, ok := val.(string)
		if ok {
			valStr = strings.TrimSpace(valStr)
			if strings.EqualFold(valStr, "dg") {
				config.DefaultCocRuleIndex = 11
			} else {
				config.DefaultCocRuleIndex, err = strconv.ParseInt(valStr, 10, 64)
				if err == nil {
					if config.DefaultCocRuleIndex > 5 || config.DefaultCocRuleIndex < 0 {
						config.DefaultCocRuleIndex = dice.DefaultConfig.DefaultCocRuleIndex
					}
				}
			}
		}
	}

	if val, ok := jsonMap["maxExecuteTime"]; ok {
		valStr, ok := val.(string)
		if ok {
			valStr = strings.TrimSpace(valStr)
			var valInt int64
			valInt, err = strconv.ParseInt(valStr, 10, 64)
			if err == nil && valInt > 0 {
				config.MaxExecuteTime = valInt
			} /* else {
				Should return some error?
			} */
		}
	}

	if val, ok := jsonMap["maxCocCardGen"]; ok {
		valStr, ok := val.(string)
		if ok {
			valStr = strings.TrimSpace(valStr)
			var valInt int64
			valInt, err = strconv.ParseInt(valStr, 10, 64)
			if err == nil && valInt > 0 {
				config.MaxCocCardGen = valInt
			} /* else {
				Should return some error?
			} */
		}
	}

	if val, ok := jsonMap["cocCardMergeForward"]; ok {
		if b, ok2 := val.(bool); ok2 {
			config.CocCardMergeForward = b
		}
	}

	if val, ok := jsonMap["personalBurst"]; ok {
		valStr, ok := val.(float64)
		if ok {
			customBurst := int64(valStr)
			if customBurst >= 1 {
				config.PersonalBurst = customBurst
			}
		}
	}

	if val, ok := jsonMap["personalReplenishRate"]; ok {
		valStr, ok := val.(string)
		if ok {
			valStr = strings.TrimSpace(valStr)
			newRate, errParse := utils.ParseRate(valStr)
			if errParse == nil && newRate != rate.Limit(0) {
				config.PersonalReplenishRate = newRate
				config.PersonalReplenishRateStr = valStr
			}
		}
	}

	if val, ok := jsonMap["groupBurst"]; ok {
		valStr, ok := val.(float64)
		if ok {
			customBurst := int64(valStr)
			if customBurst >= 1 {
				config.GroupBurst = customBurst
			}
		}
	}

	if val, ok := jsonMap["groupReplenishRate"]; ok {
		valStr, ok := val.(string)
		if ok {
			valStr = strings.TrimSpace(valStr)
			newRate, errParse := utils.ParseRate(valStr)
			if errParse == nil && newRate != rate.Limit(0) {
				config.GroupReplenishRate = newRate
				config.GroupReplenishRateStr = valStr
			}
		}
	}

	if val, ok := jsonMap["onlyLogCommandInGroup"]; ok {
		config.OnlyLogCommandInGroup = val.(bool)
	}

	if val, ok := jsonMap["onlyLogCommandInPrivate"]; ok {
		config.OnlyLogCommandInPrivate = val.(bool)
	}

	if val, ok := jsonMap["refuseGroupInvite"]; ok {
		config.RefuseGroupInvite = val.(bool)
	}

	if val, ok := jsonMap["workInQQChannel"]; ok {
		config.WorkInQQChannel = val.(bool)
	}

	if val, ok := jsonMap["QQChannelLogMessage"]; ok {
		config.QQChannelLogMessage = val.(bool)
	}

	if val, ok := jsonMap["QQChannelAutoOn"]; ok {
		config.QQChannelAutoOn = val.(bool)
	}

	if val, ok := jsonMap["botExtFreeSwitch"]; ok {
		config.BotExtFreeSwitch = val.(bool)
	}
	if val, ok := jsonMap["botExitWithoutAt"]; ok {
		config.BotExitWithoutAt = val.(bool)
	}
	if val, ok := jsonMap["rateLimitEnabled"]; ok {
		config.RateLimitEnabled = val.(bool)
	}
	if val, ok := jsonMap["trustOnlyMode"]; ok {
		config.TrustOnlyMode = val.(bool)
	}

	aliveNoticeMod := false
	if val, ok := jsonMap["aliveNoticeEnable"]; ok {
		config.AliveNoticeEnable = val.(bool)
		aliveNoticeMod = true
	}

	if val, ok := jsonMap["aliveNoticeValue"]; ok {
		config.AliveNoticeValue = val.(string)
		aliveNoticeMod = true
	}
	if aliveNoticeMod {
		myDice.ApplyAliveNotice()
	}

	if val, ok := jsonMap["UILogLimit"]; ok {
		val, err := strconv.ParseInt(val.(string), 10, 64)
		if err == nil {
			if val >= 0 {
				myDice.LogWriter.LogLimit = int(val)
			}
		}
	}

	if val, ok := jsonMap["messageDelayRangeStart"]; ok {
		f, err := strconv.ParseFloat(val.(string), 64)
		if err == nil {
			if f < 0 {
				f = 0
			}
			if config.MessageDelayRangeEnd < f {
				config.MessageDelayRangeEnd = f
			}
			config.MessageDelayRangeStart = f
		}
	}

	if val, ok := jsonMap["messageDelayRangeEnd"]; ok {
		f, err := strconv.ParseFloat(val.(string), 64)
		if err == nil {
			if f < 0 {
				f = 0
			}

			if f >= config.MessageDelayRangeStart {
				config.MessageDelayRangeEnd = f
			}
		}
	}

	if val, ok := jsonMap["friendAddComment"]; ok {
		config.FriendAddComment = strings.TrimSpace(val.(string))
	}

	if val, ok := jsonMap["uiPassword"]; ok {
		if !dm.JustForTest {
			myDice.Parent.UIPasswordHash = val.(string)
			// 清空所有现有的访问令牌，强制重新登录
			myDice.Parent.AccessTokens = dice.SyncMap[string, bool]{}
		}
	}

	if val, ok := jsonMap["extDefaultSettings"]; ok {
		data, err := json.Marshal(val)
		if err == nil {
			var items []*dice.ExtDefaultSettingItem
			err := json.Unmarshal(data, &items)
			if err == nil {
				config.ExtDefaultSettings = items
				myDice.ApplyExtDefaultSettings()
			}
		}
	}

	if val, ok := jsonMap["serveAddress"]; ok {
		if !dm.JustForTest {
			myDice.Parent.ServeAddress = val.(string)
		}
	}

	// if val, ok := jsonMap["customBotExtraText"]; ok {
	// 	myDice.CustomBotExtraText = val.(string)
	// }

	// if val, ok := jsonMap["customDrawKeysText"]; ok {
	// 	myDice.CustomDrawKeysText = val.(string)
	// }

	// if val, ok := jsonMap["customDrawKeysTextEnable"]; ok {
	// 	myDice.CustomDrawKeysTextEnable = val.(bool)
	// }

	if val, ok := jsonMap["logSizeNoticeEnable"]; ok {
		config.LogSizeNoticeEnable = val.(bool)
	}

	if val, ok := jsonMap["logSizeNoticeCount"]; ok {
		count, ok := val.(float64)
		if ok {
			config.LogSizeNoticeCount = int(count)
		}
		if !ok {
			if v, ok := val.(string); ok {
				v2, _ := strconv.ParseInt(v, 10, 64)
				config.LogSizeNoticeCount = int(v2)
			}
		}
		if config.LogSizeNoticeCount == 0 {
			// 不能为零
			config.LogSizeNoticeCount = dice.DefaultConfig.LogSizeNoticeCount
		}
	}

	if val, ok := jsonMap["customReplyConfigEnable"]; ok {
		config.CustomReplyConfigEnable = val.(bool)
	}

	if val, ok := jsonMap["textCmdTrustOnly"]; ok {
		config.TextCmdTrustOnly = val.(bool)
	}

	if val, ok := jsonMap["ignoreUnaddressedBotCmd"]; ok {
		config.IgnoreUnaddressedBotCmd = val.(bool)
	}

	if val, ok := jsonMap["QQEnablePoke"]; ok {
		config.QQEnablePoke = val.(bool)
	}

	if val, ok := jsonMap["playerNameWrapEnable"]; ok {
		config.PlayerNameWrapEnable = val.(bool)
	}

	if val, ok := jsonMap["mailEnable"]; ok {
		config.MailEnable = val.(bool)
	}
	if val, ok := jsonMap["mailFrom"]; ok {
		config.MailFrom = val.(string)
	}
	if val, ok := jsonMap["mailPassword"]; ok {
		config.MailPassword = val.(string)
	}
	if val, ok := jsonMap["mailSmtp"]; ok {
		config.MailSMTP = val.(string)
	}

	if val, ok := jsonMap["quitInactiveThreshold"]; ok {
		set := false
		switch v := val.(type) {
		case string:
			if vv, err := strconv.ParseFloat(v, 64); err == nil {
				config.QuitInactiveThreshold = time.Duration(float64(24*time.Hour) * vv)
				set = true
			}
		case float64:
			config.QuitInactiveThreshold = time.Duration(float64(24*time.Hour) * v)
			set = true
		case int64:
			config.QuitInactiveThreshold = 24 * time.Hour * time.Duration(v)
			set = true
		default:
			// ignore
		}
		if set {
			myDice.ResetQuitInactiveCron()
		}
	}

	if val, ok := jsonMap["quitInactiveBatchSize"]; ok {
		if v, ok := val.(float64); ok {
			if vv := int64(v); vv > 0 {
				config.QuitInactiveBatchSize = vv
			}
		}
	}

	if val, ok := jsonMap["quitInactiveBatchWait"]; ok {
		if v, ok := val.(float64); ok {
			if vv := int64(v); vv > 0 {
				config.QuitInactiveBatchWait = vv
			}
		}
	}

	// 统一标记为修改
	myDice.MarkModified()
	myDice.Parent.Save()
	return c.JSON(http.StatusOK, nil)
}

func DiceMailTest(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return Error(&c, "展示模式不支持该操作", Response{"testMode": true})
	}

	err := myDice.SendMail("", dice.MailTest)
	if err != nil {
		return Error(&c, err.Error(), Response{})
	}
	return Success(&c, Response{})
}

func vmVersionSet(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return Error(&c, "展示模式不支持该操作", Response{"testMode": true})
	}

	var req []struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}

	err := c.Bind(&req)
	if err != nil {
		return Error(&c, err.Error(), nil)
	}
	if len(req) == 0 {
		return Error(&c, "缺少设置vm版本的参数", Response{})
	}

	var failTypes []string
	for _, data := range req {
		if data.Type == "" || data.Value == "" {
			failTypes = append(failTypes, data.Type)
			continue
		}
		switch data.Type {
		case dice.VMVersionReply:
			(&myDice.Config).VMVersionForReply = data.Value
		case dice.VMVersionDeck:
			(&myDice.Config).VMVersionForDeck = data.Value
		case dice.VMVersionCustomText:
			(&myDice.Config).VMVersionForCustomText = data.Value
		case dice.VmVersionMsg:
			(&myDice.Config).VMVersionForMsg = data.Value
		default:
			failTypes = append(failTypes, data.Type)
		}
	}
	myDice.MarkModified()
	myDice.Save(false)

	return Success(&c, Response{
		"failTypes": failTypes,
	})
}
