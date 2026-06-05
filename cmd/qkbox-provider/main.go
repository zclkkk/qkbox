package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/zclkkk/qkbox/internal/provideripc"
	"github.com/zclkkk/qkbox/internal/providerruntime"
	"github.com/zclkkk/qkbox/shared/api"
)

type providerHandler struct {
	version string
	runtime *providerruntime.Controller
}

func main() {
	versionFlag := flag.Bool("version", false, "print qkbox provider version")
	endpointFlag := flag.Bool("endpoint", false, "print qkbox provider endpoint")
	initConfig := flag.Bool("init-config", false, "create provider/client config pair")
	serve := flag.Bool("serve", false, "serve qkbox provider IPC")
	stateDir := flag.String("state-dir", "", "qkbox state directory")
	clientConfigPath := flag.String("client-config", "", "client config path")
	serverConfigPath := flag.String("server-config", "", "server config path")
	endpoint := flag.String("provider-endpoint", "", "provider IPC endpoint")
	flag.Parse()

	dir, err := resolveStateDir(*stateDir)
	if err != nil {
		log.Fatal(err)
	}
	clientPath := *clientConfigPath
	if clientPath == "" {
		clientPath = provideripc.ClientConfigPath(dir)
	}
	serverPath := *serverConfigPath
	if serverPath == "" {
		serverPath = provideripc.ServerConfigPath(dir)
	}

	if *versionFlag {
		fmt.Println(provideripc.DefaultProviderVersion)
		return
	}
	if *endpointFlag {
		fmt.Println(resolveEndpoint(*endpoint, serverPath))
		return
	}
	if *initConfig {
		if _, _, err := provideripc.WriteConfigPair(clientPath, serverPath, *endpoint); err != nil {
			log.Fatal(err)
		}
		return
	}
	if !*serve {
		flag.Usage()
		os.Exit(2)
	}

	cfg, err := provideripc.ReadServerConfig(serverPath)
	if err != nil {
		log.Fatal(err)
	}
	listener, err := provideripc.Listen(cfg.Endpoint)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	available, reason := machineNetworkAvailable()
	handler := &providerHandler{
		version: cfg.Version,
		runtime: providerruntime.NewController(dir, available, reason),
	}
	defer handler.runtime.Close()
	if err := provideripc.NewServer(cfg.Token, handler).Serve(ctx, listener); err != nil {
		log.Fatal(err)
	}
}

func resolveStateDir(path string) (string, error) {
	if path != "" {
		return path, os.MkdirAll(path, 0o700)
	}
	return provideripc.DefaultStateDir()
}

func resolveEndpoint(flagValue, serverPath string) string {
	if flagValue != "" {
		return flagValue
	}
	if cfg, err := provideripc.ReadServerConfig(serverPath); err == nil && cfg.Endpoint != "" {
		return cfg.Endpoint
	}
	return provideripc.DefaultEndpoint()
}

func (h *providerHandler) GetStatus(context.Context, struct{}) (provideripc.StatusReply, *api.StructuredError) {
	return provideripc.StatusReply{
		Version:      h.version,
		OwnerState:   h.runtime.OwnerState(),
		Capabilities: h.runtime.Capabilities(),
	}, nil
}

func (h *providerHandler) PrepareFeature(_ context.Context, req api.PrepareFeatureRequest) (api.PrepareFeatureReply, *api.StructuredError) {
	switch req.Feature {
	case api.CapabilityBackgroundService:
		return api.PrepareFeatureReply{Feature: req.Feature, State: api.CapabilityAvailable}, nil
	case api.CapabilityTunMode, api.CapabilityDNSHijack:
		for _, cap := range h.runtime.Capabilities() {
			if cap.Name == req.Feature {
				return api.PrepareFeatureReply{Feature: req.Feature, State: cap.State, Reason: cap.Reason}, nil
			}
		}
		return api.PrepareFeatureReply{Feature: req.Feature, State: api.CapabilityUnavailable, Reason: "Machine network capability is unavailable."}, nil
	default:
		return api.PrepareFeatureReply{}, api.NewStructuredError(api.ErrorPlatformFeatureUnsupported, "Feature is not supported by the privileged provider.", "provider", true)
	}
}

