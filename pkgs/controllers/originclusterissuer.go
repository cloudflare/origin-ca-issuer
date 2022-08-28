package controllers

import (
	"context"
	"github.com/cloudflare/origin-ca-issuer/internal/cfapi"
	v1 "github.com/cloudflare/origin-ca-issuer/pkgs/apis/v1"
	"github.com/cloudflare/origin-ca-issuer/pkgs/provisioners"
	"github.com/go-logr/logr"
	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type OriginClusterIssuerController struct {
	client.Client
	Log         logr.Logger
	Clock       clock.Clock
	Factory     cfapi.Factory
	Collection  *provisioners.Collection
	CRNamespace string
}

// +kubebuilder:rbac:groups=cert-manager.k8s.cloudflare.com,resources=originclusterissuers,verbs=get;list;watch;create
// +kubebuilder:rbac:groups=cert-manager.k8s.cloudflare.com,resources=originclusterissuers/status,verbs=get;update;patch

func (r *OriginClusterIssuerController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	log := r.Log.WithValues("originclusterissuer", req.NamespacedName)

	iss := &v1.OriginClusterIssuer{}
	if err := r.Client.Get(ctx, req.NamespacedName, iss); err != nil {
		log.Error(err, "failed to retrieve OriginClusterIssuer")

		return reconcile.Result{}, client.IgnoreNotFound(err)
	}
	return ReconcileCommon(r, ctx, req, log, iss)
}

// setStatus is a helper function to set the Issuer status condition with reason and message, and update the API.
func (r *OriginClusterIssuerController) setStatus(ctx context.Context, iss CFIssuer, status v1.ConditionStatus, reason string, message string) error {
	SetIssuerCondition(iss, v1.ConditionReady, status, r.Log, r.Clock, reason, message)

	return r.Client.Status().Update(ctx, iss.(*v1.OriginClusterIssuer))
}

func (r *OriginClusterIssuerController) getClient() client.Client {
	return r.Client
}

func (r *OriginClusterIssuerController) getCollection() *provisioners.Collection {
	return r.Collection
}

func (r *OriginClusterIssuerController) getFactory() cfapi.Factory {
	return r.Factory
}

func (r *OriginClusterIssuerController) getSecretNamespace(issuer CFIssuer) string {
	return r.CRNamespace
}
