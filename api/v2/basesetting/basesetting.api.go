package basesetting

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"golang.org/x/time/rate"

	"sealdice-core/dice"
	"sealdice-core/model/common/request"
	"sealdice-core/model/common/response"
	"sealdice-core/utils"
)

const (
	uiPasswordMasked   = "------"
	mailPasswordMasked = "******"
)

type Service struct {
	dice *dice.Dice
	dm   *dice.DiceManager
}

func NewService(dm *dice.DiceManager) *Service {
	return &Service{
		dice: dm.GetDice(),
		dm:   dm,
	}
}

func (s *Service) RegisterRoutes(grp *huma.Group) {
	huma.Get(grp, "/schema", s.GetSchema, func(o *huma.Operation) {
		o.Description = "获取基本设置页 schema"
	})
	huma.Get(grp, "/value", s.GetValue, func(o *huma.Operation) {
		o.Description = "获取基本设置当前值"
	})
}

func (s *Service) RegisterProtectedRoutes(grp *huma.Group) {
	huma.Put(grp, "/value", s.SetValue, func(o *huma.Operation) {
		o.Description = "保存基本设置（支持部分字段提交）"
	})
	huma.Post(grp, "/mail-test", s.MailTest, func(o *huma.Operation) {
		o.Description = "发送测试邮件"
	})
	huma.Post(grp, "/upgrade", s.Upgrade, func(o *huma.Operation) {
		o.Description = "上传固件升级包"
	})
}

func (s *Service) GetSchema(_ context.Context, _ *request.Empty) (*response.ItemResponse[BaseSettingSchemaResp], error) {
	return response.NewItemResponse(buildBaseSettingSchema()), nil
}

func (s *Service) GetValue(_ context.Context, _ *request.Empty) (*response.ItemResponse[BaseSettingValueResp], error) {
	return response.NewItemResponse(s.buildValue()), nil
}

func (s *Service) SetValue(_ context.Context, req *BaseSettingUpdateReq) (*response.ItemResponse[BaseSettingActionResp], error) {
	if err := s.applyPatch(req.Body); err != nil {
		return nil, err
	}
	s.dice.MarkModified()
	s.dice.Parent.Save()
	return response.NewItemResponse(BaseSettingActionResp{Success: true}), nil
}

func (s *Service) MailTest(_ context.Context, _ *request.Empty) (*response.ItemResponse[BaseSettingActionResp], error) {
	if err := s.dice.SendMail("", dice.MailTest); err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}
	return response.NewItemResponse(BaseSettingActionResp{Success: true}), nil
}

func (s *Service) Upgrade(_ context.Context, req *BaseSettingUpgradeReq) (*response.ItemResponse[BaseSettingActionResp], error) {
	if s.dm.UpdateSealdiceByFile == nil {
		return nil, huma.Error500InternalServerError("骰子没有正确初始化，无法使用此功能")
	}
	if s.dm.ContainerMode {
		return nil, huma.Error400BadRequest("容器模式下禁止更新，请手动拉取最新镜像")
	}
	form, err := extractUpgradeForm(req.RawBody)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = form.File.Close()
	}()

	filename := "./new_package"
	if runtime.GOOS == "windows" {
		filename += ".zip"
	} else {
		filename += ".tar.gz"
	}
	dst, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}
	if _, err = io.Copy(dst, form.File); err != nil {
		_ = dst.Close()
		return nil, huma.Error500InternalServerError(err.Error())
	}
	_ = dst.Close()

	go func(path string) {
		if !s.dm.UpdateSealdiceByFile(path) {
			s.dice.Logger.Error("更新骰子失败")
		}
	}(filename)

	return response.NewItemResponse(BaseSettingActionResp{Success: true}), nil
}

