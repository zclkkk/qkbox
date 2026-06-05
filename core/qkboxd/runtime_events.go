package qkboxd

import "github.com/zclkkk/qkbox/internal/eventhub"

const runtimeLogRingLimit = eventhub.LogRingLimit

type RuntimeEventHub = eventhub.Hub

func NewRuntimeEventHub() *RuntimeEventHub {
	return eventhub.New()
}
