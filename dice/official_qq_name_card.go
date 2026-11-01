package dice

import (
	"html"
	"strings"
	"unicode/utf8"
)

const (
	// DefaultOfficialQQNameCardTemplate 是官方 QQ 回复名片的默认渲染模板。
	DefaultOfficialQQNameCardTemplate = `<font color="#FF0000" size="1">{card}</font>`
	// MaxOfficialQQNameCardTemplateRunes 限制用户自定义模板的长度。
	MaxOfficialQQNameCardTemplateRunes = 500
	officialQQNameCardMaxRunes         = 120
)

func isOfficialQQEndpoint(ep *EndPointInfo) bool {
	return ep != nil && ep.Platform == "QQ" && ep.ProtocolType == "official"
}

// evalOfficialQQNameCardText 求值当前玩家的 .sn 模板，返回单行名片内容。
// 与 buildOfficialQQNameCardLine 不同，它不检查展示开关，供日志等非展示场景复用。
func evalOfficialQQNameCardText(ctx *MsgContext) (string, error) {
	if ctx == nil || ctx.Player == nil {
		return "", nil
	}
	cardTemplate := strings.TrimSpace(ctx.Player.AutoSetNameTemplate)
	if cardTemplate == "" {
		return "", nil
	}
	card, err := EvalPlayerGroupCardTemplate(ctx, cardTemplate)
	if err != nil {
		return "", err
	}
	return strings.Join(strings.Fields(card), " "), nil
}

// officialQQLogNickname 返回官方 QQ 跑团日志中应记录的玩家名。
// 官方 QQ 没有改名接口，平台昵称不会变成 .sn 名片，因此日志改用名片内容作为用户名。
// 不满足条件时返回空字符串，调用方回退到 ctx.Player.Name。
// qqNickname 用于补全 $t玩家_RAW 等临时变量：日志钩子早于常规 SetTempVars 执行。
func officialQQLogNickname(ctx *MsgContext, qqNickname string) string {
	if ctx == nil || ctx.Player == nil {
		return ""
	}
	if !isOfficialQQEndpoint(ctx.EndPoint) || strings.TrimSpace(ctx.Player.AutoSetNameTemplate) == "" {
		return ""
	}
	// 日志钩子早于命令执行，使用独立 VM 求值，避免影响后续指令的 VM 状态。
	logCtx := ctx.ShallowCopy()
	logCtx.vm = nil
	SetTempVars(logCtx, qqNickname)
	card, err := evalOfficialQQNameCardText(logCtx)
	if err != nil {
		return ""
	}
	return truncateRunes(card, officialQQNameCardMaxRunes)
}

func officialQQNameCardLineOf(ctx *MsgContext) string {
	if ctx == nil {
		return ""
	}
	return ctx.officialQQNameCardLine
}

// stripOfficialQQNameCard 去掉仅用于展示的名片前缀，避免其进入日志等文本记录。
// 名片可能被拼在别名/代骰前缀之后，因此按首次出现位置移除。
func stripOfficialQQNameCard(line, text string) string {
	if line == "" {
		return text
	}
	return strings.Replace(text, line, "", 1)
}

// stripOfficialQQNameCardFromPages 返回去掉首页名片前缀的分页副本，用于翻页缓存。
func stripOfficialQQNameCardFromPages(line string, pages []string) []string {
	if line == "" || len(pages) == 0 {
		return pages
	}
	stripped := append([]string(nil), pages...)
	stripped[0] = stripOfficialQQNameCard(line, stripped[0])
	return stripped
}

// buildOfficialQQNameCardLine 求值当前玩家的 .sn 模板，生成官方 QQ 回复顶部的红字状态栏。
// 返回空字符串表示未启用、平台不匹配、没有模板或求值失败。
func buildOfficialQQNameCardLine(ctx *MsgContext) string {
	if ctx == nil || ctx.Dice == nil || ctx.Player == nil || ctx.EndPoint == nil {
		return ""
	}
	if !ctx.Dice.Config.OfficialQQUseNameCard || !ctx.Dice.Config.OfficialQQUseMarkdown {
		return ""
	}
	if !isOfficialQQEndpoint(ctx.EndPoint) {
		return ""
	}

	card, err := evalOfficialQQNameCardText(ctx)
	if err != nil {
		return ""
	}
	if card == "" {
		return ""
	}

	renderTemplate := strings.TrimSpace(ctx.Dice.Config.OfficialQQNameCardTmpl)
	if renderTemplate == "" || !strings.Contains(renderTemplate, "{card}") ||
		utf8.RuneCountInString(renderTemplate) > MaxOfficialQQNameCardTemplateRunes {
		renderTemplate = DefaultOfficialQQNameCardTemplate
	}
	line := strings.ReplaceAll(renderTemplate, "{card}", html.EscapeString(truncateRunes(card, officialQQNameCardMaxRunes)))
	if !strings.HasSuffix(line, "\n") {
		line += "\n"
	}
	return line
}

// withOfficialQQNameCard 在官方 QQ 的回复顶部插入当前玩家的 .sn 名片。
// 官方 QQ 没有修改群名片的接口，原版 QQ 则继续由平台在玩家名字旁显示名片。
func withOfficialQQNameCard(ctx *MsgContext, text string) string {
	return prependOfficialQQNameCard(buildOfficialQQNameCardLine(ctx), text)
}

func prependOfficialQQNameCard(line, text string) string {
	if line == "" || strings.TrimSpace(text) == "" {
		return text
	}
	return line + text
}

func truncateRunes(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "…"
}
