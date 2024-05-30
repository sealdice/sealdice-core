package model

import (
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"

	nanoid "github.com/matoous/go-nanoid/v2"
	ds "github.com/sealdice/dicescript"
)

// AttributesItemModel 新版人物卡
type AttributesItemModel struct {
	Id   string `json:"id" db:"id"`     // 如果是群内，那么是类似 QQ-Group:12345-QQ:678910，群外是nanoid
	Data []byte `json:"data" db:"data"` // 序列化后的卡数据，理论上[]byte不会进入字符串缓存，要更好些？

	// 这些是群组内置卡专用的，其实就是替代了绑卡关系表，作为群组内置卡时，这个字段用于存放绑卡关系
	BindingSheetId string `json:"bindingSheetId" db:"binding_sheet_id"` // 绑定的卡片ID

	// 这些是角色卡专用的
	Nickname  string `json:"nickname" db:"nickname"`    // 卡片名称
	OwnerId   string `json:"ownerId" db:"owner_id"`     // 若有明确归属，就是对应的UniformID
	SheetType string `json:"sheetType" db:"sheet_type"` // 卡片类型，如dnd5e coc7
	IsHidden  bool   `json:"isHidden" db:"is_hidden"`   // 隐藏的卡片不出现在 pc list 中

	// 通用属性
	CreatedAt int64 `json:"createdAt" db:"created_at"`
	UpdatedAt int64 `json:"updatedAt" db:"updated_at"`

	// 下面的属性并非数据库字段，而是用于内存中的缓存
	BindingGroupsNum int64 `json:"bindingGroupNum"` // 当前绑定中群数
}

func (m *AttributesItemModel) IsDataExists() bool {
	return m.Data != nil && len(m.Data) > 0
}

// TOOD: 下面这个表记得添加 unique 索引

// PlatformMappingModel 虚拟ID - 平台用户ID 映射表
type PlatformMappingModel struct {
	Id       string `json:"id" db:"id"`               // 虚拟ID，格式为 U:nanoid 意为 User / Uniform / Universal
	IMUserID string `json:"IMUserID" db:"im_user_id"` // IM平台的用户ID
}

func AttrsGetById(db *sqlx.DB, id string) (*AttributesItemModel, error) {
	var item AttributesItemModel
	err := db.Get(&item, "select * from attrs where id = $1", id)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return &item, nil
}

// AttrsGetBindingSheetIdByGroupId 获取当前正在绑定的ID
func AttrsGetBindingSheetIdByGroupId(db *sqlx.DB, id string) (string, error) {
	var item AttributesItemModel
	err := db.Get(&item, "select binding_sheet_id from attrs where id = $1", id)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}
	return item.BindingSheetId, nil
}

func AttrsGetIdByUidAndName(db *sqlx.DB, userId string, name string) (string, error) {
	var item AttributesItemModel
	err := db.Get(&item, "select id from attrs where owner_id = $1 and nickname = $2", userId, name)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}
	return item.Id, nil
}

func AttrsPutById(db *sqlx.DB, tx *sql.Tx, id string, data []byte, nickname string) error {
	// TODO: 好像还不够，需要nickname 需要sheetType，还有别的吗
	var err error
	now := time.Now().Unix()
	query := `insert into attrs (id, data, is_hidden, binding_sheet_id, created_at, updated_at, nickname)
			  values ($1, $2, true, '', $3, $3, $4)
			  on conflict (id) do update set data = $2, updated_at = $3, nickname = $4`
	args := []any{id, data, now, nickname}

	if tx != nil {
		_, err = tx.Exec(query, args...)
	} else {
		_, err = db.Exec(query, args...)
	}
	return err
}

func AttrsCharGetBindingList(db *sqlx.DB, id string) ([]string, error) {
	rows, err := db.Query(`select id from attrs where binding_sheet_id = $1`, id)
	if err != nil {
		return nil, err
	}

	lst := []string{}
	for rows.Next() {
		item := ""
		err := rows.Scan(&item)
		if err != nil {
			return nil, err
		}
		lst = append(lst, item)
	}

	return lst, err
}

func AttrsCharUnbindAll(db *sqlx.DB, id string) (int64, error) {
	rows, err := db.Exec(`update attrs set binding_sheet_id = '' where binding_sheet_id = $1`, id)
	if err != nil {
		return 0, err
	}
	affected, err := rows.RowsAffected()
	if err != nil {
		return 0, err
	}
	return affected, err
}

// AttrsNewItem 新建一个角色卡/属性容器
func AttrsNewItem(db *sqlx.DB, item *AttributesItemModel) (*AttributesItemModel, error) {
	id, _ := nanoid.New()
	now := time.Now().Unix()
	item.Id, item.CreatedAt, item.UpdatedAt = id, now, now

	var err error
	_, err = db.Exec(`
		insert into attrs (id, data, nickname, owner_id, sheet_type, is_hidden, created_at, updated_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8)`,
		item.Id, item.Data, item.Nickname, item.OwnerId, item.SheetType, item.IsHidden, item.CreatedAt, item.UpdatedAt)
	return item, err
}

func AttrsBindCharacter(db *sqlx.DB, charId string, id string) error {
	json, err := ds.VMValueNewDict(nil).V().ToJSON()
	if err != nil {
		return err
	}
	_, _ = db.Exec(`insert into attrs (id, data, is_hidden, binding_sheet_id, created_at, updated_at)
					       values ($1, $3, true, '', $2, $2)`, id, time.Now().Unix(), json)

	ret, err := db.Exec(`update attrs set binding_sheet_id = $1 where id = $2`, charId, id)
	if err == nil {
		affected, err := ret.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return errors.New("群信息不存在: " + id)
		}
	}
	return err
}

func AttrsGetCharacterListByUserId(db *sqlx.DB, userId string) (lst []*AttributesItemModel, err error) {
	rows, err := db.Queryx(`
	select id, nickname, sheet_type,
	       (select count(id) from attrs where binding_sheet_id = id)
	from attrs where owner_id = $1 and is_hidden is false
	`, userId)
	if err != nil {
		return nil, err
	}
	var items []*AttributesItemModel
	for rows.Next() {
		item := &AttributesItemModel{}
		err := rows.Scan(
			&item.Id,
			&item.Nickname,
			&item.SheetType,
			&item.BindingGroupsNum,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}
