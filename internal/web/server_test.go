package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"keyboard-sensei/internal/config"
	"keyboard-sensei/internal/engine"
)

func setupTestServer() (*httptest.Server, *Server) {
	eng := engine.NewEngine()
	srv := NewServer(0, eng)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", srv.handleStatus)
	mux.HandleFunc("/api/rules", srv.handleRules)
	mux.HandleFunc("/api/profiles", srv.handleProfiles)
	mux.HandleFunc("/api/keycodes", srv.handleKeycodes)
	mux.HandleFunc("/api/version", srv.handleVersion)
	mux.HandleFunc("/api/export", srv.handleExport)
	mux.HandleFunc("/api/devices", srv.handleDevices)
	mux.HandleFunc("/api/hyperkey", srv.handleHyperKey)
	mux.HandleFunc("/api/sequences", srv.handleSequences)

	ts := httptest.NewServer(mux)
	return ts, srv
}

func TestStatusEndpoint(t *testing.T) {
	ts, _ := setupTestServer()
	defer ts.Close()

	res, err := http.Get(ts.URL + "/api/status")
	if err != nil {
		t.Fatalf("failed to GET /api/status: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", res.StatusCode)
	}

	var data map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode status response: %v", err)
	}

	if data["version"] != AppVersion {
		t.Errorf("expected version %s, got %v", AppVersion, data["version"])
	}
}

func TestProfilesEndpoint(t *testing.T) {
	ts, _ := setupTestServer()
	defer ts.Close()

	// GET profiles
	res, err := http.Get(ts.URL + "/api/profiles")
	if err != nil {
		t.Fatalf("failed to GET /api/profiles: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", res.StatusCode)
	}

	var data struct {
		ActiveProfileID string           `json:"active_profile_id"`
		Profiles        []config.Profile `json:"profiles"`
	}
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode profiles: %v", err)
	}

	if len(data.Profiles) == 0 {
		t.Errorf("expected non-empty profiles list")
	}

	// POST new profile
	newP := `{"name":"Test Profil","description":"Açıklama"}`
	postRes, err := http.Post(ts.URL+"/api/profiles", "application/json", strings.NewReader(newP))
	if err != nil {
		t.Fatalf("failed to POST /api/profiles: %v", err)
	}
	defer postRes.Body.Close()

	if postRes.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", postRes.StatusCode)
	}
}

func TestKeycodesEndpoint(t *testing.T) {
	ts, _ := setupTestServer()
	defer ts.Close()

	res, err := http.Get(ts.URL + "/api/keycodes")
	if err != nil {
		t.Fatalf("failed to GET /api/keycodes: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", res.StatusCode)
	}

	var keys []engine.KeyInfo
	if err := json.NewDecoder(res.Body).Decode(&keys); err != nil {
		t.Fatalf("failed to decode keycodes: %v", err)
	}

	if len(keys) < 20 {
		t.Errorf("expected at least 20 keycodes, got %d", len(keys))
	}
}

func TestExportEndpoint(t *testing.T) {
	ts, _ := setupTestServer()
	defer ts.Close()

	res, err := http.Get(ts.URL + "/api/export")
	if err != nil {
		t.Fatalf("failed to GET /api/export: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", res.StatusCode)
	}

	var cfg config.Config
	if err := json.NewDecoder(res.Body).Decode(&cfg); err != nil {
		t.Fatalf("failed to decode exported config: %v", err)
	}

	if len(cfg.Profiles) == 0 {
		t.Errorf("expected profiles in exported config")
	}
}

func TestDevicesEndpoint(t *testing.T) {
	ts, _ := setupTestServer()
	defer ts.Close()

	res, err := http.Get(ts.URL + "/api/devices")
	if err != nil {
		t.Fatalf("failed to GET /api/devices: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", res.StatusCode)
	}

	var data map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode devices: %v", err)
	}
	if _, ok := data["devices"]; !ok {
		t.Errorf("expected 'devices' array in response")
	}
}

func TestHyperKeyEndpoint(t *testing.T) {
	ts, _ := setupTestServer()
	defer ts.Close()

	// GET
	res, err := http.Get(ts.URL + "/api/hyperkey")
	if err != nil {
		t.Fatalf("failed to GET /api/hyperkey: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", res.StatusCode)
	}

	// POST
	updatePayload := `{"enabled":true,"source_keycode":57,"modifiers":["cmd","alt","ctrl","shift"],"tap_action":"escape"}`
	postRes, err := http.Post(ts.URL+"/api/hyperkey", "application/json", strings.NewReader(updatePayload))
	if err != nil {
		t.Fatalf("failed to POST /api/hyperkey: %v", err)
	}
	defer postRes.Body.Close()

	if postRes.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", postRes.StatusCode)
	}

	var hk config.HyperKeyConfig
	if err := json.NewDecoder(postRes.Body).Decode(&hk); err != nil {
		t.Fatalf("failed to decode hyperkey response: %v", err)
	}
	if !hk.Enabled || hk.TapAction != "escape" {
		t.Errorf("hyperkey config update did not apply correctly")
	}
}

func TestSequencesEndpoint(t *testing.T) {
	ts, _ := setupTestServer()
	defer ts.Close()

	// GET
	res, err := http.Get(ts.URL + "/api/sequences")
	if err != nil {
		t.Fatalf("failed to GET /api/sequences: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", res.StatusCode)
	}

	var data struct {
		Enabled   bool                  `json:"enabled"`
		Sequences []config.SequenceRule `json:"sequences"`
	}
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode sequences: %v", err)
	}
	if len(data.Sequences) == 0 {
		t.Errorf("expected default sequences in response")
	}
}
