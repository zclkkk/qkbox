package ipc

import (
	"context"
	"encoding/json"
	"errors"
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

func (c *Client) Hello(ctx context.Context, req api.HelloRequest) (api.HelloReply, *api.StructuredError) {
	var reply api.HelloReply
	err := c.call(ctx, api.MethodHello, req, &reply)
	if err == nil {
		return reply, nil
	}
	var structured *api.StructuredError
	if errors.As(err, &structured) {
		return api.HelloReply{}, structured
	}
	return api.HelloReply{}, api.NewStructuredError(api.ErrorIPCTransport, err.Error(), "ipc", true)
}

func (c *Client) call(ctx context.Context, method string, params interface{}, out interface{}) error {
	if c.dial == nil {
		c.dial = Dial
	}
	ctx, cancel := context.WithTimeout(ctx, defaultCallTimeout)
	defer cancel()

	conn, err := c.dial(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(defaultCallTimeout))

	payload, err := json.Marshal(params)
	if err != nil {
		return err
	}
	req := Request{
		ID:     requestID(),
		Method: method,
		Params: payload,
	}
	if err := WriteFrame(conn, req); err != nil {
		return err
	}

	var resp Response
	if err := ReadFrame(conn, &resp); err != nil {
		return err
	}
	if resp.Error != nil {
		return resp.Error
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(resp.Result, out)
}

func requestID() string {
	return "req_" + time.Now().UTC().Format("20060102150405.000000000")
}
