package ipc

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/zclkkk/qkbox/shared/api"
)

func TestSubscriptionWritesAckEventAndCancels(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()

	cancelled := make(chan struct{})
	go func() {
		defer serverConn.Close()
		var req Request
		if err := ReadFrame(serverConn, &req); err != nil {
			t.Errorf("read request: %v", err)
			return
		}
		serveSubscription(serverConn, req, func(ctx context.Context, _ api.EngineSubscribeStatusRequest) (<-chan api.RuntimeEvent, *api.StructuredError) {
			events := make(chan api.RuntimeEvent, 1)
			events <- api.RuntimeEvent{Event: api.EventEngineStatus, Data: api.EngineStatus{State: "STARTED"}}
			go func() {
				<-ctx.Done()
				close(cancelled)
			}()
			return events, nil
		}, context.Background())
	}()

	if err := WriteFrame(clientConn, Request{ID: "sub_1", Method: api.MethodEngineSubscribeStatus, Params: []byte(`{}`)}); err != nil {
		t.Fatalf("write request: %v", err)
	}
	var ack Response
	if err := ReadFrame(clientConn, &ack); err != nil {
		t.Fatalf("read ack: %v", err)
	}
	if ack.ID != "sub_1" || ack.Error != nil {
		t.Fatalf("bad ack: %+v", ack)
	}
	var event EventFrame
	if err := ReadFrame(clientConn, &event); err != nil {
		t.Fatalf("read event: %v", err)
	}
	if event.Event != api.EventEngineStatus || len(event.Data) == 0 {
		t.Fatalf("bad event: %+v", event)
	}
	clientConn.Close()

	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("subscription context was not cancelled after client close")
	}
}
