package capability

import (
	"context"
	"time"

	"github.com/zclkkk/qkbox/internal/runtimeapi"
	"github.com/zclkkk/qkbox/shared/api"
)

type NetworkExtensionRuntime interface {
	Status(ctx context.Context) api.NetworkExtensionStatus
	Start(ctx context.Context, req NetworkExtensionStartRequest) (NetworkExtensionStartReply, *api.StructuredError)
	Stop(ctx context.Context, req NetworkExtensionStopRequest) (NetworkExtensionStopReply, *api.StructuredError)
	Heartbeat(ctx context.Context, req NetworkExtensionHeartbeatRequest) (NetworkExtensionHeartbeatReply, *api.StructuredError)
	RuntimeCapabilities(ctx context.Context, req NetworkExtensionRuntimeRequest) ([]api.Capability, *api.StructuredError)
	TrafficSnapshot(ctx context.Context, req NetworkExtensionRuntimeRequest) (api.TrafficSnapshot, *api.StructuredError)
	ConnectionSnapshot(ctx context.Context, req NetworkExtensionRuntimeRequest) (api.ConnectionSnapshot, *api.StructuredError)
	ListGroups(ctx context.Context, req NetworkExtensionRuntimeRequest) ([]api.OutboundGroup, *api.StructuredError)
	SelectOutbound(ctx context.Context, req NetworkExtensionSelectOutboundRequest) (api.OutboundGroup, *api.StructuredError)
	URLTest(ctx context.Context, req NetworkExtensionURLTestRequest) ([]api.URLTestResult, *api.StructuredError)
	CloseConnection(ctx context.Context, req NetworkExtensionCloseConnectionRequest) *api.StructuredError
	CloseAllConnections(ctx context.Context, req NetworkExtensionRuntimeRequest) *api.StructuredError
	ListenerInfo(ctx context.Context, req NetworkExtensionRuntimeRequest) ([]runtimeapi.ListenerInfo, *api.StructuredError)
	SubscribeEvents(ctx context.Context, req NetworkExtensionRuntimeRequest) (<-chan api.RuntimeEvent, *api.StructuredError)
}

type NetworkExtensionStartRequest struct {
	SessionID            string
	RuntimeID            string
	SnapshotID           string
	Mode                 string
	ConfigJSON           string
	RequiredCapabilities []string
	HeartbeatTimeout     time.Duration
}

type NetworkExtensionStartReply struct {
	OwnerState api.ProviderOwnerState
}

type NetworkExtensionStopRequest struct {
	SessionID string
	RuntimeID string
}

type NetworkExtensionStopReply struct{}

type NetworkExtensionHeartbeatRequest struct {
	SessionID string
	RuntimeID string
}

type NetworkExtensionHeartbeatReply struct {
	OwnerState api.ProviderOwnerState
}

type NetworkExtensionRuntimeRequest struct {
	SessionID string
	RuntimeID string
}

type NetworkExtensionSelectOutboundRequest struct {
	SessionID   string
	RuntimeID   string
	GroupTag    string
	OutboundTag string
}

type NetworkExtensionURLTestRequest struct {
	SessionID string
	RuntimeID string
	GroupTag  string
	Timeout   time.Duration
}

type NetworkExtensionCloseConnectionRequest struct {
	SessionID    string
	RuntimeID    string
	ConnectionID string
}

type unavailableNetworkExtensionRuntime string

func (r unavailableNetworkExtensionRuntime) Status(context.Context) api.NetworkExtensionStatus {
	reason := string(r)
	return api.NetworkExtensionStatus{
		Reason: reason,
		Capabilities: []api.Capability{
			{Name: api.CapabilityTunMode, State: api.CapabilityUnsupported, Reason: reason},
			{Name: api.CapabilityDNSHijack, State: api.CapabilityUnsupported, Reason: reason},
			{Name: api.CapabilityConnectionTracking, State: api.CapabilityUnsupported, Reason: reason},
		},
	}
}

func (r unavailableNetworkExtensionRuntime) Start(context.Context, NetworkExtensionStartRequest) (NetworkExtensionStartReply, *api.StructuredError) {
	return NetworkExtensionStartReply{}, r.err()
}

func (r unavailableNetworkExtensionRuntime) Stop(context.Context, NetworkExtensionStopRequest) (NetworkExtensionStopReply, *api.StructuredError) {
	return NetworkExtensionStopReply{}, nil
}

func (r unavailableNetworkExtensionRuntime) Heartbeat(context.Context, NetworkExtensionHeartbeatRequest) (NetworkExtensionHeartbeatReply, *api.StructuredError) {
	return NetworkExtensionHeartbeatReply{}, r.err()
}

func (r unavailableNetworkExtensionRuntime) RuntimeCapabilities(context.Context, NetworkExtensionRuntimeRequest) ([]api.Capability, *api.StructuredError) {
	return api.RuntimeCapabilityShell(), r.err()
}

func (r unavailableNetworkExtensionRuntime) TrafficSnapshot(context.Context, NetworkExtensionRuntimeRequest) (api.TrafficSnapshot, *api.StructuredError) {
	return api.TrafficSnapshot{}, r.err()
}

func (r unavailableNetworkExtensionRuntime) ConnectionSnapshot(context.Context, NetworkExtensionRuntimeRequest) (api.ConnectionSnapshot, *api.StructuredError) {
	return api.ConnectionSnapshot{}, r.err()
}

func (r unavailableNetworkExtensionRuntime) ListGroups(context.Context, NetworkExtensionRuntimeRequest) ([]api.OutboundGroup, *api.StructuredError) {
	return nil, r.err()
}

func (r unavailableNetworkExtensionRuntime) SelectOutbound(context.Context, NetworkExtensionSelectOutboundRequest) (api.OutboundGroup, *api.StructuredError) {
	return api.OutboundGroup{}, r.err()
}

func (r unavailableNetworkExtensionRuntime) URLTest(context.Context, NetworkExtensionURLTestRequest) ([]api.URLTestResult, *api.StructuredError) {
	return nil, r.err()
}

func (r unavailableNetworkExtensionRuntime) CloseConnection(context.Context, NetworkExtensionCloseConnectionRequest) *api.StructuredError {
	return r.err()
}

func (r unavailableNetworkExtensionRuntime) CloseAllConnections(context.Context, NetworkExtensionRuntimeRequest) *api.StructuredError {
	return r.err()
}

func (r unavailableNetworkExtensionRuntime) ListenerInfo(context.Context, NetworkExtensionRuntimeRequest) ([]runtimeapi.ListenerInfo, *api.StructuredError) {
	return nil, r.err()
}

func (r unavailableNetworkExtensionRuntime) SubscribeEvents(context.Context, NetworkExtensionRuntimeRequest) (<-chan api.RuntimeEvent, *api.StructuredError) {
	return nil, r.err()
}

func (r unavailableNetworkExtensionRuntime) err() *api.StructuredError {
	return api.NewStructuredError(api.ErrorNetworkExtensionUnavailable, string(r), "network_extension", true)
}
