package v150

import (
	"fmt"
	"os"
	"strings"
	"time"

	ds "github.com/sealdice/dicescript"
	"github.com/tidwall/gjson"
	"gorm.io/gorm"

	"sealdice-core/dice"
	"sealdice-core/dice/service"
	"sealdice-core/model"
	"sealdice-core/utils"
	"sealdice-core/utils/constant"
	operator "sealdice-core/utils/dboperator/engine"

	log "sealdice-core/utils/kratos"
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
		// 判断data是否为正常JSON字符串，以及ID是否存在，若不行的话也需要跳过
		if !gjson.Valid(string(row.Data)) || row.ID == "" {
			log.Warnf("[损坏数据] 跳过一行数据，用户核心数据已经损坏: %v", err)
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
			fmt.Fprintln(os.Stdout, "数据库读取出错，退出转换")
			fmt.Fprintln(os.Stdout, "ID解析失败: ", row.ID)
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
		// 判断data是否为正常JSON字符串，以及ID是否存在，若不行的话也需要跳过
		if !gjson.Valid(string(row.Data)) || row.ID == "" {
			log.Warnf("[损坏数据] 跳过一行数据，用户核心数据已经损坏: %v", err)
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
		// 判断data是否为正常JSON字符串，以及ID是否存在，若不行的话也需要跳过
		if !gjson.Valid(string(row.Data)) || row.ID == "" {
			log.Warnf("[损坏数据] 跳过一行数据，用户核心数据已经损坏: %v", err)
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
	dataDB := dboperator.GetDataDB(constant.WRITE)
	err := dataDB.Transaction(func(tx *gorm.DB) error {
		if tx.Migrator().HasTable("attrs_user") {
			count, countSheetsNum, countFailed, err0 := attrsUserMigrate(tx)
			log.Infof("数据卡转换 - 角色卡，成功人数%d 失败人数 %d 卡数 %d\n", count, countFailed, countSheetsNum)
			if err0 != nil {
				return fmt.Errorf("角色卡转换出错: %w", err0)
			}
			logf(fmt.Sprintf("数据卡转换 - 角色卡，成功人数%d 失败人数 %d 卡数 %d\n", count, countFailed, countSheetsNum))
		} else {
			logf("attrs_group_user表不存在，可能已经升级过！")
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

func V150LogsMigrate(dboperator operator.DatabaseOperator, logf func(string)) {
}
