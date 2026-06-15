package qkboxd

import (
	"context"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/zclkkk/qkbox/internal/assetcache"
	"github.com/zclkkk/qkbox/internal/persistence"
	"github.com/zclkkk/qkbox/platform/capability"
	"github.com/zclkkk/qkbox/shared/api"
	"github.com/zclkkk/qkbox/shared/model"
)

type Service struct {
	*ProfileService
	*AssetService
	*RuntimeService
	*PlatformService
	*DiagnosticsService

	db     *persistence.DB
	engine *EngineController

	// Window session tracking
	windowMu   sync.Mutex
	windowSess *WindowSession
}

const privilegedCapabilityProbeTimeout = 500 * time.Millisecond
const remoteProfileFetchLimit = 16 * 1024 * 1024
const dataAssetFetchLimit = 128 * 1024 * 1024
const remoteFetchTimeout = 5 * time.Minute

func NewService(runtimeCtx context.Context, db *persistence.DB, proxy capability.SystemProxyProvider, privileged capability.PrivilegedProvider) *Service {
	return NewServiceWithNetworkExtension(runtimeCtx, db, proxy, privileged, nil)
}

func NewServiceWithNetworkExtension(runtimeCtx context.Context, db *persistence.DB, proxy capability.SystemProxyProvider, privileged capability.PrivilegedProvider, extension capability.NetworkExtensionRuntime) *Service {
	events := NewRuntimeEventHub()
	engine := NewEngineController(runtimeCtx, events)
	engine.runtimeOwnerFactory = newRuntimeOwnerFactoryWithNetworkExtension(events, privileged, extension, newRuntimeSessionID())

	opMu := &sync.Mutex{}
	assetStore := assetcache.NewStore(db.StateDir())
	httpClient := &http.Client{Timeout: remoteFetchTimeout}

	platform := &PlatformService{db: db, engine: engine, proxy: proxy, privileged: privileged, extension: extension, opMu: opMu}
	profile := &ProfileService{db: db}
	asset := &AssetService{db: db, httpClient: httpClient, assetStore: assetStore}
	runtimeService := &RuntimeService{db: db, engine: engine, events: events, platform: platform, opMu: opMu}
	diagnostics := &DiagnosticsService{db: db, engine: engine, platform: platform, privileged: privileged, extension: extension, assetStore: assetStore}

	return &Service{
		ProfileService:     profile,
		AssetService:       asset,
		RuntimeService:     runtimeService,
		PlatformService:    platform,
		DiagnosticsService: diagnostics,
		db:                 db,
		engine:             engine,
	}
}

// EngineStateString returns the current engine state as a string.
func (s *Service) EngineStateString() string {
	return string(s.engine.GetStatus().State)
}

func (s *Service) Close() error {
	err := s.engine.Shutdown()
	s.PlatformService.bestEffortProxyRestore()
	return err
}

func (s *Service) Hello(ctx context.Context, req api.HelloRequest) (api.HelloReply, *api.StructuredError) {
	if req.ClientAPIVersion != api.APIVersion {
		return api.HelloReply{}, api.VersionUnsupported(req.ClientAPIVersion)
	}
	return api.HelloReply{
		APIVersion:             api.APIVersion,
		MinSupportedAPIVersion: api.MinSupportedAPIVersion,
		SchemaRevision:         api.SchemaRevision,
		AppVersion:             api.AppVersion,
		QKBoxDVersion:          api.QKBoxDVersion,
		Platform: model.Platform{
			OS:   runtime.GOOS,
			Arch: runtime.GOARCH,
		},
		RuntimeCapabilities:  s.engine.RuntimeCapabilities(),
		PlatformCapabilities: s.PlatformService.platformCapabilities(ctx),
	}, nil
}

func qkboxdInternalError(err error) *api.StructuredError {
	return qkboxdInternalErrorMessage(err.Error())
}

func qkboxdInternalErrorMessage(message string) *api.StructuredError {
	return api.NewStructuredError(api.ErrorInternal, message, "qkboxd", false)
}
