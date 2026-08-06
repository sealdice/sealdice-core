package dice

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

var _ EndpointLifecycleDriver = (PlatformAdapter)(nil)

type endpointLifecycleTestDriver struct {
	mu        sync.Mutex
	startRuns []endpointLifecycleTestRun
	stopCalls int
	startCh   chan endpointLifecycleTestRun
}

type endpointLifecycleTestRun struct {
	ctx context.Context
	run EndpointRunReporter
}

func newEndpointLifecycleTestDriver() *endpointLifecycleTestDriver {
	return &endpointLifecycleTestDriver{
		startCh: make(chan endpointLifecycleTestRun, 16),
	}
}

func (d *endpointLifecycleTestDriver) LifecycleStart(ctx context.Context, run EndpointRunReporter) error {
	item := endpointLifecycleTestRun{ctx: ctx, run: run}

	d.mu.Lock()
	d.startRuns = append(d.startRuns, item)
	d.mu.Unlock()

	d.startCh <- item
	return nil
}

func (d *endpointLifecycleTestDriver) LifecycleStop(context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.stopCalls++
	return nil
}

func (d *endpointLifecycleTestDriver) startCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.startRuns)
}

func (d *endpointLifecycleTestDriver) stopCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.stopCalls
}

func newEndpointLifecycleTestSupervisor(t *testing.T) (*EndpointLifecycleSupervisor, *EndPointInfo, *endpointLifecycleTestDriver) {
	t.Helper()

	ep := &EndPointInfo{}
	driver := newEndpointLifecycleTestDriver()
	supervisor := NewEndpointLifecycleSupervisor(nil, ep, driver)
	supervisor.retryInitialDelay = 10 * time.Millisecond
	supervisor.retryMaxDelay = 10 * time.Millisecond

	return supervisor, ep, driver
}

func waitLifecycleRun(t *testing.T, driver *endpointLifecycleTestDriver) endpointLifecycleTestRun {
	t.Helper()

	select {
	case run := <-driver.startCh:
		return run
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for lifecycle start")
		return endpointLifecycleTestRun{}
	}
}

func assertNoLifecycleRun(t *testing.T, driver *endpointLifecycleTestDriver, timeout time.Duration) {
	t.Helper()

	select {
	case run := <-driver.startCh:
		t.Fatalf("unexpected lifecycle start for generation %d", run.run.Generation())
	case <-time.After(timeout):
	}
}

func TestEndpointLifecycleConcurrentEnableStartsOnce(t *testing.T) {
	supervisor, ep, driver := newEndpointLifecycleTestSupervisor(t)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := supervisor.Enable(context.Background()); err != nil {
				t.Errorf("Enable returned error: %v", err)
			}
		}()
	}
	wg.Wait()

	run := waitLifecycleRun(t, driver)
	if run.run.Generation() == 0 {
		t.Fatal("expected non-zero generation")
	}
	assertNoLifecycleRun(t, driver, 50*time.Millisecond)

	if got := driver.startCount(); got != 1 {
		t.Fatalf("expected one LifecycleStart call, got %d", got)
	}
	if !ep.Enable {
		t.Fatal("expected endpoint desired enable to remain true")
	}
	if ep.State != StateConnecting {
		t.Fatalf("expected endpoint state %v, got %v", StateConnecting, ep.State)
	}
}

func TestEndpointLifecycleDisableIgnoresLateStarted(t *testing.T) {
	supervisor, ep, driver := newEndpointLifecycleTestSupervisor(t)

	if err := supervisor.Enable(context.Background()); err != nil {
		t.Fatalf("Enable returned error: %v", err)
	}
	run := waitLifecycleRun(t, driver)

	if err := supervisor.Disable(context.Background()); err != nil {
		t.Fatalf("Disable returned error: %v", err)
	}
	run.run.Started()

	if got := driver.stopCount(); got != 1 {
		t.Fatalf("expected one LifecycleStop call, got %d", got)
	}
	if ep.Enable {
		t.Fatal("expected endpoint desired enable to be false")
	}
	if ep.State != StateDisconnected {
		t.Fatalf("expected endpoint state %v, got %v", StateDisconnected, ep.State)
	}
}

func TestEndpointLifecycleReloginStopsOldGenerationBeforeStartingNew(t *testing.T) {
	supervisor, ep, driver := newEndpointLifecycleTestSupervisor(t)

	if err := supervisor.Enable(context.Background()); err != nil {
		t.Fatalf("Enable returned error: %v", err)
	}
	first := waitLifecycleRun(t, driver)
	first.run.Started()

	if err := supervisor.Relogin(context.Background()); err != nil {
		t.Fatalf("Relogin returned error: %v", err)
	}
	second := waitLifecycleRun(t, driver)

	if second.run.Generation() <= first.run.Generation() {
		t.Fatalf("expected second generation to be newer than first: first=%d second=%d", first.run.Generation(), second.run.Generation())
	}
	if got := driver.stopCount(); got != 1 {
		t.Fatalf("expected relogin to stop the old generation once, got %d stops", got)
	}

	first.run.Closed(errors.New("old generation closed"))
	second.run.Started()

	if got := driver.startCount(); got != 2 {
		t.Fatalf("expected exactly two starts, got %d", got)
	}
	if !ep.Enable {
		t.Fatal("expected endpoint desired enable to remain true")
	}
	if ep.State != StateConnected {
		t.Fatalf("expected endpoint state %v, got %v", StateConnected, ep.State)
	}
}

