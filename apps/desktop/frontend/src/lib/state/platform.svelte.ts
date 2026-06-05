import { api, formatStructuredError, QKBoxApiError } from "../api/client";
import type {
  Capability,
  GetSystemProxyStatusReply,
  NetworkExtensionStatus,
  PrepareFeatureReply,
  PrivilegedProviderStatus,
  RunRepairActionReply
} from "../api/client";

class PlatformState {
  capabilities = $state<Capability[]>([]);
  privilegedProviderStatus = $state<PrivilegedProviderStatus | null>(null);
  networkExtensionStatus = $state<NetworkExtensionStatus | null>(null);
  proxyStatus = $state<GetSystemProxyStatusReply | null>(null);
  prepareResults = $state<Record<string, PrepareFeatureReply>>({});
  repairResults = $state<Record<string, RunRepairActionReply>>({});
  loading = $state(false);
  proxyLoading = $state(false);
  preparingFeature = $state<string | null>(null);
  repairingAction = $state<string | null>(null);
  error = $state<string | null>(null);
  proxyError = $state<string | null>(null);
  platformOS = $state("");

  async refresh(platformOS = this.platformOS) {
    this.platformOS = platformOS;
    this.loading = true;
    this.error = null;
    try {
      await Promise.all([
        this.refreshCapabilities(),
        this.refreshProviderStatus(),
        this.refreshProxyStatus(),
        this.refreshNetworkExtensionStatus()
      ]);
    } finally {
      this.loading = false;
    }
  }

  async refreshCapabilities() {
    try {
      const reply = await api.platform.getCapabilities();
      this.capabilities = reply.capabilities ?? [];
    } catch (error) {
      this.capture(error);
    }
  }

  async refreshProviderStatus() {
    try {
      const reply = await api.platform.getPrivilegedProviderStatus();
      this.privilegedProviderStatus = reply.status ?? null;
    } catch (error) {
      this.privilegedProviderStatus = null;
      this.capture(error);
    }
  }

  async refreshProxyStatus() {
    this.proxyError = null;
    try {
      this.proxyStatus = await api.platform.getSystemProxyStatus();
    } catch (error) {
      this.proxyStatus = null;
      this.proxyError = this.errorText(error);
    }
  }

  async refreshNetworkExtensionStatus() {
    if (this.platformOS !== "darwin") {
      this.networkExtensionStatus = null;
      return;
    }
    try {
      const reply = await api.platform.getNetworkExtensionStatus();
      this.networkExtensionStatus = reply.status ?? null;
    } catch (error) {
      this.networkExtensionStatus = null;
      this.capture(error);
    }
  }

  async prepareFeature(feature: string) {
    this.error = null;
    this.preparingFeature = feature;
    try {
      const reply = await api.platform.prepareFeature(feature);
      this.prepareResults = { ...this.prepareResults, [feature]: reply };
      await this.refreshCapabilities();
      await this.refreshProviderStatus();
      await this.refreshNetworkExtensionStatus();
    } catch (error) {
      this.capture(error);
    } finally {
      this.preparingFeature = null;
    }
  }

  async runProviderRepair(action: string) {
    this.error = null;
    this.repairingAction = action;
    try {
      const reply = await api.platform.runRepairAction(action);
      this.repairResults = { ...this.repairResults, [action]: reply };
      await this.refreshProviderStatus();
      await this.refreshCapabilities();
    } catch (error) {
      this.capture(error);
    } finally {
      this.repairingAction = null;
    }
  }

  async toggleProxy() {
    if (!this.proxyStatus?.available || !this.proxyStatus?.supported) {
      return;
    }
    this.proxyLoading = true;
    this.proxyError = null;
    try {
      await api.platform.setSystemProxyEnabled(!this.proxyStatus.qkbox_owned);
      await this.refreshProxyStatus();
    } catch (error) {
      this.proxyError = this.errorText(error);
    } finally {
      this.proxyLoading = false;
    }
  }

  private capture(error: unknown) {
    this.error = this.errorText(error);
  }

  private errorText(error: unknown) {
    if (error instanceof QKBoxApiError) {
      return formatStructuredError(error.structured);
    }
    return error instanceof Error ? error.message : String(error);
  }
}

export const platformState = new PlatformState();
