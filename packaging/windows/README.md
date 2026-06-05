# Windows Packaging

The Windows package uses NSIS and must keep support for both:

```text
per-user install
Program Files / machine-wide install
```

Runtime data must remain per-user and must not be stored under the install directory.
