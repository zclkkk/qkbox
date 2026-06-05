<script lang="ts">
  import { onMount } from "svelte";
  import ConfirmDialog from "../components/shared/ConfirmDialog.svelte";
  import EmptyState from "../components/shared/EmptyState.svelte";
  import ErrorNotice from "../components/shared/ErrorNotice.svelte";
  import StateBadge from "../components/shared/StateBadge.svelte";
  import { formatBytes } from "../lib/format";
  import { assetState } from "../lib/state/asset.svelte";
  import { profileState } from "../lib/state/profile.svelte";

  let deleteProfileID = $state<string | null>(null);
  let JsonEditor = $state<typeof import("../components/profile/JsonEditor.svelte").default | null>(null);
  let subscriptionProfileID = "";

  onMount(() => {
    void import("../components/profile/JsonEditor.svelte").then((module) => {
      JsonEditor = module.default;
    });
  });

  $effect(() => {
    const profileID = profileState.selectedProfileID;
    if (!profileID) {
      subscriptionProfileID = "";
      assetState.clearProfileSubscriptions();
      return;
    }
    if (profileID === subscriptionProfileID) {
      return;
    }
    subscriptionProfileID = profileID;
    void assetState.refresh(profileID);
  });

  async function createProfile() {
    await profileState.createProfile();
    if (profileState.selectedProfileID) {
      await assetState.refresh(profileState.selectedProfileID);
    }
  }

  async function selectProfile(profileID: string) {
    await profileState.selectProfile(profileID);
    await assetState.refresh(profileID);
  }

  async function refreshAll() {
    await profileState.refresh();
    await assetState.refresh(profileState.selectedProfileID);
  }

  async function deleteSelectedProfile() {
    if (!deleteProfileID) {
      return;
    }
    await profileState.deleteProfile(deleteProfileID);
    await assetState.refresh(profileState.selectedProfileID);
    deleteProfileID = null;
  }
</script>

