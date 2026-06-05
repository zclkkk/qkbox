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
```

Release packages install product binaries only. Product updates replace the
whole product build; qkbox does not hot-swap runtime binaries or data assets as
partial application updates.
