package throttle

import (
	"time"

	"sealdice-core/utils"
)

var (
	lastExec = utils.SyncMap[string, time.Time]{}
)

// Do 简单的全局节流函数
// id: 任务唯一标识
// interval: 节流时间间隔
// f: 要执行的函数
func Do(id string, interval time.Duration, f func()) {
	if last, ok := lastExec.Load(id); ok && time.Since(last) < interval {
		return
	}

	f()
	lastExec.Store(id, time.Now())
}
