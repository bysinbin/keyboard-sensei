//go:build darwin

#import <ApplicationServices/ApplicationServices.h>
#import <CoreGraphics/CoreGraphics.h>
#import <Foundation/Foundation.h>
#import <AppKit/AppKit.h>
#import <IOKit/IOKitLib.h>
#import <IOKit/hid/IOHIDManager.h>
#import <pthread.h>
#import <stdbool.h>
#import <stdint.h>
#import <sys/time.h>

extern void goOnHotkeyTriggered(int ruleIndex, char* ruleId, char* output);
extern void goTrayTogglePause();
extern void goTrayOpenDashboard();
extern void goTrayToggleAutoStart();
extern void goTrayQuit();
extern void goTrayGetMenuState(char* statusTextBuf, int statusTextLen, bool* isPaused, bool* isAutoStart);

typedef struct {
    int ruleIndex;
    char ruleId[64];
    uint32_t flags; // masked modifier flags
    int keycode;
    char rawOutputUtf8[1024];
    UniChar outputChars[1024];
    int outputLen;
    bool isDynamic;
    bool enabled;
    char targetApps[256];
    char excludedApps[256];
} CRule;

#define MAX_RULES 256
static CRule g_rules[MAX_RULES];
static int g_ruleCount = 0;
static pthread_mutex_t g_rulesMutex = PTHREAD_MUTEX_INITIALIZER;

// Sequence Rules (e.g. öö -> <, çç -> >)
typedef struct {
    int keycode;
    UniChar outputChars[64];
    int outputLen;
    uint64_t timeoutMs;
    bool enabled;
} CSequenceRule;

#define MAX_SEQUENCES 32
static CSequenceRule g_seqRules[MAX_SEQUENCES];
static int g_seqRuleCount = 0;
static bool g_enableSequences = true;
static int g_lastSequenceKey = -1;
static uint64_t g_lastSequenceTime = 0;

// Hyper Key Configuration
static bool g_hyperKeyEnabled = false;
static int g_hyperKeySource = 57; // Caps Lock
static uint32_t g_hyperKeyMask = (1 << 0) | (1 << 1) | (1 << 2) | (1 << 3); // ⌘⌥⌃⇧
static int g_hyperTapAction = 1; // 1: escape, 2: caps_lock, 0: none
static bool g_hyperIsDown = false;
static uint64_t g_hyperDownTime = 0;
static bool g_hyperUsedInCombination = false;

// Device Filter Table
typedef struct {
    int vendorId;
    int productId;
    bool enabled;
} CDeviceFilter;

#define MAX_DEVICE_FILTERS 64
static CDeviceFilter g_deviceFilters[MAX_DEVICE_FILTERS];
static int g_deviceFilterCount = 0;
static pthread_mutex_t g_deviceMutex = PTHREAD_MUTEX_INITIALIZER;

static int g_lastActiveDeviceVID = 0;
static int g_lastActiveDevicePID = 0;

static char g_globalExcludedApps[512] = {0};
static pthread_mutex_t g_exclusionsMutex = PTHREAD_MUTEX_INITIALIZER;

static CFMachPortRef g_eventTap = NULL;
static CFRunLoopSourceRef g_runLoopSource = NULL;
static CFRunLoopRef g_runLoop = NULL;
static bool g_isRunning = false;
static bool g_isPaused = false;
static bool g_interceptedKeys[256];

static uint64_t getTimestampMs() {
    struct timeval tv;
    gettimeofday(&tv, NULL);
    return ((uint64_t)tv.tv_sec * 1000) + ((uint64_t)tv.tv_usec / 1000);
}

static uint32_t extractModifiers(CGEventFlags flags) {
    uint32_t mod = 0;
    if (flags & kCGEventFlagMaskCommand)   mod |= (1 << 0); // CMD
    if (flags & kCGEventFlagMaskAlternate) mod |= (1 << 1); // ALT / OPTION
    if (flags & kCGEventFlagMaskControl)   mod |= (1 << 2); // CTRL
    if (flags & kCGEventFlagMaskShift)     mod |= (1 << 3); // SHIFT
    return mod;
}

#define SENSEI_EVENT_MAGIC 0x53454E53

