# qkbox Product Architecture Plan

This plan moves qkbox from a working runtime prototype toward a mature
cross-platform client without changing the core product boundary:

```text
Wails GUI -> qkboxd product API -> RuntimeOwner -> sing-box runtime boundary
```

The plan is intentionally target-shaped. It does not preserve historical phase
structure, and it does not introduce temporary compatibility layers that later
milestones are expected to delete.

## Non-Negotiable Boundaries

- qkbox product APIs never expose sing-box, sing-tun, Clash, or platform-private
  DTOs.
- `internal/singboxadapter` remains the only Go package allowed to import
  `github.com/sagernet/sing` or `github.com/sagernet/sing-box`.
- Observability remains qkbox-owned. Do not enable or depend on Clash HTTP
  external controller as the product observability path.
- `StructuredError` keeps rich fields such as detail, source, recoverability,
  and user action. Frontend should consume them better instead of deleting them.
- qkboxd IPC and provider IPC may share low-level frame and transport code, but
  they keep separate product wrappers and semantics.
- The privileged provider is installed or repaired through explicit platform
  setup UI near machine-network/TUN controls. Installers may carry the provider
  binary, but must not silently opt the user into high-privilege behavior.
- System proxy remains a qkbox-native snapshot/restore capability because it is
  OS user-setting ownership, not sing-box runtime ownership.
- macOS machine-network mode uses the NetworkExtension runtime container path,
  not root route hacks.
- `.temp/*` directories are reference-only. They are never import, replace, or
  source-copy targets.

## Milestones

| Milestone | Title | Primary outcome |
|---|---|---|
| 1 | [Backend Product Boundary Hardening](milestone-1-backend-boundary.md) | Domain services, thin IPC handlers, shared IPC foundation, unified events |
| 2 | [Frontend Shell And State Architecture](milestone-2-frontend-architecture.md) | Component tree, domain stores, typed bridge client |
| 3 | [Profile And Configuration Authoring](milestone-3-profile-config-authoring.md) | Usable profile, subscription, asset, validation, and snapshot workflows |
| 4 | [Engine Observability UX](milestone-4-engine-observability-ux.md) | Mature runtime controls, logs, traffic, connections, groups, URLTest |
| 5 | [Platform Provider UX](milestone-5-platform-provider-ux.md) | Provider install/repair/status flows and machine-network mode UX |
| 6 | [Packaging And Release Readiness](milestone-6-packaging-release.md) | Installers, debug bundle polish, platform packaging, release checks |

## Execution Rules

Each milestone must be reviewed before commit. A milestone is acceptable only
when it improves the final architecture shape without adding future refactor
cost.

For every milestone:

- Do not implement future milestones for local completeness.
- Do not use `.temp/*` as code, module replacement, or generated source.
- Preserve existing working capabilities unless the milestone explicitly changes
  their contract.
- Run the relevant verification commands and document anything that could not
  be validated on the current platform.

Default verification:

```text
go test -tags with_clash_api ./...
go vet -tags with_clash_api ./...
npm run check
git diff --check
```