func (s *Service) buildValue() BaseSettingValueResp {
	password := ""
	if s.dice.Parent.UIPasswordHash != "" {
		password = uiPasswordMasked
	}
	s.dice.UnlockCodeUpdate(false)

	cocRule := strconv.FormatInt(s.dice.Config.DefaultCocRuleIndex, 10)
	if s.dice.Config.DefaultCocRuleIndex == 11 {
		cocRule = "dg"
	}
	maxExec := strconv.FormatInt(s.dice.Config.MaxExecuteTime, 10)
	maxCard := strconv.FormatInt(s.dice.Config.MaxCocCardGen, 10)
	emailPassword := ""
	if s.dice.Config.MailPassword != "" {
		emailPassword = mailPasswordMasked
	}

	extDefaultSettings := make([]*BaseSettingExtDefaultSettingItem, 0, len(s.dice.Config.ExtDefaultSettings))
	for _, item := range s.dice.Config.ExtDefaultSettings {
		if item == nil || !item.Loaded {
			continue
		}
		extDefaultSettings = append(extDefaultSettings, &BaseSettingExtDefaultSettingItem{
			Name:            item.Name,
			AutoActive:      item.AutoActive,
			DisabledCommand: cloneBoolMap(item.DisabledCommand),
			Loaded:          item.Loaded,
		})
	}

	return BaseSettingValueResp{
		CommandPrefix:           append([]string{}, s.dice.CommandPrefix...),
		DiceMasters:             append([]string{}, s.dice.DiceMasters...),
		NoticeIds:               append([]string{}, s.dice.Config.NoticeIDs...),
		MasterUnlockCode:        s.dice.MasterUnlockCode,
		UIPassword:              password,
		MailEnable:              s.dice.Config.MailEnable,
		MailFrom:                s.dice.Config.MailFrom,
		MailPassword:            emailPassword,
		MailSmtp:                s.dice.Config.MailSMTP,
		TrustOnlyMode:           s.dice.Config.TrustOnlyMode,
		BotExtFreeSwitch:        s.dice.Config.BotExtFreeSwitch,
		QQEnablePoke:            s.dice.Config.QQEnablePoke,
		TextCmdTrustOnly:        s.dice.Config.TextCmdTrustOnly,
		IgnoreUnaddressedBotCmd: s.dice.Config.IgnoreUnaddressedBotCmd,
		AliveNoticeEnable:       s.dice.Config.AliveNoticeEnable,
		AliveNoticeValue:        s.dice.Config.AliveNoticeValue,
		LogSizeNoticeEnable:     s.dice.Config.LogSizeNoticeEnable,
		LogSizeNoticeCount:      s.dice.Config.LogSizeNoticeCount,
		PlayerNameWrapEnable:    s.dice.Config.PlayerNameWrapEnable,
		OnlyLogCommandInGroup:   s.dice.Config.OnlyLogCommandInGroup,
		OnlyLogCommandInPrivate: s.dice.Config.OnlyLogCommandInPrivate,
		RateLimitEnabled:        s.dice.Config.RateLimitEnabled,
		PersonalReplenishRate:   s.dice.Config.PersonalReplenishRateStr,
		PersonalBurst:           s.dice.Config.PersonalBurst,
		GroupReplenishRate:      s.dice.Config.GroupReplenishRateStr,
		GroupBurst:              s.dice.Config.GroupBurst,
		ServeAddress:            s.dice.Parent.ServeAddress,
		RefuseGroupInvite:       s.dice.Config.RefuseGroupInvite,
		FriendAddComment:        s.dice.Config.FriendAddComment,
		WorkInQQChannel:         s.dice.Config.WorkInQQChannel,
		QQChannelAutoOn:         s.dice.Config.QQChannelAutoOn,
		QQChannelLogMessage:     s.dice.Config.QQChannelLogMessage,
		DefaultCocRuleIndex:     cocRule,
		MaxCocCardGen:           maxCard,
		MaxExecuteTime:          maxExec,
		MessageDelayRangeStart:  s.dice.Config.MessageDelayRangeStart,
		MessageDelayRangeEnd:    s.dice.Config.MessageDelayRangeEnd,
		QuitInactiveThreshold:   s.dice.Config.QuitInactiveThreshold.Hours() / 24,
		QuitInactiveBatchSize:   s.dice.Config.QuitInactiveBatchSize,
		QuitInactiveBatchWait:   s.dice.Config.QuitInactiveBatchWait,
		ExtDefaultSettings:      extDefaultSettings,
	}
}

