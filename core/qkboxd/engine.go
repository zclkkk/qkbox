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

type EngineStartTarget struct {
	SnapshotID string
	ConfigJSON string
}

type EngineController struct {
	mu             sync.Mutex
	runtimeCtx     context.Context
	adapterFactory func() EngineAdapter
	adapter        EngineAdapter
	status         api.EngineStatus
}

func NewEngineController(runtimeCtx context.Context) *EngineController {
	return &EngineController{
		runtimeCtx: runtimeCtx,
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

func (e *EngineController) Start(loadTarget func() (EngineStartTarget, *api.StructuredError)) *api.StructuredError {
	e.mu.Lock()
	if err := e.beginStartLocked(); err != nil {
		e.mu.Unlock()
		return err
	}
	e.mu.Unlock()

	target, structured := loadTarget()
	if structured != nil {
		e.finishStartFailure(structured.Code, structured.Message)
		return structured
	}

	e.mu.Lock()
	e.status.ActiveSnapshotID = target.SnapshotID
	adapter := e.adapterFactory()
	e.mu.Unlock()

	if err := adapter.Start(e.runtimeCtx, target.ConfigJSON); err != nil {
		code := api.ErrorSingboxAdapterStartFailed
		if ae, ok := err.(*singboxadapter.AdapterError); ok && ae.Code == "CONFIG_FAILED" {
			code = api.ErrorSingboxAdapterConfigFailed
		}
		message := err.Error()
		e.finishStartFailure(code, message)
		return api.NewStructuredError(code, message, "singboxadapter", false)
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	e.adapter = adapter
	e.status.State = model.EngineStateStarted
	e.status.StartedAt = time.Now().UnixMilli()

	return nil
}

func (e *EngineController) Stop() *api.StructuredError {
	e.mu.Lock()
	adapter, structured := e.beginStopLocked()
	if structured != nil {
		e.mu.Unlock()
		return structured
	}
	e.mu.Unlock()

	if err := adapter.Stop(); err != nil {
		message := err.Error()
		e.mu.Lock()
		e.status.State = model.EngineStateFatal
		e.status.LastErrorCode = api.ErrorSingboxAdapterStopFailed
		e.status.LastErrorMessage = message
		e.mu.Unlock()
		return api.NewStructuredError(api.ErrorSingboxAdapterStopFailed, message, "singboxadapter", false)
	}

	e.finishStopSuccess()
	return nil
}

func (e *EngineController) Shutdown() error {
	e.mu.Lock()
	adapter, structured := e.beginStopLocked()
	if structured != nil {
		e.mu.Unlock()
		if structured.Code == api.ErrorEngineNotStarted {
			return nil
		}
		return structured
	}
	e.mu.Unlock()

	if err := adapter.Stop(); err != nil {
		e.mu.Lock()
		e.status.State = model.EngineStateFatal
		e.status.LastErrorCode = api.ErrorSingboxAdapterStopFailed
		e.status.LastErrorMessage = err.Error()
		e.mu.Unlock()
		return err
	}
	e.finishStopSuccess()
	return nil
}

func (e *EngineController) finishStopSuccess() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.adapter = nil
	e.status.State = model.EngineStateIdle
	e.status.StartedAt = 0
	e.status.LastErrorCode = ""
	e.status.LastErrorMessage = ""
}

// CheckBlockMutations returns an error if the engine is in a state that should block snapshot mutations.
func (e *EngineController) CheckBlockMutations() *api.StructuredError {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.blocksMutationsLocked() {
		return api.NewStructuredError(api.ErrorEngineRunning, "Cannot mutate active snapshot while engine is running.", "qkboxd", true)
	}
	return nil
}

func (e *EngineController) beginStartLocked() *api.StructuredError {
	switch e.status.State {
	case model.EngineStateStarted:
		return api.NewStructuredError(api.ErrorEngineAlreadyStarted, "Engine is already started.", "qkboxd", true)
	case model.EngineStateStarting, model.EngineStateStopping:
		return api.NewStructuredError(api.ErrorEngineBusy, "Engine is busy.", "qkboxd", true)
	case model.EngineStateFatal:
		if e.adapter != nil {
			return api.NewStructuredError(api.ErrorEngineBusy, "Engine is in a fatal runtime state. Stop it before starting again.", "qkboxd", true)
		}
	}
	e.status.State = model.EngineStateStarting
	e.status.StartedAt = 0
	e.status.LastErrorCode = ""
	e.status.LastErrorMessage = ""
	return nil
}

func (e *EngineController) beginStopLocked() (EngineAdapter, *api.StructuredError) {
	switch e.status.State {
	case model.EngineStateStarting, model.EngineStateStopping:
		return nil, api.NewStructuredError(api.ErrorEngineBusy, "Engine is busy.", "qkboxd", true)
	case model.EngineStateStarted, model.EngineStateFatal:
		if e.adapter == nil {
			return nil, api.NewStructuredError(api.ErrorEngineNotStarted, "Engine is not running.", "qkboxd", true)
		}
		adapter := e.adapter
		e.status.State = model.EngineStateStopping
		return adapter, nil
	default:
		return nil, api.NewStructuredError(api.ErrorEngineNotStarted, "Engine is not running.", "qkboxd", true)
	}
}

func (e *EngineController) finishStartFailure(code, message string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status.State = model.EngineStateIdle
	e.adapter = nil
	e.status.StartedAt = 0
	e.status.LastErrorCode = code
	e.status.LastErrorMessage = message
}

func (e *EngineController) blocksMutationsLocked() bool {
	return e.status.State == model.EngineStateStarting ||
		e.status.State == model.EngineStateStarted ||
		e.status.State == model.EngineStateStopping ||
		(e.status.State == model.EngineStateFatal && e.adapter != nil)
}
