package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Rule struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Modifiers    []string `json:"modifiers"` // "cmd", "alt", "ctrl", "shift"
	Keycode      int      `json:"keycode"`
	KeyLabel     string   `json:"key_label"`
	Output       string   `json:"output"`
	Enabled      bool     `json:"enabled"`
	Description  string   `json:"description,omitempty"`
	TargetApps   []string `json:"target_apps,omitempty"`   // Yalnızca bu uygulamalarda aktif
	ExcludedApps []string `json:"excluded_apps,omitempty"` // Bu uygulamalarda devre dışı
	IsSnippet    bool     `json:"is_snippet,omitempty"`    // Çok satırlı veya dinamik metin genişletme
}

type Profile struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Icon         string   `json:"icon,omitempty"`
	Rules        []Rule   `json:"rules"`
	ExcludedApps []string `json:"excluded_apps,omitempty"`
}

type KeyboardDevice struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	VendorID   int    `json:"vendor_id"`
	ProductID  int    `json:"product_id"`
	Transport  string `json:"transport"`
	IsInternal bool   `json:"is_internal"`
	Enabled    bool   `json:"enabled"`
}

type HyperKeyConfig struct {
	Enabled       bool     `json:"enabled"`
	SourceKeycode int      `json:"source_keycode"` // Varsayılan: 57 (Caps Lock)
	Modifiers     []string `json:"modifiers"`      // Varsayılan: ["cmd", "alt", "ctrl", "shift"]
	TapAction     string   `json:"tap_action"`     // "escape", "none", "caps_lock"
}

type SequenceRule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Keycode     int    `json:"keycode"`
	KeyLabel    string `json:"key_label"`
	TapCount    int    `json:"tap_count"`
	Output      string `json:"output"`
	TimeoutMs   int    `json:"timeout_ms"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description,omitempty"`
}

type Config struct {
	Port            int              `json:"port"`
	AutoStart       bool             `json:"auto_start"`
	ShowTrayIcon    bool             `json:"show_tray_icon"`
	ActiveProfileID string           `json:"active_profile_id"`
	Profiles        []Profile        `json:"profiles"`
	Rules           []Rule           `json:"rules"` // Geriye dönük uyumluluk ve hızlı erişim için
	ExcludedApps    []string         `json:"excluded_apps,omitempty"`
	Devices         []KeyboardDevice `json:"devices"`
	HyperKey        HyperKeyConfig   `json:"hyper_key"`
	SequenceRules   []SequenceRule   `json:"sequence_rules"`
	EnableSequences bool             `json:"enable_sequences"`
}

var (
	configMu sync.RWMutex
	current  *Config
)

// DefaultRules returns standard ANSI Turkish Q missing key conversions
func DefaultRules() []Rule {
	return []Rule{
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
	}
}

// DeveloperRules returns extended shortcuts for coders
func DeveloperRules() []Rule {
	base := DefaultRules()
	devRules := []Rule{
		{
			ID:          "rule-double-less",
			Name:        "Çift Küçüktür (<<)",
			Modifiers:   []string{"cmd", "shift"},
			Keycode:     43,
			KeyLabel:    "ö (Virgül)",
			Output:      "<<",
			Enabled:     true,
			Description: "⌘ + ⇧ + ö ile << yazar",
		},
		{
			ID:          "rule-double-greater",
			Name:        "Çift Büyüktür (>>)",
			Modifiers:   []string{"cmd", "shift"},
			Keycode:     47,
			KeyLabel:    "ç (Nokta)",
			Output:      ">>",
			Enabled:     true,
			Description: "⌘ + ⇧ + ç ile >> yazar",
		},
		{
			ID:          "rule-arrow-fn",
			Name:        "Arrow Fonksiyonu (=>)",
			Modifiers:   []string{"alt"},
			Keycode:     41, // Physical Semicolon -> 'ş' on Turkish Q
			KeyLabel:    "ş (Noktalı Virgül)",
			Output:      "=>",
			Enabled:     true,
			Description: "⌥ + ş ile => yazar",
		},
		{
			ID:          "rule-date-stamp",
			Name:        "Tarih Damgası ({date})",
			Modifiers:   []string{"cmd", "alt"},
			Keycode:     2, // Key D
			KeyLabel:    "D",
			Output:      "{date}",
			Enabled:     true,
			IsSnippet:   true,
			Description: "⌘ + ⌥ + D ile bugünün tarihini yazar",
		},
	}
	return append(base, devRules...)
}

