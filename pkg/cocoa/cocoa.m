#import <Cocoa/Cocoa.h>
#import <OpenGL/OpenGL.h>
#import <OpenGL/gl3.h>
#import <Carbon/Carbon.h>
#include "cocoa.h"

// XKB keysym for Escape. main.go compares the reported key against this value,
// so the Cocoa backend must translate macOS key codes into the same keysyms
// the Wayland backend produces.
#define XKB_KEY_Escape 0xff1b

static NSWindow *g_window = nil;
static NSOpenGLContext *g_context = nil;
static NSView *g_view = nil;

static double g_mouse_x = 0.0;
static double g_mouse_y = 0.0;
static int g_button_state = 0;
static uint32_t g_last_key = 0;
static uint32_t g_last_key_state = 0;
static int g_input_disabled = 0;

// Global hotkey state (resident mode).
static volatile int g_hotkey_pressed = 0;
static EventHotKeyRef g_hotkey_ref = NULL;
static EventHandlerRef g_hotkey_handler = NULL;

// Borderless windows cannot become key/main by default, but the overlay needs
// keyboard (Esc) and mouse-moved events, so override these.
@interface HexWindow : NSWindow
@end

@implementation HexWindow
- (BOOL)canBecomeKeyWindow {
    return YES;
}
- (BOOL)canBecomeMainWindow {
    return YES;
}
@end

// Convert a macOS virtual key code to an XKB keysym. Only the keys the app
// reacts to need mapping; everything else reports 0 (no key).
static uint32_t map_keycode(unsigned short keyCode) {
    switch (keyCode) {
        case 53:
            return XKB_KEY_Escape;
        default:
            return 0;
    }
}

// Convert an event's window-local location (points, bottom-left origin) into
// logical points with a top-left origin. The app works in logical points (as
// the Wayland backend does), so the cursor must not be scaled by the backing
// factor — only the GL viewport runs at backing resolution.
static void update_mouse_from_event(NSEvent *event) {
    if (!g_view) {
        return;
    }
    NSPoint p = [event locationInWindow];
    NSRect bounds = [g_view bounds];
    g_mouse_x = p.x;
    g_mouse_y = bounds.size.height - p.y;
}

// Seed the cursor position from the current global mouse location (logical
// points, top-left origin). Used when (re)showing the overlay so the first
// frame is correct before any motion event arrives.
static void seed_mouse_global(void) {
    NSRect screenFrame = [[NSScreen mainScreen] frame];
    NSPoint m = [NSEvent mouseLocation];
    g_mouse_x = m.x - screenFrame.origin.x;
    g_mouse_y = screenFrame.size.height - (m.y - screenFrame.origin.y);
}

