package plugin_store

import (
	"github.com/philippgille/gokv"
	"gorm.io/gorm"

	utils "sealdice-core/utils/gokv"
)

// 传入大量pluginName时，由于使用的是指针
type GormPluginStorage struct {
	db *gorm.DB
}

// 初始化
func NewGormPluginStorage(db *gorm.DB) *GormPluginStorage {
	return &GormPluginStorage{
		db: db,
	}
}

// 获取对应的Store存储,有了这个存储，就可以直接使用它进行获取了，最后调用Close的时候不同
func (g GormPluginStorage) StorageInit(pluginName string) (gokv.Store, error) {
	options := utils.Options{
		DB:         g.db,
		PluginName: pluginName,
	}
	store, err := utils.NewStore(options)
	if err != nil {
		return nil, err
	}
	return store, nil
}

func (g GormPluginStorage) StorageClose() error {

}

func (g GormPluginStorage) StorageSet(k, v string) error {
	//TODO implement me
	panic("implement me")
}

func (g GormPluginStorage) StorageGet(k string) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (g GormPluginStorage) StorageClear(pluginName string) error {
	//TODO implement me
	panic("implement me")
}

func (g GormPluginStorage) StorageStop() error {
	//TODO implement me
	panic("implement me")
}
