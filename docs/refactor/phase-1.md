# Phase 1: Backend Architecture Refactoring

> Foundation phase. All subsequent phases depend on this.

## 1.1 Unify IPC Framework

### Problem

`internal/ipc/` (6 files) and `internal/provideripc/` (7 files) share ~90% identical code:
- `frame.go` — identical (length-prefixed JSON, 4-byte uint32 header, max 1 MiB)
- `transport_unix.go` / `transport_windows.go` — same pattern, different endpoints
- `server.go` — same `Serve` loop, `handleConn`, `dispatch[Req,Reply]`, `serveSubscription[Req]` generics
- `client.go` — same `do[Req,Reply]`, `openSubscription` patterns

Only real differences:
1. Transport endpoint (unix socket path vs named pipe path)
2. `provideripc` client authenticates before each request (token-based)
3. `provideripc` server has mandatory auth handshake + write deadlines
4. `provideripc` has config file discovery (`ClientConfig`/`ServerConfig`)

### Action

Create `internal/ipcframework/` — a single parameterized IPC framework.

#### `internal/ipcframework/frame.go`

Move from `internal/ipc/frame.go` unchanged. Same wire format: 4-byte big-endian uint32 length prefix + JSON payload.

Types:
```go
type Request struct {
    ID     string          `json:"id"`
    Method string          `json:"method"`
    Params json.RawMessage `json:"params,omitempty"`
}

type Response struct {
    ID     string               `json:"id"`
    Result json.RawMessage      `json:"result,omitempty"`
    Error  *api.StructuredError `json:"error,omitempty"`
}

type EventFrame struct {
    ID    string               `json:"id,omitempty"`
    Event string               `json:"event"`
    Data  json.RawMessage      `json:"data,omitempty"`
    Error *api.StructuredError `json:"error,omitempty"`
}

type SubscriptionAck struct{}

func ReadFrame(r io.Reader, v any) error
func WriteFrame(w io.Writer, v any) error
```

#### `internal/ipcframework/transport.go`

```go
type Transport interface {
    Listen(ctx context.Context) (net.Listener, error)
    Dial(ctx context.Context) (net.Conn, error)
    Endpoint() string
}
```

#### `internal/ipcframework/transport_unix.go`

Merge logic from both `internal/ipc/transport_unix.go` and `internal/provideripc/transport_unix.go`.

```go
type UnixTransport struct {
    SocketPath string
}

func (t *UnixTransport) Listen(ctx context.Context) (net.Listener, error)
func (t *UnixTransport) Dial(ctx context.Context) (net.Conn, error)
func (t *UnixTransport) Endpoint() string

// Factories for the two endpoints:
func QKBoxDTransport() *UnixTransport      // $XDG_RUNTIME_DIR/qkbox/qkboxd.sock
func ProviderTransport(endpoint string) *UnixTransport
```

#### `internal/ipcframework/transport_windows.go`

Merge logic from both Windows transport files.

```go
type NamedPipeTransport struct {
    PipeName string
    // Security descriptor for provider pipe
    SecurityDescriptor string // empty = default
}

func (t *NamedPipeTransport) Listen(ctx context.Context) (net.Listener, error)
func (t *NamedPipeTransport) Dial(ctx context.Context) (net.Conn, error)
func (t *NamedPipeTransport) Endpoint() string

func QKBoxDTransport() *NamedPipeTransport       // \\.\pipe\qkbox-<uid>-qkboxd
func ProviderTransport(endpoint string) *NamedPipeTransport
```

#### `internal/ipcframework/conn.go` — NEW: Persistent Multiplexed Connection

The key architectural improvement. A single connection handles concurrent requests by routing responses by request ID.

```go
type MultiplexedConnection struct {
    conn    net.Conn
    writeMu sync.Mutex            // serializes writes
    pending map[string]chan frame  // keyed by request ID
    mu      sync.Mutex            // protects pending + closed
    closed  bool
    done    chan struct{}
}

// Call sends a request and waits for the response on the same connection.
func (c *MultiplexedConnection) Call(ctx context.Context, req Request) (Response, error)

// OpenStream sends a request and returns a channel of streaming events.
// The initial Response (SubscriptionAck) is consumed internally.
func (c *MultiplexedConnection) OpenStream(ctx context.Context, req Request) (<-chan EventFrame, error)

func (c *MultiplexedConnection) Close() error

// Internal: reader goroutine dispatches Response/EventFrame to pending channels
func (c *MultiplexedConnection) readLoop()
```