<section class="panel">
  <div class="panel-title">
    <h2>Profiles</h2>
    <button type="button" onclick={refreshAll} disabled={profileState.busy}>Refresh</button>
  </div>
  <ErrorNotice message={profileState.error} />
  <div class="form-grid compact-form">
    <label>
      <span class="label">Name</span>
      <input bind:value={profileState.creatingName} />
    </label>
    <button type="button" onclick={createProfile} disabled={profileState.busy}>Create</button>
  </div>
  <div class="asset-list">
    {#each profileState.profiles as profile}
      <div class="profile-row" class:active={profileState.selectedProfileID === profile.id}>
        <button type="button" onclick={() => selectProfile(profile.id)}>
          <span>
            <strong>{profile.name}</strong>
            <small>{profile.id}</small>
          </span>
        </button>
        <div class="button-row">
          {#if profile.active_snapshot_id}
            <StateBadge state="available" label="active" />
          {/if}
          <button type="button" onclick={() => (deleteProfileID = profile.id)} disabled={profileState.busy || !!profile.active_snapshot_id}>Delete</button>
        </div>
      </div>
    {:else}
      <EmptyState message="No profiles yet" />
    {/each}
  </div>
</section>

<section class="panel editor-panel">
  <div class="panel-title">
    <h2>Draft JSON</h2>
    <div class="button-row">
      <button type="button" onclick={() => profileState.saveDraft()} disabled={!profileState.selectedProfileID || profileState.busy || !profileState.draftDirty}>Save draft</button>
      <button type="button" onclick={() => profileState.validateDraft()} disabled={!profileState.selectedProfileID || profileState.busy}>Validate</button>
      <button type="button" onclick={() => profileState.createSnapshot()} disabled={!profileState.selectedProfileID || profileState.busy}>Create snapshot</button>
    </div>
  </div>
  {#if profileState.selectedProfileID}
    <div class="editor-meta">
      <span class="label">{profileState.selectedProfileName}</span>
      {#if profileState.draftDirty}
        <StateBadge state="pending" label="unsaved" />
      {/if}
    </div>
    {#if JsonEditor}
      <JsonEditor value={profileState.draftContent} onchange={(value) => (profileState.draftContent = value)} />
    {:else}
      <EmptyState message="Loading editor..." />
    {/if}
  {:else}
    <EmptyState message="Select or create a profile" />
  {/if}
</section>

<section class="panel">
  <h2>Validation</h2>
  <div class="panel-actions">
    <span class="label">Status</span>
    <StateBadge state={profileState.validationStatus} label={profileState.validationStatus} />
  </div>
  <div class="check-list">
    {#each profileState.diagnostics as diagnostic}
      <div class="check-row">
        <div>
          <strong>{diagnostic.field || diagnostic.severity}</strong>
          <span>{diagnostic.message}</span>
        </div>
        <StateBadge state={diagnostic.severity} label={diagnostic.severity} />
      </div>
    {:else}
      <EmptyState message="No diagnostics yet" />
    {/each}
  </div>
</section>

<section class="panel">
  <h2>Active Runtime Target</h2>
  {#if profileState.activeSnapshot}
    <div class="metrics">
      <div>
        <span class="label">Snapshot</span>
        <strong>{profileState.activeSnapshot.id}</strong>
      </div>
      <div>
        <span class="label">Profile</span>
        <strong>{profileState.activeSnapshot.profile_id}</strong>
      </div>
      <div>
        <span class="label">Validation</span>
        <StateBadge state={profileState.activeSnapshot.validation_status} label={profileState.activeSnapshot.validation_status ?? "unknown"} />
      </div>
    </div>
  {:else}
    <EmptyState message="No active snapshot" />
  {/if}
</section>

<section class="panel diagnostics">
  <h2>Snapshots</h2>
  <div class="asset-list">
    {#each profileState.snapshots as snapshot}
      <div class="asset-row">
        <div>
          <strong>{snapshot.id}</strong>
          <span>{new Date(snapshot.created_at).toLocaleString()}</span>
        </div>
        <div class="button-row">
          <StateBadge state={snapshot.validation_status} label={snapshot.validation_status ?? "unknown"} />
          {#if snapshot.id === profileState.activeSnapshot?.id}
            <StateBadge state="available" label="active" />
          {:else}
            <button type="button" onclick={() => profileState.activateSnapshot(snapshot.id)} disabled={profileState.busy}>Activate</button>
            <button type="button" onclick={() => profileState.rollbackToSnapshot(snapshot.id)} disabled={profileState.busy}>Rollback</button>
          {/if}
        </div>
      </div>
    {:else}
      <EmptyState message="No snapshots for selected profile" />
    {/each}
  </div>
</section>

<section class="panel diagnostics">
  <h2>Subscriptions</h2>
  <ErrorNotice message={assetState.error} />
  <div class="form-grid subscription-form">
    <label>
      <span class="label">Display name</span>
      <input bind:value={assetState.subscriptionName} />
    </label>
    <label>
      <span class="label">URL</span>
      <input bind:value={assetState.subscriptionURL} />
    </label>
    <button type="button" onclick={() => assetState.createProfileSubscription(profileState.selectedProfileID)} disabled={assetState.busy || !profileState.selectedProfileID}>Add</button>
  </div>
  <div class="asset-list">
    {#each assetState.subscriptions as sub}
      <div class="asset-row">
        <div>
          <strong>{sub.name}</strong>
          <span>{sub.url}</span>
          {#if sub.content_sha256}
            <span>{sub.content_sha256.slice(0, 12)}</span>
          {/if}
          {#if sub.last_error_message}
            <span>{sub.last_error_code}: {sub.last_error_message}</span>
          {/if}
        </div>
        <div class="button-row">
          <StateBadge state={sub.last_status} label={sub.last_status} />
          <button type="button" onclick={() => assetState.refreshProfileSubscription(sub.id, profileState.selectedProfileID)} disabled={assetState.busy}>Refresh</button>
          <button type="button" onclick={() => assetState.deleteProfileSubscription(sub.id, profileState.selectedProfileID)} disabled={assetState.busy}>Delete</button>
        </div>
      </div>
    {:else}
      <EmptyState message="No subscriptions for selected profile" />
    {/each}
  </div>
</section>

<section class="panel diagnostics">
  <h2>Data Assets</h2>
  <div class="form-grid asset-form">
    <label>
      <span class="label">Kind</span>
      <select bind:value={assetState.assetKind}>
        <option value="geo_site">GeoSite</option>
        <option value="geo_ip">GeoIP</option>
        <option value="rule_set">Rule Set</option>
        <option value="srsc">SRSC</option>
      </select>
    </label>
    <label>
      <span class="label">Name</span>
      <input bind:value={assetState.assetName} />
    </label>
    <label>
      <span class="label">Source URL</span>
      <input bind:value={assetState.assetURL} />
    </label>
    <button type="button" onclick={() => assetState.createDataAsset()} disabled={assetState.busy}>Add</button>
  </div>
  <div class="asset-list">
    {#each assetState.dataAssets as asset}
      <div class="asset-row">
        <div>
          <strong>{asset.name}</strong>
          <span>{asset.kind} / {asset.source_url}</span>
          {#if asset.content_sha256}
            <span>{asset.content_sha256.slice(0, 12)} / {formatBytes(asset.size_bytes ?? 0)}{asset.version ? ` / ${asset.version}` : ""}</span>
          {/if}
          {#if asset.last_error_message}
            <span>{asset.last_error_code}: {asset.last_error_message}</span>
          {/if}
        </div>
        <div class="button-row">
          <StateBadge state={asset.status} label={asset.status} />
          <button type="button" onclick={() => assetState.refreshDataAsset(asset.id)} disabled={assetState.busy}>Refresh</button>
          <button type="button" onclick={() => assetState.deleteDataAsset(asset.id)} disabled={assetState.busy}>Delete</button>
        </div>
      </div>
    {:else}
      <EmptyState message="No data assets" />
    {/each}
  </div>
</section>

<ConfirmDialog
  open={deleteProfileID !== null}
  title="Delete profile"
  message="Delete this profile and its draft data? Active profiles cannot be deleted."
  confirmLabel="Delete"
  onconfirm={deleteSelectedProfile}
  oncancel={() => (deleteProfileID = null)}
/>
