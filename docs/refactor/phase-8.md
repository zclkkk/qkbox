# Phase 8: Packaging & Distribution

> Depends on all previous phases.

## 8.1 Windows NSIS Installer

### Files

- `packaging/windows/build-nsis.ps1` — Build orchestrator
- `packaging/windows/installer.nsi` — NSIS script (currently exists, needs completion)

### Installer Requirements

**Binaries included:**
- `qkbox.exe` — Wails desktop app
- `qkboxd.exe` — User-scope daemon
- `qkbox-provider.exe` — Privileged provider

**Install steps:**
1. Copy binaries to `C:\Program Files\qkbox\`
2. Register `qkbox-provider` as a Windows Service:
   ```
   sc create qkbox-provider binPath= "C:\Program Files\qkbox\qkbox-provider.exe --serve" start= auto
   sc description qkbox-provider "qkbox privileged provider for TUN mode and background service"
   ```
3. Create Start Menu shortcut for `qkbox.exe`
4. Register uninstaller in Add/Remove Programs
5. Optional: bundle WinTun driver (`wintun.dll`) and install it

**Uninstall steps:**
1. Stop and delete the qkbox-provider service
2. Remove binaries
3. Remove Start Menu shortcut
4. Remove registry entries
5. Optionally remove user data (`%APPDATA%\qkbox\`)

**`build-nsis.ps1`**:
```powershell
param(
    [string]$BinDir = "bin",
    [string]$OutputDir = "dist"
)

$ErrorActionPreference = "Stop"

# Verify binaries exist
$required = @("qkbox.exe", "qkboxd.exe", "qkbox-provider.exe")
foreach ($bin in $required) {
    if (!(Test-Path "$BinDir/$bin")) {
        Write-Error "Missing $BinDir/$bin. Run 'npm run build' first."
        exit 1
    }
}

# Run NSIS
& makensis /NOCD /V3 `
    /DQKBOX_VERSION=$env:QKBOX_VERSION `
    /DQKBOX_BIN_DIR=$BinDir `
    /DQKBOX_OUTPUT_DIR=$OutputDir `
    packaging/windows/installer.nsi
```

---

## 8.2 macOS DMG

### Files

```
packaging/macos/
  build-dmg.sh
  Info.plist
  entitlements.plist
  postinstall.sh
```

### DMG Contents

- `qkbox.app/` — Wails-generated .app bundle
  - `Contents/MacOS/qkbox` — main executable
  - `Contents/MacOS/qkboxd` — bundled as helper executable
  - `Contents/Resources/` — icons, assets
  - `Contents/Info.plist` — app metadata

### Info.plist Key Entries

```xml
<key>CFBundleIdentifier</key>
<string>com.qkbox.app</string>
<key>CFBundleVersion</key>
<string>0.1.0</string>
<key>LSMinimumSystemVersion</key>
<string>12.0</string>
<key>NSAppTransportSecurity</key>
<dict>
    <key>NSAllowsLocalNetworking</key>
    <true/>
</dict>
```

### entitlements.plist

```xml
<key>com.apple.security.network.client</key>
<true/>
<key>com.apple.security.network.server</key>
<true/>
```

### postinstall.sh

- Copy `qkboxd` to a well-known path (e.g., `/usr/local/bin/qkboxd` or inside the app bundle)
- Optionally install a LaunchAgent for qkboxd

### build-dmg.sh

```bash
#!/bin/bash
set -euo pipefail

VERSION="${QKBOX_VERSION:-0.1.0}"
APP_NAME="qkbox"
DMG_NAME="${APP_NAME}-${VERSION}.dmg"

# Create DMG
create-dmg \
  --volname "${APP_NAME}" \
  --volicon "packaging/macos/icon.icns" \
  --window-pos 200 120 \
  --window-size 600 400 \
  --icon-size 100 \
  --icon "${APP_NAME}.app" 175 190 \
  --hide-extension "${APP_NAME}.app" \
  --app-drop-link 425 190 \
  "dist/${DMG_NAME}" \
  "dist/${APP_NAME}.app"

# Sign (if certificate available)
if [ -n "${CODESIGN_IDENTITY:-}" ]; then
    codesign --deep --force --sign "$CODESIGN_IDENTITY" "dist/${APP_NAME}.app"
fi

# Notarize (if credentials available)
if [ -n "${APPLE_ID:-}" ]; then
    xcrun notarytool submit "dist/${DMG_NAME}" \
        --apple-id "$APPLE_ID" \
        --password "$APPLE_PASSWORD" \
        --team-id "$APPLE_TEAM_ID" \
        --wait
    xcrun stapler staple "dist/${DMG_NAME}"
fi
```

---

## 8.3 Linux DEB

### Files

```
packaging/linux/
  build-deb.sh
  qkbox.desktop         — Already exists
  qkbox-provider.service — systemd system unit
  debian/
    control
    postinst
    prerm
