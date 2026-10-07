//go:build darwin

#import <ApplicationServices/ApplicationServices.h>
#import <CoreGraphics/CoreGraphics.h>
#import <Foundation/Foundation.h>
#import <AppKit/AppKit.h>
#import <pthread.h>
#import <stdbool.h>
#import <stdint.h>

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

static char g_globalExcludedApps[512] = {0};
static pthread_mutex_t g_exclusionsMutex = PTHREAD_MUTEX_INITIALIZER;

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

static void* runLoopThread(void* arg) {
    g_runLoop = CFRunLoopGetCurrent();
    CFRunLoopAddSource(g_runLoop, g_runLoopSource, kCFRunLoopCommonModes);
    CGEventTapEnable(g_eventTap, true);
    g_isRunning = true;
    CFRunLoopRun();
    g_isRunning = false;
    return NULL;
}

bool c_startTap() {
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