func TestEndpointLifecycleOldGenerationClosedDoesNotReconnect(t *testing.T) {
	supervisor, _, driver := newEndpointLifecycleTestSupervisor(t)

	if err := supervisor.Enable(context.Background()); err != nil {
		t.Fatalf("Enable returned error: %v", err)
	}
	first := waitLifecycleRun(t, driver)
	first.run.Started()

	if err := supervisor.Relogin(context.Background()); err != nil {
		t.Fatalf("Relogin returned error: %v", err)
	}
	second := waitLifecycleRun(t, driver)

	first.run.Closed(errors.New("old generation closed"))
	assertNoLifecycleRun(t, driver, 50*time.Millisecond)

	second.run.Started()
	if got := driver.startCount(); got != 2 {
		t.Fatalf("expected old generation close to be ignored, got %d starts", got)
	}
}

func TestEndpointLifecycleConnectFailKeepsEnableAndRetries(t *testing.T) {
	supervisor, ep, driver := newEndpointLifecycleTestSupervisor(t)

	if err := supervisor.Enable(context.Background()); err != nil {
		t.Fatalf("Enable returned error: %v", err)
	}
	first := waitLifecycleRun(t, driver)
	first.run.Failed(errors.New("dial failed"))

	if !ep.Enable {
		t.Fatal("expected endpoint desired enable to stay true after connect failure")
	}
	if ep.State != StateConnectionFailed {
		t.Fatalf("expected endpoint state %v, got %v", StateConnectionFailed, ep.State)
	}

	second := waitLifecycleRun(t, driver)
	if second.run.Generation() <= first.run.Generation() {
		t.Fatalf("expected retry generation to be newer than failed generation")
	}
}

func TestEndpointLifecyclePermanentFailureKeepsEnableWithoutRetry(t *testing.T) {
	supervisor, ep, driver := newEndpointLifecycleTestSupervisor(t)
	supervisor.retryInitialDelay = 10 * time.Millisecond
	supervisor.retryMaxDelay = 10 * time.Millisecond

	if err := supervisor.Enable(context.Background()); err != nil {
		t.Fatalf("Enable returned error: %v", err)
	}
	first := waitLifecycleRun(t, driver)
	first.run.Failed(NewEndpointLifecycleFailure(errors.New("bad token"), LifecycleFailureStop))

	if !ep.Enable {
		t.Fatal("expected endpoint desired enable to stay true after permanent failure")
	}
	if ep.State != StateConnectionFailed {
		t.Fatalf("expected endpoint state %v, got %v", StateConnectionFailed, ep.State)
	}
	assertNoLifecycleRun(t, driver, 50*time.Millisecond)
}

func TestEndpointLifecycleDisableCancelsPendingRetry(t *testing.T) {
	supervisor, ep, driver := newEndpointLifecycleTestSupervisor(t)
	supervisor.retryInitialDelay = 80 * time.Millisecond
	supervisor.retryMaxDelay = 80 * time.Millisecond

	if err := supervisor.Enable(context.Background()); err != nil {
		t.Fatalf("Enable returned error: %v", err)
	}
	first := waitLifecycleRun(t, driver)
	first.run.Failed(errors.New("dial failed"))

	if err := supervisor.Disable(context.Background()); err != nil {
		t.Fatalf("Disable returned error: %v", err)
	}
	assertNoLifecycleRun(t, driver, 120*time.Millisecond)

	if ep.Enable {
		t.Fatal("expected endpoint desired enable to be false after disable")
	}
	if ep.State != StateDisconnected {
		t.Fatalf("expected endpoint state %v, got %v", StateDisconnected, ep.State)
	}
}

func TestEndpointLifecycleHelpersRejectMissingDriver(t *testing.T) {
	ep := &EndPointInfo{}

	if err := StartEndpointLifecycle(nil, ep); err == nil {
		t.Fatal("expected StartEndpointLifecycle to reject endpoint without adapter")
	}
	if err := StopEndpointLifecycle(nil, ep); err == nil {
		t.Fatal("expected StopEndpointLifecycle to reject endpoint without adapter")
	}
	if err := ReloginEndpointLifecycle(nil, ep); err == nil {
		t.Fatal("expected ReloginEndpointLifecycle to reject endpoint without adapter")
	}
}

func TestEndpointStateFromLifecycleCoversLifecycleStates(t *testing.T) {
	tests := []struct {
		state EndpointLifecycleState
		want  EndpointState
	}{
		{LifecycleDisconnected, StateDisconnected},
		{LifecycleConnecting, StateConnecting},
		{LifecycleConnected, StateConnected},
		{LifecycleFailed, StateConnectionFailed},
	}

	for _, tt := range tests {
		if got := EndpointStateFromLifecycle(tt.state); got != tt.want {
			t.Fatalf("EndpointStateFromLifecycle(%q) = %v, want %v", tt.state, got, tt.want)
		}
	}
}

func TestAllAdaptersImplementEndpointLifecycleDriver(t *testing.T) {
	drivers := []EndpointLifecycleDriver{
		&PlatformAdapterGocq{},
		&PlatformAdapterDiscord{},
		&PlatformAdapterDingTalk{},
		&PlatformAdapterDodo{},
		&PlatformAdapterHTTP{},
		&PlatformAdapterKook{},
		&PlatformAdapterMilky{},
		&PlatformAdapterMinecraft{},
		&PlatformAdapterOfficialQQ{},
		&PlatformAdapterOnebot{},
		&PlatformAdapterRed{},
		&PlatformAdapterSatori{},
		&PlatformAdapterSealChat{},
		&PlatformAdapterSlack{},
		&PlatformAdapterTelegram{},
		&PlatformAdapterWalleQ{},
	}

	if len(drivers) != 16 {
		t.Fatalf("expected all adapters to be lifecycle drivers")
	}
}
