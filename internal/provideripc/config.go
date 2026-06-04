package provideripc

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	EnvClientConfigPath = "QKBOX_PROVIDER_CLIENT_CONFIG"
	EnvServerConfigPath = "QKBOX_PROVIDER_SERVER_CONFIG"
	EnvEndpoint         = "QKBOX_PROVIDER_ENDPOINT"
)

type ClientConfig struct {
	Endpoint        string `json:"endpoint"`
	Token           string `json:"token"`
	ExpectedVersion string `json:"expected_version"`
}

type ServerConfig struct {
	Endpoint string `json:"endpoint"`
	Token    string `json:"token"`
	Version  string `json:"version"`
}

func ClientConfigPath(stateDir string) string {
	if path := os.Getenv(EnvClientConfigPath); path != "" {
		return path
	}
	return filepath.Join(stateDir, "provider-client.json")
}

func ServerConfigPath(stateDir string) string {
	if path := os.Getenv(EnvServerConfigPath); path != "" {
		return path
	}
	return filepath.Join(stateDir, "provider-server.json")
}

func DefaultStateDir() (string, error) {
	if dir := os.Getenv("QKBOX_STATE_DIR"); dir != "" {
		return dir, os.MkdirAll(dir, 0o700)
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "qkbox")
	return dir, os.MkdirAll(dir, 0o700)
}

func ReadClientConfig(path string) (*ClientConfig, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg ClientConfig
	if err := json.Unmarshal(content, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func ReadServerConfig(path string) (*ServerConfig, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg ServerConfig
	if err := json.Unmarshal(content, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func WriteConfigPair(clientPath, serverPath, endpoint string) (*ClientConfig, *ServerConfig, error) {
	if endpoint == "" {
		endpoint = DefaultEndpoint()
	}
	token, err := randomToken()
	if err != nil {
		return nil, nil, err
	}
	client := &ClientConfig{
		Endpoint:        endpoint,
		Token:           token,
		ExpectedVersion: DefaultProviderVersion,
	}
	server := &ServerConfig{
		Endpoint: endpoint,
		Token:    token,
		Version:  DefaultProviderVersion,
	}
	if err := writeJSON0600(serverPath, server); err != nil {
		return nil, nil, err
	}
	if err := writeJSON0600(clientPath, client); err != nil {
		return nil, nil, err
	}
	return client, server, nil
}

func writeJSON0600(path string, value interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(payload, '\n'), 0o600)
}

func randomToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
