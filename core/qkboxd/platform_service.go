package qkboxd

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/zclkkk/qkbox/internal/persistence"
	"github.com/zclkkk/qkbox/platform/capability"
	"github.com/zclkkk/qkbox/shared/api"
)

type PlatformService struct {
	db         *persistence.DB
	engine     *EngineController
	proxy      capability.SystemProxyProvider
	privileged capability.PrivilegedProvider
	extension  capability.NetworkExtensionRuntime
	opMu       *sync.Mutex
}

func (s *PlatformService) platformCapabilities(ctx context.Context) []api.Capability {
	caps := api.PlatformCapabilityShell()
	if s.proxy == nil {
		return s.applyPrivilegedCapabilities(ctx, caps)
	}
	avail := s.proxy.Availability()
	for i, cap := range caps {
		if cap.Name == api.CapabilitySystemProxy {
			if avail.Available && avail.Supported {
				caps[i].State = api.CapabilityAvailable
				caps[i].Reason = ""
			} else if avail.Available && !avail.Supported {
				caps[i].State = api.CapabilityUnavailable
				caps[i].Reason = avail.Reason
			} else {
				caps[i].State = api.CapabilityUnsupported
				caps[i].Reason = avail.Reason
			}
			break
		}
	}
	return s.applyPrivilegedCapabilities(ctx, caps)
}

func (s *PlatformService) applyPrivilegedCapabilities(ctx context.Context, caps []api.Capability) []api.Capability {
	if runtimeGOOS == "darwin" {
		return s.applyNetworkExtensionCapabilities(ctx, caps)
	}
	status := api.PrivilegedProviderStatus{Reason: "Privileged provider is not configured."}
	if s.privileged != nil {
		probeCtx, cancel := context.WithTimeout(ctx, privilegedCapabilityProbeTimeout)
		defer cancel()
		status = s.privileged.Status(probeCtx)
	}
	providerReady := status.Installed && status.Reachable && status.Authenticated && status.Version == status.ExpectedVersion
	providerCaps := map[string]api.Capability{}
	for _, cap := range status.Capabilities {
		providerCaps[cap.Name] = cap
	}
	for i, cap := range caps {
		switch cap.Name {
		case api.CapabilityBackgroundService:
			if providerReady {
				caps[i].State = api.CapabilityAvailable
				caps[i].Reason = ""
			} else {
				caps[i].State = api.CapabilityUnavailable
				caps[i].Reason = providerStatusReason(status)
			}
		case api.CapabilityTunMode, api.CapabilityDNSHijack:
			if providerReady {
				if providerCap, ok := providerCaps[cap.Name]; ok {
					caps[i].State = providerCap.State
					caps[i].Reason = providerCap.Reason
				} else {
					caps[i].State = api.CapabilityUnavailable
					caps[i].Reason = "Privileged provider did not report this capability."
				}
			} else {
				caps[i].State = api.CapabilityUnavailable
				caps[i].Reason = providerStatusReason(status)
			}
		}
	}
	return caps
}

func (s *PlatformService) applyNetworkExtensionCapabilities(ctx context.Context, caps []api.Capability) []api.Capability {
	status := api.NetworkExtensionStatus{Reason: "NetworkExtension runtime is not configured."}
	if s.extension != nil {
		probeCtx, cancel := context.WithTimeout(ctx, privilegedCapabilityProbeTimeout)
		defer cancel()
		status = s.extension.Status(probeCtx)
	}
	extensionCaps := map[string]api.Capability{}
	for _, cap := range status.Capabilities {
		extensionCaps[cap.Name] = cap
	}
	for i, cap := range caps {
		switch cap.Name {
		case api.CapabilityTunMode, api.CapabilityDNSHijack, api.CapabilityConnectionTracking:
			if extensionCap, ok := extensionCaps[cap.Name]; ok {
				caps[i].State = extensionCap.State
				caps[i].Reason = extensionCap.Reason
			} else {
				caps[i].State = api.CapabilityUnavailable
				caps[i].Reason = networkExtensionStatusReason(status)
			}
		case api.CapabilityBackgroundService:
			caps[i].State = api.CapabilityUnsupported
			caps[i].Reason = "macOS machine network mode is owned by NetworkExtension."
		}
	}
	return caps
}

func networkExtensionStatusReason(status api.NetworkExtensionStatus) string {
	if status.Reason != "" {
		return status.Reason
	}
	if !status.Installed {
		return "NetworkExtension container is not installed."
	}
	if !status.Reachable {
		return "NetworkExtension container is not reachable."
	}
	if !status.Authorized {
		return "NetworkExtension container is not authorized."
	}
	return "NetworkExtension runtime is unavailable."
}

