# Phase 6: Platform Features

> Depends on Phase 5 (engine UI for tray menu wiring).

## 6.1 System Tray

### Problem

Wails 3 has no built-in tray support. Need native system tray on all three platforms.

### Action

**Create** `internal/tray/`:

```
internal/tray/
  tray.go         — Tray interface + MenuItem type
  tray_darwin.go  — macOS: github.com/nickolaevskiernorbert/systray or native NSStatusItem
  tray_windows.go — Windows: github.com/nickolaevskiernorbert/systray or Shell_NotifyIcon
  tray_linux.go   — Linux: github.com/nickolaevskiernorbert/systray (SNI)
```

Interface:
```go
type MenuItem struct {
    Title    string
    Tooltip  string
    Disabled bool
    Checked  bool
    OnClick  func()
    Separator bool
}

type Tray struct { ... }

func New(icon []byte, tooltip string) *Tray
func (t *Tray) SetMenu(items []MenuItem)
func (t *Tray) SetIcon(icon []byte)
func (t *Tray) SetTooltip(text string)
func (t *Tray) OnLeftClick(fn func())
func (t *Tray) Close()
```

Library choice: `getlantern/systray` is the most mature cross-platform option. If it causes issues, fall back to per-platform implementations.

**Integrate in** `apps/desktop/main.go`:
```go
func main() {
    bridge := NewBridgeService()
    t := tray.New(iconBytes, "qkbox")
    defer t.Close()

    updateTrayMenu(t, bridge)

    app := application.New(...)
    // ...
}
```

Menu items:
```
qkbox v0.1.0
─────────────
Show Window          (brings window to front)
─────────────
Engine: STARTED      (read-only status)
Start Engine         (disabled when started)
Stop Engine          (disabled when idle)
─────────────
System Proxy: ON     (toggle)
─────────────
Quit
```

Tray menu updates reactively: subscribe to engine status events, update menu items when state changes.

### Verification

- [ ] Tray icon appears on macOS, Windows, Linux
- [ ] Click "Show Window" brings app to front
- [ ] Start/Stop Engine from tray works
- [ ] System Proxy toggle from tray works
- [ ] Quit from tray exits cleanly (restores proxy)
- [ ] Engine status label updates in real-time

---

## 6.2 Auto-Start

### Action

**Create** `internal/autostart/`:

```
internal/autostart/
  autostart.go        — Manager interface
  autostart_darwin.go — macOS: LaunchAgent plist
  autostart_windows.go — Windows: Registry HKCU\Software\Microsoft\Windows\CurrentVersion\Run
  autostart_linux.go  — Linux: XDG autostart ~/.config/autostart/qkbox.desktop
```

```go
type Manager struct {
    appName      string
    executablePath string
}

func New(appName, executablePath string) *Manager
func (m *Manager) IsEnabled() bool
func (m *Manager) SetEnabled(enabled bool) error
```

Platform implementations:

**macOS** (`autostart_darwin.go`):
- Write/remove `~/Library/LaunchAgents/com.qkbox.autostart.plist`
- Points to the qkboxd executable (not the desktop app, since the desktop app launches qkboxd)

**Windows** (`autostart_windows.go`):
- Set/delete `HKCU\Software\Microsoft\Windows\CurrentVersion\Run\qkbox` registry key
- Value: path to `qkboxd.exe`

**Linux** (`autostart_linux.go`):
- Write/remove `~/.config/autostart/qkbox.desktop`
- Type=Application, Exec=qkboxd, X-GNOME-Autostart-enabled=true

**Expose via IPC:**
- Add `MethodPlatformGetAutoStartStatus` and `MethodPlatformSetAutoStartEnabled`
- Request/Reply types in `shared/api/platform.go`

### Verification

- [ ] macOS: LaunchAgent plist created/removed
- [ ] Windows: Registry key set/removed
- [ ] Linux: .desktop file created/removed
- [ ] Toggle persists across `go test` runs
- [ ] IPC methods return correct current state

---

## 6.3 Auto-Update

### Action

**Create** `internal/updater/`:

```
internal/updater/
  updater.go   — Check + download + install
  verify.go    — SHA256 + optional signature verification
  platform.go  — Platform-specific install logic
```

```go
type UpdateInfo struct {
    Version     string `json:"version"`
    ReleaseURL  string `json:"html_url"`
    DownloadURL string `json:"download_url"`
    SHA256      string `json:"sha256"`
    Body        string `json:"body"` // release notes
}

type Updater struct {
    currentVersion string
    checkURL       string // GitHub Releases API URL
    httpClient     *http.Client
}

func New(currentVersion, checkURL string) *Updater

// CheckForUpdate returns nil if already on latest version.
func (u *Updater) CheckForUpdate(ctx context.Context) (*UpdateInfo, error)

// Download downloads the update to a temp file, verifies SHA256.
func (u *Updater) Download(ctx context.Context, info UpdateInfo, progress func(downloaded, total int64)) (string, error)

// Install replaces the current binary. Platform-specific.
func (u *Updater) Install(downloadPath string) error
```

Update check flow:
1. Fetch `https://api.github.com/repos/zclkkk/qkbox/releases/latest`
2. Compare `tag_name` with `currentVersion` (semver comparison)
3. If newer → return `UpdateInfo` with download URLs for current platform/arch
4. On download: stream to temp file, verify SHA256
5. On install: platform-specific binary replacement + restart

**Expose via IPC:**
- Add `MethodPlatformCheckForUpdate` and `MethodPlatformInstallUpdate`

**Frontend:**
- Add update check to bootstrap (non-blocking)
- Show notification banner when update available
- "Download & Install" button with progress bar

### Verification

- [ ] CheckForUpdate detects newer version from GitHub API
- [ ] Returns nil when already on latest
- [ ] Download streams with progress reporting
- [ ] SHA256 verification passes for correct file, fails for tampered
