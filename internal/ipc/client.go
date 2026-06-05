package ipc

import (
	"context"
	"encoding/json"
	"net"
	"time"

	"github.com/zclkkk/qkbox/shared/api"
)

type Client struct {
	dial func(context.Context) (net.Conn, error)
}

const defaultCallTimeout = 5 * time.Second
const subscriptionRefreshTimeout = 60 * time.Second
const dataAssetRefreshTimeout = 5 * time.Minute

func NewClient() *Client {
	return &Client{dial: Dial}
}

// Hello

func (c *Client) Hello(ctx context.Context, req api.HelloRequest) (api.HelloReply, *api.StructuredError) {
	return do[api.HelloRequest, api.HelloReply](c, ctx, api.MethodHello, req)
}

// Profile CRUD

func (c *Client) CreateProfile(ctx context.Context, req api.CreateProfileRequest) (api.CreateProfileReply, *api.StructuredError) {
	return do[api.CreateProfileRequest, api.CreateProfileReply](c, ctx, api.MethodCreateProfile, req)
}

func (c *Client) UpdateProfileDraft(ctx context.Context, req api.UpdateProfileDraftRequest) (api.UpdateProfileDraftReply, *api.StructuredError) {
	return do[api.UpdateProfileDraftRequest, api.UpdateProfileDraftReply](c, ctx, api.MethodUpdateProfileDraft, req)
}

func (c *Client) DeleteProfile(ctx context.Context, req api.DeleteProfileRequest) (api.DeleteProfileReply, *api.StructuredError) {
	return do[api.DeleteProfileRequest, api.DeleteProfileReply](c, ctx, api.MethodDeleteProfile, req)
}

func (c *Client) ListProfiles(ctx context.Context, req api.ListProfilesRequest) (api.ListProfilesReply, *api.StructuredError) {
	return do[api.ListProfilesRequest, api.ListProfilesReply](c, ctx, api.MethodListProfiles, req)
}

func (c *Client) GetProfile(ctx context.Context, req api.GetProfileRequest) (api.GetProfileReply, *api.StructuredError) {
	return do[api.GetProfileRequest, api.GetProfileReply](c, ctx, api.MethodGetProfile, req)
}

// Data assets and subscriptions

func (c *Client) CreateProfileSubscription(ctx context.Context, req api.CreateProfileSubscriptionRequest) (api.CreateProfileSubscriptionReply, *api.StructuredError) {
	return do[api.CreateProfileSubscriptionRequest, api.CreateProfileSubscriptionReply](c, ctx, api.MethodAssetCreateProfileSubscription, req)
}

func (c *Client) ListProfileSubscriptions(ctx context.Context, req api.ListProfileSubscriptionsRequest) (api.ListProfileSubscriptionsReply, *api.StructuredError) {
	return do[api.ListProfileSubscriptionsRequest, api.ListProfileSubscriptionsReply](c, ctx, api.MethodAssetListProfileSubscriptions, req)
}

func (c *Client) RefreshProfileSubscription(ctx context.Context, req api.RefreshProfileSubscriptionRequest) (api.RefreshProfileSubscriptionReply, *api.StructuredError) {
	return doWithTimeout[api.RefreshProfileSubscriptionRequest, api.RefreshProfileSubscriptionReply](c, ctx, api.MethodAssetRefreshProfileSubscription, req, subscriptionRefreshTimeout)
}

func (c *Client) DeleteProfileSubscription(ctx context.Context, req api.DeleteProfileSubscriptionRequest) (api.DeleteProfileSubscriptionReply, *api.StructuredError) {
	return do[api.DeleteProfileSubscriptionRequest, api.DeleteProfileSubscriptionReply](c, ctx, api.MethodAssetDeleteProfileSubscription, req)
}

func (c *Client) CreateDataAsset(ctx context.Context, req api.CreateDataAssetRequest) (api.CreateDataAssetReply, *api.StructuredError) {
	return do[api.CreateDataAssetRequest, api.CreateDataAssetReply](c, ctx, api.MethodAssetCreateDataAsset, req)
}

func (c *Client) ListDataAssets(ctx context.Context, req api.ListDataAssetsRequest) (api.ListDataAssetsReply, *api.StructuredError) {
	return do[api.ListDataAssetsRequest, api.ListDataAssetsReply](c, ctx, api.MethodAssetListDataAssets, req)
}

func (c *Client) RefreshDataAsset(ctx context.Context, req api.RefreshDataAssetRequest) (api.RefreshDataAssetReply, *api.StructuredError) {
	return doWithTimeout[api.RefreshDataAssetRequest, api.RefreshDataAssetReply](c, ctx, api.MethodAssetRefreshDataAsset, req, dataAssetRefreshTimeout)
}