func providerStatusReason(status api.PrivilegedProviderStatus) string {
	if status.Reason != "" {
		return status.Reason
	}
	if !status.Installed {
		return "Privileged provider is not installed."
	}
	if !status.Reachable {
		return "Privileged provider is not reachable."
	}
	if !status.Authenticated {
		return "Privileged provider authentication failed."
	}
	if status.ExpectedVersion != "" && status.Version != status.ExpectedVersion {
		return "Privileged provider version mismatch."
	}
	return "Privileged provider is unavailable."
}

func (s *PlatformService) prepareRuntimeStartTargetCapabilities(ctx context.Context, target RuntimeStartTarget) *api.StructuredError {
	for _, feature := range target.RequiredCapabilities {
		if !isPrivilegedFeature(feature) {
			continue
		}
		reply, structured := s.preparePlatformFeature(ctx, feature)
		if structured != nil {
			return structured
		}
		if reply.State != api.CapabilityAvailable {
			err := api.NewStructuredError(api.ErrorPlatformPrepareFailed, "Required platform capability is not available.", "provider", true)
			err.Detail = reply
			return err
		}
	}
	return nil
}

func (s *PlatformService) preparePlatformFeature(ctx context.Context, feature string) (api.PrepareFeatureReply, *api.StructuredError) {
	if !isPrivilegedFeature(feature) {
		return api.PrepareFeatureReply{}, api.NewStructuredError(api.ErrorPlatformFeatureUnsupported, "Feature is not supported by the platform capability boundary.", "qkboxd", true)
	}
	if runtimeGOOS == "darwin" && isNetworkExtensionFeature(feature) {
		status := api.NetworkExtensionStatus{Reason: "NetworkExtension runtime is not configured."}
		if s.extension != nil {
			status = s.extension.Status(ctx)
		}
		for _, cap := range status.Capabilities {
			if cap.Name == feature {
				return api.PrepareFeatureReply{Feature: feature, State: cap.State, Reason: cap.Reason}, nil
			}
		}
		return api.PrepareFeatureReply{Feature: feature, State: api.CapabilityUnavailable, Reason: networkExtensionStatusReason(status)}, nil
	}
	if s.privileged == nil {
		return api.PrepareFeatureReply{}, api.NewStructuredError(api.ErrorPlatformProviderUnavailable, "Privileged provider is not configured.", "provider", true)
	}
	return s.privileged.PrepareFeature(ctx, feature)
}

func isPrivilegedFeature(feature string) bool {
	switch feature {
	case api.CapabilityTunMode, api.CapabilityDNSHijack, api.CapabilityBackgroundService:
		return true
	default:
		return false
	}
}

func isNetworkExtensionFeature(feature string) bool {
	switch feature {
	case api.CapabilityTunMode, api.CapabilityDNSHijack:
		return true
	default:
		return false
	}
}

// Platform capabilities

func (s *PlatformService) PlatformGetCapabilities(ctx context.Context, _ api.GetPlatformCapabilitiesRequest) (api.GetPlatformCapabilitiesReply, *api.StructuredError) {
	return api.GetPlatformCapabilitiesReply{Capabilities: s.platformCapabilities(ctx)}, nil
}

func (s *PlatformService) PlatformGetPrivilegedProviderStatus(ctx context.Context, _ api.GetPrivilegedProviderStatusRequest) (api.GetPrivilegedProviderStatusReply, *api.StructuredError) {
	if s.privileged == nil {
		return api.GetPrivilegedProviderStatusReply{
			Status: api.PrivilegedProviderStatus{Reason: "Privileged provider is not configured."},
		}, nil
	}
	return api.GetPrivilegedProviderStatusReply{Status: s.privileged.Status(ctx)}, nil
}

func (s *PlatformService) PlatformGetNetworkExtensionStatus(ctx context.Context, _ api.GetNetworkExtensionStatusRequest) (api.GetNetworkExtensionStatusReply, *api.StructuredError) {
	status := api.NetworkExtensionStatus{Reason: "NetworkExtension runtime is not configured."}
	if s.extension != nil {
		status = s.extension.Status(ctx)
	}
	return api.GetNetworkExtensionStatusReply{Status: status}, nil
}

func (s *PlatformService) PlatformPrepareFeature(ctx context.Context, req api.PrepareFeatureRequest) (api.PrepareFeatureReply, *api.StructuredError) {
	return s.preparePlatformFeature(ctx, req.Feature)
}

