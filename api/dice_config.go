package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sealdice-core/dice"
	"sealdice-core/utils"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

type DiceConfigInfo struct {
	// 注：form其实不需要
	CommandPrefix           []string `json:"commandPrefix" form:"commandPrefix"`                     // 指令前缀
	DiceMasters             []string `json:"diceMasters" form:"diceMasters"`                         // 骰主设置，需要格式: 平台:帐号
	NoticeIds               []string `json:"noticeIds"`                                              // 通知设置，需要格式: 平台:帐号
	OnlyLogCommandInGroup   bool     `json:"onlyLogCommandInGroup" form:"onlyLogCommandInGroup"`     // 日志中仅记录命令
	OnlyLogCommandInPrivate bool     `json:"onlyLogCommandInPrivate" form:"onlyLogCommandInPrivate"` // 日志中仅记录命令
	WorkInQQChannel         bool     `json:"workInQQChannel"`                                        // 在QQ频道中开启
	MessageDelayRangeStart  float64  `json:"messageDelayRangeStart" form:"messageDelayRangeStart"`   // 指令延迟区间
	MessageDelayRangeEnd    float64  `json:"messageDelayRangeEnd" form:"messageDelayRangeEnd"`
	UIPassword              string   `json:"uiPassword" form:"uiPassword"`
	HelpDocEngineType       int      `json:"helpDocEngineType"`
	MasterUnlockCode        string   `json:"masterUnlockCode" form:"masterUnlockCode"`
	ServeAddress            string   `json:"serveAddress" form:"serveAddress"`
	MasterUnlockCodeTime    int64    `json:"masterUnlockCodeTime"`
	LogPageItemLimit        int64    `json:"logPageItemLimit"`
	FriendAddComment        string   `json:"friendAddComment"`
	AutoReloginEnable       bool     `json:"autoReloginEnable"`
	QQChannelAutoOn         bool     `json:"QQChannelAutoOn"`
	QQChannelLogMessage     bool     `json:"QQChannelLogMessage"`
	RefuseGroupInvite       bool     `json:"refuseGroupInvite"` // 拒绝群组邀请

	QuitInactiveThreshold float64 `json:"quitInactiveThreshold"` // 退出不活跃群组阈值(天)

	DefaultCocRuleIndex string `json:"defaultCocRuleIndex"` // 默认coc index
	MaxExecuteTime      string `json:"maxExecuteTime"`      // 最大骰点次数
	MaxCocCardGen       string `json:"maxCocCardGen"`       // 最大coc制卡数

	ExtDefaultSettings []*dice.ExtDefaultSettingItem `yaml:"extDefaultSettings" json:"extDefaultSettings"` // 新群扩展按此顺序加载
	BotExtFreeSwitch   bool                          `json:"botExtFreeSwitch"`
	TrustOnlyMode      bool                          `json:"trustOnlyMode"`
	AliveNoticeEnable  bool                          `json:"aliveNoticeEnable"`
	AliveNoticeValue   string                        `json:"aliveNoticeValue"`
	ReplyDebugMode     bool                          `json:"replyDebugMode"`

	CustomReplyConfigEnable bool `json:"customReplyConfigEnable"` // 是否开启reply

	LogSizeNoticeEnable bool `json:"logSizeNoticeEnable"` // 开启日志数量提示
	LogSizeNoticeCount  int  `json:"logSizeNoticeCount"`  // 日志数量提示阈值，默认500

	TextCmdTrustOnly        bool `json:"textCmdTrustOnly"`        // text命令只允许信任用户和master
	IgnoreUnaddressedBotCmd bool `json:"ignoreUnaddressedBotCmd"` // 不响应群聊裸bot指令
	QQEnablePoke            bool `json:"QQEnablePoke"`            // QQ允许戳一戳
	PlayerNameWrapEnable    bool `json:"playerNameWrapEnable"`    // 玩家名外框

	MailEnable   bool   `json:"mailEnable"`
	MailFrom     string `json:"mailFrom"`     // 邮箱来源
	MailPassword string `json:"mailPassword"` // 邮箱密钥/密码
	MailSMTP     string `json:"mailSmtp"`     // 邮箱 smtp 地址

	RateLimitEnabled       bool   `json:"rateLimitEnabled"`      // 是否开启限速
	PersonalReplenishRate  string `json:"personalReplenishRate"` // 个人自定义速率
	PersonalReplenishBurst int64  `json:"personalBurst"`         // 个人自定义上限
	GroupReplenishRate     string `json:"groupReplenishRate"`    // 群组自定义速率
	GroupReplenishBurst    int64  `json:"groupBurst"`            // 群组自定义上限
}

