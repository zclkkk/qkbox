<script lang="ts">
  import { RefreshCw } from "@lucide/svelte";
  import EmptyState from "../components/shared/EmptyState.svelte";
  import ErrorNotice from "../components/shared/ErrorNotice.svelte";
  import IconButton from "../components/shared/IconButton.svelte";
  import StateBadge from "../components/shared/StateBadge.svelte";
  import Toolbar from "../components/shared/Toolbar.svelte";
  import type { HelloReply } from "../lib/api/client";
  import { formatBytes } from "../lib/format";
  import { assetState } from "../lib/state/asset.svelte";
  import { diagnosticsState } from "../lib/state/diagnostics.svelte";

  let { hello }: { hello: HelloReply } = $props();
</script>

<section class="panel diagnostics">
  <h2>Diagnostics</h2>
  <dl>
    <div>
      <dt>App version</dt>
      <dd>{hello.app_version}</dd>
    </div>
    <div>
      <dt>qkboxd version</dt>
      <dd>{hello.qkboxd_version}</dd>
    </div>
    <div>
      <dt>API compatibility</dt>
      <dd>{hello.min_supported_api_version} - {hello.api_version}</dd>
    </div>
    <div>
      <dt>Schema revision</dt>
      <dd>{hello.schema_revision}</dd>
    </div>
    <div>
      <dt>Platform</dt>
      <dd>{hello.platform.os}/{hello.platform.arch}</dd>
    </div>
  </dl>

  <div class="panel-title section-title">
    <h2>Support Bundle</h2>
    <Toolbar>
      <button type="button" onclick={() => diagnosticsState.refresh()}>Refresh diagnostics</button>
      <button type="button" onclick={() => diagnosticsState.createDebugBundle()}>Create debug bundle</button>
    </Toolbar>
  </div>
  <ErrorNotice message={diagnosticsState.error} />
  <ErrorNotice tone="default" message={diagnosticsState.debugBundlePath} />
  {#if diagnosticsState.report}
    <div class="check-list">
      {#each diagnosticsState.report.checks as check}
        <div class="check-row">
          <div>
            <strong>{check.name}</strong>
            {#if check.reason}
              <span>{check.reason}</span>
            {/if}
            {#if check.recovery}
              <span>{check.recovery}</span>
            {/if}
          </div>
          <StateBadge state={check.state} label={check.state} />
        </div>
      {/each}
    </div>
  {/if}

  <div class="panel-title section-title">
    <h2>Data Plane</h2>
    <IconButton label="Refresh asset state" onclick={() => assetState.refresh()}>
      <RefreshCw size={16} />
    </IconButton>
  </div>
  <ErrorNotice message={assetState.error} />
  <h3>Subscriptions</h3>
  <div class="asset-list">
    {#each assetState.subscriptions as sub}
      <div class="asset-row">
        <div>
          <strong>{sub.name}</strong>
          <span>{sub.url}</span>
        </div>
        <StateBadge state={sub.last_status} label={sub.last_status} />
      </div>
      {#if sub.last_error_message}
        <ErrorNotice message={`${sub.last_error_code}: ${sub.last_error_message}`} />
      {/if}
    {:else}
      <EmptyState message="No profile subscriptions" />
    {/each}
  </div>

  <h3>Assets</h3>
  <div class="asset-list">
    {#each assetState.dataAssets as asset}
      <div class="asset-row">
        <div>
          <strong>{asset.name}</strong>
          <span>{asset.kind} / {asset.source_url}</span>
        </div>
        <StateBadge state={asset.status} label={asset.status} />
      </div>
      {#if asset.content_sha256}
        <p class="asset-meta">{asset.content_sha256.slice(0, 12)} / {formatBytes(asset.size_bytes ?? 0)}{asset.version ? ` / ${asset.version}` : ""}</p>
      {/if}
      {#if asset.last_error_message}
        <ErrorNotice message={`${asset.last_error_code}: ${asset.last_error_message}`} />
      {/if}
    {:else}
      <EmptyState message="No cached data assets" />
    {/each}
  </div>
</section>
