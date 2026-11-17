package dice

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	ds "github.com/sealdice/dicescript"
	"gorm.io/gorm"

	"sealdice-core/model"
	"sealdice-core/utils/constant"
	"sealdice-core/utils/dboperator/engine"
)

type officialQQIdentityMigration struct {
	appID          string
	uin            string
	oldEndpoint    string
	newEndpoint    string
	groups         map[string]string
	users          map[string]string
	characterNames map[string]string
}

const officialQQIdentityMigrationBatchSize = 200

func officialQQIdentityMigrationFindInBatches[T any](query *gorm.DB, process func([]T) error) error {
	var rows []T
	query = officialQQIdentityMigrationWithoutCache(query)
	return query.FindInBatches(&rows, officialQQIdentityMigrationBatchSize, func(_ *gorm.DB, _ int) error {
		return process(rows)
	}).Error
}

func officialQQIdentityMigrationWithoutCache(db *gorm.DB) *gorm.DB {
	return db
}

func officialQQIdentityMigrationForEachChunk[T any](items []T, process func([]T) error) error {
	for start := 0; start < len(items); start += officialQQIdentityMigrationBatchSize {
		end := min(start+officialQQIdentityMigrationBatchSize, len(items))
		if err := process(items[start:end]); err != nil {
			return err
		}
	}
	return nil
}

type officialQQIdentityMigrationError struct {
	cause error
}

func (e *officialQQIdentityMigrationError) Error() string {
	return e.cause.Error()
}

func (e *officialQQIdentityMigrationError) Unwrap() error {
	return e.cause
}

func migratedCharacterName(baseName string, suffix int) string {
	if baseName == "" {
		baseName = "未命名角色"
	}
	suffixText := fmt.Sprintf(" (%d)", suffix)
	const maxNameBytes = 90
	maxBaseBytes := maxNameBytes - len(suffixText)
	if len(baseName) > maxBaseBytes {
		baseName = baseName[:maxBaseBytes]
		for !utf8.ValidString(baseName) {
			baseName = baseName[:len(baseName)-1]
		}
	}
	return baseName + suffixText
}

func ensureOfficialQQIdentity(d *Dice, pa *PlatformAdapterOfficialQQ, botID, uin string) error {
	if d == nil || pa == nil || pa.EndPoint == nil {
		return errors.New("QQ 官方机器人运行时未初始化")
	}
	if strings.TrimSpace(uin) == "" {
		return errors.New("QQ 官方机器人 UIN 为空")
	}

	expectedEndpoint := formatDiceIDOfficialQQ(uin)
	oldEndpoint := formatDiceIDOfficialQQ(botID)
	if current := pa.EndPoint.UserID; current != "" && current != oldEndpoint && current != expectedEndpoint {
		return fmt.Errorf("无法识别旧账号 ID %q，期望 %q", current, oldEndpoint)
	}
	// The verified runtime identity is required even when the optional legacy
	// data migration is disabled or fails.
	pa.UIN = uin
	pa.EndPoint.UserID = expectedEndpoint

	if d.Config.OfficialQQMigrationEnable {
		migration := &officialQQIdentityMigration{
			appID:       pa.AppID,
			uin:         uin,
			oldEndpoint: oldEndpoint,
			newEndpoint: expectedEndpoint,
			groups:      map[string]string{},
			users:       map[string]string{},
		}
		if err := migration.run(d); err != nil {
			return &officialQQIdentityMigrationError{cause: err}
		}
	}

	d.LastUpdatedTime = time.Now().Unix()
	d.Save(false)
	return nil
}

func (m *officialQQIdentityMigration) oldGroupPrefix() string {
	return "OpenQQ-Group-T:" + m.appID + "-"
}

func officialQQIdentityMigrationPrefixBounds(prefix string) (string, string) {
	// All legacy identity prefixes are ASCII and end in '-'. Replacing that
	// byte with '.' gives the smallest exclusive upper bound for the prefix.
	return prefix, strings.TrimSuffix(prefix, "-") + "."
}

func (m *officialQQIdentityMigration) oldGroupBounds() (string, string) {
	return officialQQIdentityMigrationPrefixBounds(m.oldGroupPrefix())
}

func (m *officialQQIdentityMigration) oldMemberRootPrefix() string {
	return "OpenQQ-Member-T:" + m.appID + "-"
}

func (m *officialQQIdentityMigration) oldMemberBounds() (string, string) {
	return officialQQIdentityMigrationPrefixBounds(m.oldMemberRootPrefix())
}

func (m *officialQQIdentityMigration) oldMemberPrefix(groupOpenID string) string {
	return "OpenQQ-Member-T:" + m.appID + "-" + groupOpenID + "-"
}

func (m *officialQQIdentityMigration) addGroup(groupID string) bool {
	groupOpenID, ok := strings.CutPrefix(groupID, m.oldGroupPrefix())
	if !ok {
		return false
	}
	// Some old releases persisted one synthetic group with an empty GroupOpenID.
	// Keep its data under a deterministic non-QQ placeholder instead of dropping it.
	if groupOpenID == "" {
		groupOpenID = "legacy-empty-" + m.appID
	}
	m.groups[groupID] = formatDiceIDOfficialQQGroupOpenID(m.uin, groupOpenID)
	return true
}

func (m *officialQQIdentityMigration) groupOpenID(groupID string) (string, bool) {
	groupOpenID, ok := strings.CutPrefix(groupID, m.oldGroupPrefix())
	return groupOpenID, ok
}

func (m *officialQQIdentityMigration) addMember(groupID, userID string) bool {
	groupOpenID, ok := m.groupOpenID(groupID)
	if !ok {
		return false
	}
	memberOpenID, ok := strings.CutPrefix(userID, m.oldMemberPrefix(groupOpenID))
	if !ok || memberOpenID == "" {
		return false
	}
	m.users[userID] = formatDiceIDOfficialQQMemberOpenID(m.uin, groupOpenID, memberOpenID)
	return true
}

