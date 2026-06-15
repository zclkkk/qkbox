package qkboxd

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zclkkk/qkbox/internal/runtimeapi"
	"github.com/zclkkk/qkbox/shared/api"
	"github.com/zclkkk/qkbox/shared/model"
)

type fakeRuntimeOwner struct {
	startErr     error
	stopErr      error
	started      bool
	configJSON   string
	target       RuntimeStartTarget
	listeners    []runtimeapi.ListenerInfo
	startedCh    chan struct{}
	releaseStart <-chan struct{}
	stoppedCh    chan struct{}
}

func (f *fakeRuntimeOwner) Start(ctx context.Context, target RuntimeStartTarget) *api.StructuredError {
	f.target = target
	f.configJSON = target.ConfigJSON
	if f.startedCh != nil {
		close(f.startedCh)
	}
	if f.releaseStart != nil {
		select {
		case <-f.releaseStart:
		case <-ctx.Done():
			return api.NewStructuredError(api.ErrorSingboxAdapterStartFailed, ctx.Err().Error(), "test", false)
		}
	}
	if f.startErr != nil {
		return api.NewStructuredError(api.ErrorSingboxAdapterStartFailed, f.startErr.Error(), "test", false)
	}
	f.started = true
	return nil
}

func (f *fakeRuntimeOwner) Stop() *api.StructuredError {
	if f.stopErr != nil {
		return api.NewStructuredError(api.ErrorSingboxAdapterStopFailed, f.stopErr.Error(), "test", false)
	}
	f.started = false
	if f.stoppedCh != nil {
		close(f.stoppedCh)
	}
	return nil
}

func (f *fakeRuntimeOwner) RuntimeCapabilities() []api.Capability {
	return api.RuntimeCapabilityShell()
}

func (f *fakeRuntimeOwner) TrafficSnapshot() (api.TrafficSnapshot, *api.StructuredError) {
	return api.TrafficSnapshot{}, nil
}

func (f *fakeRuntimeOwner) ConnectionSnapshot() (api.ConnectionSnapshot, *api.StructuredError) {
	return api.ConnectionSnapshot{}, nil
}

func (f *fakeRuntimeOwner) ListGroups() ([]api.OutboundGroup, *api.StructuredError) {
	return nil, nil
}

func (f *fakeRuntimeOwner) SelectOutbound(string, string) (api.OutboundGroup, *api.StructuredError) {
	return api.OutboundGroup{}, nil
}

func (f *fakeRuntimeOwner) URLTest(context.Context, string, time.Duration) ([]api.URLTestResult, *api.StructuredError) {
	return nil, nil
}

func (f *fakeRuntimeOwner) CloseConnection(string) *api.StructuredError {
	return nil
}

func (f *fakeRuntimeOwner) CloseAllConnections() *api.StructuredError {
	return nil
}

func (f *fakeRuntimeOwner) ListenerInfo() ([]runtimeapi.ListenerInfo, *api.StructuredError) {
	if !f.started {
		return nil, api.NewStructuredError(api.ErrorEngineNotStarted, "Engine is not running.", "test", true)
	}
	if len(f.listeners) > 0 {
		return f.listeners, nil
	}
	return []runtimeapi.ListenerInfo{{Tag: "http", Type: "mixed", Address: "127.0.0.1", Port: 7890}}, nil
}

