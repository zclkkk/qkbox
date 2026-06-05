package providerruntime

import (
	"context"
	"sync"
	"time"

	"github.com/zclkkk/qkbox/internal/eventhub"
	"github.com/zclkkk/qkbox/internal/provideripc"
	"github.com/zclkkk/qkbox/internal/singboxadapter"
	"github.com/zclkkk/qkbox/shared/api"
	"github.com/zclkkk/qkbox/shared/model"
)

const defaultHeartbeatTimeout = 8 * time.Second

type Controller struct {
	mu                sync.Mutex
	opMu              sync.Mutex
	stateDir          string
	available         bool
	unavailableReason string
	events            *eventhub.Hub
	adapter           *singboxadapter.Adapter
	owner             *ownerRecord
	heartbeatCancel   context.CancelFunc
}

func NewController(stateDir string, available bool, unavailableReason string) *Controller {
	controller := &Controller{
		stateDir:          stateDir,
		available:         available,
		unavailableReason: unavailableReason,
		events:            eventhub.New(),
	}
	if record, err := loadOwnerRecord(stateDir); err == nil && record != nil {
		record.Stale = true
		record.Reason = "Provider stopped before clearing machine network runtime owner."
		record.RepairActions = []string{api.RepairActionClearMachineNetworkOwner}
		controller.owner = record
		_ = saveOwnerRecord(stateDir, record)
	} else if err != nil {
		controller.events.PublishBridgeError(api.NewStructuredError(api.ErrorProviderRuntimeStale, err.Error(), "provider", true))
	}
	return controller
}

func (c *Controller) Close() {
	c.mu.Lock()
	adapter := c.adapter
	record := c.owner
	c.mu.Unlock()
	if adapter == nil || record == nil || record.Stale {
		return
	}
	_, _ = c.RuntimeStop(context.Background(), provideripc.RuntimeStopRequest{
		SessionID: record.SessionID,
		RuntimeID: record.RuntimeID,
	})
}

func (c *Controller) Capabilities() []api.Capability {
	if c.available {
		return []api.Capability{
			{Name: api.CapabilityTunMode, State: api.CapabilityAvailable},
			{Name: api.CapabilityDNSHijack, State: api.CapabilityAvailable},
			{Name: api.CapabilityConnectionTracking, State: api.CapabilityAvailable},
		}
	}
	return []api.Capability{
		{Name: api.CapabilityTunMode, State: api.CapabilityUnavailable, Reason: c.unavailableReason},
		{Name: api.CapabilityDNSHijack, State: api.CapabilityUnavailable, Reason: c.unavailableReason},
		{Name: api.CapabilityConnectionTracking, State: api.CapabilityUnavailable, Reason: c.unavailableReason},
	}
}

func (c *Controller) OwnerState() *api.ProviderOwnerState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return providerOwnerState(c.owner)
}

