package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/zclkkk/qkbox/internal/ipc"
	"github.com/zclkkk/qkbox/shared/api"
)

type BridgeService struct {
	client *ipc.Client
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
