# macOS Packaging

The macOS baseline is the Wails app bundle plus the formal Apple
NetworkExtension runtime container path for VPN/TUN mode.

Build on macOS:

```sh
npm run build
npm run package:macos
```

The local packaging script creates `qkbox.app` with `qkbox` in
`Contents/MacOS` and private helpers (`qkbox-window`, `qkbox-provider`)
in `Contents/Helpers`, then produces a DMG in `dist/packages`.
The daemon runs as a background process with `LSUIElement=true` (no Dock icon).

Release packaging must follow Apple signing, notarization, and entitlement
requirements. qkbox does not ship macOS TUN mode through a root route mutation
installer or ad-hoc privileged helper.

Set `QKBOX_CODESIGN_IDENTITY` to sign local bundles. Production builds still
need the NetworkExtension target, entitlements, provisioning profile, and
notarization pipeline.
