{
  description = "My Nix library";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";

    snowfall = {
      url = "github:snowfallorg/lib";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = inputs:
    inputs.snowfall.mkFlake {
      inherit inputs;

      src = ./.;

      outputs-builder = channels: {
        packages = rec {
          default = tailscale-authproxy;

          tailscale-authproxy = with channels.nixpkgs;
            buildGo119Module {
              pname = "tailscale-authproxy";
              version = inputs.self.sourceInfo.rev or "dirty";

              src = ./.;

              proxyVendor = true;
              vendorSha256 = "sha256-j/bv0YqVCRcXedoHYIcy9612nUjwDNi9iJJMGWU96E4";

              meta = with lib;
                {
                  description = "A Dex AuthProxy connector for your tailnets";
                  homepage = "https://github.com/jakehamilton/tailscale-authproxy";
                  license = licenses.bsd3;
                  maintainers = with maintainers; [ jakehamilton ];
                };
            };
        };

        devShells = {
          default = with channels.nixpkgs;
            mkShell
              {
                buildInputs = [ go_1_19 gopls ];
              };
        };
      };
    };
}
