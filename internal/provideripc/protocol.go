package provideripc

import (
	"encoding/json"

	"github.com/zclkkk/qkbox/shared/api"
)

const (
	MethodAuth             = "provider.auth"
	MethodGetStatus        = "provider.getStatus"
	MethodPrepareFeature   = "provider.prepareFeature"
	MethodRunRepairAction  = "provider.runRepairAction"
	DefaultProviderVersion = api.QKBoxDVersion
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

type AuthRequest struct {
	Token string `json:"token"`
}

type AuthReply struct{}

type StatusReply struct {
	Version    string                  `json:"version"`
	OwnerState *api.ProviderOwnerState `json:"owner_state,omitempty"`
}

var MethodRegistry = map[string]struct{}{
	MethodAuth:            {},
	MethodGetStatus:       {},
	MethodPrepareFeature:  {},
	MethodRunRepairAction: {},
}
