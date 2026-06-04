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
	RuntimeCapabilities() []api.Capability
	TrafficSnapshot() (api.TrafficSnapshot, *api.StructuredError)
	ConnectionSnapshot() (api.ConnectionSnapshot, *api.StructuredError)
	ListGroups() ([]api.OutboundGroup, *api.StructuredError)
	SelectOutbound(groupTag, outboundTag string) (api.OutboundGroup, *api.StructuredError)
	URLTest(ctx context.Context, groupTag string, timeout time.Duration) ([]api.URLTestResult, *api.StructuredError)
	CloseConnection(id string) *api.StructuredError
	CloseAllConnections() *api.StructuredError
}

type EngineStartTarget struct {
	SnapshotID string
	ConfigJSON string
}

type EngineController struct {
	mu             sync.Mutex
	runtimeCtx     context.Context
	events         *RuntimeEventHub
	adapterFactory func() EngineAdapter
	adapter        EngineAdapter
	status         api.EngineStatus
}

func NewEngineController(runtimeCtx context.Context, events *RuntimeEventHub) *EngineController {
	controller := &EngineController{
		runtimeCtx: runtimeCtx,
		events:     events,
		adapterFactory: func() EngineAdapter {
			return singboxadapter.NewAdapter(events)
		},
		status: api.EngineStatus{
			State: model.EngineStateIdle,
		},
	}
	controller.publishStatus(controller.status)
	return controller
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
	status := e.status
	e.mu.Unlock()
	e.publishStatus(status)

	target, structured := loadTarget()
	if structured != nil {
		e.finishStartFailure(structured.Code, structured.Message)
		return structured
	}

	e.mu.Lock()
	e.status.ActiveSnapshotID = target.SnapshotID
	adapter := e.adapterFactory()
	status = e.status
	e.mu.Unlock()
	e.publishStatus(status)

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
	e.adapter = adapter
	e.status.State = model.EngineStateStarted
	e.status.StartedAt = time.Now().UnixMilli()
	status = e.status
	e.mu.Unlock()
	e.publishStatus(status)

	return nil
}

