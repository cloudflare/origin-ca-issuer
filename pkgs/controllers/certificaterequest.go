package controllers

import (
	"context"
	"fmt"

	v1 "github.com/cloudflare/origin-ca-issuer/pkgs/apis/v1"
	"github.com/cloudflare/origin-ca-issuer/pkgs/provisioners"
	"github.com/go-logr/logr"
	cmutil "github.com/jetstack/cert-manager/pkg/api/util"
	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// CertificateRequestController implements a controller that reconciles CertificateRequests
// that references this controller.
type CertificateRequestController struct {
	client.Client
	Log        logr.Logger
	Collection *provisioners.Collection
}

// +kubebuilder:rbac:groups=cert-manager.io,resources=certificaterequests,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificaterequests/status,verbs=get;update;patch

// Reconcile reconciles CertificateRequest by fetching a Cloudflare API provisioner from
// the referenced OriginIssuer, and providing the request's CSR.
func (r *CertificateRequestController) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	ctx := context.TODO()
	log := r.Log.WithValues("certificaterequest", req.NamespacedName)

	cr := &certmanager.CertificateRequest{}
	if err := r.Client.Get(ctx, req.NamespacedName, cr); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, client.IgnoreNotFound(err)
		}

		log.Error(err, "failed to retrieve certificate request")

		return reconcile.Result{}, err
	}

	if cr.Spec.IssuerRef.Group != "" && cr.Spec.IssuerRef.Group != v1.GroupVersion.Group {
		log.V(4).Info("resource does not specify an issuerRef group name that we are responsible for", "group", cr.Spec.IssuerRef.Group)

		return reconcile.Result{}, nil
	}

	if len(cr.Status.Certificate) > 0 {
		log.V(4).Info("existing certificate data found in status, skipping already completed certificate request")

		return reconcile.Result{}, nil
	}

	if cr.Spec.IsCA {
		log.Info("Origin Issuer does not support signing of CA certificates")

		return reconcile.Result{}, nil
	}

	iss := v1.OriginIssuer{}
	issNamespaceName := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      cr.Spec.IssuerRef.Name,
	}

	if err := r.Client.Get(ctx, issNamespaceName, &iss); err != nil {
		log.Error(err, "failed to retrieve OriginIssuer resource", "namespace", issNamespaceName.Namespace, "name", issNamespaceName.Name)
		_ = r.setStatus(ctx, cr, cmmeta.ConditionFalse, certmanager.CertificateRequestReasonPending, fmt.Sprintf("Failed to retrieve OriginIssuer resource %s: %v", issNamespaceName, err))

		return reconcile.Result{}, err
	}

	if !IssuerHasCondition(iss, v1.OriginIssuerCondition{Type: v1.ConditionReady, Status: v1.ConditionTrue}) {
		err := fmt.Errorf("resource %s is not ready", issNamespaceName)
		log.Error(err, "issuer failed readiness checks", "namespace", issNamespaceName.Namespace, "name", issNamespaceName.Name)
		_ = r.setStatus(ctx, cr, cmmeta.ConditionFalse, certmanager.CertificateRequestReasonPending, fmt.Sprintf("OriginIssuer %s is not Ready", issNamespaceName))

		return reconcile.Result{}, err
	}

	p, ok := r.Collection.Load(issNamespaceName)
	if !ok {
		err := fmt.Errorf("provisioner %s not found", issNamespaceName)
		log.Error(err, "failed to load provisioner for OriginIssuer resource")

		_ = r.setStatus(ctx, cr, cmmeta.ConditionFalse, certmanager.CertificateRequestReasonPending, fmt.Sprintf("Failed to load provisioner for OriginIssuer resource %s", issNamespaceName))

		return reconcile.Result{}, err
	}

	pem, err := p.Sign(ctx, cr)
	if err != nil {
		log.Error(err, "failed to sign certificate request")
		_ = r.setStatus(ctx, cr, cmmeta.ConditionFalse, certmanager.CertificateRequestReasonFailed, fmt.Sprintf("Failed to sign certificate request: %v", err))

		return reconcile.Result{}, err
	}

	cr.Status.Certificate = pem
	_ = r.setStatus(ctx, cr, cmmeta.ConditionTrue, certmanager.CertificateRequestReasonIssued, "Certificate issued")

	return reconcile.Result{}, nil
}

// setStatus is a helper function to set the CertifcateRequest status condition with reason and message, and update the API.
func (r *CertificateRequestController) setStatus(ctx context.Context, cr *certmanager.CertificateRequest, status cmmeta.ConditionStatus, reason, message string) error {
	cmutil.SetCertificateRequestCondition(cr, certmanager.CertificateRequestConditionReady, status, reason, message)

	return r.Client.Status().Update(ctx, cr)
}
