package service

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/pilagod/gorm-cursor-paginator/v2/paginator"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"sealdice-core/logger"
	"sealdice-core/model"
	"sealdice-core/utils/constant"
	engine2 "sealdice-core/utils/dboperator/engine"
)

type LogOne struct {
	// Version int           `json:"version,omitempty"`
	Items []model.LogOneItem `json:"items"`
	Info  model.LogInfo      `json:"info"`
}

type LinkState string

const (
	LinkStateNone  LinkState = "none"
	LinkStateFresh LinkState = "fresh"
	LinkStateStale LinkState = "stale"
)

var ErrLogNotFound = errors.New("日志不存在")

type CleanupPreview struct {
	Logs          int
	Items         int
	OldestUpdated int64
	NewestUpdated int64
	CanVacuum     bool
}

type CleanupResult struct {
	Logs     int
	Items    int
	Vacuumed bool
}

func BuildLogLinkState(logInfo *model.LogInfo) string {
	if logInfo == nil || logInfo.UploadURL == "" {
		return string(LinkStateNone)
	}
	if int64(logInfo.UploadTime) > logInfo.UpdatedAt {
		return string(LinkStateFresh)
	}
	return string(LinkStateStale)
}

// LogGetInfo 查询日志简略信息，使用通用函数替代SQLITE专属函数
// TODO: 换回去，因为现在已经分离了引擎
func LogGetInfo(operator engine2.DatabaseOperator) ([]int, error) {
	db := operator.GetLogDB(constant.READ)
	lst := []int{0, 0, 0, 0}

	var maxID sql.NullInt64      // 使用sql.NullInt64来处理NULL值
	var itemsMaxID sql.NullInt64 // 使用sql.NullInt64来处理NULL值
	// 获取 logs 表的记录数和最大 ID
	err := db.Model(&model.LogInfo{}).Select("COUNT(*)").Scan(&lst[2]).Error
	if err != nil {
		return nil, err
	}

	err = db.Model(&model.LogInfo{}).Select("MAX(id)").Scan(&maxID).Error
	if err != nil {
		return nil, err
	}
	lst[0] = int(maxID.Int64)

	// 获取 log_items 表的记录数和最大 ID
	err = db.Model(&model.LogOneItem{}).Select("COUNT(*)").Scan(&lst[3]).Error
	if err != nil {
		return nil, err
	}

	err = db.Model(&model.LogOneItem{}).Select("MAX(id)").Scan(&itemsMaxID).Error
	if err != nil {
		return nil, err
	}
	lst[1] = int(itemsMaxID.Int64)

	return lst, nil
}

// Deprecated: replaced by page
func LogGetLogs(operator engine2.DatabaseOperator) ([]*model.LogInfo, error) {
	db := operator.GetLogDB(constant.READ)
	var lst []*model.LogInfo

	// 使用 GORM 查询 logs 表
	if err := db.Model(&model.LogInfo{}).
		Select("id, name, group_id, created_at, updated_at").
		Find(&lst).Error; err != nil {
		return nil, err
	}

	return lst, nil
}

type QueryLogPage struct {
	PageNum          int    `db:"page_num"           query:"pageNum"`
	PageSize         int    `db:"page_siz"           query:"pageSize"`
	Name             string `db:"name"               query:"name"`
	GroupID          string `db:"group_id"           query:"groupId"`
	CreatedTimeBegin string `db:"created_time_begin" query:"createdTimeBegin"`
	CreatedTimeEnd   string `db:"created_time_end"   query:"createdTimeEnd"`
}

