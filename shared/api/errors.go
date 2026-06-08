package api

import "fmt"

const (
	ErrorIPCVersionUnsupported = "IPC_VERSION_UNSUPPORTED"
	ErrorIPCMethodNotFound     = "IPC_METHOD_NOT_FOUND"
	ErrorIPCInvalidRequest     = "IPC_INVALID_REQUEST"
	ErrorIPCTransport          = "IPC_TRANSPORT"
	ErrorDaemonUnavailable     = "IPC_DAEMON_UNAVAILABLE"
	ErrorDaemonLaunchFailed    = "IPC_DAEMON_LAUNCH_FAILED"
	ErrorInternal              = "INTERNAL"

	ErrorProfileNotFound     = "PROFILE_NOT_FOUND"
	ErrorProfileNameEmpty    = "PROFILE_NAME_EMPTY"
	ErrorProfileContentEmpty = "PROFILE_CONTENT_EMPTY"
	ErrorProfileHasSnapshot  = "PROFILE_HAS_ACTIVE_SNAPSHOT"

	ErrorSubscriptionNotFound    = "SUBSCRIPTION_NOT_FOUND"
	ErrorSubscriptionURLEmpty    = "SUBSCRIPTION_URL_EMPTY"
	ErrorSubscriptionURLInvalid  = "SUBSCRIPTION_URL_INVALID"
	ErrorSubscriptionFetchFailed = "SUBSCRIPTION_FETCH_FAILED"

	ErrorSnapshotNotFound      = "SNAPSHOT_NOT_FOUND"
	ErrorSnapshotCreateFailed  = "SNAPSHOT_CREATE_FAILED"
	ErrorSnapshotAlreadyActive = "SNAPSHOT_ALREADY_ACTIVE"

	ErrorConfigValidationFailed = "CONFIG_VALIDATION_FAILED"
	ErrorConfigInvalidJSON      = "CONFIG_INVALID_JSON"

	ErrorAssetNotFound         = "ASSET_NOT_FOUND"
	ErrorAssetSourceURLInvalid = "ASSET_SOURCE_URL_INVALID"
	ErrorAssetKindUnsupported  = "ASSET_KIND_UNSUPPORTED"
	ErrorAssetFetchFailed      = "ASSET_FETCH_FAILED"
	ErrorAssetValidationFailed = "ASSET_VALIDATION_FAILED"
	ErrorAssetCacheFailed      = "ASSET_CACHE_FAILED"

	ErrorDiagnosticsBundleFailed = "DIAGNOSTICS_BUNDLE_FAILED"

	ErrorEngineNoActiveSnapshot = "ENGINE_NO_ACTIVE_SNAPSHOT"
	ErrorEngineAlreadyStarted   = "ENGINE_ALREADY_STARTED"
	ErrorEngineNotStarted       = "ENGINE_NOT_STARTED"
	ErrorEngineBusy             = "ENGINE_BUSY"
	ErrorEngineRunning          = "ENGINE_RUNNING"

	ErrorSingboxAdapterConfigFailed = "SINGBOX_ADAPTER_CONFIG_FAILED"
	ErrorSingboxAdapterStartFailed  = "SINGBOX_ADAPTER_START_FAILED"
	ErrorSingboxAdapterStopFailed   = "SINGBOX_ADAPTER_STOP_FAILED"

	ErrorObservabilityUnavailable  = "OBSERVABILITY_UNAVAILABLE"
	ErrorObservabilityUnsupported  = "OBSERVABILITY_UNSUPPORTED"
	ErrorObservabilityDegraded     = "OBSERVABILITY_DEGRADED"
	ErrorRuntimeGroupNotFound      = "RUNTIME_GROUP_NOT_FOUND"
	ErrorRuntimeOutboundNotFound   = "RUNTIME_OUTBOUND_NOT_FOUND"
	ErrorRuntimeConnectionNotFound = "RUNTIME_CONNECTION_NOT_FOUND"

	ErrorPlatformProxyUnsupported = "PLATFORM_PROXY_UNSUPPORTED"
	ErrorPlatformProxyFailed      = "PLATFORM_PROXY_FAILED"
	ErrorPlatformProxyNoListener  = "PLATFORM_PROXY_NO_LISTENER"

	ErrorPlatformProviderUnavailable     = "PLATFORM_PROVIDER_UNAVAILABLE"
	ErrorPlatformProviderAuthFailed      = "PLATFORM_PROVIDER_AUTH_FAILED"
	ErrorPlatformProviderVersionMismatch = "PLATFORM_PROVIDER_VERSION_MISMATCH"
	ErrorPlatformFeatureUnsupported      = "PLATFORM_FEATURE_UNSUPPORTED"
	ErrorPlatformPrepareFailed           = "PLATFORM_PREPARE_FAILED"
	ErrorPlatformRepairActionNotFound    = "PLATFORM_REPAIR_ACTION_NOT_FOUND"
	ErrorPlatformRepairFailed            = "PLATFORM_REPAIR_FAILED"

	ErrorNetworkModeOwnedByAnotherSession = "NETWORK_MODE_OWNED_BY_ANOTHER_SESSION"
	ErrorProviderRuntimeUnavailable       = "PROVIDER_RUNTIME_UNAVAILABLE"
	ErrorProviderRuntimeStale             = "PROVIDER_RUNTIME_STALE"
	ErrorProviderRuntimeStartFailed       = "PROVIDER_RUNTIME_START_FAILED"
	ErrorProviderRuntimeStopFailed        = "PROVIDER_RUNTIME_STOP_FAILED"
	ErrorProviderRuntimeEventStreamFailed = "PROVIDER_RUNTIME_EVENT_STREAM_FAILED"

	ErrorNetworkExtensionUnavailable       = "NETWORK_EXTENSION_UNAVAILABLE"
	ErrorNetworkExtensionStartFailed       = "NETWORK_EXTENSION_START_FAILED"
	ErrorNetworkExtensionStopFailed        = "NETWORK_EXTENSION_STOP_FAILED"
	ErrorNetworkExtensionEventStreamFailed = "NETWORK_EXTENSION_EVENT_STREAM_FAILED"
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
		"Client API version is not supported by qkbox.",
		"qkboxd",
		false,
	)
	err.Detail = map[string]string{
		"client_api_version":        clientVersion,
		"api_version":               APIVersion,
		"min_supported_api_version": MinSupportedAPIVersion,
	}
	err.UserAction = "Update qkbox to a compatible version."
	return err
}
