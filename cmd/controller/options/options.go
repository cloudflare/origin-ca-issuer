package options

import (
	"fmt"
	"github.com/spf13/pflag"
	"os"
)

type ControllerOptions struct {
	KubernetesAPIQPS   float32
	KubernetesAPIBurst int

	DisableApprovedCheck bool
	CRNamespace          string
}

const (
	defaultKubernetesAPIQPS   float32 = 20
	defaultKubernetesAPIBurst int     = 50
)

func NewControllerOptions() *ControllerOptions {
	opts := &ControllerOptions{
		KubernetesAPIQPS:   defaultKubernetesAPIQPS,
		KubernetesAPIBurst: defaultKubernetesAPIBurst,
	}

	ns, found := os.LookupEnv("POD_NAMESPACE")

	if found {
		opts.CRNamespace = ns
	} else {
		opts.CRNamespace = "default"
	}
	return opts
}

func (o *ControllerOptions) AddFlags(fs *pflag.FlagSet) {
	fs.Float32Var(&o.KubernetesAPIQPS, "kube-api-qps", defaultKubernetesAPIQPS, "Maximium queries-per-second of requests to the Kubernetes apiserver.")
	fs.IntVar(&o.KubernetesAPIBurst, "kube-api-burst", defaultKubernetesAPIBurst, "Maximium queries-per-second burst of request send to the Kubernetes apiserver.")
	fs.BoolVar(&o.DisableApprovedCheck, "disable-approved-check", o.DisableApprovedCheck, "Disables waiting for CertificateRequests to have an approved condition before signing.")
	fs.StringVar(&o.CRNamespace, "cluster-resource-namespace", o.CRNamespace, "Namespace for secrets for cluster wide resources.")
}

func (o *ControllerOptions) Validate() error {
	if o.KubernetesAPIBurst <= 0 {
		return fmt.Errorf("invalid value for kube-api-burst: %v must be higher than 0", o.KubernetesAPIBurst)
	}

	if o.KubernetesAPIQPS <= 0 {
		return fmt.Errorf("invalid value for kube-api-qps: %v must be higher than 0", o.KubernetesAPIQPS)
	}

	return nil
}
