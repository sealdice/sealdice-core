package migrate

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	ds "github.com/sealdice/dicescript"

	"sealdice-core/dice"
	"sealdice-core/dice/model"
	"sealdice-core/utils"
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
			AttrsType: model.AttrsTypeCharacter,

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
var sheetIdBindByGroupUserId map[string]string

// AttrsNewItem 新建一个角色卡/属性容器
func AttrsNewItem(db *sqlx.Tx, item *model.AttributesItemModel) (*model.AttributesItemModel, error) {
	id := utils.NewID()
	now := time.Now().Unix()
	item.CreatedAt, item.UpdatedAt = now, now
	if item.Id == "" {
		item.Id = id
	}

	var err error
	_, err = db.Exec(`
		insert into attrs (id, data, binding_sheet_id, name, owner_id, sheet_type, is_hidden, created_at, updated_at, attrs_type)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		item.Id, item.Data, item.BindingSheetId, item.Name, item.OwnerId, item.SheetType, item.IsHidden,
		item.CreatedAt, item.UpdatedAt, item.AttrsType)
	return item, err
}

// 群组个人数据转换
func attrsGroupUserMigrate(db *sqlx.Tx) (int, int, error) {
	rows, err := db.NamedQuery("select id, updated_at, data from attrs_group_user", map[string]any{})
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	count := 0
	countFailed := 0
	for rows.Next() {
		var id string
		var updatedAt int64
		var data []byte

		err := rows.Scan(
			&id,
			&updatedAt,
			&data,
		)

		if err != nil {
			return count, countFailed, err
		}

		// id 为 GUID 即 GroupID-UserID
		_, userIdPart, ok := dice.UnpackGroupUserId(id)
		if !ok {
			countFailed += 1
			fmt.Println("数据库读取出错，退出转换")
			fmt.Println("ID解析失败: ", id)
			continue
		}

		mapData := make(map[string]*dice.VMValue)
		err = dice.JSONValueMapUnmarshal(data, &mapData)

		if err != nil {
			countFailed += 1
			fmt.Println("解析失败: ", string(data))
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
			fmt.Printf("群-用户 %s 的数据无法转换\n", id)
			continue
		}

		// fmt.Println("UnpackID:", id, " UserPart:", userIdPart, " Sheet:", sheetIdBindByGroupUserId[id])
		item := &model.AttributesItemModel{
			Id:        id,
			Data:      rawData,
			AttrsType: model.AttrsTypeGroupUser,

			// 当前组内绑定的卡
			BindingSheetId: sheetIdBindByGroupUserId[id],

			// 这些是角色卡专用的
			Name:      "", // 群内默认卡，无名字，还是说以后弄成和nn的名字一致？
			OwnerId:   userIdPart,
			SheetType: cardType,
			IsHidden:  true,

			CreatedAt: updatedAt,
			UpdatedAt: updatedAt,
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
func attrsGroupMigrate(db *sqlx.Tx) (int, int, error) {
	rows, err := db.NamedQuery("select id, updated_at, data from attrs_group", map[string]any{})
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	count := 0
	countFailed := 0
	for rows.Next() {
		var id string
		var updatedAt int64
		var data []byte

		err := rows.Scan(
			&id,
			&updatedAt,
			&data,
		)

		if err != nil {
			fmt.Println("数据库读取出错，退出转换")
			return count, countFailed, err
		}

		mapData := make(map[string]*dice.VMValue)
		err = dice.JSONValueMapUnmarshal(data, &mapData)

		if err != nil {
			countFailed += 1
			fmt.Println("解析失败: ", string(data))
			continue
		}

		// 基础属性
		m := &ds.ValueMap{}
		for k, v := range mapData {
			m.Store(k, v.ConvertToV2())
		}

		rawData, err := ds.NewDictVal(m).V().ToJSON()
		if err != nil {
			countFailed += 1
			fmt.Printf("群 %s 的数据无法转换\n", id)
			continue
		}

		item := &model.AttributesItemModel{
			Id:        id,
			Data:      rawData,
			AttrsType: model.AttrsTypeGroup,

			IsHidden: true,

			CreatedAt: updatedAt,
			UpdatedAt: updatedAt,
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
func attrsUserMigrate(db *sqlx.Tx) (int, int, int, error) {
	rows, err := db.NamedQuery("select id, updated_at, data from attrs_user where length(data) < 9000000", map[string]any{})
	if err != nil {
		return 0, 0, 0, err
	}
	defer rows.Close()

	count := 0
	countSheetsNum := 0
	countFailed := 0
	for rows.Next() {
		var ownerId string
		var updatedAt int64
		var data []byte

		err := rows.Scan(
			&ownerId,
			&updatedAt,
			&data,
		)

		if err != nil {
			fmt.Println("数据库读取出错，退出转换")
			return count, countSheetsNum, countFailed, err
		}

		mapData := make(map[string]*dice.VMValue)
		err = dice.JSONValueMapUnmarshal(data, &mapData)

		if err != nil {
			countFailed += 1
			continue
		}

		// fmt.Println("数据转换-用户:", ownerId)
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
				// fmt.Println("绑卡关联:", groupId, name)
				continue
			}
			if strings.HasPrefix(k, "$ch:") {
				// 处理角色卡，这里 v 是 string
				var toNew *model.AttributesItemModel
				name := k[4:]

				toNew, err = convertToNew(name, ownerId, []byte(v.ToString()), updatedAt)
				if err != nil {
					fmt.Printf("用户 %s 的角色卡 %s 无法转换", ownerId, name)
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
					// fmt.Println("GUID:", fmt.Sprintf("%s-%s", groupID, ownerId), " sheetID:", i.Id)
					sheetIdBindByGroupUserId[fmt.Sprintf("%s-%s", groupID, ownerId)] = i.Id
				}
			}
		}

		// 保存用户人物卡
		for _, i := range newSheetsList {
			_, err = AttrsNewItem(db, i)
			if err != nil {
				fmt.Printf("用户 %s 的角色卡 %s 无法写入数据库: %s\n", ownerId, i.Name, err.Error())
			}
		}

		countSheetsNum += len(newSheetsList)
		rawData, err := ds.NewDictVal(m).V().ToJSON()
		if err != nil {
			countFailed += 1
			fmt.Printf("用户 %s 的个人数据无法转换\n", ownerId)
			continue
		}

		item := &model.AttributesItemModel{
			Id:        ownerId,
			Data:      rawData,
			AttrsType: model.AttrsTypeUser,

			IsHidden:  true,
			CreatedAt: updatedAt,
			UpdatedAt: updatedAt,
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

func checkTableExists(db *sqlx.DB, tableName string) (bool, error) {
	var name string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name = $1;", tableName).Scan(&name)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// 表格不存在，继续执行
		return false, nil
	case err != nil:
		return false, err
	default:
		// 表格已经存在，说明转换完成，退出
		return true, nil
	}
}

func V150Upgrade() bool {
	dbDataPath, _ := filepath.Abs("./data/default/data.db")
	if _, err := os.Stat(dbDataPath); errors.Is(err, os.ErrNotExist) {
		return true
	}

	db, err := openDB(dbDataPath)
	if err != nil {
		fmt.Println("升级失败，无法打开数据库:", err)
		return false
	}
	defer func() {
		_ = db.Close()
	}()

	exists, err := checkTableExists(db, "attrs")
	if err != nil {
		fmt.Println("V150数据转换未知错误:", err.Error())
		return false
	}
	if exists {
		// 表格已经存在，说明转换完成，退出
		return true
	}

	fmt.Println("1.5 数据迁移")
	sheetIdBindByGroupUserId = map[string]string{}

	sqls := []string{
		`
CREATE TABLE IF NOT EXISTS attrs (
    id TEXT PRIMARY KEY,
    data BYTEA,
    attrs_type TEXT,

	-- 坏，Get这个方法太严格了，所有的字段都要有默认值，不然无法反序列化
	binding_sheet_id TEXT default '',

    name TEXT default '',
    owner_id TEXT default '',
    sheet_type TEXT default '',
    is_hidden BOOLEAN default FALSE,

    created_at INTEGER default 0,
    updated_at INTEGER  default 0
);
`,
		`create index if not exists idx_attrs_binding_sheet_id on attrs (binding_sheet_id);`,
		`create index if not exists idx_attrs_owner_id_id on attrs (owner_id);`,
		`create index if not exists idx_attrs_attrs_type_id on attrs (attrs_type);`,
	}
	for _, i := range sqls {
		_, _ = db.Exec(i)
	}

	tx, err := db.Beginx()
	if err != nil {
		fmt.Println("V150数据转换创建事务失败:", err.Error())
		return false
	}

	if exists, _ := checkTableExists(db, "attrs_user"); exists {
		count, countSheetsNum, countFailed, err2 := attrsUserMigrate(tx)
		fmt.Printf("数据卡转换 - 角色卡，成功人数%d 失败人数 %d 卡数 %d\n", count, countFailed, countSheetsNum)
		if err2 != nil {
			fmt.Println("异常", err2.Error())
			return false
		}
	}

	if exists, _ := checkTableExists(db, "attrs_group_user"); exists {
		count, countFailed, err2 := attrsGroupUserMigrate(tx)
		fmt.Printf("数据卡转换 - 群组个人数据，成功%d 失败 %d\n", count, countFailed)
		if err2 != nil {
			fmt.Println("异常", err2.Error())
			return false
		}
	}

	if exists, _ := checkTableExists(db, "attrs_group"); exists {
		count, countFailed, err2 := attrsGroupMigrate(tx)
		fmt.Printf("数据卡转换 - 群数据，成功%d 失败 %d\n", count, countFailed)
		if err2 != nil {
			fmt.Println("异常", err2.Error())
			return false
		}
	}

	// 删档
	fmt.Println("删除旧版本数据")
	_, _ = tx.Exec("drop table attrs_group")
	_, _ = tx.Exec("drop table attrs_group_user")
	_, _ = tx.Exec("drop table attrs_user")
	_, _ = db.Exec("PRAGMA wal_checkpoint(TRUNCATE);")
	_, _ = tx.Exec("VACUUM;") // 收尾

	sheetIdBindByGroupUserId = nil

	err = tx.Commit()
	if err != nil {
		fmt.Println("V150 数据转换失败:", err.Error())
		return false
	}

	fmt.Println("V150 数据转换完成")
	return true
}
