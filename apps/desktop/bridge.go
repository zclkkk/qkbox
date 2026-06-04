package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/zclkkk/qkbox/internal/ipc"
	"github.com/zclkkk/qkbox/shared/api"
)

type BridgeService struct {
	client      *ipc.Client
	eventMu     sync.Mutex
	eventCancel context.CancelFunc
}

func NewBridgeService() *BridgeService {
	return &BridgeService{client: ipc.NewClient()}
}

func (b *BridgeService) Hello(ctx context.Context) api.HelloResult {
	reply, structured := b.client.Hello(ctx, api.DefaultHelloRequest())
	if structured == nil {
		return api.HelloResult{Reply: &reply}
	}
	if structured.Code != api.ErrorIPCTransport {
		return api.HelloResult{Error: structured}
	}

	if launchErr := launchQKBoxD(); launchErr != nil {
		return api.HelloResult{
			Error: api.NewStructuredError(api.ErrorDaemonLaunchFailed, launchErr.Error(), "desktop", true),
		}
	}

	readyCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := ipc.WaitForReady(readyCtx); err != nil {
		return api.HelloResult{
			Error: api.NewStructuredError(api.ErrorDaemonUnavailable, err.Error(), "desktop", true),
		}
	}

	reply, structured = b.client.Hello(ctx, api.DefaultHelloRequest())
	if structured != nil {
		return api.HelloResult{Error: structured}
	}
	return api.HelloResult{Reply: &reply}
}

// Engine lifecycle

func (b *BridgeService) EngineStart(ctx context.Context) api.EngineStartResult {
	reply, structured := b.client.EngineStart(ctx, api.EngineStartRequest{})
	if structured != nil {
		return api.EngineStartResult{Error: structured}
	}
	return api.EngineStartResult{Reply: &reply}
}

func (b *BridgeService) EngineStop(ctx context.Context) api.EngineStopResult {
	reply, structured := b.client.EngineStop(ctx, api.EngineStopRequest{})
	if structured != nil {
		return api.EngineStopResult{Error: structured}
	}
	return api.EngineStopResult{Reply: &reply}
}

func (b *BridgeService) EngineGetStatus(ctx context.Context) api.EngineGetStatusResult {
	reply, structured := b.client.EngineGetStatus(ctx, api.EngineGetStatusRequest{})
	if structured != nil {
		return api.EngineGetStatusResult{Error: structured}
	}
	return api.EngineGetStatusResult{Reply: &reply}
}

func (b *BridgeService) StartRuntimeEventBridge(ctx context.Context) api.RuntimeEventBridgeStartResult {
	hello := b.Hello(ctx)
	if hello.Error != nil {
		return api.RuntimeEventBridgeStartResult{Error: hello.Error}
	}

	b.eventMu.Lock()
	if b.eventCancel != nil {
		b.eventCancel()
	}
	bridgeCtx, cancel := context.WithCancel(context.Background())
	b.eventCancel = cancel
	b.eventMu.Unlock()

	subscriptions := []func(context.Context) (<-chan ipc.EventFrame, *api.StructuredError){
		func(ctx context.Context) (<-chan ipc.EventFrame, *api.StructuredError) {
			return b.client.EngineSubscribeStatus(ctx, api.EngineSubscribeStatusRequest{})
		},
		func(ctx context.Context) (<-chan ipc.EventFrame, *api.StructuredError) {
			return b.client.EngineSubscribeLogs(ctx, api.EngineSubscribeLogsRequest{})
		},
		func(ctx context.Context) (<-chan ipc.EventFrame, *api.StructuredError) {
			return b.client.EngineSubscribeTraffic(ctx, api.EngineSubscribeTrafficRequest{})
		},
		func(ctx context.Context) (<-chan ipc.EventFrame, *api.StructuredError) {
			return b.client.EngineSubscribeConnections(ctx, api.EngineSubscribeConnectionsRequest{})
		},
	}
	for _, open := range subscriptions {
		events, structured := open(bridgeCtx)
		if structured != nil {
			cancel()
			return api.RuntimeEventBridgeStartResult{Error: structured}
		}
		go forwardRuntimeEvents(bridgeCtx, events)
	}
	return api.RuntimeEventBridgeStartResult{Reply: &api.RuntimeEventBridgeStartReply{}}
}

func (b *BridgeService) StopRuntimeEventBridge(context.Context) api.RuntimeEventBridgeStopResult {
	b.eventMu.Lock()
	if b.eventCancel != nil {
		b.eventCancel()
		b.eventCancel = nil
	}
	b.eventMu.Unlock()
	return api.RuntimeEventBridgeStopResult{Reply: &api.RuntimeEventBridgeStopReply{}}
}

