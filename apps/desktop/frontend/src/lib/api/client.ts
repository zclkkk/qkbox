import { Events } from "@wailsio/runtime";
import { BridgeService } from "../../../bindings/github.com/zclkkk/qkbox/apps/desktop";
import type {
  Capability,
  EngineStatus,
  GetSystemProxyStatusReply,
  HelloReply,
  OutboundGroup,
  PrivilegedProviderStatus,
  ProductDiagnosticsReport,
  StructuredError,
} from "../../../bindings/github.com/zclkkk/qkbox/shared/api/models";

export type {
  Capability,
  EngineStatus,
  GetSystemProxyStatusReply,
  HelloReply,
  OutboundGroup,
  PrivilegedProviderStatus,
  ProductDiagnosticsReport,
  StructuredError
};

export type RuntimeLogEntry = {
  seq: number;
  timestamp: number;
  source: string;
  level: string;
  message: string;
};

export type TrafficSnapshot = {
  timestamp: number;
  upload_total: number;
  download_total: number;
  upload_rate: number;
  download_rate: number;
};

export type RuntimeConnection = {
  id: string;
  network: string;
  source: string;
  destination: string;
  host?: string;
  process?: string;
  inbound?: string;
  outbound?: string;
  rule?: string;
  upload: number;
  download: number;
  started_at?: number;
};

export type ConnectionSnapshot = {
  timestamp: number;
  upload_total: number;
  download_total: number;
  connections: RuntimeConnection[];
};

type ResultWrapper<T> = {
  reply?: T | null;
  error?: StructuredError | null;
};

export class QKBoxApiError extends Error {
  structured: StructuredError;

  constructor(error: StructuredError) {
    super(formatStructuredError(error));
    this.name = "QKBoxApiError";
    this.structured = error;
  }
}

export function formatStructuredError(error: Pick<StructuredError, "code" | "message">) {
  return `${error.code}: ${error.message}`;
}

async function unwrap<T>(resultPromise: PromiseLike<ResultWrapper<T>>, fallback?: T): Promise<T> {
  const result = await resultPromise;
  if (result.error) {
    throw new QKBoxApiError(result.error);
  }
  return result.reply ?? (fallback as T);
}

export const api = {
  app: {
    hello: () => unwrap<HelloReply>(BridgeService.Hello())
  },
  engine: {
    start: () => unwrap(BridgeService.EngineStart(), {}),
    stop: () => unwrap(BridgeService.EngineStop(), {}),
    reload: (snapshotID?: string) => unwrap(BridgeService.EngineReload({ snapshot_id: snapshotID ?? "" })),
    getStatus: () => unwrap(BridgeService.EngineGetStatus()),
    getRuntimeCapabilities: () => unwrap(BridgeService.EngineGetRuntimeCapabilities(), { capabilities: [] as Capability[] }),
    listGroups: () => unwrap(BridgeService.EngineListGroups(), { groups: [] as OutboundGroup[] }),
    selectOutbound: (groupTag: string, outboundTag: string) =>
      unwrap(BridgeService.EngineSelectOutbound({ group_tag: groupTag, outbound_tag: outboundTag })),
    urlTest: (groupTag: string, timeoutMS = 10_000) =>
      unwrap(BridgeService.EngineURLTest({ group_tag: groupTag, timeout_ms: timeoutMS }), { results: [] }),
    closeAllConnections: () => unwrap(BridgeService.EngineCloseAllConnections(), {})
  },
  profile: {
    list: () => unwrap(BridgeService.ListProfiles(), { profiles: [] }),
    getActiveProfile: () => unwrap(BridgeService.GetActiveProfile(), { profile: null }),
    getActiveSnapshot: () => unwrap(BridgeService.GetActiveSnapshot(), { snapshot: null }),
    listSnapshots: (profileID = "") => unwrap(BridgeService.ListSnapshots({ profile_id: profileID }), { snapshots: [] })
  },
  asset: {
    listProfileSubscriptions: (profileID = "") =>
      unwrap(BridgeService.AssetListProfileSubscriptions({ profile_id: profileID }), { subscriptions: [] }),
    listDataAssets: (kind = "") => unwrap(BridgeService.AssetListDataAssets({ kind }), { assets: [] })
  },
  platform: {
    getCapabilities: () => unwrap(BridgeService.PlatformGetCapabilities(), { capabilities: [] as Capability[] }),
    getPrivilegedProviderStatus: () =>
      unwrap(BridgeService.PlatformGetPrivilegedProviderStatus()),
    runRepairAction: (action: string) => unwrap(BridgeService.PlatformRunRepairAction({ action })),
    getSystemProxyStatus: () => unwrap(BridgeService.PlatformGetSystemProxyStatus()),
    setSystemProxyEnabled: (enabled: boolean) => unwrap(BridgeService.PlatformSetSystemProxyEnabled({ enabled }), {})
  },
  diagnostics: {
    getReport: () => unwrap(BridgeService.DiagnosticsGetReport()),
    createDebugBundle: () =>
      unwrap(BridgeService.DiagnosticsCreateDebugBundle())
  },
  events: {
    on: Events.On,
    startRuntimeBridge: () => unwrap(BridgeService.StartRuntimeEventBridge(), {}),
    stopRuntimeBridge: () => unwrap(BridgeService.StopRuntimeEventBridge(), {})
  }
};
