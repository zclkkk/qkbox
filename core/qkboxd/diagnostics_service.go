package qkboxd

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/zclkkk/qkbox/internal/assetcache"
	"github.com/zclkkk/qkbox/internal/persistence"
	"github.com/zclkkk/qkbox/internal/redact"
	"github.com/zclkkk/qkbox/platform/capability"
	"github.com/zclkkk/qkbox/shared/api"
	"github.com/zclkkk/qkbox/shared/model"
)

type DiagnosticsService struct {
	db         *persistence.DB
	engine     *EngineController
	platform   *PlatformService
	privileged capability.PrivilegedProvider
	extension  capability.NetworkExtensionRuntime
	assetStore *assetcache.Store
}

func (s *DiagnosticsService) DiagnosticsGetReport(ctx context.Context, _ api.GetDiagnosticsReportRequest) (api.GetDiagnosticsReportReply, *api.StructuredError) {
	report, structured := s.buildDiagnosticsReport(ctx)
	if structured != nil {
		return api.GetDiagnosticsReportReply{}, structured
	}
	return api.GetDiagnosticsReportReply{Report: report}, nil
}

func (s *DiagnosticsService) DiagnosticsCreateDebugBundle(ctx context.Context, _ api.CreateDebugBundleRequest) (api.CreateDebugBundleReply, *api.StructuredError) {
	report, structured := s.buildDiagnosticsReport(ctx)
	if structured != nil {
		return api.CreateDebugBundleReply{}, structured
	}
	path, manifest, err := s.writeDebugBundle(report)
	if err != nil {
		return api.CreateDebugBundleReply{}, api.NewStructuredError(api.ErrorDiagnosticsBundleFailed, err.Error(), "qkboxd", true)
	}
	return api.CreateDebugBundleReply{BundlePath: path, Manifest: manifest, Report: report}, nil
}

func (s *DiagnosticsService) buildDiagnosticsReport(ctx context.Context) (api.ProductDiagnosticsReport, *api.StructuredError) {
	dbSchemaVersion, err := s.db.SchemaVersion()
	if err != nil {
		return api.ProductDiagnosticsReport{}, qkboxdInternalError(err)
	}

	platformCapabilities := s.platform.platformCapabilities(ctx)
	providerStatus := api.PrivilegedProviderStatus{Reason: "Privileged provider is not configured."}
	if s.privileged != nil {
		probeCtx, cancel := context.WithTimeout(ctx, privilegedCapabilityProbeTimeout)
		defer cancel()
		providerStatus = s.privileged.Status(probeCtx)
	}

	var networkExtensionStatus *api.NetworkExtensionStatus
	if runtimeGOOS == "darwin" {
		status := api.NetworkExtensionStatus{Reason: "NetworkExtension runtime is not configured."}
		if s.extension != nil {
			probeCtx, cancel := context.WithTimeout(ctx, privilegedCapabilityProbeTimeout)
			defer cancel()
			status = s.extension.Status(probeCtx)
		}
		networkExtensionStatus = &status
	}

	var systemProxyStatus *api.GetSystemProxyStatusReply
	var systemProxyError *api.StructuredError
	if reply, structured := s.platform.PlatformGetSystemProxyStatus(ctx, api.GetSystemProxyStatusRequest{}); structured != nil {
		systemProxyError = structured
	} else {
		systemProxyStatus = &reply
	}

	activeProfile, err := s.db.GetActiveProfile()
	if err != nil {
		return api.ProductDiagnosticsReport{}, qkboxdInternalError(err)
	}
	activeSnapshot, _, err := s.db.GetActiveSnapshot()
	if err != nil {
		return api.ProductDiagnosticsReport{}, qkboxdInternalError(err)
	}
	subscriptions, err := s.db.ListProfileSubscriptions("")
	if err != nil {
		return api.ProductDiagnosticsReport{}, qkboxdInternalError(err)
	}
	assets, err := s.db.ListDataAssets("")
	if err != nil {
		return api.ProductDiagnosticsReport{}, qkboxdInternalError(err)
	}

	report := api.ProductDiagnosticsReport{
		GeneratedAt:              time.Now().UnixMilli(),
		APIVersion:               api.APIVersion,
		MinSupportedAPIVersion:   api.MinSupportedAPIVersion,
		SchemaRevision:           api.SchemaRevision,
		DBSchemaVersion:          dbSchemaVersion,
		AppVersion:               api.AppVersion,
		QKBoxDVersion:            api.QKBoxDVersion,
		Platform:                 model.Platform{OS: runtime.GOOS, Arch: runtime.GOARCH},
		EngineStatus:             s.engine.GetStatus(),
		RuntimeCapabilities:      s.engine.RuntimeCapabilities(),
		PlatformCapabilities:     platformCapabilities,
		PrivilegedProviderStatus: providerStatus,
		NetworkExtensionStatus:   networkExtensionStatus,
		SystemProxyStatus:        systemProxyStatus,
		SystemProxyError:         systemProxyError,
		Subscriptions:            diagnosticSubscriptions(subscriptions),
		DataAssets:               diagnosticAssets(assets),
	}
	if activeProfile != nil {
		report.ActiveProfileID = activeProfile.ID
	}
	if activeSnapshot != nil {
		report.ActiveSnapshotID = activeSnapshot.ID
	}
	report.Checks = diagnosticChecks(report)
	return report, nil
}

