# Hexecute

A gesture-based launcher for Wayland and macOS. Launch apps by casting spells! 🪄

![Demo GIF](.github/assets/demo.gif)

## Installation

### Nix / NixOS

If you're a lucky [Nix](https://nixos.org/) user, you can:

**Try it out without installing:**
```bash
nix run github:m31-galaxy/Hexecute
```

**Install to your profile:**
```bash
nix profile install github:m31-galaxy/Hexecute
```

**Add to your NixOS configuration:**
```nix
# flake.nix
{
  inputs.hexecute.url = "github:m31-galaxy/Hexecute";
}
```
```nix
# configuration.nix
{
  environment.systemPackages = with pkgs; [
    inputs.hexecute.packages.${pkgs.system}.default
  ];
}
```

### Executable download

Download the latest version from the [release page](https://github.com/m31-galaxy/Hexecute/releases/latest), and place it somewhere in your `$PATH`.

**Don't forget to rename the downloaded binary to `hexecute` and make it executable:**
```bash
mv hexecute-1.2.3-blah hexecute
chmod +x hexecute
```

### Build from Source

Clone the repository:
```bash
git clone https://github.com/m31-galaxy/Hexecute
cd Hexecute
```

If you have [Nix](https://nixos.org/) installed, simply run `nix build`. This works on both Linux and macOS.

Otherwise:

**On Linux**, make sure you have Go (and all dependent Wayland (and X11!?) libs) installed, then run:
```bash
mkdir -p bin
go build -o bin ./...
./bin/hexecute
```

**On macOS**, make sure you have Go and the Xcode command line tools (`xcode-select --install`) installed, then run:
```bash
mkdir -p bin
go build -o bin ./...
./bin/hexecute
```
The macOS build uses a native Cocoa overlay window and the system OpenGL framework — no Wayland or X11 libraries are required.

For a proper, double-clickable **`Hexecute.app`** bundle (so Finder treats it as an Application instead of a Unix executable, and launching it doesn't open Terminal), build with Nix — `nix build` produces it at `result/Applications/Hexecute.app`. The bundle metadata lives in [`macos/Info.plist`](macos/Info.plist).

Pre-built `.app` bundles are attached to each macOS CI run as the `hexecute-macos-latest` artifact (a `.tar.gz`, which preserves the bundle's executable bit). After downloading:
```bash
tar xzf hexecute-macos.tar.gz
xattr -dr com.apple.quarantine Hexecute.app   # clear the download quarantine
open Hexecute.app                              # or double-click it in Finder
```
The app is unsigned, so the first launch may require right-click → **Open** (or the `xattr` command above).

## Usage

### Setting a Keybind
The recommended way to use Hexecute is to bind it to a keyboard shortcut in your compositor.

Listed below are some examples for popular compositors using the `SUPER` + `SPACE` keybind.

#### Hyprland

If you're using Hyprland, add the following line to your `~/.config/hypr/hyprland.conf`:

```
bind = SUPER, SPACE, exec, hexecute
```

#### Sway

If you're using Sway, add the following line to your `~/.config/sway/config`:

```
bindsym $mod+space exec hexecute
```

#### macOS

Hexecute supports two launch modes on macOS:

- **Manual launch** — double-click `Hexecute.app` (or `open -a Hexecute`, or run the binary with no arguments) to **draw a gesture immediately**: the overlay appears, you cast once, and it dismisses.
- **Resident agent** — run it with `--background` to stay alive with a warm GL context and register a global hotkey, so casting is instant and needs no third-party hotkey tool. This is the mode the autostart LaunchAgent uses.

The global hotkey is the intended way to cast with the resident agent. The default is **⌘ + ⌥ + Space** (Cmd+Option+Space): press it to show the overlay, then draw a gesture or press Esc to dismiss. The hotkey is stored in the native macOS preferences (the `defaults` system, domain `app.hexecute`), not the cross-platform settings file. Change it with:

```bash
defaults write app.hexecute hotkey "cmd+option+space"
```
Then restart the resident agent so it re-reads the value (`launchctl unload`/`load` the LaunchAgent, or quit and relaunch). Modifiers are `cmd`, `option` (alias `alt`), `ctrl`, and `shift`; the key can be a letter, digit, `space`, `return`, `tab`, or `f1`–`f12`. At least one modifier is required. Registering the hotkey needs **no Accessibility permission** (it uses the Carbon hot-key API). Pick a combination that isn't already a system shortcut (e.g. ⌘⌥Space is macOS's Finder search).

To run the resident agent at login, install the bundled LaunchAgent ([`macos/app.hexecute.plist`](macos/app.hexecute.plist)) — which launches it with `--background` — after copying `Hexecute.app` into `/Applications`:

```bash
cp macos/app.hexecute.plist ~/Library/LaunchAgents/
launchctl load ~/Library/LaunchAgents/app.hexecute.plist   # start now + at every login
```
To stop it: `launchctl unload ~/Library/LaunchAgents/app.hexecute.plist`.

> Note: macOS keeps a single instance per app, so once the resident agent is running, double-clicking `Hexecute.app` (or `open -a Hexecute`) reopens that instance rather than starting a second one — Hexecute treats this as a cast request, so a launch always shows the overlay whether or not the agent is already running. The hotkey is still the quickest way to cast.

> Note: depending on your macOS version, drawing the overlay over other applications may require granting Hexecute **Screen Recording** and/or **Accessibility** permission under System Settings → Privacy & Security. Using the `.app` bundle (rather than a bare binary) gives Hexecute a stable identity so these permissions persist across launches.

### Learning a Gesture

To configure a gesture to launch an application, run `hexecute --learn [command]` in a terminal. Hexecute should launch - simply draw your chosen gesture **3 times** and it will be mapped to the command.

> On macOS, the CLI commands (`--learn`, `--list`, `--remove`) need the binary inside the bundle, since they print progress to the terminal: run `/Applications/Hexecute.app/Contents/MacOS/hexecute --learn [command]`. Double-clicking or `open`ing the `.app` is for normal gesture-casting use.

![Gesture learning demo](assets/hexecute-learn.gif)

### Managing Gestures

To view all your configured gestures, run `hexecute --list` in a terminal.

To delete a previously assigned gesture, use the `hexecute --remove [gesture]` command.

All gestures are saved in the `~/.config/hexecute/gestures.json` file. This file can be manually shared, edited, backed up, or swapped.
