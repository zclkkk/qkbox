package qkboxd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAcquireUserLockRejectsConcurrentOwner(t *testing.T) {
	t.Setenv("QKBOX_STATE_DIR", t.TempDir())

	first, err := AcquireUserLock()
	if err != nil {
		t.Fatalf("first lock: %v", err)
	}
	defer first.Release()

	second, err := AcquireUserLock()
	if err == nil {
		second.Release()
		t.Fatal("expected second lock to fail")
	}
}

func TestAcquireUserLockIgnoresStaleFile(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("QKBOX_STATE_DIR", stateDir)
	if err := os.WriteFile(filepath.Join(stateDir, "qkboxd.lock"), []byte("stale\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	lock, err := AcquireUserLock()
	if err != nil {
		t.Fatalf("lock with stale file: %v", err)
	}
	lock.Release()
}
