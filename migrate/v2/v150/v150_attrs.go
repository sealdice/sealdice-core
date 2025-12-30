package v150

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	ds "github.com/sealdice/dicescript"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"

	"sealdice-core/dice"
	"sealdice-core/dice/service"
	"sealdice-core/logger"
	"sealdice-core/model"
	"sealdice-core/utils"
	"sealdice-core/utils/constant"
	operator "sealdice-core/utils/dboperator/engine"
)

func convertToNew(name string, ownerId string, data []byte, updatedAt int64) (*model.AttributesItemModel, error) {
	var err error
	mapData := make(map[string]*dice.VMValue)

	if err = dice.JSONValueMapUnmarshal(data, &mapData); err == nil {
		var cardType string
		if val, ok := mapData["$cardType"]; ok {
			cardType, _ = val.ReadString()
		}

		// 卡片名字: $ch:xxxx  ctx.LoadPlayerGlobalVars
		// 归属: ownerId
		// 当前绑定: ctx.ChBindCur: 卡片角色名: $:ch-bind-data:name

		m2 := &ds.ValueMap{}
		for k, v := range mapData {
			if k == "$cardType" {
				continue
			}
			if k == "$:cardName" {
				continue
			}

			m2.Store(k, v.ConvertToV2())
		}

		var rawData []byte
		rawData, err = ds.NewDictVal(m2).V().ToJSON()

		if err != nil {
			return nil, err
		}

		item := &model.AttributesItemModel{
			Id:        utils.NewID(),
			Data:      rawData,
			AttrsType: service.AttrsTypeCharacter,

			// 这些是角色卡专用的
			Name:      name,
			OwnerId:   ownerId,
			SheetType: cardType,
			IsHidden:  false,

			CreatedAt: updatedAt,
			UpdatedAt: updatedAt,
		}

		return item, nil
	}

	return nil, err
}

// Key: GUID, Value: CardBindingID
var sheetIdBindByGroupUserId = map[string]string{}

// AttrsNewItem 新建一个角色卡/属性容器
func AttrsNewItem(db *gorm.DB, item *model.AttributesItemModel) (*model.AttributesItemModel, error) {
	if item.Id == "" {
		item.Id = utils.NewID()
	}
	err := db.Create(&item).Error
	return item, err
}

// 结构体
type V146RawStruct struct {
	ID        string `gorm:"column:id"`
	UpdatedAt int64  `gorm:"column:updated_at"`
	Data      []byte `gorm:"column:data"`
}

// 群组个人数据转换
func attrsGroupUserMigrate(db *gorm.DB) (int, int, error) {
	log := zap.S().Named(logger.LogKeyDatabase)
	rows, err := db.Table("attrs_group_user").Select("id, updated_at, data").Rows()
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	count := 0
	countFailed := 0
	for rows.Next() {
		var row V146RawStruct
		// ScanRows每次扫描一行。（反正他是这么说的……）
		if err = db.ScanRows(rows, &row); err != nil {
			log.Warnf("[损坏数据] 跳过一行数据，扫描失败: %v", err)
			countFailed++
			continue
		}
		// 跳过：① 无效 JSON ② 不是 JSON 对象 ③ ID 为空
		if res := gjson.ParseBytes(row.Data); !res.IsObject() || row.ID == "" {
			log.Warnf("[损坏数据] 跳过一行数据 (%v)-(%s)，用户GroupUser核心数据已经损坏", row.ID, string(row.Data))
			countFailed++
			continue
		}
		// 若发现更新时间为0，设置为当前时间
		if row.UpdatedAt == 0 {
			row.UpdatedAt = time.Now().Unix()
		}

		// id 为 GUID 即 GroupID-UserID
		_, userIdPart, ok := dice.UnpackGroupUserId(row.ID)
		if !ok {
			countFailed += 1
			log.Errorf("数据库读取出错，退出转换")
			log.Errorf("ID解析失败: %s", row.ID)
			continue
		}

		mapData := make(map[string]*dice.VMValue)
		err = dice.JSONValueMapUnmarshal(row.Data, &mapData)

		if err != nil {
			countFailed += 1
			fmt.Fprintln(os.Stdout, "解析失败: ", string(row.Data))
			continue
		}

		var cardType string
		if val, ok := mapData["$cardType"]; ok {
			cardType, _ = val.ReadString()
		}

		// 基础属性
		m := &ds.ValueMap{}
		for k, v := range mapData {
			if k == "$cardType" {
				continue
			}
			if k == "$:cardBindMark" {
				// 绑卡标记 直接跳过
				continue
			}
			m.Store(k, v.ConvertToV2())
		}

		var rawData []byte
		rawData, err = ds.NewDictVal(m).V().ToJSON()
		if err != nil {
			countFailed += 1
			fmt.Fprintf(os.Stdout, "群-用户 %s 的数据无法转换\n", row.ID)
			continue
		}

		item := &model.AttributesItemModel{
			Id:        row.ID,
			Data:      rawData,
			AttrsType: service.AttrsTypeGroupUser,

			// 当前组内绑定的卡
			BindingSheetId: sheetIdBindByGroupUserId[row.ID],

			// 这些是角色卡专用的
			Name:      "", // 群内默认卡，无名字，还是说以后弄成和nn的名字一致？
			OwnerId:   userIdPart,
			SheetType: cardType,
			IsHidden:  true,

			CreatedAt: row.UpdatedAt,
			UpdatedAt: row.UpdatedAt,
		}

		_, err = AttrsNewItem(db, item)
		if err != nil {
			countFailed += 1
		} else {
			count += 1
		}
	}

	// 检查循环过程中是否发生了错误
	if err = rows.Err(); err != nil {
		return count, countFailed, err
	}

	return count, countFailed, nil
}