// LogGetLogPage 获取分页
func LogGetLogPage(operator engine2.DatabaseOperator, param *QueryLogPage) (int, []*model.LogInfo, error) {
	db := operator.GetLogDB(constant.READ)
	var lst []*model.LogInfo

	// 构建基础查询
	query := db.Model(&model.LogInfo{}).Select("logs.id, logs.name, logs.group_id, logs.created_at, logs.updated_at, COALESCE(logs.size, 0) as size, COALESCE(logs.upload_url, '') as upload_url, COALESCE(logs.upload_time, 0) as upload_time").Order("logs.updated_at desc")
	countQuery := db.Model(&model.LogInfo{})
	// 添加条件
	if param.Name != "" {
		query = query.Where("logs.name LIKE ?", "%"+param.Name+"%")
		countQuery = countQuery.Where("name LIKE ?", "%"+param.Name+"%")
	}
	if param.GroupID != "" {
		query = query.Where("logs.group_id LIKE ?", "%"+param.GroupID+"%")
		countQuery = countQuery.Where("group_id LIKE ?", "%"+param.GroupID+"%")
	}
	if param.CreatedTimeBegin != "" {
		query = query.Where("logs.created_at >= ?", param.CreatedTimeBegin)
		countQuery = countQuery.Where("created_at >= ?", param.CreatedTimeBegin)
	}
	if param.CreatedTimeEnd != "" {
		query = query.Where("logs.created_at <= ?", param.CreatedTimeEnd)
		countQuery = countQuery.Where("created_at <= ?", param.CreatedTimeEnd)
	}

	// 获取总数
	var count int64
	if err := countQuery.Count(&count).Error; err != nil {
		return 0, nil, err
	}

	// 分页查询
	query = query.Group("logs.id").Limit(param.PageSize).Offset((param.PageNum - 1) * param.PageSize)

	// 执行查询
	if err := query.Scan(&lst).Error; err != nil {
		return 0, nil, err
	}

	return int(count), lst, nil
}

// LogGetList 获取列表
func LogGetList(operator engine2.DatabaseOperator, groupID string) ([]string, error) {
	db := operator.GetLogDB(constant.READ)
	var lst []string

	// 执行查询
	if err := db.Model(&model.LogInfo{}).
		Select("name").
		Where("group_id = ?", groupID).
		Order("updated_at DESC").
		Pluck("name", &lst).Error; err != nil {
		return nil, err
	}

	return lst, nil
}

// getIDByGroupIDAndName 获取ID 私有函数，可以直接使用db
func getIDByGroupIDAndName(db *gorm.DB, groupID string, logName string) (logID uint64, err error) {
	var logInfo model.LogInfo
	err = db.Model(&model.LogInfo{}).
		Select("id").
		Where("group_id = ? AND name = ?", groupID, logName).
		Take(&logInfo).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, ErrLogNotFound
		}
		return 0, err
	}
	if logInfo.ID == 0 {
		return 0, ErrLogNotFound
	}
	return logInfo.ID, nil
}

func LogGetByGroupIDAndName(operator engine2.DatabaseOperator, groupID string, logName string) (*model.LogInfo, error) {
	db := operator.GetLogDB(constant.READ)
	var logInfo model.LogInfo
	err := db.Model(&model.LogInfo{}).
		Where("group_id = ? AND name = ?", groupID, logName).
		Take(&logInfo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLogNotFound
		}
		return nil, err
	}
	return &logInfo, nil
}

// LogGetUploadInfo 获取上传信息
func LogGetUploadInfo(operator engine2.DatabaseOperator, groupID string, logName string) (url string, uploadTime, updateTime int64, err error) {
	db := operator.GetLogDB(constant.READ)
	var logInfo struct {
		UpdatedAt  int64  `gorm:"column:updated_at"`
		UploadURL  string `gorm:"column:upload_url"`
		UploadTime int64  `gorm:"column:upload_time"`
	}

	err = db.Model(&model.LogInfo{}).
		Select("updated_at, upload_url, upload_time").
		Where("group_id = ? AND name = ?", groupID, logName).
		Find(&logInfo).Error

	if err != nil {
		return "", 0, 0, err
	}

	// 提取结果
	updateTime = logInfo.UpdatedAt
	url = logInfo.UploadURL
	uploadTime = logInfo.UploadTime
	return url, uploadTime, updateTime, err
}

func LogGetByID(operator engine2.DatabaseOperator, logID uint64) (*model.LogInfo, error) {
	db := operator.GetLogDB(constant.READ)
	var logInfo model.LogInfo
	err := db.Model(&model.LogInfo{}).
		Where("id = ?", logID).
		Take(&logInfo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrLogNotFound
		}
		return nil, err
	}
	return &logInfo, nil
}

