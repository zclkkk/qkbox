package api

const (
	MethodHello = "hello"

	// Profile CRUD
	MethodCreateProfile          = "createProfile"
	MethodUpdateProfile          = "updateProfile"
	MethodDeleteProfile          = "deleteProfile"
	MethodListProfiles           = "listProfiles"
	MethodGetProfile             = "getProfile"
	MethodSaveProfileContent     = "saveProfileContent"
	MethodValidateProfileContent = "validateProfileContent"
	MethodActivateProfile        = "activateProfile"
	MethodGetActiveProfile       = "getActiveProfile"

	// Data assets and subscriptions
	MethodAssetCreateProfileSubscription  = "asset.createProfileSubscription"
	MethodAssetListProfileSubscriptions   = "asset.listProfileSubscriptions"
	MethodAssetRefreshProfileSubscription = "asset.refreshProfileSubscription"
	MethodAssetDeleteProfileSubscription  = "asset.deleteProfileSubscription"
	MethodAssetCreateDataAsset            = "asset.createDataAsset"
	MethodAssetListDataAssets             = "asset.listDataAssets"
	MethodAssetRefreshDataAsset           = "asset.refreshDataAsset"
	MethodAssetDeleteDataAsset            = "asset.deleteDataAsset"

	// Diagnostics and recovery
	MethodDiagnosticsGetReport         = "diagnostics.getReport"
	MethodDiagnosticsCreateDebugBundle = "diagnostics.createDebugBundle"

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

	// Window session
	MethodWindowAttach = "window.attach"

	// Platform capabilities
	MethodPlatformGetCapabilities             = "platform.getCapabilities"
	MethodPlatformGetPrivilegedProviderStatus = "platform.getPrivilegedProviderStatus"
	MethodPlatformGetNetworkExtensionStatus   = "platform.getNetworkExtensionStatus"
	MethodPlatformPrepareFeature              = "platform.prepareFeature"
	MethodPlatformRunRepairAction             = "platform.runRepairAction"
	MethodPlatformGetSystemProxyStatus        = "platform.getSystemProxyStatus"
	MethodPlatformSetSystemProxyEnabled       = "platform.setSystemProxyEnabled"
)

var MethodRegistry = map[string]struct{}{
	MethodHello:                           {},
	MethodCreateProfile:                   {},
	MethodUpdateProfile:                   {},
	MethodDeleteProfile:                   {},
	MethodListProfiles:                    {},
	MethodGetProfile:                      {},
	MethodSaveProfileContent:              {},
	MethodValidateProfileContent:          {},
	MethodActivateProfile:                 {},
	MethodGetActiveProfile:                {},
	MethodAssetCreateProfileSubscription:  {},
	MethodAssetListProfileSubscriptions:   {},
	MethodAssetRefreshProfileSubscription: {},
	MethodAssetDeleteProfileSubscription:  {},
	MethodAssetCreateDataAsset:            {},
	MethodAssetListDataAssets:             {},
	MethodAssetRefreshDataAsset:           {},
	MethodAssetDeleteDataAsset:            {},
	MethodDiagnosticsGetReport:            {},
	MethodDiagnosticsCreateDebugBundle:    {},
	MethodEngineStart:                     {},
	MethodEngineStop:                      {},
	MethodEngineGetStatus:                 {},
	MethodEngineSubscribeStatus:           {},
	MethodEngineSubscribeLogs:             {},
	MethodEngineSubscribeTraffic:          {},
	MethodEngineSubscribeConnections:      {},
	MethodEngineGetRuntimeCapabilities:    {},
	MethodEngineListGroups:                {},
	MethodEngineSelectOutbound:            {},
	MethodEngineURLTest:                   {},
	MethodEngineCloseConnection:           {},
	MethodEngineCloseAllConnections:       {},

	MethodWindowAttach: {},

	MethodPlatformGetCapabilities:             {},
	MethodPlatformGetPrivilegedProviderStatus: {},
	MethodPlatformGetNetworkExtensionStatus:   {},
	MethodPlatformPrepareFeature:              {},
	MethodPlatformRunRepairAction:             {},
	MethodPlatformGetSystemProxyStatus:        {},
	MethodPlatformSetSystemProxyEnabled:       {},
}
