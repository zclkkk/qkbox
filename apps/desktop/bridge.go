package main

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/zclkkk/qkbox/internal/ipc"
	"github.com/zclkkk/qkbox/shared/api"
)

type BridgeService struct {
	client      *ipc.Client
	eventMu     sync.Mutex
	eventCancel context.CancelFunc
}

func NewBridgeService() *BridgeService {
	return &BridgeService{client: ipc.NewClient()}
}

// Hello performs the initial handshake. Returns error if qkbox is unreachable.
// qkbox-window does NOT try to launch qkbox — it must already be running.
func (b *BridgeService) Hello(ctx context.Context) api.HelloResult {
	reply, structured := b.client.Hello(ctx, api.DefaultHelloRequest())
	if structured != nil {
		return api.HelloResult{Error: structured}
	}
	return api.HelloResult{Reply: &reply}
}

// Engine lifecycle

func (b *BridgeService) EngineStart(ctx context.Context) api.EngineStartResult {
	reply, structured := b.client.EngineStart(ctx, api.EngineStartRequest{})
	if structured != nil {
		return api.EngineStartResult{Error: structured}
	}
	return api.EngineStartResult{Reply: &reply}
}

func (b *BridgeService) EngineStop(ctx context.Context) api.EngineStopResult {
	reply, structured := b.client.EngineStop(ctx, api.EngineStopRequest{})
	if structured != nil {
		return api.EngineStopResult{Error: structured}
	}
	return api.EngineStopResult{Reply: &reply}
}

func (b *BridgeService) EngineGetStatus(ctx context.Context) api.EngineGetStatusResult {
	reply, structured := b.client.EngineGetStatus(ctx, api.EngineGetStatusRequest{})
	if structured != nil {
		return api.EngineGetStatusResult{Error: structured}
	}
	return api.EngineGetStatusResult{Reply: &reply}
}

func (b *BridgeService) StartRuntimeEventBridge(ctx context.Context) api.RuntimeEventBridgeStartResult {
	hello := b.Hello(ctx)
	if hello.Error != nil {
		return api.RuntimeEventBridgeStartResult{Error: hello.Error}
	}

	b.eventMu.Lock()
	if b.eventCancel != nil {
		b.eventCancel()
	}
	bridgeCtx, cancel := context.WithCancel(context.Background())
	b.eventCancel = cancel
	b.eventMu.Unlock()

	// Engine event subscriptions.
	subscriptions := []func(context.Context) (<-chan ipc.EventFrame, *api.StructuredError){
		func(ctx context.Context) (<-chan ipc.EventFrame, *api.StructuredError) {
			return b.client.EngineSubscribeStatus(ctx, api.EngineSubscribeStatusRequest{})
		},
		func(ctx context.Context) (<-chan ipc.EventFrame, *api.StructuredError) {
			return b.client.EngineSubscribeLogs(ctx, api.EngineSubscribeLogsRequest{})
		},
		func(ctx context.Context) (<-chan ipc.EventFrame, *api.StructuredError) {
			return b.client.EngineSubscribeTraffic(ctx, api.EngineSubscribeTrafficRequest{})
		},
		func(ctx context.Context) (<-chan ipc.EventFrame, *api.StructuredError) {
			return b.client.EngineSubscribeConnections(ctx, api.EngineSubscribeConnectionsRequest{})
		},
	}
	for _, open := range subscriptions {
		events, structured := open(bridgeCtx)
		if structured != nil {
			cancel()
			return api.RuntimeEventBridgeStartResult{Error: structured}
		}
		go forwardRuntimeEvents(bridgeCtx, events)
	}

	// Window attach subscription — listens for ShowWindow events from the tray.
	events, structured := b.client.WindowAttach(bridgeCtx, api.WindowAttachRequest{})
	if structured != nil {
		cancel()
		return api.RuntimeEventBridgeStartResult{Error: structured}
	}
	go forwardWindowEvents(bridgeCtx, events)

	return api.RuntimeEventBridgeStartResult{Reply: &api.RuntimeEventBridgeStartReply{}}
}

func (b *BridgeService) StopRuntimeEventBridge(context.Context) api.RuntimeEventBridgeStopResult {
	b.eventMu.Lock()
	if b.eventCancel != nil {
		b.eventCancel()
		b.eventCancel = nil
	}
	b.eventMu.Unlock()
	return api.RuntimeEventBridgeStopResult{Reply: &api.RuntimeEventBridgeStopReply{}}
}