func LogGetUploadInfoByID(operator engine2.DatabaseOperator, logID uint64) (url string, uploadTime, updateTime int64, err error) {
	logInfo, err := LogGetByID(operator, logID)
	if err != nil {
		return "", 0, 0, err
	}
	return logInfo.UploadURL, int64(logInfo.UploadTime), logInfo.UpdatedAt, nil
}

// LogSetUploadInfo 设置上传信息
func LogSetUploadInfo(operator engine2.DatabaseOperator, groupID string, logName string, url string) error {
	db := operator.GetLogDB(constant.WRITE)
	if len(url) == 0 {
		return nil
	}

	now := time.Now().Unix()

	// 使用 GORM 更新上传信息
	err := db.Model(&model.LogInfo{}).Where("group_id = ? AND name = ?", groupID, logName).
		Update("upload_url", url).
		Update("upload_time", now).
		Error

	return err
}

func LogSetUploadInfoByID(operator engine2.DatabaseOperator, logID uint64, url string) error {
	db := operator.GetLogDB(constant.WRITE)
	if len(url) == 0 {
		return nil
	}

	now := time.Now().Unix()
	return db.Model(&model.LogInfo{}).
		Where("id = ?", logID).
		Updates(map[string]interface{}{
			"upload_url":  url,
			"upload_time": now,
		}).Error
}

// LogGetAllLines 获取log的所有行数据
func CreateLoggerPaginator(
	cursor paginator.Cursor,
	order *paginator.Order,
) *paginator.Paginator {
	opts := []paginator.Option{
		&paginator.Config{
			Keys:  []string{"ID", "Time"}, // 这里设置的是结构体的字段名称，而不是gorm里的名称……
			Limit: 4000,
			Order: paginator.ASC,
		},
	}
	if order != nil {
		opts = append(opts, paginator.WithOrder(*order))
	}
	if cursor.After != nil {
		opts = append(opts, paginator.WithAfter(*cursor.After))
	}
	if cursor.Before != nil {
		opts = append(opts, paginator.WithBefore(*cursor.Before))
	}
	return paginator.New(opts...)
}

func LogGetAllLines(operator engine2.DatabaseOperator, groupID string, logName string) ([]*model.LogOneItem, error) {
	db := operator.GetLogDB(constant.READ)
	// 获取log的ID
	logID, err := getIDByGroupIDAndName(db, groupID, logName)
	if err != nil {
		if errors.Is(err, ErrLogNotFound) {
			return []*model.LogOneItem{}, nil
		}
		return nil, err
	}

	var items []*model.LogOneItem

	// 查询行数据
	err = db.Model(&model.LogOneItem{}).
		Select("id, nickname, im_userid, time, message, is_dice, command_id, command_info, raw_msg_id, user_uniform_id").
		Where("log_id = ?", logID).
		Order("time ASC").
		Find(&items).Error

	if err != nil {
		return nil, err
	}
	return items, nil
}

func LogGetCommandInfoStrList(operator engine2.DatabaseOperator, groupID string, logName string) ([]string, error) {
	db := operator.GetLogDB(constant.READ)
	// 获取log的ID
	logID, err := getIDByGroupIDAndName(db, groupID, logName)
	if err != nil {
		if errors.Is(err, ErrLogNotFound) {
			return []string{}, nil
		}
		return nil, err
	}

	var items []string

	// 查询行数据
	err = db.Model(&model.LogOneItem{}).
		Where("log_id = ?", logID).
		Order("time ASC").
		Pluck("command_info", &items).Error

	if err != nil {
		return nil, err
	}
	return items, nil
}

func LogGetCursorLines(operator engine2.DatabaseOperator, groupID string, logName string, cursor paginator.Cursor) ([]model.LogOneItem, paginator.Cursor, error) {
	db := operator.GetLogDB(constant.READ)
	// 获取log的ID
	logID, err := getIDByGroupIDAndName(db, groupID, logName)
	if err != nil {
		if errors.Is(err, ErrLogNotFound) {
			return []model.LogOneItem{}, paginator.Cursor{}, nil
		}
		return nil, paginator.Cursor{}, err
	}
	var items []model.LogOneItem
	stmt := db.Model(&model.LogOneItem{}).
		Select("id, nickname, im_userid, time, message, is_dice, command_id, command_info, raw_msg_id, user_uniform_id").
		Where("log_id = ?", logID)
	// 获取游标分页器
	p := CreateLoggerPaginator(cursor, nil)
	// 进行游标分页
	result, cursor, err := p.Paginate(stmt, &items)
	if err != nil {
		return nil, paginator.Cursor{}, err
	}
	// this is gorm error
	if result.Error != nil {
		return nil, paginator.Cursor{}, result.Error
	}
	return items, cursor, nil
}

