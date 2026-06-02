# Windows Packaging

Phase 0/1 do not implement installer scripts.

The future Windows package must use NSIS and support both:

```text
per-user install
Program Files / machine-wide install
```

Runtime data must remain per-user and must not be stored under the install directory.
