package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"keyboard-sensei/internal/config"
	"keyboard-sensei/internal/engine"
	"keyboard-sensei/internal/service"
)

const AppVersion = "v1.2.0"

//go:embed static/*
var staticFS embed.FS

type Server struct {
	port   int
	engine *engine.Engine
}

func NewServer(port int, eng *engine.Engine) *Server {
	return &Server{
		port:   port,
		engine: eng,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/status/pause", s.handleTogglePause)
	mux.HandleFunc("/api/status/open_settings", s.handleOpenSettings)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/rules", s.handleRules)
	mux.HandleFunc("/api/rules/", s.handleRuleByID)
	mux.HandleFunc("/api/profiles", s.handleProfiles)
	mux.HandleFunc("/api/profiles/", s.handleProfileByID)
	mux.HandleFunc("/api/profiles/activate", s.handleActivateProfile)
	mux.HandleFunc("/api/export", s.handleExport)
	mux.HandleFunc("/api/import", s.handleImport)
	mux.HandleFunc("/api/active_app", s.handleActiveApp)
	mux.HandleFunc("/api/version", s.handleVersion)
	mux.HandleFunc("/api/keycodes", s.handleKeycodes)
	mux.HandleFunc("/api/presets/apply", s.handleApplyPreset)
	mux.HandleFunc("/api/service", s.handleService)
	mux.HandleFunc("/api/quit", s.handleQuit)
	mux.HandleFunc("/api/events", s.handleEventsSSE)
	mux.HandleFunc("/api/logs", s.handleLogs)
	mux.HandleFunc("/api/devices", s.handleDevices)
	mux.HandleFunc("/api/hyperkey", s.handleHyperKey)
	mux.HandleFunc("/api/sequences", s.handleSequences)

	// Static assets
	subFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		return err
	}
	fileServer := http.FileServer(http.FS(subFS))
	mux.Handle("/", fileServer)

	addr := fmt.Sprintf(":%d", s.port)
	fmt.Printf("🌐 Keyboard Sensei Web Arayüzü: http://localhost:%d\n", s.port)
	return http.ListenAndServe(addr, mux)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Auto-start event tap as soon as permission is granted
	if !s.engine.IsRunning() && s.engine.IsAccessibilityTrusted() {
		_ = s.engine.Start()
	}

	stats := s.engine.GetStats()
	cfg := config.Get()
	activeProfile := cfg.GetActiveProfile()
	appName, appBundleID := s.engine.GetFrontmostApp()

	resp := map[string]interface{}{
		"running":               s.engine.IsRunning(),
		"paused":                s.engine.IsPaused(),
		"accessibility_trusted": s.engine.IsAccessibilityTrusted(),
		"port":                  s.port,
		"rule_count":            len(cfg.Rules),
		"active_profile_id":     cfg.ActiveProfileID,
		"active_profile_name":   "",
		"service_installed":     service.IsInstalled(),
		"service_running":       service.IsRunning(),
		"version":               AppVersion,
		"active_app_name":       appName,
		"active_app_bundle":     appBundleID,
		"stats":                 stats,
		"config_file":           config.GetConfigFilePath(),
		"hyper_key_enabled":     cfg.HyperKey.Enabled,
		"sequences_enabled":     cfg.EnableSequences,
		"sequences_count":       len(cfg.SequenceRules),
	}

	if activeProfile != nil {
		resp["active_profile_name"] = activeProfile.Name
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleTogglePause(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Paused *bool `json:"paused"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Paused == nil {
		newVal := !s.engine.IsPaused()
		s.engine.SetPaused(newVal)
	} else {
		s.engine.SetPaused(*body.Paused)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"paused": s.engine.IsPaused(),
	})
}

func (s *Server) handleOpenSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_ = exec.Command("open", "x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility").Run()
	s.engine.PromptAccessibility()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "Sistem Ayarları açıldı",
	})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg := config.Get()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(cfg)

	case http.MethodPost:
		var updated config.Config
		if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := config.SaveConfig(&updated); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		s.engine.UpdateRules(updated.Rules)
		s.engine.SetGlobalExcludedApps(updated.ExcludedApps)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(updated)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleRules(w http.ResponseWriter, r *http.Request) {
	cfg := config.Get()

	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(cfg.Rules)

	case http.MethodPost:
		var newRule config.Rule
		if err := json.NewDecoder(r.Body).Decode(&newRule); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if newRule.ID == "" {
			newRule.ID = fmt.Sprintf("rule-%d", time.Now().UnixNano())
		}
		if newRule.KeyLabel == "" {
			if info, ok := engine.VirtualKeys[newRule.Keycode]; ok {
				newRule.KeyLabel = info.Name
			} else {
				newRule.KeyLabel = fmt.Sprintf("#%d", newRule.Keycode)
			}
		}

		// Add to active profile
		active := cfg.GetActiveProfile()
		if active != nil {
			active.Rules = append(active.Rules, newRule)
		} else {
			cfg.Rules = append(cfg.Rules, newRule)
		}
		cfg.SyncActiveRules()

		if err := config.SaveConfig(cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.engine.UpdateRules(cfg.Rules)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(newRule)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleRuleByID(w http.ResponseWriter, r *http.Request) {
	ruleID := strings.TrimPrefix(r.URL.Path, "/api/rules/")
	if ruleID == "" {
		http.Error(w, "Rule ID required", http.StatusBadRequest)
		return
	}

	cfg := config.Get()
	active := cfg.GetActiveProfile()
	if active == nil {
		http.Error(w, "Aktif profil bulunamadı", http.StatusInternalServerError)
		return
	}

	foundIdx := -1
	for i, rule := range active.Rules {
		if rule.ID == ruleID {
			foundIdx = i
			break
		}
	}

	switch r.Method {
	case http.MethodDelete:
		if foundIdx == -1 {
			http.Error(w, "Rule not found", http.StatusNotFound)
			return
		}
		active.Rules = append(active.Rules[:foundIdx], active.Rules[foundIdx+1:]...)
		cfg.SyncActiveRules()
		if err := config.SaveConfig(cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.engine.UpdateRules(cfg.Rules)
		w.WriteHeader(http.StatusNoContent)

	case http.MethodPut:
		if foundIdx == -1 {
			http.Error(w, "Rule not found", http.StatusNotFound)
			return
		}
		var updated config.Rule
		if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		updated.ID = ruleID
		active.Rules[foundIdx] = updated
		cfg.SyncActiveRules()
		if err := config.SaveConfig(cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.engine.UpdateRules(cfg.Rules)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(updated)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleProfiles(w http.ResponseWriter, r *http.Request) {
	cfg := config.Get()

	switch r.Method {
	case http.MethodGet:
		resp := map[string]interface{}{
			"active_profile_id": cfg.ActiveProfileID,
			"profiles":          cfg.Profiles,
			"presets":           config.BuiltinPresets(),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)

	case http.MethodPost:
		var newProfile config.Profile
		if err := json.NewDecoder(r.Body).Decode(&newProfile); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := cfg.AddProfile(newProfile); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := config.SaveConfig(cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(newProfile)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleProfileByID(w http.ResponseWriter, r *http.Request) {
	profileID := strings.TrimPrefix(r.URL.Path, "/api/profiles/")
	if profileID == "" {
		http.Error(w, "Profile ID required", http.StatusBadRequest)
		return
	}

	cfg := config.Get()

	switch r.Method {
	case http.MethodPut:
		var updated config.Profile
		if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		updated.ID = profileID
		if err := cfg.UpdateProfile(updated); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := config.SaveConfig(cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.engine.UpdateRules(cfg.Rules)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(updated)

	case http.MethodDelete:
		if err := cfg.DeleteProfile(profileID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := config.SaveConfig(cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.engine.UpdateRules(cfg.Rules)
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleActivateProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		ProfileID string `json:"profile_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ProfileID == "" {
		http.Error(w, "Geçerli bir profile_id gereklidir", http.StatusBadRequest)
		return
	}

	cfg := config.Get()
	if err := cfg.SetActiveProfile(body.ProfileID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := config.SaveConfig(cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.engine.UpdateRules(cfg.Rules)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"active_profile_id": cfg.ActiveProfileID,
		"rules":             cfg.Rules,
	})
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg := config.Get()
	data, err := cfg.ExportJSON()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=\"keyboard-sensei-backup.json\"")
	_, _ = w.Write(data)
}

