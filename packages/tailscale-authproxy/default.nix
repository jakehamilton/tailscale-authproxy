{ lib, buildGo119Module, self, ... }:

let
  src = lib.snowfall.fs.get-file "";
in
buildGo119Module {
  pname = "tailscale-authproxy";
  version = self.sourceInfo.rev or "dirty";

  inherit src;

  proxyVendor = true;
  vendorSha256 = "sha256-j/bv0YqVCRcXedoHYIcy9612nUjwDNi9iJJMGWU96E4";

  meta = with lib;
    {
      description = "A Dex AuthProxy connector for your tailnets";
      homepage = "https://github.com/jakehamilton/tailscale-authproxy";
      license = licenses.bsd3;
      maintainers = with maintainers; [ jakehamilton ];
    };
}
