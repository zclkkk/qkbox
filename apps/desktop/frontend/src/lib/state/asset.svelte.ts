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
  subscriptionName = $state("");
  subscriptionURL = $state("");
  assetKind = $state("geo_site");
  assetName = $state("");
  assetURL = $state("");
  busy = $state(false);
  error = $state<string | null>(null);

  async refresh(profileID = "") {
    this.error = null;
    try {
      const [subscriptions, dataAssets] = await Promise.all([
        api.asset.listProfileSubscriptions(profileID),
        api.asset.listDataAssets("")
      ]);
      this.subscriptions = (subscriptions.subscriptions ?? []) as ProfileSubscription[];
      this.dataAssets = (dataAssets.assets ?? []) as DataAsset[];
    } catch (error) {
      this.capture(error);
    }
  }

  clearProfileSubscriptions() {
    this.subscriptions = [];
  }

  async createProfileSubscription(profileID: string) {
    if (!profileID) {
      this.error = "Select a profile before adding a subscription.";
      return;
    }
    await this.withBusy(async () => {
      await api.asset.createProfileSubscription(profileID, this.subscriptionName.trim(), this.subscriptionURL.trim());
      this.subscriptionName = "";
      this.subscriptionURL = "";
      await this.refresh(profileID);
    });
  }

  async refreshProfileSubscription(subscriptionID: string, profileID: string) {
    await this.withBusy(async () => {
      await api.asset.refreshProfileSubscription(subscriptionID);
      await this.refresh(profileID);
    });
  }

  async deleteProfileSubscription(subscriptionID: string, profileID: string) {
    await this.withBusy(async () => {
      await api.asset.deleteProfileSubscription(subscriptionID);
      await this.refresh(profileID);
    });
  }

  async createDataAsset() {
    await this.withBusy(async () => {
      await api.asset.createDataAsset(this.assetKind, this.assetName.trim(), this.assetURL.trim());
      this.assetName = "";
      this.assetURL = "";
      await this.refresh();
    });
  }

  async refreshDataAsset(assetID: string) {
    await this.withBusy(async () => {
      await api.asset.refreshDataAsset(assetID);
      await this.refresh();
    });
  }

  async deleteDataAsset(assetID: string) {
    await this.withBusy(async () => {
      await api.asset.deleteDataAsset(assetID);
      await this.refresh();
    });
  }

  private async withBusy(fn: () => Promise<void>) {
    this.busy = true;
    this.error = null;
    try {
      await fn();
    } catch (error) {
      this.capture(error);
    } finally {
      this.busy = false;
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