// 群数据转换
func attrsGroupMigrate(db *gorm.DB) (int, int, error) {
	log := zap.S().Named(logger.LogKeyDatabase)

	// V146RawStruct
	rows, err := db.Table("attrs_group").Select("id, updated_at, data").Rows()
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	count := 0
	countFailed := 0
	for rows.Next() {
		var row V146RawStruct
		// ScanRows每次扫描一行。（反正他是这么说的……）
		if err = db.ScanRows(rows, &row); err != nil {
			log.Warnf("[损坏数据] 跳过一行数据，扫描失败: %v", err)
			countFailed++
			continue
		}
		// 跳过：① 无效 JSON ② 不是 JSON 对象 ③ ID 为空
		if res := gjson.ParseBytes(row.Data); !res.IsObject() || row.ID == "" {
			log.Warnf("[损坏数据] 跳过一行数据 (%v)-(%s)，用户Group核心数据已经损坏", row.ID, string(row.Data))
			countFailed++
			continue
		}
		// 若发现更新时间为0，设置为当前时间
		if row.UpdatedAt == 0 {
			row.UpdatedAt = time.Now().Unix()
		}

		mapData := make(map[string]*dice.VMValue)
		err = dice.JSONValueMapUnmarshal(row.Data, &mapData)

		if err != nil {
			countFailed += 1
			fmt.Fprintln(os.Stdout, "解析失败: ", string(row.Data))
			continue
		}

		// 基础属性
		m := &ds.ValueMap{}
		for k, v := range mapData {
			m.Store(k, v.ConvertToV2())
		}
		var rawData []byte
		rawData, err = ds.NewDictVal(m).V().ToJSON()
		if err != nil {
			countFailed += 1
			fmt.Fprintf(os.Stdout, "群 %s 的数据无法转换\n", row.ID)
			continue
		}

		item := &model.AttributesItemModel{
			Id:        row.ID,
			Data:      rawData,
			AttrsType: service.AttrsTypeGroup,

			IsHidden: true,

			CreatedAt: row.UpdatedAt,
			UpdatedAt: row.UpdatedAt,
		}

		_, err = AttrsNewItem(db, item)
		if err != nil {
			countFailed += 1
		} else {
			count += 1
		}
	}

	// 检查循环过程中是否发生了错误
	if err = rows.Err(); err != nil {
		return count, countFailed, err
	}

	return count, countFailed, nil
}

// 全局个人数据转换、对应attrs_user和玩家人物卡
func attrsUserMigrate(db *gorm.DB) (int, int, int, error) {
	log := zap.S().Named(logger.LogKeyDatabase)
	// 使用rows是因为146有莫名其妙的数据损坏问题，直接扫可能会把数据不小心丢进去
	rows, err := db.Table("attrs_user").Select("id,updated_at,data").Rows()
	if err != nil {
		return 0, 0, 0, err
	}
	defer rows.Close()

	count := 0
	countSheetsNum := 0
	countFailed := 0

	for rows.Next() {
		var row V146RawStruct
		// ScanRows每次扫描一行。（反正他是这么说的……）
		if err = db.ScanRows(rows, &row); err != nil {
			log.Warnf("[损坏数据] 跳过一行数据，扫描失败: %v", err)
			countFailed++
			continue
		}
		// 跳过：① 无效 JSON ② 不是 JSON 对象 ③ ID 为空
		if res := gjson.ParseBytes(row.Data); !res.IsObject() || row.ID == "" {
			log.Warnf("[损坏数据] 跳过一行数据 (%v)-(%s)，用户核心数据已经损坏", row.ID, string(row.Data))
			countFailed++
			continue
		}
		// 若发现更新时间为0，设置为当前时间
		if row.UpdatedAt == 0 {
			row.UpdatedAt = time.Now().Unix()
		}

		mapData := make(map[string]*dice.VMValue)
		err = dice.JSONValueMapUnmarshal(row.Data, &mapData)

		if err != nil {
			countFailed += 1
			continue
		}

		var newSheetsList []*model.AttributesItemModel
		var sheetNameBindByGroupId = map[string]string{}

		// 基础属性
		m := &ds.ValueMap{}
		for k, v := range mapData {
			if k == "$cardType" {
				continue
			}
			if k == "$:cardName" {
				continue
			}
			if strings.HasPrefix(k, "$:group-bind:") {
				// 这是绑卡信息，冒号后面的信息是GroupID，v是VMValue字符串
				// $:group-bind:群号  = 卡片名
				groupId := k[len("$:group-bind:"):]
				name, _ := v.ReadString()
				sheetNameBindByGroupId[groupId] = name
				continue
			}
			if strings.HasPrefix(k, "$ch:") {
				// 处理角色卡，这里 v 是 string
				var toNew *model.AttributesItemModel
				name := k[4:]

				toNew, err = convertToNew(name, row.ID, []byte(v.ToString()), row.UpdatedAt)
				if err != nil {
					fmt.Fprintf(os.Stdout, "用户 %s 的角色卡 %s 无法转换", row.ID, name)
					continue
				}
				newSheetsList = append(newSheetsList, toNew)
				continue
			}
			m.Store(k, v.ConvertToV2())
		}

		for _, i := range newSheetsList {
			// 一次性，双循环罢
			for groupID, j := range sheetNameBindByGroupId {
				if j == i.Name {
					sheetIdBindByGroupUserId[fmt.Sprintf("%s-%s", groupID, row.ID)] = i.Id
				}
			}
		}

		// 保存用户人物卡
		for _, i := range newSheetsList {
			_, err = AttrsNewItem(db, i)
			if err != nil {
				fmt.Fprintf(os.Stdout, "用户 %s 的角色卡 %s 无法写入数据库: %s\n", row.ID, i.Name, err.Error())
			}
		}

		countSheetsNum += len(newSheetsList)
		var rawData []byte
		rawData, err = ds.NewDictVal(m).V().ToJSON()
		if err != nil {
			countFailed += 1
			fmt.Fprintf(os.Stdout, "用户 %s 的个人数据无法转换\n", row.ID)
			continue
		}

		item := &model.AttributesItemModel{
			Id:        row.ID,
			Data:      rawData,
			AttrsType: service.AttrsTypeUser,
			// 非角色卡，而是用户卡，部分属性不存在

			IsHidden:  true,
			CreatedAt: row.UpdatedAt,
			UpdatedAt: row.UpdatedAt,
		}

		_, err = AttrsNewItem(db, item)
		if err != nil {
			countFailed += 1
		} else {
			count += 1
		}
	}

	// 检查循环过程中是否发生了错误
	if err = rows.Err(); err != nil {
		return count, countSheetsNum, countFailed, err
	}

	return count, countSheetsNum, countFailed, nil
}

