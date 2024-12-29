package model

import (
	"errors"
	"fmt"
	"os"

	"gorm.io/gorm"

	"sealdice-core/dice/model/database"
	log "sealdice-core/utils/kratos"
)

type MYSQLEngine struct {
	DSN string
	DB  *gorm.DB
}

type LogInfoHookMySQL struct {
	ID         uint64  `json:"id" gorm:"primaryKey;autoIncrement;column:id"`
	Name       string  `json:"name" gorm:"column:name"`
	GroupID    string  `json:"groupId" gorm:"column:group_id"`
	CreatedAt  int64   `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt  int64   `json:"updatedAt" gorm:"column:updated_at"`
	Size       *int    `json:"size" gorm:"<-:false"`
	Extra      *string `json:"-" gorm:"column:extra"`
	UploadURL  string  `json:"-" gorm:"column:upload_url"`
	UploadTime int     `json:"-" gorm:"column:upload_time"`
}

func (*LogInfoHookMySQL) TableName() string {
	return "logs"
}

type LogOneItemHookMySQL struct {
	ID             uint64      `json:"id" gorm:"primaryKey;autoIncrement;column:id"`
	LogID          uint64      `json:"-" gorm:"column:log_id"`
	GroupID        string      `gorm:"column:group_id"`
	Nickname       string      `json:"nickname" gorm:"column:nickname"`
	IMUserID       string      `json:"IMUserId" gorm:"column:im_userid"`
	Time           int64       `json:"time" gorm:"column:time"`
	Message        string      `json:"message"  gorm:"column:message"`
	IsDice         bool        `json:"isDice"  gorm:"column:is_dice"`
	CommandID      int64       `json:"commandId"  gorm:"column:command_id"`
	CommandInfo    interface{} `json:"commandInfo" gorm:"-"`
	CommandInfoStr string      `json:"-" gorm:"column:command_info"`
	RawMsgID       interface{} `json:"rawMsgId" gorm:"-"`
	RawMsgIDStr    string      `json:"-" gorm:"column:raw_msg_id"`
	UniformID      string      `json:"uniformId" gorm:"column:user_uniform_id"`
	Channel        string      `json:"channel" gorm:"-"`
	Removed        *int        `gorm:"column:removed" json:"-"`
	ParentID       *int        `gorm:"column:parent_id" json:"-"`
}

func (*LogOneItemHookMySQL) TableName() string {
	return "log_items"
}

// 利用前缀索引，规避索引BUG
// 创建不出来也没关系，反正MYSQL数据库
func createIndexForLogInfo(db *gorm.DB) (err error) {
	// 创建前缀索引
	// 检查并创建索引
	if !db.Migrator().HasIndex(&LogInfoHookMySQL{}, "idx_log_name") {
		err = db.Exec("CREATE INDEX idx_log_name ON logs (name(20));").Error
		if err != nil {
			log.Errorf("创建idx_log_name索引失败,原因为 %v", err)
		}
	}

	if !db.Migrator().HasIndex(&LogInfoHookMySQL{}, "idx_logs_group") {
		err = db.Exec("CREATE INDEX idx_logs_group ON logs (group_id(20));").Error
		if err != nil {
			log.Errorf("创建idx_logs_group索引失败,原因为 %v", err)
		}
	}

	if !db.Migrator().HasIndex(&LogInfoHookMySQL{}, "idx_logs_updated_at") {
		err = db.Exec("CREATE INDEX idx_logs_updated_at ON logs (updated_at);").Error
		if err != nil {
			log.Errorf("创建idx_logs_updated_at索引失败,原因为 %v", err)
		}
	}
	return nil
}

func createIndexForLogOneItem(db *gorm.DB) (err error) {
	// 创建前缀索引
	// 检查并创建索引
	if !db.Migrator().HasIndex(&LogOneItemHookMySQL{}, "idx_log_items_group_id") {
		err = db.Exec("CREATE INDEX idx_log_items_group_id ON log_items(group_id(20))").Error
		if err != nil {
			log.Errorf("创建idx_logs_group索引失败,原因为 %v", err)
		}
	}
	if !db.Migrator().HasIndex(&LogOneItemHookMySQL{}, "idx_raw_msg_id") {
		err = db.Exec("CREATE INDEX idx_raw_msg_id ON log_items(raw_msg_id(20))").Error
		if err != nil {
			log.Errorf("创建idx_log_group_id_name索引失败,原因为 %v", err)
		}
	}
	// MYSQL似乎不能创建前缀联合索引，放弃所有的前缀联合索引
	return nil
}

func (s *MYSQLEngine) Init() error {
	s.DSN = os.Getenv("DB_DSN")
	if s.DSN == "" {
		return errors.New("DB_DSN is missing")
	}
	var err error
	s.DB, err = database.MySQLDBInit(s.DSN)
	if err != nil {
		return err
	}
	return nil
}

// DBCheck DB检查
func (s *MYSQLEngine) DBCheck() {
	fmt.Fprintln(os.Stdout, "MYSQL 海豹不提供检查，请自行检查数据库！")
}

// DataDBInit 初始化
func (s *MYSQLEngine) DataDBInit() (*gorm.DB, error) {
	err := s.DB.AutoMigrate(
		// TODO: 这个的索引有没有必要进行修改
		&GroupPlayerInfoBase{},
		&GroupInfo{},
		&BanInfo{},
		&EndpointInfo{},
		&AttributesItemModel{},
	)
	if err != nil {
		return nil, err
	}
	return s.DB, nil
}

func (s *MYSQLEngine) LogDBInit() (*gorm.DB, error) {
	// logs特殊建表
	if err := s.DB.AutoMigrate(&LogInfoHookMySQL{}, &LogOneItemHookMySQL{}); err != nil {
		return nil, err
	}
	// logs建立索引
	err := createIndexForLogInfo(s.DB)
	if err != nil {
		return nil, err
	}
	err = createIndexForLogOneItem(s.DB)
	if err != nil {
		return nil, err
	}
	return s.DB, nil
}

func (s *MYSQLEngine) CensorDBInit() (*gorm.DB, error) {
	// 创建基本的表结构，并通过标签定义索引
	if err := s.DB.AutoMigrate(&CensorLog{}); err != nil {
		return nil, err
	}
	return s.DB, nil
}
