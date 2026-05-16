package dice

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	ds "github.com/sealdice/dicescript"

	"sealdice-core/dice/service"
	"sealdice-core/utils/satori"
)

// 哨兵错误
var errMissingCharIdOrName = errors.New("missing id or name")

type sealChatCharacterTarget struct {
	FormattedGroupID string
	FormattedUserID  string
	Attrs            *AttributesItem
	Ctx              *MsgContext
	Template         *GameSystemTemplate
}

type sealChatCharacterOutputEntry struct {
	Value    any
	Priority int
}

type sealChatCharacterAttrSnapshot struct {
	Key   string
	Value *ds.VMValue
}

type sealChatCharacterAttrCandidate struct {
	RawKey string
	Value  *ds.VMValue
}

const sealChatCharacterValueMaxDepth = 32

func ensureSealChatCharacterAttrs(target *sealChatCharacterTarget) error {
	if target == nil {
		return errors.New("missing character target")
	}
	if target.Attrs != nil {
		return nil
	}
	return errors.New("missing character attrs")
}

func (pa *PlatformAdapterSealChat) resolveSealChatCharacterTarget(d *Dice, groupID string, userID string) (*sealChatCharacterTarget, error) {
	formattedGroupID := FormatDiceIDSealChatGroup(groupID)
	formattedUserID := FormatDiceIDSealChat(userID)

	attrs, err := d.AttrsManager.Load(formattedGroupID, formattedUserID)
	if err != nil {
		return nil, err
	}
	if attrs == nil {
		id := fmt.Sprintf("%s-%s", formattedGroupID, formattedUserID)
		d.AttrsManager.m.Delete(id)
		attrs, err = d.AttrsManager.LoadById(id)
		if err != nil {
			return nil, err
		}
	}

	msg := &Message{
		MessageType: "group",
		GroupID:     formattedGroupID,
		Platform:    "SEALCHAT",
		Sender: SenderBase{
			UserID:   formattedUserID,
			Nickname: "SealChat User",
		},
	}
	if attrs != nil && strings.TrimSpace(attrs.Name) != "" {
		msg.Sender.Nickname = attrs.Name
	}

	ctx := CreateTempCtx(pa.EndPoint, msg)
	tmpl := ctx.Group.GetCharTemplate(d)
	if attrs != nil && strings.TrimSpace(attrs.SheetType) != "" {
		if cardTmpl, ok := d.GameSystemMap.Load(attrs.SheetType); ok && cardTmpl != nil {
			tmpl = cardTmpl
		}
	}
	ctx.SystemTemplate = tmpl

	return &sealChatCharacterTarget{
		FormattedGroupID: formattedGroupID,
		FormattedUserID:  formattedUserID,
		Attrs:            attrs,
		Ctx:              ctx,
		Template:         tmpl,
	}, nil
}

func sealChatLoadAliasMapValue(aliasMap *SyncMap[string, string], key string) (canonical string, ok bool) {
	if aliasMap == nil {
		return "", false
	}

	defer func() {
		if recover() != nil {
			canonical = ""
			ok = false
		}
	}()

	rawValue, exists := aliasMap.m.Load(key)
	if !exists {
		return "", false
	}

	canonical, ok = rawValue.(string)
	return canonical, ok
}

func sealChatLookupCanonicalAttrKey(tmpl *GameSystemTemplate, key string) (string, bool) {
	if tmpl == nil || tmpl.AliasMap == nil {
		return key, false
	}

	normalized := strings.ToLower(strings.TrimSpace(key))
	if normalized == "" {
		return key, false
	}
	if value, ok := sealChatLoadAliasMapValue(tmpl.AliasMap, normalized); ok {
		return value, true
	}

	normalized = chsS2T.Read(normalized)
	if value, ok := sealChatLoadAliasMapValue(tmpl.AliasMap, normalized); ok {
		return value, true
	}
	return key, false
}

func splitSealChatDisplayLabelKey(key string) (string, string, bool) {
	for _, pair := range [][2]string{{"（", "）"}, {"(", ")"}} {
		openIdx := strings.Index(key, pair[0])
		if openIdx <= 0 {
			continue
		}
		closeIdx := strings.LastIndex(key, pair[1])
		if closeIdx <= openIdx+len(pair[0]) || closeIdx != len(key)-len(pair[1]) {
			continue
		}

		left := strings.TrimSpace(key[:openIdx])
		right := strings.TrimSpace(key[openIdx+len(pair[0]) : closeIdx])
		if left == "" || right == "" {
			continue
		}
		return left, right, true
	}

	return "", "", false
}

