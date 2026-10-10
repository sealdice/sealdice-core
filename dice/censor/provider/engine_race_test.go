package provider_test

import (
	"sync"
	"testing"

	"sealdice-core/dice/censor"
	"sealdice-core/dice/censor/provider"
)

func TestEngine_ConcurrentCheckAndReload(t *testing.T) {
	c := newCensor(map[string]censor.Level{"bad": censor.Danger})
	p := provider.NewLocalAC(c)
	e := provider.NewEngine(p)

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 200 {
				_, _ = e.Check(t.Context(), provider.Request{Text: "a bad b"})
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 50 {
			_ = p.Reload()
		}
	}()
	wg.Wait()
}
