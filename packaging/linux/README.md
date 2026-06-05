# Linux Packaging

The first Linux package target is DEB for modern Debian/Ubuntu family systems.

Build on Linux:

```sh
npm run build
npm run package:linux
```

The DEB package installs `qkbox`, `qkboxd`, and `qkbox-provider` under
`/usr/bin` plus a desktop entry. User data remains in the qkbox state directory,
not under `/usr`.