func resolveSealChatCharacterAttrKey(key string, tmpl *GameSystemTemplate) (string, error) {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" || tmpl == nil {
		return trimmed, nil
	}

	if canonical, ok := sealChatLookupCanonicalAttrKey(tmpl, trimmed); ok {
		return canonical, nil
	}

	left, right, ok := splitSealChatDisplayLabelKey(trimmed)
	if !ok {
		return trimmed, nil
	}

	leftCanonical, leftOK := sealChatLookupCanonicalAttrKey(tmpl, left)
	rightCanonical, rightOK := sealChatLookupCanonicalAttrKey(tmpl, right)

	switch {
	case leftOK && rightOK && leftCanonical == rightCanonical:
		return leftCanonical, nil
	case rightOK && !leftOK:
		return rightCanonical, nil
	case leftOK && !rightOK:
		return leftCanonical, nil
	case leftOK && rightOK && leftCanonical != rightCanonical:
		return "", fmt.Errorf("ambiguous attribute display label %q", trimmed)
	default:
		return trimmed, nil
	}
}

func findSealChatExistingCanonicalAttrValue(attrs *AttributesItem, canonicalKey string, tmpl *GameSystemTemplate) (*ds.VMValue, bool) {
	if attrs == nil {
		return nil, false
	}
	if value, exists := attrs.LoadX(canonicalKey); exists {
		return value, true
	}

	var matchedValue *ds.VMValue
	matched := false
	consistent := true
	attrs.Range(func(key string, value *ds.VMValue) bool {
		resolvedKey, err := resolveSealChatCharacterAttrKey(key, tmpl)
		if err != nil || resolvedKey != canonicalKey || key == canonicalKey {
			return true
		}
		if !matched {
			matchedValue = value
			matched = true
			return true
		}
		if !ds.ValueEqual(matchedValue, value, false) {
			consistent = false
			return false
		}
		return true
	})

	if matched && consistent {
		return matchedValue, true
	}
	return nil, false
}

func normalizeSealChatCharacterAttrs(attrsData map[string]any, tmpl *GameSystemTemplate, existingAttrs *AttributesItem) (map[string]*ds.VMValue, error) {
	grouped := make(map[string][]sealChatCharacterAttrCandidate, len(attrsData))
	for key, rawValue := range attrsData {
		if strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("invalid attribute key %q", key)
		}

		canonicalKey, err := resolveSealChatCharacterAttrKey(key, tmpl)
		if err != nil {
			return nil, err
		}
		if canonicalKey == "" {
			return nil, fmt.Errorf("invalid attribute key %q", key)
		}

		value, err := anyToVMValue(rawValue)
		if err != nil {
			return nil, fmt.Errorf("invalid value for attribute %q: %w", key, err)
		}

		grouped[canonicalKey] = append(grouped[canonicalKey], sealChatCharacterAttrCandidate{
			RawKey: key,
			Value:  value,
		})
	}

	normalized := make(map[string]*ds.VMValue, len(grouped))
	for canonicalKey, candidates := range grouped {
		uniqueCandidates := make([]sealChatCharacterAttrCandidate, 0, len(candidates))
		for _, candidate := range candidates {
			duplicated := false
			for _, existing := range uniqueCandidates {
				if ds.ValueEqual(existing.Value, candidate.Value, false) {
					duplicated = true
					break
				}
			}
			if !duplicated {
				uniqueCandidates = append(uniqueCandidates, candidate)
			}
		}

		if len(uniqueCandidates) == 1 {
			normalized[canonicalKey] = uniqueCandidates[0].Value
			continue
		}

		if existingValue, exists := findSealChatExistingCanonicalAttrValue(existingAttrs, canonicalKey, tmpl); exists {
			changedCandidates := make([]sealChatCharacterAttrCandidate, 0, len(uniqueCandidates))
			for _, candidate := range uniqueCandidates {
				if ds.ValueEqual(existingValue, candidate.Value, false) {
					continue
				}
				changedCandidates = append(changedCandidates, candidate)
			}
			if len(changedCandidates) == 1 {
				normalized[canonicalKey] = changedCandidates[0].Value
				continue
			}
			if len(changedCandidates) == 0 {
				normalized[canonicalKey] = uniqueCandidates[0].Value
				continue
			}
		}

		return nil, fmt.Errorf("conflicting values for attribute %q", canonicalKey)
	}
	return normalized, nil
}

