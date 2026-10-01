package web

import (
	"embed"
	"encoding/json"
	"fmt"
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
	mux.HandleFunc("/api/keycodes", s.handleKeycodes)
	mux.HandleFunc("/api/presets/apply", s.handleApplyPreset)
	mux.HandleFunc("/api/service", s.handleService)
	mux.HandleFunc("/api/quit", s.handleQuit)
	mux.HandleFunc("/api/events", s.handleEventsSSE)
	mux.HandleFunc("/api/logs", s.handleLogs)

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

	resp := map[string]interface{}{
		"running":               s.engine.IsRunning(),
		"paused":                s.engine.IsPaused(),
		"accessibility_trusted": s.engine.IsAccessibilityTrusted(),
		"port":                  s.port,
		"rule_count":            len(cfg.Rules),
		"service_installed":     service.IsInstalled(),
		"service_running":       service.IsRunning(),
		"stats":                 stats,
		"config_file":           config.GetConfigFilePath(),
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
		// Toggle if not specified
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

	// Open macOS Accessibility settings directly
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

		cfg.Rules = append(cfg.Rules, newRule)
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
	foundIdx := -1
	for i, rule := range cfg.Rules {
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
		cfg.Rules = append(cfg.Rules[:foundIdx], cfg.Rules[foundIdx+1:]...)
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
		cfg.Rules[foundIdx] = updated
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
		Preset string `json:"preset"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	cfg := config.Get()
	if body.Preset == "ansi_turkish" || body.Preset == "" {
		def := config.DefaultConfig()
		cfg.Rules = def.Rules
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
		os.Exit(0)
	}()
}
