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

// LogGetInfo 查询日志简略信息（近似估算）
// 需求：此处无需精准统计，优先选择“轻量、近似”的获取方式，在失败时回退到通用查询。
// 返回结构保持不变：
//   lst[0] = logs 表最大ID（近似）；lst[1] = log_items 表最大ID（近似）；
//   lst[2] = logs 记录数（近似）；lst[3] = log_items 记录数（近似）。
// 近似方案按引擎选择：
// - SQLite：从 sqlite_sequence 读取 seq 作为最大ID与近似行数；缺失或失败时回退 MAX(id)/COUNT(*)。
// - MySQL：从 information_schema.tables 读取 AUTO_INCREMENT-1 作为最大ID，TABLE_ROWS 作为近似行数；失败时回退。
// - PostgreSQL：通过 pg_get_serial_sequence 获取序列，再读序列 last_value 作为最大ID；
//                通过 pg_class.reltuples 作为近似行数；失败时回退。
// - 其他引擎：以 MAX(id) 作为最大ID与近似行数的估算来源，失败时回退 COUNT(*)。
func LogGetInfo(operator engine2.DatabaseOperator) ([]int, error) {
    db := operator.GetLogDB(constant.READ)
    lst := []int{0, 0, 0, 0}

    // 近似最大ID与近似行数
    var maxID sql.NullInt64      // logs 最大ID（近似，可能为空）
    var itemsMaxID sql.NullInt64 // log_items 最大ID（近似，可能为空）
    var logsApproxRows sql.NullInt64
    var itemsApproxRows sql.NullInt64

    switch operator.Type() {
    case constant.SQLITE:
        // SQLite：seq 近似代表当前最大id，也可近似代表行数（有删除时会偏大）
        _ = db.Raw("SELECT seq FROM sqlite_sequence WHERE name = ?", "logs").Scan(&maxID).Error
        _ = db.Raw("SELECT seq FROM sqlite_sequence WHERE name = ?", "log_items").Scan(&itemsMaxID).Error
        // 行数近似采用 seq，失败时回退 COUNT(*)
        logsApproxRows = maxID
        itemsApproxRows = itemsMaxID
        if !logsApproxRows.Valid {
            if err := db.Model(&model.LogInfo{}).Select("COUNT(*)").Scan(&lst[2]).Error; err != nil {
                return nil, err
            }
        }
        if !itemsApproxRows.Valid {
            if err := db.Model(&model.LogOneItem{}).Select("COUNT(*)").Scan(&lst[3]).Error; err != nil {
                return nil, err
            }
        }

    case constant.MYSQL:
        // MySQL：AUTO_INCREMENT-1 近似最大ID；TABLE_ROWS 为近似行数（InnoDB 下为估算）。
        _ = db.Raw(
            "SELECT AUTO_INCREMENT - 1 FROM information_schema.tables WHERE TABLE_NAME = ? AND TABLE_SCHEMA = DATABASE()",
            "logs",
        ).Scan(&maxID).Error
        _ = db.Raw(
            "SELECT AUTO_INCREMENT - 1 FROM information_schema.tables WHERE TABLE_NAME = ? AND TABLE_SCHEMA = DATABASE()",
            "log_items",
        ).Scan(&itemsMaxID).Error
        _ = db.Raw(
            "SELECT TABLE_ROWS FROM information_schema.tables WHERE TABLE_NAME = ? AND TABLE_SCHEMA = DATABASE()",
            "logs",
        ).Scan(&logsApproxRows).Error
        _ = db.Raw(
            "SELECT TABLE_ROWS FROM information_schema.tables WHERE TABLE_NAME = ? AND TABLE_SCHEMA = DATABASE()",
            "log_items",
        ).Scan(&itemsApproxRows).Error
        // 如果近似值不可用，回退 COUNT(*)
        if !logsApproxRows.Valid {
            if err := db.Model(&model.LogInfo{}).Select("COUNT(*)").Scan(&lst[2]).Error; err != nil {
                return nil, err
            }
        }
        if !itemsApproxRows.Valid {
            if err := db.Model(&model.LogOneItem{}).Select("COUNT(*)").Scan(&lst[3]).Error; err != nil {
                return nil, err
            }
        }

    case constant.POSTGRESQL:
        // PostgreSQL：序列 last_value 近似最大ID；pg_class.reltuples 为近似行数。
        var seqName string
        if err := db.Raw("SELECT pg_get_serial_sequence('logs', 'id')").Scan(&seqName).Error; err == nil && len(seqName) > 0 {
            _ = db.Raw(fmt.Sprintf("SELECT last_value FROM %s", seqName)).Scan(&maxID).Error
        }
        seqName = ""
        if err := db.Raw("SELECT pg_get_serial_sequence('log_items', 'id')").Scan(&seqName).Error; err == nil && len(seqName) > 0 {
            _ = db.Raw(fmt.Sprintf("SELECT last_value FROM %s", seqName)).Scan(&itemsMaxID).Error
        }
        _ = db.Raw("SELECT reltuples::BIGINT FROM pg_class WHERE oid = 'logs'::regclass").Scan(&logsApproxRows).Error
        _ = db.Raw("SELECT reltuples::BIGINT FROM pg_class WHERE oid = 'log_items'::regclass").Scan(&itemsApproxRows).Error
        // 如果近似值不可用，回退 COUNT(*)
        if !logsApproxRows.Valid {
            if err := db.Model(&model.LogInfo{}).Select("COUNT(*)").Scan(&lst[2]).Error; err != nil {
                return nil, err
            }
        }
        if !itemsApproxRows.Valid {
            if err := db.Model(&model.LogOneItem{}).Select("COUNT(*)").Scan(&lst[3]).Error; err != nil {
                return nil, err
            }
        }

    default:
        // 未识别引擎：以 MAX(id) 近似行数与最大ID（失败时回退 COUNT(*)）
        _ = db.Model(&model.LogInfo{}).Select("MAX(id)").Scan(&maxID).Error
        _ = db.Model(&model.LogOneItem{}).Select("MAX(id)").Scan(&itemsMaxID).Error
        logsApproxRows = maxID
        itemsApproxRows = itemsMaxID
        if !logsApproxRows.Valid {
            if err := db.Model(&model.LogInfo{}).Select("COUNT(*)").Scan(&lst[2]).Error; err != nil {
                return nil, err
            }
        }
        if !itemsApproxRows.Valid {
            if err := db.Model(&model.LogOneItem{}).Select("COUNT(*)").Scan(&lst[3]).Error; err != nil {
                return nil, err
            }
        }
    }

    // 写入结果（如果表为空，NullInt64.Int64 为 0）
    lst[0] = int(maxID.Int64)
    lst[1] = int(itemsMaxID.Int64)
    // 近似行数赋值（若前面已填充确切值则保持）
    if lst[2] == 0 && logsApproxRows.Valid {
        lst[2] = int(logsApproxRows.Int64)
    }
    if lst[3] == 0 && itemsApproxRows.Valid {
        lst[3] = int(itemsApproxRows.Int64)
    }
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
	query := db.Model(&model.LogInfo{}).Select("logs.id, logs.name, logs.group_id, logs.created_at, logs.updated_at,COALESCE(logs.size, 0) as size").Order("logs.updated_at desc")
	// 添加条件
	if param.Name != "" {
		query = query.Where("logs.name LIKE ?", "%"+param.Name+"%")
	}
	if param.GroupID != "" {
		query = query.Where("logs.group_id LIKE ?", "%"+param.GroupID+"%")
	}
	if param.CreatedTimeBegin != "" {
		query = query.Where("logs.created_at >= ?", param.CreatedTimeBegin)
	}
	if param.CreatedTimeEnd != "" {
		query = query.Where("logs.created_at <= ?", param.CreatedTimeEnd)
	}

	// 获取总数
	var count int64
	if err := db.Model(&model.LogInfo{}).Count(&count).Error; err != nil {
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
	err = db.Model(&model.LogInfo{}).
		Select("id").
		Where("group_id = ? AND name = ?", groupID, logName).
		Scan(&logID).Error

	if err != nil {
		// 如果出现错误，判断是否没有找到对应的记录
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}

	return logID, nil
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
	PageNum  int    `query:"pageNum"`
	PageSize int    `query:"pageSize"`
	GroupID  string `query:"groupId"`
	LogName  string `query:"logName"`
}

// LogGetLinePage 获取log的行分页
func LogGetLinePage(operator engine2.DatabaseOperator, param *QueryLogLinePage) ([]*model.LogOneItem, error) {
	db := operator.GetLogDB(constant.READ)
	// 获取log的ID
	logID, err := getIDByGroupIDAndName(db, param.GroupID, param.LogName)
	if err != nil {
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

// LogLinesCountGet 获取日志行数
func LogLinesCountGet(operator engine2.DatabaseOperator, groupID string, logName string) (int64, bool) {
	db := operator.GetLogDB(constant.READ)
	// 获取日志 ID
	logID, err := getIDByGroupIDAndName(db, groupID, logName)
	if err != nil || logID == 0 {
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
func LogDelete(operator engine2.DatabaseOperator, groupID string, logName string) bool {
	db := operator.GetLogDB(constant.WRITE)
	// 获取 log ID
	logID, err := getIDByGroupIDAndName(db, groupID, logName)
	if err != nil || logID == 0 {
		return false
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

	return err == nil
}

// LogAppend 向指定的log中添加一条信息
func LogAppend(operator engine2.DatabaseOperator, groupID string, logName string, logItem *model.LogOneItem) bool {
	db := operator.GetLogDB(constant.WRITE)
	// 获取 log ID
	logID, err := getIDByGroupIDAndName(db, groupID, logName)
	if err != nil {
		return false
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
				return err
			}
			logID = newLog.ID
		}
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