func cleanupSealChatCharacterAliasKeys(attrs *AttributesItem, tmpl *GameSystemTemplate, normalizedAttrs map[string]*ds.VMValue) {
	if attrs == nil || tmpl == nil || len(normalizedAttrs) == 0 {
		return
	}

	toDelete := make([]string, 0)
	attrs.Range(func(key string, value *ds.VMValue) bool {
		canonicalKey, err := resolveSealChatCharacterAttrKey(key, tmpl)
		if err != nil || canonicalKey == "" || canonicalKey == key {
			return true
		}
		if _, exists := normalizedAttrs[canonicalKey]; exists {
			toDelete = append(toDelete, key)
		}
		return true
	})

	for _, key := range toDelete {
		attrs.Delete(key)
	}
}

func (pa *PlatformAdapterSealChat) setSealChatCharacterAttrs(d *Dice, groupID string, userID string, attrsData map[string]any) (*sealChatCharacterTarget, error) {
	target, err := pa.resolveSealChatCharacterTarget(d, groupID, userID)
	if err != nil {
		return nil, err
	}
	if ensureErr := ensureSealChatCharacterAttrs(target); ensureErr != nil {
		return nil, ensureErr
	}

	normalizedAttrs, err := normalizeSealChatCharacterAttrs(attrsData, target.Template, target.Attrs)
	if err != nil {
		return nil, err
	}

	for key, value := range normalizedAttrs {
		target.Attrs.Store(key, value)
	}
	cleanupSealChatCharacterAliasKeys(target.Attrs, target.Template, normalizedAttrs)

	return target, nil
}

func snapshotSealChatCharacterAttrs(attrs *AttributesItem, tmpl *GameSystemTemplate) ([]sealChatCharacterAttrSnapshot, map[string]struct{}) {
	if attrs == nil {
		return nil, nil
	}

	snapshots := make([]sealChatCharacterAttrSnapshot, 0)
	canonicalKeys := make(map[string]struct{})
	attrs.Range(func(key string, value *ds.VMValue) bool {
		snapshots = append(snapshots, sealChatCharacterAttrSnapshot{
			Key:   key,
			Value: value,
		})
		if canonicalKey, err := resolveSealChatCharacterAttrKey(key, tmpl); err == nil && canonicalKey == key && canonicalKey != "" {
			canonicalKeys[canonicalKey] = struct{}{}
		}
		return true
	})
	return snapshots, canonicalKeys
}

func buildSealChatCharacterOutputAttrs(target *sealChatCharacterTarget) map[string]any {
	attrsData := make(map[string]any)
	if target == nil || target.Attrs == nil {
		return attrsData
	}

	if target.Template != nil && target.Template.GameSystemTemplateV2 != nil && target.Ctx != nil {
		target.Ctx.syncAttrsForTemplate(target.Attrs, target.Template.GameSystemTemplateV2)
	}

	snapshots, canonicalKeys := snapshotSealChatCharacterAttrs(target.Attrs, target.Template)

	entries := make(map[string]sealChatCharacterOutputEntry)
	for _, snapshot := range snapshots {
		key := snapshot.Key
		value := snapshot.Value
		outputKey := key
		priority := 0

		if canonicalKey, err := resolveSealChatCharacterAttrKey(key, target.Template); err == nil && canonicalKey != "" {
			if canonicalKey == key {
				outputKey = canonicalKey
				priority = 2
			} else if _, exists := canonicalKeys[canonicalKey]; exists {
				outputKey = canonicalKey
				priority = 1
			}
		}

		if existing, exists := entries[outputKey]; exists {
			if existing.Priority > priority {
				continue
			}
			if existing.Priority == priority && existing.Value != nil && value != nil {
				existingValue, err := anyToVMValue(existing.Value)
				if err == nil && ds.ValueEqual(existingValue, value, false) {
					continue
				}
			}
		}

		entries[outputKey] = sealChatCharacterOutputEntry{
			Value:    vmValueToAny(value),
			Priority: priority,
		}
	}

	for key, entry := range entries {
		attrsData[key] = entry.Value
	}
	return attrsData
}

