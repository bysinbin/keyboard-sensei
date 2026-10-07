package engine

import (
	"strings"
	"sync"
	"time"

	"keyboard-sensei/internal/config"
)

type TriggerEvent struct {
	RuleID    string    `json:"rule_id"`
	RuleName  string    `json:"rule_name"`
	Shortcut  string    `json:"shortcut"`
	Output    string    `json:"output"`
	Timestamp time.Time `json:"timestamp"`
}

type EngineStats struct {
	TotalTriggered int64            `json:"total_triggered"`
	RuleHits       map[string]int64 `json:"rule_hits"`
	StartedAt      time.Time        `json:"started_at"`
}

type Engine struct {
	mu                 sync.RWMutex
	rules              []config.Rule
	globalExcludedApps []string
	triggerEvents      chan TriggerEvent
	recentLogs         []TriggerEvent
	listeners          map[chan TriggerEvent]struct{}
	stats              EngineStats
}

var globalEngine *Engine

func NewEngine() *Engine {
	e := &Engine{
		rules:              make([]config.Rule, 0),
		globalExcludedApps: make([]string, 0),
		triggerEvents:      make(chan TriggerEvent, 100),
		recentLogs:         make([]TriggerEvent, 0, 50),
		listeners:          make(map[chan TriggerEvent]struct{}),
		stats: EngineStats{
			RuleHits:  make(map[string]int64),
			StartedAt: time.Now(),
		},
	}
	globalEngine = e
	go e.processEvents()
	return e
}

func GetEngine() *Engine {
	return globalEngine
}

func (e *Engine) processEvents() {
	for ev := range e.triggerEvents {
		e.mu.Lock()
		// Update stats
		e.stats.TotalTriggered++
		e.stats.RuleHits[ev.RuleID]++

		// Keep last 50 events
		e.recentLogs = append([]TriggerEvent{ev}, e.recentLogs...)
		if len(e.recentLogs) > 50 {
			e.recentLogs = e.recentLogs[:50]
		}

		// Broadcast to web SSE listeners
		for ch := range e.listeners {
			select {
			case ch <- ev:
			default:
			}
		}
		e.mu.Unlock()
	}
}

func (e *Engine) handleTrigger(ruleIndex int, ruleID string, output string) {
	e.mu.RLock()
	var ruleName string
	var shortcut string
	for _, r := range e.rules {
		if r.ID == ruleID {
			ruleName = r.Name
			shortcut = FormatShortcut(r.Modifiers, r.Keycode)
			break
		}
	}
	e.mu.RUnlock()

	if ruleName == "" {
		ruleName = "Kural #" + ruleID
	}

	ev := TriggerEvent{
		RuleID:    ruleID,
		RuleName:  ruleName,
		Shortcut:  shortcut,
		Output:    output,
		Timestamp: time.Now(),
	}

	select {
	case e.triggerEvents <- ev:
	default:
	}
}

func (e *Engine) SubscribeLogs() (<-chan TriggerEvent, func()) {
	ch := make(chan TriggerEvent, 20)
	e.mu.Lock()
	e.listeners[ch] = struct{}{}
	e.mu.Unlock()

	cancel := func() {
		e.mu.Lock()
		delete(e.listeners, ch)
		close(ch)
		e.mu.Unlock()
	}
	return ch, cancel
}

func (e *Engine) GetRecentLogs() []TriggerEvent {
	e.mu.RLock()
	defer e.mu.RUnlock()
	copied := make([]TriggerEvent, len(e.recentLogs))
	copy(copied, e.recentLogs)
	return copied
}

func (e *Engine) GetStats() EngineStats {
	e.mu.RLock()
	defer e.mu.RUnlock()
	hitsCopy := make(map[string]int64, len(e.stats.RuleHits))
	for k, v := range e.stats.RuleHits {
		hitsCopy[k] = v
	}
	return EngineStats{
		TotalTriggered: e.stats.TotalTriggered,
		RuleHits:       hitsCopy,
		StartedAt:      e.stats.StartedAt,
	}
}

func (e *Engine) SetGlobalExcludedApps(apps []string) {
	e.mu.Lock()
	e.globalExcludedApps = make([]string, len(apps))
	copy(e.globalExcludedApps, apps)
	e.mu.Unlock()

	e.updateDarwinGlobalExclusions(strings.Join(apps, ","))
}

func (e *Engine) GetGlobalExcludedApps() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	copied := make([]string, len(e.globalExcludedApps))
	copy(copied, e.globalExcludedApps)
	return copied
}

