package singboxadapter

import (
	"context"
	"errors"
	"testing"

	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
)

type fakeBox struct {
	startErr error
	closeErr error
	closed   bool
}

func (f *fakeBox) Start() error {
	return f.startErr
}

func (f *fakeBox) Close() error {
	f.closed = true
	return f.closeErr
}

func TestAdapterClosesBoxWhenStartFails(t *testing.T) {
	box := &fakeBox{startErr: errors.New("start failed")}
	adapter := &Adapter{
		newBox: func(context.Context, string, log.PlatformWriter) (boxHandle, error) {
			return box, nil
		},
	}

	err := adapter.Start(context.Background(), "{}")
	if err == nil {
		t.Fatal("expected start error")
	}
	if !box.closed {
		t.Fatal("expected failed box to be closed")
	}
	if adapter.b != nil {
		t.Fatal("failed box must not be retained")
	}
}

func TestDisableExternalClashController(t *testing.T) {
	options := option.Options{
		Experimental: &option.ExperimentalOptions{
			ClashAPI: &option.ClashAPIOptions{ExternalController: "127.0.0.1:9090"},
		},
	}
	disableExternalClashController(&options)
	if options.Experimental.ClashAPI.ExternalController != "" {
		t.Fatalf("external controller = %q", options.Experimental.ClashAPI.ExternalController)
	}
}