func (c *Controller) RuntimeStart(ctx context.Context, req provideripc.RuntimeStartRequest) (provideripc.RuntimeStartReply, *api.StructuredError) {
	c.opMu.Lock()
	defer c.opMu.Unlock()

	if !c.available {
		return provideripc.RuntimeStartReply{}, api.NewStructuredError(api.ErrorPlatformFeatureUnsupported, c.unavailableReason, "provider", true)
	}
	if req.SessionID == "" || req.RuntimeID == "" || req.SnapshotID == "" {
		return provideripc.RuntimeStartReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, "session_id, runtime_id, and snapshot_id are required.", "provider", true)
	}
	if req.Mode != api.RuntimeModeMachineNetwork {
		return provideripc.RuntimeStartReply{}, api.NewStructuredError(api.ErrorPlatformFeatureUnsupported, "Runtime mode is not supported by this provider.", "provider", true)
	}

	c.mu.Lock()
	if c.owner != nil {
		if c.owner.Stale {
			state := providerOwnerState(c.owner)
			c.mu.Unlock()
			err := api.NewStructuredError(api.ErrorProviderRuntimeStale, "Machine network runtime owner is stale and must be repaired before starting.", "provider", true)
			err.Detail = state
			return provideripc.RuntimeStartReply{}, err
		}
		if c.owner.SessionID == req.SessionID && c.owner.RuntimeID == req.RuntimeID {
			state := providerOwnerState(c.owner)
			c.mu.Unlock()
			return provideripc.RuntimeStartReply{OwnerState: *state}, nil
		}
		state := providerOwnerState(c.owner)
		c.mu.Unlock()
		err := api.NewStructuredError(api.ErrorNetworkModeOwnedByAnotherSession, "Machine network mode is owned by another qkbox session.", "provider", true)
		err.Detail = state
		return provideripc.RuntimeStartReply{}, err
	}

	now := time.Now().UnixMilli()
	record := &ownerRecord{
		Owned:           true,
		SessionID:       req.SessionID,
		RuntimeID:       req.RuntimeID,
		SnapshotID:      req.SnapshotID,
		Mode:            req.Mode,
		StartedAt:       now,
		LastHeartbeatAt: now,
	}
	if err := saveOwnerRecord(c.stateDir, record); err != nil {
		c.mu.Unlock()
		return provideripc.RuntimeStartReply{}, api.NewStructuredError(api.ErrorInternal, err.Error(), "provider", false)
	}
	adapter := singboxadapter.NewAdapter(c.events)
	c.owner = record
	c.mu.Unlock()

	if err := adapter.Start(ctx, req.ConfigJSON); err != nil {
		_ = adapter.Stop()
		_ = deleteOwnerRecord(c.stateDir)
		c.mu.Lock()
		if c.adapter == adapter {
			c.adapter = nil
			c.owner = nil
		}
		c.mu.Unlock()
		structured := api.NewStructuredError(api.ErrorProviderRuntimeStartFailed, err.Error(), "provider", false)
		return provideripc.RuntimeStartReply{}, structured
	}

	c.mu.Lock()
	if c.owner == record {
		c.adapter = adapter
	}
	c.mu.Unlock()
	c.startHeartbeatMonitor(timeoutFromRequest(req.HeartbeatTimeoutMS))
	state := providerOwnerState(record)
	return provideripc.RuntimeStartReply{OwnerState: *state}, nil
}

func (c *Controller) RuntimeStop(_ context.Context, req provideripc.RuntimeStopRequest) (provideripc.RuntimeStopReply, *api.StructuredError) {
	c.opMu.Lock()
	defer c.opMu.Unlock()

	c.mu.Lock()
	adapter := c.adapter
	record := c.owner
	if record == nil {
		c.mu.Unlock()
		return provideripc.RuntimeStopReply{}, nil
	}
	if req.SessionID == "" || req.RuntimeID == "" {
		c.mu.Unlock()
		return provideripc.RuntimeStopReply{}, api.NewStructuredError(api.ErrorIPCInvalidRequest, "session_id and runtime_id are required.", "provider", true)
	}
	if record.SessionID != req.SessionID || record.RuntimeID != req.RuntimeID {
		state := providerOwnerState(record)
		c.mu.Unlock()
		err := api.NewStructuredError(api.ErrorNetworkModeOwnedByAnotherSession, "Machine network mode is owned by another qkbox session.", "provider", true)
		err.Detail = state
		return provideripc.RuntimeStopReply{}, err
	}
	if record.Stale && adapter == nil {
		state := providerOwnerState(record)
		c.mu.Unlock()
		err := api.NewStructuredError(api.ErrorProviderRuntimeStale, "Machine network runtime owner is stale and must be repaired.", "provider", true)
		err.Detail = state
		return provideripc.RuntimeStopReply{}, err
	}
	cancel := c.heartbeatCancel
	c.heartbeatCancel = nil
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}

	if adapter != nil {
		if err := adapter.Stop(); err != nil {
			structured := api.NewStructuredError(api.ErrorProviderRuntimeStopFailed, err.Error(), "provider", true)
			c.markStale(structured.Message)
			return provideripc.RuntimeStopReply{}, structured
		}
	}
	if err := deleteOwnerRecord(c.stateDir); err != nil {
		c.mu.Lock()
		c.adapter = nil
		if c.owner != nil {
			c.owner.Stale = true
			c.owner.Reason = "Runtime stopped, but provider owner lock could not be deleted: " + err.Error()
			c.owner.RepairActions = []string{api.RepairActionClearMachineNetworkOwner}
		}
		c.mu.Unlock()
		return provideripc.RuntimeStopReply{}, api.NewStructuredError(api.ErrorInternal, err.Error(), "provider", false)
	}

	c.mu.Lock()
	c.adapter = nil
	c.owner = nil
	c.mu.Unlock()
	return provideripc.RuntimeStopReply{}, nil
}

