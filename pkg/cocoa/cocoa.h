#ifndef HEXECUTE_COCOA_H
#define HEXECUTE_COCOA_H

#include <stdint.h>

// cocoa_init creates the shared application, a transparent borderless
// fullscreen overlay window (initially hidden), and a current OpenGL 4.1 core
// context. Returns 0 on success, non-zero on failure.
int cocoa_init(void);

// cocoa_show orders the overlay front, activates it, hides the system cursor,
// and (re)enables input capture.
void cocoa_show(void);

// cocoa_hide orders the overlay out, restores the cursor, and yields activation.
void cocoa_hide(void);

// cocoa_register_hotkey installs a system-wide hot key (Carbon API; no
// Accessibility permission needed). keyCode is a macOS virtual key code;
// modifiers are Carbon masks (cmdKey/optionKey/controlKey/shiftKey).
// Returns 0 on success.
int cocoa_register_hotkey(uint32_t keyCode, uint32_t modifiers);

// cocoa_setup_menu_bar adds a status-bar (menu bar) item for the resident agent,
// with a menu to cast on demand or quit. Idempotent; must run on the main thread.
void cocoa_setup_menu_bar(void);

// cocoa_wait_for_show blocks, pumping the event loop, until a show is requested
// by the hot key or by a relaunch (reopen Apple event).
void cocoa_wait_for_show(void);

// cocoa_get_hotkey returns the hot-key spec stored in NSUserDefaults (defaults
// domain app.hexecute), seeding fallback as the registered default. Returns a
// heap-allocated string the caller must free, or NULL if unset and fallback is
// NULL.
const char *cocoa_get_hotkey(const char *fallback);

// cocoa_get_dimensions reports the overlay size in logical points, which is the
// single coordinate space used throughout (drawable, viewport, gl_FragCoord,
// cursor), matching the Wayland backend.
void cocoa_get_dimensions(int32_t *width, int32_t *height);

// cocoa_swap_buffers presents the current frame.
void cocoa_swap_buffers(void);

// cocoa_poll_events drains and dispatches pending Cocoa events, updating the
// cached mouse position, button state, and last key.
void cocoa_poll_events(void);

// cocoa_get_mouse_pos reports the cursor position in logical points with a
// top-left origin (matching the Wayland backend's convention).
void cocoa_get_mouse_pos(double *x, double *y);

// cocoa_get_button_state returns 1 while the left mouse button is held.
int cocoa_get_button_state(void);

// cocoa_disable_input makes the overlay ignore further input (used during the
// exit animation).
void cocoa_disable_input(void);

// cocoa_get_last_key returns the last key as an XKB keysym (0 if none).
uint32_t cocoa_get_last_key(void);

// cocoa_get_last_key_state returns 1 for press, 0 for release.
uint32_t cocoa_get_last_key_state(void);

// cocoa_clear_last_key clears the cached last key.
void cocoa_clear_last_key(void);

// cocoa_destroy tears down the GL context and window.
void cocoa_destroy(void);

#endif
