# Packaging

Current decisions:

```text
Windows: NSIS only
Linux: DEB only
macOS: Wails app bundle baseline
```

Runtime data remains per-user and must not be stored under the install
directory. Privileged provider installation remains an explicit platform flow.

Build entry points:

```text
npm run package:windows
npm run package:linux
npm run package:macos
npm run release:verify
```

Release packages install product binaries only. Product updates replace the
whole product build; qkbox does not hot-swap runtime binaries or data assets as
partial application updates.

Provider binaries may be included in packages, but package installation does not
enable privileged runtime ownership, install provider credentials, start systemd
services, or authorize Apple NetworkExtension. Those remain explicit product
setup actions.

Host-specific packaging commands intentionally fail on the wrong OS. `release:verify`
runs the full local check/build suite and then packages only for the current host.
