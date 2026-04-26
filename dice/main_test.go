package dice_test

import (
	"testing"

	"go.uber.org/goleak"

	cache "sealdice-core/utils/cache"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.Cleanup(func(int) {
			cache.CloseAllOtterCachePlugins()
		}),
		// ants pool spawns background purge/ticktock goroutines that persist
		// until ants.Release() is called. Even after Release, they may take a
		// brief moment to exit.
		goleak.IgnoreTopFunction("github.com/panjf2000/ants/v2.(*poolCommon).purgeStaleWorkers"),
		goleak.IgnoreTopFunction("github.com/panjf2000/ants/v2.(*poolCommon).ticktock"),
		// bleve creates AnalysisWorker goroutines when an index is opened;
		// they are stopped only when the index is closed.
		goleak.IgnoreTopFunction("github.com/blevesearch/bleve_index_api.AnalysisWorker"),
	)
}
