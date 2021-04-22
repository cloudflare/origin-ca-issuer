module github.com/cloudflare/origin-ca-issuer

go 1.15

require (
	github.com/butonic/zerologr v0.0.0-20191210074216-d798ee237d84
	github.com/go-logr/logr v0.2.1-0.20200730175230-ee2de8da5be6
	github.com/google/go-cmp v0.4.1
	github.com/jetstack/cert-manager v1.3.1
	k8s.io/api v0.19.0
	k8s.io/apimachinery v0.19.0
	k8s.io/client-go v0.19.0
	k8s.io/utils v0.0.0-20200729134348-d5654de09c73
	sigs.k8s.io/controller-runtime v0.6.3
)
