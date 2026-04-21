{
  description = "Sunbeams dev environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            gotools
            go-tools
            golangci-lint
            goreleaser
            delve
            gnumake
            git
          ] ++ pkgs.lib.optionals pkgs.stdenv.isLinux [
            pkgs.edid-decode
          ];

          shellHook = ''
            echo "sunbeams dev shell"
            echo "go: $(go version)"
          '';
        };
      });
}
