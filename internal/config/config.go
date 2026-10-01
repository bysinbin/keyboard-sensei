package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Rule struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Modifiers   []string `json:"modifiers"` // "cmd", "alt", "ctrl", "shift"
	Keycode     int      `json:"keycode"`
	KeyLabel    string   `json:"key_label"`
	Output      string   `json:"output"`
	Enabled     bool     `json:"enabled"`
	Description string   `json:"description,omitempty"`
}

type Config struct {
	Port      int    `json:"port"`
	AutoStart bool   `json:"auto_start"`
	Rules     []Rule `json:"rules"`
}

var (
	configMu sync.RWMutex
	current  *Config
)

func DefaultConfig() *Config {
	return &Config{
		Port:      5252,
		AutoStart: false,
		Rules: []Rule{
			{
				ID:          "rule-less-than",
				Name:        "Küçüktür İşareti (<)",
				Modifiers:   []string{"cmd"},
				Keycode:     43, // Physical Comma on ANSI -> 'ö' on Turkish Q
				KeyLabel:    "ö (Virgül)",
				Output:      "<",
				Enabled:     true,
				Description: "⌘ + ö ile küçüktür (<) yazar",
			},
			{
				ID:          "rule-greater-than",
				Name:        "Büyüktür İşareti (>)",
				Modifiers:   []string{"cmd"},
				Keycode:     47, // Physical Period on ANSI -> 'ç' on Turkish Q
				KeyLabel:    "ç (Nokta)",
				Output:      ">",
				Enabled:     true,
				Description: "⌘ + ç ile büyüktür (>) yazar",
			},
			{
				ID:          "rule-pipe",
				Name:        "Dikey Çizgi / Pipe (|)",
				Modifiers:   []string{"alt"},
				Keycode:     44, // Physical Slash on ANSI -> '.' on Turkish Q
				KeyLabel:    ". (Bölü / Nokta)",
				Output:      "|",
				Enabled:     true,
				Description: "⌥ + . ile dikey çizgi (|) yazar",
			},
			{
				ID:          "rule-tilde",
				Name:        "Tilde İşareti (~)",
				Modifiers:   []string{"alt"},
				Keycode:     27, // Physical '-' on ANSI -> '*' on Turkish Q
				KeyLabel:    "- (Tire)",
				Output:      "~",
				Enabled:     true,
				Description: "⌥ + - ile tilde (~) yazar",
			},
			{
				ID:          "rule-backtick",
				Name:        "Backtick İşareti (`)",
				Modifiers:   []string{"alt"},
				Keycode:     42, // Physical '\' on ANSI -> ',' on Turkish Q
				KeyLabel:    "\\ (Ters Eğik Çizgi)",
				Output:      "`",
				Enabled:     true,
				Description: "⌥ + \\ ile backtick (`) yazar",
			},
		},
	}
}

func GetConfigFilePath() string {
	home, err := os.UserHomeDir()
	if err == nil {
		dir := filepath.Join(home, ".config", "keyboard-sensei")
		_ = os.MkdirAll(dir, 0755)
		return filepath.Join(dir, "config.json")
	}
	return "config.json"
}

func LoadConfig() (*Config, error) {
	configMu.Lock()
	defer configMu.Unlock()

	filePath := GetConfigFilePath()
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Fallback to local config.json if present
		if _, localErr := os.Stat("config.json"); localErr == nil {
			filePath = "config.json"
		} else {
			current = DefaultConfig()
			_ = saveConfigLocked(filePath, current)
			return current, nil
		}
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		current = DefaultConfig()
		return current, err
	}

	cfg := &Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		current = DefaultConfig()
		return current, err
	}

	if cfg.Port <= 0 {
		cfg.Port = 5252
	}

	current = cfg
	return current, nil
}

func SaveConfig(cfg *Config) error {
	configMu.Lock()
	defer configMu.Unlock()

	current = cfg
	filePath := GetConfigFilePath()
	return saveConfigLocked(filePath, cfg)
}

func saveConfigLocked(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func Get() *Config {
	configMu.RLock()
	defer configMu.RUnlock()
	if current == nil {
		return DefaultConfig()
	}
	return current
}
