package capability

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/zclkkk/qkbox/internal/provideripc"
	"github.com/zclkkk/qkbox/shared/api"
)

type PrivilegedProvider interface {
	Status(ctx context.Context) api.PrivilegedProviderStatus
	PrepareFeature(ctx context.Context, feature string) (api.PrepareFeatureReply, *api.StructuredError)
	RunRepairAction(ctx context.Context, action string) (api.RunRepairActionReply, *api.StructuredError)
}

type privilegedProviderClient struct {
	clientConfigPath string
}

func NewPrivilegedProvider(stateDir string) PrivilegedProvider {
	return &privilegedProviderClient{
		clientConfigPath: provideripc.ClientConfigPath(stateDir),
	}
}

func (p *privilegedProviderClient) Status(ctx context.Context) api.PrivilegedProviderStatus {
	status := api.PrivilegedProviderStatus{}
	cfg, err := provideripc.ReadClientConfig(p.clientConfigPath)
	if err != nil {
		status.Reason = err.Error()
		if errors.Is(err, os.ErrNotExist) {
			status.Reason = "Privileged provider client config is missing."
		}
		return status
	}
	status.Installed = true
	status.Endpoint = cfg.Endpoint
	status.ExpectedVersion = cfg.ExpectedVersion

	reply, structured := provideripc.NewClient(cfg).GetStatus(ctx)
	if structured != nil {
		status.Reason = structured.Message
		if structured.Code == api.ErrorPlatformProviderAuthFailed {
			status.Reachable = true
			status.Authenticated = false
		}
		return status
	}

	status.Reachable = true
	status.Authenticated = true
	status.Version = reply.Version
	status.OwnerState = reply.OwnerState
	if cfg.ExpectedVersion != "" && reply.Version != cfg.ExpectedVersion {
		status.Reason = fmt.Sprintf("Provider version %s does not match expected version %s.", reply.Version, cfg.ExpectedVersion)
	}
	return status
}

func (p *privilegedProviderClient) PrepareFeature(ctx context.Context, feature string) (api.PrepareFeatureReply, *api.StructuredError) {
	cfg, structured := p.loadReadyClientConfig(ctx)
	if structured != nil {
		return api.PrepareFeatureReply{}, structured
	}
	return provideripc.NewClient(cfg).PrepareFeature(ctx, api.PrepareFeatureRequest{Feature: feature})
}

func (p *privilegedProviderClient) RunRepairAction(ctx context.Context, action string) (api.RunRepairActionReply, *api.StructuredError) {
	cfg, structured := p.loadReadyClientConfig(ctx)
	if structured != nil {
		return api.RunRepairActionReply{}, structured
	}
	return provideripc.NewClient(cfg).RunRepairAction(ctx, api.RunRepairActionRequest{Action: action})
}

func (p *privilegedProviderClient) loadReadyClientConfig(ctx context.Context) (*provideripc.ClientConfig, *api.StructuredError) {
	cfg, err := provideripc.ReadClientConfig(p.clientConfigPath)
	if err != nil {
		return nil, api.NewStructuredError(api.ErrorPlatformProviderUnavailable, "Privileged provider client config is missing.", "provider", true)
	}
	status := p.Status(ctx)
	if !status.Reachable {
		return nil, api.NewStructuredError(api.ErrorPlatformProviderUnavailable, status.Reason, "provider", true)
	}
	if !status.Authenticated {
		return nil, api.NewStructuredError(api.ErrorPlatformProviderAuthFailed, status.Reason, "provider", false)
	}
	if cfg.ExpectedVersion != "" && status.Version != cfg.ExpectedVersion {
		return nil, api.NewStructuredError(api.ErrorPlatformProviderVersionMismatch, status.Reason, "provider", true)
	}
	return cfg, nil
}