// handleCharacterGet 处理获取角色卡请求
func (pa *PlatformAdapterSealChat) handleCharacterGet(msg satori.ScApiMsgPayload, d *Dice) {
	dataMap, ok := msg.Data.(map[string]any)
	if !ok {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "invalid data format"})
		return
	}

	groupID, _ := dataMap["group_id"].(string)
	userID, _ := dataMap["user_id"].(string)

	if groupID == "" || userID == "" {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "missing group_id or user_id"})
		return
	}

	target, err := pa.resolveSealChatCharacterTarget(d, groupID, userID)
	if err != nil {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	attrs := target.Attrs
	attrsData := buildSealChatCharacterOutputAttrs(target)

	pa.sendApiResponse(msg.Echo, map[string]any{
		"ok":   true,
		"data": attrsData,
		"name": attrs.Name,
		"type": attrs.SheetType,
	})
}

// vmValueToAny 将 VMValue 转换为可 JSON 序列化的值
func vmValueToAny(v *ds.VMValue) any {
	if v == nil {
		return nil
	}

	value, err := vmValueToAnyDepth(v, 0)
	if err != nil {
		return v.ToString()
	}
	return value
}

func vmValueToAnyDepth(v *ds.VMValue, depth int) (any, error) {
	if depth >= sealChatCharacterValueMaxDepth {
		return nil, fmt.Errorf("attribute value exceeds max nesting depth %d", sealChatCharacterValueMaxDepth)
	}

	switch v.TypeId {
	case ds.VMTypeInt:
		return v.MustReadInt(), nil
	case ds.VMTypeFloat:
		return v.MustReadFloat(), nil
	case ds.VMTypeString:
		s, _ := v.ReadString()
		return s, nil
	case ds.VMTypeArray:
		arrayData := v.MustReadArray()
		result := make([]any, 0, len(arrayData.List))
		for _, item := range arrayData.List {
			if item == nil {
				result = append(result, nil)
				continue
			}
			converted, err := vmValueToAnyDepth(item, depth+1)
			if err != nil {
				return nil, err
			}
			result = append(result, converted)
		}
		return result, nil
	case ds.VMTypeDict:
		dictData := v.MustReadDictData()
		result := make(map[string]any)
		dictData.Dict.Range(func(key string, value *ds.VMValue) bool {
			if value == nil {
				result[key] = nil
				return true
			}
			converted, err := vmValueToAnyDepth(value, depth+1)
			if err != nil {
				result = nil
				return false
			}
			result[key] = converted
			return true
		})
		if result == nil {
			return nil, fmt.Errorf("attribute value exceeds max nesting depth %d", sealChatCharacterValueMaxDepth)
		}
		return result, nil
	default:
		// 对于复杂类型，尝试转换为字符串
		return v.ToString(), nil
	}
}

func parseSealChatComputedString(value string) (*ds.VMValue, bool, error) {
	trimmed := strings.TrimSpace(value)
	if !strings.HasPrefix(trimmed, "&(") || !strings.HasSuffix(trimmed, ")") {
		return nil, false, nil
	}

	expr := strings.TrimSpace(trimmed[2 : len(trimmed)-1])
	if expr == "" {
		return nil, true, errors.New("empty computed expression")
	}
	return ds.NewComputedVal(expr), true, nil
}

func sealChatTypedEnvelopeType(v any) (ds.VMValueType, bool) {
	switch val := v.(type) {
	case float64:
		if val != float64(int64(val)) {
			return 0, false
		}
		return ds.VMValueType(int64(val)), true
	case int:
		return ds.VMValueType(val), true
	case int64:
		return ds.VMValueType(val), true
	default:
		return 0, false
	}
}

func parseSealChatTypedVMValue(v any) (*ds.VMValue, bool, error) {
	rawMap, ok := v.(map[string]any)
	if !ok {
		return nil, false, nil
	}
	rawType, exists := rawMap["t"]
	if !exists {
		return nil, false, nil
	}
	typeID, ok := sealChatTypedEnvelopeType(rawType)
	if !ok || typeID != ds.VMTypeComputedValue {
		return nil, false, nil
	}
	if _, exists := rawMap["v"]; !exists {
		return nil, true, errors.New("missing computed value payload")
	}

	data, err := json.Marshal(rawMap)
	if err != nil {
		return nil, true, err
	}
	value, err := ds.VMValueFromJSON(data)
	if err != nil {
		return nil, true, err
	}
	computed, ok := value.ReadComputed()
	if !ok || strings.TrimSpace(computed.Expr) == "" {
		return nil, true, errors.New("empty computed expression")
	}
	return value, true, nil
}

// anyToVMValue 将 any 类型转换为 VMValue
func anyToVMValue(v any) (*ds.VMValue, error) {
	return anyToVMValueDepth(v, 0)
}

