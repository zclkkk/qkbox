package ipc

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"sync"

	"github.com/zclkkk/qkbox/shared/api"
)

type Handler interface {
	Hello(context.Context, api.HelloRequest) (api.HelloReply, *api.StructuredError)

	// Profile CRUD
	CreateProfile(context.Context, api.CreateProfileRequest) (api.CreateProfileReply, *api.StructuredError)
	UpdateProfileDraft(context.Context, api.UpdateProfileDraftRequest) (api.UpdateProfileDraftReply, *api.StructuredError)
	DeleteProfile(context.Context, api.DeleteProfileRequest) (api.DeleteProfileReply, *api.StructuredError)
	ListProfiles(context.Context, api.ListProfilesRequest) (api.ListProfilesReply, *api.StructuredError)
	GetProfile(context.Context, api.GetProfileRequest) (api.GetProfileReply, *api.StructuredError)

	// Data assets and subscriptions
	CreateProfileSubscription(context.Context, api.CreateProfileSubscriptionRequest) (api.CreateProfileSubscriptionReply, *api.StructuredError)
	ListProfileSubscriptions(context.Context, api.ListProfileSubscriptionsRequest) (api.ListProfileSubscriptionsReply, *api.StructuredError)
	RefreshProfileSubscription(context.Context, api.RefreshProfileSubscriptionRequest) (api.RefreshProfileSubscriptionReply, *api.StructuredError)
	DeleteProfileSubscription(context.Context, api.DeleteProfileSubscriptionRequest) (api.DeleteProfileSubscriptionReply, *api.StructuredError)
	CreateDataAsset(context.Context, api.CreateDataAssetRequest) (api.CreateDataAssetReply, *api.StructuredError)
	ListDataAssets(context.Context, api.ListDataAssetsRequest) (api.ListDataAssetsReply, *api.StructuredError)
	RefreshDataAsset(context.Context, api.RefreshDataAssetRequest) (api.RefreshDataAssetReply, *api.StructuredError)
	DeleteDataAsset(context.Context, api.DeleteDataAssetRequest) (api.DeleteDataAssetReply, *api.StructuredError)

	// Snapshot lifecycle
	ValidateProfileDraft(context.Context, api.ValidateProfileDraftRequest) (api.ValidateProfileDraftReply, *api.StructuredError)
	GetProfileDiagnostics(context.Context, api.GetProfileDiagnosticsRequest) (api.GetProfileDiagnosticsReply, *api.StructuredError)
	CreateProfileSnapshot(context.Context, api.CreateProfileSnapshotRequest) (api.CreateProfileSnapshotReply, *api.StructuredError)
	ActivateProfileSnapshot(context.Context, api.ActivateProfileSnapshotRequest) (api.ActivateProfileSnapshotReply, *api.StructuredError)
	GetActiveProfile(context.Context, api.GetActiveProfileRequest) (api.GetActiveProfileReply, *api.StructuredError)
	GetActiveSnapshot(context.Context, api.GetActiveSnapshotRequest) (api.GetActiveSnapshotReply, *api.StructuredError)
	ListSnapshots(context.Context, api.ListSnapshotsRequest) (api.ListSnapshotsReply, *api.StructuredError)
	RollbackToSnapshot(context.Context, api.RollbackToSnapshotRequest) (api.RollbackToSnapshotReply, *api.StructuredError)

	// Engine lifecycle
	EngineStart(context.Context, api.EngineStartRequest) (api.EngineStartReply, *api.StructuredError)
	EngineStop(context.Context, api.EngineStopRequest) (api.EngineStopReply, *api.StructuredError)
	EngineReload(context.Context, api.EngineReloadRequest) (api.EngineReloadReply, *api.StructuredError)
	EngineGetStatus(context.Context, api.EngineGetStatusRequest) (api.EngineGetStatusReply, *api.StructuredError)
	EngineSubscribeStatus(context.Context, api.EngineSubscribeStatusRequest) (<-chan api.RuntimeEvent, *api.StructuredError)
	EngineSubscribeLogs(context.Context, api.EngineSubscribeLogsRequest) (<-chan api.RuntimeEvent, *api.StructuredError)
	EngineSubscribeTraffic(context.Context, api.EngineSubscribeTrafficRequest) (<-chan api.RuntimeEvent, *api.StructuredError)
	EngineSubscribeConnections(context.Context, api.EngineSubscribeConnectionsRequest) (<-chan api.RuntimeEvent, *api.StructuredError)
	EngineGetRuntimeCapabilities(context.Context, api.EngineGetRuntimeCapabilitiesRequest) (api.EngineGetRuntimeCapabilitiesReply, *api.StructuredError)
	EngineListGroups(context.Context, api.EngineListGroupsRequest) (api.EngineListGroupsReply, *api.StructuredError)
	EngineSelectOutbound(context.Context, api.EngineSelectOutboundRequest) (api.EngineSelectOutboundReply, *api.StructuredError)
	EngineURLTest(context.Context, api.EngineURLTestRequest) (api.EngineURLTestReply, *api.StructuredError)
	EngineCloseConnection(context.Context, api.EngineCloseConnectionRequest) (api.EngineCloseConnectionReply, *api.StructuredError)
	EngineCloseAllConnections(context.Context, api.EngineCloseAllConnectionsRequest) (api.EngineCloseAllConnectionsReply, *api.StructuredError)

	// Platform capabilities
	PlatformGetCapabilities(context.Context, api.GetPlatformCapabilitiesRequest) (api.GetPlatformCapabilitiesReply, *api.StructuredError)
	PlatformGetPrivilegedProviderStatus(context.Context, api.GetPrivilegedProviderStatusRequest) (api.GetPrivilegedProviderStatusReply, *api.StructuredError)
	PlatformPrepareFeature(context.Context, api.PrepareFeatureRequest) (api.PrepareFeatureReply, *api.StructuredError)
	PlatformRunRepairAction(context.Context, api.RunRepairActionRequest) (api.RunRepairActionReply, *api.StructuredError)
	PlatformGetSystemProxyStatus(context.Context, api.GetSystemProxyStatusRequest) (api.GetSystemProxyStatusReply, *api.StructuredError)
	PlatformSetSystemProxyEnabled(context.Context, api.SetSystemProxyEnabledRequest) (api.SetSystemProxyEnabledReply, *api.StructuredError)
}