func (b *BridgeService) EngineGetRuntimeCapabilities(ctx context.Context) api.EngineGetRuntimeCapabilitiesResult {
	reply, structured := b.client.EngineGetRuntimeCapabilities(ctx, api.EngineGetRuntimeCapabilitiesRequest{})
	if structured != nil {
		return api.EngineGetRuntimeCapabilitiesResult{Error: structured}
	}
	return api.EngineGetRuntimeCapabilitiesResult{Reply: &reply}
}

func (b *BridgeService) EngineListGroups(ctx context.Context) api.EngineListGroupsResult {
	reply, structured := b.client.EngineListGroups(ctx, api.EngineListGroupsRequest{})
	if structured != nil {
		return api.EngineListGroupsResult{Error: structured}
	}
	return api.EngineListGroupsResult{Reply: &reply}
}

func (b *BridgeService) EngineSelectOutbound(ctx context.Context, req api.EngineSelectOutboundRequest) api.EngineSelectOutboundResult {
	reply, structured := b.client.EngineSelectOutbound(ctx, req)
	if structured != nil {
		return api.EngineSelectOutboundResult{Error: structured}
	}
	return api.EngineSelectOutboundResult{Reply: &reply}
}

func (b *BridgeService) EngineURLTest(ctx context.Context, req api.EngineURLTestRequest) api.EngineURLTestResult {
	reply, structured := b.client.EngineURLTest(ctx, req)
	if structured != nil {
		return api.EngineURLTestResult{Error: structured}
	}
	return api.EngineURLTestResult{Reply: &reply}
}

func (b *BridgeService) EngineCloseConnection(ctx context.Context, req api.EngineCloseConnectionRequest) api.EngineCloseConnectionResult {
	reply, structured := b.client.EngineCloseConnection(ctx, req)
	if structured != nil {
		return api.EngineCloseConnectionResult{Error: structured}
	}
	return api.EngineCloseConnectionResult{Reply: &reply}
}

func (b *BridgeService) EngineCloseAllConnections(ctx context.Context) api.EngineCloseAllConnectionsResult {
	reply, structured := b.client.EngineCloseAllConnections(ctx, api.EngineCloseAllConnectionsRequest{})
	if structured != nil {
		return api.EngineCloseAllConnectionsResult{Error: structured}
	}
	return api.EngineCloseAllConnectionsResult{Reply: &reply}
}

// Platform capabilities

func (b *BridgeService) PlatformGetSystemProxyStatus(ctx context.Context) api.GetSystemProxyStatusResult {
	reply, structured := b.client.PlatformGetSystemProxyStatus(ctx, api.GetSystemProxyStatusRequest{})
	if structured != nil {
		return api.GetSystemProxyStatusResult{Error: structured}
	}
	return api.GetSystemProxyStatusResult{Reply: &reply}
}

func (b *BridgeService) PlatformSetSystemProxyEnabled(ctx context.Context, req api.SetSystemProxyEnabledRequest) api.SetSystemProxyEnabledResult {
	reply, structured := b.client.PlatformSetSystemProxyEnabled(ctx, req)
	if structured != nil {
		return api.SetSystemProxyEnabledResult{Error: structured}
	}
	return api.SetSystemProxyEnabledResult{Reply: &reply}
}

func forwardRuntimeEvents(ctx context.Context, events <-chan ipc.EventFrame) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			emitRuntimeEvent(event)
		}
	}
}

func emitRuntimeEvent(frame ipc.EventFrame) {
	if frame.Error != nil {
		application.Get().Event.Emit(api.EventEngineEventBridgeError, frame.Error)
		return
	}
	if frame.Event == "" {
		return
	}
	var payload interface{}
	if len(frame.Data) > 0 {
		if err := json.Unmarshal(frame.Data, &payload); err != nil {
			application.Get().Event.Emit(api.EventEngineEventBridgeError, api.NewStructuredError(api.ErrorIPCTransport, err.Error(), "desktop", true))
			return
		}
	}
	application.Get().Event.Emit(frame.Event, payload)
}

func launchQKBoxD() error {
	path, err := findQKBoxD()
	if err != nil {
		return err
	}
	cmd := exec.Command(path)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	prepareDetachedCmd(cmd)
	return cmd.Start()
}

func findQKBoxD() (string, error) {
	if path := os.Getenv("QKBOX_QKBOXD_PATH"); path != "" {
		return path, nil
	}
	name := "qkboxd"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}

	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		for _, candidate := range []string{
			filepath.Join(dir, name),
			filepath.Join(dir, "..", name),
			filepath.Join(dir, "..", "..", "bin", name),
		} {
			if exists(candidate) {
				return candidate, nil
			}
		}
	}

	wd, err := os.Getwd()
	if err == nil {
		for _, candidate := range []string{
			filepath.Join(wd, "bin", name),
			filepath.Join(wd, "..", "..", "bin", name),
		} {
			if exists(candidate) {
				return candidate, nil
			}
		}
	}
	return "", errors.New("qkboxd binary not found; run npm run build:qkboxd or set QKBOX_QKBOXD_PATH")
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
