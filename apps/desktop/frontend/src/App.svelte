<script lang="ts">
  import { onMount } from "svelte";
  import { Activity, CircleAlert, Cpu, PlugZap, RefreshCw, ShieldCheck } from "@lucide/svelte";
  import { BridgeService } from "../bindings/github.com/zclkkk/qkbox/apps/desktop";
  import { type Capability, type HelloReply } from "../bindings/github.com/zclkkk/qkbox/shared/api/models";

  type View = "engine" | "platform" | "diagnostics";

  let loading = $state(true);
  let reply = $state<HelloReply | null>(null);
  let error = $state<string | null>(null);
  let activeView = $state<View>("engine");
  let lastChecked = $state<string>("Never");

  async function bootstrap() {
    loading = true;
    error = null;
    try {
      const result = await BridgeService.Hello();
      if (result.error) {
        error = `${result.error.code}: ${result.error.message}`;
        reply = null;
      } else {
        reply = result.reply as HelloReply;
        lastChecked = new Date().toLocaleTimeString();
      }
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
      reply = null;
    } finally {
      loading = false;
    }
  }

  function show(view: View) {
    activeView = view;
  }

  onMount(() => {
    void bootstrap();
  });
</script>

<main class="shell">
  <aside class="sidebar">
    <div class="brand">
      <ShieldCheck size={24} strokeWidth={1.8} />
      <span>qkbox</span>
    </div>
    <nav class="nav">
      <button class="nav-item" class:active={activeView === "engine"} aria-pressed={activeView === "engine"} type="button" onclick={() => show("engine")}><Activity size={18} />Engine</button>
      <button class="nav-item" class:active={activeView === "platform"} aria-pressed={activeView === "platform"} type="button" onclick={() => show("platform")}><PlugZap size={18} />Platform</button>
      <button class="nav-item" class:active={activeView === "diagnostics"} aria-pressed={activeView === "diagnostics"} type="button" onclick={() => show("diagnostics")}><Cpu size={18} />Diagnostics</button>
    </nav>
  </aside>

  <section class="content">
    <header class="topbar">
      <div>
        <h1>Control Plane</h1>
        <p>{loading ? "Refreshing qkboxd handshake..." : `Last checked ${lastChecked}`}</p>
      </div>
      <button class="icon-button" type="button" aria-label="Refresh qkboxd handshake" onclick={bootstrap} disabled={loading}>
        <span class:spin={loading}>
          <RefreshCw size={18} />
        </span>
      </button>
    </header>

    {#if error}
      <section class="notice error">
        <CircleAlert size={18} />
        <span>{error}</span>
      </section>
    {:else if reply}
      <section class="summary">
        <div>
          <span class="label">API</span>
          <strong>{reply.api_version}</strong>
        </div>
        <div>
          <span class="label">Schema</span>
          <strong>{reply.schema_revision}</strong>
        </div>
        <div>
          <span class="label">qkboxd</span>
          <strong>{reply.qkboxd_version}</strong>
        </div>
        <div>
          <span class="label">Platform</span>
          <strong>{reply.platform.os}/{reply.platform.arch}</strong>
        </div>
      </section>

      <section class="columns">
        {#if activeView === "engine"}
          {@render capabilityList("Runtime", reply.runtime_capabilities)}
        {:else if activeView === "platform"}
          {@render capabilityList("Platform", reply.platform_capabilities)}
        {:else}
          {@render diagnostics(reply)}
        {/if}
      </section>
    {:else}
      <section class="notice">
        <span>Connecting to qkboxd...</span>
      </section>
    {/if}
  </section>
</main>

{#snippet capabilityList(title: string, items: Capability[])}
  <section class="panel">
    <h2>{title}</h2>
    <div class="capabilities">
      {#each items as item}
        <div class="capability">
          <div>
            <strong>{item.name}</strong>
            {#if item.reason}
              <span>{item.reason}</span>
            {/if}
          </div>
          <span class="state" data-state={item.state}>{item.state}</span>
        </div>
      {/each}
    </div>
  </section>
{/snippet}

{#snippet diagnostics(data: HelloReply)}
  <section class="panel diagnostics">
    <h2>Diagnostics</h2>
    <dl>
      <div>
        <dt>App version</dt>
        <dd>{data.app_version}</dd>
      </div>
      <div>
        <dt>qkboxd version</dt>
        <dd>{data.qkboxd_version}</dd>
      </div>
      <div>
        <dt>API compatibility</dt>
        <dd>{data.min_supported_api_version} - {data.api_version}</dd>
      </div>
      <div>
        <dt>Schema revision</dt>
        <dd>{data.schema_revision}</dd>
      </div>
      <div>
        <dt>Platform</dt>
        <dd>{data.platform.os}/{data.platform.arch}</dd>
      </div>
    </dl>
  </section>
{/snippet}
