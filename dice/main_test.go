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
		// otter cache workers are started by the mainline GORM cache plugin.
		goleak.IgnoreTopFunction("github.com/maypok86/otter/internal/unixtime.startTimer.func1"),
		goleak.IgnoreAnyFunction("github.com/maypok86/otter/internal/core.(*Cache[...]).cleanup"),
		goleak.IgnoreAnyFunction("github.com/maypok86/otter/internal/core.(*Cache[...]).process"),
	)
}
