<script lang="ts">
  import { Activity, Cpu, FileText, PlugZap, RefreshCw, ShieldCheck } from "@lucide/svelte";
  import type { HelloReply } from "../../lib/api/client";
  import type { Route } from "../../lib/routing";
  import ErrorNotice from "../shared/ErrorNotice.svelte";
  import IconButton from "../shared/IconButton.svelte";

  let {
    route,
    hello,
    loading,
    lastChecked,
    error,
    onrefresh,
    onnavigate,
    children
  }: {
    route: Route;
    hello: HelloReply | null;
    loading: boolean;
    lastChecked: string;
    error: string | null;
    onrefresh: () => void | Promise<void>;
    onnavigate: (route: Route) => void;
    children: import("svelte").Snippet;
  } = $props();

  const navItems: { route: Route; label: string; icon: typeof Activity }[] = [
    { route: "engine", label: "Engine", icon: Activity },
    { route: "profiles", label: "Profiles", icon: FileText },
    { route: "platform", label: "Platform", icon: PlugZap },
    { route: "diagnostics", label: "Diagnostics", icon: Cpu }
  ];
</script>

<main class="shell">
  <aside class="sidebar">
    <div class="brand">
      <ShieldCheck size={24} strokeWidth={1.8} />
      <span>qkbox</span>
    </div>
    <nav class="nav">
      {#each navItems as item}
        {@const Icon = item.icon}
        <button
          class="nav-item"
          class:active={route === item.route}
          aria-pressed={route === item.route}
          type="button"
          onclick={() => onnavigate(item.route)}
        >
          <Icon size={18} />
          {item.label}
        </button>
      {/each}
    </nav>
  </aside>

  <section class="content">
    <header class="topbar">
      <div>
        <h1>Control Plane</h1>
        <p>{loading ? "Refreshing qkboxd handshake..." : `Last checked ${lastChecked}`}</p>
      </div>
      <IconButton label="Refresh qkboxd handshake" onclick={onrefresh} disabled={loading} spinning={loading}>
        <RefreshCw size={18} />
      </IconButton>
    </header>

    <ErrorNotice message={error} />

    {#if hello}
      <section class="summary">
        <div>
          <span class="label">API</span>
          <strong>{hello.api_version}</strong>
        </div>
        <div>
          <span class="label">Schema</span>
          <strong>{hello.schema_revision}</strong>
        </div>
        <div>
          <span class="label">qkboxd</span>
          <strong>{hello.qkboxd_version}</strong>
        </div>
        <div>
          <span class="label">Platform</span>
          <strong>{hello.platform.os}/{hello.platform.arch}</strong>
        </div>
      </section>

      <section class="columns">
        {@render children()}
      </section>
    {:else if !error}
      <section class="notice">
        <span>Connecting to qkboxd...</span>
      </section>
    {/if}
  </section>
</main>