static void injectUnicodeString(const UniChar* chars, int len) {
    if (!chars || len <= 0) return;
    
    CGEventSourceRef src = CGEventSourceCreate(kCGEventSourceStatePrivate);
    
    CGEventRef down = CGEventCreateKeyboardEvent(src, 0, true);
    CGEventKeyboardSetUnicodeString(down, len, chars);
    CGEventSetFlags(down, 0);
    CGEventSetIntegerValueField(down, kCGEventSourceUserData, SENSEI_EVENT_MAGIC);
    CGEventPost(kCGSessionEventTap, down);
    CFRelease(down);
    
    CGEventRef up = CGEventCreateKeyboardEvent(src, 0, false);
    CGEventKeyboardSetUnicodeString(up, len, chars);
    CGEventSetFlags(up, 0);
    CGEventSetIntegerValueField(up, kCGEventSourceUserData, SENSEI_EVENT_MAGIC);
    CGEventPost(kCGSessionEventTap, up);
    CFRelease(up);
    
    CFRelease(src);
}

static void injectBackspace() {
    CGEventSourceRef src = CGEventSourceCreate(kCGEventSourceStatePrivate);
    CGEventRef down = CGEventCreateKeyboardEvent(src, 51, true); // 51: Backspace
    CGEventSetFlags(down, 0);
    CGEventSetIntegerValueField(down, kCGEventSourceUserData, SENSEI_EVENT_MAGIC);
    CGEventPost(kCGSessionEventTap, down);
    CFRelease(down);
    
    CGEventRef up = CGEventCreateKeyboardEvent(src, 51, false);
    CGEventSetFlags(up, 0);
    CGEventSetIntegerValueField(up, kCGEventSourceUserData, SENSEI_EVENT_MAGIC);
    CGEventPost(kCGSessionEventTap, up);
    CFRelease(up);
    CFRelease(src);
}

static void injectEscape() {
    CGEventSourceRef src = CGEventSourceCreate(kCGEventSourceStatePrivate);
    CGEventRef down = CGEventCreateKeyboardEvent(src, 53, true); // 53: ESC
    CGEventSetFlags(down, 0);
    CGEventSetIntegerValueField(down, kCGEventSourceUserData, SENSEI_EVENT_MAGIC);
    CGEventPost(kCGSessionEventTap, down);
    CFRelease(down);
    
    CGEventRef up = CGEventCreateKeyboardEvent(src, 53, false);
    CGEventSetFlags(up, 0);
    CGEventSetIntegerValueField(up, kCGEventSourceUserData, SENSEI_EVENT_MAGIC);
    CGEventPost(kCGSessionEventTap, up);
    CFRelease(up);
    CFRelease(src);
}

static NSString* expandDynamicTokens(NSString *input) {
    if (!input || [input rangeOfString:@"{"].location == NSNotFound) {
        return input;
    }
    NSDate *now = [NSDate date];
    NSDateFormatter *df = [[NSDateFormatter alloc] init];
    
    if ([input containsString:@"{date}"]) {
        [df setDateFormat:@"yyyy-MM-dd"];
        input = [input stringByReplacingOccurrencesOfString:@"{date}" withString:[df stringFromDate:now]];
    }
    if ([input containsString:@"{time}"]) {
        [df setDateFormat:@"HH:mm:ss"];
        input = [input stringByReplacingOccurrencesOfString:@"{time}" withString:[df stringFromDate:now]];
    }
    if ([input containsString:@"{datetime}"]) {
        [df setDateFormat:@"yyyy-MM-dd HH:mm:ss"];
        input = [input stringByReplacingOccurrencesOfString:@"{datetime}" withString:[df stringFromDate:now]];
    }
    if ([input containsString:@"{year}"]) {
        [df setDateFormat:@"yyyy"];
        input = [input stringByReplacingOccurrencesOfString:@"{year}" withString:[df stringFromDate:now]];
    }
    if ([input containsString:@"{uuid}"]) {
        NSString *uuidStr = [[NSUUID UUID] UUIDString];
        input = [input stringByReplacingOccurrencesOfString:@"{uuid}" withString:uuidStr];
    }
    return input;
}

