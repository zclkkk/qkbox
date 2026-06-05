# macOS Packaging

The macOS baseline is the Wails app bundle plus the formal Apple
NetworkExtension runtime container path for VPN/TUN mode.

Build on macOS:

```sh
npm run build
npm run package:macos
```

The local packaging script creates `qkbox.app` with `qkbox`, `qkboxd`, and
`qkbox-provider` in `Contents/MacOS`, then produces a DMG in `dist/packages`.
Helper discovery is deterministic because helpers sit next to the app
executable.

Release packaging must follow Apple signing, notarization, and entitlement
requirements. qkbox does not ship macOS TUN mode through a root route mutation
installer or ad-hoc privileged helper.

Set `QKBOX_CODESIGN_IDENTITY` to sign local bundles. Production builds still
need the NetworkExtension target, entitlements, provisioning profile, and
notarization pipeline.
