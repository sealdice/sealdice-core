package v150

import (
	"fmt"
	"os"
	"strings"
	"time"

	ds "github.com/sealdice/dicescript"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	"gorm.io/gorm"

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

		rawData, err := ds.NewDictVal(m).V().ToJSON()
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
	// AutoMigrate初始化 TODO: 用CreateTable是不是更合适呢？
	// data建表
	err := writeDB.AutoMigrate(
		&model.GroupPlayerInfoBase{},
		&model.GroupInfo{},
		&model.BanInfo{},
		&model.EndpointInfo{},
		&model.AttributesItemModel{},
	)
	if err != nil {
		return err
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

func logDBInit(dboperator operator.DatabaseOperator, logf func(string)) error {
	// 获取LogDB的
	writeDB := dboperator.GetLogDB(constant.WRITE)
	if dboperator.Type() != "mysql" {
		// logs建表
		if err := writeDB.AutoMigrate(&model.LogInfo{}); err != nil {
			return err
		}
		err := writeDB.AutoMigrate(&model.LogOneItem{})
		if err != nil {
			return err
		}
		logf("LOGS 记录日志表初始化完成")
		return nil
	} else {
		// MySQL logs 特化建表
		if err := writeDB.AutoMigrate(&model.LogInfoHookMySQL{}); err != nil {
			return err
		}
		err := writeDB.AutoMigrate(&model.LogOneItemHookMySQL{})
		if err != nil {
			return err
		}
		// MYSQL 特化 logs建立索引
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
		logf("LOGS 记录日志表初始化完成")
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
	writeDB := dboperator.GetLogDB(constant.WRITE)
	// 创建基本的表结构，并通过标签定义索引
	if err := writeDB.AutoMigrate(&model.CensorLog{}); err != nil {
		return err
	}
	logf("censorDB记录日志表初始化完成")
	return nil
}
