import { api, formatStructuredError, QKBoxApiError } from "../api/client";
import type { HelloReply } from "../api/client";
import { assetState } from "./asset.svelte";
import { diagnosticsState } from "./diagnostics.svelte";
import { engineState } from "./engine.svelte";
import { platformState } from "./platform.svelte";
import { profileState } from "./profile.svelte";

class AppState {
  loading = $state(true);
  hello = $state<HelloReply | null>(null);
  error = $state<string | null>(null);
  lastChecked = $state("Never");

  async bootstrap() {
    this.loading = true;
    this.error = null;
    let ready = false;
    try {
      this.hello = await api.app.hello();
      engineState.capabilities = this.hello.runtime_capabilities;
      platformState.capabilities = this.hello.platform_capabilities;
      this.lastChecked = new Date().toLocaleTimeString();
      ready = true;
    } catch (error) {
      this.hello = null;
      this.capture(error);
    }

    if (ready) {
      await Promise.all([
        engineState.refresh(),
        platformState.refresh(),
        profileState.refresh(),
        assetState.refresh(),
        diagnosticsState.refresh()
      ]);
    }

    this.loading = false;
  }

  private capture(error: unknown) {
    if (error instanceof QKBoxApiError) {
      this.error = formatStructuredError(error.structured);
      return;
    }
    this.error = error instanceof Error ? error.message : String(error);
  }
}

export const appState = new AppState();
