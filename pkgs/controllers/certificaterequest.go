package controllers

import (
	"context"
	"embed"
	"errors"
	"fmt"

	cmutil "github.com/cert-manager/cert-manager/pkg/api/util"
	certmanager "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/cloudflare/origin-ca-issuer/internal/cfapi"
	v1 "github.com/cloudflare/origin-ca-issuer/pkgs/apis/v1"
	"github.com/cloudflare/origin-ca-issuer/pkgs/provisioners"
	"github.com/go-logr/logr"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

//go:embed certificates
var certificateFS embed.FS

var (
	rsaCAPEM = MustReadFile("certificates/origin_ca_rsa_root.pem", certificateFS)
	eccCAPEM = MustReadFile("certificates/origin_ca_ecc_root.pem", certificateFS)
)

const originDBWriteErrorCode = 1100

// CertificateRequestController implements a controller that reconciles CertificateRequests
// that references this controller.
type CertificateRequestController struct {
	client.Client
	Reader                   client.Reader
	ClusterResourceNamespace string
	Log                      logr.Logger
	Builder                  *cfapi.Builder

	Clock                  clock.Clock
	CheckApprovedCondition bool
}

// +kubebuilder:rbac:groups=cert-manager.io,resources=certificaterequests,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificaterequests/status,verbs=get;update;patch

// Reconcile reconciles CertificateRequest by fetching a Cloudflare API provisioner from
// the referenced OriginIssuer, and providing the request's CSR.
func (r *CertificateRequestController) Reconcile(ctx context.Context, cr *certmanager.CertificateRequest) (reconcile.Result, error) {
	log := r.Log.WithValues("namespace", cr.Namespace, "certificaterequest", cr.Name)

	if cr.Spec.IssuerRef.Group != v1.GroupVersion.Group {
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

	var (
		secretNamespace string
		issuerspec      v1.OriginIssuerSpec
	)

	switch cr.Spec.IssuerRef.Kind {
	case "OriginIssuer":
		iss := v1.OriginIssuer{}
		issNamespaceName := types.NamespacedName{
			Namespace: cr.Namespace,
			Name:      cr.Spec.IssuerRef.Name,
		}

		if err := r.Client.Get(ctx, issNamespaceName, &iss); err != nil {
			log.Error(err, "failed to retrieve OriginIssuer resource", "namespace", issNamespaceName.Namespace, "name", issNamespaceName.Name)
			_ = r.setStatus(ctx, cr, cmmeta.ConditionFalse, certmanager.CertificateRequestReasonPending, fmt.Sprintf("Failed to retrieve OriginIssuer resource %s: %v", issNamespaceName, err))

			return reconcile.Result{}, err
		}

		if !IssuerStatusHasCondition(iss.Status, v1.OriginIssuerCondition{Type: v1.ConditionReady, Status: v1.ConditionTrue}) {
			err := fmt.Errorf("resource %s is not ready", issNamespaceName)
			log.Error(err, "issuer failed readiness checks", "namespace", issNamespaceName.Namespace, "name", issNamespaceName.Name)
			_ = r.setStatus(ctx, cr, cmmeta.ConditionFalse, certmanager.CertificateRequestReasonPending, fmt.Sprintf("OriginIssuer %s is not Ready", issNamespaceName))

			return reconcile.Result{}, err
		}

		secretNamespace = iss.Namespace
		issuerspec = iss.Spec
	case "ClusterOriginIssuer":
		iss := v1.ClusterOriginIssuer{}
		issNamespaceName := types.NamespacedName{
			Name: cr.Spec.IssuerRef.Name,
		}

		if err := r.Client.Get(ctx, issNamespaceName, &iss); err != nil {
			log.Error(err, "failed to retrieve OriginIssuer resource", "namespace", issNamespaceName.Namespace, "name", issNamespaceName.Name)
			_ = r.setStatus(ctx, cr, cmmeta.ConditionFalse, certmanager.CertificateRequestReasonPending, fmt.Sprintf("Failed to retrieve OriginIssuer resource %s: %v", issNamespaceName, err))

			return reconcile.Result{}, err
		}

		if !IssuerStatusHasCondition(iss.Status, v1.OriginIssuerCondition{Type: v1.ConditionReady, Status: v1.ConditionTrue}) {
			err := fmt.Errorf("resource %s is not ready", issNamespaceName)
			log.Error(err, "issuer failed readiness checks", "namespace", issNamespaceName.Namespace, "name", issNamespaceName.Name)
			_ = r.setStatus(ctx, cr, cmmeta.ConditionFalse, certmanager.CertificateRequestReasonPending, fmt.Sprintf("OriginIssuer %s is not Ready", issNamespaceName))

			return reconcile.Result{}, err
		}

		secretNamespace = r.ClusterResourceNamespace
		issuerspec = iss.Spec
	default:
		err := fmt.Errorf("unknown issuer kind: %s", cr.Spec.IssuerRef.Kind)
		log.Error(err, "certificate request references unknown issuer kind", "namespace", cr.Namespace, "name", cr.Name)
		_ = r.setStatus(ctx, cr, cmmeta.ConditionFalse, certmanager.CertificateRequestReasonFailed, fmt.Sprintf("Unknown issuer kind: %s", cr.Spec.IssuerRef.Kind))

		return reconcile.Result{}, err
	}

	var c *cfapi.Client
	switch {
	case issuerspec.Auth.ServiceKeyRef != nil:
		var secret core.Secret

		secretNamespaceName := types.NamespacedName{
			Namespace: secretNamespace,
			Name:      issuerspec.Auth.ServiceKeyRef.Name,
		}

		if err := r.Reader.Get(ctx, secretNamespaceName, &secret); err != nil {
			log.Error(err, "failed to retieve OriginIssuer auth secret", "namespace", secretNamespaceName.Namespace, "name", secretNamespaceName.Name)
			if apierrors.IsNotFound(err) {
				_ = r.setStatus(ctx, cr, cmmeta.ConditionFalse, "NotFound", fmt.Sprintf("Failed to retrieve auth secret: %v", err))
			} else {
				_ = r.setStatus(ctx, cr, cmmeta.ConditionFalse, "Error", fmt.Sprintf("Failed to retrieve auth secret: %v", err))
			}

			return reconcile.Result{}, err
		}

		serviceKey, ok := secret.Data[issuerspec.Auth.ServiceKeyRef.Key]
		if !ok {
			err := fmt.Errorf("secret %s does not contain key %q", secret.Name, issuerspec.Auth.ServiceKeyRef.Key)
			log.Error(err, "failed to retrieve OriginIssuer auth secret")
			_ = r.setStatus(ctx, cr, cmmeta.ConditionFalse, "NotFound", fmt.Sprintf("Failed to retrieve auth secret: %v", err))

			return reconcile.Result{}, err
		}

		c = r.Builder.Clone().WithServiceKey(serviceKey).Build()
	case issuerspec.Auth.TokenRef != nil:
		var secret core.Secret

		secretNamespaceName := types.NamespacedName{
			Namespace: secretNamespace,
			Name:      issuerspec.Auth.TokenRef.Name,
		}

		if err := r.Reader.Get(ctx, secretNamespaceName, &secret); err != nil {
			log.Error(err, "failed to retieve OriginIssuer auth secret", "namespace", secretNamespaceName.Namespace, "name", secretNamespaceName.Name)
			if apierrors.IsNotFound(err) {
				_ = r.setStatus(ctx, cr, cmmeta.ConditionFalse, "NotFound", fmt.Sprintf("Failed to retrieve auth secret: %v", err))
			} else {
				_ = r.setStatus(ctx, cr, cmmeta.ConditionFalse, "Error", fmt.Sprintf("Failed to retrieve auth secret: %v", err))
			}

			return reconcile.Result{}, err
		}

		token, ok := secret.Data[issuerspec.Auth.TokenRef.Key]
		if !ok {
			err := fmt.Errorf("secret %s does not contain key %q", secret.Name, issuerspec.Auth.TokenRef.Key)
			log.Error(err, "failed to retrieve OriginIssuer auth secret")
			_ = r.setStatus(ctx, cr, cmmeta.ConditionFalse, "NotFound", fmt.Sprintf("Failed to retrieve auth secret: %v", err))

			return reconcile.Result{}, err
		}

		c = r.Builder.Clone().WithToken(token).Build()
	default:
		// This issuer should not be ready!
		err := fmt.Errorf("issuer %s does not have an authentication method configured", cr.Spec.IssuerRef.Name)
		log.Error(err, "failed to retrieve issuer auth secret")
		return reconcile.Result{}, err
	}

	p, err := provisioners.New(c, issuerspec.RequestType, log)
	if err != nil {
		log.Error(err, "failed to create provisioner")

		_ = r.setStatus(ctx, cr, cmmeta.ConditionFalse, "Error", "Failed initialize provisioner")

		return reconcile.Result{}, err
	}

	pem, err := p.Sign(ctx, cr)

	var apiError *cfapi.APIError
	if errors.As(err, &apiError) {
		if apiError.Code == originDBWriteErrorCode {
			log.Error(err, "requeue-ing after API error")
			return reconcile.Result{}, err
		}
	}

	if err != nil {
		log.Error(err, "failed to sign certificate request")
		_ = r.setStatus(ctx, cr, cmmeta.ConditionFalse, certmanager.CertificateRequestReasonFailed, fmt.Sprintf("Failed to sign certificate request: %v", err))

		return reconcile.Result{}, err
	}

	cr.Status.Certificate = pem
	switch issuerspec.RequestType {
	case v1.RequestTypeOriginECC:
		cr.Status.CA = eccCAPEM
	case v1.RequestTypeOriginRSA:
		cr.Status.CA = rsaCAPEM
	}
	_ = r.setStatus(ctx, cr, cmmeta.ConditionTrue, certmanager.CertificateRequestReasonIssued, "Certificate issued")

	return reconcile.Result{}, nil
}

// setStatus is a helper function to set the CertifcateRequest status condition with reason and message, and update the API.
func (r *CertificateRequestController) setStatus(ctx context.Context, cr *certmanager.CertificateRequest, status cmmeta.ConditionStatus, reason, message string) error {
	cmutil.SetCertificateRequestCondition(cr, certmanager.CertificateRequestConditionReady, status, reason, message)

	return r.Client.Status().Update(ctx, cr)
}

func MustReadFile(filename string, fs embed.FS) []byte {
	b, err := fs.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return b
}