Implementation notes:
- `readLoop` runs in a goroutine, reads frames, inspects ID field, dispatches to `pending[id]`
- For `EventFrame`: dispatch to the stream channel associated with that ID
- For `Response`: dispatch to the call channel, then remove from pending
- On connection close: close all pending channels, set `closed = true`
- `Call` creates a channel, registers in `pending`, writes request, waits for response or ctx cancel
- `OpenStream` creates a buffered channel (cap 64), registers in `pending`, writes request, reads initial Response, returns channel

#### `internal/ipcframework/client.go`

```go
type ClientConfig struct {
    Transport   Transport
    AuthToken   string        // empty = no auth
    CallTimeout time.Duration // default 5s
}

type Client struct {
    cfg  ClientConfig
    conn *MultiplexedConnection
    mu   sync.Mutex
}

func NewClient(cfg ClientConfig) *Client

// EnsureConnected establishes the persistent connection if not already open.
// Called automatically by Call/Subscribe. Reconnects on connection loss.
func (c *Client) EnsureConnected(ctx context.Context) error

// Call sends a request and returns the typed reply.
func Call[Req, Reply any](c *Client, ctx context.Context, method string, req Req) (Reply, *api.StructuredError)

// Subscribe opens a streaming subscription.
func Subscribe[Req any](c *Client, ctx context.Context, method string, req Req) (<-chan EventFrame, *api.StructuredError)

// Close tears down the persistent connection.
func (c *Client) Close() error
```

Implementation:
- `EnsureConnected`: if `conn` is nil or closed, dial new connection, authenticate if `AuthToken` set, start `readLoop`
- `Call`: ensure connected, marshal request, set deadline via ctx, `conn.Call`, unmarshal reply
- `Subscribe`: ensure connected, marshal request, `conn.OpenStream`, return event channel
- Auto-reconnect: if `Call` gets a connection error, try `EnsureConnected` once, retry

#### `internal/ipcframework/server.go`

Replace the 200-line switch dispatch with a registry.

```go
type MethodHandler func(ctx context.Context, params json.RawMessage) (json.RawMessage, *api.StructuredError)
type SubscriptionHandler func(ctx context.Context, params json.RawMessage) (<-chan api.RuntimeEvent, *api.StructuredError)

type ServerConfig struct {
    Transport     Transport
    AuthFunc      func(token string) bool // nil = no auth
    Methods       map[string]MethodHandler
    Subscriptions map[string]SubscriptionHandler
    IOTimeout     time.Duration // default 5s
}

type Server struct { ... }

func NewServer(cfg ServerConfig) *Server
func (s *Server) Serve(ctx context.Context) error

// Generic registration helpers:
func RegisterMethod[Req, Reply any](
    methods map[string]MethodHandler,
    method string,
    fn func(ctx context.Context, req Req) (Reply, *api.StructuredError),
)

func RegisterSubscription[Req any](
    subscriptions map[string]SubscriptionHandler,
    method string,
    fn func(ctx context.Context, req Req) (<-chan api.RuntimeEvent, *api.StructuredError),
)
```

