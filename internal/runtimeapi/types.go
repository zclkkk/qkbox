package runtimeapi

// ListenerInfo describes a running inbound listener.
// This is an internal DTO — not exposed via IPC or shared/model.
type ListenerInfo struct {
	Tag     string `json:"tag"`
	Type    string `json:"type"`    // "http", "mixed"
	Address string `json:"address"` // normalized (0.0.0.0/::/"" -> 127.0.0.1)
	Port    int    `json:"port"`
}
