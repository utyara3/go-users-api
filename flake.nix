{
  description = "Go development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
    in
    {
      devShells.${system}.default = pkgs.mkShell {
        buildInputs = with pkgs; [
          go_1_27
          gopls
          gofumpt
          golangci-lint
          gnumake
        ];

        shellHook = ''
          echo "Go development environment loaded!"
          go version
        '';
      };
    };
}
