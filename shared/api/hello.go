package api

import "github.com/zclkkk/qkbox/shared/model"

type HelloRequest struct {
	ClientName       string `json:"client_name"`
	ClientVersion    string `json:"client_version"`
	ClientAPIVersion string `json:"client_api_version"`
}

type HelloReply struct {
	APIVersion             string         `json:"api_version"`
	MinSupportedAPIVersion string         `json:"min_supported_api_version"`
	SchemaRevision         string         `json:"schema_revision"`
	AppVersion             string         `json:"app_version"`
	QKBoxDVersion          string         `json:"qkboxd_version"`
	Platform               model.Platform `json:"platform"`
	RuntimeCapabilities    []Capability   `json:"runtime_capabilities"`
	PlatformCapabilities   []Capability   `json:"platform_capabilities"`
}

type HelloResult struct {
	Reply *HelloReply      `json:"reply,omitempty"`
	Error *StructuredError `json:"error,omitempty"`
}

func DefaultHelloRequest() HelloRequest {
	return HelloRequest{
		ClientName:       "qkbox-desktop",
		ClientVersion:    AppVersion,
		ClientAPIVersion: APIVersion,
	}
}