func (c *Client) DeleteDataAsset(ctx context.Context, req api.DeleteDataAssetRequest) (api.DeleteDataAssetReply, *api.StructuredError) {
	return do[api.DeleteDataAssetRequest, api.DeleteDataAssetReply](c, ctx, api.MethodAssetDeleteDataAsset, req)
}

// Diagnostics and recovery

func (c *Client) DiagnosticsGetReport(ctx context.Context, req api.GetDiagnosticsReportRequest) (api.GetDiagnosticsReportReply, *api.StructuredError) {
	return do[api.GetDiagnosticsReportRequest, api.GetDiagnosticsReportReply](c, ctx, api.MethodDiagnosticsGetReport, req)
}

func (c *Client) DiagnosticsCreateDebugBundle(ctx context.Context, req api.CreateDebugBundleRequest) (api.CreateDebugBundleReply, *api.StructuredError) {
	return do[api.CreateDebugBundleRequest, api.CreateDebugBundleReply](c, ctx, api.MethodDiagnosticsCreateDebugBundle, req)
}

// Snapshot lifecycle

func (c *Client) ValidateProfileDraft(ctx context.Context, req api.ValidateProfileDraftRequest) (api.ValidateProfileDraftReply, *api.StructuredError) {
	return do[api.ValidateProfileDraftRequest, api.ValidateProfileDraftReply](c, ctx, api.MethodValidateProfileDraft, req)
}

func (c *Client) GetProfileDiagnostics(ctx context.Context, req api.GetProfileDiagnosticsRequest) (api.GetProfileDiagnosticsReply, *api.StructuredError) {
	return do[api.GetProfileDiagnosticsRequest, api.GetProfileDiagnosticsReply](c, ctx, api.MethodGetProfileDiagnostics, req)
}

func (c *Client) CreateProfileSnapshot(ctx context.Context, req api.CreateProfileSnapshotRequest) (api.CreateProfileSnapshotReply, *api.StructuredError) {
	return do[api.CreateProfileSnapshotRequest, api.CreateProfileSnapshotReply](c, ctx, api.MethodCreateProfileSnapshot, req)
}

func (c *Client) ActivateProfileSnapshot(ctx context.Context, req api.ActivateProfileSnapshotRequest) (api.ActivateProfileSnapshotReply, *api.StructuredError) {
	return do[api.ActivateProfileSnapshotRequest, api.ActivateProfileSnapshotReply](c, ctx, api.MethodActivateProfileSnapshot, req)
}

func (c *Client) GetActiveProfile(ctx context.Context, req api.GetActiveProfileRequest) (api.GetActiveProfileReply, *api.StructuredError) {
	return do[api.GetActiveProfileRequest, api.GetActiveProfileReply](c, ctx, api.MethodGetActiveProfile, req)
}

func (c *Client) GetActiveSnapshot(ctx context.Context, req api.GetActiveSnapshotRequest) (api.GetActiveSnapshotReply, *api.StructuredError) {
	return do[api.GetActiveSnapshotRequest, api.GetActiveSnapshotReply](c, ctx, api.MethodGetActiveSnapshot, req)
}

func (c *Client) ListSnapshots(ctx context.Context, req api.ListSnapshotsRequest) (api.ListSnapshotsReply, *api.StructuredError) {
	return do[api.ListSnapshotsRequest, api.ListSnapshotsReply](c, ctx, api.MethodListSnapshots, req)
}

func (c *Client) RollbackToSnapshot(ctx context.Context, req api.RollbackToSnapshotRequest) (api.RollbackToSnapshotReply, *api.StructuredError) {
	return do[api.RollbackToSnapshotRequest, api.RollbackToSnapshotReply](c, ctx, api.MethodRollbackToSnapshot, req)
}

// Engine lifecycle

func (c *Client) EngineStart(ctx context.Context, req api.EngineStartRequest) (api.EngineStartReply, *api.StructuredError) {
	return do[api.EngineStartRequest, api.EngineStartReply](c, ctx, api.MethodEngineStart, req)
}

func (c *Client) EngineStop(ctx context.Context, req api.EngineStopRequest) (api.EngineStopReply, *api.StructuredError) {
	return do[api.EngineStopRequest, api.EngineStopReply](c, ctx, api.MethodEngineStop, req)
}

