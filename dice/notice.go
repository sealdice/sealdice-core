package dice

import "strings"

// NoticeType 表示通知内容分类，用于按通知目标进行颗粒化过滤。
type NoticeType string

const (
	NoticeTypeGroup    NoticeType = "group"
	NoticeTypeInvite   NoticeType = "invite"
	NoticeTypeBan      NoticeType = "ban"
	NoticeTypeCensor   NoticeType = "censor"
	NoticeTypeInactive NoticeType = "inactive"
	NoticeTypeSend     NoticeType = "send"
	NoticeTypeSystem   NoticeType = "system"
)

// AllNoticeTypes 是配置和 UI 使用的稳定分类顺序。
var AllNoticeTypes = []NoticeType{
	NoticeTypeGroup,
	NoticeTypeInvite,
	NoticeTypeBan,
	NoticeTypeCensor,
	NoticeTypeInactive,
	NoticeTypeSend,
	NoticeTypeSystem,
}

var validNoticeTypes = func() map[NoticeType]struct{} {
	result := make(map[NoticeType]struct{}, len(AllNoticeTypes))
	for _, noticeType := range AllNoticeTypes {
		result[noticeType] = struct{}{}
	}
	return result
}()

// NoticeTarget 是 NoticeIDs 中单项的运行时表示。
//
// 持久化格式保持为字符串，以兼容旧配置：
//   - QQ:12345                             启用全部分类
//   - QQ:12345:disable                     禁用
//   - QQ:12345:only=group,ban              仅启用指定分类
//   - QQ:12345:disable:only=group,ban      禁用并保留分类选择
//
// 元数据仅从末尾识别，因此 OpenQQ 等内部包含多个冒号的统一 ID 仍可正常解析。
type NoticeTarget struct {
	ID                  string
	Disabled            bool
	NoticeTypes         []NoticeType
	HasNoticeTypeFilter bool
}

// ParseNoticeTarget 解析一条兼容旧格式的通知目标配置。
func ParseNoticeTarget(raw string) NoticeTarget {
	parts := strings.Split(strings.TrimSpace(raw), ":")
	target := NoticeTarget{}
	end := len(parts)

	for end > 1 {
		suffix := strings.TrimSpace(parts[end-1])
		switch {
		case suffix == "disable":
			target.Disabled = true
			end--
		case strings.HasPrefix(suffix, "only="):
			// 多个 only 后缀出现时，以最右侧（最后配置）的值为准。
			if !target.HasNoticeTypeFilter {
				target.HasNoticeTypeFilter = true
				values := strings.Split(strings.TrimPrefix(suffix, "only="), ",")
				for _, value := range values {
					noticeType := NoticeType(strings.TrimSpace(value))
					if _, ok := validNoticeTypes[noticeType]; ok {
						target.NoticeTypes = append(target.NoticeTypes, noticeType)
					}
				}
			}
			end--
		default:
			target.ID = strings.TrimSpace(strings.Join(parts[:end], ":"))
			target.NoticeTypes = normalizeNoticeTypes(target.NoticeTypes)
			return target
		}
	}

	target.ID = strings.TrimSpace(strings.Join(parts[:end], ":"))
	target.NoticeTypes = normalizeNoticeTypes(target.NoticeTypes)
	return target
}

// Allows 判断目标是否允许接收指定分类。
func (target NoticeTarget) Allows(noticeType NoticeType) bool {
	if target.Disabled || target.ID == "" {
		return false
	}
	if !target.HasNoticeTypeFilter {
		return true
	}
	for _, allowed := range target.NoticeTypes {
		if allowed == noticeType {
			return true
		}
	}
	return false
}

// String 返回通知目标的规范持久化格式。
func (target NoticeTarget) String() string {
	id := strings.TrimSpace(target.ID)
	if id == "" {
		return ""
	}

	var result strings.Builder
	result.WriteString(id)
	if target.Disabled {
		result.WriteString(":disable")
	}

	types := normalizeNoticeTypes(target.NoticeTypes)
	if target.HasNoticeTypeFilter && len(types) != len(AllNoticeTypes) {
		result.WriteString(":only=")
		for index, noticeType := range types {
			if index > 0 {
				result.WriteByte(',')
			}
			result.WriteString(string(noticeType))
		}
	}
	return result.String()
}

// Platform 返回通知 ID 对应的平台。Mail 目标也会返回 Mail，由调用方决定是否跳过。
func (target NoticeTarget) Platform() (string, bool) {
	prefix, _, ok := strings.Cut(target.ID, ":")
	if !ok || prefix == "" {
		return "", false
	}
	platform, _, _ := strings.Cut(prefix, "-")
	return platform, platform != ""
}

// MatchesEndpoint 判断通知目标是否可由指定平台和协议的 Endpoint 发送。
//
// 官方 QQ 的统一 ID 使用 OpenQQ/OpenQQCH 前缀，但 Endpoint 平台仍是 QQ，
// 因此还必须结合 protocolType 区分它与普通 QQ 连接。
func (target NoticeTarget) MatchesEndpoint(platform, protocolType string) bool {
	targetPlatform, ok := target.Platform()
	if !ok || targetPlatform == "Mail" {
		return false
	}

	switch targetPlatform {
	case "OpenQQ", "OpenQQCH":
		return platform == "QQ" && protocolType == "official"
	case "QQ":
		return platform == "QQ" && protocolType != "official"
	default:
		return targetPlatform == platform
	}
}

// IsGroup 判断目标是否为群、频道或服务器消息目标。
func (target NoticeTarget) IsGroup() bool {
	prefix, _, ok := strings.Cut(target.ID, ":")
	if !ok {
		return false
	}
	return strings.HasSuffix(prefix, "-Group") ||
		strings.HasSuffix(prefix, "-Channel") ||
		strings.HasSuffix(prefix, "-Guild")
}

func normalizeNoticeTypes(types []NoticeType) []NoticeType {
	selected := make(map[NoticeType]struct{}, len(types))
	for _, noticeType := range types {
		if _, ok := validNoticeTypes[noticeType]; ok {
			selected[noticeType] = struct{}{}
		}
	}

	result := make([]NoticeType, 0, len(selected))
	for _, noticeType := range AllNoticeTypes {
		if _, ok := selected[noticeType]; ok {
			result = append(result, noticeType)
		}
	}
	return result
}

func filterNoticeTargets(rawTargets []string, noticeType NoticeType) []NoticeTarget {
	targets := make([]NoticeTarget, 0, len(rawTargets))
	for _, raw := range rawTargets {
		target := ParseNoticeTarget(raw)
		if target.Allows(noticeType) {
			targets = append(targets, target)
		}
	}
	return targets
}