func LogGetExportCursorLines(operator engine2.DatabaseOperator, groupID string, logName string, cursor paginator.Cursor) ([]model.LogOneItemParquet, paginator.Cursor, error) {
	db := operator.GetLogDB(constant.READ)
	// 获取log的ID
	logID, err := getIDByGroupIDAndName(db, groupID, logName)
	if err != nil {
		if errors.Is(err, ErrLogNotFound) {
			return []model.LogOneItemParquet{}, paginator.Cursor{}, nil
		}
		return nil, paginator.Cursor{}, err
	}
	var items []model.LogOneItemParquet
	stmt := db.Model(&model.LogOneItemParquet{}).
		Select("id, nickname, im_userid, time, message, is_dice, command_id, command_info, raw_msg_id, user_uniform_id").
		Where("log_id = ?", logID)
	// 获取游标分页器
	p := CreateLoggerPaginator(cursor, nil)
	// 进行游标分页
	result, cursor, err := p.Paginate(stmt, &items)
	if err != nil {
		return nil, paginator.Cursor{}, err
	}
	// this is gorm error
	if result.Error != nil {
		return nil, paginator.Cursor{}, result.Error
	}
	return items, cursor, nil
}

type QueryLogLinePage struct {
	LogID    uint64 `query:"logId"`
	PageNum  int    `query:"pageNum"`
	PageSize int    `query:"pageSize"`
	GroupID  string `query:"groupId"`
	LogName  string `query:"logName"`
}

// LogGetLinePage 获取log的行分页
func LogGetLinePage(operator engine2.DatabaseOperator, param *QueryLogLinePage) ([]*model.LogOneItem, error) {
	db := operator.GetLogDB(constant.READ)
	var (
		logID uint64
		err   error
	)
	if param.LogID != 0 {
		logID = param.LogID
	} else {
		logID, err = getIDByGroupIDAndName(db, param.GroupID, param.LogName)
	}
	if err != nil {
		if errors.Is(err, ErrLogNotFound) {
			return []*model.LogOneItem{}, nil
		}
		return nil, err
	}

	var items []*model.LogOneItem

	// 查询行数据
	err = db.Model(&model.LogOneItem{}).
		Select("id, nickname, im_userid, time, message, is_dice, command_id, command_info, raw_msg_id, user_uniform_id").
		Where("log_id = ?", logID).
		Order("time ASC").
		Limit(param.PageSize).
		Offset((param.PageNum - 1) * param.PageSize).
		Scan(&items).Error

	if err != nil {
		return nil, err
	}

	return items, nil
}

func LogGetLinePageByID(operator engine2.DatabaseOperator, logID uint64, pageNum int, pageSize int) ([]*model.LogOneItem, error) {
	return LogGetLinePage(operator, &QueryLogLinePage{
		LogID:    logID,
		PageNum:  pageNum,
		PageSize: pageSize,
	})
}

// LogLinesCountGet 获取日志行数
func LogLinesCountGet(operator engine2.DatabaseOperator, groupID string, logName string) (int64, bool) {
	db := operator.GetLogDB(constant.READ)
	// 获取日志 ID
	logID, err := getIDByGroupIDAndName(db, groupID, logName)
	if err != nil {
		return 0, false
	}

	// 获取日志行数
	var count int64
	err = db.Model(&model.LogOneItem{}).
		Where("log_id = ? and removed IS NULL", logID).
		Count(&count).Error

	if err != nil {
		return 0, false
	}

	return count, true
}

