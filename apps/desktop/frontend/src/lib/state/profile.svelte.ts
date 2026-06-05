import { api, formatStructuredError, QKBoxApiError } from "../api/client";

type ProfileSummary = {
  id: string;
  name: string;
  active_snapshot_id?: string;
  updated_at?: number;
};

type ProfileSnapshot = {
  id: string;
  profile_id: string;
  created_at: number;
  content_sha256?: string;
};

class ProfileState {
  profiles = $state<ProfileSummary[]>([]);
  activeProfile = $state<unknown | null>(null);
  activeSnapshot = $state<ProfileSnapshot | null>(null);
  snapshots = $state<ProfileSnapshot[]>([]);
  selectedProfileID = $state<string>("");
  error = $state<string | null>(null);

  async refresh() {
    this.error = null;
    await Promise.all([this.refreshProfiles(), this.refreshActive()]);
  }

  async refreshProfiles() {
    try {
      const reply = await api.profile.list();
      this.profiles = (reply.profiles ?? []) as ProfileSummary[];
      if (!this.selectedProfileID && this.profiles.length > 0) {
        this.selectedProfileID = this.profiles[0].id;
      }
      await this.refreshSnapshots();
    } catch (error) {
      this.capture(error);
    }
  }

  async refreshActive() {
    try {
      const [profile, snapshot] = await Promise.all([api.profile.getActiveProfile(), api.profile.getActiveSnapshot()]);
      this.activeProfile = profile.profile ?? null;
      this.activeSnapshot = (snapshot.snapshot as ProfileSnapshot | null) ?? null;
    } catch (error) {
      this.capture(error);
    }
  }

  async selectProfile(profileID: string) {
    this.selectedProfileID = profileID;
    await this.refreshSnapshots();
  }

  async refreshSnapshots() {
    if (!this.selectedProfileID) {
      this.snapshots = [];
      return;
    }
    try {
      const reply = await api.profile.listSnapshots(this.selectedProfileID);
      this.snapshots = (reply.snapshots ?? []) as ProfileSnapshot[];
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

export const profileState = new ProfileState();
