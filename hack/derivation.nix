{ config, lib, pkgs, ... }:

pkgs.buildGoModule rec {
  pname = "origin-ca-issuer";
  version = "0.6.0";

  src = lib.sourceFilesBySuffices ../. [ ".go" ".mod" ".sum" ];

  vendorSha256 = "0ijs6fq2agbg63bckzflap96nxr18shfcwb8nxglmi41d7jafb6l";

  subPackages = [ "cmd/controller" ];

  meta = with lib; { platforms = platforms.linux; };
}
