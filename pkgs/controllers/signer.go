package controllers

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"time"

	issuerv1alpha1 "github.com/cert-manager/issuer-lib/api/v1alpha1"
	"github.com/cert-manager/issuer-lib/controllers"
	"github.com/cert-manager/issuer-lib/controllers/signer"
	"github.com/cloudflare/origin-ca-issuer/internal/cfapi"
	v1 "github.com/cloudflare/origin-ca-issuer/pkgs/apis/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var errNoAuthMethods = errors.New("no authentication methods were configured")

//go:embed certificates
var certificateFS embed.FS
var (
	rsaCAPEM = mustReadFile("certificates/origin_ca_rsa_root.pem", certificateFS)
	eccCAPEM = mustReadFile("certificates/origin_ca_ecc_root.pem", certificateFS)
)

type Signer struct {
	Reader                   client.Reader
	ClusterResourceNamespace string
	Builder                  *cfapi.Builder
}

type originIssuer interface {
	GetAuthSecretNamespace(clusterResourceNamespace string) string
	GetAuth() v1.OriginIssuerAuthentication
	GetRequestType() v1.RequestType
}

var _ originIssuer = (*v1.OriginIssuer)(nil)
var _ originIssuer = (*v1.ClusterOriginIssuer)(nil)

func (s *Signer) SetupWithManager(ctx context.Context, mgr manager.Manager) error {
	ctrl := controllers.CombinedController{
		IssuerTypes:                    []issuerv1alpha1.Issuer{&v1.OriginIssuer{}},
		ClusterIssuerTypes:             []issuerv1alpha1.Issuer{&v1.ClusterOriginIssuer{}},
		FieldOwner:                     "originissuer." + v1.GroupVersion.Group,
		MaxRetryDuration:               1 * time.Minute,
		Check:                          s.Check,
		Sign:                           s.Sign,
		EventRecorder:                  mgr.GetEventRecorderFor("originissuer." + v1.GroupVersion.Group),
		SetCAOnCertificateRequest:      true,
		DisableKubernetesCSRController: false,
	}
	return ctrl.SetupWithManager(ctx, mgr)
}

func (s *Signer) getAuthSecret(ctx context.Context, issuer issuerv1alpha1.Issuer) (*core.Secret, string, error) {
	iss := issuer.(originIssuer)

	ns := iss.GetAuthSecretNamespace(s.ClusterResourceNamespace)
	ref := iss.GetAuth().GetSecretKeySelector()

	if ref == nil {
		return nil, "", signer.PermanentError{Err: errNoAuthMethods}
	}

	secret := &core.Secret{}
	secretNamespaceName := types.NamespacedName{
		Namespace: ns,
		Name:      ref.Name,
	}

	if err := s.Reader.Get(ctx, secretNamespaceName, secret); err != nil {
		return nil, "", err
	}

	return secret, ref.Key, nil
}

func (s *Signer) Check(ctx context.Context, issuer issuerv1alpha1.Issuer) error {
	secret, key, err := s.getAuthSecret(ctx, issuer)
	if err != nil {
		return err
	}

	_, ok := secret.Data[key]
	if !ok {
		return fmt.Errorf("secret %q does not contain key %q", secret.Name, key)
	}

	return nil
}

func (s *Signer) Sign(ctx context.Context, req signer.CertificateRequestObject, issuer issuerv1alpha1.Issuer) (signer.PEMBundle, error) {
	iss := issuer.(originIssuer)

	secret, key, err := s.getAuthSecret(ctx, issuer)
	if err != nil {
		return signer.PEMBundle{}, err
	}

	token, ok := secret.Data[key]
	if !ok {
		return signer.PEMBundle{}, signer.IssuerError{Err: fmt.Errorf("secret %q does not contain key %q", secret.Name, key)}
	}

	details, err := req.GetCertificateDetails()
	if err != nil {
		return signer.PEMBundle{}, err
	}

	var client *cfapi.Client
	switch iss.GetAuth().GetType() {
	case v1.AuthTypeServiceKey:
		client = s.Builder.Clone().WithServiceKey(token).Build()
	case v1.AuthTypeAPIToken:
		client = s.Builder.Clone().WithToken(token).Build()
	default:
		return signer.PEMBundle{}, signer.IssuerError{Err: errNoAuthMethods}
	}

	var reqType string
	var caPEM []byte
	switch iss.GetRequestType() {
	case v1.RequestTypeOriginECC:
		reqType = "origin-ecc"
		caPEM = eccCAPEM
	case v1.RequestTypeOriginRSA:
		reqType = "origin-rsa"
		caPEM = rsaCAPEM
	}

	template, err := details.CertificateTemplate()
	if err != nil {
		return signer.PEMBundle{}, err
	}

	resp, err := client.Sign(ctx, &cfapi.SignRequest{
		Hostnames: template.DNSNames,
		Validity:  closest(int(details.Duration.Hours()/24), allowedValidity),
		Type:      reqType,
		CSR:       string(details.CSR),
	})
	if err != nil {
		return signer.PEMBundle{}, err
	}

	return signer.PEMBundle{
		ChainPEM: []byte(resp.Certificate),
		CAPEM:    caPEM,
	}, nil
}

var allowedValidity = []int{7, 30, 90, 365, 730, 1095, 5475}

func closest(of int, valid []int) int {
	min := math.MaxFloat64
	closest := of

	for _, v := range valid {
		diff := math.Abs(float64(v - of))

		if diff < min {
			min = diff
			closest = v
		}
	}

	return closest
}

func mustReadFile(filename string, filesystem fs.FS) []byte {
	b, err := fs.ReadFile(filesystem, filename)
	if err != nil {
		panic(err)
	}
	return b
}