static bool isCurrentDeviceEnabled() {
    pthread_mutex_lock(&g_deviceMutex);
    if (g_deviceFilterCount == 0 || (g_lastActiveDeviceVID == 0 && g_lastActiveDevicePID == 0)) {
        pthread_mutex_unlock(&g_deviceMutex);
        return true; // enabled by default
    }
    for (int i = 0; i < g_deviceFilterCount; i++) {
        if (g_deviceFilters[i].vendorId == g_lastActiveDeviceVID &&
            g_deviceFilters[i].productId == g_lastActiveDevicePID) {
            bool en = g_deviceFilters[i].enabled;
            pthread_mutex_unlock(&g_deviceMutex);
            return en;
        }
    }
    pthread_mutex_unlock(&g_deviceMutex);
    return true;
}

static bool isCurrentAppExcluded(const char* ruleExcluded, const char* ruleTarget) {
    @autoreleasepool {
        NSRunningApplication *frontApp = [[NSWorkspace sharedWorkspace] frontmostApplication];
        if (!frontApp) return false;
        
        NSString *bundleId = frontApp.bundleIdentifier ? frontApp.bundleIdentifier : @"";
        NSString *appName = frontApp.localizedName ? frontApp.localizedName : @"";
        
        // 1. Check global exclusions
        pthread_mutex_lock(&g_exclusionsMutex);
        if (g_globalExcludedApps[0] != '\0') {
            NSString *globStr = [NSString stringWithUTF8String:g_globalExcludedApps];
            pthread_mutex_unlock(&g_exclusionsMutex);
            for (NSString *item in [globStr componentsSeparatedByString:@","]) {
                NSString *trimmed = [item stringByTrimmingCharactersInSet:[NSCharacterSet whitespaceCharacterSet]];
                if (trimmed.length > 0) {
                    if ([bundleId localizedCaseInsensitiveContainsString:trimmed] ||
                        [appName localizedCaseInsensitiveContainsString:trimmed]) {
                        return true;
                    }
                }
            }
        } else {
            pthread_mutex_unlock(&g_exclusionsMutex);
        }
        
        // 2. Check rule-specific exclusions
        if (ruleExcluded && ruleExcluded[0] != '\0') {
            NSString *exStr = [NSString stringWithUTF8String:ruleExcluded];
            for (NSString *item in [exStr componentsSeparatedByString:@","]) {
                NSString *trimmed = [item stringByTrimmingCharactersInSet:[NSCharacterSet whitespaceCharacterSet]];
                if (trimmed.length > 0) {
                    if ([bundleId localizedCaseInsensitiveContainsString:trimmed] ||
                        [appName localizedCaseInsensitiveContainsString:trimmed]) {
                        return true;
                    }
                }
            }
        }
        
        // 3. Check rule target whitelist (if specified)
        if (ruleTarget && ruleTarget[0] != '\0') {
            NSString *tgStr = [NSString stringWithUTF8String:ruleTarget];
            bool found = false;
            for (NSString *item in [tgStr componentsSeparatedByString:@","]) {
                NSString *trimmed = [item stringByTrimmingCharactersInSet:[NSCharacterSet whitespaceCharacterSet]];
                if (trimmed.length > 0) {
                    if ([bundleId localizedCaseInsensitiveContainsString:trimmed] ||
                        [appName localizedCaseInsensitiveContainsString:trimmed]) {
                        found = true;
                        break;
                    }
                }
            }
            if (!found) return true; // not in target list
        }
        
        return false;
    }
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

    // Never re-intercept our own synthetic events
    if (CGEventGetIntegerValueField(event, kCGEventSourceUserData) == SENSEI_EVENT_MAGIC) {
        return event;
    }
    
    int64_t keycode = CGEventGetIntegerValueField(event, kCGKeyboardEventKeycode);
    if (keycode < 0 || keycode >= 256) return event;
    
    // Check device filter
    if (!isCurrentDeviceEnabled()) {
        return event;
    }

    // ---------------------------------------------------------
    // Hyper Key Handling (Caps Lock remap)
    // ---------------------------------------------------------
    if (g_hyperKeyEnabled && keycode == g_hyperKeySource) {
        if (type == kCGEventFlagsChanged || type == kCGEventKeyDown || type == kCGEventKeyUp) {
            CGEventFlags flags = CGEventGetFlags(event);
            bool isAlphaActive = (flags & kCGEventFlagMaskAlphaShift) != 0;
            
            if (type == kCGEventKeyDown || (type == kCGEventFlagsChanged && !g_hyperIsDown && isAlphaActive)) {
                g_hyperIsDown = true;
                g_hyperDownTime = getTimestampMs();
                g_hyperUsedInCombination = false;
                return NULL;
            } else if (type == kCGEventKeyUp || (type == kCGEventFlagsChanged && g_hyperIsDown && !isAlphaActive)) {
                g_hyperIsDown = false;
                uint64_t elapsed = getTimestampMs() - g_hyperDownTime;
                if (!g_hyperUsedInCombination && elapsed < 350) {
                    if (g_hyperTapAction == 1) { // ESC
                        injectEscape();
                    }
                }
                return NULL;
            }
        }
        return NULL;
    }

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

    // Apply Hyper Key modifiers if held down
    if (g_hyperKeyEnabled && g_hyperIsDown) {
        eventMods |= g_hyperKeyMask;
        g_hyperUsedInCombination = true;
    }
    
    // ---------------------------------------------------------
    // Double-Tap Sequence Detection (e.g. öö -> <, çç -> >)
    // ---------------------------------------------------------
    if (g_enableSequences && eventMods == 0) {
        for (int s = 0; s < g_seqRuleCount; s++) {
            if (g_seqRules[s].enabled && g_seqRules[s].keycode == (int)keycode) {
                uint64_t nowMs = getTimestampMs();
                if (g_lastSequenceKey == (int)keycode && (nowMs - g_lastSequenceTime) <= g_seqRules[s].timeoutMs) {
                    // Double tap triggered!
                    g_lastSequenceKey = -1;
                    g_lastSequenceTime = 0;
                    
                    injectBackspace();
                    injectUnicodeString(g_seqRules[s].outputChars, g_seqRules[s].outputLen);
                    return NULL;
                } else {
                    g_lastSequenceKey = (int)keycode;
                    g_lastSequenceTime = nowMs;
                    return event;
                }
            }
        }
        g_lastSequenceKey = -1;
    } else {
        g_lastSequenceKey = -1;
    }

    // ---------------------------------------------------------
    // Hotkey Rules Matching
    // ---------------------------------------------------------
    pthread_mutex_lock(&g_rulesMutex);
    for (int i = 0; i < g_ruleCount; i++) {
        if (g_rules[i].enabled && g_rules[i].keycode == (int)keycode && g_rules[i].flags == eventMods) {
            CRule match = g_rules[i];
            pthread_mutex_unlock(&g_rulesMutex);
            
            // Check application filtering
            if (isCurrentAppExcluded(match.excludedApps, match.targetApps)) {
                return event;
            }
            
            g_interceptedKeys[keycode] = true;
            
            char outBuf[1024] = {0};
            if (match.isDynamic) {
                @autoreleasepool {
                    NSString *raw = [NSString stringWithUTF8String:match.rawOutputUtf8];
                    NSString *expanded = expandDynamicTokens(raw);
                    UniChar dynBuf[1024];
                    NSUInteger len = [expanded length];
                    if (len > 1024) len = 1024;
                    [expanded getCharacters:dynBuf range:NSMakeRange(0, len)];
                    injectUnicodeString(dynBuf, (int)len);
                    [expanded getCString:outBuf maxLength:sizeof(outBuf) encoding:NSUTF8StringEncoding];
                }
            } else {
                injectUnicodeString(match.outputChars, match.outputLen);
                CFStringRef cfOut = CFStringCreateWithCharacters(kCFAllocatorDefault, match.outputChars, match.outputLen);
                if (cfOut) {
                    CFStringGetCString(cfOut, outBuf, sizeof(outBuf), kCFStringEncodingUTF8);
                    CFRelease(cfOut);
                }
            }
            
            goOnHotkeyTriggered(match.ruleIndex, match.ruleId, outBuf);
            return NULL;
        }
    }
    pthread_mutex_unlock(&g_rulesMutex);
    
    return event;
}

