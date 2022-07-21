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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// CertificateRequestController implements a controller that reconciles CertificateRequests
// that references this controller.
type CertificateRequestController struct {
	client.Client
	Log        logr.Logger
	Collection *provisioners.Collection

	Clock                  clock.Clock
	CheckApprovedCondition bool
}

// +kubebuilder:rbac:groups=cert-manager.io,resources=certificaterequests,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificaterequests/status,verbs=get;update;patch

// Reconcile reconciles CertificateRequest by fetching a Cloudflare API provisioner from
// the referenced OriginIssuer, and providing the request's CSR.
func (r *CertificateRequestController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	log := r.Log.WithValues("certificaterequest", req.NamespacedName)

	cr := &certmanager.CertificateRequest{}
	if err := r.Client.Get(ctx, req.NamespacedName, cr); err != nil {
		log.Error(err, "failed to retrieve certificate request")

		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	if cr.Spec.IssuerRef.Group != "" && cr.Spec.IssuerRef.Group != v1.GroupVersion.Group {
		log.V(4).Info("resource does not specify an issuerRef group name that we are responsible for", "group", cr.Spec.IssuerRef.Group)
		return reconcile.Result{}, nil
	}

	// Ignore CertificateRequest if it is already Ready
	if cmutil.CertificateRequestHasCondition(cr, certmanager.CertificateRequestCondition{
		Type:   certmanager.CertificateRequestConditionReady,
		Status: cmmeta.ConditionTrue,
	}) {
		log.V(4).Info("CertificateRequest is Ready. Ignoring.")
		return reconcile.Result{}, nil
	}
	// Ignore CertificateRequest if it is already Failed
	if cmutil.CertificateRequestHasCondition(cr, certmanager.CertificateRequestCondition{
		Type:   certmanager.CertificateRequestConditionReady,
		Status: cmmeta.ConditionFalse,
		Reason: certmanager.CertificateRequestReasonFailed,
	}) {
		log.V(4).Info("CertificateRequest is Failed. Ignoring.")
		return reconcile.Result{}, nil
	}
	// Ignore CertificateRequest if it already has a Denied Ready Reason
	if cmutil.CertificateRequestHasCondition(cr, certmanager.CertificateRequestCondition{
		Type:   certmanager.CertificateRequestConditionReady,
		Status: cmmeta.ConditionFalse,
		Reason: certmanager.CertificateRequestReasonDenied,
	}) {
		log.V(4).Info("CertificateRequest already has a Ready condition with Denied Reason. Ignoring.")
		return reconcile.Result{}, nil
	}

	// If CertificateRequest has been denied, mark the CertificateRequest as
	// Ready=Denied and set FailureTime if not already.
	if cmutil.CertificateRequestIsDenied(cr) {
		log.V(4).Info("CertificateRequest has been denied. Marking as failed.")

		if cr.Status.FailureTime == nil {
			nowTime := metav1.NewTime(r.Clock.Now())
			cr.Status.FailureTime = &nowTime
		}

		message := "The CertificateRequest was denied by an approval controller"
		return reconcile.Result{}, r.setStatus(ctx, cr, cmmeta.ConditionFalse, certmanager.CertificateRequestReasonDenied, message)
	}

	if r.CheckApprovedCondition {
		// If CertificateRequest has not been approved, exit early.
		if !cmutil.CertificateRequestIsApproved(cr) {
			log.V(4).Info("certificate request has not been approved")
			return reconcile.Result{}, nil
		}
	}

	if len(cr.Status.Certificate) > 0 {
		log.V(4).Info("existing certificate data found in status, skipping already completed certificate request")

		return reconcile.Result{}, nil
	}

	if cr.Spec.IsCA {
		log.Info("Origin Issuer does not support signing of CA certificates")

		return reconcile.Result{}, nil
	}

	p, err := r.getProvisioner(cr, req, log, ctx)
	if err != nil {
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

func (r *CertificateRequestController) getProvisioner(cr *certmanager.CertificateRequest, req reconcile.Request, log logr.Logger, ctx context.Context) (*provisioners.Provisioner, error) {
	issuer := &v1.OriginIssuer{}
	var issNamespaceName types.NamespacedName
	var iss CFIssuer

	if cr.Spec.IssuerRef.Kind == string(issuer.GetType()) {
		iss = &v1.OriginIssuer{}
		issNamespaceName = types.NamespacedName{
			Namespace: req.Namespace,
			Name:      cr.Spec.IssuerRef.Name,
		}
	} else {
		iss = &v1.OriginClusterIssuer{}
		issNamespaceName = types.NamespacedName{
			Name: cr.Spec.IssuerRef.Name,
		}
	}

	if err := r.Client.Get(ctx, issNamespaceName, iss.(client.Object)); err != nil {
		log.Error(err, "failed to retrieve OriginIssuer resource", "resource", issNamespaceName)
		_ = r.setStatus(ctx, cr, cmmeta.ConditionFalse, certmanager.CertificateRequestReasonPending, fmt.Sprintf("Failed to retrieve %s resource %s: %v", cr.Spec.IssuerRef.Kind, issNamespaceName, err))

		return nil, err
	}

	if !IssuerHasCondition(iss, v1.OriginIssuerCondition{Type: v1.ConditionReady, Status: v1.ConditionTrue}) {
		err := fmt.Errorf("resource %s is not ready", issNamespaceName)
		log.Error(err, "issuer failed readiness checks", "resource", issNamespaceName)
		_ = r.setStatus(ctx, cr, cmmeta.ConditionFalse, certmanager.CertificateRequestReasonPending, fmt.Sprintf("OriginIssuer %s is not Ready", issNamespaceName))

		return nil, err
	}

	p, ok := r.Collection.Load(issNamespaceName)

	if !ok {
		err := fmt.Errorf("provisioner %s not found", issNamespaceName)
		log.Error(err, "failed to load provisioner for OriginIssuer resource")

		_ = r.setStatus(ctx, cr, cmmeta.ConditionFalse, certmanager.CertificateRequestReasonPending, fmt.Sprintf("Failed to load provisioner for OriginIssuer resource %s", issNamespaceName))

		return nil, err
	}
	return p, nil

}

// setStatus is a helper function to set the CertifcateRequest status condition with reason and message, and update the API.
func (r *CertificateRequestController) setStatus(ctx context.Context, cr *certmanager.CertificateRequest, status cmmeta.ConditionStatus, reason, message string) error {
	cmutil.SetCertificateRequestCondition(cr, certmanager.CertificateRequestConditionReady, status, reason, message)

	return r.Client.Status().Update(ctx, cr)
}