// RDPRules returns rules optimized for Microsoft Remote Desktop sessions
func RDPRules() []Rule {
	base := DefaultRules()
	rdpExtra := []Rule{
		{
			ID:          "rule-rdp-backslash",
			Name:        "RDP Ters Slash (\\)",
			Modifiers:   []string{"cmd", "alt"},
			Keycode:     44, // Slash
			KeyLabel:    ". (Bölü / Nokta)",
			Output:      "\\",
			Enabled:     true,
			Description: "⌘ + ⌥ + . ile Windows için ters slash yazar",
		},
	}
	return append(base, rdpExtra...)
}

// BuiltinPresets returns all pre-configured profiles
func BuiltinPresets() []Profile {
	return []Profile{
		{
			ID:          "profile-ansi-tr",
			Name:        "ANSI ➔ Türkçe Q (Varsayılan)",
			Description: "İngilizce fiziksel klavyede eksik olan <, >, |, ~, ` karakterlerini sağlar.",
			Icon:        "🥋",
			Rules:       DefaultRules(),
		},
		{
			ID:          "profile-dev",
			Name:        "Geliştirici / Kodlama",
			Description: "Yazılımcılar için <<, >>, => ve tarih damgası gibi ek pratik kısayollar.",
			Icon:        "💻",
			Rules:       DeveloperRules(),
		},
		{
			ID:          "profile-rdp",
			Name:        "Uzak Masaüstü (RDP / Windows)",
			Description: "Windows Uzak Masaüstü oturumları için optimize edilmiş tuş dizilimi.",
			Icon:        "🖥️",
			Rules:       RDPRules(),
		},
	}
}

// DefaultSequenceRules returns double-tap sequences (e.g. öö -> <, çç -> >)
func DefaultSequenceRules() []SequenceRule {
	return []SequenceRule{
		{
			ID:          "seq-less-than",
			Name:        "Çift ö (öö) ➔ <",
			Keycode:     43, // ö
			KeyLabel:    "ö",
			TapCount:    2,
			Output:      "<",
			TimeoutMs:   280,
			Enabled:     true,
			Description: "Hızlıca 2 kez ö tuşuna basınca < yazar",
		},
		{
			ID:          "seq-greater-than",
			Name:        "Çift ç (çç) ➔ >",
			Keycode:     47, // ç
			KeyLabel:    "ç",
			TapCount:    2,
			Output:      ">",
			TimeoutMs:   280,
			Enabled:     true,
			Description: "Hızlıca 2 kez ç tuşuna basınca > yazar",
		},
		{
			ID:          "seq-pipe",
			Name:        "Çift nokta (..) ➔ |",
			Keycode:     44, // .
			KeyLabel:    ".",
			TapCount:    2,
			Output:      "|",
			TimeoutMs:   280,
			Enabled:     true,
			Description: "Hızlıca 2 kez . tuşuna basınca | yazar",
		},
		{
			ID:          "seq-tilde",
			Name:        "Çift tire (--) ➔ ~",
			Keycode:     27, // -
			KeyLabel:    "-",
			TapCount:    2,
			Output:      "~",
			TimeoutMs:   280,
			Enabled:     true,
			Description: "Hızlıca 2 kez - tuşuna basınca ~ yazar",
		},
	}
}

func DefaultHyperKey() HyperKeyConfig {
	return HyperKeyConfig{
		Enabled:       false,
		SourceKeycode: 57, // Caps Lock
		Modifiers:     []string{"cmd", "alt", "ctrl", "shift"},
		TapAction:     "escape",
	}
}

