package singboxadapter

import (
	"context"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	sjson "github.com/sagernet/sing/common/json"
)

type Adapter struct {
	b *box.Box
}

func NewAdapter() *Adapter {
	return &Adapter{}
}

func (a *Adapter) Start(ctx context.Context, configJSON string) error {
	ctx = include.Context(ctx)
	
	options, err := sjson.UnmarshalExtendedContext[option.Options](ctx, []byte(configJSON))
	if err != nil {
		return &AdapterError{Code: "CONFIG_FAILED", Err: err}
	}

	b, err := box.New(box.Options{
		Context: ctx,
		Options: options,
	})
	if err != nil {
		return &AdapterError{Code: "START_FAILED", Err: err}
	}

	if err := b.Start(); err != nil {
		return &AdapterError{Code: "START_FAILED", Err: err}
	}

	a.b = b
	return nil
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
