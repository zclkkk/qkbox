package ipc

import (
	"github.com/zclkkk/qkbox/internal/ipcframework"
	"github.com/zclkkk/qkbox/shared/api"
)

func newRegistry(handler Handler) *ipcframework.Registry {
	registry := ipcframework.NewRegistry("qkboxd")

	ipcframework.RegisterMethod(registry, api.MethodHello, handler.Hello)

	ipcframework.RegisterMethod(registry, api.MethodCreateProfile, handler.CreateProfile)
	ipcframework.RegisterMethod(registry, api.MethodUpdateProfile, handler.UpdateProfile)
	ipcframework.RegisterMethod(registry, api.MethodDeleteProfile, handler.DeleteProfile)
	ipcframework.RegisterMethod(registry, api.MethodListProfiles, handler.ListProfiles)
	ipcframework.RegisterMethod(registry, api.MethodGetProfile, handler.GetProfile)
	ipcframework.RegisterMethod(registry, api.MethodSaveProfileContent, handler.SaveProfileContent)
	ipcframework.RegisterMethod(registry, api.MethodValidateProfileContent, handler.ValidateProfileContent)
	ipcframework.RegisterMethod(registry, api.MethodActivateProfile, handler.ActivateProfile)
	ipcframework.RegisterMethod(registry, api.MethodGetActiveProfile, handler.GetActiveProfile)

	ipcframework.RegisterMethod(registry, api.MethodAssetCreateProfileSubscription, handler.CreateProfileSubscription)
	ipcframework.RegisterMethod(registry, api.MethodAssetListProfileSubscriptions, handler.ListProfileSubscriptions)
	ipcframework.RegisterMethod(registry, api.MethodAssetRefreshProfileSubscription, handler.RefreshProfileSubscription)
	ipcframework.RegisterMethod(registry, api.MethodAssetDeleteProfileSubscription, handler.DeleteProfileSubscription)
	ipcframework.RegisterMethod(registry, api.MethodAssetCreateDataAsset, handler.CreateDataAsset)
	ipcframework.RegisterMethod(registry, api.MethodAssetListDataAssets, handler.ListDataAssets)
	ipcframework.RegisterMethod(registry, api.MethodAssetRefreshDataAsset, handler.RefreshDataAsset)
	ipcframework.RegisterMethod(registry, api.MethodAssetDeleteDataAsset, handler.DeleteDataAsset)

	ipcframework.RegisterMethod(registry, api.MethodDiagnosticsGetReport, handler.DiagnosticsGetReport)
	ipcframework.RegisterMethod(registry, api.MethodDiagnosticsCreateDebugBundle, handler.DiagnosticsCreateDebugBundle)

	ipcframework.RegisterMethod(registry, api.MethodEngineStart, handler.EngineStart)
	ipcframework.RegisterMethod(registry, api.MethodEngineStop, handler.EngineStop)
	ipcframework.RegisterMethod(registry, api.MethodEngineGetStatus, handler.EngineGetStatus)
	ipcframework.RegisterSubscription(registry, api.MethodEngineSubscribeStatus, api.SubscriptionAck{}, handler.EngineSubscribeStatus)
	ipcframework.RegisterSubscription(registry, api.MethodEngineSubscribeLogs, api.SubscriptionAck{}, handler.EngineSubscribeLogs)
	ipcframework.RegisterSubscription(registry, api.MethodEngineSubscribeTraffic, api.SubscriptionAck{}, handler.EngineSubscribeTraffic)
	ipcframework.RegisterSubscription(registry, api.MethodEngineSubscribeConnections, api.SubscriptionAck{}, handler.EngineSubscribeConnections)
	ipcframework.RegisterMethod(registry, api.MethodEngineGetRuntimeCapabilities, handler.EngineGetRuntimeCapabilities)
	ipcframework.RegisterMethod(registry, api.MethodEngineListGroups, handler.EngineListGroups)
	ipcframework.RegisterMethod(registry, api.MethodEngineSelectOutbound, handler.EngineSelectOutbound)
	ipcframework.RegisterMethod(registry, api.MethodEngineURLTest, handler.EngineURLTest)
	ipcframework.RegisterMethod(registry, api.MethodEngineCloseConnection, handler.EngineCloseConnection)
	ipcframework.RegisterMethod(registry, api.MethodEngineCloseAllConnections, handler.EngineCloseAllConnections)

	ipcframework.RegisterSubscription(registry, api.MethodWindowAttach, api.SubscriptionAck{}, handler.WindowAttach)

	ipcframework.RegisterMethod(registry, api.MethodPlatformGetCapabilities, handler.PlatformGetCapabilities)
	ipcframework.RegisterMethod(registry, api.MethodPlatformGetPrivilegedProviderStatus, handler.PlatformGetPrivilegedProviderStatus)
	ipcframework.RegisterMethod(registry, api.MethodPlatformGetNetworkExtensionStatus, handler.PlatformGetNetworkExtensionStatus)
	ipcframework.RegisterMethod(registry, api.MethodPlatformPrepareFeature, handler.PlatformPrepareFeature)
	ipcframework.RegisterMethod(registry, api.MethodPlatformRunRepairAction, handler.PlatformRunRepairAction)
	ipcframework.RegisterMethod(registry, api.MethodPlatformGetSystemProxyStatus, handler.PlatformGetSystemProxyStatus)
	ipcframework.RegisterMethod(registry, api.MethodPlatformSetSystemProxyEnabled, handler.PlatformSetSystemProxyEnabled)

	return registry
}
