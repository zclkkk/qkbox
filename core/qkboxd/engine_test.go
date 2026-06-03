package qkboxd

import (
	"context"
	"errors"
	"testing"

	"github.com/zclkkk/qkbox/shared/api"
	"github.com/zclkkk/qkbox/shared/model"
)

type fakeAdapter struct {
	startErr error
	stopErr  error
	started  bool
}

func (f *fakeAdapter) Start(ctx context.Context, configJSON string) error {
	if f.startErr != nil {
		return f.startErr
	}
	f.started = true
	return nil
}

func (f *fakeAdapter) Stop() error {
	if f.stopErr != nil {
		return f.stopErr
	}
	f.started = false
	return nil
}

func TestEngineController_StartStopTransitions(t *testing.T) {
	ctrl := NewEngineController()
	fake := &fakeAdapter{}
	ctrl.adapterFactory = func() EngineAdapter {
		return fake
	}

	// 1. Initial State
	if ctrl.GetStatus().State != model.EngineStateIdle {
		t.Fatalf("expected IDLE, got %s", ctrl.GetStatus().State)
	}

	// 2. Start Success
	err := ctrl.Start("{}", "snap1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ctrl.GetStatus().State != model.EngineStateStarted {
		t.Fatalf("expected STARTED, got %s", ctrl.GetStatus().State)
	}
	if !fake.started {
		t.Fatalf("fake adapter was not started")
	}
	if ctrl.GetStatus().ActiveSnapshotID != "snap1" {
		t.Fatalf("expected snapshot ID snap1")
	}

	// 3. Prevent duplicate Start
	err = ctrl.Start("{}", "snap2")
	if err == nil || err.Code != api.ErrorEngineAlreadyStarted {
		t.Fatalf("expected ENGINE_ALREADY_STARTED")
	}

	// 4. Block Mutations
	mutErr := ctrl.CheckBlockMutations()
	if mutErr == nil || mutErr.Code != api.ErrorEngineRunning {
		t.Fatalf("expected ENGINE_RUNNING when mutating")
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
		t.Fatalf("fake adapter was not stopped")
	}
	if ctrl.GetStatus().ActiveSnapshotID != "snap1" {
		t.Fatalf("ActiveSnapshotID should NOT be cleared after Stop")
	}

	// 6. Stop when already stopped
	err = ctrl.Stop()
	if err == nil || err.Code != api.ErrorEngineNotStarted {
		t.Fatalf("expected ENGINE_NOT_STARTED")
	}
}

func TestEngineController_AdapterStartFailure(t *testing.T) {
	ctrl := NewEngineController()
	fake := &fakeAdapter{startErr: errors.New("boom")}
	ctrl.adapterFactory = func() EngineAdapter {
		return fake
	}

	err := ctrl.Start("{}", "snap1")
	if err == nil || err.Code != api.ErrorSingboxAdapterStartFailed {
		t.Fatalf("expected SINGBOX_ADAPTER_START_FAILED")
	}
	if ctrl.GetStatus().State != model.EngineStateFatal {
		t.Fatalf("expected FATAL, got %s", ctrl.GetStatus().State)
	}
	if ctrl.GetStatus().LastErrorMessage != "boom" {
		t.Fatalf("expected error message to be recorded")
	}
}
