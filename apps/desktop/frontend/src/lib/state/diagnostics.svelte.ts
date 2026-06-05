import { api, formatStructuredError, QKBoxApiError } from "../api/client";
import type { ProductDiagnosticsReport } from "../api/client";

class DiagnosticsState {
  report = $state<ProductDiagnosticsReport | null>(null);
  debugBundlePath = $state<string | null>(null);
  error = $state<string | null>(null);

  async refresh() {
    this.error = null;
    try {
      const reply = await api.diagnostics.getReport();
      this.report = reply.report ?? null;
    } catch (error) {
      this.report = null;
      this.capture(error);
    }
  }

  async createDebugBundle() {
    this.error = null;
    this.debugBundlePath = null;
    try {
      const reply = await api.diagnostics.createDebugBundle();
      this.debugBundlePath = reply.bundle_path ?? null;
      this.report = reply.report ?? null;
    } catch (error) {
      this.capture(error);
    }
  }

  private capture(error: unknown) {
    if (error instanceof QKBoxApiError) {
      this.error = formatStructuredError(error.structured);
      return;
    }
    this.error = error instanceof Error ? error.message : String(error);
  }
}

export const diagnosticsState = new DiagnosticsState();
