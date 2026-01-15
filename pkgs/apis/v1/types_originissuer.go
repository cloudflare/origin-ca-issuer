package v1

import (
	issuerv1alpha1 "github.com/cert-manager/issuer-lib/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// An OriginIssuer represents the Cloudflare Origin CA as an external cert-manager issuer.
// It is scoped to a single namespace, so it can be used only by resources in the same
// namespace.
type OriginIssuer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Desired state of the OriginIssuer resource
	Spec OriginIssuerSpec `json:"spec,omitempty"`

	// Status of the OriginIssuer. This is set and managed automatically.
	// +optional
	Status issuerv1alpha1.IssuerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OriginIssuerList is a list of OriginIssuers.
type OriginIssuerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata.omitempty"`

	Items []OriginIssuer `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status

// A ClusterOriginIssuer represents the Cloudflare Origin CA as an external cert-manager issuer.
// It is scoped to a single namespace, so it can be used only by resources in the same
// namespace.
type ClusterOriginIssuer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the desired state of the ClusterOriginIssuer resource.
	Spec OriginIssuerSpec `json:"spec,omitempty"`

	// Status of the ClusterOriginIssuer. This is set and managed automatically.
	// +optional
	Status issuerv1alpha1.IssuerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterOriginIssuerList is a list of OriginIssuers.
type ClusterOriginIssuerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata.omitempty"`

	Items []ClusterOriginIssuer `json:"items"`
}

// OriginIssuerSpec is the specification of an OriginIssuer. This includes any
// configuration required for the issuer.
type OriginIssuerSpec struct {
	// RequestType is the signature algorithm Cloudflare should use to sign the certificate.
	RequestType RequestType `json:"requestType"`

	// Auth configures how to authenticate with the Cloudflare API.
	Auth OriginIssuerAuthentication `json:"auth"`
}

// OriginIssuerAuthentication defines how to authenticate with the Cloudflare API.
// Only one of `serviceKeyRef` may be specified.
//
// +kubebuilder:validation:ExactlyOneOf=serviceKeyRef;tokenRef
type OriginIssuerAuthentication struct {
	// ServiceKeyRef authenticates with an API Service Key (the "Origin CA Key").
	// +optional
	ServiceKeyRef *SecretKeySelector `json:"serviceKeyRef,omitempty"`

	// TokenRef authenticates with an API Token.
	// +optional
	TokenRef *SecretKeySelector `json:"tokenRef,omitempty"`
}

// SecretKeySelector contains a reference to a secret.
type SecretKeySelector struct {
	// Name of the secret in the issuer's namespace to select. If a cluster-scoped
	// issuer, the secret is selected from the "cluster resource namespace" configured
	// on the controller.
	Name string `json:"name"`
	// Key of the secret to select from. Must be a valid secret key.
	Key string `json:"key"`
}

// +kubebuilder:validation:Enum=OriginRSA;OriginECC

// RequestType represents the signature algorithm used to sign certificates.
type RequestType string

const (
	// RequestTypeOriginRSA represents an RSA256 signature.
	RequestTypeOriginRSA RequestType = "OriginRSA"

	// RequestTypeOriginECC represents an ECDSA signature.
	RequestTypeOriginECC RequestType = "OriginECC"
)
