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
	"github.com/zclkkk/qkbox/shared/api"
)

type providerHandler struct {
	version string
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

	handler := &providerHandler{version: cfg.Version}
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
		Version: h.version,
	}, nil
}

func (h *providerHandler) PrepareFeature(_ context.Context, req api.PrepareFeatureRequest) (api.PrepareFeatureReply, *api.StructuredError) {
	switch req.Feature {
	case api.CapabilityBackgroundService:
		return api.PrepareFeatureReply{Feature: req.Feature, State: api.CapabilityAvailable}, nil
	case api.CapabilityTunMode, api.CapabilityDNSHijack:
		return api.PrepareFeatureReply{
			Feature: req.Feature,
			State:   api.CapabilityUnavailable,
			Reason:  "Privileged network mutation is not available in this provider build.",
		}, nil
	default:
		return api.PrepareFeatureReply{}, api.NewStructuredError(api.ErrorPlatformFeatureUnsupported, "Feature is not supported by the privileged provider.", "provider", true)
	}
}

func (h *providerHandler) RunRepairAction(_ context.Context, req api.RunRepairActionRequest) (api.RunRepairActionReply, *api.StructuredError) {
	return api.RunRepairActionReply{}, api.NewStructuredError(api.ErrorPlatformRepairActionNotFound, "Repair action is not allowlisted.", "provider", true)
}