func anyToVMValueDepth(v any, depth int) (*ds.VMValue, error) {
	if depth >= sealChatCharacterValueMaxDepth {
		return nil, fmt.Errorf("attribute value exceeds max nesting depth %d", sealChatCharacterValueMaxDepth)
	}

	if typedValue, matched, err := parseSealChatTypedVMValue(v); matched || err != nil {
		return typedValue, err
	}

	switch val := v.(type) {
	case float64:
		// JSON 解析时数字默认为 float64
		if val == float64(int64(val)) {
			return ds.NewIntVal(ds.IntType(val)), nil
		}
		return ds.NewFloatVal(val), nil
	case int:
		return ds.NewIntVal(ds.IntType(val)), nil
	case int64:
		return ds.NewIntVal(ds.IntType(val)), nil
	case string:
		if computedValue, matched, err := parseSealChatComputedString(val); matched || err != nil {
			return computedValue, err
		}
		return ds.NewStrVal(val), nil
	case []any:
		items := make([]*ds.VMValue, 0, len(val))
		for _, item := range val {
			converted, err := anyToVMValueDepth(item, depth+1)
			if err != nil {
				return nil, err
			}
			items = append(items, converted)
		}
		return ds.NewArrayValRaw(items), nil
	case map[string]any:
		dict := &ds.ValueMap{}
		for key, item := range val {
			converted, err := anyToVMValueDepth(item, depth+1)
			if err != nil {
				return nil, err
			}
			dict.Store(key, converted)
		}
		return ds.NewDictVal(dict).V(), nil
	case bool:
		if val {
			return ds.NewIntVal(1), nil
		}
		return ds.NewIntVal(0), nil
	default:
		return ds.NewStrVal(fmt.Sprintf("%v", v)), nil
	}
}

// handleCharacterSet 处理写入角色卡请求
// 安全限制：仅允许通过此API写入SealChat平台的角色卡
func (pa *PlatformAdapterSealChat) handleCharacterSet(msg satori.ScApiMsgPayload, d *Dice) {
	dataMap, ok := msg.Data.(map[string]any)
	if !ok {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "invalid data format"})
		return
	}

	groupID, _ := dataMap["group_id"].(string)
	userID, _ := dataMap["user_id"].(string)
	attrsData, _ := dataMap["attrs"].(map[string]any)
	nameData, hasName := dataMap["name"].(string)

	if groupID == "" || userID == "" {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "missing group_id or user_id"})
		return
	}

	// 验证至少提供了 attrs 或 name
	if attrsData == nil && (!hasName || nameData == "") {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "missing attrs or name"})
		return
	}

	// 格式化 ID
	formattedGroupID := FormatDiceIDSealChatGroup(groupID)

	// 安全限制：仅允许写入 SealChat 平台的角色卡
	if !strings.HasPrefix(formattedGroupID, "SEALCHAT-Group:") {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "write access denied: only SealChat platform cards can be modified"})
		return
	}

	// 速率限制：60次/分钟
	if pa.characterSetLimiter != nil && !pa.characterSetLimiter.Allow() {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "rate limit exceeded: max 60 writes per minute"})
		return
	}

	target, err := pa.setSealChatCharacterAttrs(d, groupID, userID, attrsData)
	if err != nil {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	attrs := target.Attrs

	// 更新名称（如果提供）
	if hasName && nameData != "" {
		attrs.Name = nameData
		attrs.SetModified()
	}

	pa.sendApiResponse(msg.Echo, map[string]any{"ok": true})
}

// handleCharacterList 处理获取角色卡列表请求
func (pa *PlatformAdapterSealChat) handleCharacterList(msg satori.ScApiMsgPayload, d *Dice) {
	dataMap, ok := msg.Data.(map[string]any)
	if !ok {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "invalid data format"})
		return
	}

	userID, _ := dataMap["user_id"].(string)
	if userID == "" {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "missing user_id"})
		return
	}

	formattedUserID := FormatDiceIDSealChat(userID)
	list, err := d.AttrsManager.GetCharacterList(formattedUserID)
	if err != nil {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	// 构建返回列表
	result := make([]map[string]any, 0, len(list))
	for _, item := range list {
		result = append(result, map[string]any{
			"id":         item.Id,
			"name":       item.Name,
			"sheet_type": item.SheetType,
			"updated_at": item.UpdatedAt,
		})
	}

	pa.sendApiResponse(msg.Echo, map[string]any{
		"ok":   true,
		"list": result,
	})
}

