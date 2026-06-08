package qkboxd

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/zclkkk/qkbox/shared/api"
)

// WindowSession tracks the attached qkbox-window process.
// Lifetime = IPC connection lifetime (context-based).
type WindowSession struct {
	Events chan api.RuntimeEvent
	Cancel context.CancelFunc
}

// Window session state on Service.
var (
	windowMu   sync.Mutex
	windowSess *WindowSession
)

// WindowAttach is the IPC handler for the "window.attach" subscription.
// It registers a long-lived session and returns an event stream.
// The session is automatically removed when the client disconnects (ctx.Done).
func (s *Service) WindowAttach(ctx context.Context, _ api.WindowAttachRequest) (<-chan api.RuntimeEvent, *api.StructuredError) {
	ctx, cancel := context.WithCancel(ctx)
	events := make(chan api.RuntimeEvent, 8)

	windowMu.Lock()
	// Close previous session if any (safety — should not happen with a single window).
	if windowSess != nil && windowSess.Cancel != nil {
		windowSess.Cancel()
	}
	sess := &WindowSession{Events: events, Cancel: cancel}
	windowSess = sess
	windowMu.Unlock()

	// Clean up when window disconnects.
	go func() {
		<-ctx.Done()
		windowMu.Lock()
		if windowSess == sess {
			windowSess = nil
		}
		windowMu.Unlock()
	}()

	return events, nil
}

// HasWindowSession reports whether a qkbox-window is currently attached.
func (s *Service) HasWindowSession() bool {
	windowMu.Lock()
	defer windowMu.Unlock()
	return windowSess != nil
}

// NotifyWindowShow sends a "show window" event to the attached session.
// Returns true if the event was sent, false if no session exists.
func (s *Service) NotifyWindowShow() bool {
	windowMu.Lock()
	defer windowMu.Unlock()
	if windowSess == nil {
		return false
	}
	select {
	case windowSess.Events <- api.RuntimeEvent{Event: api.EventWindowShow, Data: json.RawMessage(`{}`)}:
		return true
	default:
		return false // channel full — window unresponsive
	}
}