int cocoa_init(void) {
    @autoreleasepool {
        [NSApplication sharedApplication];
        // Accessory: foreground-capable overlay without a Dock icon.
        [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];

        NSScreen *screen = [NSScreen mainScreen];
        if (!screen) {
            return 1;
        }
        NSRect frame = [screen frame];

        g_window = [[HexWindow alloc] initWithContentRect:frame
                                               styleMask:NSWindowStyleMaskBorderless
                                                 backing:NSBackingStoreBuffered
                                                   defer:NO];
        if (!g_window) {
            return 2;
        }

        [g_window setOpaque:NO];
        [g_window setBackgroundColor:[NSColor clearColor]];
        [g_window setLevel:NSStatusWindowLevel];
        [g_window setHasShadow:NO];
        [g_window setAcceptsMouseMovedEvents:YES];
        [g_window setCollectionBehavior:NSWindowCollectionBehaviorCanJoinAllSpaces |
                                        NSWindowCollectionBehaviorFullScreenAuxiliary];

        g_view = [[NSView alloc] initWithFrame:frame];
        // Render in a logical-point coordinate space: keep the drawable sized in
        // points rather than backing pixels. The shaders compare gl_FragCoord
        // against the points-based resolution and cursor uniforms (e.g. the
        // background cursor light), so a backing-pixel drawable offsets anything
        // that reads gl_FragCoord by the Retina scale factor.
        [g_view setWantsBestResolutionOpenGLSurface:NO];
        [g_window setContentView:g_view];

        NSOpenGLPixelFormatAttribute attrs[] = {
            NSOpenGLPFAOpenGLProfile, NSOpenGLProfileVersion4_1Core,
            NSOpenGLPFAColorSize, 24,
            NSOpenGLPFAAlphaSize, 8,
            NSOpenGLPFADoubleBuffer,
            NSOpenGLPFAAccelerated,
            0
        };
        NSOpenGLPixelFormat *pf = [[NSOpenGLPixelFormat alloc] initWithAttributes:attrs];
        if (!pf) {
            return 3;
        }

        g_context = [[NSOpenGLContext alloc] initWithFormat:pf shareContext:nil];
        if (!g_context) {
            return 4;
        }

        [g_context setView:g_view];

        // Make the GL surface itself transparent so the overlay composites over
        // the desktop (the analogue of the Wayland alpha overlay).
        GLint opacity = 0;
        [g_context setValues:&opacity forParameter:NSOpenGLContextParameterSurfaceOpacity];

        [g_context makeCurrentContext];

        // The window is created hidden; cocoa_show() orders it front and starts
        // capturing input. This lets the resident agent keep a warm GL context
        // and only present the overlay on demand. Seed the cursor and viewport
        // so OpenGL initialisation (which runs before the first show) is valid.
        seed_mouse_global();

        [g_context update];
        NSRect viewport = [g_view bounds];
        glViewport(0, 0, (GLsizei)viewport.size.width, (GLsizei)viewport.size.height);

        return 0;
    }
}

// cocoa_show orders the overlay front, activates it, hides the system cursor,
// and (re)enables input capture. Safe to call repeatedly across casts.
void cocoa_show(void) {
    @autoreleasepool {
        g_input_disabled = 0;
        g_button_state = 0;
        g_last_key = 0;
        g_last_key_state = 0;
        g_hotkey_pressed = 0;

        if (g_window) {
            [g_window setIgnoresMouseEvents:NO];
            [g_window makeKeyAndOrderFront:nil];
            [NSApp activateIgnoringOtherApps:YES];
            // Hide the system cursor while the overlay is active. The Wayland
            // backend achieves the same by setting a NULL pointer cursor.
            [NSCursor hide];
        }

        [g_context makeCurrentContext];
        [g_context update];
        seed_mouse_global();
        NSRect viewport = [g_view bounds];
        glViewport(0, 0, (GLsizei)viewport.size.width, (GLsizei)viewport.size.height);
    }
}

// cocoa_hide orders the overlay out, restores the cursor, and yields activation
// so focus returns to the app beneath (or the one a matched command launches).
void cocoa_hide(void) {
    @autoreleasepool {
        [NSCursor unhide];
        if (g_window) {
            [g_window orderOut:nil];
        }
        [NSApp deactivate];
        g_hotkey_pressed = 0;
    }
}

// Carbon hot-key handler: flag the press and wake the event loop so a blocking
// cocoa_wait_for_hotkey() returns promptly.
static OSStatus hotkey_handler(EventHandlerCallRef next, EventRef event, void *userData) {
    (void)next;
    (void)event;
    (void)userData;
    g_hotkey_pressed = 1;
    @autoreleasepool {
        NSEvent *wake = [NSEvent otherEventWithType:NSEventTypeApplicationDefined
                                           location:NSMakePoint(0, 0)
                                      modifierFlags:0
                                          timestamp:0
                                       windowNumber:0
                                            context:nil
                                            subtype:0
                                              data1:0
                                              data2:0];
        if (wake) {
            [NSApp postEvent:wake atStart:YES];
        }
    }
    return noErr;
}

