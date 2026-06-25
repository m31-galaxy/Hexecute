//go:build linux

package platform

import "github.com/m31-galaxy/Hexecute/pkg/wayland"

// Compile-time check that the Wayland backend satisfies the Window interface.
var _ Window = (*wayland.WaylandWindow)(nil)

// NewWindow constructs the Wayland-backed window on Linux.
func NewWindow() (Window, error) {
	return wayland.NewWaylandWindow()
}
