package basesetting

import (
	"context"
	"io"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/danielgtaylor/huma/v2"
	"github.com/robfig/cron/v3"
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
	mu   sync.RWMutex
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

func (s *Service) GetValue(_ context.Context, _ *request.Empty) (*response.ItemResponse[BaseSettingValueResp], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return response.NewItemResponse(s.buildValue()), nil
}

func (s *Service) SetValue(_ context.Context, req *BaseSettingUpdateReq) (*response.ItemResponse[BaseSettingActionResp], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
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
		CommandPrefix:            append([]string{}, s.dice.CommandPrefix...),
		DiceMasters:              append([]string{}, s.dice.DiceMasters...),
		NoticeIds:                append([]string{}, s.dice.Config.NoticeIDs...),
		MasterUnlockCode:         s.dice.MasterUnlockCode,
		UIPassword:               password,
		MailEnable:               s.dice.Config.MailEnable,
		MailFrom:                 s.dice.Config.MailFrom,
		MailPassword:             emailPassword,
		MailSmtp:                 s.dice.Config.MailSMTP,
		TrustOnlyMode:            s.dice.Config.TrustOnlyMode,
		BotExtFreeSwitch:         s.dice.Config.BotExtFreeSwitch,
		QQEnablePoke:             s.dice.Config.QQEnablePoke,
		OfficialQQFileSendBase64: s.dice.Config.OfficialQQFileSendBase64,
		OfficialQQUseMarkdown:    s.dice.Config.OfficialQQUseMarkdown,
		OfficialQQUseNameCard:    s.dice.Config.OfficialQQUseNameCard && s.dice.Config.OfficialQQUseMarkdown,
		OfficialQQNameCardTmpl:   s.dice.Config.OfficialQQNameCardTmpl,
		TextCmdTrustOnly:         s.dice.Config.TextCmdTrustOnly,
		IgnoreUnaddressedBotCmd:  s.dice.Config.IgnoreUnaddressedBotCmd,
		AliveNoticeEnable:        s.dice.Config.AliveNoticeEnable,
		AliveNoticeValue:         s.dice.Config.AliveNoticeValue,
		LogSizeNoticeEnable:      s.dice.Config.LogSizeNoticeEnable,
		LogSizeNoticeCount:       s.dice.Config.LogSizeNoticeCount,
		PlayerNameWrapEnable:     s.dice.Config.PlayerNameWrapEnable,
		DiceRandomMode:           string(dice.NormalizeDiceRandomMode(s.dice.Config.DiceRandomMode)),
		DiceRandomGMEnabled:      s.dice.Config.DiceRandomGMEnabled,
		OnlyLogCommandInGroup:    s.dice.Config.OnlyLogCommandInGroup,
		OnlyLogCommandInPrivate:  s.dice.Config.OnlyLogCommandInPrivate,
		RateLimitEnabled:         s.dice.Config.RateLimitEnabled,
		PersonalReplenishRate:    s.dice.Config.PersonalReplenishRateStr,
		PersonalBurst:            s.dice.Config.PersonalBurst,
		GroupReplenishRate:       s.dice.Config.GroupReplenishRateStr,
		GroupBurst:               s.dice.Config.GroupBurst,
		ServeAddress:             s.dice.Parent.ServeAddress,
		TrayTooltip:              s.dice.Parent.GetTrayTooltip(),
		RefuseGroupInvite:        s.dice.Config.RefuseGroupInvite,
		FriendAddComment:         s.dice.Config.FriendAddComment,
		WorkInQQChannel:          s.dice.Config.WorkInQQChannel,
		QQChannelAutoOn:          s.dice.Config.QQChannelAutoOn,
		QQChannelLogMessage:      s.dice.Config.QQChannelLogMessage,
		DefaultCocRuleIndex:      cocRule,
		MaxCocCardGen:            maxCard,
		MaxExecuteTime:           maxExec,
		MessageDelayRangeStart:   s.dice.Config.MessageDelayRangeStart,
		MessageDelayRangeEnd:     s.dice.Config.MessageDelayRangeEnd,
		QuitInactiveThreshold:    s.dice.Config.QuitInactiveThreshold.Hours() / 24,
		QuitInactiveBatchSize:    s.dice.Config.QuitInactiveBatchSize,
		QuitInactiveBatchWait:    s.dice.Config.QuitInactiveBatchWait,
		ExtDefaultSettings:       extDefaultSettings,
	}
}

