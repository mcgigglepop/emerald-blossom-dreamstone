package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds application configuration
type Config struct {
	AWSRegion         string `json:"aws_region"`
	TableName         string `json:"table_name"`
	UserID            string `json:"user_id"`
	VaultPath         string `json:"vault_path"`
	SessionSecretName string `json:"session_secret_name,omitempty"` // AWS Secrets Manager secret name for session key
	ConfigPath        string `json:"-"`                             // Not stored, just for reference
}

// GetSessionPath returns the path to the session file
func (c *Config) GetSessionPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".vaultctl", "session.json")
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	return &Config{
		AWSRegion:         "us-west-2",
		TableName:         "vaultctl_vaults",
		UserID:            "default",
		VaultPath:         filepath.Join(homeDir, ".vaultctl", "vault.db"),
		SessionSecretName: "vaultctl/session-key",
		ConfigPath:        filepath.Join(homeDir, ".vaultctl", "config.json"),
	}
}

// LoadConfig loads configuration from file
func LoadConfig() (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(cfg.ConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	cfg.ConfigPath = filepath.Join(filepath.Dir(cfg.ConfigPath), "config.json")
	return cfg, nil
}

// SaveConfig saves configuration to file
func (c *Config) SaveConfig() error {
	dir := filepath.Dir(c.ConfigPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(c.ConfigPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}