Server implementation:
- `Serve`: accept loop, each connection in a goroutine
- Per-connection: if auth required, read first frame expecting auth request, validate token
- Then read request loop (not single-frame): read `Request`, look up in `Methods` or `Subscriptions` map, dispatch
- This is a critical change: **server now handles multiple requests per connection** (matching the client's persistent connection)
- `handleConn` loop: `for { readFrame → dispatch → writeResponse }`
- Subscription: write `SubscriptionAck`, then stream `EventFrame` messages. Client cancellation: client sends a cancel frame (empty request with same ID), server stops subscription.

#### `internal/ipcframework/auth.go`

```go
func TokenAuthMiddleware(validToken string) func(token string) bool {
    return func(token string) bool {
        return subtle.ConstantTimeCompare([]byte(token), []byte(validToken)) == 1
    }
}
```

#### Migration

**Delete entirely:** `internal/ipc/` (6 files), `internal/provideripc/` (7 files)

**Update consumers:**

| Consumer | Current | After |
|---|---|---|
| `cmd/qkboxd/main.go` | `ipc.Listen()` + `ipc.NewServer(service).Serve()` | `ipcframework.NewServer(cfg).Serve()` |
| `core/qkboxd/daemon.go` | `ipc.Listen()` + `ipc.NewServer(service)` | same pattern, new package |
| `apps/desktop/bridge.go` | `ipc.NewClient()` + per-call dial | `ipcframework.NewClient(cfg)` + persistent conn |
| `cmd/qkbox-provider/main.go` | `provideripc.NewServer(...)` | `ipcframework.NewServer(cfg)` with `AuthFunc` |

### Verification

- [ ] `go test ./...` passes
- [ ] `internal/ipc/` and `internal/provideripc/` are deleted
- [ ] Desktop → qkboxd IPC uses persistent connection (verify: only 1 `Dial` per session)
- [ ] qkboxd → provider IPC uses persistent connection with auth
- [ ] Subscriptions work over the persistent connection
- [ ] Connection loss triggers auto-reconnect on next call

---

## 1.2 Split service.go Into Domain Handlers

### Problem

`core/qkboxd/service.go`: 1787 lines, 10 fields, 40+ methods, single `opMu` mutex.

### Action

Split into 6 handler files + 1 wire file. `service.go` becomes the thin orchestrator.

#### `core/qkboxd/service.go` — Trimmed

```go
type Service struct {
    db         *persistence.DB
    key        []byte
    engine     *EngineController
    events     *eventhub.Hub
    proxy      capability.SystemProxyProvider
    privileged capability.PrivilegedProvider
    extension  capability.NetworkExtensionRuntime
    assetStore *assetcache.Store
    httpClient *http.Client
}

func NewService(ctx, db, key, proxy, privileged, extension) *Service
func (s *Service) Close() error
func (s *Service) Hello(ctx, req) (HelloReply, *StructuredError)
func (s *Service) RegisterHandlers(methods, subscriptions)
```

#### `core/qkboxd/handler_profile.go`

```go
type ProfileHandler struct {
    db  *persistence.DB
    key []byte
}

func (h *ProfileHandler) CreateProfile(ctx, req) (reply, *StructuredError)
func (h *ProfileHandler) UpdateProfileDraft(ctx, req) (reply, *StructuredError)
func (h *ProfileHandler) DeleteProfile(ctx, req) (reply, *StructuredError)
func (h *ProfileHandler) ListProfiles(ctx, req) (reply, *StructuredError)
func (h *ProfileHandler) GetProfile(ctx, req) (reply, *StructuredError)
```

Move from service.go: `CreateProfile`, `UpdateProfileDraft`, `DeleteProfile`, `ListProfiles`, `GetProfile` + helpers `encryptedContent`, `decryptContent`.

#### `core/qkboxd/handler_asset.go`

```go
type AssetHandler struct {
    db         *persistence.DB
    key        []byte
    httpClient *http.Client
    assetStore *assetcache.Store
}

func (h *AssetHandler) CreateProfileSubscription(ctx, req) (reply, *StructuredError)
func (h *AssetHandler) ListProfileSubscriptions(ctx, req) (reply, *StructuredError)
func (h *AssetHandler) RefreshProfileSubscription(ctx, req) (reply, *StructuredError)
func (h *AssetHandler) DeleteProfileSubscription(ctx, req) (reply, *StructuredError)
func (h *AssetHandler) CreateDataAsset(ctx, req) (reply, *StructuredError)
func (h *AssetHandler) ListDataAssets(ctx, req) (reply, *StructuredError)
func (h *AssetHandler) RefreshDataAsset(ctx, req) (reply, *StructuredError)
func (h *AssetHandler) DeleteDataAsset(ctx, req) (reply, *StructuredError)
```

Move from service.go: all subscription/data-asset methods + helpers `fetchRemote`, `validateRemoteURL`, `normalizeSubscriptionUpdatePolicy`, `normalizeDataAssetKind`, `validateDataAssetContent`, `recordProfileSubscriptionFailure`, `recordDataAssetFailure`, `sha256Hex`.

#### `core/qkboxd/handler_snapshot.go`

```go
type SnapshotHandler struct {
    db     *persistence.DB
    key    []byte
    engine *EngineController
}

func (h *SnapshotHandler) ValidateProfileDraft(ctx, req) (reply, *StructuredError)
func (h *SnapshotHandler) GetProfileDiagnostics(ctx, req) (reply, *StructuredError)
func (h *SnapshotHandler) CreateProfileSnapshot(ctx, req) (reply, *StructuredError)
func (h *SnapshotHandler) ActivateProfileSnapshot(ctx, req) (reply, *StructuredError)
func (h *SnapshotHandler) GetActiveProfile(ctx, req) (reply, *StructuredError)
func (h *SnapshotHandler) GetActiveSnapshot(ctx, req) (reply, *StructuredError)
func (h *SnapshotHandler) ListSnapshots(ctx, req) (reply, *StructuredError)
func (h *SnapshotHandler) RollbackToSnapshot(ctx, req) (reply, *StructuredError)
```

Move from service.go: all snapshot/validation methods + helpers `validateContent`, `validateArrayField`, `extractRequiredCapabilities`, `arrayObjects`. Absorb content of `core/qkboxd/validate.go`.

#### `core/qkboxd/handler_engine.go`

```go
type EngineHandler struct {
    mu         sync.Mutex
    engine     *EngineController
    db         *persistence.DB
    key        []byte
    proxy      capability.SystemProxyProvider
    privileged capability.PrivilegedProvider
    extension  capability.NetworkExtensionRuntime
    events     *eventhub.Hub
}

func (h *EngineHandler) Start(ctx, req) (reply, *StructuredError)
func (h *EngineHandler) Stop(ctx, req) (reply, *StructuredError)
func (h *EngineHandler) Reload(ctx, req) (reply, *StructuredError)
func (h *EngineHandler) GetStatus(ctx, req) (reply, *StructuredError)
func (h *EngineHandler) SubscribeStatus(ctx, req) (<-chan RuntimeEvent, *StructuredError)
func (h *EngineHandler) SubscribeLogs(ctx, req) (<-chan RuntimeEvent, *StructuredError)
func (h *EngineHandler) SubscribeTraffic(ctx, req) (<-chan RuntimeEvent, *StructuredError)
func (h *EngineHandler) SubscribeConnections(ctx, req) (<-chan RuntimeEvent, *StructuredError)
func (h *EngineHandler) GetRuntimeCapabilities(ctx, req) (reply, *StructuredError)
func (h *EngineHandler) ListGroups(ctx, req) (reply, *StructuredError)
func (h *EngineHandler) SelectOutbound(ctx, req) (reply, *StructuredError)
func (h *EngineHandler) URLTest(ctx, req) (reply, *StructuredError)
func (h *EngineHandler) CloseConnection(ctx, req) (reply, *StructuredError)
func (h *EngineHandler) CloseAllConnections(ctx, req) (reply, *StructuredError)
```

Move from service.go: EngineStart through EngineCloseAllConnections + helpers `loadActiveRuntimeStartTarget`, `loadPreparedRuntimeStartTarget`, `loadRuntimeStartTargetByID`, `startPreparedSnapshotTarget`, `prepareRuntimeStartTargetCapabilities`, `preparePlatformFeature`, `isPrivilegedFeature`, `isNetworkExtensionFeature`, `reloadOutcomeForTargetLoadFailure`.

**This handler owns its own `mu`** — replaces the shared `opMu` for engine operations.

#### `core/qkboxd/handler_platform.go`

```go
type PlatformHandler struct {
    proxy      capability.SystemProxyProvider
    privileged capability.PrivilegedProvider
    extension  capability.NetworkExtensionRuntime
    engine     *EngineController
    db         *persistence.DB
}

func (h *PlatformHandler) GetCapabilities(ctx, req) (reply, *StructuredError)
func (h *PlatformHandler) GetPrivilegedProviderStatus(ctx, req) (reply, *StructuredError)
func (h *PlatformHandler) PrepareFeature(ctx, req) (reply, *StructuredError)
func (h *PlatformHandler) RunRepairAction(ctx, req) (reply, *StructuredError)
func (h *PlatformHandler) GetSystemProxyStatus(ctx, req) (reply, *StructuredError)
func (h *PlatformHandler) SetSystemProxyEnabled(ctx, req) (reply, *StructuredError)
```

Move from service.go: all Platform* methods + helpers `platformCapabilities`, `applyPrivilegedCapabilities`, `applyNetworkExtensionCapabilities`, `providerStatusReason`, `networkExtensionStatusReason`, `enableProxy`, `disableProxy`, `restoreProxyIfOwned`, `bestEffortProxyRestore`.

#### `core/qkboxd/handler_diagnostic.go`

```go
type DiagnosticHandler struct {
    db         *persistence.DB
    engine     *EngineController
    proxy      capability.SystemProxyProvider
    privileged capability.PrivilegedProvider
    extension  capability.NetworkExtensionRuntime
    assetStore *assetcache.Store
}

func (h *DiagnosticHandler) GetReport(ctx, req) (reply, *StructuredError)
func (h *DiagnosticHandler) CreateDebugBundle(ctx, req) (reply, *StructuredError)
```

Move from service.go: `buildDiagnosticsReport`, `writeDebugBundle`, `diagnosticChecks`, `diagnosticSubscriptions`, `diagnosticAssets`.

#### `core/qkboxd/wire.go`

```go
func (s *Service) RegisterHandlers(
    methods map[string]ipcframework.MethodHandler,
    subscriptions map[string]ipcframework.SubscriptionHandler,
) {
    profile := &ProfileHandler{db: s.db, key: s.key}
    asset := &AssetHandler{db: s.db, key: s.key, httpClient: s.httpClient, assetStore: s.assetStore}
    snapshot := &SnapshotHandler{db: s.db, key: s.key, engine: s.engine}
    engine := &EngineHandler{engine: s.engine, db: s.db, key: s.key, proxy: s.proxy, privileged: s.privileged, extension: s.extension, events: s.events}
    platform := &PlatformHandler{proxy: s.proxy, privileged: s.privileged, extension: s.extension, engine: s.engine, db: s.db}
    diagnostic := &DiagnosticHandler{db: s.db, engine: s.engine, proxy: s.proxy, privileged: s.privileged, extension: s.extension, assetStore: s.assetStore}

    ipcframework.RegisterMethod(methods, api.MethodHello, s.Hello)
    ipcframework.RegisterMethod(methods, api.MethodCreateProfile, profile.CreateProfile)
    // ... all 34 methods
    ipcframework.RegisterSubscription(subscriptions, api.MethodEngineSubscribeStatus, engine.SubscribeStatus)
    // ... all 4 subscriptions
}
```

### Verification

- [ ] `service.go` is ~30 lines (struct + constructor + Close + Hello + RegisterHandlers)
- [ ] Each handler file is < 300 lines
- [ ] `go test ./...` passes
- [ ] `validate.go` deleted, content absorbed into `handler_snapshot.go`

---

## 1.3 Unify Event Hubs

### Problem

`core/qkboxd/runtime_events.go` (exported `RuntimeEventHub`) and `internal/providerruntime/events.go` (unexported `eventHub`) are near-identical ring-buffer + subscriber implementations.

### Action

Create `internal/eventhub/eventhub.go`:

```go
package eventhub

type Topic string

const (
    TopicStatus Topic = "status"
    TopicLog    Topic = "log"
)

type Hub struct {
    mu          sync.Mutex
    subscribers map[Topic]map[uint64]chan api.RuntimeEvent
    ringBuffers map[Topic][]api.RuntimeEvent
    lastValues  map[Topic]api.RuntimeEvent  // for status replay
    nextID      atomic.Uint64
    nextSeq     atomic.Uint64
    ringLimit   int
}

func New(ringLimit int) *Hub

// Publish sends an event to all subscribers of the given topic.
func (h *Hub) Publish(topic Topic, event api.RuntimeEvent)

// Convenience methods:
func (h *Hub) PublishStatus(status api.EngineStatus)
func (h *Hub) PublishLog(source, level, message string)
func (h *Hub) PublishBridgeError(err *api.StructuredError)

// Subscribe returns a channel that receives events for the topic.
// New subscribers get replay: last status value (for TopicStatus) or full ring buffer (for TopicLog).
func (h *Hub) Subscribe(ctx context.Context, topic Topic) <-chan api.RuntimeEvent
```

**Delete:** `core/qkboxd/runtime_events.go`, `internal/providerruntime/events.go`

**Update consumers:**
- `core/qkboxd/engine.go`: `RuntimeEventHub` → `*eventhub.Hub`, `PublishStatus` → `hub.PublishStatus`
- `core/qkboxd/runtime_owner.go`: same
- `core/qkboxd/provider_runtime_owner.go`: same
- `internal/providerruntime/controller.go`: `eventHub` → `*eventhub.Hub`, `PublishRuntimeLog` → `hub.PublishLog`

### Verification

- [ ] `core/qkboxd/runtime_events.go` and `internal/providerruntime/events.go` deleted
- [ ] `TestRuntimeEventHubReplaysLogRing` ported to `internal/eventhub/eventhub_test.go`
- [ ] `TestEngineControllerPublishesStatusEvents` still passes

---

## 1.4 Simplify Error System

### Problem

`StructuredError` has 7 fields. Frontend only uses `Code` and `Message`. `Detail`, `Source`, `Recoverable`, `UserAction`, `DebugRef` are never consumed.

### Action

In `shared/api/errors.go`:

```go
type StructuredError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

func NewError(code, message string) *StructuredError {
    return &StructuredError{Code: code, Message: message}
}
```

Remove: `Detail`, `Source`, `Recoverable`, `UserAction`, `DebugRef` fields. Remove `NewStructuredError` constructor. Remove `VersionUnsupported` function.

**Mechanical replacement across ~15 Go files:**
- `api.NewStructuredError(code, msg, source, recoverable)` → `api.NewError(code, msg)`
- Places that set `Detail`: fold into `Message` string or drop
- Places that check `Recoverable`/`Source`: remove checks

Files to update (grep for `NewStructuredError` and `StructuredError`):
- `core/qkboxd/service.go` (will be split into handlers)
- `core/qkboxd/engine.go`
- `core/qkboxd/runtime_owner.go`
- `core/qkboxd/provider_runtime_owner.go`
- `internal/ipc/server.go` (will be replaced by ipcframework)
- `internal/provideripc/server.go` (will be replaced by ipcframework)
- `internal/singboxadapter/errors.go`
- `internal/providerruntime/controller.go`
- `platform/capability/privileged.go`
- `shared/api/errors.go`
- `shared/api/contract_test.go`
- `apps/desktop/bridge.go`

### Verification

- [ ] `StructuredError` has exactly 2 fields
- [ ] No references to `NewStructuredError`, `Detail`, `Source`, `Recoverable`, `UserAction` remain
- [ ] `go test ./...` passes
- [ ] `contract_test.go` updated for new shape

---

## 1.5 Simplify Version Negotiation

### Problem

Desktop and qkboxd are co-released. `MinSupportedAPIVersion` enforcement is premature.

### Action

`shared/api/version.go`:
```go
const (
    APIVersion     = "1"
    SchemaRevision = "2026-06-05.release-diagnostics"
    AppVersion     = "0.1.0"
    QKBoxDVersion  = "0.1.0"
)
```
Remove `MinSupportedAPIVersion`.

`shared/api/hello.go`:
- Remove `HelloRequest.ClientAPIVersion` field
- Remove `DefaultHelloRequest()` (or simplify to `HelloRequest{}`)
- Keep `HelloReply` as-is (APIVersion, SchemaRevision, etc. are informational)

`core/qkboxd/service.go` (Hello handler):
- Remove version check: `if req.ClientAPIVersion != api.APIVersion`
- Just return the reply

### Verification

- [ ] `MinSupportedAPIVersion` constant removed
- [ ] `HelloRequest` has no `ClientAPIVersion` field
- [ ] Hello handler does not reject based on version
- [ ] `go test ./...` passes
