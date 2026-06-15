package qkboxd

import (
	"context"
	"testing"
	"time"

	"github.com/zclkkk/qkbox/shared/api"
	"github.com/zclkkk/qkbox/shared/model"
)

func TestRuntimeEventHubReplaysLogRing(t *testing.T) {
	hub := NewRuntimeEventHub()
	for i := 0; i < runtimeLogRingLimit+1; i++ {
		hub.PublishRuntimeLog("test", "info", "message")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := hub.SubscribeLogs(ctx)
	var first api.RuntimeLogEntry
	for i := 0; i < runtimeLogRingLimit; i++ {
		select {
		case event := <-events:
			entry := event.Data.(api.RuntimeLogEntry)
			if i == 0 {
				first = entry
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for log replay")
		}
	}
	if first.Seq != 2 {
		t.Fatalf("first replayed seq = %d, want 2", first.Seq)
	}
}

func TestEngineControllerPublishesStatusEvents(t *testing.T) {
	hub := NewRuntimeEventHub()
	ctrl := NewEngineController(context.Background(), hub)
	fake := &fakeRuntimeOwner{}
	ctrl.runtimeOwnerFactory = func(RuntimeStartTarget) RuntimeOwner {
		return fake
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := hub.SubscribeStatus(ctx)

	expectStatus(t, events, model.EngineStateIdle)
	err := ctrl.Start(RuntimeStartTarget{ProfileID: "profile", ConfigJSON: "{}"})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	expectStatus(t, events, model.EngineStateStarting)
	expectStatus(t, events, model.EngineStateStarting)
	expectStatus(t, events, model.EngineStateStarted)
}

func expectStatus(t *testing.T, events <-chan api.RuntimeEvent, state model.EngineState) {
	t.Helper()
	select {
	case event := <-events:
		status := event.Data.(api.EngineStatus)
		if status.State != state {
			t.Fatalf("state = %s, want %s", status.State, state)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", state)
	}
}
