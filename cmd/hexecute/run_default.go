//go:build !darwin

package main

import (
	"github.com/m31-galaxy/Hexecute/internal/config"
	"github.com/m31-galaxy/Hexecute/internal/models"
)

// runMain runs a single overlay session per launch, matching the compositor
// keybind model used on Linux/Wayland (each hotkey press spawns a fresh
// process). Settings are unused here.
func runMain(app *models.App, _ *config.Settings) {
	runOnce(app)
}