// getCharIdFromRequest 从请求中获取角色卡ID（支持id或name两种方式）
func (pa *PlatformAdapterSealChat) getCharIdFromRequest(d *Dice, dataMap map[string]any, userID string) (string, string, error) {
	formattedUserID := FormatDiceIDSealChat(userID)

	// 优先使用 id
	if id, ok := dataMap["id"].(string); ok && id != "" {
		// 验证 id 属于该用户且为角色卡类型
		item, err := service.AttrsGetById(d.DBOperator, id)
		if err != nil {
			return "", "", err
		}
		if !item.IsDataExists() {
			return "", "", fmt.Errorf("character not found: %s", id)
		}
		// 安全检查：验证归属用户和类型
		if item.OwnerId != formattedUserID || item.AttrsType != service.AttrsTypeCharacter {
			return "", "", errors.New("character not owned by user or invalid type")
		}
		return id, item.Name, nil
	}

	// 其次使用 name
	if name, ok := dataMap["name"].(string); ok && name != "" {
		charId, err := d.AttrsManager.CharIdGetByName(formattedUserID, name)
		if err != nil {
			return "", "", err
		}
		if charId == "" {
			return "", "", fmt.Errorf("character not found: %s", name)
		}
		return charId, name, nil
	}

	return "", "", errMissingCharIdOrName
}