func (c *Controller) RuntimeHeartbeat(_ context.Context, req provideripc.RuntimeHeartbeatRequest) (provideripc.RuntimeHeartbeatReply, *api.StructuredError) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.validateRunningOwnerLocked(req.SessionID, req.RuntimeID); err != nil {
		return provideripc.RuntimeHeartbeatReply{}, err
	}
	c.owner.LastHeartbeatAt = time.Now().UnixMilli()
	if err := saveOwnerRecord(c.stateDir, c.owner); err != nil {
		return provideripc.RuntimeHeartbeatReply{}, api.NewStructuredError(api.ErrorInternal, err.Error(), "provider", false)
	}
	return provideripc.RuntimeHeartbeatReply{OwnerState: *providerOwnerState(c.owner)}, nil
}

func (c *Controller) RuntimeGetStatus(_ context.Context, req provideripc.RuntimeGetStatusRequest) (provideripc.RuntimeGetStatusReply, *api.StructuredError) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.owner != nil {
		if err := c.validateOwnerLocked(req.SessionID, req.RuntimeID); err != nil {
			return provideripc.RuntimeGetStatusReply{}, err
		}
	}
	state := model.EngineStateIdle
	startedAt := int64(0)
	snapshotID := ""
	if c.owner != nil {
		snapshotID = c.owner.SnapshotID
		startedAt = c.owner.StartedAt
		if c.owner.Stale {
			state = model.EngineStateFatal
		} else if c.adapter != nil {
			state = model.EngineStateStarted
		}
	}
	return provideripc.RuntimeGetStatusReply{
		Status: api.EngineStatus{
			State:            state,
			ActiveSnapshotID: snapshotID,
			StartedAt:        startedAt,
		},
		OwnerState: providerOwnerState(c.owner),
	}, nil
}

func (c *Controller) RuntimeGetRuntimeCapabilities(_ context.Context, req provideripc.RuntimeGetRuntimeCapabilitiesRequest) (provideripc.RuntimeGetRuntimeCapabilitiesReply, *api.StructuredError) {
	adapter, err := c.runningAdapterFor(req.SessionID, req.RuntimeID)
	if err != nil {
		return provideripc.RuntimeGetRuntimeCapabilitiesReply{}, err
	}
	return provideripc.RuntimeGetRuntimeCapabilitiesReply{Capabilities: adapter.RuntimeCapabilities()}, nil
}

func (c *Controller) RuntimeGetTraffic(_ context.Context, req provideripc.RuntimeGetTrafficRequest) (provideripc.RuntimeGetTrafficReply, *api.StructuredError) {
	adapter, err := c.runningAdapterFor(req.SessionID, req.RuntimeID)
	if err != nil {
		return provideripc.RuntimeGetTrafficReply{}, err
	}
	snapshot, structured := adapter.TrafficSnapshot()
	return provideripc.RuntimeGetTrafficReply{Snapshot: snapshot}, structured
}

func (c *Controller) RuntimeGetConnections(_ context.Context, req provideripc.RuntimeGetConnectionsRequest) (provideripc.RuntimeGetConnectionsReply, *api.StructuredError) {
	adapter, err := c.runningAdapterFor(req.SessionID, req.RuntimeID)
	if err != nil {
		return provideripc.RuntimeGetConnectionsReply{}, err
	}
	snapshot, structured := adapter.ConnectionSnapshot()
	return provideripc.RuntimeGetConnectionsReply{Snapshot: snapshot}, structured
}

func (c *Controller) RuntimeListGroups(_ context.Context, req provideripc.RuntimeListGroupsRequest) (provideripc.RuntimeListGroupsReply, *api.StructuredError) {
	adapter, err := c.runningAdapterFor(req.SessionID, req.RuntimeID)
	if err != nil {
		return provideripc.RuntimeListGroupsReply{}, err
	}
	groups, structured := adapter.ListGroups()
	return provideripc.RuntimeListGroupsReply{Groups: groups}, structured
}

