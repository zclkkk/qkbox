package api

type GetSystemProxyStatusRequest struct{}

type GetSystemProxyStatusReply struct {
	Available  bool   `json:"available"`
	Supported  bool   `json:"supported"`
	Reason     string `json:"reason,omitempty"`
	OSEnabled  bool   `json:"os_enabled"`
	QKBoxOwned bool   `json:"qkbox_owned"`
	Address    string `json:"address,omitempty"`
	Port       int    `json:"port,omitempty"`
}

type GetSystemProxyStatusResult struct {
	Reply *GetSystemProxyStatusReply `json:"reply,omitempty"`
	Error *StructuredError           `json:"error,omitempty"`
}

type SetSystemProxyEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

type SetSystemProxyEnabledReply struct{}

type SetSystemProxyEnabledResult struct {
	Reply *SetSystemProxyEnabledReply `json:"reply,omitempty"`
	Error *StructuredError            `json:"error,omitempty"`
}