func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cfg := config.Get()
	if err := cfg.ImportJSON(data); err != nil {
		http.Error(w, fmt.Sprintf("İçe aktarma hatası: %v", err), http.StatusBadRequest)
		return
	}

	if err := config.SaveConfig(cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.engine.UpdateRules(cfg.Rules)
	s.engine.SetGlobalExcludedApps(cfg.ExcludedApps)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Konfigürasyon başarıyla içe aktarıldı",
		"config":  cfg,
	})
}

func (s *Server) handleActiveApp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name, bundleID := s.engine.GetFrontmostApp()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"name":      name,
		"bundle_id": bundleID,
	})
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := map[string]interface{}{
		"version":    AppVersion,
		"github_url": "https://github.com/bysinbin/keyboard-sensei",
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleKeycodes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	keys := make([]engine.KeyInfo, 0, len(engine.VirtualKeys))
	for _, k := range engine.VirtualKeys {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Code < keys[j].Code
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(keys)
}

func (s *Server) handleApplyPreset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Preset string `json:"preset"` // "profile-ansi-tr", "profile-dev", "profile-rdp"
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	cfg := config.Get()
	presetName := body.Preset
	if presetName == "" || presetName == "ansi_turkish" {
		presetName = "profile-ansi-tr"
	}

	presets := config.BuiltinPresets()
	for _, p := range presets {
		if p.ID == presetName {
			active := cfg.GetActiveProfile()
			if active != nil {
				active.Rules = p.Rules
			} else {
				cfg.Rules = p.Rules
			}
			cfg.SyncActiveRules()
			break
		}
	}

	if err := config.SaveConfig(cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.engine.UpdateRules(cfg.Rules)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(cfg)
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	logs := s.engine.GetRecentLogs()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(logs)
}

func (s *Server) handleEventsSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	logChan, cancel := s.engine.SubscribeLogs()
	defer cancel()

	// Initial heartbeat
	fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"ok\"}\n\n")
	flusher.Flush()

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case ev, ok := <-logChan:
			if !ok {
				return
			}
			data, _ := json.Marshal(ev)
			fmt.Fprintf(w, "event: trigger\ndata: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func (s *Server) handleService(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"installed":  service.IsInstalled(),
			"running":    service.IsRunning(),
			"plist_path": service.GetPlistPath(),
			"log_path":   service.GetLogPath(),
		})

	case http.MethodPost:
		var body struct {
			Action string `json:"action"` // "install", "uninstall", "toggle"
		}
		_ = json.NewDecoder(r.Body).Decode(&body)

		var err error
		if body.Action == "install" || (body.Action == "toggle" && !service.IsInstalled()) {
			err = service.Install("")
		} else if body.Action == "uninstall" || (body.Action == "toggle" && service.IsInstalled()) {
			err = service.Uninstall()
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"installed":  service.IsInstalled(),
			"running":    service.IsRunning(),
			"plist_path": service.GetPlistPath(),
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleQuit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "Uygulama ve servis kapatılıyor...",
	})

	go func() {
		time.Sleep(200 * time.Millisecond)
		s.engine.Stop()
		_ = service.Uninstall()
		engine.StopMacAppLoop()
		os.Exit(0)
	}()
}

