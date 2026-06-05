package qkboxd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	"github.com/zclkkk/qkbox/internal/provideripc"
	"github.com/zclkkk/qkbox/internal/runtimeapi"
	"github.com/zclkkk/qkbox/platform/capability"
	"github.com/zclkkk/qkbox/shared/api"
)

const (
	providerHeartbeatInterval = 2 * time.Second
	providerHeartbeatTimeout  = 8 * time.Second
)

type providerRuntimeOwner struct {
	provider  capability.PrivilegedProvider
	events    *RuntimeEventHub
	sessionID string

	mu        sync.Mutex
	runtimeID string
	started   bool
	cancel    context.CancelFunc
}

func newProviderRuntimeOwner(provider capability.PrivilegedProvider, events *RuntimeEventHub, sessionID string) RuntimeOwner {
	return &providerRuntimeOwner{
		provider:  provider,
		events:    events,
		sessionID: sessionID,
	}
}

func (o *providerRuntimeOwner) Start(ctx context.Context, target RuntimeStartTarget) *api.StructuredError {
	if o.provider == nil {
		return api.NewStructuredError(api.ErrorPlatformProviderUnavailable, "Privileged provider is not configured.", "provider", true)
	}
	runtimeID, err := randomRuntimeID()
	if err != nil {
		return qkboxdInternalError(err)
	}
	_, structured := o.provider.RuntimeStart(ctx, provideripc.RuntimeStartRequest{
		SessionID:            o.sessionID,
		RuntimeID:            runtimeID,
		SnapshotID:           target.SnapshotID,
		Mode:                 api.RuntimeModeMachineNetwork,
		ConfigJSON:           target.ConfigJSON,
		RequiredCapabilities: append([]string(nil), target.RequiredCapabilities...),
		HeartbeatTimeoutMS:   providerHeartbeatTimeout.Milliseconds(),
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
		_, cleanup := o.provider.RuntimeStop(context.Background(), provideripc.RuntimeStopRequest{
			SessionID: o.sessionID,
			RuntimeID: runtimeID,
		})
		o.mu.Lock()
		o.runtimeID = ""
		o.started = false
		o.cancel = nil
		o.mu.Unlock()
		if cleanup != nil {
			cleanup.Detail = structured
			return cleanup
		}
		return structured
	}
	o.startHeartbeat(ownerCtx)
	return nil
}

func (o *providerRuntimeOwner) Stop() *api.StructuredError {
	o.mu.Lock()
	runtimeID := o.runtimeID
	cancel := o.cancel
	started := o.started
	o.cancel = nil
	o.started = false
	o.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if !started {
		return nil
	}
	_, structured := o.provider.RuntimeStop(context.Background(), provideripc.RuntimeStopRequest{
		SessionID: o.sessionID,
		RuntimeID: runtimeID,
	})
	return structured
}

func (o *providerRuntimeOwner) RuntimeCapabilities() []api.Capability {
	if err := o.requireStarted(); err != nil {
		return api.RuntimeCapabilityShell()
	}
	reply, structured := o.provider.RuntimeGetRuntimeCapabilities(context.Background(), provideripc.RuntimeGetRuntimeCapabilitiesRequest{
		SessionID: o.sessionID,
		RuntimeID: o.currentRuntimeID(),
	})
	if structured != nil {
		return api.RuntimeCapabilityShell()
	}
	return reply.Capabilities
}

func (o *providerRuntimeOwner) TrafficSnapshot() (api.TrafficSnapshot, *api.StructuredError) {
	if err := o.requireStarted(); err != nil {
		return api.TrafficSnapshot{}, err
	}
	reply, structured := o.provider.RuntimeGetTraffic(context.Background(), provideripc.RuntimeGetTrafficRequest{
		SessionID: o.sessionID,
		RuntimeID: o.currentRuntimeID(),
	})
	return reply.Snapshot, structured
}

func (o *providerRuntimeOwner) ConnectionSnapshot() (api.ConnectionSnapshot, *api.StructuredError) {
	if err := o.requireStarted(); err != nil {
		return api.ConnectionSnapshot{}, err
	}
	reply, structured := o.provider.RuntimeGetConnections(context.Background(), provideripc.RuntimeGetConnectionsRequest{
		SessionID: o.sessionID,
		RuntimeID: o.currentRuntimeID(),
	})
	return reply.Snapshot, structured
}

func (o *providerRuntimeOwner) ListGroups() ([]api.OutboundGroup, *api.StructuredError) {
	if err := o.requireStarted(); err != nil {
		return nil, err
	}
	reply, structured := o.provider.RuntimeListGroups(context.Background(), provideripc.RuntimeListGroupsRequest{
		SessionID: o.sessionID,
		RuntimeID: o.currentRuntimeID(),
	})
	return reply.Groups, structured
}

func (o *providerRuntimeOwner) SelectOutbound(groupTag, outboundTag string) (api.OutboundGroup, *api.StructuredError) {
	if err := o.requireStarted(); err != nil {
		return api.OutboundGroup{}, err
	}
	reply, structured := o.provider.RuntimeSelectOutbound(context.Background(), provideripc.RuntimeSelectOutboundRequest{
		SessionID:   o.sessionID,
		RuntimeID:   o.currentRuntimeID(),
		GroupTag:    groupTag,
		OutboundTag: outboundTag,
	})
	return reply.Group, structured
}

func (o *providerRuntimeOwner) URLTest(ctx context.Context, groupTag string, timeout time.Duration) ([]api.URLTestResult, *api.StructuredError) {
	if err := o.requireStarted(); err != nil {
		return nil, err
	}
	reply, structured := o.provider.RuntimeURLTest(ctx, provideripc.RuntimeURLTestRequest{
		SessionID: o.sessionID,
		RuntimeID: o.currentRuntimeID(),
		GroupTag:  groupTag,
		TimeoutMS: timeout.Milliseconds(),
	})
	return reply.Results, structured
}

func (o *providerRuntimeOwner) CloseConnection(id string) *api.StructuredError {
	if err := o.requireStarted(); err != nil {
		return err
	}
	_, structured := o.provider.RuntimeCloseConnection(context.Background(), provideripc.RuntimeCloseConnectionRequest{
		SessionID:    o.sessionID,
		RuntimeID:    o.currentRuntimeID(),
		ConnectionID: id,
	})
	return structured
}

func (o *providerRuntimeOwner) CloseAllConnections() *api.StructuredError {
	if err := o.requireStarted(); err != nil {
		return err
	}
	_, structured := o.provider.RuntimeCloseAllConnections(context.Background(), provideripc.RuntimeCloseAllConnectionsRequest{
		SessionID: o.sessionID,
		RuntimeID: o.currentRuntimeID(),
	})
	return structured
}

func (o *providerRuntimeOwner) ListenerInfo() ([]runtimeapi.ListenerInfo, *api.StructuredError) {
	if err := o.requireStarted(); err != nil {
		return nil, err
	}
	reply, structured := o.provider.RuntimeListenerInfo(context.Background(), provideripc.RuntimeListenerInfoRequest{
		SessionID: o.sessionID,
		RuntimeID: o.currentRuntimeID(),
	})
	if structured != nil {
		return nil, structured
	}
	listeners := make([]runtimeapi.ListenerInfo, 0, len(reply.Listeners))
	for _, listener := range reply.Listeners {
		listeners = append(listeners, runtimeapi.ListenerInfo{
			Tag:     listener.Tag,
			Type:    listener.Type,
			Address: listener.Address,
			Port:    listener.Port,
		})
	}
	return listeners, nil
}

func (o *providerRuntimeOwner) startHeartbeat(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(providerHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, structured := o.provider.RuntimeHeartbeat(ctx, provideripc.RuntimeHeartbeatRequest{
					SessionID: o.sessionID,
					RuntimeID: o.currentRuntimeID(),
				})
				if structured != nil {
					o.publishProviderError(structured)
					return
				}
			}
		}
	}()
}

