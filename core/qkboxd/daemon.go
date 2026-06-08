package qkboxd

import (
	"context"
	"fmt"
	"net"

	qkboxcrypto "github.com/zclkkk/qkbox/internal/crypto"
	"github.com/zclkkk/qkbox/internal/ipc"
	"github.com/zclkkk/qkbox/internal/persistence"
	"github.com/zclkkk/qkbox/platform/capability"
)

// StartOpts configures the daemon instance. Phase 0A: empty.
type StartOpts struct{}

// Instance is a running daemon. Use Wait() to block until shutdown, Close() to request it.
type Instance struct {
	Service *Service

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
	err    error

	// resources to clean up
	lock     *UserLock
	db       *persistence.DB
	listener net.Listener
}

// Start initialises and runs the daemon in the background.
// The returned Instance exposes the Service handle and lifecycle controls.
func Start(parent context.Context, _ StartOpts) (*Instance, error) {
	ctx, cancel := context.WithCancel(parent)

	lock, err := AcquireUserLock()
	if err != nil {
		cancel()
		return nil, err
	}

	stateDir, err := userStateDir()
	if err != nil {
		lock.Release()
		cancel()
		return nil, err
	}

	db, err := persistence.Open(stateDir)
	if err != nil {
		lock.Release()
		cancel()
		return nil, err
	}

	keyStore := qkboxcrypto.NewFileKeyStore(stateDir)
	key, err := keyStore.GetOrCreateKey()
	if err != nil {
		db.Close()
		lock.Release()
		cancel()
		return nil, err
	}

	proxy := capability.NewSystemProxyProvider()
	repairStaleProxy(db, proxy)
	privileged := capability.NewPrivilegedProvider(stateDir)
	extension := capability.NewNetworkExtensionRuntime(stateDir)

	listener, err := ipc.Listen()
	if err != nil {
		db.Close()
		lock.Release()
		cancel()
		return nil, err
	}

	service := NewServiceWithNetworkExtension(ctx, db, key, proxy, privileged, extension)

	inst := &Instance{
		Service:  service,
		ctx:      ctx,
		cancel:   cancel,
		done:     make(chan struct{}),
		lock:     lock,
		db:       db,
		listener: listener,
	}

	// Run the IPC server in the background.
	go func() {
		defer close(inst.done)
		inst.err = ipc.NewServer(service).Serve(ctx, listener)
		// Unified cleanup: all shutdown paths (signal, Close, fatal) run this.
		service.Close()
		listener.Close()
		db.Close()
		lock.Release()
	}()

	return inst, nil
}

// Wait blocks until the daemon exits and returns the first error (if any).
func (inst *Instance) Wait() error {
	<-inst.done
	return inst.err
}

// Close requests a graceful shutdown. Non-blocking, idempotent.
func (inst *Instance) Close() {
	inst.cancel()
}

// Shutdown requests a graceful shutdown and waits for cleanup to complete.
// Use this for user-initiated quit where all resources must be released before exit.
func (inst *Instance) Shutdown(ctx context.Context) error {
	inst.cancel()
	select {
	case <-inst.done:
		return inst.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// EngineState returns the current engine state string (IDLE, STARTING, STARTED, etc.).
// Safe to call from any goroutine.
func (inst *Instance) EngineState() string {
	return inst.Service.EngineStateString()
}

// Run is a backward-compatible blocking wrapper around Start + Wait.
func Run(ctx context.Context) error {
	inst, err := Start(ctx, StartOpts{})
	if err != nil {
		return err
	}
	return inst.Wait()
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