func TestEngineController_StartStopTransitions(t *testing.T) {
	ctrl := NewEngineController(context.Background(), NewRuntimeEventHub())
	fake := &fakeRuntimeOwner{}
	ctrl.runtimeOwnerFactory = func(RuntimeStartTarget) RuntimeOwner {
		return fake
	}

	// 1. Initial State
	if ctrl.GetStatus().State != model.EngineStateIdle {
		t.Fatalf("expected IDLE, got %s", ctrl.GetStatus().State)
	}

	// 2. Start Success
	err := ctrl.Start(RuntimeStartTarget{ProfileID: "profile1", ConfigJSON: "{}"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ctrl.GetStatus().State != model.EngineStateStarted {
		t.Fatalf("expected STARTED, got %s", ctrl.GetStatus().State)
	}
	if !fake.started {
		t.Fatalf("fake runtime owner was not started")
	}
	if ctrl.GetStatus().ActiveProfileID != "profile1" {
		t.Fatalf("expected profile ID profile1")
	}

	// 3. Prevent duplicate Start
	err = ctrl.Start(RuntimeStartTarget{ProfileID: "profile2", ConfigJSON: "{}"})
	if err == nil || err.Code != api.ErrorEngineAlreadyStarted {
		t.Fatalf("expected ENGINE_ALREADY_STARTED")
	}

	// 4. Activation switch is allowed from STARTED and handled by the service layer.
	if _, activationErr := ctrl.CheckActivationAllowed(); activationErr != nil {
		t.Fatalf("activation should be allowed while STARTED, got %v", activationErr)
	}

	// 5. Stop Success
	err = ctrl.Stop()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ctrl.GetStatus().State != model.EngineStateIdle {
		t.Fatalf("expected IDLE after stop, got %s", ctrl.GetStatus().State)
	}
	if fake.started {
		t.Fatalf("fake runtime owner was not stopped")
	}
	if ctrl.GetStatus().ActiveProfileID != "profile1" {
		t.Fatalf("ActiveProfileID should NOT be cleared after Stop")
	}

	// 6. Stop when already stopped
	err = ctrl.Stop()
	if err == nil || err.Code != api.ErrorEngineNotStarted {
		t.Fatalf("expected ENGINE_NOT_STARTED")
	}
}

func TestEngineController_RuntimeOwnerStartFailure(t *testing.T) {
	ctrl := NewEngineController(context.Background(), NewRuntimeEventHub())
	fake := &fakeRuntimeOwner{startErr: errors.New("boom")}
	ctrl.runtimeOwnerFactory = func(RuntimeStartTarget) RuntimeOwner {
		return fake
	}

	err := ctrl.Start(RuntimeStartTarget{ProfileID: "profile1", ConfigJSON: "{}"})
	if err == nil || err.Code != api.ErrorSingboxAdapterStartFailed {
		t.Fatalf("expected SINGBOX_ADAPTER_START_FAILED")
	}
	if ctrl.GetStatus().State != model.EngineStateIdle {
		t.Fatalf("expected IDLE, got %s", ctrl.GetStatus().State)
	}
	if ctrl.GetStatus().LastErrorMessage != "boom" {
		t.Fatalf("expected error message to be recorded")
	}
}

func TestEngineControllerRuntimeOwnerFactoryReceivesStartTarget(t *testing.T) {
	ctrl := NewEngineController(context.Background(), NewRuntimeEventHub())
	fake := &fakeRuntimeOwner{}
	var factoryTarget RuntimeStartTarget
	ctrl.runtimeOwnerFactory = func(target RuntimeStartTarget) RuntimeOwner {
		factoryTarget = target
		return fake
	}

	target := RuntimeStartTarget{
		ProfileID:            "profile1",
		ConfigJSON:           `{"inbounds":[],"outbounds":[{"type":"direct","tag":"direct"}]}`,
		RequiredCapabilities: []string{api.CapabilityTunMode},
	}
	if err := ctrl.Start(target); err != nil {
		t.Fatalf("start: %v", err)
	}
	if factoryTarget.ProfileID != target.ProfileID {
		t.Fatalf("factory target profile = %s, want %s", factoryTarget.ProfileID, target.ProfileID)
	}
	if len(factoryTarget.RequiredCapabilities) != 1 || factoryTarget.RequiredCapabilities[0] != api.CapabilityTunMode {
		t.Fatalf("factory target capabilities = %#v", factoryTarget.RequiredCapabilities)
	}
	if fake.target.ConfigJSON != target.ConfigJSON {
		t.Fatalf("owner start target config = %q", fake.target.ConfigJSON)
	}
}

func TestEngineController_StartIsObservableWhileRuntimeOwnerStarts(t *testing.T) {
	ctrl := NewEngineController(context.Background(), NewRuntimeEventHub())
	started := make(chan struct{})
	release := make(chan struct{})
	fake := &fakeRuntimeOwner{startedCh: started, releaseStart: release}
	ctrl.runtimeOwnerFactory = func(RuntimeStartTarget) RuntimeOwner {
		return fake
	}

	done := make(chan *api.StructuredError, 1)
	go func() {
		done <- ctrl.Start(RuntimeStartTarget{ProfileID: "profile1", ConfigJSON: "{}"})
	}()

	waitFor(t, started)
	if status := ctrl.GetStatus(); status.State != model.EngineStateStarting {
		t.Fatalf("expected STARTING while runtime owner starts, got %s", status.State)
	}
	if _, err := ctrl.CheckActivationAllowed(); err == nil || err.Code != api.ErrorEngineBusy {
		t.Fatalf("expected activation busy while STARTING")
	}
	if err := ctrl.Stop(); err == nil || err.Code != api.ErrorEngineBusy {
		t.Fatalf("expected ENGINE_BUSY while STARTING")
	}

	close(release)
	if err := waitResult(t, done); err != nil {
		t.Fatalf("start: %v", err)
	}
	if status := ctrl.GetStatus(); status.State != model.EngineStateStarted {
		t.Fatalf("expected STARTED, got %s", status.State)
	}
}

func TestEngineController_StopFailureBlocksMutationsUntilStopped(t *testing.T) {
	ctrl := NewEngineController(context.Background(), NewRuntimeEventHub())
	fake := &fakeRuntimeOwner{stopErr: errors.New("stop failed")}
	ctrl.runtimeOwnerFactory = func(RuntimeStartTarget) RuntimeOwner {
		return fake
	}

	if err := ctrl.Start(RuntimeStartTarget{ProfileID: "profile1", ConfigJSON: "{}"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	err := ctrl.Stop()
	if err == nil || err.Code != api.ErrorSingboxAdapterStopFailed {
		t.Fatalf("expected SINGBOX_ADAPTER_STOP_FAILED")
	}
	if status := ctrl.GetStatus(); status.State != model.EngineStateFatal {
		t.Fatalf("expected FATAL, got %s", status.State)
	}
	if _, err := ctrl.CheckActivationAllowed(); err == nil || err.Code != api.ErrorEngineBusy {
		t.Fatalf("expected activation busy after fatal stop failure")
	}

	fake.stopErr = nil
	if err := ctrl.Stop(); err != nil {
		t.Fatalf("retry stop: %v", err)
	}
	if status := ctrl.GetStatus(); status.State != model.EngineStateIdle {
		t.Fatalf("expected IDLE, got %s", status.State)
	}
}

func TestEngineControllerSubscribeTrafficUnavailableWhenStopped(t *testing.T) {
	ctrl := NewEngineController(context.Background(), NewRuntimeEventHub())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := ctrl.SubscribeTraffic(ctx)

	select {
	case event := <-events:
		if event.Event != api.EventEngineEventBridgeError {
			t.Fatalf("event = %s", event.Event)
		}
		if event.Error == nil || event.Error.Code != api.ErrorObservabilityUnavailable {
			t.Fatalf("error = %v", event.Error)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for unavailable event")
	}
}

func waitFor(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for signal")
	}
}

func waitResult(t *testing.T, ch <-chan *api.StructuredError) *api.StructuredError {
	t.Helper()
	select {
	case err := <-ch:
		return err
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for result")
	}
	return nil
}