func (e *EngineController) Stop() *api.StructuredError {
	e.mu.Lock()
	adapter, structured := e.beginStopLocked()
	if structured != nil {
		e.mu.Unlock()
		return structured
	}
	status := e.status
	e.mu.Unlock()
	e.publishStatus(status)

	if err := adapter.Stop(); err != nil {
		message := err.Error()
		e.mu.Lock()
		e.status.State = model.EngineStateFatal
		e.status.LastErrorCode = api.ErrorSingboxAdapterStopFailed
		e.status.LastErrorMessage = message
		status = e.status
		e.mu.Unlock()
		e.publishStatus(status)
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
	status := e.status
	e.mu.Unlock()
	e.publishStatus(status)

	if err := adapter.Stop(); err != nil {
		e.mu.Lock()
		e.status.State = model.EngineStateFatal
		e.status.LastErrorCode = api.ErrorSingboxAdapterStopFailed
		e.status.LastErrorMessage = err.Error()
		status = e.status
		e.mu.Unlock()
		e.publishStatus(status)
		return err
	}
	e.finishStopSuccess()
	return nil
}

func (e *EngineController) finishStopSuccess() {
	e.mu.Lock()
	e.adapter = nil
	e.status.State = model.EngineStateIdle
	e.status.StartedAt = 0
	e.status.LastErrorCode = ""
	e.status.LastErrorMessage = ""
	status := e.status
	e.mu.Unlock()
	e.publishStatus(status)
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
	e.status.State = model.EngineStateIdle
	e.adapter = nil
	e.status.StartedAt = 0
	e.status.LastErrorCode = code
	e.status.LastErrorMessage = message
	status := e.status
	e.mu.Unlock()
	e.publishStatus(status)
}

func (e *EngineController) blocksMutationsLocked() bool {
	return e.status.State == model.EngineStateStarting ||
		e.status.State == model.EngineStateStarted ||
		e.status.State == model.EngineStateStopping ||
		(e.status.State == model.EngineStateFatal && e.adapter != nil)
}

func (e *EngineController) RuntimeCapabilities() []api.Capability {
	adapter, err := e.runningAdapter()
	if err != nil {
		return api.RuntimeCapabilityShell()
	}
	return adapter.RuntimeCapabilities()
}

func (e *EngineController) TrafficSnapshot() (api.TrafficSnapshot, *api.StructuredError) {
	adapter, err := e.runningAdapter()
	if err != nil {
		return api.TrafficSnapshot{}, err
	}
	return adapter.TrafficSnapshot()
}

func (e *EngineController) ConnectionSnapshot() (api.ConnectionSnapshot, *api.StructuredError) {
	adapter, err := e.runningAdapter()
	if err != nil {
		return api.ConnectionSnapshot{}, err
	}
	return adapter.ConnectionSnapshot()
}

func (e *EngineController) ListGroups() ([]api.OutboundGroup, *api.StructuredError) {
	adapter, err := e.runningAdapter()
	if err != nil {
		return nil, err
	}
	return adapter.ListGroups()
}

func (e *EngineController) SelectOutbound(groupTag, outboundTag string) (api.OutboundGroup, *api.StructuredError) {
	adapter, err := e.runningAdapter()
	if err != nil {
		return api.OutboundGroup{}, err
	}
	return adapter.SelectOutbound(groupTag, outboundTag)
}

func (e *EngineController) URLTest(ctx context.Context, groupTag string, timeout time.Duration) ([]api.URLTestResult, *api.StructuredError) {
	adapter, err := e.runningAdapter()
	if err != nil {
		return nil, err
	}
	return adapter.URLTest(ctx, groupTag, timeout)
}

func (e *EngineController) CloseConnection(id string) *api.StructuredError {
	adapter, err := e.runningAdapter()
	if err != nil {
		return err
	}
	return adapter.CloseConnection(id)
}

func (e *EngineController) CloseAllConnections() *api.StructuredError {
	adapter, err := e.runningAdapter()
	if err != nil {
		return err
	}
	return adapter.CloseAllConnections()
}

func (e *EngineController) SubscribeTraffic(ctx context.Context) <-chan api.RuntimeEvent {
	return e.subscribeSnapshots(ctx, api.EventEngineTraffic, "Traffic source is unavailable while engine is not started.", func() (interface{}, *api.StructuredError) {
		snapshot, err := e.TrafficSnapshot()
		if err != nil {
			return nil, err
		}
		return snapshot, nil
	})
}

func (e *EngineController) SubscribeConnections(ctx context.Context) <-chan api.RuntimeEvent {
	return e.subscribeSnapshots(ctx, api.EventEngineConnections, "Connection source is unavailable while engine is not started.", func() (interface{}, *api.StructuredError) {
		snapshot, err := e.ConnectionSnapshot()
		if err != nil {
			return nil, err
		}
		return snapshot, nil
	})
}

func (e *EngineController) subscribeSnapshots(ctx context.Context, eventName, unavailableMessage string, load func() (interface{}, *api.StructuredError)) <-chan api.RuntimeEvent {
	ch := make(chan api.RuntimeEvent, 8)
	go func() {
		defer close(ch)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			data, err := load()
			event := api.RuntimeEvent{Event: eventName, Data: data}
			if err != nil {
				if err.Code == api.ErrorEngineNotStarted {
					err = api.NewStructuredError(api.ErrorObservabilityUnavailable, unavailableMessage, "qkboxd", true)
				}
				event = api.RuntimeEvent{Event: api.EventEngineEventBridgeError, Error: err}
			}
			select {
			case ch <- event:
			case <-ctx.Done():
				return
			}
			select {
			case <-ticker.C:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch
}

func (e *EngineController) runningAdapter() (EngineAdapter, *api.StructuredError) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.status.State != model.EngineStateStarted || e.adapter == nil {
		return nil, api.NewStructuredError(api.ErrorEngineNotStarted, "Engine is not running.", "qkboxd", true)
	}
	return e.adapter, nil
}

func (e *EngineController) publishStatus(status api.EngineStatus) {
	if e.events != nil {
		e.events.PublishStatus(status)
	}
}
