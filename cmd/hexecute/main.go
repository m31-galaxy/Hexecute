package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/m31-galaxy/Hexecute/internal/config"
	"github.com/m31-galaxy/Hexecute/internal/draw"
	"github.com/m31-galaxy/Hexecute/internal/execute"
	gestures "github.com/m31-galaxy/Hexecute/internal/gesture"
	"github.com/m31-galaxy/Hexecute/internal/models"
	"github.com/m31-galaxy/Hexecute/internal/opengl"
	"github.com/m31-galaxy/Hexecute/internal/platform"
	"github.com/m31-galaxy/Hexecute/internal/spawn"
	"github.com/m31-galaxy/Hexecute/internal/stroke"
	"github.com/m31-galaxy/Hexecute/internal/update"
)

func init() {
	runtime.LockOSThread()
}

type App struct {
	*models.App
}

func main() {
	learnCommand := flag.String("learn", "", "Learn a new gesture for the specified command")
	listGestures := flag.Bool("list", false, "List all registered gestures")
	removeGesture := flag.String("remove", "", "Remove a gesture by command name")
	background := flag.Bool("background", false, "macOS: run as a resident agent that shows the overlay on the global hotkey, instead of drawing a gesture immediately (used by the autostart LaunchAgent)")
	flag.Parse()

	if flag.NArg() > 0 {
		log.Fatalf("Unknown arguments: %v", flag.Args())
	}

	if *listGestures {
		gestures, err := gestures.LoadGestures()
		if err != nil {
			log.Fatal("Failed to load gestures:", err)
		}
		if len(gestures) == 0 {
			println("No gestures registered")
		} else {
			println("Registered gestures:")
			for _, g := range gestures {
				println("  ", g.Command)
			}
		}
		return
	}

	if *removeGesture != "" {
		gestures, err := gestures.LoadGestures()
		if err != nil {
			log.Fatal("Failed to load gestures:", err)
		}

		found := false
		for i, g := range gestures {
			if g.Command == *removeGesture {
				gestures = append(gestures[:i], gestures[i+1:]...)
				found = true
				break
			}
		}

		if !found {
			log.Fatalf("Gesture not found: %s", *removeGesture)
		}

		configFile, err := config.GetPath()
		if err != nil {
			log.Fatal("Failed to get config path:", err)
		}

		data, err := json.Marshal(gestures)
		if err != nil {
			log.Fatal("Failed to marshal gestures:", err)
		}

		if err := os.WriteFile(configFile, data, 0644); err != nil {
			log.Fatal("Failed to save gestures:", err)
		}

		println("Removed gesture:", *removeGesture)
		return
	}

	settings, err := config.LoadSettings()
	if err != nil {
		log.Fatal("Failed to load settings:", err)
	}

	app := &models.App{
		StartTime: time.Now(),
		Settings:  settings,
	}

	if *learnCommand != "" {
		app.LearnMode = true
		app.LearnCommand = *learnCommand
		log.Printf("Learn mode: Draw the gesture 3 times for command '%s'", *learnCommand)
		runOnce(app)
		return
	}

	loaded, err := gestures.LoadGestures()
	if err != nil {
		log.Fatal("Failed to load gestures:", err)
	}
	app.SavedGestures = loaded
	log.Printf("Loaded %d gesture(s)", len(loaded))

	if *background {
		// Resident background agent: wait for the global hotkey (macOS). On
		// Linux there is no resident mode, so runMain falls back to one session.
		runMain(app, settings)
	} else {
		// Manual launch (double-click / open / terminal): draw a gesture now.
		runOnce(app)
	}
}

// initGLAndWarm compiles shaders and pumps a few clear frames so the first real
// frame is ready.
func initGLAndWarm(app *models.App, window platform.Window) error {
	o := opengl.New(app)
	if err := o.InitGL(); err != nil {
		return err
	}

	gl.ClearColor(0, 0, 0, 0)
	for range 5 {
		window.PollEvents()
		gl.Clear(gl.COLOR_BUFFER_BIT)
		window.SwapBuffers()
	}
	return nil
}

