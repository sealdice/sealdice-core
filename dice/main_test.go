package dice_test

import (
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		// ants pool spawns background purge/ticktock goroutines that persist
		// until ants.Release() is called. Even after Release, they may take a
		// brief moment to exit.
		goleak.IgnoreTopFunction("github.com/panjf2000/ants/v2.(*poolCommon).purgeStaleWorkers"),
		goleak.IgnoreTopFunction("github.com/panjf2000/ants/v2.(*poolCommon).ticktock"),
		// 其它 otter 缓存（如 onebot 群缓存）会启动后台协程。
		goleak.IgnoreTopFunction("github.com/maypok86/otter/internal/unixtime.startTimer.func1"),
		goleak.IgnoreAnyFunction("github.com/maypok86/otter/internal/core.(*Cache[...]).cleanup"),
		// buntdb 的后台协程只会在下一次 1s tick 时发现库已关闭并退出，
		// 快于 goleak 的重试窗口之外，属于库自身行为。
		goleak.IgnoreTopFunction("github.com/tidwall/buntdb.(*DB).backgroundManager"),
		goleak.IgnoreAnyFunction("github.com/maypok86/otter/internal/core.(*Cache[...]).process"),
	)
}
