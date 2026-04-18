package dice

import (
	"errors"
	"fmt"
	"strings"

	ds "github.com/sealdice/dicescript"

	"sealdice-core/dice/service"
	"sealdice-core/utils/satori"
)

// 哨兵错误
var errMissingCharIdOrName = errors.New("missing id or name")

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

	// 格式化 ID
	formattedGroupID := FormatDiceIDSealChatGroup(groupID)
	formattedUserID := FormatDiceIDSealChat(userID)

	attrs, err := d.AttrsManager.Load(formattedGroupID, formattedUserID)
	if err != nil {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	// 将属性导出为 map
	attrsData := make(map[string]any)
	if attrs != nil {
		attrs.Range(func(key string, value *ds.VMValue) bool {
			// 将 VMValue 转换为可 JSON 序列化的值
			attrsData[key] = vmValueToAny(value)
			return true
		})
	}

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
	switch v.TypeId {
	case ds.VMTypeInt:
		return v.MustReadInt()
	case ds.VMTypeFloat:
		return v.MustReadFloat()
	case ds.VMTypeString:
		s, _ := v.ReadString()
		return s
	default:
		// 对于复杂类型，尝试转换为字符串
		return v.ToString()
	}
}

// anyToVMValue 将 any 类型转换为 VMValue
func anyToVMValue(v any) *ds.VMValue {
	switch val := v.(type) {
	case float64:
		// JSON 解析时数字默认为 float64
		if val == float64(int64(val)) {
			return ds.NewIntVal(ds.IntType(val))
		}
		return ds.NewFloatVal(val)
	case int:
		return ds.NewIntVal(ds.IntType(val))
	case int64:
		return ds.NewIntVal(ds.IntType(val))
	case string:
		return ds.NewStrVal(val)
	case bool:
		if val {
			return ds.NewIntVal(1)
		}
		return ds.NewIntVal(0)
	default:
		return ds.NewStrVal(fmt.Sprintf("%v", v))
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
	formattedUserID := FormatDiceIDSealChat(userID)

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

	attrs, err := d.AttrsManager.Load(formattedGroupID, formattedUserID)
	if err != nil {
		pa.sendApiResponse(msg.Echo, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	// 写入属性
	for k, v := range attrsData {
		attrs.Store(k, anyToVMValue(v))
	}

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
