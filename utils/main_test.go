//nolint:testpackage
package utils

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
	)
}