// resetSession clears per-cast state so a reused window (resident mode) starts a
// fresh gesture each time the overlay is shown.
func resetSession(app *models.App, window platform.Window) {
	app.StartTime = time.Now()
	app.IsExiting = false
	app.IsDrawing = false
	app.Points = nil
	app.Particles = nil
	app.CursorVelocity = 0
	app.SmoothVelocity = 0
	app.SmoothRotation = 0
	app.SmoothDrawing = 0

	x, y := window.GetCursorPos()
	app.LastCursorX = float32(x)
	app.LastCursorY = float32(y)
}

// runOnce creates a window, runs a single gesture session, and tears it down.
// Used for `--learn` (all platforms) and the per-launch path on non-macOS.
func runOnce(app *models.App) {
	window, err := platform.NewWindow()
	if err != nil {
		log.Fatal("Failed to create window:", err)
	}
	defer window.Destroy()

	if err := initGLAndWarm(app, window); err != nil {
		log.Fatal("Failed to initialize OpenGL:", err)
	}

	x, y := window.GetCursorPos()
	app.LastCursorX = float32(x)
	app.LastCursorY = float32(y)

	runSession(app, window)
}

// runSession runs the gesture-capture loop on an already-shown window, returning
// when the cast ends (gesture executed, Esc, or learn complete).
func runSession(app *models.App, window platform.Window) {
	lastTime := time.Now()
	var wasPressed bool

	for !window.ShouldClose() {
		now := time.Now()
		dt := float32(now.Sub(lastTime).Seconds())
		lastTime = now

		window.PollEvents()
		update := update.New(app)
		update.UpdateCursor(window)

		if key, state, hasKey := window.GetLastKey(); hasKey {
			if state == 1 && key == 0xff1b {
				if !app.IsExiting {
					if app.IsDrawing || len(app.Points) > 0 {
						log.Println("Esc key pressed, aborting gesture")
					} else {
						log.Println("Esc key pressed, exiting")
					}
					app.IsExiting = true
					app.ExitStartTime = time.Now()
					app.Points = nil
					window.DisableInput()
					x, y := window.GetCursorPos()
					spawn := spawn.New(app)
					spawn.SpawnExitWisps(float32(x), float32(y))
				}
			}
			window.ClearLastKey()
		}

		if app.IsExiting {
			if time.Since(app.ExitStartTime).Seconds() > 0.8 {
				break
			}
		}
		isPressed := window.GetMouseButton()
		if isPressed && !wasPressed {
			app.IsDrawing = true
			log.Println("Gesture started")
		} else if !isPressed && wasPressed {
			app.IsDrawing = false

			if app.LearnMode && len(app.Points) > 0 {
				log.Println("Gesture completed")
				processed := stroke.ProcessStroke(app.Points)
				app.LearnGestures = append(app.LearnGestures, processed)
				app.LearnCount++
				log.Printf("Captured gesture %d/3", app.LearnCount)

				app.Points = nil

				if app.LearnCount >= 3 {
					if err := gestures.SaveGesture(app.LearnCommand, app.LearnGestures); err != nil {
						log.Fatal("Failed to save gesture:", err)
					}
					log.Printf("Gesture saved for command: %s", app.LearnCommand)

					app.IsExiting = true
					app.ExitStartTime = time.Now()
					window.DisableInput()
					x, y := window.GetCursorPos()
					spawn := spawn.New(app)
					spawn.SpawnExitWisps(float32(x), float32(y))
				}
			} else if !app.LearnMode && !app.IsExiting && len(app.Points) > 0 {
				log.Println("Gesture completed")
				x, y := window.GetCursorPos()
				exec := execute.New(app)
				exec.RecognizeAndExecute(window, float32(x), float32(y))
				app.Points = nil
			}
		}
		wasPressed = isPressed

		if app.IsDrawing {
			x, y := window.GetCursorPos()
			gesture := gestures.New(app)
			gesture.AddPoint(float32(x), float32(y))

			spawn := spawn.New(app)
			spawn.SpawnCursorSparkles(float32(x), float32(y))
		}

		update.UpdateParticles(dt)
		drawer := draw.New(app)
		drawer.Draw(window)
		window.SwapBuffers()
	}
}
