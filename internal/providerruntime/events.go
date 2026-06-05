package providerruntime

import (
	"context"
	"sync"
	"time"

	"github.com/zclkkk/qkbox/shared/api"
)

const runtimeLogRingLimit = 512

type eventHub struct {
	mu               sync.Mutex
	nextSubscriberID uint64
	nextLogSeq       uint64
	logSubscribers   map[uint64]chan api.RuntimeEvent
	logs             []api.RuntimeLogEntry
}

func newEventHub() *eventHub {
	return &eventHub{
		logSubscribers: make(map[uint64]chan api.RuntimeEvent),
	}
}

func (h *eventHub) PublishRuntimeLog(source, level, message string) {
	if h == nil {
		return
	}
	h.mu.Lock()
	h.nextLogSeq++
	entry := api.RuntimeLogEntry{
		Seq:       h.nextLogSeq,
		Timestamp: time.Now().UnixMilli(),
		Source:    source,
		Level:     level,
		Message:   message,
	}
	if len(h.logs) >= runtimeLogRingLimit {
		copy(h.logs, h.logs[1:])
		h.logs[len(h.logs)-1] = entry
	} else {
		h.logs = append(h.logs, entry)
	}
	h.broadcastLocked(api.RuntimeEvent{Event: api.EventEngineLog, Data: entry})
	h.mu.Unlock()
}

func (h *eventHub) publishBridgeError(err *api.StructuredError) {
	if h == nil || err == nil {
		return
	}
	h.mu.Lock()
	h.broadcastLocked(api.RuntimeEvent{Event: api.EventEngineEventBridgeError, Error: err})
	h.mu.Unlock()
}

func (h *eventHub) subscribe(ctx context.Context) <-chan api.RuntimeEvent {
	ch := make(chan api.RuntimeEvent, runtimeLogRingLimit+16)
	h.mu.Lock()
	h.nextSubscriberID++
	id := h.nextSubscriberID
	h.logSubscribers[id] = ch
	for _, entry := range h.logs {
		sendEvent(ch, api.RuntimeEvent{Event: api.EventEngineLog, Data: entry})
	}
	h.mu.Unlock()
	go func() {
		<-ctx.Done()
		h.mu.Lock()
		delete(h.logSubscribers, id)
		h.mu.Unlock()
	}()
	return ch
}

func (h *eventHub) broadcastLocked(event api.RuntimeEvent) {
	for _, ch := range h.logSubscribers {
		sendEvent(ch, event)
	}
}

func sendEvent(ch chan api.RuntimeEvent, event api.RuntimeEvent) {
	select {
	case ch <- event:
	default:
	}
}
