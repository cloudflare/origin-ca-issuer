{ config, lib, pkgs, ... }:

pkgs.buildGo117Module rec {
  pname = "origin-ca-issuer";
  version = "0.6.1";

  src = lib.sourceFilesBySuffices ../. [ ".go" ".mod" ".sum" ];

  vendorSha256 = "1qr3mh1ws9ff21qs6pf7d4hf5v1aw33zailzi9r3r6dykndjkhwg";

  subPackages = [ "cmd/controller" ];

  meta = with lib; { platforms = platforms.linux; };
}
