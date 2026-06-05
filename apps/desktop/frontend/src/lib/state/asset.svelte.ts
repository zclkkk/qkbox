import { api, formatStructuredError, QKBoxApiError } from "../api/client";

export type ProfileSubscription = {
  id: string;
  profile_id: string;
  name: string;
  url: string;
  update_policy: string;
  last_status: string;
  last_error_code?: string;
  last_error_message?: string;
  last_checked_at?: number;
  last_updated_at?: number;
  content_sha256?: string;
};

export type DataAsset = {
  id: string;
  kind: string;
  name: string;
  source_url: string;
  status: string;
  version?: string;
  content_sha256?: string;
  size_bytes?: number;
  last_error_code?: string;
  last_error_message?: string;
};

class AssetState {
  subscriptions = $state<ProfileSubscription[]>([]);
  dataAssets = $state<DataAsset[]>([]);
  error = $state<string | null>(null);

  async refresh() {
    this.error = null;
    try {
      const [subscriptions, dataAssets] = await Promise.all([
        api.asset.listProfileSubscriptions(""),
        api.asset.listDataAssets("")
      ]);
      this.subscriptions = (subscriptions.subscriptions ?? []) as ProfileSubscription[];
      this.dataAssets = (dataAssets.assets ?? []) as DataAsset[];
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

export const assetState = new AssetState();