func (o *providerRuntimeOwner) startEventBridge(ctx context.Context) *api.StructuredError {
	events, structured := o.provider.RuntimeSubscribeEvents(ctx, provideripc.RuntimeSubscribeEventsRequest{
		SessionID: o.sessionID,
		RuntimeID: o.currentRuntimeID(),
	})
	if structured != nil {
		return structured
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case frame, ok := <-events:
				if !ok {
					if ctx.Err() != nil {
						return
					}
					o.publishProviderError(api.NewStructuredError(api.ErrorProviderRuntimeEventStreamFailed, "Provider runtime event stream closed.", "provider", true))
					return
				}
				o.forwardEventFrame(frame)
			}
		}
	}()
	return nil
}

func (o *providerRuntimeOwner) forwardEventFrame(frame provideripc.EventFrame) {
	if frame.Error != nil {
		o.publishProviderError(frame.Error)
		return
	}
	switch frame.Event {
	case api.EventEngineLog:
		var entry api.RuntimeLogEntry
		if err := json.Unmarshal(frame.Data, &entry); err != nil {
			o.publishProviderError(api.NewStructuredError(api.ErrorProviderRuntimeEventStreamFailed, err.Error(), "provider", true))
			return
		}
		if entry.Source == "" {
			entry.Source = "provider"
		}
		o.events.PublishRuntimeLog(entry.Source, entry.Level, entry.Message)
	case api.EventEngineEventBridgeError:
		o.publishProviderError(api.NewStructuredError(api.ErrorProviderRuntimeEventStreamFailed, "Provider runtime event bridge reported an error.", "provider", true))
	}
}

func (o *providerRuntimeOwner) publishProviderError(err *api.StructuredError) {
	if o.events != nil {
		o.events.PublishBridgeError(err)
	}
}

func (o *providerRuntimeOwner) requireStarted() *api.StructuredError {
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.started || o.runtimeID == "" {
		return api.NewStructuredError(api.ErrorEngineNotStarted, "Provider runtime is not running.", "qkboxd", true)
	}
	return nil
}

func (o *providerRuntimeOwner) currentRuntimeID() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.runtimeID
}

func randomRuntimeID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func newRuntimeSessionID() string {
	id, err := randomRuntimeID()
	if err == nil {
		return id
	}
	return time.Now().UTC().Format("20060102150405.000000000")
}

func supportsProviderHostedMachineRuntime() bool {
	return runtimeGOOS == "windows"
}
