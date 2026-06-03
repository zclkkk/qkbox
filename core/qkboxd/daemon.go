package qkboxd

import (
	"context"

	qkboxcrypto "github.com/zclkkk/qkbox/internal/crypto"
	"github.com/zclkkk/qkbox/internal/ipc"
	"github.com/zclkkk/qkbox/internal/persistence"
)

func Run(ctx context.Context) error {
	lock, err := AcquireUserLock()
	if err != nil {
		return err
	}
	defer lock.Release()

	stateDir, err := userStateDir()
	if err != nil {
		return err
	}

	db, err := persistence.Open(stateDir)
	if err != nil {
		return err
	}
	defer db.Close()

	keyStore := qkboxcrypto.NewFileKeyStore(stateDir)
	key, err := keyStore.GetOrCreateKey()
	if err != nil {
		return err
	}

	listener, err := ipc.Listen()
	if err != nil {
		return err
	}
	defer listener.Close()

	return ipc.NewServer(NewService(db, key)).Serve(ctx, listener)
}
