package provideripc

import (
	"encoding/json"

	"github.com/zclkkk/qkbox/shared/api"
)

const (
	MethodAuth                          = "provider.auth"
	MethodGetStatus                     = "provider.getStatus"
	MethodPrepareFeature                = "provider.prepareFeature"
	MethodRunRepairAction               = "provider.runRepairAction"
	MethodRuntimeStart                  = "provider.runtime.start"
	MethodRuntimeStop                   = "provider.runtime.stop"
	MethodRuntimeHeartbeat              = "provider.runtime.heartbeat"
	MethodRuntimeGetStatus              = "provider.runtime.getStatus"
	MethodRuntimeGetRuntimeCapabilities = "provider.runtime.getRuntimeCapabilities"
	MethodRuntimeGetTraffic             = "provider.runtime.getTraffic"
	MethodRuntimeGetConnections         = "provider.runtime.getConnections"
	MethodRuntimeListGroups             = "provider.runtime.listGroups"
	MethodRuntimeSelectOutbound         = "provider.runtime.selectOutbound"
	MethodRuntimeURLTest                = "provider.runtime.urlTest"
	MethodRuntimeCloseConnection        = "provider.runtime.closeConnection"
	MethodRuntimeCloseAllConnections    = "provider.runtime.closeAllConnections"
	MethodRuntimeListenerInfo           = "provider.runtime.listenerInfo"
	MethodRuntimeSubscribeEvents        = "provider.runtime.subscribeEvents"
	DefaultProviderVersion              = api.QKBoxDVersion
)

type Request struct {
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	ID     string               `json:"id"`
	Result json.RawMessage      `json:"result,omitempty"`
	Error  *api.StructuredError `json:"error,omitempty"`
}

type EventFrame struct {
	ID    string               `json:"id,omitempty"`
	Event string               `json:"event"`
	Data  json.RawMessage      `json:"data,omitempty"`
	Error *api.StructuredError `json:"error,omitempty"`
}

type AuthRequest struct {
	Token string `json:"token"`
}

type AuthReply struct{}

type StatusReply struct {
	Version      string                  `json:"version"`
	OwnerState   *api.ProviderOwnerState `json:"owner_state,omitempty"`
	Capabilities []api.Capability        `json:"capabilities,omitempty"`
}

type RuntimeStartRequest struct {
	SessionID            string   `json:"session_id"`
	RuntimeID            string   `json:"runtime_id"`
	SnapshotID           string   `json:"snapshot_id"`
	Mode                 string   `json:"mode"`
	ConfigJSON           string   `json:"config_json"`
	RequiredCapabilities []string `json:"required_capabilities,omitempty"`
	HeartbeatTimeoutMS   int64    `json:"heartbeat_timeout_ms,omitempty"`
}

type RuntimeStartReply struct {
	OwnerState api.ProviderOwnerState `json:"owner_state"`
}

type RuntimeStopRequest struct {
	SessionID string `json:"session_id"`
	RuntimeID string `json:"runtime_id"`
}

type RuntimeStopReply struct{}

type RuntimeHeartbeatRequest struct {
	SessionID string `json:"session_id"`
	RuntimeID string `json:"runtime_id"`
}

type RuntimeHeartbeatReply struct {
	OwnerState api.ProviderOwnerState `json:"owner_state"`
}

type RuntimeGetStatusRequest struct {
	SessionID string `json:"session_id"`
	RuntimeID string `json:"runtime_id"`
}

type RuntimeGetStatusReply struct {
	Status     api.EngineStatus        `json:"status"`
	OwnerState *api.ProviderOwnerState `json:"owner_state,omitempty"`
}

type RuntimeGetRuntimeCapabilitiesRequest struct {
	SessionID string `json:"session_id"`
	RuntimeID string `json:"runtime_id"`
}

type RuntimeGetRuntimeCapabilitiesReply struct {
	Capabilities []api.Capability `json:"capabilities"`
}

type RuntimeGetTrafficRequest struct {
	SessionID string `json:"session_id"`
	RuntimeID string `json:"runtime_id"`
}

type RuntimeGetTrafficReply struct {
	Snapshot api.TrafficSnapshot `json:"snapshot"`
}

type RuntimeGetConnectionsRequest struct {
	SessionID string `json:"session_id"`
	RuntimeID string `json:"runtime_id"`
}

type RuntimeGetConnectionsReply struct {
	Snapshot api.ConnectionSnapshot `json:"snapshot"`
}

type RuntimeListGroupsRequest struct {
	SessionID string `json:"session_id"`
	RuntimeID string `json:"runtime_id"`
}

type RuntimeListGroupsReply struct {
	Groups []api.OutboundGroup `json:"groups"`
}

type RuntimeSelectOutboundRequest struct {
	SessionID   string `json:"session_id"`
	RuntimeID   string `json:"runtime_id"`
	GroupTag    string `json:"group_tag"`
	OutboundTag string `json:"outbound_tag"`
}

type RuntimeSelectOutboundReply struct {
	Group api.OutboundGroup `json:"group"`
}

type RuntimeURLTestRequest struct {
	SessionID string `json:"session_id"`
	RuntimeID string `json:"runtime_id"`
	GroupTag  string `json:"group_tag"`
	TimeoutMS int64  `json:"timeout_ms,omitempty"`
}

type RuntimeURLTestReply struct {
	Results []api.URLTestResult `json:"results"`
}

type RuntimeCloseConnectionRequest struct {
	SessionID    string `json:"session_id"`
	RuntimeID    string `json:"runtime_id"`
	ConnectionID string `json:"connection_id"`
}

type RuntimeCloseConnectionReply struct{}

type RuntimeCloseAllConnectionsRequest struct {
	SessionID string `json:"session_id"`
	RuntimeID string `json:"runtime_id"`
}

type RuntimeCloseAllConnectionsReply struct{}

type RuntimeListenerInfoRequest struct {
	SessionID string `json:"session_id"`
	RuntimeID string `json:"runtime_id"`
}

type RuntimeListenerInfoReply struct {
	Listeners []ListenerInfo `json:"listeners"`
}

type ListenerInfo struct {
	Tag     string `json:"tag"`
	Type    string `json:"type"`
	Address string `json:"address"`
	Port    int    `json:"port"`
}

type RuntimeSubscribeEventsRequest struct {
	SessionID string `json:"session_id"`
	RuntimeID string `json:"runtime_id"`
}

type RuntimeSubscribeEventsReply struct{}

var MethodRegistry = map[string]struct{}{
	MethodAuth:                          {},
	MethodGetStatus:                     {},
	MethodPrepareFeature:                {},
	MethodRunRepairAction:               {},
	MethodRuntimeStart:                  {},
	MethodRuntimeStop:                   {},
	MethodRuntimeHeartbeat:              {},
	MethodRuntimeGetStatus:              {},
	MethodRuntimeGetRuntimeCapabilities: {},
	MethodRuntimeGetTraffic:             {},
	MethodRuntimeGetConnections:         {},
	MethodRuntimeListGroups:             {},
	MethodRuntimeSelectOutbound:         {},
	MethodRuntimeURLTest:                {},
	MethodRuntimeCloseConnection:        {},
	MethodRuntimeCloseAllConnections:    {},
	MethodRuntimeListenerInfo:           {},
	MethodRuntimeSubscribeEvents:        {},
}
