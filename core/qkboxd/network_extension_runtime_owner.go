package qkboxd

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/zclkkk/qkbox/internal/runtimeapi"
	"github.com/zclkkk/qkbox/platform/capability"
	"github.com/zclkkk/qkbox/shared/api"
)

type networkExtensionRuntimeOwner struct {
	extension capability.NetworkExtensionRuntime
	events    *RuntimeEventHub
	sessionID string

	mu        sync.Mutex
	runtimeID string
	started   bool
	cancel    context.CancelFunc
}

func newNetworkExtensionRuntimeOwner(extension capability.NetworkExtensionRuntime, events *RuntimeEventHub, sessionID string) RuntimeOwner {
	return &networkExtensionRuntimeOwner{
		extension: extension,
		events:    events,
		sessionID: sessionID,
	}
}

func (o *networkExtensionRuntimeOwner) Start(ctx context.Context, target RuntimeStartTarget) *api.StructuredError {
	if o.extension == nil {
		return api.NewStructuredError(api.ErrorNetworkExtensionUnavailable, "NetworkExtension runtime is not configured.", "network_extension", true)
	}
	runtimeID, err := randomRuntimeID()
	if err != nil {
		return qkboxdInternalError(err)
	}
	_, structured := o.extension.Start(ctx, capability.NetworkExtensionStartRequest{
		SessionID:            o.sessionID,
		RuntimeID:            runtimeID,
		SnapshotID:           target.SnapshotID,
		Mode:                 api.RuntimeModeAppleNetworkExtension,
		ConfigJSON:           target.ConfigJSON,
		RequiredCapabilities: append([]string(nil), target.RequiredCapabilities...),
		HeartbeatTimeout:     providerHeartbeatTimeout,
	})
	if structured != nil {
		return structured
	}

	ownerCtx, cancel := context.WithCancel(ctx)
	o.mu.Lock()
	o.runtimeID = runtimeID
	o.started = true
	o.cancel = cancel
	o.mu.Unlock()

	if structured := o.startEventBridge(ownerCtx); structured != nil {
		cancel()
		_, cleanup := o.extension.Stop(context.Background(), capability.NetworkExtensionStopRequest{
			SessionID: o.sessionID,
			RuntimeID: runtimeID,
		})
		o.clearStarted()
		if cleanup != nil {
			cleanup.Detail = structured
			return cleanup
		}
		return structured
	}
	o.startHeartbeat(ownerCtx)
	return nil
}

func (o *networkExtensionRuntimeOwner) Stop() *api.StructuredError {
	o.mu.Lock()
	runtimeID := o.runtimeID
	cancel := o.cancel
	started := o.started
	o.cancel = nil
	o.started = false
	o.runtimeID = ""
	o.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if !started {
		return nil
	}
	_, structured := o.extension.Stop(context.Background(), capability.NetworkExtensionStopRequest{
		SessionID: o.sessionID,
		RuntimeID: runtimeID,
	})
	return structured
}

func (o *networkExtensionRuntimeOwner) RuntimeCapabilities() []api.Capability {
	if err := o.requireStarted(); err != nil {
		return api.RuntimeCapabilityShell()
	}
	caps, structured := o.extension.RuntimeCapabilities(context.Background(), o.runtimeRequest())
	if structured != nil {
		return api.RuntimeCapabilityShell()
	}
	return caps
}

func (o *networkExtensionRuntimeOwner) TrafficSnapshot() (api.TrafficSnapshot, *api.StructuredError) {
	if err := o.requireStarted(); err != nil {
		return api.TrafficSnapshot{}, err
	}
	return o.extension.TrafficSnapshot(context.Background(), o.runtimeRequest())
}

func (o *networkExtensionRuntimeOwner) ConnectionSnapshot() (api.ConnectionSnapshot, *api.StructuredError) {
	if err := o.requireStarted(); err != nil {
		return api.ConnectionSnapshot{}, err
	}
	return o.extension.ConnectionSnapshot(context.Background(), o.runtimeRequest())
}

func (o *networkExtensionRuntimeOwner) ListGroups() ([]api.OutboundGroup, *api.StructuredError) {
	if err := o.requireStarted(); err != nil {
		return nil, err
	}
	return o.extension.ListGroups(context.Background(), o.runtimeRequest())
}

func (o *networkExtensionRuntimeOwner) SelectOutbound(groupTag, outboundTag string) (api.OutboundGroup, *api.StructuredError) {
	if err := o.requireStarted(); err != nil {
		return api.OutboundGroup{}, err
	}
	return o.extension.SelectOutbound(context.Background(), capability.NetworkExtensionSelectOutboundRequest{
		SessionID:   o.sessionID,
		RuntimeID:   o.currentRuntimeID(),
		GroupTag:    groupTag,
		OutboundTag: outboundTag,
	})
}

