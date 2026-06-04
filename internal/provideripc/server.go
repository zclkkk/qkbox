package provideripc

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/zclkkk/qkbox/shared/api"
)

const defaultServerIOTimeout = 5 * time.Second

type Handler interface {
	GetStatus(context.Context, struct{}) (StatusReply, *api.StructuredError)
	PrepareFeature(context.Context, api.PrepareFeatureRequest) (api.PrepareFeatureReply, *api.StructuredError)
	RunRepairAction(context.Context, api.RunRepairActionRequest) (api.RunRepairActionReply, *api.StructuredError)
}

type Server struct {
	token     string
	handler   Handler
	ioTimeout time.Duration
}

func NewServer(token string, handler Handler) *Server {
	return &Server{token: token, handler: handler, ioTimeout: defaultServerIOTimeout}
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

	switch req.Method {
	case MethodGetStatus:
		dispatch(conn, req, s.handler.GetStatus, ctx)
	case MethodPrepareFeature:
		dispatch(conn, req, s.handler.PrepareFeature, ctx)
	case MethodRunRepairAction:
		dispatch(conn, req, s.handler.RunRepairAction, ctx)
	default:
		writeError(conn, req.ID, api.NewStructuredError(api.ErrorIPCMethodNotFound, "Unknown provider method.", "provider", false))
	}
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
	if auth.Token == "" || subtle.ConstantTimeCompare([]byte(auth.Token), []byte(s.token)) != 1 {
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

func dispatch[Req any, Reply any](conn net.Conn, req Request, fn func(context.Context, Req) (Reply, *api.StructuredError), ctx context.Context) {
	var params Req
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeError(conn, req.ID, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "provider", true))
		return
	}
	reply, structured := fn(ctx, params)
	if structured != nil {
		writeError(conn, req.ID, structured)
		return
	}
	writeResult(conn, req.ID, reply)
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