func (s *Service) applyPatch(jsonMap map[string]interface{}) error {
	stringConvert := func(val interface{}) []string {
		var list []string
		for _, item := range val.([]interface{}) {
			text, ok := item.(string)
			if ok && text != "" {
				list = append(list, text)
			}
		}
		return list
	}

	if val, ok := jsonMap["commandPrefix"]; ok {
		s.dice.CommandPrefix = stringConvert(val)
	}
	if val, ok := jsonMap["diceMasters"]; ok {
		data := stringConvert(val)
		var masters []string
		for _, item := range data {
			item = strings.ReplaceAll(item, "qq：", "QQ:")
			item = strings.ReplaceAll(item, "QQ：", "QQ:")
			if _, errConv := strconv.Atoi(item); errConv == nil {
				item = "QQ:" + item
			}
			masters = append(masters, item)
		}
		s.dice.DiceMasters = masters
	}

	config := &s.dice.Config
	if val, ok := jsonMap["noticeIds"]; ok {
		config.NoticeIDs = stringConvert(val)
	}
	if val, ok := jsonMap["defaultCocRuleIndex"]; ok {
		if valStr, ok := stringify(val); ok {
			valStr = strings.TrimSpace(valStr)
			if strings.EqualFold(valStr, "dg") {
				config.DefaultCocRuleIndex = 11
			} else if parsed, err := strconv.ParseInt(valStr, 10, 64); err == nil {
				if parsed <= 5 && parsed >= 0 {
					config.DefaultCocRuleIndex = parsed
				} else {
					config.DefaultCocRuleIndex = dice.DefaultConfig.DefaultCocRuleIndex
				}
			}
		}
	}
	if val, ok := jsonMap["maxExecuteTime"]; ok {
		if parsed, ok := parsePositiveInt64String(val); ok {
			config.MaxExecuteTime = parsed
		}
	}
	if val, ok := jsonMap["maxCocCardGen"]; ok {
		if parsed, ok := parsePositiveInt64String(val); ok {
			config.MaxCocCardGen = parsed
		}
	}
	if val, ok := jsonMap["personalBurst"]; ok {
		if parsed, ok := parsePositiveInt64(val); ok {
			config.PersonalBurst = parsed
		}
	}
	if val, ok := jsonMap["personalReplenishRate"]; ok {
		if value, ok := stringify(val); ok {
			value = strings.TrimSpace(value)
			if newRate, err := utils.ParseRate(value); err == nil && newRate != rate.Limit(0) {
				config.PersonalReplenishRate = newRate
				config.PersonalReplenishRateStr = value
			}
		}
	}
	if val, ok := jsonMap["groupBurst"]; ok {
		if parsed, ok := parsePositiveInt64(val); ok {
			config.GroupBurst = parsed
		}
	}
	if val, ok := jsonMap["groupReplenishRate"]; ok {
		if value, ok := stringify(val); ok {
			value = strings.TrimSpace(value)
			if newRate, err := utils.ParseRate(value); err == nil && newRate != rate.Limit(0) {
				config.GroupReplenishRate = newRate
				config.GroupReplenishRateStr = value
			}
		}
	}
	if val, ok := jsonMap["onlyLogCommandInGroup"]; ok {
		if parsed, ok := val.(bool); ok {
			config.OnlyLogCommandInGroup = parsed
		}
	}
	if val, ok := jsonMap["onlyLogCommandInPrivate"]; ok {
		if parsed, ok := val.(bool); ok {
			config.OnlyLogCommandInPrivate = parsed
		}
	}
	if val, ok := jsonMap["refuseGroupInvite"]; ok {
		if parsed, ok := val.(bool); ok {
			config.RefuseGroupInvite = parsed
		}
	}
	if val, ok := jsonMap["workInQQChannel"]; ok {
		if parsed, ok := val.(bool); ok {
			config.WorkInQQChannel = parsed
		}
	}
	if val, ok := jsonMap["QQChannelLogMessage"]; ok {
		if parsed, ok := val.(bool); ok {
			config.QQChannelLogMessage = parsed
		}
	}
	if val, ok := jsonMap["QQChannelAutoOn"]; ok {
		if parsed, ok := val.(bool); ok {
			config.QQChannelAutoOn = parsed
		}
	}
	if val, ok := jsonMap["botExtFreeSwitch"]; ok {
		if parsed, ok := val.(bool); ok {
			config.BotExtFreeSwitch = parsed
		}
	}
	if val, ok := jsonMap["rateLimitEnabled"]; ok {
		if parsed, ok := val.(bool); ok {
			config.RateLimitEnabled = parsed
		}
	}
	if val, ok := jsonMap["trustOnlyMode"]; ok {
		if parsed, ok := val.(bool); ok {
			config.TrustOnlyMode = parsed
		}
	}
	aliveNoticeMod := false
	if val, ok := jsonMap["aliveNoticeEnable"]; ok {
		if parsed, ok := val.(bool); ok {
			config.AliveNoticeEnable = parsed
			aliveNoticeMod = true
		}
	}
	if val, ok := jsonMap["aliveNoticeValue"]; ok {
		if parsed, ok := val.(string); ok {
			config.AliveNoticeValue = parsed
			aliveNoticeMod = true
		}
	}
	if aliveNoticeMod {
		s.dice.ApplyAliveNotice()
	}
	if val, ok := jsonMap["messageDelayRangeStart"]; ok {
		if parsed, ok := parseNonNegativeFloat64(val); ok {
			if config.MessageDelayRangeEnd < parsed {
				config.MessageDelayRangeEnd = parsed
			}
			config.MessageDelayRangeStart = parsed
		}
	}
	if val, ok := jsonMap["messageDelayRangeEnd"]; ok {
		if parsed, ok := parseNonNegativeFloat64(val); ok && parsed >= config.MessageDelayRangeStart {
			config.MessageDelayRangeEnd = parsed
		}
	}
	if val, ok := jsonMap["friendAddComment"]; ok {
		if parsed, ok := val.(string); ok {
			config.FriendAddComment = strings.TrimSpace(parsed)
		}
	}
	if val, ok := jsonMap["uiPassword"]; ok {
		if parsed, ok := val.(string); ok && parsed != "" && parsed != uiPasswordMasked {
			s.dice.Parent.UIPasswordHash = parsed
			s.dice.Parent.AccessTokens = dice.SyncMap[string, bool]{}
		}
	}
	if val, ok := jsonMap["extDefaultSettings"]; ok {
		data, err := json.Marshal(val)
		if err == nil {
			var items []*dice.ExtDefaultSettingItem
			if err = json.Unmarshal(data, &items); err == nil {
				config.ExtDefaultSettings = items
				s.dice.ApplyExtDefaultSettings()
			}
		}
	}
	if val, ok := jsonMap["serveAddress"]; ok {
		if parsed, ok := val.(string); ok {
			s.dice.Parent.ServeAddress = parsed
		}
	}
	if val, ok := jsonMap["logSizeNoticeEnable"]; ok {
		if parsed, ok := val.(bool); ok {
			config.LogSizeNoticeEnable = parsed
		}
	}
	if val, ok := jsonMap["logSizeNoticeCount"]; ok {
		if parsed, ok := parseInt(val); ok {
			config.LogSizeNoticeCount = parsed
			if config.LogSizeNoticeCount == 0 {
				config.LogSizeNoticeCount = dice.DefaultConfig.LogSizeNoticeCount
			}
		}
	}
	if val, ok := jsonMap["textCmdTrustOnly"]; ok {
		if parsed, ok := val.(bool); ok {
			config.TextCmdTrustOnly = parsed
		}
	}
	if val, ok := jsonMap["ignoreUnaddressedBotCmd"]; ok {
		if parsed, ok := val.(bool); ok {
			config.IgnoreUnaddressedBotCmd = parsed
		}
	}
	if val, ok := jsonMap["QQEnablePoke"]; ok {
		if parsed, ok := val.(bool); ok {
			config.QQEnablePoke = parsed
		}
	}
	if val, ok := jsonMap["playerNameWrapEnable"]; ok {
		if parsed, ok := val.(bool); ok {
			config.PlayerNameWrapEnable = parsed
		}
	}
	if val, ok := jsonMap["mailEnable"]; ok {
		if parsed, ok := val.(bool); ok {
			config.MailEnable = parsed
		}
	}
	if val, ok := jsonMap["mailFrom"]; ok {
		if parsed, ok := val.(string); ok {
			config.MailFrom = parsed
		}
	}
	if val, ok := jsonMap["mailPassword"]; ok {
		if parsed, ok := val.(string); ok && parsed != "" && parsed != mailPasswordMasked {
			config.MailPassword = parsed
		}
	}
	if val, ok := jsonMap["mailSmtp"]; ok {
		if parsed, ok := val.(string); ok {
			config.MailSMTP = parsed
		}
	}
	if val, ok := jsonMap["quitInactiveThreshold"]; ok {
		if parsed, ok := parseFloat64(val); ok {
			config.QuitInactiveThreshold = time.Duration(float64(24*time.Hour) * parsed)
			s.dice.ResetQuitInactiveCron()
		}
	}
	if val, ok := jsonMap["quitInactiveBatchSize"]; ok {
		if parsed, ok := parsePositiveInt64(val); ok {
			config.QuitInactiveBatchSize = parsed
		}
	}
	if val, ok := jsonMap["quitInactiveBatchWait"]; ok {
		if parsed, ok := parsePositiveInt64(val); ok {
			config.QuitInactiveBatchWait = parsed
		}
	}
	return nil
}