func (m *officialQQIdentityMigration) addMemberID(userID string) bool {
	if _, exists := m.users[userID]; exists {
		return true
	}
	rest, ok := strings.CutPrefix(userID, "OpenQQ-Member-T:"+m.appID+"-")
	if !ok {
		return false
	}
	for _, oldGroupID := range m.sortedOldGroups() {
		groupOpenID, exists := m.groupOpenID(oldGroupID)
		if !exists {
			continue
		}
		memberOpenID, matches := strings.CutPrefix(rest, groupOpenID+"-")
		if matches && memberOpenID != "" {
			m.users[userID] = formatDiceIDOfficialQQMemberOpenID(m.uin, groupOpenID, memberOpenID)
			return true
		}
	}
	separator := strings.LastIndexByte(rest, '-')
	if separator <= 0 || separator == len(rest)-1 {
		return false
	}
	memberOpenID := rest[separator+1:]
	m.users[userID] = formatDiceIDOfficialQQMemberOpenID(m.uin, rest[:separator], memberOpenID)
	return true
}

func (m *officialQQIdentityMigration) migrateGroupID(id string) string {
	if value, ok := m.groups[id]; ok {
		return value
	}
	return id
}

func (m *officialQQIdentityMigration) migrateUserID(id string) string {
	if value, ok := m.users[id]; ok {
		return value
	}
	if m.addMemberID(id) {
		return m.users[id]
	}
	return id
}

func (m *officialQQIdentityMigration) migrateAnyID(id string) string {
	if id == m.oldEndpoint {
		return m.newEndpoint
	}
	if value := m.migrateGroupID(id); value != id {
		return value
	}
	return m.migrateUserID(id)
}

func (m *officialQQIdentityMigration) migrateContextUserID(groupID, userID string) string {
	if _, ok := m.users[userID]; !ok {
		m.addMember(groupID, userID)
	}
	return m.migrateUserID(userID)
}

func (m *officialQQIdentityMigration) sortedOldGroups() []string {
	groups := make([]string, 0, len(m.groups))
	for groupID := range m.groups {
		groups = append(groups, groupID)
	}
	sort.Slice(groups, func(i, j int) bool { return len(groups[i]) > len(groups[j]) })
	return groups
}

func (m *officialQQIdentityMigration) migrateAttrsID(id, attrsType string) string {
	if attrsType == "character" {
		return id
	}
	if value := m.migrateAnyID(id); value != id {
		return value
	}
	if attrsType != "" && attrsType != "group_user" {
		return id
	}
	for _, oldGroupID := range m.sortedOldGroups() {
		userID, ok := strings.CutPrefix(id, oldGroupID+"-")
		if !ok || userID == "" {
			continue
		}
		newUserID := m.migrateContextUserID(oldGroupID, userID)
		if newUserID == userID {
			return id
		}
		return m.groups[oldGroupID] + "-" + newUserID
	}
	return id
}

func (m *officialQQIdentityMigration) run(d *Dice) error {
	if d.DBOperator != nil {
		shouldMigrate, err := m.shouldRunIdentityMigration(d.DBOperator)
		if err != nil {
			return fmt.Errorf("检查 QQ 官方旧群组数据失败: %w", err)
		}
		if !shouldMigrate {
			if err := m.syncCharacterCache(d, d.DBOperator); err != nil {
				return fmt.Errorf("同步角色卡缓存失败: %w", err)
			}
			m.migrateMemory(d)
			return nil
		}
		if err := m.collectDatabase(d.DBOperator); err != nil {
			return err
		}
	}
	m.collectMemory(d)

	if d.AttrsManager != nil {
		if err := d.AttrsManager.CheckForSave(); err != nil {
			return fmt.Errorf("迁移前保存属性数据失败: %w", err)
		}
	}
	if d.DBOperator != nil {
		if err := m.migrateLogDB(d.DBOperator); err != nil {
			return fmt.Errorf("迁移日志数据库失败: %w", err)
		}
		if err := m.migrateCensorDB(d.DBOperator); err != nil {
			return fmt.Errorf("迁移敏感词数据库失败: %w", err)
		}
		if err := m.migrateDataDB(d.DBOperator); err != nil {
			return fmt.Errorf("迁移主数据库失败: %w", err)
		}
		if err := m.syncCharacterCache(d, d.DBOperator); err != nil {
			return fmt.Errorf("同步角色卡缓存失败: %w", err)
		}
	}
	m.migrateMemory(d)
	return nil
}

func (m *officialQQIdentityMigration) syncCharacterCache(d *Dice, operator engine.DatabaseOperator) error {
	if d.AttrsManager == nil {
		return nil
	}
	var ids []string
	d.AttrsManager.m.Range(func(id string, _ *AttributesItem) bool {
		ids = append(ids, id)
		return true
	})
	if len(ids) == 0 {
		return nil
	}

	db := operator.GetDataDB(constant.READ)
	if !hasTable(db, &model.AttributesItemModel{}) {
		return nil
	}
	db = officialQQIdentityMigrationWithoutCache(db)
	return officialQQIdentityMigrationForEachChunk(ids, func(batchIDs []string) error {
		var rows []model.AttributesItemModel
		if err := db.Select("id, name").Where("attrs_type = ? AND id IN ?", "character", batchIDs).Find(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			if item, ok := d.AttrsManager.m.Load(row.Id); ok && item != nil {
				item.Name = row.Name
				item.IsSaved = true
			}
		}
		return nil
	})
}

func hasTable(db *gorm.DB, value any) bool {
	return db != nil && db.Migrator().HasTable(value)
}