// cocoa_register_hotkey installs (or replaces) a system-wide hot key using the
// Carbon API, which works for background agents and needs no Accessibility
// permission. keyCode is a macOS virtual key code; modifiers are Carbon masks
// (cmdKey, optionKey, controlKey, shiftKey). Returns 0 on success.
int cocoa_register_hotkey(uint32_t keyCode, uint32_t modifiers) {
    @autoreleasepool {
        EventTypeSpec spec;
        spec.eventClass = kEventClassKeyboard;
        spec.eventKind = kEventHotKeyPressed;

        if (!g_hotkey_handler) {
            if (InstallApplicationEventHandler(NewEventHandlerUPP(hotkey_handler), 1, &spec,
                                               NULL, &g_hotkey_handler) != noErr) {
                return 1;
            }
        }

        if (g_hotkey_ref) {
            UnregisterEventHotKey(g_hotkey_ref);
            g_hotkey_ref = NULL;
        }

        EventHotKeyID hkID;
        hkID.signature = 'hexe';
        hkID.id = 1;

        if (RegisterEventHotKey(keyCode, modifiers, hkID, GetApplicationEventTarget(), 0,
                                &g_hotkey_ref) != noErr) {
            return 2;
        }
        return 0;
    }
}

// cocoa_wait_for_hotkey blocks (pumping the event loop, so the hot key and other
// events are dispatched) until the registered hot key fires.
void cocoa_wait_for_hotkey(void) {
    while (!g_hotkey_pressed) {
        @autoreleasepool {
            NSEvent *event = [NSApp nextEventMatchingMask:NSEventMaskAny
                                                untilDate:[NSDate distantFuture]
                                                   inMode:NSDefaultRunLoopMode
                                                  dequeue:YES];
            if (event) {
                [NSApp sendEvent:event];
            }
        }
    }
    g_hotkey_pressed = 0;
}

void cocoa_get_dimensions(int32_t *width, int32_t *height) {
    if (!g_view) {
        *width = 0;
        *height = 0;
        return;
    }
    NSRect bounds = [g_view bounds];
    *width = (int32_t)bounds.size.width;
    *height = (int32_t)bounds.size.height;
}

void cocoa_make_current(void) {
    [g_context makeCurrentContext];
}

void cocoa_swap_buffers(void) {
    [g_context flushBuffer];
}

void cocoa_poll_events(void) {
    @autoreleasepool {
        NSEvent *event;
        while ((event = [NSApp nextEventMatchingMask:NSEventMaskAny
                                           untilDate:[NSDate distantPast]
                                              inMode:NSDefaultRunLoopMode
                                             dequeue:YES])) {
            if (!g_input_disabled) {
                switch ([event type]) {
                    case NSEventTypeMouseMoved:
                    case NSEventTypeLeftMouseDragged:
                        update_mouse_from_event(event);
                        break;
                    case NSEventTypeLeftMouseDown:
                        g_button_state = 1;
                        update_mouse_from_event(event);
                        break;
                    case NSEventTypeLeftMouseUp:
                        g_button_state = 0;
                        update_mouse_from_event(event);
                        break;
                    case NSEventTypeKeyDown: {
                        uint32_t k = map_keycode([event keyCode]);
                        if (k != 0) {
                            g_last_key = k;
                            g_last_key_state = 1;
                        }
                        break;
                    }
                    case NSEventTypeKeyUp: {
                        uint32_t k = map_keycode([event keyCode]);
                        if (k != 0) {
                            g_last_key = k;
                            g_last_key_state = 0;
                        }
                        break;
                    }
                    default:
                        break;
                }
            }
            [NSApp sendEvent:event];
        }
    }
}

void cocoa_get_mouse_pos(double *x, double *y) {
    *x = g_mouse_x;
    *y = g_mouse_y;
}

int cocoa_get_button_state(void) {
    return g_button_state;
}

void cocoa_disable_input(void) {
    g_input_disabled = 1;
    g_button_state = 0;
    if (g_window) {
        [g_window setIgnoresMouseEvents:YES];
    }
}

uint32_t cocoa_get_last_key(void) {
    return g_last_key;
}

uint32_t cocoa_get_last_key_state(void) {
    return g_last_key_state;
}

void cocoa_clear_last_key(void) {
    g_last_key = 0;
    g_last_key_state = 0;
}

void cocoa_destroy(void) {
    [NSCursor unhide];
    if (g_context) {
        [g_context clearDrawable];
        g_context = nil;
    }
    if (g_window) {
        [g_window close];
        g_window = nil;
    }
    g_view = nil;
}
