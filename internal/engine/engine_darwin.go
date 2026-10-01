//go:build darwin

package engine

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation -framework ApplicationServices -framework Foundation

#import <ApplicationServices/ApplicationServices.h>
#import <CoreGraphics/CoreGraphics.h>
#import <Foundation/Foundation.h>
#import <pthread.h>

extern void goOnHotkeyTriggered(int ruleIndex, char* ruleId, char* output);

typedef struct {
    int ruleIndex;
    char ruleId[64];
    uint32_t flags; // masked modifier flags
    int keycode;
    UniChar outputChars[64];
    int outputLen;
    bool enabled;
} CRule;

#define MAX_RULES 256
static CRule g_rules[MAX_RULES];
static int g_ruleCount = 0;
static pthread_mutex_t g_rulesMutex = PTHREAD_MUTEX_INITIALIZER;

static CFMachPortRef g_eventTap = NULL;
static CFRunLoopSourceRef g_runLoopSource = NULL;
static CFRunLoopRef g_runLoop = NULL;
static bool g_isRunning = false;
static bool g_isPaused = false;
static bool g_interceptedKeys[256];

static uint32_t extractModifiers(CGEventFlags flags) {
    uint32_t mod = 0;
    if (flags & kCGEventFlagMaskCommand)   mod |= (1 << 0); // CMD
    if (flags & kCGEventFlagMaskAlternate) mod |= (1 << 1); // ALT / OPTION
    if (flags & kCGEventFlagMaskControl)   mod |= (1 << 2); // CTRL
    if (flags & kCGEventFlagMaskShift)     mod |= (1 << 3); // SHIFT
    return mod;
}

static void injectUnicodeString(const UniChar* chars, int len) {
    if (!chars || len <= 0) return;
    
    CGEventSourceRef src = CGEventSourceCreate(kCGEventSourceStatePrivate);
    
    CGEventRef down = CGEventCreateKeyboardEvent(src, 0, true);
    CGEventKeyboardSetUnicodeString(down, len, chars);
    CGEventSetFlags(down, 0);
    CGEventPost(kCGAnnotatedSessionEventTap, down);
    CFRelease(down);
    
    CGEventRef up = CGEventCreateKeyboardEvent(src, 0, false);
    CGEventKeyboardSetUnicodeString(up, len, chars);
    CGEventSetFlags(up, 0);
    CGEventPost(kCGAnnotatedSessionEventTap, up);
    CFRelease(up);
    
    CFRelease(src);
}

static CGEventRef eventTapCallback(CGEventTapProxy proxy, CGEventType type, CGEventRef event, void *refcon) {
    if (type == kCGEventTapDisabledByTimeout || type == kCGEventTapDisabledByUserInput) {
        if (g_eventTap) {
            CGEventTapEnable(g_eventTap, true);
        }
        return event;
    }
    
    if (g_isPaused) {
        return event;
    }
    
    int64_t keycode = CGEventGetIntegerValueField(event, kCGKeyboardEventKeycode);
    if (keycode < 0 || keycode >= 256) return event;
    
    if (type == kCGEventKeyUp) {
        if (g_interceptedKeys[keycode]) {
            g_interceptedKeys[keycode] = false;
            return NULL;
        }
        return event;
    }
    
    if (type != kCGEventKeyDown) {
        return event;
    }
    
    CGEventFlags rawFlags = CGEventGetFlags(event);
    uint32_t eventMods = extractModifiers(rawFlags);
    
    pthread_mutex_lock(&g_rulesMutex);
    for (int i = 0; i < g_ruleCount; i++) {
        if (g_rules[i].enabled && g_rules[i].keycode == (int)keycode && g_rules[i].flags == eventMods) {
            CRule match = g_rules[i];
            pthread_mutex_unlock(&g_rulesMutex);
            
            g_interceptedKeys[keycode] = true;
            injectUnicodeString(match.outputChars, match.outputLen);
            
            char outBuf[128] = {0};
            CFStringRef cfOut = CFStringCreateWithCharacters(kCFAllocatorDefault, match.outputChars, match.outputLen);
            if (cfOut) {
                CFStringGetCString(cfOut, outBuf, sizeof(outBuf), kCFStringEncodingUTF8);
                CFRelease(cfOut);
            }
            goOnHotkeyTriggered(match.ruleIndex, match.ruleId, outBuf);
            
            return NULL;
        }
    }
    pthread_mutex_unlock(&g_rulesMutex);
    
    return event;
}

