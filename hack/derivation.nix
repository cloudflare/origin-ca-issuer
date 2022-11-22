{ config, lib, pkgs, ... }:

pkgs.buildGo119Module rec {
  pname = "origin-ca-issuer";
  version = "0.6.1";

  src = lib.sourceFilesBySuffices ../. [ ".go" ".mod" ".sum" ];

  vendorSha256 = "sha256-lmIWrrjAGf6hP4KtyHSwZ9i5N5DcBjNBhYeZElvfHT0=";

  subPackages = [ "cmd/controller" ];

  meta = with lib; { platforms = platforms.linux; };
}
