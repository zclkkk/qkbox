# Phase 0/1 Baseline

This document records the initial repository shape implemented for Phase 0 and Phase 1.

## Implemented Scope

```text
Phase 0
  root Go module
  root npm commands
  Wails desktop placeholder
  qkboxd command placeholder
  shared API/model packages
  platform placeholders
  packaging placeholders
  local check/test/build entry points

Phase 1
  user-scope qkboxd startup
  per-user qkboxd lock
  Windows Named Pipe transport
  Unix Domain Socket transport for macOS/Linux
  length-prefixed JSON frames
  Hello / HelloReply contract
  API version compatibility check
  structured IPC errors
  runtime/platform capability shells
  Wails bridge method for GUI bootstrap
```

## Deliberately Not Implemented

```text
sing-box runtime start
profile persistence
encrypted storage
privileged provider
system proxy
TUN / route / DNS mutation
runtime logs
traffic / connections / groups
remote asset updates
installer integration for qkboxd
```
