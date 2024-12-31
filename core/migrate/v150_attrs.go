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
			fmt.Fprintln(os.Stdout, "数据库读取出错，退出转换")
			fmt.Fprintln(os.Stdout, "ID解析失败: ", id)
			continue
		}

		mapData := make(map[string]*dice.VMValue)
		err = dice.JSONValueMapUnmarshal(data, &mapData)

		if err != nil {
			countFailed += 1
			fmt.Fprintln(os.Stdout, "解析失败: ", string(data))
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
			fmt.Fprintf(os.Stdout, "群-用户 %s 的数据无法转换\n", id)
			continue
		}

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
			fmt.Fprintln(os.Stdout, "数据库读取出错，退出转换")
			return count, countFailed, err
		}

		mapData := make(map[string]*dice.VMValue)
		err = dice.JSONValueMapUnmarshal(data, &mapData)

		if err != nil {
			countFailed += 1
			fmt.Fprintln(os.Stdout, "解析失败: ", string(data))
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
			fmt.Fprintf(os.Stdout, "群 %s 的数据无法转换\n", id)
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
			fmt.Fprintln(os.Stdout, "数据库读取出错，退出转换")
			return count, countSheetsNum, countFailed, err
		}

		mapData := make(map[string]*dice.VMValue)
		err = dice.JSONValueMapUnmarshal(data, &mapData)

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

				toNew, err = convertToNew(name, ownerId, []byte(v.ToString()), updatedAt)
				if err != nil {
					fmt.Fprintf(os.Stdout, "用户 %s 的角色卡 %s 无法转换", ownerId, name)
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
					sheetIdBindByGroupUserId[fmt.Sprintf("%s-%s", groupID, ownerId)] = i.Id
				}
			}
		}

		// 保存用户人物卡
		for _, i := range newSheetsList {
			_, err = AttrsNewItem(db, i)
			if err != nil {
				fmt.Fprintf(os.Stdout, "用户 %s 的角色卡 %s 无法写入数据库: %s\n", ownerId, i.Name, err.Error())
			}
		}

		countSheetsNum += len(newSheetsList)
		rawData, err := ds.NewDictVal(m).V().ToJSON()
		if err != nil {
			countFailed += 1
			fmt.Fprintf(os.Stdout, "用户 %s 的个人数据无法转换\n", ownerId)
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

// Pinenutn: 2024-10-28 我要把这个注释全文背诵，它扰乱了GORM的初始化逻辑
// -- 坏，Get这个方法太严格了，所有的字段都要有默认值，不然无法反序列化
var v150sqls = []string{
	`
CREATE TABLE IF NOT EXISTS attrs (
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
`,
	`create index if not exists idx_attrs_binding_sheet_id on attrs (binding_sheet_id);`,
	`create index if not exists idx_attrs_owner_id_id on attrs (owner_id);`,
	`create index if not exists idx_attrs_attrs_type_id on attrs (attrs_type);`,
}

func V150Upgrade() error {
	dbDataPath, _ := filepath.Abs("./data/default/data.db")
	if _, err := os.Stat(dbDataPath); errors.Is(err, os.ErrNotExist) {
		log.Error("未找到旧版本数据库，若您启动全新海豹，可安全忽略。")
		return nil
	}

	db, err := openDB(dbDataPath)
	if err != nil {
		return fmt.Errorf("升级失败，无法打开数据库: %w", err)
	}
	defer db.Close()

	tx, err := db.Beginx()
	if err != nil {
		return fmt.Errorf("创建事务失败: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			err = tx.Rollback()
			if err != nil {
				log.Errorf("回滚事务时出错: %v", err)
			}
			panic(p) // 继续传播 panic
		} else if err != nil {
			log.Errorf("日志处理时出现异常行为: %v", err)
			err = tx.Rollback()
			if err != nil {
				log.Errorf("回滚事务时出错: %v", err)
				return
			}
		} else {
			err = tx.Commit()
			if err != nil {
				log.Errorf("提交事务时出错: %v", err)
			}
		}
	}()

	exists, err := checkTableExists(db, "attrs")
	if err != nil {
		return fmt.Errorf("检查表是否存在时出错: %w", err)
	}
	// 特判146->150的倒霉蛋
	exists146, err := checkTableExists(db, "attrs_group")

	if exists {
		if exists146 {
			log.Errorf("1.4.6的数据部分迁移！您可能是150部分版本的受害者，请联系开发者")
			return errors.New("150和146的数据库共同存在，请联系开发者")
		}
		// 表格已经存在，说明转换完成
		return nil
	}

	log.Info("1.5 数据迁移")
	sheetIdBindByGroupUserId = map[string]string{}

	for _, singleSql := range v150sqls {
		if _, err = tx.Exec(singleSql); err != nil {
			return fmt.Errorf("执行 SQL 出错: %w", err)
		}
	}

	if exists, _ = checkTableExists(db, "attrs_user"); exists {
		count, countSheetsNum, countFailed, err0 := attrsUserMigrate(tx)
		log.Infof("数据卡转换 - 角色卡，成功人数%d 失败人数 %d 卡数 %d\n", count, countFailed, countSheetsNum)
		if err0 != nil {
			return fmt.Errorf("角色卡转换出错: %w", err0)
		}
	}

	if exists, _ = checkTableExists(db, "attrs_group_user"); exists {
		count, countFailed, err1 := attrsGroupUserMigrate(tx)
		log.Infof("数据卡转换 - 群组个人数据，成功%d 失败 %d\n", count, countFailed)
		if err1 != nil {
			return fmt.Errorf("群组个人数据转换出错: %w", err1)
		}
	}

	if exists, _ = checkTableExists(db, "attrs_group"); exists {
		count, countFailed, err2 := attrsGroupMigrate(tx)
		log.Infof("数据卡转换 - 群数据，成功%d 失败 %d\n", count, countFailed)
		if err2 != nil {
			return fmt.Errorf("群数据转换出错: %w", err2)
		}
	}

	// 删除旧版本数据
	log.Info("删除旧版本数据")
	deleteSQLs := []string{
		"drop table attrs_group",
		"drop table attrs_group_user",
		"drop table attrs_user",
	}
	for _, deleteSQL := range deleteSQLs {
		if _, err = tx.Exec(deleteSQL); err != nil {
			return fmt.Errorf("删除旧数据时出错: %w", err)
		}
	}
	// 放在这里保证能执行
	_, _ = db.Exec("PRAGMA wal_checkpoint(TRUNCATE);")
	_, _ = db.Exec("VACUUM;")
	sheetIdBindByGroupUserId = nil
	log.Info("V150 数据转换完成")
	return nil
}
