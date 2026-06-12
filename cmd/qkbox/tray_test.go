package main

import (
	"context"
	"testing"
	"time"

	"github.com/zclkkk/qkbox/core/qkboxd"
	"github.com/zclkkk/qkbox/shared/api"
)

func TestWindowLaunchPendingIsExclusive(t *testing.T) {
	resetWindowLaunchPendingForTest()

	if !beginWindowLaunch() {
		t.Fatal("first beginWindowLaunch returned false")
	}
	if beginWindowLaunch() {
		t.Fatal("second beginWindowLaunch returned true while launch is pending")
	}

	endWindowLaunch()
	if !beginWindowLaunch() {
		t.Fatal("beginWindowLaunch returned false after endWindowLaunch")
	}
	endWindowLaunch()
}

func TestClearWindowLaunchWhenAttached(t *testing.T) {
	resetWindowLaunchPendingForTest()
	if !beginWindowLaunch() {
		t.Fatal("beginWindowLaunch returned false")
	}

	inst := &qkboxd.Instance{Service: &qkboxd.Service{}}
	done := make(chan struct{})
	go func() {
		clearWindowLaunchWhenAttached(inst)
		close(done)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if _, structured := inst.Service.WindowAttach(ctx, api.WindowAttachRequest{}); structured != nil {
		t.Fatal(structured)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for launch pending cleanup")
	}
	if isWindowLaunchPendingForTest() {
		t.Fatal("window launch still pending after attach")
	}
}

func resetWindowLaunchPendingForTest() {
	windowLaunchMu.Lock()
	windowLaunchPending = false
	windowLaunchMu.Unlock()
}

func isWindowLaunchPendingForTest() bool {
	windowLaunchMu.Lock()
	defer windowLaunchMu.Unlock()
	return windowLaunchPending
}
