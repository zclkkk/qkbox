package qkboxd

import (
	"context"
	"encoding/json"

	"github.com/zclkkk/qkbox/shared/api"
)

// WindowSession tracks the attached qkbox-window process.
// Lifetime = IPC connection lifetime (context-based).
type WindowSession struct {
	Events chan api.RuntimeEvent
	Cancel context.CancelFunc
}

// WindowAttach is the IPC handler for the "window.attach" subscription.
// It registers a long-lived session and returns an event stream.
// The session is automatically removed when the client disconnects (ctx.Done).
func (s *Service) WindowAttach(ctx context.Context, _ api.WindowAttachRequest) (<-chan api.RuntimeEvent, *api.StructuredError) {
	ctx, cancel := context.WithCancel(ctx)
	events := make(chan api.RuntimeEvent, 8)

	s.windowMu.Lock()
	// Close previous session if any (safety — should not happen with a single window).
	if s.windowSess != nil && s.windowSess.Cancel != nil {
		s.windowSess.Cancel()
	}
	sess := &WindowSession{Events: events, Cancel: cancel}
	s.windowSess = sess
	s.windowMu.Unlock()

	// Clean up when window disconnects.
	go func() {
		<-ctx.Done()
		s.windowMu.Lock()
		if s.windowSess == sess {
			s.windowSess = nil
		}
		s.windowMu.Unlock()
	}()

	return events, nil
}

// HasWindowSession reports whether a qkbox-window is currently attached.
func (s *Service) HasWindowSession() bool {
	s.windowMu.Lock()
	defer s.windowMu.Unlock()
	return s.windowSess != nil
}

// NotifyWindowShow sends a "show window" event to the attached session.
// Returns (sent, hasSession): sent=false+hasSession=true means the window exists but is unresponsive.
func (s *Service) NotifyWindowShow() (sent bool, hasSession bool) {
	s.windowMu.Lock()
	defer s.windowMu.Unlock()
	if s.windowSess == nil {
		return false, false
	}
	select {
	case s.windowSess.Events <- api.RuntimeEvent{Event: api.EventWindowShow, Data: json.RawMessage(`{}`)}:
		return true, true
	default:
		return false, true // channel full — window unresponsive
	}
}
