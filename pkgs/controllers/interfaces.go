package controllers

import (
	"context"
	"github.com/cloudflare/origin-ca-issuer/internal/cfapi"
	"github.com/cloudflare/origin-ca-issuer/pkgs/apis/v1"
	"github.com/cloudflare/origin-ca-issuer/pkgs/provisioners"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CFIssuer interface {
	GetSpec() v1.OriginIssuerSpec
	GetStatus() v1.OriginIssuerStatus
	GetSecretName() string
	GetSecretNamespace() string
	GetSecretKey() string
	GetRequestType() v1.RequestType
	SetConditions([]v1.OriginIssuerCondition)
	GetType() v1.IssuerType
}

var _ CFIssuer = &v1.OriginIssuer{}
var _ CFIssuer = &v1.OriginClusterIssuer{}

type CFController interface {
	setStatus(context.Context, CFIssuer, v1.ConditionStatus, string, string) error
	getClient() client.Client
	getCollection() *provisioners.Collection
	getFactory() cfapi.Factory
	getSecretNamespace(CFIssuer) string
}

var _ CFController = &OriginIssuerController{}
var _ CFController = &OriginClusterIssuerController{}
