# Milestone 6: Packaging And Release Readiness

## Objective

Make the application packageable and diagnosable across Windows, macOS, and
Linux without changing runtime architecture.

## Decisions

- Windows packaging uses NSIS.
- Linux packaging starts with DEB.
- macOS packaging uses an app bundle and DMG path.
- Provider binaries may be included in packages.
- High-privilege provider setup remains explicit product setup, not hidden
  installer behavior.
- Debug bundles remain redacted and product-shaped.

## Work Items

### 1. Windows NSIS

Package:

- `qkbox.exe`
- `qkboxd.exe`
- `qkbox-provider.exe`

Requirements:

- default to per-user install path.
- allow Program Files installation with elevation.
- create Start Menu shortcut.
- register uninstaller.
- do not silently enable privileged provider behavior.
- preserve user data on uninstall unless explicitly requested.

### 2. Linux DEB

Package:

- `/usr/bin/qkbox`
- `/usr/bin/qkboxd`
- `/usr/bin/qkbox-provider`
- desktop entry.
- provider service unit file if supported by the setup flow.

Requirements:

- package build script is reproducible.
- DEB metadata is explicit.
- provider service is not silently activated without the product setup policy
  being satisfied.

### 3. macOS Bundle And DMG

Package:

- Wails app bundle.
- qkboxd helper.
- resources and icons.
- documented NetworkExtension packaging expectations.

Requirements:

- app launches from bundle.
- helper path discovery is deterministic.
- signing/notarization inputs are documented but optional for local builds.

### 4. Diagnostics And Support Bundle

Polish diagnostics:

- product versions.
- schema revision.
- runtime/platform capabilities.
- provider and NetworkExtension status.
- system proxy ownership.
- active profile and snapshot IDs.
- redacted subscription and asset metadata.
- diagnostic checks with recovery text.

Debug bundle must not include:

- profile content.
- encrypted blobs.
- provider tokens.
- proxy owner snapshots.
- URL userinfo, query strings, fragments, or sensitive paths.

### 5. Release Verification Matrix

Document and script platform checks where possible:

- Windows local build and NSIS package.
- Linux DEB package build on Linux.
- macOS bundle build on macOS.
- cross-platform Go compile checks for platform-specific files where practical.

## Out Of Scope

- New product features.
- Runtime architecture changes.
- Silent provider setup.

## Verification

- `go test -tags with_clash_api ./...`
- `go vet -tags with_clash_api ./...`
- `npm run check`
- `npm run build`
- Windows: `npm run package:windows`
- Linux: package script runs on a Linux host.
- macOS: app bundle and DMG scripts run on a macOS host.
- Debug bundle contents are inspected and redaction is verified.

## Acceptance Criteria

- Packages contain the expected binaries and metadata.
- Installation paths are predictable.
- Provider binary presence does not imply hidden provider activation.
- Debug bundles are safe to attach to support reports.
- Packaging scripts fail loudly when required artifacts are missing.

