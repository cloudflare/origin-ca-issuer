package options

import (
	"fmt"

	"github.com/spf13/pflag"
)

type ControllerOptions struct {
	KubernetesAPIQPS         float32
	KubernetesAPIBurst       int
	ClusterResourceNamespace string
	LeaderElectionNamespace  string
	LeaderElectionID         string

	DisableApprovedCheck bool
}

const (
	defaultKubernetesAPIQPS   float32 = 20
	defaultKubernetesAPIBurst int     = 50
	defaultLeaderElectionID   string  = "origin-ca-issuer"
)

func NewControllerOptions() *ControllerOptions {
	return &ControllerOptions{
		KubernetesAPIQPS:   defaultKubernetesAPIQPS,
		KubernetesAPIBurst: defaultKubernetesAPIBurst,
		LeaderElectionID:   defaultLeaderElectionID,
	}
}

func (o *ControllerOptions) AddFlags(fs *pflag.FlagSet) {
	fs.Float32Var(&o.KubernetesAPIQPS, "kube-api-qps", defaultKubernetesAPIQPS, "Maximium queries-per-second of requests to the Kubernetes apiserver.")
	fs.IntVar(&o.KubernetesAPIBurst, "kube-api-burst", defaultKubernetesAPIBurst, "Maximium queries-per-second burst of request send to the Kubernetes apiserver.")
	fs.BoolVar(&o.DisableApprovedCheck, "disable-approved-check", o.DisableApprovedCheck, "Disables waiting for CertificateRequests to have an approved condition before signing.")
	fs.StringVar(&o.ClusterResourceNamespace, "cluster-resource-namespace", o.ClusterResourceNamespace, "Namespace used for cluster-scoped resources, such as secrets used by ClusterOriginIssuer")
	fs.StringVar(&o.LeaderElectionNamespace, "leader-election-namespace", o.LeaderElectionNamespace, "Namespace used for leader election leases (defaults to pod's namespace).")
	fs.StringVar(&o.LeaderElectionID, "leader-election-id", o.LeaderElectionID, "Lease name used for leader election.")
}

func (o *ControllerOptions) Validate() error {
	if o.KubernetesAPIBurst <= 0 {
		return fmt.Errorf("invalid value for kube-api-burst: %v must be higher than 0", o.KubernetesAPIBurst)
	}

	if o.KubernetesAPIQPS <= 0 {
		return fmt.Errorf("invalid value for kube-api-qps: %v must be higher than 0", o.KubernetesAPIQPS)
	}

	if o.ClusterResourceNamespace == "" {
		return fmt.Errorf("invalid value for cluster-resource-namespace: must be set")
	}

	if o.LeaderElectionID == "" {
		return fmt.Errorf("invalid value for leader-election-id: must be set")
	}

	return nil
}
