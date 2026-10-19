package dice

import (
	"fmt"
	"strings"
	"time"

	randcore "sealdice-core/utils/random"
)

type gmEnableConfirmState struct {
	Code      string
	ExpiresAt int64
}

var gmEnableConfirmCodes SyncMap[string, *gmEnableConfirmState]

const gmEnableConfirmTTL = 5 * time.Minute

func issueGMEnableConfirmCode(userID string) string {
	code := generateFourDigitCode()
	gmEnableConfirmCodes.Store(userID, &gmEnableConfirmState{
		Code:      code,
		ExpiresAt: time.Now().Add(gmEnableConfirmTTL).Unix(),
	})
	return code
}

func consumeGMEnableConfirmCode(userID, code string) bool {
	state, ok := gmEnableConfirmCodes.Load(userID)
	if !ok || state == nil {
		return false
	}
	gmEnableConfirmCodes.Delete(userID)
	return time.Now().Unix() <= state.ExpiresAt && state.Code == code
}

func formatSupportedDiceRandomModesText() string {
	lines := make([]string, 0, len(supportedDiceRandomModes))
	for _, mode := range supportedDiceRandomModes {
		spec := randcore.ModeSpecFor(mode)
		desc := spec.ShortDesc
		switch mode {
		case randcore.ModeGM:
			desc += "（默认禁用，启用后约 +32MB 内存）"
		case randcore.ModeHybrid:
			desc += "（国密源启用后才加入）"
		default:
		}
		lines = append(lines, fmt.Sprintf("%s // %s", mode, desc))
	}
	return strings.Join(lines, "\n")
}

func formatDiceRandomModeHelpText() string {
	return strings.Join([]string{
		"查看随机算法:",
		".randalgo // 查看当前随机算法、对应规范和简介",
		".randalgo get [模式|all] [面数] // 只测当前/指定随机源，all 测试全部已启用源",
		".randalgo set <模式> // 设置随机模式，仅Master可用",
		".randalgo set gm <确认码> // 启用国密源（约 +32MB 常驻内存）",
		".randalgo set gm off // 禁用国密源（重启海豹后释放内存）",
		"",
		"国密源默认禁用。启用后需要约 32MB 常驻内存，用于采集系统熵并保证 SM3 随机数的随机性和不可预测性；",
		"这 32MB 不是骰点缓存或日志缓存，启用后会一直占用，只有重启海豹才能释放。",
		"启用后混合模式会加入国密源，NIST 模式会混入一段国密种子。",
		"支持的模式:",
		formatSupportedDiceRandomModesText(),
	}, "\n")
}

func formatDiceRandomModeGMEnableConfirmText(code string) string {
	return strings.Join([]string{
		"国密源默认禁用。启用后需要约 32MB 常驻内存，用于采集系统熵并保证 SM3 随机数的随机性和不可预测性。",
		"这 32MB 在本次运行期间不会释放，只有重启海豹才能释放。",
		"启用后混合模式会加入国密源，NIST 模式会混入国密种子。",
		"如确认启用，请在 5 分钟内执行：",
		fmt.Sprintf(".randalgo set gm %s", code),
		"如想减少占用，请执行 .randalgo set gm off 后重启海豹。",
	}, "\n")
}

func formatDiceRandomModeGMConfirmInvalidText() string {
	return "确认码无效或已过期，请重新执行 .randalgo set gm 获取新的确认码。"
}

func formatDiceRandomModeGMDisabledText() string {
	return "国密源已禁用。本次运行期间已占用的内存不会释放，重启海豹后释放。"
}

func formatDiceRandomModeGMAlreadyDisabledText() string {
	return "国密源当前已经是禁用状态。"
}

// handleGMSetCommand 处理 .randalgo set gm 的启用确认和关闭流程。
// 返回 true 表示命令已处理并回复；返回 false 表示继续执行普通模式切换。
func handleGMSetCommand(ctx *MsgContext, msg *Message, confirmArg string) bool {
	if strings.EqualFold(confirmArg, "off") {
		if !ctx.Dice.Config.DiceRandomGMEnabled && !globalRandSource.GMEnabled() {
			ReplyToSender(ctx, msg, formatDiceRandomModeGMAlreadyDisabledText())
			return true
		}
		if err := ctx.Dice.DisableDiceRandomGM(); err != nil {
			ReplyToSender(ctx, msg, formatDiceRandomModeSetUnavailableText(DiceRandomModeGM, err))
			return true
		}
		ctx.Dice.MarkModified()
		ctx.Dice.Save(false)
		ReplyToSender(ctx, msg, formatDiceRandomModeGMDisabledText())
		return true
	}
	if globalRandSource.GMEnabled() {
		return false
	}

	operatorID := ""
	if ctx.Player != nil {
		operatorID = ctx.Player.UserID
	}
	if operatorID == "" {
		operatorID = msg.Sender.UserID
	}
	if confirmArg == "" {
		code := issueGMEnableConfirmCode(operatorID)
		ReplyToSender(ctx, msg, formatDiceRandomModeGMEnableConfirmText(code))
		return true
	}
	if !consumeGMEnableConfirmCode(operatorID, confirmArg) {
		ReplyToSender(ctx, msg, formatDiceRandomModeGMConfirmInvalidText())
		return true
	}
	if err := ctx.Dice.EnableDiceRandomGM(); err != nil {
		ReplyToSender(ctx, msg, formatDiceRandomModeSetUnavailableText(DiceRandomModeGM, err))
		return true
	}
	return false
}

func formatDiceRandomModeSetSuccessText(mode DiceRandomMode) string {
	return fmt.Sprintf("已切换随机模式为 %s，使用 .randalgo 查看详情", mode)
}

func formatDiceRandomModeSetMissingModeText() string {
	return "请提供随机模式。\n支持的模式:\n" + formatSupportedDiceRandomModesText()
}

func formatDiceRandomModeSetInvalidModeText(raw string) string {
	return fmt.Sprintf("不支持的随机模式: %s\n支持的模式:\n%s", raw, formatSupportedDiceRandomModesText())
}

func formatDiceRandomModeSetUnavailableText(mode DiceRandomMode, err error) string {
	if err == nil {
		return fmt.Sprintf("随机模式 %s 当前不可用", mode)
	}
	return fmt.Sprintf("随机模式 %s 当前不可用: %v", mode, err)
}

func formatDiceRandomModeGetInvalidPointsText(raw string) string {
	return fmt.Sprintf("无效的骰面: %s\n请提供一个大于 0 的整数，例如 `.randalgo get 20` 或 `.randalgo get gm 20`", raw)
}
