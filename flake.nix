{
  description = "Hexecute";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
  };

  outputs =
    inputs:
    let
      supportedSystems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems =
        f:
        inputs.nixpkgs.lib.genAttrs supportedSystems (
          system: f system inputs.nixpkgs.legacyPackages.${system}
        );

      mkHexecute =
        system: pkgs:
        let
          inherit (pkgs) lib stdenv;
          isDarwin = stdenv.isDarwin;
        in
        pkgs.buildGoModule {
          pname = "hexecute";
          version = "0.1.0";

          src = ./.;

          vendorHash = "sha256-CIlYhcX7F08Xwrr3/0tkgrfuP68UU0CeQ+HV63b6Ddg=";

          nativeBuildInputs = with pkgs; [
            pkg-config
          ] ++ lib.optionals (!isDarwin) [ makeWrapper ];

          buildInputs =
            if isDarwin then
              # On modern nixpkgs the macOS SDK (Cocoa, OpenGL, ...) is provided
              # by the stdenv itself, so the cgo `-framework` flags resolve
              # without any explicit framework derivations — those legacy
              # `darwin.apple_sdk.frameworks.*` stubs have been removed.
              [ ]
            else
              with pkgs; [
                wayland
                wayland-protocols
                libxkbcommon
                libGL
                libGLU
                mesa
                xorg.libX11
              ];

          # The Wayland/EGL driver path wrapping only applies on Linux.
          postFixup = lib.optionalString (!isDarwin) ''
            wrapProgram $out/bin/hexecute \
              --prefix __EGL_VENDOR_LIBRARY_DIRS : "/run/opengl-driver/share/glvnd/egl_vendor.d" \
              --prefix __EGL_VENDOR_LIBRARY_DIRS : "${pkgs.mesa}/share/glvnd/egl_vendor.d" \
              --prefix LIBGL_DRIVERS_PATH : "/run/opengl-driver/lib/dri" \
              --prefix LIBGL_DRIVERS_PATH : "${pkgs.mesa}/lib/dri"
          '';

          meta = {
            description = "Launch apps by casting spells! 🪄";
            homepage = "https://hexecute.app";
            license = lib.licenses.gpl3;
            platforms = lib.platforms.linux ++ lib.platforms.darwin;
          };
        };
    in
    {
      packages = forAllSystems (
        system: pkgs:
        let
          hexecute = mkHexecute system pkgs;
        in
        {
          inherit hexecute;
          default = hexecute;
        }
      );

      devShells = forAllSystems (
        system: pkgs:
        {
          default = pkgs.mkShell {
            name = "hexecute";

            packages =
              with pkgs;
              [
                go
                pkg-config
                gcc
              ]
              ++ pkgs.lib.optionals (!pkgs.stdenv.isDarwin) [
                # Wayland libraries
                wayland
                wayland-protocols
                wayland-scanner
                libxkbcommon

                # EGL and OpenGL
                libGL
                libGLU
                mesa
              ];
          };
        }
      );
    };
}
