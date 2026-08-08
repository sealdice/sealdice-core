package dice

import (
	"fmt"
	"strings"

	randcore "sealdice-core/utils/random"
)

func formatSupportedDiceRandomModesText() string {
	lines := make([]string, 0, len(supportedDiceRandomModes))
	for _, mode := range supportedDiceRandomModes {
		spec := randcore.ModeSpecFor(mode)
		lines = append(lines, fmt.Sprintf("%s // %s", mode, spec.ShortDesc))
	}
	return strings.Join(lines, "\n")
}

func formatDiceRandomModeHelpText() string {
	return strings.Join([]string{
		"查看随机算法:",
		".randalgo // 查看当前随机算法、对应规范和简介",
		".randalgo get [面数] // 对全部随机源各掷一次并显示单次耗时",
		".randalgo set <模式> // 设置随机模式，仅Master可用",
		"支持的模式:",
		formatSupportedDiceRandomModesText(),
	}, "\n")
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
	return fmt.Sprintf("无效的骰面: %s\n请提供一个大于 0 的整数，例如 `.randalgo get 20`", raw)
}