func (c *Controller) RuntimeSelectOutbound(_ context.Context, req provideripc.RuntimeSelectOutboundRequest) (provideripc.RuntimeSelectOutboundReply, *api.StructuredError) {
	adapter, err := c.runningAdapterFor(req.SessionID, req.RuntimeID)
	if err != nil {
		return provideripc.RuntimeSelectOutboundReply{}, err
	}
	group, structured := adapter.SelectOutbound(req.GroupTag, req.OutboundTag)
	return provideripc.RuntimeSelectOutboundReply{Group: group}, structured
}

func (c *Controller) RuntimeURLTest(ctx context.Context, req provideripc.RuntimeURLTestRequest) (provideripc.RuntimeURLTestReply, *api.StructuredError) {
	adapter, err := c.runningAdapterFor(req.SessionID, req.RuntimeID)
	if err != nil {
		return provideripc.RuntimeURLTestReply{}, err
	}
	timeout := 10 * time.Second
	if req.TimeoutMS > 0 {
		timeout = time.Duration(req.TimeoutMS) * time.Millisecond
	}
	results, structured := adapter.URLTest(ctx, req.GroupTag, timeout)
	return provideripc.RuntimeURLTestReply{Results: results}, structured
}

func (c *Controller) RuntimeCloseConnection(_ context.Context, req provideripc.RuntimeCloseConnectionRequest) (provideripc.RuntimeCloseConnectionReply, *api.StructuredError) {
	adapter, err := c.runningAdapterFor(req.SessionID, req.RuntimeID)
	if err != nil {
		return provideripc.RuntimeCloseConnectionReply{}, err
	}
	return provideripc.RuntimeCloseConnectionReply{}, adapter.CloseConnection(req.ConnectionID)
}

func (c *Controller) RuntimeCloseAllConnections(_ context.Context, req provideripc.RuntimeCloseAllConnectionsRequest) (provideripc.RuntimeCloseAllConnectionsReply, *api.StructuredError) {
	adapter, err := c.runningAdapterFor(req.SessionID, req.RuntimeID)
	if err != nil {
		return provideripc.RuntimeCloseAllConnectionsReply{}, err
	}
	return provideripc.RuntimeCloseAllConnectionsReply{}, adapter.CloseAllConnections()
}

func (c *Controller) RuntimeListenerInfo(_ context.Context, req provideripc.RuntimeListenerInfoRequest) (provideripc.RuntimeListenerInfoReply, *api.StructuredError) {
	adapter, err := c.runningAdapterFor(req.SessionID, req.RuntimeID)
	if err != nil {
		return provideripc.RuntimeListenerInfoReply{}, err
	}
	listeners, structured := adapter.ListenerInfo()
	if structured != nil {
		return provideripc.RuntimeListenerInfoReply{}, structured
	}
	out := make([]provideripc.ListenerInfo, 0, len(listeners))
	for _, listener := range listeners {
		out = append(out, provideripc.ListenerInfo{
			Tag:     listener.Tag,
			Type:    listener.Type,
			Address: listener.Address,
			Port:    listener.Port,
		})
	}
	return provideripc.RuntimeListenerInfoReply{Listeners: out}, nil
}

func (c *Controller) RuntimeSubscribeEvents(ctx context.Context, req provideripc.RuntimeSubscribeEventsRequest) (<-chan api.RuntimeEvent, *api.StructuredError) {
	c.mu.Lock()
	err := c.validateRunningOwnerLocked(req.SessionID, req.RuntimeID)
	c.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return c.events.SubscribeRuntimeEvents(ctx), nil
}

