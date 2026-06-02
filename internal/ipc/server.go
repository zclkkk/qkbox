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
		var params api.HelloRequest
		if err := json.Unmarshal(req.Params, &params); err != nil {
			writeError(conn, req.ID, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", true))
			return
		}
		reply, structured := s.handler.Hello(ctx, params)
		if structured != nil {
			writeError(conn, req.ID, structured)
			return
		}
		writeResult(conn, req.ID, reply)
	default:
		writeError(conn, req.ID, api.NewStructuredError(api.ErrorIPCMethodNotFound, "Unknown IPC method.", "qkboxd", false))
	}
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
