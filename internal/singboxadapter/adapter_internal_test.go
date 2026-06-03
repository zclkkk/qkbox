package singboxadapter

import (
	"context"
	"errors"
	"testing"
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
		newBox: func(context.Context, string) (boxHandle, error) {
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