static void* runLoopThread(void* arg) {
    g_runLoop = CFRunLoopGetCurrent();
    CFRunLoopAddSource(g_runLoop, g_runLoopSource, kCFRunLoopCommonModes);
    CGEventTapEnable(g_eventTap, true);
    g_isRunning = true;
    CFRunLoopRun();
    g_isRunning = false;
    return NULL;
}

static bool c_startTap() {
    if (g_isRunning) return true;
    
    CGEventMask mask = CGEventMaskBit(kCGEventKeyDown) | CGEventMaskBit(kCGEventKeyUp);
    g_eventTap = CGEventTapCreate(
        kCGSessionEventTap,
        kCGHeadInsertEventTap,
        kCGEventTapOptionDefault,
        mask,
        eventTapCallback,
        NULL
    );
    
    if (!g_eventTap) {
        return false;
    }
    
    g_runLoopSource = CFMachPortCreateRunLoopSource(kCFAllocatorDefault, g_eventTap, 0);
    if (!g_runLoopSource) {
        CFRelease(g_eventTap);
        g_eventTap = NULL;
        return false;
    }
    
    pthread_t thread;
    pthread_create(&thread, NULL, runLoopThread, NULL);
    pthread_detach(thread);
    
    return true;
}

static void c_stopTap() {
    if (!g_isRunning) return;
    if (g_runLoop) {
        CFRunLoopStop(g_runLoop);
    }
    if (g_eventTap) {
        CGEventTapEnable(g_eventTap, false);
    }
}

static void c_setPaused(bool paused) {
    g_isPaused = paused;
}

static bool c_isPaused() {
    return g_isPaused;
}

static bool c_isRunning() {
    return g_isRunning;
}

static bool c_checkAccessibility() {
    NSDictionary *options = @{(__bridge id)kAXTrustedCheckOptionPrompt: @NO};
    return AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)options);
}

static void c_promptAccessibility() {
    NSDictionary *options = @{(__bridge id)kAXTrustedCheckOptionPrompt: @YES};
    AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)options);
}

static void c_clearRules() {
    pthread_mutex_lock(&g_rulesMutex);
    g_ruleCount = 0;
    pthread_mutex_unlock(&g_rulesMutex);
}

static void c_addRule(int index, const char* ruleId, uint32_t flags, int keycode, const char* outputUtf8, bool enabled) {
    pthread_mutex_lock(&g_rulesMutex);
    if (g_ruleCount >= MAX_RULES) {
        pthread_mutex_unlock(&g_rulesMutex);
        return;
    }
    
    CRule *r = &g_rules[g_ruleCount];
    r->ruleIndex = index;
    strncpy(r->ruleId, ruleId ? ruleId : "", sizeof(r->ruleId) - 1);
    r->flags = flags;
    r->keycode = keycode;
    r->enabled = enabled;
    r->outputLen = 0;
    
    if (outputUtf8) {
        CFStringRef cf = CFStringCreateWithCString(kCFAllocatorDefault, outputUtf8, kCFStringEncodingUTF8);
        if (cf) {
            CFIndex len = CFStringGetLength(cf);
            if (len > 64) len = 64;
            CFStringGetCharacters(cf, CFRangeMake(0, len), r->outputChars);
            r->outputLen = (int)len;
            CFRelease(cf);
        }
    }
    
    g_ruleCount++;
    pthread_mutex_unlock(&g_rulesMutex);
}
*/
import "C"

import (
	"errors"
	"unsafe"

	"keyboard-sensei/internal/config"
)

//export goOnHotkeyTriggered
func goOnHotkeyTriggered(ruleIndex C.int, ruleId *C.char, output *C.char) {
	if globalEngine != nil {
		id := C.GoString(ruleId)
		out := C.GoString(output)
		globalEngine.handleTrigger(int(ruleIndex), id, out)
	}
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
		C.c_addRule(C.int(i), cID, C.uint32_t(mask), C.int(r.Keycode), cOut, C.bool(r.Enabled))
		C.free(unsafe.Pointer(cID))
		C.free(unsafe.Pointer(cOut))
	}
}
