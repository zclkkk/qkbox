package api

const (
	MethodHello = "hello"

	// Profile CRUD
	MethodCreateProfile      = "createProfile"
	MethodUpdateProfileDraft = "updateProfileDraft"
	MethodDeleteProfile      = "deleteProfile"
	MethodListProfiles       = "listProfiles"
	MethodGetProfile         = "getProfile"

	// Data assets and subscriptions
	MethodAssetCreateProfileSubscription  = "asset.createProfileSubscription"
	MethodAssetListProfileSubscriptions   = "asset.listProfileSubscriptions"
	MethodAssetRefreshProfileSubscription = "asset.refreshProfileSubscription"
	MethodAssetDeleteProfileSubscription  = "asset.deleteProfileSubscription"
	MethodAssetCreateDataAsset            = "asset.createDataAsset"
	MethodAssetListDataAssets             = "asset.listDataAssets"
	MethodAssetRefreshDataAsset           = "asset.refreshDataAsset"
	MethodAssetDeleteDataAsset            = "asset.deleteDataAsset"

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
	MethodEngineReload                 = "engine.reload"
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
	MethodPlatformGetCapabilities             = "platform.getCapabilities"
	MethodPlatformGetPrivilegedProviderStatus = "platform.getPrivilegedProviderStatus"
	MethodPlatformPrepareFeature              = "platform.prepareFeature"
	MethodPlatformRunRepairAction             = "platform.runRepairAction"
	MethodPlatformGetSystemProxyStatus        = "platform.getSystemProxyStatus"
	MethodPlatformSetSystemProxyEnabled       = "platform.setSystemProxyEnabled"
)

var MethodRegistry = map[string]struct{}{
	MethodHello:                           {},
	MethodCreateProfile:                   {},
	MethodUpdateProfileDraft:              {},
	MethodDeleteProfile:                   {},
	MethodListProfiles:                    {},
	MethodGetProfile:                      {},
	MethodAssetCreateProfileSubscription:  {},
	MethodAssetListProfileSubscriptions:   {},
	MethodAssetRefreshProfileSubscription: {},
	MethodAssetDeleteProfileSubscription:  {},
	MethodAssetCreateDataAsset:            {},
	MethodAssetListDataAssets:             {},
	MethodAssetRefreshDataAsset:           {},
	MethodAssetDeleteDataAsset:            {},
	MethodValidateProfileDraft:            {},
	MethodGetProfileDiagnostics:           {},
	MethodCreateProfileSnapshot:           {},
	MethodActivateProfileSnapshot:         {},
	MethodGetActiveProfile:                {},
	MethodGetActiveSnapshot:               {},
	MethodListSnapshots:                   {},
	MethodRollbackToSnapshot:              {},

	MethodEngineStart:                  {},
	MethodEngineStop:                   {},
	MethodEngineReload:                 {},
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

	MethodPlatformGetCapabilities:             {},
	MethodPlatformGetPrivilegedProviderStatus: {},
	MethodPlatformPrepareFeature:              {},
	MethodPlatformRunRepairAction:             {},
	MethodPlatformGetSystemProxyStatus:        {},
	MethodPlatformSetSystemProxyEnabled:       {},
}