func (c *Client) EngineReload(ctx context.Context, req api.EngineReloadRequest) (api.EngineReloadReply, *api.StructuredError) {
	return do[api.EngineReloadRequest, api.EngineReloadReply](c, ctx, api.MethodEngineReload, req)
}

func (c *Client) EngineGetStatus(ctx context.Context, req api.EngineGetStatusRequest) (api.EngineGetStatusReply, *api.StructuredError) {
	return do[api.EngineGetStatusRequest, api.EngineGetStatusReply](c, ctx, api.MethodEngineGetStatus, req)
}

func (c *Client) EngineSubscribeStatus(ctx context.Context, req api.EngineSubscribeStatusRequest) (<-chan EventFrame, *api.StructuredError) {
	return openSubscription(c, ctx, api.MethodEngineSubscribeStatus, req)
}

func (c *Client) EngineSubscribeLogs(ctx context.Context, req api.EngineSubscribeLogsRequest) (<-chan EventFrame, *api.StructuredError) {
	return openSubscription(c, ctx, api.MethodEngineSubscribeLogs, req)
}

func (c *Client) EngineSubscribeTraffic(ctx context.Context, req api.EngineSubscribeTrafficRequest) (<-chan EventFrame, *api.StructuredError) {
	return openSubscription(c, ctx, api.MethodEngineSubscribeTraffic, req)
}

func (c *Client) EngineSubscribeConnections(ctx context.Context, req api.EngineSubscribeConnectionsRequest) (<-chan EventFrame, *api.StructuredError) {
	return openSubscription(c, ctx, api.MethodEngineSubscribeConnections, req)
}

func (c *Client) EngineGetRuntimeCapabilities(ctx context.Context, req api.EngineGetRuntimeCapabilitiesRequest) (api.EngineGetRuntimeCapabilitiesReply, *api.StructuredError) {
	return do[api.EngineGetRuntimeCapabilitiesRequest, api.EngineGetRuntimeCapabilitiesReply](c, ctx, api.MethodEngineGetRuntimeCapabilities, req)
}

func (c *Client) EngineListGroups(ctx context.Context, req api.EngineListGroupsRequest) (api.EngineListGroupsReply, *api.StructuredError) {
	return do[api.EngineListGroupsRequest, api.EngineListGroupsReply](c, ctx, api.MethodEngineListGroups, req)
}

func (c *Client) EngineSelectOutbound(ctx context.Context, req api.EngineSelectOutboundRequest) (api.EngineSelectOutboundReply, *api.StructuredError) {
	return do[api.EngineSelectOutboundRequest, api.EngineSelectOutboundReply](c, ctx, api.MethodEngineSelectOutbound, req)
}

func (c *Client) EngineURLTest(ctx context.Context, req api.EngineURLTestRequest) (api.EngineURLTestReply, *api.StructuredError) {
	return do[api.EngineURLTestRequest, api.EngineURLTestReply](c, ctx, api.MethodEngineURLTest, req)
}

func (c *Client) EngineCloseConnection(ctx context.Context, req api.EngineCloseConnectionRequest) (api.EngineCloseConnectionReply, *api.StructuredError) {
	return do[api.EngineCloseConnectionRequest, api.EngineCloseConnectionReply](c, ctx, api.MethodEngineCloseConnection, req)
}

func (c *Client) EngineCloseAllConnections(ctx context.Context, req api.EngineCloseAllConnectionsRequest) (api.EngineCloseAllConnectionsReply, *api.StructuredError) {
	return do[api.EngineCloseAllConnectionsRequest, api.EngineCloseAllConnectionsReply](c, ctx, api.MethodEngineCloseAllConnections, req)
}

// Platform capabilities

func (c *Client) PlatformGetCapabilities(ctx context.Context, req api.GetPlatformCapabilitiesRequest) (api.GetPlatformCapabilitiesReply, *api.StructuredError) {
	return do[api.GetPlatformCapabilitiesRequest, api.GetPlatformCapabilitiesReply](c, ctx, api.MethodPlatformGetCapabilities, req)
}

func (c *Client) PlatformGetPrivilegedProviderStatus(ctx context.Context, req api.GetPrivilegedProviderStatusRequest) (api.GetPrivilegedProviderStatusReply, *api.StructuredError) {
	return do[api.GetPrivilegedProviderStatusRequest, api.GetPrivilegedProviderStatusReply](c, ctx, api.MethodPlatformGetPrivilegedProviderStatus, req)
}

func (c *Client) PlatformPrepareFeature(ctx context.Context, req api.PrepareFeatureRequest) (api.PrepareFeatureReply, *api.StructuredError) {
	return do[api.PrepareFeatureRequest, api.PrepareFeatureReply](c, ctx, api.MethodPlatformPrepareFeature, req)
}

