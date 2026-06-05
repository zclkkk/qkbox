<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import AppShell from "./components/layout/AppShell.svelte";
  import { router } from "./lib/routing.svelte";
  import { appState } from "./lib/state/app.svelte";
  import { runtimeEvents } from "./lib/state/runtime-events.svelte";
  import DiagnosticsView from "./views/DiagnosticsView.svelte";
  import EngineView from "./views/EngineView.svelte";
  import PlatformView from "./views/PlatformView.svelte";
  import ProfilesView from "./views/ProfilesView.svelte";

  onMount(() => {
    const stopRouting = router.start();
    let stopEvents: (() => Promise<void>) | null = null;

    void appState.bootstrap();
    void runtimeEvents.start().then((stop) => {
      stopEvents = stop;
    });

    return () => {
      stopRouting();
      if (stopEvents) {
        void stopEvents();
      }
    };
  });
</script>

<AppShell
  route={router.current}
  hello={appState.hello}
  loading={appState.loading}
  lastChecked={appState.lastChecked}
  error={appState.error}
  onrefresh={() => appState.bootstrap()}
  onnavigate={(route) => router.navigate(route)}
>
  {#if router.current === "engine"}
    <EngineView />
  {:else if router.current === "profiles"}
    <ProfilesView />
  {:else if router.current === "platform"}
    <PlatformView />
  {:else if appState.hello}
    <DiagnosticsView hello={appState.hello} />
  {/if}
</AppShell>
