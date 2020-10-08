with (import ./hack/nixpkgs.nix { });

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
    version = "0.4.0";

    src = fetchFromGitHub {
      owner = "kubernetes-sigs";
      repo = "controller-tools";
      rev = "v${version}";
      sha256 = "0ix7m1fi06mhp8xxfg2r82jzyphzx2lm8jmx23l9ai654bcnnnwh";
    };
    subPackages = [ "cmd/controller-gen" ];

    vendorSha256 = "04nh4ql50w8w8zsqwg7rr0fz3lfnvky4l77rli8fpywg58z77n7k";
  };

in pkgs.mkShell {
  nativeBuildInputs = [
    go
    gopls
    goimports
    golangci-lint
    code-gen
    controller-tools
    etcd
    kubernetes
    kubectl
  ];

  TEST_ASSET_KUBE_APISERVER = "${kubernetes}/bin/kube-apiserver";
  TEST_ASSET_ETCD = "${etcd}/bin/etcd";
  TEST_ASSET_KUBECTL = "${kubectl}/bin/kubectl";
}