type Server struct {
	handler Handler
}

func NewServer(handler Handler) *Server {
	return &Server{handler: handler}
}

func (s *Server) Serve(ctx context.Context, listener net.Listener) error {
	var wg sync.WaitGroup
	defer wg.Wait()
	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.handleConn(ctx, conn)
		}()
	}
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	var req Request
	if err := ReadFrame(conn, &req); err != nil {
		writeError(conn, "", api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", true))
		return
	}

	switch req.Method {
	case api.MethodHello:
		dispatch(conn, req, s.handler.Hello, ctx)

	case api.MethodCreateProfile:
		dispatch(conn, req, s.handler.CreateProfile, ctx)
	case api.MethodUpdateProfileDraft:
		dispatch(conn, req, s.handler.UpdateProfileDraft, ctx)
	case api.MethodDeleteProfile:
		dispatch(conn, req, s.handler.DeleteProfile, ctx)
	case api.MethodListProfiles:
		dispatch(conn, req, s.handler.ListProfiles, ctx)
	case api.MethodGetProfile:
		dispatch(conn, req, s.handler.GetProfile, ctx)

	case api.MethodAssetCreateProfileSubscription:
		dispatch(conn, req, s.handler.CreateProfileSubscription, ctx)
	case api.MethodAssetListProfileSubscriptions:
		dispatch(conn, req, s.handler.ListProfileSubscriptions, ctx)
	case api.MethodAssetRefreshProfileSubscription:
		dispatch(conn, req, s.handler.RefreshProfileSubscription, ctx)
	case api.MethodAssetDeleteProfileSubscription:
		dispatch(conn, req, s.handler.DeleteProfileSubscription, ctx)
	case api.MethodAssetCreateDataAsset:
		dispatch(conn, req, s.handler.CreateDataAsset, ctx)
	case api.MethodAssetListDataAssets:
		dispatch(conn, req, s.handler.ListDataAssets, ctx)
	case api.MethodAssetRefreshDataAsset:
		dispatch(conn, req, s.handler.RefreshDataAsset, ctx)
	case api.MethodAssetDeleteDataAsset:
		dispatch(conn, req, s.handler.DeleteDataAsset, ctx)

	case api.MethodValidateProfileDraft:
		dispatch(conn, req, s.handler.ValidateProfileDraft, ctx)
	case api.MethodGetProfileDiagnostics:
		dispatch(conn, req, s.handler.GetProfileDiagnostics, ctx)
	case api.MethodCreateProfileSnapshot:
		dispatch(conn, req, s.handler.CreateProfileSnapshot, ctx)
	case api.MethodActivateProfileSnapshot:
		dispatch(conn, req, s.handler.ActivateProfileSnapshot, ctx)
	case api.MethodGetActiveProfile:
		dispatch(conn, req, s.handler.GetActiveProfile, ctx)
	case api.MethodGetActiveSnapshot:
		dispatch(conn, req, s.handler.GetActiveSnapshot, ctx)
	case api.MethodListSnapshots:
		dispatch(conn, req, s.handler.ListSnapshots, ctx)
	case api.MethodRollbackToSnapshot:
		dispatch(conn, req, s.handler.RollbackToSnapshot, ctx)

	case api.MethodEngineStart:
		dispatch(conn, req, s.handler.EngineStart, ctx)
	case api.MethodEngineStop:
		dispatch(conn, req, s.handler.EngineStop, ctx)
	case api.MethodEngineReload:
		dispatch(conn, req, s.handler.EngineReload, ctx)
	case api.MethodEngineGetStatus:
		dispatch(conn, req, s.handler.EngineGetStatus, ctx)
	case api.MethodEngineSubscribeStatus:
		serveSubscription(conn, req, s.handler.EngineSubscribeStatus, ctx)
	case api.MethodEngineSubscribeLogs:
		serveSubscription(conn, req, s.handler.EngineSubscribeLogs, ctx)
	case api.MethodEngineSubscribeTraffic:
		serveSubscription(conn, req, s.handler.EngineSubscribeTraffic, ctx)
	case api.MethodEngineSubscribeConnections:
		serveSubscription(conn, req, s.handler.EngineSubscribeConnections, ctx)
	case api.MethodEngineGetRuntimeCapabilities:
		dispatch(conn, req, s.handler.EngineGetRuntimeCapabilities, ctx)
	case api.MethodEngineListGroups:
		dispatch(conn, req, s.handler.EngineListGroups, ctx)
	case api.MethodEngineSelectOutbound:
		dispatch(conn, req, s.handler.EngineSelectOutbound, ctx)
	case api.MethodEngineURLTest:
		dispatch(conn, req, s.handler.EngineURLTest, ctx)
	case api.MethodEngineCloseConnection:
		dispatch(conn, req, s.handler.EngineCloseConnection, ctx)
	case api.MethodEngineCloseAllConnections:
		dispatch(conn, req, s.handler.EngineCloseAllConnections, ctx)

	case api.MethodPlatformGetCapabilities:
		dispatch(conn, req, s.handler.PlatformGetCapabilities, ctx)
	case api.MethodPlatformGetPrivilegedProviderStatus:
		dispatch(conn, req, s.handler.PlatformGetPrivilegedProviderStatus, ctx)
	case api.MethodPlatformPrepareFeature:
		dispatch(conn, req, s.handler.PlatformPrepareFeature, ctx)
	case api.MethodPlatformRunRepairAction:
		dispatch(conn, req, s.handler.PlatformRunRepairAction, ctx)
	case api.MethodPlatformGetSystemProxyStatus:
		dispatch(conn, req, s.handler.PlatformGetSystemProxyStatus, ctx)
	case api.MethodPlatformSetSystemProxyEnabled:
		dispatch(conn, req, s.handler.PlatformSetSystemProxyEnabled, ctx)

	default:
		writeError(conn, req.ID, api.NewStructuredError(api.ErrorIPCMethodNotFound, "Unknown IPC method.", "qkboxd", false))
	}
}

