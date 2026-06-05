package qkboxd

import (
	"context"
	"time"

	"github.com/zclkkk/qkbox/internal/runtimeapi"
	"github.com/zclkkk/qkbox/internal/singboxadapter"
	"github.com/zclkkk/qkbox/platform/capability"
	"github.com/zclkkk/qkbox/shared/api"
)

// RuntimeOwner is the component that holds a live runtime instance.
type RuntimeOwner interface {
	Start(ctx context.Context, target RuntimeStartTarget) *api.StructuredError
	Stop() *api.StructuredError
	RuntimeCapabilities() []api.Capability
	TrafficSnapshot() (api.TrafficSnapshot, *api.StructuredError)
	ConnectionSnapshot() (api.ConnectionSnapshot, *api.StructuredError)
	ListGroups() ([]api.OutboundGroup, *api.StructuredError)
	SelectOutbound(groupTag, outboundTag string) (api.OutboundGroup, *api.StructuredError)
	URLTest(ctx context.Context, groupTag string, timeout time.Duration) ([]api.URLTestResult, *api.StructuredError)
	CloseConnection(id string) *api.StructuredError
	CloseAllConnections() *api.StructuredError
	ListenerInfo() ([]runtimeapi.ListenerInfo, *api.StructuredError)
}

type RuntimeStartTarget struct {
	SnapshotID           string
	ConfigJSON           string
	RequiredCapabilities []string
}

type RuntimeOwnerFactory func(target RuntimeStartTarget) RuntimeOwner

func newRuntimeOwnerFactory(events *RuntimeEventHub, privileged capability.PrivilegedProvider, sessionID string) RuntimeOwnerFactory {
	local := newLocalRuntimeOwnerFactory(events)
	return func(target RuntimeStartTarget) RuntimeOwner {
		if requiresProviderHostedRuntime(target) {
			if supportsProviderHostedMachineRuntime() {
				return newProviderRuntimeOwner(privileged, events, sessionID)
			}
			return unsupportedRuntimeOwner{
				err: api.NewStructuredError(api.ErrorPlatformFeatureUnsupported, "Provider-hosted machine network mode is not available on this platform.", "qkboxd", true),
			}
		}
		return local(target)
	}
}

func requiresProviderHostedRuntime(target RuntimeStartTarget) bool {
	for _, capName := range target.RequiredCapabilities {
		switch capName {
		case api.CapabilityTunMode, api.CapabilityDNSHijack:
			return true
		}
	}
	return false
}

type unsupportedRuntimeOwner struct {
	err *api.StructuredError
}

func (o unsupportedRuntimeOwner) Start(context.Context, RuntimeStartTarget) *api.StructuredError {
	return o.err
}

func (o unsupportedRuntimeOwner) Stop() *api.StructuredError {
	return nil
}

func (o unsupportedRuntimeOwner) RuntimeCapabilities() []api.Capability {
	return api.RuntimeCapabilityShell()
}

func (o unsupportedRuntimeOwner) TrafficSnapshot() (api.TrafficSnapshot, *api.StructuredError) {
	return api.TrafficSnapshot{}, o.err
}

func (o unsupportedRuntimeOwner) ConnectionSnapshot() (api.ConnectionSnapshot, *api.StructuredError) {
	return api.ConnectionSnapshot{}, o.err
}

func (o unsupportedRuntimeOwner) ListGroups() ([]api.OutboundGroup, *api.StructuredError) {
	return nil, o.err
}

func (o unsupportedRuntimeOwner) SelectOutbound(string, string) (api.OutboundGroup, *api.StructuredError) {
	return api.OutboundGroup{}, o.err
}

func (o unsupportedRuntimeOwner) URLTest(context.Context, string, time.Duration) ([]api.URLTestResult, *api.StructuredError) {
	return nil, o.err
}

func (o unsupportedRuntimeOwner) CloseConnection(string) *api.StructuredError {
	return o.err
}

func (o unsupportedRuntimeOwner) CloseAllConnections() *api.StructuredError {
	return o.err
}

func (o unsupportedRuntimeOwner) ListenerInfo() ([]runtimeapi.ListenerInfo, *api.StructuredError) {
	return nil, o.err
}

func newLocalRuntimeOwnerFactory(events *RuntimeEventHub) RuntimeOwnerFactory {
	return func(RuntimeStartTarget) RuntimeOwner {
		return &localRuntimeOwner{adapter: singboxadapter.NewAdapter(events)}
	}
}

type localRuntimeOwner struct {
	adapter *singboxadapter.Adapter
}

func (o *localRuntimeOwner) Start(ctx context.Context, target RuntimeStartTarget) *api.StructuredError {
	if err := o.adapter.Start(ctx, target.ConfigJSON); err != nil {
		code := api.ErrorSingboxAdapterStartFailed
		if ae, ok := err.(*singboxadapter.AdapterError); ok && ae.Code == "CONFIG_FAILED" {
			code = api.ErrorSingboxAdapterConfigFailed
		}
		return api.NewStructuredError(code, err.Error(), "singboxadapter", false)
	}
	return nil
}

func (o *localRuntimeOwner) Stop() *api.StructuredError {
	if err := o.adapter.Stop(); err != nil {
		return api.NewStructuredError(api.ErrorSingboxAdapterStopFailed, err.Error(), "singboxadapter", false)
	}
	return nil
}

func (o *localRuntimeOwner) RuntimeCapabilities() []api.Capability {
	return o.adapter.RuntimeCapabilities()
}

func (o *localRuntimeOwner) TrafficSnapshot() (api.TrafficSnapshot, *api.StructuredError) {
	return o.adapter.TrafficSnapshot()
}

func (o *localRuntimeOwner) ConnectionSnapshot() (api.ConnectionSnapshot, *api.StructuredError) {
	return o.adapter.ConnectionSnapshot()
}

func (o *localRuntimeOwner) ListGroups() ([]api.OutboundGroup, *api.StructuredError) {
	return o.adapter.ListGroups()
}

func (o *localRuntimeOwner) SelectOutbound(groupTag, outboundTag string) (api.OutboundGroup, *api.StructuredError) {
	return o.adapter.SelectOutbound(groupTag, outboundTag)
}

func (o *localRuntimeOwner) URLTest(ctx context.Context, groupTag string, timeout time.Duration) ([]api.URLTestResult, *api.StructuredError) {
	return o.adapter.URLTest(ctx, groupTag, timeout)
}

func (o *localRuntimeOwner) CloseConnection(id string) *api.StructuredError {
	return o.adapter.CloseConnection(id)
}

func (o *localRuntimeOwner) CloseAllConnections() *api.StructuredError {
	return o.adapter.CloseAllConnections()
}

func (o *localRuntimeOwner) ListenerInfo() ([]runtimeapi.ListenerInfo, *api.StructuredError) {
	return o.adapter.ListenerInfo()
}
