# Platform Boundary

This directory contains platform capability providers and platform-facing
boundary code.

Platform code may own native OS integration such as:

```text
system proxy mutation
privileged provider client wiring
privileged helper installation
platform diagnostics
```

Machine-level TUN, route, and DNS behavior must be implemented through the
formal runtime owner/provider path and official sing-box/sing-tun mechanics, not
through ad hoc qkbox route or DNS code.