func (o *networkExtensionRuntimeOwner) URLTest(ctx context.Context, groupTag string, timeout time.Duration) ([]api.URLTestResult, *api.StructuredError) {
	if err := o.requireStarted(); err != nil {
		return nil, err
	}
	return o.extension.URLTest(ctx, capability.NetworkExtensionURLTestRequest{
		SessionID: o.sessionID,
		RuntimeID: o.currentRuntimeID(),
		GroupTag:  groupTag,
		Timeout:   timeout,
	})
}

func (o *networkExtensionRuntimeOwner) CloseConnection(id string) *api.StructuredError {
	if err := o.requireStarted(); err != nil {
		return err
	}
	return o.extension.CloseConnection(context.Background(), capability.NetworkExtensionCloseConnectionRequest{
		SessionID:    o.sessionID,
		RuntimeID:    o.currentRuntimeID(),
		ConnectionID: id,
	})
}

func (o *networkExtensionRuntimeOwner) CloseAllConnections() *api.StructuredError {
	if err := o.requireStarted(); err != nil {
		return err
	}
	return o.extension.CloseAllConnections(context.Background(), o.runtimeRequest())
}

func (o *networkExtensionRuntimeOwner) ListenerInfo() ([]runtimeapi.ListenerInfo, *api.StructuredError) {
	if err := o.requireStarted(); err != nil {
		return nil, err
	}
	return o.extension.ListenerInfo(context.Background(), o.runtimeRequest())
}

func (o *networkExtensionRuntimeOwner) startHeartbeat(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(providerHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, structured := o.extension.Heartbeat(ctx, capability.NetworkExtensionHeartbeatRequest{
					SessionID: o.sessionID,
					RuntimeID: o.currentRuntimeID(),
				})
				if structured != nil {
					o.publishExtensionError(structured)
					return
				}
			}
		}
	}()
}

func (o *networkExtensionRuntimeOwner) startEventBridge(ctx context.Context) *api.StructuredError {
	events, structured := o.extension.SubscribeEvents(ctx, o.runtimeRequest())
	if structured != nil {
		return structured
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-events:
				if !ok {
					if ctx.Err() != nil {
						return
					}
					o.publishExtensionError(api.NewStructuredError(api.ErrorNetworkExtensionEventStreamFailed, "NetworkExtension runtime event stream closed.", "network_extension", true))
					return
				}
				o.forwardEvent(event)
			}
		}
	}()
	return nil
}

func (o *networkExtensionRuntimeOwner) forwardEvent(event api.RuntimeEvent) {
	if event.Error != nil {
		o.publishExtensionError(event.Error)
		return
	}
	switch event.Event {
	case api.EventEngineLog:
		var entry api.RuntimeLogEntry
		data, err := json.Marshal(event.Data)
		if err != nil {
			o.publishExtensionError(api.NewStructuredError(api.ErrorNetworkExtensionEventStreamFailed, err.Error(), "network_extension", true))
			return
		}
		if err := json.Unmarshal(data, &entry); err != nil {
			o.publishExtensionError(api.NewStructuredError(api.ErrorNetworkExtensionEventStreamFailed, err.Error(), "network_extension", true))
			return
		}
		if entry.Source == "" {
			entry.Source = "network_extension"
		}
		o.events.PublishRuntimeLog(entry.Source, entry.Level, entry.Message)
	case api.EventEngineEventBridgeError:
		o.publishExtensionError(api.NewStructuredError(api.ErrorNetworkExtensionEventStreamFailed, "NetworkExtension runtime event bridge reported an error.", "network_extension", true))
	}
}

func (o *networkExtensionRuntimeOwner) publishExtensionError(err *api.StructuredError) {
	if o.events != nil {
		o.events.PublishBridgeError(err)
	}
}

func (o *networkExtensionRuntimeOwner) requireStarted() *api.StructuredError {
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.started || o.runtimeID == "" {
		return api.NewStructuredError(api.ErrorEngineNotStarted, "NetworkExtension runtime is not running.", "qkboxd", true)
	}
	return nil
}

func (o *networkExtensionRuntimeOwner) runtimeRequest() capability.NetworkExtensionRuntimeRequest {
	return capability.NetworkExtensionRuntimeRequest{SessionID: o.sessionID, RuntimeID: o.currentRuntimeID()}
}

func (o *networkExtensionRuntimeOwner) currentRuntimeID() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.runtimeID
}

func (o *networkExtensionRuntimeOwner) clearStarted() {
	o.mu.Lock()
	o.runtimeID = ""
	o.started = false
	o.cancel = nil
	o.mu.Unlock()
}

func supportsAppleNetworkExtensionRuntime() bool {
	return runtimeGOOS == "darwin"
}