```

### Package Contents

```
/usr/bin/qkbox                    — Wails desktop app
/usr/bin/qkboxd                   — User-scope daemon
/usr/bin/qkbox-provider           — Privileged provider
/usr/share/applications/qkbox.desktop
/etc/systemd/system/qkbox-provider.service
```

### debian/control

```
Package: qkbox
Version: 0.1.0
Architecture: amd64
Maintainer: qkbox <noreply@qkbox.dev>
Depends: libc6
Description: qkbox desktop proxy client
 A cross-platform proxy client built on sing-box.
```

### qkbox-provider.service

```ini
[Unit]
Description=qkbox privileged provider
After=network.target

[Service]
Type=simple
ExecStart=/usr/bin/qkbox-provider --serve
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### postinst

```bash
#!/bin/bash
set -e
systemctl daemon-reload
systemctl enable qkbox-provider
systemctl start qkbox-provider
```

### prerm

```bash
#!/bin/bash
set -e
systemctl stop qkbox-provider || true
systemctl disable qkbox-provider || true
```

### build-deb.sh

```bash
#!/bin/bash
set -euo pipefail

VERSION="${QKBOX_VERSION:-0.1.0}"
ARCH="amd64"
PKG_DIR="dist/qkbox_${VERSION}_${ARCH}"

# Create package structure
mkdir -p "$PKG_DIR"/{usr/bin,usr/share/applications,etc/systemd/system,DEBIAN}

# Copy binaries
cp bin/qkbox "$PKG_DIR/usr/bin/"
cp bin/qkboxd "$PKG_DIR/usr/bin/"
cp bin/qkbox-provider "$PKG_DIR/usr/bin/"
chmod 755 "$PKG_DIR/usr/bin/"*

# Copy desktop file
cp packaging/linux/qkbox.desktop "$PKG_DIR/usr/share/applications/"

# Copy systemd unit
cp packaging/linux/qkbox-provider.service "$PKG_DIR/etc/systemd/system/"

# Write control file
cat > "$PKG_DIR/DEBIAN/control" << EOF
Package: qkbox
Version: $VERSION
Architecture: $ARCH
Maintainer: qkbox <noreply@qkbox.dev>
Depends: libc6
Description: qkbox desktop proxy client
EOF

# Copy maintainer scripts
cp packaging/linux/debian/postinst "$PKG_DIR/DEBIAN/postinst"
cp packaging/linux/debian/prerm "$PKG_DIR/DEBIAN/prerm"
chmod 755 "$PKG_DIR/DEBIAN/"*

# Build
dpkg-deb --build "$PKG_DIR" "dist/qkbox_${VERSION}_${ARCH}.deb"
```

---

## 8.4 CI/CD Pipeline

### File

`.github/workflows/build.yml`

### Matrix

| OS | Go arch | Artifacts |
|---|---|---|
| ubuntu-latest | amd64 | .deb |
| windows-latest | amd64 | .exe (NSIS installer) |
| macos-latest | arm64 + amd64 | .dmg (universal or separate) |

### Pipeline

```yaml
name: Build and Package

on:
  push:
    tags: ['v*']
  pull_request:

jobs:
  build:
    strategy:
      matrix:
        os: [ubuntu-latest, windows-latest, macos-latest]
    runs-on: ${{ matrix.os }}

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.26'

      - uses: actions/setup-node@v4
        with:
          node-version: '22'

      - name: Install frontend deps
        run: npm run frontend:install

      - name: Generate bindings
        run: npm run frontend:check

      - name: Build qkboxd
        run: npm run build:qkboxd

      - name: Build qkbox-provider
        run: npm run build:qkbox-provider

      - name: Build desktop app
        run: npm run build:desktop

      - name: Run tests
        run: npm run go:test

      - name: Package (Windows)
        if: matrix.os == 'windows-latest'
        run: npm run package:windows

      - name: Package (Linux)
        if: matrix.os == 'ubuntu-latest'
        run: npm run package:linux

      - name: Package (macOS)
        if: matrix.os == 'macos-latest'
        run: bash packaging/macos/build-dmg.sh

      - uses: actions/upload-artifact@v4
        with:
          name: ${{ matrix.os }}-packages
          path: dist/

  release:
    needs: build
    if: startsWith(github.ref, 'refs/tags/')
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: actions/download-artifact@v4
      - uses: softprops/action-gh-release@v2
        with:
          files: |
            ubuntu-latest-packages/*
            windows-latest-packages/*
            macos-latest-packages/*
```

### Verification

- [ ] Windows: NSIS installer installs all 3 binaries, registers service, creates shortcuts, uninstaller works
- [ ] macOS: DMG opens, app launches, qkboxd helper accessible
- [ ] Linux: DEB installs correctly, systemd services registered, `qkbox` command works
- [ ] CI produces artifacts on PR and release assets on tag push
- [ ] Each platform's installer is < 50 MB
