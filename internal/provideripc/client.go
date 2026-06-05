package provideripc

import (
	"context"
	"encoding/json"
	"net"
	"time"

	"github.com/zclkkk/qkbox/shared/api"
)

const defaultCallTimeout = 5 * time.Second

type Client struct {
	config *ClientConfig
	dial   func(context.Context, string) (net.Conn, error)
}

func NewClient(config *ClientConfig) *Client {
	return &Client{config: config, dial: Dial}
}

func (c *Client) GetStatus(ctx context.Context) (StatusReply, *api.StructuredError) {
	return do[struct{}, StatusReply](c, ctx, MethodGetStatus, struct{}{})
}

func (c *Client) PrepareFeature(ctx context.Context, req api.PrepareFeatureRequest) (api.PrepareFeatureReply, *api.StructuredError) {
	return do[api.PrepareFeatureRequest, api.PrepareFeatureReply](c, ctx, MethodPrepareFeature, req)
}

func (c *Client) RunRepairAction(ctx context.Context, req api.RunRepairActionRequest) (api.RunRepairActionReply, *api.StructuredError) {
	return do[api.RunRepairActionRequest, api.RunRepairActionReply](c, ctx, MethodRunRepairAction, req)
}

func (c *Client) RuntimeStart(ctx context.Context, req RuntimeStartRequest) (RuntimeStartReply, *api.StructuredError) {
	return do[RuntimeStartRequest, RuntimeStartReply](c, ctx, MethodRuntimeStart, req)
}

func (c *Client) RuntimeStop(ctx context.Context, req RuntimeStopRequest) (RuntimeStopReply, *api.StructuredError) {
	return do[RuntimeStopRequest, RuntimeStopReply](c, ctx, MethodRuntimeStop, req)
}

func (c *Client) RuntimeHeartbeat(ctx context.Context, req RuntimeHeartbeatRequest) (RuntimeHeartbeatReply, *api.StructuredError) {
	return do[RuntimeHeartbeatRequest, RuntimeHeartbeatReply](c, ctx, MethodRuntimeHeartbeat, req)
}

func (c *Client) RuntimeGetStatus(ctx context.Context, req RuntimeGetStatusRequest) (RuntimeGetStatusReply, *api.StructuredError) {
	return do[RuntimeGetStatusRequest, RuntimeGetStatusReply](c, ctx, MethodRuntimeGetStatus, req)
}

func (c *Client) RuntimeGetRuntimeCapabilities(ctx context.Context, req RuntimeGetRuntimeCapabilitiesRequest) (RuntimeGetRuntimeCapabilitiesReply, *api.StructuredError) {
	return do[RuntimeGetRuntimeCapabilitiesRequest, RuntimeGetRuntimeCapabilitiesReply](c, ctx, MethodRuntimeGetRuntimeCapabilities, req)
}

func (c *Client) RuntimeGetTraffic(ctx context.Context, req RuntimeGetTrafficRequest) (RuntimeGetTrafficReply, *api.StructuredError) {
	return do[RuntimeGetTrafficRequest, RuntimeGetTrafficReply](c, ctx, MethodRuntimeGetTraffic, req)
}

func (c *Client) RuntimeGetConnections(ctx context.Context, req RuntimeGetConnectionsRequest) (RuntimeGetConnectionsReply, *api.StructuredError) {
	return do[RuntimeGetConnectionsRequest, RuntimeGetConnectionsReply](c, ctx, MethodRuntimeGetConnections, req)
}

func (c *Client) RuntimeListGroups(ctx context.Context, req RuntimeListGroupsRequest) (RuntimeListGroupsReply, *api.StructuredError) {
	return do[RuntimeListGroupsRequest, RuntimeListGroupsReply](c, ctx, MethodRuntimeListGroups, req)
}

func (c *Client) RuntimeSelectOutbound(ctx context.Context, req RuntimeSelectOutboundRequest) (RuntimeSelectOutboundReply, *api.StructuredError) {
	return do[RuntimeSelectOutboundRequest, RuntimeSelectOutboundReply](c, ctx, MethodRuntimeSelectOutbound, req)
}

func (c *Client) RuntimeURLTest(ctx context.Context, req RuntimeURLTestRequest) (RuntimeURLTestReply, *api.StructuredError) {
	return do[RuntimeURLTestRequest, RuntimeURLTestReply](c, ctx, MethodRuntimeURLTest, req)
}

func (c *Client) RuntimeCloseConnection(ctx context.Context, req RuntimeCloseConnectionRequest) (RuntimeCloseConnectionReply, *api.StructuredError) {
	return do[RuntimeCloseConnectionRequest, RuntimeCloseConnectionReply](c, ctx, MethodRuntimeCloseConnection, req)
}