func diagnosticSubscriptions(subscriptions []model.ProfileSubscription) []api.DiagnosticSubscriptionSummary {
	out := make([]api.DiagnosticSubscriptionSummary, 0, len(subscriptions))
	for _, sub := range subscriptions {
		out = append(out, api.DiagnosticSubscriptionSummary{
			ID:               sub.ID,
			ProfileID:        sub.ProfileID,
			Name:             sub.Name,
			RedactedURL:      redact.URL(sub.URL),
			UpdatePolicy:     string(sub.UpdatePolicy),
			LastStatus:       string(sub.LastStatus),
			LastErrorCode:    sub.LastErrorCode,
			LastErrorMessage: sub.LastErrorMessage,
			LastCheckedAt:    sub.LastCheckedAt,
			LastUpdatedAt:    sub.LastUpdatedAt,
			ContentSHA256:    sub.ContentSHA256,
		})
	}
	return out
}

func diagnosticAssets(assets []model.DataAsset) []api.DiagnosticAssetSummary {
	out := make([]api.DiagnosticAssetSummary, 0, len(assets))
	for _, asset := range assets {
		out = append(out, api.DiagnosticAssetSummary{
			ID:                asset.ID,
			Kind:              string(asset.Kind),
			Name:              asset.Name,
			RedactedSourceURL: redact.URL(asset.SourceURL),
			Status:            string(asset.Status),
			CacheKey:          asset.CacheKey,
			Version:           asset.Version,
			ContentSHA256:     asset.ContentSHA256,
			SizeBytes:         asset.SizeBytes,
			LastErrorCode:     asset.LastErrorCode,
			LastErrorMessage:  asset.LastErrorMessage,
			LastCheckedAt:     asset.LastCheckedAt,
			LastUpdatedAt:     asset.LastUpdatedAt,
		})
	}
	return out
}

