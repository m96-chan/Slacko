{
  description = "Slacko — A lightweight, keyboard-driven TUI client for Slack";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "slacko";
          version = self.shortRev or self.dirtyShortRev or "dev";

          src = ./.;

          vendorHash = null;

          ldflags = [
            "-s"
            "-w"
            "-X main.version=${self.shortRev or self.dirtyShortRev or "dev"}"
          ];

          meta = with pkgs.lib; {
            description = "A lightweight, keyboard-driven TUI client for Slack";
            homepage = "https://github.com/m96-chan/Slacko";
            license = licenses.mit;
            mainProgram = "slacko";
          };
        };
      }
    );
}
