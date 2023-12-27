// +k8s:deepcopy-gen=package
// +groupName=cert-manager.k8s.cloudflare.com

// Package v1 is the v1 version of the OriginIssuer API
package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

//go:generate controller-gen object crd paths=./. output:crd:artifacts:config=../../../deploy/crds

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "cert-manager.k8s.cloudflare.com", Version: "v1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func init() {
	SchemeBuilder.Register(&OriginIssuer{}, &OriginIssuerList{})
}
