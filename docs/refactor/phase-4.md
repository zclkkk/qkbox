# Phase 4: Profile & Configuration Management UI

> Depends on Phase 3 (frontend components + state management).

## 4.1 Profile List

**Create** `components/profile/ProfileList.svelte`

- Table of profiles: name, created/updated dates, active snapshot indicator
- Actions: Create (modal with name input), Delete (confirm dialog), Select (loads editor)
- Integrates with `profileState.refresh()`, `profileState.create()`, `profileState.delete()`

## 4.2 Profile Editor

**Create** `components/profile/ProfileEditor.svelte`

Split-pane layout:
- **Left**: JSON editor with syntax highlighting
- **Right**: Validation diagnostics panel (errors/warnings with field references)

**JSON Editor**: Integrate `codemirror` (`@codemirror/lang-json` + `@codemirror/theme-one-dark`).
- JSON syntax highlighting
- Basic JSON validation (syntax errors shown inline)
- Large file support (CodeMirror handles this natively)

**Add to frontend deps:**
```bash
npm install codemirror @codemirror/lang-json @codemirror/theme-one-dark @codemirror/view @codemirror/state
```

**Actions:**
- Save Draft → `api.profile.updateDraft()`
- Validate → `api.profile.validate()` → show diagnostics in right pane
- Create Snapshot → `api.profile.createSnapshot()`
- Activate Snapshot → `api.profile.activateSnapshot()`

## 4.3 Snapshot History

**Create** `components/profile/SnapshotHistory.svelte`

- Table: snapshot ID, created date, validation status, required capabilities
- Actions: Activate (sets as active), Rollback (alias for Activate)
- Shows which snapshot is currently active

## 4.4 Subscription Management

**Create** `components/profile/SubscriptionPanel.svelte`

- Table of subscriptions for selected profile
- Add form: URL input, name, update policy (dropdown, only "manual" for now)
- Each row: name, URL (truncated), last status badge, last checked/updated timestamps, content SHA256
- Actions: Refresh, Delete
- Status indicators: pending (yellow), updated (green), failed (red)

## 4.5 Data Asset Management

**Create** `components/profile/DataAssetPanel.svelte`

- Table of all data assets
- Add form: kind dropdown (rule-set, geosite, geoip, srsc), source URL, name
- Each row: name, kind, status badge, version, size, content SHA256
- Actions: Refresh, Delete
- Size formatting via `formatBytes()`

## 4.6 ProfilesView

**Create** `views/ProfilesView.svelte`

Assembles: ProfileList + ProfileEditor + SnapshotHistory + SubscriptionPanel + DataAssetPanel

Layout:
```
┌─────────────────┬──────────────────────────┐
│ Profile List    │ Profile Editor           │
│                 │ (CodeMirror + diagnostics)│
├─────────────────┤                          │
│ Snapshot History│                          │
├─────────────────┴──────────────────────────┤
│ Subscriptions │ Data Assets                │
└───────────────────────────────────────────┘
```

## Verification

- [ ] Create a profile → appears in list
- [ ] Edit profile JSON in CodeMirror → syntax highlighted
- [ ] Validate → diagnostics shown in right pane with error/warning indicators
- [ ] Create snapshot → appears in snapshot history
- [ ] Activate snapshot → active indicator updates
- [ ] Add subscription URL → refresh → status updates
- [ ] Add data asset (geoip URL) → refresh → size/version shown
- [ ] Delete profile/snapshot/subscription/asset with confirmation
