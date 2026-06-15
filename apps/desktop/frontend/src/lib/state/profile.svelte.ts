import { api, formatStructuredError, QKBoxApiError } from "../api/client";

export type ValidationDiagnostic = {
  severity: string;
  field?: string;
  message: string;
};

export type ProfileSummary = {
  id: string;
  name: string;
  created_at?: number;
  updated_at?: number;
};

export type Profile = ProfileSummary;

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
  activeProfile = $state<Profile | null>(null);
  selectedProfileID = $state<string>("");
  selectedProfileName = $state<string>("");
  content = $state(defaultProfileContent);
  savedContent = $state(defaultProfileContent);
  diagnostics = $state<ValidationDiagnostic[]>([]);
  validationStatus = $state<string>("unknown");
  creatingName = $state("New Profile");
  busy = $state(false);
  error = $state<string | null>(null);

  get contentDirty() {
    return this.content !== this.savedContent;
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
      }
    } catch (error) {
      this.capture(error);
    }
  }

  async refreshActive() {
    try {
      const reply = await api.profile.getActiveProfile();
      this.activeProfile = (reply.profile as Profile | null) ?? null;
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
      this.content = reply.content || "";
      this.savedContent = this.content;
      this.diagnostics = [];
      this.validationStatus = "unknown";
    } catch (error) {
      this.capture(error);
    }
  }

  async saveContent() {
    if (!this.selectedProfileID) {
      return;
    }
    await this.withBusy(async () => {
      await api.profile.saveContent(this.selectedProfileID, this.content);
      this.savedContent = this.content;
      await this.refreshProfiles();
    });
  }

  async validateContent() {
    if (!this.selectedProfileID) {
      return;
    }
    await this.withBusy(async () => {
      const reply = await api.profile.validateContent(this.selectedProfileID, this.content);
      this.validationStatus = reply.diagnostics.status;
      this.diagnostics = reply.diagnostics.entries ?? [];
    });
  }

  async activateProfile(profileID: string) {
    await this.withBusy(async () => {
      await api.profile.activate(profileID);
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

  private clearSelection() {
    this.selectedProfileID = "";
    this.selectedProfileName = "";
    this.content = defaultProfileContent;
    this.savedContent = defaultProfileContent;
    this.diagnostics = [];
    this.validationStatus = "unknown";
  }
}

export const profileState = new ProfileState();