func (s *PlatformService) PlatformRunRepairAction(ctx context.Context, req api.RunRepairActionRequest) (api.RunRepairActionReply, *api.StructuredError) {
	if s.privileged == nil {
		return api.RunRepairActionReply{}, api.NewStructuredError(api.ErrorPlatformProviderUnavailable, "Privileged provider is not configured.", "provider", true)
	}
	return s.privileged.RunRepairAction(ctx, req.Action)
}

func (s *PlatformService) PlatformGetSystemProxyStatus(_ context.Context, _ api.GetSystemProxyStatusRequest) (api.GetSystemProxyStatusReply, *api.StructuredError) {
	reply := api.GetSystemProxyStatusReply{}

	if s.proxy == nil {
		return reply, nil
	}
	avail := s.proxy.Availability()
	reply.Available = avail.Available
	reply.Supported = avail.Supported
	reply.Reason = avail.Reason
	if !avail.Available || !avail.Supported {
		return reply, nil
	}

	state, err := s.proxy.CurrentState()
	if err != nil {
		return reply, api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "platform", true)
	}
	reply.OSEnabled = state.Enabled
	reply.Address = state.Addr
	reply.Port = state.Port

	record, err := loadProxyOwner(s.db)
	if err != nil {
		return reply, api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "qkboxd", false)
	}
	if record != nil && record.QKBoxOwned {
		if proxyOwnerMatches(state, record) {
			reply.QKBoxOwned = true
			reply.Address = record.ProxyAddr
			reply.Port = record.ProxyPort
		}
	}

	return reply, nil
}

func (s *PlatformService) PlatformSetSystemProxyEnabled(_ context.Context, req api.SetSystemProxyEnabledRequest) (api.SetSystemProxyEnabledReply, *api.StructuredError) {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	if s.proxy == nil || !s.proxy.Availability().Available || !s.proxy.Availability().Supported {
		return api.SetSystemProxyEnabledReply{}, api.NewStructuredError(api.ErrorPlatformProxyUnsupported, "System proxy is not available on this platform.", "platform", false)
	}

	if !req.Enabled {
		return api.SetSystemProxyEnabledReply{}, s.disableProxy()
	}

	return api.SetSystemProxyEnabledReply{}, s.enableProxy()
}

func (s *PlatformService) enableProxy() *api.StructuredError {
	listeners, sErr := s.engine.ListenerInfo()
	if sErr != nil {
		return sErr
	}
	if len(listeners) == 0 {
		return api.NewStructuredError(api.ErrorPlatformProxyNoListener, "No HTTP/mixed inbound found in active config.", "qkboxd", true)
	}
	target := listeners[0]
	addr := target.Address
	port := target.Port

	record, err := loadProxyOwner(s.db)
	if err != nil {
		return api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "qkboxd", false)
	}

	if record != nil && record.QKBoxOwned {
		state, err := s.proxy.CurrentState()
		if err != nil {
			return api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "platform", true)
		}
		if proxyOwnerMatches(state, record) {
			return nil
		}
		record.ProxyAddr = addr
		record.ProxyPort = port
		record.EnabledAt = time.Now().UnixMilli()
		if err := saveProxyOwner(s.db, record); err != nil {
			return api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "qkboxd", false)
		}
		if err := s.proxy.Apply(addr, port); err != nil {
			return api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "platform", true)
		}
		return nil
	}

	snapshot, err := s.proxy.Snapshot()
	if err != nil {
		return api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "platform", true)
	}

	newRecord := &proxyOwnerRecord{
		QKBoxOwned: true,
		Snapshot:   snapshot,
		ProxyAddr:  addr,
		ProxyPort:  port,
		EnabledAt:  time.Now().UnixMilli(),
	}
	if err := saveProxyOwner(s.db, newRecord); err != nil {
		return api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "qkboxd", false)
	}

	if err := s.proxy.Apply(addr, port); err != nil {
		if restoreErr := s.proxy.Restore(snapshot); restoreErr == nil {
			_ = deleteProxyOwner(s.db)
		}
		return api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "platform", true)
	}

	return nil
}

func (s *PlatformService) disableProxy() *api.StructuredError {
	record, err := loadProxyOwner(s.db)
	if err != nil {
		return api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "qkboxd", false)
	}
	if record == nil || !record.QKBoxOwned {
		return nil
	}

	state, err := s.proxy.CurrentState()
	if err != nil {
		return api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "platform", true)
	}
	if !proxyOwnerMatches(state, record) {
		_ = deleteProxyOwner(s.db)
		return nil
	}

	if err := s.proxy.Restore(record.Snapshot); err != nil {
		return api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "platform", true)
	}
	_ = deleteProxyOwner(s.db)
	return nil
}

