package plugin_store

import (
	sealkv "sealdice-core/utils/gokv"
)

// 又得定义一个PluginStorage的接口封装了

// RemoveStorage 删除对应插件的KVStore持有，并且干掉这个KVStore 有无必要？

type PluginStorage interface {
	// GetStorageInstance 初始化，获取对应它的KVStore，后面的操作都应该由这个KVStore来完成
	GetStorageInstance(pluginName string) (sealkv.SealDiceKVStore, error)
	// StorageClose 关闭整个插件存储，用于最后优雅退出
	StorageClose() error
}

// 实现检查 copied from platform
var (
	_ PluginStorage = (*BuntDBPluginStorage)(nil)
	_ PluginStorage = (*GormPluginStorage)(nil)
)