func (b *BridgeService) EngineGetRuntimeCapabilities(ctx context.Context) api.EngineGetRuntimeCapabilitiesResult {
	reply, structured := b.client.EngineGetRuntimeCapabilities(ctx, api.EngineGetRuntimeCapabilitiesRequest{})
	if structured != nil {
		return api.EngineGetRuntimeCapabilitiesResult{Error: structured}
	}
	return api.EngineGetRuntimeCapabilitiesResult{Reply: &reply}
}

func (b *BridgeService) EngineListGroups(ctx context.Context) api.EngineListGroupsResult {
	reply, structured := b.client.EngineListGroups(ctx, api.EngineListGroupsRequest{})
	if structured != nil {
		return api.EngineListGroupsResult{Error: structured}
	}
	return api.EngineListGroupsResult{Reply: &reply}
}

func (b *BridgeService) EngineSelectOutbound(ctx context.Context, req api.EngineSelectOutboundRequest) api.EngineSelectOutboundResult {
	reply, structured := b.client.EngineSelectOutbound(ctx, req)
	if structured != nil {
		return api.EngineSelectOutboundResult{Error: structured}
	}
	return api.EngineSelectOutboundResult{Reply: &reply}
}

func (b *BridgeService) EngineURLTest(ctx context.Context, req api.EngineURLTestRequest) api.EngineURLTestResult {
	reply, structured := b.client.EngineURLTest(ctx, req)
	if structured != nil {
		return api.EngineURLTestResult{Error: structured}
	}
	return api.EngineURLTestResult{Reply: &reply}
}

func (b *BridgeService) EngineCloseConnection(ctx context.Context, req api.EngineCloseConnectionRequest) api.EngineCloseConnectionResult {
	reply, structured := b.client.EngineCloseConnection(ctx, req)
	if structured != nil {
		return api.EngineCloseConnectionResult{Error: structured}
	}
	return api.EngineCloseConnectionResult{Reply: &reply}
}

func (b *BridgeService) EngineCloseAllConnections(ctx context.Context) api.EngineCloseAllConnectionsResult {
	reply, structured := b.client.EngineCloseAllConnections(ctx, api.EngineCloseAllConnectionsRequest{})
	if structured != nil {
		return api.EngineCloseAllConnectionsResult{Error: structured}
	}
	return api.EngineCloseAllConnectionsResult{Reply: &reply}
}

// Profiles

func (b *BridgeService) CreateProfile(ctx context.Context, req api.CreateProfileRequest) api.CreateProfileResult {
	reply, structured := b.client.CreateProfile(ctx, req)
	if structured != nil {
		return api.CreateProfileResult{Error: structured}
	}
	return api.CreateProfileResult{Reply: &reply}
}

func (b *BridgeService) UpdateProfile(ctx context.Context, req api.UpdateProfileRequest) api.UpdateProfileResult {
	reply, structured := b.client.UpdateProfile(ctx, req)
	if structured != nil {
		return api.UpdateProfileResult{Error: structured}
	}
	return api.UpdateProfileResult{Reply: &reply}
}

func (b *BridgeService) DeleteProfile(ctx context.Context, req api.DeleteProfileRequest) api.DeleteProfileResult {
	reply, structured := b.client.DeleteProfile(ctx, req)
	if structured != nil {
		return api.DeleteProfileResult{Error: structured}
	}
	return api.DeleteProfileResult{Reply: &reply}
}

func (b *BridgeService) ListProfiles(ctx context.Context) api.ListProfilesResult {
	reply, structured := b.client.ListProfiles(ctx, api.ListProfilesRequest{})
	if structured != nil {
		return api.ListProfilesResult{Error: structured}
	}
	return api.ListProfilesResult{Reply: &reply}
}

func (b *BridgeService) GetProfile(ctx context.Context, req api.GetProfileRequest) api.GetProfileResult {
	reply, structured := b.client.GetProfile(ctx, req)
	if structured != nil {
		return api.GetProfileResult{Error: structured}
	}
	return api.GetProfileResult{Reply: &reply}
}

func (b *BridgeService) SaveProfileContent(ctx context.Context, req api.SaveProfileContentRequest) api.SaveProfileContentResult {
	reply, structured := b.client.SaveProfileContent(ctx, req)
	if structured != nil {
		return api.SaveProfileContentResult{Error: structured}
	}
	return api.SaveProfileContentResult{Reply: &reply}
}

func (b *BridgeService) ValidateProfileContent(ctx context.Context, req api.ValidateProfileContentRequest) api.ValidateProfileContentResult {
	reply, structured := b.client.ValidateProfileContent(ctx, req)
	if structured != nil {
		return api.ValidateProfileContentResult{Error: structured}
	}
	return api.ValidateProfileContentResult{Reply: &reply}
}

