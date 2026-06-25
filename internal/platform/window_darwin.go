//go:build darwin

package platform

import "github.com/m31-galaxy/Hexecute/pkg/cocoa"

// Compile-time check that the Cocoa backend satisfies the Window interface.
var _ Window = (*cocoa.CocoaWindow)(nil)

// NewWindow constructs the Cocoa-backed window on macOS.
func NewWindow() (Window, error) {
	return cocoa.NewCocoaWindow()
}
