package config

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Port != 5252 {
		t.Errorf("expected port 5252, got %d", cfg.Port)
	}
	if len(cfg.Profiles) < 3 {
		t.Errorf("expected at least 3 builtin profiles, got %d", len(cfg.Profiles))
	}
	if cfg.ActiveProfileID != "profile-ansi-tr" {
		t.Errorf("expected active profile 'profile-ansi-tr', got %s", cfg.ActiveProfileID)
	}
	if len(cfg.Rules) != 5 {
		t.Errorf("expected 5 default ANSI rules, got %d", len(cfg.Rules))
	}

	// Verify key outputs
	expectedOutputs := map[string]bool{"<": false, ">": false, "|": false, "~": false, "`": false}
	for _, r := range cfg.Rules {
		if _, ok := expectedOutputs[r.Output]; ok {
			expectedOutputs[r.Output] = true
		}
	}
	for out, found := range expectedOutputs {
		if !found {
			t.Errorf("expected default rule for output %s not found", out)
		}
	}
}

func TestProfileManagement(t *testing.T) {
	cfg := DefaultConfig()

	// Switch profile to dev
	err := cfg.SetActiveProfile("profile-dev")
	if err != nil {
		t.Fatalf("failed to set active profile: %v", err)
	}
	if cfg.ActiveProfileID != "profile-dev" {
		t.Errorf("expected active profile 'profile-dev', got %s", cfg.ActiveProfileID)
	}
	if len(cfg.Rules) <= 5 {
		t.Errorf("expected dev profile to have more than 5 rules, got %d", len(cfg.Rules))
	}

	// Add new custom profile
	newP := Profile{
		ID:          "profile-custom-test",
		Name:        "Custom Test Profile",
		Description: "A profile created in unit test",
		Icon:        "🧪",
		Rules: []Rule{
			{
				ID:        "test-rule-1",
				Name:      "Test Rule",
				Modifiers: []string{"cmd"},
				Keycode:   43,
				Output:    "TEST",
				Enabled:   true,
			},
		},
	}
	err = cfg.AddProfile(newP)
	if err != nil {
		t.Fatalf("failed to add profile: %v", err)
	}

	// Switch to new profile
	err = cfg.SetActiveProfile(newP.ID)
	if err != nil {
		t.Fatalf("failed to activate custom profile: %v", err)
	}
	if len(cfg.Rules) != 1 || cfg.Rules[0].Output != "TEST" {
		t.Errorf("active rules not synchronized correctly for custom profile")
	}

	// Delete profile
	err = cfg.DeleteProfile(newP.ID)
	if err != nil {
		t.Fatalf("failed to delete profile: %v", err)
	}
	if cfg.ActiveProfileID == newP.ID {
		t.Errorf("active profile should fallback after deletion")
	}
}

func TestJSONExportImport(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Port = 8080
	cfg.ExcludedApps = []string{"Terminal", "iTerm2"}

	data, err := cfg.ExportJSON()
	if err != nil {
		t.Fatalf("failed to export JSON: %v", err)
	}

	importedCfg := &Config{}
	err = importedCfg.ImportJSON(data)
	if err != nil {
		t.Fatalf("failed to import JSON: %v", err)
	}

	if importedCfg.Port != 8080 {
		t.Errorf("expected port 8080, got %d", importedCfg.Port)
	}
	if len(importedCfg.ExcludedApps) != 2 || importedCfg.ExcludedApps[0] != "Terminal" {
		t.Errorf("expected excluded apps to match exported")
	}
	if len(importedCfg.Profiles) != len(cfg.Profiles) {
		t.Errorf("profile counts do not match after round-trip")
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "sensei-config-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	cfg := DefaultConfig()
	cfg.Port = 9999

	err = saveConfigLocked(tmpFile.Name(), cfg)
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Read and verify
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to read back config: %v", err)
	}
	loaded := &Config{}
	if err := loaded.ImportJSON(data); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}
	if loaded.Port != 9999 {
		t.Errorf("expected port 9999, got %d", loaded.Port)
	}
}

func TestDefaultSequenceRules(t *testing.T) {
	seqs := DefaultSequenceRules()
	if len(seqs) < 4 {
		t.Errorf("expected at least 4 default sequences, got %d", len(seqs))
	}

	foundLess := false
	foundGreater := false
	for _, s := range seqs {
		if s.Output == "<" && s.Keycode == 43 {
			foundLess = true
		}
		if s.Output == ">" && s.Keycode == 47 {
			foundGreater = true
		}
	}
	if !foundLess || !foundGreater {
		t.Errorf("expected default double-tap sequences for < and >")
	}
}

func TestHyperKeyAndDevices(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.HyperKey.SourceKeycode != 57 {
		t.Errorf("expected default source keycode 57 (Caps Lock), got %d", cfg.HyperKey.SourceKeycode)
	}
	if cfg.HyperKey.TapAction != "escape" {
		t.Errorf("expected default tap action 'escape', got %s", cfg.HyperKey.TapAction)
	}

	// Add test device
	cfg.Devices = append(cfg.Devices, KeyboardDevice{
		ID:         "test-dev-1",
		Name:       "Test External Keyboard",
		VendorID:   1234,
		ProductID:  5678,
		Transport:  "USB",
		IsInternal: false,
		Enabled:    true,
	})

	data, err := cfg.ExportJSON()
	if err != nil {
		t.Fatalf("failed to export: %v", err)
	}

	imported := &Config{}
	if err := imported.ImportJSON(data); err != nil {
		t.Fatalf("failed to import: %v", err)
	}

	if len(imported.Devices) != 1 || imported.Devices[0].VendorID != 1234 {
		t.Errorf("devices did not round-trip correctly")
	}
}