func (b *BridgeService) ActivateProfile(ctx context.Context, req api.ActivateProfileRequest) api.ActivateProfileResult {
	reply, structured := b.client.ActivateProfile(ctx, req)
	if structured != nil {
		return api.ActivateProfileResult{Error: structured}
	}
	return api.ActivateProfileResult{Reply: &reply}
}

func (b *BridgeService) GetActiveProfile(ctx context.Context) api.GetActiveProfileResult {
	reply, structured := b.client.GetActiveProfile(ctx, api.GetActiveProfileRequest{})
	if structured != nil {
		return api.GetActiveProfileResult{Error: structured}
	}
	return api.GetActiveProfileResult{Reply: &reply}
}

// Data assets and subscriptions

func (b *BridgeService) AssetCreateProfileSubscription(ctx context.Context, req api.CreateProfileSubscriptionRequest) api.CreateProfileSubscriptionResult {
	reply, structured := b.client.CreateProfileSubscription(ctx, req)
	if structured != nil {
		return api.CreateProfileSubscriptionResult{Error: structured}
	}
	return api.CreateProfileSubscriptionResult{Reply: &reply}
}

func (b *BridgeService) AssetListProfileSubscriptions(ctx context.Context, req api.ListProfileSubscriptionsRequest) api.ListProfileSubscriptionsResult {
	reply, structured := b.client.ListProfileSubscriptions(ctx, req)
	if structured != nil {
		return api.ListProfileSubscriptionsResult{Error: structured}
	}
	return api.ListProfileSubscriptionsResult{Reply: &reply}
}

func (b *BridgeService) AssetRefreshProfileSubscription(ctx context.Context, req api.RefreshProfileSubscriptionRequest) api.RefreshProfileSubscriptionResult {
	reply, structured := b.client.RefreshProfileSubscription(ctx, req)
	if structured != nil {
		return api.RefreshProfileSubscriptionResult{Error: structured}
	}
	return api.RefreshProfileSubscriptionResult{Reply: &reply}
}

func (b *BridgeService) AssetDeleteProfileSubscription(ctx context.Context, req api.DeleteProfileSubscriptionRequest) api.DeleteProfileSubscriptionResult {
	reply, structured := b.client.DeleteProfileSubscription(ctx, req)
	if structured != nil {
		return api.DeleteProfileSubscriptionResult{Error: structured}
	}
	return api.DeleteProfileSubscriptionResult{Reply: &reply}
}

func (b *BridgeService) AssetCreateDataAsset(ctx context.Context, req api.CreateDataAssetRequest) api.CreateDataAssetResult {
	reply, structured := b.client.CreateDataAsset(ctx, req)
	if structured != nil {
		return api.CreateDataAssetResult{Error: structured}
	}
	return api.CreateDataAssetResult{Reply: &reply}
}

func (b *BridgeService) AssetListDataAssets(ctx context.Context, req api.ListDataAssetsRequest) api.ListDataAssetsResult {
	reply, structured := b.client.ListDataAssets(ctx, req)
	if structured != nil {
		return api.ListDataAssetsResult{Error: structured}
	}
	return api.ListDataAssetsResult{Reply: &reply}
}

func (b *BridgeService) AssetRefreshDataAsset(ctx context.Context, req api.RefreshDataAssetRequest) api.RefreshDataAssetResult {
	reply, structured := b.client.RefreshDataAsset(ctx, req)
	if structured != nil {
		return api.RefreshDataAssetResult{Error: structured}
	}
	return api.RefreshDataAssetResult{Reply: &reply}
}

func (b *BridgeService) AssetDeleteDataAsset(ctx context.Context, req api.DeleteDataAssetRequest) api.DeleteDataAssetResult {
	reply, structured := b.client.DeleteDataAsset(ctx, req)
	if structured != nil {
		return api.DeleteDataAssetResult{Error: structured}
	}
	return api.DeleteDataAssetResult{Reply: &reply}
}

// Diagnostics and recovery

func (b *BridgeService) DiagnosticsGetReport(ctx context.Context) api.GetDiagnosticsReportResult {
	reply, structured := b.client.DiagnosticsGetReport(ctx, api.GetDiagnosticsReportRequest{})
	if structured != nil {
		return api.GetDiagnosticsReportResult{Error: structured}
	}
	return api.GetDiagnosticsReportResult{Reply: &reply}
}

