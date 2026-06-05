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

The NSIS installer includes `qkbox.exe`, `qkboxd.exe`, and
`qkbox-provider.exe`. The default install location is per-user, and the
directory page still allows an elevated Program Files install.
