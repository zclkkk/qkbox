# Windows Packaging

The Windows package uses NSIS and must keep support for both:

```text
per-user install
Program Files / machine-wide install
```

Runtime data must remain per-user and must not be stored under the install directory.

Build:

```powershell
npm run build
npm run package:windows
```

The NSIS installer includes `qkbox.exe`, `qkbox-window.exe`, and
`qkbox-provider.exe`. The default install location is per-user, and the
directory page still allows an elevated Program Files install.

The installer creates a Start Menu shortcut and an HKCU uninstall entry. It does
not install provider credentials, does not start the provider, and does not
remove qkbox user data during uninstall.
