//go:build darwin

package cocoa

/*
#cgo darwin LDFLAGS: -framework Cocoa -framework OpenGL
#cgo darwin CFLAGS: -Wno-deprecated-declarations
#include <stdlib.h>
#include "cocoa.h"
*/
import "C"

import (
	"fmt"
)

// CocoaError describes a failure setting up the macOS overlay window.
type CocoaError struct {
	msg string
}

func (e *CocoaError) Error() string {
	return e.msg
}

// CocoaWindow is the macOS backend implementing platform.Window. It wraps a
// transparent, borderless, fullscreen NSWindow overlay with an OpenGL 4.1 core
// context, mirroring the behaviour of the Wayland layer-shell overlay.
type CocoaWindow struct {
	width, height int32
}

// NewCocoaWindow creates the overlay window and a current GL context and shows
// it immediately, satisfying platform.NewWindow.
func NewCocoaWindow() (*CocoaWindow, error) {
	if ret := C.cocoa_init(); ret != 0 {
		return nil, &CocoaError{fmt.Sprintf("failed to initialise Cocoa window (code %d)", int(ret))}
	}

	w := &CocoaWindow{}
	var width, height C.int32_t
	C.cocoa_get_dimensions(&width, &height)
	w.width = int32(width)
	w.height = int32(height)

	if w.width == 0 || w.height == 0 {
		w.width = 1920
		w.height = 1080
	}

	return w, nil
}

func (w *CocoaWindow) GetSize() (int, int) {
	var width, height C.int32_t
	C.cocoa_get_dimensions(&width, &height)
	if width > 0 && height > 0 {
		w.width = int32(width)
		w.height = int32(height)
	}
	return int(w.width), int(w.height)
}

func (w *CocoaWindow) ShouldClose() bool {
	return false
}

func (w *CocoaWindow) SwapBuffers() {
	C.cocoa_swap_buffers()
}

func (w *CocoaWindow) PollEvents() {
	C.cocoa_poll_events()
}

func (w *CocoaWindow) GetCursorPos() (float64, float64) {
	var x, y C.double
	C.cocoa_get_mouse_pos(&x, &y)
	return float64(x), float64(y)
}

func (w *CocoaWindow) GetMouseButton() bool {
	return C.cocoa_get_button_state() == 1
}

func (w *CocoaWindow) DisableInput() {
	C.cocoa_disable_input()
}

func (w *CocoaWindow) GetLastKey() (uint32, uint32, bool) {
	key := uint32(C.cocoa_get_last_key())
	state := uint32(C.cocoa_get_last_key_state())
	return key, state, key != 0
}

func (w *CocoaWindow) ClearLastKey() {
	C.cocoa_clear_last_key()
}

func (w *CocoaWindow) Destroy() {
	C.cocoa_destroy()
}
