package qkboxd

import (
	"context"

	"github.com/zclkkk/qkbox/internal/ipc"
)

func Run(ctx context.Context) error {
	lock, err := AcquireUserLock()
	if err != nil {
		return err
	}
	defer lock.Release()

	listener, err := ipc.Listen()
	if err != nil {
		return err
	}
	defer listener.Close()

	return ipc.NewServer(NewService()).Serve(ctx, listener)
}
