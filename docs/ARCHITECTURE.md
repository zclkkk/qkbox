# qkbox Architecture

This document is the target architecture baseline for qkbox. It defines product
ownership, runtime ownership, platform boundaries, and the official upstream
components qkbox relies on.

## Product Shape

qkbox is a Windows, macOS, and Linux desktop GUI client:

```text
qkbox = Wails GUI + user-scope qkboxd + RuntimeOwner + Platform Capability Boundary
```

The GUI is only the control surface. It never owns runtime truth, profile data,
secrets, TUN devices, route state, DNS state, firewall state, or operating system
proxy state.

## Non-Negotiable Boundaries

1. GUI consumes qkbox product APIs only.
2. `qkboxd` owns user data, profile orchestration, snapshot orchestration,
   reload semantics, and product diagnostics.
3. Runtime ownership is explicit. `qkboxd` is the default RuntimeOwner, but
   machine-level network modes can use a provider-hosted runtime container.
4. Privileged or platform-scoped mutation stays behind the platform capability
   boundary.
5. Public IPC, shared models, persistence, and frontend code never expose
   sing-box, sing-tun, Clash, or platform-internal DTOs.
6. Machine-level network mode is exclusive for one qkbox owner at a time.
7. Runtime changes happen through snapshot/reload coordination.
8. Data assets can update independently, but asset updates do not directly
   mutate the active runtime.
9. Cleanup and repair failures must be visible. qkbox does not hide degraded
   platform state.

## Official Components

qkbox uses official SagerNet components as the network substrate. qkbox does not
rebuild this substrate at the product layer.

| Component | qkbox use | qkbox must not do |
| --- | --- | --- |
| sing-box | Only runtime core. `internal/singboxadapter` embeds and adapts it. | Spawn sing-box CLI as product runtime, expose sing-box structs in public APIs, or build another proxy core. |
| sing-tun | Used through sing-box for transparent proxy, TUN, route, strict route, DNS hijack, Wintun, WFP, nftables, and platform network mechanics. | Reimplement Wintun handling, route tables, WFP, nftables, DNS hijack, or TUN packet stacks. |
| sing-dns | Used through sing-box DNS runtime. | Build a qkbox resolver/cache/DoH/DoT engine. |
| sing-geosite / sing-geoip | Data assets for rule-set and geo matching. | Invent a competing geo database format. |
| srsc | Candidate rule-set converter for the data asset plane. | Build a full custom rule-set converter before evaluating srsc. |
| sing-box Apple clients / NetworkExtension path | Reference for formal Apple TUN/VPN runtime shape. | Ship macOS TUN mode as a temporary root route hack. |

`.temp/sing-box`, `.temp/sing-tun`, and other upstream checkouts are reference
material only. Product code imports upstream modules through Go module
dependencies, never through local replace sources.

## Logical Components

### Wails GUI

The GUI runs with normal user privileges.

It is responsible for:

```text
profile and subscription UI
runtime dashboard
logs, traffic, connection, group, and URLTest views
system proxy toggle UI
network mode status UI
permission and repair entry points
update and diagnostics entry points
```

It is not responsible for:

```text
runtime lifecycle
runtime config compilation
secret persistence
system proxy implementation
TUN, route, DNS, or firewall mutation
provider IPC
runtime owner selection
```

### qkboxd

`qkboxd` runs in user scope. It owns the product control plane and user data.

It is responsible for:

```text
profile persistence
encrypted content storage
snapshot lifecycle
active snapshot selection
validation and capability classification
reload and rollback semantics
runtime owner selection
runtime status machine
runtime event fan-out
system proxy ownership coordination
provider status and repair coordination
structured product diagnostics
```

It must not run as root, LocalSystem, or a machine service in order to store user
profiles or secrets.

### RuntimeOwner

RuntimeOwner is the component that actually holds a live sing-box runtime.

RuntimeOwner interface semantics are:

```text
Start(snapshot runtime target)
Stop()
Status / capabilities
logs / traffic / connections / groups / URLTest where supported
listener info where supported
```

Supported owner classes:

```text
local embedded owner
  qkboxd process owns embedded sing-box
  used for non-privileged runtime and system proxy mode

provider-hosted owner
  privileged provider or platform runtime container owns embedded sing-box
  used for machine-level TUN / route / DNS mode

Apple NetworkExtension owner
  Network Extension owns the runtime container for macOS VPN/TUN mode
  qkboxd remains user-scope coordinator
```

The GUI does not know which owner class is active.

### singboxadapter

`internal/singboxadapter` is the only package allowed to import sing-box or
sing packages.

It is responsible for:

```text
include.Context setup
option parsing inside the sing-box boundary
box.New / Start / Close
qkbox runtime DTO mapping
log bridge
traffic and connection tracker bridge
outbound group / selector / URLTest bridge
listener info extraction for product-owned features
```

