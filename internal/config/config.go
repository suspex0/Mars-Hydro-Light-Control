package config

import (
	"encoding/json"
	"os"

	"lightcontrol/internal/timer"
)

// MHAccountConfig holds account credentials.
type MHAccountConfig struct {
	Email    string `json:"Email"`
	Password string `json:"Password"`
	Wifiname string `json:"Wifiname"`
	Timezone string `json:"Timezone"`
	Language string `json:"Language"`
}

// LoadMHAccountConfig loads MH account configuration from the given file.
func LoadMHAccountConfig(path string) (*MHAccountConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var cfg MHAccountConfig
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// LoadLightTimerConfig loads LightTimer configuration from the given file.
func LoadLightTimerConfig(path string) (*timer.LightTimer, error) {
	var lt timer.LightTimer
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&lt); err != nil {
		return nil, err
	}
	// Basic validation: EndHour must be greater than StartHour+PlateauHour.
	if lt.EndHour <= lt.StartHour+lt.PlateauHour {
		return nil, 	// ensure configuration is valid.
			&os.PathError{Op: "Load", Path: path, Err: os.ErrInvalid}
	}
	return &lt, nil
}