func (b *BridgeService) DiagnosticsCreateDebugBundle(ctx context.Context) api.CreateDebugBundleResult {
	reply, structured := b.client.DiagnosticsCreateDebugBundle(ctx, api.CreateDebugBundleRequest{})
	if structured != nil {
		return api.CreateDebugBundleResult{Error: structured}
	}
	return api.CreateDebugBundleResult{Reply: &reply}
}

// Platform capabilities

func (b *BridgeService) PlatformGetCapabilities(ctx context.Context) api.GetPlatformCapabilitiesResult {
	reply, structured := b.client.PlatformGetCapabilities(ctx, api.GetPlatformCapabilitiesRequest{})
	if structured != nil {
		return api.GetPlatformCapabilitiesResult{Error: structured}
	}
	return api.GetPlatformCapabilitiesResult{Reply: &reply}
}

func (b *BridgeService) PlatformGetPrivilegedProviderStatus(ctx context.Context) api.GetPrivilegedProviderStatusResult {
	reply, structured := b.client.PlatformGetPrivilegedProviderStatus(ctx, api.GetPrivilegedProviderStatusRequest{})
	if structured != nil {
		return api.GetPrivilegedProviderStatusResult{Error: structured}
	}
	return api.GetPrivilegedProviderStatusResult{Reply: &reply}
}

func (b *BridgeService) PlatformGetNetworkExtensionStatus(ctx context.Context) api.GetNetworkExtensionStatusResult {
	reply, structured := b.client.PlatformGetNetworkExtensionStatus(ctx, api.GetNetworkExtensionStatusRequest{})
	if structured != nil {
		return api.GetNetworkExtensionStatusResult{Error: structured}
	}
	return api.GetNetworkExtensionStatusResult{Reply: &reply}
}

func (b *BridgeService) PlatformPrepareFeature(ctx context.Context, req api.PrepareFeatureRequest) api.PrepareFeatureResult {
	reply, structured := b.client.PlatformPrepareFeature(ctx, req)
	if structured != nil {
		return api.PrepareFeatureResult{Error: structured}
	}
	return api.PrepareFeatureResult{Reply: &reply}
}

func (b *BridgeService) PlatformRunRepairAction(ctx context.Context, req api.RunRepairActionRequest) api.RunRepairActionResult {
	reply, structured := b.client.PlatformRunRepairAction(ctx, req)
	if structured != nil {
		return api.RunRepairActionResult{Error: structured}
	}
	return api.RunRepairActionResult{Reply: &reply}
}

func (b *BridgeService) PlatformGetSystemProxyStatus(ctx context.Context) api.GetSystemProxyStatusResult {
	reply, structured := b.client.PlatformGetSystemProxyStatus(ctx, api.GetSystemProxyStatusRequest{})
	if structured != nil {
		return api.GetSystemProxyStatusResult{Error: structured}
	}
	return api.GetSystemProxyStatusResult{Reply: &reply}
}

func (b *BridgeService) PlatformSetSystemProxyEnabled(ctx context.Context, req api.SetSystemProxyEnabledRequest) api.SetSystemProxyEnabledResult {
	reply, structured := b.client.PlatformSetSystemProxyEnabled(ctx, req)
	if structured != nil {
		return api.SetSystemProxyEnabledResult{Error: structured}
	}
	return api.SetSystemProxyEnabledResult{Reply: &reply}
}

// Event forwarding

func forwardRuntimeEvents(ctx context.Context, events <-chan ipc.EventFrame) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			emitRuntimeEvent(event)
		}
	}
}

// forwardWindowEvents handles the window.attach event stream.
// EventWindowShow is intercepted and triggers the window to show/focus.
func forwardWindowEvents(ctx context.Context, events <-chan ipc.EventFrame) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			if event.Error != nil {
				continue
			}
			if event.Event == api.EventWindowShow {
				windows := application.Get().Window.GetAll()
				if len(windows) > 0 {
					windows[0].Show()
					windows[0].Focus()
				}
				continue
			}
			// Forward other window events to the frontend (future use).
			emitRuntimeEvent(event)
		}
	}
}

func emitRuntimeEvent(frame ipc.EventFrame) {
	if frame.Error != nil {
		application.Get().Event.Emit(api.EventEngineEventBridgeError, frame.Error)
		return
	}
	if frame.Event == "" {
		return
	}
	var payload interface{}
	if len(frame.Data) > 0 {
		if err := json.Unmarshal(frame.Data, &payload); err != nil {
			application.Get().Event.Emit(api.EventEngineEventBridgeError, api.NewStructuredError(api.ErrorIPCTransport, err.Error(), "desktop", true))
			return
		}
	}
	application.Get().Event.Emit(frame.Event, payload)
}
