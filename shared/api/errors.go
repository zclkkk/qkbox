package api

import "fmt"

const (
	ErrorIPCVersionUnsupported = "IPC_VERSION_UNSUPPORTED"
	ErrorIPCMethodNotFound     = "IPC_METHOD_NOT_FOUND"
	ErrorIPCInvalidRequest     = "IPC_INVALID_REQUEST"
	ErrorIPCTransport          = "IPC_TRANSPORT"
	ErrorDaemonUnavailable     = "IPC_DAEMON_UNAVAILABLE"
	ErrorDaemonLaunchFailed    = "IPC_DAEMON_LAUNCH_FAILED"
)

type StructuredError struct {
	Code        string      `json:"code"`
	Message     string      `json:"message"`
	Detail      interface{} `json:"detail,omitempty"`
	Source      string      `json:"source"`
	Recoverable bool        `json:"recoverable"`
	UserAction  string      `json:"user_action,omitempty"`
	DebugRef    string      `json:"debug_ref,omitempty"`
}

func (e *StructuredError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func NewStructuredError(code, message, source string, recoverable bool) *StructuredError {
	return &StructuredError{
		Code:        code,
		Message:     message,
		Source:      source,
		Recoverable: recoverable,
	}
}

func VersionUnsupported(clientVersion string) *StructuredError {
	err := NewStructuredError(
		ErrorIPCVersionUnsupported,
		"Client API version is not supported by qkboxd.",
		"qkboxd",
		false,
	)
	err.Detail = map[string]string{
		"client_api_version":        clientVersion,
		"api_version":               APIVersion,
		"min_supported_api_version": MinSupportedAPIVersion,
	}
	err.UserAction = "Update qkbox and qkboxd to compatible versions."
	return err
}
