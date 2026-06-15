package qkboxd

import (
	"context"
	"sync"
	"time"

	"github.com/zclkkk/qkbox/internal/runtimeapi"
	"github.com/zclkkk/qkbox/shared/api"
	"github.com/zclkkk/qkbox/shared/model"
)

type EngineController struct {
	mu                  sync.Mutex
	runtimeCtx          context.Context
	events              *RuntimeEventHub
	runtimeOwnerFactory RuntimeOwnerFactory
	runtimeOwner        RuntimeOwner
	status              api.EngineStatus
}

func NewEngineController(runtimeCtx context.Context, events *RuntimeEventHub) *EngineController {
	controller := &EngineController{
		runtimeCtx:          runtimeCtx,
		events:              events,
		runtimeOwnerFactory: newLocalRuntimeOwnerFactory(events),
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

func (e *EngineController) Start(loadTarget func() (RuntimeStartTarget, *api.StructuredError)) *api.StructuredError {
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
	e.status.ActiveProfileID = target.ProfileID
	owner := e.runtimeOwnerFactory(target)
	status = e.status
	e.mu.Unlock()
	e.publishStatus(status)

	if err := owner.Start(e.runtimeCtx, target); err != nil {
		e.finishStartFailure(err.Code, err.Message)
		return err
	}

	e.mu.Lock()
	e.runtimeOwner = owner
	e.status.State = model.EngineStateStarted
	e.status.StartedAt = time.Now().UnixMilli()
	status = e.status
	e.mu.Unlock()
	e.publishStatus(status)

	return nil
}

func (e *EngineController) Stop() *api.StructuredError {
	e.mu.Lock()
	owner, structured := e.beginStopLocked()
	if structured != nil {
		e.mu.Unlock()
		return structured
	}
	status := e.status
	e.mu.Unlock()
	e.publishStatus(status)

	if err := owner.Stop(); err != nil {
		e.mu.Lock()
		e.status.State = model.EngineStateFatal
		e.status.LastErrorCode = err.Code
		e.status.LastErrorMessage = err.Message
		status = e.status
		e.mu.Unlock()
		e.publishStatus(status)
		return err
	}

	e.finishStopSuccess()
	return nil
}

func (e *EngineController) Shutdown() error {
	e.mu.Lock()
	owner, structured := e.beginStopLocked()
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

	if err := owner.Stop(); err != nil {
		e.mu.Lock()
		e.status.State = model.EngineStateFatal
		e.status.LastErrorCode = err.Code
		e.status.LastErrorMessage = err.Message
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
	e.runtimeOwner = nil
	e.status.State = model.EngineStateIdle
	e.status.StartedAt = 0
	e.status.LastErrorCode = ""
	e.status.LastErrorMessage = ""
	status := e.status
	e.mu.Unlock()
	e.publishStatus(status)
}

func (e *EngineController) CheckProfileSelectionMutation() *api.StructuredError {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.blocksMutationsLocked() {
		return api.NewStructuredError(api.ErrorEngineRunning, "Cannot change active profile while engine is running.", "qkboxd", true)
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
		if e.runtimeOwner != nil {
			return api.NewStructuredError(api.ErrorEngineBusy, "Engine is in a fatal runtime state. Stop it before starting again.", "qkboxd", true)
		}
	}
	e.status.State = model.EngineStateStarting
	e.status.StartedAt = 0
	e.status.LastErrorCode = ""
	e.status.LastErrorMessage = ""
	return nil
}

func (e *EngineController) beginStopLocked() (RuntimeOwner, *api.StructuredError) {
	switch e.status.State {
	case model.EngineStateStarting, model.EngineStateStopping:
		return nil, api.NewStructuredError(api.ErrorEngineBusy, "Engine is busy.", "qkboxd", true)
	case model.EngineStateStarted, model.EngineStateFatal:
		if e.runtimeOwner == nil {
			return nil, api.NewStructuredError(api.ErrorEngineNotStarted, "Engine is not running.", "qkboxd", true)
		}
		owner := e.runtimeOwner
		e.status.State = model.EngineStateStopping
		return owner, nil
	default:
		return nil, api.NewStructuredError(api.ErrorEngineNotStarted, "Engine is not running.", "qkboxd", true)
	}
}

func (e *EngineController) finishStartFailure(code, message string) {
	e.mu.Lock()
	e.status.State = model.EngineStateIdle
	e.runtimeOwner = nil
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
		(e.status.State == model.EngineStateFatal && e.runtimeOwner != nil)
}

func (e *EngineController) RuntimeCapabilities() []api.Capability {
	owner, err := e.runningOwner()
	if err != nil {
		return api.RuntimeCapabilityShell()
	}
	return owner.RuntimeCapabilities()
}

func (e *EngineController) TrafficSnapshot() (api.TrafficSnapshot, *api.StructuredError) {
	owner, err := e.runningOwner()
	if err != nil {
		return api.TrafficSnapshot{}, err
	}
	return owner.TrafficSnapshot()
}

func (e *EngineController) ConnectionSnapshot() (api.ConnectionSnapshot, *api.StructuredError) {
	owner, err := e.runningOwner()
	if err != nil {
		return api.ConnectionSnapshot{}, err
	}
	return owner.ConnectionSnapshot()
}

func (e *EngineController) ListGroups() ([]api.OutboundGroup, *api.StructuredError) {
	owner, err := e.runningOwner()
	if err != nil {
		return nil, err
	}
	return owner.ListGroups()
}

func (e *EngineController) SelectOutbound(groupTag, outboundTag string) (api.OutboundGroup, *api.StructuredError) {
	owner, err := e.runningOwner()
	if err != nil {
		return api.OutboundGroup{}, err
	}
	return owner.SelectOutbound(groupTag, outboundTag)
}

func (e *EngineController) URLTest(ctx context.Context, groupTag string, timeout time.Duration) ([]api.URLTestResult, *api.StructuredError) {
	owner, err := e.runningOwner()
	if err != nil {
		return nil, err
	}
	return owner.URLTest(ctx, groupTag, timeout)
}

func (e *EngineController) CloseConnection(id string) *api.StructuredError {
	owner, err := e.runningOwner()
	if err != nil {
		return err
	}
	return owner.CloseConnection(id)
}

func (e *EngineController) CloseAllConnections() *api.StructuredError {
	owner, err := e.runningOwner()
	if err != nil {
		return err
	}
	return owner.CloseAllConnections()
}

func (e *EngineController) ListenerInfo() ([]runtimeapi.ListenerInfo, *api.StructuredError) {
	owner, err := e.runningOwner()
	if err != nil {
		return nil, err
	}
	return owner.ListenerInfo()
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

func (e *EngineController) runningOwner() (RuntimeOwner, *api.StructuredError) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.status.State != model.EngineStateStarted || e.runtimeOwner == nil {
		return nil, api.NewStructuredError(api.ErrorEngineNotStarted, "Engine is not running.", "qkboxd", true)
	}
	return e.runtimeOwner, nil
}

func (e *EngineController) publishStatus(status api.EngineStatus) {
	if e.events != nil {
		e.events.PublishStatus(status)
	}
}