func (c *Controller) RunRepairAction(_ context.Context, req api.RunRepairActionRequest) (api.RunRepairActionReply, *api.StructuredError) {
	if req.Action != api.RepairActionClearMachineNetworkOwner {
		return api.RunRepairActionReply{}, api.NewStructuredError(api.ErrorPlatformRepairActionNotFound, "Repair action is not allowlisted.", "provider", true)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.adapter != nil {
		return api.RunRepairActionReply{}, api.NewStructuredError(api.ErrorPlatformRepairFailed, "Live machine network runtime cannot be cleared as stale state.", "provider", true)
	}
	if c.owner == nil {
		return api.RunRepairActionReply{Action: req.Action, Outcome: "noop", Reason: "No stale owner state exists."}, nil
	}
	if !c.owner.Stale {
		return api.RunRepairActionReply{}, api.NewStructuredError(api.ErrorPlatformRepairFailed, "Machine network owner state is not marked stale.", "provider", true)
	}
	if err := deleteOwnerRecord(c.stateDir); err != nil {
		return api.RunRepairActionReply{}, api.NewStructuredError(api.ErrorPlatformRepairFailed, err.Error(), "provider", true)
	}
	c.owner = nil
	return api.RunRepairActionReply{Action: req.Action, Outcome: "success"}, nil
}

func (c *Controller) runningAdapterFor(sessionID, runtimeID string) (*singboxadapter.Adapter, *api.StructuredError) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.validateRunningOwnerLocked(sessionID, runtimeID); err != nil {
		return nil, err
	}
	return c.adapter, nil
}

func (c *Controller) validateRunningOwnerLocked(sessionID, runtimeID string) *api.StructuredError {
	if err := c.validateOwnerLocked(sessionID, runtimeID); err != nil {
		return err
	}
	if c.adapter == nil {
		return api.NewStructuredError(api.ErrorProviderRuntimeUnavailable, "Provider runtime is not running.", "provider", true)
	}
	return nil
}

func (c *Controller) validateOwnerLocked(sessionID, runtimeID string) *api.StructuredError {
	if sessionID == "" || runtimeID == "" {
		return api.NewStructuredError(api.ErrorIPCInvalidRequest, "session_id and runtime_id are required.", "provider", true)
	}
	if c.owner == nil {
		return api.NewStructuredError(api.ErrorProviderRuntimeUnavailable, "Provider runtime is not running.", "provider", true)
	}
	if c.owner.Stale {
		err := api.NewStructuredError(api.ErrorProviderRuntimeStale, "Machine network runtime owner is stale and must be repaired.", "provider", true)
		err.Detail = providerOwnerState(c.owner)
		return err
	}
	if c.owner.SessionID != sessionID || c.owner.RuntimeID != runtimeID {
		err := api.NewStructuredError(api.ErrorNetworkModeOwnedByAnotherSession, "Machine network mode is owned by another qkbox session.", "provider", true)
		err.Detail = providerOwnerState(c.owner)
		return err
	}
	return nil
}

func (c *Controller) startHeartbeatMonitor(timeout time.Duration) {
	c.mu.Lock()
	if c.heartbeatCancel != nil {
		c.heartbeatCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.heartbeatCancel = cancel
	c.mu.Unlock()

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.mu.Lock()
				record := c.owner
				lastHeartbeat := int64(0)
				if record != nil {
					lastHeartbeat = record.LastHeartbeatAt
				}
				c.mu.Unlock()
				if record == nil || lastHeartbeat == 0 {
					continue
				}
				if time.Since(time.UnixMilli(lastHeartbeat)) <= timeout {
					continue
				}
				_, structured := c.RuntimeStop(context.Background(), provideripc.RuntimeStopRequest{
					SessionID: record.SessionID,
					RuntimeID: record.RuntimeID,
				})
				if structured != nil {
					c.markStale("Heartbeat timed out and runtime stop failed: " + structured.Message)
				}
				return
			}
		}
	}()
}

func (c *Controller) markStale(reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.owner == nil {
		return
	}
	c.owner.Stale = true
	c.owner.Reason = reason
	c.owner.RepairActions = []string{api.RepairActionClearMachineNetworkOwner}
	_ = saveOwnerRecord(c.stateDir, c.owner)
	c.events.PublishBridgeError(api.NewStructuredError(api.ErrorProviderRuntimeStale, reason, "provider", true))
}

func timeoutFromRequest(timeoutMS int64) time.Duration {
	if timeoutMS <= 0 {
		return defaultHeartbeatTimeout
	}
	return time.Duration(timeoutMS) * time.Millisecond
}
