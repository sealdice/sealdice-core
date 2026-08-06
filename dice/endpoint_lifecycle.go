package dice

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	loopfsm "github.com/looplab/fsm"
)

// EndpointLifecycleState is the adapter-independent runtime state used by the
// endpoint lifecycle supervisor. It intentionally mirrors pureonebot's FSM
// shape so migrated adapters share the same enable/disable/retry semantics.
type EndpointLifecycleState string

const (
	// LifecycleDisconnected means no active run is owned by the supervisor.
	LifecycleDisconnected EndpointLifecycleState = "disconnected"
	// LifecycleConnecting means a driver Start call is in progress or waiting
	// for its asynchronous connected callback.
	LifecycleConnecting EndpointLifecycleState = "connecting"
	// LifecycleConnected means the current generation has reported a successful
	// connection and may receive traffic.
	LifecycleConnected EndpointLifecycleState = "connected"
	// LifecycleFailed means the current generation failed to start. The
	// supervisor may retry if the endpoint is still desired-enabled.
	LifecycleFailed EndpointLifecycleState = "failed"
)

// EndpointLifecycleEvent names every transition that can move the shared FSM.
// Keeping these values centralized avoids spreading string literals and magic
// numeric endpoint states across adapter implementations.
type EndpointLifecycleEvent string

const (
	// LifecycleEventEnable requests a transition from disconnected/failed to
	// connecting.
	LifecycleEventEnable EndpointLifecycleEvent = "enable"
	// LifecycleEventDisable requests a transition to disconnected and stops the
	// active generation.
	LifecycleEventDisable EndpointLifecycleEvent = "disable"
	// LifecycleEventConnectOK is reported by the current generation once the
	// protocol connection is usable.
	LifecycleEventConnectOK EndpointLifecycleEvent = "connect_ok"
	// LifecycleEventConnectFail is reported by the current generation when start
	// fails before a usable connection is established.
	LifecycleEventConnectFail EndpointLifecycleEvent = "connect_fail"
	// LifecycleEventConnectionLost is reported by the current generation when an
	// established connection is lost unexpectedly.
	LifecycleEventConnectionLost EndpointLifecycleEvent = "connection_lost"
	// LifecycleEventRelogin is represented by the public Relogin method. It is
	// kept as an enum so logs/tests can refer to the operation without inventing
	// another literal.
	LifecycleEventRelogin EndpointLifecycleEvent = "relogin"
)

// EndpointLifecycleFailurePolicy tells the supervisor whether a failed
// generation should be retried. The default is retryable so existing adapters
// can report ordinary network failures with run.Failed(err).
type EndpointLifecycleFailurePolicy int

const (
	// LifecycleFailureRetry means the endpoint remains desired-enabled and the
	// supervisor should retry according to its backoff policy.
	LifecycleFailureRetry EndpointLifecycleFailurePolicy = iota
	// LifecycleFailureStop means the failure is not useful to retry immediately,
	// for example invalid configuration, container-mode restriction, or device
	// lock/login rejection.
	LifecycleFailureStop
)

// EndpointLifecycleFailure wraps a driver error with retry policy metadata
// without forcing the reporter interface to grow more methods.
type EndpointLifecycleFailure struct {
	err    error
	policy EndpointLifecycleFailurePolicy
}

// NewEndpointLifecycleFailure marks err with a retry policy for supervisor
// decisions. A nil err is normalized so callers can still express the policy.
func NewEndpointLifecycleFailure(err error, policy EndpointLifecycleFailurePolicy) error {
	if err == nil {
		err = errors.New("endpoint lifecycle failure")
	}
	return &EndpointLifecycleFailure{err: err, policy: policy}
}

// Error implements error while preserving the wrapped protocol error text.
func (e *EndpointLifecycleFailure) Error() string {
	if e == nil || e.err == nil {
		return "endpoint lifecycle failure"
	}
	return e.err.Error()
}

// Unwrap exposes the protocol error to errors.Is/errors.As callers.
func (e *EndpointLifecycleFailure) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// Policy returns the supervisor retry policy carried by this error.
func (e *EndpointLifecycleFailure) Policy() EndpointLifecycleFailurePolicy {
	if e == nil {
		return LifecycleFailureRetry
	}
	return e.policy
}

func endpointLifecycleFailurePolicy(err error) EndpointLifecycleFailurePolicy {
	var lifecycleFailure *EndpointLifecycleFailure
	if errors.As(err, &lifecycleFailure) {
		return lifecycleFailure.Policy()
	}
	return LifecycleFailureRetry
}

// EndpointStateFromLifecycle maps the internal lifecycle enum to the legacy
// endpoint state enum consumed by API/UI clients.
func EndpointStateFromLifecycle(state EndpointLifecycleState) EndpointState {
	switch state {
	case LifecycleDisconnected:
		return StateDisconnected
	case LifecycleConnecting:
		return StateConnecting
	case LifecycleConnected:
		return StateConnected
	case LifecycleFailed:
		return StateConnectionFailed
	default:
		return StateConnectionFailed
	}
}

