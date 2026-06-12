package qkboxd

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/zclkkk/qkbox/shared/api"
)

func TestNotifyWindowShowWithoutSession(t *testing.T) {
	svc := &Service{}

	sent, hasSession := svc.NotifyWindowShow()
	if sent || hasSession {
		t.Fatalf("NotifyWindowShow() = sent %v, hasSession %v; want false, false", sent, hasSession)
	}
}

func TestWindowAttachReceivesShowEventAndCleansUp(t *testing.T) {
	svc := &Service{}
	ctx, cancel := context.WithCancel(context.Background())

	events, structured := svc.WindowAttach(ctx, api.WindowAttachRequest{})
	if structured != nil {
		t.Fatal(structured)
	}
	if !svc.HasWindowSession() {
		t.Fatal("expected attached window session")
	}

	sent, hasSession := svc.NotifyWindowShow()
	if !sent || !hasSession {
		t.Fatalf("NotifyWindowShow() = sent %v, hasSession %v; want true, true", sent, hasSession)
	}

	select {
	case event := <-events:
		if event.Event != api.EventWindowShow {
			t.Fatalf("event = %q, want %q", event.Event, api.EventWindowShow)
		}
		data, ok := event.Data.(json.RawMessage)
		if !ok {
			t.Fatalf("event data type = %T, want json.RawMessage", event.Data)
		}
		if string(data) != "{}" {
			t.Fatalf("event data = %s, want {}", data)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for window show event")
	}

	cancel()
	waitForNoWindowSession(t, svc)
}

func TestWindowAttachReplacesPreviousSession(t *testing.T) {
	svc := &Service{}
	firstCtx, firstCancel := context.WithCancel(context.Background())
	defer firstCancel()
	secondCtx, secondCancel := context.WithCancel(context.Background())
	defer secondCancel()

	firstEvents, structured := svc.WindowAttach(firstCtx, api.WindowAttachRequest{})
	if structured != nil {
		t.Fatal(structured)
	}
	secondEvents, structured := svc.WindowAttach(secondCtx, api.WindowAttachRequest{})
	if structured != nil {
		t.Fatal(structured)
	}

	sent, hasSession := svc.NotifyWindowShow()
	if !sent || !hasSession {
		t.Fatalf("NotifyWindowShow() = sent %v, hasSession %v; want true, true", sent, hasSession)
	}

	select {
	case event := <-secondEvents:
		if event.Event != api.EventWindowShow {
			t.Fatalf("event = %q, want %q", event.Event, api.EventWindowShow)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for replacement session event")
	}

	select {
	case event := <-firstEvents:
		t.Fatalf("old session received event: %+v", event)
	default:
	}
}

func TestNotifyWindowShowReportsUnresponsiveSession(t *testing.T) {
	svc := &Service{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events, structured := svc.WindowAttach(ctx, api.WindowAttachRequest{})
	if structured != nil {
		t.Fatal(structured)
	}
	_ = events

	for i := 0; i < 8; i++ {
		sent, hasSession := svc.NotifyWindowShow()
		if !sent || !hasSession {
			t.Fatalf("NotifyWindowShow() at fill %d = sent %v, hasSession %v; want true, true", i, sent, hasSession)
		}
	}
	sent, hasSession := svc.NotifyWindowShow()
	if sent || !hasSession {
		t.Fatalf("NotifyWindowShow() with full channel = sent %v, hasSession %v; want false, true", sent, hasSession)
	}
}

func waitForNoWindowSession(t *testing.T, svc *Service) {
	t.Helper()
	deadline := time.After(time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for window session cleanup")
		case <-ticker.C:
			if !svc.HasWindowSession() {
				return
			}
		}
	}
}
