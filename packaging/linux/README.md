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

The package also installs `qkbox-provider.service` as a disabled systemd unit.
It is not enabled or started by maintainer scripts. The provider only becomes
active after an explicit product setup flow creates the server config and asks
the OS to start the service.
