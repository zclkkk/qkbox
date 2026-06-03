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

	// Snapshot lifecycle
	ValidateProfileDraft(context.Context, api.ValidateProfileDraftRequest) (api.ValidateProfileDraftReply, *api.StructuredError)
	GetProfileDiagnostics(context.Context, api.GetProfileDiagnosticsRequest) (api.GetProfileDiagnosticsReply, *api.StructuredError)
	CreateProfileSnapshot(context.Context, api.CreateProfileSnapshotRequest) (api.CreateProfileSnapshotReply, *api.StructuredError)
	ActivateProfileSnapshot(context.Context, api.ActivateProfileSnapshotRequest) (api.ActivateProfileSnapshotReply, *api.StructuredError)
	GetActiveProfile(context.Context, api.GetActiveProfileRequest) (api.GetActiveProfileReply, *api.StructuredError)
	GetActiveSnapshot(context.Context, api.GetActiveSnapshotRequest) (api.GetActiveSnapshotReply, *api.StructuredError)
	ListSnapshots(context.Context, api.ListSnapshotsRequest) (api.ListSnapshotsReply, *api.StructuredError)
	RollbackToSnapshot(context.Context, api.RollbackToSnapshotRequest) (api.RollbackToSnapshotReply, *api.StructuredError)
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

	default:
		writeError(conn, req.ID, api.NewStructuredError(api.ErrorIPCMethodNotFound, "Unknown IPC method.", "qkboxd", false))
	}
}

func dispatch[Req any, Reply any](conn net.Conn, req Request, fn func(context.Context, Req) (Reply, *api.StructuredError), ctx context.Context) {
	var params Req
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeError(conn, req.ID, api.NewStructuredError(api.ErrorIPCInvalidRequest, err.Error(), "qkboxd", true))
		return
	}
	reply, structured := fn(ctx, params)
	if structured != nil {
		writeError(conn, req.ID, structured)
		return
	}
	writeResult(conn, req.ID, reply)
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
