//go:build darwin

package engine

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation -framework ApplicationServices -framework Foundation -framework AppKit

#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

extern void goOnHotkeyTriggered(int ruleIndex, char* ruleId, char* output);
extern void goTrayTogglePause();
extern void goTrayOpenDashboard();
extern void goTrayToggleAutoStart();
extern void goTrayQuit();
extern void goTrayGetMenuState(char* statusTextBuf, int statusTextLen, bool* isPaused, bool* isAutoStart);

bool c_startTap();
void c_stopTap();
void c_setPaused(bool paused);
bool c_isPaused();
bool c_isRunning();
bool c_checkAccessibility();
void c_promptAccessibility();
void c_clearRules();
void c_setGlobalExcludedApps(const char* apps);
void c_addRule(int index, const char* ruleId, uint32_t flags, int keycode, const char* outputUtf8, bool enabled, const char* targetApps, const char* excludedApps);
void c_getFrontmostApp(char* nameBuf, int nameBufLen, char* idBuf, int idBufLen);
void c_runCocoaLoop();
void c_stopCocoaLoop();
*/
import "C"

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unsafe"

	"keyboard-sensei/internal/config"
	"keyboard-sensei/internal/service"
)

//export goOnHotkeyTriggered
func goOnHotkeyTriggered(ruleIndex C.int, ruleId *C.char, output *C.char) {
	if globalEngine != nil {
		id := C.GoString(ruleId)
		out := C.GoString(output)
		globalEngine.handleTrigger(int(ruleIndex), id, out)
	}
}

//export goTrayTogglePause
func goTrayTogglePause() {
	if globalEngine != nil {
		globalEngine.SetPaused(!globalEngine.IsPaused())
	}
}

//export goTrayOpenDashboard
func goTrayOpenDashboard() {
	cfg := config.Get()
	url := fmt.Sprintf("http://localhost:%d", cfg.Port)
	_ = exec.Command("open", url).Start()
}

//export goTrayToggleAutoStart
func goTrayToggleAutoStart() {
	if service.IsInstalled() {
		_ = service.Uninstall()
	} else {
		_ = service.Install("")
	}
}

//export goTrayQuit
func goTrayQuit() {
	if globalEngine != nil {
		globalEngine.Stop()
	}
	C.c_stopCocoaLoop()
	os.Exit(0)
}

//export goTrayGetMenuState
func goTrayGetMenuState(statusTextBuf *C.char, statusTextLen C.int, isPaused *C.bool, isAutoStart *C.bool) {
	trusted := true
	paused := false
	if globalEngine != nil {
		trusted = globalEngine.IsAccessibilityTrusted()
		paused = globalEngine.IsPaused()
	}
	autoStart := service.IsInstalled()

	var statusText string
	if !trusted {
		statusText = "⚠️ Erişilebilirlik İzni Gerekli"
	} else if paused {
		statusText = "⏸️ Duraklatıldı"
	} else {
		statusText = "✅ Motor Aktif"
	}

	cStr := C.CString(statusText)
	C.strncpy(statusTextBuf, cStr, C.size_t(statusTextLen-1))
	C.free(unsafe.Pointer(cStr))

	*isPaused = C.bool(paused)
	*isAutoStart = C.bool(autoStart)
}

func (e *Engine) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !bool(C.c_checkAccessibility()) {
		C.c_promptAccessibility()
	}

	success := bool(C.c_startTap())
	if !success {
		return errors.New("CGEventTap başlatılamadı. Lütfen Sistem Ayarları > Gizlilik ve Güvenlik > Erişilebilirlik iznini kontrol edin.")
	}
	return nil
}

func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	C.c_stopTap()
}

func (e *Engine) SetPaused(paused bool) {
	C.c_setPaused(C.bool(paused))
}

func (e *Engine) IsPaused() bool {
	return bool(C.c_isPaused())
}

func (e *Engine) IsRunning() bool {
	return bool(C.c_isRunning())
}

func (e *Engine) IsAccessibilityTrusted() bool {
	return bool(C.c_checkAccessibility())
}

func (e *Engine) PromptAccessibility() {
	C.c_promptAccessibility()
}

func (e *Engine) updateDarwinGlobalExclusions(appsCsv string) {
	cApps := C.CString(appsCsv)
	defer C.free(unsafe.Pointer(cApps))
	C.c_setGlobalExcludedApps(cApps)
}

func (e *Engine) UpdateRules(rules []config.Rule) {
	e.mu.Lock()
	e.rules = make([]config.Rule, len(rules))
	copy(e.rules, rules)
	e.mu.Unlock()

	C.c_clearRules()
	for i, r := range rules {
		mask := ModifiersToMask(r.Modifiers)
		cID := C.CString(r.ID)
		cOut := C.CString(r.Output)
		cTarget := C.CString(strings.Join(r.TargetApps, ","))
		cExcluded := C.CString(strings.Join(r.ExcludedApps, ","))

		C.c_addRule(
			C.int(i),
			cID,
			C.uint32_t(mask),
			C.int(r.Keycode),
			cOut,
			C.bool(r.Enabled),
			cTarget,
			cExcluded,
		)

		C.free(unsafe.Pointer(cID))
		C.free(unsafe.Pointer(cOut))
		C.free(unsafe.Pointer(cTarget))
		C.free(unsafe.Pointer(cExcluded))
	}
}

// GetFrontmostApp returns the name and bundle ID of the currently focused macOS app
func (e *Engine) GetFrontmostApp() (name string, bundleID string) {
	var nameBuf [256]C.char
	var idBuf [256]C.char
	C.c_getFrontmostApp(&nameBuf[0], 256, &idBuf[0], 256)
	return C.GoString(&nameBuf[0]), C.GoString(&idBuf[0])
}

// RunMacAppLoop starts the macOS Cocoa event loop with status bar icon
func RunMacAppLoop() {
	C.c_runCocoaLoop()
}

// StopMacAppLoop terminates the macOS Cocoa application loop
func StopMacAppLoop() {
	C.c_stopCocoaLoop()
}
