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

Hexecute is launched on demand rather than running resident, so bind it to a global hotkey using a tool of your choice — for example [skhd](https://github.com/koekeishiya/skhd), [Raycast](https://www.raycast.com/), or an Automator Quick Action assigned a keyboard shortcut. An example `~/.skhdrc` entry using `cmd` + `space` (pick a combination not already taken by Spotlight):

```
cmd - space : open -a /Applications/Hexecute.app
```

> Note: depending on your macOS version, drawing the overlay over other applications may require granting Hexecute **Screen Recording** and/or **Accessibility** permission under System Settings → Privacy & Security. Using the `.app` bundle (rather than a bare binary) gives Hexecute a stable identity so these permissions persist across launches.

### Learning a Gesture

To configure a gesture to launch an application, run `hexecute --learn [command]` in a terminal. Hexecute should launch - simply draw your chosen gesture **3 times** and it will be mapped to the command.

> On macOS, the CLI commands (`--learn`, `--list`, `--remove`) need the binary inside the bundle, since they print progress to the terminal: run `/Applications/Hexecute.app/Contents/MacOS/hexecute --learn [command]`. Double-clicking or `open`ing the `.app` is for normal gesture-casting use.

![Gesture learning demo](assets/hexecute-learn.gif)

### Managing Gestures

To view all your configured gestures, run `hexecute --list` in a terminal.

To delete a previously assigned gesture, use the `hexecute --remove [gesture]` command.

All gestures are saved in the `~/.config/hexecute/gestures.json` file. This file can be manually shared, edited, backed up, or swapped.
