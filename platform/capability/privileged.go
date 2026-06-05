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
	RuntimeStart(ctx context.Context, req provideripc.RuntimeStartRequest) (provideripc.RuntimeStartReply, *api.StructuredError)
	RuntimeStop(ctx context.Context, req provideripc.RuntimeStopRequest) (provideripc.RuntimeStopReply, *api.StructuredError)
	RuntimeHeartbeat(ctx context.Context, req provideripc.RuntimeHeartbeatRequest) (provideripc.RuntimeHeartbeatReply, *api.StructuredError)
	RuntimeGetStatus(ctx context.Context, req provideripc.RuntimeGetStatusRequest) (provideripc.RuntimeGetStatusReply, *api.StructuredError)
	RuntimeGetRuntimeCapabilities(ctx context.Context, req provideripc.RuntimeGetRuntimeCapabilitiesRequest) (provideripc.RuntimeGetRuntimeCapabilitiesReply, *api.StructuredError)
	RuntimeGetTraffic(ctx context.Context, req provideripc.RuntimeGetTrafficRequest) (provideripc.RuntimeGetTrafficReply, *api.StructuredError)
	RuntimeGetConnections(ctx context.Context, req provideripc.RuntimeGetConnectionsRequest) (provideripc.RuntimeGetConnectionsReply, *api.StructuredError)
	RuntimeListGroups(ctx context.Context, req provideripc.RuntimeListGroupsRequest) (provideripc.RuntimeListGroupsReply, *api.StructuredError)
	RuntimeSelectOutbound(ctx context.Context, req provideripc.RuntimeSelectOutboundRequest) (provideripc.RuntimeSelectOutboundReply, *api.StructuredError)
	RuntimeURLTest(ctx context.Context, req provideripc.RuntimeURLTestRequest) (provideripc.RuntimeURLTestReply, *api.StructuredError)
	RuntimeCloseConnection(ctx context.Context, req provideripc.RuntimeCloseConnectionRequest) (provideripc.RuntimeCloseConnectionReply, *api.StructuredError)
	RuntimeCloseAllConnections(ctx context.Context, req provideripc.RuntimeCloseAllConnectionsRequest) (provideripc.RuntimeCloseAllConnectionsReply, *api.StructuredError)
	RuntimeListenerInfo(ctx context.Context, req provideripc.RuntimeListenerInfoRequest) (provideripc.RuntimeListenerInfoReply, *api.StructuredError)
	RuntimeSubscribeEvents(ctx context.Context, req provideripc.RuntimeSubscribeEventsRequest) (<-chan provideripc.EventFrame, *api.StructuredError)
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
	status.Capabilities = reply.Capabilities
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

func (p *privilegedProviderClient) RuntimeStart(ctx context.Context, req provideripc.RuntimeStartRequest) (provideripc.RuntimeStartReply, *api.StructuredError) {
	cfg, structured := p.loadReadyClientConfig(ctx)
	if structured != nil {
		return provideripc.RuntimeStartReply{}, structured
	}
	return provideripc.NewClient(cfg).RuntimeStart(ctx, req)
}

func (p *privilegedProviderClient) RuntimeStop(ctx context.Context, req provideripc.RuntimeStopRequest) (provideripc.RuntimeStopReply, *api.StructuredError) {
	cfg, structured := p.loadClientConfig()
	if structured != nil {
		return provideripc.RuntimeStopReply{}, structured
	}
	return provideripc.NewClient(cfg).RuntimeStop(ctx, req)
}

func (p *privilegedProviderClient) RuntimeHeartbeat(ctx context.Context, req provideripc.RuntimeHeartbeatRequest) (provideripc.RuntimeHeartbeatReply, *api.StructuredError) {
	cfg, structured := p.loadClientConfig()
	if structured != nil {
		return provideripc.RuntimeHeartbeatReply{}, structured
	}
	return provideripc.NewClient(cfg).RuntimeHeartbeat(ctx, req)
}

func (p *privilegedProviderClient) RuntimeGetStatus(ctx context.Context, req provideripc.RuntimeGetStatusRequest) (provideripc.RuntimeGetStatusReply, *api.StructuredError) {
	cfg, structured := p.loadClientConfig()
	if structured != nil {
		return provideripc.RuntimeGetStatusReply{}, structured
	}
	return provideripc.NewClient(cfg).RuntimeGetStatus(ctx, req)
}

func (p *privilegedProviderClient) RuntimeGetRuntimeCapabilities(ctx context.Context, req provideripc.RuntimeGetRuntimeCapabilitiesRequest) (provideripc.RuntimeGetRuntimeCapabilitiesReply, *api.StructuredError) {
	cfg, structured := p.loadClientConfig()
	if structured != nil {
		return provideripc.RuntimeGetRuntimeCapabilitiesReply{}, structured
	}
	return provideripc.NewClient(cfg).RuntimeGetRuntimeCapabilities(ctx, req)
}

