import { api, formatStructuredError, QKBoxApiError } from "../api/client";
import type { ConnectionSnapshot, EngineStatus, RuntimeLogEntry, StructuredError, TrafficSnapshot } from "../api/client";
import { engineState } from "./engine.svelte";

const logLimit = 128;

class RuntimeEventState {
  logs = $state<RuntimeLogEntry[]>([]);
  traffic = $state<TrafficSnapshot | null>(null);
  connections = $state<ConnectionSnapshot | null>(null);
  bridgeError = $state<string | null>(null);
  running = $state(false);

  async start() {
    const offStatus = api.events.on("qkbox.engine.status", (event: { data: unknown }) => {
      engineState.setStatus(event.data as EngineStatus);
    });
    const offLog = api.events.on("qkbox.engine.log", (event: { data: unknown }) => {
      this.logs = [...this.logs.slice(-(logLimit - 1)), event.data as RuntimeLogEntry];
    });
    const offTraffic = api.events.on("qkbox.engine.traffic", (event: { data: unknown }) => {
      this.traffic = event.data as TrafficSnapshot;
    });
    const offConnections = api.events.on("qkbox.engine.connections", (event: { data: unknown }) => {
      this.connections = event.data as ConnectionSnapshot;
    });
    const offBridgeError = api.events.on("qkbox.engine.eventBridgeError", (event: { data: unknown }) => {
      const error = event.data as StructuredError;
      engineState.setEventError(error);
      this.bridgeError = `${error.code}: ${error.message}`;
    });

    try {
      await api.events.startRuntimeBridge();
      this.running = true;
    } catch (error) {
      offStatus();
      offLog();
      offTraffic();
      offConnections();
      offBridgeError();
      this.running = false;
      this.bridgeError = this.errorText(error);
      return async () => {};
    }

    return async () => {
      offStatus();
      offLog();
      offTraffic();
      offConnections();
      offBridgeError();
      this.running = false;
      await api.events.stopRuntimeBridge();
    };
  }

  private errorText(error: unknown) {
    if (error instanceof QKBoxApiError) {
      return formatStructuredError(error.structured);
    }
    return error instanceof Error ? error.message : String(error);
  }
}

export const runtimeEvents = new RuntimeEventState();