It must not leak sing-box, sing-tun, Clash, or platform DTOs into shared API,
persistence, or GUI code.

### Platform Capability Boundary

The platform capability boundary exposes allowlisted product operations. It does
not expose arbitrary shell, arbitrary filesystem access, or sing-box internals.

It can provide:

```text
provider installation and status
runtime container status
machine-network owner lock
stale owner repair
system service integration
platform diagnostics
process lookup where platform-scoped
```

It must not persist user profiles, subscriptions, secrets, or decrypted runtime
config. Provider-hosted runtime may receive decrypted config in memory for the
duration of a start/reload operation and must never persist or log it.

## Platform Targets

### Windows

```text
qkbox.exe
  Wails GUI

qkboxd.exe
  user-scope product coordinator
  local embedded RuntimeOwner for non-privileged modes
  system proxy coordinator

privileged provider
  provider-hosted RuntimeOwner for machine network mode
  embedded sing-box uses sing-tun for Wintun / route / DNS / WFP mechanics
  owner lock and repair state

IPC
  authenticated local product IPC
  authenticated provider IPC over named pipe with OS ACLs
```

Windows machine network mode is built by running embedded sing-box inside the
provider-hosted owner. qkbox does not implement Wintun, route, DNS hijack, or WFP
logic itself.

### macOS

```text
qkbox.app
  Wails GUI

qkboxd
  user-scope product coordinator
  local embedded RuntimeOwner for non-privileged modes
  system proxy coordinator

Network Extension
  formal runtime container for VPN/TUN mode
  embedded sing-box uses Apple-compatible sing-box/sing-tun paths

IPC
  authenticated Unix socket product IPC
  extension/provider communication shaped by Apple platform requirements
```

macOS TUN/VPN mode uses NetworkExtension as the native product path. Temporary
root route mutation is not a product architecture.

### Linux

```text
qkbox
  Wails GUI

qkboxd
  user-scope product coordinator
  local embedded RuntimeOwner for non-privileged modes
  system proxy coordinator

privileged provider
  provider-hosted RuntimeOwner for machine network mode
  embedded sing-box uses sing-tun for TUN / route / DNS / nftables mechanics
  owner lock and repair state

IPC
  authenticated Unix socket product IPC
  provider IPC with systemd/root helper or polkit-class authorization
```

Linux machine network mode should use sing-box/sing-tun mechanics rather than a
qkbox-specific route or DNS engine.

## Persistence

qkboxd-owned persistence stores:

```text
profiles
drafts
snapshots
encrypted content
active snapshot pointer
remote subscription metadata
asset metadata
system proxy owner record
product diagnostics metadata
```

Provider-owned persistence is minimal and platform-scoped:

```text
owner lock
runtime owner identity
stale state marker
repair metadata
```

Provider storage must not contain user profile content, subscription content,
secrets, or decrypted runtime config.

## Runtime And Reload

Runtime start always targets a snapshot. Draft content is never a runtime source.

Reload semantics:

```text
load target snapshot
validate product/runtime shape
classify required capabilities and runtime owner class
prepare required platform/runtime owner capabilities
start target owner
commit active snapshot only after target start succeeds
on failure, attempt rollback through the same prepare/start pipeline
surface cleanup or rollback degradation explicitly
```

Rollback does not pretend to be atomic platform rollback. It is best-effort,
observable, and repairable.

## System Proxy

System proxy is a product-owned OS user setting, separate from sing-tun.

qkbox uses native snapshot/apply/restore providers for system proxy because it
must preserve and restore the user's pre-existing OS proxy settings. The config
compiler must not silently delegate this ownership to sing-box inbound options.

System proxy uses the active local runtime listener. If the active runtime has no
HTTP or mixed listener, enabling system proxy returns a product error rather than
creating a hidden fallback listener.

## Observability

Runtime observability is capability-aware:

```text
status stream
log stream
traffic snapshots
connection snapshots
outbound groups
selector mutations
URLTest
connection close
```

When a RuntimeOwner cannot provide a capability, qkbox returns unavailable or
unsupported product state. GUI must not fabricate counters, connections, groups,
or logs.

## Data Asset Plane

Data assets are product data, not runtime truth.

Supported asset categories:

```text
remote profile content
subscription metadata
rule-set assets
geo assets
provider/runtime cache metadata
```

sing-geosite, sing-geoip, and srsc are the preferred upstream components to
evaluate for geo and rule-set workflows. Asset updates create product-visible
state and enter runtime only through snapshot/reload coordination.

## Explicitly Rejected Shapes

```text
GUI -> sing-box CLI
GUI -> libbox / FFI runtime
GUI process -> long-lived runtime
GUI -> platform mutation
qkboxd running as machine service to store user data
provider storing profiles or secrets
qkbox reimplementing sing-tun platform networking
qkbox reimplementing DNS runtime
macOS TUN mode via temporary root route hack
unauthenticated localhost control plane
```