func (m *officialQQIdentityMigration) shouldRunIdentityMigration(operator engine.DatabaseOperator) (bool, error) {
	groupPrefix, groupUpperBound := m.oldGroupBounds()
	memberPrefix, memberUpperBound := m.oldMemberBounds()

	probes := []*gorm.DB{}
	dataDB := operator.GetDataDB(constant.READ)
	if !hasTable(dataDB, &model.GroupInfo{}) {
		// Without group_info the optimized database probe cannot rule out
		// legacy identities held only in memory, so preserve the full fallback.
		return true, nil
	}
	probes = append(probes, dataDB.Model(&model.GroupInfo{}).Where("id >= ? AND id < ?", groupPrefix, groupUpperBound))

	logDB := operator.GetLogDB(constant.READ)
	if hasTable(logDB, &model.LogInfo{}) {
		probes = append(probes, logDB.Model(&model.LogInfo{}).Where("group_id >= ? AND group_id < ?", groupPrefix, groupUpperBound))
	}
	if hasTable(logDB, &model.LogOneItem{}) {
		probes = append(probes, logDB.Model(&model.LogOneItem{}).Where("group_id >= ? AND group_id < ?", groupPrefix, groupUpperBound))
	}

	if hasTable(dataDB, &model.GroupPlayerInfoBase{}) {
		probes = append(probes, dataDB.Model(&model.GroupPlayerInfoBase{}).Where("group_id >= ? AND group_id < ?", groupPrefix, groupUpperBound))
	}
	if hasTable(dataDB, &model.AttributesItemModel{}) {
		probes = append(probes, m.legacyAttrsQuery(dataDB.Model(&model.AttributesItemModel{})))
	}
	if hasTable(dataDB, &model.BanInfo{}) {
		probes = append(probes, dataDB.Model(&model.BanInfo{}).Where(
			"(id >= ? AND id < ?) OR (id >= ? AND id < ?)",
			groupPrefix,
			groupUpperBound,
			memberPrefix,
			memberUpperBound,
		))
	}
	if hasTable(dataDB, &model.EndpointInfo{}) {
		probes = append(probes, dataDB.Model(&model.EndpointInfo{}).Where("user_id = ?", m.oldEndpoint))
	}

	censorDB := operator.GetCensorDB(constant.READ)
	if hasTable(censorDB, &model.CensorLog{}) {
		probes = append(probes, censorDB.Model(&model.CensorLog{}).Where("group_id >= ? AND group_id < ?", groupPrefix, groupUpperBound))
	}

	for _, probe := range probes {
		var found int
		result := officialQQIdentityMigrationWithoutCache(probe).Select("1").Limit(1).Scan(&found)
		if result.Error != nil {
			return false, result.Error
		}
		if result.RowsAffected != 0 {
			return true, nil
		}
	}
	return false, nil
}