// handleCharacterNew 处理新建角色卡请求 (对应 .pc new)
func (pa *PlatformAdapterSealChat) handleCharacterNew(msg satori.ScApiMsgPayload, d *Dice) {
	dataMap, ok := msg.Data.(map[string]any)
	if !ok {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "invalid data format"})
		return
	}

	groupID, _ := dataMap["group_id"].(string)
	userID, _ := dataMap["user_id"].(string)
	name, _ := dataMap["name"].(string)
	sheetType, _ := dataMap["sheet_type"].(string)

	if groupID == "" || userID == "" || name == "" {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "missing group_id, user_id or name"})
		return
	}

	// 速率限制
	if pa.characterSetLimiter != nil && !pa.characterSetLimiter.Allow() {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "rate limit exceeded: max 60 writes per minute"})
		return
	}

	formattedUserID := FormatDiceIDSealChat(userID)
	formattedGroupID := FormatDiceIDSealChatGroup(groupID)

	// 检查角色名是否已存在
	if d.AttrsManager.CharCheckExists(formattedUserID, name) {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "character already exists"})
		return
	}

	// 默认 sheet_type
	if sheetType == "" {
		sheetType = "coc7"
	}

	// 创建角色卡
	item, err := d.AttrsManager.CharNew(formattedUserID, name, sheetType)
	if err != nil {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	// 绑定到当前群
	if err := d.AttrsManager.CharBind(item.Id, formattedGroupID, formattedUserID); err != nil {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	pa.sendApiResponse(msg.Echo, map[string]any{
		"ok":             true,
		"id":             item.Id,
		"name":           name,
		"sheet_type":     sheetType,
		"bound_group_id": formattedGroupID,
	})
}

// handleCharacterSave 处理保存独立卡为角色卡请求 (对应 .pc save)
func (pa *PlatformAdapterSealChat) handleCharacterSave(msg satori.ScApiMsgPayload, d *Dice) {
	dataMap, ok := msg.Data.(map[string]any)
	if !ok {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "invalid data format"})
		return
	}

	groupID, _ := dataMap["group_id"].(string)
	userID, _ := dataMap["user_id"].(string)
	name, _ := dataMap["name"].(string)
	sheetType, _ := dataMap["sheet_type"].(string)

	if groupID == "" || userID == "" {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "missing group_id or user_id"})
		return
	}

	// 速率限制
	if pa.characterSetLimiter != nil && !pa.characterSetLimiter.Allow() {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "rate limit exceeded: max 60 writes per minute"})
		return
	}

	formattedUserID := FormatDiceIDSealChat(userID)
	formattedGroupID := FormatDiceIDSealChatGroup(groupID)

	// 加载当前群的独立卡数据
	currentAttrs, err := d.AttrsManager.Load(formattedGroupID, formattedUserID)
	if err != nil {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	// 如果未提供 name，使用当前角色名
	if name == "" {
		name = currentAttrs.Name
	}
	if name == "" {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "name is required"})
		return
	}

	// 默认 sheet_type
	if sheetType == "" {
		sheetType = currentAttrs.SheetType
	}
	if sheetType == "" {
		sheetType = "coc7"
	}

	var action string
	var charId string

	// 检查角色卡是否已存在
	existingId, _ := d.AttrsManager.CharIdGetByName(formattedUserID, name)
	if existingId == "" {
		// 不存在，创建新卡
		newItem, err := d.AttrsManager.CharNew(formattedUserID, name, sheetType)
		if err != nil {
			pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		charId = newItem.Id
		action = "created"

		// 复制当前独立卡数据到新卡
		newAttrs, err := d.AttrsManager.LoadById(charId)
		if err != nil {
			pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		currentAttrs.Range(func(key string, value *ds.VMValue) bool {
			newAttrs.Store(key, value)
			return true
		})
		newAttrs.Name = name
		newAttrs.SheetType = sheetType
	} else {
		// 已存在，检查是否被绑定
		bindingGroups := d.AttrsManager.CharGetBindingGroupIdList(existingId)
		if len(bindingGroups) > 0 {
			pa.sendApiResponse(msg.Echo, map[string]any{
				"ok":             false,
				"error":          "character is bound, cannot overwrite",
				"binding_groups": bindingGroups,
			})
			return
		}

		// 覆盖现有卡数据
		charId = existingId
		action = "updated"

		existingAttrs, err := d.AttrsManager.LoadById(charId)
		if err != nil {
			pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		// 先清空现有数据，避免残留
		existingAttrs.Clear()
		// 复制源数据
		currentAttrs.Range(func(key string, value *ds.VMValue) bool {
			existingAttrs.Store(key, value)
			return true
		})
		// 同步 Name 和 SheetType
		existingAttrs.Name = name
		existingAttrs.SheetType = sheetType
		existingAttrs.SetModified()
	}

	pa.sendApiResponse(msg.Echo, map[string]any{
		"ok":     true,
		"id":     charId,
		"name":   name,
		"action": action,
	})
}

// handleCharacterTag 处理绑定/解绑角色卡请求 (对应 .pc tag)
func (pa *PlatformAdapterSealChat) handleCharacterTag(msg satori.ScApiMsgPayload, d *Dice) {
	dataMap, ok := msg.Data.(map[string]any)
	if !ok {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "invalid data format"})
		return
	}

	groupID, _ := dataMap["group_id"].(string)
	userID, _ := dataMap["user_id"].(string)

	if groupID == "" || userID == "" {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "missing group_id or user_id"})
		return
	}

	// 速率限制
	if pa.characterSetLimiter != nil && !pa.characterSetLimiter.Allow() {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "rate limit exceeded: max 60 writes per minute"})
		return
	}

	formattedUserID := FormatDiceIDSealChat(userID)
	formattedGroupID := FormatDiceIDSealChatGroup(groupID)

	// 检查是否提供了 id 或 name（绑定操作）
	charId, charName, err := pa.getCharIdFromRequest(d, dataMap, userID)

	if err != nil && !errors.Is(err, errMissingCharIdOrName) {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	if charId != "" {
		// 绑定操作
		if err := d.AttrsManager.CharBind(charId, formattedGroupID, formattedUserID); err != nil {
			pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		pa.sendApiResponse(msg.Echo, map[string]any{
			"ok":     true,
			"action": "bind",
			"id":     charId,
			"name":   charName,
		})
	} else {
		// 解绑操作
		currentBindingId, _ := d.AttrsManager.CharGetBindingId(formattedGroupID, formattedUserID)
		if currentBindingId == "" {
			pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "no character bound"})
			return
		}

		// 获取当前绑定角色名
		currentAttrs, _ := d.AttrsManager.LoadById(currentBindingId)
		var unboundName string
		if currentAttrs != nil {
			unboundName = currentAttrs.Name
		}

		// 解绑（绑定空字符串）
		if err := d.AttrsManager.CharBind("", formattedGroupID, formattedUserID); err != nil {
			pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		pa.sendApiResponse(msg.Echo, map[string]any{
			"ok":     true,
			"action": "unbind",
			"id":     currentBindingId,
			"name":   unboundName,
		})
	}
}