// -------------------------------------------------------------
// IOHIDManager Keyboard Input Callback (Device identification)
// -------------------------------------------------------------
static IOHIDManagerRef g_hidManager = NULL;

static void hidInputCallback(void *context, IOReturn result, void *sender, IOHIDValueRef value) {
    IOHIDDeviceRef device = (IOHIDDeviceRef)sender;
    if (!device) return;
    
    CFNumberRef vendorId = (CFNumberRef)IOHIDDeviceGetProperty(device, CFSTR(kIOHIDVendorIDKey));
    CFNumberRef productId = (CFNumberRef)IOHIDDeviceGetProperty(device, CFSTR(kIOHIDProductIDKey));
    
    if (vendorId) CFNumberGetValue(vendorId, kCFNumberIntType, &g_lastActiveDeviceVID);
    if (productId) CFNumberGetValue(productId, kCFNumberIntType, &g_lastActiveDevicePID);
}

static void c_setupHIDManager() {
    if (g_hidManager) return;
    g_hidManager = IOHIDManagerCreate(kCFAllocatorDefault, kIOHIDOptionsTypeNone);
    
    CFMutableDictionaryRef matching = CFDictionaryCreateMutable(kCFAllocatorDefault, 0,
        &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
    int page = kHIDPage_GenericDesktop;
    int usage = kHIDUsage_GD_Keyboard;
    CFNumberRef pageNum = CFNumberCreate(kCFAllocatorDefault, kCFNumberIntType, &page);
    CFNumberRef usageNum = CFNumberCreate(kCFAllocatorDefault, kCFNumberIntType, &usage);
    CFDictionarySetValue(matching, CFSTR(kIOHIDDeviceUsagePageKey), pageNum);
    CFDictionarySetValue(matching, CFSTR(kIOHIDDeviceUsageKey), usageNum);
    CFRelease(pageNum);
    CFRelease(usageNum);
    
    IOHIDManagerSetDeviceMatching(g_hidManager, matching);
    CFRelease(matching);
    
    IOHIDManagerRegisterInputValueCallback(g_hidManager, hidInputCallback, NULL);
    IOHIDManagerScheduleWithRunLoop(g_hidManager, CFRunLoopGetCurrent(), kCFRunLoopCommonModes);
    IOHIDManagerOpen(g_hidManager, kIOHIDOptionsTypeNone);
}

static void* runLoopThread(void* arg) {
    g_runLoop = CFRunLoopGetCurrent();
    c_setupHIDManager();
    CFRunLoopAddSource(g_runLoop, g_runLoopSource, kCFRunLoopCommonModes);
    CGEventTapEnable(g_eventTap, true);
    g_isRunning = true;
    CFRunLoopRun();
    g_isRunning = false;
    return NULL;
}

bool c_startTap() {
    if (g_isRunning) return true;
    
    CGEventMask mask = CGEventMaskBit(kCGEventKeyDown) | CGEventMaskBit(kCGEventKeyUp) | CGEventMaskBit(kCGEventFlagsChanged);
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

void c_stopTap() {
    if (!g_isRunning) return;
    if (g_runLoop) {
        CFRunLoopStop(g_runLoop);
    }
    if (g_eventTap) {
        CGEventTapEnable(g_eventTap, false);
    }
}

void c_setPaused(bool paused) {
    g_isPaused = paused;
}

bool c_isPaused() {
    return g_isPaused;
}

bool c_isRunning() {
    return g_isRunning;
}

bool c_checkAccessibility() {
    NSDictionary *options = @{(__bridge id)kAXTrustedCheckOptionPrompt: @NO};
    return AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)options);
}