func DefaultConfig() *Config {
	presets := BuiltinPresets()
	return &Config{
		Port:            5252,
		AutoStart:       false,
		ShowTrayIcon:    true,
		ActiveProfileID: "profile-ansi-tr",
		Profiles:        presets,
		Rules:           presets[0].Rules,
		ExcludedApps:    []string{},
		Devices:         []KeyboardDevice{},
		HyperKey:        DefaultHyperKey(),
		SequenceRules:   DefaultSequenceRules(),
		EnableSequences: true,
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

	// Auto-upgrade configuration if profiles are missing
	if len(cfg.Profiles) == 0 {
		presets := BuiltinPresets()
		if len(cfg.Rules) > 0 {
			// Migrate existing custom rules to user's default profile
			presets[0].Rules = cfg.Rules
		}
		cfg.Profiles = presets
		cfg.ActiveProfileID = presets[0].ID
	}

	if cfg.ActiveProfileID == "" {
		cfg.ActiveProfileID = cfg.Profiles[0].ID
	}

	if len(cfg.SequenceRules) == 0 {
		cfg.SequenceRules = DefaultSequenceRules()
		cfg.EnableSequences = true
	}

	if cfg.HyperKey.SourceKeycode == 0 {
		cfg.HyperKey = DefaultHyperKey()
	}

	// Sync active rules
	cfg.SyncActiveRules()
	current = cfg
	return current, nil
}

func SaveConfig(cfg *Config) error {
	configMu.Lock()
	defer configMu.Unlock()

	cfg.SyncActiveRules()
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

// SyncActiveRules updates cfg.Rules to match the currently active profile
func (c *Config) SyncActiveRules() {
	if len(c.Profiles) == 0 {
		if len(c.Rules) == 0 {
			c.Rules = DefaultRules()
		}
		return
	}

	for _, p := range c.Profiles {
		if p.ID == c.ActiveProfileID {
			c.Rules = make([]Rule, len(p.Rules))
			copy(c.Rules, p.Rules)
			return
		}
	}

	// Fallback to first profile if active not found
	c.ActiveProfileID = c.Profiles[0].ID
	c.Rules = make([]Rule, len(c.Profiles[0].Rules))
	copy(c.Rules, c.Profiles[0].Rules)
}

func (c *Config) GetActiveProfile() *Profile {
	for i := range c.Profiles {
		if c.Profiles[i].ID == c.ActiveProfileID {
			return &c.Profiles[i]
		}
	}
	if len(c.Profiles) > 0 {
		return &c.Profiles[0]
	}
	return nil
}

func (c *Config) SetActiveProfile(id string) error {
	for _, p := range c.Profiles {
		if p.ID == id {
			c.ActiveProfileID = id
			c.SyncActiveRules()
			return nil
		}
	}
	return fmt.Errorf("profil bulunamadı: %s", id)
}

func (c *Config) AddProfile(p Profile) error {
	if p.ID == "" {
		p.ID = fmt.Sprintf("profile-%d", time.Now().UnixNano())
	}
	for _, existing := range c.Profiles {
		if existing.ID == p.ID {
			return errors.New("bu profil ID'si zaten mevcut")
		}
	}
	c.Profiles = append(c.Profiles, p)
	return nil
}

func (c *Config) UpdateProfile(p Profile) error {
	for i := range c.Profiles {
		if c.Profiles[i].ID == p.ID {
			c.Profiles[i] = p
			if c.ActiveProfileID == p.ID {
				c.SyncActiveRules()
			}
			return nil
		}
	}
	return fmt.Errorf("güncellenecek profil bulunamadı: %s", p.ID)
}

func (c *Config) DeleteProfile(id string) error {
	if len(c.Profiles) <= 1 {
		return errors.New("en az bir profil kalmalıdır, son profil silinemez")
	}
	for i := range c.Profiles {
		if c.Profiles[i].ID == id {
			c.Profiles = append(c.Profiles[:i], c.Profiles[i+1:]...)
			if c.ActiveProfileID == id {
				c.ActiveProfileID = c.Profiles[0].ID
				c.SyncActiveRules()
			}
			return nil
		}
	}
	return fmt.Errorf("silinecek profil bulunamadı: %s", id)
}

// ExportJSON exports configuration as formatted JSON
func (c *Config) ExportJSON() ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

// ImportJSON imports profiles and settings from JSON data
func (c *Config) ImportJSON(data []byte) error {
	var imported Config
	if err := json.Unmarshal(data, &imported); err != nil {
		return err
	}
	if len(imported.Profiles) > 0 {
		c.Profiles = imported.Profiles
		c.ActiveProfileID = imported.ActiveProfileID
	}
	if len(imported.Rules) > 0 && len(c.Profiles) == 0 {
		c.Rules = imported.Rules
	}
	if imported.Port > 0 {
		c.Port = imported.Port
	}
	if len(imported.ExcludedApps) > 0 {
		c.ExcludedApps = imported.ExcludedApps
	}
	if len(imported.Devices) > 0 {
		c.Devices = imported.Devices
	}
	if imported.HyperKey.SourceKeycode > 0 {
		c.HyperKey = imported.HyperKey
	}
	if len(imported.SequenceRules) > 0 {
		c.SequenceRules = imported.SequenceRules
		c.EnableSequences = imported.EnableSequences
	}
	c.SyncActiveRules()
	return nil
}

