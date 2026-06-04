package singboxadapter

import (
	"context"
	"fmt"
	"net"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	sbAdapter "github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing/common/buf"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"github.com/zclkkk/qkbox/shared/api"
)

type runtimeLogWriter struct {
	sink RuntimeEventSink
}

func (w runtimeLogWriter) WriteMessage(level log.Level, message string) {
	if w.sink == nil {
		return
	}
	w.sink.PublishRuntimeLog("runtime", log.FormatLevel(level), message)
}

type trafficTracker struct {
	nextID        atomic.Uint64
	uploadTotal   atomic.Int64
	downloadTotal atomic.Int64
	mu            sync.RWMutex
	connections   map[string]*trackedConnection
}

type trackedConnection struct {
	id          string
	network     string
	source      string
	destination string
	host        string
	process     string
	inbound     string
	outbound    string
	rule        string
	startedAt   int64
	upload      atomic.Int64
	download    atomic.Int64
	closeOnce   sync.Once
	closeFunc   func() error
	onClose     func(string)
}

func newTrafficTracker() *trafficTracker {
	return &trafficTracker{connections: make(map[string]*trackedConnection)}
}

func (t *trafficTracker) RuntimeCapabilities() []api.Capability {
	return []api.Capability{
		{Name: "RUNTIME_LIFECYCLE", State: api.CapabilityAvailable},
		{Name: "STATUS_STREAM", State: api.CapabilityAvailable},
		{Name: "LOG_SOURCE", State: api.CapabilityAvailable},
		{Name: "TRAFFIC_SOURCE", State: api.CapabilityAvailable},
		{Name: "CONNECTION_SOURCE", State: api.CapabilityAvailable},
		{Name: "GROUP_SOURCE", State: api.CapabilityAvailable},
		{Name: "SELECT_OUTBOUND", State: api.CapabilityAvailable},
		{Name: "URL_TEST_SOURCE", State: api.CapabilityAvailable},
	}
}

func (t *trafficTracker) RoutedConnection(_ context.Context, conn net.Conn, metadata sbAdapter.InboundContext, matchedRule sbAdapter.Rule, matchOutbound sbAdapter.Outbound) net.Conn {
	tracked := t.addConnection(metadata, matchedRule, matchOutbound)
	wrapped := &trackedTCPConn{Conn: conn, tracker: t, tracked: tracked}
	tracked.closeFunc = wrapped.closeUnderlying
	return wrapped
}

func (t *trafficTracker) RoutedPacketConnection(_ context.Context, conn N.PacketConn, metadata sbAdapter.InboundContext, matchedRule sbAdapter.Rule, matchOutbound sbAdapter.Outbound) N.PacketConn {
	tracked := t.addConnection(metadata, matchedRule, matchOutbound)
	wrapped := &trackedPacketConn{PacketConn: conn, tracker: t, tracked: tracked}
	tracked.closeFunc = wrapped.closeUnderlying
	return wrapped
}

func (t *trafficTracker) addConnection(metadata sbAdapter.InboundContext, matchedRule sbAdapter.Rule, matchOutbound sbAdapter.Outbound) *trackedConnection {
	id := fmt.Sprintf("conn_%d", t.nextID.Add(1))
	tracked := &trackedConnection{
		id:          id,
		network:     metadata.Network,
		source:      socksaddrString(metadata.Source),
		destination: socksaddrString(metadata.Destination),
		host:        connectionHost(metadata),
		process:     processString(metadata),
		inbound:     inboundString(metadata),
		outbound:    outboundString(matchOutbound),
		rule:        ruleString(matchedRule),
		startedAt:   time.Now().UnixMilli(),
		onClose:     t.remove,
	}
	t.mu.Lock()
	t.connections[id] = tracked
	t.mu.Unlock()
	return tracked
}

func (t *trafficTracker) addUpload(tracked *trackedConnection, n int) {
	if n <= 0 {
		return
	}
	size := int64(n)
	tracked.upload.Add(size)
	t.uploadTotal.Add(size)
}