func (h *providerHandler) RunRepairAction(ctx context.Context, req api.RunRepairActionRequest) (api.RunRepairActionReply, *api.StructuredError) {
	return h.runtime.RunRepairAction(ctx, req)
}

func (h *providerHandler) RuntimeStart(ctx context.Context, req provideripc.RuntimeStartRequest) (provideripc.RuntimeStartReply, *api.StructuredError) {
	return h.runtime.RuntimeStart(ctx, req)
}

func (h *providerHandler) RuntimeStop(ctx context.Context, req provideripc.RuntimeStopRequest) (provideripc.RuntimeStopReply, *api.StructuredError) {
	return h.runtime.RuntimeStop(ctx, req)
}

func (h *providerHandler) RuntimeHeartbeat(ctx context.Context, req provideripc.RuntimeHeartbeatRequest) (provideripc.RuntimeHeartbeatReply, *api.StructuredError) {
	return h.runtime.RuntimeHeartbeat(ctx, req)
}

func (h *providerHandler) RuntimeGetStatus(ctx context.Context, req provideripc.RuntimeGetStatusRequest) (provideripc.RuntimeGetStatusReply, *api.StructuredError) {
	return h.runtime.RuntimeGetStatus(ctx, req)
}

func (h *providerHandler) RuntimeGetRuntimeCapabilities(ctx context.Context, req provideripc.RuntimeGetRuntimeCapabilitiesRequest) (provideripc.RuntimeGetRuntimeCapabilitiesReply, *api.StructuredError) {
	return h.runtime.RuntimeGetRuntimeCapabilities(ctx, req)
}

func (h *providerHandler) RuntimeGetTraffic(ctx context.Context, req provideripc.RuntimeGetTrafficRequest) (provideripc.RuntimeGetTrafficReply, *api.StructuredError) {
	return h.runtime.RuntimeGetTraffic(ctx, req)
}

func (h *providerHandler) RuntimeGetConnections(ctx context.Context, req provideripc.RuntimeGetConnectionsRequest) (provideripc.RuntimeGetConnectionsReply, *api.StructuredError) {
	return h.runtime.RuntimeGetConnections(ctx, req)
}

func (h *providerHandler) RuntimeListGroups(ctx context.Context, req provideripc.RuntimeListGroupsRequest) (provideripc.RuntimeListGroupsReply, *api.StructuredError) {
	return h.runtime.RuntimeListGroups(ctx, req)
}

func (h *providerHandler) RuntimeSelectOutbound(ctx context.Context, req provideripc.RuntimeSelectOutboundRequest) (provideripc.RuntimeSelectOutboundReply, *api.StructuredError) {
	return h.runtime.RuntimeSelectOutbound(ctx, req)
}

func (h *providerHandler) RuntimeURLTest(ctx context.Context, req provideripc.RuntimeURLTestRequest) (provideripc.RuntimeURLTestReply, *api.StructuredError) {
	return h.runtime.RuntimeURLTest(ctx, req)
}

func (h *providerHandler) RuntimeCloseConnection(ctx context.Context, req provideripc.RuntimeCloseConnectionRequest) (provideripc.RuntimeCloseConnectionReply, *api.StructuredError) {
	return h.runtime.RuntimeCloseConnection(ctx, req)
}

func (h *providerHandler) RuntimeCloseAllConnections(ctx context.Context, req provideripc.RuntimeCloseAllConnectionsRequest) (provideripc.RuntimeCloseAllConnectionsReply, *api.StructuredError) {
	return h.runtime.RuntimeCloseAllConnections(ctx, req)
}

func (h *providerHandler) RuntimeListenerInfo(ctx context.Context, req provideripc.RuntimeListenerInfoRequest) (provideripc.RuntimeListenerInfoReply, *api.StructuredError) {
	return h.runtime.RuntimeListenerInfo(ctx, req)
}

func (h *providerHandler) RuntimeSubscribeEvents(ctx context.Context, req provideripc.RuntimeSubscribeEventsRequest) (<-chan api.RuntimeEvent, *api.StructuredError) {
	return h.runtime.RuntimeSubscribeEvents(ctx, req)
}
