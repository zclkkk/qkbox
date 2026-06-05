# Milestone 3: Profile And Configuration Authoring

## Objective

Make profile, subscription, data asset, validation, and snapshot workflows
usable from the GUI. This is the first milestone that turns the backend profile
plane into a normal client workflow.

## Decisions

- Keep qkbox public APIs product-shaped.
- Keep raw sing-box JSON as the first authoring surface.
- Add helpers only when they improve a real workflow.
- Do not create a string-template system before the product import/compiler
  model is settled.
- Profile subscription refresh writes draft content only. It never mutates the
  active snapshot or running runtime.

## Work Items

### 1. Profile List

Implement a profile list with:

- create profile.
- delete profile with confirmation.
- select profile.
- active snapshot indicator.
- created/updated timestamps.

### 2. Profile Editor

Implement a JSON editor for selected profile draft content.

Requirements:

- syntax highlighting.
- save draft.
- validate draft.
- diagnostics panel with errors and warnings.
- clear display of validation source and recoverable actions.

CodeMirror is acceptable if it is the smallest robust way to provide JSON
editing. Do not build an ad hoc editor.

### 3. Snapshot History

Implement snapshot workflow:

- create snapshot from draft.
- list snapshots.
- activate snapshot.
- rollback to snapshot.
- show active snapshot and validation status.
- show required runtime/platform capabilities.

### 4. Subscription Management

Implement profile subscription management:

- add subscription URL and display name.
- list subscriptions for selected profile.
- refresh subscription.
- show last checked time, status, content SHA, and backend errors.
- delete subscription.

Manual refresh is enough for this milestone.

### 5. Data Asset Management

Implement data asset management:

- add asset by kind and source URL.
- list assets.
- refresh asset.
- show status, version, size, content SHA, and backend errors.
- delete asset.

Asset refresh validates and caches metadata. It does not mutate runtime state.

## Out Of Scope

- Visual rule builder.
- Subscription-to-sing-box compiler beyond existing backend behavior.
- Runtime reload after subscription refresh.
- Platform setup flows.

## Verification

- `npm run check`
- `npm run build`
- `go test -tags with_clash_api ./...`
- Create profile -> edit draft -> validate -> create snapshot -> activate.
- Refresh a subscription and confirm only draft content changes.
- Refresh a data asset and confirm cache metadata updates.
- Engine mutation guards still block active snapshot changes while runtime runs.

## Acceptance Criteria

- A user can create and activate a listener-bearing profile without raw IPC.
- Validation diagnostics are understandable and preserve structured backend
  detail.
- Subscription and data asset failures are visible and actionable.
- No future compiler/import model is faked by temporary templates.

