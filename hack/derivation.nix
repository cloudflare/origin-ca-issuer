{ config, lib, pkgs, ... }:

pkgs.buildGoModule rec {
  pname = "origin-ca-issuer";
  version = "0.0.0";

  src = lib.sourceFilesBySuffices ../. [ ".go" ".mod" ".sum" ];

  vendorSha256 = "08pplaif6par75fgm7xbibvfbl9cy4shyd4bs4hawfgx5b84djfp";

  subPackages = [ "cmd/controller" ];

  meta = with lib; { platforms = platforms.linux; };
}
