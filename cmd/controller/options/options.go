package options

import (
	"fmt"

	"github.com/spf13/pflag"
)

type ControllerOptions struct {
	KubernetesAPIQPS   float32
	KubernetesAPIBurst int
}

const (
	defaultKubernetesAPIQPS   float32 = 20
	defaultKubernetesAPIBurst int     = 50
)

func NewControllerOptions() *ControllerOptions {
	return &ControllerOptions{
		KubernetesAPIQPS:   defaultKubernetesAPIQPS,
		KubernetesAPIBurst: defaultKubernetesAPIBurst,
	}
}

func (o *ControllerOptions) AddFlags(fs *pflag.FlagSet) {
	fs.Float32Var(&o.KubernetesAPIQPS, "kube-api-qps", defaultKubernetesAPIQPS, "Maximium queries-per-second of requests to the Kubernetes apiserver.")
	fs.IntVar(&o.KubernetesAPIBurst, "kube-api-burst", defaultKubernetesAPIBurst, "Maximium queries-per-second burst of request send to the Kubernetes apiserver.")
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
