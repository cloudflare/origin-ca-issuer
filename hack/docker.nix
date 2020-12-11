{ pkgs ? import ./nixpkgs.nix { }, ... }:

with pkgs;

let controller = callPackage ./derivation.nix { };

in dockerTools.buildLayeredImage {
  name = "origin-ca-issuer";
  config.Entrypoint = [ "${controller}/bin/controller" ];
}
