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

func do[Req any, Reply any](c *Client, ctx context.Context, method string, req Req) (Reply, *api.StructuredError) {
	if c.config == nil {
		return zero[Reply](), api.NewStructuredError(api.ErrorPlatformProviderUnavailable, "Privileged provider config is missing.", "provider", true)
	}
	if c.dial == nil {
		c.dial = Dial
	}
	ctx, cancel := context.WithTimeout(ctx, defaultCallTimeout)
	defer cancel()

	conn, err := c.dial(ctx, c.config.Endpoint)
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
