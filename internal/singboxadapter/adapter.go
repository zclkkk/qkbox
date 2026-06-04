package singboxadapter

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"

	box "github.com/sagernet/sing-box"
	sbAdapter "github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	sjson "github.com/sagernet/sing/common/json"
	"github.com/zclkkk/qkbox/shared/api"
)

type boxHandle interface {
	Start() error
	Close() error
}

type observableBoxHandle interface {
	boxHandle
	Router() sbAdapter.Router
	Outbound() sbAdapter.OutboundManager
}

type RuntimeEventSink interface {
	PublishRuntimeLog(source, level, message string)
}

type Adapter struct {
	b           boxHandle
	tracker     *trafficTracker
	sink        RuntimeEventSink
	trafficMu   sync.Mutex
	lastTraffic api.TrafficSnapshot
	newBox      func(ctx context.Context, configJSON string, platformWriter log.PlatformWriter) (boxHandle, error)
}

func NewAdapter(sinks ...RuntimeEventSink) *Adapter {
	var sink RuntimeEventSink
	if len(sinks) > 0 {
		sink = sinks[0]
	}
	return &Adapter{sink: sink, tracker: newTrafficTracker(), newBox: newBox}
}

func (a *Adapter) Start(ctx context.Context, configJSON string) error {
	if a.newBox == nil {
		a.newBox = newBox
	}
	if a.tracker == nil {
		a.tracker = newTrafficTracker()
	}
	if a.b != nil {
		return &AdapterError{Code: "START_FAILED", Err: errors.New("adapter already started")}
	}

	b, err := a.newBox(ctx, configJSON, runtimeLogWriter{sink: a.sink})
	if err != nil {
		return err
	}
	if observable, ok := b.(observableBoxHandle); ok {
		observable.Router().AppendTracker(a.tracker)
	}
	if err := b.Start(); err != nil {
		if closeErr := b.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
		return &AdapterError{Code: "START_FAILED", Err: err}
	}

	a.b = b
	return nil
}

func newBox(ctx context.Context, configJSON string, platformWriter log.PlatformWriter) (boxHandle, error) {
	ctx = include.Context(ctx)

	options, err := sjson.UnmarshalExtendedContext[option.Options](ctx, []byte(configJSON))
	if err != nil {
		return nil, &AdapterError{Code: "CONFIG_FAILED", Err: err}
	}
	disableExternalClashController(&options)
	cacheDir, err := isolateImplicitCacheFile(&options, platformWriter)
	if err != nil {
		return nil, &AdapterError{Code: "START_FAILED", Err: err}
	}

	b, err := box.New(box.Options{
		Context:           ctx,
		Options:           options,
		PlatformLogWriter: platformWriter,
	})
	if err != nil {
		cleanupRuntimeCache(cacheDir)
		return nil, &AdapterError{Code: "START_FAILED", Err: err}
	}

	return &managedBox{observableBoxHandle: b, cacheDir: cacheDir}, nil
}

type managedBox struct {
	observableBoxHandle
	cacheDir string
}

func (b *managedBox) Close() error {
	err := b.observableBoxHandle.Close()
	cleanupRuntimeCache(b.cacheDir)
	return err
}

func isolateImplicitCacheFile(options *option.Options, platformWriter log.PlatformWriter) (string, error) {
	if platformWriter == nil {
		return "", nil
	}
	if options.Experimental == nil {
		options.Experimental = new(option.ExperimentalOptions)
	}
	if options.Experimental.CacheFile != nil {
		return "", nil
	}
	cacheDir, err := os.MkdirTemp("", "qkbox-sing-box-cache-*")
	if err != nil {
		return "", err
	}
	options.Experimental.CacheFile = &option.CacheFileOptions{Path: filepath.Join(cacheDir, "cache.db")}
	return cacheDir, nil
}

func disableExternalClashController(options *option.Options) {
	if options.Experimental == nil || options.Experimental.ClashAPI == nil {
		return
	}
	options.Experimental.ClashAPI.ExternalController = ""
}

func cleanupRuntimeCache(cacheDir string) {
	if cacheDir != "" {
		_ = os.RemoveAll(cacheDir)
	}
}

func (a *Adapter) Stop() error {
	if a.b == nil {
		return nil
	}
	if a.tracker != nil {
		a.tracker.CloseAll()
	}
	if err := a.b.Close(); err != nil {
		return &AdapterError{Code: "STOP_FAILED", Err: err}
	}
	a.b = nil
	a.trafficMu.Lock()
	a.lastTraffic = api.TrafficSnapshot{}
	a.trafficMu.Unlock()
	return nil
}
