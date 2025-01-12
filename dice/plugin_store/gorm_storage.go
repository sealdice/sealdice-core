package plugin_store

import (
	"errors"

	"gorm.io/gorm"

	sealkv "sealdice-core/utils/gokv"
	"sealdice-core/utils/gokv/gormkv"
)

// 传入大量pluginName时，由于使用的是指针
type GormPluginStorage struct {
	db *gorm.DB
}

func (g *GormPluginStorage) GetStorageInstance(pluginName string) (sealkv.SealDiceKVStore, error) {
	options := gormkv.Options{
		DB:         g.db,
		PluginName: pluginName,
	}
	return gormkv.NewStore(options)
}

func (g *GormPluginStorage) StorageClose() error {
	db, err := g.db.DB()
	if err != nil {
		return err
	}
	return db.Close()
}

// 初始化
func NewGormPluginStorage(db *gorm.DB) (*GormPluginStorage, error) {
	if db == nil {
		return nil, errors.New("db is nil")
	}
	return &GormPluginStorage{
		db: db,
	}, nil
}
