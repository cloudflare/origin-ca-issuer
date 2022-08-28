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

// OriginIssuerController implements a controller that watches for changes
// to OriginIssuer resources.
type OriginIssuerController struct {
	client.Client
	Log        logr.Logger
	Clock      clock.Clock
	Factory    cfapi.Factory
	Collection *provisioners.Collection
}

// +kubebuilder:rbac:groups=cert-manager.k8s.cloudflare.com,resources=originissuers,verbs=get;list;watch;create
// +kubebuilder:rbac:groups=cert-manager.k8s.cloudflare.com,resources=originissuers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile reconciles OriginIssuer resources by managing Cloudflare API provisioners.
func (r *OriginIssuerController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	log := r.Log.WithValues("originissuer", req.NamespacedName)

	iss := &v1.OriginIssuer{}
	if err := r.Client.Get(ctx, req.NamespacedName, iss); err != nil {
		log.Error(err, "failed to retrieve OriginIssuer")

		return reconcile.Result{}, client.IgnoreNotFound(err)
	}
	return ReconcileCommon(r, ctx, req, log, iss)
}

// setStatus is a helper function to set the Issuer status condition with reason and message, and update the API.
func (r *OriginIssuerController) setStatus(ctx context.Context, iss CFIssuer, status v1.ConditionStatus, reason string, message string) error {
	SetIssuerCondition(iss, v1.ConditionReady, status, r.Log, r.Clock, reason, message)

	return r.Client.Status().Update(ctx, iss.(*v1.OriginIssuer))
}

func (r *OriginIssuerController) getClient() client.Client {
	return r.Client
}

func (r *OriginIssuerController) getCollection() *provisioners.Collection {
	return r.Collection
}

func (r *OriginIssuerController) getFactory() cfapi.Factory {
	return r.Factory
}

func (r *OriginIssuerController) getSecretNamespace(issuer CFIssuer) string {
	return issuer.GetSecretNamespace()
}
