//nolint:testpackage
package dice

import (
	"testing"
	"time"

	milky "github.com/Szzrain/Milky-go-sdk"
	"sealdice-core/utils/procs"
)

func assertMilkyState(t *testing.T, pa *PlatformAdapterMilky, want EndpointState) {
	t.Helper()
	deadline := time.Now().Add(6 * time.Second)
	for {
		pa.lifecycleMu.Lock()
		got := pa.EndPoint.State
		pa.lifecycleMu.Unlock()
		if got == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("state = %d, want %d", got, want)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestMilkyProcessExitAndOldProcessIsolation(t *testing.T) {
	for _, mode := range []string{"yogurt", "lagrangeV2"} {
		t.Run(mode, func(t *testing.T) {
			pa := &PlatformAdapterMilky{EndPoint: &EndPointInfo{}, BuiltInMode: mode}
			generation, _ := pa.beginMilkyProcess()
			old := procs.NewProcess("old")
			pa.registerMilkyProcess(generation, old)
			pa.finishMilkyProcess(generation, old)
			assertMilkyState(t, pa, StateDisconnected)
			generation, _ = pa.beginMilkyProcess()
			pa.registerMilkyProcess(generation, old)
			currentGeneration, _ := pa.beginMilkyProcess()
			current := procs.NewProcess("current")
			pa.registerMilkyProcess(currentGeneration, current)
			pa.EndPoint.State = StateConnected
			pa.finishMilkyProcess(generation, old)
			assertMilkyState(t, pa, StateConnected)
			if pa.MilkyProcess != current {
				t.Fatal("old exit cleared the new process")
			}
			if pa.startMilkySession(&milky.Session{}, generation) {
				t.Fatal("old process started a new session")
			}
		})
	}
}
