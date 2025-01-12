package plugin_store

import "github.com/philippgille/gokv"

// 又得定义一个PluginStorage的接口封装了

type PluginStorage interface {
	// StorageInit 初始化
	StorageInit(pluginName string) (gokv.Store, error)
	// StorageClose 关闭
	StorageClose() error
	// StorageSet 为某个插件设置
	StorageSet(pluginName, k, v string) error
	// StorageGet 获取
	StorageGet(pluginName string, k string) (string, error)
	// StorageClear 清空对应插件数据库的数据
	StorageClear(pluginName string) error
	// StorageStop 停止PluginStorage，释放所有资源，安全做存储 -> TODO: 是在数据库层面关闭，还是将这部分逻辑挪移给Close合适？
	StorageStop() error
}

// 实现检查 copied from platform
var (
	_ PluginStorage = (*BuntDBPluginStorage)(nil)
	_ PluginStorage = (*GormPluginStorage)(nil)
)
