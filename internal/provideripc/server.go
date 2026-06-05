package provideripc

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/zclkkk/qkbox/internal/ipcframework"
	"github.com/zclkkk/qkbox/shared/api"
)

const defaultServerIOTimeout = 5 * time.Second

type Handler interface {
	GetStatus(context.Context, struct{}) (StatusReply, *api.StructuredError)
	PrepareFeature(context.Context, api.PrepareFeatureRequest) (api.PrepareFeatureReply, *api.StructuredError)
	RunRepairAction(context.Context, api.RunRepairActionRequest) (api.RunRepairActionReply, *api.StructuredError)
	RuntimeStart(context.Context, RuntimeStartRequest) (RuntimeStartReply, *api.StructuredError)
	RuntimeStop(context.Context, RuntimeStopRequest) (RuntimeStopReply, *api.StructuredError)
	RuntimeHeartbeat(context.Context, RuntimeHeartbeatRequest) (RuntimeHeartbeatReply, *api.StructuredError)
	RuntimeGetStatus(context.Context, RuntimeGetStatusRequest) (RuntimeGetStatusReply, *api.StructuredError)
	RuntimeGetRuntimeCapabilities(context.Context, RuntimeGetRuntimeCapabilitiesRequest) (RuntimeGetRuntimeCapabilitiesReply, *api.StructuredError)
	RuntimeGetTraffic(context.Context, RuntimeGetTrafficRequest) (RuntimeGetTrafficReply, *api.StructuredError)
	RuntimeGetConnections(context.Context, RuntimeGetConnectionsRequest) (RuntimeGetConnectionsReply, *api.StructuredError)
	RuntimeListGroups(context.Context, RuntimeListGroupsRequest) (RuntimeListGroupsReply, *api.StructuredError)
	RuntimeSelectOutbound(context.Context, RuntimeSelectOutboundRequest) (RuntimeSelectOutboundReply, *api.StructuredError)
	RuntimeURLTest(context.Context, RuntimeURLTestRequest) (RuntimeURLTestReply, *api.StructuredError)
	RuntimeCloseConnection(context.Context, RuntimeCloseConnectionRequest) (RuntimeCloseConnectionReply, *api.StructuredError)
	RuntimeCloseAllConnections(context.Context, RuntimeCloseAllConnectionsRequest) (RuntimeCloseAllConnectionsReply, *api.StructuredError)
	RuntimeListenerInfo(context.Context, RuntimeListenerInfoRequest) (RuntimeListenerInfoReply, *api.StructuredError)
	RuntimeSubscribeEvents(context.Context, RuntimeSubscribeEventsRequest) (<-chan api.RuntimeEvent, *api.StructuredError)
}

type Server struct {
	token     string
	registry  *ipcframework.Registry
	ioTimeout time.Duration
}

func NewServer(token string, handler Handler) *Server {
	return &Server{token: token, registry: newRegistry(handler), ioTimeout: defaultServerIOTimeout}
}

func (s *Server) Serve(ctx context.Context, listener net.Listener) error {
	var wg sync.WaitGroup
	var connMu sync.Mutex
	conns := map[net.Conn]struct{}{}

	addConn := func(conn net.Conn) {
		connMu.Lock()
		conns[conn] = struct{}{}
		connMu.Unlock()
	}
	removeConn := func(conn net.Conn) {
		connMu.Lock()
		delete(conns, conn)
		connMu.Unlock()
	}
	closeListenerAndConns := func() {
		_ = listener.Close()
		connMu.Lock()
		for conn := range conns {
			_ = conn.Close()
		}
		connMu.Unlock()
	}
	done := make(chan struct{})
	defer func() {
		closeListenerAndConns()
		close(done)
		wg.Wait()
	}()
	go func() {
		select {
		case <-ctx.Done():
			closeListenerAndConns()
		case <-done:
		}
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		addConn(conn)
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer removeConn(conn)
			s.handleConn(ctx, conn)
		}()
	}
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	if !s.authenticate(conn) {
		return
	}

	var req Request
	s.setReadDeadline(conn)
	if err := ReadFrame(conn, &req); err != nil {
		writeError(conn, "", api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "provider", true))
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

	writeError(conn, req.ID, api.NewStructuredError(api.ErrorIPCMethodNotFound, "Unknown provider method.", "provider", false))
}

func (s *Server) authenticate(conn net.Conn) bool {
	var req Request
	s.setReadDeadline(conn)
	if err := ReadFrame(conn, &req); err != nil {
		writeError(conn, "", api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "provider", true))
		return false
	}
	if req.Method != MethodAuth {
		writeError(conn, req.ID, api.NewStructuredError(api.ErrorPlatformProviderAuthFailed, "Provider connection must authenticate first.", "provider", false))
		return false
	}
	var auth AuthRequest
	if err := json.Unmarshal(req.Params, &auth); err != nil {
		writeError(conn, req.ID, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "provider", true))
		return false
	}
	if !ipcframework.TokenMatches(auth.Token, s.token) {
		writeError(conn, req.ID, api.NewStructuredError(api.ErrorPlatformProviderAuthFailed, "Privileged provider authentication failed.", "provider", false))
		return false
	}
	writeResult(conn, req.ID, AuthReply{})
	return true
}

func (s *Server) setReadDeadline(conn net.Conn) {
	timeout := s.ioTimeout
	if timeout <= 0 {
		timeout = defaultServerIOTimeout
	}
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
}

func dispatch(conn net.Conn, req Request, handler ipcframework.MethodHandler, ctx context.Context) {
	reply, structured := handler(ctx, req.Params)
	if structured != nil {
		writeError(conn, req.ID, structured)
		return
	}
	writeResult(conn, req.ID, reply)
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
	_ = conn.SetReadDeadline(time.Time{})

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

func writeResult(conn net.Conn, id string, result interface{}) {
	_ = conn.SetWriteDeadline(time.Now().Add(defaultServerIOTimeout))
	payload, err := json.Marshal(result)
	if err != nil {
		writeError(conn, id, api.NewStructuredError(api.ErrorIPCTransport, err.Error(), "provider", true))
		return
	}
	_ = WriteFrame(conn, Response{ID: id, Result: payload})
}

func writeError(conn net.Conn, id string, err *api.StructuredError) {
	_ = conn.SetWriteDeadline(time.Now().Add(defaultServerIOTimeout))
	_ = WriteFrame(conn, Response{ID: id, Error: err})
}

func writeEvent(conn net.Conn, id string, event api.RuntimeEvent) error {
	_ = conn.SetWriteDeadline(time.Now().Add(defaultServerIOTimeout))
	frame := EventFrame{ID: id, Event: event.Event, Error: event.Error}
	if event.Data != nil {
		payload, err := json.Marshal(event.Data)
		if err != nil {
			frame.Error = api.NewStructuredError(api.ErrorIPCTransport, err.Error(), "provider", false)
		} else {
			frame.Data = payload
		}
	}
	return WriteFrame(conn, frame)
}
