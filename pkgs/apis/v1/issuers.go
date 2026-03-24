package v1

import (
	issuerv1alpha1 "github.com/cert-manager/issuer-lib/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AuthType int

const (
	AuthTypeUnknown  AuthType = 0
	AuthTypeAPIToken AuthType = 2
)

var _ issuerv1alpha1.Issuer = (*OriginIssuer)(nil)
var _ issuerv1alpha1.Issuer = (*ClusterOriginIssuer)(nil)

func (iss *OriginIssuer) GetConditions() []metav1.Condition {
	return iss.Status.Conditions
}

func (iss *OriginIssuer) GetIssuerTypeIdentifier() string {
	return "originissuers." + GroupVersion.Group
}

func (iss *OriginIssuer) GetAuthSecretNamespace(_ string) string {
	return iss.Namespace
}

func (iss *OriginIssuer) GetAuth() OriginIssuerAuthentication {
	return iss.Spec.Auth
}

func (iss *OriginIssuer) GetRequestType() RequestType {
	return iss.Spec.RequestType
}

func (iss *ClusterOriginIssuer) GetConditions() []metav1.Condition {
	return iss.Status.Conditions
}

func (iss *ClusterOriginIssuer) GetIssuerTypeIdentifier() string {
	return "clusteroriginissuers." + GroupVersion.Group
}

func (iss *ClusterOriginIssuer) GetAuthSecretNamespace(clusterResourceNamespace string) string {
	return clusterResourceNamespace
}

func (iss *ClusterOriginIssuer) GetAuth() OriginIssuerAuthentication {
	return iss.Spec.Auth
}

func (iss *ClusterOriginIssuer) GetRequestType() RequestType {
	return iss.Spec.RequestType
}

func (a OriginIssuerAuthentication) GetSecretKeySelector() *SecretKeySelector {
	return a.TokenRef
}

func (a OriginIssuerAuthentication) GetType() AuthType {
	if a.TokenRef != nil {
		return AuthTypeAPIToken
	}
	return AuthTypeUnknown
}