func buildBaseSettingSchema() BaseSettingSchemaResp {
	option := func(label, value string) *BaseSettingOption {
		return &BaseSettingOption{Label: label, Value: value}
	}
	field := func(id, key, label, kind string, keywords ...string) *BaseSettingFieldSchema {
		return &BaseSettingFieldSchema{
			ID:       id,
			Key:      key,
			Label:    label,
			Kind:     kind,
			Keywords: keywords,
		}
	}
	return BaseSettingSchemaResp{
		Tabs: []*BaseSettingTabSchema{
			{
				ID:    "master-notice",
				Title: "Master 与通知",
				Groups: []*BaseSettingGroupSchema{
					{
						ID:    "master",
						Title: "Master 管理",
						Fields: []*BaseSettingFieldSchema{
							{ID: "master-unlock-code", Key: "masterUnlockCode", Label: "Master 解锁码", Kind: "unlock-code", Readonly: true, Keywords: []string{"master", "解锁码", "抢占", "骰主"}},
							{ID: "dice-masters", Key: "diceMasters", Label: "Master 列表", Kind: "string-list", Keywords: []string{"骰主", "master列表", "管理权限"}},
						},
					},
					{
						ID:    "notice-mail",
						Title: "通知与邮件",
						Fields: []*BaseSettingFieldSchema{
							{ID: "notice-ids", Key: "noticeIds", Label: "消息通知列表", Kind: "string-list", Keywords: []string{"通知列表", "通知ID", "邮件通知目标"}},
							field("mail-enable", "mailEnable", "邮箱通知", "boolean", "邮件", "邮件通知"),
							field("mail-from", "mailFrom", "发件邮箱", "text", "邮箱", "发件人"),
							{ID: "mail-password", Key: "mailPassword", Label: "邮箱密钥", Kind: "password", Sensitive: true, Keywords: []string{"邮箱密钥", "授权码", "邮箱密码"}},
							field("mail-smtp", "mailSmtp", "SMTP 地址", "text", "smtp", "邮箱服务器"),
							{ID: "mail-test", Label: "发送测试邮件", Kind: "action", Keywords: []string{"测试邮件", "验证邮件"}},
						},
					},
				},
			},
			{
				ID:    "behavior-message",
				Title: "行为与消息",
				Groups: []*BaseSettingGroupSchema{
					{
						ID:    "behavior",
						Title: "行为控制",
						Fields: []*BaseSettingFieldSchema{
							field("trust-only-mode", "trustOnlyMode", "私骰模式", "boolean", "私骰", "信任用户"),
							field("bot-ext-free-switch", "botExtFreeSwitch", "允许自由开关", "boolean", "自由开关", "bot on", "ext on"),
							field("text-cmd-trust-only", "textCmdTrustOnly", "限制 .text 指令", "boolean", ".text", "信任限制"),
							field("ignore-unaddressed-bot", "ignoreUnaddressedBotCmd", "忽略 .bot 裸指令", "boolean", ".bot", "裸指令"),
							{ID: "player-name-wrap", Key: "playerNameWrapEnable", Label: "<玩家名> 外框", Kind: "boolean", ConfirmMessage: "不推荐：用户可能会改名为 .bot/.dismiss 等指令，并利用骰点播报让群内其他骰子刷屏，确定要关闭吗？", Keywords: []string{"玩家名外框", "名称外框", "刷屏防护"}},
						},
					},
					{
						ID:    "message",
						Title: "消息与日志",
						Fields: []*BaseSettingFieldSchema{
							field("alive-notice-enable", "aliveNoticeEnable", "存活确认", "boolean", "骰狗", "存活确认"),
							field("alive-notice-value", "aliveNoticeValue", "存活消息间隔", "text", "存活间隔", "cron"),
							field("log-size-notice-enable", "logSizeNoticeEnable", "日志记录提示", "boolean", "日志提示"),
							field("log-size-notice-count", "logSizeNoticeCount", "日志提示阈值", "number", "日志条数阈值"),
							field("only-log-group", "onlyLogCommandInGroup", "日志仅记录指令（群聊）", "boolean", "日志仅记录指令", "群聊"),
							field("only-log-private", "onlyLogCommandInPrivate", "日志仅记录指令（私聊）", "boolean", "日志仅记录指令", "私聊"),
							{ID: "command-prefix", Key: "commandPrefix", Label: "指令前缀", Kind: "string-list", Keywords: []string{"前缀", "指令前导", "命令前缀"}},
							{ID: "message-delay-range", Label: "QQ 回复延迟(秒)", Kind: "number-pair", Keys: []string{"messageDelayRangeStart", "messageDelayRangeEnd"}, Keywords: []string{"回复延迟", "消息延迟", "QQ回复"}},
						},
					},
				},
			},
			{
				ID:    "rate-limit",
				Title: "刷屏警告",
				Groups: []*BaseSettingGroupSchema{
					{
						ID:    "rate-limit-main",
						Title: "刷屏警告设置",
						Notes: []*BaseSettingNote{
							{Tone: "info", Lines: []string{
								"每群每用户独立有一个装令牌的桶，桶最多能装“上限”枚令牌。",
								"每次指令视作拿走一枚令牌；当桶里没有令牌时，将触发警告。",
								"桶会按“速率”自动补充令牌；所有更改重启后生效。",
							}},
						},
						Fields: []*BaseSettingFieldSchema{
							field("rate-limit-enabled", "rateLimitEnabled", "刷屏警告开关", "boolean", "速率限制", "刷屏警告"),
							field("personal-rate", "personalReplenishRate", "个人速率", "text", "个人速率", "每秒", "@every"),
							field("personal-burst", "personalBurst", "个人上限", "number", "个人上限"),
							field("group-rate", "groupReplenishRate", "群组速率", "text", "群组速率", "每秒", "@every"),
							field("group-burst", "groupBurst", "群组上限", "number", "群组上限"),
						},
					},
				},
			},
			{
				ID:    "access-security",
				Title: "访问与安全",
				Groups: []*BaseSettingGroupSchema{
					{
						ID:    "ui-access",
						Title: "UI 访问控制",
						Fields: []*BaseSettingFieldSchema{
							{ID: "serve-address", Key: "serveAddress", Label: "UI 界面地址", Kind: "select", AllowCustomValue: true, Options: []*BaseSettingOption{option("0.0.0.0:3211", "0.0.0.0:3211"), option("127.0.0.1:3211", "127.0.0.1:3211")}, Keywords: []string{"服务地址", "UI地址", "端口", "公网"}},
							{ID: "ui-password", Key: "uiPassword", Label: "UI 界面密码", Kind: "password", Sensitive: true, Keywords: []string{"UI密码", "后台密码", "登录密码"}},
						},
					},
					{
						ID:    "friend-group",
						Title: "好友与群组",
						Fields: []*BaseSettingFieldSchema{
							field("friend-add-comment", "friendAddComment", "加好友验证", "text", "好友验证", "加好友"),
							field("refuse-group-invite", "refuseGroupInvite", "拒绝加入新群", "boolean", "拒绝加群", "群邀请"),
						},
					},
				},
			},
			{
				ID:    "platform-special",
				Title: "平台特殊配置",
				Groups: []*BaseSettingGroupSchema{
					{
						ID:    "qq-channel",
						Title: "QQ 频道设置",
						Fields: []*BaseSettingFieldSchema{
							field("work-in-qq-channel", "workInQQChannel", "总开关", "boolean", "QQ频道", "频道消息"),
							field("qq-channel-auto-on", "QQChannelAutoOn", "自动 bot on", "boolean", "自动开启", "频道自动开启"),
							field("qq-channel-log-message", "QQChannelLogMessage", "记录消息日志", "boolean", "频道日志", "QQ频道日志"),
							field("qq-enable-poke", "QQEnablePoke", "启用戳一戳", "boolean", "戳一戳", "QQ特性"),
						},
					},
				},
			},
			{
				ID:    "game-extension",
				Title: "游戏与扩展",
				Groups: []*BaseSettingGroupSchema{
					{
						ID:    "game",
						Title: "游戏配置",
						Fields: []*BaseSettingFieldSchema{
							field("default-coc-rule-index", "defaultCocRuleIndex", "COC 默认房规", "text", "coc房规", "dg"),
							field("max-coc-card-gen", "maxCocCardGen", "COC 制卡上限", "text", "制卡上限"),
							field("max-execute-time", "maxExecuteTime", "骰点轮数上限", "text", "骰点上限", ".r n#"),
						},
					},
					{
						ID:              "ext-default-settings",
						Title:           "扩展与扩展指令",
						Collapsible:     true,
						DefaultExpanded: false,
						Fields: []*BaseSettingFieldSchema{
							field("ext-default-settings-field", "extDefaultSettings", "扩展默认设置", "ext-default-settings", "扩展", "默认指令", "自动开启"),
						},
					},
				},
			},
			{
				ID:    "maintenance-low-frequency",
				Title: "维护与低频",
				Groups: []*BaseSettingGroupSchema{
					{
						ID:          "quit-inactive",
						Title:       "自动退群",
						Collapsible: true,
						Fields: []*BaseSettingFieldSchema{
							field("quit-inactive-threshold", "quitInactiveThreshold", "自动退群阈值", "number", "不活跃退群", "多少天"),
							field("quit-inactive-batch-size", "quitInactiveBatchSize", "退群批次大小", "number", "退群批次"),
							field("quit-inactive-batch-wait", "quitInactiveBatchWait", "退群批次间隔(分)", "number", "退群间隔"),
						},
					},
					{
						ID:              "upgrade",
						Title:           "固件升级",
						Collapsible:     true,
						DefaultExpanded: false,
						Notes: []*BaseSettingNote{
							{Tone: "info", Lines: []string{
								"使用指定的压缩包对当前海豹进行覆盖，上传完成后会自动重启海豹。",
								"容器模式下禁止直接更新，请手动拉取最新镜像。",
								"尽量不要从高版本降低到低版本，数据库有可能不兼容。",
							}},
						},
						Fields: []*BaseSettingFieldSchema{
							field("upgrade-package", "", "固件升级", "upload", "升级", "固件包", "上传压缩包"),
						},
					},
				},
			},
		},
	}
}

