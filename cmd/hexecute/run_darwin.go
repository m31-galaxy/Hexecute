//go:build darwin

package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/m31-galaxy/Hexecute/internal/config"
	"github.com/m31-galaxy/Hexecute/internal/models"
	"github.com/m31-galaxy/Hexecute/pkg/cocoa"
)

const defaultHotkey = "cmd+option+space"

// Carbon modifier masks (Events.h), as expected by RegisterEventHotKey.
const (
	cmdKey     = 0x0100
	shiftKey   = 0x0200
	optionKey  = 0x0800
	controlKey = 0x1000
)

var modifierAliases = map[string]uint32{
	"cmd": cmdKey, "command": cmdKey, "super": cmdKey, "meta": cmdKey, "win": cmdKey,
	"opt": optionKey, "option": optionKey, "alt": optionKey,
	"ctrl": controlKey, "control": controlKey,
	"shift": shiftKey,
}

// macOS virtual key codes (Carbon kVK_ANSI_* / kVK_*).
var keyCodes = map[string]uint32{
	"space": 49, "return": 36, "enter": 36, "tab": 48, "escape": 53, "esc": 53,
	"a": 0, "b": 11, "c": 8, "d": 2, "e": 14, "f": 3, "g": 5, "h": 4, "i": 34,
	"j": 38, "k": 40, "l": 37, "m": 46, "n": 45, "o": 31, "p": 35, "q": 12,
	"r": 15, "s": 1, "t": 17, "u": 32, "v": 9, "w": 13, "x": 7, "y": 16, "z": 6,
	"0": 29, "1": 18, "2": 19, "3": 20, "4": 21, "5": 23, "6": 22, "7": 26,
	"8": 28, "9": 25,
	"f1": 122, "f2": 120, "f3": 99, "f4": 118, "f5": 96, "f6": 97, "f7": 98,
	"f8": 100, "f9": 101, "f10": 109, "f11": 103, "f12": 111,
}

// parseHotkey turns a string like "cmd+option+space" into a macOS virtual key
// code and a Carbon modifier mask.
func parseHotkey(spec string) (keyCode uint32, modifiers uint32, err error) {
	var keyName string
	for _, part := range strings.Split(spec, "+") {
		token := strings.ToLower(strings.TrimSpace(part))
		if token == "" {
			continue
		}
		if mod, ok := modifierAliases[token]; ok {
			modifiers |= mod
			continue
		}
		if keyName != "" {
			return 0, 0, fmt.Errorf("more than one non-modifier key (%q and %q)", keyName, token)
		}
		keyName = token
	}
	if keyName == "" {
		return 0, 0, fmt.Errorf("no key specified")
	}
	code, ok := keyCodes[keyName]
	if !ok {
		return 0, 0, fmt.Errorf("unknown key %q", keyName)
	}
	if modifiers == 0 {
		return 0, 0, fmt.Errorf("at least one modifier (cmd/option/ctrl/shift) is required")
	}
	return code, modifiers, nil
}

// runMain runs Hexecute as a resident agent on macOS: it keeps a warm GL context
// and a global hot key, showing/hiding the overlay per cast rather than
// relaunching, so there is no per-cast startup cost. Relaunching the app
// (open -a / double-click) reopens this instance and casts too, so a launch is
// never a no-op; the hot key remains the intended main flow.
func runMain(app *models.App, _ *config.Settings) {
	// The hot key is stored in NSUserDefaults (defaults domain app.hexecute), not
	// the cross-platform settings file. defaultHotkey is seeded as the default.
	spec := cocoa.HotkeyString(defaultHotkey)
	keyCode, modifiers, err := parseHotkey(spec)
	if err != nil {
		log.Printf("Invalid hotkey %q (%v); using default %q", spec, err, defaultHotkey)
		spec = defaultHotkey
		keyCode, modifiers, _ = parseHotkey(defaultHotkey)
	}

	window, err := cocoa.NewHiddenWindow()
	if err != nil {
		log.Fatal("Failed to create window:", err)
	}
	defer window.Destroy()

	if err := initGLAndWarm(app, window); err != nil {
		log.Fatal("Failed to initialize OpenGL:", err)
	}

	if err := window.RegisterHotkey(keyCode, modifiers); err != nil {
		log.Fatal("Failed to register global hotkey:", err)
	}
	log.Printf("Hexecute is running; press %s (or relaunch the app) to cast a gesture.", spec)

	for {
		window.WaitForShow()
		window.Show()
		resetSession(app, window)
		runSession(app, window)
		window.Hide()
	}
}