// EndpointLifecycleDriver is the small protocol-specific surface needed by the
// shared supervisor. Implementations should not start top-level reconnection
// loops by themselves; they should report lifecycle changes through run.
type EndpointLifecycleDriver interface {
	LifecycleStart(ctx context.Context, run EndpointRunReporter) error
	LifecycleStop(ctx context.Context) error
}

// EndpointRunReporter is scoped to one supervisor generation. Late reports from
// an old generation are ignored, which prevents stale websocket sessions from
// resurrecting endpoint state or scheduling new reconnects.
type EndpointRunReporter interface {
	Generation() uint64
	Started()
	Failed(error)
	Closed(error)
}

// EndpointLifecycleSupervisor serializes lifecycle operations for one endpoint.
// It owns the FSM, desired-enabled flag, current generation, cancellation, and
// retry timer so adapters do not need their own ad-hoc "serving" booleans.
type EndpointLifecycleSupervisor struct {
	opMu sync.Mutex
	mu   sync.Mutex

	d      *Dice
	ep     *EndPointInfo
	driver EndpointLifecycleDriver
	fsm    *loopfsm.FSM

	desiredEnabled bool
	generation     uint64
	runCtx         context.Context
	runCancel      context.CancelFunc

	retryTimer        *time.Timer
	retryAttempt      int
	retryInitialDelay time.Duration
	retryMaxDelay     time.Duration
	retryMaxAttempts  int
	stopTimeout       time.Duration
	failurePolicy     EndpointLifecycleFailurePolicy
}

// NewEndpointLifecycleSupervisor creates a lifecycle supervisor for tests and
// runtime binding. Runtime code should normally use ensureEndpointLifecycle so
// the supervisor remains attached to the endpoint.
func NewEndpointLifecycleSupervisor(d *Dice, ep *EndPointInfo, driver EndpointLifecycleDriver) *EndpointLifecycleSupervisor {
	s := &EndpointLifecycleSupervisor{
		d:                 d,
		ep:                ep,
		driver:            driver,
		desiredEnabled:    ep != nil && ep.Enable,
		retryInitialDelay: 2 * time.Second,
		retryMaxDelay:     30 * time.Second,
		retryMaxAttempts:  5,
		stopTimeout:       5 * time.Second,
	}
	s.fsm = loopfsm.NewFSM(
		string(LifecycleDisconnected),
		loopfsm.Events{
			{Name: string(LifecycleEventEnable), Src: []string{string(LifecycleDisconnected), string(LifecycleFailed)}, Dst: string(LifecycleConnecting)},
			{Name: string(LifecycleEventConnectOK), Src: []string{string(LifecycleConnecting)}, Dst: string(LifecycleConnected)},
			{Name: string(LifecycleEventConnectFail), Src: []string{string(LifecycleConnecting)}, Dst: string(LifecycleFailed)},
			{Name: string(LifecycleEventConnectionLost), Src: []string{string(LifecycleConnected)}, Dst: string(LifecycleConnecting)},
			{Name: string(LifecycleEventDisable), Src: []string{string(LifecycleConnecting), string(LifecycleConnected), string(LifecycleFailed), string(LifecycleDisconnected)}, Dst: string(LifecycleDisconnected)},
		},
		loopfsm.Callbacks{
			"enter_" + string(LifecycleConnecting):   s.onEnterConnecting,
			"enter_" + string(LifecycleConnected):    s.onEnterConnected,
			"enter_" + string(LifecycleFailed):       s.onEnterFailed,
			"enter_" + string(LifecycleDisconnected): s.onEnterDisconnected,
		},
	)
	return s
}

// Enable records the user's desired state and starts the endpoint unless it is
// already starting or running.
func (s *EndpointLifecycleSupervisor) Enable(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	s.opMu.Lock()
	defer s.opMu.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.desiredEnabled = true
	if s.ep != nil {
		s.ep.Enable = true
	}

	switch EndpointLifecycleState(s.fsm.Current()) {
	case LifecycleConnecting, LifecycleConnected:
		s.updateAndSaveLocked()
		return nil
	default:
		return s.eventLocked(ctx, LifecycleEventEnable)
	}
}

// Disable records the user's desired state, cancels retry/start work, and asks
// the driver to close the current protocol resources.
func (s *EndpointLifecycleSupervisor) Disable(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	s.opMu.Lock()
	defer s.opMu.Unlock()

	cancel := s.prepareStopLocked(ctx, false)
	if cancel != nil {
		cancel()
	}
	return s.stopDriver(ctx)
}