func (m *officialQQIdentityMigration) collectDatabase(operator engine.DatabaseOperator) error {
	groupPrefix, groupUpperBound := m.oldGroupBounds()
	memberPrefix, memberUpperBound := m.oldMemberBounds()
	dataDB := operator.GetDataDB(constant.READ)
	if hasTable(dataDB, &model.GroupInfo{}) {
		err := officialQQIdentityMigrationFindInBatches(
			dataDB.Select("id").Where("id >= ? AND id < ?", groupPrefix, groupUpperBound),
			func(rows []model.GroupInfo) error {
				for _, row := range rows {
					m.addGroup(row.ID)
				}
				return nil
			},
		)
		if err != nil {
			return err
		}
	}
	if hasTable(dataDB, &model.GroupPlayerInfoBase{}) {
		err := officialQQIdentityMigrationFindInBatches(
			dataDB.Select("id, group_id, user_id").Where("group_id >= ? AND group_id < ?", groupPrefix, groupUpperBound),
			func(rows []model.GroupPlayerInfoBase) error {
				for _, row := range rows {
					m.addGroup(row.GroupID)
					m.addMember(row.GroupID, row.UserID)
				}
				return nil
			},
		)
		if err != nil {
			return err
		}
	}

	logDB := operator.GetLogDB(constant.READ)
	if hasTable(logDB, &model.LogInfo{}) {
		err := officialQQIdentityMigrationFindInBatches(
			logDB.Select("id, group_id").Where("group_id >= ? AND group_id < ?", groupPrefix, groupUpperBound),
			func(rows []model.LogInfo) error {
				for _, row := range rows {
					m.addGroup(row.GroupID)
				}
				return nil
			},
		)
		if err != nil {
			return err
		}
	}
	if hasTable(logDB, &model.LogOneItem{}) {
		err := officialQQIdentityMigrationFindInBatches(
			logDB.Select("id, group_id, im_userid, user_uniform_id").Where("group_id >= ? AND group_id < ?", groupPrefix, groupUpperBound),
			func(rows []model.LogOneItem) error {
				for _, row := range rows {
					m.addGroup(row.GroupID)
					m.addMember(row.GroupID, row.IMUserID)
					m.addMember(row.GroupID, row.UniformID)
				}
				return nil
			},
		)
		if err != nil {
			return err
		}
	}

	censorDB := operator.GetCensorDB(constant.READ)
	if hasTable(censorDB, &model.CensorLog{}) {
		err := officialQQIdentityMigrationFindInBatches(
			censorDB.Select("id, group_id, user_id").Where("group_id >= ? AND group_id < ?", groupPrefix, groupUpperBound),
			func(rows []model.CensorLog) error {
				for _, row := range rows {
					m.addGroup(row.GroupID)
					m.addMember(row.GroupID, row.UserID)
				}
				return nil
			},
		)
		if err != nil {
			return err
		}
	}

	if hasTable(dataDB, &model.AttributesItemModel{}) {
		groupUserMarker := "-OpenQQ-Member-T:" + m.appID + "-"
		err := officialQQIdentityMigrationFindInBatches(
			m.legacyAttrsQuery(dataDB.Select("id, owner_id")),
			func(rows []model.AttributesItemModel) error {
				for _, row := range rows {
					if marker := strings.Index(row.Id, groupUserMarker); marker > 0 {
						oldGroupID := row.Id[:marker]
						if m.addGroup(oldGroupID) {
							m.addMember(oldGroupID, row.Id[marker+1:])
						}
					} else if strings.HasPrefix(row.Id, m.oldGroupPrefix()) {
						m.addGroup(row.Id)
					}
					m.addMemberID(row.Id)
					m.addMemberID(row.OwnerId)
				}
				return nil
			},
		)
		if err != nil {
			return err
		}
	}

	if hasTable(dataDB, &model.BanInfo{}) {
		err := officialQQIdentityMigrationFindInBatches(
			dataDB.Select("id, data").Where(
				"(id >= ? AND id < ?) OR (id >= ? AND id < ?)",
				groupPrefix,
				groupUpperBound,
				memberPrefix,
				memberUpperBound,
			),
			func(rows []model.BanInfo) error {
				for _, row := range rows {
					var item BanListInfoItem
					if json.Unmarshal(row.Data, &item) != nil {
						continue
					}
					for _, place := range item.Places {
						m.addGroup(place)
						m.addMember(place, row.ID)
					}
				}
				return nil
			},
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *officialQQIdentityMigration) collectMemory(d *Dice) {
	if d.ImSession != nil && d.ImSession.ServiceAtNew != nil {
		d.ImSession.ServiceAtNew.Range(func(groupID string, group *GroupInfo) bool {
			if !m.addGroup(groupID) {
				return true
			}
			if group != nil {
				m.addMember(groupID, group.InviteUserID)
				if group.Players != nil {
					group.Players.Range(func(userID string, _ *GroupPlayerInfo) bool {
						m.addMember(groupID, userID)
						return true
					})
				}
			}
			return true
		})
	}
	if d.Config.BanList != nil && d.Config.BanList.Map != nil {
		d.Config.BanList.Map.Range(func(id string, item *BanListInfoItem) bool {
			if item != nil {
				for _, place := range item.Places {
					m.addGroup(place)
					m.addMember(place, id)
				}
			}
			return true
		})
	}
}

func ensureNoTarget(db *gorm.DB, value any, where string, args ...any) error {
	var count int64
	if err := db.Model(value).Where(where, args...).Count(&count).Error; err != nil {
		return err
	}
	if count != 0 {
		return fmt.Errorf("目标数据已存在: %s %v", where, args)
	}
	return nil
}

func (m *officialQQIdentityMigration) legacyAttrsQuery(db *gorm.DB) *gorm.DB {
	groupPrefix, groupUpperBound := m.oldGroupBounds()
	memberPrefix, memberUpperBound := m.oldMemberBounds()
	return db.Where(
		"(id >= ? AND id < ?) OR (id >= ? AND id < ?) OR (owner_id >= ? AND owner_id < ?)",
		groupPrefix,
		groupUpperBound,
		memberPrefix,
		memberUpperBound,
		memberPrefix,
		memberUpperBound,
	)
}

const officialQQIdentityMigrationAttrsMetadataColumns = "id, attrs_type, binding_sheet_id, name, owner_id, sheet_type, is_hidden, created_at, updated_at"

func mergeOfficialQQUserAttrs(tx *gorm.DB, rows []model.AttributesItemModel) (model.AttributesItemModel, error) {
	if len(rows) == 0 {
		return model.AttributesItemModel{}, errors.New("没有可合并的 QQ 官方用户属性")
	}

	ordered := append([]model.AttributesItemModel(nil), rows...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].UpdatedAt != ordered[j].UpdatedAt {
			return ordered[i].UpdatedAt < ordered[j].UpdatedAt
		}
		if ordered[i].CreatedAt != ordered[j].CreatedAt {
			return ordered[i].CreatedAt < ordered[j].CreatedAt
		}
		return ordered[i].Id < ordered[j].Id
	})

	merged := &ds.ValueMap{}
	createdAt := ordered[0].CreatedAt
	updatedAt := ordered[0].UpdatedAt
	for _, row := range ordered {
		if row.AttrsType != "" && row.AttrsType != "user" {
			return model.AttributesItemModel{}, fmt.Errorf("无法合并 QQ 官方用户属性 %s: attrs_type=%q", row.Id, row.AttrsType)
		}
		if row.BindingSheetId != "" {
			return model.AttributesItemModel{}, fmt.Errorf("无法合并带角色绑定的 QQ 官方用户属性 %s", row.Id)
		}
		if row.CreatedAt < createdAt {
			createdAt = row.CreatedAt
		}
		if row.UpdatedAt > updatedAt {
			updatedAt = row.UpdatedAt
		}
	}
	tx = officialQQIdentityMigrationWithoutCache(tx)
	if err := officialQQIdentityMigrationForEachChunk(ordered, func(batch []model.AttributesItemModel) error {
		ids := make([]string, 0, len(batch))
		for _, row := range batch {
			ids = append(ids, row.Id)
		}
		var dataRows []model.AttributesItemModel
		if err := tx.Select("id, data").Where("id IN ?", ids).Find(&dataRows).Error; err != nil {
			return err
		}
		dataByID := make(map[string][]byte, len(dataRows))
		for _, row := range dataRows {
			dataByID[row.Id] = row.Data
		}
		for _, row := range batch {
			data, exists := dataByID[row.Id]
			if !exists {
				return fmt.Errorf("QQ 官方用户属性 %s 在合并期间不存在", row.Id)
			}
			value, err := ds.VMValueFromJSON(data)
			if err != nil {
				return fmt.Errorf("解析 QQ 官方用户属性 %s 失败: %w", row.Id, err)
			}
			dict, ok := value.ReadDictData()
			if !ok {
				return fmt.Errorf("QQ 官方用户属性 %s 不是字典", row.Id)
			}
			if dict.Dict != nil {
				dict.Dict.Range(func(key string, value *ds.VMValue) bool {
					merged.Store(key, value)
					return true
				})
			}
		}
		return nil
	}); err != nil {
		return model.AttributesItemModel{}, err
	}

	data, err := ds.NewDictVal(merged).V().ToJSON()
	if err != nil {
		return model.AttributesItemModel{}, fmt.Errorf("序列化合并后的 QQ 官方用户属性失败: %w", err)
	}
	result := ordered[len(ordered)-1]
	result.Data = data
	result.AttrsType = "user"
	result.BindingSheetId = ""
	result.CreatedAt = createdAt
	result.UpdatedAt = updatedAt
	return result, nil
}

