package platform

// Window is the platform-agnostic windowing, input, and GL-surface abstraction
// consumed by the rest of Hexecute. Backends (Wayland on Linux, Cocoa on macOS)
// implement this interface, and NewWindow (defined per-OS in a build-tagged
// file) returns the appropriate backend.
type Window interface {
	GetSize() (int, int)
	ShouldClose() bool
	SwapBuffers()
	PollEvents()
	GetCursorPos() (float64, float64)
	GetMouseButton() bool
	DisableInput()
	GetLastKey() (key uint32, state uint32, ok bool)
	ClearLastKey()
	Destroy()
}
