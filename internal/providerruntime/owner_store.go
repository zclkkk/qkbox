package providerruntime

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/zclkkk/qkbox/shared/api"
)

const ownerRecordFile = "provider-runtime-owner.json"

type ownerRecord struct {
	Owned           bool     `json:"owned"`
	Stale           bool     `json:"stale,omitempty"`
	SessionID       string   `json:"session_id,omitempty"`
	RuntimeID       string   `json:"runtime_id,omitempty"`
	SnapshotID      string   `json:"snapshot_id,omitempty"`
	Mode            string   `json:"mode,omitempty"`
	StartedAt       int64    `json:"started_at,omitempty"`
	LastHeartbeatAt int64    `json:"last_heartbeat_at,omitempty"`
	Reason          string   `json:"reason,omitempty"`
	RepairActions   []string `json:"repair_actions,omitempty"`
}

func ownerRecordPath(stateDir string) string {
	return filepath.Join(stateDir, ownerRecordFile)
}

func loadOwnerRecord(stateDir string) (*ownerRecord, error) {
	payload, err := os.ReadFile(ownerRecordPath(stateDir))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var record ownerRecord
	if err := json.Unmarshal(payload, &record); err != nil {
		return nil, err
	}
	if !record.Owned {
		return nil, nil
	}
	return &record, nil
}

func saveOwnerRecord(stateDir string, record *ownerRecord) error {
	payload, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(ownerRecordPath(stateDir), append(payload, '\n'), 0o600)
}

func deleteOwnerRecord(stateDir string) error {
	err := os.Remove(ownerRecordPath(stateDir))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func providerOwnerState(record *ownerRecord) *api.ProviderOwnerState {
	if record == nil || !record.Owned {
		return nil
	}
	return &api.ProviderOwnerState{
		Owned:           record.Owned,
		Stale:           record.Stale,
		SessionID:       record.SessionID,
		RuntimeID:       record.RuntimeID,
		SnapshotID:      record.SnapshotID,
		Mode:            record.Mode,
		StartedAt:       record.StartedAt,
		LastHeartbeatAt: record.LastHeartbeatAt,
		Reason:          record.Reason,
		RepairActions:   append([]string(nil), record.RepairActions...),
	}
}