func (m *officialQQIdentityMigration) mergeCollidingUserAttrs(tx *gorm.DB) error {
	tx = officialQQIdentityMigrationWithoutCache(tx)
	byTarget := map[string][]model.AttributesItemModel{}
	err := officialQQIdentityMigrationFindInBatches(
		m.legacyAttrsQuery(tx).Select(officialQQIdentityMigrationAttrsMetadataColumns),
		func(rows []model.AttributesItemModel) error {
			for _, row := range rows {
				newID := m.migrateUserID(row.Id)
				if newID == row.Id {
					continue
				}
				byTarget[newID] = append(byTarget[newID], row)
			}
			return nil
		},
	)
	if err != nil {
		return err
	}

	targets := make([]string, 0, len(byTarget))
	for target, sources := range byTarget {
		if len(sources) > 1 {
			targets = append(targets, target)
		}
	}
	sort.Strings(targets)

	for _, target := range targets {
		sources := byTarget[target]
		var existing model.AttributesItemModel
		result := tx.Select(officialQQIdentityMigrationAttrsMetadataColumns).Where("id = ?", target).Limit(1).Find(&existing)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 0 {
			sources = append(sources, existing)
		}

		merged, err := mergeOfficialQQUserAttrs(tx, sources)
		if err != nil {
			return err
		}
		merged.Id = target
		merged.OwnerId = m.migrateUserID(merged.OwnerId)

		ids := make([]string, 0, len(sources))
		for _, source := range sources {
			ids = append(ids, source.Id)
		}
		if err := officialQQIdentityMigrationForEachChunk(ids, func(batchIDs []string) error {
			return tx.Where("id IN ?", batchIDs).Delete(&model.AttributesItemModel{}).Error
		}); err != nil {
			return err
		}
		if err := tx.Create(&merged).Error; err != nil {
			return err
		}
	}
	return nil
}

func (m *officialQQIdentityMigration) planCharacterNames(tx *gorm.DB) (map[string]string, error) {
	tx = officialQQIdentityMigrationWithoutCache(tx)
	type ownerName struct {
		owner string
		name  string
	}
	type candidate struct {
		id        string
		owner     string
		name      string
		createdAt int64
	}

	owners := map[string]struct{}{}
	groups := map[ownerName][]candidate{}
	err := officialQQIdentityMigrationFindInBatches(
		m.legacyAttrsQuery(tx).Select("id, attrs_type, name, owner_id, created_at"),
		func(rows []model.AttributesItemModel) error {
			for _, row := range rows {
				newOwnerID := m.migrateUserID(row.OwnerId)
				if row.AttrsType != "character" || newOwnerID == row.OwnerId {
					continue
				}
				owners[newOwnerID] = struct{}{}
				key := ownerName{owner: newOwnerID, name: row.Name}
				groups[key] = append(groups[key], candidate{
					id:        row.Id,
					owner:     newOwnerID,
					name:      row.Name,
					createdAt: row.CreatedAt,
				})
			}
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
	if len(owners) == 0 {
		return map[string]string{}, nil
	}

	ownerIDs := make([]string, 0, len(owners))
	for ownerID := range owners {
		ownerIDs = append(ownerIDs, ownerID)
	}

	usedNames := map[string]map[string]struct{}{}
	for ownerID := range owners {
		usedNames[ownerID] = map[string]struct{}{}
	}
	if err := officialQQIdentityMigrationForEachChunk(ownerIDs, func(batchOwnerIDs []string) error {
		return officialQQIdentityMigrationFindInBatches(
			tx.Select("id, owner_id, name").Where("attrs_type = ? AND owner_id IN ?", "character", batchOwnerIDs),
			func(rows []model.AttributesItemModel) error {
				for _, row := range rows {
					usedNames[row.OwnerId][row.Name] = struct{}{}
				}
				return nil
			},
		)
	}); err != nil {
		return nil, err
	}

	keys := make([]ownerName, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].owner != keys[j].owner {
			return keys[i].owner < keys[j].owner
		}
		return keys[i].name < keys[j].name
	})

	var duplicates []candidate
	for _, key := range keys {
		items := groups[key]
		sort.Slice(items, func(i, j int) bool {
			if items[i].createdAt != items[j].createdAt {
				return items[i].createdAt < items[j].createdAt
			}
			return items[i].id < items[j].id
		})
		if _, exists := usedNames[key.owner][key.name]; !exists {
			usedNames[key.owner][key.name] = struct{}{}
			items = items[1:]
		}
		duplicates = append(duplicates, items...)
	}

	sort.Slice(duplicates, func(i, j int) bool {
		if duplicates[i].owner != duplicates[j].owner {
			return duplicates[i].owner < duplicates[j].owner
		}
		if duplicates[i].createdAt != duplicates[j].createdAt {
			return duplicates[i].createdAt < duplicates[j].createdAt
		}
		return duplicates[i].id < duplicates[j].id
	})

	renames := map[string]string{}
	for _, item := range duplicates {
		for suffix := 2; ; suffix++ {
			newName := migratedCharacterName(item.name, suffix)
			if _, exists := usedNames[item.owner][newName]; exists {
				continue
			}
			usedNames[item.owner][newName] = struct{}{}
			renames[item.id] = newName
			break
		}
	}
	m.characterNames = renames
	return renames, nil
}