func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	cfg := config.Get()

	switch r.Method {
	case http.MethodGet:
		connected := s.engine.GetConnectedKeyboards()

		// Merge with saved enabled/disabled status in config
		savedMap := make(map[string]bool)
		for _, d := range cfg.Devices {
			key := fmt.Sprintf("%d:%d", d.VendorID, d.ProductID)
			savedMap[key] = d.Enabled
		}

		result := make([]config.KeyboardDevice, len(connected))
		for i, dev := range connected {
			result[i] = dev
			key := fmt.Sprintf("%d:%d", dev.VendorID, dev.ProductID)
			if enabled, exists := savedMap[key]; exists {
				result[i].Enabled = enabled
			} else {
				result[i].Enabled = true
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"devices": result,
		})

	case http.MethodPost:
		var body struct {
			Devices []config.KeyboardDevice `json:"devices"`
			Device  *config.KeyboardDevice  `json:"device,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if body.Device != nil {
			// Single device toggle/update
			updated := false
			for i, d := range cfg.Devices {
				if (d.VendorID == body.Device.VendorID && d.ProductID == body.Device.ProductID) || d.ID == body.Device.ID {
					cfg.Devices[i].Enabled = body.Device.Enabled
					if body.Device.Name != "" {
						cfg.Devices[i].Name = body.Device.Name
					}
					updated = true
					break
				}
			}
			if !updated {
				cfg.Devices = append(cfg.Devices, *body.Device)
			}
		} else if len(body.Devices) > 0 {
			cfg.Devices = body.Devices
		}

		if err := config.SaveConfig(cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		s.engine.UpdateDeviceFilters(cfg.Devices)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"devices": cfg.Devices,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleHyperKey(w http.ResponseWriter, r *http.Request) {
	cfg := config.Get()

	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(cfg.HyperKey)

	case http.MethodPost:
		var updated config.HyperKeyConfig
		if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if updated.SourceKeycode == 0 {
			updated.SourceKeycode = 57 // Caps Lock
		}
		if len(updated.Modifiers) == 0 {
			updated.Modifiers = []string{"cmd", "alt", "ctrl", "shift"}
		}
		if updated.TapAction == "" {
			updated.TapAction = "escape"
		}

		cfg.HyperKey = updated
		if err := config.SaveConfig(cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		s.engine.UpdateHyperKey(cfg.HyperKey)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(cfg.HyperKey)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSequences(w http.ResponseWriter, r *http.Request) {
	cfg := config.Get()

	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"enabled":   cfg.EnableSequences,
			"sequences": cfg.SequenceRules,
		})

	case http.MethodPost:
		var body struct {
			Enabled   *bool                 `json:"enabled"`
			Sequences []config.SequenceRule `json:"sequences"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if body.Enabled != nil {
			cfg.EnableSequences = *body.Enabled
		}
		if body.Sequences != nil {
			cfg.SequenceRules = body.Sequences
		}

		if err := config.SaveConfig(cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		s.engine.UpdateSequenceRules(cfg.SequenceRules, cfg.EnableSequences)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"enabled":   cfg.EnableSequences,
			"sequences": cfg.SequenceRules,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
