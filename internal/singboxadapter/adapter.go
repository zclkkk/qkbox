package singboxadapter

import (
	"context"
	"errors"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	sjson "github.com/sagernet/sing/common/json"
)

type boxHandle interface {
	Start() error
	Close() error
}

type Adapter struct {
	b      boxHandle
	newBox func(ctx context.Context, configJSON string) (boxHandle, error)
}

func NewAdapter() *Adapter {
	return &Adapter{newBox: newBox}
}

func (a *Adapter) Start(ctx context.Context, configJSON string) error {
	if a.newBox == nil {
		a.newBox = newBox
	}
	if a.b != nil {
		return &AdapterError{Code: "START_FAILED", Err: errors.New("adapter already started")}
	}

	b, err := a.newBox(ctx, configJSON)
	if err != nil {
		return err
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

func newBox(ctx context.Context, configJSON string) (boxHandle, error) {
	ctx = include.Context(ctx)

	options, err := sjson.UnmarshalExtendedContext[option.Options](ctx, []byte(configJSON))
	if err != nil {
		return nil, &AdapterError{Code: "CONFIG_FAILED", Err: err}
	}

	b, err := box.New(box.Options{
		Context: ctx,
		Options: options,
	})
	if err != nil {
		return nil, &AdapterError{Code: "START_FAILED", Err: err}
	}

	return b, nil
}

func (a *Adapter) Stop() error {
	if a.b == nil {
		return nil
	}
	if err := a.b.Close(); err != nil {
		return &AdapterError{Code: "STOP_FAILED", Err: err}
	}
	a.b = nil
	return nil
}
