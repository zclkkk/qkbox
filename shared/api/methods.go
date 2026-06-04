package api

const (
	MethodHello = "hello"

	// Profile CRUD
	MethodCreateProfile      = "createProfile"
	MethodUpdateProfileDraft = "updateProfileDraft"
	MethodDeleteProfile      = "deleteProfile"
	MethodListProfiles       = "listProfiles"
	MethodGetProfile         = "getProfile"

	// Snapshot lifecycle
	MethodValidateProfileDraft    = "validateProfileDraft"
	MethodGetProfileDiagnostics   = "getProfileDiagnostics"
	MethodCreateProfileSnapshot   = "createProfileSnapshot"
	MethodActivateProfileSnapshot = "activateProfileSnapshot"
	MethodGetActiveProfile        = "getActiveProfile"
	MethodGetActiveSnapshot       = "getActiveSnapshot"
	MethodListSnapshots           = "listSnapshots"
	MethodRollbackToSnapshot      = "rollbackToSnapshot"

	// Engine
	MethodEngineStart                  = "engine.start"
	MethodEngineStop                   = "engine.stop"
	MethodEngineGetStatus              = "engine.getStatus"
	MethodEngineSubscribeStatus        = "engine.subscribeStatus"
	MethodEngineSubscribeLogs          = "engine.subscribeLogs"
	MethodEngineSubscribeTraffic       = "engine.subscribeTraffic"
	MethodEngineSubscribeConnections   = "engine.subscribeConnections"
	MethodEngineGetRuntimeCapabilities = "engine.getRuntimeCapabilities"
	MethodEngineListGroups             = "engine.listGroups"
	MethodEngineSelectOutbound         = "engine.selectOutbound"
	MethodEngineURLTest                = "engine.urlTest"
	MethodEngineCloseConnection        = "engine.closeConnection"
	MethodEngineCloseAllConnections    = "engine.closeAllConnections"

	// Platform capabilities
	MethodPlatformGetSystemProxyStatus  = "platform.getSystemProxyStatus"
	MethodPlatformSetSystemProxyEnabled = "platform.setSystemProxyEnabled"
)

var MethodRegistry = map[string]struct{}{
	MethodHello:                   {},
	MethodCreateProfile:           {},
	MethodUpdateProfileDraft:      {},
	MethodDeleteProfile:           {},
	MethodListProfiles:            {},
	MethodGetProfile:              {},
	MethodValidateProfileDraft:    {},
	MethodGetProfileDiagnostics:   {},
	MethodCreateProfileSnapshot:   {},
	MethodActivateProfileSnapshot: {},
	MethodGetActiveProfile:        {},
	MethodGetActiveSnapshot:       {},
	MethodListSnapshots:           {},
	MethodRollbackToSnapshot:      {},

	MethodEngineStart:                  {},
	MethodEngineStop:                   {},
	MethodEngineGetStatus:              {},
	MethodEngineSubscribeStatus:        {},
	MethodEngineSubscribeLogs:          {},
	MethodEngineSubscribeTraffic:       {},
	MethodEngineSubscribeConnections:   {},
	MethodEngineGetRuntimeCapabilities: {},
	MethodEngineListGroups:             {},
	MethodEngineSelectOutbound:         {},
	MethodEngineURLTest:                {},
	MethodEngineCloseConnection:        {},
	MethodEngineCloseAllConnections:    {},

	MethodPlatformGetSystemProxyStatus:  {},
	MethodPlatformSetSystemProxyEnabled: {},
}