func (p *privilegedProviderClient) RuntimeGetTraffic(ctx context.Context, req provideripc.RuntimeGetTrafficRequest) (provideripc.RuntimeGetTrafficReply, *api.StructuredError) {
	cfg, structured := p.loadClientConfig()
	if structured != nil {
		return provideripc.RuntimeGetTrafficReply{}, structured
	}
	return provideripc.NewClient(cfg).RuntimeGetTraffic(ctx, req)
}

func (p *privilegedProviderClient) RuntimeGetConnections(ctx context.Context, req provideripc.RuntimeGetConnectionsRequest) (provideripc.RuntimeGetConnectionsReply, *api.StructuredError) {
	cfg, structured := p.loadClientConfig()
	if structured != nil {
		return provideripc.RuntimeGetConnectionsReply{}, structured
	}
	return provideripc.NewClient(cfg).RuntimeGetConnections(ctx, req)
}

func (p *privilegedProviderClient) RuntimeListGroups(ctx context.Context, req provideripc.RuntimeListGroupsRequest) (provideripc.RuntimeListGroupsReply, *api.StructuredError) {
	cfg, structured := p.loadClientConfig()
	if structured != nil {
		return provideripc.RuntimeListGroupsReply{}, structured
	}
	return provideripc.NewClient(cfg).RuntimeListGroups(ctx, req)
}

func (p *privilegedProviderClient) RuntimeSelectOutbound(ctx context.Context, req provideripc.RuntimeSelectOutboundRequest) (provideripc.RuntimeSelectOutboundReply, *api.StructuredError) {
	cfg, structured := p.loadClientConfig()
	if structured != nil {
		return provideripc.RuntimeSelectOutboundReply{}, structured
	}
	return provideripc.NewClient(cfg).RuntimeSelectOutbound(ctx, req)
}

func (p *privilegedProviderClient) RuntimeURLTest(ctx context.Context, req provideripc.RuntimeURLTestRequest) (provideripc.RuntimeURLTestReply, *api.StructuredError) {
	cfg, structured := p.loadClientConfig()
	if structured != nil {
		return provideripc.RuntimeURLTestReply{}, structured
	}
	return provideripc.NewClient(cfg).RuntimeURLTest(ctx, req)
}

func (p *privilegedProviderClient) RuntimeCloseConnection(ctx context.Context, req provideripc.RuntimeCloseConnectionRequest) (provideripc.RuntimeCloseConnectionReply, *api.StructuredError) {
	cfg, structured := p.loadClientConfig()
	if structured != nil {
		return provideripc.RuntimeCloseConnectionReply{}, structured
	}
	return provideripc.NewClient(cfg).RuntimeCloseConnection(ctx, req)
}

func (p *privilegedProviderClient) RuntimeCloseAllConnections(ctx context.Context, req provideripc.RuntimeCloseAllConnectionsRequest) (provideripc.RuntimeCloseAllConnectionsReply, *api.StructuredError) {
	cfg, structured := p.loadClientConfig()
	if structured != nil {
		return provideripc.RuntimeCloseAllConnectionsReply{}, structured
	}
	return provideripc.NewClient(cfg).RuntimeCloseAllConnections(ctx, req)
}

func (p *privilegedProviderClient) RuntimeListenerInfo(ctx context.Context, req provideripc.RuntimeListenerInfoRequest) (provideripc.RuntimeListenerInfoReply, *api.StructuredError) {
	cfg, structured := p.loadClientConfig()
	if structured != nil {
		return provideripc.RuntimeListenerInfoReply{}, structured
	}
	return provideripc.NewClient(cfg).RuntimeListenerInfo(ctx, req)
}

func (p *privilegedProviderClient) RuntimeSubscribeEvents(ctx context.Context, req provideripc.RuntimeSubscribeEventsRequest) (<-chan provideripc.EventFrame, *api.StructuredError) {
	cfg, structured := p.loadClientConfig()
	if structured != nil {
		return nil, structured
	}
	return provideripc.NewClient(cfg).RuntimeSubscribeEvents(ctx, req)
}

func (p *privilegedProviderClient) loadClientConfig() (*provideripc.ClientConfig, *api.StructuredError) {
	cfg, err := provideripc.ReadClientConfig(p.clientConfigPath)
	if err != nil {
		message := err.Error()
		if errors.Is(err, os.ErrNotExist) {
			message = "Privileged provider client config is missing."
		}
		return nil, api.NewStructuredError(api.ErrorPlatformProviderUnavailable, message, "provider", true)
	}
	return cfg, nil
}

func (p *privilegedProviderClient) loadReadyClientConfig(ctx context.Context) (*provideripc.ClientConfig, *api.StructuredError) {
	cfg, structured := p.loadClientConfig()
	if structured != nil {
		return nil, structured
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