func V150AttrsMigrate(dboperator operator.DatabaseOperator, logf func(string)) error {
	log := zap.S().Named(logger.LogKeyDatabase)
	err := dataDBInit(dboperator, logf)
	if err != nil {
		logf(fmt.Sprintf("数据表初始化失败: %v", err))
		return err
	}
	err = censorDBInit(dboperator, logf)
	if err != nil {
		logf(fmt.Sprintf("数据表初始化失败: %v", err))
		return err
	}
	err = logDBInit(dboperator, logf)
	if err != nil {
		logf(fmt.Sprintf("数据表初始化失败: %v", err))
		return err
	}
	dataDB := dboperator.GetDataDB(constant.WRITE)
	err = dataDB.Transaction(func(tx *gorm.DB) error {
		if tx.Migrator().HasTable("attrs_user") {
			count, countSheetsNum, countFailed, err0 := attrsUserMigrate(tx)
			log.Infof("数据卡转换 - 角色卡，成功人数%d 失败人数 %d 卡数 %d\n", count, countFailed, countSheetsNum)
			if err0 != nil {
				return fmt.Errorf("角色卡转换出错: %w", err0)
			}
			logf(fmt.Sprintf("数据卡转换 - 角色卡，成功人数%d 失败人数 %d 卡数 %d\n", count, countFailed, countSheetsNum))
		} else {
			logf("attrs_user表不存在，可能已经升级过！")
		}
		if tx.Migrator().HasTable("attrs_group_user") {
			count, countFailed, err1 := attrsGroupUserMigrate(tx)
			log.Infof("数据卡转换 - 群组个人数据，成功%d 失败 %d\n", count, countFailed)
			if err1 != nil {
				return fmt.Errorf("群组个人数据转换出错: %w", err1)
			}
		} else {
			logf("attrs_group_user表不存在，可能已经升级过！")
		}
		if tx.Migrator().HasTable("attrs_group") {
			count, countFailed, err2 := attrsGroupMigrate(tx)
			logf(fmt.Sprintf("数据卡转换 - 群组个人数据，成功%d 失败 %d\n", count, countFailed))
			log.Infof("数据卡转换 - 群数据，成功%d 失败 %d\n", count, countFailed)
			if err2 != nil {
				return fmt.Errorf("群数据转换出错: %w", err2)
			}
			logf(fmt.Sprintf("数据卡转换 - 群数据，成功%d 失败 %d\n", count, countFailed))
		} else {
			logf("attrs_group表不存在，可能已经升级过！")
		}
		// 删除
		_ = tx.Migrator().DropTable("attrs_group")
		_ = tx.Migrator().DropTable("attrs_group_user")
		_ = tx.Migrator().DropTable("attrs_user")
		logf("删除旧版本的历史遗留数据")
		return nil
	})
	sheetIdBindByGroupUserId = nil
	if err != nil {
		return err
	}

	// 如果是SQLITE，还需要执行
	if dboperator.Type() == "sqlite" {
		dataDB.Exec("PRAGMA wal_checkpoint(TRUNCATE);")
		dataDB.Exec("VACUUM;")
	}
	log.Info("V150 数据转换完成")
	logf("V150 数据转换完成")
	return nil
}

const createSql = `
CREATE TABLE attrs__temp (
    id TEXT PRIMARY KEY,
    data BYTEA,
    attrs_type TEXT,
    binding_sheet_id TEXT default '',
    name TEXT default '',
    owner_id TEXT default '',
    sheet_type TEXT default '',
    is_hidden BOOLEAN default FALSE,
    created_at INTEGER default 0,
    updated_at INTEGER default 0
);
`

