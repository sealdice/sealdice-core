package sealkv

import (
	"github.com/philippgille/gokv"

	"sealdice-core/utils/gokv/bunt"
	"sealdice-core/utils/gokv/gormkv"
)

type SealDiceKVStore interface {
	gokv.Store
	// Clear 清理某个插件的所有数据
	Clear() error
}

// 实现检查 copied from platform
var (
	_ SealDiceKVStore = (*bunt.Store)(nil)
	_ SealDiceKVStore = (*gormkv.Store)(nil)
)