void c_promptAccessibility() {
    NSDictionary *options = @{(__bridge id)kAXTrustedCheckOptionPrompt: @YES};
    AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)options);
}

void c_clearRules() {
    pthread_mutex_lock(&g_rulesMutex);
    g_ruleCount = 0;
    pthread_mutex_unlock(&g_rulesMutex);
}

void c_setGlobalExcludedApps(const char* apps) {
    pthread_mutex_lock(&g_exclusionsMutex);
    if (apps) {
        strncpy(g_globalExcludedApps, apps, sizeof(g_globalExcludedApps) - 1);
    } else {
        g_globalExcludedApps[0] = '\0';
    }
    pthread_mutex_unlock(&g_exclusionsMutex);
}

void c_addRule(int index, const char* ruleId, uint32_t flags, int keycode, const char* outputUtf8, bool enabled, const char* targetApps, const char* excludedApps) {
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
    r->isDynamic = false;
    
    if (targetApps) {
        strncpy(r->targetApps, targetApps, sizeof(r->targetApps) - 1);
    } else {
        r->targetApps[0] = '\0';
    }
    
    if (excludedApps) {
        strncpy(r->excludedApps, excludedApps, sizeof(r->excludedApps) - 1);
    } else {
        r->excludedApps[0] = '\0';
    }
    
    if (outputUtf8) {
        strncpy(r->rawOutputUtf8, outputUtf8, sizeof(r->rawOutputUtf8) - 1);
        if (strstr(outputUtf8, "{date}") || strstr(outputUtf8, "{time}") ||
            strstr(outputUtf8, "{datetime}") || strstr(outputUtf8, "{year}") ||
            strstr(outputUtf8, "{uuid}")) {
            r->isDynamic = true;
        }
        
        CFStringRef cf = CFStringCreateWithCString(kCFAllocatorDefault, outputUtf8, kCFStringEncodingUTF8);
        if (cf) {
            CFIndex len = CFStringGetLength(cf);
            if (len > 1024) len = 1024;
            CFStringGetCharacters(cf, CFRangeMake(0, len), r->outputChars);
            r->outputLen = (int)len;
            CFRelease(cf);
        }
    }
    
    g_ruleCount++;
    pthread_mutex_unlock(&g_rulesMutex);
}

