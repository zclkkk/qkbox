import { api, formatStructuredError, QKBoxApiError } from "../api/client";

export type ValidationDiagnostic = {
  severity: string;
  field?: string;
  message: string;
};

export type ProfileSummary = {
  id: string;
  name: string;
  has_draft?: boolean;
  has_active_snapshot?: boolean;
  active_snapshot_id?: string;
  created_at?: number;
  updated_at?: number;
};

export type ProfileSnapshot = {
  id: string;
  profile_id: string;
  validation_status?: string;
  diagnostics?: ValidationDiagnostic[];
  required_capabilities?: string[];
  created_at: number;
  content_sha256?: string;
};

const defaultProfileContent = `{
  "inbounds": [
    {
      "type": "mixed",
      "tag": "mixed-in",
      "listen": "127.0.0.1",
      "listen_port": 7890
    }
  ],
  "outbounds": [
    {
      "type": "direct",
      "tag": "direct"
    }
  ]
}
`;

class ProfileState {
  profiles = $state<ProfileSummary[]>([]);
  activeProfile = $state<unknown | null>(null);
  activeSnapshot = $state<ProfileSnapshot | null>(null);
  selectedProfileID = $state<string>("");
  selectedProfileName = $state<string>("");
  draftContent = $state(defaultProfileContent);
  savedDraftContent = $state(defaultProfileContent);
  snapshots = $state<ProfileSnapshot[]>([]);
  diagnostics = $state<ValidationDiagnostic[]>([]);
  validationStatus = $state<string>("unknown");
  creatingName = $state("New Profile");
  busy = $state(false);
  error = $state<string | null>(null);

  get draftDirty() {
    return this.draftContent !== this.savedDraftContent;
  }

  async refresh() {
    this.error = null;
    await this.refreshProfiles();
    await this.refreshActive();
  }

  async createProfile() {
    const name = this.creatingName.trim();
    if (!name) {
      this.error = "Profile name is required.";
      return;
    }
    await this.withBusy(async () => {
      const reply = await api.profile.create(name, defaultProfileContent);
      this.creatingName = "New Profile";
      await this.refreshProfiles();
      await this.selectProfile(reply.profile.id);
    });
  }

  async deleteProfile(profileID: string) {
    await this.withBusy(async () => {
      await api.profile.delete(profileID);
      if (this.selectedProfileID === profileID) {
        this.clearSelection();
      }
      await this.refreshProfiles();
      await this.refreshActive();
    });
  }

  async refreshProfiles() {
    try {
      const reply = await api.profile.list();
      this.profiles = (reply.profiles ?? []) as ProfileSummary[];
      if (!this.selectedProfileID && this.profiles.length > 0) {
        await this.selectProfile(this.profiles[0].id);
      } else if (this.selectedProfileID) {
        const selected = this.profiles.find((profile) => profile.id === this.selectedProfileID);
        if (!selected) {
          this.clearSelection();
          if (this.profiles.length > 0) {
            await this.selectProfile(this.profiles[0].id);
          }
          return;
        }
        this.selectedProfileName = selected.name;
        await this.refreshSnapshots();
      }
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
    this.error = null;
    try {
      const reply = await api.profile.get(profileID);
      this.selectedProfileName = reply.profile.name;
      this.draftContent = reply.content || "";
      this.savedDraftContent = this.draftContent;
      this.diagnostics = [];
      this.validationStatus = "unknown";
      await this.refreshSnapshots();
    } catch (error) {
      this.capture(error);
    }
  }

  async saveDraft() {
    if (!this.selectedProfileID) {
      return;
    }
    await this.withBusy(async () => {
      await api.profile.updateDraft(this.selectedProfileID, this.draftContent);
      this.savedDraftContent = this.draftContent;
      await this.refreshProfiles();
    });
  }

  async validateDraft() {
    if (!this.selectedProfileID) {
      return;
    }
    await this.withBusy(async () => {
      if (this.draftDirty) {
        await api.profile.updateDraft(this.selectedProfileID, this.draftContent);
        this.savedDraftContent = this.draftContent;
      }
      const reply = await api.profile.validateDraft(this.selectedProfileID);
      this.validationStatus = reply.diagnostics.status;
      this.diagnostics = reply.diagnostics.entries ?? [];
    });
  }

  async createSnapshot() {
    if (!this.selectedProfileID) {
      return;
    }
    await this.withBusy(async () => {
      if (this.draftDirty) {
        await api.profile.updateDraft(this.selectedProfileID, this.draftContent);
        this.savedDraftContent = this.draftContent;
      }
      const reply = await api.profile.createSnapshot(this.selectedProfileID);
      this.validationStatus = reply.snapshot.validation_status ?? "unknown";
      this.diagnostics = reply.snapshot.diagnostics ?? [];
      await this.refreshSnapshots();
    });
  }

  async activateSnapshot(snapshotID: string) {
    await this.withBusy(async () => {
      await api.profile.activateSnapshot(snapshotID);
      await this.refresh();
    });
  }

  async rollbackToSnapshot(snapshotID: string) {
    await this.withBusy(async () => {
      await api.profile.rollbackToSnapshot(snapshotID);
      await this.refresh();
    });
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

  private clearSelection() {
    this.selectedProfileID = "";
    this.selectedProfileName = "";
    this.draftContent = defaultProfileContent;
    this.savedDraftContent = defaultProfileContent;
    this.snapshots = [];
    this.diagnostics = [];
    this.validationStatus = "unknown";
  }
}

export const profileState = new ProfileState();
