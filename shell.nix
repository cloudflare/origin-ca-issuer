{ pkgs ? import ./hack/nixpkgs.nix { }, ... }:

with pkgs;

let
  code-gen = buildGo117Module rec {
    pname = "code-generator";
    version = "0.24.0";

    src = fetchFromGitHub {
      owner = "kubernetes";
      repo = "code-generator";
      rev = "v${version}";
      sha256 = "1gzw1fs458ffnvd9hpk9n4azsbm7k90sq6b5lfyf270xzw4cm4hx";
    };
    subPackages = [ "cmd/deepcopy-gen" "cmd/register-gen" ];

    vendorSha256 = "0g5mjk3kzdzary841cx8c2j21v7yzr3vmqcl2pdkkpqldsy4zrx7";

    # deepcopy-gen calls `go list` which wants to inspect GOROOT.
    # I could specify when run, but we're building everything with a specific
    # Go version anyways.
    allowGoReference = true;
  };
  controller-tools = buildGo117Module rec {
    pname = "controller-tools";
    version = "0.8.0";

    src = fetchFromGitHub {
      owner = "kubernetes-sigs";
      repo = "controller-tools";
      rev = "v${version}";
      sha256 = "0cqmpj4gk1h6g6x6y63clbg3b4amfpm3qbypwj1dj4nc7ybgyygs";
    };
    subPackages = [ "cmd/controller-gen" ];

    vendorSha256 = "18az4a9lwm1w2a5ml9gil4khg74grar1d15x3643h0jly2qpf8a0";
  };

in pkgs.mkShell {
  nativeBuildInputs =
    [ go_1_17 gopls goimports golangci-lint code-gen controller-tools ];

  TEST_ASSET_KUBE_APISERVER = "${kubernetes}/bin/kube-apiserver";
  TEST_ASSET_ETCD = "${etcd}/bin/etcd";
  TEST_ASSET_KUBECTL = "${kubectl}/bin/kubectl";
}
