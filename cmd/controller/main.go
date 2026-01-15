package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	certmanager "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/cloudflare/origin-ca-issuer/cmd/controller/options"
	"github.com/cloudflare/origin-ca-issuer/internal/cfapi"
	v1 "github.com/cloudflare/origin-ca-issuer/pkgs/apis/v1"
	"github.com/cloudflare/origin-ca-issuer/pkgs/controllers"
	"github.com/go-logr/logr"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	crlog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

//go:generate go tool controller-gen rbac:roleName=originissuer-control paths=./. output:rbac:artifacts:config=../../deploy/rbac

// +kubebuilder:rbac:groups=cert-manager.io,resources=certificaterequests,verbs=get;list;watch
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificaterequests/status,verbs=patch

// +kubebuilder:rbac:groups=certificates.k8s.io,resources=certificatesigningrequests,verbs=get;list;watch
// +kubebuilder:rbac:groups=certificates.k8s.io,resources=certificatesigningrequests/status,verbs=patch
// +kubebuilder:rbac:groups=certificates.k8s.io,resources=signers,verbs=sign,resourceNames=originissuers.cert-manager.k8s.cloudflare.com/*;clusteroriginissuers.cert-manager.k8s.cloudflare.com/*

// +kubebuilder:rbac:groups=cert-manager.k8s.cloudflare.com,resources=originissuers;clusteroriginissuers,verbs=get;list;watch
// +kubebuilder:rbac:groups=cert-manager.k8s.cloudflare.com,resources=originissuers/status;clusteroriginissuers/status,verbs=patch

// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
//
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;create;update

func main() {
	fs := pflag.CommandLine
	o := options.NewControllerOptions()
	o.AddFlags(fs)

	_ = fs.Parse(os.Args[1:])

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	crlog.SetLogger(logr.FromSlogHandler(logger.Handler()))

	if err := o.Validate(); err != nil {
		logger.Error("error validating options", "error", err)
		os.Exit(1)
	}

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		logger.Error("could not add to scheme", "error", err)
		os.Exit(1)
	}
	if err := certmanager.AddToScheme(scheme); err != nil {
		logger.Error("could not add to scheme", "error", err)
		os.Exit(1)
	}
	if err := v1.AddToScheme(scheme); err != nil {
		logger.Error("could not add to scheme", "error", err)
		os.Exit(1)
	}

	kubeCfg, err := config.GetConfig()
	if err != nil {
		logger.Error("could not load kubeconfig", "error", err)
		os.Exit(1)
	}

	kubeCfg.QPS = o.KubernetesAPIQPS
	kubeCfg.Burst = o.KubernetesAPIBurst

	mgr, err := manager.New(kubeCfg, manager.Options{
		Scheme:                        scheme,
		LeaderElection:                true,
		LeaderElectionReleaseOnCancel: true,
		LeaderElectionNamespace:       o.LeaderElectionNamespace,
		LeaderElectionID:              o.LeaderElectionID,
	})
	if err != nil {
		logger.Error("could not create manager", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(signals.SetupSignalHandler())
	defer cancel()

	signer := &controllers.Signer{
		Reader:                   mgr.GetAPIReader(),
		ClusterResourceNamespace: o.ClusterResourceNamespace,
		Builder: cfapi.NewBuilder().WithClient(&http.Client{
			Timeout: 30 * time.Second,
		}),
	}
	if err := signer.SetupWithManager(ctx, mgr); err != nil {
		logger.Error("could not setup controllers with manager", "error", err)
		os.Exit(1)
	}

	if err := mgr.Start(ctx); err != nil {
		logger.Error("could not start manager", "error", err)
		os.Exit(1)
	}
}
