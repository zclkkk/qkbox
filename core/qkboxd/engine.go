package qkboxd

import (
	"context"
	"sync"
	"time"

	"github.com/zclkkk/qkbox/internal/singboxadapter"
	"github.com/zclkkk/qkbox/shared/api"
	"github.com/zclkkk/qkbox/shared/model"
)

type EngineAdapter interface {
	Start(ctx context.Context, configJSON string) error
	Stop() error
}

type EngineController struct {
	mu             sync.Mutex
	adapterFactory func() EngineAdapter
	adapter        EngineAdapter
	status         api.EngineStatus
}

func NewEngineController() *EngineController {
	return &EngineController{
		adapterFactory: func() EngineAdapter {
			return singboxadapter.NewAdapter()
		},
		status: api.EngineStatus{
			State: model.EngineStateIdle,
		},
	}
}

func (e *EngineController) GetStatus() api.EngineStatus {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.status
}

func (e *EngineController) Start(configJSON string, snapshotID string) *api.StructuredError {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.status.State == model.EngineStateStarting || e.status.State == model.EngineStateStarted || e.status.State == model.EngineStateStopping {
		return api.NewStructuredError(api.ErrorEngineAlreadyStarted, "Engine is already running or busy.", "qkboxd", true)
	}

	e.status.State = model.EngineStateStarting
	e.status.ActiveSnapshotID = snapshotID
	e.status.LastErrorCode = ""
	e.status.LastErrorMessage = ""

	if e.adapter == nil {
		e.adapter = e.adapterFactory()
	}

	// Currently creating a dummy context here, later it should be tied to daemon lifetime
	// or pass it explicitly. For now `context.Background()` is fine.
	importContext := context.Background()
	
	if err := e.adapter.Start(importContext, configJSON); err != nil {
		e.status.State = model.EngineStateFatal
		e.status.LastErrorCode = api.ErrorSingboxAdapterStartFailed
		e.status.LastErrorMessage = err.Error()

		if ae, ok := err.(*singboxadapter.AdapterError); ok {
			if ae.Code == "CONFIG_FAILED" {
				e.status.LastErrorCode = api.ErrorSingboxAdapterConfigFailed
			}
		}

		return api.NewStructuredError(e.status.LastErrorCode, e.status.LastErrorMessage, "singboxadapter", false)
	}

	e.status.State = model.EngineStateStarted
	e.status.StartedAt = time.Now().UnixMilli()

	return nil
}

func (e *EngineController) Stop() *api.StructuredError {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.status.State != model.EngineStateStarted && e.status.State != model.EngineStateFatal {
		return api.NewStructuredError(api.ErrorEngineNotStarted, "Engine is not running.", "qkboxd", true)
	}

	e.status.State = model.EngineStateStopping

	if e.adapter != nil {
		if err := e.adapter.Stop(); err != nil {
			e.status.State = model.EngineStateFatal
			e.status.LastErrorCode = api.ErrorSingboxAdapterStopFailed
			e.status.LastErrorMessage = err.Error()
			return api.NewStructuredError(e.status.LastErrorCode, e.status.LastErrorMessage, "singboxadapter", false)
		}
		e.adapter = nil
	}

	e.status.State = model.EngineStateIdle
	e.status.StartedAt = 0
	e.status.LastErrorCode = ""
	e.status.LastErrorMessage = ""

	return nil
}

// CheckBlockMutations returns an error if the engine is in a state that should block snapshot mutations.
func (e *EngineController) CheckBlockMutations() *api.StructuredError {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.status.State == model.EngineStateStarting || e.status.State == model.EngineStateStarted || e.status.State == model.EngineStateStopping {
		return api.NewStructuredError(api.ErrorEngineRunning, "Cannot mutate active snapshot while engine is running.", "qkboxd", true)
	}
	return nil
}