func serveSubscription[Req any](conn net.Conn, req Request, fn func(context.Context, Req) (<-chan api.RuntimeEvent, *api.StructuredError), ctx context.Context) {
	var params Req
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeError(conn, req.ID, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", true))
		return
	}

	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	events, structured := fn(subCtx, params)
	if structured != nil {
		writeError(conn, req.ID, structured)
		return
	}
	writeResult(conn, req.ID, api.SubscriptionAck{})

	go func() {
		var discard Request
		_ = ReadFrame(conn, &discard)
		cancel()
	}()

	for {
		select {
		case <-subCtx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			if err := writeEvent(conn, req.ID, event); err != nil {
				return
			}
		}
	}
}

func dispatch[Req any, Reply any](conn net.Conn, req Request, fn func(context.Context, Req) (Reply, *api.StructuredError), ctx context.Context) {
	var params Req
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeError(conn, req.ID, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", true))
		return
	}
	reqCtx, cancel := requestLifetimeContext(ctx, conn)
	defer cancel()

	reply, structured := fn(reqCtx, params)
	if structured != nil {
		writeError(conn, req.ID, structured)
		return
	}
	writeResult(conn, req.ID, reply)
}

func requestLifetimeContext(ctx context.Context, conn net.Conn) (context.Context, context.CancelFunc) {
	reqCtx, cancel := context.WithCancel(ctx)
	go func() {
		var discard Request
		_ = ReadFrame(conn, &discard)
		cancel()
	}()
	return reqCtx, cancel
}

func writeResult(conn net.Conn, id string, value interface{}) {
	payload, err := json.Marshal(value)
	if err != nil {
		writeError(conn, id, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false))
		return
	}
	_ = WriteFrame(conn, Response{ID: id, Result: payload})
}

func writeError(conn net.Conn, id string, err *api.StructuredError) {
	_ = WriteFrame(conn, Response{ID: id, Error: err})
}

func writeEvent(conn net.Conn, id string, event api.RuntimeEvent) error {
	frame := EventFrame{ID: id, Event: event.Event, Error: event.Error}
	if event.Data != nil {
		payload, err := json.Marshal(event.Data)
		if err != nil {
			frame.Error = api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", false)
		} else {
			frame.Data = payload
		}
	}
	return WriteFrame(conn, frame)
}
