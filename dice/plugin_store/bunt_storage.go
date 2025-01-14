package plugin_store

import (
	"errors"
	"path"

	"sealdice-core/utils"
	sealkv "sealdice-core/utils/gokv"
	"sealdice-core/utils/gokv/bunt"
	log "sealdice-core/utils/kratos"
)

// BuntDBPluginStorage 需要直接将buntdb的实现成gokv的实现
// 是否应该提供一个Migrator，将BuntDB的数据迁移到gokv中？这个事情涉及比较大，或许得进一步研究
// 当前考虑: 如果设置PLUGINDB = xxxx的时候，执行迁移，并删除原有数据（或者改个名如何？）
type BuntDBPluginStorage struct {
	// path,应当来源于d.GetExtDataDir(i.Name)
	Path string
	// 按道理说，这个家伙有概率并发读，但应该没可能并发写？
	// 以防万一，用SyncMap罢
	KVMap utils.SyncMap[string, *bunt.Store]
}

func (b *BuntDBPluginStorage) GetStorageInstance(pluginName string) (sealkv.SealDiceKVStore, error) {
	// 先看KVMap里有没有，若没有，则创建
	value, ok := b.KVMap.Load(pluginName)
	if ok {
		return value, nil
	}
	// 尝试进行初始化
	options := bunt.Options{
		Path: path.Join(b.Path, pluginName, "storage.db"),
	}
	store, err := bunt.NewStore(options)
	if err != nil {
		return nil, err
	}
	// 存储对它们的引用
	b.KVMap.Store(pluginName, &store)
	return store, nil
}

func (b *BuntDBPluginStorage) StorageClose() error {
	log.Infof("closing buntdb plugin storage")
	b.KVMap.Range(func(key string, value *bunt.Store) bool {
		v := value
		value = nil
		err := v.Close()
		if err != nil {
			log.Errorf("close buntdb error: %v", err)
			return true
		}
		return true
	})
	return nil
}

// 初始化
func NewBuntDBPluginStorage(path string) (*BuntDBPluginStorage, error) {
	if path == "" {
		return nil, errors.New("path is nil")
	}
	return &BuntDBPluginStorage{
		Path: path,
	}, nil
}
