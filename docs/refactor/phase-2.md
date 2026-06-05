# Phase 2: sing-box Integration Deepening

> Depends on Phase 1 (unified IPC framework).
> Can run in parallel with Phase 3.

## 2.1 Replace Custom Traffic Tracking with Clash API

### Problem

`internal/singboxadapter/observability.go` (370 lines) re-implements connection tracking by wrapping every `net.Conn`/`N.PacketConn` with custom `trackedTCPConn`/`trackedPacketConn` that count bytes on Read/Write. Meanwhile, `adapter.go` line 149 explicitly disables sing-box's Clash API:

```go
func disableExternalClashController(options *option.Options) {
    options.Experimental.ClashAPI.ExternalController = ""
}
```

sing-box's Clash API provides all the same data (connections, traffic, groups, URLTest) natively, with richer metadata (rule matches, DNS resolution, process names).

### Action

#### Create `internal/singboxadapter/clash_bridge.go`

An HTTP client wrapping the Clash-compatible REST API that sing-box exposes.

```go
type ClashBridge struct {
    baseURL    string       // http://127.0.0.1:<random-port>
    httpClient *http.Client
}

func NewClashBridge(baseURL string) *ClashBridge

// Traffic monitoring (WebSocket stream)
func (b *ClashBridge) StreamTraffic(ctx context.Context) (<-chan api.TrafficSnapshot, error)

// Connections
func (b *ClashBridge) GetConnections(ctx context.Context) (api.ConnectionSnapshot, error)
func (b *ClashBridge) CloseConnection(ctx context.Context, id string) error
func (b *ClashBridge) CloseAllConnections(ctx context.Context) error

// Outbound groups
func (b *ClashBridge) GetGroups(ctx context.Context) ([]api.OutboundGroup, error)
func (b *ClashBridge) SelectOutbound(ctx context.Context, group, outbound string) error

// URL test
func (b *ClashBridge) URLTest(ctx context.Context, group string, timeout time.Duration) ([]api.URLTestResult, error)

// Log stream (WebSocket)
func (b *ClashBridge) StreamLogs(ctx context.Context) (<-chan api.RuntimeLogEntry, error)
```

Clash API endpoints used:
- `GET /connections` → connection list
- `DELETE /connections/:id` → close connection
- `DELETE /connections` → close all
- `GET /traffic` (WebSocket) → real-time traffic up/down
- `GET /logs` (WebSocket) → real-time log stream
- `GET /providers/proxies` → outbound group list
- `GET /proxies/:name` → group details
- `PUT /proxies/:name` → select outbound
- `GET /proxies/:name/delay` → URLTest

#### Modify `internal/singboxadapter/adapter.go`

Replace `disableExternalClashController` with dynamic port allocation:

```go
func configureClashAPI(options *option.Options) (int, error) {
    port, err := randomPort()
    if err != nil {
        return 0, err
    }
    if options.Experimental == nil {
        options.Experimental = new(option.ExperimentalOptions)
    }
    if options.Experimental.ClashAPI == nil {
        options.Experimental.ClashAPI = new(option.ClashAPIOptions)
    }
    options.Experimental.ClashAPI.ExternalController = fmt.Sprintf("127.0.0.1:%d", port)
    options.Experimental.ClashAPI.ExternalUI = "" // no external UI needed
    return port, nil
}
```

After box starts:
```go
clashBridge := NewClashBridge(fmt.Sprintf("http://127.0.0.1:%d", port))
```

#### Modify `internal/singboxadapter/runtime_api.go`

Rewrite all observability methods to delegate to `ClashBridge`:

```go
func (a *Adapter) TrafficSnapshot() (api.TrafficSnapshot, *api.StructuredError) {
    return a.clash.GetConnections(...)  // or StreamTraffic for real-time
}
```

#### Delete `internal/singboxadapter/observability.go`

The entire file: `trafficTracker`, `trackedConnection`, `trackedTCPConn`, `trackedPacketConn`, `RoutedConnection`, `RoutedPacketConnection`, all byte-counting wrappers.

### Verification