// LogDelete 删除log
func LogDelete(operator engine2.DatabaseOperator, groupID string, logName string) error {
	db := operator.GetLogDB(constant.WRITE)
	// 获取 log ID
	logID, err := getIDByGroupIDAndName(db, groupID, logName)
	if err != nil {
		return err
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		// 删除 log_id 相关的 log_items 记录
		if err = tx.Where("log_id = ?", logID).Delete(&model.LogOneItem{}).Error; err != nil {
			return err
		}

		// 删除 log_id 相关的 logs 记录
		if err = tx.Where("id = ?", logID).Delete(&model.LogInfo{}).Error; err != nil {
			return err
		}
		return nil
	})

	return err
}

func LogDeleteByID(operator engine2.DatabaseOperator, logID uint64) error {
	db := operator.GetLogDB(constant.WRITE)
	_, err := LogGetByID(operator, logID)
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err = tx.Where("log_id = ?", logID).Delete(&model.LogOneItem{}).Error; err != nil {
			return err
		}
		if err = tx.Where("id = ?", logID).Delete(&model.LogInfo{}).Error; err != nil {
			return err
		}
		return nil
	})
}

func LogGetAllLinesByID(operator engine2.DatabaseOperator, logID uint64) ([]*model.LogOneItem, error) {
	db := operator.GetLogDB(constant.READ)
	var items []*model.LogOneItem
	err := db.Model(&model.LogOneItem{}).
		Select("id, nickname, im_userid, time, message, is_dice, command_id, command_info, raw_msg_id, user_uniform_id").
		Where("log_id = ?", logID).
		Order("time ASC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func LogGetCleanupPreviewByMonths(operator engine2.DatabaseOperator, months int) (*CleanupPreview, error) {
	if months <= 0 {
		return &CleanupPreview{CanVacuum: operator.Type() == constant.SQLITE}, nil
	}
	db := operator.GetLogDB(constant.READ)
	cutoff := time.Now().AddDate(0, -months, 0).Unix()

	type result struct {
		Logs          int64 `gorm:"column:logs"`
		Items         int64 `gorm:"column:items"`
		OldestUpdated int64 `gorm:"column:oldest_updated"`
		NewestUpdated int64 `gorm:"column:newest_updated"`
	}
	var row result
	err := db.Table("logs").
		Joins("LEFT JOIN log_items ON log_items.log_id = logs.id").
		Select(`COUNT(DISTINCT logs.id) as logs,
COUNT(log_items.id) as items,
COALESCE(MIN(logs.updated_at), 0) as oldest_updated,
COALESCE(MAX(logs.updated_at), 0) as newest_updated`).
		Where("logs.updated_at <= ?", cutoff).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	return &CleanupPreview{
		Logs:          int(row.Logs),
		Items:         int(row.Items),
		OldestUpdated: row.OldestUpdated,
		NewestUpdated: row.NewestUpdated,
		CanVacuum:     operator.Type() == constant.SQLITE,
	}, nil
}

func LogCleanupByMonths(operator engine2.DatabaseOperator, months int, vacuum bool) (*CleanupResult, error) {
	preview, err := LogGetCleanupPreviewByMonths(operator, months)
	if err != nil {
		return nil, err
	}
	if preview.Logs == 0 {
		return &CleanupResult{Vacuumed: false}, nil
	}

	db := operator.GetLogDB(constant.WRITE)
	cutoff := time.Now().AddDate(0, -months, 0).Unix()
	var ids []uint64
	if err = db.Model(&model.LogInfo{}).Where("updated_at <= ?", cutoff).Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return &CleanupResult{Vacuumed: false}, nil
	}
	if err = db.Transaction(func(tx *gorm.DB) error {
		if err = tx.Where("log_id IN ?", ids).Delete(&model.LogOneItem{}).Error; err != nil {
			return err
		}
		if err = tx.Where("id IN ?", ids).Delete(&model.LogInfo{}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	result := &CleanupResult{
		Logs:  preview.Logs,
		Items: preview.Items,
	}
	if vacuum && operator.Type() == constant.SQLITE {
		if err = LogVacuum(operator); err != nil {
			return nil, err
		}
		result.Vacuumed = true
	}
	return result, nil
}

func LogVacuum(operator engine2.DatabaseOperator) error {
	if operator.Type() != constant.SQLITE {
		return nil
	}
	return operator.GetLogDB(constant.WRITE).Exec("VACUUM;").Error
}

// LogAppend 向指定的log中添加一条信息
func LogAppend(operator engine2.DatabaseOperator, groupID string, logName string, logItem *model.LogOneItem) bool {
	db := operator.GetLogDB(constant.WRITE)
	// 获取 log ID
	logID, err := getIDByGroupIDAndName(db, groupID, logName)
	if err != nil {
		if errors.Is(err, ErrLogNotFound) {
			logID = 0
		} else {
			return false
		}
	}

	// 获取当前时间戳
	now := time.Now()
	nowTimestamp := now.Unix()
	newLogItem := model.LogOneItem{
		LogID:       logID,
		GroupID:     groupID,
		Nickname:    logItem.Nickname,
		IMUserID:    logItem.IMUserID,
		Time:        nowTimestamp,
		Message:     logItem.Message,
		IsDice:      logItem.IsDice,
		CommandID:   logItem.CommandID,
		CommandInfo: logItem.CommandInfo,
		RawMsgID:    logItem.RawMsgID,
		UniformID:   logItem.UniformID,
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		if logID == 0 {
			// 创建一个新的 log
			newLog := model.LogInfo{Name: logName, GroupID: groupID, CreatedAt: nowTimestamp, UpdatedAt: nowTimestamp}
			if err = tx.Create(&newLog).Error; err != nil {
				// 并发场景下可能被其他事务先创建，回查既有log并继续写入
				existedLogID, findErr := getIDByGroupIDAndName(tx, groupID, logName)
				if findErr != nil {
					return err
				}
				logID = existedLogID
			} else {
				logID = newLog.ID
			}
		}
		newLogItem.LogID = logID
		if err = tx.Create(&newLogItem).Error; err != nil {
			return err
		}
		// 更新 logs 表中的 updated_at 字段 和 size 字段
		if err = tx.Model(&model.LogInfo{}).
			Where("id = ?", logID).
			Updates(map[string]interface{}{
				"updated_at": nowTimestamp,
				"size":       gorm.Expr("COALESCE(size, 0) + ?", 1),
			}).Error; err != nil {
			return err
		}
		return nil
	})

	return err == nil
}

// LogMarkDeleteByMsgID 撤回删除
func LogMarkDeleteByMsgID(operator engine2.DatabaseOperator, groupID string, logName string, rawID interface{}) error {
	db := operator.GetLogDB(constant.WRITE)
	// 获取 log id
	logID, err := getIDByGroupIDAndName(db, groupID, logName)
	if err != nil {
		if errors.Is(err, ErrLogNotFound) {
			return nil
		}
		return err
	}
	rid := fmt.Sprintf("%v", rawID)
	err = db.Transaction(func(tx *gorm.DB) error {
		if err = tx.Where("log_id = ? AND raw_msg_id = ?", logID, rid).Delete(&model.LogOneItem{}).Error; err != nil {
			zap.S().Named(logger.LogKeyDatabase).Errorf("log delete error %s", err.Error())
			return err
		}
		// 更新 logs 表中的 updated_at 字段 和 size 字段
		// 真的有默认为NULL还能触发删除的情况吗？！
		if err = tx.Model(&model.LogInfo{}).Where("id = ?", logID).Updates(map[string]interface{}{
			"updated_at": time.Now().Unix(),
			"size":       gorm.Expr("COALESCE(size, 0) - ?", 1),
		}).Error; err != nil {
			return err
		}
		return nil
	})
	return err
}

// LogEditByMsgID 编辑日志
func LogEditByMsgID(operator engine2.DatabaseOperator, groupID, logName, newContent string, rawID interface{}) error {
	db := operator.GetLogDB(constant.WRITE)
	logID, err := getIDByGroupIDAndName(db, groupID, logName)
	if err != nil {
		if errors.Is(err, ErrLogNotFound) {
			return nil
		}
		return err
	}

	rid := ""
	if rawID != nil {
		rid = fmt.Sprintf("%v", rawID)
	}

	// 更新 log_items 表中的内容
	if err := db.Model(&model.LogOneItem{}).
		Where("log_id = ? AND raw_msg_id = ?", logID, rid).
		Update("message", newContent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("log edit: %w", err)
	}

	return nil
}
