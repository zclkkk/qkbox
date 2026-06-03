package model

type EngineState string

const (
	EngineStateUninitialized EngineState = "UNINITIALIZED"
	EngineStateIdle          EngineState = "IDLE"
	EngineStateValidating    EngineState = "VALIDATING"
	EngineStateStarting      EngineState = "STARTING"
	EngineStateStarted       EngineState = "STARTED"
	EngineStateStopping      EngineState = "STOPPING"
	EngineStateFatal         EngineState = "FATAL"
)
