package ipc

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"sync"

	"github.com/zclkkk/qkbox/internal/ipcframework"
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

	// Diagnostics and recovery
	DiagnosticsGetReport(context.Context, api.GetDiagnosticsReportRequest) (api.GetDiagnosticsReportReply, *api.StructuredError)
	DiagnosticsCreateDebugBundle(context.Context, api.CreateDebugBundleRequest) (api.CreateDebugBundleReply, *api.StructuredError)

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

	// Window session
	WindowAttach(context.Context, api.WindowAttachRequest) (<-chan api.RuntimeEvent, *api.StructuredError)

	// Platform capabilities
	PlatformGetCapabilities(context.Context, api.GetPlatformCapabilitiesRequest) (api.GetPlatformCapabilitiesReply, *api.StructuredError)
	PlatformGetPrivilegedProviderStatus(context.Context, api.GetPrivilegedProviderStatusRequest) (api.GetPrivilegedProviderStatusReply, *api.StructuredError)
	PlatformGetNetworkExtensionStatus(context.Context, api.GetNetworkExtensionStatusRequest) (api.GetNetworkExtensionStatusReply, *api.StructuredError)
	PlatformPrepareFeature(context.Context, api.PrepareFeatureRequest) (api.PrepareFeatureReply, *api.StructuredError)
	PlatformRunRepairAction(context.Context, api.RunRepairActionRequest) (api.RunRepairActionReply, *api.StructuredError)
	PlatformGetSystemProxyStatus(context.Context, api.GetSystemProxyStatusRequest) (api.GetSystemProxyStatusReply, *api.StructuredError)
	PlatformSetSystemProxyEnabled(context.Context, api.SetSystemProxyEnabledRequest) (api.SetSystemProxyEnabledReply, *api.StructuredError)
}

type Server struct {
	registry *ipcframework.Registry
}

func NewServer(handler Handler) *Server {
	return &Server{registry: newRegistry(handler)}
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

	if handler, ok := s.registry.Method(req.Method); ok {
		dispatch(conn, req, handler, ctx)
		return
	}
	if handler, ok := s.registry.Subscription(req.Method); ok {
		serveSubscription(conn, req, handler, ctx)
		return
	}

	writeError(conn, req.ID, api.NewStructuredError(api.ErrorIPCMethodNotFound, "Unknown IPC method.", "qkboxd", false))
}

func serveSubscription(conn net.Conn, req Request, handler ipcframework.SubscriptionHandler, ctx context.Context) {
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	events, ack, structured := handler(subCtx, req.Params)
	if structured != nil {
		writeError(conn, req.ID, structured)
		return
	}
	writeResult(conn, req.ID, ack)

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

func dispatch(conn net.Conn, req Request, handler ipcframework.MethodHandler, ctx context.Context) {
	reqCtx, cancel := requestLifetimeContext(ctx, conn)
	defer cancel()

	reply, structured := handler(reqCtx, req.Params)
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
