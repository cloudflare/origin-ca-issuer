package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type IssuerType string

const (
	OriginIssuerType        IssuerType = "OriginIssuer"
	OriginClusterIssuerType IssuerType = "OriginClusterIssuer"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
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
	Status OriginIssuerStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OriginIssuerList is a list of OriginIssuers.
type OriginIssuerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata.omitempty"`

	Items []OriginIssuer `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// An OriginClusterIssuer represents the Cloudflare Origin CA as an external cert-manager issuer.
// It is cluster wide.
type OriginClusterIssuer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Desired state of the OriginIssuer resource
	Spec OriginIssuerSpec `json:"spec,omitempty"`

	// Status of the OriginIssuer. This is set and managed automatically.
	// +optional
	Status OriginIssuerStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OriginClusterIssuerList is a list of OriginClusterIssuer.
type OriginClusterIssuerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata.omitempty"`

	Items []OriginClusterIssuer `json:"items"`
}

// OriginIssuerSpec is the specification of an OriginIssuer. This includes any
// configuration required for the issuer.
type OriginIssuerSpec struct {
	// RequestType is the signature algorithm Cloudflare should use to sign the certificate.
	RequestType RequestType `json:"requestType"`

	// Auth configures how to authenticate with the Cloudflare API.
	Auth OriginIssuerAuthentication `json:"auth"`
}

// OriginIssuerStatus contains status information about an OriginIssuer
type OriginIssuerStatus struct {
	// List of status conditions to indicate the status of an OriginIssuer
	// Known condition types are `Ready`.
	// +optional
	Conditions []OriginIssuerCondition `json:"conditions,omitempty"`
}

// OriginIssuerAuthentication defines how to authenticate with the Cloudflare API.
// Only one of `serviceKeyRef` may be specified.
type OriginIssuerAuthentication struct {
	// ServiceKeyRef authenticates with an API Service Key.
	// +optional
	ServiceKeyRef SecretKeySelector `json:"serviceKeyRef,omitempty"`
}

// SecretKeySelector contains a reference to a secret.
type SecretKeySelector struct {
	// Name of the secret in the OriginIssuer's namespace to select from.
	Name string `json:"name"`
	// Namespace of the secret in the OriginIssuer's namespace to select from.
	Namespace string `json:"namespace,omitempty"`
	// Key of the secret to select from. Must be a valid secret key.
	Key string `json:"key"`
}

// OriginIssuerCondition contains condition information for the OriginIssuer.
type OriginIssuerCondition struct {
	// Type of the condition, known values are ('Ready')
	Type ConditionType `json:"type"`

	// Status of the condition, one of ('True', 'False', 'Unknown')
	Status ConditionStatus `json:"status"`

	// LastTransitionTime is the timestamp corresponding to the last status
	// change of this condition.
	// +optional
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`

	// Reason is a brief machine readable explanation for the condition's last
	// transition.
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message is a human readable description of the details of the last
	// transition1, complementing reason.
	// +optional
	Message string `json:"message,omitempty"`
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

// +kubebuilder:validation:Enum=Ready

// ConditionType represents an OriginIssuer condition value.
type ConditionType string

const (
	// ConditionReady represents that an OriginIssuer condition is in
	// a ready state and able to issue certificates.
	// If the `status` of this condition is `False`, CertificateRequest
	// controllers should prevent attempts to sign certificates.
	ConditionReady ConditionType = "Ready"
)

// +kubebuilder:validation:Enum=True;False;Unknown

// ConditionStatus represents a condition's status.
type ConditionStatus string

const (
	// ConditionTrue represents the fact that a given condition is true.
	ConditionTrue ConditionStatus = "True"

	// ConditionFalse represents the fact that a given condition is false.
	ConditionFalse ConditionStatus = "False"

	// ConditionUnknown represents the fact that a given condition is unknown.
	ConditionUnknown ConditionStatus = "Unknown"
)

func (o *OriginIssuer) GetSpec() OriginIssuerSpec {
	return o.Spec
}

func (o *OriginIssuer) GetStatus() OriginIssuerStatus {
	return o.Status
}

func (o *OriginIssuer) GetSecretName() string {
	return o.Spec.Auth.ServiceKeyRef.Name
}

func (o *OriginIssuer) GetSecretNamespace() string {
	return o.Spec.Auth.ServiceKeyRef.Namespace
}

func (o *OriginIssuer) GetSecretKey() string {
	return o.Spec.Auth.ServiceKeyRef.Key
}

func (o *OriginIssuer) GetRequestType() RequestType {
	return o.Spec.RequestType
}

func (o *OriginIssuer) SetConditions(conditions []OriginIssuerCondition) {
	o.Status.Conditions = conditions
}

func (o *OriginIssuer) GetType() IssuerType {
	return OriginIssuerType
}

func (o *OriginClusterIssuer) GetSpec() OriginIssuerSpec {
	return o.Spec
}

func (o *OriginClusterIssuer) GetStatus() OriginIssuerStatus {
	return o.Status
}

func (o *OriginClusterIssuer) GetSecretName() string {
	return o.Spec.Auth.ServiceKeyRef.Name
}

func (o *OriginClusterIssuer) GetSecretNamespace() string {
	return o.Spec.Auth.ServiceKeyRef.Namespace
}

func (o *OriginClusterIssuer) GetSecretKey() string {
	return o.Spec.Auth.ServiceKeyRef.Key
}

func (o *OriginClusterIssuer) GetRequestType() RequestType {
	return o.Spec.RequestType
}

func (o *OriginClusterIssuer) SetConditions(conditions []OriginIssuerCondition) {
	o.Status.Conditions = conditions
}

func (o *OriginClusterIssuer) GetType() IssuerType {
	return OriginClusterIssuerType
}
