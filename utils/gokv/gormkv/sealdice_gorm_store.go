package gormkv

import (
	"errors"
	"strconv"

	"github.com/cespare/xxhash/v2"
	"github.com/philippgille/gokv/encoding"
	gutil "github.com/philippgille/gokv/util"
	"gorm.io/gorm"

	"sealdice-core/utils"
)

// 需求梳理：
// 0. 选用gokv库并自定义一个gorm实现，则可以方便的做一层抽象。同时，我其实还考虑用它日后尝试实现SyncMap的底层
// 1. 由于最终都是存储在一个gorm的数据库里，默认情况下，传入一个DB实例是很合理的
// 2. 考虑到支持多数据库的情况，同时还要兼容当前的逻辑（一个数据一个文件），得传一个“列名“，也就是上面的src
// 3. 一个DB，多个列名，也就是说需要共享同一个DB实例，DB实例应该直接在一开始就初始化后传入，然后返回对应的Store对象
// 4. 可以给实现增加一些其他的功能，比如Len等，gorm kv可能不需要这些功能，但我们可以做。
// 5. 应当允许传入表名自定义，以满足一些日后可能的情况
// 6. 考虑最后使用JSON来进行数据存储，从而能保证即使上面的代码崩溃，也可以从数据库里取出数据

var modelTableName = "plugins"

type Options struct {
	// 表名称
	TableName string
	// 存储的DB
	DB *gorm.DB
	// 插件名（应该）
	PluginName string
}

type BaseModel struct {
	ID        uint `gorm:"column:id;primarykey"`
	CreatedAt utils.Time
	UpdatedAt utils.Time
	// 保持和原本的BuntDB一样的方案，留存删除痕迹
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// KVRecord GORM模型
type KVRecord struct {
	BaseModel
	PluginName string `gorm:"column:plugin_name;index:idx_plugin_name_key"` // 来源文件夹，事实上就是插件名 - TODO: 有没有可能插件名会被替换，导致数据无法迁移？
	Key        string `gorm:"column:key;index:idx_plugin_name_key"`         // 哈希Key, 将用户的Key使用xxHash进行64位哈希，此举为了是兼容MySQL，防止长度过长导致索引失败
	RawKey     string `gorm:"column:raw_key;"`                              // 用户原本的Key。
	// 思路：Key 使用哈希计算，然后 RawKey 进行原本的key存储。对Key进行索引，就能避免过长的问题
	Value []byte `gorm:"column:value;"` // value，JSON序列化后存储，不使用GOB是为了可以直接读取数据库内容，防止GOKV库不靠谱
}

func (*KVRecord) TableName() string {
	return modelTableName
}

// Store GORM
type Store struct {
	// 被传入的DB
	DB *gorm.DB
	// 对应的插件名称
	PluginName string
	// 默认的序列化
	Codec encoding.JSONcodec
}

func (s Store) Set(k string, v any) error {
	if err := gutil.CheckKeyAndValue(k, v); err != nil {
		return err
	}
	marshal, err := s.Codec.Marshal(v)
	if err != nil {
		return err
	}
	hashKey := strconv.FormatUint(xxhash.Sum64String(k), 10)
	// 计算k的哈希值
	record := KVRecord{
		PluginName: s.PluginName,
		Key:        hashKey,
		RawKey:     k,
		Value:      marshal,
	}
	// 无论找没找到数据，都要进行更新
	return s.DB.
		Where("plugin_name = ? and key = ?", s.PluginName, hashKey).
		Assign(record).
		FirstOrCreate(&KVRecord{}).Error
}

// Get 此处的v用来接收，应该是个指针
func (s Store) Get(k string, v any) (bool, error) {
	if err := gutil.CheckKeyAndValue(k, v); err != nil {
		return false, err
	}
	hashKey := strconv.FormatUint(xxhash.Sum64String(k), 10)
	// 根据pluginName + k(哈希) 取对应数据
	var record KVRecord
	result := s.DB.
		Where("plugin_name = ? and key = ?", s.PluginName, hashKey).
		Limit(1).
		Find(&record)
	// 如果查询错误
	if result.Error != nil {
		return false, result.Error
	}
	// 如果查询不到 报错
	if result.RowsAffected == 0 {
		return false, errors.New("record not found")
	}
	// 否则返回
	return true, s.Codec.Unmarshal(record.Value, v)
}

func (s Store) Delete(k string) error {
	// 计算哈希key
	hashKey := strconv.FormatUint(xxhash.Sum64String(k), 10)
	// 针对这个pluginName + hashKey进行处理
	return s.DB.
		Where("plugin_name = ? and key = ?", s.PluginName, hashKey).
		Delete(&KVRecord{}).Error
}

// Close 关闭对应Store的数据库连接 在上层封装的时候进行处理
func (s Store) Close() error {
	db, err := s.DB.DB()
	if err != nil {
		return err
	}
	return db.Close()
}

// Clear 清理插件存储
func (s Store) Clear() error {
	// 清理掉所有某个插件的数据
	return s.DB.Where("plugin_name = ?", s.PluginName).Delete(&KVRecord{}).Error
}

// NewStore result.DB.AutoMigrate(&KVRecord{}) 自动建表移动到初始化数据库处 此处仅获取GOKV
func NewStore(options Options) (Store, error) {
	if options.DB == nil {
		return Store{}, errors.New("db is nil, you must write it before")
	}
	if options.TableName != "" {
		modelTableName = options.TableName
	}
	if options.PluginName == "" {
		return Store{}, errors.New("pluginName is nil, you must write it before")
	}
	result := Store{
		DB:         options.DB,
		PluginName: options.PluginName,
		Codec:      encoding.JSON,
	}
	return result, nil
}
