import { api, formatStructuredError, QKBoxApiError } from "../api/client";
import type { Capability, EngineReloadReply, EngineStatus, OutboundGroup, StructuredError, URLTestResult } from "../api/client";
import type { ProfileSnapshot } from "./profile.svelte";

class EngineState {
  loading = $state(false);
  status = $state<EngineStatus | null>(null);
  capabilities = $state<Capability[]>([]);
  groups = $state<OutboundGroup[]>([]);
  reloadSnapshots = $state<ProfileSnapshot[]>([]);
  reloadTargetSnapshotID = $state("");
  lastReloadResult = $state<EngineReloadReply | null>(null);
  urlTestResults = $state<Record<string, URLTestResult[]>>({});
  urlTestingGroup = $state<string | null>(null);
  error = $state<string | null>(null);

  get started() {
    return this.status?.state === "STARTED";
  }

  setStatus(status: EngineStatus) {
    this.status = status;
    if (status.state !== "STARTED") {
      this.groups = [];
    }
  }

  setEventError(error: StructuredError) {
    if (error.code === "ENGINE_NOT_STARTED" || error.code === "OBSERVABILITY_UNAVAILABLE") {
      return;
    }
    this.error = formatStructuredError(error);
  }

  async refresh() {
    this.error = null;
    await Promise.all([this.refreshStatus(), this.refreshCapabilities(), this.refreshGroups()]);
  }

  async refreshStatus() {
    try {
      const reply = await api.engine.getStatus();
      this.status = reply.status;
    } catch (error) {
      this.capture(error);
    }
  }

  async refreshCapabilities() {
    try {
      const reply = await api.engine.getRuntimeCapabilities();
      this.capabilities = reply.capabilities ?? [];
    } catch (error) {
      this.capture(error);
    }
  }

  async refreshGroups() {
    try {
      const reply = await api.engine.listGroups();
      this.groups = reply.groups ?? [];
    } catch (error) {
      if (error instanceof QKBoxApiError && error.structured.code === "ENGINE_NOT_STARTED") {
        this.groups = [];
        return;
      }
      this.groups = [];
      this.capture(error);
    }
  }

  async refreshReloadTargets(profileID: string) {
    if (!profileID) {
      this.reloadSnapshots = [];
      this.reloadTargetSnapshotID = "";
      return;
    }
    try {
      const reply = await api.profile.listSnapshots(profileID);
      this.reloadSnapshots = (reply.snapshots ?? []) as ProfileSnapshot[];
      if (this.reloadTargetSnapshotID && !this.reloadSnapshots.some((snapshot) => snapshot.id === this.reloadTargetSnapshotID)) {
        this.reloadTargetSnapshotID = "";
      }
    } catch (error) {
      this.reloadSnapshots = [];
      this.capture(error);
    }
  }

  async start() {
    await this.withLoading(async () => {
      await api.engine.start();
      await this.refresh();
    });
  }

  async stop() {
    await this.withLoading(async () => {
      await api.engine.stop();
      await this.refresh();
    });
  }

  async reload(snapshotID?: string) {
    await this.withLoading(async () => {
      const reply = await api.engine.reload(snapshotID);
      this.lastReloadResult = reply as EngineReloadReply;
      await this.refresh();
    });
  }

  async selectOutbound(groupTag: string, outboundTag: string) {
    this.error = null;
    try {
      await api.engine.selectOutbound(groupTag, outboundTag);
      await this.refreshGroups();
    } catch (error) {
      this.capture(error);
    }
  }

  async urlTest(groupTag: string) {
    this.error = null;
    this.urlTestingGroup = groupTag;
    try {
      const reply = await api.engine.urlTest(groupTag);
      this.urlTestResults = { ...this.urlTestResults, [groupTag]: reply.results ?? [] };
      await this.refreshGroups();
    } catch (error) {
      this.capture(error);
    } finally {
      this.urlTestingGroup = null;
    }
  }

  async closeConnection(connectionID: string) {
    this.error = null;
    try {
      await api.engine.closeConnection(connectionID);
    } catch (error) {
      this.capture(error);
    }
  }

  async closeAllConnections() {
    this.error = null;
    try {
      await api.engine.closeAllConnections();
    } catch (error) {
      this.capture(error);
    }
  }

  private async withLoading(fn: () => Promise<void>) {
    this.loading = true;
    this.error = null;
    try {
      await fn();
    } catch (error) {
      this.capture(error);
    } finally {
      this.loading = false;
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

export const engineState = new EngineState();