// -------------------------------------------------------------
// Sequence Rules (öö -> <)
// -------------------------------------------------------------
void c_clearSequenceRules() {
    g_seqRuleCount = 0;
}

void c_addSequenceRule(int keycode, const char* outputUtf8, int timeoutMs, bool enabled) {
    if (g_seqRuleCount >= MAX_SEQUENCES) return;
    
    CSequenceRule *sr = &g_seqRules[g_seqRuleCount];
    sr->keycode = keycode;
    sr->timeoutMs = timeoutMs > 0 ? timeoutMs : 280;
    sr->enabled = enabled;
    sr->outputLen = 0;
    
    if (outputUtf8) {
        CFStringRef cf = CFStringCreateWithCString(kCFAllocatorDefault, outputUtf8, kCFStringEncodingUTF8);
        if (cf) {
            CFIndex len = CFStringGetLength(cf);
            if (len > 64) len = 64;
            CFStringGetCharacters(cf, CFRangeMake(0, len), sr->outputChars);
            sr->outputLen = (int)len;
            CFRelease(cf);
        }
    }
    g_seqRuleCount++;
}

void c_setEnableSequences(bool enabled) {
    g_enableSequences = enabled;
}

// -------------------------------------------------------------
// Hyper Key Configuration
// -------------------------------------------------------------
void c_setHyperKey(bool enabled, int sourceKeycode, uint32_t mask, int tapAction) {
    g_hyperKeyEnabled = enabled;
    g_hyperKeySource = sourceKeycode > 0 ? sourceKeycode : 57;
    g_hyperKeyMask = mask;
    g_hyperTapAction = tapAction;
}

// -------------------------------------------------------------
// Device Filters & Hardware Listing
// -------------------------------------------------------------
void c_clearDeviceFilters() {
    pthread_mutex_lock(&g_deviceMutex);
    g_deviceFilterCount = 0;
    pthread_mutex_unlock(&g_deviceMutex);
}

void c_addDeviceFilter(int vendorId, int productId, bool enabled) {
    pthread_mutex_lock(&g_deviceMutex);
    if (g_deviceFilterCount < MAX_DEVICE_FILTERS) {
        g_deviceFilters[g_deviceFilterCount].vendorId = vendorId;
        g_deviceFilters[g_deviceFilterCount].productId = productId;
        g_deviceFilters[g_deviceFilterCount].enabled = enabled;
        g_deviceFilterCount++;
    }
    pthread_mutex_unlock(&g_deviceMutex);
}

