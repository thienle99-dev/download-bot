package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
)

const (
	KeyAIBaseURL      = "ai.base_url"
	KeyAIAPIKey       = "ai.api_key"
	KeyAIModel        = "ai.model"
	KeyAISystemPrompt = "ai.system_prompt"
	KeyAIEnabled      = "ai.enabled"
)

// AIConfig holds the full configuration for the OpenAI-compatible AI chatbot.
type AIConfig struct {
	BaseURL      string `json:"base_url"`
	APIKey       string `json:"api_key"`
	Model        string `json:"model"`
	SystemPrompt string `json:"system_prompt"`
	Enabled      bool   `json:"enabled"`
}

// GetSetting retrieves a single setting value by key.
// Returns ("", nil) if key does not exist.
func (db *DB) GetSetting(key string) (string, error) {
	var value string
	err := db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return value, nil
}

// SetSetting upserts a setting key-value pair.
func (db *DB) SetSetting(key, value string) error {
	_, err := db.Exec(
		`INSERT INTO settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP`,
		key, value,
	)
	return err
}

// GetAIConfig reads all AI-related settings and returns a populated AIConfig struct.
// Falls back to empty strings (disabled) if not yet configured.
func (db *DB) GetAIConfig() (AIConfig, error) {
	cfg := AIConfig{
		SystemPrompt: "Bạn là một trợ lý AI thông minh và hữu ích. Hãy trả lời bằng ngôn ngữ mà người dùng đang dùng.",
		Enabled:      false,
	}

	keys := map[string]*string{
		KeyAIBaseURL:      &cfg.BaseURL,
		KeyAIAPIKey:       &cfg.APIKey,
		KeyAIModel:        &cfg.Model,
		KeyAISystemPrompt: &cfg.SystemPrompt,
	}

	for key, dest := range keys {
		val, err := db.GetSetting(key)
		if err != nil {
			return cfg, err
		}
		if val != "" {
			*dest = val
		}
	}

	enabledStr, err := db.GetSetting(KeyAIEnabled)
	if err != nil {
		return cfg, err
	}
	cfg.Enabled = enabledStr == "true"

	return cfg, nil
}

// SetAIConfig saves all AI config fields into the settings table.
func (db *DB) SetAIConfig(cfg AIConfig) error {
	enabledStr := "false"
	if cfg.Enabled {
		enabledStr = "true"
	}

	pairs := map[string]string{
		KeyAIBaseURL:      cfg.BaseURL,
		KeyAIAPIKey:       cfg.APIKey,
		KeyAIModel:        cfg.Model,
		KeyAISystemPrompt: cfg.SystemPrompt,
		KeyAIEnabled:      enabledStr,
	}

	for key, val := range pairs {
		if err := db.SetSetting(key, val); err != nil {
			return err
		}
	}
	return nil
}

// MarshalAIConfig encodes an AIConfig to JSON, masking the API key.
func MarshalAIConfig(cfg AIConfig, maskKey bool) ([]byte, error) {
	type safe struct {
		BaseURL      string `json:"base_url"`
		APIKey       string `json:"api_key"`
		Model        string `json:"model"`
		SystemPrompt string `json:"system_prompt"`
		Enabled      bool   `json:"enabled"`
	}
	s := safe{
		BaseURL:      cfg.BaseURL,
		APIKey:       cfg.APIKey,
		Model:        cfg.Model,
		SystemPrompt: cfg.SystemPrompt,
		Enabled:      cfg.Enabled,
	}
	if maskKey && cfg.APIKey != "" {
		s.APIKey = "***"
	}
	return json.Marshal(s)
}