func (s *PlatformService) restoreProxyIfOwned() *api.StructuredError {
	if s.proxy == nil {
		return nil
	}
	record, err := loadProxyOwner(s.db)
	if err != nil {
		return api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "qkboxd", false)
	}
	if record == nil || !record.QKBoxOwned {
		return nil
	}
	avail := s.proxy.Availability()
	if !avail.Available || !avail.Supported {
		return api.NewStructuredError(api.ErrorPlatformProxyFailed, "System proxy owner record exists but the platform provider is unavailable.", "platform", true)
	}

	state, err := s.proxy.CurrentState()
	if err != nil {
		return api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "platform", true)
	}
	if !proxyOwnerMatches(state, record) {
		_ = deleteProxyOwner(s.db)
		return nil
	}

	if err := s.proxy.Restore(record.Snapshot); err != nil {
		return api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "platform", true)
	}
	_ = deleteProxyOwner(s.db)
	return nil
}

func (s *PlatformService) captureOwnedProxyForActivation() (*proxyOwnerRecord, *api.StructuredError) {
	if s.proxy == nil {
		return nil, nil
	}
	record, err := loadProxyOwner(s.db)
	if err != nil {
		return nil, api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "qkboxd", false)
	}
	if record == nil || !record.QKBoxOwned {
		return nil, nil
	}
	avail := s.proxy.Availability()
	if !avail.Available || !avail.Supported {
		return nil, api.NewStructuredError(api.ErrorPlatformProxyFailed, "System proxy owner record exists but the platform provider is unavailable.", "platform", true)
	}
	state, err := s.proxy.CurrentState()
	if err != nil {
		return nil, api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "platform", true)
	}
	if !proxyOwnerMatches(state, record) {
		_ = deleteProxyOwner(s.db)
		return nil, nil
	}
	return record, nil
}

func (s *PlatformService) restoreCapturedProxy(record *proxyOwnerRecord) *api.StructuredError {
	if s.proxy == nil || record == nil {
		return nil
	}
	if err := s.proxy.Restore(record.Snapshot); err != nil {
		return api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "platform", true)
	}
	_ = deleteProxyOwner(s.db)
	return nil
}

func (s *PlatformService) bindCapturedProxyToRuntime(record *proxyOwnerRecord) *api.StructuredError {
	if s.proxy == nil || record == nil {
		return nil
	}
	avail := s.proxy.Availability()
	if !avail.Available || !avail.Supported {
		return api.NewStructuredError(api.ErrorPlatformProxyFailed, "System proxy owner record exists but the platform provider is unavailable.", "platform", true)
	}
	listeners, sErr := s.engine.ListenerInfo()
	if sErr != nil {
		return sErr
	}
	if len(listeners) == 0 {
		return api.NewStructuredError(api.ErrorPlatformProxyNoListener, "No HTTP/mixed inbound found in active config.", "qkboxd", true)
	}
	target := listeners[0]
	updated := *record
	updated.ProxyAddr = target.Address
	updated.ProxyPort = target.Port
	updated.EnabledAt = time.Now().UnixMilli()
	if err := saveProxyOwner(s.db, &updated); err != nil {
		return api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "qkboxd", false)
	}
	if err := s.proxy.Apply(updated.ProxyAddr, updated.ProxyPort); err != nil {
		if restoreErr := s.proxy.Restore(record.Snapshot); restoreErr == nil {
			_ = deleteProxyOwner(s.db)
		}
		return api.NewStructuredError(api.ErrorPlatformProxyFailed, err.Error(), "platform", true)
	}
	return nil
}

func (s *PlatformService) bestEffortProxyRestore() {
	if s.proxy == nil {
		return
	}
	record, err := loadProxyOwner(s.db)
	if err != nil || record == nil || !record.QKBoxOwned {
		return
	}
	avail := s.proxy.Availability()
	if !avail.Available || !avail.Supported {
		fmt.Printf("warning: system proxy owner record kept because provider is unavailable: %s\n", avail.Reason)
		return
	}
	state, err := s.proxy.CurrentState()
	if err != nil {
		fmt.Printf("warning: failed to read system proxy on shutdown: %v\n", err)
		return
	}
	if !proxyOwnerMatches(state, record) {
		_ = deleteProxyOwner(s.db)
		return
	}
	if err := s.proxy.Restore(record.Snapshot); err != nil {
		fmt.Printf("warning: failed to restore system proxy on shutdown: %v\n", err)
		return
	}
	_ = deleteProxyOwner(s.db)
}
