package qkboxd

import (
	"context"
	"time"

	"github.com/zclkkk/qkbox/internal/runtimeapi"
	"github.com/zclkkk/qkbox/internal/singboxadapter"
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
