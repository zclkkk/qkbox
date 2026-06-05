# Phase 5: Engine & Observability UI

> Depends on Phase 3 (frontend components + state management).

## 5.1 Engine Panel

**Create** `components/engine/EnginePanel.svelte`

- State badge (IDLE=gray, STARTING=yellow, STARTED=green, STOPPING=yellow, FATAL=red)
- Start button: disabled when STARTED/STARTING/STOPPING/FATAL
- Stop button: disabled when IDLE/UNINITIALIZED/STARTING/STOPPING
- Reload button: dropdown to select target snapshot, shows reload outcome
- Active snapshot ID display
- Error display from `engineStatus.last_error_message`
- Uptime counter (from `started_at`, updates every second)

Integrates: `engineState.start()`, `engineState.stop()`, `engineState.reload()`

## 5.2 Traffic Panel

**Create** `components/engine/TrafficPanel.svelte`

- 4 metric cards: Upload Total, Download Total, Upload Rate, Download Rate
- Rate shown as `/s`, totals formatted with `formatBytes()`
- When engine not started: "Traffic source unavailable"
- When engine started but no data yet: "Waiting for traffic source"

Data source: `runtimeEvents.traffic` (updated every second from event bridge)

**Optional enhancement**: Add a rate history chart (SVG-based, last 60 seconds). Use a simple SVG polyline, no chart library needed:

```typescript
// Keep last 60 data points
let rateHistory = $state<{up: number[], down: number[]}>({up: [], down: []});
$effect(() => {
    if (runtimeEvents.traffic) {
        rateHistory.up = [...rateHistory.up.slice(-59), runtimeEvents.traffic.upload_rate];
        rateHistory.down = [...rateHistory.down.slice(-59), runtimeEvents.traffic.download_rate];
    }
});
```

SVG chart: 200x60 viewport, two polylines (up=blue, down=green), no axis labels needed.

## 5.3 Connection Panel

**Create** `components/engine/ConnectionPanel.svelte`

- Table with columns: Host/Destination, Network, Outbound, Upload, Download
- Sortable by any column (click header to toggle)
- Filter input: text search against host/destination
- Close button per row
- "Close All" button in panel header
- Empty state: "No active connections"

Data source: `runtimeEvents.connections`

## 5.4 Log Panel

**Create** `components/engine/LogPanel.svelte`

- Auto-scrolling list of log entries
- Each entry: timestamp, level badge (color-coded), source, message
- Level filter: checkboxes for error/warn/info/debug/trace
- Source filter: dropdown populated from unique sources in log buffer
- Text search input
- Max 128 entries displayed (ring buffer in events state)

Data source: `runtimeEvents.logs`

Level colors:
- error → red
- warn → orange
- info → blue
- debug → gray
- trace → dim gray

## 5.5 Group Panel

**Create** `components/engine/GroupPanel.svelte`

- Cards for each outbound group
- Group header: tag, type (selector/urltest), selected outbound
- **Selector groups**: clickable outbound list, active outbound highlighted, click to switch
- **URLTest groups**: outbound list + "URLTest" button
  - URLTest shows results: outbound tag + delay (ms) for each
  - Failed outbounds shown in red
- Empty state: "No runtime groups"

Integrates: `engineState.selectOutbound()`, `engineState.urlTest()`

## 5.6 EngineView

**Create** `views/EngineView.svelte`

Layout:
```
┌──────────────┬──────────────┐
│ Engine Panel │ Traffic Panel│
├──────────────┴──────────────┤
│ Outbound Groups             │
├─────────────────────────────┤
│ Connections                 │
├─────────────────────────────┤
│ Logs                        │
└─────────────────────────────┘
```

## Verification

- [ ] Engine start/stop buttons work, state badge updates in real-time
- [ ] Traffic metrics update every second when engine is running
- [ ] Connection table shows active connections with live upload/download
- [ ] Can close individual and all connections
- [ ] Log viewer shows logs with level filtering
- [ ] Selector group: click outbound → engine switches → UI updates
- [ ] URLTest group: click URLTest → results appear with delays
- [ ] All panels show appropriate empty/offline states
