package eventhub

import (
	"context"
	"sync"
	"time"

	"github.com/zclkkk/qkbox/shared/api"
)

const LogRingLimit = 512

type Hub struct {
	mu                sync.Mutex
	nextSubscriberID  uint64
	nextLogSeq        uint64
	lastStatus        api.EngineStatus
	statusSubscribers map[uint64]chan api.RuntimeEvent
	logSubscribers    map[uint64]chan api.RuntimeEvent
	logs              []api.RuntimeLogEntry
}

func New() *Hub {
	return &Hub{
		statusSubscribers: make(map[uint64]chan api.RuntimeEvent),
		logSubscribers:    make(map[uint64]chan api.RuntimeEvent),
	}
}

func (h *Hub) PublishStatus(status api.EngineStatus) {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lastStatus = status
	h.broadcastLocked(h.statusSubscribers, api.RuntimeEvent{Event: api.EventEngineStatus, Data: status})
}

func (h *Hub) PublishRuntimeLog(source, level, message string) {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.nextLogSeq++
	entry := api.RuntimeLogEntry{
		Seq:       h.nextLogSeq,
		Timestamp: time.Now().UnixMilli(),
		Source:    source,
		Level:     level,
		Message:   message,
	}
	if len(h.logs) >= LogRingLimit {
		copy(h.logs, h.logs[1:])
		h.logs[len(h.logs)-1] = entry
	} else {
		h.logs = append(h.logs, entry)
	}
	h.broadcastLocked(h.logSubscribers, api.RuntimeEvent{Event: api.EventEngineLog, Data: entry})
}

func (h *Hub) PublishBridgeError(err *api.StructuredError) {
	if h == nil || err == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.broadcastLocked(h.logSubscribers, api.RuntimeEvent{Event: api.EventEngineEventBridgeError, Error: err})
}

func (h *Hub) SubscribeStatus(ctx context.Context) <-chan api.RuntimeEvent {
	ch := make(chan api.RuntimeEvent, LogRingLimit+16)
	h.mu.Lock()
	id := h.addSubscriberLocked(ch, h.statusSubscribers)
	if h.lastStatus.State != "" {
		send(ch, api.RuntimeEvent{Event: api.EventEngineStatus, Data: h.lastStatus})
	}
	h.mu.Unlock()
	h.removeSubscriberOnDone(ctx, id, h.statusSubscribers)
	return ch
}

func (h *Hub) SubscribeLogs(ctx context.Context) <-chan api.RuntimeEvent {
	ch := make(chan api.RuntimeEvent, LogRingLimit+16)
	h.mu.Lock()
	id := h.addSubscriberLocked(ch, h.logSubscribers)
	for _, entry := range h.logs {
		send(ch, api.RuntimeEvent{Event: api.EventEngineLog, Data: entry})
	}
	h.mu.Unlock()
	h.removeSubscriberOnDone(ctx, id, h.logSubscribers)
	return ch
}

func (h *Hub) SubscribeRuntimeEvents(ctx context.Context) <-chan api.RuntimeEvent {
	return h.SubscribeLogs(ctx)
}

func (h *Hub) addSubscriberLocked(ch chan api.RuntimeEvent, subscribers map[uint64]chan api.RuntimeEvent) uint64 {
	h.nextSubscriberID++
	id := h.nextSubscriberID
	subscribers[id] = ch
	return id
}

func (h *Hub) removeSubscriberOnDone(ctx context.Context, id uint64, subscribers map[uint64]chan api.RuntimeEvent) {
	go func() {
		<-ctx.Done()
		h.mu.Lock()
		delete(subscribers, id)
		h.mu.Unlock()
	}()
}

func (h *Hub) broadcastLocked(subscribers map[uint64]chan api.RuntimeEvent, event api.RuntimeEvent) {
	for _, ch := range subscribers {
		send(ch, event)
	}
}

func send(ch chan api.RuntimeEvent, event api.RuntimeEvent) {
	select {
	case ch <- event:
	default:
	}
}
