# Phase 7: Provider & TUN Mode

> Depends on Phase 2 (sing-box integration deepening).

## 7.1 Provider Runtime Verification

### Status

`internal/providerruntime/controller.go` (490 lines) already implements the full runtime lifecycle:
- `RuntimeStart` — validates session/ownership, creates `singboxadapter.Adapter`, starts sing-box
- `RuntimeStop` — stops adapter, cleans up owner record
- `RuntimeHeartbeat` — keeps owner alive, auto-stops on timeout (8s)
- `RuntimeGetStatus/Traffic/Connections/Groups/SelectOutbound/URLTest/CloseConnection/CloseAllConnections/ListenerInfo/SubscribeEvents` — delegate to adapter after ownership validation
- Stale owner detection and repair
- Heartbeat monitor with automatic shutdown

`cmd/qkbox-provider/main.go` delegates all IPC methods to `providerruntime.Controller`.

**This is already implemented.** The remaining work is:
1. End-to-end testing on each platform
2. Ensuring TUN config works with sing-box's TUN implementation
3. Service installation and lifecycle management

### Action

- Write integration tests that start qkbox-provider, send RuntimeStart with a TUN config, verify sing-box creates the tun interface, then RuntimeStop cleans up
- Test heartbeat timeout: start runtime, stop sending heartbeats, verify auto-shutdown after 8s
- Test stale owner recovery: kill provider mid-runtime, restart, verify stale detection

---

## 7.2 TUN Mode via Provider

### Linux

The provider must run as root to create `/dev/net/tun`.

**Service installation:**
```
/etc/systemd/system/qkbox-provider.service
```
```ini
[Unit]
Description=qkbox privileged provider
After=network.target

[Service]
ExecStart=/usr/bin/qkbox-provider --serve --state-dir /var/lib/qkbox
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

**Capability detection** (`platform/capability/tun_linux.go`):
```go
func TUNAvailable() (bool, string) {
    if os.Geteuid() != 0 {
        return false, "TUN requires root privileges. Install qkbox-provider as a system service."
    }
    if _, err := os.Stat("/dev/net/tun"); err != nil {
        return false, "/dev/net/tun not available. Load the tun kernel module."
    }
    return true, ""
}
```

### Windows

The provider must run as elevated (Administrator or SYSTEM).

**Service installation:**
- Register `qkbox-provider` as a Windows Service via `sc create`
- Or use a scheduled task running as SYSTEM with interactive session

**Capability detection** (`platform/capability/tun_windows.go`):
```go
func TUNAvailable() (bool, string) {
    if !isElevated() {
        return false, "TUN requires administrator privileges. Install qkbox-provider as a service."
    }
    // Check if WinTun driver is installed
    if !isWinTunInstalled() {
        return false, "WinTun driver not installed."
    }
    return true, ""
}
```

### macOS

TUN mode on macOS goes through NetworkExtension (Phase 7.3), not the provider. The provider on macOS is only used for BACKGROUND_SERVICE capability.

---

## 7.3 NetworkExtension Integration (macOS)

### Problem

`platform/capability/network_extension_darwin.go` currently returns `unavailableNetworkExtensionRuntime` — a stub that always returns "not available".

macOS TUN mode requires a Network Extension (NE), which is a Swift/AppKit process that runs in a sandboxed environment managed by the OS.

### Architecture

```
┌──────────┐    IPC (local socket)    ┌─────────────────────┐
│ qkboxd   │ ◄──────────────────────► │ Network Extension   │
│ (Go)     │                           │ (Swift, sandboxed)  │
└──────────┘                           │   - NEPacketTunnel   │
                                       │   - sing-box (Go lib)│
                                       └─────────────────────┘
```

The NE is a separate build target (Xcode project, Swift). The Go side communicates via a local Unix socket.

### Action

This phase is partially outside Go codebase. The Go side needs:

**Implement** `platform/capability/network_extension_darwin.go`:

```go
type darwinNetworkExtensionRuntime struct {
    socketPath string
}

func NewNetworkExtensionRuntime(stateDir string) NetworkExtensionRuntime {
    return &darwinNetworkExtensionRuntime{
        socketPath: filepath.Join(stateDir, "ne-runtime.sock"),
    }
}

func (r *darwinNetworkExtensionRuntime) Status(ctx context.Context) api.NetworkExtensionStatus {
    // Try to connect to the NE socket
    // If unreachable → not installed or not running
    // If reachable → send status request, return capabilities
}

func (r *darwinNetworkExtensionRuntime) Start(ctx context.Context, req ...) error {
    // Send start request over IPC to NE
    // NE receives config JSON, starts sing-box with NEPacketTunnelProvider
}
// ... similar for Stop, Heartbeat, runtime queries
```

The NE Swift side is a separate project (similar to how sing-box-for-apple works). It:
1. Implements `NEPacketTunnelProvider`
2. Embeds sing-box as a Go library (via gomobile/cgo)
3. Opens a local socket for IPC with qkboxd
4. Receives config JSON, starts sing-box TUN runtime inside the NE

### Scope

- **Go side (this repo):** Implement the IPC client in `network_extension_darwin.go`
- **Swift side (separate repo):** NE app extension implementation
- This is the most complex platform feature. Consider deferring the Swift side and focusing on Linux/Windows TUN via provider first.

### Verification

- [ ] Linux: provider starts sing-box with TUN inbound, tun interface created, traffic routes
- [ ] Windows: provider starts sing-box with TUN inbound, WinTun driver engaged
- [ ] macOS: NE status reported correctly (installed/reachable/authorized)
- [ ] TUN capability shows "available" when provider/NE is running
- [ ] TUN capability shows "unavailable" with clear reason when not
