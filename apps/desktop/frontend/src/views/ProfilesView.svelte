<script lang="ts">
  import EmptyState from "../components/shared/EmptyState.svelte";
  import ErrorNotice from "../components/shared/ErrorNotice.svelte";
  import StateBadge from "../components/shared/StateBadge.svelte";
  import { profileState } from "../lib/state/profile.svelte";
</script>

<section class="panel">
  <div class="panel-title">
    <h2>Profiles</h2>
    <button type="button" onclick={() => profileState.refresh()}>Refresh</button>
  </div>
  <ErrorNotice message={profileState.error} />
  <div class="asset-list">
    {#each profileState.profiles as profile}
      <button
        class="profile-row"
        class:active={profileState.selectedProfileID === profile.id}
        type="button"
        onclick={() => profileState.selectProfile(profile.id)}
      >
        <span>
          <strong>{profile.name}</strong>
          <small>{profile.id}</small>
        </span>
        {#if profile.active_snapshot_id}
          <StateBadge state="available" label="active" />
        {/if}
      </button>
    {:else}
      <EmptyState message="No profiles yet" />
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
      {#if profileState.activeSnapshot.content_sha256}
        <div>
          <span class="label">Content</span>
          <strong>{profileState.activeSnapshot.content_sha256.slice(0, 12)}</strong>
        </div>
      {/if}
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
          <span>{snapshot.profile_id}</span>
        </div>
        <StateBadge state={snapshot.id === profileState.activeSnapshot?.id ? "available" : "unavailable"} label={snapshot.id === profileState.activeSnapshot?.id ? "Active" : "Stored"} />
      </div>
    {:else}
      <EmptyState message="No snapshots for selected profile" />
    {/each}
  </div>
</section>