// handleCharacterUntagAll 处理从所有群解绑请求 (对应 .pc untagAll)
func (pa *PlatformAdapterSealChat) handleCharacterUntagAll(msg satori.ScApiMsgPayload, d *Dice) {
	dataMap, ok := msg.Data.(map[string]any)
	if !ok {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "invalid data format"})
		return
	}

	userID, _ := dataMap["user_id"].(string)
	groupID, _ := dataMap["group_id"].(string) // 可选，用于获取当前绑定卡

	if userID == "" {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "missing user_id"})
		return
	}

	// 速率限制
	if pa.characterSetLimiter != nil && !pa.characterSetLimiter.Allow() {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "rate limit exceeded: max 60 writes per minute"})
		return
	}

	formattedUserID := FormatDiceIDSealChat(userID)

	// 获取角色卡 ID
	charId, charName, err := pa.getCharIdFromRequest(d, dataMap, userID)
	if errors.Is(err, errMissingCharIdOrName) {
		// 未提供 id/name，使用当前群绑定的卡
		if groupID != "" {
			formattedGroupID := FormatDiceIDSealChatGroup(groupID)
			charId, _ = d.AttrsManager.CharGetBindingId(formattedGroupID, formattedUserID)
			if charId != "" {
				attrs, _ := d.AttrsManager.LoadById(charId)
				if attrs != nil {
					charName = attrs.Name
				}
			}
		}
	} else if err != nil {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	if charId == "" {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "no character specified or bound"})
		return
	}

	// 执行全部解绑
	unboundGroups := d.AttrsManager.CharUnbindAll(charId)

	pa.sendApiResponse(msg.Echo, map[string]any{
		"ok":             true,
		"id":             charId,
		"name":           charName,
		"unbound_count":  len(unboundGroups),
		"unbound_groups": unboundGroups,
	})
}

// handleCharacterLoad 处理加载角色卡到独立卡请求 (对应 .pc load)
func (pa *PlatformAdapterSealChat) handleCharacterLoad(msg satori.ScApiMsgPayload, d *Dice) {
	dataMap, ok := msg.Data.(map[string]any)
	if !ok {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "invalid data format"})
		return
	}

	groupID, _ := dataMap["group_id"].(string)
	userID, _ := dataMap["user_id"].(string)

	if groupID == "" || userID == "" {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "missing group_id or user_id"})
		return
	}

	// 速率限制
	if pa.characterSetLimiter != nil && !pa.characterSetLimiter.Allow() {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "rate limit exceeded: max 60 writes per minute"})
		return
	}

	formattedUserID := FormatDiceIDSealChat(userID)
	formattedGroupID := FormatDiceIDSealChatGroup(groupID)

	// 获取目标角色卡
	charId, charName, err := pa.getCharIdFromRequest(d, dataMap, userID)
	if err != nil {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	// 加载目标角色卡
	sourceAttrs, err := d.AttrsManager.LoadById(charId)
	if err != nil {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	// 加载当前群的独立卡
	targetAttrs, err := d.AttrsManager.Load(formattedGroupID, formattedUserID)
	if err != nil {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	// 先清空目标卡数据，避免残留
	targetAttrs.Clear()

	// 复制数据
	sourceAttrs.Range(func(key string, value *ds.VMValue) bool {
		targetAttrs.Store(key, value)
		return true
	})
	targetAttrs.Name = sourceAttrs.Name
	targetAttrs.SheetType = sourceAttrs.SheetType
	targetAttrs.SetModified()

	pa.sendApiResponse(msg.Echo, map[string]any{
		"ok":         true,
		"id":         charId,
		"name":       charName,
		"sheet_type": sourceAttrs.SheetType,
	})
}

// handleCharacterDelete 处理删除角色卡请求 (对应 .pc del)
func (pa *PlatformAdapterSealChat) handleCharacterDelete(msg satori.ScApiMsgPayload, d *Dice) {
	dataMap, ok := msg.Data.(map[string]any)
	if !ok {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "invalid data format"})
		return
	}

	userID, _ := dataMap["user_id"].(string)

	if userID == "" {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "missing user_id"})
		return
	}

	// 速率限制
	if pa.characterSetLimiter != nil && !pa.characterSetLimiter.Allow() {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": "rate limit exceeded: max 60 writes per minute"})
		return
	}

	// 获取角色卡
	charId, charName, err := pa.getCharIdFromRequest(d, dataMap, userID)
	if err != nil {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	// 检查是否被绑定
	bindingGroups := d.AttrsManager.CharGetBindingGroupIdList(charId)
	if len(bindingGroups) > 0 {
		pa.sendApiResponse(msg.Echo, map[string]any{
			"ok":             false,
			"error":          "character is bound, use untagAll first",
			"binding_groups": bindingGroups,
		})
		return
	}

	// 删除角色卡
	if err := d.AttrsManager.CharDelete(charId); err != nil {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	pa.sendApiResponse(msg.Echo, map[string]any{
		"ok":   true,
		"id":   charId,
		"name": charName,
	})
}