func DiceConfig(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	password := ""
	if myDice.Parent.UIPasswordHash != "" {
		password = "------"
	}

	limit := myDice.UILogLimit
	if limit == 0 {
		limit = 100
	}
	myDice.UnlockCodeUpdate(false)

	cocRule := strconv.FormatInt(myDice.DefaultCocRuleIndex, 10)
	if myDice.DefaultCocRuleIndex == 11 {
		cocRule = "dg"
	}

	maxExec := strconv.FormatInt(myDice.MaxExecuteTime, 10)

	maxCard := strconv.FormatInt(myDice.MaxCocCardGen, 10)

	emailPasswordMasked := ""
	if myDice.MailPassword != "" {
		emailPasswordMasked = "******"
	}

	// 过滤掉未加载的: 包括关闭的和已经删除的
	// TODO(Xiangze Li): 如果前端能支持区分显示未加载插件的配置(.loaded字段), 这里就的过滤就可以去掉
	extDefaultSettings := make([]*dice.ExtDefaultSettingItem, 0, len(myDice.ExtDefaultSettings))
	for _, i := range myDice.ExtDefaultSettings {
		if i.Loaded {
			extDefaultSettings = append(extDefaultSettings, i)
		}
	}

	info := DiceConfigInfo{
		CommandPrefix:           myDice.CommandPrefix,
		DiceMasters:             myDice.DiceMasters,
		NoticeIds:               myDice.NoticeIDs,
		OnlyLogCommandInPrivate: myDice.OnlyLogCommandInPrivate,
		OnlyLogCommandInGroup:   myDice.OnlyLogCommandInGroup,
		MessageDelayRangeStart:  myDice.MessageDelayRangeStart,
		MessageDelayRangeEnd:    myDice.MessageDelayRangeEnd,
		UIPassword:              password,
		MasterUnlockCode:        myDice.MasterUnlockCode,
		MasterUnlockCodeTime:    myDice.MasterUnlockCodeTime,
		WorkInQQChannel:         myDice.WorkInQQChannel,
		LogPageItemLimit:        limit,
		FriendAddComment:        myDice.FriendAddComment,
		AutoReloginEnable:       myDice.AutoReloginEnable,
		QQChannelAutoOn:         myDice.QQChannelAutoOn,
		QQChannelLogMessage:     myDice.QQChannelLogMessage,
		RefuseGroupInvite:       myDice.RefuseGroupInvite,
		RateLimitEnabled:        myDice.RateLimitEnabled,

		ExtDefaultSettings:  extDefaultSettings,
		DefaultCocRuleIndex: cocRule,

		QuitInactiveThreshold: myDice.QuitInactiveThreshold.Hours() / 24,

		BotExtFreeSwitch:  myDice.BotExtFreeSwitch,
		TrustOnlyMode:     myDice.TrustOnlyMode,
		AliveNoticeEnable: myDice.AliveNoticeEnable,
		AliveNoticeValue:  myDice.AliveNoticeValue,
		ReplyDebugMode:    myDice.ReplyDebugMode,

		ServeAddress:      myDice.Parent.ServeAddress,
		HelpDocEngineType: myDice.Parent.HelpDocEngineType,

		// 1.0 正式
		LogSizeNoticeEnable:     myDice.LogSizeNoticeEnable,
		LogSizeNoticeCount:      myDice.LogSizeNoticeCount,
		CustomReplyConfigEnable: myDice.CustomReplyConfigEnable,

		// 1.2
		TextCmdTrustOnly:     myDice.TextCmdTrustOnly,
		QQEnablePoke:         myDice.QQEnablePoke,
		PlayerNameWrapEnable: myDice.PlayerNameWrapEnable,

		// 1.3?
		MailEnable:              myDice.MailEnable,
		MailFrom:                myDice.MailFrom,
		MailPassword:            emailPasswordMasked,
		MailSMTP:                myDice.MailSMTP,
		MaxExecuteTime:          maxExec,
		MaxCocCardGen:           maxCard,
		PersonalReplenishRate:   myDice.PersonalReplenishRateStr,
		PersonalReplenishBurst:  myDice.PersonalBurst,
		GroupReplenishRate:      myDice.GroupReplenishRateStr,
		GroupReplenishBurst:     myDice.GroupBurst,
		IgnoreUnaddressedBotCmd: myDice.IgnoreUnaddressedBotCmd,
	}
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
		fmt.Println(err)
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

	if val, ok := jsonMap["noticeIds"]; ok {
		myDice.NoticeIDs = stringConvert(val)
	}

	if val, ok := jsonMap["defaultCocRuleIndex"]; ok { //nolint:nestif
		valStr, ok := val.(string)
		if ok {
			valStr = strings.TrimSpace(valStr)
			if strings.EqualFold(valStr, "dg") {
				myDice.DefaultCocRuleIndex = 11
			} else {
				myDice.DefaultCocRuleIndex, err = strconv.ParseInt(valStr, 10, 64)
				if err == nil {
					if myDice.DefaultCocRuleIndex > 5 || myDice.DefaultCocRuleIndex < 0 {
						myDice.DefaultCocRuleIndex = 0
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
				myDice.MaxExecuteTime = valInt
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
				myDice.MaxCocCardGen = valInt
			} /* else {
				Should return some error?
			} */
		}
	}

	if val, ok := jsonMap["personalBurst"]; ok {
		valStr, ok := val.(float64)
		if ok {
			customBurst := int64(valStr)
			if customBurst >= 1 {
				myDice.PersonalBurst = customBurst
			}
		}
	}

	if val, ok := jsonMap["personalReplenishRate"]; ok {
		valStr, ok := val.(string)
		if ok {
			valStr = strings.TrimSpace(valStr)
			newRate, errParse := utils.ParseRate(valStr)
			if errParse == nil && newRate != rate.Limit(0) {
				myDice.PersonalReplenishRate = newRate
				myDice.PersonalReplenishRateStr = valStr
			}
		}
	}

	if val, ok := jsonMap["groupBurst"]; ok {
		valStr, ok := val.(float64)
		if ok {
			customBurst := int64(valStr)
			if customBurst >= 1 {
				myDice.GroupBurst = customBurst
			}
		}
	}

	if val, ok := jsonMap["groupReplenishRate"]; ok {
		valStr, ok := val.(string)
		if ok {
			valStr = strings.TrimSpace(valStr)
			newRate, errParse := utils.ParseRate(valStr)
			if errParse == nil && newRate != rate.Limit(0) {
				myDice.GroupReplenishRate = newRate
				myDice.GroupReplenishRateStr = valStr
			}
		}
	}

	if val, ok := jsonMap["onlyLogCommandInGroup"]; ok {
		myDice.OnlyLogCommandInGroup = val.(bool)
	}

	if val, ok := jsonMap["onlyLogCommandInPrivate"]; ok {
		myDice.OnlyLogCommandInPrivate = val.(bool)
	}

	if val, ok := jsonMap["autoReloginEnable"]; ok {
		myDice.AutoReloginEnable = val.(bool)
	}

	if val, ok := jsonMap["refuseGroupInvite"]; ok {
		myDice.RefuseGroupInvite = val.(bool)
	}

	if val, ok := jsonMap["workInQQChannel"]; ok {
		myDice.WorkInQQChannel = val.(bool)
	}

	if val, ok := jsonMap["QQChannelLogMessage"]; ok {
		myDice.QQChannelLogMessage = val.(bool)
	}

	if val, ok := jsonMap["QQChannelAutoOn"]; ok {
		myDice.QQChannelAutoOn = val.(bool)
	}

	if val, ok := jsonMap["botExtFreeSwitch"]; ok {
		myDice.BotExtFreeSwitch = val.(bool)
	}
	if val, ok := jsonMap["rateLimitEnabled"]; ok {
		myDice.RateLimitEnabled = val.(bool)
	}
	if val, ok := jsonMap["trustOnlyMode"]; ok {
		myDice.TrustOnlyMode = val.(bool)
	}

	aliveNoticeMod := false
	if val, ok := jsonMap["aliveNoticeEnable"]; ok {
		myDice.AliveNoticeEnable = val.(bool)
		aliveNoticeMod = true
	}

	if val, ok := jsonMap["aliveNoticeValue"]; ok {
		myDice.AliveNoticeValue = val.(string)
		aliveNoticeMod = true
	}
	if aliveNoticeMod {
		myDice.ApplyAliveNotice()
	}

	if val, ok := jsonMap["UILogLimit"]; ok {
		val, err := strconv.ParseInt(val.(string), 10, 64)
		if err == nil {
			if val >= 0 {
				myDice.LogWriter.LogLimit = val
			}
		}
	}

	if val, ok := jsonMap["messageDelayRangeStart"]; ok {
		f, err := strconv.ParseFloat(val.(string), 64)
		if err == nil {
			if f < 0 {
				f = 0
			}
			if myDice.MessageDelayRangeEnd < f {
				myDice.MessageDelayRangeEnd = f
			}
			myDice.MessageDelayRangeStart = f
		}
	}

	if val, ok := jsonMap["messageDelayRangeEnd"]; ok {
		f, err := strconv.ParseFloat(val.(string), 64)
		if err == nil {
			if f < 0 {
				f = 0
			}

			if f >= myDice.MessageDelayRangeStart {
				myDice.MessageDelayRangeEnd = f
			}
		}
	}

	if val, ok := jsonMap["friendAddComment"]; ok {
		myDice.FriendAddComment = strings.TrimSpace(val.(string))
	}

	if val, ok := jsonMap["uiPassword"]; ok {
		if !dm.JustForTest {
			myDice.Parent.UIPasswordHash = val.(string)
		}
	}

	if val, ok := jsonMap["extDefaultSettings"]; ok {
		data, err := json.Marshal(val)
		if err == nil {
			var items []*dice.ExtDefaultSettingItem
			err := json.Unmarshal(data, &items)
			if err == nil {
				myDice.ExtDefaultSettings = items
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
		myDice.LogSizeNoticeEnable = val.(bool)
	}

	if val, ok := jsonMap["logSizeNoticeCount"]; ok {
		count, ok := val.(float64)
		if ok {
			myDice.LogSizeNoticeCount = int(count)
		}
		if !ok {
			if v, ok := val.(string); ok {
				v2, _ := strconv.ParseInt(v, 10, 64)
				myDice.LogSizeNoticeCount = int(v2)
			}
		}
		if myDice.LogSizeNoticeCount == 0 {
			// 不能为零
			myDice.LogSizeNoticeCount = 500
		}
	}

	if val, ok := jsonMap["customReplyConfigEnable"]; ok {
		myDice.CustomReplyConfigEnable = val.(bool)
	}

	if val, ok := jsonMap["textCmdTrustOnly"]; ok {
		myDice.TextCmdTrustOnly = val.(bool)
	}

	if val, ok := jsonMap["ignoreUnaddressedBotCmd"]; ok {
		myDice.IgnoreUnaddressedBotCmd = val.(bool)
	}

	if val, ok := jsonMap["QQEnablePoke"]; ok {
		myDice.QQEnablePoke = val.(bool)
	}

	if val, ok := jsonMap["playerNameWrapEnable"]; ok {
		myDice.PlayerNameWrapEnable = val.(bool)
	}

	if val, ok := jsonMap["mailEnable"]; ok {
		myDice.MailEnable = val.(bool)
	}
	if val, ok := jsonMap["mailFrom"]; ok {
		myDice.MailFrom = val.(string)
	}
	if val, ok := jsonMap["mailPassword"]; ok {
		myDice.MailPassword = val.(string)
	}
	if val, ok := jsonMap["mailSmtp"]; ok {
		myDice.MailSMTP = val.(string)
	}

	if val, ok := jsonMap["quitInactiveThreshold"]; ok {
		set := false
		switch v := val.(type) {
		case string:
			if vv, err := strconv.ParseFloat(v, 64); err == nil {
				myDice.QuitInactiveThreshold = time.Duration(float64(24*time.Hour) * vv)
				set = true
			}
		case float64:
			myDice.QuitInactiveThreshold = time.Duration(float64(24*time.Hour) * v)
			set = true
		case int64:
			myDice.QuitInactiveThreshold = 24 * time.Hour * time.Duration(v)
			set = true
		default:
			// ignore
		}
		if set {
			myDice.ResetQuitInactiveCron()
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