// Relogin replaces the active generation with a fresh one. Stopping the old
// generation before starting the new one is the core guard against split
// websocket connections.
func (s *EndpointLifecycleSupervisor) Relogin(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	s.opMu.Lock()
	defer s.opMu.Unlock()

	cancel := s.prepareStopLocked(ctx, true)
	if cancel != nil {
		cancel()
	}
	if err := s.stopDriver(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.desiredEnabled = true
	if s.ep != nil {
		s.ep.Enable = true
	}
	return s.eventLocked(ctx, LifecycleEventEnable)
}

func (s *EndpointLifecycleSupervisor) prepareStopLocked(ctx context.Context, keepDesiredEnabled bool) context.CancelFunc {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.desiredEnabled = keepDesiredEnabled
	if s.ep != nil {
		s.ep.Enable = keepDesiredEnabled
	}
	s.stopRetryLocked()

	cancel := s.runCancel
	s.runCancel = nil
	s.runCtx = nil
	_ = s.eventLocked(ctx, LifecycleEventDisable)
	return cancel
}

func (s *EndpointLifecycleSupervisor) stopDriver(parent context.Context) error {
	if s.driver == nil {
		return nil
	}
	timeout := s.stopTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	return s.driver.LifecycleStop(ctx)
}

func (s *EndpointLifecycleSupervisor) eventLocked(ctx context.Context, event EndpointLifecycleEvent) error {
	err := s.fsm.Event(ctx, string(event))
	var noTransition loopfsm.NoTransitionError
	if errors.As(err, &noTransition) {
		return nil
	}
	var invalid loopfsm.InvalidEventError
	if errors.As(err, &invalid) {
		return nil
	}
	return err
}

func (s *EndpointLifecycleSupervisor) onEnterConnecting(_ context.Context, _ *loopfsm.Event) {
	s.stopRetryLocked()
	s.failurePolicy = LifecycleFailureRetry
	s.generation++

	runCtx, cancel := context.WithCancel(context.Background())
	s.runCtx = runCtx
	s.runCancel = cancel

	if s.ep != nil {
		s.ep.State = EndpointStateFromLifecycle(LifecycleConnecting)
		s.ep.Enable = s.desiredEnabled
	}
	s.updateAndSaveLocked()

	generation := s.generation
	driver := s.driver
	reporter := &endpointLifecycleRunReporter{supervisor: s, generation: generation}
	go func() {
		if driver == nil {
			reporter.Failed(fmt.Errorf("endpoint lifecycle driver is nil"))
			return
		}
		if err := driver.LifecycleStart(runCtx, reporter); err != nil {
			reporter.Failed(err)
		}
	}()
}

func (s *EndpointLifecycleSupervisor) onEnterConnected(_ context.Context, _ *loopfsm.Event) {
	s.retryAttempt = 0
	if s.ep != nil {
		s.ep.State = EndpointStateFromLifecycle(LifecycleConnected)
		s.ep.Enable = true
	}
	s.updateAndSaveLocked()
}

func (s *EndpointLifecycleSupervisor) onEnterFailed(_ context.Context, _ *loopfsm.Event) {
	if s.ep != nil {
		s.ep.State = EndpointStateFromLifecycle(LifecycleFailed)
		s.ep.Enable = s.desiredEnabled
	}
	s.updateAndSaveLocked()
	if s.desiredEnabled && s.failurePolicy == LifecycleFailureRetry {
		s.scheduleRetryLocked()
	}
}

func (s *EndpointLifecycleSupervisor) onEnterDisconnected(_ context.Context, _ *loopfsm.Event) {
	if s.ep != nil {
		s.ep.State = EndpointStateFromLifecycle(LifecycleDisconnected)
		s.ep.Enable = s.desiredEnabled
	}
	s.updateAndSaveLocked()
}

func (s *EndpointLifecycleSupervisor) scheduleRetryLocked() {
	if s.retryMaxAttempts > 0 && s.retryAttempt >= s.retryMaxAttempts {
		return
	}

	s.retryAttempt++
	delay := s.retryInitialDelay
	if delay <= 0 {
		delay = 2 * time.Second
	}
	for i := 1; i < s.retryAttempt; i++ {
		delay *= 2
	}
	if s.retryMaxDelay > 0 && delay > s.retryMaxDelay {
		delay = s.retryMaxDelay
	}

	generation := s.generation
	s.stopRetryLocked()
	s.retryTimer = time.AfterFunc(delay, func() {
		s.retry(generation)
	})
}

func (s *EndpointLifecycleSupervisor) stopRetryLocked() {
	if s.retryTimer != nil {
		s.retryTimer.Stop()
		s.retryTimer = nil
	}
}

func (s *EndpointLifecycleSupervisor) retry(generation uint64) {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if generation != s.generation || !s.desiredEnabled || EndpointLifecycleState(s.fsm.Current()) != LifecycleFailed {
		return
	}
	_ = s.eventLocked(context.Background(), LifecycleEventEnable)
}

func (s *EndpointLifecycleSupervisor) reportStarted(generation uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if generation != s.generation || EndpointLifecycleState(s.fsm.Current()) != LifecycleConnecting {
		return
	}
	_ = s.eventLocked(context.Background(), LifecycleEventConnectOK)
}

func (s *EndpointLifecycleSupervisor) reportFailed(generation uint64, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if generation != s.generation {
		return
	}
	s.failurePolicy = endpointLifecycleFailurePolicy(err)
	switch EndpointLifecycleState(s.fsm.Current()) {
	case LifecycleConnecting:
		_ = s.eventLocked(context.Background(), LifecycleEventConnectFail)
	case LifecycleConnected:
		_ = s.eventLocked(context.Background(), LifecycleEventConnectionLost)
	}
}

func (s *EndpointLifecycleSupervisor) reportClosed(generation uint64, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if generation != s.generation || !s.desiredEnabled {
		return
	}
	s.failurePolicy = endpointLifecycleFailurePolicy(err)
	switch EndpointLifecycleState(s.fsm.Current()) {
	case LifecycleConnected:
		_ = s.eventLocked(context.Background(), LifecycleEventConnectionLost)
	case LifecycleConnecting:
		_ = s.eventLocked(context.Background(), LifecycleEventConnectFail)
	}
}

func (s *EndpointLifecycleSupervisor) updateAndSaveLocked() {
	if s.d == nil {
		return
	}
	s.d.LastUpdatedTime = time.Now().Unix()
	s.d.Save(false)
}

type endpointLifecycleRunReporter struct {
	supervisor *EndpointLifecycleSupervisor
	generation uint64
}

// Generation returns the run generation owned by this reporter.
func (r *endpointLifecycleRunReporter) Generation() uint64 {
	return r.generation
}

// Started marks this generation as connected if it is still current.
func (r *endpointLifecycleRunReporter) Started() {
	r.supervisor.reportStarted(r.generation)
}

// Failed marks this generation as failed if it is still current.
func (r *endpointLifecycleRunReporter) Failed(err error) {
	r.supervisor.reportFailed(r.generation, err)
}

// Closed reports an unexpected close for this generation if it is still current.
func (r *endpointLifecycleRunReporter) Closed(err error) {
	r.supervisor.reportClosed(r.generation, err)
}

func endpointLifecycleDriver(ep *EndPointInfo) (EndpointLifecycleDriver, bool) {
	if ep == nil || ep.Adapter == nil {
		return nil, false
	}
	driver, ok := ep.Adapter.(EndpointLifecycleDriver)
	return driver, ok
}

func endpointDice(d *Dice, ep *EndPointInfo) *Dice {
	if d != nil {
		return d
	}
	if ep != nil && ep.Session != nil {
		return ep.Session.Parent
	}
	return nil
}

func ensureEndpointLifecycle(d *Dice, ep *EndPointInfo, driver EndpointLifecycleDriver) *EndpointLifecycleSupervisor {
	if ep.lifecycle != nil {
		ep.lifecycle.d = endpointDice(d, ep)
		ep.lifecycle.driver = driver
		return ep.lifecycle
	}
	ep.lifecycle = NewEndpointLifecycleSupervisor(endpointDice(d, ep), ep, driver)
	return ep.lifecycle
}

// StartEndpointLifecycle enables an endpoint through the shared supervisor when
// the adapter supports it. Legacy adapters fall back to their existing method.
func StartEndpointLifecycle(d *Dice, ep *EndPointInfo) error {
	driver, ok := endpointLifecycleDriver(ep)
	if !ok {
		return fmt.Errorf("endpoint lifecycle driver is unavailable")
	}
	return ensureEndpointLifecycle(d, ep, driver).Enable(context.Background())
}

// StopEndpointLifecycle disables an endpoint through the shared supervisor when
// available. Legacy adapters fall back to their existing method.
func StopEndpointLifecycle(d *Dice, ep *EndPointInfo) error {
	driver, ok := endpointLifecycleDriver(ep)
	if !ok {
		return fmt.Errorf("endpoint lifecycle driver is unavailable")
	}
	return ensureEndpointLifecycle(d, ep, driver).Disable(context.Background())
}

// ReloginEndpointLifecycle restarts an endpoint through the shared supervisor
// when available. Legacy adapters fall back to their existing relogin method.
func ReloginEndpointLifecycle(d *Dice, ep *EndPointInfo) error {
	driver, ok := endpointLifecycleDriver(ep)
	if !ok {
		return fmt.Errorf("endpoint lifecycle driver is unavailable")
	}
	return ensureEndpointLifecycle(d, ep, driver).Relogin(context.Background())
}