func diagnosticChecks(report api.ProductDiagnosticsReport) []api.DiagnosticCheck {
	checks := []api.DiagnosticCheck{
		{Name: "schema", State: api.CapabilityAvailable, Reason: fmt.Sprintf("API %s / DB schema %d / revision %s", report.APIVersion, report.DBSchemaVersion, report.SchemaRevision)},
	}
	if report.PrivilegedProviderStatus.ExpectedVersion != "" && report.PrivilegedProviderStatus.Version != "" && report.PrivilegedProviderStatus.Version != report.PrivilegedProviderStatus.ExpectedVersion {
		checks = append(checks, api.DiagnosticCheck{
			Name:     "privileged_provider_version",
			State:    api.CapabilityDegraded,
			Reason:   "Privileged provider version does not match qkbox.",
			Recovery: "Reinstall or update the privileged provider from the same qkbox build.",
		})
	}
	if owner := report.PrivilegedProviderStatus.OwnerState; owner != nil && owner.Stale {
		checks = append(checks, api.DiagnosticCheck{
			Name:     "machine_network_owner",
			State:    api.CapabilityDegraded,
			Reason:   owner.Reason,
			Recovery: "Use the listed provider recovery action or restart the privileged provider.",
		})
	}
	if report.SystemProxyStatus != nil && report.SystemProxyStatus.QKBoxOwned && report.EngineStatus.State != model.EngineStateStarted {
		checks = append(checks, api.DiagnosticCheck{
			Name:     "system_proxy_owner",
			State:    api.CapabilityDegraded,
			Reason:   "System proxy is owned by qkbox while the engine is not started.",
			Recovery: "Disable system proxy or start the engine before using the system proxy mode.",
		})
	}
	for _, sub := range report.Subscriptions {
		if sub.LastStatus == string(model.SubscriptionStatusFailed) {
			checks = append(checks, api.DiagnosticCheck{
				Name:     "subscription:" + sub.ID,
				State:    api.CapabilityDegraded,
				Reason:   sub.LastErrorMessage,
				Recovery: "Fix the remote subscription source and refresh it again.",
			})
		}
	}
	for _, asset := range report.DataAssets {
		if asset.Status == string(model.DataAssetStatusFailed) {
			checks = append(checks, api.DiagnosticCheck{
				Name:     "asset:" + asset.ID,
				State:    api.CapabilityDegraded,
				Reason:   asset.LastErrorMessage,
				Recovery: "Fix the data asset source and refresh it again.",
			})
		}
	}
	if len(checks) == 1 {
		checks = append(checks, api.DiagnosticCheck{Name: "product_state", State: api.CapabilityAvailable, Reason: "No degraded product state detected."})
	}
	return checks
}

func (s *DiagnosticsService) writeDebugBundle(report api.ProductDiagnosticsReport) (string, api.DebugBundleManifest, error) {
	dir := filepath.Join(s.db.StateDir(), "debug-bundles")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", api.DebugBundleManifest{}, err
	}
	name := "qkbox-debug-" + time.Now().UTC().Format("20060102T150405.000000000Z") + ".zip"
	path := filepath.Join(dir, name)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", api.DebugBundleManifest{}, err
	}
	defer file.Close()

	manifest := api.DebugBundleManifest{
		CreatedAt:      report.GeneratedAt,
		APIVersion:     api.APIVersion,
		SchemaRevision: api.SchemaRevision,
		AppVersion:     api.AppVersion,
		QKBoxDVersion:  api.QKBoxDVersion,
		Platform:       report.Platform.OS + "/" + report.Platform.Arch,
		Files:          []string{"manifest.json", "diagnostics.json", "README.txt"},
		Redaction:      "Profile content, encrypted blobs, proxy snapshots, provider tokens, URL userinfo, URL paths, query strings, and URL fragments are not included.",
	}

	zw := zip.NewWriter(file)
	if err := writeZipJSON(zw, "manifest.json", manifest); err != nil {
		_ = zw.Close()
		return "", api.DebugBundleManifest{}, err
	}
	if err := writeZipJSON(zw, "diagnostics.json", report); err != nil {
		_ = zw.Close()
		return "", api.DebugBundleManifest{}, err
	}
	if err := writeZipText(zw, "README.txt", "qkbox debug bundle\n\nThis bundle is generated by qkbox and excludes profile content, encrypted content, provider tokens, raw platform owner snapshots, and sensitive URL components.\n"); err != nil {
		_ = zw.Close()
		return "", api.DebugBundleManifest{}, err
	}
	if err := zw.Close(); err != nil {
		return "", api.DebugBundleManifest{}, err
	}
	return path, manifest, nil
}

func writeZipJSON(zw *zip.Writer, name string, value interface{}) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	return writeZipBytes(zw, name, payload)
}

func writeZipText(zw *zip.Writer, name string, value string) error {
	return writeZipBytes(zw, name, []byte(value))
}

func writeZipBytes(zw *zip.Writer, name string, payload []byte) error {
	file, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = file.Write(payload)
	return err
}