typedef struct {
    char name[128];
    int vendorId;
    int productId;
    char transport[64];
    bool isInternal;
} CHIDDeviceInfo;

int c_getConnectedKeyboards(CHIDDeviceInfo *outDevices, int maxDevices) {
    IOHIDManagerRef mgr = IOHIDManagerCreate(kCFAllocatorDefault, kIOHIDOptionsTypeNone);
    
    CFMutableDictionaryRef matching = CFDictionaryCreateMutable(kCFAllocatorDefault, 0,
        &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
    int page = kHIDPage_GenericDesktop;
    int usage = kHIDUsage_GD_Keyboard;
    CFNumberRef pageNum = CFNumberCreate(kCFAllocatorDefault, kCFNumberIntType, &page);
    CFNumberRef usageNum = CFNumberCreate(kCFAllocatorDefault, kCFNumberIntType, &usage);
    CFDictionarySetValue(matching, CFSTR(kIOHIDDeviceUsagePageKey), pageNum);
    CFDictionarySetValue(matching, CFSTR(kIOHIDDeviceUsageKey), usageNum);
    CFRelease(pageNum);
    CFRelease(usageNum);
    
    IOHIDManagerSetDeviceMatching(mgr, matching);
    CFRelease(matching);
    IOHIDManagerOpen(mgr, kIOHIDOptionsTypeNone);
    
    CFSetRef deviceSet = IOHIDManagerCopyDevices(mgr);
    if (!deviceSet) {
        CFRelease(mgr);
        return 0;
    }
    
    CFIndex count = CFSetGetCount(deviceSet);
    const void *devices[count];
    CFSetGetValues(deviceSet, devices);
    
    int added = 0;
    for (CFIndex i = 0; i < count && added < maxDevices; i++) {
        IOHIDDeviceRef dev = (IOHIDDeviceRef)devices[i];
        
        CFStringRef product = (CFStringRef)IOHIDDeviceGetProperty(dev, CFSTR(kIOHIDProductKey));
        CFNumberRef vendorId = (CFNumberRef)IOHIDDeviceGetProperty(dev, CFSTR(kIOHIDVendorIDKey));
        CFNumberRef productId = (CFNumberRef)IOHIDDeviceGetProperty(dev, CFSTR(kIOHIDProductIDKey));
        CFStringRef transport = (CFStringRef)IOHIDDeviceGetProperty(dev, CFSTR(kIOHIDTransportKey));
        CFBooleanRef builtIn = (CFBooleanRef)IOHIDDeviceGetProperty(dev, CFSTR(kIOHIDBuiltInKey));
        
        CHIDDeviceInfo *d = &outDevices[added];
        strncpy(d->name, "Bilinmeyen Klavye", sizeof(d->name) - 1);
        if (product && CFGetTypeID(product) == CFStringGetTypeID()) {
            CFStringGetCString(product, d->name, sizeof(d->name), kCFStringEncodingUTF8);
        }
        
        strncpy(d->transport, "USB", sizeof(d->transport) - 1);
        if (transport && CFGetTypeID(transport) == CFStringGetTypeID()) {
            CFStringGetCString(transport, d->transport, sizeof(d->transport), kCFStringEncodingUTF8);
        }
        
        d->vendorId = 0;
        d->productId = 0;
        if (vendorId) CFNumberGetValue(vendorId, kCFNumberIntType, &d->vendorId);
        if (productId) CFNumberGetValue(productId, kCFNumberIntType, &d->productId);
        d->isInternal = builtIn ? CFBooleanGetValue(builtIn) : false;
        
        // Skip virtual or touchbar devices without proper products
        if (strstr(d->name, "TouchBarUserDevice") || d->vendorId == 0) {
            continue;
        }
        
        added++;
    }
    
    CFRelease(deviceSet);
    CFRelease(mgr);
    return added;
}

void c_getFrontmostApp(char* nameBuf, int nameBufLen, char* idBuf, int idBufLen) {
    @autoreleasepool {
        NSRunningApplication *frontApp = [[NSWorkspace sharedWorkspace] frontmostApplication];
        if (frontApp) {
            if (frontApp.localizedName) {
                strncpy(nameBuf, [frontApp.localizedName UTF8String], nameBufLen - 1);
            }
            if (frontApp.bundleIdentifier) {
                strncpy(idBuf, [frontApp.bundleIdentifier UTF8String], idBufLen - 1);
            }
        }
    }
}

// -------------------------------------------------------------
// macOS Status Bar (Menu Bar) Item Integration
// -------------------------------------------------------------

@interface SenseiTrayHandler : NSObject <NSMenuDelegate>
@property (strong, nonatomic) NSStatusItem *statusItem;
@property (strong, nonatomic) NSMenu *menu;
@end

static SenseiTrayHandler *g_trayHandler = nil;

@implementation SenseiTrayHandler
- (void)setupMenu {
    self.statusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSSquareStatusItemLength];
    if (self.statusItem.button) {
        self.statusItem.button.title = @"🥋";
        self.statusItem.button.toolTip = @"Keyboard Sensei — macOS Tuş Dönüştürücü";
    }
    
    self.menu = [[NSMenu alloc] init];
    self.menu.delegate = self;
    self.statusItem.menu = self.menu;
}

