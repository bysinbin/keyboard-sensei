//go:build !darwin

package engine

import (
	"errors"

	"keyboard-sensei/internal/config"
)

func (e *Engine) Start() error {
	return errors.New("keyboard-sensei yalnızca macOS platformunu destekler")
}

func (e *Engine) Stop()               {}
func (e *Engine) SetPaused(bool)      {}
func (e *Engine) IsPaused() bool      { return false }
func (e *Engine) IsRunning() bool     { return false }
func (e *Engine) IsAccessibilityTrusted() bool { return false }
func (e *Engine) PromptAccessibility() {}
func (e *Engine) UpdateRules([]config.Rule) {}
