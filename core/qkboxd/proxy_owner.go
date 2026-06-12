package qkboxd

import (
	"encoding/json"

	"github.com/zclkkk/qkbox/internal/persistence"
	"github.com/zclkkk/qkbox/platform/capability"
)

const proxyOwnerSettingsKey = "proxy_owner"

type proxyOwnerRecord struct {
	QKBoxOwned bool                            `json:"qkbox_owned"`
	Snapshot   *capability.SystemProxySnapshot `json:"snapshot"`
	ProxyAddr  string                          `json:"proxy_addr"`
	ProxyPort  int                             `json:"proxy_port"`
	EnabledAt  int64                           `json:"enabled_at"`
}

func saveProxyOwner(db *persistence.DB, record *proxyOwnerRecord) error {
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return db.SetSetting(proxyOwnerSettingsKey, data)
}

func loadProxyOwner(db *persistence.DB) (*proxyOwnerRecord, error) {
	data, err := db.GetSetting(proxyOwnerSettingsKey)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	var record proxyOwnerRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func deleteProxyOwner(db *persistence.DB) error {
	return db.DeleteSetting(proxyOwnerSettingsKey)
}

func proxyOwnerMatches(state capability.SystemProxyCurrentState, record *proxyOwnerRecord) bool {
	return state.Enabled && state.Addr == record.ProxyAddr && state.Port == record.ProxyPort
}