- [ ] `observability.go` deleted
- [ ] `ClashBridge.GetConnections()` returns live connection data
- [ ] `ClashBridge.StreamTraffic()` returns real-time traffic via WebSocket
- [ ] `ClashBridge.GetGroups()` returns outbound groups
- [ ] `ClashBridge.SelectOutbound()` switches active outbound
- [ ] `ClashBridge.URLTest()` runs latency test and returns results
- [ ] No `AppendTracker` calls remain in the codebase

---

## 2.2 Proper Configuration Validation

### Problem

`core/qkboxd/validate.go` only checks JSON syntax + presence of `inbounds`/`outbounds` arrays. Does not validate against sing-box's actual option schema. A config with `"type": "invalid_protocol"` passes validation.

### Action

#### Create `internal/singboxadapter/validate.go`

```go
func ValidateConfig(configJSON string) model.Diagnostics
```

Implementation:
1. Parse with `sjson.UnmarshalExtendedContext[option.Options]` (same parser sing-box uses internally)
2. If parse fails → return error diagnostic with the parse error message
3. If parse succeeds → validate structure:
   - `len(outbounds) == 0` → error
   - `len(inbounds) == 0` → warning
   - Each inbound/outbound has a recognized `type`
   - Route rules reference existing outbound tags
   - DNS servers reference valid types
4. Return `model.Diagnostics{Status, Entries}`

#### Modify snapshot handler

Replace calls to `validateContent()` (from old `validate.go`) with `singboxadapter.ValidateConfig()`.

Move `extractRequiredCapabilities` into the snapshot handler (it's snapshot-specific logic, not validation).

#### Delete `core/qkboxd/validate.go`

All content either moves to snapshot handler or to `internal/singboxadapter/validate.go`.

### Verification

- [ ] `validate.go` deleted from `core/qkboxd/`
- [ ] A config with `"type": "invalid_protocol"` produces an error diagnostic
- [ ] A config with missing `outbounds` produces an error
- [ ] A config with missing `inbounds` produces a warning
- [ ] A valid sing-box config passes validation

---

## 2.3 Config Template System

### Problem

Users must write raw JSON from scratch. No building blocks for common configurations.

### Action

#### Create `internal/configtemplate/`

```
internal/configtemplate/
  templates.go  — Template/Parameter types, Registry
  render.go     — Go template rendering
  builtin.go    — Built-in templates
```

```go
// templates.go
type Template struct {
    ID          string      `json:"id"`
    Name        string      `json:"name"`
    Description string      `json:"description"`
    Category    string      `json:"category"` // "proxy", "tun", "advanced"
    Parameters  []Parameter `json:"parameters"`
    Body        string      `json:"body"` // Go text/template
}

type Parameter struct {
    Name     string   `json:"name"`
    Label    string   `json:"label"`
    Type     string   `json:"type"` // "string", "int", "select", "bool"
    Default  string   `json:"default"`
    Required bool     `json:"required"`
    Options  []string `json:"options,omitempty"`
}

func Registry() []Template
func Render(tmpl Template, params map[string]string) (string, error)
```

#### Built-in templates (`builtin.go`)

1. **HTTP/Mixed Proxy** — mixed inbound on port 7890, direct outbound, basic DNS
2. **SOCKS5 Proxy** — socks inbound on port 1080, direct outbound
3. **TUN Mode** — tun inbound, fake-ip DNS, route rules for common domains
4. **Remote Subscription** — template that takes a subscription URL and wraps it with local inbounds + DNS

Each template is a valid sing-box JSON config with Go template placeholders:
```json
{
  "inbounds": [
    {
      "type": "mixed",
      "listen": "{{.listen_addr}}",
      "listen_port": {{.listen_port}}
    }
  ],
  ...
}
```

#### Expose via IPC

Add to profile handler:
```go
func (h *ProfileHandler) ListTemplates(ctx, req) (reply, *StructuredError)
func (h *ProfileHandler) RenderTemplate(ctx, req) (reply, *StructuredError)
```

Add method constants: `MethodListTemplates`, `MethodRenderTemplate`.

### Verification

- [ ] `Registry()` returns 4+ templates
- [ ] `Render()` produces valid JSON that passes `singboxadapter.ValidateConfig()`
- [ ] IPC methods return template list and rendered config
- [ ] Template parameters have sensible defaults