// 初始化逻辑移动
func dataDBInit(dboperator operator.DatabaseOperator, logf func(string)) error {
	writeDB := dboperator.GetDataDB(constant.WRITE)
	readDB := dboperator.GetDataDB(constant.READ)
	// 如果是SQLITE建表，检查是否有这个影响的注释
	if dboperator.Type() == "sqlite" {
		var count int64
		err := readDB.Raw("SELECT count(*) FROM `sqlite_master` WHERE tbl_name = 'attrs' AND `sql` LIKE '%这个方法太严格了%'").Count(&count).Error
		if err != nil {
			return err
		}
		if count > 0 {
			// 特殊情况建表语句处置
			err = writeDB.Transaction(func(tx *gorm.DB) error {
				logf("数据库 attrs 表结构为前置测试版本150,重建中")
				// 创建临时表
				err = tx.Exec(createSql).Error
				if err != nil {
					return err
				}
				// 迁移数据
				err = tx.Exec("INSERT INTO `attrs__temp` SELECT * FROM `attrs`").Error
				if err != nil {
					return err
				}
				// 删除旧的表
				err = tx.Exec("DROP TABLE `attrs`").Error
				if err != nil {
					return err
				}
				// 改名
				err = tx.Exec("ALTER TABLE `attrs__temp` RENAME TO `attrs`").Error
				if err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
	}
	if dboperator.Type() == "sqlite" {
		if err := ensureSQLiteSimpleTable(writeDB, &model.GroupPlayerInfoBase{}); err != nil {
			return err
		}
		if err := ensureSQLiteSimpleTable(writeDB, &model.GroupInfo{}); err != nil {
			return err
		}
		if err := ensureSQLiteSimpleTable(writeDB, &model.BanInfo{}); err != nil {
			return err
		}
		if err := ensureSQLiteSimpleTable(writeDB, &model.EndpointInfo{}); err != nil {
			return err
		}
		if err := ensureSQLiteAttrsTable(writeDB); err != nil {
			return err
		}
	} else {
		if err := writeDB.AutoMigrate(
			&model.GroupPlayerInfoBase{},
			&model.GroupInfo{},
			&model.BanInfo{},
			&model.EndpointInfo{},
			&model.AttributesItemModel{},
		); err != nil {
			return err
		}
	}

	logf("DataDB 数据表初始化完成")
	return nil
}

// 利用前缀索引，规避索引BUG
// 创建不出来也没关系，反正MYSQL数据库
func createMySQLIndexForLogInfo(db *gorm.DB) (err error) {
	log := zap.S().Named(logger.LogKeyDatabase)
	// 创建前缀索引
	// 检查并创建索引
	if !db.Migrator().HasIndex(&model.LogInfoHookMySQL{}, "idx_log_name") {
		err = db.Exec("CREATE INDEX idx_log_name ON logs (name(20));").Error
		if err != nil {
			log.Errorf("创建idx_log_name索引失败,原因为 %v", err)
		}
	}

	if !db.Migrator().HasIndex(&model.LogInfoHookMySQL{}, "idx_logs_group") {
		err = db.Exec("CREATE INDEX idx_logs_group ON logs (group_id(20));").Error
		if err != nil {
			log.Errorf("创建idx_logs_group索引失败,原因为 %v", err)
		}
	}

	if !db.Migrator().HasIndex(&model.LogInfoHookMySQL{}, "idx_logs_updated_at") {
		err = db.Exec("CREATE INDEX idx_logs_updated_at ON logs (updated_at);").Error
		if err != nil {
			log.Errorf("创建idx_logs_updated_at索引失败,原因为 %v", err)
		}
	}
	return nil
}

func createMySQLIndexForLogOneItem(db *gorm.DB) (err error) {
	log := zap.S().Named(logger.LogKeyDatabase)
	// 创建前缀索引
	// 检查并创建索引
	if !db.Migrator().HasIndex(&model.LogOneItemHookMySQL{}, "idx_log_items_group_id") {
		err = db.Exec("CREATE INDEX idx_log_items_group_id ON log_items(group_id(20))").Error
		if err != nil {
			log.Errorf("创建idx_logs_group索引失败,原因为 %v", err)
		}
	}
	if !db.Migrator().HasIndex(&model.LogOneItemHookMySQL{}, "idx_raw_msg_id") {
		err = db.Exec("CREATE INDEX idx_raw_msg_id ON log_items(raw_msg_id(20))").Error
		if err != nil {
			log.Errorf("创建idx_log_group_id_name索引失败,原因为 %v", err)
		}
	}
	// MYSQL似乎不能创建前缀联合索引，放弃所有的前缀联合索引
	return nil
}

const (
	sqliteLogsTempTable     = "logs__tmp_v150"
	sqliteAttrsTempTable    = "attrs__tmp_v150"
	sqliteLogItemsTempTable = "log_items__tmp_v150"
	sqliteCopyBatchSize     = int64(500)
)

type sqlitePragmaColumn struct {
	Name    string         `gorm:"column:name"`
	Type    string         `gorm:"column:type"`
	NotNull int            `gorm:"column:notnull"`
	Default sql.NullString `gorm:"column:dflt_value"`
	PK      int            `gorm:"column:pk"`
}

type sqliteExpectedColumn struct {
	Name       string
	Type       string
	PrimaryKey bool
}

var expectedSQLiteLogsColumns = []sqliteExpectedColumn{
	{Name: "id", Type: "INTEGER", PrimaryKey: true},
	{Name: "name", Type: "TEXT"},
	{Name: "group_id", Type: "TEXT"},
	{Name: "created_at", Type: "INTEGER"},
	{Name: "updated_at", Type: "INTEGER"},
	{Name: "size", Type: "INTEGER"},
	{Name: "extra", Type: "TEXT"},
	{Name: "upload_url", Type: "TEXT"},
	{Name: "upload_time", Type: "INTEGER"},
}

var expectedSQLiteAttrsColumns = []sqliteExpectedColumn{
	{Name: "id", Type: "TEXT", PrimaryKey: true},
	{Name: "data", Type: "BLOB"},
	{Name: "attrs_type", Type: "TEXT"},
	{Name: "binding_sheet_id", Type: "TEXT"},
	{Name: "name", Type: "TEXT"},
	{Name: "owner_id", Type: "TEXT"},
	{Name: "sheet_type", Type: "TEXT"},
	{Name: "is_hidden", Type: "NUMERIC"},
	{Name: "created_at", Type: "INTEGER"},
	{Name: "updated_at", Type: "INTEGER"},
}

var expectedSQLiteLogItemsColumns = []sqliteExpectedColumn{
	{Name: "id", Type: "INTEGER", PrimaryKey: true},
	{Name: "log_id", Type: "INTEGER"},
	{Name: "group_id", Type: "TEXT"},
	{Name: "nickname", Type: "TEXT"},
	{Name: "im_userid", Type: "TEXT"},
	{Name: "time", Type: "INTEGER"},
	{Name: "message", Type: "TEXT"},
	{Name: "is_dice", Type: "INTEGER"},
	{Name: "command_id", Type: "INTEGER"},
	{Name: "command_info", Type: "TEXT"},
	{Name: "raw_msg_id", Type: "TEXT"},
	{Name: "user_uniform_id", Type: "TEXT"},
	{Name: "removed", Type: "INTEGER"},
	{Name: "parent_id", Type: "INTEGER"},
}

func ensureSQLiteLogSchema(db *gorm.DB) error {
	// 处理列表表
	if err := ensureSQLiteLogsTable(db); err != nil {
		return err
	}
	// 处理日志项表
	if err := ensureSQLiteLogItemsTable(db); err != nil {
		return err
	}
	return nil
}

func ensureSQLiteLogsTable(db *gorm.DB) error {
	if !db.Migrator().HasTable("logs") {
		if err := createSQLiteLogsTable(db, "logs"); err != nil {
			return err
		}
		return ensureSQLiteLogsIndexes(db)
	}

	columns, err := loadSQLiteTableColumns(db, "logs")
	if err != nil {
		return err
	}

	if !sqliteColumnsMatch(columns, expectedSQLiteLogsColumns) {
		if err := recreateSQLiteLogsTable(db, columns); err != nil {
			return err
		}
	}

	return ensureSQLiteLogsIndexes(db)
}

func ensureSQLiteLogsIndexes(db *gorm.DB) error {
	stmts := []string{
		"CREATE INDEX IF NOT EXISTS idx_logs_group ON `logs` (group_id)",
		"CREATE INDEX IF NOT EXISTS idx_logs_updated_at ON `logs` (updated_at)",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_log_group_id_name ON `logs` (group_id, name)",
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

func recreateSQLiteLogsTable(db *gorm.DB, actual []sqlitePragmaColumn) error {
	if err := dropSQLiteIndexes(db, []string{"idx_logs_group", "idx_logs_updated_at", "idx_log_group_id_name"}); err != nil {
		return err
	}
	actualMap := make(map[string]sqlitePragmaColumn, len(actual))
	for _, col := range actual {
		actualMap[strings.ToLower(col.Name)] = col
	}
	if _, ok := actualMap["id"]; !ok {
		return errors.New("logs 表缺少 id 列，无法迁移")
	}

	if err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quoteSQLiteIdentifier(sqliteLogsTempTable))).Error; err != nil {
		return err
	}
	if err := createSQLiteLogsTable(db, sqliteLogsTempTable); err != nil {
		return err
	}

	insertColumns := make([]string, 0, len(expectedSQLiteLogsColumns))
	selectColumns := make([]string, 0, len(expectedSQLiteLogsColumns))
	for _, exp := range expectedSQLiteLogsColumns {
		insertColumns = append(insertColumns, quoteSQLiteIdentifier(exp.Name))
		if _, ok := actualMap[strings.ToLower(exp.Name)]; ok {
			selectColumns = append(selectColumns, quoteSQLiteIdentifier(exp.Name))
		} else {
			selectColumns = append(selectColumns, defaultValueForMissingColumn(exp.Name))
		}
	}

	if err := bulkCopySQLiteTable(db, "logs", sqliteLogsTempTable, insertColumns, selectColumns); err != nil {
		return err
	}

	if err := db.Exec(fmt.Sprintf("DROP TABLE %s", quoteSQLiteIdentifier("logs"))).Error; err != nil {
		return err
	}
	if err := db.Exec(fmt.Sprintf("ALTER TABLE %s RENAME TO %s", quoteSQLiteIdentifier(sqliteLogsTempTable), quoteSQLiteIdentifier("logs"))).Error; err != nil {
		return err
	}
	return nil
}

func ensureSQLiteLogItemsTable(db *gorm.DB) error {
	if !db.Migrator().HasTable("log_items") {
		if err := createSQLiteLogItemsTable(db, "log_items"); err != nil {
			return err
		}
		return ensureSQLiteLogItemsIndexes(db)
	}

	columns, err := loadSQLiteTableColumns(db, "log_items")
	if err != nil {
		return err
	}

	if !sqliteColumnsMatch(columns, expectedSQLiteLogItemsColumns) {
		if err := recreateSQLiteLogItemsTable(db, columns); err != nil {
			return err
		}
	}

	return ensureSQLiteLogItemsIndexes(db)
}

func loadSQLiteTableColumns(db *gorm.DB, table string) ([]sqlitePragmaColumn, error) {
	query := fmt.Sprintf("PRAGMA table_info(%s)", quoteSQLiteString(table))
	var columns []sqlitePragmaColumn
	if err := db.Raw(query).Scan(&columns).Error; err != nil {
		return nil, err
	}
	return columns, nil
}

func sqliteColumnsMatch(actual []sqlitePragmaColumn, expected []sqliteExpectedColumn) bool {
	if len(actual) != len(expected) {
		return false
	}
	actualMap := make(map[string]sqlitePragmaColumn, len(actual))
	for _, col := range actual {
		actualMap[strings.ToLower(col.Name)] = col
	}
	for _, exp := range expected {
		col, ok := actualMap[strings.ToLower(exp.Name)]
		if !ok {
			return false
		}
		if normalizeSQLiteType(col.Type) != normalizeSQLiteType(exp.Type) {
			return false
		}
		if exp.PrimaryKey != (col.PK != 0) {
			return false
		}
	}
	return true
}

func normalizeSQLiteType(t string) string {
	n := strings.ToUpper(strings.TrimSpace(t))
	n = strings.ReplaceAll(n, " PRIMARY KEY", "")
	n = strings.ReplaceAll(n, " AUTOINCREMENT", "")
	n = strings.ReplaceAll(n, " UNSIGNED", "")
	n = strings.TrimSpace(n)
	switch n {
	case "NUMERIC":
		return "INTEGER"
	case "BOOL", "BOOLEAN":
		return "INTEGER"
	default:
		return n
	}
}
func ensureSQLiteSimpleTable(db *gorm.DB, model interface{}) error {
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(model); err != nil {
		return err
	}
	table := stmt.Schema.Table
	if table == "" {
		table = stmt.Table
	}
	if table == "" {
		return fmt.Errorf("unable to determine table name for model %T", model)
	}
	if !db.Migrator().HasTable(table) {
		return db.AutoMigrate(model)
	}
	columns, err := loadSQLiteTableColumns(db, table)
	if err != nil {
		return err
	}
	expected := make([]string, 0, len(stmt.Schema.Fields))
	for _, field := range stmt.Schema.Fields {
		if field.IgnoreMigration {
			continue
		}
		if field.DBName != "" {
			expected = append(expected, strings.ToLower(field.DBName))
		}
	}
	if !sqliteColumnNamesMatch(columns, expected) {
		return db.AutoMigrate(model)
	}
	return nil
}

func sqliteColumnNamesMatch(actual []sqlitePragmaColumn, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	actualSet := make(map[string]struct{}, len(actual))
	for _, col := range actual {
		actualSet[strings.ToLower(col.Name)] = struct{}{}
	}
	for _, name := range expected {
		if _, ok := actualSet[strings.ToLower(name)]; !ok {
			return false
		}
	}
	return true
}

func recreateSQLiteLogItemsTable(db *gorm.DB, actual []sqlitePragmaColumn) error {
	if err := dropSQLiteIndexes(db, []string{"idx_log_items_group_id", "idx_log_items_log_id", "idx_raw_msg_id"}); err != nil {
		return err
	}
	actualMap := make(map[string]sqlitePragmaColumn, len(actual))
	for _, col := range actual {
		actualMap[strings.ToLower(col.Name)] = col
	}
	if _, ok := actualMap["id"]; !ok {
		return errors.New("log_items 表缺少 id 列，无法迁移")
	}

	if err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quoteSQLiteIdentifier(sqliteLogItemsTempTable))).Error; err != nil {
		return err
	}
	if err := createSQLiteLogItemsTable(db, sqliteLogItemsTempTable); err != nil {
		return err
	}

	insertColumns := make([]string, 0, len(expectedSQLiteLogItemsColumns))
	selectColumns := make([]string, 0, len(expectedSQLiteLogItemsColumns))
	for _, exp := range expectedSQLiteLogItemsColumns {
		insertColumns = append(insertColumns, quoteSQLiteIdentifier(exp.Name))
		if _, ok := actualMap[strings.ToLower(exp.Name)]; ok {
			selectColumns = append(selectColumns, quoteSQLiteIdentifier(exp.Name))
		} else {
			selectColumns = append(selectColumns, defaultValueForMissingColumn(exp.Name))
		}
	}

	if err := bulkCopySQLiteTable(db, "log_items", sqliteLogItemsTempTable, insertColumns, selectColumns); err != nil {
		return err
	}

	if err := db.Exec(fmt.Sprintf("DROP TABLE %s", quoteSQLiteIdentifier("log_items"))).Error; err != nil {
		return err
	}
	if err := db.Exec(fmt.Sprintf("ALTER TABLE %s RENAME TO %s", quoteSQLiteIdentifier(sqliteLogItemsTempTable), quoteSQLiteIdentifier("log_items"))).Error; err != nil {
		return err
	}
	return nil
}
func ensureSQLiteAttrsTable(db *gorm.DB) error {
	if !db.Migrator().HasTable("attrs") {
		if err := createSQLiteAttrsTable(db, "attrs"); err != nil {
			return err
		}
		return ensureSQLiteAttrsIndexes(db)
	}

	columns, err := loadSQLiteTableColumns(db, "attrs")
	if err != nil {
		return err
	}

	if !sqliteColumnsMatch(columns, expectedSQLiteAttrsColumns) {
		if err := recreateSQLiteAttrsTable(db, columns); err != nil {
			return err
		}
	}

	return ensureSQLiteAttrsIndexes(db)
}

func ensureSQLiteAttrsIndexes(db *gorm.DB) error {
	stmts := []string{
		"CREATE INDEX IF NOT EXISTS idx_attrs_attrs_type_id ON `attrs` (attrs_type)",
		"CREATE INDEX IF NOT EXISTS idx_attrs_binding_sheet_id ON `attrs` (binding_sheet_id)",
		"CREATE INDEX IF NOT EXISTS idx_attrs_owner_id_id ON `attrs` (owner_id)",
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

func recreateSQLiteAttrsTable(db *gorm.DB, actual []sqlitePragmaColumn) error {
	if err := dropSQLiteIndexes(db, []string{"idx_attrs_attrs_type_id", "idx_attrs_binding_sheet_id", "idx_attrs_owner_id_id"}); err != nil {
		return err
	}
	actualMap := make(map[string]sqlitePragmaColumn, len(actual))
	for _, col := range actual {
		actualMap[strings.ToLower(col.Name)] = col
	}
	if _, ok := actualMap["id"]; !ok {
		return errors.New("attrs 表缺少 id 列，无法迁移")
	}

	if err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quoteSQLiteIdentifier(sqliteAttrsTempTable))).Error; err != nil {
		return err
	}
	if err := createSQLiteAttrsTable(db, sqliteAttrsTempTable); err != nil {
		return err
	}

	insertColumns := make([]string, 0, len(expectedSQLiteAttrsColumns))
	selectColumns := make([]string, 0, len(expectedSQLiteAttrsColumns))
	for _, exp := range expectedSQLiteAttrsColumns {
		insertColumns = append(insertColumns, quoteSQLiteIdentifier(exp.Name))
		if _, ok := actualMap[strings.ToLower(exp.Name)]; ok {
			selectColumns = append(selectColumns, quoteSQLiteIdentifier(exp.Name))
		} else {
			selectColumns = append(selectColumns, defaultValueForMissingColumn(exp.Name))
		}
	}

	if err := bulkCopySQLiteTable(db, "attrs", sqliteAttrsTempTable, insertColumns, selectColumns); err != nil {
		return err
	}

	if err := db.Exec(fmt.Sprintf("DROP TABLE %s", quoteSQLiteIdentifier("attrs"))).Error; err != nil {
		return err
	}
	if err := db.Exec(fmt.Sprintf("ALTER TABLE %s RENAME TO %s", quoteSQLiteIdentifier(sqliteAttrsTempTable), quoteSQLiteIdentifier("attrs"))).Error; err != nil {
		return err
	}
	return nil
}

func createSQLiteAttrsTable(db *gorm.DB, table string) error {
	session := db.Session(&gorm.Session{})
	if table != "" {
		session = session.Table(table)
	}
	err := session.Migrator().CreateTable(&model.AttributesItemModel{})
	if err != nil && strings.Contains(err.Error(), "already exists") {
		// 忽略索引或表已存在的错误，这在迁移过程中是正常的
		return nil
	}
	return err
}

func createSQLiteLogsTable(db *gorm.DB, table string) error {
	session := db.Session(&gorm.Session{})
	if table != "" {
		session = session.Table(table)
	}
	err := session.Migrator().CreateTable(&model.LogInfo{})
	if err != nil && strings.Contains(err.Error(), "already exists") {
		// 忽略索引或表已存在的错误，这在迁移过程中是正常的
		return nil
	}
	return err
}

func createSQLiteLogItemsTable(db *gorm.DB, table string) error {
	session := db.Session(&gorm.Session{})
	if table != "" {
		session = session.Table(table)
	}
	err := session.Migrator().CreateTable(&model.LogOneItem{})
	if err != nil && strings.Contains(err.Error(), "already exists") {
		// 忽略索引或表已存在的错误，这在迁移过程中是正常的
		return nil
	}
	return err
}

func bulkCopySQLiteTable(db *gorm.DB, src, dst string, insertColumns, selectColumns []string) error {
	if len(insertColumns) == 0 || len(selectColumns) == 0 {
		return nil
	}

	var minRow struct {
		Value sql.NullInt64 `gorm:"column:value"`
	}
	var maxRow struct {
		Value sql.NullInt64 `gorm:"column:value"`
	}

	minQuery := fmt.Sprintf("SELECT rowid AS value FROM %s ORDER BY rowid ASC LIMIT 1", quoteSQLiteIdentifier(src))
	if err := db.Raw(minQuery).Scan(&minRow).Error; err != nil {
		return err
	}

	maxQuery := fmt.Sprintf("SELECT rowid AS value FROM %s ORDER BY rowid DESC LIMIT 1", quoteSQLiteIdentifier(src))
	if err := db.Raw(maxQuery).Scan(&maxRow).Error; err != nil {
		return err
	}

	if !minRow.Value.Valid || !maxRow.Value.Valid {
		return nil
	}

	insertClause := strings.Join(insertColumns, ",")
	selectClause := strings.Join(selectColumns, ",")

	silentButError := gormLogger.New(
		log.New(os.Stderr, "", 0),
		gormLogger.Config{
			LogLevel: gormLogger.Error, // 只打印 ERROR 及以上
		},
	)

	idDst := quoteSQLiteIdentifier(dst)
	idSrc := quoteSQLiteIdentifier(src)
	log := zap.S().Named(logger.LogKeyDatabase)
	for startRow := minRow.Value.Int64; startRow <= maxRow.Value.Int64; startRow += sqliteCopyBatchSize {
		endRow := startRow + sqliteCopyBatchSize - 1
		copySQL := fmt.Sprintf(
			"INSERT INTO %s (%s) SELECT %s FROM %s WHERE rowid BETWEEN ? AND ?",
			idDst,
			insertClause,
			selectClause,
			idSrc,
		)
		if err := db.Session(&gorm.Session{Logger: silentButError}).
			Exec(copySQL, startRow, endRow).Error; err != nil {
			return err
		}

		val1 := startRow - minRow.Value.Int64
		val2 := maxRow.Value.Int64 - minRow.Value.Int64
		log.Infof("已迁移 %d/%d 行到 %s - %.2f%%", val1, val2, idDst, float64(val1)/float64(val2)*100)
	}

	return nil
}

func dropSQLiteIndexes(db *gorm.DB, indexes []string) error {
	for _, name := range indexes {
		stmt := fmt.Sprintf("DROP INDEX IF EXISTS %s", quoteSQLiteIdentifier(name))
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

func ensureSQLiteLogItemsIndexes(db *gorm.DB) error {
	stmts := []string{
		"CREATE INDEX IF NOT EXISTS idx_log_items_group_id ON `log_items` (group_id)",
		"CREATE INDEX IF NOT EXISTS idx_log_items_log_id ON `log_items` (log_id)",
		"CREATE INDEX IF NOT EXISTS idx_raw_msg_id ON `log_items` (raw_msg_id)",
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

func quoteSQLiteIdentifier(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func quoteSQLiteString(name string) string {
	return "'" + strings.ReplaceAll(name, "'", "''") + "'"
}

func defaultValueForMissingColumn(name string) string {
	switch name {
	case "log_id", "command_id", "time", "is_dice", "removed", "parent_id", "created_at", "updated_at", "size", "upload_time":
		return "0"
	case "name", "group_id", "attrs_type", "binding_sheet_id", "owner_id", "sheet_type", "upload_url":
		return "''"
	case "data", "extra":
		return "NULL"
	default:
		return "NULL"
	}
}

func logDBInit(dboperator operator.DatabaseOperator, logf func(string)) error {
	writeDB := dboperator.GetLogDB(constant.WRITE)
	switch dboperator.Type() {
	case "sqlite":
		writeDB.Exec("PRAGMA mmap_size = 16777216") // 16 MB 窗口足够
		writeDB.Exec("PRAGMA cache_size = -32768")  // 32 MB page cache
		if err := ensureSQLiteLogSchema(writeDB); err != nil {
			return err
		}
		logf("LOGS 日志数据库初始化完成")
		return nil
	case "mysql":
		if err := writeDB.AutoMigrate(&model.LogInfoHookMySQL{}); err != nil {
			return err
		}
		err := writeDB.AutoMigrate(&model.LogOneItemHookMySQL{})
		if err != nil {
			return err
		}
		err = createMySQLIndexForLogInfo(writeDB)
		if err != nil {
			return err
		}
		err = createMySQLIndexForLogOneItem(writeDB)
		if err != nil {
			return err
		}
		err = calculateLogSize(writeDB)
		if err != nil {
			return err
		}
		logf("LOGS 日志数据库初始化完成")
		return nil
	default:
		if err := writeDB.AutoMigrate(&model.LogInfo{}); err != nil {
			return err
		}
		err := writeDB.AutoMigrate(&model.LogOneItem{})
		if err != nil {
			return err
		}
		logf("LOGS 日志数据库初始化完成")
		return nil
	}
}

func calculateLogSize(logsDB *gorm.DB) error {
	log := zap.S().Named(logger.LogKeyDatabase)
	// TODO: 将这段逻辑挪移到Migrator上 现在暂时没空动它
	var ids []uint64
	var logItemSums []struct {
		LogID uint64
		Count int64
	}
	logsDB.Model(&model.LogInfo{}).Pluck("id", &ids) // 获取所有 LogInfo 的 IDs，由于是一次性的，所以不需要再判断了
	if len(ids) > 0 {
		// 根据 LogInfo 表中的 IDs 查找对应的 LogOneItem 记录
		err := logsDB.Model(&model.LogOneItem{}).
			Where("log_id IN ?", ids).
			Group("log_id").
			Select("log_id, COUNT(*) AS count"). // 如果需要求和其他字段，可以使用 Sum
			Scan(&logItemSums).Error
		if err != nil {
			// 错误处理
			log.Infof("Error querying LogOneItem: %v", err)
			return err
		}

		// 2. 更新 LogInfo 表的 Size 字段
		for _, sum := range logItemSums {
			// 将求和结果更新到对应的 LogInfo 的 Size 字段
			err = logsDB.Model(&model.LogInfo{}).
				Where("id = ?", sum.LogID).
				UpdateColumn("size", sum.Count).Error // 或者是 sum.Time 等，如果要是其他字段的求和
			if err != nil {
				// 错误处理
				log.Errorf("Error updating LogInfo: %v", err)
				return err
			}
		}
	}
	return nil
}

func censorDBInit(dboperator operator.DatabaseOperator, logf func(string)) error {
	// 获取LogDB的
	writeDB := dboperator.GetCensorDB(constant.WRITE)
	// 创建基本的表结构，并通过标签定义索引
	if err := writeDB.AutoMigrate(&model.CensorLog{}); err != nil {
		return err
	}
	logf("censorDB记录日志表初始化完成")
	return nil
}
