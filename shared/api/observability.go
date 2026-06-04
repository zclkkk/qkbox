package api

const (
	EventEngineStatus           = "qkbox.engine.status"
	EventEngineLog              = "qkbox.engine.log"
	EventEngineTraffic          = "qkbox.engine.traffic"
	EventEngineConnections      = "qkbox.engine.connections"
	EventEngineEventBridgeError = "qkbox.engine.eventBridgeError"
)

type RuntimeEvent struct {
	Event string           `json:"event"`
	Data  interface{}      `json:"data,omitempty"`
	Error *StructuredError `json:"error,omitempty"`
}

type SubscriptionAck struct{}

type RuntimeLogEntry struct {
	Seq       uint64 `json:"seq"`
	Timestamp int64  `json:"timestamp"`
	Source    string `json:"source"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

type TrafficSnapshot struct {
	Timestamp     int64 `json:"timestamp"`
	UploadTotal   int64 `json:"upload_total"`
	DownloadTotal int64 `json:"download_total"`
	UploadRate    int64 `json:"upload_rate"`
	DownloadRate  int64 `json:"download_rate"`
}

type RuntimeConnection struct {
	ID          string `json:"id"`
	Network     string `json:"network"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Host        string `json:"host,omitempty"`
	Process     string `json:"process,omitempty"`
	Inbound     string `json:"inbound,omitempty"`
	Outbound    string `json:"outbound,omitempty"`
	Rule        string `json:"rule,omitempty"`
	Upload      int64  `json:"upload"`
	Download    int64  `json:"download"`
	StartedAt   int64  `json:"started_at"`
}

type ConnectionSnapshot struct {
	Timestamp     int64               `json:"timestamp"`
	UploadTotal   int64               `json:"upload_total"`
	DownloadTotal int64               `json:"download_total"`
	Connections   []RuntimeConnection `json:"connections"`
}

type OutboundOption struct {
	Tag     string `json:"tag"`
	Type    string `json:"type"`
	DelayMS int64  `json:"delay_ms,omitempty"`
}

type OutboundGroup struct {
	Tag       string           `json:"tag"`
	Type      string           `json:"type"`
	Selected  string           `json:"selected"`
	Outbounds []OutboundOption `json:"outbounds"`
}

type URLTestResult struct {
	Outbound     string `json:"outbound"`
	DelayMS      int64  `json:"delay_ms,omitempty"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type EngineSubscribeStatusRequest struct{}
type EngineSubscribeLogsRequest struct{}
type EngineSubscribeTrafficRequest struct{}
type EngineSubscribeConnectionsRequest struct{}

type EngineGetRuntimeCapabilitiesRequest struct{}
type EngineGetRuntimeCapabilitiesReply struct {
	Capabilities []Capability `json:"capabilities"`
}
type EngineGetRuntimeCapabilitiesResult struct {
	Reply *EngineGetRuntimeCapabilitiesReply `json:"reply,omitempty"`
	Error *StructuredError                   `json:"error,omitempty"`
}

type EngineListGroupsRequest struct{}
type EngineListGroupsReply struct {
	Groups []OutboundGroup `json:"groups"`
}
type EngineListGroupsResult struct {
	Reply *EngineListGroupsReply `json:"reply,omitempty"`
	Error *StructuredError       `json:"error,omitempty"`
}

type EngineSelectOutboundRequest struct {
	GroupTag    string `json:"group_tag"`
	OutboundTag string `json:"outbound_tag"`
}
type EngineSelectOutboundReply struct {
	Group OutboundGroup `json:"group"`
}
type EngineSelectOutboundResult struct {
	Reply *EngineSelectOutboundReply `json:"reply,omitempty"`
	Error *StructuredError           `json:"error,omitempty"`
}

type EngineURLTestRequest struct {
	GroupTag  string `json:"group_tag"`
	TimeoutMS int64  `json:"timeout_ms,omitempty"`
}
type EngineURLTestReply struct {
	Results []URLTestResult `json:"results"`
}
type EngineURLTestResult struct {
	Reply *EngineURLTestReply `json:"reply,omitempty"`
	Error *StructuredError    `json:"error,omitempty"`
}

type EngineCloseConnectionRequest struct {
	ConnectionID string `json:"connection_id"`
}
type EngineCloseConnectionReply struct{}
type EngineCloseConnectionResult struct {
	Reply *EngineCloseConnectionReply `json:"reply,omitempty"`
	Error *StructuredError            `json:"error,omitempty"`
}

type EngineCloseAllConnectionsRequest struct{}
type EngineCloseAllConnectionsReply struct{}
type EngineCloseAllConnectionsResult struct {
	Reply *EngineCloseAllConnectionsReply `json:"reply,omitempty"`
	Error *StructuredError                `json:"error,omitempty"`
}

type RuntimeEventBridgeStartReply struct{}
type RuntimeEventBridgeStartResult struct {
	Reply *RuntimeEventBridgeStartReply `json:"reply,omitempty"`
	Error *StructuredError              `json:"error,omitempty"`
}

type RuntimeEventBridgeStopReply struct{}
type RuntimeEventBridgeStopResult struct {
	Reply *RuntimeEventBridgeStopReply `json:"reply,omitempty"`
	Error *StructuredError             `json:"error,omitempty"`
}
