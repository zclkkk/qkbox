package providerruntime

import (
	"context"
	"testing"

	"github.com/zclkkk/qkbox/internal/provideripc"
	"github.com/zclkkk/qkbox/shared/api"
	"github.com/zclkkk/qkbox/shared/model"
)

const directRuntimeConfig = `{"inbounds":[],"outbounds":[{"type":"direct","tag":"direct"}]}`

func TestControllerStartStopOwnsRuntimeLock(t *testing.T) {
	dir := t.TempDir()
	controller := NewController(dir, true, "")
	defer controller.Close()

	start, structured := controller.RuntimeStart(context.Background(), provideripc.RuntimeStartRequest{
		SessionID:          "session-1",
		RuntimeID:          "runtime-1",
		SnapshotID:         "snapshot-1",
		Mode:               api.RuntimeModeMachineNetwork,
		ConfigJSON:         directRuntimeConfig,
		HeartbeatTimeoutMS: 10_000,
	})
	if structured != nil {
		t.Fatalf("start: %v", structured)
	}
	if !start.OwnerState.Owned || start.OwnerState.SessionID != "session-1" {
		t.Fatalf("owner state = %+v", start.OwnerState)
	}
	record, err := loadOwnerRecord(dir)
	if err != nil {
		t.Fatal(err)
	}
	if record == nil || record.RuntimeID != "runtime-1" {
		t.Fatalf("record = %+v", record)
	}

	status, structured := controller.RuntimeGetStatus(context.Background(), provideripc.RuntimeGetStatusRequest{
		SessionID: "session-1",
		RuntimeID: "runtime-1",
	})
	if structured != nil {
		t.Fatalf("status: %v", structured)
	}
	if status.Status.State != model.EngineStateStarted || status.Status.ActiveSnapshotID != "snapshot-1" {
		t.Fatalf("status = %+v", status.Status)
	}

	if _, structured := controller.RuntimeStop(context.Background(), provideripc.RuntimeStopRequest{SessionID: "session-1", RuntimeID: "runtime-1"}); structured != nil {
		t.Fatalf("stop: %v", structured)
	}
	record, err = loadOwnerRecord(dir)
	if err != nil {
		t.Fatal(err)
	}
	if record != nil {
		t.Fatalf("owner record should be deleted, got %+v", record)
	}
}

func TestControllerRejectsRuntimeObservationWithoutOwnerIDs(t *testing.T) {
	controller := NewController(t.TempDir(), true, "")
	defer controller.Close()

	_, structured := controller.RuntimeStart(context.Background(), provideripc.RuntimeStartRequest{
		SessionID:          "session-1",
		RuntimeID:          "runtime-1",
		SnapshotID:         "snapshot-1",
		Mode:               api.RuntimeModeMachineNetwork,
		ConfigJSON:         directRuntimeConfig,
		HeartbeatTimeoutMS: 10_000,
	})
	if structured != nil {
		t.Fatalf("start: %v", structured)
	}

	_, structured = controller.RuntimeGetTraffic(context.Background(), provideripc.RuntimeGetTrafficRequest{})
	if structured == nil || structured.Code != api.ErrorIPCInvalidRequest {
		t.Fatalf("expected invalid request, got %v", structured)
	}

	_, structured = controller.RuntimeGetTraffic(context.Background(), provideripc.RuntimeGetTrafficRequest{
		SessionID: "session-2",
		RuntimeID: "runtime-2",
	})
	if structured == nil || structured.Code != api.ErrorNetworkModeOwnedByAnotherSession {
		t.Fatalf("expected owner conflict, got %v", structured)
	}

	_, structured = controller.RuntimeStop(context.Background(), provideripc.RuntimeStopRequest{})
	if structured == nil || structured.Code != api.ErrorIPCInvalidRequest {
		t.Fatalf("expected invalid stop request, got %v", structured)
	}
}

func TestControllerRejectsAnotherRuntimeOwner(t *testing.T) {
	controller := NewController(t.TempDir(), true, "")
	defer controller.Close()

	_, structured := controller.RuntimeStart(context.Background(), provideripc.RuntimeStartRequest{
		SessionID:          "session-1",
		RuntimeID:          "runtime-1",
		SnapshotID:         "snapshot-1",
		Mode:               api.RuntimeModeMachineNetwork,
		ConfigJSON:         directRuntimeConfig,
		HeartbeatTimeoutMS: 10_000,
	})
	if structured != nil {
		t.Fatalf("start: %v", structured)
	}

	_, structured = controller.RuntimeStart(context.Background(), provideripc.RuntimeStartRequest{
		SessionID:  "session-2",
		RuntimeID:  "runtime-2",
		SnapshotID: "snapshot-2",
		Mode:       api.RuntimeModeMachineNetwork,
		ConfigJSON: directRuntimeConfig,
	})
	if structured == nil || structured.Code != api.ErrorNetworkModeOwnedByAnotherSession {
		t.Fatalf("expected owner conflict, got %v", structured)
	}
}

func TestControllerKeepsAndRepairsStaleOwnerRecord(t *testing.T) {
	dir := t.TempDir()
	if err := saveOwnerRecord(dir, &ownerRecord{
		Owned:      true,
		SessionID:  "old-session",
		RuntimeID:  "old-runtime",
		SnapshotID: "old-snapshot",
		Mode:       api.RuntimeModeMachineNetwork,
	}); err != nil {
		t.Fatal(err)
	}

	controller := NewController(dir, true, "")
	state := controller.OwnerState()
	if state == nil || !state.Stale {
		t.Fatalf("state = %+v, want stale owner", state)
	}

	_, structured := controller.RuntimeStart(context.Background(), provideripc.RuntimeStartRequest{
		SessionID:  "session-1",
		RuntimeID:  "runtime-1",
		SnapshotID: "snapshot-1",
		Mode:       api.RuntimeModeMachineNetwork,
		ConfigJSON: directRuntimeConfig,
	})
	if structured == nil || structured.Code != api.ErrorProviderRuntimeStale {
		t.Fatalf("expected stale error, got %v", structured)
	}

	reply, structured := controller.RunRepairAction(context.Background(), api.RunRepairActionRequest{Action: api.RepairActionClearMachineNetworkOwner})
	if structured != nil {
		t.Fatalf("repair: %v", structured)
	}
	if reply.Outcome != "success" {
		t.Fatalf("repair reply = %+v", reply)
	}
	record, err := loadOwnerRecord(dir)
	if err != nil {
		t.Fatal(err)
	}
	if record != nil {
		t.Fatalf("record should be deleted, got %+v", record)
	}
}