func (c *Client) RuntimeCloseAllConnections(ctx context.Context, req RuntimeCloseAllConnectionsRequest) (RuntimeCloseAllConnectionsReply, *api.StructuredError) {
	return do[RuntimeCloseAllConnectionsRequest, RuntimeCloseAllConnectionsReply](c, ctx, MethodRuntimeCloseAllConnections, req)
}

func (c *Client) RuntimeListenerInfo(ctx context.Context, req RuntimeListenerInfoRequest) (RuntimeListenerInfoReply, *api.StructuredError) {
	return do[RuntimeListenerInfoRequest, RuntimeListenerInfoReply](c, ctx, MethodRuntimeListenerInfo, req)
}

func (c *Client) RuntimeSubscribeEvents(ctx context.Context, req RuntimeSubscribeEventsRequest) (<-chan EventFrame, *api.StructuredError) {
	return openSubscription(c, ctx, MethodRuntimeSubscribeEvents, req)
}

func do[Req any, Reply any](c *Client, ctx context.Context, method string, req Req) (Reply, *api.StructuredError) {
	if c.config == nil {
		return zero[Reply](), api.NewStructuredError(api.ErrorPlatformProviderUnavailable, "Privileged provider config is missing.", "provider", true)
	}
	ctx, cancel := context.WithTimeout(ctx, defaultCallTimeout)
	defer cancel()

	dial := c.dial
	if dial == nil {
		dial = Dial
	}
	conn, err := dial(ctx, c.config.Endpoint)
	if err != nil {
		return zero[Reply](), api.NewStructuredError(api.ErrorPlatformProviderUnavailable, err.Error(), "provider", true)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(defaultCallTimeout))

	if structured := authenticate(conn, c.config.Token); structured != nil {
		return zero[Reply](), structured
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return zero[Reply](), api.NewStructuredError(api.ErrorIPCTransport, err.Error(), "provider", true)
	}
	if err := WriteFrame(conn, Request{ID: requestID(), Method: method, Params: payload}); err != nil {
		return zero[Reply](), api.NewStructuredError(api.ErrorIPCTransport, err.Error(), "provider", true)
	}

	var resp Response
	if err := ReadFrame(conn, &resp); err != nil {
		return zero[Reply](), api.NewStructuredError(api.ErrorIPCTransport, err.Error(), "provider", true)
	}
	if resp.Error != nil {
		return zero[Reply](), resp.Error
	}

	var reply Reply
	if err := json.Unmarshal(resp.Result, &reply); err != nil {
		return zero[Reply](), api.NewStructuredError(api.ErrorIPCTransport, err.Error(), "provider", true)
	}
	return reply, nil
}

func authenticate(conn net.Conn, token string) *api.StructuredError {
	payload, err := json.Marshal(AuthRequest{Token: token})
	if err != nil {
		return api.NewStructuredError(api.ErrorIPCTransport, err.Error(), "provider", true)
	}
	if err := WriteFrame(conn, Request{ID: requestID(), Method: MethodAuth, Params: payload}); err != nil {
		return api.NewStructuredError(api.ErrorIPCTransport, err.Error(), "provider", true)
	}
	var resp Response
	if err := ReadFrame(conn, &resp); err != nil {
		return api.NewStructuredError(api.ErrorIPCTransport, err.Error(), "provider", true)
	}
	return resp.Error
}

func zero[T any]() T {
	var v T
	return v
}

func requestID() string {
	return time.Now().UTC().Format("20060102150405.000000000")
}

func openSubscription[Req any](c *Client, ctx context.Context, method string, req Req) (<-chan EventFrame, *api.StructuredError) {
	if c.config == nil {
		return nil, api.NewStructuredError(api.ErrorPlatformProviderUnavailable, "Privileged provider config is missing.", "provider", true)
	}
	setupCtx, cancelSetup := context.WithTimeout(ctx, defaultCallTimeout)
	defer cancelSetup()

	dial := c.dial
	if dial == nil {
		dial = Dial
	}
	conn, err := dial(setupCtx, c.config.Endpoint)
	if err != nil {
		return nil, api.NewStructuredError(api.ErrorPlatformProviderUnavailable, err.Error(), "provider", true)
	}
	_ = conn.SetDeadline(time.Now().Add(defaultCallTimeout))

	if structured := authenticate(conn, c.config.Token); structured != nil {
		conn.Close()
		return nil, structured
	}

	payload, err := json.Marshal(req)
	if err != nil {
		conn.Close()
		return nil, api.NewStructuredError(api.ErrorIPCTransport, err.Error(), "provider", true)
	}
	if err := WriteFrame(conn, Request{ID: requestID(), Method: method, Params: payload}); err != nil {
		conn.Close()
		return nil, api.NewStructuredError(api.ErrorIPCTransport, err.Error(), "provider", true)
	}

	var resp Response
	if err := ReadFrame(conn, &resp); err != nil {
		conn.Close()
		return nil, api.NewStructuredError(api.ErrorIPCTransport, err.Error(), "provider", true)
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
