package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Settings holds user preferences stored locally.
type Settings struct {
	LeewayDaysPerWeek int `json:"leeway_days_per_week"`
}

func filePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".pomo")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "settings.json"), nil
}

// Load reads settings from disk, returning defaults if the file doesn't exist.
func Load() (Settings, error) {
	path, err := filePath()
	if err != nil {
		return Settings{}, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Settings{LeewayDaysPerWeek: 0}, nil
	}
	if err != nil {
		return Settings{}, fmt.Errorf("reading settings: %w", err)
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return Settings{}, fmt.Errorf("decoding settings: %w", err)
	}
	return s, nil
}

// Save writes settings to disk.
func Save(s Settings) error {
	path, err := filePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
