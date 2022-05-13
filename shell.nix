{ pkgs ? import ./hack/nixpkgs.nix { }, ... }:

with pkgs;

let
  code-gen = buildGoModule rec {
    pname = "code-generator";
    version = "0.18.9";

    src = fetchFromGitHub {
      owner = "kubernetes";
      repo = "code-generator";
      rev = "v${version}";
      sha256 = "1aj2h6b1aj0kxxcx3hkms01fipmw70v40ws583lldz2ch7pw78gd";
    };
    subPackages = [ "cmd/deepcopy-gen" "cmd/register-gen" ];

    vendorSha256 = "14csplyhfy56c10bmm8vvzhc14gk7dw4859ifyn3kbprj2av6kzb";

    # deepcopy-gen calls `go list` which wants to inspect GOROOT.
    # I could specify when run, but we're building everything with a specific
    # Go version anyways.
    allowGoReference = true;
  };
  controller-tools = buildGoModule rec {
    pname = "controller-tools";
    version = "0.4.1";

    src = fetchFromGitHub {
      owner = "kubernetes-sigs";
      repo = "controller-tools";
      rev = "v${version}";
      sha256 = "0hbnz5my2bwds16hdb9fzbf2ri6lhpn3jd4si7z7lbaiv0zm429m";
    };
    subPackages = [ "cmd/controller-gen" ];

    vendorSha256 = "04nh4ql50w8w8zsqwg7rr0fz3lfnvky4l77rli8fpywg58z77n7k";
  };
  # pin controller-runtime's envtest to Kubernetes 1.19.5 due to
  # incompatibilites with 1.20:
  #   https://github.com/kubernetes-sigs/controller-runtime/issues/1357
  kubePkgs = import (builtins.fetchTarball {
    name = "nixpkgs-2021-02-08-k8s-1.19.5";
    url =
      "https://github.com/NixOS/nixpkgs/archive/bed08131cd29a85f19716d9351940bdc34834492.tar.gz";
    sha256 = "19gxrzk9y4g2f09x2a4g5699ccw35h5frznn9n0pbsyv45n9vxix";
  }) { };

in pkgs.mkShell {
  nativeBuildInputs =
    [ go_1_17 gopls goimports golangci-lint code-gen controller-tools ];

  TEST_ASSET_KUBE_APISERVER = "${kubePkgs.kubernetes}/bin/kube-apiserver";
  TEST_ASSET_ETCD = "${etcd}/bin/etcd";
  TEST_ASSET_KUBECTL = "${kubePkgs.kubectl}/bin/kubectl";
}
