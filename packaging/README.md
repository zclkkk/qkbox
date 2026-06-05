# Packaging

Current decisions:

```text
Windows: NSIS only
Linux: DEB only
macOS: Wails app bundle baseline
```

Runtime data remains per-user and must not be stored under the install
directory. Privileged provider installation remains an explicit platform flow.