func (t *trafficTracker) addDownload(tracked *trackedConnection, n int) {
	if n <= 0 {
		return
	}
	size := int64(n)
	tracked.download.Add(size)
	t.downloadTotal.Add(size)
}

func (t *trafficTracker) remove(id string) {
	t.mu.Lock()
	delete(t.connections, id)
	t.mu.Unlock()
}

func (t *trafficTracker) TrafficSnapshot(previous api.TrafficSnapshot) api.TrafficSnapshot {
	now := time.Now().UnixMilli()
	uploadTotal := t.uploadTotal.Load()
	downloadTotal := t.downloadTotal.Load()
	elapsed := now - previous.Timestamp
	var uploadRate int64
	var downloadRate int64
	if elapsed > 0 {
		uploadRate = (uploadTotal - previous.UploadTotal) * 1000 / elapsed
		downloadRate = (downloadTotal - previous.DownloadTotal) * 1000 / elapsed
	}
	return api.TrafficSnapshot{
		Timestamp:     now,
		UploadTotal:   uploadTotal,
		DownloadTotal: downloadTotal,
		UploadRate:    uploadRate,
		DownloadRate:  downloadRate,
	}
}

func (t *trafficTracker) ConnectionSnapshot() api.ConnectionSnapshot {
	t.mu.RLock()
	connections := make([]api.RuntimeConnection, 0, len(t.connections))
	for _, tracked := range t.connections {
		connections = append(connections, api.RuntimeConnection{
			ID:          tracked.id,
			Network:     tracked.network,
			Source:      tracked.source,
			Destination: tracked.destination,
			Host:        tracked.host,
			Process:     tracked.process,
			Inbound:     tracked.inbound,
			Outbound:    tracked.outbound,
			Rule:        tracked.rule,
			Upload:      tracked.upload.Load(),
			Download:    tracked.download.Load(),
			StartedAt:   tracked.startedAt,
		})
	}
	t.mu.RUnlock()
	sort.Slice(connections, func(i, j int) bool {
		return connections[i].StartedAt < connections[j].StartedAt
	})
	return api.ConnectionSnapshot{
		Timestamp:     time.Now().UnixMilli(),
		UploadTotal:   t.uploadTotal.Load(),
		DownloadTotal: t.downloadTotal.Load(),
		Connections:   connections,
	}
}

func (t *trafficTracker) CloseConnection(id string) bool {
	t.mu.RLock()
	tracked := t.connections[id]
	t.mu.RUnlock()
	if tracked == nil {
		return false
	}
	_ = tracked.close()
	return true
}

func (t *trafficTracker) CloseAll() {
	t.mu.RLock()
	connections := make([]*trackedConnection, 0, len(t.connections))
	for _, tracked := range t.connections {
		connections = append(connections, tracked)
	}
	t.mu.RUnlock()
	for _, tracked := range connections {
		_ = tracked.close()
	}
}

func (c *trackedConnection) close() error {
	var err error
	c.closeOnce.Do(func() {
		if c.onClose != nil {
			c.onClose(c.id)
		}
		if c.closeFunc != nil {
			err = c.closeFunc()
		}
	})
	return err
}

type trackedTCPConn struct {
	net.Conn
	tracker *trafficTracker
	tracked *trackedConnection
}

func (c *trackedTCPConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	c.tracker.addUpload(c.tracked, n)
	return n, err
}

func (c *trackedTCPConn) Write(p []byte) (int, error) {
	n, err := c.Conn.Write(p)
	c.tracker.addDownload(c.tracked, n)
	return n, err
}

func (c *trackedTCPConn) Close() error {
	return c.tracked.close()
}

func (c *trackedTCPConn) closeUnderlying() error {
	return c.Conn.Close()
}

type trackedPacketConn struct {
	N.PacketConn
	tracker *trafficTracker
	tracked *trackedConnection
}

