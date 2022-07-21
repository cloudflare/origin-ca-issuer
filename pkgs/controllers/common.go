package controllers

import (
	"context"
	"errors"
	"fmt"

	v1 "github.com/cloudflare/origin-ca-issuer/pkgs/apis/v1"
	"github.com/cloudflare/origin-ca-issuer/pkgs/provisioners"
	"github.com/go-logr/logr"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func ReconcileCommon(r CFController, ctx context.Context, req reconcile.Request, log logr.Logger, iss CFIssuer) (reconcile.Result, error) {

	if err := validateOriginIssuer(iss.GetSpec()); err != nil {
		log.Error(err, "failed to validate resource")

		return reconcile.Result{}, err
	}

	secret := core.Secret{}
	secretNamespaceName := types.NamespacedName{
		Name: iss.GetSecretName(),
	}

	if iss.GetType() == v1.OriginClusterIssuerType {
		if iss.GetSecretNamespace() == "" {
			err := errors.New("no namespace defined for secret")
			log.Error(err, "unable to parse auth secret")
			return reconcile.Result{}, err
		}

		secretNamespaceName.Namespace = iss.GetSecretNamespace()
	} else {
		secretNamespaceName.Namespace = req.Namespace
	}

	if err := r.getClient().Get(ctx, secretNamespaceName, &secret); err != nil {
		log.Error(err, "failed to retrieve auth secret", "namespace", secretNamespaceName.Namespace, "name", secretNamespaceName.Name)

		if apierrors.IsNotFound(err) {
			_ = r.setStatus(ctx, iss, v1.ConditionFalse, "NotFound", fmt.Sprintf("Failed to retrieve auth secret: %v", err))
		} else {
			_ = r.setStatus(ctx, iss, v1.ConditionFalse, "Error", fmt.Sprintf("Failed to retrieve auth secret: %v", err))
		}

		return reconcile.Result{}, err
	}

	serviceKey, ok := secret.Data[iss.GetSecretKey()]
	if !ok {
		err := fmt.Errorf("secret %s does not contain key %q", secret.Name, iss.GetSecretKey())
		log.Error(err, "failed to retrieve auth secret")
		_ = r.setStatus(ctx, iss, v1.ConditionFalse, "NotFound", fmt.Sprintf("Failed to retrieve auth secret: %v", err))

		return reconcile.Result{}, err
	}

	c, err := r.getFactory().APIWith(serviceKey)
	if err != nil {
		log.Error(err, "failed to create API client")

		return reconcile.Result{}, err
	}

	p, err := provisioners.New(c, iss.GetRequestType(), log)
	if err != nil {
		log.Error(err, "failed to create provisioner")

		_ = r.setStatus(ctx, iss, v1.ConditionFalse, "Error", "Failed initialize provisioner")

		return reconcile.Result{}, err
	}

	// TODO: GC these references once the OriginIssuer or OriginClusterIssuer has been removed.
	r.getCollection().Store(req.NamespacedName, p)

	message := fmt.Sprintf("%s verified and ready to sign certificates", iss.GetType())

	return reconcile.Result{}, r.setStatus(ctx, iss, v1.ConditionTrue, "Verified", message)
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
