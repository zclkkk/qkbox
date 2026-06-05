# qkbox Implementation Roadmap

This roadmap is the current target implementation sequence. Each milestone must
land inside the architecture described in `docs/ARCHITECTURE.md`; no milestone
may introduce a temporary runtime path that will later be removed.

## Execution Rules

1. Preserve product, runtime owner, platform, and type boundaries.
2. Prefer a smaller imperfect product slice over a polished workaround.
3. Do not expose sing-box, sing-tun, Clash, or platform-internal DTOs in public
   APIs, shared models, persistence, or frontend code.
4. Do not add CLI-spawn, sidecar, or FFI runtime paths.
5. If a milestone touches platform state, it must define ownership, cleanup,
   repair, and diagnostics in the same milestone.
6. If a milestone adds public API, add contract tests.
7. If a milestone changes runtime state, verify reload and rollback behavior.
8. If a milestone needs upstream network behavior, use official SagerNet
   components instead of reimplementing that behavior in qkbox.

## Current Baseline

The product already has:

```text
Wails desktop shell
user-scope qkboxd
authenticated local IPC
version and capability handshake
profile draft and encrypted content persistence
snapshot lifecycle and active snapshot selection
embedded local sing-box runtime
runtime status/log/traffic/connection/group/URLTest observability
system proxy snapshot/apply/restore ownership
privileged provider auth/status/prepare/repair shell
Windows provider-hosted machine network runtime
Linux provider-hosted machine network runtime
provider runtime owner lock and stale owner repair
provider runtime status/log/traffic/connection/group/URLTest bridge
Apple NetworkExtension runtime owner and capability boundary
reload with validation, prepare, rollback, and degradation reporting
remote profile subscription metadata and refresh
content-addressed data asset cache
rule-set and geo asset metadata, validation, and diagnostics
```

The next work is release diagnostics and recovery.

## Milestone: Release, Diagnostics, And Recovery

### Goal

Make qkbox supportable as an installed desktop product.

### Work

```text
Windows NSIS packaging
Linux DEB packaging
macOS packaging direction
debug bundle
provider/runtime version diagnostics
stale owner repair UI
schema and compatibility diagnostics
full product update coordination
```

### Acceptance

```text
installed product can be diagnosed without exposing secrets
provider/runtime version mismatch is visible
cleanup failures have repair actions or clear instructions
updates replace the whole product instead of hot-swapping individual binaries
```

## Global Verification

Every milestone must preserve:

```text
go test -tags with_clash_api ./...
go vet -tags with_clash_api ./...
npm run check
architecture import tests
public contract tests for changed APIs
```
