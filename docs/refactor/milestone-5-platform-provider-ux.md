# Milestone 5: Platform Provider UX

## Objective

Make platform capabilities understandable and operable from the GUI, especially
the privileged provider and machine-network/TUN mode setup path.

## Decisions

- Provider setup is explicit and visible near the mode that needs it.
- Installers may include provider binaries, but GUI owns the user-facing setup
  and repair flow.
- Windows and Linux use provider-hosted machine-network mode.
- macOS uses the NetworkExtension runtime container path for machine-network
  mode.
- System proxy remains separate from provider-hosted runtime ownership.

## Work Items

### 1. Platform Capability View

Implement a capability view that shows:

- system proxy.
- background service/provider.
- TUN mode.
- DNS hijack.
- connection tracking.
- process lookup if available.

Each capability must show state, reason, and available repair/setup actions.

### 2. Provider Status Panel

Implement provider status UX:

- installed.
- reachable.
- authenticated.
- version and expected version.
- endpoint.
- owner state.
- stale owner state.
- repair actions.

Repair actions must be allowlisted and visible only when valid.

### 3. Provider Setup Flow

Implement explicit setup entry points near machine-network/TUN controls:

- install provider.
- repair provider.
- remove provider if supported by platform package flow.
- explain required privileges before taking action.

The backend may add platform APIs for these actions only when they correspond to
real provider setup behavior. Do not add fake setup APIs for appearance.

### 4. Machine-Network Mode UX

Expose mode selection:

- system proxy mode.
- machine-network/TUN mode when available.

Starting machine-network mode must use the existing RuntimeOwner selection path.
Do not bypass snapshot validation or capability preparation.

### 5. macOS NetworkExtension UX

Show NetworkExtension status:

- installed.
- reachable.
- authorized.
- bundle/version.
- owner state.

The Go side may expose the status and runtime client path. The extension
container implementation is platform-specific and must not be replaced with root
route hacks.

### 6. System Proxy UX

Improve existing system proxy panel:

- binds toggle to `qkbox_owned`, not raw OS proxy enabled state.
- explains when user-owned proxy is present.
- shows listener missing error clearly.
- restores qkbox-owned proxy before engine stop.

## Out Of Scope

- Silent high-privilege provider service installation.
- macOS root route based machine-network mode.
- Replacing system proxy snapshot/restore with sing-box config delegation.
- New runtime core.

## Verification

- `npm run check`
- `npm run build`
- `go test -tags with_clash_api ./...`
- Provider status and repair actions render correctly.
- TUN/machine-network start path goes through RuntimeOwner selection.
- System proxy still snapshots/restores user state correctly.
- macOS status path reports NetworkExtension unavailability clearly when the
  container is absent.

## Acceptance Criteria

- Users can understand why a platform feature is unavailable.
- Provider install/repair is explicit and not hidden inside unrelated installer
  behavior.
- Machine-network mode does not bypass qkboxd capability preparation or
  RuntimeOwner state.
- Cross-platform differences are visible but product concepts remain shared.

