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

// generic dispatch

func do[Req any, Reply any](c *Client, ctx context.Context, method string, req Req) (Reply, *api.StructuredError) {
	if c.dial == nil {
		c.dial = Dial
	}
	ctx, cancel := context.WithTimeout(ctx, defaultCallTimeout)
	defer cancel()

	conn, err := c.dial(ctx)
	if err != nil {
		return zero[Reply](), api.NewStructuredError(api.ErrorIPCTransport, err.Error(), "ipc", true)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(defaultCallTimeout))

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
	if c.dial == nil {
		c.dial = Dial
	}
	conn, err := c.dial(ctx)
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
