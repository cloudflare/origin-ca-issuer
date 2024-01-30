package controllers

import (
	"context"
	"fmt"

	"github.com/cloudflare/origin-ca-issuer/internal/cfapi"
	v1 "github.com/cloudflare/origin-ca-issuer/pkgs/apis/v1"
	"github.com/cloudflare/origin-ca-issuer/pkgs/provisioners"
	"github.com/go-logr/logr"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
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

//go:generate controller-gen rbac:roleName=originissuer-control paths=./. output:rbac:artifacts:config=../../deploy/rbac

// +kubebuilder:rbac:groups=cert-manager.k8s.cloudflare.com,resources=originissuers,verbs=get;list;watch;create
// +kubebuilder:rbac:groups=cert-manager.k8s.cloudflare.com,resources=originissuers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile reconciles OriginIssuer resources by managing Cloudflare API provisioners.
func (r *OriginIssuerController) Reconcile(ctx context.Context, iss *v1.OriginIssuer) (reconcile.Result, error) {
	log := r.Log.WithValues("namespace", iss.Namespace, "originissuer", iss.Name)

	if err := validateOriginIssuer(iss.Spec); err != nil {
		log.Error(err, "failed to validate OriginIssuer resource")

		return reconcile.Result{}, err
	}

	secret := core.Secret{}
	secretNamespaceName := types.NamespacedName{
		Namespace: iss.Namespace,
		Name:      iss.Spec.Auth.ServiceKeyRef.Name,
	}

	if err := r.Client.Get(ctx, secretNamespaceName, &secret); err != nil {
		log.Error(err, "failed to retieve OriginIssuer auth secret", "namespace", secretNamespaceName.Namespace, "name", secretNamespaceName.Name)

		if apierrors.IsNotFound(err) {
			_ = r.setStatus(ctx, iss, v1.ConditionFalse, "NotFound", fmt.Sprintf("Failed to retrieve auth secret: %v", err))
		} else {
			_ = r.setStatus(ctx, iss, v1.ConditionFalse, "Error", fmt.Sprintf("Failed to retrieve auth secret: %v", err))
		}

		return reconcile.Result{}, err
	}

	serviceKey, ok := secret.Data[iss.Spec.Auth.ServiceKeyRef.Key]
	if !ok {
		err := fmt.Errorf("secret %s does not contain key %q", secret.Name, iss.Spec.Auth.ServiceKeyRef.Key)
		log.Error(err, "failed to retrieve OriginIssuer auth secret")
		_ = r.setStatus(ctx, iss, v1.ConditionFalse, "NotFound", fmt.Sprintf("Failed to retrieve auth secret: %v", err))

		return reconcile.Result{}, err
	}

	c, err := r.Factory.APIWith(serviceKey)
	if err != nil {
		log.Error(err, "failed to create API client")

		return reconcile.Result{}, err
	}

	p, err := provisioners.New(c, iss.Spec.RequestType, log)
	if err != nil {
		log.Error(err, "failed to create provisioner")

		_ = r.setStatus(ctx, iss, v1.ConditionFalse, "Error", "Failed initialize provisioner")

		return reconcile.Result{}, err
	}

	// TODO: GC these references once the OriginIssuer has been removed.
	r.Collection.Store(types.NamespacedName{Name: iss.Name, Namespace: iss.Namespace}, p)

	return reconcile.Result{}, r.setStatus(ctx, iss, v1.ConditionTrue, "Verified", "OriginIssuer verified and ready to sign certificates")
}

// setStatus is a helper function to set the Issuer status condition with reason and message, and update the API.
func (r *OriginIssuerController) setStatus(ctx context.Context, iss *v1.OriginIssuer, status v1.ConditionStatus, reason, message string) error {
	SetIssuerCondition(iss, v1.ConditionReady, status, r.Log, r.Clock, reason, message)

	return r.Client.Status().Update(ctx, iss)
}

// validateOriginIssuer ensures required fields are set, and enums are correctly set.
// TODO: move this to another package?
func validateOriginIssuer(s v1.OriginIssuerSpec) error {
	switch {
	case s.Auth.ServiceKeyRef.Name == "":
		return fmt.Errorf("spec.auth.serviceKeyRef.name cannot be empty")
	case s.Auth.ServiceKeyRef.Key == "":
		return fmt.Errorf("spec.auth.serviceKeyRef.key cannot be empty")
	case s.RequestType == "":
		return fmt.Errorf("spec.requestType cannot be empty")
	case s.RequestType != v1.RequestTypeOriginRSA && s.RequestType != v1.RequestTypeOriginECC:
		return fmt.Errorf("spec.requestType has invalid value %q", s.RequestType)
	}

	return nil
}
