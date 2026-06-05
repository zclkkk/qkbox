import { Events } from "@wailsio/runtime";
import { BridgeService } from "../../../bindings/github.com/zclkkk/qkbox/apps/desktop";
import type {
  Capability,
  EngineReloadReply,
  EngineStatus,
  GetSystemProxyStatusReply,
  HelloReply,
  OutboundGroup,
  PrivilegedProviderStatus,
  ProductDiagnosticsReport,
  StructuredError,
  URLTestResult,
} from "../../../bindings/github.com/zclkkk/qkbox/shared/api/models";

export type {
  Capability,
  EngineReloadReply,
  EngineStatus,
  GetSystemProxyStatusReply,
  HelloReply,
  OutboundGroup,
  PrivilegedProviderStatus,
  ProductDiagnosticsReport,
  StructuredError,
  URLTestResult
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
    closeConnection: (connectionID: string) => unwrap(BridgeService.EngineCloseConnection({ connection_id: connectionID }), {}),
    closeAllConnections: () => unwrap(BridgeService.EngineCloseAllConnections(), {})
  },
  profile: {
    create: (name: string, content: string) => unwrap(BridgeService.CreateProfile({ name, content })),
    updateDraft: (profileID: string, content: string) =>
      unwrap(BridgeService.UpdateProfileDraft({ profile_id: profileID, content })),
    delete: (profileID: string) => unwrap(BridgeService.DeleteProfile({ profile_id: profileID })),
    list: () => unwrap(BridgeService.ListProfiles(), { profiles: [] }),
    get: (profileID: string) => unwrap(BridgeService.GetProfile({ profile_id: profileID })),
    validateDraft: (profileID: string) => unwrap(BridgeService.ValidateProfileDraft({ profile_id: profileID })),
    createSnapshot: (profileID: string) => unwrap(BridgeService.CreateProfileSnapshot({ profile_id: profileID })),
    activateSnapshot: (snapshotID: string) => unwrap(BridgeService.ActivateProfileSnapshot({ snapshot_id: snapshotID })),
    getActiveProfile: () => unwrap(BridgeService.GetActiveProfile(), { profile: null }),
    getActiveSnapshot: () => unwrap(BridgeService.GetActiveSnapshot(), { snapshot: null }),
    listSnapshots: (profileID = "") => unwrap(BridgeService.ListSnapshots({ profile_id: profileID }), { snapshots: [] }),
    rollbackToSnapshot: (snapshotID: string) => unwrap(BridgeService.RollbackToSnapshot({ snapshot_id: snapshotID }))
  },
  asset: {
    createProfileSubscription: (profileID: string, name: string, url: string) =>
      unwrap(BridgeService.AssetCreateProfileSubscription({ profile_id: profileID, name, url, update_policy: "manual" })),
    listProfileSubscriptions: (profileID = "") =>
      unwrap(BridgeService.AssetListProfileSubscriptions({ profile_id: profileID }), { subscriptions: [] }),
    refreshProfileSubscription: (subscriptionID: string) =>
      unwrap(BridgeService.AssetRefreshProfileSubscription({ subscription_id: subscriptionID })),
    deleteProfileSubscription: (subscriptionID: string) =>
      unwrap(BridgeService.AssetDeleteProfileSubscription({ subscription_id: subscriptionID })),
    createDataAsset: (kind: string, name: string, sourceURL: string) =>
      unwrap(BridgeService.AssetCreateDataAsset({ kind, name, source_url: sourceURL })),
    listDataAssets: (kind = "") => unwrap(BridgeService.AssetListDataAssets({ kind }), { assets: [] }),
    refreshDataAsset: (assetID: string) => unwrap(BridgeService.AssetRefreshDataAsset({ asset_id: assetID })),
    deleteDataAsset: (assetID: string) => unwrap(BridgeService.AssetDeleteDataAsset({ asset_id: assetID }))
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
