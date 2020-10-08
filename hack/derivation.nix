{ config, lib, pkgs, ... }:

pkgs.buildGoModule rec {
  pname = "origin-ca-issuer";
  version = "0.0.0";

  src = lib.sourceFilesBySuffices ../. [ ".go" ".mod" ".sum" ];

  vendorSha256 = "0bda8pw8v9frs1n9sva0qlfhvk9c8jqj6xy2x6kpcq7igiflsm24";

  subPackages = [ "cmd/controller" ];

  meta = with lib; { platforms = platforms.linux; };
}
