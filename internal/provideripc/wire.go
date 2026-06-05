package provideripc

import (
	"github.com/zclkkk/qkbox/internal/ipcframework"
)

func newRegistry(handler Handler) *ipcframework.Registry {
	registry := ipcframework.NewRegistry("provider")

	ipcframework.RegisterMethod(registry, MethodGetStatus, handler.GetStatus)
	ipcframework.RegisterMethod(registry, MethodPrepareFeature, handler.PrepareFeature)
	ipcframework.RegisterMethod(registry, MethodRunRepairAction, handler.RunRepairAction)
	ipcframework.RegisterMethod(registry, MethodRuntimeStart, handler.RuntimeStart)
	ipcframework.RegisterMethod(registry, MethodRuntimeStop, handler.RuntimeStop)
	ipcframework.RegisterMethod(registry, MethodRuntimeHeartbeat, handler.RuntimeHeartbeat)
	ipcframework.RegisterMethod(registry, MethodRuntimeGetStatus, handler.RuntimeGetStatus)
	ipcframework.RegisterMethod(registry, MethodRuntimeGetRuntimeCapabilities, handler.RuntimeGetRuntimeCapabilities)
	ipcframework.RegisterMethod(registry, MethodRuntimeGetTraffic, handler.RuntimeGetTraffic)
	ipcframework.RegisterMethod(registry, MethodRuntimeGetConnections, handler.RuntimeGetConnections)
	ipcframework.RegisterMethod(registry, MethodRuntimeListGroups, handler.RuntimeListGroups)
	ipcframework.RegisterMethod(registry, MethodRuntimeSelectOutbound, handler.RuntimeSelectOutbound)
	ipcframework.RegisterMethod(registry, MethodRuntimeURLTest, handler.RuntimeURLTest)
	ipcframework.RegisterMethod(registry, MethodRuntimeCloseConnection, handler.RuntimeCloseConnection)
	ipcframework.RegisterMethod(registry, MethodRuntimeCloseAllConnections, handler.RuntimeCloseAllConnections)
	ipcframework.RegisterMethod(registry, MethodRuntimeListenerInfo, handler.RuntimeListenerInfo)
	ipcframework.RegisterSubscription(registry, MethodRuntimeSubscribeEvents, RuntimeSubscribeEventsReply{}, handler.RuntimeSubscribeEvents)

	return registry
}