- (void)menuWillOpen:(NSMenu *)menu {
    [menu removeAllItems];
    
    char statusBuf[128] = {0};
    bool isPaused = false;
    bool isAutoStart = false;
    goTrayGetMenuState(statusBuf, sizeof(statusBuf), &isPaused, &isAutoStart);
    
    NSString *statusStr = [NSString stringWithUTF8String:statusBuf];
    if (!statusStr || statusStr.length == 0) statusStr = @"🥋 Keyboard Sensei";
    
    NSMenuItem *header = [[NSMenuItem alloc] initWithTitle:statusStr action:nil keyEquivalent:@""];
    [header setEnabled:NO];
    [menu addItem:header];
    
    [menu addItem:[NSMenuItem separatorItem]];
    
    NSString *pauseTitle = isPaused ? @"▶️  Devam Et" : @"⏸️  Duraklat";
    NSMenuItem *pauseItem = [[NSMenuItem alloc] initWithTitle:pauseTitle action:@selector(onTogglePause:) keyEquivalent:@"p"];
    pauseItem.target = self;
    [menu addItem:pauseItem];
    
    NSMenuItem *dashItem = [[NSMenuItem alloc] initWithTitle:@"🌐  Web Yönetim Paneli" action:@selector(onOpenDashboard:) keyEquivalent:@"o"];
    dashItem.target = self;
    [menu addItem:dashItem];
    
    [menu addItem:[NSMenuItem separatorItem]];
    
    NSMenuItem *autoItem = [[NSMenuItem alloc] initWithTitle:@"🚀  Açılışta Otomatik Başlat" action:@selector(onToggleAutoStart:) keyEquivalent:@""];
    autoItem.target = self;
    autoItem.state = isAutoStart ? NSControlStateValueOn : NSControlStateValueOff;
    [menu addItem:autoItem];
    
    [menu addItem:[NSMenuItem separatorItem]];
    
    NSMenuItem *quitItem = [[NSMenuItem alloc] initWithTitle:@"🛑  Çıkış" action:@selector(onQuit:) keyEquivalent:@"q"];
    quitItem.target = self;
    [menu addItem:quitItem];
}

- (void)onTogglePause:(id)sender {
    goTrayTogglePause();
}

- (void)onOpenDashboard:(id)sender {
    goTrayOpenDashboard();
}

- (void)onToggleAutoStart:(id)sender {
    goTrayToggleAutoStart();
}

- (void)onQuit:(id)sender {
    goTrayQuit();
}
@end

void c_setupTray() {
    @autoreleasepool {
        if (!g_trayHandler) {
            g_trayHandler = [[SenseiTrayHandler alloc] init];
            [g_trayHandler setupMenu];
        }
    }
}

void c_runCocoaLoop() {
    @autoreleasepool {
        if (![NSThread isMainThread]) {
            return;
        }
        NSApplication *app = [NSApplication sharedApplication];
        [app setActivationPolicy:NSApplicationActivationPolicyAccessory];
        c_setupTray();
        [app run];
    }
}

void c_stopCocoaLoop() {
    dispatch_async(dispatch_get_main_queue(), ^{
        [NSApp terminate:nil];
    });
}