// applyPatch prepares and validates the entire candidate before publishing any
// setting or running callbacks. Persistence keeps the existing DiceManager policy.
func (s *Service) applyPatch(patch BaseSettingPatch) error {
	config := s.dice.Config
	setIfProvided(&config.MailEnable, patch.MailEnable)
	setIfProvided(&config.MailFrom, patch.MailFrom)
	setIfProvided(&config.MailSMTP, patch.MailSmtp)
	setIfProvided(&config.TrustOnlyMode, patch.TrustOnlyMode)
	setIfProvided(&config.BotExtFreeSwitch, patch.BotExtFreeSwitch)
	setIfProvided(&config.QQEnablePoke, patch.QQEnablePoke)
	setIfProvided(&config.OfficialQQFileSendBase64, patch.OfficialQQFileSendBase64)
	setIfProvided(&config.OfficialQQUseMarkdown, patch.OfficialQQUseMarkdown)
	setIfProvided(&config.OfficialQQUseNameCard, patch.OfficialQQUseNameCard)
	if patch.OfficialQQNameCardTmpl != nil {
		value := strings.TrimSpace(*patch.OfficialQQNameCardTmpl)
		if utf8.RuneCountInString(value) > dice.MaxOfficialQQNameCardTemplateRunes {
			return huma.Error400BadRequest("名片渲染模板不能超过" + strconv.Itoa(dice.MaxOfficialQQNameCardTemplateRunes) + "个字符")
		}
		config.OfficialQQNameCardTmpl = value
	}
	if config.OfficialQQUseNameCard && !config.OfficialQQUseMarkdown {
		if patch.OfficialQQUseNameCard != nil && *patch.OfficialQQUseNameCard {
			return huma.Error400BadRequest("启用回复名片前必须启用 Markdown")
		}
		config.OfficialQQUseNameCard = false
	}
	setIfProvided(&config.TextCmdTrustOnly, patch.TextCmdTrustOnly)
	setIfProvided(&config.IgnoreUnaddressedBotCmd, patch.IgnoreUnaddressedBotCmd)
	setIfProvided(&config.AliveNoticeEnable, patch.AliveNoticeEnable)
	setIfProvided(&config.AliveNoticeValue, patch.AliveNoticeValue)
	setIfProvided(&config.LogSizeNoticeEnable, patch.LogSizeNoticeEnable)
	setIfProvided(&config.LogSizeNoticeCount, patch.LogSizeNoticeCount)
	setIfProvided(&config.PlayerNameWrapEnable, patch.PlayerNameWrapEnable)
	setIfProvided(&config.OnlyLogCommandInGroup, patch.OnlyLogCommandInGroup)
	setIfProvided(&config.OnlyLogCommandInPrivate, patch.OnlyLogCommandInPrivate)
	setIfProvided(&config.RateLimitEnabled, patch.RateLimitEnabled)
	setIfProvided(&config.RefuseGroupInvite, patch.RefuseGroupInvite)
	setIfProvided(&config.WorkInQQChannel, patch.WorkInQQChannel)
	setIfProvided(&config.QQChannelAutoOn, patch.QQChannelAutoOn)
	setIfProvided(&config.QQChannelLogMessage, patch.QQChannelLogMessage)
	setIfProvided(&config.MessageDelayRangeStart, patch.MessageDelayRangeStart)
	setIfProvided(&config.MessageDelayRangeEnd, patch.MessageDelayRangeEnd)
	for _, field := range []struct {
		name   string
		value  *string
		target *int64
	}{
		{"maxExecuteTime", patch.MaxExecuteTime, &config.MaxExecuteTime},
		{"maxCocCardGen", patch.MaxCocCardGen, &config.MaxCocCardGen},
	} {
		if field.value != nil {
			n, err := strconv.ParseInt(strings.TrimSpace(*field.value), 10, 64)
			if err != nil || n <= 0 {
				return huma.Error400BadRequest("字段必须为正整数: " + field.name)
			}
			*field.target = n
		}
	}
	for _, field := range []struct {
		name   string
		value  *int64
		target *int64
	}{
		{"personalBurst", patch.PersonalBurst, &config.PersonalBurst},
		{"groupBurst", patch.GroupBurst, &config.GroupBurst},
		{"quitInactiveBatchSize", patch.QuitInactiveBatchSize, &config.QuitInactiveBatchSize},
		{"quitInactiveBatchWait", patch.QuitInactiveBatchWait, &config.QuitInactiveBatchWait},
	} {
		if field.value != nil {
			if *field.value <= 0 {
				return huma.Error400BadRequest("字段必须为正整数: " + field.name)
			}
			*field.target = *field.value
		}
	}
	for _, field := range []struct {
		name   string
		value  *string
		text   *string
		target *rate.Limit
	}{
		{"personalReplenishRate", patch.PersonalReplenishRate, &config.PersonalReplenishRateStr, &config.PersonalReplenishRate},
		{"groupReplenishRate", patch.GroupReplenishRate, &config.GroupReplenishRateStr, &config.GroupReplenishRate},
	} {
		if field.value != nil {
			value := strings.TrimSpace(*field.value)
			parsed, err := utils.ParseRate(value)
			if err != nil || parsed <= 0 {
				return huma.Error400BadRequest("字段格式不正确: " + field.name)
			}
			*field.text, *field.target = value, parsed
		}
	}
	if patch.DefaultCocRuleIndex != nil {
		value := strings.TrimSpace(*patch.DefaultCocRuleIndex)
		if strings.EqualFold(value, "dg") {
			config.DefaultCocRuleIndex = 11
		} else {
			n, err := strconv.ParseInt(value, 10, 64)
			if err != nil || n < 0 || n > 5 {
				return huma.Error400BadRequest("defaultCocRuleIndex 取值范围不正确")
			}
			config.DefaultCocRuleIndex = n
		}
	}
	if patch.AliveNoticeValue != nil {
		if _, err := cron.ParseStandard(config.AliveNoticeValue); err != nil {
			return huma.Error400BadRequest("存活消息间隔格式不正确")
		}
	}
	if patch.MessageDelayRangeStart != nil || patch.MessageDelayRangeEnd != nil {
		start, end := config.MessageDelayRangeStart, config.MessageDelayRangeEnd
		if math.IsNaN(start) || math.IsNaN(end) || math.IsInf(start, 0) || math.IsInf(end, 0) || start < 0 || end < 0 {
			return huma.Error400BadRequest("消息延迟必须为非负数")
		}
		// Preserve the single-start-field update behavior, while validating a
		// supplied pair against its new values rather than the old start.
		if patch.MessageDelayRangeEnd == nil && start > end {
			config.MessageDelayRangeEnd = start
		} else if start > end {
			return huma.Error400BadRequest("messageDelayRangeEnd 不能小于 messageDelayRangeStart")
		}
	}
	if patch.QuitInactiveThreshold != nil {
		days := *patch.QuitInactiveThreshold
		if math.IsNaN(days) || math.IsInf(days, 0) || days < 0 || days >= float64(math.MaxInt64)/float64(24*time.Hour) {
			return huma.Error400BadRequest("自动退群阈值超出范围")
		}
		config.QuitInactiveThreshold = time.Duration(days * float64(24*time.Hour))
	}
	trayTooltip := ""
	if patch.TrayTooltip != nil {
		trayTooltip = strings.TrimSpace(*patch.TrayTooltip)
		if utf8.RuneCountInString(trayTooltip) > dice.MaxTrayTooltipPrefixLength {
			return huma.Error400BadRequest("托盘提示文本不能超过" + strconv.Itoa(dice.MaxTrayTooltipPrefixLength) + "个字符")
		}
	}
	if patch.ExtDefaultSettings != nil {
		config.ExtDefaultSettings = make([]*dice.ExtDefaultSettingItem, 0, len(*patch.ExtDefaultSettings))
		seen := map[string]bool{}
		for _, item := range *patch.ExtDefaultSettings {
			if item == nil || strings.TrimSpace(item.Name) == "" || seen[item.Name] {
				return huma.Error400BadRequest("扩展默认设置必须包含不重复的扩展名")
			}
			seen[item.Name] = true
			config.ExtDefaultSettings = append(config.ExtDefaultSettings, &dice.ExtDefaultSettingItem{
				Name: item.Name, AutoActive: item.AutoActive, DisabledCommand: cloneBoolMap(item.DisabledCommand),
			})
		}
	}
	if patch.NoticeIds != nil {
		config.NoticeIDs = make([]string, 0, len(*patch.NoticeIds))
		for _, value := range *patch.NoticeIds {
			if target := dice.ParseNoticeTarget(value); target.ID != "" {
				config.NoticeIDs = append(config.NoticeIDs, target.String())
			}
		}
	}
	if patch.LogSizeNoticeCount != nil && config.LogSizeNoticeCount == 0 {
		config.LogSizeNoticeCount = dice.DefaultConfig.LogSizeNoticeCount
	}
	if patch.FriendAddComment != nil {
		config.FriendAddComment = strings.TrimSpace(*patch.FriendAddComment)
	}
	if patch.MailPassword != nil && *patch.MailPassword != "" && *patch.MailPassword != mailPasswordMasked {
		config.MailPassword = *patch.MailPassword
	}
	gmEnabled := config.DiceRandomGMEnabled
	if patch.DiceRandomGMEnabled != nil {
		if !*patch.DiceRandomGMEnabled {
			return huma.Error400BadRequest("关闭国密源请执行 .randalgo set gm off 后重启海豹")
		}
		if !gmEnabled {
			if err := s.dice.EnableDiceRandomGM(); err != nil {
				return huma.Error400BadRequest("启用国密源失败: " + err.Error())
			}
			gmEnabled = true
			config.DiceRandomGMEnabled = true
		}
	}
	if patch.DiceRandomMode != nil {
		mode := dice.NormalizeDiceRandomMode(*patch.DiceRandomMode)
		if mode == dice.DiceRandomModeGM && !gmEnabled {
			return huma.Error400BadRequest("国密源默认禁用。启用后需要约 32MB 常驻内存，用于采集系统熵并保证 SM3 随机数的随机性和不可预测性。请在界面上确认启用国密源。")
		}
		config.DiceRandomMode = string(mode)
	}

	s.dice.Config = config
	if patch.CommandPrefix != nil {
		s.dice.CommandPrefix = nonEmptyStrings(*patch.CommandPrefix)
	}
	if patch.DiceMasters != nil {
		masters := nonEmptyStrings(*patch.DiceMasters)
		for i, item := range masters {
			item = strings.ReplaceAll(strings.ReplaceAll(item, "qq：", "QQ:"), "QQ：", "QQ:")
			if _, err := strconv.Atoi(item); err == nil {
				item = "QQ:" + item
			}
			masters[i] = item
		}
		s.dice.DiceMasters = masters
	}
	if patch.TrayTooltip != nil {
		s.dm.SetTrayTooltip(trayTooltip)
	}
	setIfProvided(&s.dm.ServeAddress, patch.ServeAddress)
	if patch.UIPassword != nil && *patch.UIPassword != "" && *patch.UIPassword != uiPasswordMasked {
		s.dm.UIPasswordHash = *patch.UIPassword
		s.dm.AccessTokens = dice.SyncMap[string, bool]{}
	}
	if patch.DiceRandomMode != nil || patch.DiceRandomGMEnabled != nil {
		if err := s.dice.ActivateDiceRandomMode(); err != nil && s.dice.Logger != nil {
			s.dice.Logger.Warnf("[随机源] 应用管理界面随机模式失败，已使用 PCG 回退: %v", err)
		}
	}
	if patch.AliveNoticeEnable != nil || patch.AliveNoticeValue != nil {
		s.dice.ApplyAliveNotice()
	}
	if patch.ExtDefaultSettings != nil {
		s.dice.ApplyExtDefaultSettings()
	}
	if patch.QuitInactiveThreshold != nil {
		s.dice.ResetQuitInactiveCron()
	}
	return nil
}

func setIfProvided[T any](target *T, value *T) {
	if value != nil {
		*target = *value
	}
}

func nonEmptyStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			result = append(result, value)
		}
	}
	return result
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
