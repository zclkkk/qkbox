package api

// WindowAttachRequest is the request for the window.attach subscription.
// Empty: PID is untrusted and unused. Session lifetime = IPC connection lifetime.
type WindowAttachRequest struct{}

// Event names emitted through the window.attach event stream.
const (
	EventWindowShow = "qkbox.window.show"
)
