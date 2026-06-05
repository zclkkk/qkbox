package qkboxd

import (
	"context"
	"fmt"

	qkboxcrypto "github.com/zclkkk/qkbox/internal/crypto"
	"github.com/zclkkk/qkbox/internal/ipc"
	"github.com/zclkkk/qkbox/internal/persistence"
	"github.com/zclkkk/qkbox/platform/capability"
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

	proxy := capability.NewSystemProxyProvider()
	repairStaleProxy(db, proxy)
	privileged := capability.NewPrivilegedProvider(stateDir)
	extension := capability.NewNetworkExtensionRuntime(stateDir)

	listener, err := ipc.Listen()
	if err != nil {
		return err
	}
	defer listener.Close()

	service := NewServiceWithNetworkExtension(ctx, db, key, proxy, privileged, extension)
	defer service.Close()

	return ipc.NewServer(service).Serve(ctx, listener)
}

func repairStaleProxy(db *persistence.DB, proxy capability.SystemProxyProvider) {
	if proxy == nil {
		return
	}
	avail := proxy.Availability()
	if !avail.Available || !avail.Supported {
		return
	}
	record, err := loadProxyOwner(db)
	if err != nil || record == nil || !record.QKBoxOwned {
		return
	}
	state, err := proxy.CurrentState()
	if err != nil {
		fmt.Printf("warning: stale proxy state check failed, will retry next startup: %v\n", err)
		return
	}
	if !proxyOwnerMatches(state, record) {
		_ = deleteProxyOwner(db)
		return
	}
	if err := proxy.Restore(record.Snapshot); err != nil {
		fmt.Printf("warning: stale proxy restore failed, will retry next startup: %v\n", err)
		return
	}
	_ = deleteProxyOwner(db)
	fmt.Println("warning: restored stale system proxy from previous session")
}