func (c *Client) PlatformRunRepairAction(ctx context.Context, req api.RunRepairActionRequest) (api.RunRepairActionReply, *api.StructuredError) {
	return do[api.RunRepairActionRequest, api.RunRepairActionReply](c, ctx, api.MethodPlatformRunRepairAction, req)
}

func (c *Client) PlatformGetSystemProxyStatus(ctx context.Context, req api.GetSystemProxyStatusRequest) (api.GetSystemProxyStatusReply, *api.StructuredError) {
	return do[api.GetSystemProxyStatusRequest, api.GetSystemProxyStatusReply](c, ctx, api.MethodPlatformGetSystemProxyStatus, req)
}

func (c *Client) PlatformSetSystemProxyEnabled(ctx context.Context, req api.SetSystemProxyEnabledRequest) (api.SetSystemProxyEnabledReply, *api.StructuredError) {
	return do[api.SetSystemProxyEnabledRequest, api.SetSystemProxyEnabledReply](c, ctx, api.MethodPlatformSetSystemProxyEnabled, req)
}

// generic dispatch

func do[Req any, Reply any](c *Client, ctx context.Context, method string, req Req) (Reply, *api.StructuredError) {
	return doWithTimeout[Req, Reply](c, ctx, method, req, defaultCallTimeout)
}

func doWithTimeout[Req any, Reply any](c *Client, ctx context.Context, method string, req Req, timeout time.Duration) (Reply, *api.StructuredError) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	dial := c.dial
	if dial == nil {
		dial = Dial
	}
	conn, err := dial(ctx)
	if err != nil {
		return zero[Reply](), api.NewStructuredError(api.ErrorIPCTransport, err.Error(), "ipc", true)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	payload, err := json.Marshal(req)
	if err != nil {
		return zero[Reply](), api.NewStructuredError(api.ErrorIPCTransport, err.Error(), "ipc", true)
	}
	if err := WriteFrame(conn, Request{ID: requestID(), Method: method, Params: payload}); err != nil {
		return zero[Reply](), api.NewStructuredError(api.ErrorIPCTransport, err.Error(), "ipc", true)
	}

	var resp Response
	if err := ReadFrame(conn, &resp); err != nil {
		return zero[Reply](), api.NewStructuredError(api.ErrorIPCTransport, err.Error(), "ipc", true)
	}
	if resp.Error != nil {
		return zero[Reply](), resp.Error
	}
	var reply Reply
	if err := json.Unmarshal(resp.Result, &reply); err != nil {
		return zero[Reply](), api.NewStructuredError(api.ErrorIPCTransport, err.Error(), "ipc", true)
	}
	return reply, nil
}

func zero[T any]() T {
	var t T
	return t
}

func openSubscription[Req any](c *Client, ctx context.Context, method string, req Req) (<-chan EventFrame, *api.StructuredError) {
	dial := c.dial
	if dial == nil {
		dial = Dial
	}
	conn, err := dial(ctx)
	if err != nil {
		return nil, api.NewStructuredError(api.ErrorIPCTransport, err.Error(), "ipc", true)
	}
	_ = conn.SetDeadline(time.Now().Add(defaultCallTimeout))

	payload, err := json.Marshal(req)
	if err != nil {
		conn.Close()
		return nil, api.NewStructuredError(api.ErrorIPCTransport, err.Error(), "ipc", true)
	}
	if err := WriteFrame(conn, Request{ID: requestID(), Method: method, Params: payload}); err != nil {
		conn.Close()
		return nil, api.NewStructuredError(api.ErrorIPCTransport, err.Error(), "ipc", true)
	}

	var resp Response
	if err := ReadFrame(conn, &resp); err != nil {
		conn.Close()
		return nil, api.NewStructuredError(api.ErrorIPCTransport, err.Error(), "ipc", true)
	}
	if resp.Error != nil {
		conn.Close()
		return nil, resp.Error
	}
	_ = conn.SetDeadline(time.Time{})

	events := make(chan EventFrame, 64)
	go func() {
		stopClose := context.AfterFunc(ctx, func() {
			_ = conn.Close()
		})
		defer stopClose()
		defer conn.Close()
		defer close(events)
		for {
			var event EventFrame
			if err := ReadFrame(conn, &event); err != nil {
				return
			}
			select {
			case events <- event:
			case <-ctx.Done():
				return
			}
		}
	}()
	return events, nil
}

func requestID() string {
	return "req_" + time.Now().UTC().Format("20060102150405.000000000")
}
