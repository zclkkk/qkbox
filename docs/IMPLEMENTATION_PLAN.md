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
reload with validation, prepare, rollback, and degradation reporting
```

The next work is a foundation refactor, not a feature layer.

## Milestone: SagerNet-First Foundation Refactor

### Goal

Make the repository look as if official SagerNet components were considered from
the first architecture pass.

### Work

```text
rewrite docs into target-shape architecture and roadmap
make RuntimeOwner the explicit engine runtime abstraction
inject runtime owner factory into EngineController
keep local embedded owner behavior unchanged
keep provider-hosted runtime unimplemented until the machine-network milestone
keep sing-box/sing imports confined to internal/singboxadapter
```

### Acceptance

```text
docs describe sing-box / sing-tun / sing-dns / sing-geosite / sing-geoip / srsc adoption
docs do not frame current architecture as an early-stage historical scaffold
EngineController is not directly bound to singboxadapter.NewAdapter
existing local runtime, reload, observability, system proxy, and provider status behavior still pass
```

## Milestone: Windows Provider-Hosted Machine Network Mode

### Goal

Deliver the first machine-level network mode on the platform available for local
development, while preserving the three-platform target shape.

### Work

```text
classify snapshots that require machine network mode
select provider-hosted RuntimeOwner for those snapshots
extend provider IPC with runtime start/stop/status and event bridge
run embedded sing-box inside the provider-hosted owner
let sing-box/sing-tun perform Wintun, route, DNS, and WFP mechanics
store provider owner lock and stale runtime state only
surface NETWORK_MODE_OWNED_BY_ANOTHER_SESSION and repairable stale state
keep qkboxd user-scope and keep user data out of provider storage
```

### Acceptance

```text
Windows machine network runtime starts from an active snapshot
only one owner can hold machine network mode
clean stop releases owner state
stale state is diagnosable and repairable
runtime events still reach GUI through qkbox product events
provider never persists decrypted config or secrets
```

## Milestone: Apple NetworkExtension Runtime Container

### Goal

Implement macOS VPN/TUN mode through the native Apple runtime container shape.

### Work

```text
introduce NetworkExtension RuntimeOwner
bridge qkboxd product commands to the extension container
run embedded sing-box through Apple-compatible sing-box/sing-tun paths
map extension status and failure reasons into qkbox diagnostics
keep system proxy as qkbox native snapshot/restore behavior
```

### Acceptance

```text
macOS TUN/VPN mode does not use root route hacks
extension runtime is selected only for snapshots requiring that mode
GUI remains unaware of process/container details
cleanup and degraded states are product-visible
```

## Milestone: Linux Provider-Hosted Machine Network Mode

### Goal

Implement Linux machine network mode with a formal privileged runtime container.

### Work

```text
use provider-hosted RuntimeOwner for machine network snapshots
run embedded sing-box inside the privileged provider
let sing-box/sing-tun perform TUN, route, DNS, and nftables mechanics
integrate systemd/root helper or polkit-class authorization
store owner lock and stale runtime state only
map platform support reasons into qkbox capabilities
```

### Acceptance

```text
Linux machine network mode uses official sing-box/sing-tun networking
provider owner state is exclusive, cleanable, and repairable
qkbox does not implement its own route or DNS engine
unsupported environments return structured reasons
```

## Milestone: Data Asset And Subscription Plane

### Goal

Support data refresh independently from binary/runtime updates while preserving
snapshot/reload ownership.

### Work

```text
remote profile content update
subscription metadata and update policy
rule-set asset cache
geo asset cache
asset validation and version diagnostics
evaluate sing-geosite, sing-geoip, and srsc for asset workflows
coordinate runtime changes through draft/snapshot/reload only
```

### Acceptance

```text
asset updates never replace binaries
asset updates do not directly mutate active runtime
failed asset updates do not corrupt active snapshots
runtime sees asset changes only after product-approved snapshot/reload
```

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