func (m *officialQQIdentityMigration) migrateDataDB(operator engine.DatabaseOperator) error {
	db := operator.GetDataDB(constant.WRITE)
	if db == nil {
		return nil
	}
	groupPrefix, groupUpperBound := m.oldGroupBounds()
	return db.Transaction(func(tx *gorm.DB) error {
		if hasTable(tx, &model.GroupInfo{}) {
			err := officialQQIdentityMigrationFindInBatches(
				tx.Select("id, data").Where("id >= ? AND id < ?", groupPrefix, groupUpperBound),
				func(rows []model.GroupInfo) error {
					for _, row := range rows {
						newID := m.migrateGroupID(row.ID)
						if newID == row.ID {
							continue
						}
						if err := ensureNoTarget(tx, &model.GroupInfo{}, "id = ?", newID); err != nil {
							return err
						}
						var group GroupInfo
						if err := json.Unmarshal(row.Data, &group); err != nil {
							return fmt.Errorf("解析群组 %s 失败: %w", row.ID, err)
						}
						m.migrateGroup(&group, row.ID)
						data, err := json.Marshal(&group)
						if err != nil {
							return err
						}
						if err := tx.Model(&model.GroupInfo{}).Where("id = ?", row.ID).Updates(map[string]any{"id": newID, "data": data}).Error; err != nil {
							return err
						}
					}
					return nil
				},
			)
			if err != nil {
				return err
			}
		}

		if hasTable(tx, &model.GroupPlayerInfoBase{}) {
			err := officialQQIdentityMigrationFindInBatches(
				tx.Select("id, group_id, user_id").Where("group_id >= ? AND group_id < ?", groupPrefix, groupUpperBound),
				func(rows []model.GroupPlayerInfoBase) error {
					for _, row := range rows {
						newGroupID := m.migrateGroupID(row.GroupID)
						newUserID := m.migrateContextUserID(row.GroupID, row.UserID)
						if err := tx.Model(&model.GroupPlayerInfoBase{}).Where("id = ?", row.ID).Updates(map[string]any{"group_id": newGroupID, "user_id": newUserID}).Error; err != nil {
							return err
						}
					}
					return nil
				},
			)
			if err != nil {
				return err
			}
		}

		if hasTable(tx, &model.AttributesItemModel{}) {
			characterNames, err := m.planCharacterNames(tx)
			if err != nil {
				return err
			}
			if mergeErr := m.mergeCollidingUserAttrs(tx); mergeErr != nil {
				return mergeErr
			}
			err = officialQQIdentityMigrationFindInBatches(
				m.legacyAttrsQuery(tx).Select("id, attrs_type, owner_id"),
				func(rows []model.AttributesItemModel) error {
					for _, row := range rows {
						newID := m.migrateAttrsID(row.Id, row.AttrsType)
						newOwnerID := m.migrateUserID(row.OwnerId)
						newName, rename := characterNames[row.Id]
						if newID == row.Id && newOwnerID == row.OwnerId && !rename {
							continue
						}
						if newID != row.Id {
							if targetErr := ensureNoTarget(tx, &model.AttributesItemModel{}, "id = ?", newID); targetErr != nil {
								return targetErr
							}
						}
						updates := map[string]any{"id": newID, "owner_id": newOwnerID}
						if rename {
							updates["name"] = newName
						}
						if updateErr := tx.Model(&model.AttributesItemModel{}).Where("id = ?", row.Id).Updates(updates).Error; updateErr != nil {
							return updateErr
						}
					}
					return nil
				},
			)
			if err != nil {
				return err
			}
		}

		if hasTable(tx, &model.BanInfo{}) {
			err := officialQQIdentityMigrationFindInBatches(
				tx.Select("id, data"),
				func(rows []model.BanInfo) error {
					for _, row := range rows {
						newID := m.migrateAnyID(row.ID)
						var item BanListInfoItem
						if err := json.Unmarshal(row.Data, &item); err != nil {
							continue
						}
						changed := m.migrateBanItem(&item) || newID != row.ID
						if !changed {
							continue
						}
						data, err := json.Marshal(&item)
						if err != nil {
							return err
						}
						if newID != row.ID {
							if err := ensureNoTarget(tx, &model.BanInfo{}, "id = ?", newID); err != nil {
								return err
							}
						}
						if err := tx.Model(&model.BanInfo{}).Where("id = ?", row.ID).Updates(map[string]any{"id": newID, "data": data}).Error; err != nil {
							return err
						}
					}
					return nil
				},
			)
			if err != nil {
				return err
			}
		}

		if hasTable(tx, &model.EndpointInfo{}) {
			var old model.EndpointInfo
			result := tx.Where("user_id = ?", m.oldEndpoint).Limit(1).Find(&old)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 0 {
				var target model.EndpointInfo
				targetResult := tx.Where("user_id = ?", m.newEndpoint).Limit(1).Find(&target)
				if targetResult.Error != nil {
					return targetResult.Error
				}
				if targetResult.RowsAffected == 0 {
					if err := tx.Model(&model.EndpointInfo{}).Where("user_id = ?", m.oldEndpoint).Update("user_id", m.newEndpoint).Error; err != nil {
						return err
					}
				} else {
					updates := map[string]any{
						"cmd_num":       max(old.CmdNum, target.CmdNum),
						"cmd_last_time": max(old.CmdLastTime, target.CmdLastTime),
						"online_time":   max(old.OnlineTime, target.OnlineTime),
						"updated_at":    max(old.UpdatedAt, target.UpdatedAt),
					}
					if err := tx.Model(&model.EndpointInfo{}).Where("user_id = ?", m.newEndpoint).Updates(updates).Error; err != nil {
						return err
					}
					if err := tx.Where("user_id = ?", m.oldEndpoint).Delete(&model.EndpointInfo{}).Error; err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
}

func (m *officialQQIdentityMigration) migrateLogDB(operator engine.DatabaseOperator) error {
	db := operator.GetLogDB(constant.WRITE)
	if db == nil {
		return nil
	}
	groupPrefix, groupUpperBound := m.oldGroupBounds()
	return db.Transaction(func(tx *gorm.DB) error {
		if hasTable(tx, &model.LogInfo{}) {
			err := officialQQIdentityMigrationFindInBatches(
				tx.Select("id, name, group_id").Where("group_id >= ? AND group_id < ?", groupPrefix, groupUpperBound),
				func(rows []model.LogInfo) error {
					for _, row := range rows {
						newGroupID := m.migrateGroupID(row.GroupID)
						if err := ensureNoTarget(tx, &model.LogInfo{}, "group_id = ? AND name = ?", newGroupID, row.Name); err != nil {
							return err
						}
						if err := tx.Model(&model.LogInfo{}).Where("id = ?", row.ID).UpdateColumn("group_id", newGroupID).Error; err != nil {
							return err
						}
					}
					return nil
				},
			)
			if err != nil {
				return err
			}
		}
		if hasTable(tx, &model.LogOneItem{}) {
			err := officialQQIdentityMigrationFindInBatches(
				tx.Select("id, group_id, im_userid, user_uniform_id").Where("group_id >= ? AND group_id < ?", groupPrefix, groupUpperBound),
				func(rows []model.LogOneItem) error {
					for _, row := range rows {
						updates := map[string]any{
							"group_id":        m.migrateGroupID(row.GroupID),
							"im_userid":       m.migrateContextUserID(row.GroupID, row.IMUserID),
							"user_uniform_id": m.migrateContextUserID(row.GroupID, row.UniformID),
						}
						if err := tx.Model(&model.LogOneItem{}).Where("id = ?", row.ID).Updates(updates).Error; err != nil {
							return err
						}
					}
					return nil
				},
			)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (m *officialQQIdentityMigration) migrateCensorDB(operator engine.DatabaseOperator) error {
	db := operator.GetCensorDB(constant.WRITE)
	if db == nil || !hasTable(db, &model.CensorLog{}) {
		return nil
	}
	groupPrefix, groupUpperBound := m.oldGroupBounds()
	return db.Transaction(func(tx *gorm.DB) error {
		err := officialQQIdentityMigrationFindInBatches(
			tx.Select("id, group_id, user_id").Where("group_id >= ? AND group_id < ?", groupPrefix, groupUpperBound),
			func(rows []model.CensorLog) error {
				for _, row := range rows {
					updates := map[string]any{
						"group_id": m.migrateGroupID(row.GroupID),
						"user_id":  m.migrateContextUserID(row.GroupID, row.UserID),
					}
					if err := tx.Model(&model.CensorLog{}).Where("id = ?", row.ID).Updates(updates).Error; err != nil {
						return err
					}
				}
				return nil
			},
		)
		if err != nil {
			return err
		}
		return nil
	})
}

func migrateBoolMapKeys(values *SyncMap[string, bool], migrate func(string) string) {
	if values == nil {
		return
	}
	type entry struct {
		old   string
		value bool
	}
	var entries []entry
	values.Range(func(key string, value bool) bool {
		if newKey := migrate(key); newKey != key {
			entries = append(entries, entry{old: key, value: value})
		}
		return true
	})
	for _, item := range entries {
		values.Delete(item.old)
		values.Store(migrate(item.old), item.value)
	}
}

func (m *officialQQIdentityMigration) migrateGroup(group *GroupInfo, oldGroupID string) {
	if group == nil {
		return
	}
	group.GroupID = m.migrateGroupID(oldGroupID)
	group.GuildID = m.migrateAnyID(group.GuildID)
	group.ChannelID = m.migrateAnyID(group.ChannelID)
	group.InviteUserID = m.migrateContextUserID(oldGroupID, group.InviteUserID)
	migrateBoolMapKeys(group.DiceIDActiveMap, m.migrateAnyID)
	migrateBoolMapKeys(group.DiceIDExistsMap, m.migrateAnyID)
	migrateBoolMapKeys(group.BotList, m.migrateAnyID)

	if group.Players != nil {
		type playerEntry struct {
			old    string
			player *GroupPlayerInfo
		}
		var players []playerEntry
		group.Players.Range(func(userID string, player *GroupPlayerInfo) bool {
			players = append(players, playerEntry{old: userID, player: player})
			return true
		})
		for _, item := range players {
			newUserID := m.migrateContextUserID(oldGroupID, item.old)
			if item.player != nil {
				item.player.UserID = newUserID
				item.player.GroupID = group.GroupID
			}
			if newUserID != item.old {
				group.Players.Delete(item.old)
				group.Players.Store(newUserID, item.player)
			}
		}
	}

	if group.PlayerGroups != nil {
		type teamEntry struct {
			old    string
			values []string
		}
		var teams []teamEntry
		group.PlayerGroups.Range(func(key string, values []string) bool {
			teams = append(teams, teamEntry{old: key, values: values})
			return true
		})
		for _, item := range teams {
			newKey := m.migrateAnyID(item.old)
			for index, value := range item.values {
				item.values[index] = m.migrateAnyID(value)
			}
			if newKey != item.old {
				group.PlayerGroups.Delete(item.old)
			}
			group.PlayerGroups.Store(newKey, item.values)
		}
	}
}

func (m *officialQQIdentityMigration) migrateBanItem(item *BanListInfoItem) bool {
	if item == nil {
		return false
	}
	changed := false
	newID := m.migrateAnyID(item.ID)
	if newID != item.ID {
		item.ID = newID
		changed = true
	}
	for index, place := range item.Places {
		newPlace := m.migrateAnyID(place)
		if newPlace != place {
			item.Places[index] = newPlace
			changed = true
		}
	}
	return changed
}

func (m *officialQQIdentityMigration) migrateMemory(d *Dice) {
	if d.ImSession != nil && d.ImSession.ServiceAtNew != nil {
		type groupEntry struct {
			old   string
			group *GroupInfo
		}
		var groups []groupEntry
		d.ImSession.ServiceAtNew.Range(func(groupID string, group *GroupInfo) bool {
			if _, ok := m.groups[groupID]; ok {
				groups = append(groups, groupEntry{old: groupID, group: group})
			}
			return true
		})
		for _, item := range groups {
			m.migrateGroup(item.group, item.old)
			d.ImSession.ServiceAtNew.Delete(item.old)
			d.ImSession.ServiceAtNew.Store(m.groups[item.old], item.group)
		}
	}
	if d.ImSession != nil && d.ImSession.PendingQuits != nil {
		type pendingQuitEntry struct {
			old  string
			info *PendingQuitInfo
		}
		var entries []pendingQuitEntry
		d.ImSession.PendingQuits.Range(func(key string, info *PendingQuitInfo) bool {
			groupID, endpointID, ok := strings.Cut(key, "\x00")
			if !ok {
				return true
			}
			newKey := makePendingQuitKey(m.migrateGroupID(groupID), m.migrateAnyID(endpointID))
			if newKey != key {
				entries = append(entries, pendingQuitEntry{old: key, info: info})
			}
			return true
		})
		for _, entry := range entries {
			groupID, endpointID, _ := strings.Cut(entry.old, "\x00")
			d.ImSession.PendingQuits.Delete(entry.old)
			d.ImSession.PendingQuits.Store(makePendingQuitKey(m.migrateGroupID(groupID), m.migrateAnyID(endpointID)), entry.info)
		}
	}

	if d.DirtyGroups != nil {
		type dirtyEntry struct {
			old   string
			value int64
		}
		var entries []dirtyEntry
		d.DirtyGroups.Range(func(groupID string, value int64) bool {
			if _, ok := m.groups[groupID]; ok {
				entries = append(entries, dirtyEntry{old: groupID, value: value})
			}
			return true
		})
		for _, item := range entries {
			d.DirtyGroups.Delete(item.old)
			d.DirtyGroups.Store(m.groups[item.old], item.value)
		}
	}

	if d.AttrsManager != nil {
		type attrsEntry struct {
			old  string
			item *AttributesItem
		}
		var entries []attrsEntry
		d.AttrsManager.m.Range(func(id string, item *AttributesItem) bool {
			if newName, ok := m.characterNames[id]; ok && item != nil {
				item.Name = newName
				item.IsSaved = true
			}
			newID := m.migrateAttrsID(id, "group_user")
			if newID != id {
				entries = append(entries, attrsEntry{old: id, item: item})
			}
			return true
		})
		for _, entry := range entries {
			newID := m.migrateAttrsID(entry.old, "group_user")
			d.AttrsManager.m.Delete(entry.old)
			entry.item.ID = newID
			d.AttrsManager.m.Store(newID, entry.item)
		}
	}

	if d.Config.BanList != nil && d.Config.BanList.Map != nil {
		type banEntry struct {
			old  string
			item *BanListInfoItem
		}
		var entries []banEntry
		d.Config.BanList.Map.Range(func(id string, item *BanListInfoItem) bool {
			if newID := m.migrateAnyID(id); newID != id {
				entries = append(entries, banEntry{old: id, item: item})
			} else {
				m.migrateBanItem(item)
			}
			return true
		})
		for _, entry := range entries {
			newID := m.migrateAnyID(entry.old)
			d.Config.BanList.Map.Delete(entry.old)
			m.migrateBanItem(entry.item)
			d.Config.BanList.Map.Store(newID, entry.item)
		}
	}

	for index, id := range d.DiceMasters {
		d.DiceMasters[index] = m.migrateAnyID(id)
	}
	for index, rawTarget := range d.Config.NoticeIDs {
		target := ParseNoticeTarget(rawTarget)
		target.ID = m.migrateAnyID(target.ID)
		d.Config.NoticeIDs[index] = target.String()
	}
	d.Config.UpgradeWindowID = m.migrateAnyID(d.Config.UpgradeWindowID)

	if d.Parent != nil {
		m.migrateNameCache(&d.Parent.GroupNameCache, m.migrateGroupID)
		m.migrateNameCache(&d.Parent.UserNameCache, m.migrateUserID)
	}
}

func (m *officialQQIdentityMigration) migrateNameCache(cache *SyncMap[string, *GroupNameCacheItem], migrate func(string) string) {
	if cache == nil {
		return
	}
	type cacheEntry struct {
		old  string
		item *GroupNameCacheItem
	}
	var entries []cacheEntry
	cache.Range(func(id string, item *GroupNameCacheItem) bool {
		if newID := migrate(id); newID != id {
			entries = append(entries, cacheEntry{old: id, item: item})
		}
		return true
	})
	for _, entry := range entries {
		cache.Delete(entry.old)
		cache.Store(migrate(entry.old), entry.item)
	}
}
