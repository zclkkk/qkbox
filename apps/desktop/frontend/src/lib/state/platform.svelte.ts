import { api, formatStructuredError, QKBoxApiError } from "../api/client";
import type { Capability, GetSystemProxyStatusReply, PrivilegedProviderStatus } from "../api/client";

class PlatformState {
  capabilities = $state<Capability[]>([]);
  privilegedProviderStatus = $state<PrivilegedProviderStatus | null>(null);
  proxyStatus = $state<GetSystemProxyStatusReply | null>(null);
  loading = $state(false);
  proxyLoading = $state(false);
  error = $state<string | null>(null);
  proxyError = $state<string | null>(null);

  async refresh() {
    await Promise.all([this.refreshCapabilities(), this.refreshProviderStatus(), this.refreshProxyStatus()]);
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

  async runProviderRepair(action: string) {
    this.error = null;
    try {
      await api.platform.runRepairAction(action);
      await this.refreshProviderStatus();
      await this.refreshCapabilities();
    } catch (error) {
      this.capture(error);
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