func stringify(value interface{}) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), true
	case int:
		return strconv.Itoa(v), true
	case int64:
		return strconv.FormatInt(v, 10), true
	default:
		return "", false
	}
}

func parsePositiveInt64String(value interface{}) (int64, bool) {
	text, ok := stringify(value)
	if !ok {
		return 0, false
	}
	parsed, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
	return parsed, err == nil && parsed > 0
}

func parsePositiveInt64(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case float64:
		if vv := int64(v); vv >= 1 {
			return vv, true
		}
	case int64:
		if v >= 1 {
			return v, true
		}
	case int:
		if v >= 1 {
			return int64(v), true
		}
	case string:
		if parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil && parsed >= 1 {
			return parsed, true
		}
	}
	return 0, false
}

func parseInt(value interface{}) (int, bool) {
	switch v := value.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case int64:
		return int(v), true
	case string:
		if parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
			return int(parsed), true
		}
	}
	return 0, false
}

func parseFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case int64:
		return float64(v), true
	case int:
		return float64(v), true
	case string:
		if parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			return parsed, true
		}
	}
	return 0, false
}

func parseNonNegativeFloat64(value interface{}) (float64, bool) {
	if parsed, ok := parseFloat64(value); ok {
		if parsed < 0 {
			return 0, true
		}
		return parsed, true
	}
	return 0, false
}

func cloneBoolMap(source map[string]bool) map[string]bool {
	if len(source) == 0 {
		return map[string]bool{}
	}
	out := make(map[string]bool, len(source))
	for key, value := range source {
		out[key] = value
	}
	return out
}

func extractUpgradeForm(raw huma.MultipartFormFiles[BaseSettingUpgradeForm]) (*BaseSettingUpgradeForm, error) {
	data := raw.Data()
	if data != nil && data.File.IsSet {
		return data, nil
	}
	if raw.Form == nil || len(raw.Form.File["file"]) == 0 {
		return nil, huma.Error400BadRequest("missing file")
	}
	fh := raw.Form.File["file"][0]
	file, err := fh.Open()
	if err != nil {
		return nil, huma.Error400BadRequest("failed to open file")
	}
	return &BaseSettingUpgradeForm{
		File: huma.FormFile{
			File:        file,
			ContentType: fh.Header.Get("Content-Type"),
			IsSet:       true,
			Size:        fh.Size,
			Filename:    fh.Filename,
		},
	}, nil
}
