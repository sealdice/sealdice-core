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

	cardTemplate := strings.TrimSpace(ctx.Player.AutoSetNameTemplate)
	if cardTemplate == "" {
		return ""
	}

	card, err := EvalPlayerGroupCardTemplate(ctx, cardTemplate)
	if err != nil {
		return ""
	}
	card = strings.Join(strings.Fields(card), " ")
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
	line := buildOfficialQQNameCardLine(ctx)
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