func (c *trackedPacketConn) ReadPacket(buffer *buf.Buffer) (M.Socksaddr, error) {
	destination, err := c.PacketConn.ReadPacket(buffer)
	c.tracker.addUpload(c.tracked, buffer.Len())
	return destination, err
}

func (c *trackedPacketConn) WritePacket(buffer *buf.Buffer, destination M.Socksaddr) error {
	c.tracker.addDownload(c.tracked, buffer.Len())
	return c.PacketConn.WritePacket(buffer, destination)
}

func (c *trackedPacketConn) Close() error {
	return c.tracked.close()
}

func (c *trackedPacketConn) closeUnderlying() error {
	return c.PacketConn.Close()
}

func socksaddrString(addr M.Socksaddr) string {
	if !addr.IsValid() {
		return ""
	}
	return addr.String()
}

func connectionHost(metadata sbAdapter.InboundContext) string {
	if metadata.Domain != "" {
		return metadata.Domain
	}
	return metadata.Destination.Fqdn
}

func processString(metadata sbAdapter.InboundContext) string {
	if metadata.ProcessInfo == nil {
		return ""
	}
	if metadata.ProcessInfo.ProcessPath != "" {
		return metadata.ProcessInfo.ProcessPath
	}
	if len(metadata.ProcessInfo.AndroidPackageNames) > 0 {
		return metadata.ProcessInfo.AndroidPackageNames[0]
	}
	if metadata.ProcessInfo.UserName != "" {
		return metadata.ProcessInfo.UserName
	}
	if metadata.ProcessInfo.UserId != -1 {
		return fmt.Sprintf("%d", metadata.ProcessInfo.UserId)
	}
	return ""
}

func inboundString(metadata sbAdapter.InboundContext) string {
	if metadata.Inbound != "" && metadata.InboundType != "" {
		return metadata.InboundType + "/" + metadata.Inbound
	}
	if metadata.Inbound != "" {
		return metadata.Inbound
	}
	return metadata.InboundType
}

func outboundString(outbound sbAdapter.Outbound) string {
	if outbound == nil {
		return ""
	}
	return outbound.Tag()
}

func ruleString(rule sbAdapter.Rule) string {
	if rule == nil {
		return "final"
	}
	return rule.String() + " => " + rule.Action().String()
}

type selectableGroup interface {
	SelectOutbound(tag string) bool
}

func outboundGroupDTO(group sbAdapter.Outbound, manager sbAdapter.OutboundManager) api.OutboundGroup {
	outboundGroup := group.(sbAdapter.OutboundGroup)
	options := make([]api.OutboundOption, 0, len(outboundGroup.All()))
	for _, tag := range outboundGroup.All() {
		option := api.OutboundOption{Tag: tag}
		if outbound, ok := manager.Outbound(tag); ok {
			option.Type = outbound.Type()
		}
		options = append(options, option)
	}
	return api.OutboundGroup{
		Tag:       group.Tag(),
		Type:      group.Type(),
		Selected:  outboundGroup.Now(),
		Outbounds: options,
	}
}

func urlTestResults(ctx context.Context, group sbAdapter.Outbound) []api.URLTestResult {
	if urlTestGroup, ok := group.(sbAdapter.URLTestGroup); ok {
		results, err := urlTestGroup.URLTest(ctx)
		if err != nil {
			return []api.URLTestResult{{
				Outbound:     group.Tag(),
				ErrorCode:    api.ErrorObservabilityDegraded,
				ErrorMessage: err.Error(),
			}}
		}
		mapped := make([]api.URLTestResult, 0, len(results))
		for tag, delay := range results {
			mapped = append(mapped, api.URLTestResult{Outbound: tag, DelayMS: int64(delay)})
		}
		sort.Slice(mapped, func(i, j int) bool { return mapped[i].Outbound < mapped[j].Outbound })
		return mapped
	}

	return nil
}
