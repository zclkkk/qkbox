package singboxadapter

import (
	"context"
	"time"

	sbAdapter "github.com/sagernet/sing-box/adapter"
	"github.com/zclkkk/qkbox/shared/api"
)

func (a *Adapter) RuntimeCapabilities() []api.Capability {
	if a.tracker == nil {
		return api.RuntimeCapabilityShell()
	}
	return a.tracker.RuntimeCapabilities()
}

func (a *Adapter) TrafficSnapshot() (api.TrafficSnapshot, *api.StructuredError) {
	if a.tracker == nil {
		return api.TrafficSnapshot{}, api.NewStructuredError(api.ErrorObservabilityUnavailable, "Traffic source is unavailable.", "singboxadapter", true)
	}
	a.trafficMu.Lock()
	snapshot := a.tracker.TrafficSnapshot(a.lastTraffic)
	a.lastTraffic = snapshot
	a.trafficMu.Unlock()
	return snapshot, nil
}

func (a *Adapter) ConnectionSnapshot() (api.ConnectionSnapshot, *api.StructuredError) {
	if a.tracker == nil {
		return api.ConnectionSnapshot{}, api.NewStructuredError(api.ErrorObservabilityUnavailable, "Connection source is unavailable.", "singboxadapter", true)
	}
	return a.tracker.ConnectionSnapshot(), nil
}

func (a *Adapter) ListGroups() ([]api.OutboundGroup, *api.StructuredError) {
	ob, err := a.observableBox()
	if err != nil {
		return nil, err
	}
	manager := ob.Outbound()
	groups := make([]api.OutboundGroup, 0)
	for _, outbound := range manager.Outbounds() {
		if _, ok := outbound.(sbAdapter.OutboundGroup); !ok {
			continue
		}
		groups = append(groups, outboundGroupDTO(outbound, manager))
	}
	return groups, nil
}

func (a *Adapter) SelectOutbound(groupTag, outboundTag string) (api.OutboundGroup, *api.StructuredError) {
	ob, err := a.observableBox()
	if err != nil {
		return api.OutboundGroup{}, err
	}
	manager := ob.Outbound()
	group, structured := findGroup(manager, groupTag)
	if structured != nil {
		return api.OutboundGroup{}, structured
	}
	selector, ok := group.(selectableGroup)
	if !ok {
		return api.OutboundGroup{}, api.NewStructuredError(api.ErrorObservabilityUnsupported, "Group does not support outbound selection.", "singboxadapter", true)
	}
	if !selector.SelectOutbound(outboundTag) {
		return api.OutboundGroup{}, api.NewStructuredError(api.ErrorRuntimeOutboundNotFound, "Outbound not found in group.", "singboxadapter", true)
	}
	return outboundGroupDTO(group, manager), nil
}

func (a *Adapter) URLTest(ctx context.Context, groupTag string, timeout time.Duration) ([]api.URLTestResult, *api.StructuredError) {
	ob, err := a.observableBox()
	if err != nil {
		return nil, err
	}
	manager := ob.Outbound()
	group, structured := findGroup(manager, groupTag)
	if structured != nil {
		return nil, structured
	}
	if _, ok := group.(sbAdapter.URLTestGroup); !ok {
		return nil, api.NewStructuredError(api.ErrorObservabilityUnsupported, "Group does not support URLTest.", "singboxadapter", true)
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	testCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return urlTestResults(testCtx, group), nil
}

func (a *Adapter) CloseConnection(id string) *api.StructuredError {
	if a.tracker == nil {
		return api.NewStructuredError(api.ErrorObservabilityUnavailable, "Connection source is unavailable.", "singboxadapter", true)
	}
	if !a.tracker.CloseConnection(id) {
		return api.NewStructuredError(api.ErrorRuntimeConnectionNotFound, "Connection not found.", "singboxadapter", true)
	}
	return nil
}

func (a *Adapter) CloseAllConnections() *api.StructuredError {
	if a.tracker != nil {
		a.tracker.CloseAll()
	}
	return nil
}

func (a *Adapter) observableBox() (observableBoxHandle, *api.StructuredError) {
	if a.b == nil {
		return nil, api.NewStructuredError(api.ErrorEngineNotStarted, "Engine is not running.", "singboxadapter", true)
	}
	ob, ok := a.b.(observableBoxHandle)
	if !ok {
		return nil, api.NewStructuredError(api.ErrorObservabilityUnsupported, "Runtime does not expose observability sources.", "singboxadapter", true)
	}
	return ob, nil
}

func findGroup(manager sbAdapter.OutboundManager, tag string) (sbAdapter.Outbound, *api.StructuredError) {
	for _, outbound := range manager.Outbounds() {
		if outbound.Tag() != tag {
			continue
		}
		if _, ok := outbound.(sbAdapter.OutboundGroup); !ok {
			return nil, api.NewStructuredError(api.ErrorObservabilityUnsupported, "Outbound is not a group.", "singboxadapter", true)
		}
		return outbound, nil
	}
	return nil, api.NewStructuredError(api.ErrorRuntimeGroupNotFound, "Runtime group not found.", "singboxadapter", true)
}
