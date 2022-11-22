{ config, lib, pkgs, ... }:

pkgs.buildGo117Module rec {
  pname = "origin-ca-issuer";
  version = "0.6.1";

  src = lib.sourceFilesBySuffices ../. [ ".go" ".mod" ".sum" ];

  vendorSha256 = "sha256-YZYR6e07kZFcGYTGYJG6ywJI2sMJBOQi8I3m3GSgIBM=";

  subPackages = [ "cmd/controller" ];

  meta = with lib; { platforms = platforms.linux; };
}
