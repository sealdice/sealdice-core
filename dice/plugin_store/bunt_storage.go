package plugin_store

import "github.com/philippgille/gokv"

// BuntDBPluginStorage 需要直接将buntdb的实现成gokv的实现
// 是否应该提供一个Migrator，将BuntDB的数据迁移到gokv中？这个事情涉及比较大，或许得进一步研究
// 当前考虑: 如果设置PLUGINDB = xxxx的时候，执行迁移，并删除原有数据（或者改个名如何？）
type BuntDBPluginStorage struct {
}

func NewBuntDBPluginStorage() *BuntDBPluginStorage {
	return &BuntDBPluginStorage{}
}

func (b BuntDBPluginStorage) StorageInit(pluginName string) (gokv.Store, error) {
	//TODO implement me
	panic("implement me")
}

func (b BuntDBPluginStorage) StorageClose() error {
	//TODO implement me
	panic("implement me")
}

func (b BuntDBPluginStorage) StorageSet(k, v string) error {
	//TODO implement me
	panic("implement me")
}

func (b BuntDBPluginStorage) StorageGet(k string) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (b BuntDBPluginStorage) StorageClear(pluginName string) error {
	//TODO implement me
	panic("implement me")
}

func (b BuntDBPluginStorage) StorageStop() error {
	//TODO implement me
	panic("implement me")
}
