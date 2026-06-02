package ipc

import (
	"encoding/json"

	"github.com/zclkkk/qkbox/shared/api"
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
